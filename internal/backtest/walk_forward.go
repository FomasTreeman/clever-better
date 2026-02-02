package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourusername/clever-better/internal/strategy"
)

// WalkForwardConfig configures walk-forward optimization
type WalkForwardConfig struct {
	TrainingWindowDays   int
	ValidationWindowDays int
	TestWindowDays       int
	StepSizeDays         int
	MinTradesPerWindow   int
}

// WalkForwardWindow represents one walk-forward window
type WalkForwardWindow struct {
	WindowID   int
	TrainStart time.Time
	TrainEnd   time.Time
	ValStart   time.Time
	ValEnd     time.Time
	TestStart  time.Time
	TestEnd    time.Time
	TrainMetrics Metrics
	ValMetrics   Metrics
	TestMetrics  Metrics
}

// WalkForwardResult represents walk-forward optimization result
type WalkForwardResult struct {
	Windows           []WalkForwardWindow `json:"windows"`
	AggregatedMetrics Metrics             `json:"aggregated_metrics"`
	ConsistencyScore  float64             `json:"consistency_score"`
	OverfitScore      float64             `json:"overfit_score"`
}

// RunWalkForward performs walk-forward optimization
func RunWalkForward(ctx context.Context, engine *Engine, strat strategy.Strategy, cfg WalkForwardConfig) (WalkForwardResult, error) {
	if engine == nil {
		return WalkForwardResult{}, fmt.Errorf("engine is required")
	}
	_ = strat
	if cfg.StepSizeDays <= 0 {
		cfg.StepSizeDays = cfg.TestWindowDays
	}

	start := engine.config.StartDate
	end := engine.config.EndDate
	windows := []WalkForwardWindow{}
	windowID := 0

	for current := start; current.Before(end); current = current.AddDate(0, 0, cfg.StepSizeDays) {
		trainStart := current
		trainEnd := trainStart.AddDate(0, 0, cfg.TrainingWindowDays)
		valStart := trainEnd
		valEnd := valStart.AddDate(0, 0, cfg.ValidationWindowDays)
		testStart := valEnd
		testEnd := testStart.AddDate(0, 0, cfg.TestWindowDays)
		if testStart.After(end) {
			break
		}
		if testEnd.After(end) {
			testEnd = end
		}

		windowID++
		trainState, trainMetrics, err := engine.Run(ctx, trainStart, trainEnd)
		if err != nil {
			return WalkForwardResult{}, err
		}
		valState, valMetrics, err := engine.Run(ctx, valStart, valEnd)
		if err != nil {
			return WalkForwardResult{}, err
		}
		testState, testMetrics, err := engine.Run(ctx, testStart, testEnd)
		if err != nil {
			return WalkForwardResult{}, err
		}
		if !meetsTradeThreshold(cfg.MinTradesPerWindow, trainState, valState, testState) {
			continue
		}

		window := WalkForwardWindow{
			WindowID:     windowID,
			TrainStart:   trainStart,
			TrainEnd:     trainEnd,
			ValStart:     valStart,
			ValEnd:       valEnd,
			TestStart:    testStart,
			TestEnd:      testEnd,
			TrainMetrics: trainMetrics,
			ValMetrics:   valMetrics,
			TestMetrics:  testMetrics,
		}
		windows = append(windows, window)
	}

	aggregated := aggregateWalkForward(windows)
	consistency := CalculateConsistency(windows)
	overfit := calculateOverfitScore(windows)

	return WalkForwardResult{
		Windows:           windows,
		AggregatedMetrics: aggregated,
		ConsistencyScore:  consistency,
		OverfitScore:      overfit,
	}, nil
}

func meetsTradeThreshold(minTrades int, train *BacktestState, val *BacktestState, test *BacktestState) bool {
	if minTrades <= 0 {
		return true
	}
	if test == nil || len(test.Bets) < minTrades {
		return false
	}
	if train != nil && len(train.Bets) < minTrades {
		return false
	}
	if val != nil && len(val.Bets) < minTrades {
		return false
	}
	return true
}

// CalculateConsistency calculates percentage of profitable windows
func CalculateConsistency(windows []WalkForwardWindow) float64 {
	if len(windows) == 0 {
		return 0
	}
	profitable := 0
	for _, w := range windows {
		if w.TestMetrics.TotalReturn > 0 {
			profitable++
		}
	}
	return float64(profitable) / float64(len(windows))
}

func calculateOverfitScore(windows []WalkForwardWindow) float64 {
	if len(windows) == 0 {
		return 0
	}
	trainReturn := 0.0
	testReturn := 0.0
	for _, w := range windows {
		trainReturn += w.TrainMetrics.TotalReturn
		testReturn += w.TestMetrics.TotalReturn
	}
	if trainReturn == 0 {
		return 0
	}
	return (trainReturn - testReturn) / trainReturn
}

func aggregateWalkForward(windows []WalkForwardWindow) Metrics {
	if len(windows) == 0 {
		return Metrics{}
	}
	metrics := Metrics{}
	for _, w := range windows {
		metrics.TotalReturn += w.TestMetrics.TotalReturn
		metrics.SharpeRatio += w.TestMetrics.SharpeRatio
		metrics.MaxDrawdown += w.TestMetrics.MaxDrawdown
	}
	metrics.TotalReturn /= float64(len(windows))
	metrics.SharpeRatio /= float64(len(windows))
	metrics.MaxDrawdown /= float64(len(windows))
	return metrics
}

// ExportForML exports walk-forward result for ML consumption
func (w WalkForwardResult) ExportForML() string {
	data, _ := json.Marshal(w)
	return string(data)
}
