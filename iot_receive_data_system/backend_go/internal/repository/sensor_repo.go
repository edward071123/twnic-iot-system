package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend_go/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Package repository contains PostgreSQL persistence implementation.
// repository 套件包含 PostgreSQL 持久化實作。

// SensorRepository defines storage operations used by ingest handler.
// SensorRepository 定義給 ingest handler 使用的儲存層操作介面。
// Keeping this interface small makes it easy to mock in tests.
// 介面保持精簡可讓單元測試更容易做 mock。
type SensorRepository interface {
	Ping(ctx context.Context) error
	Close() error
	MaintainStorage(ctx context.Context) error
	SeedBaseData(ctx context.Context) error
	LookupSensorID(ctx context.Context, sensorNumber string, deviceType model.DeviceType) (int, bool, error)
	InsertSensorBatch(ctx context.Context, dt model.DeviceType, rows []model.SensorInsertRow) ([]model.SensorDataWriteRef, error)
	InsertThermalFrameBatch(ctx context.Context, rows []model.ThermalFrameInsertRow) error
	InsertRawLogBatch(ctx context.Context, dt model.DeviceType, rows []model.RawLogInsertRow) error
	ListWardFloors(ctx context.Context) ([]model.WardFloorOption, error)
	GetWardFloorOverview(ctx context.Context, floor int) (model.WardOverview, error)
	GetSensorHistory(ctx context.Context, sensorNumber string, start, end *time.Time, limit int) (model.SensorHistoryResponse, error)
	GetSensorThermalTimeline(ctx context.Context, sensorNumber string, start, end *time.Time) (model.WardThermalTimelineResponse, error)
	GetSensorThermalFrame(ctx context.Context, sensorNumber string, dataID int64) (*model.WardSensorDataPoint, error)
	GetSensorLatestThermalFrame(ctx context.Context, sensorNumber string) (*model.WardSensorDataPoint, error)
}

// sensorCacheEntry stores sensor_id lookup cache records with TTL.
// sensorCacheEntry 儲存 sensor_id 查詢快取資料與過期時間。
type sensorCacheEntry struct {
	SensorID  int
	ExpiresAt time.Time
}

// sensorCache is a lightweight in-memory TTL cache for sensor_number -> sensor_id.
// sensorCache 是輕量的記憶體 TTL 快取，用於 sensor_number -> sensor_id。
type sensorCache struct {
	ttl   time.Duration
	mu    sync.RWMutex
	items map[string]sensorCacheEntry
}

// newSensorCache creates lookup cache with configured TTL.
// newSensorCache 依設定 TTL 建立查詢快取。
func newSensorCache(ttl time.Duration) *sensorCache {
	return &sensorCache{
		ttl:   ttl,
		items: make(map[string]sensorCacheEntry),
	}
}

// get returns cached sensor_id when entry exists and has not expired.
// get 在快取存在且未過期時回傳 sensor_id。
func (c *sensorCache) get(key string) (int, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.ExpiresAt) {
		return 0, false
	}
	return entry.SensorID, true
}

