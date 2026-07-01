package logger

import (
	"go.uber.org/zap"
)

// Package logger provides shared logging construction.
// logger 套件提供共用的日誌建立邏輯。

// New creates the shared Zap logger instance used by all layers.
// New 建立所有層共用的 Zap logger 實例。
// Caller info is disabled to reduce log payload size in high-throughput ingest.
// 關閉 caller 資訊以降低高吞吐場景下的日誌負載大小。
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.DisableCaller = true
	return cfg.Build()
}
