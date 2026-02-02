# Database Architecture and Schema

## Overview

The Clever Better trading bot uses PostgreSQL 15 with the TimescaleDB extension for efficient time-series data storage and analysis. The database layer is built on a repository pattern with context-aware methods for proper timeout and cancellation handling.

## Technology Stack

- **Database Engine**: PostgreSQL 15
- **Time-Series Extension**: TimescaleDB 2.11+
- **Go Driver**: jackc/pgx v5
- **Connection Pooling**: pgxpool (built-in with pgx)
- **Migration Tool**: golang-migrate/migrate
- **ORM Pattern**: Custom repository pattern with interfaces

## Database Schema

### Core Tables

#### `races`
Stores racing event information.

```sql
id UUID (PRIMARY KEY)
scheduled_start TIMESTAMPTZ (NOT NULL)
actual_start TIMESTAMPTZ
track VARCHAR(255) (NOT NULL)
race_type VARCHAR(50) (NOT NULL)  -- e.g., 'standard', 'hurdle', 'chase'
distance INTEGER (NOT NULL)        -- in yards
grade VARCHAR(50)                  -- e.g., 'Class 1', 'Handicap'
conditions VARCHAR(255)             -- track conditions
status VARCHAR(50) (NOT NULL)      -- 'scheduled', 'in_progress', 'finished', 'abandoned'
created_at TIMESTAMPTZ (DEFAULT NOW())
updated_at TIMESTAMPTZ (DEFAULT NOW())
```

**Indexes**:
- `idx_races_scheduled_start`: Speeds up upcoming races queries
- `idx_races_status`: Quick filtering by race status

#### `runners`
Stores greyhound/horse information for each race.

```sql
id UUID (PRIMARY KEY)
race_id UUID (FOREIGN KEY → races.id) ON DELETE CASCADE
trap_number INTEGER (NOT NULL)      -- 1-8 for greyhounds
name VARCHAR(255) (NOT NULL)
form_rating DECIMAL(4,2)            -- 0-10 scale
weight DECIMAL(7,2)                 -- in kg
trainer VARCHAR(255)
days_since_last_race INTEGER
metadata JSONB                      -- breed, age, historical stats, etc.
created_at TIMESTAMPTZ (DEFAULT NOW())
updated_at TIMESTAMPTZ (DEFAULT NOW())
```

**Indexes**:
- `idx_runners_race_id`: Quick lookup of runners for a race
- `idx_runners_trap_number`: Query by trap position

#### `strategies`
Stores trading strategy configurations and versions.

```sql
id UUID (PRIMARY KEY)
name VARCHAR(255) (NOT NULL, UNIQUE per version)
description TEXT
parameters JSONB (NOT NULL)         -- strategy parameters (threshold, staking, etc.)
active BOOLEAN (DEFAULT false)
version VARCHAR(50) (NOT NULL)
created_at TIMESTAMPTZ (DEFAULT NOW())
updated_at TIMESTAMPTZ (DEFAULT NOW())
```

**Indexes**:
- `idx_strategies_name`: Look up strategy by name
- `idx_strategies_active`: Query active strategies

#### `models`
Stores ML model metadata and metrics.

```sql
id UUID (PRIMARY KEY)
name VARCHAR(255) (NOT NULL)
version VARCHAR(50) (NOT NULL)
model_type VARCHAR(100) (NOT NULL)  -- 'xgboost', 'neural_network', 'ensemble'
hyperparameters JSONB               -- learning_rate, layers, etc.
metrics JSONB                       -- accuracy, precision, recall, etc.
active BOOLEAN (DEFAULT false)
created_at TIMESTAMPTZ (DEFAULT NOW())
updated_at TIMESTAMPTZ (DEFAULT NOW())
```

**Indexes**:
- `idx_models_name_version`: Query specific model version
- `idx_models_active`: Get active models

### Time-Series Tables (Hypertables)

#### `odds_snapshots` (Hypertable)
High-frequency odds data partitioned by time.

