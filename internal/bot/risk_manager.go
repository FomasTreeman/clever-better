package bot

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/repository"
)

// RiskMetrics represents current risk exposure and limits
type RiskMetrics struct {
	CurrentExposure   float64   `json:"current_exposure"`
	DailyLoss         float64   `json:"daily_loss"`
	MaxExposure       float64   `json:"max_exposure"`
	MaxDailyLoss      float64   `json:"max_daily_loss"`
	RemainingCapacity float64   `json:"remaining_capacity"`
	BetsToday         int       `json:"bets_today"`
	LastUpdate        time.Time `json:"last_update"`
}

// RiskManager handles position sizing and risk limit validation
type RiskManager struct {
	config             *config.TradingConfig
	betRepo            repository.BetRepository
	currentExposure    float64
	dailyLoss          float64
	dailyLossResetTime time.Time
	mu                 sync.RWMutex
	logger             *logrus.Logger
}

// NewRiskManager creates a new risk manager
func NewRiskManager(cfg *config.TradingConfig, betRepo repository.BetRepository, logger *logrus.Logger) *RiskManager {
	now := time.Now()
	resetTime := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	return &RiskManager{
		config:             cfg,
		betRepo:            betRepo,
		currentExposure:    0,
		dailyLoss:          0,
		dailyLossResetTime: resetTime,
		logger:             logger,
	}
}

// CalculatePositionSize calculates stake using Kelly Criterion with fractional sizing
func (rm *RiskManager) CalculatePositionSize(odds float64, bankroll float64, confidence float64, edgeEstimate float64) (float64, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Kelly Criterion: f = (bp - q) / b
	// where f = fraction of bankroll to bet
	//       b = decimal odds - 1
	//       p = probability of winning (confidence)
	//       q = probability of losing (1 - p)

	b := odds - 1.0
	p := confidence
	q := 1.0 - p

	// Calculate Kelly fraction
	kellyFraction := (b*p - q) / b

	// Apply fractional Kelly (typically 0.25 to 0.5 of full Kelly for safety)
	fractionalKelly := kellyFraction * 0.25

	// Ensure non-negative
	if fractionalKelly < 0 {
		rm.logger.WithFields(logrus.Fields{
			"odds":       odds,
			"confidence": confidence,
			"kelly":      kellyFraction,
		}).Debug("Negative Kelly fraction, no bet recommended")
		return 0, nil
	}

	// Calculate stake
	stake := bankroll * fractionalKelly

	// Apply maximum stake limit
	if stake > rm.config.MaxStakePerBet {
		rm.logger.WithFields(logrus.Fields{
			"calculated_stake": stake,
			"max_stake":        rm.config.MaxStakePerBet,
		}).Debug("Stake capped at maximum")
		stake = rm.config.MaxStakePerBet
	}

	// Minimum stake check (avoid dust bets)
	minStake := 2.0
	if stake < minStake {
		rm.logger.WithFields(logrus.Fields{
			"calculated_stake": stake,
			"min_stake":        minStake,
		}).Debug("Stake below minimum, no bet recommended")
		return 0, nil
	}

	rm.logger.WithFields(logrus.Fields{
		"bankroll":         bankroll,
		"odds":             odds,
		"confidence":       confidence,
		"kelly_fraction":   kellyFraction,
		"fractional_kelly": fractionalKelly,
		"stake":            stake,
	}).Debug("Position size calculated")

	return stake, nil
}

