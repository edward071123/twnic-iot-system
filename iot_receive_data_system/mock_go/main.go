package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// AppConfig defines runtime knobs for mock sender.
// AppConfig 定義 mock 發送器的執行期參數。
type AppConfig struct {
	PGHost     string
	PGPort     string
	PGUser     string
	PGPassword string
	PGDBName   string

	APIBaseURL      string
	TargetSensors   int
	TemperatureSize int
	TickInterval    time.Duration
	Workers         int
	HTTPTimeout     time.Duration
	BaselineEnabled bool
	ScenarioEnabled bool
	ScenarioRate    int
	OutOfBedSensors map[string]struct{}
	OfflineSensors  map[string]struct{}
	WarningSensors  map[string]struct{}
}

// MockSensor is one fixed sender identity from mock_sensors table.
// MockSensor 代表 mock_sensors 表內的一個固定發送身分。
type MockSensor struct {
	SensorIP     string
	SensorNumber string
	DeviceType   string
}

// ingestPayload matches backend ingest JSON schema.
// ingestPayload 對應後端 ingest JSON 結構。
type ingestPayload struct {
	SensorNumber            string    `json:"sensor_number"`
	SensorNo                string    `json:"sensorNo"`
	DeviceType              string    `json:"devicetype"`
	Mac                     string    `json:"mac"`
	MOutOfRange             int       `json:"m_outOfRange"`
	MDistance               float64   `json:"m_distance"`
	MAnglesFirst            int       `json:"m_angles_first"`
	MAnglesSecond           int       `json:"m_angles_second"`
	MMovementState          int       `json:"m_movementState"`
	MMovementLevel          int       `json:"m_movementLevel"`
	MBreathRateState        int       `json:"m_breath_rate_state"`
	MBreathRateLastTransmit int64     `json:"m_breath_rate_last_Transmit"`
	MBreathRate             int       `json:"m_breath_rate"`
	MHeartRateState         int       `json:"m_heart_rate_state"`
	MHeartRateLastTransmit  int64     `json:"m_heart_rate_last_Transmit"`
	MHeartRate              int       `json:"m_heart_rate"`
	HeartRate               int       `json:"heart_rate"`
	BreathRate              int       `json:"breath_rate"`
	Thermistor              float64   `json:"thermistor"`
	Temperature             []float64 `json:"temperature"`
}

type scenarioKind int

const (
	defaultThermalPointCount = 24 * 32

	scenarioStandard scenarioKind = iota
	scenarioNormalFragmentation
	scenarioStaleBufferRecovery
	scenarioLeadingGarbage
	scenarioMalformedJunk
	scenarioCorruptedConcat
	scenarioKindCount
)

func (k scenarioKind) label() string {
	switch k {
	case scenarioStandard:
		return "standard_json"
	case scenarioNormalFragmentation:
		return "normal_fragmentation"
	case scenarioStaleBufferRecovery:
		return "stale_buffer_recovery"
	case scenarioLeadingGarbage:
		return "leading_garbage"
	case scenarioMalformedJunk:
		return "malformed_junk_logging"
	case scenarioCorruptedConcat:
		return "corrupted_concatenation"
	default:
		return "unknown"
	}
}

type tickStats struct {
	Sensors        int
	Requests       int64
	Success        int64
	Fail           int64
	BaselineSent   int64
	ScenarioSent   int64
	StartSkew      time.Duration
	ScenarioCounts [scenarioKindCount]int64
}