```sql
time TIMESTAMPTZ (PARTITION KEY)
race_id UUID
runner_id UUID
back_price DECIMAL(8,2)             -- back odds
back_size DECIMAL(10,2)             -- size available
lay_price DECIMAL(8,2)              -- lay odds
lay_size DECIMAL(10,2)              -- size available
ltp DECIMAL(8,2)                    -- last traded price
total_volume DECIMAL(15,2)          -- market volume
PRIMARY KEY (time, race_id, runner_id) USING BRIN
FOREIGN KEY (race_id) → races.id
FOREIGN KEY (runner_id) → runners.id
```

**TimescaleDB Features**:
- Hypertable with 1-day time partitions
- BRIN compression after 7 days
- 2-year retention policy
- Indexes on (race_id, time) and (runner_id, time) for range queries

#### `bets` (Hypertable)
Trading activity partitioned by placement time.

```sql
id UUID (PRIMARY KEY)
time TIMESTAMPTZ (PARTITION KEY)    -- placed_at becomes partition key
race_id UUID
runner_id UUID
strategy_id UUID
market_type VARCHAR(50)              -- 'win', 'place'
side VARCHAR(10)                     -- 'back', 'lay'
odds DECIMAL(8,2)
stake DECIMAL(10,2)
status VARCHAR(50)                   -- 'pending', 'matched', 'settled', 'cancelled'
placed_at TIMESTAMPTZ
matched_at TIMESTAMPTZ
settled_at TIMESTAMPTZ
profit_loss DECIMAL(12,2)
commission DECIMAL(8,2)
created_at TIMESTAMPTZ
updated_at TIMESTAMPTZ
FOREIGN KEY (race_id) → races.id
FOREIGN KEY (runner_id) → runners.id
FOREIGN KEY (strategy_id) → strategies.id
```

**TimescaleDB Features**:
- Hypertable with 1-day time partitions
- No compression or retention (audit trail)
- Indexes on (strategy_id, placed_at) and (race_id, placed_at)

#### `predictions` (Hypertable)
ML model predictions partitioned by prediction time.

```sql
id UUID (PRIMARY KEY)
time TIMESTAMPTZ (PARTITION KEY)    -- predicted_at becomes partition key
race_id UUID
runner_id UUID
model_id UUID
predicted_outcome VARCHAR(50)        -- 'win', 'place', 'lose'
confidence DECIMAL(5,4)              -- 0-1 probability
features JSONB                       -- feature values used
predicted_at TIMESTAMPTZ
created_at TIMESTAMPTZ
FOREIGN KEY (race_id) → races.id
FOREIGN KEY (runner_id) → runners.id
FOREIGN KEY (model_id) → models.id
```

**TimescaleDB Features**:
- Hypertable with 1-day time partitions
- BRIN compression after 7 days
- 1-year retention policy
- Indexes on (model_id, predicted_at) and (race_id, predicted_at)

#### `strategy_performance` (Hypertable)
Aggregated strategy performance metrics partitioned by time.

```sql
time TIMESTAMPTZ (PARTITION KEY)
strategy_id UUID
total_bets INTEGER
matched_bets INTEGER
won_bets INTEGER
lost_bets INTEGER
profit_loss DECIMAL(12,2)
roi DECIMAL(8,4)
win_rate DECIMAL(5,4)
profit_factor DECIMAL(8,4)
expectancy DECIMAL(8,4)
market_type VARCHAR(50)
PRIMARY KEY (time, strategy_id) USING BRIN
FOREIGN KEY (strategy_id) → strategies.id
```

**TimescaleDB Features**:
- Hypertable with 1-hour time partitions
- Continuous aggregate for daily rollup: `strategy_performance_daily`
- BRIN compression after 7 days
- 2-year retention policy

### Continuous Aggregates

#### `strategy_performance_daily`
Daily rollup of hourly strategy performance data.

```sql
-- Automatically maintained aggregate
TIME_BUCKET('1 day', time) AS time
strategy_id
SUM(total_bets) AS total_bets
SUM(matched_bets) AS matched_bets
SUM(won_bets) AS won_bets
SUM(lost_bets) AS lost_bets
SUM(profit_loss) AS profit_loss
AVG(roi) AS roi
AVG(win_rate) AS win_rate
AVG(profit_factor) AS profit_factor
AVG(expectancy) AS expectancy
market_type
```

