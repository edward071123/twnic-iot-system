package model

import "time"

// Package model defines shared ingest data structures.
// model 套件定義共用的接收資料結構。

// DeviceType differentiates sensor source protocol families.
// DeviceType 用來區分不同感測器來源協定。
type DeviceType string

const (
	// DeviceTypeESP32 identifies ESP32 source payloads.
	// DeviceTypeESP32 標記 ESP32 來源 payload。
	DeviceTypeESP32 DeviceType = "esp32"
	// DeviceTypeSTM32 identifies STM32 source payloads.
	// DeviceTypeSTM32 標記 STM32 來源 payload。
	DeviceTypeSTM32 DeviceType = "stm32"
)

// DeviceTypeConfig maps each device type to its target storage tables and log tag.
// DeviceTypeConfig 定義平台對應的資料表與日誌標籤。
type DeviceTypeConfig struct {
	DataTable string
	Tag       string
}

// DeviceTypeConfigs defines storage mapping for all supported ingest device types.
// DeviceTypeConfigs 定義所有支援接收設備類型的儲存映射。
var DeviceTypeConfigs = map[DeviceType]DeviceTypeConfig{
	DeviceTypeESP32: {
		DataTable: "sensor_datas",
		Tag:       "ESP32",
	},
	DeviceTypeSTM32: {
		DataTable: "sensor_datas",
		Tag:       "STM32",
	},
}

// SensorInsertRow is the normalized row shape for sensor_datas table.
// SensorInsertRow 是寫入 sensor_datas 的標準化列結構。
type SensorInsertRow struct {
	DeviceType             DeviceType
	SensorID               int
	BreathRate             int
	BreathRateState        int
	HeartRate              int
	Rhythm                 string
	OutOfRange             int
	AnglesFirst            int
	AnglesSecond           int
	MovementState          int
	MovementLevel          int
	BreathRateLastTransmit int64
	HeartRateLastTransmit  int64
	Distance               any
	Thermistor             any
	HighTemperature        any
	ThermalFrameJSON       string
	HasThermalFrame        bool
}

// ThermalFrameInsertRow is the normalized row shape for sensor_thermal_frames.
// ThermalFrameInsertRow 是寫入 sensor_thermal_frames 的標準化列結構。
type ThermalFrameInsertRow struct {
	SensorID        int
	SensorDataID    int64
	Timestamp       time.Time
	HighTemperature any
	FrameJSON       string
}

// RawLogInsertRow is the raw packet log shape for unified sensor_raw_logs table.
// RawLogInsertRow 是寫入整合後 sensor_raw_logs 的原始封包日誌結構。
type RawLogInsertRow struct {
	SensorNumber        string
	ClientIP            string
	RawContent          string
	ProcessedContent    any
	ProcessedStatus     int
	SensorDataID        any
	SensorDataTimestamp any
}

// SensorDataWriteRef identifies the sensor_datas row created by an insert.
// SensorDataWriteRef 標記寫入 sensor_datas 後產生的資料列。
type SensorDataWriteRef struct {
	DataID    int64
	Timestamp time.Time
}

// SensorWriteJob carries a normalized sensor row plus its raw-log trace row.
// SensorWriteJob 攜帶標準化 sensor 資料與對應 raw-log 追蹤資料。
type SensorWriteJob struct {
	SensorRow SensorInsertRow
	RawLogRow RawLogInsertRow
}

const (
	// RawLogStatusRawOnly means the request chunk was only buffered or not complete yet.
	// RawLogStatusRawOnly 表示 request chunk 僅保留原始資料，尚未形成完整可寫入資料。
	RawLogStatusRawOnly = 0
	// RawLogStatusProcessedSaved means a reassembled JSON was written into sensor_datas.
	// RawLogStatusProcessedSaved 表示重組後 JSON 已寫入 sensor_datas。
	RawLogStatusProcessedSaved = 1
	// RawLogStatusMalformed means the request produced malformed JSON that could not be persisted.
	// RawLogStatusMalformed 表示 request 產生無法寫入的格式錯誤 JSON。
	RawLogStatusMalformed = 2
	// RawLogStatusDuplicate means the reassembled event was recognized as duplicate.
	// RawLogStatusDuplicate 表示重組後事件被判定為重複資料。
	RawLogStatusDuplicate = 3
)

