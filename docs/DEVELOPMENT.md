# Development Guide

This document provides guidelines for developing Clever Better, including environment setup, coding standards, and workflow practices.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Database Setup](#database-setup)
- [Code Organization](#code-organization)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Git Workflow](#git-workflow)
- [Debugging](#debugging)
- [Performance](#performance)

## Development Environment Setup

### Prerequisites

```bash
# macOS
brew install go python@3.11 docker docker-compose terraform migrate jq

# Ubuntu
sudo apt update
sudo apt install golang python3.11 docker.io docker-compose terraform golang-migrate jq
```

### Initial Setup

```bash
# Clone repository
git clone https://github.com/yourusername/clever-better.git
cd clever-better

# Install development tools
make install-tools

# Set up Python virtual environment
cd ml-service
python3.11 -m venv venv
source venv/bin/activate
pip install -r requirements-dev.txt
cd ..

# Copy configuration
cp config/config.yaml.example config/config.yaml
# Edit config/config.yaml with your settings
```

## Database Setup

### Prerequisites

Ensure PostgreSQL 15 and TimescaleDB extension are installed and running locally.

```bash
# macOS
brew install postgresql@15 timescaledb

# Ubuntu
sudo apt install postgresql-15 postgresql-15-timescaledb

# Start PostgreSQL
pg_ctl -D /usr/local/var/postgres start  # macOS
sudo systemctl start postgresql            # Ubuntu
```

### Initial Database Setup

```bash
# Create database and run migrations
make db-setup

# Verify connection
make db-health-check

# Check migration status
make db-migrate-status
```

### Database Commands

```bash
# Create database
make db-create

# Run migrations up
make db-migrate-up

# Rollback all migrations
make db-migrate-down

# Drop database
make db-drop

# Reset database (drop and recreate)
make db-reset

# Create new migration
make db-migrate-create NAME=add_new_column

# Force migration to specific version
make db-migrate-force VERSION=3
```

### Database Configuration

Edit `config/config.yaml`:

```yaml
database:
  host: localhost
  port: 5432
  name: clever_better
  user: postgres
  password: postgres
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  statement_cache_mode: describe
```

### TimescaleDB Verification

After database setup, verify TimescaleDB installation:

```bash
psql -U postgres -d clever_better -c "SELECT default_version, installed_version FROM pg_available_extensions WHERE name = 'timescaledb';"
```

Expected output should show installed TimescaleDB version.

### Working with Migrations

Migrations use `golang-migrate/migrate`. Each migration consists of up (.up.sql) and down (.down.sql) files.

#### Current Migrations

1. `000001_create_core_tables` - Core races and runners tables
2. `000002_create_hypertables` - TimescaleDB odds_snapshots hypertable
3. `000003_create_trading_tables` - Strategies and bets hypertable
4. `000004_create_ml_tables` - Models and predictions hypertable
5. `000005_create_strategy_performance` - Strategy performance hypertable and continuous aggregate

#### Creating New Migrations

```bash
# Create migration pair
make db-migrate-create NAME=add_new_feature

# Edit migrations/[timestamp]_add_new_feature.up.sql
# Edit migrations/[timestamp]_add_new_feature.down.sql

# Test migration
make db-reset
make db-migrate-up
```

#### Testing Migrations

```bash
# Apply migrations
make db-migrate-up

# Verify schema
psql -U postgres -d clever_better -c "\d"

# Rollback
make db-migrate-down
```

### Data Access Layer

See [Database Architecture](./DATABASE.md) for detailed schema and repository documentation.

#### Using Repositories

```go
// Initialize database
db, err := database.NewDB(config.Database)
defer db.Close(ctx)

// Create repositories
repos, err := repository.NewRepositories(db)

// Query examples
races, err := repos.Race.GetUpcoming(ctx, 10)
runners, err := repos.Runner.GetByRaceID(ctx, raceID)
bets, err := repos.Bet.GetByStrategyID(ctx, strategyID, start, end)
```

#### Query Patterns

```go
// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Batch operations
odds := []*models.OddsSnapshot{...}
err := repos.Odds.InsertBatch(ctx, odds)

// Transactions
err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
    // Execute transactional operations
    return nil
})
```

# Verify setup
make test
```

### IDE Setup

#### VS Code

Recommended extensions:
```json
{
  "recommendations": [
    "golang.go",
    "ms-python.python",
    "hashicorp.terraform",
    "ms-azuretools.vscode-docker",
    "editorconfig.editorconfig",
    "streetsidesoftware.code-spell-checker"
  ]
}
```

Settings (`.vscode/settings.json`):
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.formatTool": "goimports",
  "python.linting.enabled": true,
  "python.linting.flake8Enabled": true,
  "python.formatting.provider": "black",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

#### GoLand / PyCharm

- Enable Go modules integration
- Configure golangci-lint as external tool
- Set Python interpreter to virtual environment

## Code Organization

### Go Package Structure

```
internal/
├── backtest/           # Backtesting engine
│   ├── engine.go      # Core backtest logic
│   ├── portfolio.go   # Portfolio simulation
│   ├── metrics.go     # Performance metrics
│   └── engine_test.go # Tests
│
├── betfair/           # Betfair API client
│   ├── client.go      # HTTP client
│   ├── stream.go      # WebSocket streaming
│   ├── auth.go        # Authentication
│   ├── types.go       # API types
│   └── client_test.go # Tests
│
├── bot/               # Trading bot orchestration
│   ├── bot.go         # Main bot logic
│   ├── market.go      # Market handling
│   ├── execution.go   # Trade execution
│   └── bot_test.go    # Tests
│
├── config/            # Configuration management
│   ├── config.go      # Config loading
│   ├── validation.go  # Config validation
│   └── config_test.go # Tests
│
├── models/            # Domain models
│   ├── race.go        # Race model
│   ├── runner.go      # Runner model
│   ├── trade.go       # Trade model
│   └── prediction.go  # Prediction model
│
├── repository/        # Data access layer
│   ├── race.go        # Race repository
│   ├── trade.go       # Trade repository
│   └── postgres.go    # PostgreSQL implementation
│
└── service/           # Business logic
    ├── trading.go     # Trading service
    ├── prediction.go  # Prediction service
    └── trading_test.go
```

### Python Package Structure

```
ml-service/
├── app/
│   ├── __init__.py
│   ├── main.py           # FastAPI application
│   ├── api/
│   │   ├── __init__.py
│   │   ├── routes.py     # API routes
│   │   └── models.py     # Pydantic models
│   ├── core/
│   │   ├── __init__.py
│   │   ├── config.py     # Configuration
│   │   └── logging.py    # Logging setup
│   ├── ml/
│   │   ├── __init__.py
│   │   ├── features.py   # Feature engineering
│   │   ├── model.py      # Model wrapper
│   │   ├── training.py   # Training pipeline
│   │   └── inference.py  # Prediction serving
│   └── db/
│       ├── __init__.py
│       ├── connection.py # Database connection
│       └── queries.py    # SQL queries
│
├── tests/
│   ├── __init__.py
│   ├── conftest.py       # Pytest fixtures
│   ├── test_api.py       # API tests
│   ├── test_features.py  # Feature tests
│   └── test_model.py     # Model tests
│
├── models/               # Saved model artifacts
├── requirements.txt
├── requirements-dev.txt
└── Dockerfile
```

## Coding Standards

### Go Standards

**Naming Conventions:**
```go
// Package names: lowercase, single word
package betfair

// Exported types: PascalCase
type RaceResult struct { ... }

// Unexported types: camelCase
type marketCache struct { ... }

// Constants: PascalCase for exported, camelCase for unexported
const MaxRetries = 3
const defaultTimeout = 30 * time.Second

// Interfaces: verb-er pattern when possible
type Runner interface {
    Run(ctx context.Context) error
}
```

**Error Handling:**
```go
// Always check errors
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Use sentinel errors for expected conditions
var ErrNotFound = errors.New("not found")

func GetRace(id string) (*Race, error) {
    race, err := repo.FindByID(id)
    if err != nil {
        return nil, fmt.Errorf("get race %s: %w", id, err)
    }
    if race == nil {
        return nil, ErrNotFound
    }
    return race, nil
}
```

**Context Usage:**
```go
// Always accept context as first parameter
func (s *Service) ProcessRace(ctx context.Context, raceID string) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Pass context to downstream calls
    return s.repo.Save(ctx, race)
}
```

### Python Standards

**Type Hints:**
```python
from typing import Optional, List, Dict
from datetime import datetime

def predict(
    race_id: str,
    runner_id: str,
    features: Dict[str, float],
) -> Optional[Prediction]:
    """Generate prediction for a runner.

    Args:
        race_id: Unique race identifier
        runner_id: Unique runner identifier
        features: Feature dictionary

    Returns:
        Prediction object or None if prediction failed
    """
    ...
```

**Dataclasses:**
```python
from dataclasses import dataclass
from datetime import datetime

@dataclass
class Prediction:
    race_id: str
    runner_id: str
    win_probability: float
    place_probability: float
    confidence: float
    model_version: str
    predicted_at: datetime
```

**Exception Handling:**
```python
class PredictionError(Exception):
    """Base exception for prediction errors."""
    pass

class ModelNotLoadedError(PredictionError):
    """Raised when model is not loaded."""
    pass

def predict(features: dict) -> Prediction:
    if self.model is None:
        raise ModelNotLoadedError("Model not loaded")

    try:
        result = self.model.predict(features)
    except Exception as e:
        logger.error(f"Prediction failed: {e}")
        raise PredictionError(f"Prediction failed: {e}") from e
```

## Testing

### Go Testing

```go
// Table-driven tests
func TestCalculateROI(t *testing.T) {
    tests := []struct {
        name     string
        initial  float64
        final    float64
        expected float64
    }{
        {"positive return", 1000, 1100, 0.10},
        {"negative return", 1000, 900, -0.10},
        {"no change", 1000, 1000, 0.00},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculateROI(tt.initial, tt.final)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

// Mocking interfaces
type mockRepository struct {
    races map[string]*Race
}

func (m *mockRepository) FindByID(ctx context.Context, id string) (*Race, error) {
    race, ok := m.races[id]
    if !ok {
        return nil, nil
    }
    return race, nil
}
```

### Python Testing

```python
import pytest
from unittest.mock import Mock, patch

@pytest.fixture
def mock_model():
    model = Mock()
    model.predict.return_value = np.array([0.25, 0.55])
    return model

def test_predict_success(mock_model):
    service = PredictionService(model=mock_model)

    result = service.predict(
        race_id="123",
        runner_id="456",
        features={"trap": 1, "form": 0.8}
    )

    assert result.win_probability == pytest.approx(0.25, rel=0.01)
    mock_model.predict.assert_called_once()

@pytest.mark.asyncio
async def test_api_predict(client):
    response = await client.post("/api/v1/predict", json={
        "race_id": "123",
        "runner_id": "456",
        "current_odds": 4.0,
        "features": {}
    })

    assert response.status_code == 200
    assert "win_probability" in response.json()
```

### Running Tests

```bash
# All tests
make test

# Go tests with coverage
make go-test
go tool cover -html=coverage.out

# Python tests with coverage
make py-test-cov

# Integration tests (requires docker-up)
make test-integration
```

## Git Workflow

### Branch Naming

```
feature/add-monte-carlo-simulation
bugfix/fix-odds-calculation
hotfix/patch-security-vulnerability
refactor/simplify-backtest-engine
docs/update-api-reference
```

### Commit Messages

```
feat(backtest): add Monte Carlo simulation support

Implement Monte Carlo simulation for strategy validation with
configurable number of iterations and confidence intervals.

- Add MonteCarloSimulator type
- Implement parallel simulation execution
- Add confidence interval calculation
- Update CLI to support --monte-carlo flag

Closes #123
```

**Commit Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Tests
- `chore`: Maintenance

### Pull Request Process

1. Create feature branch from `main`
2. Make changes with atomic commits
3. Push branch and create PR
4. Wait for CI checks to pass
5. Request review from team member
6. Address review feedback
7. Squash and merge when approved

## Debugging

### Go Debugging

```bash
# Run with delve debugger
dlv debug ./cmd/bot

# Attach to running process
dlv attach <pid>

# Debug test
dlv test ./internal/backtest
```

**Log Levels:**
```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

logger.Debug("processing race", "race_id", raceID)
logger.Info("trade executed", "trade_id", tradeID, "amount", amount)
logger.Error("failed to connect", "error", err)
```

### Python Debugging

```python
# Use pdb
import pdb; pdb.set_trace()

# Or ipdb for better experience
import ipdb; ipdb.set_trace()

# Or breakpoint() in Python 3.7+
breakpoint()
```

**Structured Logging:**
```python
import structlog

logger = structlog.get_logger()

logger.info("prediction_generated",
    race_id=race_id,
    probability=0.25,
    confidence=0.85
)
```

## Performance

### Go Profiling

```go
import (
    "net/http"
    _ "net/http/pprof"
)

func main() {
    // Enable pprof endpoint
    go func() {
        http.ListenAndServe(":6060", nil)
    }()

    // ... rest of application
}
```

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine dump
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

### Python Profiling

```python
import cProfile
import pstats

# Profile a function
cProfile.run('predict(features)', 'output.prof')

# Analyze results
stats = pstats.Stats('output.prof')
stats.sort_stats('cumulative')
stats.print_stats(20)
```

### Database Query Optimization

```sql
-- Enable query logging (development only)
SET log_statement = 'all';
SET log_duration = on;

-- Explain query plan
EXPLAIN ANALYZE
SELECT * FROM odds_snapshots
WHERE race_id = $1
  AND time > NOW() - INTERVAL '1 hour'
ORDER BY time DESC;

-- Check for missing indexes
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats
WHERE tablename = 'odds_snapshots';
```

## Local Health Check Testing

### Testing Health Endpoints

The Go services expose health check endpoints for container orchestration:

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/health` | Basic liveness | `{"status": "ok", "service": "bot"}` |
| `/ready` | Readiness check (includes DB) | `{"status": "ok", "checks": {...}}` |
| `/live` | Kubernetes liveness probe | `{"status": "ok"}` |

### Quick Health Check

```bash
# Start services
make docker-up

# Test health endpoints
make health-check-local

# Or manually:
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/ready | jq .
curl -s http://localhost:8000/health | jq .
```

### Health Check Script

Use the provided script for CI/CD or debugging:

```bash
# Basic usage
./scripts/health-check.sh http://localhost:8080/health

# With custom timeout and retries
./scripts/health-check.sh http://localhost:8080/health 10 5
```

## Testing Docker Builds Locally

### Building with Version Information

```bash
# Build with version info
make docker-build

# Or with custom version
VERSION=v1.2.3 make docker-build

# Verify version embedded
docker run --rm clever-better-bot:latest /app/bin/bot --version
```

### Testing the Full Stack

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f bot

# Run health checks
make health-check-local

# Tear down
docker-compose down
```

### Building for CI

```bash
# Build binaries for Linux (CI target platform)
make ci-build

# Build Docker images with proper tags
ENV=dev make ci-docker-build
```

## Running CI Pipeline Locally

You can run the CI pipeline locally using [act](https://github.com/nektos/act):

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run the build-test job
act -j build-test

# Run with secrets
act -j build-test --secret-file .secrets

# List available jobs
act -l
```

### Secrets File Format (.secrets)

```
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
SLACK_WEBHOOK_URL=https://hooks.slack.com/...
```

## Makefile Targets for Development

```bash
# Show all targets
make help

# Build with version info
make build

# Run tests with coverage
make go-test

# CI-specific targets
make ci-test      # Run tests with CI output
make ci-build     # Build Linux binaries
make version      # Show version info
```
