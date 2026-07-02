package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend_go/internal/model"
	"backend_go/internal/repository"
	"backend_go/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Package handler implements HTTP ingest pipeline orchestration.
// handler 套件負責 HTTP 接收管線編排。

// errQueueBackpressure is returned when queue offer times out under load.
// errQueueBackpressure 表示佇列在高負載下入列逾時。
var errQueueBackpressure = errors.New("queue backpressure")

// SensorHandler owns the full ingest pipeline:
// SensorHandler 負責完整資料接收管線：
// HTTP validation -> fragmentation reassembly -> dedup -> shard queue -> batch persistence.
// HTTP 驗證 -> 碎片重組 -> 去重 -> 分片佇列 -> 批次落庫。
type SensorHandler struct {
	cfg  model.AppConfig
	repo repository.SensorRepository
	log  *zap.Logger

	buffers      map[model.DeviceType]*ipBufferStore
	sensorQueues map[model.DeviceType][]chan model.SensorWriteJob
	rawLogQueues map[model.DeviceType][]chan model.RawLogInsertRow

	requestDedup *ttlDedupCache
	eventDedup   *ttlDedupCache
	realtime     *wardRealtimeHub
}

// NewSensorHandler initializes in-memory stores and dedup caches.
// NewSensorHandler 初始化記憶體緩衝與去重快取。
func NewSensorHandler(cfg model.AppConfig, repo repository.SensorRepository, log *zap.Logger) *SensorHandler {
	return &SensorHandler{
		cfg:  cfg,
		repo: repo,
		log:  log,
		buffers: map[model.DeviceType]*ipBufferStore{
			model.DeviceTypeESP32: newIPBufferStore(),
			model.DeviceTypeSTM32: newIPBufferStore(),
		},
		sensorQueues: make(map[model.DeviceType][]chan model.SensorWriteJob),
		rawLogQueues: make(map[model.DeviceType][]chan model.RawLogInsertRow),
		requestDedup: newTTLDedupCache(cfg.RequestDedupTTL, cfg.RequestDedupMaxKeys),
		eventDedup:   newTTLDedupCache(cfg.EventDedupTTL, cfg.EventDedupMaxKeys),
		realtime:     newWardRealtimeHub(),
	}
}

// StartWorkers starts per-deviceType worker pools for sensor data and raw logs.
// StartWorkers 針對每個設備類型啟動 sensor/raw 的 worker pool。
// Sensor queue sharding is done by sensorNumber; raw queue sharding uses client IP.
// sensor 佇列以 sensorNumber 分片；raw 佇列以 client IP 分片。
func (h *SensorHandler) StartWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for dt := range model.DeviceTypeConfigs {
		sensorChans := make([]chan model.SensorWriteJob, 0, h.cfg.SensorWorkers)
		for i := 0; i < h.cfg.SensorWorkers; i++ {
			ch := make(chan model.SensorWriteJob, h.cfg.QueueSize)
			sb := &sensorBatcher{
				deviceType: dt,
				repo:       h.repo,
				log:        h.log,
				ch:         ch,
				batchSize:  h.cfg.SensorBatchSize,
				batchEvery: h.cfg.BatchInterval,
				cfg:        h.cfg,
				realtime:   h.realtime,
			}
			sb.start(ctx, wg)
			sensorChans = append(sensorChans, ch)
		}
		h.sensorQueues[dt] = sensorChans

		rawChans := make([]chan model.RawLogInsertRow, 0, h.cfg.RawWorkers)
		for i := 0; i < h.cfg.RawWorkers; i++ {
			ch := make(chan model.RawLogInsertRow, h.cfg.QueueSize)
			rb := &rawLogBatcher{
				deviceType: dt,
				repo:       h.repo,
				log:        h.log,
				ch:         ch,
				batchSize:  h.cfg.RawBatchSize,
				batchEvery: h.cfg.BatchInterval,
			}
			rb.start(ctx, wg)
			rawChans = append(rawChans, ch)
		}
		h.rawLogQueues[dt] = rawChans
	}
}

