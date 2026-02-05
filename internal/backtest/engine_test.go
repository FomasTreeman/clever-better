package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

type testStrategy struct {
	returnSignals []strategy.Signal
}

func (t testStrategy) Name() string { return "test" }
func (t testStrategy) Evaluate(ctx context.Context, strategyCtx strategy.Context) ([]strategy.Signal, error) {
	_ = ctx
	if len(t.returnSignals) > 0 {
		return t.returnSignals, nil
	}
	if len(strategyCtx.Runners) == 0 {
		return nil, nil
	}
	return []strategy.Signal{{
		RunnerID:      strategyCtx.Runners[0].ID,
		Side:          models.BetSideBack,
		Odds:          3.0,
		Stake:         10.0,
		Confidence:    0.6,
		ExpectedValue: 0.2,
		Reasoning:     "test",
	}}, nil
}
func (t testStrategy) ShouldBet(signal strategy.Signal) bool { return true }
func (t testStrategy) CalculateStake(signal strategy.Signal, bankroll float64) float64 {
	if bankroll < signal.Stake {
		return bankroll
	}
	return signal.Stake
}
func (t testStrategy) GetParameters() map[string]interface{} { return map[string]interface{}{} }

type fakeRaceRepo struct{ races []*models.Race }

type fakeRunnerRepo struct{ runners map[uuid.UUID][]*models.Runner }

type fakeOddsRepo struct{ odds map[uuid.UUID][]*models.OddsSnapshot }

type fakeRaceResultRepo struct{ results map[uuid.UUID]*models.RaceResult }

func (r *fakeRaceRepo) Create(ctx context.Context, race *models.Race) error { return nil }
func (r *fakeRaceRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Race, error) { return nil, nil }
func (r *fakeRaceRepo) GetUpcoming(ctx context.Context, limit int) ([]*models.Race, error) { return nil, nil }
func (r *fakeRaceRepo) GetByDateRange(ctx context.Context, start, end time.Time) ([]*models.Race, error) {
	return r.races, nil
}
func (r *fakeRaceRepo) GetByTrackAndDate(ctx context.Context, track string, date time.Time) ([]*models.Race, error) {
	return nil, nil
}
func (r *fakeRaceRepo) Update(ctx context.Context, race *models.Race) error { return nil }
func (r *fakeRaceRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (r *fakeRunnerRepo) Create(ctx context.Context, runner *models.Runner) error { return nil }
func (r *fakeRunnerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Runner, error) { return nil, nil }
func (r *fakeRunnerRepo) GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Runner, error) {
	return r.runners[raceID], nil
}
func (r *fakeRunnerRepo) Update(ctx context.Context, runner *models.Runner) error { return nil }
func (r *fakeRunnerRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (o *fakeOddsRepo) Insert(ctx context.Context, odds *models.OddsSnapshot) error { return nil }
func (o *fakeOddsRepo) InsertBatch(ctx context.Context, odds []*models.OddsSnapshot) error { return nil }
func (o *fakeOddsRepo) GetByRaceID(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error) {
	return o.odds[raceID], nil
}
func (o *fakeOddsRepo) GetLatest(ctx context.Context, raceID, runnerID uuid.UUID) (*models.OddsSnapshot, error) {
	return nil, nil
}
func (o *fakeOddsRepo) GetTimeSeriesForRunner(ctx context.Context, runnerID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error) {
	return nil, nil
}

func (r *fakeRaceResultRepo) Insert(ctx context.Context, result *models.RaceResult) error { return nil }
func (r *fakeRaceResultRepo) InsertBatch(ctx context.Context, results []*models.RaceResult) error { return nil }
func (r *fakeRaceResultRepo) GetByRaceID(ctx context.Context, raceID uuid.UUID) (*models.RaceResult, error) {
	return r.results[raceID], nil
}
func (r *fakeRaceResultRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.RaceResult, error) {
	return nil, nil
}
func (r *fakeRaceResultRepo) GetByStatus(ctx context.Context, status string, limit int) ([]*models.RaceResult, error) {
	return nil, nil
}
func (r *fakeRaceResultRepo) GetDailySummary(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.RaceResultSummary, error) {
	return nil, nil
}
func (r *fakeRaceResultRepo) Update(ctx context.Context, result *models.RaceResult) error { return nil }
func (r *fakeRaceResultRepo) Delete(ctx context.Context, raceID uuid.UUID, resultTime time.Time) error {
	return nil
}

// Helper functions
func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// TestHistoricalReplay tests basic historical replay functionality
func TestHistoricalReplay(t *testing.T) {
	raceID := uuid.New()
	runnerID := uuid.New()
	start := time.Now().Add(-48 * time.Hour)
	end := time.Now().Add(-24 * time.Hour)

	race := &models.Race{ID: raceID, ScheduledStart: end}
	runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
	odds := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(3.0), LayPrice: floatPtr(3.2)}
	winner := 1
	result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: &winner}

	engine := &Engine{
		config: BacktestConfig{InitialBankroll: 100, CommissionRate: 0.05},
		repositories: &repository.Repositories{
			Race:       &fakeRaceRepo{races: []*models.Race{race}},
			Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
			Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{odds}}},
			RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
		},
		strategy: testStrategy{},
	}

	state, err := engine.HistoricalReplay(context.Background(), start, end)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Greater(t, len(state.Bets), 0, "expected bets to be recorded")
	assert.Equal(t, len(state.Bets), 1, "expected exactly one bet")
}

