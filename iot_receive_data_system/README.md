# 區網預設URL
http://192.168.10.94:3002

# 環境變數
- 主要使用專案根目錄 `.env`
- 不再使用 `backend/.env`
- 使用 PostgreSQL：`PG_HOST / PG_PORT / PG_USER / PG_PASSWORD / PG_DB_NAME`
- Docker 內部 backend 位址只改 `BACKEND_INTERNAL_BASE_URL`；compose 會轉給 frontend proxy 的 `VITE_PROXY_TARGET`
- 前端呼叫 API 路徑用 `VITE_API_BASE_URL=/api`
- 啟動時預設會自動執行 `golang-migrate`（可用 `GO_MIGRATE_ON_START=false` 關閉）
- `backend_go` 服務目前為 Go API server
- `frontend_vue` 服務目前為 Vue/Vite frontend
- 目前提供單一上拋端點（設備類型為路由參數）：
  - `POST /sensor/data/v2/:deviceType`
  - `:deviceType` 僅允許 `esp32` 或 `stm32`

# backend_go 架構
- `backend_go/cmd/main.go`：程式進入點
- `backend_go/internal/model`：資料模型與設定
- `backend_go/internal/handler`：HTTP handler 與資料處理管線
- `backend_go/internal/repository`：資料庫存取層
- `backend_go/internal/migration`：`golang-migrate` 版本化 migration
- `backend_go/internal/router`：路由設定
- `backend_go/pkg/logger`：Zap logger 封裝
- `backend_go/pkg/response`：統一 API 回傳工具
- `backend_go/docs`：Swagger 文件輸出目錄（保留）

# Go 高併發可調參數（可選）
- `GO_SENSOR_WORKERS`（預設 4，依 `sensorNumber` hash 分片，保序寫入）
- `GO_MIGRATE_ON_START`（預設 true，啟動時執行 migration）
- `GO_PARTITION_MAINTAIN_ON_START`（預設 true，啟動時執行分區與留存維護）
- `GO_RAW_LOG_RETENTION_DAYS`（預設 30，raw log 分區留存天數）
- `GO_RAW_LOG_PARTITION_BACK_DAYS`（預設 1，raw log 向前預建分區天數）
- `GO_RAW_LOG_PARTITION_AHEAD_DAYS`（預設 7，raw log 向後預建分區天數）
- `GO_SENSOR_DATA_PARTITION_BACK_MONTHS`（預設 1，sensor data 向前預建分區月數）
- `GO_SENSOR_DATA_PARTITION_AHEAD_MONTHS`（預設 3，sensor data 向後預建分區月數）
- `GO_SEED_ON_START`（預設 true，啟動時執行基礎資料 seeder）
- `GO_SEED_HOSPITAL_NAME`（預設 `養護中心`）
- `GO_RAW_WORKERS`（預設 2，依 client IP hash 分片）
- `GO_SENSOR_BATCH_SIZE`（預設 100）
- `GO_RAW_BATCH_SIZE`（預設 100）
- `GO_BATCH_INTERVAL_MS`（預設 200）
- `GO_QUEUE_SIZE`（預設 5000）
- `GO_MAX_BUFFER_SIZE`（預設 10240）
- `GO_QUEUE_OFFER_TIMEOUT_MS`（預設 100）
- `GO_SENSOR_CACHE_TTL_SEC`（預設 60）
- `GO_THERMAL_FRAME_INTERVAL_SEC`（預設 10，熱像 frame 降頻寫入秒數；生命徵象仍每秒寫入）
- `GO_REQUEST_DEDUP_TTL_SEC`（預設 8）
- `GO_REQUEST_DEDUP_MAX_KEYS`（預設 200000）
- `GO_EVENT_DEDUP_TTL_SEC`（預設 300）
- `GO_EVENT_DEDUP_MAX_KEYS`（預設 500000）
- `GO_HTTP_LOG_EVERY_REQUEST`（預設 true，記錄每一筆 HTTP request）
- `GO_HTTP_LOG_REQUEST_BODY`（預設 true，記錄每筆 request body；熱像資料建議設為 false）
- `GO_HTTP_LOG_REQUEST_BODY_MAX_BYTES`（預設 65536，body 日誌上限 bytes；停用 body log 時可調低）
- queue 滿載時會回 `503`（含 `Retry-After: 1`），避免默默丟資料

## 熱像資料路徑
- `GET /api/ward/floors/:floor/overview` 不回傳 `temperature_json`，只帶生命徵象、高溫、離床/上線與警示狀態。
- 熱像 frame 只在選中單床時透過 `GET /api/ward/sensors/:sensorNumber/thermal/latest` 或 `GET /api/ward/sensors/:sensorNumber/thermal/:dataID` 取得。
- `sensor_datas` 保存每秒生命徵象與 `high_temperature`；`sensor_thermal_frames` 依 `GO_THERMAL_FRAME_INTERVAL_SEC` 降頻保存 768 點 frame。

# 啟動指令 
docker compose up -d --build

# 啟動 mock_go 模擬上傳
docker compose up -d --build mock_go

# 快速 dump 熱像分析資料
docker compose run --rm sensor_dump dump --out /dumps/sensor_analysis.tar.gz --jobs 8

# 還原 dump（建議先停 mock/backend，避免還原時同時寫入）
docker compose stop mock_go backend_go
docker compose run --rm sensor_dump restore --in /dumps/sensor_analysis.tar.gz --jobs 8 --truncate
docker compose up -d backend_go mock_go