// WardFloorStream pushes latest sensor data updates for one floor over SSE.
// WardFloorStream 透過 SSE 推送單一樓層最新感測資料。
func (h *SensorHandler) WardFloorStream(c *gin.Context) {
	floor, err := strconv.Atoi(strings.TrimSpace(c.Param("floor")))
	if err != nil || floor <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid floor")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	id, ch := h.realtime.subscribe(floor)
	defer h.realtime.unsubscribe(floor, id)

	c.SSEvent("ready", gin.H{
		"floor": floor,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent("sensor_data", event)
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-heartbeat.C:
			c.SSEvent("heartbeat", gin.H{
				"time": time.Now().UTC().Format(time.RFC3339),
			})
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// Healthz returns service liveness information.
// Healthz 回傳服務存活狀態。
func (h *SensorHandler) Healthz(c *gin.Context) {
	response.JSON(c, http.StatusOK, gin.H{
		"ok":   true,
		"app":  "backend_go",
		"time": time.Now().UTC().Format(time.RFC3339),
	})
}

// WardFloors returns selectable ward floors for the frontend.
// WardFloors 回傳前端可選擇的病房樓層。
func (h *SensorHandler) WardFloors(c *gin.Context) {
	floors, err := h.repo.ListWardFloors(c.Request.Context())
	if err != nil {
		h.log.Error("list ward floors failed", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "list ward floors failed")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    floors,
	})
}

// WardFloorOverview returns the latest bed snapshot for the frontend floor plan.
// WardFloorOverview 回傳前端平面圖使用的最新床位快照。
func (h *SensorHandler) WardFloorOverview(c *gin.Context) {
	floor, err := strconv.Atoi(strings.TrimSpace(c.Param("floor")))
	if err != nil || floor <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid floor")
		return
	}

	overview, err := h.repo.GetWardFloorOverview(c.Request.Context(), floor)
	if err != nil {
		h.log.Error("get ward floor overview failed", zap.Int("floor", floor), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "get ward floor overview failed")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// SensorHistory returns historical chart points for one sensor number.
// SensorHistory 回傳單一 sensor number 的歷史圖表資料。
func (h *SensorHandler) SensorHistory(c *gin.Context) {
	sensorNumber := strings.TrimSpace(c.Param("sensorNumber"))
	if sensorNumber == "" {
		response.Error(c, http.StatusBadRequest, "sensorNumber is required")
		return
	}

	start, err := parseOptionalTimeQuery(c.Query("start"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid start")
		return
	}
	end, err := parseOptionalTimeQuery(c.Query("end"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid end")
		return
	}
	history, err := h.repo.GetSensorHistory(c.Request.Context(), sensorNumber, start, end, 0)
	if err != nil {
		h.log.Error("get sensor history failed", zap.String("sensor_number", sensorNumber), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "get sensor history failed")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// SensorThermalTimeline returns lightweight thermal frame references for slider use.
// SensorThermalTimeline 回傳輕量熱像時間軸，避免一次載入所有 frame 影像資料。
func (h *SensorHandler) SensorThermalTimeline(c *gin.Context) {
	sensorNumber := strings.TrimSpace(c.Param("sensorNumber"))
	if sensorNumber == "" {
		response.Error(c, http.StatusBadRequest, "sensorNumber is required")
		return
	}

	start, err := parseOptionalTimeQuery(c.Query("start"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid start")
		return
	}
	end, err := parseOptionalTimeQuery(c.Query("end"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid end")
		return
	}

	timeline, err := h.repo.GetSensorThermalTimeline(c.Request.Context(), sensorNumber, start, end)
	if err != nil {
		h.log.Error("get sensor thermal timeline failed", zap.String("sensor_number", sensorNumber), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "get sensor thermal timeline failed")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    timeline,
	})
}

// SensorThermalFrame returns a single thermal frame by data_id.
// SensorThermalFrame 依 data_id 回傳單張熱像 frame。
func (h *SensorHandler) SensorThermalFrame(c *gin.Context) {
	sensorNumber := strings.TrimSpace(c.Param("sensorNumber"))
	if sensorNumber == "" {
		response.Error(c, http.StatusBadRequest, "sensorNumber is required")
		return
	}
	dataID, err := strconv.ParseInt(strings.TrimSpace(c.Param("dataID")), 10, 64)
	if err != nil || dataID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid dataID")
		return
	}

	frame, err := h.repo.GetSensorThermalFrame(c.Request.Context(), sensorNumber, dataID)
	if err != nil {
		h.log.Error("get sensor thermal frame failed", zap.String("sensor_number", sensorNumber), zap.Int64("data_id", dataID), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "get sensor thermal frame failed")
		return
	}
	if frame == nil {
		response.Error(c, http.StatusNotFound, "thermal frame not found")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    frame,
	})
}

// SensorLatestThermalFrame returns the latest thermal frame for one sensor.
// SensorLatestThermalFrame 回傳單一 sensor 最新熱像 frame。
func (h *SensorHandler) SensorLatestThermalFrame(c *gin.Context) {
	sensorNumber := strings.TrimSpace(c.Param("sensorNumber"))
	if sensorNumber == "" {
		response.Error(c, http.StatusBadRequest, "sensorNumber is required")
		return
	}

	frame, err := h.repo.GetSensorLatestThermalFrame(c.Request.Context(), sensorNumber)
	if err != nil {
		h.log.Error("get latest sensor thermal frame failed", zap.String("sensor_number", sensorNumber), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "get latest sensor thermal frame failed")
		return
	}
	if frame == nil {
		response.Error(c, http.StatusNotFound, "thermal frame not found")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"success": true,
		"data":    frame,
	})
}

// DatabaseDump streams a PostgreSQL custom-format dump for one-off remote backups.
// DatabaseDump 串流 PostgreSQL custom-format dump，用於一次性遠端備份下載。
func (h *SensorHandler) DatabaseDump(c *gin.Context) {
	if strings.TrimSpace(h.cfg.DBDumpToken) == "" {
		response.Error(c, http.StatusNotFound, "route not found in backend_go")
		return
	}
	if c.GetHeader("X-Dump-Token") != h.cfg.DBDumpToken {
		response.Error(c, http.StatusForbidden, "invalid dump token")
		return
	}

	cmd := exec.CommandContext(
		c.Request.Context(),
		"pg_dump",
		"-h", h.cfg.PGHost,
		"-p", h.cfg.PGPort,
		"-U", h.cfg.PGUser,
		"-d", h.cfg.PGDBName,
		"-Fc",
	)
	cmd.Env = append([]string{}, "PGPASSWORD="+h.cfg.PGPassword)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		h.log.Error("create pg_dump stdout pipe failed", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "create dump stream failed")
		return
	}
	if err := cmd.Start(); err != nil {
		h.log.Error("start pg_dump failed", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "start dump failed")
		return
	}

	filename := fmt.Sprintf("%s_%s.dump", h.cfg.PGDBName, time.Now().UTC().Format("20060102T150405Z"))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, stdout); err != nil {
		h.log.Error("stream pg_dump failed", zap.Error(err))
	}
	if err := cmd.Wait(); err != nil {
		h.log.Error("pg_dump failed", zap.Error(err), zap.String("stderr", stderr.String()))
	}
}

// IngestV2 handles unified v2 ingest endpoint with dynamic deviceType parameter.
// IngestV2 處理統一 v2 上拋端點，設備類型由動態參數決定。
func (h *SensorHandler) IngestV2(c *gin.Context) {
	deviceTypeRaw := strings.TrimSpace(strings.ToLower(c.Param("deviceType")))
	dt, ok := parseDeviceType(deviceTypeRaw)
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid deviceType, use esp32 or stm32")
		return
	}
	h.handleV2(c, dt)
}

// handleV2 validates HTTP payload and maps pipeline errors to API responses.
// handleV2 驗證 HTTP payload，並將管線錯誤映射為 API 回應。
func (h *SensorHandler) handleV2(c *gin.Context, dt model.DeviceType) {
	cfg := model.DeviceTypeConfigs[dt]

	// Read request body with hard upper bound to prevent oversized payload abuse.
	// 讀取 request body 並限制上限，避免過大 payload 壓垮服務。
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, h.cfg.MaxBodyBytes+1))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "read body failed")
		return
	}
	if int64(len(body)) > h.cfg.MaxBodyBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}

	chunk := strings.TrimSpace(string(body))
	if chunk == "" {
		response.Error(c, http.StatusBadRequest, "empty payload")
		return
	}

	// Request-level dedup: suppress immediate retries of the same packet chunk.
	// 請求層級去重：抑制相同封包片段的即時重送。
	clientIP := extractClientIP(c.Request)
	now := time.Now()
	requestKey := buildRequestDedupKey(dt, clientIP, chunk)
	if h.requestDedup.isDuplicate(requestKey, now) {
		response.JSON(c, http.StatusOK, gin.H{
			"success": true,
			"deduped": true,
			"message": fmt.Sprintf("[%s] duplicated retry ignored", cfg.Tag),
		})
		return
	}

	result, err := h.processChunk(c.Request.Context(), dt, clientIP, chunk)
	if err != nil {
		// Map internal pipeline errors to stable API status codes.
		// 將內部管線錯誤映射成穩定的 API 狀態碼。
		if strings.Contains(err.Error(), "buffer overflow") {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("[%s] buffer size limit exceeded", cfg.Tag))
			return
		}
		if errors.Is(err, errQueueBackpressure) {
			c.Header("Retry-After", "1")
			response.Error(c, http.StatusServiceUnavailable, fmt.Sprintf("[%s] queue busy, retry later", cfg.Tag))
			return
		}
		h.log.Error("ingest process failed", zap.String("device_type", string(dt)), zap.Error(err))
		response.Error(c, http.StatusInternalServerError, fmt.Sprintf("[%s] internal server error", cfg.Tag))
		return
	}

	// Mark request key only after processing completes successfully.
	// 僅在處理成功後才標記 request key，避免誤判去重。
	h.requestDedup.mark(requestKey, now)

	if result.Saved > 0 {
		response.JSON(c, http.StatusCreated, gin.H{
			"success": true,
			"saved":   result.Saved,
			"deduped": result.Deduped,
		})
		return
	}
	if result.Deduped > 0 {
		response.JSON(c, http.StatusOK, gin.H{
			"success": true,
			"saved":   0,
			"deduped": result.Deduped,
			"message": fmt.Sprintf("[%s] duplicated events ignored", cfg.Tag),
		})
		return
	}
	response.JSON(c, http.StatusAccepted, gin.H{
		"success": true,
		"message": fmt.Sprintf("[%s] data buffered or processing", cfg.Tag),
	})
}

