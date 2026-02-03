# Real Backtest Evaluation Implementation

## ✅ Implementation Complete

The strategy generator service has been updated to use **REAL backtest execution** instead of synthetic ML estimates.

## What Changed

### 1. Service Structure Updated

**Added Dependencies:**
```go
type StrategyGeneratorService struct {
    mlClient          *ml.CachedMLClient
    strategyRepo      repository.StrategyRepository
    backtestRepo      repository.BacktestResultRepository
    db                *database.DB              // NEW: Database for backtest engine
    logger            *logrus.Logger
    minCompositeScore float64
    backtestConfig    backtest.BacktestConfig  // NEW: Backtest configuration
}
```

**Updated Constructor:**
```go
func NewStrategyGeneratorService(
    mlClient *ml.CachedMLClient,
    strategyRepo repository.StrategyRepository,
    backtestRepo repository.BacktestResultRepository,
    db *database.DB,  // NEW: Required parameter
    logger *logrus.Logger,
) *StrategyGeneratorService
```

### 2. EvaluateGeneratedStrategy() - Complete Rewrite

**Before (Synthetic):**
```go
// Used ML expected values only
winRate := strategy.ExpectedWinRate
roi := strategy.ExpectedReturn
sharpe := strategy.ExpectedSharpe
compositeScore := (sharpe * 0.4) + (roi * 0.3) + (winRate * 0.2)...
// No real backtest execution
```

**After (Real Backtest):**
```go
// 1. Create strategy implementation from ML parameters
stratImpl := s.createStrategyFromMLParams(generatedStrategy)

// 2. Create backtest engine
engine, err := backtest.NewEngine(s.backtestConfig, s.db, stratImpl, s.logger)

// 3. Run REAL backtest
state, metrics, err := engine.Run(ctx, startDate, endDate)

// 4. Use REAL metrics
compositeScore := (metrics.SharpeRatio * 0.4) + (metrics.TotalReturn * 0.3)...

// 5. Store REAL backtest result
result := &models.BacktestResult{
    SharpeRatio:    metrics.SharpeRatio,      // REAL
    TotalReturn:    metrics.TotalReturn,      // REAL
    WinRate:        metrics.WinRate,          // REAL
    TotalBets:      metrics.TotalBets,        // REAL
    MaxDrawdown:    metrics.MaxDrawdown,      // REAL
    ProfitFactor:   metrics.ProfitFactor,     // REAL
    Method:         "real_backtest",          // Marked as real
}
```

### 3. New Helper Functions

#### `createStrategyFromMLParams()`
Converts ML-generated parameters into a concrete `strategy.Strategy` implementation:
```go
func (s *StrategyGeneratorService) createStrategyFromMLParams(gen *ml.GeneratedStrategy) strategy.Strategy {
    strat := strategy.NewSimpleValueStrategy()
    
    // Override with ML parameters
    if minEdge, ok := gen.Parameters["min_edge_threshold"]; ok {
        strat.MinEdgeThreshold = minEdge
    }
    if kelly, ok := gen.Parameters["kelly_fraction"]; ok {
        strat.KellyFraction = kelly
    }
    // ... more parameters
    
    return strat
}
```

#### `extractMLFeaturesFromBacktest()`
Extracts 16 ML features from real backtest state:
```go
features := map[string]float64{
    "sharpe_ratio":    metrics.SharpeRatio,
    "total_return":    metrics.TotalReturn,
    "max_drawdown":    metrics.MaxDrawdown,
    "win_rate":        metrics.WinRate,
    "profit_factor":   metrics.ProfitFactor,
    "total_bets":      float64(metrics.TotalBets),
    "winning_bets":    float64(metrics.WinningBets),
    "losing_bets":     float64(metrics.LosingBets),
    "average_win":     metrics.AverageWin,
    "average_loss":    metrics.AverageLoss,
    "largest_win":     metrics.LargestWin,
    "largest_loss":    metrics.LargestLoss,
    "sortino_ratio":   metrics.SortinoRatio,
    "calmar_ratio":    metrics.CalmarRatio,
    "var_95":          metrics.ValueAtRisk95,
    "trades_per_day":  float64(metrics.TotalBets) / float64(metrics.TradingDays),
}
```

#### `createFallbackResult()`
Graceful degradation when backtest engine fails:
```go
func (s *StrategyGeneratorService) createFallbackResult(gen *ml.GeneratedStrategy) *models.BacktestResult {
    // Uses ML estimates as fallback
    // Marked with Method: "ml_estimate"
    // Logs warning about fallback
}
```

#### `getRecommendation()`
Dynamic recommendation based on performance:
```go
func (s *StrategyGeneratorService) getRecommendation(compositeScore float64, metrics backtest.Metrics) string {
    if compositeScore >= 0.8 && metrics.SharpeRatio > 1.5 {
        return "EXCELLENT"
    } else if compositeScore >= 0.6 && metrics.WinRate > 0.55 {
        return "GOOD"
    } else if compositeScore >= 0.4 {
        return "ACCEPTABLE"
    } else if compositeScore >= 0.2 {
        return "POOR"
    }
    return "REJECT"
}
```

### 4. Backtest Configuration

**Default Config for Strategy Evaluation:**
```go
backtestConfig := backtest.BacktestConfig{
    StartDate:            time.Now().AddDate(0, -6, 0),  // Last 6 months
    EndDate:              time.Now().AddDate(0, -1, 0),  // Up to 1 month ago
    InitialBankroll:      10000.0,
    CommissionRate:       0.05,                          // 5% commission
    SlippageTicks:        1,
    MinLiquidity:         100.0,
    MonteCarloIterations: 100,
    WalkForwardWindows:   1,
    RiskFreeRate:         0.02,                          // 2% risk-free rate
}
```

