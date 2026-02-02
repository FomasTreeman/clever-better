# ML Integration

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
