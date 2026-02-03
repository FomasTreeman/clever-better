// Package logger provides audit logging.
package logger

import (
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLogger provides dedicated audit trail logging.
type AuditLogger struct {
	*logrus.Entry
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(baseLogger *logrus.Logger) *AuditLogger {
	return &AuditLogger{
		Entry: baseLogger.WithField("component", "audit"),
	}
}

// LogBetPlacement logs a bet placement event.
func (al *AuditLogger) LogBetPlacement(betID, strategyID, marketID string, selectionID int64, betType string, stake, odds float64, timestamp time.Time, paperTrading bool) {
	al.WithFields(logrus.Fields{
		"bet_id":        betID,
		"strategy_id":   strategyID,
		"market_id":     marketID,
		"selection_id":  selectionID,
		"bet_type":      betType,
		"stake":         stake,
		"odds":          odds,
		"timestamp":     timestamp.Unix(),
		"paper_trading": paperTrading,
	}).Info("Bet placement recorded")
}

// LogBetStateChange logs a bet state change.
func (al *AuditLogger) LogBetStateChange(betID string, oldState, newState string, matchedAmount, unmatchedAmount float64) {
	al.WithFields(logrus.Fields{
		"bet_id":             betID,
		"old_state":          oldState,
		"new_state":          newState,
		"matched_amount":     matchedAmount,
		"unmatched_amount":   unmatchedAmount,
	}).Info("Bet state changed")
}

// LogStrategyParameterChange logs strategy parameter changes.
func (al *AuditLogger) LogStrategyParameterChange(strategyID, parameterName string, oldValue, newValue interface{}, changedBy string) {
	al.WithFields(logrus.Fields{
		"strategy_id":    strategyID,
		"parameter_name": parameterName,
		"old_value":      oldValue,
		"new_value":      newValue,
		"changed_by":     changedBy,
	}).Info("Strategy parameter changed")
}

// LogCircuitBreakerEvent logs circuit breaker events.
func (al *AuditLogger) LogCircuitBreakerEvent(eventType, reason string, metricsSnapshot map[string]interface{}, actionTaken string) {
	al.WithFields(logrus.Fields{
		"event_type":       eventType,
		"reason":           reason,
		"metrics_snapshot": metricsSnapshot,
		"action_taken":     actionTaken,
	}).Warn("Circuit breaker event recorded")
}

// LogEmergencyShutdown logs emergency shutdown events with system state.
func (al *AuditLogger) LogEmergencyShutdown(reason string, systemState map[string]interface{}) {
	al.WithFields(logrus.Fields{
		"reason":       reason,
		"system_state": systemState,
	}).Fatal("Emergency shutdown initiated")
}
