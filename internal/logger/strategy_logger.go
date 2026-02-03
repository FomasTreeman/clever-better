// Package logger provides strategy-specific logging.
package logger

import (
	"github.com/sirupsen/logrus"
)

// StrategyLogger provides dedicated logging for strategy operations.
type StrategyLogger struct {
	*logrus.Entry
}

// NewStrategyLogger creates a new strategy logger.
func NewStrategyLogger(baseLogger *logrus.Logger) *StrategyLogger {
	return &StrategyLogger{
		Entry: baseLogger.WithField("component", "strategy"),
	}
}

// LogStrategyEvaluation logs a strategy evaluation event.
func (sl *StrategyLogger) LogStrategyEvaluation(strategyID, strategyName, raceID string, runnersEvaluated, signalsGenerated int, durationMs float64) {
	sl.WithFields(logrus.Fields{
		"strategy_id":          strategyID,
		"strategy_name":        strategyName,
		"race_id":              raceID,
		"runners_evaluated":    runnersEvaluated,
		"signals_generated":    signalsGenerated,
		"evaluation_duration_ms": durationMs,
	}).Info("Strategy evaluation completed")
}

// LogStrategyDecision logs a strategy decision.
func (sl *StrategyLogger) LogStrategyDecision(strategyID, strategyName, decision string, confidence, edgeValue, kellyFraction, stakeAmount float64, selectionID int64, odds float64) {
	sl.WithFields(logrus.Fields{
		"strategy_id":   strategyID,
		"strategy_name": strategyName,
		"decision":      decision,
		"confidence":    confidence,
		"edge_value":    edgeValue,
		"kelly_fraction": kellyFraction,
		"stake_amount":  stakeAmount,
		"selection_id":  selectionID,
		"odds":          odds,
	}).Info("Strategy decision made")
}

// LogMLFiltering logs ML filtering results.
func (sl *StrategyLogger) LogMLFiltering(strategyID, strategyName, mlModel string, mlConfidence float64, mlRecommendation string, signalsBefore, signalsAfter int) {
	sl.WithFields(logrus.Fields{
		"strategy_id":        strategyID,
		"strategy_name":      strategyName,
		"ml_model_used":      mlModel,
		"ml_confidence":      mlConfidence,
		"ml_recommendation":  mlRecommendation,
		"signals_before":     signalsBefore,
		"signals_after":      signalsAfter,
	}).Info("ML filtering applied to strategy signals")
}

// LogStrategyActivation logs strategy activation.
func (sl *StrategyLogger) LogStrategyActivation(strategyID, strategyName, reason string) {
	sl.WithFields(logrus.Fields{
		"strategy_id":   strategyID,
		"strategy_name": strategyName,
		"event_type":    "activation",
		"reason":        reason,
	}).Info("Strategy activated")
}

// LogStrategyDeactivation logs strategy deactivation.
func (sl *StrategyLogger) LogStrategyDeactivation(strategyID, strategyName, reason string) {
	sl.WithFields(logrus.Fields{
		"strategy_id":   strategyID,
		"strategy_name": strategyName,
		"event_type":    "deactivation",
		"reason":        reason,
	}).Info("Strategy deactivated")
}

// LogStrategyPnLUpdate logs strategy P&L updates.
func (sl *StrategyLogger) LogStrategyPnLUpdate(strategyID, strategyName string, pnl, bankroll float64, winStreak, lossStreak int) {
	sl.WithFields(logrus.Fields{
		"strategy_id":   strategyID,
		"strategy_name": strategyName,
		"pnl":           pnl,
		"bankroll":      bankroll,
		"win_streak":    winStreak,
		"loss_streak":   lossStreak,
	}).Info("Strategy P&L updated")
}

// LogStrategyDrawdown logs drawdown events.
func (sl *StrategyLogger) LogStrategyDrawdown(strategyID, strategyName string, drawdownPercent float64, peakBankroll, currentBankroll float64) {
	sl.WithFields(logrus.Fields{
		"strategy_id":        strategyID,
		"strategy_name":      strategyName,
		"drawdown_percent":   drawdownPercent,
		"peak_bankroll":      peakBankroll,
		"current_bankroll":   currentBankroll,
	}).Warn("Strategy drawdown threshold exceeded")
}
