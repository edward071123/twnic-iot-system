-- [EN] Reverse migration for 000001_init_schema.up.sql.
-- [ZH] 000001_init_schema.up.sql 的反向還原腳本。

-- [EN] Drop trigger/function first, then drop tables in dependency-safe order.
-- [ZH] 先刪除 trigger/function，再依相依安全順序刪除資料表。
DROP TRIGGER IF EXISTS trg_patient_set_updated_at ON patient;
DROP FUNCTION IF EXISTS set_utc_updated_at();
DROP FUNCTION IF EXISTS prune_sensor_raw_log_partitions(INTEGER);
DROP FUNCTION IF EXISTS ensure_sensor_raw_log_partitions(INTEGER, INTEGER);
DROP FUNCTION IF EXISTS ensure_sensor_data_monthly_partitions(INTEGER, INTEGER);

DROP TABLE IF EXISTS temperature_calibration CASCADE;
DROP TABLE IF EXISTS is_abnormal CASCADE;
DROP TABLE IF EXISTS sensor_raw_logs CASCADE;
DROP TABLE IF EXISTS sensor_datas CASCADE;
DROP TABLE IF EXISTS patient CASCADE;
DROP TABLE IF EXISTS sensors CASCADE;
DROP TABLE IF EXISTS mock_sensors CASCADE;
DROP TABLE IF EXISTS rooms CASCADE;
DROP TABLE IF EXISTS floors CASCADE;
DROP TABLE IF EXISTS employee CASCADE;
DROP TABLE IF EXISTS hospitals CASCADE;
