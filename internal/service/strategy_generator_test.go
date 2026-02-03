package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/clever-better/internal/models"
)

func TestAggregateMLFeatures(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	svc := &StrategyGeneratorService{
		logger: logger,
	}

	// Create test backtest results with ML features
	features1 := map[string]float64{
		"market_volatility": 0.15,
		"liquidity":         0.85,
		"trend_strength":    0.65,
	}
	features1JSON, _ := json.Marshal(features1)

	features2 := map[string]float64{
		"market_volatility": 0.25,
		"liquidity":         0.75,
		"trend_strength":    0.55,
	}
	features2JSON, _ := json.Marshal(features2)

	results := []*models.BacktestResult{
		{
			ID:             uuid.New(),
			StrategyID:     uuid.New(),
			CompositeScore: 0.75,
			SharpeRatio:    1.5,
			WinRate:        0.65,
			MLFeatures:     features1JSON,
			CreatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			StrategyID:     uuid.New(),
			CompositeScore: 0.80,
			SharpeRatio:    1.8,
			WinRate:        0.70,
			MLFeatures:     features2JSON,
			CreatedAt:      time.Now(),
		},
	}

	aggregated, err := svc.aggregateMLFeatures(results)
	if err != nil {
		t.Fatalf("Failed to aggregate features: %v", err)
	}

	// Verify aggregated features exist
	expectedKeys := []string{
		"market_volatility_mean", "market_volatility_std", "market_volatility_min", "market_volatility_max",
		"liquidity_mean", "liquidity_std", "liquidity_min", "liquidity_max",
		"trend_strength_mean", "trend_strength_std", "trend_strength_min", "trend_strength_max",
	}

	for _, key := range expectedKeys {
		if _, exists := aggregated[key]; !exists {
			t.Errorf("Expected aggregated feature %s not found", key)
		}
	}

	// Verify mean calculations
	if mean := aggregated["market_volatility_mean"]; mean != 0.20 {
		t.Errorf("Expected market_volatility_mean=0.20, got %f", mean)
	}

	if mean := aggregated["liquidity_mean"]; mean != 0.80 {
		t.Errorf("Expected liquidity_mean=0.80, got %f", mean)
	}

	t.Logf("Successfully aggregated %d features from %d results", len(aggregated), len(results))
}

func TestExtractTopMetrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	svc := &StrategyGeneratorService{
		logger: logger,
	}

	results := []*models.BacktestResult{
		{
			ID:             uuid.New(),
			CompositeScore: 0.75,
			SharpeRatio:    1.5,
			TotalReturn:    0.25,
			WinRate:        0.65,
			MaxDrawdown:    0.12,
			ProfitFactor:   2.1,
		},
		{
			ID:             uuid.New(),
			CompositeScore: 0.85,
			SharpeRatio:    1.8,
			TotalReturn:    0.35,
			WinRate:        0.70,
			MaxDrawdown:    0.10,
			ProfitFactor:   2.5,
		},
		{
			ID:             uuid.New(),
			CompositeScore: 0.80,
			SharpeRatio:    1.6,
			TotalReturn:    0.30,
			WinRate:        0.68,
			MaxDrawdown:    0.11,
			ProfitFactor:   2.3,
		},
	}

	metrics := svc.extractTopMetrics(results)

	// Verify top metrics (should be from result with highest composite score = 0.85)
	if topComposite := metrics["top_composite_score"]; topComposite != 0.85 {
		t.Errorf("Expected top_composite_score=0.85, got %f", topComposite)
	}

	if topSharpe := metrics["top_sharpe_ratio"]; topSharpe != 1.8 {
		t.Errorf("Expected top_sharpe_ratio=1.8, got %f", topSharpe)
	}

	if topROI := metrics["top_roi"]; topROI != 0.35 {
		t.Errorf("Expected top_roi=0.35, got %f", topROI)
	}

	// Verify averages
	expectedAvgSharpe := (1.5 + 1.8 + 1.6) / 3.0
	if avgSharpe := metrics["avg_sharpe_ratio"]; avgSharpe != expectedAvgSharpe {
		t.Errorf("Expected avg_sharpe_ratio=%f, got %f", expectedAvgSharpe, avgSharpe)
	}

	// Verify all required metrics exist
	requiredMetrics := []string{
		"top_composite_score", "top_sharpe_ratio", "top_roi", "top_win_rate",
		"avg_sharpe_ratio", "avg_roi", "avg_win_rate", "avg_composite_score",
		"std_sharpe_ratio", "std_roi",
	}

	for _, key := range requiredMetrics {
		if _, exists := metrics[key]; !exists {
			t.Errorf("Expected metric %s not found", key)
		}
	}

	t.Logf("Successfully extracted %d metrics from %d results", len(metrics), len(results))
}

func TestEmptyBacktestResults(t *testing.T) {
	logger := logrus.New()
	svc := &StrategyGeneratorService{
		logger: logger,
	}

	// Test with empty results
	aggregated, err := svc.aggregateMLFeatures([]*models.BacktestResult{})
	if err != nil {
		t.Fatalf("Should not error on empty results: %v", err)
	}
	if len(aggregated) != 0 {
		t.Errorf("Expected empty aggregated features, got %d", len(aggregated))
	}

	metrics := svc.extractTopMetrics([]*models.BacktestResult{})
	if len(metrics) != 0 {
		t.Errorf("Expected empty metrics, got %d", len(metrics))
	}
}
