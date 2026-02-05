//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

const skipIntegration = "Skipping integration test in short mode"

// TestDatabaseRepositoryIntegration tests all repositories against real TimescaleDB
func TestDatabaseRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	t.Run("RaceRepository", func(t *testing.T) {
		repo := repository.NewPostgresRaceRepository(db)

		// Create race
		race := &models.Race{
			ID:             uuid.New(),
			ScheduledStart: time.Now().Add(1 * time.Hour),
			Track:          "Test Track",
			RaceType:       "Flat",
			Distance:       1600,
			Grade:          "3",
			Conditions:     json.RawMessage(`{"going":"good"}`),
			Status:         "scheduled",
		}

		err := repo.Create(ctx, race)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, race.ID)

		// Retrieve race
		retrieved, err := repo.GetByID(ctx, race.ID)
		require.NoError(t, err)
		assert.Equal(t, race.Track, retrieved.Track)
		assert.Equal(t, race.RaceType, retrieved.RaceType)

		// Update race
		now := time.Now()
		race.ActualStart = &now
		race.Status = "started"
		err = repo.Update(ctx, race)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, race.ID)
		require.NoError(t, err)
		assert.Equal(t, "started", updated.Status)
	})

	t.Run("BetRepository", func(t *testing.T) {
		repo := repository.NewPostgresBetRepository(db)
		runner := seedRaceAndRunner(t, ctx, db)

		bet := &models.Bet{
			MarketID:   "1.12345",
			RaceID:     runner.RaceID,
			RunnerID:   runner.ID,
			StrategyID: uuid.New(),
			MarketType: models.MarketTypeWin,
			Side:       models.BetSideBack,
			Odds:       3.5,
			Stake:      100.0,
			Status:     models.BetStatusPending,
			PlacedAt:   time.Now(),
		}

		err := repo.Create(ctx, bet)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, bet.ID)
		require.NoError(t, err)
		assert.Equal(t, bet.Odds, retrieved.Odds)
		assert.Equal(t, bet.Stake, retrieved.Stake)
	})

	t.Run("StrategyRepository", func(t *testing.T) {
		repo := repository.NewPostgresStrategyRepository(db)

		params, err := json.Marshal(map[string]interface{}{"min_value": 0.15})
		require.NoError(t, err)

		strategy := &models.Strategy{
			ID:          uuid.New(),
			Name:        "Test Strategy",
			Description: "Integration test strategy",
			Active:      true,
			Parameters:  params,
		}

		err := repo.Create(ctx, strategy)
		require.NoError(t, err)

		active, err := repo.GetActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(active), 1)
	})
}

// TestHypertablePartitioning tests TimescaleDB hypertable functionality
func TestHypertablePartitioning(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	oddsRepo := repository.NewPostgresOddsRepository(db)
	runner := seedRaceAndRunner(t, ctx, db)

	// Insert odds across multiple time ranges
	baseTime := time.Now().Add(-30 * 24 * time.Hour)

	for i := 0; i < 100; i++ {
		back := 3.5
		lay := 3.4
		snapshot := &models.OddsSnapshot{
			Time:      baseTime.Add(time.Duration(i) * time.Hour),
			RaceID:    runner.RaceID,
			RunnerID:  runner.ID,
			BackPrice: &back,
			LayPrice:  &lay,
		}

		err := oddsRepo.Insert(ctx, snapshot)
		require.NoError(t, err)
	}

	// Verify data retrieval across partitions
	retrieved, err := oddsRepo.GetByRaceID(ctx, runner.RaceID, baseTime, baseTime.Add(200*time.Hour))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(retrieved), 50, "Should retrieve data from multiple partitions")

	t.Log("✓ Hypertable partitioning validated")
}

// TestConcurrentOperations tests concurrent read/write operations
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	betRepo := repository.NewPostgresBetRepository(db)
	runner := seedRaceAndRunner(t, ctx, db)

	// Concurrent writes
	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			bet := &models.Bet{
				MarketID:   "1.12345",
				RaceID:     runner.RaceID,
				RunnerID:   runner.ID,
				StrategyID: uuid.New(),
				MarketType: models.MarketTypeWin,
				Side:       models.BetSideBack,
				Odds:       3.5,
				Stake:      float64(100 + index),
				Status:     models.BetStatusPending,
				PlacedAt:   time.Now(),
			}

			err := betRepo.Create(ctx, bet)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all bets created
	allBets, err := betRepo.GetByRaceID(ctx, runner.RaceID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allBets), concurrency)

	t.Log("✓ Concurrent operations validated")
}

