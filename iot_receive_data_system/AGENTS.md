# Agent Context & Memory

## 專案身分
- **角色**：開發主力
- **技術棧**：Golang, PostgreSQL, Vue.js, Node.js, Html, Css, JavaScript, Docker

## 核心記憶 (Key Decisions)
- 數據庫使用 PostgreSQL 16
- 區網運作上傳資料，由實際裝置透過 ingest API 上拋。
- 上傳資料可能會有碎片, 接收到需要拼揍, 處理後存入
- 房號統一使用四位數字串：`0201`, `0202`, `0301`；sensor_number 使用 `{room_number}_{bed_no}`，例如 `0201_01`
- 基礎 seeder 只建立並保留 2F/3F；每層房號為 `01`, `02`, `03`, `05`, `06`, `07`, `08`, `09`, `10`（跳過 04），例如 `0201`。
- 目前 `mock_sensors` 建立 2F/3F 共 108 個，用於確認切換與感測器對應；其他樓層先不建立也不顯示。
- 熱成像顯示需對齊 `doc/GYMCU90640.py`：`temperature` 若為 768 點，前端按 `24x32`（row-major）渲染；不做強制左右翻轉（`fliplr` 預設關閉）。
- 前端平面圖即時更新不可只依賴 SSE；`WardFloorMonitor.vue` 需每秒 polling overview，選中床位時每秒 polling history 來更新右側圖表。
- 前端平面圖分兩個版本：`/` 為大眾版（無熱成像），`/engineer` 為工程模式（保留熱成像與完整資訊）。
- 工程模式需登入才可進入：未登入導向 `/engineer/login`，登入狀態以前端 session 保存（`ward_engineer_authed`）。
- 工程模式登入帳密由 `.env` 提供：`VITE_ENGINEER_USERNAME`、`VITE_ENGINEER_PASSWORD`。
- 工程模式登出按鈕位於左側標題下方；視覺上需與左側平面圖左邊界對齊，且不可撐高 header。
- `WardFloorMonitor.vue` 平面圖版型與床位規則已抽離到 `frontend_vue/src/config/wardLayouts.js`；主元件只保留渲染與即時資料更新邏輯。
- 新增樓層時優先只改 `wardLayouts.js`：`wardFloorLayouts`（房間座標）、`floorBedNumberMaps`（床號映射）、`floorRoomGridOverrides`（床位格局）；未定義樓層自動回退 `default` 版型。
- `sensors` 直接保存 `floor_id` / `room_id`，不再使用 `room_has_sensors`
- 接收資料新增 sensor 時，依 `sensor_number` 前兩碼判斷樓層、前四碼判斷房間，寫入 `sensors.floor_id` / `sensors.room_id`
- `sensor_raw_logs.raw_content` 保留每次 request 收到的原始 chunk
- `sensor_raw_logs.processed_content` 只存「成功重組且接受寫入」的完整 JSON；碎片尚未完成或壞資料不可寫入 processed_content
- `sensor_raw_logs.processed_status` 取代舊的 `is_saved`
- `sensor_raw_logs.sensor_data_id` / `sensor_raw_logs.sensor_data_timestamp` 追蹤成功寫入 `sensor_datas` 的 `data_id` 與 `timestamp`
- 碎片重組以 `deviceType + client IP` 分開維護 buffer；若舊 buffer 被壞資料污染，遇到新的 `{` 開頭 JSON 時會丟棄舊 buffer 並重新解析

### `processed_status` 狀態碼
| 狀態碼 | key | 名稱 | 說明 | 追蹤欄位 |
|---:|---|---|---|---|
| `0` | `raw_only` | 只保留原始資料 | 尚未完成或只保留原始 chunk，未寫入 `sensor_datas` | `sensor_data_id` / `sensor_data_timestamp` 應為 `NULL` |
| `1` | `processed_saved` | 已寫入感測資料 | 重組成功且已寫入 `sensor_datas` | `sensor_data_id` / `sensor_data_timestamp` 應有值 |
| `2` | `malformed` | 格式錯誤 | 解析到疑似完整 JSON，但格式錯誤，不能寫入 `sensor_datas` | `sensor_data_id` / `sensor_data_timestamp` 應為 `NULL` |
| `3` | `duplicate` | 重複資料 | 重組後判定為重複事件，沒有重複寫入 `sensor_datas` | `sensor_data_id` / `sensor_data_timestamp` 應為 `NULL` |

