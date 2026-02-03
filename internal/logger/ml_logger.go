// Package logger provides ML-specific logging.
package logger

import (
	"github.com/sirupsen/logrus"
)

// MLLogger provides dedicated logging for ML operations.
type MLLogger struct {
	*logrus.Entry
}

// NewMLLogger creates a new ML logger.
func NewMLLogger(baseLogger *logrus.Logger) *MLLogger {
	return &MLLogger{
		Entry: baseLogger.WithField("component", "ml"),
	}
}

// LogMLPredictionRequest logs an ML prediction request.
func (ml *MLLogger) LogMLPredictionRequest(modelType string, featuresCount int, cacheHit bool, latencyMs float64) {
	ml.WithFields(logrus.Fields{
		"model_type":    modelType,
		"features_count": featuresCount,
		"cache_hit":     cacheHit,
		"latency_ms":    latencyMs,
	}).Info("ML prediction request completed")
}

// LogStrategyGeneration logs ML strategy generation.
func (ml *MLLogger) LogStrategyGeneration(constraints map[string]interface{}, candidatesGenerated int, topCompositeScore float64, aggregatedFeaturesCount int) {
	ml.WithFields(logrus.Fields{
		"constraints":                constraints,
		"candidates_generated":       candidatesGenerated,
		"top_composite_score":        topCompositeScore,
		"aggregated_features_count":  aggregatedFeaturesCount,
	}).Info("ML strategy generation completed")
}

// LogBacktestFeedback logs backtest-to-ML feedback submission.
func (ml *MLLogger) LogBacktestFeedback(backtestMethod string, metricsSent int, mlFeaturesExtracted int) {
	ml.WithFields(logrus.Fields{
		"backtest_method":         backtestMethod,
		"metrics_sent":            metricsSent,
		"ml_features_extracted":   mlFeaturesExtracted,
	}).Info("Backtest feedback submitted to ML service")
}

// LogModelTraining logs model training events.
func (ml *MLLogger) LogModelTraining(modelName string, trainingDuration float64, metrics map[string]float64, hyperparameters map[string]interface{}) {
	ml.WithFields(logrus.Fields{
		"model_name":        modelName,
		"training_duration": trainingDuration,
		"metrics":           metrics,
		"hyperparameters":   hyperparameters,
	}).Info("Model training completed")
}

// LogStrategyRankingUpdate logs strategy ranking updates.
func (ml *MLLogger) LogStrategyRankingUpdate(totalStrategies int, topStrategyID string, rankingCriteria string) {
	ml.WithFields(logrus.Fields{
		"total_strategies": totalStrategies,
		"top_strategy_id":  topStrategyID,
		"ranking_criteria": rankingCriteria,
	}).Info("Strategy ranking updated")
}

// LogMLPredictionError logs ML prediction errors.
func (ml *MLLogger) LogMLPredictionError(modelType string, errorReason string) {
	ml.WithFields(logrus.Fields{
		"model_type":   modelType,
		"error_reason": errorReason,
	}).Error("ML prediction failed")
}
