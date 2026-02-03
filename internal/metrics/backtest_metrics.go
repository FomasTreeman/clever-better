// Package metrics defines backtesting-specific metrics.
package metrics

import "github.com/prometheus/client_golang/prometheus"

// Backtest counter vectors
var (
	BacktestRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "backtest_runs_total",
		Help:      "Total number of backtest runs by method and status",
	}, []string{"method", "status"})
)

// Backtest histogram vectors
var (
	BacktestCompositeScore = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "clever_better",
		Name:      "backtest_composite_score",
		Help:      "Composite scores from backtest runs by strategy and method",
		Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	}, []string{"strategy_id", "method"})
)

// Backtest gauge vectors
var (
	BacktestAggregatedScore = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "backtest_aggregated_score",
		Help:      "Aggregated composite score for each strategy across all backtest methods",
	}, []string{"strategy_id"})
)

// RecordBacktestRun records a backtest run event.
// method should be one of: "historical_replay", "monte_carlo", "walk_forward"
// status should be one of: "success", "failure", "timeout"
func RecordBacktestRun(method, status string) {
	BacktestRunsTotal.WithLabelValues(method, status).Inc()
}

// RecordCompositeScore records a composite score from a backtest run.
func RecordCompositeScore(strategyID, method string, score float64) {
	BacktestCompositeScore.WithLabelValues(strategyID, method).Observe(score)
}

// UpdateAggregatedScore updates the aggregated composite score for a strategy.
func UpdateAggregatedScore(strategyID string, score float64) {
	BacktestAggregatedScore.WithLabelValues(strategyID).Set(score)
}