## 常用指令
- 啟動：`docker compose up -d --build`
- 啟動 mock_go 模擬上傳：`docker compose up -d --build mock_go`
- 停止 mock_go：`docker compose stop mock_go`
- 查看 mock_go 即時送資料：`docker logs -f irds_mock_go_v1`
- mock_go 固定情境：`MOCK_OUT_OF_BED_SENSORS` 為上線離床清單，`MOCK_OFFLINE_SENSORS` 為不送資料的離線清單，`MOCK_WARNING_SENSORS` 為上線且在床但生命徵象超標清單；其餘床固定上線且在床。
- 查看服務：`docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'`
- 查看 backend ingest/request log：`docker logs -f irds_backend_go_v1`
- 查看 frontend Vite log：`docker logs -f irds_frontend_vue_v1`
- frontend build 驗證：`docker exec irds_frontend_vue_v1 npm run build`
- backend Go 測試：`docker run --rm -v "$PWD/backend_go":/app -w /app golang:1.23-alpine sh -lc '/usr/local/go/bin/go test ./internal/handler ./internal/repository ./internal/model ./internal/router'`
- 驗證樓層清單 API：`docker exec irds_frontend_vue_v1 node -e "fetch('http://backend_go:3002/api/ward/floors').then(r=>r.text()).then(console.log)"`
- 驗證 2F overview 每秒更新：`docker exec irds_frontend_vue_v1 node -e "let prev='';(async()=>{for(let i=0;i<5;i++){const r=await fetch('http://backend_go:3002/api/ward/floors/2/overview');const j=await r.json();const b=j.data.rooms[0].beds[0];const line=[b.sensor_number,b.latest?.data_id,b.latest?.timestamp,b.latest?.heart_rate,b.latest?.breath_rate].join(' ');console.log(line,line===prev?'SAME':'CHANGED');prev=line;await new Promise(res=>setTimeout(res,1000));}})()"`
- 驗證單床 history 每秒更新：`docker exec irds_frontend_vue_v1 node -e "const sensor='0201_01';let prev='';(async()=>{for(let i=0;i<3;i++){const start=new Date(Date.now()-5*60*1000).toISOString();const r=await fetch('http://backend_go:3002/api/ward/sensors/'+sensor+'/history?start='+encodeURIComponent(start)+'&limit=20');const j=await r.json();const pts=j.data.points;const last=pts[pts.length-1];const line=[pts.length,last?.data_id,last?.timestamp,last?.heart_rate,last?.breath_rate].join(' ');console.log(line,line===prev?'SAME':'CHANGED');prev=line;await new Promise(res=>setTimeout(res,1000));}})()"`
- 驗證 DB 最新 sensor_datas：`docker exec irds_db_postgresql_v1 psql -U postgres -d sg_tch_v3 -c "select count(*) as total, max(timestamp) as latest from sensor_datas; select s.sensor_number, sd.timestamp, sd.heart_rate, sd.breath_rate from sensor_datas sd join sensors s on s.sensor_id=sd.sensor_id order by sd.timestamp desc limit 5;"`
- dump 熱像分析資料：`docker compose run --rm sensor_dump dump --out /dumps/sensor_analysis.tar.gz --jobs 8`
- 還原 dump：`docker compose stop mock_go backend_go && docker compose run --rm sensor_dump restore --in /dumps/sensor_analysis.tar.gz --jobs 8 --truncate && docker compose up -d backend_go mock_go`

## 環境變數
- 主要使用專案根目錄 `.env`
- 使用 PostgreSQL：`PG_HOST / PG_PORT / PG_USER / PG_PASSWORD / PG_DB_NAME`
- Docker 內部 backend 位址只改 `BACKEND_INTERNAL_BASE_URL`；compose 會轉給 frontend proxy 的 `VITE_PROXY_TARGET`
- 前端呼叫 API 路徑用 `VITE_API_BASE_URL=/api`
- 工程模式前端帳密用 `VITE_ENGINEER_USERNAME` / `VITE_ENGINEER_PASSWORD`（由 compose 傳給 `frontend_vue`）
- 啟動時預設會自動執行 `golang-migrate`（可用 `GO_MIGRATE_ON_START=false` 關閉）
- `backend_go` 服務目前為 Go API server
- `frontend_vue` 服務目前為 Vue/Vite frontend
- 目前提供單一上拋端點（設備類型為路由參數）：
  - `POST /sensor/data/v2/:deviceType`
  - `:deviceType` 僅允許 `esp32` 或 `stm32`
