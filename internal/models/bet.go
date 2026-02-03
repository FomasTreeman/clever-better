package models

import (
	"time"

	"github.com/google/uuid"
)

// BetSide represents the side of a bet (BACK or LAY)
type BetSide string

const (
	BetSideBack BetSide = "BACK"
	BetSideLay  BetSide = "LAY"
)

// MarketType represents the type of market (WIN or PLACE)
type MarketType string

const (
	MarketTypeWin   MarketType = "WIN"
	MarketTypePlace MarketType = "PLACE"
)

// BetStatus represents the status of a bet
type BetStatus string

const (
	BetStatusPending   BetStatus = "pending"
	BetStatusMatched   BetStatus = "matched"
	BetStatusSettled   BetStatus = "settled"
	BetStatusCancelled BetStatus = "cancelled"
)

// Bet represents a betting transaction
type Bet struct {
	ID        uuid.UUID   `db:"id" json:"id" validate:"required,uuid4"`
	BetID     string      `db:"bet_id" json:"bet_id"` // Betfair bet identifier
	MarketID  string      `db:"market_id" json:"market_id"` // Betfair market identifier
	RaceID    uuid.UUID   `db:"race_id" json:"race_id" validate:"required,uuid4"`
	RunnerID  uuid.UUID   `db:"runner_id" json:"runner_id" validate:"required,uuid4"`
	StrategyID uuid.UUID  `db:"strategy_id" json:"strategy_id" validate:"required,uuid4"`
	MarketType MarketType `db:"market_type" json:"market_type" validate:"required,oneof=WIN PLACE"`
	Side      BetSide    `db:"side" json:"side" validate:"required,oneof=BACK LAY"`
	Odds      float64    `db:"odds" json:"odds" validate:"required,gt=1"`
	Stake     float64    `db:"stake" json:"stake" validate:"required,gt=0"`
	MatchedPrice *float64  `db:"matched_price" json:"matched_price"` // Actual matched price
	MatchedSize  *float64  `db:"matched_size" json:"matched_size"`   // Actual matched size
	Status    BetStatus  `db:"status" json:"status" validate:"required"`
	PlacedAt  time.Time  `db:"placed_at" json:"placed_at" validate:"required"`
	MatchedAt *time.Time `db:"matched_at" json:"matched_at"`
	SettledAt *time.Time `db:"settled_at" json:"settled_at"`
	CancelledAt *time.Time `db:"cancelled_at" json:"cancelled_at"`
	ProfitLoss *float64  `db:"profit_loss" json:"profit_loss"`
	Commission *float64  `db:"commission" json:"commission"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// CalculateProfitLoss calculates potential profit or loss on the bet
func (b *Bet) CalculateProfitLoss() float64 {
	if b.Status != BetStatusSettled {
		return 0
	}
	if b.ProfitLoss == nil {
		return 0
	}
	return *b.ProfitLoss
}

// IsSettled checks if the bet has been settled
func (b *Bet) IsSettled() bool {
	return b.Status == BetStatusSettled && b.SettledAt != nil
}

// GetROI returns the return on investment percentage
func (b *Bet) GetROI() float64 {
	if b.Stake == 0 {
		return 0
	}
	pl := b.CalculateProfitLoss()
	return (pl / b.Stake) * 100
}
