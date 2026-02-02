-- Create TimescaleDB hypertables for time-series data
-- Requires TimescaleDB extension to be installed

CREATE TABLE IF NOT EXISTS odds_snapshots (
    time TIMESTAMPTZ NOT NULL,
    race_id UUID NOT NULL,
    runner_id UUID NOT NULL,
    back_price DECIMAL(10, 2),
    back_size DECIMAL(12, 2),
    lay_price DECIMAL(10, 2),
    lay_size DECIMAL(12, 2),
    ltp DECIMAL(10, 2),
    total_volume DECIMAL(14, 2)
);

SELECT create_hypertable('odds_snapshots', 'time', if_not_exists => TRUE);
SELECT add_dimension('odds_snapshots', by_hash('race_id', 32), if_not_exists => TRUE);

CREATE INDEX idx_odds_snapshots_race_id ON odds_snapshots(race_id, time DESC);
CREATE INDEX idx_odds_snapshots_runner_id ON odds_snapshots(runner_id, time DESC);

-- Add compression policy (compress after 7 days)
SELECT add_compression_policy('odds_snapshots', INTERVAL '7 days', if_not_exists => TRUE);

-- Add retention policy (keep for 2 years)
SELECT add_retention_policy('odds_snapshots', INTERVAL '2 years', if_not_exists => TRUE);
