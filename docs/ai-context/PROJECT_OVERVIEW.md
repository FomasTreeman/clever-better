# Project Overview for AI

This document provides a high-level summary of the Clever Better project optimized for AI-assisted development.

## Quick Reference

| Aspect | Details |
|--------|---------|
| **Project Name** | Clever Better |
| **Domain** | Greyhound racing betting automation |
| **Primary Language** | Go 1.22+ |
| **ML Language** | Python 3.11+ |
| **Database** | TimescaleDB (PostgreSQL extension) |
| **Cloud Platform** | AWS (ECS Fargate, RDS, VPC) |
| **External API** | Betfair Exchange API |

## Project Purpose

Clever Better is an automated betting system that:
1. Collects greyhound racing data from multiple sources
2. Uses machine learning to identify profitable betting opportunities
3. Validates strategies through rigorous backtesting
4. Executes trades automatically on the Betfair Exchange

## Key Objectives

1. **Profitability**: Achieve positive expected value through data-driven strategy
2. **Risk Management**: Never risk more than predefined limits
3. **Reliability**: Operate autonomously with minimal intervention
4. **Transparency**: Full audit trail of all decisions and trades

## System Boundaries

### What the System Does
- Ingest race data and historical odds
- Train ML models to predict race outcomes
- Backtest strategies against historical data
- Execute bets on Betfair Exchange
- Monitor and report on performance

### What the System Does NOT Do
- Make deposits or withdrawals
- Manage user accounts
- Provide a user interface (CLI only)
- Guarantee profits (this is gambling)

## Critical Business Rules

### Risk Limits (Non-Negotiable)

```yaml
max_stake_per_bet: 10.00      # Maximum stake on any single bet
max_daily_loss: 100.00        # Stop trading if daily loss exceeds
max_exposure: 500.00          # Maximum total exposure at any time
min_confidence: 0.65          # Don't bet below this confidence
min_expected_value: 0.02      # Require 2% edge minimum
```

### Betting Constraints

1. **Back bets only** (initially) - no lay betting
2. **WIN and PLACE markets only** - no exotic bets
3. **Pre-race only** - no in-play betting
4. **Greyhounds only** - UK and Irish tracks

### Data Integrity

1. Never modify historical data once recorded
2. All predictions must be logged before race start
3. All trades must be reconciled with Betfair records

## External Dependencies

### Betfair Exchange API
- **Purpose**: Place bets, stream market data
- **Authentication**: Certificate-based (SSL client cert)
- **Rate Limits**: Varies by operation (typically 5-20 req/sec)
- **Documentation**: https://developer.betfair.com/

### Data Sources
- **Betfair Historical Data**: Official historical odds/results
- **Racing Data Providers**: Form data, track conditions (TBD)

### AWS Services
- **ECS Fargate**: Container hosting
- **RDS**: TimescaleDB hosting
- **Secrets Manager**: Credential storage
- **CloudWatch**: Logging and monitoring
- **S3**: Model artifact storage

## Technology Rationale

### Why Go for Backend?
- **Performance**: Sub-millisecond latency for trading decisions
- **Concurrency**: Handle multiple market streams efficiently
- **Reliability**: Strong typing, compile-time checks
- **Deployment**: Single binary, no runtime dependencies

### Why Python for ML?
- **Ecosystem**: Best-in-class ML libraries (TensorFlow, PyTorch, scikit-learn)
- **Productivity**: Rapid experimentation and prototyping
- **Visualization**: Matplotlib, pandas for analysis

### Why TimescaleDB?
- **Time-series optimized**: Built for odds snapshots and trading data
- **PostgreSQL compatible**: Familiar SQL, rich tooling
- **Compression**: Efficient storage for large datasets
- **Continuous aggregates**: Pre-computed analytics

### Why ECS Fargate?
- **Serverless containers**: No EC2 management
- **Cost-effective**: Pay only for running tasks
- **Scalability**: Easy horizontal scaling
- **Integration**: Native AWS service integration

## Key Workflows

### 1. Data Ingestion (Daily)
```
External Sources → Fetch → Validate → Transform → Store (TimescaleDB)
```

### 2. Model Training (Weekly)
```
Historical Data → Feature Engineering → Train → Validate → Deploy (S3)
```

### 3. Live Trading (Continuous)
```
Market Stream → Analyze → Predict → Decide → Execute → Log
```

### 4. Backtesting (On-demand)
```
Historical Data → Replay → Simulate → Measure → Report
```

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| ROI | > 2% monthly | P&L / Capital deployed |
| Sharpe Ratio | > 1.0 | Risk-adjusted returns |
| Win Rate | > 35% | Winning bets / Total bets |
| Max Drawdown | < 25% | Peak-to-trough decline |
| Uptime | > 99.5% | Trading hours available |

## Common Tasks for AI Assistance

When working on this codebase, you may be asked to:

1. **Implement features**: Add new functionality to Go services or Python ML
2. **Write tests**: Unit tests, integration tests, backtest scenarios
3. **Optimize queries**: Improve database query performance
4. **Debug issues**: Investigate errors in trading or predictions
5. **Update documentation**: Keep docs in sync with code changes
6. **Review strategies**: Analyze backtest results and suggest improvements
