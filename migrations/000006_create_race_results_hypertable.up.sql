-- Create race_results hypertable for time-series race outcome data
-- Requires TimescaleDB extension to be installed

CREATE TABLE IF NOT EXISTS race_results (
    time TIMESTAMPTZ NOT NULL,
    race_id UUID NOT NULL,
    winner_trap INT,
    positions JSONB,
    total_payouts DECIMAL(18, 2),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create the hypertable with time as the primary dimension
SELECT create_hypertable('race_results', 'time', if_not_exists => TRUE);

-- Add spatial dimension for race_id to enable better data distribution
SELECT add_dimension('race_results', by_hash('race_id', 32), if_not_exists => TRUE);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_race_results_race_id ON race_results(race_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_race_results_status ON race_results(status, time DESC);
CREATE INDEX IF NOT EXISTS idx_race_results_time ON race_results(time DESC);

-- Add compression policy (compress after 7 days)
SELECT add_compression_policy('race_results', INTERVAL '7 days', if_not_exists => TRUE);

-- Add retention policy (keep for 2 years)
SELECT add_retention_policy('race_results', INTERVAL '2 years', if_not_exists => TRUE);

-- Create continuous aggregate for daily race results summary
CREATE MATERIALIZED VIEW IF NOT EXISTS race_results_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket(INTERVAL '1 day', time) as day,
    race_id,
    COUNT(*) as total_races,
    COUNT(DISTINCT winner_trap) as winners,
    SUM(total_payouts) as total_payouts_sum,
    COUNT(DISTINCT status) as status_count,
    MAX(updated_at) as last_updated
FROM race_results
GROUP BY day, race_id;

-- Add a retention policy to the continuous aggregate
SELECT add_retention_policy('race_results_daily', INTERVAL '2 years', if_not_exists => TRUE);