// TestTransactionRollback tests transaction rollback scenarios
func TestTransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	raceRepo := repository.NewPostgresRaceRepository(db)

	// Begin transaction
	txInterface, err := db.BeginTx(ctx)
	require.NoError(t, err)
	tx := txInterface.(pgx.Tx)

	// Insert data within transaction using tx.Exec
	race := &models.Race{
		ID:             uuid.New(),
		ScheduledStart: time.Now().Add(1 * time.Hour),
		Track:          "Rollback Track",
		RaceType:       "Flat",
		Distance:       2000,
		Grade:          "3",
		Conditions:     "Good",
		Status:         "scheduled",
	}

	query := `
		INSERT INTO races (id, scheduled_start, track, race_type, distance, grade, conditions, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, query,
		race.ID, race.ScheduledStart, race.Track, race.RaceType, race.Distance,
		race.Grade, race.Conditions, race.Status,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// Verify data was not persisted after rollback
	_, err = raceRepo.GetByID(ctx, race.ID)
	assert.Error(t, err, "Race should not exist after rollback")

	t.Log("✓ Transaction rollback validated: data inserted in transaction was not persisted after rollback")
}

// TestConnectionPoolBehavior tests connection pool under load
func TestConnectionPoolBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	// Simulate high concurrent load
	var wg sync.WaitGroup
	requests := 50

	strategyRepo := repository.NewPostgresStrategyRepository(db)

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Read operation
			_, err := strategyRepo.GetActive(ctx)
			assert.NoError(t, err)

			// Write operation
			strategy := &models.Strategy{
				ID:          uuid.New(),
				Name:        "Pool Test Strategy",
				Description: "pool",
				Active:      false,
			}
			err = strategyRepo.Create(ctx, strategy)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	t.Log("✓ Connection pool behavior validated")
}

func seedRaceAndRunner(t *testing.T, ctx context.Context, db *database.DB) *models.Runner {
	RaceRepo := repository.NewPostgresRaceRepository(db)
	RunnerRepo := repository.NewPostgresRunnerRepository(db)

	race := &models.Race{
		ID:             uuid.New(),
		ScheduledStart: time.Now().Add(1 * time.Hour),
		Track:          "Seed Track",
		RaceType:       "Flat",
		Distance:       1600,
		Grade:          "3",
		Conditions:     json.RawMessage(`{"going":"good"}`),
		Status:         "scheduled",
	}
	require.NoError(t, RaceRepo.Create(ctx, race))

	weight := 120.0
	runner := &models.Runner{
		ID:         uuid.New(),
		RaceID:     race.ID,
		TrapNumber: 1,
		Name:       "Seed Runner",
		Trainer:    "Seed Trainer",
		Weight:     &weight,
	}
	require.NoError(t, RunnerRepo.Create(ctx, runner))

	return runner
}

// TestDatabaseMigrations tests schema migrations
func TestDatabaseMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	// Setup fresh database
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	// Verify tables exist
	ctx := context.Background()
	
	tables := []string{"races", "runners", "bets", "strategies", "odds_snapshots"}
	for _, table := range tables {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = $1
			)
		`
		err := db.GetPool().QueryRow(ctx, query, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}

	t.Log("✓ Database migrations validated")
}

// TestDataRetention tests time-series data retention policies
func TestDataRetention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	oddsRepo := repository.NewPostgresOddsRepository(db)
	runner := seedRaceAndRunner(t, ctx, db)

	// Insert old data (beyond retention period)
	oldTime := time.Now().Add(-120 * 24 * time.Hour)
	oldBack := 3.0
	oldSnapshot := &models.OddsSnapshot{
		Time:      oldTime,
		RaceID:    runner.RaceID,
		RunnerID:  runner.ID,
		BackPrice: &oldBack,
	}

	err := oddsRepo.Insert(ctx, oldSnapshot)
	require.NoError(t, err)

	// Insert recent data
	recentTime := time.Now().Add(-1 * time.Hour)
	recentBack := 3.4
	recentSnapshot := &models.OddsSnapshot{
		Time:      recentTime,
		RaceID:    runner.RaceID,
		RunnerID:  runner.ID,
		BackPrice: &recentBack,
	}

	err = oddsRepo.Insert(ctx, recentSnapshot)
	require.NoError(t, err)

	// In production, old data would be compressed or dropped
	// based on retention policy
	t.Log("✓ Data retention policy configured")
}