// processChunk reassembles fragmented JSON streams, runs dedup, and enqueues rows.
// processChunk 負責碎片 JSON 重組、去重與資料入佇列。
func (h *SensorHandler) processChunk(ctx context.Context, dt model.DeviceType, clientIP, chunk string) (model.ChunkProcessResult, error) {
	cfg := model.DeviceTypeConfigs[dt]
	bufStore := h.buffers[dt]
	result := model.ChunkProcessResult{}

	combined, err := bufStore.appendChunk(clientIP, chunk, h.cfg.MaxBufferSize)
	if err != nil {
		return result, err
	}

	// Extract complete JSON objects and keep unfinished tail for next request.
	// 擷取完整 JSON 物件，未完成尾段留待下次請求接續。
	jsonObjects, remainder := extractJSONObjects(combined)
	bufStore.setRemainder(clientIP, remainder)

	var lastProcessedContent any
	var lastSensorNumber string
	malformedCount := 0
	for _, obj := range jsonObjects {
		payload := map[string]any{}
		decoder := json.NewDecoder(strings.NewReader(obj))
		decoder.UseNumber()
		if err := decoder.Decode(&payload); err != nil {
			// Skip malformed object but continue processing other objects in stream.
			// 略過單筆格式錯誤物件，不中斷整包其餘資料處理。
			malformedCount++
			continue
		}

		sensorNumber := firstString(payload, "sensor_number", "sensorNo")
		if sensorNumber == "" {
			continue
		}
		// Persist one representative sensor_number in raw log for quick traceability.
		// 在 raw log 保留代表性的 sensor_number，方便快速追查來源。
		lastSensorNumber = sensorNumber

		// Event-level dedup protects against retried partial writes.
		// 事件層級去重可避免部分成功後重送造成重複寫入。
		eventKey := buildEventDedupKey(dt, sensorNumber, payload, obj)
		now := time.Now()
		if h.eventDedup.isDuplicate(eventKey, now) {
			result.Deduped++
			continue
		}

		row, ok, _ := h.buildSensorRow(ctx, payload, sensorNumber, dt)
		if !ok {
			continue
		}
		row.DeviceType = dt

		queues := h.sensorQueues[dt]
		if len(queues) == 0 {
			return result, fmt.Errorf("[%s] no sensor workers configured", cfg.Tag)
		}

		// Shard by sensorNumber so the same sensor keeps processing order.
		// 以 sensorNumber 分片，確保同一 sensor 的處理順序一致。
		job := model.SensorWriteJob{
			SensorRow: row,
			RawLogRow: model.RawLogInsertRow{
				SensorNumber:        sensorNumber,
				ClientIP:            clientIP,
				RawContent:          chunk,
				ProcessedContent:    obj,
				ProcessedStatus:     model.RawLogStatusProcessedSaved,
				SensorDataID:        nil,
				SensorDataTimestamp: nil,
			},
		}

		queue := queues[shardIndex(sensorNumber, len(queues))]
		if !offerToQueue(ctx, queue, job, h.cfg.QueueOfferTimout) {
			h.log.Warn("sensor queue busy",
				zap.String("device_type", string(dt)),
				zap.String("ip", clientIP),
				zap.String("sensor_number", sensorNumber),
			)
			return result, errQueueBackpressure
		}
		h.eventDedup.mark(eventKey, now)
		result.Saved++
	}

	if result.Saved > 0 {
		return result, nil
	}

	processedStatus := model.RawLogStatusRawOnly
	if result.Deduped > 0 {
		processedStatus = model.RawLogStatusDuplicate
	} else if malformedCount > 0 || (len(jsonObjects) > 0 && remainder == "") {
		processedStatus = model.RawLogStatusMalformed
	}

	raw := model.RawLogInsertRow{
		SensorNumber:     lastSensorNumber,
		ClientIP:         clientIP,
		RawContent:       chunk,
		ProcessedContent: lastProcessedContent,
		ProcessedStatus:  processedStatus,
	}
	rawQueues := h.rawLogQueues[dt]
	if len(rawQueues) == 0 {
		return result, fmt.Errorf("[%s] no raw log workers configured", cfg.Tag)
	}
	rawQueue := rawQueues[shardIndex(clientIP, len(rawQueues))]
	if !offerToQueue(ctx, rawQueue, raw, h.cfg.QueueOfferTimout) {
		h.log.Warn("raw log queue busy", zap.String("device_type", string(dt)), zap.String("ip", clientIP))
		return result, errQueueBackpressure
	}

	return result, nil
}