## Go Models

All models map to database tables with proper struct tags and validation.

### Model Tags

- `db:"column_name"` - Database column mapping
- `json:"field_name"` - JSON serialization
- `validate:"required,min=1"` - Validation rules

### Error Handling

Custom errors in `internal/models/errors.go`:
- `ErrNotFound` - Record doesn't exist
- `ErrDuplicateKey` - Constraint violation
- `ErrInvalidID` - Invalid UUID format
- `ErrStrategyNameRequired` - Strategy name missing

## Repository Pattern

### Interfaces

All data access is abstracted through repository interfaces in `internal/repository/interfaces.go`.

```go
RaceRepository
RunnerRepository
OddsRepository
BetRepository
StrategyRepository
ModelRepository
PredictionRepository
StrategyPerformanceRepository
```

### Factory

Create all repositories with a single call:

```go
repos, err := repository.NewRepositories(db)
if err != nil {
    log.Fatal(err)
}

// Use repositories
races, err := repos.Race.GetUpcoming(ctx, 10)
```

## Connection Management

### Pool Configuration

Connection pool settings in `config.yaml`:

```yaml
database:
  host: localhost
  port: 5432
  name: clever_better
  user: postgres
  password: secret
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  statement_cache_mode: describe
```

### Lifecycle

```go
// Create connection
db, err := database.NewDB(cfg.Database)
defer db.Close(ctx)

// Health check
if err := db.HealthCheck(ctx); err != nil {
    log.Fatal("database unhealthy")
}

// Transactions
err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
    // Execute transactional operations
    return nil
})
```

## Migrations

Using `golang-migrate/migrate` for schema versioning.

### Running Migrations

```bash
# Up
migrate -path migrations -database "postgres://user:password@localhost/clever_better" up

# Down
migrate -path migrations -database "postgres://user:password@localhost/clever_better" down

# Specific version
migrate -path migrations -database "postgres://..." goto 3
```

### Migration Files

- `migrations/000001_create_core_tables.up.sql` - Races and runners
- `migrations/000002_create_hypertables.up.sql` - Odds snapshots
- `migrations/000003_create_trading_tables.up.sql` - Strategies and bets
- `migrations/000004_create_ml_tables.up.sql` - Models and predictions
- `migrations/000005_create_strategy_performance.up.sql` - Strategy performance

## Performance Considerations

### Indexes

- Composite indexes on (foreign_key, time) for range queries
- BRIN indexes for time-series data (compressed)
- Separate indexes on common filter columns

### TimescaleDB Optimization

- Hypertables automatically partition by time
- Compression reduces storage for older data
- Continuous aggregates avoid re-computing rollups
- Retention policies auto-purge old data

### Query Tips

- Always include time range in time-series queries
- Use batch inserts (CopyFrom) for high-volume data
- Leverage continuous aggregates for reporting
- Consider connection pooling with pgxpool

## Testing

### Test Database Setup

```go
db := database.SetupTestDB(t)
defer database.TeardownTestDB(t, db)

// Run migrations
migrate -path migrations -database "postgres://test_db" up

// Test queries
runner, err := repos.Runner.GetByRaceID(ctx, raceID)
```

### Isolation

Tests should use separate database or schema to avoid conflicts.

## Monitoring and Maintenance

### Key Metrics

Monitor these PostgreSQL metrics:
- `pg_stat_statements` - Slow queries
- `pg_stat_user_tables` - Table sizes
- Connection pool utilization
- Replication lag (if using replication)

### Maintenance Tasks

```sql
-- Analyze table statistics
ANALYZE;

-- Vacuum dead rows
VACUUM ANALYZE;

-- Check TimescaleDB status
SELECT * FROM timescaledb_information.continuous_aggregates;
SELECT * FROM timescaledb_information.hypertables;
```

## Architecture Diagrams

See [docs/diagrams/data-flow.mmd](../diagrams/data-flow.mmd) for visual representation of data flow through the system.
