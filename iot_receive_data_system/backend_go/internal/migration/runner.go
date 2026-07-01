package migration

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	"backend_go/internal/model"
	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// Package migration runs versioned SQL schema changes at startup.
// migration 套件負責在啟動時執行版本化 SQL 結構變更。

// migrationFS embeds versioned SQL files used by golang-migrate.
// migrationFS 內嵌 golang-migrate 使用的版本化 SQL 檔案。
//
//go:embed sql/*.sql
var migrationFS embed.FS

// Run executes all pending migrations against PostgreSQL.
// Run 會對 PostgreSQL 執行所有尚未套用的 migration。
func Run(ctx context.Context, cfg model.AppConfig, log *zap.Logger) error {
	// Open a dedicated sql.DB for migration lifecycle.
	// 建立 migration 專用的 sql.DB 連線。
	db, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	// Build migrate source from embedded SQL files.
	// 由內嵌 SQL 檔建立 migrate source。
	sourceDriver, err := iofs.New(migrationFS, "sql")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()

	done := make(chan error, 1)
	go func() {
		done <- m.Up()
	}()

	// Respect startup timeout/cancel while waiting migration result.
	// 等待 migration 完成時遵守啟動逾時與取消訊號。
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	version, dirty, verr := m.Version()
	if verr == nil {
		log.Info("migration complete", zap.Uint("version", version), zap.Bool("dirty", dirty))
		return nil
	}
	if errors.Is(verr, migrate.ErrNilVersion) {
		log.Info("migration complete", zap.String("state", "no version recorded"))
		return nil
	}
	return verr
}