// buildSensorRow normalizes input payload into DB row format.
// buildSensorRow 將輸入 payload 標準化為 DB 列格式。
// Return values: row, ok, unknownSensor.
// 回傳值依序為：row、ok、unknownSensor。
func (h *SensorHandler) buildSensorRow(ctx context.Context, payload map[string]any, sensorNumber string, deviceType model.DeviceType) (model.SensorInsertRow, bool, bool) {
	// Resolve sensor_id from canonical sensors table.
	// 先從 sensors 主檔解析對應 sensor_id。
	sensorID, ok, err := h.repo.LookupSensorID(ctx, sensorNumber, deviceType)
	if err != nil || !ok {
		return model.SensorInsertRow{}, false, err == nil && !ok
	}

	heartRate, okHeart := firstNumber(payload, "heart_rate", "m_heart_rate")
	breathRate, okBreath := firstNumber(payload, "breath_rate", "m_breath_rate")
	rhythm := firstString(payload, "rhythm", "heart_rhythm")
	if !okHeart {
		heartRate = -1
	}
	if !okBreath {
		breathRate = -1
	}

	breathRateState, _ := firstNumber(payload, "m_breath_rate_state", "breath_rate_state")
	outOfRange, _ := firstNumber(payload, "m_outOfRange", "out_of_range")
	anglesFirst, _ := firstNumber(payload, "m_angles_first", "angles_first")
	anglesSecond, _ := firstNumber(payload, "m_angles_second", "angles_second")
	movementState, _ := firstNumber(payload, "m_movementState", "movement_state")
	movementLevel, _ := firstNumber(payload, "m_movementLevel", "movement_level")
	breathRateLastTransmit, _ := firstNumber(payload, "m_breath_rate_last_Transmit", "breath_rate_last_transmit")
	heartRateLastTransmit, _ := firstNumber(payload, "m_heart_rate_last_Transmit", "heart_rate_last_transmit")

	distance, okDistance := firstNumber(payload, "distance", "m_distance")
	thermistor, okThermistor := firstNumber(payload, "thermistor")

	temps := parseTemperatureArray(payload["temperature"])
	temperatureJSONBytes, _ := json.Marshal(temps)
	temperatureJSON := string(temperatureJSONBytes)
	highTempVal, hasHighTemp := maxFloat(temps)
	hasThermalFrame := len(temps) > 1 && hasHighTemp

	row := model.SensorInsertRow{
		SensorID:               sensorID,
		BreathRate:             int(breathRate),
		BreathRateState:        int(breathRateState),
		HeartRate:              int(heartRate),
		Rhythm:                 rhythm,
		OutOfRange:             int(outOfRange),
		AnglesFirst:            int(anglesFirst),
		AnglesSecond:           int(anglesSecond),
		MovementState:          int(movementState),
		MovementLevel:          int(movementLevel),
		BreathRateLastTransmit: int64(breathRateLastTransmit),
		HeartRateLastTransmit:  int64(heartRateLastTransmit),
		Distance:               nil,
		Thermistor:             nil,
		HighTemperature:        nil,
		ThermalFrameJSON:       temperatureJSON,
		HasThermalFrame:        hasThermalFrame,
	}

	if okDistance {
		row.Distance = distance
	}
	if okThermistor {
		row.Thermistor = thermistor
	}
	if hasHighTemp {
		row.HighTemperature = highTempVal
	} else {
		row.HighTemperature = 0
	}
	return row, true, false
}

