package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// RaceRepository defines the interface for race data access
type RaceRepository interface {
	Create(ctx context.Context, race *models.Race) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Race, error)
	GetUpcoming(ctx context.Context, limit int) ([]*models.Race, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]*models.Race, error)
	Update(ctx context.Context, race *models.Race) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RunnerRepository defines the interface for runner data access
type RunnerRepository interface {
	Create(ctx context.Context, runner *models.Runner) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Runner, error)
	GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Runner, error)
	Update(ctx context.Context, runner *models.Runner) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// OddsRepository defines the interface for odds data access
type OddsRepository interface {
	Insert(ctx context.Context, odds *models.OddsSnapshot) error
	InsertBatch(ctx context.Context, odds []*models.OddsSnapshot) error
	GetByRaceID(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error)
	GetLatest(ctx context.Context, raceID, runnerID uuid.UUID) (*models.OddsSnapshot, error)
	GetTimeSeriesForRunner(ctx context.Context, runnerID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error)
}

// BetRepository defines the interface for bet data access
type BetRepository interface {
	Create(ctx context.Context, bet *models.Bet) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error)
	GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Bet, error)
	GetByStrategyID(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.Bet, error)
	Update(ctx context.Context, bet *models.Bet) error
	GetPendingBets(ctx context.Context) ([]*models.Bet, error)
	GetSettledBets(ctx context.Context, start, end time.Time) ([]*models.Bet, error)
}

// StrategyRepository defines the interface for strategy data access
type StrategyRepository interface {
	Create(ctx context.Context, strategy *models.Strategy) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Strategy, error)
	GetByName(ctx context.Context, name string) (*models.Strategy, error)
	GetActive(ctx context.Context) ([]*models.Strategy, error)
	Update(ctx context.Context, strategy *models.Strategy) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ModelRepository defines the interface for ML model data access
type ModelRepository interface {
	Create(ctx context.Context, model *models.Model) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Model, error)
	GetActive(ctx context.Context) ([]*models.Model, error)
	GetByVersion(ctx context.Context, name, version string) (*models.Model, error)
	Update(ctx context.Context, model *models.Model) error
	SetActive(ctx context.Context, id uuid.UUID) error
}

// PredictionRepository defines the interface for prediction data access
type PredictionRepository interface {
	Insert(ctx context.Context, prediction *models.Prediction) error
	InsertBatch(ctx context.Context, predictions []*models.Prediction) error
	GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Prediction, error)
	GetByModelID(ctx context.Context, modelID uuid.UUID, start, end time.Time) ([]*models.Prediction, error)
}

// StrategyPerformanceRepository defines the interface for strategy performance data access
type StrategyPerformanceRepository interface {
	Insert(ctx context.Context, perf *models.StrategyPerformance) error
	GetByStrategyID(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.StrategyPerformance, error)
	GetDailyRollup(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.StrategyPerformance, error)
}