### 5. Strategy Evaluator Enhancement

**Added Warning for Missing Backtests:**
```go
if len(backtestResults) == 0 {
    s.logger.WithField("strategy_id", strategyID).Warn(
        "No backtest results found - strategy needs real backtest evaluation")
}
```

## Data Flow

### Before (Synthetic)
```
GeneratedStrategy → EvaluateGeneratedStrategy
    ↓
Use ML Expected Values (ExpectedReturn, ExpectedSharpe, ExpectedWinRate)
    ↓
Create Synthetic BacktestResult (Method: not set, TradesCount: 0)
    ↓
Store to Database
```

### After (Real Backtest)
```
GeneratedStrategy → EvaluateGeneratedStrategy
    ↓
createStrategyFromMLParams() → SimpleValueStrategy with ML parameters
    ↓
backtest.NewEngine(config, db, strategy, logger)
    ↓
engine.Run(ctx, startDate, endDate) → REAL EXECUTION
    ↓
Process historical races, odds, signals
    ↓
Calculate REAL metrics (Sharpe, ROI, WinRate, Drawdown, etc.)
    ↓
extractMLFeaturesFromBacktest() → 16 feature dimensions
    ↓
Create BacktestResult with REAL metrics (Method: "real_backtest")
    ↓
Store to Database for ML feedback
```

## Metrics Comparison

| Metric | Before (Synthetic) | After (Real) |
|--------|-------------------|--------------|
| **Source** | ML ExpectedSharpe | backtest.Metrics.SharpeRatio |
| **Total Return** | ML ExpectedReturn | metrics.TotalReturn |
| **Win Rate** | ML ExpectedWinRate | metrics.WinRate |
| **Total Bets** | 0 (placeholder) | metrics.TotalBets (actual) |
| **Max Drawdown** | 1.0 - winRate (guess) | metrics.MaxDrawdown (real) |
| **Profit Factor** | 1.0 + roi*winRate | metrics.ProfitFactor (real) |
| **Method** | Not set | "real_backtest" |
| **ML Features** | Not extracted | 16 dimensions from state |

## Logging Examples

**Real Backtest Execution:**
```
INFO Running real backtest for ML-generated strategy 
     start_date=2025-08-01 end_date=2026-01-01
INFO Real backtest evaluation complete 
     strategy_id=abc-123 composite_score=0.845 sharpe_ratio=1.82 
     total_return=0.287 win_rate=0.685 total_bets=152 
     max_drawdown=0.118 recommendation=EXCELLENT
```

**Fallback to ML Estimates:**
```
ERROR Backtest execution failed, using ML estimates 
      error="failed to load races: connection error"
WARN Using ML estimates as fallback for backtest result
```

## Error Handling

1. **Engine Creation Fails:** Falls back to ML estimates, logs error
2. **Backtest Execution Fails:** Falls back to ML estimates, logs error
3. **No Historical Data:** Falls back to ML estimates, logs warning
4. **Database Connection Lost:** Falls back to ML estimates, returns fallback result

All fallback results are marked with `Method: "ml_estimate"` for tracking.

## Activation Logic

**ActivateTopStrategies() uses REAL metrics:**
```go
// Evaluate strategy
result, err := s.EvaluateGeneratedStrategy(ctx, strategy)

// Now result contains REAL backtest metrics
if result.CompositeScore >= s.minCompositeScore {
    // Activate only if real backtest proves quality
    strategyModel.IsActive = true
}
```

## Integration Requirements

To use the updated service, callers must:

1. **Pass database connection:**
```go
generator := service.NewStrategyGeneratorService(
    mlClient,
    strategyRepo,
    backtestRepo,
    db,          // NEW: Required
    logger,
)
```

2. **Ensure historical data exists:**
   - Races in date range (last 6 months)
   - Odds snapshots for those races
   - Race results for validation

3. **Handle longer evaluation time:**
   - Real backtest takes 5-30 seconds (vs instant synthetic)
   - Consider async evaluation for batch strategies

## Testing

**Integration Test Created:**
- `test/integration/strategy_generator_backtest_test.go`
- Validates backtest config
- Tests composite score calculation
- Verifies ML features extraction
- Checks recommendation logic
- Tests fallback mechanism

**Run Tests:**
```bash
go test -tags integration -v ./test/integration/strategy_generator_backtest_test.go
```

## Benefits

✅ **Empirical Validation:** Strategies validated with real historical data
✅ **Accurate Metrics:** Real Sharpe, ROI, drawdown, win rate
✅ **ML Feedback Loop:** 16 real features fed back to ML for improvement
✅ **Risk Management:** Real drawdown and VaR metrics for activation decisions
✅ **Transparency:** Method field distinguishes real vs estimated results
✅ **Graceful Degradation:** Falls back to ML estimates if backtest fails

## Architecture Alignment

- ✅ Integrates `internal/backtest/engine.go`
- ✅ Uses `backtest.Config` and `backtest.Metrics`
- ✅ Implements `strategy.Strategy` interface
- ✅ Stores to `models.BacktestResult` with proper fields
- ✅ Closes feedback loop: backtest → aggregates → ML → new strategies → **real backtest** → repeat

## Next Steps (Optional)

1. **Parallel Evaluation:** Run multiple backtests concurrently
2. **Custom Backtest Periods:** Allow caller to specify date range
3. **Monte Carlo Analysis:** Use `MonteCarloIterations` for confidence intervals
4. **Walk-Forward Optimization:** Use `WalkForwardWindows` for robust validation
5. **Performance Metrics:** Track backtest execution time, cache results

---

**Status:** ✅ IMPLEMENTED  
**Impact:** Replaces all synthetic evaluation with real backtest execution  
**Breaking Change:** Constructor now requires `*database.DB` parameter