// CheckRiskLimits validates proposed stake against risk limits
func (rm *RiskManager) CheckRiskLimits(ctx context.Context, proposedStake float64) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Check if daily loss reset is needed
	if time.Now().After(rm.dailyLossResetTime) {
		rm.mu.RUnlock()
		if err := rm.UpdateDailyLoss(ctx); err != nil {
			rm.mu.RLock()
			rm.logger.WithError(err).Error("Failed to update daily loss")
		} else {
			rm.mu.RLock()
		}
	}

	// Check max stake per bet
	if proposedStake > rm.config.MaxStakePerBet {
		return fmt.Errorf("proposed stake %.2f exceeds max stake per bet %.2f", 
			proposedStake, rm.config.MaxStakePerBet)
	}

	// Check max exposure
	newExposure := rm.currentExposure + proposedStake
	if newExposure > rm.config.MaxExposure {
		return fmt.Errorf("proposed stake would exceed max exposure (current: %.2f, proposed: %.2f, max: %.2f)", 
			rm.currentExposure, proposedStake, rm.config.MaxExposure)
	}

	// Check max daily loss
	if rm.dailyLoss >= rm.config.MaxDailyLoss {
		return fmt.Errorf("daily loss limit reached (current: %.2f, max: %.2f)", 
			rm.dailyLoss, rm.config.MaxDailyLoss)
	}

	rm.logger.WithFields(logrus.Fields{
		"proposed_stake":    proposedStake,
		"current_exposure":  rm.currentExposure,
		"daily_loss":        rm.dailyLoss,
		"max_exposure":      rm.config.MaxExposure,
		"max_daily_loss":    rm.config.MaxDailyLoss,
	}).Debug("Risk limits validated successfully")

	return nil
}

// UpdateExposure recalculates current exposure from pending bets
func (rm *RiskManager) UpdateExposure(ctx context.Context) error {
	pendingBets, err := rm.betRepo.GetPendingBets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending bets: %w", err)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	totalExposure := 0.0
	for _, bet := range pendingBets {
		totalExposure += bet.Stake
	}

	rm.currentExposure = totalExposure

	rm.logger.WithFields(logrus.Fields{
		"pending_bets":      len(pendingBets),
		"current_exposure":  rm.currentExposure,
		"max_exposure":      rm.config.MaxExposure,
		"exposure_used_pct": (rm.currentExposure / rm.config.MaxExposure) * 100,
	}).Info("Exposure updated")

	return nil
}

// UpdateDailyLoss calculates P&L for current day and resets at midnight
func (rm *RiskManager) UpdateDailyLoss(ctx context.Context) error {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	settledBets, err := rm.betRepo.GetSettledBets(ctx, startOfDay, endOfDay)
	if err != nil {
		return fmt.Errorf("failed to get settled bets: %w", err)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	totalPL := 0.0
	for _, bet := range settledBets {
		if bet.ProfitLoss != nil {
			totalPL += *bet.ProfitLoss
		}
	}

	// Daily loss is negative P&L
	if totalPL < 0 {
		rm.dailyLoss = math.Abs(totalPL)
	} else {
		rm.dailyLoss = 0
	}

	// Reset time for next day
	rm.dailyLossResetTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	rm.logger.WithFields(logrus.Fields{
		"settled_bets_today": len(settledBets),
		"total_pl":           totalPL,
		"daily_loss":         rm.dailyLoss,
		"max_daily_loss":     rm.config.MaxDailyLoss,
		"next_reset":         rm.dailyLossResetTime,
	}).Info("Daily loss updated")

	return nil
}

// IsWithinLimits checks if current state allows new bets
func (rm *RiskManager) IsWithinLimits() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.currentExposure >= rm.config.MaxExposure {
		rm.logger.Warn("Max exposure limit reached")
		return false
	}

	if rm.dailyLoss >= rm.config.MaxDailyLoss {
		rm.logger.Warn("Max daily loss limit reached")
		return false
	}

	return true
}

// GetRiskMetrics returns current risk metrics for monitoring
func (rm *RiskManager) GetRiskMetrics() RiskMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return RiskMetrics{
		CurrentExposure:   rm.currentExposure,
		DailyLoss:         rm.dailyLoss,
		MaxExposure:       rm.config.MaxExposure,
		MaxDailyLoss:      rm.config.MaxDailyLoss,
		RemainingCapacity: rm.config.MaxExposure - rm.currentExposure,
		LastUpdate:        time.Now(),
	}
}
