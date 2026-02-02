# Coding Conventions

This document defines the coding standards and patterns used throughout the Clever Better codebase.

## Go Conventions

### Package Organization

```
internal/
├── betfair/           # One package per domain concept
│   ├── client.go      # Public types and functions
│   ├── client_test.go # Tests in same package
│   ├── stream.go      # Related functionality grouped
│   ├── types.go       # Type definitions
│   └── internal/      # Package-internal helpers (rare)
```

### Naming

```go
// Package names: lowercase, no underscores
package betfair  // Good
package bet_fair // Bad

// Exported (public): PascalCase
type MarketBook struct{}
func NewClient() *Client {}

// Unexported (private): camelCase
type marketCache struct{}
func parseResponse() {}

// Acronyms: consistent casing
type HTTPClient struct{}  // Good
type HttpClient struct{}  // Bad
func GetHTTPStatus() {}   // Good

// Interfaces: describe behavior
type Reader interface{}   // Good (verb-er)
type Stringer interface{} // Good (verb-er)
type IReader interface{}  // Bad (no I prefix)

// Constants
const MaxRetries = 3              // Exported
const defaultTimeout = 30         // Unexported
const (
    StatusPending   = "pending"   // Grouped constants
    StatusCompleted = "completed"
)
```

### Error Handling

```go
// Always handle errors explicitly
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// Define sentinel errors for expected conditions
var ErrNotFound = errors.New("not found")
var ErrInvalidInput = errors.New("invalid input")

// Check specific errors
if errors.Is(err, ErrNotFound) {
    // Handle not found
}

// Custom error types for rich context
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Wrap errors with context
return fmt.Errorf("failed to fetch race %s: %w", raceID, err)
```

### Context Usage

```go
// Always accept context as first parameter
func (s *Service) GetRace(ctx context.Context, id string) (*Race, error)

// Pass context to all downstream calls
func (s *Service) ProcessRace(ctx context.Context, raceID string) error {
    race, err := s.repo.GetRace(ctx, raceID)
    if err != nil {
        return err
    }
    return s.processor.Process(ctx, race)
}

// Respect cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue processing
}
```

### Struct Initialization

```go
// Use named fields for clarity
client := &Client{
    baseURL:    "https://api.betfair.com",
    timeout:    30 * time.Second,
    httpClient: http.DefaultClient,
}

// Use constructor functions for complex initialization
func NewClient(config *Config) (*Client, error) {
    if config.BaseURL == "" {
        return nil, errors.New("base URL required")
    }
    return &Client{
        baseURL: config.BaseURL,
        timeout: config.Timeout,
    }, nil
}

// Use options pattern for many optional parameters
type ClientOption func(*Client)

func WithTimeout(d time.Duration) ClientOption {
    return func(c *Client) { c.timeout = d }
}

func NewClient(baseURL string, opts ...ClientOption) *Client {
    c := &Client{baseURL: baseURL, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### Testing

```go
// Table-driven tests
func TestCalculate(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
        wantErr  bool
    }{
        {"positive", 5, 10, false},
        {"zero", 0, 0, false},
        {"negative", -1, 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Calculate(tt.input)

            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %d, want %d", result, tt.expected)
            }
        })
    }
}

// Use interfaces for mocking
type Repository interface {
    Get(ctx context.Context, id string) (*Entity, error)
}

type mockRepository struct {
    getFunc func(ctx context.Context, id string) (*Entity, error)
}

func (m *mockRepository) Get(ctx context.Context, id string) (*Entity, error) {
    return m.getFunc(ctx, id)
}
```

### Logging

```go
import "log/slog"

// Use structured logging
logger := slog.Default()

logger.Info("trade executed",
    "trade_id", trade.ID,
    "amount", trade.Amount,
    "odds", trade.Odds,
)

logger.Error("failed to place order",
    "error", err,
    "market_id", marketID,
)

// Include context in logs
logger.InfoContext(ctx, "processing request",
    "request_id", requestID,
)
```

---

## Python Conventions

### Package Structure

```
ml-service/
├── app/
│   ├── __init__.py
│   ├── main.py           # Application entry
│   ├── api/
│   │   ├── __init__.py
│   │   └── routes.py     # API routes
│   ├── core/
│   │   ├── __init__.py
│   │   └── config.py     # Configuration
│   └── ml/
│       ├── __init__.py
│       └── model.py      # ML code
└── tests/
    ├── __init__.py
    └── test_api.py
