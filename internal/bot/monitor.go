package bot

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// MonitorMetrics tracks monitoring statistics
type MonitorMetrics struct {
	UpdatesPerformed int64     `json:"updates_performed"`
	LastUpdateTime   time.Time `json:"last_update_time"`
	UpdateErrors     int64     `json:"update_errors"`
}

// LivePerformance represents real-time strategy performance
type LivePerformance struct {
	StrategyID   uuid.UUID `json:"strategy_id"`
	TotalBets    int       `json:"total_bets"`
	WinningBets  int       `json:"winning_bets"`
	LosingBets   int       `json:"losing_bets"`
	PendingBets  int       `json:"pending_bets"`
	TotalPL      float64   `json:"total_pl"`
	WinRate      float64   `json:"win_rate"`
	ROI          float64   `json:"roi"`
	AverageStake float64   `json:"average_stake"`
	LargestWin   float64   `json:"largest_win"`
	LargestLoss  float64   `json:"largest_loss"`
	CurrentStreak int      `json:"current_streak"` // Positive for wins, negative for losses
	UpdatedAt    time.Time `json:"updated_at"`
}

// DashboardData aggregates monitoring information
type DashboardData struct {
	TotalStrategies   int                `json:"total_strategies"`
	ActiveStrategies  int                `json:"active_strategies"`
	TotalBetsToday    int                `json:"total_bets_today"`
	TotalPLToday      float64            `json:"total_pl_today"`
	TopPerformers     []*LivePerformance `json:"top_performers"`
	RecentBets        []*models.Bet      `json:"recent_bets"`
}

// Monitor handles live performance tracking
type Monitor struct {
	betRepo          repository.BetRepository
	strategyRepo     repository.StrategyRepository
	strategyPerfRepo repository.StrategyPerformanceRepository
	circuitBreaker   *CircuitBreaker
	baseBankroll     float64
	updateInterval   time.Duration
	logger           *logrus.Logger
	metrics          *MonitorMetrics
	mu               sync.RWMutex
	done             chan struct{}
}

// NewMonitor creates a new performance monitor
func NewMonitor(
	betRepo repository.BetRepository,
	strategyRepo repository.StrategyRepository,
	strategyPerfRepo repository.StrategyPerformanceRepository,
	circuitBreaker *CircuitBreaker,
	baseBankroll float64,
	updateInterval time.Duration,
	logger *logrus.Logger,
) *Monitor {
	return &Monitor{
		betRepo:          betRepo,
		strategyRepo:     strategyRepo,
		strategyPerfRepo: strategyPerfRepo,
		circuitBreaker:   circuitBreaker,
		baseBankroll:     baseBankroll,
		updateInterval:   updateInterval,
		logger:           logger,
		metrics: &MonitorMetrics{
			LastUpdateTime: time.Now(),
		},
		done: make(chan struct{}),
	}
}

// Start begins the monitoring loop
func (m *Monitor) Start(ctx context.Context) error {
	m.logger.WithField("update_interval", m.updateInterval).Info("Starting performance monitor")

	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	// Perform initial update
	if err := m.UpdatePerformance(ctx); err != nil {
		m.logger.WithError(err).Error("Initial performance update failed")
	}

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Performance monitor stopped by context")
			return ctx.Err()

		case <-m.done:
			m.logger.Info("Performance monitor stopped")
			return nil

		case <-ticker.C:
			if err := m.UpdatePerformance(ctx); err != nil {
				m.logger.WithError(err).Error("Performance update failed")
			}
		}
	}
}

// Stop gracefully stops the monitor
func (m *Monitor) Stop() error {
	close(m.done)
	return nil
}