# 固定情境可用 .env 調整
MOCK_OUT_OF_BED_SENSORS=0201_01,0203_03,0205_05
MOCK_OFFLINE_SENSORS=0201_02,0203_04,0205_06
MOCK_WARNING_SENSORS=0207_01,0207_03,0209_01

# 查看 mock_go 即時送資料
docker logs -f -t irds_mock_go_v1

# 停止 mock_go
docker compose stop mock_go

# 查看所有服務指令 
docker compose ps

# 關閉指令 
docker compose down

# 關閉指令 且刪除所有資料
docker compose down -v

# 網址
http://localhost

# API
http://localhost/api/xxxx
or
http://localhost:3002/xxxx

# 查看後台API接收端
docker logs -f -t irds_backend_go_v1

# 每筆 request 監看（關鍵字過濾）
docker logs -f -t irds_backend_go_v1 | grep http_request

# 每筆 request + body（會很大量）
docker logs -f -t irds_backend_go_v1 | grep request_body

# 測試指令（V2 ESP32）
curl -X POST http://127.0.0.1:3002/sensor/data/v2/esp32 -H "Content-Type: application/json" -d '{"test":"ok"}'

# 測試指令（V2 STM32）
curl -X POST http://127.0.0.1:3002/sensor/data/v2/stm32 -H "Content-Type: application/json" -d '{"test":"ok_go1"}'


# 連上tailscale
內部IP: 100.89.107.4



{"sensor_number":"ROST205_S03","devicetype":"stm32","mac":"none","m_outOfRange":0,"m_distance":0,"m_angles_first":0,"m_angles_second":0,"m_movementState":0,"m_movementLevel":0,"m_breath_rate_state":0,"m_breath_rate_last_Transmit":0,"m_breath_rate":-1,"m_heart_rate_state":0,"m_heart_rate_last_Transmit":0,"m_heart_rate":-1,"temperature":[24,23.75,25.5,25.75,25,24.5,24.25,24,22.25,23.5,24.5,25,24.5,23,22.5,21.25,22.75,24,24.25,23.25,23,22,21.25,19.75,23.75,23.25,22.5,21.25,22,20.5,20.25,18,19.75,20.25,20,20,20,19.5,18.75,17.75,17.5,18.25,18,19.25,18.75,18.25,18,15.5,15.5,16,16.75,17.25,16.75,16.75,16.5,13.75,13.5,12.5,14.25,14.5,14.25,13.75,13.25,10]}


{"sensorNo":"ROOM207_06","devicetype":"esp32","mac":"24:6F:28:BF:9D:30","m_outOfRange":0,"m_distance":66,"m_angles_first":74,"m_angles_second":90,"m_movementState":0,"m_movementLevel":1,"m_breath_rate_state":1,"m_breath_rate":16,"m_breath_rate_last_Transmit":444299,"m_heart_rate_state":0,"m_heart_rate":72,"m_heart_rate_last_Transmit":443299,"thermistor":31.75,"temperature":[18.25,18,18.25,18.25,18.5,18,17.5,17.75,18.5,18,18.5,18.75,20.25,19.5,18.25,17.25,18.75,18.25,18.75,20.25,23,21,19,17.5,18.75,18.75,19,21,21.5,20,18.25,18,18.5,18.5,19.5,21.5,20.75,19.25,18.5,17.5,18.5,18.5,18.75,20.5,19.25,19,18.5,17.25,17.75,18.5,19,19,18.75,19.25,18,18,18.75,18.25,18.75,19.25,18.75,18.25,18.25,17]}


| 欄位                           | 中文意思    | 說明（工程用）             |
| ----------------------------- | ------- | ------------------- |
| sensorNo                    | 感測器編號   | 對應床位（例如 ROOM207_07） |
| devicetype                  | 設備類型    | esp32 / stm32       |
| mac                         | MAC位址   | 裝置唯一識別              |
| m_outOfRange                | 是否離開範圍  | 1=離床 / 超出偵測範圍   做事件紀錄    |
| m_distance                  | 距離      | 感測器到目標距離（cm）(先不使用)        |
| m_angles_first              | 角度1     | 第一個偵測角度  (先不使用)           |
| m_angles_second             | 角度2     | 第二個偵測角度  (先不使用)           |
| m_movementState             | 動作狀態    | 是否有動作       0 沒有活動 1 有活動 可做事件紀錄        |
| m_movementLevel             | 動作強度    | 動作強弱         做曲線圖       |
| m_breath_rate_state         | 呼吸狀態    | 呼吸是否有效     0 正常 不為0 不正常 可做事件紀錄         |
| m_breath_rate               | 呼吸率     | 每分鐘呼吸數（RPM）   做曲線      |
| m_breath_rate_last_Transmit | 呼吸最後時間  | 傳送時間戳               |
| m_heart_rate_state          | 心跳狀態    | 心跳是否有效      0 正常 不為0 不正常 可做事件紀錄         |
| m_heart_rate                | 心率      | BPM               做曲線  |
| m_heart_rate_last_Transmit  | 心跳最後時間  | 傳送時間戳               |
| thermistor                  | 熱敏電阻溫度  | 單點溫度（°C） 做曲線 Stm32 沒有這數據           |
| temperature                 | 熱影像溫度陣列 | 8x8 溫度（64點）   做熱成像圖      |


TRUNCATE TABLE sensor_raw_logs RESTART IDENTITY CASCADE;
TRUNCATE TABLE sensor_datas RESTART IDENTITY CASCADE;

SELECT
    s.sensor_number,
    sd.*
  FROM sensor_datas sd
  JOIN sensors s ON s.sensor_id = sd.sensor_id
  ORDER BY sd.timestamp DESC LIMIT 100;
