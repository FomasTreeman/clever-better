# End-to-End Tests

This directory contains end-to-end tests that validate the complete Clever Better trading bot workflow from data ingestion through ML-driven trading decisions.

## Test Coverage

- **Complete Trading Workflow**: Data ingestion → historical data storage → backtesting → ML training → strategy generation → simulated live trading
- **ML Feedback Loop**: Backtest results → ML service training → strategy recommendations → evaluation
- **Strategy Lifecycle**: Strategy discovery → backtesting validation → activation → performance monitoring → deactivation
- **Failure Recovery**: Database connection loss → reconnection → data consistency verification
- **Multi-Strategy**: Multiple strategies running concurrently with risk management

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ installed
- Test environment configured

## Running E2E Tests

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run all E2E tests
make test-e2e

# Run specific test
go test -v -tags=e2e -run TestCompleteTrading ./test/e2e/

# Clean up
docker-compose -f docker-compose.test.yml down -v
```

## Test Data

Test fixtures are located in `fixtures/` directory:
- `sample_race_data.json`: Realistic historical race data
- `sample_odds_data.json`: Historical odds snapshots
- `sample_backtest_config.yaml`: Backtest configuration

## Test Environment

The test environment uses:
- TimescaleDB for time-series data storage
- Redis for caching
- ML service for predictions
- Mocked Betfair API responses

## Writing New E2E Tests

1. Add test data fixtures to `fixtures/`
2. Implement test function with `//go:build e2e` tag
3. Use test helpers from `test/helpers/`
4. Clean up test data after each run
5. Document test in this README

## Troubleshooting

- **Test database connection failures**: Ensure Docker containers are running
- **Slow test execution**: Check Docker resource allocation
- **Flaky tests**: Review timing assumptions and add appropriate waits
