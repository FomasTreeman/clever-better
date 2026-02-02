# Integration Guide - Database Layer

This guide explains how to integrate the database layer into your application code.

## Table of Contents

1. [Basic Setup](#basic-setup)
2. [Query Examples](#query-examples)
3. [Transaction Management](#transaction-management)
4. [Error Handling](#error-handling)
5. [Testing](#testing)
6. [Performance Optimization](#performance-optimization)

## Basic Setup

### Application Initialization

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	db, err := database.NewDB(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	// Verify connection
	if err := db.Ping(ctx); err != nil {
		log.Fatal("database connection failed:", err)
	}

	// Initialize database (check TimescaleDB and migrations)
	if err := database.Initialize(ctx, db); err != nil {
		log.Fatal("database initialization failed:", err)
	}

	// Create repositories
	repos, err := repository.NewRepositories(db)
	if err != nil {
		log.Fatal(err)
	}

	// Use repositories
	runApplication(ctx, repos)
}

func runApplication(ctx context.Context, repos *repository.Repositories) {
	// Your application code here
}
```

## Query Examples

### Getting Data

```go
// Get upcoming races
races, err := repos.Race.GetUpcoming(ctx, 10)
if err != nil {
	log.Printf("failed to get races: %v", err)
	return
}

for _, race := range races {
	log.Printf("Race: %s at %s (%d yards)", race.Track, race.ScheduledStart, race.Distance)
}

// Get race participants
runners, err := repos.Runner.GetByRaceID(ctx, race.ID)
if err != nil {
	log.Printf("failed to get runners: %v", err)
	return
}

// Get latest odds for a runner
odds, err := repos.Odds.GetLatest(ctx, race.ID, runners[0].ID)
if err == models.ErrNotFound {
	log.Println("no odds available")
} else if err != nil {
	log.Printf("error getting odds: %v", err)
}
```

### Creating Records

```go
// Create a new strategy
strategy := &models.Strategy{
	ID:          uuid.New(),
	Name:        "MomentumStrategy",
	Description: "Bet on runners with rising odds",
	Parameters:  json.RawMessage(`{"threshold": 0.05, "min_odds": 2.0}`),
	Active:      true,
	Version:     "1.0.0",
}

if err := repos.Strategy.Create(ctx, strategy); err != nil {
	log.Printf("failed to create strategy: %v", err)
	return
}
```

### Batch Operations

```go
// Batch insert odds (high performance)
snapshots := make([]*models.OddsSnapshot, 1000)
for i := 0; i < 1000; i++ {
	snapshots[i] = &models.OddsSnapshot{
		Time:        time.Now().Add(time.Duration(i) * time.Second),
		RaceID:      raceID,
		RunnerID:    runnerID,
		BackPrice:   3.5,
		BackSize:    100,
		LayPrice:    3.6,
		LaySize:     50,
		TotalVolume: 5000,
	}
}

if err := repos.Odds.InsertBatch(ctx, snapshots); err != nil {
	log.Printf("failed to batch insert odds: %v", err)
	return
}
```

### Range Queries

```go
// Get bets for strategy in date range
startDate := time.Now().AddDate(0, 0, -7)  // Last 7 days
endDate := time.Now()

bets, err := repos.Bet.GetByStrategyID(ctx, strategyID, startDate, endDate)
if err != nil {
	log.Printf("failed to get strategy bets: %v", err)
	return
}

log.Printf("Found %d bets for strategy in past week", len(bets))

// Get strategy performance rollup
dailyPerf, err := repos.StrategyPerformance.GetDailyRollup(ctx, strategyID, startDate, endDate)
if err != nil {
	log.Printf("failed to get performance: %v", err)
	return
}

for _, day := range dailyPerf {
	log.Printf("%v: %d bets, %.2f ROI, %.1f%% win rate",
		day.Time, day.TotalBets, day.ROI, day.WinRate*100)
}
```

## Transaction Management

### Basic Transactions

```go
// Wrap multiple operations in a transaction
err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
	// All operations use the transaction context
	
	// Create bet
	if _, err := tx.Exec(ctx,
		"INSERT INTO bets (id, race_id, runner_id, strategy_id, odds, stake, status, placed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		bet.ID, bet.RaceID, bet.RunnerID, bet.StrategyID, bet.Odds, bet.Stake, "pending", time.Now(),
	); err != nil {
		return err
	}

	// Update strategy balance
	if _, err := tx.Exec(ctx,
		"UPDATE strategies SET balance = balance - $1 WHERE id = $2",
		bet.Stake, bet.StrategyID,
	); err != nil {
		return err
	}

	// If any operation fails, transaction automatically rolls back
	return nil
})

if err != nil {
	log.Printf("transaction failed: %v", err)
}
```

### Complex Transactions

```go
// Settle a bet with complex logic
err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
	// 1. Get current bet
	var bet models.Bet
	row := tx.QueryRow(ctx, 
		"SELECT id, stake, odds, status FROM bets WHERE id = $1", betID)
	if err := row.Scan(&bet.ID, &bet.Stake, &bet.Odds, &bet.Status); err != nil {
		return err
	}

	// 2. Calculate winnings
	winnings := bet.Stake * bet.Odds

	// 3. Update bet status
	if _, err := tx.Exec(ctx,
		"UPDATE bets SET status = 'settled', profit_loss = $1, settled_at = NOW() WHERE id = $2",
		winnings - bet.Stake, betID,
	); err != nil {
		return err
	}

	// 4. Update strategy balance
	if _, err := tx.Exec(ctx,
		"UPDATE strategies SET balance = balance + $1 WHERE id = $2",
		winnings - bet.Stake, bet.StrategyID,
	); err != nil {
		return err
	}

	// 5. Record transaction
	if _, err := tx.Exec(ctx,
		"INSERT INTO transactions (id, strategy_id, type, amount, created_at) VALUES ($1, $2, $3, $4, NOW())",
		uuid.New(), bet.StrategyID, "settlement", winnings - bet.Stake,
	); err != nil {
		return err
	}

	return nil
})

if err != nil {
	log.Printf("failed to settle bet: %v", err)
}
```

## Error Handling

### Model Errors

```go
// Models define custom errors
import "github.com/yourusername/clever-better/internal/models"

race, err := repos.Race.GetByID(ctx, raceID)
if err == models.ErrNotFound {
	log.Printf("race %s not found", raceID)
	return
} else if err != nil {
	log.Printf("database error: %v", err)
	return
}

// Check for duplicate key errors
strategy := &models.Strategy{
	Name: "DuplicateName",
}
err := repos.Strategy.Create(ctx, strategy)
if err == models.ErrDuplicateKey {
	log.Println("strategy with this name already exists")
} else if err != nil {
	log.Printf("error creating strategy: %v", err)
}
```

### Validation Errors

```go
// Models include validation
strategy := &models.Strategy{
	Name: "",  // Empty name
}

// Validation happens during operations
if err := strategy.Validate(); err != nil {
	log.Printf("strategy validation failed: %v", err)
}
```

## Testing

### Setup & Teardown

```go
func TestStrategyRepository(t *testing.T) {
	// Setup test database
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	// Create repositories
	repos, err := repository.NewRepositories(db)
	if err != nil {
		t.Fatalf("failed to create repositories: %v", err)
	}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Your test code here
	strategy := &models.Strategy{
		ID:    uuid.New(),
		Name:  "TestStrategy",
		Active: true,
	}

	if err := repos.Strategy.Create(ctx, strategy); err != nil {
		t.Fatalf("failed to create strategy: %v", err)
	}

	retrieved, err := repos.Strategy.GetByID(ctx, strategy.ID)
	if err != nil {
		t.Fatalf("failed to retrieve strategy: %v", err)
	}

	if retrieved.Name != "TestStrategy" {
		t.Errorf("expected name 'TestStrategy', got '%s'", retrieved.Name)
	}
}
```

## Performance Optimization

### Connection Pooling

Connection pooling is automatically configured based on config.yaml:

```yaml
database:
  max_connections: 25      # Max connections to database
  max_idle_connections: 5  # Idle connections kept open
  connection_max_lifetime: 5m  # Recycle connections
```

### Query Optimization

```go
// 1. Use indexed columns in WHERE clauses
// ✓ Good - queries on indexed columns
races, err := repos.Race.GetByDateRange(ctx, startDate, endDate)

// 2. Use batch operations for bulk inserts
// ✓ Good - CopyFrom is faster than individual inserts
err := repos.Odds.InsertBatch(ctx, snapshots)

// 3. Use time ranges for time-series data
// ✓ Good - limits data scanned by hypertable
odds, err := repos.Odds.GetByRaceID(ctx, raceID, startTime, endTime)

// 4. Avoid N+1 queries
// ✓ Good - single query for all runners
runners, err := repos.Runner.GetByRaceID(ctx, raceID)

// ✗ Bad - N+1: one query per runner
for _, runner := range runners {
	details, err := repos.Runner.GetByID(ctx, runner.ID)  // Repeated queries
}
```

### Monitoring

```go
// Check database health
if err := db.HealthCheck(ctx); err != nil {
	log.Printf("database health check failed: %v", err)
}

// Get connection pool stats
pool := db.GetPool()
stat := pool.Stat()
log.Printf("Open connections: %d, In-use: %d, Idle: %d, Max: %d",
	stat.OpenConnections(), stat.CurrentlyOpenConnections(), 
	stat.IdleConnections(), stat.MaxConnections())
```

## Complete Example

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

func main() {
	ctx := context.Background()

	// Setup
	cfg, _ := config.Load("config/config.yaml")
	db, _ := database.NewDB(cfg.Database)
	defer db.Close(ctx)
	repos, _ := repository.NewRepositories(db)

	// Get upcoming races
	races, _ := repos.Race.GetUpcoming(ctx, 5)

	for _, race := range races {
		log.Printf("Processing race: %s", race.Track)

		// Get runners
		runners, _ := repos.Runner.GetByRaceID(ctx, race.ID)
		log.Printf("  Found %d runners", len(runners))

		// Get active strategies
		strategies, _ := repos.Strategy.GetActive(ctx)

		// Place bets for each strategy
		for _, strategy := range strategies {
			for _, runner := range runners {
				bet := &models.Bet{
					ID:         uuid.New(),
					RaceID:     race.ID,
					RunnerID:   runner.ID,
					StrategyID: strategy.ID,
					Stake:      10.0,
					Status:     "pending",
					PlacedAt:   time.Now(),
				}

				if err := repos.Bet.Create(ctx, bet); err != nil {
					log.Printf("failed to create bet: %v", err)
					continue
				}

				log.Printf("  Placed bet: %s for %s", strategy.Name, runner.Name)
			}
		}
	}

	log.Println("Complete!")
}
```
