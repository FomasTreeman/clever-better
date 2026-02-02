# Backtesting Implementation

## Overview

This document describes the implementation of the backtesting framework, including engine orchestration, strategy interfaces, simulation modes, and ML export outputs. The design focuses on temporal safety, transaction cost modeling, and robust metrics for strategy discovery.

## Architecture

### Core Components

- **Strategy Interface**: Pluggable strategy definitions with consistent inputs and outputs.
- **Backtest Engine**: Orchestrates historical replay, bet execution simulation, and settlement.
- **Metrics Calculator**: Computes risk-adjusted returns and trading statistics.
- **Monte Carlo**: Probabilistic simulation of outcomes from historical signals.
- **Walk-Forward**: Out-of-sample testing with rolling windows.
- **Aggregator**: Weighted combination of multiple method results.
- **ML Export**: Structured JSON output for ML feature generation.

### Temporal Safety

- Strategy evaluation only uses odds snapshots at or before `CurrentTime`.
- Backtest engine queries odds with `end <= race.ScheduledStart`.
- Walk-forward windows isolate training, validation, and test slices.

## Usage Examples

### Historical Replay

```bash
./bin/backtest --mode historical --strategy simple_value
```

### Monte Carlo

```bash
./bin/backtest --mode monte-carlo --strategy simple_value
```

### Walk-Forward

```bash
./bin/backtest --mode walk-forward --strategy simple_value
```

### Run All Methods

```bash
./bin/backtest --mode all --strategy simple_value --ml-export --output ./output/backtest_results.json
```

## ML Export Schema

The `MLExport` struct includes:

- Strategy metadata and parameters
- Backtest summary and equity curve
- Metrics from historical replay, Monte Carlo, and walk-forward
- Bet-by-bet history with features
- Risk profile (VaR, drawdown)

## Strategy Implementation Guide

A strategy must implement:

- `Name()`
- `Evaluate(ctx, race, runners, odds)`
- `ShouldBet(signal)`
- `CalculateStake(signal, bankroll)`
- `GetParameters()`

Use `BaseStrategy` for common validations and expected value calculation.

## Troubleshooting

- **No bets generated**: Check `MinEdgeThreshold` and liquidity thresholds.
- **Lookahead violations**: Ensure odds timestamps are not after evaluation time.
- **Low Sharpe ratio**: Increase sample size or adjust risk parameters.
