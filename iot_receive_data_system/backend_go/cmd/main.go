package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"backend_go/internal/handler"
	"backend_go/internal/migration"
	"backend_go/internal/model"
	"backend_go/internal/repository"
	"backend_go/internal/router"
	"backend_go/pkg/logger"
	"go.uber.org/zap"
)

// Package main is the process bootstrap for backend_go.
// main 套件是 backend_go 的程序啟動入口。
//
// It assembles config, logging, migration, repository, workers, and HTTP server.
// 它會組裝設定、日誌、migration、repository、背景 workers 與 HTTP 服務。

// main wires all runtime dependencies and starts the HTTP service.
// main 負責組裝所有執行期相依並啟動 HTTP 服務。
// Startup flow:
// 啟動流程：
// 1) load config from env
// 1) 從環境變數載入設定
// 2) run DB migration (optional)
// 2) 執行資料庫 migration（可選）
// 3) open/ping DB
// 3) 開啟資料庫連線並做連通性檢查
// 4) start background workers
// 4) 啟動背景批次 worker
// 5) start HTTP server with graceful shutdown
// 5) 啟動 HTTP 服務並支援優雅關機
func main() {
	// Initialize structured logger first so all startup errors are observable.
	// 先初始化結構化日誌，確保啟動錯誤都可被觀測。
	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	// Load runtime knobs from environment variables.
	// 從環境變數載入執行期參數。
	cfg := model.LoadFromEnv()

	// Root context is cancelled when SIGINT/SIGTERM is received.
	// Root context 會在收到 SIGINT/SIGTERM 時被取消。
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.MigrateOnStart {
		// Run golang-migrate before serving traffic.
		// 在服務開始收流量前先執行 golang-migrate。
		migrateCtx, migrateCancel := context.WithTimeout(rootCtx, 60*time.Second)
		defer migrateCancel()
		if err := migration.Run(migrateCtx, cfg, log); err != nil {
			log.Fatal("run migration failed", zap.Error(err))
		}
	} else {
		log.Info("migration skipped", zap.Bool("migrate_on_start", false))
	}

	// Repository owns DB connectivity and SQL operations.
	// Repository 層負責 DB 連線與 SQL 存取。
	repo, err := repository.NewSQLSensorRepository(cfg)
	if err != nil {
		log.Fatal("open db failed", zap.Error(err))
	}
	defer repo.Close()

	pingCtx, pingCancel := context.WithTimeout(rootCtx, 5*time.Second)
	defer pingCancel()
	if err := repo.Ping(pingCtx); err != nil {
		log.Fatal("ping db failed", zap.Error(err))
	}

	// Ensure partitions are pre-created and retention policy is applied.
	// 確保分區預先建立，並套用留存策略。
	if cfg.PartitionMaintainOnStart {
		maintainCtx, maintainCancel := context.WithTimeout(rootCtx, 25*time.Second)
		defer maintainCancel()
		if err := repo.MaintainStorage(maintainCtx); err != nil {
			log.Fatal("maintain partition/retention failed", zap.Error(err))
		}
	} else {
		log.Info("partition maintenance skipped", zap.Bool("partition_maintain_on_start", false))
	}

	// Seed baseline data for hospitals/floors/rooms if enabled.
	// 若啟用則灌入醫院/樓層/房間基礎資料。
	if cfg.SeedOnStart {
		seedCtx, seedCancel := context.WithTimeout(rootCtx, 25*time.Second)
		defer seedCancel()
		if err := repo.SeedBaseData(seedCtx); err != nil {
			log.Fatal("run base-data seeder failed", zap.Error(err))
		}
	} else {
		log.Info("base-data seeder skipped", zap.Bool("seed_on_start", false))
	}

	// Handler owns ingest pipeline (validation, dedup, sharding, queueing).
	// Handler 層負責接收資料流程（驗證、去重、分片、入佇列）。
	h := handler.NewSensorHandler(cfg, repo, log)

	var workerWG sync.WaitGroup
	// Start sensor/raw batch workers before accepting traffic.
	// 在開始收流量前，先啟動 sensor/raw 批次 worker。
	h.StartWorkers(rootCtx, &workerWG)

	// Build HTTP engine and server-level network timeouts.
	// 建立 HTTP engine 與 server 層級網路逾時設定。
	engine := router.New(h, cfg, log)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	// Stop HTTP server when process receives SIGINT/SIGTERM.
	// 當程序收到 SIGINT/SIGTERM 時，關閉 HTTP 服務。
	go func() {
		<-rootCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("http shutdown error", zap.Error(err))
		}
	}()

	log.Info("backend_go started",
		zap.String("port", cfg.Port),
		zap.String("provider", "postgres"),
		zap.Bool("http_log_every_request", cfg.HTTPLogEveryRequest),
		zap.Bool("http_log_request_body", cfg.HTTPLogRequestBody),
		zap.Int("http_log_request_body_max_bytes", cfg.HTTPLogRequestBodyBytes),
		zap.Int("sensor_batch", cfg.SensorBatchSize),
		zap.Int("raw_batch", cfg.RawBatchSize),
		zap.Int("sensor_workers", cfg.SensorWorkers),
		zap.Int("raw_workers", cfg.RawWorkers),
		zap.Int("raw_log_retention_days", cfg.RawLogRetentionDays),
		zap.Int("raw_log_partition_ahead_days", cfg.RawLogPartitionAheadDays),
		zap.Duration("db_query_timeout", cfg.DBQueryTimeout),
		zap.Duration("http_write_timeout", cfg.HTTPWriteTimeout),
		zap.Bool("seed_on_start", cfg.SeedOnStart),
	)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("http server error", zap.Error(err))
	}

	// Wait until background workers flush remaining buffered data.
	// 等待背景 worker 把剩餘緩衝資料寫完再結束程序。
	workerWG.Wait()
}
