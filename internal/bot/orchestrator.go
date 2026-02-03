package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

// Repositories holds all repository dependencies
type Repositories struct {
	Strategy           repository.StrategyRepository
	Race               repository.RaceRepository
	Runner             repository.RunnerRepository
	Odds               repository.OddsRepository
	Bet                repository.BetRepository
	StrategyPerformance repository.StrategyPerformanceRepository
}

// OrchestratorStatus represents current bot status
type OrchestratorStatus struct {
	Running             bool            `json:"running"`
	PaperTradingMode    bool            `json:"paper_trading_mode"`
	ActiveStrategies    int             `json:"active_strategies"`
	CircuitBreakerState CircuitState    `json:"circuit_breaker_state"`
	RiskMetrics         RiskMetrics     `json:"risk_metrics"`
	MonitorMetrics      MonitorMetrics  `json:"monitor_metrics"`
	ExecutorMetrics     ExecutorMetrics `json:"executor_metrics"`
	LastUpdate          time.Time       `json:"last_update"`
}

// Orchestrator coordinates all bot components
type Orchestrator struct {
	config           *config.Config
	db               *database.DB
	mlClient         *ml.CachedMLClient
	bettingService   *betfair.BettingService
	orderManager     *betfair.OrderManager
	strategyRepo     repository.StrategyRepository
	raceRepo         repository.RaceRepository
	runnerRepo       repository.RunnerRepository
	oddsRepo         repository.OddsRepository
	betRepo          repository.BetRepository
	riskManager      *RiskManager
	executor         *Executor
	monitor          *Monitor
	circuitBreaker   *CircuitBreaker
	activeStrategies map[uuid.UUID]strategy.Strategy
	logger           *logrus.Logger
	done             chan struct{}
	running          bool
	mu               sync.RWMutex
}

// NewOrchestrator creates a new bot orchestrator
func NewOrchestrator(
	cfg *config.Config,
	db *database.DB,
	mlClient *ml.CachedMLClient,
	bettingService *betfair.BettingService,
	orderManager *betfair.OrderManager,
	repos Repositories,
	logger *logrus.Logger,
) (*Orchestrator, error) {
	// Initialize risk manager
	riskManager := NewRiskManager(&cfg.Trading, repos.Bet, logger)

	// Initialize executor (paper trading mode from config)
	executor := NewExecutor(
		bettingService,
		repos.Bet,
		riskManager,
		cfg.Features.PaperTradingEnabled,
		cfg.Features.LiveTradingEnabled,
		logger,
	)

	// Initialize circuit breaker
	circuitBreakerConfig := CircuitBreakerConfig{
		MaxConsecutiveLosses: cfg.Bot.MaxConsecutiveLosses,
		MaxDrawdownPercent:   cfg.Bot.MaxDrawdownPercent,
		MaxFailureCount:      10,
		FailureTimeWindow:    5 * time.Minute,
		CooldownPeriod:       30 * time.Minute,
	}
	circuitBreaker := NewCircuitBreaker(circuitBreakerConfig, logger)

	// Initialize monitor
	updateInterval := time.Duration(cfg.Bot.PerformanceUpdateInterval) * time.Second
	baseBankroll := cfg.Backtest.InitialBankroll
	monitor := NewMonitor(
		repos.Bet,
		repos.Strategy,
		repos.StrategyPerformance,
		circuitBreaker,
		baseBankroll,
		updateInterval,
		logger,
	)

	o := &Orchestrator{
		config:           cfg,
		db:               db,
		mlClient:         mlClient,
		bettingService:   bettingService,
		orderManager:     orderManager,
		strategyRepo:     repos.Strategy,
		raceRepo:         repos.Race,
		runnerRepo:       repos.Runner,
		oddsRepo:         repos.Odds,
		betRepo:          repos.Bet,
		riskManager:      riskManager,
		executor:         executor,
		monitor:          monitor,
		circuitBreaker:   circuitBreaker,
		activeStrategies: make(map[uuid.UUID]strategy.Strategy),
		logger:           logger,
		done:             make(chan struct{}),
	}

	// Register emergency shutdown callback
	if cfg.Trading.EmergencyShutdownEnabled {
		circuitBreaker.RegisterShutdownCallback(func(reason string) error {
			logger.WithField("reason", reason).Error("Emergency shutdown callback triggered")
			return o.Stop()
		})
	}

	// Load active strategies
	if err := o.loadActiveStrategies(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load active strategies: %w", err)
	}

	logger.Info("Bot orchestrator initialized successfully")

	return o, nil
}

