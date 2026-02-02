-- Create strategy performance tracking table

CREATE TABLE IF NOT EXISTS strategy_performance (
    time TIMESTAMPTZ NOT NULL,
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    total_bets INT DEFAULT 0,
    winning_bets INT DEFAULT 0,
    losing_bets INT DEFAULT 0,
    gross_profit DECIMAL(12, 2) DEFAULT 0,
    gross_loss DECIMAL(12, 2) DEFAULT 0,
    net_profit DECIMAL(12, 2) DEFAULT 0,
    roi DECIMAL(8, 4) DEFAULT 0,
    sharpe_ratio DECIMAL(8, 4),
    max_drawdown DECIMAL(8, 4)
);

SELECT create_hypertable('strategy_performance', 'time', if_not_exists => TRUE);
SELECT add_dimension('strategy_performance', by_hash('strategy_id', 32), if_not_exists => TRUE);

CREATE INDEX idx_strategy_performance_strategy_id ON strategy_performance(strategy_id, time DESC);

-- Add compression policy (compress after 30 days)
SELECT add_compression_policy('strategy_performance', INTERVAL '30 days', if_not_exists => TRUE);

-- Add retention policy (keep for 2 years)
SELECT add_retention_policy('strategy_performance', INTERVAL '2 years', if_not_exists => TRUE);

-- Create continuous aggregate for daily rollups
CREATE MATERIALIZED VIEW IF NOT EXISTS strategy_performance_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    strategy_id,
    SUM(total_bets)::INT as total_bets,
    SUM(winning_bets)::INT as winning_bets,
    SUM(losing_bets)::INT as losing_bets,
    SUM(gross_profit) as gross_profit,
    SUM(gross_loss) as gross_loss,
    SUM(net_profit) as net_profit,
    AVG(roi) as avg_roi,
    AVG(sharpe_ratio) as avg_sharpe_ratio,
    MIN(max_drawdown) as min_max_drawdown
FROM strategy_performance
GROUP BY day, strategy_id;

SELECT add_continuous_aggregate_policy('strategy_performance_daily',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);