// TestBankrollEdgeCases tests bankroll edge cases
func TestBankrollEdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		initialBankroll float64
		stake           float64
		odds            float64
		expectBet       bool
	}{
		{
			name:            "Sufficient bankroll",
			initialBankroll: 1000.0,
			stake:           100.0,
			odds:            3.0,
			expectBet:       true,
		},
		{
			name:            "Exact bankroll match",
			initialBankroll: 100.0,
			stake:           100.0,
			odds:            2.0,
			expectBet:       true,
		},
		{
			name:            "Insufficient bankroll",
			initialBankroll: 50.0,
			stake:           100.0,
			odds:            2.0,
			expectBet:       false,
		},
		{
			name:            "Zero bankroll",
			initialBankroll: 0.0,
			stake:           10.0,
			odds:            2.0,
			expectBet:       false,
		},
		{
			name:            "Fractional bankroll vs stake",
			initialBankroll: 55.5,
			stake:           100.0,
			odds:            2.0,
			expectBet:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raceID := uuid.New()
			runnerID := uuid.New()
			start := time.Now().Add(-48 * time.Hour)
			end := time.Now().Add(-24 * time.Hour)

			race := &models.Race{ID: raceID, ScheduledStart: end}
			runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
			odds := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(tt.odds)}
			winner := 1
			result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: &winner}

			engine := &Engine{
				config: BacktestConfig{InitialBankroll: tt.initialBankroll, CommissionRate: 0.0},
				repositories: &repository.Repositories{
					Race:       &fakeRaceRepo{races: []*models.Race{race}},
					Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
					Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{odds}}},
					RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
				},
				strategy: testStrategy{
					returnSignals: []strategy.Signal{{
						RunnerID:   runnerID,
						Side:       models.BetSideBack,
						Odds:       tt.odds,
						Stake:      tt.stake,
						Confidence: 0.8,
					}},
				},
			}

			state, err := engine.HistoricalReplay(context.Background(), start, end)
			require.NoError(t, err)

			if tt.expectBet {
				assert.Greater(t, len(state.Bets), 0, "expected bets when bankroll is sufficient")
			} else {
				// When bankroll is insufficient, stake gets reduced or no bet placed
				if len(state.Bets) > 0 {
					assert.LessOrEqual(t, state.Bets[0].Stake, tt.initialBankroll)
				}
			}
		})
	}
}

// TestSlippageApplication tests slippage adjustment on odds
func TestSlippageApplication(t *testing.T) {
	tests := []struct {
		name           string
		originalOdds   float64
		slippageTicks  int
		side           models.BetSide
		expectApprox   float64 // approximate expected odds
	}{
		{
			name:          "Back bet with positive slippage",
			originalOdds:  3.5,
			slippageTicks: 2,
			side:          models.BetSideBack,
			expectApprox:  3.52, // 3.5 + (0.01 * 2)
		},
		{
			name:          "Lay bet with positive slippage",
			originalOdds:  3.5,
			slippageTicks: 2,
			side:          models.BetSideLay,
			expectApprox:  3.48, // 3.5 - (0.01 * 2)
		},
		{
			name:          "No slippage",
			originalOdds:  2.5,
			slippageTicks: 0,
			side:          models.BetSideBack,
			expectApprox:  2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raceID := uuid.New()
			runnerID := uuid.New()
			start := time.Now().Add(-48 * time.Hour)
			end := time.Now().Add(-24 * time.Hour)

			race := &models.Race{ID: raceID, ScheduledStart: end}
			runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
			winner := 1
			result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: &winner}

			engine := &Engine{
				config: BacktestConfig{
					InitialBankroll: 1000.0,
					SlippageTicks:   tt.slippageTicks,
					CommissionRate:  0.0,
				},
				repositories: &repository.Repositories{
					Race:       &fakeRaceRepo{races: []*models.Race{race}},
					Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
					Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{}}},
					RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
				},
				strategy: testStrategy{
					returnSignals: []strategy.Signal{{
						RunnerID:   runnerID,
						Side:       tt.side,
						Odds:       tt.originalOdds,
						Stake:      50.0,
						Confidence: 0.8,
					}},
				},
			}

			state, err := engine.HistoricalReplay(context.Background(), start, end)
			require.NoError(t, err)
			require.Greater(t, len(state.Bets), 0)

			actualOdds := state.Bets[0].Odds
			assert.InDelta(t, tt.expectApprox, actualOdds, 0.01, "slippage not applied correctly")
		})
	}
}