// Start starts all bot components and begins trading loop
func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is already running")
	}
	o.running = true
	o.mu.Unlock()

	o.logger.WithFields(logrus.Fields{
		"paper_trading":      o.config.Features.PaperTradingEnabled,
		"active_strategies":  len(o.activeStrategies),
		"circuit_breaker":    o.circuitBreaker.GetState().String(),
	}).Info("Starting bot orchestrator")

	// Start order manager for bet monitoring
	if o.orderManager != nil && o.config.Features.LiveTradingEnabled {
		go func() {
			if err := o.orderManager.MonitorOrders(ctx); err != nil {
				o.logger.WithError(err).Error("Order manager stopped")
			}
		}()
	}

	// Start performance monitor
	go func() {
		if err := o.monitor.Start(ctx); err != nil {
			o.logger.WithError(err).Error("Performance monitor stopped")
		}
	}()

	// Update risk metrics initially
	if err := o.riskManager.UpdateExposure(ctx); err != nil {
		o.logger.WithError(err).Warn("Failed to update initial exposure")
	}
	if err := o.riskManager.UpdateDailyLoss(ctx); err != nil {
		o.logger.WithError(err).Warn("Failed to update initial daily loss")
	}

	// Start trading loop in goroutine
	go o.tradingLoop(ctx)

	o.logger.Info("Bot orchestrator started successfully")

	return nil
}

// Stop gracefully stops all bot components
func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return nil
	}
	o.running = false
	o.mu.Unlock()

	o.logger.Info("Stopping bot orchestrator")

	// Signal trading loop to stop
	close(o.done)

	// Stop monitor
	if err := o.monitor.Stop(); err != nil {
		o.logger.WithError(err).Error("Failed to stop monitor")
	}

	// Stop order manager
	if o.orderManager != nil {
		if err := o.orderManager.Stop(); err != nil {
			o.logger.WithError(err).Error("Failed to stop order manager")
		}
	}

	o.logger.Info("Bot orchestrator stopped")

	return nil
}

// tradingLoop main trading loop that evaluates strategies and executes signals
func (o *Orchestrator) tradingLoop(ctx context.Context) {
	evaluationInterval := time.Duration(o.config.Trading.StrategyEvaluationInterval) * time.Second
	ticker := time.NewTicker(evaluationInterval)
	defer ticker.Stop()

	o.logger.WithField("evaluation_interval", evaluationInterval).Info("Trading loop started")

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Trading loop stopped by context")
			return

		case <-o.done:
			o.logger.Info("Trading loop stopped")
			return

		case <-ticker.C:
			// Check circuit breaker
			if o.circuitBreaker.IsOpen() {
				o.logger.Warn("Trading halted: circuit breaker is open")
				continue
			}

			// Update risk metrics
			if err := o.riskManager.UpdateExposure(ctx); err != nil {
				o.logger.WithError(err).Error("Failed to update exposure")
				o.circuitBreaker.RecordFailure(err)
				continue
			}

			// Check risk limits
			if !o.riskManager.IsWithinLimits() {
				o.logger.Warn("Trading halted: risk limits exceeded")
				continue
			}

			// Get upcoming races
			now := time.Now()
			windowStart := now.Add(time.Duration(o.config.Trading.MinTimeToStartSeconds) * time.Second)
			windowEnd := now.Add(time.Duration(o.config.Trading.PreRaceWindowMinutes) * time.Minute)

			races, err := o.raceRepo.GetUpcomingRaces(ctx, windowStart, windowEnd)
			if err != nil {
				o.logger.WithError(err).Error("Failed to get upcoming races")
				o.circuitBreaker.RecordFailure(err)
				continue
			}

			o.logger.WithField("race_count", len(races)).Debug("Processing upcoming races")

			// Evaluate strategies for each race
			for _, race := range races {
				signals, err := o.evaluateStrategies(ctx, race)
				if err != nil {
					o.logger.WithFields(logrus.Fields{
						"race_id": race.ID,
						"error":   err.Error(),
					}).Error("Failed to evaluate strategies for race")
					continue
				}

				if len(signals) == 0 {
					continue
				}

				// Filter signals with ML predictions if enabled
				if o.config.Features.MLPredictionsEnabled {
					signals, err = o.filterSignalsWithML(ctx, signals)
					if err != nil {
						o.logger.WithError(err).Warn("Failed to filter signals with ML")
						// Continue with unfiltered signals
					}
				}

				// Execute approved signals
				bets, err := o.executor.ExecuteBatch(ctx, signals)
				if err != nil {
					o.logger.WithError(err).Warn("Batch execution had errors")
				}

				o.logger.WithFields(logrus.Fields{
					"race_id":     race.ID,
					"signals":     len(signals),
					"bets_placed": len(bets),
				}).Info("Race evaluation completed")

				// Record success
				o.circuitBreaker.RecordSuccess()
			}
		}
	}
}

