package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

type testStrategy struct{}

func (t testStrategy) Name() string { return "test" }
func (t testStrategy) Evaluate(ctx context.Context, strategyCtx strategy.Context) ([]strategy.Signal, error) {
	_ = ctx
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
	if err != nil {
		t.Fatalf("HistoricalReplay failed: %v", err)
	}
	if len(state.Bets) == 0 {
		t.Fatalf("expected bets to be recorded")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
