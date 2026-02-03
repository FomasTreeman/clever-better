package bot

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/models"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// CircuitClosed means trading is active
	CircuitClosed CircuitState = iota
	// CircuitHalfOpen means trading is resuming after cooldown
	CircuitHalfOpen
	// CircuitOpen means trading is halted
	CircuitOpen
)

// String returns string representation of circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	case CircuitOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines circuit breaker thresholds
type CircuitBreakerConfig struct {
	MaxConsecutiveLosses int           `json:"max_consecutive_losses"`
	MaxDrawdownPercent   float64       `json:"max_drawdown_percent"`
	MaxFailureCount      int           `json:"max_failure_count"`
	FailureTimeWindow    time.Duration `json:"failure_time_window"`
	CooldownPeriod       time.Duration `json:"cooldown_period"`
}

// ShutdownCallback is called when emergency shutdown is triggered
type ShutdownCallback func(reason string) error

// CircuitBreaker implements emergency trading shutdown mechanisms
type CircuitBreaker struct {
	config            CircuitBreakerConfig
	state             CircuitState
	failureCount      int
	lastFailureTime   time.Time
	consecutiveLosses int
	drawdown          float64
	peakBankroll      float64
	mu                sync.RWMutex
	logger            *logrus.Logger
	callbacks         []ShutdownCallback
	openedAt          time.Time
}

// NewCircuitBreaker creates a new circuit breaker with default config
func NewCircuitBreaker(config CircuitBreakerConfig, logger *logrus.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config:       config,
		state:        CircuitClosed,
		peakBankroll: 0,
		logger:       logger,
		callbacks:    make([]ShutdownCallback, 0),
	}
}

// RecordBetResult tracks bet outcomes for loss streaks and drawdown
func (cb *CircuitBreaker) RecordBetResult(bet *models.Bet, currentBankroll float64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Update peak bankroll
	if currentBankroll > cb.peakBankroll {
		cb.peakBankroll = currentBankroll
	}

	// Calculate current drawdown
	if cb.peakBankroll > 0 {
		cb.drawdown = (cb.peakBankroll - currentBankroll) / cb.peakBankroll
	}

	// Check for bet loss
	if bet.ProfitLoss != nil && *bet.ProfitLoss < 0 {
		cb.consecutiveLosses++

		cb.logger.WithFields(logrus.Fields{
			"consecutive_losses": cb.consecutiveLosses,
			"max_allowed":        cb.config.MaxConsecutiveLosses,
			"drawdown":           cb.drawdown,
			"max_drawdown":       cb.config.MaxDrawdownPercent,
		}).Warn("Consecutive loss recorded")

		// Check max consecutive losses
		if cb.consecutiveLosses >= cb.config.MaxConsecutiveLosses {
			cb.triggerEmergencyShutdownLocked(fmt.Sprintf(
				"Max consecutive losses exceeded (%d >= %d)",
				cb.consecutiveLosses, cb.config.MaxConsecutiveLosses,
			))
			return
		}

		// Check max drawdown
		if cb.drawdown >= cb.config.MaxDrawdownPercent {
			cb.triggerEmergencyShutdownLocked(fmt.Sprintf(
				"Max drawdown exceeded (%.2f%% >= %.2f%%)",
				cb.drawdown*100, cb.config.MaxDrawdownPercent*100,
			))
			return
		}
	} else if bet.ProfitLoss != nil && *bet.ProfitLoss > 0 {
		// Reset consecutive losses on win
		cb.consecutiveLosses = 0
	}
}

// RecordFailure increments failure count and opens circuit if threshold exceeded
func (cb *CircuitBreaker) RecordFailure(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// Reset failure count if outside time window
	if now.Sub(cb.lastFailureTime) > cb.config.FailureTimeWindow {
		cb.failureCount = 0
	}

	cb.failureCount++
	cb.lastFailureTime = now

	cb.logger.WithFields(logrus.Fields{
		"failure_count": cb.failureCount,
		"max_allowed":   cb.config.MaxFailureCount,
		"time_window":   cb.config.FailureTimeWindow,
		"error":         err.Error(),
	}).Warn("Failure recorded")

	if cb.failureCount >= cb.config.MaxFailureCount {
		cb.triggerEmergencyShutdownLocked(fmt.Sprintf(
			"Max failure count exceeded (%d >= %d) within %v",
			cb.failureCount, cb.config.MaxFailureCount, cb.config.FailureTimeWindow,
		))
	}
}

// RecordSuccess resets failure count
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
}

// IsOpen returns true if circuit is open (trading halted)
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Check if cooldown period has passed for half-open state
	if cb.state == CircuitOpen && time.Since(cb.openedAt) > cb.config.CooldownPeriod {
		cb.mu.RUnlock()
		cb.mu.Lock()
		cb.state = CircuitHalfOpen
		cb.logger.Info("Circuit breaker entering half-open state after cooldown")
		cb.mu.Unlock()
		cb.mu.RLock()
	}

	return cb.state == CircuitOpen
}

// GetState returns current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}

// Reset manually resets circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.consecutiveLosses = 0

	cb.logger.WithFields(logrus.Fields{
		"old_state": oldState.String(),
		"new_state": cb.state.String(),
	}).Info("Circuit breaker manually reset")
}

// RegisterShutdownCallback registers a callback for emergency shutdown
func (cb *CircuitBreaker) RegisterShutdownCallback(callback ShutdownCallback) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.callbacks = append(cb.callbacks, callback)
}

// TriggerEmergencyShutdown opens circuit and executes all callbacks
func (cb *CircuitBreaker) TriggerEmergencyShutdown(reason string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.triggerEmergencyShutdownLocked(reason)
}

// triggerEmergencyShutdownLocked is internal version that assumes lock is held
func (cb *CircuitBreaker) triggerEmergencyShutdownLocked(reason string) {
	if cb.state == CircuitOpen {
		cb.logger.Warn("Emergency shutdown already triggered, ignoring duplicate call")
		return
	}

	oldState := cb.state
	cb.state = CircuitOpen
	cb.openedAt = time.Now()

	cb.logger.WithFields(logrus.Fields{
		"old_state":          oldState.String(),
		"new_state":          cb.state.String(),
		"reason":             reason,
		"consecutive_losses": cb.consecutiveLosses,
		"drawdown":           cb.drawdown,
		"failure_count":      cb.failureCount,
		"cooldown_period":    cb.config.CooldownPeriod,
	}).Error("EMERGENCY SHUTDOWN TRIGGERED")

	// Execute all shutdown callbacks
	for i, callback := range cb.callbacks {
		if err := callback(reason); err != nil {
			cb.logger.WithFields(logrus.Fields{
				"callback_index": i,
				"error":          err.Error(),
			}).Error("Shutdown callback failed")
		}
	}
}
