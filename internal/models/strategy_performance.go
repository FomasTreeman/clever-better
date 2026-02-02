package models

import (
	"time"

	"github.com/google/uuid"
)

// StrategyPerformance represents performance metrics for a strategy
type StrategyPerformance struct {
	Time          time.Time  `db:"time" json:"time" validate:"required"`
	StrategyID    uuid.UUID  `db:"strategy_id" json:"strategy_id" validate:"required,uuid4"`
	TotalBets     int        `db:"total_bets" json:"total_bets" validate:"gte=0"`
	WinningBets   int        `db:"winning_bets" json:"winning_bets" validate:"gte=0"`
	LosingBets    int        `db:"losing_bets" json:"losing_bets" validate:"gte=0"`
	GrossProfit   float64    `db:"gross_profit" json:"gross_profit"`
	GrossLoss     float64    `db:"gross_loss" json:"gross_loss"`
	NetProfit     float64    `db:"net_profit" json:"net_profit"`
	ROI           float64    `db:"roi" json:"roi"`
	SharpeRatio   *float64   `db:"sharpe_ratio" json:"sharpe_ratio"`
	MaxDrawdown   *float64   `db:"max_drawdown" json:"max_drawdown"`
}

// GetWinRate calculates the win rate as a percentage
func (sp *StrategyPerformance) GetWinRate() float64 {
	if sp.TotalBets == 0 {
		return 0
	}
	return (float64(sp.WinningBets) / float64(sp.TotalBets)) * 100
}

// GetProfitFactor calculates profit factor (gross_profit / gross_loss)
func (sp *StrategyPerformance) GetProfitFactor() float64 {
	if sp.GrossLoss == 0 {
		if sp.GrossProfit > 0 {
			return 999 // Arbitrary high value for pure wins
		}
		return 0
	}
	return sp.GrossProfit / sp.GrossLoss
}

// GetExpectancy calculates average profit per bet
func (sp *StrategyPerformance) GetExpectancy() float64 {
	if sp.TotalBets == 0 {
		return 0
	}
	return sp.NetProfit / float64(sp.TotalBets)
}