// ipBufferStore keeps per-source chunk fragments for reassembly.
// ipBufferStore 依來源維護碎片緩衝，用於重組封包。
type ipBufferStore struct {
	mu     sync.Mutex
	buffer map[string]string
}

// newIPBufferStore creates fragment store for one deviceType.
// newIPBufferStore 建立單一平台的碎片緩衝儲存。
func newIPBufferStore() *ipBufferStore {
	return &ipBufferStore{buffer: make(map[string]string)}
}

// appendChunk merges a new chunk with the existing remainder for a key.
// appendChunk 將新 chunk 與既有 remainder 合併。
func (s *ipBufferStore) appendChunk(key, chunk string, maxSize int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.buffer[key]
	if existing != "" && strings.HasPrefix(strings.TrimSpace(chunk), "{") {
		candidate := existing + chunk
		if !json.Valid([]byte(candidate)) {
			existing = ""
		}
	}

	combined := existing + chunk
	if len(combined) > maxSize {
		delete(s.buffer, key)
		return "", fmt.Errorf("buffer overflow for key=%s", key)
	}
	return combined, nil
}

// setRemainder stores unfinished JSON tail for next request.
// setRemainder 儲存未完成的 JSON 尾段供下次請求續接。
func (s *ipBufferStore) setRemainder(key, remainder string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if remainder == "" {
		delete(s.buffer, key)
		return
	}
	s.buffer[key] = remainder
}

