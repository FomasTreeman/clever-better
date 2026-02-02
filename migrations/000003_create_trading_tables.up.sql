-- Create trading tables for strategies and bets

CREATE TABLE IF NOT EXISTS strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    parameters JSONB,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_strategies_active ON strategies(active);
CREATE INDEX idx_strategies_created_at ON strategies(created_at DESC);

CREATE TABLE IF NOT EXISTS bets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    runner_id UUID NOT NULL REFERENCES runners(id) ON DELETE CASCADE,
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE RESTRICT,
    market_type VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    odds DECIMAL(10, 2) NOT NULL,
    stake DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    placed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    matched_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    profit_loss DECIMAL(10, 2),
    commission DECIMAL(10, 2),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

SELECT create_hypertable('bets', 'placed_at', if_not_exists => TRUE);

CREATE INDEX idx_bets_race_id ON bets(race_id, placed_at DESC);
CREATE INDEX idx_bets_runner_id ON bets(runner_id, placed_at DESC);
CREATE INDEX idx_bets_strategy_id ON bets(strategy_id, placed_at DESC);
CREATE INDEX idx_bets_status ON bets(status, placed_at DESC);

-- No retention policy for bets (keep indefinitely for audit trail)
-- Compression is optional for bets but can be added later
