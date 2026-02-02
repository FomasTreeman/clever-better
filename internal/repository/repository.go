package repository

import (
	"fmt"

	"github.com/yourusername/clever-better/internal/database"
)

// Repositories holds all repository implementations
type Repositories struct {
	Race                RaceRepository
	Runner              RunnerRepository
	Odds                OddsRepository
	Bet                 BetRepository
	Strategy            StrategyRepository
	Model               ModelRepository
	Prediction          PredictionRepository
	StrategyPerformance StrategyPerformanceRepository
	RaceResult          RaceResultRepository
}

// NewRepositories creates and returns all repository implementations
func NewRepositories(db *database.DB) (*Repositories, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	return &Repositories{
		Race:                NewPostgresRaceRepository(db),
		Runner:              NewPostgresRunnerRepository(db),
		Odds:                NewPostgresOddsRepository(db),
		Bet:                 NewPostgresBetRepository(db),
		Strategy:            NewPostgresStrategyRepository(db),
		Model:               NewPostgresModelRepository(db),
		Prediction:          NewPostgresPredictionRepository(db),
		StrategyPerformance: NewPostgresStrategyPerformanceRepository(db),
		RaceResult:          NewPostgresRaceResultRepository(db),
	}, nil
}