func main() {
	cfg := loadConfig()

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := openDBWithRetry(rootCtx, cfg, 40, 1500*time.Millisecond)
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	defer db.Close()

	sensors, err := loadMockSensors(rootCtx, db, cfg.TargetSensors)
	if err != nil {
		log.Fatalf("load mock sensors failed: %v", err)
	}

	log.Printf(
		"mock sender started target=%d tick=%s workers=%d temperature_points=%d api=%s baseline_enabled=%t scenario_enabled=%t scenario_rate=%d",
		len(sensors),
		cfg.TickInterval,
		cfg.Workers,
		cfg.TemperatureSize,
		strings.TrimRight(cfg.APIBaseURL, "/"),
		cfg.BaselineEnabled,
		cfg.ScenarioEnabled,
		cfg.ScenarioRate,
	)

	client := &http.Client{
		Timeout: cfg.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Workers * 4,
			MaxIdleConnsPerHost: cfg.Workers * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	ticker := time.NewTicker(cfg.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rootCtx.Done():
			log.Printf("mock sender stopped: %v", rootCtx.Err())
			return
		case tickAt := <-ticker.C:
			start := time.Now()
			stats := sendOneTick(rootCtx, client, cfg, sensors, tickAt)
			log.Printf(
				"tick=%s sensors=%d baseline=%d scenario_requests=%d requests=%d success=%d fail=%d start_skew=%s duration=%s scenarios={%s}",
				tickAt.UTC().Format(time.RFC3339),
				stats.Sensors,
				stats.BaselineSent,
				stats.ScenarioSent,
				stats.Requests,
				stats.Success,
				stats.Fail,
				stats.StartSkew,
				time.Since(start),
				formatScenarioSummary(stats.ScenarioCounts),
			)
		}
	}
}

func loadConfig() AppConfig {
	targetSensors := envIntOrDefault("MOCK_TARGET_SENSORS", 108)
	return AppConfig{
		PGHost:          envOrDefault("PG_HOST", "127.0.0.1"),
		PGPort:          envOrDefault("PG_PORT", "5432"),
		PGUser:          envOrDefault("PG_USER", "postgres"),
		PGPassword:      envOrDefault("PG_PASSWORD", "postgres"),
		PGDBName:        envOrDefault("PG_DB_NAME", ""),
		APIBaseURL:      envOrDefault("MOCK_API_BASE_URL", "http://127.0.0.1:3002"),
		TargetSensors:   targetSensors,
		TemperatureSize: envIntOrDefault("MOCK_TEMPERATURE_POINTS", defaultThermalPointCount),
		TickInterval:    time.Duration(envIntOrDefault("MOCK_TICK_MS", 1000)) * time.Millisecond,
		Workers:         envIntOrDefault("MOCK_WORKERS", targetSensors),
		HTTPTimeout:     time.Duration(envIntOrDefault("MOCK_HTTP_TIMEOUT_MS", 900)) * time.Millisecond,
		BaselineEnabled: envBoolOrDefault("MOCK_BASELINE_ENABLED", true),
		ScenarioEnabled: envBoolOrDefault("MOCK_SCENARIO_ENABLED", false),
		ScenarioRate:    envPercentOrDefault("MOCK_SCENARIO_RATE", 0),
		OutOfBedSensors: envSetOrDefault("MOCK_OUT_OF_BED_SENSORS", "0201_01,0203_03,0205_05"),
		OfflineSensors:  envSetOrDefault("MOCK_OFFLINE_SENSORS", "0201_02,0203_04,0205_06"),
		WarningSensors:  envSetOrDefault("MOCK_WARNING_SENSORS", "0207_01,0207_03,0209_01"),
	}
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

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

func envPercentOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return clampPercent(fallback)
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return clampPercent(fallback)
	}
	return clampPercent(n)
}

func envSetOrDefault(key, fallback string) map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = fallback
	}
	result := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		result[value] = struct{}{}
	}
	return result
}

