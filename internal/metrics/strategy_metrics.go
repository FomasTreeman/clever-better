// Package metrics defines strategy-specific metrics.
package metrics

import "github.com/prometheus/client_golang/prometheus"

// Strategy-specific counter vectors
var (
	StrategyDecisionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "strategy_decisions_total",
		Help:      "Total number of strategy decisions by type and outcome",
	}, []string{"strategy_id", "strategy_name", "decision_type", "outcome"})

	MLStrategyRecommendationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "clever_better",
		Name:      "ml_strategy_recommendations_total",
		Help:      "Total number of ML strategy recommendations by type",
	}, []string{"recommendation", "confidence_bucket"})
)

// Strategy-specific histogram vectors
var (
	StrategyConfidenceScore = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "clever_better",
		Name:      "strategy_confidence_score",
		Help:      "Confidence scores for strategy decisions",
		Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	}, []string{"strategy_id", "strategy_name"})
)

// Strategy-specific gauge vectors
var (
	StrategyActiveBets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "clever_better",
		Name:      "strategy_active_bets",
		Help:      "Number of active bets for each strategy",
	}, []string{"strategy_id", "strategy_name"})
)

// RecordStrategyDecision records a strategy decision.
func RecordStrategyDecision(strategyID, strategyName, decisionType, outcome string) {
	StrategyDecisionsTotal.WithLabelValues(strategyID, strategyName, decisionType, outcome).Inc()
}

// RecordStrategyConfidence records a strategy confidence score.
func RecordStrategyConfidence(strategyID, strategyName string, score float64) {
	StrategyConfidenceScore.WithLabelValues(strategyID, strategyName).Observe(score)
}

// UpdateStrategyActiveBets updates the active bets count for a strategy.
func UpdateStrategyActiveBets(strategyID, strategyName string, count float64) {
	StrategyActiveBets.WithLabelValues(strategyID, strategyName).Set(count)
}

// RecordMLRecommendation records an ML strategy recommendation.
func RecordMLRecommendation(recommendation, confidenceBucket string) {
	MLStrategyRecommendationsTotal.WithLabelValues(recommendation, confidenceBucket).Inc()
}
