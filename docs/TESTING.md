# Testing Guide

## Overview

This document provides comprehensive guidance on testing the Clever Better betting bot system. It covers all test types, setup procedures, and best practices.

## Test Categories

### 1. Unit Tests

**Purpose**: Test individual components in isolation

**Location**: `internal/*/test.go` files

**Running**:
```bash
make test-unit
# or
go test ./internal/... -v
```

**Key Areas**:
- Strategy evaluation logic
- Data validation rules  
- ML client caching and retry behavior
- Configuration parsing
- Metrics collection

**Example**:
```go
func TestStrategyEvaluator_CalculateCompositeScore(t *testing.T) {
    evaluator := NewStrategyEvaluator(mockRepo)
    score := evaluator.CalculateCompositeScore(metrics)
    assert.Greater(t, score, 0.0)
}
```

### 2. Integration Tests

**Purpose**: Test interactions with external dependencies

**Location**: `test/integration/`

**Running**:
```bash
make test-integration
# or
go test ./test/integration/... -tags=integration -v
```

**Requirements**:
- Docker (for TimescaleDB)
- Test environment variables

**Coverage**:
- Database operations (TimescaleDB)
- Betfair API integration
- ML service API calls
- Caching behavior with Redis

**Setup**:
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./test/integration/...

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### 3. End-to-End Tests

**Purpose**: Validate complete workflows from start to finish

**Location**: `test/e2e/`

**Running**:
```bash
make test-e2e
# or
go test ./test/e2e/... -tags=e2e -v -timeout 10m
```

**Workflows Tested**:
1. Complete trading workflow (data ingestion → prediction → bet placement)
2. ML feedback loop (predictions → outcomes → model updates)
3. Strategy lifecycle (activation → execution → deactivation)
4. Failure recovery (service restart, data loss scenarios)
5. Multi-strategy execution with bankroll management

**Test Environment**:
```bash
# Start full stack
docker-compose up -d

# Verify services
docker-compose ps

# Run E2E tests
go test -tags=e2e ./test/e2e/... -v
```

### 4. Python ML Service Tests

**Purpose**: Test ML model training, predictions, and API endpoints

**Location**: `ml-service/tests/`

**Running**:
```bash
# From ml-service directory
pytest tests/ -v --cov=app

# Specific test types
pytest tests/test_api_integration.py -v
pytest tests/test_features.py -v
pytest tests/test_model_io.py -v
```

**Coverage**:
- Feature engineering
- Model training workflows
- Prediction endpoints (single and batch)
- Model persistence and versioning
- API integration and error handling

## Test Data and Fixtures

### Race Data Fixtures

Located in `test/fixtures/races.json`:
```json
{
  "event_id": "1.198765432",
  "market_id": "1.198765433",
  "start_time": "2026-02-15T14:30:00Z",
  "venue": "Ascot",
  "distance": 1600,
  "going": "Good",
  "runners": [...]
}
```

### Odds Snapshots

Located in `test/fixtures/odds_snapshots.json`:
- Time-series odds data
- Back and lay prices
- Market depth information

### Backtest Results

Located in `test/fixtures/backtest_results.json`:
- Historical backtest outcomes
- Performance metrics
- Strategy comparisons

## Test Environment Setup

### Local Development

1. **Install Dependencies**:
   ```bash
   # Go dependencies
   go mod download
   
   # Python dependencies  
   cd ml-service
   pip install -r requirements-dev.txt
   ```

2. **Start Test Infrastructure**:
   ```bash
   docker-compose -f docker-compose.test.yml up -d
   ```

3. **Configure Environment**:
   ```bash
   export DATABASE_URL="postgres://test:test@localhost:5432/clever_better_test"
   export ML_SERVICE_URL="http://localhost:8001"
   export REDIS_URL="redis://localhost:6379/1"
   ```

### CI/CD Environment

Tests run automatically on:
- Pull requests (unit + integration)
- Main branch commits (all tests)
- Release tags (full test suite + smoke tests)

**GitHub Actions Workflow**:
```yaml
- name: Run Unit Tests
  run: make test-unit

- name: Run Integration Tests
  run: make test-integration
  
- name: Run E2E Tests
  run: make test-e2e
```

## Test Coverage

### Coverage Goals

- **Unit Tests**: 80%+ coverage
- **Integration Tests**: Critical paths covered
- **E2E Tests**: All major user workflows

