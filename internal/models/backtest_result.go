package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BacktestResult represents a persisted backtest run
type BacktestResult struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	StrategyID     uuid.UUID       `db:"strategy_id" json:"strategy_id"`
	RunDate        time.Time       `db:"run_date" json:"run_date"`
	StartDate      time.Time       `db:"start_date" json:"start_date"`
	EndDate        time.Time       `db:"end_date" json:"end_date"`
	InitialCapital float64         `db:"initial_capital" json:"initial_capital"`
	FinalCapital   float64         `db:"final_capital" json:"final_capital"`
	TotalReturn    float64         `db:"total_return" json:"total_return"`
	SharpeRatio    float64         `db:"sharpe_ratio" json:"sharpe_ratio"`
	MaxDrawdown    float64         `db:"max_drawdown" json:"max_drawdown"`
	TotalBets      int             `db:"total_bets" json:"total_bets"`
	WinRate        float64         `db:"win_rate" json:"win_rate"`
	ProfitFactor   float64         `db:"profit_factor" json:"profit_factor"`
	Method         string          `db:"method" json:"method"`
	CompositeScore float64         `db:"composite_score" json:"composite_score"`
	Recommendation string          `db:"recommendation" json:"recommendation"`
	MLFeatures     json.RawMessage `db:"ml_features" json:"ml_features"`
	FullResults    json.RawMessage `db:"full_results" json:"full_results"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}
