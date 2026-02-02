package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// SimpleValueStrategy implements a basic value betting strategy
// Strategy logic: bet when (model_probability * odds) - 1 > min_edge_threshold
type SimpleValueStrategy struct {
	BaseStrategy
	NameValue         string
	MinEdgeThreshold  float64
	MinConfidence     float64
	DefaultStake      float64
}

// NewSimpleValueStrategy creates a new simple value strategy
func NewSimpleValueStrategy() *SimpleValueStrategy {
	return &SimpleValueStrategy{
		BaseStrategy: BaseStrategy{
			MinOdds:          1.01,
			MaxOdds:          1000,
			MinLiquidity:     5,
			KellyFraction:    0.5,
			MinEdgeThreshold: 0.02,
		},
		NameValue:        "simple_value",
		MinEdgeThreshold: 0.02,
		MinConfidence:    0.55,
		DefaultStake:     5,
	}
}

// Name returns strategy name
func (s *SimpleValueStrategy) Name() string {
	return s.NameValue
}

// Evaluate evaluates race data and generates betting signals
func (s *SimpleValueStrategy) Evaluate(ctx context.Context, strategyCtx Context) ([]Signal, error) {
	_ = ctx
	if strategyCtx.Race == nil {
		return nil, fmt.Errorf("race is required")
	}

	currentTime := strategyCtx.CurrentTime
	if err := s.ValidateTemporalSafety(currentTime, strategyCtx.OddsHistory); err != nil {
		return nil, err
	}

	latestOdds := latestOddsByRunner(strategyCtx.OddsHistory, currentTime)
	var signals []Signal

	for _, runner := range strategyCtx.Runners {
		signal, ok := s.buildSignal(runner, latestOdds)
		if !ok {
			continue
		}
		if s.ShouldBet(signal) {
			signals = append(signals, signal)
		}
	}

	return signals, nil
}

// ShouldBet determines if a signal should be executed
func (s *SimpleValueStrategy) ShouldBet(signal Signal) bool {
	return signal.ExpectedValue > 0 && signal.Stake > 0
}

// CalculateStake calculates stake based on bankroll
func (s *SimpleValueStrategy) CalculateStake(signal Signal, bankroll float64) float64 {
	if bankroll <= 0 {
		return 0
	}
	stake := signal.Stake
	if stake <= 0 {
		stake = s.DefaultStake
	}
	if stake > bankroll {
		return bankroll
	}
	return stake
}

// GetParameters returns strategy parameters for ML export
func (s *SimpleValueStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"min_edge_threshold": s.MinEdgeThreshold,
		"min_confidence":     s.MinConfidence,
		"default_stake":      s.DefaultStake,
	}
}

func (s *SimpleValueStrategy) buildSignal(runner *models.Runner, latestOdds map[uuid.UUID]*models.OddsSnapshot) (Signal, bool) {
	snapshot, ok := latestOdds[runner.ID]
	if !ok {
		return Signal{}, false
	}
	if !s.CheckLiquidity(snapshot) {
		return Signal{}, false
	}
	odds := snapshot.GetMidPrice()
	if err := s.ValidateOdds(odds); err != nil {
		return Signal{}, false
	}

	modelProbability := s.NormalizeProbability(s.estimateProbability(runner, odds))
	edge := (modelProbability * odds) - 1.0
	if edge <= s.MinEdgeThreshold || modelProbability < s.MinConfidence {
		return Signal{}, false
	}

	stake := s.DefaultStake
	if stake <= 0 {
		stake = s.ApplyKellyCriterion(modelProbability, odds, 100)
	}

	signal := Signal{
		RunnerID:      runner.ID,
		Side:          models.BetSideBack,
		Odds:          odds,
		Stake:         stake,
		Confidence:    modelProbability,
		ExpectedValue: s.CalculateExpectedValue(modelProbability, odds, stake),
		Reasoning:     "Value edge exceeds threshold",
		Features: map[string]any{
			"edge":              edge,
			"model_probability": modelProbability,
			"runner_name":       runner.Name,
		},
	}
	return signal, true
}

func (s *SimpleValueStrategy) estimateProbability(runner *models.Runner, odds float64) float64 {
	implied := 0.0
	if odds > 0 {
		implied = 1.0 / odds
	}
	formBoost := 0.0
	if runner.FormRating != nil {
		formBoost = *runner.FormRating * 0.01
	}
	return implied + formBoost
}

func latestOddsByRunner(oddsHistory []*models.OddsSnapshot, current time.Time) map[uuid.UUID]*models.OddsSnapshot {
	latest := make(map[uuid.UUID]*models.OddsSnapshot)
	for _, snapshot := range oddsHistory {
		if snapshot.Time.After(current) {
			continue
		}
		existing, ok := latest[snapshot.RunnerID]
		if !ok || snapshot.Time.After(existing.Time) {
			latest[snapshot.RunnerID] = snapshot
		}
	}
	return latest
}