- `frontend_vue` 平面圖資料來源：
  - `GET /api/ward/floors`：取得 DB 目前所有可選樓層，前端不可寫死樓層清單
  - `GET /api/ward/floors/:floor/overview`：取得樓層所有房間/床位最新狀態
  - `GET /api/ward/sensors/:sensorNumber/history?start=&end=&limit=`：取得右側溫度、心跳、呼吸圖表資料
  - `GET /api/ward/sensors/:sensorNumber/thermal/latest`：取得選中床位最新 `temperature_json` 熱像 frame
  - `GET /api/ward/floors/:floor/stream`：SSE 即時推送；目前只能當輔助路徑，不能作為唯一更新來源
  - 前端優先吃 API；API 不可用時才退回本機 mock 資料，避免畫面空白

## frontend_vue 需要的資料
| 用途 | 欄位 | 來源 |
|---|---|---|
| 平面圖床位 | `bed_id`, `room_number`, `bed_number`, `sensor_number`, `device_type` | `mock_sensors` + `rooms` + `floors` |
| 裝置狀態 | `device_online` | 最新 `sensor_datas.timestamp` 是否在 90 秒內 |
| 即時生命徵象 | `heart_rate`, `breath_rate`, `temperature`, `rhythm` | 最新 `sensor_datas` |
| 右側呼吸圖 | `breath_rate` | 直接使用 `sensor_datas.breath_rate` |
| 熱成像 | `temperature_json`, `high_temperature` | 選中床位最新 `sensor_datas`（latest frame，不做 history 播放） |
| 病患姓名 | `patient_name` | 目前以房間現有未出院 `patient` 聚合顯示，未來若有床位對應表要改成 bed-level |

# backend_go 架構
- `backend_go/cmd/main.go`：程式進入點
- `backend_go/internal/model` : 資料模型與設定
- `backend_go/internal/handler` : HTTP handler 與資料處理管線
- `backend_go/internal/repository` : 資料庫存取層
- `backend_go/internal/migration` : `golang-migrate` 版本化 migration
- `backend_go/internal/router` : 路由設定
- `backend_go/pkg/logger` : Zap logger 封裝
- `backend_go/pkg/response` : 統一 API 回傳工具
- `backend_go/docs` : Swagger 文件輸出目錄（保留）

# frontend_vue 架構
- `frontend_vue/src/main.js`：Vue app 進入點，掛載 router
- `frontend_vue/src/App.vue`：只保留 `<RouterView />`
- `frontend_vue/src/router`：前端路由設定
- `frontend_vue/src/views`：頁面層，負責組合 page-level component
- `frontend_vue/src/components`：可重用 UI/平面圖元件，2F 平面圖在 `WardFloorMonitor.vue`
- `frontend_vue/src/views/EngineerLoginView.vue`：工程模式登入頁（`/engineer/login`）
- `frontend_vue/src/auth/engineerAuth.js`：工程模式登入狀態（session）工具
- `frontend_vue/src/config/wardLayouts.js`：平面圖配置來源（樓層房間座標、床號映射、床位格局覆寫）；新增/調整樓層版型以此檔為主。
- `frontend_vue/src/api`：前端 API client，統一串 `backend_go`
- `frontend_vue/src/i18n`：Vue i18n 語系設定與語系檔，目前支援 `en` / `zh-TW` / `vi`
- 不再用 iframe 載入 standalone HTML 作為正式入口；平面圖功能應維護在 `.vue` component 內
- `WardFloorMonitor.vue` 的 header 需保持單行：樓層標籤、標題、警告燈圖例在同一行；右上角控制區上下排列，語言切換在上、樓層切換在下。
- 語言切換需和樓層切換一樣有左右箭頭與橫向 window，語言順序為 `EN`、`中`、`VI`，預設語系為 `zh-TW`；會寫入 `localStorage.locale`。
- 標題文字需保留固定寬度，避免英文/越南文切換造成圖例跑版。
- 護理站文字需依語系縮字：中文 28px、英文 20px、越南文 18px，避免越南文超出護理站格子。
- 床位格內第一行顯示 `sensor_number`，第二行顯示病患姓名；不要在床位格內顯示心跳/呼吸數值，狀態用五個燈號表示即可。
- 右側床位資訊標題列：總摘要放在「床位 xxxx」標題右邊，摘要永遠看全部床位，不要依選取房間切成 `xx房摘要`。
- 右側床位資訊欄位需左右兩欄：左欄為病患、心律、溫度；右欄為心跳、呼吸、裝置。
- 不使用獨立的下方樓層切換列；平面圖頁面不顯示「暫停更新」與「重置」按鈕。

