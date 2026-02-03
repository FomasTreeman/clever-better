// Package metrics provides centralized Prometheus metrics registry for the trading bot.
package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Global registry instance
var (
	registry *prometheus.Registry
	once     sync.Once
)

// Counter metrics
var (
	BetsPlacedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "bets_placed_total",
		Help:      "Total number of bets placed",
	})
	BetsMatchedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "bets_matched_total",
		Help:      "Total number of bets matched",
	})
	BetsSettledTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "bets_settled_total",
		Help:      "Total number of bets settled",
	})
	StrategyEvaluationsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "strategy_evaluations_total",
		Help:      "Total number of strategy evaluations",
	})
	StrategySignalsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "strategy_signals_total",
		Help:      "Total number of strategy signals generated",
	})
	CircuitBreakerTripsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "circuit_breaker_trips_total",
		Help:      "Total number of circuit breaker trips",
	})
)

// Gauge metrics
var (
	ActiveStrategies = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "active_strategies",
		Help:      "Number of currently active strategies",
	})
	CurrentBankroll = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "current_bankroll",
		Help:      "Current bankroll in currency units",
	})
	TotalExposure = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "total_exposure",
		Help:      "Total bet exposure across all strategies",
	})
	DailyPnL = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "daily_pnl",
		Help:      "Daily profit and loss",
	})
	StrategyCompositeScore = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "strategy_composite_score",
		Help:      "Composite score for each strategy",
	}, []string{"strategy_id", "strategy_name"})
)

// Histogram metrics
var (
	BetPlacementLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "clever_better",
		Name:      "bet_placement_latency_seconds",
		Help:      "Latency of bet placement operations in seconds",
		Buckets:   prometheus.DefBuckets,
	})
	StrategyEvaluationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "clever_better",
		Name:      "strategy_evaluation_duration_seconds",
		Help:      "Duration of strategy evaluation in seconds",
		Buckets:   prometheus.DefBuckets,
	})
	BacktestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "clever_better",
		Name:      "backtest_duration_seconds",
		Help:      "Duration of backtest runs in seconds",
		Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800},
	})
)

// InitRegistry initializes the global Prometheus registry.
func InitRegistry() *prometheus.Registry {
	once.Do(func() {
		registry = prometheus.NewRegistry()

		// Register counter metrics
		registry.MustRegister(BetsPlacedTotal)
		registry.MustRegister(BetsMatchedTotal)
		registry.MustRegister(BetsSettledTotal)
		registry.MustRegister(StrategyEvaluationsTotal)
		registry.MustRegister(StrategySignalsTotal)
		registry.MustRegister(CircuitBreakerTripsTotal)

		// Register gauge metrics
		registry.MustRegister(ActiveStrategies)
		registry.MustRegister(CurrentBankroll)
		registry.MustRegister(TotalExposure)
		registry.MustRegister(DailyPnL)
		registry.MustRegister(StrategyCompositeScore)

		// Register histogram metrics
		registry.MustRegister(BetPlacementLatency)
		registry.MustRegister(StrategyEvaluationDuration)
		registry.MustRegister(BacktestDuration)

		// Register strategy metrics
		registry.MustRegister(StrategyDecisionsTotal)
		registry.MustRegister(StrategyConfidenceScore)
		registry.MustRegister(StrategyActiveBets)
		registry.MustRegister(MLStrategyRecommendationsTotal)

		// Register backtest metrics
		registry.MustRegister(BacktestRunsTotal)
		registry.MustRegister(BacktestCompositeScore)
		registry.MustRegister(BacktestAggregatedScore)
	})
	return registry
}

// GetRegistry returns the global Prometheus registry.
func GetRegistry() *prometheus.Registry {
	if registry == nil {
		return InitRegistry()
	}
	return registry
}

// Handler returns the Prometheus HTTP handler.
func Handler() http.Handler {
	return promhttp.HandlerFor(GetRegistry(), promhttp.HandlerOpts{})
}

// RecordBetPlaced records a bet placement event.
func RecordBetPlaced() {
	BetsPlacedTotal.Inc()
}

// RecordBetMatched records a bet match event.
func RecordBetMatched() {
	BetsMatchedTotal.Inc()
}

// RecordBetSettled records a bet settlement event.
func RecordBetSettled() {
	BetsSettledTotal.Inc()
}

// RecordStrategyEvaluation records a strategy evaluation event.
func RecordStrategyEvaluation(durationSeconds float64) {
	StrategyEvaluationsTotal.Inc()
	StrategyEvaluationDuration.Observe(durationSeconds)
}

// RecordStrategySignal records a strategy signal event.
func RecordStrategySignal() {
	StrategySignalsTotal.Inc()
}

// RecordCircuitBreakerTrip records a circuit breaker trip event.
func RecordCircuitBreakerTrip() {
	CircuitBreakerTripsTotal.Inc()
}

// UpdateBankroll updates the current bankroll gauge.
func UpdateBankroll(amount float64) {
	CurrentBankroll.Set(amount)
}

// UpdateExposure updates the total exposure gauge.
func UpdateExposure(amount float64) {
	TotalExposure.Set(amount)
}

// UpdateActivities updates the active strategies gauge.
func UpdateActiveStrategies(count float64) {
	ActiveStrategies.Set(count)
}

// UpdateDailyPnL updates the daily P&L gauge.
func UpdateDailyPnL(pnl float64) {
	DailyPnL.Set(pnl)
}

// RecordBetPlacementLatency records bet placement latency.
func RecordBetPlacementLatency(durationSeconds float64) {
	BetPlacementLatency.Observe(durationSeconds)
}

// RecordBacktestDuration records backtest duration.
func RecordBacktestDuration(durationSeconds float64) {
	BacktestDuration.Observe(durationSeconds)
}