// ttlDedupCache is a small bounded in-memory TTL map for dedup keys.
// ttlDedupCache 是有上限的記憶體 TTL 去重快取。
type ttlDedupCache struct {
	ttl     time.Duration
	maxKeys int
	mu      sync.Mutex
	items   map[string]time.Time
}

// newTTLDedupCache returns nil when TTL is disabled (<=0).
// newTTLDedupCache 在 TTL <= 0 時回傳 nil（代表停用）。
func newTTLDedupCache(ttl time.Duration, maxKeys int) *ttlDedupCache {
	if ttl <= 0 {
		return nil
	}
	if maxKeys <= 0 {
		maxKeys = 100000
	}
	return &ttlDedupCache{
		ttl:     ttl,
		maxKeys: maxKeys,
		items:   make(map[string]time.Time),
	}
}

// isDuplicate returns true when key exists and is still valid.
// isDuplicate 在 key 存在且尚未過期時回傳 true。
func (c *ttlDedupCache) isDuplicate(key string, now time.Time) bool {
	if c == nil || key == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if expireAt, ok := c.items[key]; ok {
		if now.Before(expireAt) {
			return true
		}
		delete(c.items, key)
	}
	return false
}

// mark inserts/refreshes key and performs opportunistic cleanup if oversized.
// mark 新增/更新 key，並在超過上限時執行機會式清理。
func (c *ttlDedupCache) mark(key string, now time.Time) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = now.Add(c.ttl)
	if len(c.items) <= c.maxKeys {
		return
	}

	for k, expireAt := range c.items {
		if !now.Before(expireAt) {
			delete(c.items, k)
		}
	}
	if len(c.items) <= c.maxKeys {
		return
	}

	extra := len(c.items) - c.maxKeys
	for k := range c.items {
		delete(c.items, k)
		extra--
		if extra <= 0 {
			break
		}
	}
}

// sensorBatcher accumulates sensor rows and flushes in batched INSERTs.
// sensorBatcher 聚合 sensor 列並以批次 INSERT 落庫。
type sensorBatcher struct {
	deviceType model.DeviceType
	repo       repository.SensorRepository
	log        *zap.Logger
	ch         chan model.SensorWriteJob
	batchSize  int
	batchEvery time.Duration
	cfg        model.AppConfig
	realtime   *wardRealtimeHub
}