// ChunkProcessResult reports per-request pipeline outcomes.
// ChunkProcessResult 回報單次請求在管線中的處理結果。
type ChunkProcessResult struct {
	Saved   int
	Deduped int
}

// WardOverview is the frontend floor-plan snapshot response.
// WardOverview 是前端平面圖總覽快照回應。
type WardOverview struct {
	Floor       int                `json:"floor"`
	GeneratedAt time.Time          `json:"generated_at"`
	Rooms       []WardOverviewRoom `json:"rooms"`
}

// WardFloorOption is one selectable ward floor for frontend navigation.
// WardFloorOption 是前端可選擇的病房樓層。
type WardFloorOption struct {
	Floor int    `json:"floor"`
	Label string `json:"label"`
}

// WardOverviewRoom groups bed snapshots by room.
// WardOverviewRoom 依房間彙整床位快照。
type WardOverviewRoom struct {
	RoomID     int               `json:"room_id"`
	RoomNumber string            `json:"room_number"`
	Beds       []WardOverviewBed `json:"beds"`
}

// WardOverviewBed describes the latest known state for one visual bed.
// WardOverviewBed 描述單一視覺床位的最新狀態。
type WardOverviewBed struct {
	BedID        string               `json:"bed_id"`
	RoomID       int                  `json:"room_id"`
	RoomNumber   string               `json:"room_number"`
	BedNumber    int                  `json:"bed_number"`
	SensorID     any                  `json:"sensor_id"`
	SensorNumber string               `json:"sensor_number"`
	DeviceType   string               `json:"device_type"`
	DeviceOnline bool                 `json:"device_online"`
	Presence     bool                 `json:"presence"`
	PatientName  string               `json:"patient_name"`
	Latest       *WardSensorDataPoint `json:"latest"`
}

// WardSensorDataPoint is one chartable sensor data point.
// WardSensorDataPoint 是可供圖表使用的單筆感測資料。
type WardSensorDataPoint struct {
	DataID          int64     `json:"data_id"`
	Timestamp       time.Time `json:"timestamp"`
	HeartRate       int       `json:"heart_rate"`
	BreathRate      int       `json:"breath_rate"`
	Temperature     float64   `json:"temperature"`
	HighTemperature float64   `json:"high_temperature"`
	TemperatureJSON []float64 `json:"temperature_json,omitempty"`
	BPSys           int       `json:"bp_sys"`
	BPDia           int       `json:"bp_dia"`
	Rhythm          string    `json:"rhythm"`
	OutOfRange      int       `json:"out_of_range"`
	Presence        bool      `json:"presence"`
}

// WardSensorRealtimeEvent is one pushed latest data update for floor-plan clients.
// WardSensorRealtimeEvent 是推送給平面圖前端的單筆最新資料更新。
type WardSensorRealtimeEvent struct {
	Floor        int                 `json:"floor"`
	SensorNumber string              `json:"sensor_number"`
	DeviceType   string              `json:"device_type"`
	DeviceOnline bool                `json:"device_online"`
	Presence     bool                `json:"presence"`
	Latest       WardSensorDataPoint `json:"latest"`
}

// SensorHistoryResponse is the historical chart data response for one sensor.
// SensorHistoryResponse 是單一 sensor 的歷史圖表資料回應。
type SensorHistoryResponse struct {
	SensorNumber string                `json:"sensor_number"`
	Start        *time.Time            `json:"start,omitempty"`
	End          *time.Time            `json:"end,omitempty"`
	Points       []WardSensorDataPoint `json:"points"`
}

// WardThermalTimelineFrame is one lightweight thermal frame reference.
// WardThermalTimelineFrame 是一筆輕量熱像時間軸參照，不包含 768 點影像資料。
type WardThermalTimelineFrame struct {
	DataID    int64     `json:"data_id"`
	Timestamp time.Time `json:"timestamp"`
}

// WardThermalTimelineResponse is the slider timeline for one sensor.
// WardThermalTimelineResponse 是單一 sensor 的熱像 slider 時間軸。
type WardThermalTimelineResponse struct {
	SensorNumber string                     `json:"sensor_number"`
	Start        *time.Time                 `json:"start,omitempty"`
	End          *time.Time                 `json:"end,omitempty"`
	Frames       []WardThermalTimelineFrame `json:"frames"`
}