// UpdatePerformance calculates and stores strategy performance metrics
func (m *Monitor) UpdatePerformance(ctx context.Context) error {
	m.logger.Debug("Updating strategy performance metrics")

	// Get all active strategies
	strategies, err := m.strategyRepo.GetAll(ctx)
	if err != nil {
		m.mu.Lock()
		m.metrics.UpdateErrors++
		m.mu.Unlock()
		return fmt.Errorf("failed to get strategies: %w", err)
	}

	// Filter for active strategies
	activeStrategies := make([]*models.Strategy, 0)
	for _, strategy := range strategies {
		if strategy.IsActive {
			activeStrategies = append(activeStrategies, strategy)
		}
	}

	// Calculate metrics for each active strategy
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for _, strategy := range activeStrategies {
		// Get all bets for this strategy in current month
		bets, err := m.betRepo.GetByStrategyID(ctx, strategy.ID, startOfMonth, now)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"strategy_id": strategy.ID,
				"error":       err.Error(),
			}).Error("Failed to get bets for strategy")
			continue
		}

		if len(bets) == 0 {
			continue
		}

		// Calculate performance metrics
		var (
			totalBets    = len(bets)
			winningBets  = 0
			losingBets   = 0
			totalPL      = 0.0
			totalStake   = 0.0
			largestWin   = 0.0
			largestLoss  = 0.0
		)

		settledBets := make([]*models.Bet, 0)

		for _, bet := range bets {
			totalStake += bet.Stake
			if bet.Status == models.BetStatusSettled {
				settledBets = append(settledBets, bet)
			}

			if bet.ProfitLoss != nil {
				pl := *bet.ProfitLoss
				totalPL += pl

				if pl > 0 {
					winningBets++
					if pl > largestWin {
						largestWin = pl
					}
				} else if pl < 0 {
					losingBets++
					if pl < largestLoss {
						largestLoss = pl
					}
				}
			}
		}

		// Feed settled bet results into circuit breaker (loss/drawdown tracking)
		if m.circuitBreaker != nil && len(settledBets) > 0 {
			sort.Slice(settledBets, func(i, j int) bool {
				left := settledBets[i].SettledAt
				right := settledBets[j].SettledAt
				if left == nil && right == nil {
					return settledBets[i].PlacedAt.Before(settledBets[j].PlacedAt)
				}
				if left == nil {
					return false
				}
				if right == nil {
					return true
				}
				return left.Before(*right)
			})

			cumulativePL := 0.0
			for _, bet := range settledBets {
				if bet.ProfitLoss == nil {
					continue
				}
				cumulativePL += *bet.ProfitLoss
				currentBankroll := m.baseBankroll + cumulativePL
				m.circuitBreaker.RecordBetResult(bet, currentBankroll)
			}
		}

		// Calculate derived metrics
		winRate := 0.0
		if totalBets > 0 {
			winRate = float64(winningBets) / float64(totalBets)
		}

		roi := 0.0
		if totalStake > 0 {
			roi = totalPL / totalStake
		}

		avgStake := 0.0
		if totalBets > 0 {
			avgStake = totalStake / float64(totalBets)
		}

		// Store performance record
		perfRecord := &models.StrategyPerformance{
			ID:               uuid.New(),
			StrategyID:       strategy.ID,
			TotalBets:        totalBets,
			WinningBets:      winningBets,
			LosingBets:       losingBets,
			TotalProfitLoss:  totalPL,
			WinRate:          winRate,
			ROI:              roi,
			AverageStake:     avgStake,
			LargestWin:       largestWin,
			LargestLoss:      largestLoss,
			Period:           "monthly",
			PeriodStart:      startOfMonth,
			PeriodEnd:        now,
		}

		if err := m.strategyPerfRepo.Create(ctx, perfRecord); err != nil {
			m.logger.WithFields(logrus.Fields{
				"strategy_id": strategy.ID,
				"error":       err.Error(),
			}).Error("Failed to store performance record")
			continue
		}

		m.logger.WithFields(logrus.Fields{
			"strategy_id":  strategy.ID,
			"total_bets":   totalBets,
			"winning_bets": winningBets,
			"total_pl":     totalPL,
			"win_rate":     winRate,
			"roi":          roi,
		}).Info("Strategy performance updated")
	}

	m.mu.Lock()
	m.metrics.UpdatesPerformed++
	m.metrics.LastUpdateTime = time.Now()
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"strategies_updated": len(activeStrategies),
		"total_strategies":   len(strategies),
	}).Info("Performance update completed")

	return nil
}

// GetLiveMetrics returns real-time performance for a strategy
func (m *Monitor) GetLiveMetrics(ctx context.Context, strategyID uuid.UUID) (*LivePerformance, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get all bets for this strategy today
	bets, err := m.betRepo.GetByStrategyID(ctx, strategyID, startOfDay, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get bets for strategy: %w", err)
	}

	perf := &LivePerformance{
		StrategyID: strategyID,
		UpdatedAt:  now,
	}

	var (
		totalStake   = 0.0
		currentStreak = 0
		lastBetWon    = false
	)

	for _, bet := range bets {
		totalStake += bet.Stake

		if bet.Status == models.BetStatusPending {
			perf.PendingBets++
		}

		if bet.ProfitLoss != nil {
			pl := *bet.ProfitLoss
			perf.TotalPL += pl

			if pl > 0 {
				perf.WinningBets++
				if pl > perf.LargestWin {
					perf.LargestWin = pl
				}
				if lastBetWon {
					currentStreak++
				} else {
					currentStreak = 1
					lastBetWon = true
				}
			} else if pl < 0 {
				perf.LosingBets++
				if pl < perf.LargestLoss {
					perf.LargestLoss = pl
				}
				if !lastBetWon {
					currentStreak--
				} else {
					currentStreak = -1
					lastBetWon = false
				}
			}
		}
	}

	perf.TotalBets = len(bets)
	perf.CurrentStreak = currentStreak

	if perf.TotalBets > 0 {
		perf.WinRate = float64(perf.WinningBets) / float64(perf.TotalBets)
		perf.AverageStake = totalStake / float64(perf.TotalBets)
	}

	if totalStake > 0 {
		perf.ROI = perf.TotalPL / totalStake
	}

	return perf, nil
}

// GetDashboardData aggregates data for monitoring dashboard
func (m *Monitor) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get all strategies
	strategies, err := m.strategyRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategies: %w", err)
	}

	activeCount := 0
	for _, strategy := range strategies {
		if strategy.IsActive {
			activeCount++
		}
	}

	// Get today's bets
	todayBets, err := m.betRepo.GetSettledBets(ctx, startOfDay, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's bets: %w", err)
	}

	totalPLToday := 0.0
	for _, bet := range todayBets {
		if bet.ProfitLoss != nil {
			totalPLToday += *bet.ProfitLoss
		}
	}

	// Get top performers (simplified - would need more complex query in production)
	topPerformers := make([]*LivePerformance, 0)
	for _, strategy := range strategies {
		if !strategy.IsActive {
			continue
		}

		perf, err := m.GetLiveMetrics(ctx, strategy.ID)
		if err != nil {
			m.logger.WithError(err).Error("Failed to get live metrics for strategy")
			continue
		}

		topPerformers = append(topPerformers, perf)
	}

	// Get recent bets (last 10)
	recentBets := make([]*models.Bet, 0)
	if len(todayBets) > 0 {
		limit := 10
		if len(todayBets) < limit {
			limit = len(todayBets)
		}
		recentBets = todayBets[:limit]
	}

	return &DashboardData{
		TotalStrategies:  len(strategies),
		ActiveStrategies: activeCount,
		TotalBetsToday:   len(todayBets),
		TotalPLToday:     totalPLToday,
		TopPerformers:    topPerformers,
		RecentBets:       recentBets,
	}, nil
}
