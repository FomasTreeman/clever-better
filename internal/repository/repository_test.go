package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

const skipIntegrationMsg = "Integration test - requires database setup"

// TestRaceRepositoryCreate tests race creation
func TestRaceRepositoryCreate(t *testing.T) {
	// db := database.SetupTestDB(t)
	// defer database.TeardownTestDB(t, db)

	// repos, err := NewRepositories(db)
	// if err != nil {
	// 	t.Fatalf("failed to create repositories: %v", err)
	// }

	// race := &models.Race{
	// 	ID:             uuid.New(),
	// 	ScheduledStart: time.Now().Add(24 * time.Hour),
	// 	Track:          "Wimbledon",
	// 	RaceType:       "standard",
	// 	Distance:       450,
	// 	Grade:          "Class 1",
	// 	Status:         "scheduled",
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// err = repos.Race.Create(ctx, race)
	// if err != nil {
	// 	t.Fatalf("failed to create race: %v", err)
	// }

	// retrieved, err := repos.Race.GetByID(ctx, race.ID)
	// if err != nil {
	// 	t.Fatalf("failed to retrieve race: %v", err)
	// }

	// if retrieved.ID != race.ID {
	// 	t.Errorf("expected race ID %v, got %v", race.ID, retrieved.ID)
	// }
	t.Skip(skipIntegrationMsg)
}

// TestBetRepositoryBatch tests batch bet operations
func TestBetRepositoryBatch(t *testing.T) {
	// db := database.SetupTestDB(t)
	// defer database.TeardownTestDB(t, db)

	// repos, err := NewRepositories(db)
	// if err != nil {
	// 	t.Fatalf("failed to create repositories: %v", err)
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// // Create test data
	// strategyID := uuid.New()
	// raceID := uuid.New()
	// runnerID := uuid.New()

	// bets := make([]*models.Bet, 100)
	// for i := 0; i < 100; i++ {
	// 	bets[i] = &models.Bet{
	// 		ID:         uuid.New(),
	// 		RaceID:     raceID,
	// 		RunnerID:   runnerID,
	// 		StrategyID: strategyID,
	// 		MarketType: "WIN",
	// 		Side:       "BACK",
	// 		Odds:       3.5,
	// 		Stake:      10.0,
	// 		Status:     "pending",
	// 		PlacedAt:   time.Now(),
	// 	}
	// }

	// // Batch insert would go here if implemented
	// // err := repos.Bet.InsertBatch(ctx, bets)

	// t.Logf("would insert %d bets", len(bets))
	t.Skip(skipIntegrationMsg)
}

// TestOddsRepositoryTimeSeries tests time-series odds queries
func TestOddsRepositoryTimeSeries(t *testing.T) {
	// db := database.SetupTestDB(t)
	// defer database.TeardownTestDB(t, db)

	// repos, err := NewRepositories(db)
	// if err != nil {
	// 	t.Fatalf("failed to create repositories: %v", err)
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// raceID := uuid.New()
	// runnerID := uuid.New()
	// now := time.Now()

	// // Create test odds data
	// snapshots := make([]*models.OddsSnapshot, 50)
	// for i := 0; i < 50; i++ {
	// 	snapshots[i] = &models.OddsSnapshot{
	// 		Time:        now.Add(time.Duration(i) * time.Minute),
	// 		RaceID:      raceID,
	// 		RunnerID:    runnerID,
	// 		BackPrice:   3.5 + float64(i)*0.01,
	// 		BackSize:    100.0,
	// 		LayPrice:    3.6 + float64(i)*0.01,
	// 		LaySize:     50.0,
	// 		TotalVolume: 5000.0,
	// 	}
	// }

	// // Batch insert
	// err = repos.Odds.InsertBatch(ctx, snapshots)
	// if err != nil {
	// 	t.Fatalf("failed to batch insert odds: %v", err)
	// }

	// // Query time series
	// retrieved, err := repos.Odds.GetTimeSeriesForRunner(ctx, runnerID, now, now.Add(1*time.Hour))
	// if err != nil {
	// 	t.Fatalf("failed to retrieve odds time series: %v", err)
	// }

	// if len(retrieved) != 50 {
	// 	t.Errorf("expected 50 snapshots, got %d", len(retrieved))
	// }
	t.Skip(skipIntegrationMsg)
}

// TestStrategyPerformanceRepository tests strategy performance aggregation
func TestStrategyPerformanceRepository(t *testing.T) {
	// db := database.SetupTestDB(t)
	// defer database.TeardownTestDB(t, db)

	// repos, err := NewRepositories(db)
	// if err != nil {
	// 	t.Fatalf("failed to create repositories: %v", err)
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// strategyID := uuid.New()
	// perf := &models.StrategyPerformance{
	// 	Time:           time.Now(),
	// 	StrategyID:     strategyID,
	// 	TotalBets:      100,
	// 	MatchedBets:    95,
	// 	WonBets:        60,
	// 	LostBets:       35,
	// 	ProfitLoss:     450.25,
	// 	ROI:            0.045,
	// 	WinRate:        0.60,
	// 	ProfitFactor:   2.15,
	// 	Expectancy:     4.50,
	// 	MarketType:     "WIN",
	// }

	// err = repos.StrategyPerformance.Insert(ctx, perf)
	// if err != nil {
	// 	t.Fatalf("failed to insert performance: %v", err)
	// }

	// // Query daily rollup
	// startDate := time.Now().AddDate(0, 0, -1)
	// endDate := time.Now().AddDate(0, 0, 1)
	// dailyPerformance, err := repos.StrategyPerformance.GetDailyRollup(ctx, strategyID, startDate, endDate)
	// if err != nil {
	// 	t.Fatalf("failed to get daily rollup: %v", err)
	// }

	// t.Logf("retrieved %d daily performance records", len(dailyPerformance))
	t.Skip(skipIntegrationMsg)
}
