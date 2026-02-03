// Package ml provides Prometheus metrics for ML operations.
package ml

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MLPredictionsTotal tracks total ML predictions
	MLPredictionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_predictions_total",
			Help: "Total number of ML predictions made",
		},
		[]string{"model_type", "cache_hit"},
	)

	// MLPredictionLatency tracks ML prediction latency
	MLPredictionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ml_prediction_latency_seconds",
			Help:    "ML prediction latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model_type"},
	)

	// MLFeedbackSubmittedTotal tracks feedback submissions
	MLFeedbackSubmittedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ml_feedback_submitted_total",
			Help: "Total number of backtest results submitted as feedback",
		},
	)

	// MLStrategyGenerationTotal tracks strategy generation attempts
	MLStrategyGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_strategy_generation_total",
			Help: "Total number of strategy generation attempts",
		},
		[]string{"status"}, // success, failure
	)

	// MLCacheHitRatio tracks cache hit ratio
	MLCacheHitRatio = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ml_cache_hit_ratio",
			Help: "ML prediction cache hit ratio",
		},
	)

	// MLGRPCErrorsTotal tracks gRPC errors
	MLGRPCErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_grpc_errors_total",
			Help: "Total number of gRPC errors",
		},
		[]string{"method", "error_type"},
	)

	// MLTrainingJobsTotal tracks training jobs
	MLTrainingJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_training_jobs_total",
			Help: "Total number of ML training jobs",
		},
		[]string{"model_type", "status"},
	)
)
