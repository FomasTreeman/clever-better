# Component Guide for AI

This document provides detailed information about each system component, their responsibilities, and interaction patterns.

## Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Entry Points                             │
├───────────────┬───────────────┬─────────────────────────────────┤
│   cmd/bot     │ cmd/backtest  │       cmd/data-ingestion        │
│ Trading Bot   │ Backtest CLI  │      Data Collection            │
└───────┬───────┴───────┬───────┴─────────────┬───────────────────┘
        │               │                     │
        ▼               ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Internal Packages                           │
├─────────┬─────────┬─────────┬─────────┬─────────┬───────────────┤
│ betfair │ strategy│ backtest│ ml      │ service │ repository    │
│ API     │ Engine  │ Engine  │ Client  │ Logic   │ Data Access   │
└─────────┴─────────┴─────────┴────┬────┴─────────┴───────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                    External Services                            │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   ML Service    │    TimescaleDB  │      Betfair API            │
│   (Python)      │   (PostgreSQL)  │      (External)             │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

## Entry Points (cmd/)

### cmd/bot - Trading Bot

**Purpose**: Main application that executes live trading.

**Responsibilities**:
- Initialize all services and connections
- Subscribe to Betfair market streams
- Orchestrate prediction → decision → execution flow
- Handle graceful shutdown

**Key Dependencies**:
- `internal/betfair` - API communication
- `internal/ml` - Prediction requests
- `internal/strategy` - Trading logic
- `internal/repository` - Data persistence

**Configuration Required**:
- Betfair credentials (secrets)
- ML service endpoint
- Database connection
- Risk parameters

**When to Modify**:
- Adding new market types
- Changing initialization logic
- Adding new command-line flags

---

### cmd/backtest - Backtesting CLI

**Purpose**: Command-line tool for strategy validation.

**Responsibilities**:
- Parse command-line arguments
- Load historical data for specified period
- Run backtesting simulation
- Generate performance reports

**Key Dependencies**:
- `internal/backtest` - Simulation engine
- `internal/strategy` - Strategy interface
- `internal/ml` - Historical predictions
- `internal/repository` - Historical data access

**Example Usage**:
```bash
./backtest --start 2023-01-01 --end 2023-12-31 --strategy ml-ensemble
./backtest --monte-carlo --iterations 10000
./backtest --walk-forward --window 30
```

**When to Modify**:
- Adding new backtest modes
- Adding new report formats
- Adding new command-line options

---

### cmd/data-ingestion - Data Collection

**Purpose**: Service that collects and stores racing data.

**Responsibilities**:
- Fetch data from external sources
- Validate and transform data
- Store in TimescaleDB
- Handle incremental updates

**Key Dependencies**:
- `internal/datasource` - External source clients
- `internal/repository` - Data persistence
- `internal/config` - Configuration

**When to Modify**:
- Adding new data sources
- Changing data validation rules
- Modifying ingestion schedule

---

## Internal Packages (internal/)

### internal/betfair

**Purpose**: Betfair Exchange API client.

**Key Types**:
```go
// Client for REST API
type Client struct {
    httpClient *http.Client
    appKey     string
    sessionKey string
}

// StreamClient for real-time data
type StreamClient struct {
    conn       *websocket.Conn
    subscribed map[string]bool
}

// Common response types
type MarketBook struct { ... }
type Runner struct { ... }
type PlaceOrder struct { ... }
```

**Key Functions**:
- `NewClient(config)` - Create authenticated client
- `ListMarketCatalogue(filter)` - Find markets
- `ListMarketBook(marketIds)` - Get market data
- `PlaceOrders(marketId, orders)` - Execute bets

**When to Modify**:
- Adding new API endpoints
- Improving error handling
- Adding retry logic

---

### internal/strategy

**Purpose**: Strategy interface and implementations.

**Key Types**:
```go
// Strategy interface all strategies must implement
type Strategy interface {
    Name() string
    Evaluate(ctx context.Context, market *Market) ([]Signal, error)
    Parameters() map[string]interface{}
}

// Signal represents a trading signal
type Signal struct {
    RunnerID    string
    Side        Side      // BACK or LAY
    Confidence  float64
    TargetOdds  float64
    SuggestedStake float64
}
```

**When to Modify**:
- Implementing new strategies
- Changing signal generation logic
- Adding strategy parameters

---

### internal/backtest

**Purpose**: Backtesting engine for strategy validation.

**Key Types**:
```go
// Engine runs backtests
type Engine struct {
    strategy   Strategy
    portfolio  *Portfolio
    repository Repository
    mlClient   MLClient
}

// Portfolio tracks simulated capital
type Portfolio struct {
    InitialCapital float64
    CurrentCapital float64
    Positions      []Position
    TradeHistory   []Trade
}

// Metrics holds performance statistics
type Metrics struct {
    TotalReturn   float64
    SharpeRatio   float64
    MaxDrawdown   float64
    WinRate       float64
    ProfitFactor  float64
}
```

**Key Functions**:
- `NewEngine(config)` - Create backtest engine
- `Run(startDate, endDate)` - Execute backtest
- `CalculateMetrics()` - Compute performance stats

