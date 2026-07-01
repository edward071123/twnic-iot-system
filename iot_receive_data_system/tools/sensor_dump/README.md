# sensor_dump

Parallel dump/restore tool for sensor analysis data.

It wraps PostgreSQL directory-format `pg_dump` / `pg_restore` so large tables can be dumped with `--jobs`, compressed, and restored later.

Default dump contents:

- `sensors`
- `sensor_datas*`
- `sensor_thermal_frames*`

`sensors` is included so `sensor_id` can still map back to `sensor_number` after restore.

## Dump

```bash
docker compose run --rm sensor_dump dump --out /dumps/sensor_analysis.tar.gz --jobs 8
```

## Restore

Stop writers first if restoring into the running project:

```bash
docker compose stop mock_go backend_go
docker compose run --rm sensor_dump restore --in /dumps/sensor_analysis.tar.gz --jobs 8 --truncate
docker compose up -d backend_go mock_go
```

`--truncate` clears `sensor_thermal_frames`, `sensor_datas`, and `sensors` before restore.

## Only The Two Large Tables

If the target DB already has matching `sensors.sensor_id` values:

```bash
docker compose run --rm sensor_dump dump --out /dumps/sensor_tables_only.tar.gz --jobs 8 --without-sensors
```
