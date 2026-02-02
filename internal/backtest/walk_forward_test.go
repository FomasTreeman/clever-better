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

func TestRunWalkForward(t *testing.T) {
	engine := buildTestEngine()
	result, err := RunWalkForward(context.Background(), engine, testStrategy{}, WalkForwardConfig{
		TrainingWindowDays:   1,
		ValidationWindowDays: 1,
		TestWindowDays:       1,
		StepSizeDays:         1,
		MinTradesPerWindow:   1,
	})
	if err != nil {
		t.Fatalf("RunWalkForward failed: %v", err)
	}
	if len(result.Windows) == 0 {
		t.Fatalf("expected at least one window")
	}
}

func buildTestEngine() *Engine {
	raceID := uuid.New()
	runnerID := uuid.New()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now().Add(-24 * time.Hour)

	race := &models.Race{ID: raceID, ScheduledStart: end}
	runner := &models.Runner{ID: runnerID, RaceID: raceID, TrapNumber: 1, Name: "Runner"}
	odds := &models.OddsSnapshot{RaceID: raceID, RunnerID: runnerID, Time: start, BackPrice: floatPtr(3.0), LayPrice: floatPtr(3.2)}
	result := &models.RaceResult{RaceID: raceID, Time: end}

	return &Engine{
		config: BacktestConfig{StartDate: start, EndDate: end, InitialBankroll: 100, CommissionRate: 0.05},
		repositories: &repository.Repositories{
			Race:       &fakeRaceRepo{races: []*models.Race{race}},
			Runner:     &fakeRunnerRepo{runners: map[uuid.UUID][]*models.Runner{raceID: []*models.Runner{runner}}},
			Odds:       &fakeOddsRepo{odds: map[uuid.UUID][]*models.OddsSnapshot{raceID: []*models.OddsSnapshot{odds}}},
			RaceResult: &fakeRaceResultRepo{results: map[uuid.UUID]*models.RaceResult{raceID: result}},
		},
		strategy: strategy.NewSimpleValueStrategy(),
	}
}
