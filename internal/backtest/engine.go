package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

// Engine orchestrates backtesting runs
type Engine struct {
	config       BacktestConfig
	db           *database.DB
	repositories *repository.Repositories
	strategy     strategy.Strategy
	logger       *logrus.Logger
}

// NewEngine creates a new backtesting engine
func NewEngine(cfg BacktestConfig, db *database.DB, strat strategy.Strategy, logger *logrus.Logger) (*Engine, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if strat == nil {
		return nil, fmt.Errorf("strategy is required")
	}
	if logger == nil {
		logger = logrus.New()
	}
	repos, err := repository.NewRepositories(db)
	if err != nil {
		return nil, err
	}

	return &Engine{
		config:       cfg,
		db:           db,
		repositories: repos,
		strategy:     strat,
		logger:       logger,
	}, nil
}

// Config returns the backtest configuration
func (e *Engine) Config() BacktestConfig {
	return e.config
}

// Logger returns the engine logger
func (e *Engine) Logger() *logrus.Logger {
	return e.logger
}

// Repositories returns the repository container
func (e *Engine) Repositories() *repository.Repositories {
	return e.repositories
}

// Close releases engine resources
func (e *Engine) Close(ctx context.Context) error {
	if e.db == nil {
		return nil
	}
	return e.db.Close(ctx)
}

// Run orchestrates backtest execution
func (e *Engine) Run(ctx context.Context, startDate, endDate time.Time) (*BacktestState, Metrics, error) {
	e.logger.WithFields(logrus.Fields{"start": startDate, "end": endDate}).Info("Starting backtest run")
	state, err := e.HistoricalReplay(ctx, startDate, endDate)
	if err != nil {
		return nil, Metrics{}, err
	}
	metrics := CalculateMetrics(state, e.config)
	return state, metrics, nil
}

// HistoricalReplay replays historical races and simulates betting
func (e *Engine) HistoricalReplay(ctx context.Context, startDate, endDate time.Time) (*BacktestState, error) {
	state := NewBacktestState(e.config.InitialBankroll)

	races, err := e.repositories.Race.GetByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load races: %w", err)
	}

	for _, race := range races {
		if err := e.processRace(ctx, race, startDate, state); err != nil {
			return nil, err
		}
	}

	return state, nil
}

func (e *Engine) processRace(ctx context.Context, race *models.Race, startDate time.Time, state *BacktestState) error {
	runners, err := e.repositories.Runner.GetByRaceID(ctx, race.ID)
	if err != nil {
		return fmt.Errorf("failed to load runners: %w", err)
	}

	oddsSnapshots, err := e.repositories.Odds.GetByRaceID(ctx, race.ID, startDate, race.ScheduledStart)
	if err != nil {
		return fmt.Errorf("failed to load odds: %w", err)
	}

	decisionTime := race.ScheduledStart
	filteredOdds := filterOddsByTime(oddsSnapshots, decisionTime)
	strategyCtx := strategy.Context{
		Race:        race,
		Runners:     runners,
		OddsHistory: filteredOdds,
		CurrentTime: decisionTime,
	}

	signals, err := e.strategy.Evaluate(ctx, strategyCtx)
	if err != nil {
		return fmt.Errorf("strategy evaluation failed: %w", err)
	}

	result, err := e.repositories.RaceResult.GetByRaceID(ctx, race.ID)
	if err != nil {
		return fmt.Errorf("failed to load race result: %w", err)
	}

	runnerByID := make(map[uuid.UUID]*models.Runner)
	for _, runner := range runners {
		runnerByID[runner.ID] = runner
	}

	for _, signal := range signals {
		if !e.strategy.ShouldBet(signal) {
			continue
		}
		stake := e.strategy.CalculateStake(signal, state.CurrentBankroll)
		if stake <= 0 {
			continue
		}
		adjusted := signal
		adjusted.Stake = stake

		bet := e.SimulateBetExecution(adjusted, filteredOdds)
		if bet == nil {
			continue
		}
		bet.RaceID = race.ID

		runner := runnerByID[signal.RunnerID]
		pnl := e.SettleBet(bet, result, runner, e.config.CommissionRate)
		state.UpdateState(bet, pnl)
		if bet.SettledAt != nil {
			state.RecordEquityPoint(bet.SettledAt.UTC(), state.CurrentBankroll)
		}
	}

	return nil
}

func filterOddsByTime(odds []*models.OddsSnapshot, cutoff time.Time) []*models.OddsSnapshot {
	filtered := make([]*models.OddsSnapshot, 0, len(odds))
	for _, snapshot := range odds {
		if snapshot.Time.After(cutoff) {
			continue
		}
		filtered = append(filtered, snapshot)
	}
	return filtered
}

// SimulateBetExecution simulates execution with slippage and transaction costs
func (e *Engine) SimulateBetExecution(signal strategy.Signal, oddsHistory []*models.OddsSnapshot) *models.Bet {
	_ = oddsHistory
	if signal.Stake <= 0 || signal.Odds <= 1 {
		return nil
	}

	odds := applySlippage(signal.Odds, signal.Side, e.config.SlippageTicks)
	betID := uuid.New()
	now := time.Now().UTC()

	bet := &models.Bet{
		ID:         betID,
		RaceID:     uuid.Nil,
		RunnerID:   signal.RunnerID,
		StrategyID: uuid.Nil,
		MarketType: models.MarketTypeWin,
		Side:       signal.Side,
		Odds:       odds,
		Stake:      signal.Stake,
		Status:     models.BetStatusMatched,
		PlacedAt:   now,
		MatchedAt:  &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return bet
}

// SettleBet settles a bet against race results and returns PnL
func (e *Engine) SettleBet(bet *models.Bet, result *models.RaceResult, runner *models.Runner, commissionRate float64) float64 {
	if bet == nil || result == nil {
		return 0
	}
	win := isRunnerWinner(runner, result)
	pnl := calculatePnL(bet, win)
	commission := 0.0
	if pnl > 0 && commissionRate > 0 {
		commission = pnl * commissionRate
		pnl -= commission
	}

	settledAt := result.Time
	bet.Status = models.BetStatusSettled
	bet.SettledAt = &settledAt
	bet.ProfitLoss = &pnl
	bet.Commission = &commission
	bet.UpdatedAt = time.Now().UTC()

	return pnl
}

func calculatePnL(bet *models.Bet, win bool) float64 {
	if bet.Side == models.BetSideBack {
		if win {
			return (bet.Odds - 1.0) * bet.Stake
		}
		return -bet.Stake
	}

	// Lay bet
	if win {
		return -(bet.Odds-1.0) * bet.Stake
	}
	return bet.Stake
}

func isRunnerWinner(runner *models.Runner, result *models.RaceResult) bool {
	if result == nil || runner == nil {
		return false
	}
	if result.WinnerTrap != nil {
		return runner.TrapNumber == *result.WinnerTrap
	}
	positions, err := result.ParsePositions()
	if err != nil {
		return false
	}
	for _, entry := range positions.Runners {
		if entry.RunnerID == runner.ID && entry.Position == 1 {
			return true
		}
	}
	return false
}

func applySlippage(odds float64, side models.BetSide, ticks int) float64 {
	if ticks <= 0 {
		return odds
	}
	adjustment := 0.01 * float64(ticks)
	if side == models.BetSideBack {
		return math.Max(1.01, odds+adjustment)
	}
	return math.Max(1.01, odds-adjustment)
}