func clampPercent(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func (c AppConfig) dsn() string {
	return strings.TrimSpace(
		"host=" + c.PGHost +
			" port=" + c.PGPort +
			" user=" + c.PGUser +
			" password=" + c.PGPassword +
			" dbname=" + c.PGDBName +
			" sslmode=disable TimeZone=UTC",
	)
}

func openDBWithRetry(ctx context.Context, cfg AppConfig, maxAttempts int, wait time.Duration) (*sql.DB, error) {
	var lastErr error
	for i := 1; i <= maxAttempts; i++ {
		db, err := sql.Open("pgx", cfg.dsn())
		if err != nil {
			lastErr = err
		} else {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = db.PingContext(pingCtx)
			cancel()
			if err == nil {
				return db, nil
			}
			_ = db.Close()
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, fmt.Errorf("db not ready after %d attempts: %w", maxAttempts, lastErr)
}

func loadMockSensors(ctx context.Context, db *sql.DB, target int) ([]MockSensor, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(
		queryCtx,
		`SELECT sensor_ip, sensor_number, device_type
		 FROM mock_sensors
		 ORDER BY sensor_number`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sensors := make([]MockSensor, 0, target)
	for rows.Next() {
		var s MockSensor
		if err := rows.Scan(&s.SensorIP, &s.SensorNumber, &s.DeviceType); err != nil {
			return nil, err
		}
		s.DeviceType = strings.TrimSpace(strings.ToLower(s.DeviceType))
		sensors = append(sensors, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(sensors) < target {
		return nil, fmt.Errorf("mock_sensors not enough: have=%d need=%d", len(sensors), target)
	}
	if len(sensors) > target {
		log.Printf("mock_sensors larger than target: have=%d use_first=%d", len(sensors), target)
		sensors = sensors[:target]
	}
	return sensors, nil
}

func sendOneTick(ctx context.Context, client *http.Client, cfg AppConfig, sensors []MockSensor, tickAt time.Time) tickStats {
	var requests int64
	var success int64
	var fail int64
	var baselineSent int64
	var scenarioSent int64
	var scenarioCounts [scenarioKindCount]int64
	var wg sync.WaitGroup
	var ready sync.WaitGroup
	sem := make(chan struct{}, cfg.Workers)
	startGate := make(chan struct{})
	firstStart := int64(0)
	lastStart := int64(0)

	for idx, sensor := range sensors {
		wg.Add(1)
		ready.Add(1)
		go func(i int, s MockSensor) {
			defer wg.Done()
			ready.Done()

			select {
			case <-ctx.Done():
				atomic.AddInt64(&fail, 1)
				return
			case <-startGate:
			}

			startedAt := time.Now().UnixNano()
			if atomic.CompareAndSwapInt64(&firstStart, 0, startedAt) {
				atomic.StoreInt64(&lastStart, startedAt)
			} else {
				for {
					current := atomic.LoadInt64(&lastStart)
					if startedAt <= current || atomic.CompareAndSwapInt64(&lastStart, current, startedAt) {
						break
					}
				}
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			if _, offline := cfg.OfflineSensors[s.SensorNumber]; offline {
				return
			}

			if cfg.BaselineEnabled {
				_, outOfBed := cfg.OutOfBedSensors[s.SensorNumber]
				_, warning := cfg.WarningSensors[s.SensorNumber]
				chunk, err := buildStandardPayloadJSON(s, outOfBed, warning, tickAt, rand.New(rand.NewSource(tickAt.UnixNano()+int64(i)*7919)), cfg.TemperatureSize)
				if err != nil {
					atomic.AddInt64(&fail, 1)
					return
				}
				atomic.AddInt64(&baselineSent, 1)
				atomic.AddInt64(&requests, 1)
				if sendChunk(ctx, client, cfg, s, chunk) {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&fail, 1)
				}
			}

			kind, chunks, ok, err := buildScenarioChunks(cfg, s, tickAt, i)
			if err != nil {
				atomic.AddInt64(&fail, 1)
				return
			}
			if !ok {
				return
			}
			atomic.AddInt64(&scenarioCounts[int(kind)], 1)

			for _, chunk := range chunks {
				atomic.AddInt64(&scenarioSent, 1)
				atomic.AddInt64(&requests, 1)
				if sendChunk(ctx, client, cfg, s, chunk) {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&fail, 1)
				}
			}
		}(idx, sensor)
	}

	ready.Wait()
	close(startGate)
	wg.Wait()

	var startSkew time.Duration
	if first := atomic.LoadInt64(&firstStart); first > 0 {
		startSkew = time.Duration(atomic.LoadInt64(&lastStart) - first)
	}

	stats := tickStats{
		Sensors:      len(sensors),
		Requests:     atomic.LoadInt64(&requests),
		Success:      atomic.LoadInt64(&success),
		Fail:         atomic.LoadInt64(&fail),
		BaselineSent: atomic.LoadInt64(&baselineSent),
		ScenarioSent: atomic.LoadInt64(&scenarioSent),
		StartSkew:    startSkew,
	}
	for i := 0; i < int(scenarioKindCount); i++ {
		stats.ScenarioCounts[i] = atomic.LoadInt64(&scenarioCounts[i])
	}
	return stats
}

func sendChunk(ctx context.Context, client *http.Client, cfg AppConfig, sensor MockSensor, chunk string) bool {
	url := strings.TrimRight(cfg.APIBaseURL, "/") + "/sensor/data/v2/" + sensor.DeviceType
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(chunk)))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", sensor.SensorIP)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func buildScenarioChunks(cfg AppConfig, sensor MockSensor, tickAt time.Time, sequence int) (scenarioKind, []string, bool, error) {
	rng := rand.New(rand.NewSource(tickAt.UnixNano() + int64(sequence)*7919))

	useScenario := cfg.ScenarioEnabled && rng.Intn(100) < cfg.ScenarioRate
	if !useScenario {
		return scenarioStandard, nil, false, nil
	}

	kind := chooseRandomScenario(rng)
	heartRate, breathRate := randomRates(rng)
	temps := randomTemperatureArray(rng, cfg.TemperatureSize)
	tempJSON, err := marshalJSONToString(temps)
	if err != nil {
		return kind, nil, false, err
	}

	switch kind {
	case scenarioNormalFragmentation:
		chunk1 := fmt.Sprintf(
			`{"sensor_number":"%s","sensorNo":"%s","devicetype":"%s","heart_rate":`,
			sensor.SensorNumber,
			sensor.SensorNumber,
			sensor.DeviceType,
		)
		chunk2 := fmt.Sprintf(
			`%d,"breath_rate":%d,"temperature":%s}`,
			heartRate,
			breathRate,
			tempJSON,
		)
		return kind, []string{chunk1, chunk2}, true, nil

	case scenarioStaleBufferRecovery:
		recoveryHeart, recoveryBreath := randomRates(rng)
		recoveryTemps := randomTemperatureArray(rng, cfg.TemperatureSize)
		recoveryJSON, err := buildMinimalPayloadJSON(sensor, recoveryHeart, recoveryBreath, recoveryTemps)
		if err != nil {
			return kind, nil, false, err
		}
		chunk1 := fmt.Sprintf(`{"sensorNo":"%s","sensor_number":"%s","broken": `,
			sensor.SensorNumber,
			sensor.SensorNumber,
		)
		return kind, []string{chunk1, recoveryJSON}, true, nil

	case scenarioLeadingGarbage:
		validJSON, err := buildMinimalPayloadJSON(sensor, heartRate, breathRate, temps)
		if err != nil {
			return kind, nil, false, err
		}
		chunk := fmt.Sprintf(`GARBAGE_%d%s`, rng.Intn(1_000_000), validJSON)
		return kind, []string{chunk}, true, nil

	case scenarioMalformedJunk:
		validJSON, err := buildMinimalPayloadJSON(sensor, heartRate, breathRate, temps)
		if err != nil {
			return kind, nil, false, err
		}
		junk := `"{389238 , dhhdewiwehui {cdsoi,cdshcis}  "`
		return kind, []string{junk, validJSON}, true, nil

	case scenarioCorruptedConcat:
		heartRate2, breathRate2 := randomRates(rng)
		temps2 := randomTemperatureArray(rng, cfg.TemperatureSize)
		tempJSON2, err := marshalJSONToString(temps2)
		if err != nil {
			return kind, nil, false, err
		}
		trimmedTemp := strings.TrimSuffix(tempJSON, "]")
		chunk := fmt.Sprintf(
			`{"heart_rate":%d,"breath_rate":-1,"temperature":%s,2{"heart_rate":%d,"breath_rate":%d,"temperature":%s,"sensorNo":"%s","sensor_number":"%s","devicetype":"%s"}`,
			heartRate,
			trimmedTemp,
			heartRate2,
			breathRate2,
			tempJSON2,
			sensor.SensorNumber,
			sensor.SensorNumber,
			sensor.DeviceType,
		)
		return kind, []string{chunk}, true, nil

	default:
		return scenarioStandard, nil, false, nil
	}
}

func buildStandardPayloadJSON(sensor MockSensor, outOfBed, warning bool, tickAt time.Time, rng *rand.Rand, temperatureSize int) (string, error) {
	movementState := 1
	movementLevel := 35

	distance := 60.0
	outOfRange := 0
	heartRate, breathRate, thermistor := normalVitals(sensor.SensorNumber, tickAt)
	temps := normalTemperatureArray(sensor.SensorNumber, tickAt, rng, temperatureSize)

	if outOfBed {
		outOfRange = 1
		movementState = 0
		movementLevel = 0
		distance = 0
		heartRate = 0
		breathRate = 0
		thermistor = 0
		temps = []float64{0}
	} else if warning {
		heartRate, breathRate, thermistor = warningVitals(sensor.SensorNumber, tickAt)
		temps = warningTemperatureArray(sensor.SensorNumber, tickAt, temperatureSize)
	}

	now := time.Now().Unix()

	payload := ingestPayload{
		SensorNumber:            sensor.SensorNumber,
		SensorNo:                sensor.SensorNumber,
		DeviceType:              sensor.DeviceType,
		Mac:                     "none",
		MOutOfRange:             outOfRange,
		MDistance:               distance,
		MAnglesFirst:            45,
		MAnglesSecond:           90,
		MMovementState:          movementState,
		MMovementLevel:          movementLevel,
		MBreathRateState:        0,
		MBreathRateLastTransmit: now,
		MBreathRate:             breathRate,
		MHeartRateState:         0,
		MHeartRateLastTransmit:  now,
		MHeartRate:              heartRate,
		HeartRate:               heartRate,
		BreathRate:              breathRate,
		Thermistor:              thermistor,
		Temperature:             temps,
	}
	return marshalJSONToString(payload)
}

func normalVitals(sensorNumber string, tickAt time.Time) (int, int, float64) {
	sum := 0
	for _, ch := range sensorNumber {
		sum += int(ch)
	}
	phase := float64(sum%360) * math.Pi / 180
	seconds := float64(tickAt.Unix())
	heartRate := 74 + int(math.Round(5*math.Sin(seconds/7.0+phase)))
	breathRate := 17 + int(math.Round(2*math.Sin(seconds/11.0+phase/2)))
	temperature := roundTo(36.45+0.28*math.Sin(seconds/13.0+phase/3), 2)
	return heartRate, breathRate, temperature
}

func warningVitals(sensorNumber string, tickAt time.Time) (int, int, float64) {
	sum := 0
	for _, ch := range sensorNumber {
		sum += int(ch)
	}
	phase := float64(sum%360) * math.Pi / 180
	seconds := float64(tickAt.Unix())
	heartRate := 106 + int(math.Round(10*math.Sin(seconds/9.0+phase)))
	breathRate := 23 + int(math.Round(5*math.Sin(seconds/11.0+phase/2)))
	temperature := roundTo(37.4+0.72*math.Sin(seconds/13.0+phase/3), 2)
	return heartRate, breathRate, temperature
}

func buildMinimalPayloadJSON(sensor MockSensor, heartRate, breathRate int, temps []float64) (string, error) {
	now := time.Now().Unix()
	payload := map[string]any{
		"sensor_number":               sensor.SensorNumber,
		"sensorNo":                    sensor.SensorNumber,
		"devicetype":                  sensor.DeviceType,
		"heart_rate":                  heartRate,
		"breath_rate":                 breathRate,
		"temperature":                 temps,
		"m_distance":                  10.5,
		"thermistor":                  36.5,
		"m_outOfRange":                0,
		"m_angles_first":              45,
		"m_angles_second":             90,
		"m_movementState":             1,
		"m_movementLevel":             50,
		"m_breath_rate_state":         0,
		"m_breath_rate_last_Transmit": now,
		"m_heart_rate_last_Transmit":  now,
	}
	return marshalJSONToString(payload)
}

func chooseRandomScenario(rng *rand.Rand) scenarioKind {
	switch rng.Intn(5) {
	case 0:
		return scenarioNormalFragmentation
	case 1:
		return scenarioStaleBufferRecovery
	case 2:
		return scenarioLeadingGarbage
	case 3:
		return scenarioMalformedJunk
	default:
		return scenarioCorruptedConcat
	}
}

func randomRates(rng *rand.Rand) (int, int) {
	heartRate := 55 + rng.Intn(66)
	breathRate := 10 + rng.Intn(16)
	if rng.Intn(20) == 0 {
		breathRate = -1
	}
	if rng.Intn(20) == 0 {
		heartRate = -1
	}
	return heartRate, breathRate
}

func formatScenarioSummary(counts [scenarioKindCount]int64) string {
	parts := make([]string, 0, int(scenarioKindCount))
	for i := 0; i < int(scenarioKindCount); i++ {
		k := scenarioKind(i)
		parts = append(parts, fmt.Sprintf("%s=%d", k.label(), counts[i]))
	}
	return strings.Join(parts, ", ")
}

func marshalJSONToString(v any) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func randomTemperatureArray(rng *rand.Rand, size int) []float64 {
	out := make([]float64, 0, size)
	base := 36.2 + rng.Float64()*0.8
	for i := 0; i < size; i++ {
		v := base + (rng.Float64()*0.8 - 0.4)
		if rng.Intn(80) == 0 {
			v = 37.8 + rng.Float64()*1.7
		}
		if rng.Intn(120) == 0 {
			v = 35.0 + rng.Float64()*0.7
		}
		out = append(out, roundQuarter(v))
	}
	return out
}

func normalTemperatureArray(sensorNumber string, tickAt time.Time, rng *rand.Rand, size int) []float64 {
	out := make([]float64, 0, size)
	sum := 0
	for _, ch := range sensorNumber {
		sum += int(ch)
	}
	phase := float64(sum%360) * math.Pi / 180
	seconds := float64(tickAt.Unix())
	base := 36.45 + 0.28*math.Sin(seconds/13.0+phase/3)
	for i := 0; i < size; i++ {
		spatial := 0.08 * math.Sin(float64(i)/19.0+phase)
		noise := rng.Float64()*0.08 - 0.04
		out = append(out, roundTo(base+spatial+noise, 2))
	}
	return out
}

func warningTemperatureArray(sensorNumber string, tickAt time.Time, size int) []float64 {
	out := make([]float64, 0, size)
	sum := 0
	for _, ch := range sensorNumber {
		sum += int(ch)
	}
	phase := float64(sum%360) * math.Pi / 180
	seconds := float64(tickAt.Unix())
	base := 37.4 + 0.72*math.Sin(seconds/13.0+phase/3)
	for i := 0; i < size; i++ {
		spatial := 0.08 * math.Sin(float64(i)/19.0+phase)
		out = append(out, roundTo(base+spatial, 2))
	}
	return out
}

func roundQuarter(v float64) float64 {
	return math.Round(v*4.0) / 4.0
}

func roundTo(v float64, digits int) float64 {
	scale := math.Pow10(digits)
	return math.Round(v*scale) / scale
}