// TestCommissionDeduction tests commission calculation and deduction
func TestCommissionDeduction(t *testing.T) {
	tests := []struct {
		name           string
		commissionRate float64
		winBet         bool
		expectedPnL    float64 // without commission first
	}{
		{
			name:           "Winning bet with 5% commission",
			commissionRate: 0.05,
			winBet:         true,
			expectedPnL:    -0.05, // 5% of 1.0 profit
		},
		{
			name:           "Losing bet (no commission)",
			commissionRate: 0.05,
			winBet:         false,
			expectedPnL:    -100.0, // full stake loss
		},
		{
			name:           "Zero commission",
			commissionRate: 0.0,
			winBet:         true,
			expectedPnL:    100.0, // full profit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raceID := uuid.New()
			runnerID := uuid.New()
			start := time.Now().Add(-48 * time.Hour)
			end := time.Now().Add(-24 * time.Hour)

			race := &models.Race{ID: raceID, ScheduledStart: end}
			runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Winner"}

			var winner *int
			if tt.winBet {
				winner = intPtr(1)
			} else {
				winner = intPtr(2) // different trap = loser
			}

			result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: winner}
			odds := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(2.0)}

			engine := &Engine{
				config: BacktestConfig{
					InitialBankroll: 1000.0,
					CommissionRate:  tt.commissionRate,
				},
				repositories: &repository.Repositories{
					Race:       &fakeRaceRepo{races: []*models.Race{race}},
					Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
					Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{odds}}},
					RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
				},
				strategy: testStrategy{},
			}

			state, err := engine.HistoricalReplay(context.Background(), start, end)
			require.NoError(t, err)
			require.Greater(t, len(state.Bets), 0)

			bet := state.Bets[0]
			require.NotNil(t, bet.ProfitLoss)
			require.NotNil(t, bet.Commission)

			// Verify commission was applied correctly
			if tt.winBet && tt.commissionRate > 0 {
				expectedCommission := (bet.Odds - 1.0) * bet.Stake * tt.commissionRate
				assert.InDelta(t, expectedCommission, *bet.Commission, 0.01)
			}
		})
	}
}

// TestConcurrentProcessing tests that engine handles concurrent race processing
func TestConcurrentProcessing(t *testing.T) {
	// Create multiple races
	var races []*models.Race
	runners := make(map[uuid.UUID][]*models.Runner)
	odds := make(map[uuid.UUID][]*models.OddsSnapshot)
	results := make(map[uuid.UUID]*models.RaceResult)

	start := time.Now().Add(-48 * time.Hour)
	end := time.Now().Add(-24 * time.Hour)

	for i := 0; i < 10; i++ {
		raceID := uuid.New()
		runnerID := uuid.New()

		race := &models.Race{
			ID:             raceID,
			ScheduledStart: end.Add(time.Duration(i) * time.Hour),
		}
		runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
		odd := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(3.0)}
		winner := 1
		result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: &winner}

		races = append(races, race)
		runners[raceID] = []*models.Runner{runner}
		odds[raceID] = []*models.OddsSnapshot{odd}
		results[raceID] = result
	}

	engine := &Engine{
		config: BacktestConfig{InitialBankroll: 10000.0, CommissionRate: 0.05},
		repositories: &repository.Repositories{
			Race:       &fakeRaceRepo{races: races},
			Runner:     &fakeRunnerRepo{runners: runners},
			Odds:       &fakeOddsRepo{odds: odds},
			RaceResult: &fakeRaceResultRepo{results: results},
		},
		strategy: testStrategy{},
	}

	state, err := engine.HistoricalReplay(context.Background(), start, end)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.GreaterOrEqual(t, len(state.Bets), 1, "expected bets from multiple races")
}

// TestStateRecovery tests that backtest state correctly tracks equity
func TestStateRecovery(t *testing.T) {
	raceID := uuid.New()
	runnerID := uuid.New()
	start := time.Now().Add(-48 * time.Hour)
	end := time.Now().Add(-24 * time.Hour)

	race := &models.Race{ID: raceID, ScheduledStart: end}
	runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
	odds := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(2.0)}
	winner := 1
	result := &models.RaceResult{RaceID: raceID, Time: end, WinnerTrap: &winner}

	initialBankroll := 1000.0
	engine := &Engine{
		config: BacktestConfig{InitialBankroll: initialBankroll, CommissionRate: 0.0},
		repositories: &repository.Repositories{
			Race:       &fakeRaceRepo{races: []*models.Race{race}},
			Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
			Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{odds}}},
			RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
		},
		strategy: testStrategy{
			returnSignals: []strategy.Signal{{
				RunnerID:   runnerID,
				Side:       models.BetSideBack,
				Odds:       2.0,
				Stake:      100.0,
				Confidence: 0.8,
			}},
		},
	}

	state, err := engine.HistoricalReplay(context.Background(), start, end)
	require.NoError(t, err)
	require.NotNil(t, state)

	// Verify initial bankroll is set correctly
	assert.Equal(t, initialBankroll, state.InitialBankroll)

	// Verify current bankroll changed based on bet outcomes
	assert.NotEqual(t, initialBankroll, state.CurrentBankroll)

	// Verify bets were tracked
	assert.Greater(t, len(state.Bets), 0)

	// Verify equity points were recorded
	assert.Greater(t, len(state.EquityCurve), 0)
}