```

### Naming

```python
# Modules: lowercase with underscores
feature_engineering.py  # Good
FeatureEngineering.py   # Bad

# Classes: PascalCase
class PredictionService:
    pass

# Functions and variables: snake_case
def calculate_expected_value():
    pass

max_iterations = 100

# Constants: UPPERCASE
MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30

# Private: leading underscore
def _internal_helper():
    pass

class MyClass:
    def __init__(self):
        self._private_attr = None
```

### Type Hints

```python
from typing import Optional, List, Dict, Union
from datetime import datetime

# Function signatures
def predict(
    race_id: str,
    runner_id: str,
    features: Dict[str, float],
    confidence_threshold: float = 0.5,
) -> Optional[Prediction]:
    """Generate prediction for a runner."""
    ...

# Class attributes
class PredictionService:
    model: Optional[Model]
    threshold: float

    def __init__(self, model_path: str, threshold: float = 0.5) -> None:
        self.model = None
        self.threshold = threshold

# Complex types
Features = Dict[str, float]
PredictionResult = Dict[str, Union[float, str]]
```

### Dataclasses

```python
from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional

@dataclass
class Prediction:
    """Prediction result from ML model."""
    race_id: str
    runner_id: str
    win_probability: float
    place_probability: float
    confidence: float
    model_version: str
    predicted_at: datetime = field(default_factory=datetime.utcnow)

@dataclass
class Race:
    """Race domain model."""
    id: str
    scheduled_start: datetime
    track: str
    runners: List["Runner"] = field(default_factory=list)

    def is_upcoming(self) -> bool:
        return self.scheduled_start > datetime.utcnow()
```

### Exception Handling

```python
# Define custom exceptions
class PredictionError(Exception):
    """Base exception for prediction errors."""
    pass

class ModelNotLoadedError(PredictionError):
    """Model not loaded."""
    pass

class FeatureError(PredictionError):
    """Feature engineering error."""
    pass

# Use specific exceptions
def predict(features: dict) -> Prediction:
    if self.model is None:
        raise ModelNotLoadedError("Model not loaded")

    try:
        result = self.model.predict(features)
    except ValueError as e:
        raise FeatureError(f"Invalid features: {e}") from e

    return Prediction(...)

# Handle exceptions explicitly
try:
    prediction = service.predict(features)
except ModelNotLoadedError:
    logger.error("Model not loaded, returning fallback")
    return fallback_prediction()
except FeatureError as e:
    logger.warning(f"Feature error: {e}")
    raise HTTPException(status_code=400, detail=str(e))
```

### Async/Await

```python
import asyncio
from typing import List

# Async function definition
async def fetch_predictions(
    race_ids: List[str],
) -> List[Prediction]:
    tasks = [fetch_prediction(race_id) for race_id in race_ids]
    return await asyncio.gather(*tasks)

# Async context manager
async def get_db_connection():
    conn = await asyncpg.connect(DATABASE_URL)
    try:
        yield conn
    finally:
        await conn.close()

# Using async with FastAPI
@app.post("/predict")
async def predict_endpoint(request: PredictRequest) -> PredictResponse:
    prediction = await prediction_service.predict(request)
    return PredictResponse.from_prediction(prediction)
```

### Testing

```python
import pytest
from unittest.mock import Mock, patch, AsyncMock

# Fixtures
@pytest.fixture
def mock_model():
    model = Mock()
    model.predict.return_value = np.array([0.25])
    return model

@pytest.fixture
def prediction_service(mock_model):
    return PredictionService(model=mock_model)

# Basic test
def test_predict_returns_prediction(prediction_service):
    result = prediction_service.predict(
        race_id="123",
        features={"trap": 1}
    )

    assert result is not None
    assert 0 <= result.win_probability <= 1

# Parametrized test
@pytest.mark.parametrize("confidence,expected", [
    (0.9, True),
    (0.5, True),
    (0.3, False),
])
def test_meets_threshold(confidence, expected):
    result = meets_threshold(confidence, threshold=0.4)
    assert result == expected

