package model

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Package model defines runtime configs and ingest domain models.
// model 套件定義執行期設定與資料接收領域模型。

// AppConfig centralizes all runtime knobs for backend_go.
// AppConfig 集中管理 backend_go 的所有執行期參數。
// It includes DB connectivity, ingest pipeline sizing, timeouts, and dedup settings.
// 內容包含 DB 連線、接收管線容量、逾時與去重設定。
type AppConfig struct {
	// HTTP / Frontend gateway settings.
	// HTTP / 前端閘道設定。
	Port                    string
	HTTPLogEveryRequest     bool
	HTTPLogRequestBody      bool
	HTTPLogRequestBodyBytes int

	// PostgreSQL connectivity.
	// PostgreSQL 連線設定。
	PGHost         string
	PGPort         string
	PGUser         string
	PGPassword     string
	PGDBName       string
	MigrateOnStart bool
	DBDumpToken    string

	// Ingest pipeline sizing and payload limits.
	// 接收管線容量與 payload 限制。
	MaxBufferSize        int
	MaxBodyBytes         int64
	SensorBatchSize      int
	RawBatchSize         int
	BatchInterval        time.Duration
	SensorWorkers        int
	RawWorkers           int
	QueueSize            int
	QueueOfferTimout     time.Duration
	SensorCacheTTL       time.Duration
	ThermalFrameInterval time.Duration

	// Dedup cache controls.
	// 去重快取控制參數。
	RequestDedupTTL     time.Duration
	RequestDedupMaxKeys int
	EventDedupTTL       time.Duration
	EventDedupMaxKeys   int

	// Database pool and timeout controls.
	// 資料庫連線池與逾時控制。
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetime    time.Duration
	DBQueryTimeout       time.Duration
	DBInsertTimeout      time.Duration
	HTTPReadTimeout      time.Duration
	HTTPWriteTimeout     time.Duration
	HTTPIdleTimeout      time.Duration
	HTTPShutdownTimeout  time.Duration
	DeviceOfflineTimeout time.Duration

	// Partition and retention maintenance controls.
	// 分區與留存維護控制參數。
	PartitionMaintainOnStart       bool
	RawLogRetentionDays            int
	RawLogPartitionBackDays        int
	RawLogPartitionAheadDays       int
	SensorDataPartitionBackMonths  int
	SensorDataPartitionAheadMonths int

	// Base-data seeder controls.
	// 基礎資料 seeder 控制參數。
	SeedOnStart      bool
	SeedHospitalName string
}