**When to Modify**:
- Adding new simulation modes (Monte Carlo, etc.)
- Adding new metrics
- Optimizing replay performance

---

### internal/ml

**Purpose**: ML service client for predictions.

**Key Types**:
```go
// Client communicates with ML service
type Client struct {
    httpClient *http.Client
    baseURL    string
    timeout    time.Duration
}

// PredictionRequest for API call
type PredictionRequest struct {
    RaceID    string             `json:"race_id"`
    RunnerID  string             `json:"runner_id"`
    Odds      float64            `json:"current_odds"`
    Features  map[string]float64 `json:"features"`
}

// PredictionResponse from API
type PredictionResponse struct {
    WinProbability   float64 `json:"win_probability"`
    PlaceProbability float64 `json:"place_probability"`
    ExpectedValue    float64 `json:"expected_value"`
    Confidence       float64 `json:"confidence"`
}
```

**When to Modify**:
- Adding batch prediction support
- Adding gRPC client
- Changing feature format

---

### internal/repository

**Purpose**: Data access layer for all persistence.

**Key Types**:
```go
// RaceRepository interface
type RaceRepository interface {
    FindByID(ctx context.Context, id string) (*Race, error)
    FindUpcoming(ctx context.Context, limit int) ([]Race, error)
    Save(ctx context.Context, race *Race) error
}

// TradeRepository interface
type TradeRepository interface {
    FindByID(ctx context.Context, id string) (*Trade, error)
    FindByRace(ctx context.Context, raceID string) ([]Trade, error)
    Save(ctx context.Context, trade *Trade) error
}

// OddsRepository interface
type OddsRepository interface {
    GetLatest(ctx context.Context, raceID, runnerID string) (*OddsSnapshot, error)
    GetHistory(ctx context.Context, raceID, runnerID string, since time.Time) ([]OddsSnapshot, error)
    SaveBatch(ctx context.Context, snapshots []OddsSnapshot) error
}
```

**When to Modify**:
- Adding new query methods
- Optimizing queries
- Adding caching

---

### internal/models

**Purpose**: Domain models shared across packages.

**Key Types**:
```go
type Race struct {
    ID             string
    ScheduledStart time.Time
    Track          string
    Grade          string
    Distance       int
    Runners        []Runner
    Status         RaceStatus
}

type Runner struct {
    ID         string
    RaceID     string
    TrapNumber int
    Name       string
    Form       string
    Metadata   map[string]interface{}
}

type Trade struct {
    ID         string
    RaceID     string
    RunnerID   string
    Side       Side
    Odds       float64
    Stake      float64
    Status     TradeStatus
    ProfitLoss float64
    ExecutedAt time.Time
}
```

**When to Modify**:
- Adding new fields to models
- Adding new model types
- Changing validation rules

---

## ML Service (ml-service/)

### app/api - FastAPI Routes

**Purpose**: HTTP API endpoints for predictions.

**Key Endpoints**:
- `POST /api/v1/predict` - Single prediction
- `POST /api/v1/predict/batch` - Batch predictions
- `GET /api/v1/model/info` - Model metadata
- `GET /health` - Health check

**When to Modify**:
- Adding new API endpoints
- Changing request/response schemas
- Adding validation

---

### app/ml - Machine Learning Core

**Purpose**: Feature engineering, model training, inference.

**Key Classes**:
```python
class FeatureGenerator:
    """Generates features from raw data."""
    def generate(self, race: Race, runner: Runner) -> dict[str, float]: ...

class ModelWrapper:
    """Wraps trained model for inference."""
    def predict(self, features: dict) -> Prediction: ...
    def predict_batch(self, features_list: list[dict]) -> list[Prediction]: ...

class TrainingPipeline:
    """Orchestrates model training."""
    def train(self, data: DataFrame) -> Model: ...
    def evaluate(self, model: Model, test_data: DataFrame) -> Metrics: ...
```

**When to Modify**:
- Adding new features
- Changing model architecture
- Modifying training pipeline

---

## Data Flow Summary

### Live Trading Flow
```
Betfair Stream → internal/betfair/stream.go
     ↓
internal/bot/market.go (process market update)
     ↓
internal/ml/client.go → ML Service API
     ↓
internal/strategy/evaluate.go (generate signals)
     ↓
internal/bot/execution.go → internal/betfair/client.go
     ↓
internal/repository/trade.go → TimescaleDB
```

### Backtesting Flow
```
cmd/backtest/main.go (parse args)
     ↓
internal/backtest/engine.go (initialize)
     ↓
internal/repository/odds.go → TimescaleDB (load history)
     ↓
internal/backtest/replay.go (simulate market)
     ↓
internal/ml/client.go → ML Service (get predictions)
     ↓
internal/backtest/portfolio.go (simulate trades)
     ↓
internal/backtest/metrics.go (calculate results)
```

### Data Ingestion Flow
```
cmd/data-ingestion/main.go (start)
     ↓
internal/datasource/betfair.go (fetch data)
     ↓
internal/datasource/validator.go (validate)
     ↓
internal/datasource/transformer.go (transform)
     ↓
internal/repository/race.go → TimescaleDB (save)
```