// start launches one sensor batch worker.
// start 啟動一個 sensor 批次 worker。
func (b *sensorBatcher) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(b.batchEvery)
		defer ticker.Stop()

		batch := make([]model.SensorWriteJob, 0, b.batchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			rows := make([]model.SensorInsertRow, 0, len(batch))
			for _, job := range batch {
				rows = append(rows, job.SensorRow)
			}
			// Persist current batch in one DB round-trip for throughput.
			// 以單次 DB 往返寫入整批資料以提升吞吐量。
			refs, err := b.repo.InsertSensorBatch(ctx, b.deviceType, rows)
			if err != nil {
				b.log.Error("sensor batch flush error",
					zap.String("device_type", string(b.deviceType)),
					zap.Error(err),
				)
				batch = batch[:0]
				return
			}
			rawRows := make([]model.RawLogInsertRow, 0, len(batch))
			thermalRows := make([]model.ThermalFrameInsertRow, 0, len(batch))
			for i, job := range batch {
				job.RawLogRow.SensorDataID = refs[i].DataID
				job.RawLogRow.SensorDataTimestamp = refs[i].Timestamp
				rawRows = append(rawRows, job.RawLogRow)
				if shouldPersistThermalFrame(job.SensorRow, refs[i], b.cfg.ThermalFrameInterval) {
					thermalRows = append(thermalRows, model.ThermalFrameInsertRow{
						SensorID:        job.SensorRow.SensorID,
						SensorDataID:    refs[i].DataID,
						Timestamp:       refs[i].Timestamp,
						HighTemperature: job.SensorRow.HighTemperature,
						FrameJSON:       job.SensorRow.ThermalFrameJSON,
					})
				}
				b.publishRealtime(job, refs[i])
			}
			if len(thermalRows) > 0 {
				if err := b.repo.InsertThermalFrameBatch(ctx, thermalRows); err != nil {
					b.log.Error("thermal frame batch flush error",
						zap.String("device_type", string(b.deviceType)),
						zap.Error(err),
					)
				}
			}
			if err := b.repo.InsertRawLogBatch(ctx, b.deviceType, rawRows); err != nil {
				b.log.Error("sensor raw-log trace flush error",
					zap.String("device_type", string(b.deviceType)),
					zap.Error(err),
				)
			}
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case row := <-b.ch:
				batch = append(batch, row)
				if len(batch) >= b.batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

func shouldPersistThermalFrame(row model.SensorInsertRow, ref model.SensorDataWriteRef, interval time.Duration) bool {
	if !row.HasThermalFrame || strings.TrimSpace(row.ThermalFrameJSON) == "" {
		return false
	}
	if interval <= 0 {
		return true
	}
	seconds := int64(interval / time.Second)
	if seconds <= 0 {
		return true
	}
	return ref.Timestamp.Unix()%seconds == 0
}

func (b *sensorBatcher) publishRealtime(job model.SensorWriteJob, ref model.SensorDataWriteRef) {
	if b.realtime == nil {
		return
	}
	floor, ok := floorFromSensorNumber(job.RawLogRow.SensorNumber)
	if !ok {
		return
	}
	temp := 0.0
	if n, ok := numberFromAny(job.SensorRow.HighTemperature); ok {
		temp = n
	}
	rhythm := strings.TrimSpace(job.SensorRow.Rhythm)
	if rhythm == "" {
		rhythm = estimateRhythm(job.SensorRow.HeartRate)
	}
	bpSys, bpDia := estimateBloodPressure(job.SensorRow.HeartRate, temp)
	b.realtime.publish(model.WardSensorRealtimeEvent{
		Floor:        floor,
		SensorNumber: job.RawLogRow.SensorNumber,
		DeviceType:   string(job.SensorRow.DeviceType),
		DeviceOnline: true,
		Presence:     true,
		Latest: model.WardSensorDataPoint{
			DataID:          ref.DataID,
			Timestamp:       ref.Timestamp,
			HeartRate:       job.SensorRow.HeartRate,
			BreathRate:      job.SensorRow.BreathRate,
			Temperature:     temp,
			HighTemperature: temp,
			BPSys:           bpSys,
			BPDia:           bpDia,
			Rhythm:          rhythm,
			OutOfRange:      job.SensorRow.OutOfRange,
			Presence:        true,
		},
	})
}

// rawLogBatcher accumulates raw log rows and flushes in batches.
// rawLogBatcher 聚合 raw log 列並批次寫入。
type rawLogBatcher struct {
	deviceType model.DeviceType
	repo       repository.SensorRepository
	log        *zap.Logger
	ch         chan model.RawLogInsertRow
	batchSize  int
	batchEvery time.Duration
}

// start launches one raw log batch worker.
// start 啟動一個 raw log 批次 worker。
func (b *rawLogBatcher) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(b.batchEvery)
		defer ticker.Stop()

		batch := make([]model.RawLogInsertRow, 0, b.batchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			// Persist raw logs in batch to reduce per-row overhead.
			// 以批次方式寫入 raw logs，降低逐筆寫入成本。
			if err := b.repo.InsertRawLogBatch(ctx, b.deviceType, batch); err != nil {
				b.log.Error("raw log batch flush error",
					zap.String("device_type", string(b.deviceType)),
					zap.Error(err),
				)
			}
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case row := <-b.ch:
				batch = append(batch, row)
				if len(batch) >= b.batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

// buildRequestDedupKey dedups immediate whole-request retries.
// buildRequestDedupKey 用於整包請求重試的即時去重。
func buildRequestDedupKey(dt model.DeviceType, clientIP, chunk string) string {
	hash := hashString(chunk)
	return fmt.Sprintf("%s|%s|%s", dt, clientIP, hash)
}

// buildEventDedupKey dedups per-event retries when payload contains time/sequence markers.
// buildEventDedupKey 在 payload 含時間/序號欄位時做事件級去重。
func buildEventDedupKey(dt model.DeviceType, sensorNumber string, payload map[string]any, rawObject string) string {
	ts := firstToken(payload, "timestamp", "time", "event_time", "packet_time", "device_time", "created_at", "ts", "m_time")
	seq := firstToken(payload, "sequence", "seq", "packet_id", "msg_id", "frame_id", "counter")
	if ts == "" && seq == "" {
		return ""
	}
	if ts == "" {
		ts = "-"
	}
	if seq == "" {
		seq = hashString(rawObject)
	}
	return fmt.Sprintf("%s|%s|%s|%s", dt, sensorNumber, ts, seq)
}

// hashString produces a stable compact hash for dedup keys.
// hashString 產生穩定且短小的 hash 字串供去重鍵使用。
func hashString(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
}

// shardIndex picks queue index by deterministic hash.
// shardIndex 以決定性 hash 選擇對應佇列索引。
func shardIndex(key string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(shardCount))
}

// offerToQueue attempts enqueue within timeout; used for backpressure.
// offerToQueue 在 timeout 內嘗試入列，作為背壓控制基礎。
func offerToQueue[T any](ctx context.Context, ch chan T, value T, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ch <- value:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

// extractJSONObjects scans concatenated/fragmented stream text and extracts complete JSON objects.
// extractJSONObjects 掃描串接/碎片化字串並擷取完整 JSON 物件。
func extractJSONObjects(input string) ([]string, string) {
	if input == "" {
		return nil, ""
	}

	start := -1
	depth := 0
	inString := false
	escaped := false
	out := make([]string, 0, 4)

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if start == -1 {
			if ch == '{' {
				start = i
				depth = 1
				inString = false
				escaped = false
			}
			continue
		}

		if inString {
			// Ignore braces inside quoted strings.
			// 忽略字串內的大括號，避免誤判 JSON 邊界。
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			if depth > 0 && isLikelyCorruptedObjectRestart(input, i) {
				start = i
				depth = 1
				inString = false
				escaped = false
				continue
			}
			depth++
		case '}':
			depth--
			if depth == 0 {
				out = append(out, input[start:i+1])
				start = -1
			}
		}
	}

	if start != -1 {
		return out, input[start:]
	}
	return out, ""
}

// isLikelyCorruptedObjectRestart detects a fresh JSON object embedded after invalid bytes.
// isLikelyCorruptedObjectRestart 偵測壞資料中重新開始的新 JSON 物件。
func isLikelyCorruptedObjectRestart(input string, openBraceIndex int) bool {
	for i := openBraceIndex - 1; i >= 0; i-- {
		switch input[i] {
		case ' ', '\n', '\r', '\t':
			continue
		case ':', '[', ',':
			return false
		default:
			return true
		}
	}
	return false
}

// parseTemperatureArray converts generic JSON array into float64 values.
// parseTemperatureArray 將通用 JSON 陣列轉為 float64 列表。
func parseTemperatureArray(v any) []float64 {
	if n, ok := numberFromAny(v); ok {
		return []float64{n}
	}
	arr, ok := v.([]any)
	if !ok {
		return []float64{}
	}
	out := make([]float64, 0, len(arr))
	for _, item := range arr {
		if n, ok := numberFromAny(item); ok {
			out = append(out, n)
		}
	}
	return out
}

func firstString(payload map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := payload[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case json.Number:
				return strings.TrimSpace(t.String())
			}
		}
	}
	return ""
}

