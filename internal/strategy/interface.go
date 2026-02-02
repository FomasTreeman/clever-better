package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// Strategy defines the interface for backtesting strategies
type Strategy interface {
	Name() string
	Evaluate(ctx context.Context, strategyCtx Context) ([]Signal, error)
	ShouldBet(signal Signal) bool
	CalculateStake(signal Signal, bankroll float64) float64
	GetParameters() map[string]interface{}
}

// Signal represents a betting signal emitted by a strategy
type Signal struct {
	RunnerID      uuid.UUID         `json:"runner_id"`
	Side          models.BetSide    `json:"side"`
	Odds          float64           `json:"odds"`
	Stake         float64           `json:"stake"`
	Confidence    float64           `json:"confidence"`
	ExpectedValue float64           `json:"expected_value"`
	Reasoning     string            `json:"reasoning"`
	Features      map[string]any    `json:"features,omitempty"`
}

// Context provides the strategy with temporal-safe inputs
type Context struct {
	Race              *models.Race
	Runners           []*models.Runner
	OddsHistory       []*models.OddsSnapshot
	HistoricalResults []*models.RaceResult
	CurrentTime       time.Time
}

// StrategyMetadata describes a strategy for tracking and ML export
type StrategyMetadata struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}
