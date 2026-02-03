-- Add Betfair integration fields to bets table
ALTER TABLE bets ADD COLUMN bet_id VARCHAR(255);
ALTER TABLE bets ADD COLUMN market_id VARCHAR(255);
ALTER TABLE bets ADD COLUMN matched_price DECIMAL(10,2);
ALTER TABLE bets ADD COLUMN matched_size DECIMAL(10,2);
ALTER TABLE bets ADD COLUMN cancelled_at TIMESTAMPTZ;

-- Add index on bet_id for fast Betfair bet lookups
CREATE INDEX idx_bets_bet_id ON bets(bet_id) WHERE bet_id IS NOT NULL;

-- Add index on market_id for market-based queries
CREATE INDEX idx_bets_market_id ON bets(market_id) WHERE market_id IS NOT NULL;
