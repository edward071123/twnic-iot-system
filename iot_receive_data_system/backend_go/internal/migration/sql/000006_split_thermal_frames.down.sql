ALTER TABLE sensor_datas
ADD COLUMN IF NOT EXISTS temperature_json JSONB NOT NULL DEFAULT '[]'::jsonb;

DROP FUNCTION IF EXISTS ensure_sensor_thermal_monthly_partitions(INTEGER, INTEGER);
DROP TABLE IF EXISTS sensor_thermal_frames CASCADE;