// evaluateStrategies evaluates all active strategies for a race
func (o *Orchestrator) evaluateStrategies(ctx context.Context, race *models.Race) ([]SignalWithContext, error) {
	o.mu.RLock()
	strategies := make(map[uuid.UUID]strategy.Strategy, len(o.activeStrategies))
	for id, strat := range o.activeStrategies {
		strategies[id] = strat
	}
	o.mu.RUnlock()

	signals := make([]SignalWithContext, 0)

	for strategyID, strat := range strategies {
		// Create strategy context
		stratCtx := &strategy.StrategyContext{
			RaceID:    race.ID,
			EventID:   race.EventID,
			StartTime: race.StartTime,
		}

		// Evaluate strategy
		stratSignals, err := strat.Evaluate(ctx, stratCtx)
		if err != nil {
			o.logger.WithFields(logrus.Fields{
				"strategy_id": strategyID,
				"race_id":     race.ID,
				"error":       err.Error(),
			}).Warn("Strategy evaluation failed")
			continue
		}

		// Wrap signals with context
		for _, sig := range stratSignals {
			signals = append(signals, SignalWithContext{
				Signal:      sig,
				StrategyID:  strategyID,
				RaceID:      race.ID,
				MarketID:    race.MarketID,
				SelectionID: sig.SelectionID,
			})
		}
	}

	return signals, nil
}

// filterSignalsWithML uses ML predictions to filter/rank signals
func (o *Orchestrator) filterSignalsWithML(ctx context.Context, signals []SignalWithContext) ([]SignalWithContext, error) {
	// TODO: Implement ML filtering logic
	// For now, return all signals
	return signals, nil
}

// loadActiveStrategies loads active strategies from database and instantiates them
func (o *Orchestrator) loadActiveStrategies(ctx context.Context) error {
	strategies, err := o.strategyRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get strategies: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.activeStrategies = make(map[uuid.UUID]strategy.Strategy)

	for _, stratModel := range strategies {
		if !stratModel.IsActive {
			continue
		}

		// Instantiate strategy based on type
		var strat strategy.Strategy
		switch stratModel.Type {
		case "simple_value":
			strat = strategy.NewSimpleValueStrategy(o.logger)
		default:
			o.logger.WithField("strategy_type", stratModel.Type).Warn("Unknown strategy type, skipping")
			continue
		}

		o.activeStrategies[stratModel.ID] = strat

		o.logger.WithFields(logrus.Fields{
			"strategy_id":   stratModel.ID,
			"strategy_name": stratModel.Name,
			"strategy_type": stratModel.Type,
		}).Info("Active strategy loaded")
	}

	return nil
}

// GetStatus returns current orchestrator status
func (o *Orchestrator) GetStatus() *OrchestratorStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return &OrchestratorStatus{
		Running:             o.running,
		PaperTradingMode:    o.config.Features.PaperTradingEnabled,
		ActiveStrategies:    len(o.activeStrategies),
		CircuitBreakerState: o.circuitBreaker.GetState(),
		RiskMetrics:         o.riskManager.GetRiskMetrics(),
		MonitorMetrics:      *o.monitor.metrics,
		ExecutorMetrics:     o.executor.GetMetrics(),
		LastUpdate:          time.Now(),
	}
}