## frontend_vue 即時更新規則
- `WardFloorMonitor.vue` 初次載入先抓 floors、render beds、抓 overview，再啟動 SSE。
- 前端仍需每秒 polling `GET /api/ward/floors/:floor/overview`，即使 SSE 顯示 connected，也不能停止 polling。
- 有選中床位時，每秒 polling `GET /api/ward/sensors/:sensorNumber/history?start=最近5分鐘&limit=600`，右側溫度、心跳、呼吸圖表要持續更新。
- 有選中床位時，每秒 polling `GET /api/ward/sensors/:sensorNumber/thermal/latest`，護理站下方熱像圖要持續更新最新 frame。
- `WardFloorMonitor.vue` 熱像 frame 解析規則：優先支援 `24x32`（768 點）；若非 768 且為平方長度才回退為 `NxN`，其他長度視為無效 frame。
- 沒有手動設定區間時，右側圖表使用最近 5 分鐘 live window；按快速區間或手動區間時尊重固定區間；按清除區間後回 live window。
- `API_PREFIX` 需使用 `VITE_API_BASE_URL`，預設 `/api`，避免硬寫導致部署時不一致。

## frontend_vue 語系規則
- 使用 `vue-i18n`，入口在 `frontend_vue/src/i18n/index.js`。
- 新增 UI 文案時必須同步補 `zh-TW.js`、`en.js`、`vi.js`。
- 後端 `rhythm` 目前可能回中文，前端顯示前需 normalize 成 key：`sinus`、`afib`、`tachycardia`、`bradycardia`、`unknown`，再走 i18n 顯示。
- 語系切換時需同步更新 `locale.value`、`localStorage.locale`、`document.documentElement.lang`，並重繪 summary、選中床位資訊與圖表文字。

## frontend_vue 平面圖警告燈圖例
- `WardFloorMonitor.vue` 標題旁警告燈圖例需每個監測項目獨立一格：心跳、心律、溫度、呼吸、人在床。
- 所有正常狀態統一使用綠色，所有警告/異常狀態統一使用紅色。
- 人在床顯示綠色，離床顯示紅色。
- 目前色票：
  - 正常 / 在床：`#16a34a`
  - 警告 / 離床：`#dc2626`

## 平面圖警告判斷標準
- 裝置狀態：最新 `sensor_datas.timestamp` 超過 `GO_DEVICE_OFFLINE_SECONDS` 未更新即離線；目前預設 30 秒。
- 心跳：`50 <= heart_rate <= 110` 為正常；小於 50 或大於 110 為警告。
- 心律：`rhythm == "竇性心律"` 為正常；其他心律（例如 `心房顫動`、`心搏過速`、`心搏過緩`、`未知`）為警告。
- 溫度：`35.8 <= temperature <= 37.6` 為正常；低於 35.8 或高於 37.6 為警告。
- 呼吸：`10 <= breath_rate <= 25` 為正常；小於 10 或大於 25 為警告。
- 人在床：`presence !== false` 為在床；`presence === false` 為離床警告。
- 床位外框警示：只有警告燈項目（心跳、心律、溫度、呼吸、人在床）任一異常時，床位外框才顯示紅色粗框。
- 裝置離線顯示：離線不觸發紅色粗框；離線時整張床底色反灰，線上時維持正常床位底色。
- 右側摘要只顯示 `上線` / `離線` / `警示` 三個計數，不顯示心跳、溫度、呼吸平均值，也不顯示 `注意`；摘要永遠統計全部床位，`警示` 代表全部床位中任一床位有任一警告燈異常。
- 後端若 payload 沒有提供 `rhythm`，會依 heart_rate 推估：`heart_rate >= 110` 為 `心搏過速`，`heart_rate <= 50` 為 `心搏過緩`，其他有效心跳為 `竇性心律`，無效心跳為 `未知`。

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
- `GO_DEVICE_OFFLINE_SECONDS`（預設 30，最新 `sensor_datas.timestamp` 超過此秒數未更新即視為裝置離線）
- `GO_THERMAL_FRAME_INTERVAL_SEC`（預設 10，熱像 frame 降頻寫入秒數；生命徵象仍每秒寫入）
- `GO_REQUEST_DEDUP_TTL_SEC`（預設 8）
- `GO_REQUEST_DEDUP_MAX_KEYS`（預設 200000）
- `GO_EVENT_DEDUP_TTL_SEC`（預設 300）
- `GO_EVENT_DEDUP_MAX_KEYS`（預設 500000）
- `GO_HTTP_LOG_EVERY_REQUEST`（預設 true，記錄每一筆 HTTP request）
- `GO_HTTP_LOG_REQUEST_BODY`（預設 true，記錄每筆 request body；熱像資料建議設為 false）
- `GO_HTTP_LOG_REQUEST_BODY_MAX_BYTES`（預設 65536，body 日誌上限 bytes；停用 body log 時可調低）
- queue 滿載時會回 `503`（含 `Retry-After: 1`），避免默默丟資料