### Measuring Coverage

```bash
# Go coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Python coverage
pytest --cov=app --cov-report=html
```

### Coverage Reports

Reports are automatically uploaded to Codecov on CI runs.

## Test Helpers

### Database Helpers

Located in `test/helpers/helpers.go`:

```go
// Setup test database
db := helpers.SetupTestDB(t)
defer helpers.TeardownTestDB(t, db)

// Load fixtures
helpers.LoadFixtures(t, db, "races.json")
```

### Mock Clients

```go
// Mock Betfair API
mockBetfair := helpers.NewMockBetfairClient()
mockBetfair.On("GetMarketData", mock.Anything).Return(marketData, nil)

// Mock ML service
mockML := helpers.NewMockMLClient()
mockML.On("Predict", mock.Anything).Return(prediction, nil)
```

## Writing New Tests

### Unit Test Template

```go
func TestMyComponent_MyMethod(t *testing.T) {
    // Arrange
    component := NewMyComponent(mockDependency)
    input := "test input"
    
    // Act
    result, err := component.MyMethod(input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

### Integration Test Template

```go
//go:build integration

func TestDatabaseIntegration(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)
    
    // Test logic
}
```

### E2E Test Template

```go
//go:build e2e

func TestCompleteWorkflow(t *testing.T) {
    // Setup test environment
    ctx := setupE2EEnvironment(t)
    defer teardownE2EEnvironment(t, ctx)
    
    // Execute workflow
    // Verify outcomes
}
```

## Best Practices

### 1. Test Isolation

- Each test should be independent
- Use `t.Parallel()` for concurrent tests
- Clean up resources in `defer` statements

### 2. Descriptive Names

```go
// Good
func TestStrategyEvaluator_CalculateScore_WithMissingMetrics_ReturnsError(t *testing.T)

// Bad
func TestScore(t *testing.T)
```

### 3. AAA Pattern

- **Arrange**: Set up test data and mocks
- **Act**: Execute the function under test
- **Assert**: Verify outcomes

### 4. Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct{
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"invalid input", "invalid", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 5. Mocking External Dependencies

- Use interfaces for dependencies
- Mock external APIs and services
- Use real databases only for integration tests

## Troubleshooting

### Common Issues

**Test Timeouts**:
```bash
# Increase timeout for long-running tests
go test -timeout 30m ./test/e2e/...
```

**Database Connection Issues**:
```bash
# Verify database is running
docker-compose ps

# Check connection
psql -h localhost -U test -d clever_better_test
```

**Port Conflicts**:
```bash
# Find process using port
lsof -i :5432

# Stop conflicting service
docker-compose down
```

**Flaky Tests**:
- Add retry logic for network operations
- Increase timeouts for async operations
- Use `t.Parallel()` carefully with shared resources

### Debug Mode

```bash
# Verbose output
go test -v ./...

# Print test names only
go test -v ./... | grep -E "^(PASS|FAIL)"

# Run specific test
go test -v -run TestMySpecificTest ./internal/service/
```

## Continuous Integration

### Pre-commit Checks

Install pre-commit hooks:
```bash
# Run tests before commit
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
make test-unit
if [ $? -ne 0 ]; then
    echo "Unit tests failed. Commit aborted."
    exit 1
fi
EOF

chmod +x .git/hooks/pre-commit
```

### PR Requirements

Before merging:
- ✅ All unit tests pass
- ✅ Integration tests pass
- ✅ Coverage >= 80%
- ✅ No linter warnings
- ✅ E2E tests pass (for main branch)

## Performance Testing

### Load Testing

```bash
# Use vegeta for load testing
echo "GET http://localhost:8080/api/predictions" | \
  vegeta attack -duration=60s -rate=100 | \
  vegeta report
```

### Benchmarking

```go
func BenchmarkStrategyEvaluation(b *testing.B) {
    evaluator := NewStrategyEvaluator(repo)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        evaluator.Evaluate(strategy)
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./internal/service/
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Pytest Documentation](https://docs.pytest.org/)
- [Docker Compose for Testing](https://docs.docker.com/compose/)

## Maintenance

This guide should be reviewed and updated:
- When adding new test types
- After major architectural changes
- Quarterly as part of documentation review

Last Updated: 2026-02-15
