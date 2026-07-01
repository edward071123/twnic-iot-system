CREATE INDEX IF NOT EXISTS idx_sensor_datas_sensor_id_ts_desc
ON sensor_datas (sensor_id, "timestamp" DESC);
