package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

// SignalWithContext wraps a strategy signal with execution context
type SignalWithContext struct {
	Signal      strategy.Signal `json:"signal"`
	StrategyID  uuid.UUID       `json:"strategy_id"`
	RaceID      uuid.UUID       `json:"race_id"`
	MarketID    string          `json:"market_id"`
	SelectionID uint64          `json:"selection_id"`
}

// ExecutorMetrics tracks execution statistics
type ExecutorMetrics struct {
	OrdersExecuted       int64         `json:"orders_executed"`
	OrdersRejected       int64         `json:"orders_rejected"`
	PaperTrades          int64         `json:"paper_trades"`
	LiveTrades           int64         `json:"live_trades"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecutionTime    time.Time     `json:"last_execution_time"`
}

// Executor handles order execution for both live and paper trading
type Executor struct {
	bettingService   *betfair.BettingService
	betRepo          repository.BetRepository
	riskManager      *RiskManager
	paperTradingMode bool
	liveTradingEnabled bool
	logger           *logrus.Logger
	metrics          *ExecutorMetrics
	mu               sync.Mutex
}

// NewExecutor creates a new order executor
func NewExecutor(
	bettingService *betfair.BettingService,
	betRepo repository.BetRepository,
	riskManager *RiskManager,
	paperTradingMode bool,
	liveTradingEnabled bool,
	logger *logrus.Logger,
) *Executor {
	if !liveTradingEnabled {
		paperTradingMode = true
	}
	return &Executor{
		bettingService:   bettingService,
		betRepo:          betRepo,
		riskManager:      riskManager,
		paperTradingMode: paperTradingMode,
		liveTradingEnabled: liveTradingEnabled,
		logger:           logger,
		metrics: &ExecutorMetrics{
			LastExecutionTime: time.Now(),
		},
	}
}

// ExecuteSignal executes a single trading signal
func (e *Executor) ExecuteSignal(
	ctx context.Context,
	signal strategy.Signal,
	strategyID uuid.UUID,
	raceID uuid.UUID,
	marketID string,
	selectionID uint64,
) (*models.Bet, error) {
	startTime := time.Now()
	defer func() {
		e.updateExecutionMetrics(time.Since(startTime))
	}()

	// Validate signal with risk manager
	if err := e.riskManager.CheckRiskLimits(ctx, signal.Stake); err != nil {
		e.logger.WithFields(logrus.Fields{
			"strategy_id": strategyID,
			"race_id":     raceID,
			"runner_id":   signal.RunnerID,
			"stake":       signal.Stake,
			"reason":      err.Error(),
		}).Warn("Signal rejected by risk manager")

		e.mu.Lock()
		e.metrics.OrdersRejected++
		e.mu.Unlock()

		return nil, fmt.Errorf("risk limit check failed: %w", err)
	}

	// Create bet record
	bet := &models.Bet{
		ID:         uuid.New(),
		MarketID:   marketID,
		RaceID:     raceID,
		RunnerID:   signal.RunnerID,
		StrategyID: strategyID,
		MarketType: models.MarketTypeWin,
		Side:       models.BetSideBack,
		Odds:       signal.Odds,
		Stake:      signal.Stake,
		Status:     models.BetStatusPending,
		PlacedAt:   time.Now(),
	}

	// Store bet in database first
	if err := e.betRepo.Create(ctx, bet); err != nil {
		e.logger.WithError(err).Error("Failed to create bet record")
		e.mu.Lock()
		e.metrics.OrdersRejected++
		e.mu.Unlock()
		return nil, fmt.Errorf("failed to create bet record: %w", err)
	}

	// Paper trading mode: simulate execution
	if e.paperTradingMode {
		e.logger.WithFields(logrus.Fields{
			"bet_id":      bet.ID,
			"strategy_id": strategyID,
			"race_id":     raceID,
			"runner_id":   signal.RunnerID,
			"odds":        signal.Odds,
			"stake":       signal.Stake,
			"confidence":  signal.Confidence,
		}).Info("Paper trade executed (simulated)")

		e.mu.Lock()
		e.metrics.OrdersExecuted++
		e.metrics.PaperTrades++
		e.mu.Unlock()

		return bet, nil
	}

	if !e.liveTradingEnabled {
		e.logger.WithFields(logrus.Fields{
			"bet_id":      bet.ID,
			"strategy_id": strategyID,
			"race_id":     raceID,
			"runner_id":   signal.RunnerID,
			"odds":        signal.Odds,
			"stake":       signal.Stake,
		}).Warn("Live trading disabled; refusing live bet placement")

		e.mu.Lock()
		e.metrics.OrdersRejected++
		e.mu.Unlock()

		return nil, fmt.Errorf("live trading disabled")
	}

	if e.bettingService == nil {
		return nil, fmt.Errorf("betting service is not initialized")
	}

	// Live trading mode: execute via Betfair API
	betfairBetID, err := e.bettingService.PlaceBet(ctx, &betfair.PlaceBetRequest{
		MarketID:    marketID,
		SelectionID: selectionID,
		Side:        string(bet.Side),
		Odds:        bet.Odds,
		Stake:       bet.Stake,
	})

	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"bet_id":    bet.ID,
			"market_id": marketID,
			"runner_id": signal.RunnerID,
			"error":     err.Error(),
		}).Error("Failed to place bet with Betfair")

		// Update bet status to cancelled
		bet.Status = models.BetStatusCancelled
		now := time.Now()
		bet.CancelledAt = &now
		if updateErr := e.betRepo.Update(ctx, bet); updateErr != nil {
			e.logger.WithError(updateErr).Error("Failed to update cancelled bet")
		}

		e.mu.Lock()
		e.metrics.OrdersRejected++
		e.mu.Unlock()

		return nil, fmt.Errorf("failed to place bet with Betfair: %w", err)
	}

	// Update bet record with Betfair bet ID
	bet.BetID = betfairBetID
	if err := e.betRepo.Update(ctx, bet); err != nil {
		e.logger.WithError(err).Error("Failed to update bet with Betfair ID")
		// Note: bet was placed successfully, so we don't return error
	}

	e.logger.WithFields(logrus.Fields{
		"bet_id":         bet.ID,
		"betfair_bet_id": betfairBetID,
		"strategy_id":    strategyID,
		"race_id":        raceID,
		"runner_id":      signal.RunnerID,
		"market_id":      marketID,
		"side":           bet.Side,
		"odds":           bet.Odds,
		"stake":          bet.Stake,
		"confidence":     signal.Confidence,
	}).Info("Live bet executed successfully")

	e.mu.Lock()
	e.metrics.OrdersExecuted++
	e.metrics.LiveTrades++
	e.mu.Unlock()

	return bet, nil
}

// ExecuteBatch executes multiple signals efficiently
func (e *Executor) ExecuteBatch(ctx context.Context, signals []SignalWithContext) ([]*models.Bet, error) {
	bets := make([]*models.Bet, 0, len(signals))
	errors := make([]error, 0)

	e.logger.WithField("signal_count", len(signals)).Info("Executing batch of signals")

	for _, signalCtx := range signals {
		bet, err := e.ExecuteSignal(
			ctx,
			signalCtx.Signal,
			signalCtx.StrategyID,
			signalCtx.RaceID,
			signalCtx.MarketID,
			signalCtx.SelectionID,
		)

		if err != nil {
			e.logger.WithFields(logrus.Fields{
				"strategy_id": signalCtx.StrategyID,
				"race_id":     signalCtx.RaceID,
				"error":       err.Error(),
			}).Warn("Failed to execute signal in batch")
			errors = append(errors, err)
			continue
		}

		bets = append(bets, bet)
	}

	e.logger.WithFields(logrus.Fields{
		"total_signals":    len(signals),
		"successful_bets":  len(bets),
		"failed_bets":      len(errors),
		"paper_trading":    e.paperTradingMode,
	}).Info("Batch execution completed")

	if len(errors) > 0 {
		return bets, fmt.Errorf("batch execution completed with %d errors", len(errors))
	}

	return bets, nil
}

// CancelBet cancels an unmatched bet via Betfair API
func (e *Executor) CancelBet(ctx context.Context, betID uuid.UUID) error {
	bet, err := e.betRepo.GetByID(ctx, betID)
	if err != nil {
		return fmt.Errorf("failed to get bet: %w", err)
	}

	if bet.Status != models.BetStatusPending {
		return fmt.Errorf("cannot cancel bet with status %s", bet.Status)
	}

	// Paper trading mode: just mark as cancelled
	if e.paperTradingMode {
		bet.Status = models.BetStatusCancelled
		now := time.Now()
		bet.CancelledAt = &now
		if err := e.betRepo.Update(ctx, bet); err != nil {
			return fmt.Errorf("failed to update cancelled bet: %w", err)
		}

		e.logger.WithField("bet_id", betID).Info("Paper trade cancelled (simulated)")
		return nil
	}

	// Live trading mode: cancel via Betfair API
	if bet.BetID == "" {
		return fmt.Errorf("bet has no Betfair bet ID")
	}

	if err := e.bettingService.CancelBet(ctx, bet.MarketID, bet.BetID); err != nil {
		e.logger.WithFields(logrus.Fields{
			"bet_id":         betID,
			"betfair_bet_id": bet.BetID,
			"error":          err.Error(),
		}).Error("Failed to cancel bet with Betfair")
		return fmt.Errorf("failed to cancel bet with Betfair: %w", err)
	}

	// Update bet status
	bet.Status = models.BetStatusCancelled
	now := time.Now()
	bet.CancelledAt = &now
	if err := e.betRepo.Update(ctx, bet); err != nil {
		return fmt.Errorf("failed to update cancelled bet: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"bet_id":         betID,
		"betfair_bet_id": bet.BetID,
	}).Info("Bet cancelled successfully")

	return nil
}

// GetMetrics returns current execution metrics
func (e *Executor) GetMetrics() ExecutorMetrics {
	e.mu.Lock()
	defer e.mu.Unlock()

	return *e.metrics
}

// updateExecutionMetrics updates execution time statistics
func (e *Executor) updateExecutionMetrics(duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Calculate rolling average execution time
	if e.metrics.OrdersExecuted > 0 {
		totalTime := e.metrics.AverageExecutionTime * time.Duration(e.metrics.OrdersExecuted-1)
		e.metrics.AverageExecutionTime = (totalTime + duration) / time.Duration(e.metrics.OrdersExecuted)
	} else {
		e.metrics.AverageExecutionTime = duration
	}

	e.metrics.LastExecutionTime = time.Now()
}
