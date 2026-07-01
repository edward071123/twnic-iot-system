package router

import (
	"net/http"

	"backend_go/internal/handler"
	"backend_go/internal/model"
	"backend_go/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Package router defines HTTP routes exposed by backend_go.
// router 套件定義 backend_go 對外提供的 HTTP 路由。

// New builds the HTTP routing table for backend_go.
// New 建立 backend_go 的 HTTP 路由表。
// This router exposes ward monitoring read endpoints and sensor ingest endpoint.
// 此路由提供病房監測讀取端點與感測資料上拋端點。
func New(h *handler.SensorHandler, cfg model.AppConfig, log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	if cfg.HTTPLogEveryRequest {
		r.Use(requestLogger(log, cfg))
	}

	// Health endpoint for liveness checks.
	// 存活檢查端點。
	r.GET("/healthz", h.Healthz)
	// Read endpoints for frontend ward floor-plan data.
	// 前端病房平面圖讀取端點。
	r.GET("/api/ward/floors", h.WardFloors)
	r.GET("/api/ward/floors/:floor/overview", h.WardFloorOverview)
	r.GET("/api/ward/floors/:floor/stream", h.WardFloorStream)
	r.GET("/api/ward/sensors/:sensorNumber/history", h.SensorHistory)
	r.GET("/api/ward/sensors/:sensorNumber/thermal/timeline", h.SensorThermalTimeline)
	r.GET("/api/ward/sensors/:sensorNumber/thermal/latest", h.SensorLatestThermalFrame)
	r.GET("/api/ward/sensors/:sensorNumber/thermal/:dataID", h.SensorThermalFrame)
	r.GET("/api/admin/db/dump", h.DatabaseDump)
	// Unified v2 ingest endpoint: deviceType is a route parameter.
	// 統一 v2 資料接收端點：設備類型使用路由參數。
	r.POST("/sensor/data/v2/:deviceType", h.IngestV2)
	// Fallback response for undefined routes.
	// 未定義路由的統一回應。
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "route not found in backend_go")
	})

	return r
}