# Async test
@pytest.mark.asyncio
async def test_async_predict():
    result = await async_predict(features)
    assert result is not None

# Mocking
@patch("app.ml.model.load_model")
def test_load_model(mock_load):
    mock_load.return_value = Mock()
    service = PredictionService()
    mock_load.assert_called_once()
```

---

## SQL Conventions

### Naming

```sql
-- Tables: plural snake_case
CREATE TABLE races (...);
CREATE TABLE odds_snapshots (...);

-- Columns: snake_case
CREATE TABLE races (
    id UUID PRIMARY KEY,
    scheduled_start TIMESTAMPTZ,
    track_name VARCHAR(100)
);

-- Indexes: idx_table_columns
CREATE INDEX idx_races_scheduled_start ON races (scheduled_start);
CREATE INDEX idx_odds_race_runner_time ON odds_snapshots (race_id, runner_id, time);

-- Constraints: table_column_type
CONSTRAINT races_track_name_not_empty CHECK (track_name <> '')
```

### Query Style

```sql
-- Multi-line for readability
SELECT
    r.id,
    r.scheduled_start,
    r.track_name,
    COUNT(run.id) as runner_count
FROM races r
LEFT JOIN runners run ON run.race_id = r.id
WHERE r.scheduled_start > NOW()
  AND r.status = 'scheduled'
GROUP BY r.id, r.scheduled_start, r.track_name
HAVING COUNT(run.id) >= 6
ORDER BY r.scheduled_start ASC
LIMIT 100;

-- Always use parameters, never string interpolation
-- Bad: f"SELECT * FROM races WHERE id = '{race_id}'"
-- Good: "SELECT * FROM races WHERE id = $1", race_id
```

---

## Git Conventions

### Commit Messages

```
<type>(<scope>): <short description>

<optional body>

<optional footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting (no code change)
- `refactor`: Code restructure (no behavior change)
- `test`: Tests
- `chore`: Maintenance

**Examples:**
```
feat(backtest): add Monte Carlo simulation

Implement Monte Carlo simulation for strategy validation.
- Add MonteCarloSimulator class
- Support configurable iterations
- Calculate confidence intervals

Closes #123
```

```
fix(betfair): handle rate limit errors

Add exponential backoff when rate limited.
```

### Branch Naming

```
feature/add-monte-carlo-backtest
bugfix/fix-odds-calculation
hotfix/patch-auth-vulnerability
refactor/simplify-repository-interface
docs/update-api-reference
```

---

## Documentation Conventions

### Go Documentation

```go
// Package betfair provides a client for the Betfair Exchange API.
//
// It supports both REST API calls and streaming market data.
package betfair

// Client provides methods for interacting with the Betfair Exchange API.
// It handles authentication, rate limiting, and connection management.
type Client struct {
    // ...
}

// PlaceOrders submits one or more orders to the specified market.
// It returns the order results or an error if the operation failed.
//
// Example:
//
//     orders := []PlaceInstruction{{...}}
//     results, err := client.PlaceOrders(ctx, marketID, orders)
func (c *Client) PlaceOrders(ctx context.Context, marketID string, orders []PlaceInstruction) ([]PlaceResult, error) {
    // ...
}
```

### Python Documentation

```python
"""
ML Service for Clever Better.

This module provides machine learning predictions for greyhound racing.
"""

class PredictionService:
    """Service for generating race predictions.

    This service wraps the trained ML model and provides methods for
    single and batch predictions.

    Attributes:
        model: The loaded ML model.
        threshold: Minimum confidence threshold for predictions.

    Example:
        >>> service = PredictionService("models/v1.pkl")
        >>> prediction = service.predict("race123", "runner456", features)
        >>> print(prediction.win_probability)
        0.25
    """

    def predict(
        self,
        race_id: str,
        runner_id: str,
        features: Dict[str, float],
    ) -> Optional[Prediction]:
        """Generate prediction for a single runner.

        Args:
            race_id: Unique identifier for the race.
            runner_id: Unique identifier for the runner.
            features: Dictionary of feature name to value.

        Returns:
            Prediction object if successful, None if prediction failed.

        Raises:
            ModelNotLoadedError: If model is not loaded.
            FeatureError: If features are invalid.
        """
        ...
```