// set inserts or refreshes a cached sensor_id entry.
// set 新增或更新快取中的 sensor_id 資料。
func (c *sensorCache) set(key string, sensorID int) {
	c.mu.Lock()
	c.items[key] = sensorCacheEntry{
		SensorID:  sensorID,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// SQLSensorRepository is the concrete PostgreSQL-backed repository implementation.
// SQLSensorRepository 是以 PostgreSQL 為基礎的儲存層實作。
type SQLSensorRepository struct {
	db    *sql.DB
	cfg   model.AppConfig
	cache *sensorCache
}

// NewSQLSensorRepository opens DB connection, configures pool, and initializes caches.
// NewSQLSensorRepository 會開啟 DB 連線、設定連線池並初始化快取。
func NewSQLSensorRepository(cfg model.AppConfig) (*SQLSensorRepository, error) {
	db, err := openDB(cfg)
	if err != nil {
		return nil, err
	}
	configureDBPool(db, cfg)

	return &SQLSensorRepository{
		db:    db,
		cfg:   cfg,
		cache: newSensorCache(cfg.SensorCacheTTL),
	}, nil
}

// Ping validates DB connectivity on startup/health checks.
// Ping 用於啟動階段或健康檢查時驗證 DB 連線可用性。
func (r *SQLSensorRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// MaintainStorage ensures partitions exist and applies raw log retention policy.
// MaintainStorage 確保分區存在並套用 raw log 留存策略。
func (r *SQLSensorRepository) MaintainStorage(ctx context.Context) error {
	maintainCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(
		maintainCtx,
		`SELECT
		   ensure_sensor_data_monthly_partitions($1, $2),
		   ensure_sensor_thermal_monthly_partitions($1, $2),
		   ensure_sensor_raw_log_partitions($3, $4),
		   prune_sensor_raw_log_partitions($5)`,
		r.cfg.SensorDataPartitionBackMonths,
		r.cfg.SensorDataPartitionAheadMonths,
		r.cfg.RawLogPartitionBackDays,
		r.cfg.RawLogPartitionAheadDays,
		r.cfg.RawLogRetentionDays,
	)
	return err
}

// SeedBaseData inserts baseline hospital/floor/room records in an idempotent way.
// SeedBaseData 以可重複執行方式插入基礎醫院/樓層/房間資料。
func (r *SQLSensorRepository) SeedBaseData(ctx context.Context) error {
	seedCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(seedCtx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1) hospital
	// 1) 醫院
	if _, err := tx.ExecContext(seedCtx,
		`INSERT INTO hospitals (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`,
		r.cfg.SeedHospitalName,
	); err != nil {
		return err
	}

	// 2) floors: keep only 2F and 3F for the current ward scope.
	// 2) 樓層：目前病房範圍只保留 2F 與 3F。
	if _, err := tx.ExecContext(seedCtx,
		`INSERT INTO floors (hospital_id, floor_number)
		 SELECT h.id, gs.floor_number
		 FROM hospitals h
		 CROSS JOIN generate_series(2, 3) AS gs(floor_number)
		 WHERE h.name = $1
		 ON CONFLICT (hospital_id, floor_number) DO NOTHING`,
		r.cfg.SeedHospitalName,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(seedCtx,
		`DELETE FROM floors f
		 USING hospitals h
		 WHERE h.id = f.hospital_id
		   AND h.name = $1
		   AND f.floor_number NOT IN (2, 3)`,
		r.cfg.SeedHospitalName,
	); err != nil {
		return err
	}

	// 3) rooms: 01~10 without 04 for every seeded floor.
	// 3) 房間：每個樓層建立 01~10（不含 04）。
	if _, err := tx.ExecContext(seedCtx,
		`INSERT INTO rooms (floors_id, room_number, bed_number)
		 SELECT f.floors_id, format('%s%s', lpad(f.floor_number::text, 2, '0'), lpad(v.base_room::text, 2, '0')) AS room_number, 0
		 FROM floors f
		 JOIN hospitals h ON h.id = f.hospital_id
		 JOIN (
		   VALUES (1),(2),(3),(5),(6),(7),(8),(9),(10)
		 ) AS v(base_room) ON TRUE
		 WHERE h.name = $1
		   AND f.floor_number BETWEEN 2 AND 3
		 ON CONFLICT (floors_id, room_number) DO NOTHING`,
		r.cfg.SeedHospitalName,
	); err != nil {
		return err
	}

	// 4) fixed mock sensors based on the current 2F/3F floor-plan bed counts.
	// 4) 固定 mock 感測器：依目前 2F/3F 平面圖床位數量建立。
	if _, err := tx.ExecContext(seedCtx,
		`WITH room_beds AS (
		   SELECT
		     f.floor_number,
		     r.room_id,
		     r.room_number,
		     rb.bed_count
		   FROM floors f
		   JOIN hospitals h ON h.id = f.hospital_id
		   JOIN rooms r ON r.floors_id = f.floors_id
		   JOIN (VALUES
		     (1,8),(2,2),(3,8),(5,8),(6,6),(7,8),(8,6),(9,6),(10,2)
		   ) AS rb(base_room, bed_count)
		     ON r.room_number = format('%s%s', lpad(f.floor_number::text, 2, '0'), lpad(rb.base_room::text, 2, '0'))
		   WHERE h.name = $1
		     AND f.floor_number BETWEEN 2 AND 3
		 ),
		 slots AS (
		   SELECT
		     rb.floor_number,
		     rb.room_id,
		     rb.room_number,
		     gs.slot_no,
		     row_number() OVER (ORDER BY rb.room_number, gs.slot_no) AS rn,
		     count(*) OVER () AS total_count
		   FROM room_beds rb
		   CROSS JOIN LATERAL generate_series(1, rb.bed_count) AS gs(slot_no)
		 )
		 INSERT INTO mock_sensors (sensor_ip, sensor_number, device_type, room_id)
		 SELECT
		   format('10.89.%s.%s', ((rn - 1) / 250) + 1, ((rn - 1) % 250) + 1) AS sensor_ip,
		   format('%s_%s', room_number, lpad(slot_no::text, 2, '0')) AS sensor_number,
		   CASE WHEN rn <= total_count / 2 THEN 'esp32' ELSE 'stm32' END AS device_type,
		   room_id
		 FROM slots
		 ON CONFLICT (sensor_number) DO NOTHING`,
		r.cfg.SeedHospitalName,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// Close releases DB resources.
// Close 釋放 DB 連線資源。
func (r *SQLSensorRepository) Close() error {
	return r.db.Close()
}

// LookupSensorID resolves sensor_number to sensor_id with cache/lookup first,
// and only performs upsert when the sensor does not exist.
// LookupSensorID 先走快取/查詢解析 sensor_number 對應的 sensor_id，
// 僅在查無資料時才執行 upsert。
func (r *SQLSensorRepository) LookupSensorID(ctx context.Context, sensorNumber string, deviceType model.DeviceType) (int, bool, error) {
	// Fast path: resolve from in-memory TTL cache.
	// 快速路徑：先從記憶體 TTL 快取查詢。
	if id, ok := r.cache.get(sensorNumber); ok {
		return id, true, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	floorID, roomID, err := r.resolveSensorLocation(queryCtx, sensorNumber)
	if err != nil {
		return 0, false, err
	}

	// Query from DB when cache misses.
	// 快取未命中時改查資料庫。
	query := "SELECT sensor_id FROM sensors WHERE sensor_number = ?"
	query = toPostgresPlaceholders(query)

	var sensorID int
	err = r.db.QueryRowContext(queryCtx, query, sensorNumber).Scan(&sensorID)
	if errors.Is(err, sql.ErrNoRows) {
		// Miss path: create sensor row once, or reuse existing row via ON CONFLICT.
		// 未命中路徑：只在不存在時建立，衝突時重用既有資料列。
		upsertSQL := `
			INSERT INTO sensors (sensor_number, device_type, floor_id, room_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (sensor_number)
			DO UPDATE SET
			  device_type = EXCLUDED.device_type,
			  floor_id = EXCLUDED.floor_id,
			  room_id = EXCLUDED.room_id
			RETURNING sensor_id`
		err = r.db.QueryRowContext(queryCtx, upsertSQL, sensorNumber, string(deviceType), floorID, roomID).Scan(&sensorID)
		if err != nil {
			return 0, false, err
		}
		r.cache.set(sensorNumber, sensorID)
		return sensorID, true, nil
	}
	if err != nil {
		return 0, false, err
	}

	if _, err := r.db.ExecContext(
		queryCtx,
		`UPDATE sensors
		 SET device_type = $2,
		     floor_id = COALESCE($3, floor_id),
		     room_id = COALESCE($4, room_id)
		 WHERE sensor_number = $1`,
		sensorNumber,
		string(deviceType),
		floorID,
		roomID,
	); err != nil {
		return 0, false, err
	}

	r.cache.set(sensorNumber, sensorID)
	return sensorID, true, nil
}

func (r *SQLSensorRepository) resolveSensorLocation(ctx context.Context, sensorNumber string) (any, any, error) {
	if len(sensorNumber) < 4 {
		return nil, nil, nil
	}
	floorRaw := sensorNumber[:2]
	roomNumber := sensorNumber[:4]
	floorNumber, err := strconv.Atoi(floorRaw)
	if err != nil {
		return nil, nil, nil
	}

	var floorID sql.NullInt64
	var roomID sql.NullInt64
	err = r.db.QueryRowContext(ctx, `
		SELECT f.floors_id, r.room_id
		FROM floors f
		JOIN rooms r ON r.floors_id = f.floors_id
		WHERE f.floor_number = $1::int
		  AND r.room_number = $2
		LIMIT 1`, floorNumber, roomNumber).Scan(&floorID, &roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return nullableInt64(floorID), nullableInt64(roomID), nil
}

// InsertSensorBatch writes normalized sensor rows in one multi-value INSERT.
// InsertSensorBatch 以單次多值 INSERT 批次寫入感測資料列。
func (r *SQLSensorRepository) InsertSensorBatch(ctx context.Context, dt model.DeviceType, rows []model.SensorInsertRow) ([]model.SensorDataWriteRef, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	cfg, ok := model.DeviceTypeConfigs[dt]
	if !ok {
		return nil, fmt.Errorf("unknown device type: %s", dt)
	}

	sqlText, args := buildSensorInsertSQL("postgres", cfg.DataTable, rows)
	insertCtx, cancel := context.WithTimeout(ctx, r.cfg.DBInsertTimeout)
	defer cancel()

	resultRows, err := r.db.QueryContext(insertCtx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer resultRows.Close()

	refs := make([]model.SensorDataWriteRef, 0, len(rows))
	for resultRows.Next() {
		var ref model.SensorDataWriteRef
		if err := resultRows.Scan(&ref.DataID, &ref.Timestamp); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := resultRows.Err(); err != nil {
		return nil, err
	}
	if len(refs) != len(rows) {
		return nil, fmt.Errorf("sensor insert returning mismatch: got=%d want=%d", len(refs), len(rows))
	}
	return refs, nil
}

// InsertRawLogBatch writes raw packet logs in one multi-value INSERT.
// InsertRawLogBatch 以單次多值 INSERT 批次寫入原始封包日誌。
func (r *SQLSensorRepository) InsertRawLogBatch(ctx context.Context, dt model.DeviceType, rows []model.RawLogInsertRow) error {
	if len(rows) == 0 {
		return nil
	}
	_, ok := model.DeviceTypeConfigs[dt]
	if !ok {
		return fmt.Errorf("unknown device type: %s", dt)
	}

	sqlText, args := buildRawLogInsertSQL("postgres", "sensor_raw_logs", dt, rows)
	insertCtx, cancel := context.WithTimeout(ctx, r.cfg.DBInsertTimeout)
	defer cancel()
	_, err := r.db.ExecContext(insertCtx, sqlText, args...)
	return err
}

// ListWardFloors returns floor options available in the current database.
// ListWardFloors 回傳目前資料庫可選擇的樓層。
func (r *SQLSensorRepository) ListWardFloors(ctx context.Context) ([]model.WardFloorOption, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(queryCtx, `
		SELECT DISTINCT floor_number
		FROM floors
		WHERE floor_number IN (2, 3)
		ORDER BY floor_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	floors := []model.WardFloorOption{}
	for rows.Next() {
		var floor int
		if err := rows.Scan(&floor); err != nil {
			return nil, err
		}
		floors = append(floors, model.WardFloorOption{
			Floor: floor,
			Label: fmt.Sprintf("%dF", floor),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return floors, nil
}

// GetWardFloorOverview returns the latest bed states used by the floor-plan frontend.
// GetWardFloorOverview 回傳平面圖前端使用的最新床位狀態。
func (r *SQLSensorRepository) GetWardFloorOverview(ctx context.Context, floor int) (model.WardOverview, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(queryCtx, `
		WITH room_limits AS (
		  SELECT format('%s%s', lpad(($1::int)::text, 2, '0'), lpad(base_room::text, 2, '0')) AS room_number, bed_count
		  FROM (VALUES
		    (1, 8), (2, 2), (3, 8), (5, 8), (6, 6),
		    (7, 8), (8, 6), (9, 6), (10, 2)
		  ) AS v(base_room, bed_count)
		),
		bed_slots AS (
		  SELECT room_number, generate_series(1, bed_count) AS bed_number
		  FROM room_limits
		),
		patient_by_room AS (
		  SELECT
		    p.room_id,
		    string_agg(p.patient_name, '、' ORDER BY p.patient_id) AS patient_name
		  FROM patient p
		  WHERE p.discharge_date IS NULL
		  GROUP BY p.room_id
		)
		SELECT
		  r.room_id,
		  r.room_number::text,
		  bs.bed_number,
		  COALESCE(ms.sensor_number, format('%s_%s', r.room_number, lpad(bs.bed_number::text, 2, '0'))) AS sensor_number,
		  COALESCE(ms.device_type, s.device_type, '') AS device_type,
		  s.sensor_id,
		  COALESCE(pbr.patient_name, '') AS patient_name,
		  sd.data_id,
		  sd."timestamp",
		  sd.heart_rate,
		  sd.rhythm,
		  sd.breath_rate,
		  sd.out_of_range,
		  sd.high_temperature,
		  (sd."timestamp" IS NOT NULL AND sd."timestamp" >= timezone('UTC', now()) - ($2::int * interval '1 second')) AS device_online
		FROM rooms r
		JOIN floors f ON f.floors_id = r.floors_id
		JOIN room_limits rl ON rl.room_number = r.room_number
		JOIN bed_slots bs ON bs.room_number = r.room_number
		LEFT JOIN mock_sensors ms
		  ON ms.room_id = r.room_id
		 AND ms.sensor_number = format('%s_%s', r.room_number, lpad(bs.bed_number::text, 2, '0'))
		LEFT JOIN sensors s
		  ON s.sensor_number = COALESCE(ms.sensor_number, format('%s_%s', r.room_number, lpad(bs.bed_number::text, 2, '0')))
		LEFT JOIN patient_by_room pbr ON pbr.room_id = r.room_id
		LEFT JOIN LATERAL (
		  SELECT data_id, "timestamp", heart_rate, rhythm, breath_rate, out_of_range, high_temperature
		  FROM sensor_datas sd
		  WHERE sd.sensor_id = s.sensor_id
		  ORDER BY sd."timestamp" DESC
		  LIMIT 1
		) sd ON TRUE
		WHERE f.floor_number = $1::int
		ORDER BY r.room_number, bs.bed_number`, floor, int(r.cfg.DeviceOfflineTimeout.Seconds()))
	if err != nil {
		return model.WardOverview{}, err
	}
	defer rows.Close()

	overview := model.WardOverview{
		Floor:       floor,
		GeneratedAt: time.Now().UTC(),
		Rooms:       []model.WardOverviewRoom{},
	}
	roomIndex := map[string]int{}

	for rows.Next() {
		var (
			roomID          int
			roomNumber      string
			bedNumber       int
			sensorNumber    string
			deviceType      string
			sensorID        sql.NullInt64
			patientName     string
			dataID          sql.NullInt64
			timestamp       sql.NullTime
			heartRate       sql.NullInt64
			rhythm          sql.NullString
			breathRate      sql.NullInt64
			outOfRange      sql.NullInt64
			highTemperature sql.NullFloat64
			deviceOnline    bool
		)
		if err := rows.Scan(
			&roomID,
			&roomNumber,
			&bedNumber,
			&sensorNumber,
			&deviceType,
			&sensorID,
			&patientName,
			&dataID,
			&timestamp,
			&heartRate,
			&rhythm,
			&breathRate,
			&outOfRange,
			&highTemperature,
			&deviceOnline,
		); err != nil {
			return model.WardOverview{}, err
		}

		idx, ok := roomIndex[roomNumber]
		if !ok {
			idx = len(overview.Rooms)
			roomIndex[roomNumber] = idx
			overview.Rooms = append(overview.Rooms, model.WardOverviewRoom{
				RoomID:     roomID,
				RoomNumber: roomNumber,
				Beds:       []model.WardOverviewBed{},
			})
		}

		bed := model.WardOverviewBed{
			BedID:        fmt.Sprintf("%s-%02d", roomNumber, bedNumber),
			RoomID:       roomID,
			RoomNumber:   roomNumber,
			BedNumber:    bedNumber,
			SensorID:     nullableInt64(sensorID),
			SensorNumber: sensorNumber,
			DeviceType:   deviceType,
			DeviceOnline: deviceOnline,
			Presence:     true,
			PatientName:  patientName,
			Latest:       buildWardDataPoint(dataID, timestamp, heartRate, rhythm, breathRate, outOfRange, highTemperature, sql.NullString{}),
		}
		overview.Rooms[idx].Beds = append(overview.Rooms[idx].Beds, bed)
	}
	if err := rows.Err(); err != nil {
		return model.WardOverview{}, err
	}
	return overview, nil
}

// GetSensorHistory returns historical chart points without thermal pixels.
// GetSensorHistory 回傳歷史圖表點，但不包含熱像 768 點資料，避免大區間查詢爆量。
func (r *SQLSensorRepository) GetSensorHistory(ctx context.Context, sensorNumber string, start, end *time.Time, _ int) (model.SensorHistoryResponse, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(queryCtx, `
		SELECT sd.data_id, sd."timestamp", sd.heart_rate, sd.rhythm, sd.breath_rate, sd.out_of_range, sd.high_temperature
		FROM sensor_datas sd
		JOIN sensors s ON s.sensor_id = sd.sensor_id
		WHERE s.sensor_number = $1
		  AND ($2::timestamp IS NULL OR sd."timestamp" >= $2)
		  AND ($3::timestamp IS NULL OR sd."timestamp" <= $3)
		ORDER BY sd."timestamp" ASC`, sensorNumber, start, end)
	if err != nil {
		return model.SensorHistoryResponse{}, err
	}
	defer rows.Close()

	resp := model.SensorHistoryResponse{
		SensorNumber: sensorNumber,
		Start:        start,
		End:          end,
		Points:       []model.WardSensorDataPoint{},
	}
	for rows.Next() {
		var (
			dataID          sql.NullInt64
			timestamp       sql.NullTime
			heartRate       sql.NullInt64
			rhythm          sql.NullString
			breathRate      sql.NullInt64
			outOfRange      sql.NullInt64
			highTemperature sql.NullFloat64
		)
		if err := rows.Scan(&dataID, &timestamp, &heartRate, &rhythm, &breathRate, &outOfRange, &highTemperature); err != nil {
			return model.SensorHistoryResponse{}, err
		}
		point := buildWardDataPoint(dataID, timestamp, heartRate, rhythm, breathRate, outOfRange, highTemperature, sql.NullString{})
		if point != nil {
			resp.Points = append(resp.Points, *point)
		}
	}
	if err := rows.Err(); err != nil {
		return model.SensorHistoryResponse{}, err
	}
	resp.Points = downsampleWardHistoryPoints(resp.Points, 1200)
	return resp, nil
}

func downsampleWardHistoryPoints(points []model.WardSensorDataPoint, maxBuckets int) []model.WardSensorDataPoint {
	if maxBuckets <= 0 || len(points) <= maxBuckets {
		return points
	}
	firstTs := points[0].Timestamp
	lastTs := points[len(points)-1].Timestamp
	span := lastTs.Sub(firstTs)
	if span <= 0 {
		return points
	}

	type bucket struct {
		first     *model.WardSensorDataPoint
		tempMin   *model.WardSensorDataPoint
		tempMax   *model.WardSensorDataPoint
		hrMin     *model.WardSensorDataPoint
		hrMax     *model.WardSensorDataPoint
		breathMin *model.WardSensorDataPoint
		breathMax *model.WardSensorDataPoint
		last      *model.WardSensorDataPoint
	}

	buckets := make([]bucket, maxBuckets)
	for i := range points {
		point := points[i]
		ratio := float64(point.Timestamp.Sub(firstTs)) / float64(span)
		idx := int(ratio * float64(maxBuckets))
		if idx < 0 {
			idx = 0
		}
		if idx >= maxBuckets {
			idx = maxBuckets - 1
		}
		b := &buckets[idx]
		if b.first == nil {
			b.first = &point
			b.tempMin = &point
			b.tempMax = &point
			b.hrMin = &point
			b.hrMax = &point
			b.breathMin = &point
			b.breathMax = &point
		}
		if point.Temperature < b.tempMin.Temperature {
			b.tempMin = &point
		}
		if point.Temperature > b.tempMax.Temperature {
			b.tempMax = &point
		}
		if point.HeartRate >= 0 && (b.hrMin == nil || b.hrMin.HeartRate < 0 || point.HeartRate < b.hrMin.HeartRate) {
			b.hrMin = &point
		}
		if point.HeartRate >= 0 && (b.hrMax == nil || point.HeartRate > b.hrMax.HeartRate) {
			b.hrMax = &point
		}
		if point.BreathRate >= 0 && (b.breathMin == nil || b.breathMin.BreathRate < 0 || point.BreathRate < b.breathMin.BreathRate) {
			b.breathMin = &point
		}
		if point.BreathRate >= 0 && (b.breathMax == nil || point.BreathRate > b.breathMax.BreathRate) {
			b.breathMax = &point
		}
		b.last = &point
		buckets[idx] = *b
	}

	result := make([]model.WardSensorDataPoint, 0, maxBuckets*4)
	seen := make(map[int64]struct{}, maxBuckets*4)
	add := func(point *model.WardSensorDataPoint) {
		if point == nil {
			return
		}
		key := point.DataID
		if key == 0 {
			key = point.Timestamp.UnixNano()
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, *point)
	}

	for i := range buckets {
		b := buckets[i]
		add(b.first)
		add(b.tempMin)
		add(b.tempMax)
		add(b.hrMin)
		add(b.hrMax)
		add(b.breathMin)
		add(b.breathMax)
		add(b.last)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result
}

// GetSensorThermalTimeline returns data_id/timestamp only for thermal slider use.
// GetSensorThermalTimeline 只回傳熱像時間軸，不回傳 768 點影像資料。
func (r *SQLSensorRepository) GetSensorThermalTimeline(ctx context.Context, sensorNumber string, start, end *time.Time) (model.WardThermalTimelineResponse, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(queryCtx, `
		SELECT frame_id, "timestamp"
		FROM (
		  SELECT DISTINCT ON (date_bin('10 seconds'::interval, tf."timestamp", '1970-01-01'::timestamp))
		    tf.frame_id,
		    tf."timestamp",
		    date_bin('10 seconds'::interval, tf."timestamp", '1970-01-01'::timestamp) AS bucket_ts,
		    tf.high_temperature
		  FROM sensor_thermal_frames tf
		  JOIN sensors s ON s.sensor_id = tf.sensor_id
		  WHERE s.sensor_number = $1
		    AND ($2::timestamp IS NULL OR tf."timestamp" >= $2)
		    AND ($3::timestamp IS NULL OR tf."timestamp" <= $3)
		  ORDER BY bucket_ts, tf.high_temperature DESC, tf."timestamp" DESC
		) picked
		ORDER BY "timestamp" ASC`, sensorNumber, start, end)
	if err != nil {
		return model.WardThermalTimelineResponse{}, err
	}
	defer rows.Close()

	resp := model.WardThermalTimelineResponse{
		SensorNumber: sensorNumber,
		Start:        start,
		End:          end,
		Frames:       []model.WardThermalTimelineFrame{},
	}
	for rows.Next() {
		var frame model.WardThermalTimelineFrame
		if err := rows.Scan(&frame.DataID, &frame.Timestamp); err != nil {
			return model.WardThermalTimelineResponse{}, err
		}
		resp.Frames = append(resp.Frames, frame)
	}
	if err := rows.Err(); err != nil {
		return model.WardThermalTimelineResponse{}, err
	}
	return resp, nil
}

// GetSensorThermalFrame returns one thermal frame by frame_id.
// GetSensorThermalFrame 依 frame_id 回傳單張熱像 frame。
func (r *SQLSensorRepository) GetSensorThermalFrame(ctx context.Context, sensorNumber string, frameID int64) (*model.WardSensorDataPoint, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	var (
		id              sql.NullInt64
		timestamp       sql.NullTime
		heartRate       sql.NullInt64
		rhythm          sql.NullString
		breathRate      sql.NullInt64
		outOfRange      sql.NullInt64
		highTemperature sql.NullFloat64
		temperatureJSON sql.NullString
	)
	err := r.db.QueryRowContext(queryCtx, `
		SELECT tf.frame_id, tf."timestamp", sd.heart_rate, sd.rhythm, sd.breath_rate, sd.out_of_range, tf.high_temperature, tf.frame_json::text
		FROM sensor_thermal_frames tf
		JOIN sensors s ON s.sensor_id = tf.sensor_id
		LEFT JOIN sensor_datas sd ON sd.data_id = tf.sensor_data_id AND sd.sensor_id = tf.sensor_id
		WHERE s.sensor_number = $1
		  AND tf.frame_id = $2
		LIMIT 1`, sensorNumber, frameID).Scan(&id, &timestamp, &heartRate, &rhythm, &breathRate, &outOfRange, &highTemperature, &temperatureJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return buildWardDataPoint(id, timestamp, heartRate, rhythm, breathRate, outOfRange, highTemperature, temperatureJSON), nil
}

// GetSensorLatestThermalFrame returns the newest thermal frame for one sensor.
// GetSensorLatestThermalFrame 回傳單一 sensor 最新熱像 frame。
func (r *SQLSensorRepository) GetSensorLatestThermalFrame(ctx context.Context, sensorNumber string) (*model.WardSensorDataPoint, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.cfg.DBQueryTimeout)
	defer cancel()

	var (
		id              sql.NullInt64
		timestamp       sql.NullTime
		heartRate       sql.NullInt64
		rhythm          sql.NullString
		breathRate      sql.NullInt64
		outOfRange      sql.NullInt64
		highTemperature sql.NullFloat64
		temperatureJSON sql.NullString
	)
	err := r.db.QueryRowContext(queryCtx, `
		SELECT tf.frame_id, tf."timestamp", sd.heart_rate, sd.rhythm, sd.breath_rate, sd.out_of_range, tf.high_temperature, tf.frame_json::text
		FROM sensor_thermal_frames tf
		JOIN sensors s ON s.sensor_id = tf.sensor_id
		LEFT JOIN sensor_datas sd ON sd.data_id = tf.sensor_data_id AND sd.sensor_id = tf.sensor_id
		WHERE s.sensor_number = $1
		ORDER BY tf."timestamp" DESC
		LIMIT 1`, sensorNumber).Scan(&id, &timestamp, &heartRate, &rhythm, &breathRate, &outOfRange, &highTemperature, &temperatureJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return buildWardDataPoint(id, timestamp, heartRate, rhythm, breathRate, outOfRange, highTemperature, temperatureJSON), nil
}

// buildSensorInsertSQL builds device-type-specific INSERT SQL and arguments.
// buildSensorInsertSQL 建立設備類型對應的 INSERT SQL 與參數列表。
// For PostgreSQL it emits numbered placeholders ($1..$n), otherwise '?'.
// PostgreSQL 使用編號 placeholder（$1..$n），其他方言使用 '?'。
func buildSensorInsertSQL(dialect, table string, rows []model.SensorInsertRow) (string, []any) {
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (device_type, sensor_id, breath_rate, breath_rate_state, heart_rate, rhythm, out_of_range, angles_first, angles_second, movement_state, movement_level, breath_rate_last_transmit, heart_rate_last_transmit, distance, thermistor, high_temperature, timestamp) VALUES ")

	args := make([]any, 0, len(rows)*16)
	argIndex := 1
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j := 0; j < 16; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			if dialect == "postgres" {
				sb.WriteString(fmt.Sprintf("$%d", argIndex))
				argIndex++
			} else {
				sb.WriteString("?")
			}
		}
		sb.WriteString(", NOW())")

		// Keep args order exactly aligned with placeholder order.
		// 參數順序必須與 placeholder 順序完全一致。
		args = append(args,
			string(row.DeviceType),
			row.SensorID,
			row.BreathRate,
			row.BreathRateState,
			row.HeartRate,
			nullableString(row.Rhythm),
			row.OutOfRange,
			row.AnglesFirst,
			row.AnglesSecond,
			row.MovementState,
			row.MovementLevel,
			row.BreathRateLastTransmit,
			row.HeartRateLastTransmit,
			row.Distance,
			row.Thermistor,
			row.HighTemperature,
		)
	}
	sb.WriteString(` RETURNING data_id, "timestamp"`)
	return sb.String(), args
}

// InsertThermalFrameBatch inserts decimated thermal frames into the dedicated table.
// InsertThermalFrameBatch 將降頻後的熱像 frame 寫入獨立資料表。
func (r *SQLSensorRepository) InsertThermalFrameBatch(ctx context.Context, rows []model.ThermalFrameInsertRow) error {
	if len(rows) == 0 {
		return nil
	}
	insertCtx, cancel := context.WithTimeout(ctx, r.cfg.DBInsertTimeout)
	defer cancel()

	var sb strings.Builder
	sb.WriteString(`INSERT INTO sensor_thermal_frames (sensor_id, sensor_data_id, high_temperature, frame_json, "timestamp") VALUES `)
	args := make([]any, 0, len(rows)*5)
	argIndex := 1
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j := 0; j < 5; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", argIndex))
			argIndex++
		}
		sb.WriteString(")")
		args = append(args, row.SensorID, row.SensorDataID, row.HighTemperature, row.FrameJSON, row.Timestamp)
	}
	_, err := r.db.ExecContext(insertCtx, sb.String(), args...)
	return err
}

// buildRawLogInsertSQL builds INSERT SQL for unified raw log rows.
// buildRawLogInsertSQL 建立整合 raw log 的批次寫入 SQL。
func buildRawLogInsertSQL(dialect, table string, deviceType model.DeviceType, rows []model.RawLogInsertRow) (string, []any) {
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (device_type, sensor_number, client_ip, raw_content, processed_content, processed_status, sensor_data_id, sensor_data_timestamp) VALUES ")

	args := make([]any, 0, len(rows)*8)
	argIndex := 1
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j := 0; j < 8; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			if dialect == "postgres" {
				sb.WriteString(fmt.Sprintf("$%d", argIndex))
				argIndex++
			} else {
				sb.WriteString("?")
			}
		}
		sb.WriteString(")")
		// Keep args order exactly aligned with placeholder order.
		// 參數順序必須與 placeholder 順序完全一致。
		args = append(args,
			string(deviceType),
			row.SensorNumber,
			row.ClientIP,
			row.RawContent,
			row.ProcessedContent,
			row.ProcessedStatus,
			row.SensorDataID,
			row.SensorDataTimestamp,
		)
	}
	return sb.String(), args
}

func nullableInt64(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func buildWardDataPoint(dataID sql.NullInt64, ts sql.NullTime, heartRate sql.NullInt64, rhythm sql.NullString, breathRate sql.NullInt64, outOfRange sql.NullInt64, highTemperature sql.NullFloat64, temperatureJSON sql.NullString) *model.WardSensorDataPoint {
	if !dataID.Valid || !ts.Valid {
		return nil
	}
	temp := 0.0
	if highTemperature.Valid {
		temp = highTemperature.Float64
	}
	hr := int(heartRate.Int64)
	br := int(breathRate.Int64)
	sys, dia := estimateBloodPressure(hr, temp)
	out := 0
	if outOfRange.Valid {
		out = int(outOfRange.Int64)
	}
	rhythmText := strings.TrimSpace(rhythm.String)
	if !rhythm.Valid || rhythmText == "" {
		rhythmText = estimateRhythm(hr)
	}
	return &model.WardSensorDataPoint{
		DataID:          dataID.Int64,
		Timestamp:       ts.Time,
		HeartRate:       hr,
		BreathRate:      br,
		Temperature:     temp,
		HighTemperature: temp,
		TemperatureJSON: parseTemperatureJSON(temperatureJSON),
		BPSys:           sys,
		BPDia:           dia,
		Rhythm:          rhythmText,
		OutOfRange:      out,
		Presence:        true,
	}
}

func parseTemperatureJSON(raw sql.NullString) []float64 {
	if !raw.Valid {
		return nil
	}
	content := strings.TrimSpace(raw.String)
	if content == "" {
		return nil
	}
	values := []float64{}
	if err := json.Unmarshal([]byte(content), &values); err != nil || len(values) == 0 {
		return nil
	}
	return values
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// toPostgresPlaceholders converts '?' placeholders into '$n' style.
// toPostgresPlaceholders 將 '?' 轉成 PostgreSQL 的 '$n' 形式。
func toPostgresPlaceholders(query string) string {
	var sb strings.Builder
	idx := 1
	// Replace each '?' with '$n' in scan order.
	// 依掃描順序將每個 '?' 替換為 '$n'。
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			sb.WriteString(fmt.Sprintf("$%d", idx))
			idx++
			continue
		}
		sb.WriteByte(query[i])
	}
	return sb.String()
}

// openDB creates PostgreSQL connection from PG_* settings.
// openDB 使用 PG_* 設定建立 PostgreSQL 連線。
func openDB(cfg model.AppConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return nil, err
	}
	return db, nil
}

// configureDBPool applies connection pool sizing from config.
// configureDBPool 套用設定檔中的 DB 連線池參數。
func configureDBPool(db *sql.DB, cfg model.AppConfig) {
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
}
