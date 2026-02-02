# Database Implementation Verification Checklist

## File Structure Verification

### Migrations (10 files in `migrations/`)
- [ ] `000001_create_core_tables.up.sql` - Races and runners tables created
- [ ] `000001_create_core_tables.down.sql` - Rollback script present
- [ ] `000002_create_hypertables.up.sql` - odds_snapshots hypertable created
- [ ] `000002_create_hypertables.down.sql` - Rollback script present
- [ ] `000003_create_trading_tables.up.sql` - Strategies and bets tables created
- [ ] `000003_create_trading_tables.down.sql` - Rollback script present
- [ ] `000004_create_ml_tables.up.sql` - Models and predictions tables created
- [ ] `000004_create_ml_tables.down.sql` - Rollback script present
- [ ] `000005_create_strategy_performance.up.sql` - Performance hypertable created
- [ ] `000005_create_strategy_performance.down.sql` - Rollback script present

### Models (9 files in `internal/models/`)
- [ ] `race.go` - Race model with validation tags
- [ ] `runner.go` - Runner model with helper methods
- [ ] `odds.go` - OddsSnapshot model with financial calculations
- [ ] `bet.go` - Bet model with enums and methods
- [ ] `strategy.go` - Strategy model with JSONB support
- [ ] `model.go` - ML Model struct
- [ ] `prediction.go` - Prediction model
- [ ] `strategy_performance.go` - Performance aggregation model
- [ ] `errors.go` - Custom error definitions

### Database Layer (3 files in `internal/database/`)
- [ ] `postgres.go` - Connection pool manager
- [ ] `init.go` - Database initialization
- [ ] `testing.go` - Test utilities
- [ ] `init_example.go` - Example integration code

### Repository Layer (9 files in `internal/repository/`)
- [ ] `interfaces.go` - All 8 repository interface definitions
- [ ] `race_repository.go` - Race repository implementation
- [ ] `runner_repository.go` - Runner repository implementation
- [ ] `odds_repository.go` - Odds repository with batch insert
- [ ] `bet_repository.go` - Bet repository implementation
- [ ] `strategy_repository.go` - Strategy repository implementation
- [ ] `model_repository.go` - ML model repository implementation
- [ ] `prediction_repository.go` - Prediction repository implementation
- [ ] `strategy_performance_repository.go` - Performance repository implementation
- [ ] `repository.go` - Repository factory
- [ ] `repository_test.go` - Integration test stubs

### Documentation (4 new files)
- [ ] `docs/DATABASE.md` - Complete schema and architecture documentation
- [ ] `docs/INTEGRATION_GUIDE.md` - Integration examples and patterns
- [ ] `docs/MAKEFILE_TARGETS.md` - Database target documentation
- [ ] `DATABASE_IMPLEMENTATION.md` - Implementation summary

### Updated Files
- [ ] `docs/DEVELOPMENT.md` - Database setup section added
- [ ] `config/config.yaml.example` - Database configuration examples
- [ ] `README.md` - Quick Start includes database setup
- [ ] `go.mod` - pgx and uuid dependencies added
- [ ] `Makefile` - Database targets added

## Dependency Verification

```bash
# Check go.mod includes these packages
grep "github.com/jackc/pgx/v5" go.mod
grep "github.com/google/uuid" go.mod
```

Expected output:
```
github.com/jackc/pgx/v5 v5.5.1
github.com/google/uuid v1.5.0
```

## Database Setup Verification

```bash
# Prerequisites installed
which psql                    # PostgreSQL client
which migrate                 # golang-migrate

# Create database
make db-setup

# Verify connection
make db-health-check

# Check migrations
make db-migrate-status

# View schema
psql -U postgres clever_better -c "\d"
```

Expected schema tables:
- [ ] `races`
- [ ] `runners`
- [ ] `odds_snapshots` (hypertable)
- [ ] `strategies`
- [ ] `bets` (hypertable)
- [ ] `models`
- [ ] `predictions` (hypertable)
- [ ] `strategy_performance` (hypertable)
- [ ] `strategy_performance_daily` (continuous aggregate)
- [ ] `schema_migrations` (migration tracking)

## Code Quality Verification

### Syntax & Compilation
```bash
# Check Go code compiles
go build ./cmd/bot
go build ./cmd/backtest
go build ./cmd/data-ingestion

# Check for import errors
go mod tidy
go mod verify
```