// maxFloat returns max value and false for empty input.
// maxFloat 回傳最大值；若輸入為空則回傳 false。
func maxFloat(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	maxV := -math.MaxFloat64
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	return maxV, true
}

// firstToken returns normalized string token for dedup keys (supports number/string types).
// firstToken 回傳去重用標準化字串 token（支援數值與字串）。
func firstToken(payload map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := payload[k]; ok {
			switch t := v.(type) {
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					return s
				}
			case float64:
				return strconv.FormatFloat(t, 'f', -1, 64)
			case int:
				return strconv.Itoa(t)
			case int64:
				return strconv.FormatInt(t, 10)
			case json.Number:
				return t.String()
			}
		}
	}
	return ""
}

// firstNumber returns first numeric field among candidate keys.
// firstNumber 從候選鍵中取第一個可解析數值。
func firstNumber(payload map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := payload[k]; ok {
			if n, ok := numberFromAny(v); ok {
				return n, true
			}
		}
	}
	return 0, false
}

// numberFromAny parses common numeric JSON representations.
// numberFromAny 解析常見 JSON 數值表示型別。
func numberFromAny(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case int32:
		return float64(t), true
	case json.Number:
		n, err := t.Float64()
		if err == nil {
			return n, true
		}
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func floorFromSensorNumber(sensorNumber string) (int, bool) {
	sensorNumber = strings.TrimSpace(sensorNumber)
	if len(sensorNumber) < 2 {
		return 0, false
	}
	floor, err := strconv.Atoi(sensorNumber[:2])
	if err != nil || floor <= 0 {
		return 0, false
	}
	return floor, true
}

func estimateRhythm(heartRate int) string {
	switch {
	case heartRate <= 0:
		return "未知"
	case heartRate >= 110:
		return "心搏過速"
	case heartRate <= 50:
		return "心搏過緩"
	default:
		return "竇性心律"
	}
}

func estimateBloodPressure(heartRate int, temperature float64) (int, int) {
	if heartRate <= 0 {
		heartRate = 75
	}
	if temperature <= 0 {
		temperature = 36.8
	}
	shift := float64(heartRate-75)*0.35 + (temperature-36.8)*8
	sys := clampInt(int(112+shift), 85, 190)
	dia := clampInt(int(72+shift*0.5), 50, 120)
	if sys < dia+15 {
		sys = dia + 15
	}
	return sys, dia
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func parseOptionalTimeQuery(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return &t, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// extractClientIP resolves caller IP from X-Forwarded-For or remote address.
// extractClientIP 優先從 X-Forwarded-For，否則使用 remote address 解析來源 IP。
func extractClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// parseDeviceType validates and normalizes route deviceType parameter.
// parseDeviceType 驗證並正規化路由中的平台參數。
func parseDeviceType(raw string) (model.DeviceType, bool) {
	switch raw {
	case string(model.DeviceTypeESP32):
		return model.DeviceTypeESP32, true
	case string(model.DeviceTypeSTM32):
		return model.DeviceTypeSTM32, true
	default:
		return "", false
	}
}
