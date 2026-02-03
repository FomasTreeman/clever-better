-- Drop indexes
DROP INDEX IF EXISTS idx_bets_market_id;
DROP INDEX IF EXISTS idx_bets_bet_id;

-- Remove Betfair integration fields from bets table
ALTER TABLE bets DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE bets DROP COLUMN IF EXISTS matched_size;
ALTER TABLE bets DROP COLUMN IF EXISTS matched_price;
ALTER TABLE bets DROP COLUMN IF EXISTS market_id;
ALTER TABLE bets DROP COLUMN IF EXISTS bet_id;