### Package Structure
- [ ] All imports use correct package paths (`github.com/yourusername/clever-better/...`)
- [ ] No circular dependencies
- [ ] Each file has proper package declaration
- [ ] Error types properly imported from models package

### Repository Patterns
- [ ] All repositories implement their interfaces
- [ ] All methods accept `ctx context.Context` as first parameter
- [ ] All methods return proper error types
- [ ] Batch operations use `pgx.CopyFromRows()`
- [ ] Parameterized queries prevent SQL injection

### Database Features
- [ ] Connection pooling configured in postgres.go
- [ ] Transaction support via WithTransaction()
- [ ] Health check implementation present
- [ ] Context timeout handling in all queries
- [ ] Proper cleanup in Close() method

## Integration Checklist

### Model Validation
- [ ] All models have validation tags
- [ ] Custom validators implemented (e.g., markets validation)
- [ ] Error messages are clear and actionable

### Configuration
- [ ] Database config section in config.yaml.example
- [ ] Environment variable expansion documented
- [ ] AWS Secrets Manager support documented

### Documentation
- [ ] Schema documentation complete (DATABASE.md)
- [ ] Integration examples provided (INTEGRATION_GUIDE.md)
- [ ] Makefile targets documented (MAKEFILE_TARGETS.md)
- [ ] Development guide updated with DB setup

## Testing Verification

```bash
# Run unit tests (existing tests should pass)
make test

# Check test utilities work
# (SetupTestDB and TeardownTestDB can be used in test files)
```

## Common Issues & Fixes

### Issue: `psql: command not found`
**Fix**: Install PostgreSQL
```bash
brew install postgresql@15        # macOS
apt install postgresql-15         # Ubuntu
```

### Issue: `migrate: command not found`
**Fix**: Install golang-migrate
```bash
brew install golang-migrate       # macOS
apt install golang-migrate        # Ubuntu
```

### Issue: Database connection refused
**Fix**: Ensure PostgreSQL is running
```bash
pg_ctl status                     # macOS
sudo systemctl status postgresql  # Ubuntu
```

### Issue: TimescaleDB extension not found
**Fix**: Install TimescaleDB extension
```bash
CREATE EXTENSION IF NOT EXISTS timescaledb;  # In psql
```

### Issue: Migration version mismatch
**Fix**: Check and potentially reset migrations
```bash
make db-migrate-status
make db-migrate-down
make db-reset
make db-migrate-up
```

## Performance Verification

```bash
# Check hypertable status
psql -U postgres clever_better -c \
  "SELECT * FROM timescaledb_information.hypertables;"

# Check continuous aggregates
psql -U postgres clever_better -c \
  "SELECT * FROM timescaledb_information.continuous_aggregates;"

# View table sizes
psql -U postgres clever_better -c \
  "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
```

## Next Steps After Verification

1. **Integration with Application Entry Points**:
   - [ ] Update `cmd/bot/main.go` to initialize database
   - [ ] Update `cmd/backtest/main.go` to load historical data
   - [ ] Update `cmd/data-ingestion/main.go` to insert data

2. **Service Layer Implementation**:
   - [ ] Create trading service using repositories
   - [ ] Create backtesting service using repositories
   - [ ] Create ML service integration

3. **Testing**:
   - [ ] Implement integration tests in repository_test.go
   - [ ] Create end-to-end test scenarios
   - [ ] Set up CI/CD database testing

4. **Deployment**:
   - [ ] Configure production database connection
   - [ ] Set up database backups
   - [ ] Configure monitoring and alerts

## Sign-Off

- [ ] All files created and verified
- [ ] Database setup successful
- [ ] Schema matches specification
- [ ] Repository pattern implemented
- [ ] Documentation complete
- [ ] Ready for application integration

**Completed Date**: ________________  
**Verified By**: ________________

---

For questions or issues, refer to:
- [DATABASE.md](docs/DATABASE.md) - Schema documentation
- [INTEGRATION_GUIDE.md](docs/INTEGRATION_GUIDE.md) - Integration examples
- [MAKEFILE_TARGETS.md](docs/MAKEFILE_TARGETS.md) - Database commands
- [DEVELOPMENT.md](docs/DEVELOPMENT.md) - Development setup