// LoadFromEnv reads all supported env vars and applies safe defaults.
// LoadFromEnv 讀取所有支援的環境變數並套用安全預設值。
// Empty or invalid numeric values automatically fall back to defaults.
// 若數值為空或格式不正確，會自動回退到預設值。
func LoadFromEnv() AppConfig {
	// Convert second-based env values to duration once, then reuse.
	// 將以秒為單位的環境變數先轉成 duration，後續重複使用。
	cacheTTLSeconds := envIntOrDefault("GO_SENSOR_CACHE_TTL_SEC", 60)
	requestDedupTTLSeconds := envIntOrDefault("GO_REQUEST_DEDUP_TTL_SEC", 8)
	eventDedupTTLSeconds := envIntOrDefault("GO_EVENT_DEDUP_TTL_SEC", 300)
	dbLifetimeSec := envIntOrDefault("GO_DB_CONN_MAX_LIFETIME_SEC", 300)

	return AppConfig{
		Port:                    envOrDefault("PORT", "3002"),
		HTTPLogEveryRequest:     envBoolOrDefault("GO_HTTP_LOG_EVERY_REQUEST", true),
		HTTPLogRequestBody:      envBoolOrDefault("GO_HTTP_LOG_REQUEST_BODY", false),
		HTTPLogRequestBodyBytes: envIntOrDefault("GO_HTTP_LOG_REQUEST_BODY_MAX_BYTES", 65536),

		PGHost:         envOrDefault("PG_HOST", "localhost"),
		PGPort:         envOrDefault("PG_PORT", "5432"),
		PGUser:         envOrDefault("PG_USER", ""),
		PGPassword:     envOrDefault("PG_PASSWORD", ""),
		PGDBName:       envOrDefault("PG_DB_NAME", ""),
		MigrateOnStart: envBoolOrDefault("GO_MIGRATE_ON_START", true),
		DBDumpToken:    envOrDefault("GO_DB_DUMP_TOKEN", ""),

		MaxBufferSize:        envIntOrDefault("GO_MAX_BUFFER_SIZE", 10*1024),
		MaxBodyBytes:         int64(envIntOrDefault("GO_MAX_BODY_BYTES", 64*1024)),
		SensorBatchSize:      envIntOrDefault("GO_SENSOR_BATCH_SIZE", 100),
		RawBatchSize:         envIntOrDefault("GO_RAW_BATCH_SIZE", 100),
		BatchInterval:        time.Duration(envIntOrDefault("GO_BATCH_INTERVAL_MS", 200)) * time.Millisecond,
		SensorWorkers:        envIntOrDefault("GO_SENSOR_WORKERS", 4),
		RawWorkers:           envIntOrDefault("GO_RAW_WORKERS", 2),
		QueueSize:            envIntOrDefault("GO_QUEUE_SIZE", 5000),
		QueueOfferTimout:     time.Duration(envIntOrDefault("GO_QUEUE_OFFER_TIMEOUT_MS", 100)) * time.Millisecond,
		SensorCacheTTL:       time.Duration(cacheTTLSeconds) * time.Second,
		ThermalFrameInterval: time.Duration(envIntOrDefault("GO_THERMAL_FRAME_INTERVAL_SEC", 10)) * time.Second,

		RequestDedupTTL:     time.Duration(requestDedupTTLSeconds) * time.Second,
		RequestDedupMaxKeys: envIntOrDefault("GO_REQUEST_DEDUP_MAX_KEYS", 200000),
		EventDedupTTL:       time.Duration(eventDedupTTLSeconds) * time.Second,
		EventDedupMaxKeys:   envIntOrDefault("GO_EVENT_DEDUP_MAX_KEYS", 500000),

		DBMaxOpenConns:       envIntOrDefault("GO_DB_MAX_OPEN_CONNS", 60),
		DBMaxIdleConns:       envIntOrDefault("GO_DB_MAX_IDLE_CONNS", 20),
		DBConnMaxLifetime:    time.Duration(dbLifetimeSec) * time.Second,
		DBQueryTimeout:       time.Duration(envIntOrDefault("GO_DB_QUERY_TIMEOUT_SEC", 120)) * time.Second,
		DBInsertTimeout:      time.Duration(envIntOrDefault("GO_DB_INSERT_TIMEOUT_SEC", 5)) * time.Second,
		HTTPReadTimeout:      time.Duration(envIntOrDefault("GO_HTTP_READ_TIMEOUT_SEC", 5)) * time.Second,
		HTTPWriteTimeout:     time.Duration(envIntOrDefault("GO_HTTP_WRITE_TIMEOUT_SEC", 300)) * time.Second,
		HTTPIdleTimeout:      time.Duration(envIntOrDefault("GO_HTTP_IDLE_TIMEOUT_SEC", 60)) * time.Second,
		HTTPShutdownTimeout:  time.Duration(envIntOrDefault("GO_HTTP_SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
		DeviceOfflineTimeout: time.Duration(envIntOrDefault("GO_DEVICE_OFFLINE_SECONDS", 30)) * time.Second,

		PartitionMaintainOnStart:       envBoolOrDefault("GO_PARTITION_MAINTAIN_ON_START", true),
		RawLogRetentionDays:            envIntOrDefault("GO_RAW_LOG_RETENTION_DAYS", 30),
		RawLogPartitionBackDays:        envIntOrDefault("GO_RAW_LOG_PARTITION_BACK_DAYS", 1),
		RawLogPartitionAheadDays:       envIntOrDefault("GO_RAW_LOG_PARTITION_AHEAD_DAYS", 7),
		SensorDataPartitionBackMonths:  envIntOrDefault("GO_SENSOR_DATA_PARTITION_BACK_MONTHS", 1),
		SensorDataPartitionAheadMonths: envIntOrDefault("GO_SENSOR_DATA_PARTITION_AHEAD_MONTHS", 3),
		SeedOnStart:                    envBoolOrDefault("GO_SEED_ON_START", true),
		SeedHospitalName:               envOrDefault("GO_SEED_HOSPITAL_NAME", "養護中心"),
	}
}

// envOrDefault returns fallback when the env var is missing or blank.
// envOrDefault 在環境變數不存在或為空字串時回傳 fallback。
func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

// envIntOrDefault parses positive integer env values, otherwise returns fallback.
// envIntOrDefault 解析正整數環境變數，失敗時回傳 fallback。
func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// envBoolOrDefault parses common boolean env values, otherwise returns fallback.
// envBoolOrDefault 解析常見布林環境變數，失敗時回傳 fallback。
func envBoolOrDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

// PostgresDSN builds a PostgreSQL DSN with UTC time zone.
// PostgresDSN 產生帶 UTC 時區設定的 PostgreSQL 連線字串。
func (c AppConfig) PostgresDSN() string {
	return strings.TrimSpace(
		"host=" + c.PGHost +
			" port=" + c.PGPort +
			" user=" + c.PGUser +
			" password=" + c.PGPassword +
			" dbname=" + c.PGDBName +
			" sslmode=disable TimeZone=UTC",
	)
}
