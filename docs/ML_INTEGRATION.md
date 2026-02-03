# ML Service Integration Architecture

## Overview

The ML Service Integration provides bidirectional communication between the Go backtesting service and the Python ML service using gRPC (low-latency predictions) and HTTP (batch operations). This enables automated strategy generation, evaluation, and feedback loops for continuous model improvement.

## Key Components

### 1. ML Client (`internal/ml/client.go`)
Primary gRPC interface with connection pooling, retry logic, and metrics collection.

**Methods**:
- `GetPrediction()` - Single prediction
- `EvaluateStrategy()` - Strategy ML evaluation
- `SubmitBacktestFeedback()` - Training feedback
- `GenerateStrategy()` - Strategy generation
- `BatchPredict()` - Bulk predictions

### 2. Cached ML Client (`internal/ml/cached_client.go`)
Wraps MLClient with in-memory LRU caching layer.

**Features**:
- 2-level caching (Check cache → Call gRPC → Cache)
- Partial batch caching (Only fetch uncached predictions)
- Strategy-aware cache invalidation
- Cache statistics tracking
- Prometheus metrics integration

### 3. HTTP Client (`internal/ml/http_client.go`)
REST client for batch operations and model training management.

**Methods**:
- `TrainModels()` - Initiate training job
- `GetTrainingStatus()` - Check job status
- `GetModelMetrics()` - Performance metrics
- `HealthCheck()` - Service health

### 4. Services

#### Strategy Generator (`internal/service/strategy_generator.go`)
Generates betting strategies from ML models using backtest results.

#### ML Feedback Service (`internal/service/ml_feedback.go`)
Manages feedback submission and periodic model retraining.

#### Strategy Evaluator (`internal/service/strategy_evaluator.go`)
Evaluates and ranks active strategies using ML + backtest metrics.

#### ML Orchestrator (`internal/service/ml_orchestrator.go`)
Orchestrates complete strategy discovery pipeline.

## Configuration

```yaml
ml_service:
  url: http://localhost:8000
  http_address: http://localhost:8000
  grpc_address: localhost:50051
  timeout_seconds: 30
  request_timeout_seconds: 30
  retry_attempts: 3
  cache_ttl_seconds: 3600          # 1 hour
  cache_max_size: 10000
  enable_strategy_generation: true
  enable_feedback_loop: true
  feedback_batch_size: 100
  retraining_interval_hours: 24
```

## Usage

### Strategy Discovery
```bash
make strategy-discovery
./cmd/strategy-discovery/main.go -c config/custom.yaml
```

### Feedback Submission
```bash
./cmd/ml-feedback/main.go submit --batch-size 100
./cmd/ml-feedback/main.go retrain
./cmd/ml-feedback/main.go status
```

### Check ML Status
```bash
./cmd/ml-status/main.go
```

## Metrics

**Prometheus Metrics**:
- `ml_predictions_total` - Prediction count by model type and cache hit
- `ml_prediction_latency_seconds` - Prediction latency histogram
- `ml_cache_hit_ratio` - Cache hit ratio gauge
- `ml_feedback_submitted_total` - Feedback submission count
- `ml_strategy_generation_total` - Strategy generation count by status
- `ml_training_jobs_total` - Training job count by model and status
- `ml_grpc_errors_total` - gRPC error count by method and type

## Testing

```bash
go test -v ./internal/ml/... -run TestPredictionCache
go test -v ./test/integration/... -run TestMLIntegration
```

## Performance

- **Cached Predictions**: <5ms
- **gRPC Predictions**: 50-200ms
- **HTTP Predictions**: 200-500ms
- **Throughput (Cached)**: 10,000+ predictions/second
- **Cache Memory**: ~100 bytes per entry

## Error Handling

Common errors and solutions:
- `ErrMLServiceUnavailable` - Check ML service health and network
- `ErrConnectionFailed` - Verify ML service address
- `ErrTimeout` - Increase timeout_seconds or check ML service performance
- `ErrInvalidPrediction` - Verify ML service model version
- `ErrStrategyGenerationFailed` - Verify constraints are valid

## See Also
- [API_REFERENCE.md](API_REFERENCE.md) - API documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment guide


## Purpose

Backtesting outputs are structured for ML strategy discovery. Each run exports a JSON payload containing strategy parameters, performance metrics, risk features, and bet-level signals.

## Export Format

The `MLExport` object includes:

- Strategy metadata (ID, name, version, parameters)
- Backtest summary (dates, capital, total bets)
- Metrics for each method (historical, Monte Carlo, walk-forward)
- Bet history with features (odds, confidence, EV, outcome)
- Equity curve and risk profile

## Feature Engineering

The backtest export is designed to provide:

- Risk-adjusted returns (Sharpe, Sortino, Calmar)
- Drawdown distribution and tail risk
- Consistency and overfit scores
- Odds and liquidity statistics
- Strategy parameter hashes for tracking experiments

## Workflow

1. Run backtest with `--ml-export` enabled.
2. Parse `output/backtest_results.json` into the ML service.
3. Extract features and label performance outcomes.
4. Train models to identify promising strategies.
5. Feed results into strategy selection pipeline.

## Example Pipeline

```bash
./bin/backtest --mode all --strategy simple_value --ml-export --output ./output/backtest_results.json
python ml-service/train.py --input ./output/backtest_results.json
```

## ML Service API

REST endpoints exposed by the ML service:

- GET /health
- GET /api/v1/health
- GET /api/v1/backtest-results
- GET /api/v1/backtest-results/{id}
- POST /api/v1/preprocess
- POST /api/v1/features/extract
- GET /api/v1/strategies/{strategy_id}/performance
- POST /api/v1/strategies/rank

gRPC service (port 50051):

- GetPrediction
- EvaluateStrategy
- GetFeatures
- HealthCheck
