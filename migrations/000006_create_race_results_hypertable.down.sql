-- Drop race_results hypertable and continuous aggregate

-- Drop continuous aggregate
DROP MATERIALIZED VIEW IF EXISTS race_results_daily CASCADE;

-- Drop the hypertable (cascades to all indexes)
DROP TABLE IF EXISTS race_results CASCADE;
