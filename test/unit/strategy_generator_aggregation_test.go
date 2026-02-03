// +build standalone

// Standalone test for strategy generator aggregation functions
// Run with: go test -tags standalone -v ./test/unit/strategy_generator_aggregation_test.go
package unit

import (
	"encoding/json"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// Mock implementation of aggregation logic for testing
func aggregateMLFeatures(results []*models.BacktestResult) (map[string]float64, error) {
	if len(results) == 0 {
		return make(map[string]float64), nil
	}

	allFeatures := make(map[string][]float64)

	for _, result := range results {
		if len(result.MLFeatures) == 0 {
			continue
		}

		var features map[string]float64
		if err := json.Unmarshal(result.MLFeatures, &features); err != nil {
			continue
		}

		for key, value := range features {
			allFeatures[key] = append(allFeatures[key], value)
		}
	}

	aggregated := make(map[string]float64)

	for key, values := range allFeatures {
		if len(values) == 0 {
			continue
		}

		sum := 0.0
		for _, v := range values {
			sum += v
		}
		mean := sum / float64(len(values))
		aggregated[key+"_mean"] = mean

		if len(values) > 1 {
			variance := 0.0
			for _, v := range values {
				diff := v - mean
				variance += diff * diff
			}
			stdDev := math.Sqrt(variance / float64(len(values)-1))
			aggregated[key+"_std"] = stdDev
		}

		min := values[0]
		max := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		aggregated[key+"_min"] = min
		aggregated[key+"_max"] = max
	}

	return aggregated, nil
}

func extractTopMetrics(results []*models.BacktestResult) map[string]float64 {
	if len(results) == 0 {
		return make(map[string]float64)
	}

	sortedResults := make([]*models.BacktestResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].CompositeScore > sortedResults[j].CompositeScore
	})

	top := sortedResults[0]

	metrics := map[string]float64{
		"top_composite_score": top.CompositeScore,
		"top_sharpe_ratio":    top.SharpeRatio,
		"top_roi":             top.TotalReturn,
		"top_win_rate":        top.WinRate,
		"top_max_drawdown":    top.MaxDrawdown,
		"top_profit_factor":   top.ProfitFactor,
	}

	avgSharpe := 0.0
	avgROI := 0.0
	avgWinRate := 0.0
	avgDrawdown := 0.0
	avgComposite := 0.0

	for _, result := range results {
		avgSharpe += result.SharpeRatio
		avgROI += result.TotalReturn
		avgWinRate += result.WinRate
		avgDrawdown += result.MaxDrawdown
		avgComposite += result.CompositeScore
	}

	n := float64(len(results))
	metrics["avg_sharpe_ratio"] = avgSharpe / n
	metrics["avg_roi"] = avgROI / n
	metrics["avg_win_rate"] = avgWinRate / n
	metrics["avg_max_drawdown"] = avgDrawdown / n
	metrics["avg_composite_score"] = avgComposite / n

	sharpeVariance := 0.0
	roiVariance := 0.0
	for _, result := range results {
		sharpeVariance += math.Pow(result.SharpeRatio-metrics["avg_sharpe_ratio"], 2)
		roiVariance += math.Pow(result.TotalReturn-metrics["avg_roi"], 2)
	}
	metrics["std_sharpe_ratio"] = math.Sqrt(sharpeVariance / n)
	metrics["std_roi"] = math.Sqrt(roiVariance / n)

	return metrics
}

func TestAggregateMLFeatures(t *testing.T) {
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

	aggregated, err := aggregateMLFeatures(results)
	if err != nil {
		t.Fatalf("Failed to aggregate features: %v", err)
	}

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

	if mean := aggregated["market_volatility_mean"]; mean != 0.20 {
		t.Errorf("Expected market_volatility_mean=0.20, got %f", mean)
	}

	if mean := aggregated["liquidity_mean"]; mean != 0.80 {
		t.Errorf("Expected liquidity_mean=0.80, got %f", mean)
	}

	t.Logf("✓ Successfully aggregated %d features from %d results", len(aggregated), len(results))
	t.Logf("✓ Sample features: market_volatility_mean=%.2f, liquidity_mean=%.2f",
		aggregated["market_volatility_mean"], aggregated["liquidity_mean"])
}

func TestExtractTopMetrics(t *testing.T) {
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

	metrics := extractTopMetrics(results)

	if topComposite := metrics["top_composite_score"]; topComposite != 0.85 {
		t.Errorf("Expected top_composite_score=0.85, got %f", topComposite)
	}

	if topSharpe := metrics["top_sharpe_ratio"]; topSharpe != 1.8 {
		t.Errorf("Expected top_sharpe_ratio=1.8, got %f", topSharpe)
	}

	if topROI := metrics["top_roi"]; topROI != 0.35 {
		t.Errorf("Expected top_roi=0.35, got %f", topROI)
	}

	expectedAvgSharpe := (1.5 + 1.8 + 1.6) / 3.0
	if avgSharpe := metrics["avg_sharpe_ratio"]; math.Abs(avgSharpe-expectedAvgSharpe) > 0.001 {
		t.Errorf("Expected avg_sharpe_ratio=%f, got %f", expectedAvgSharpe, avgSharpe)
	}

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

	t.Logf("✓ Successfully extracted %d metrics from %d results", len(metrics), len(results))
	t.Logf("✓ Top metrics: composite=%.2f, sharpe=%.2f, roi=%.2f",
		metrics["top_composite_score"], metrics["top_sharpe_ratio"], metrics["top_roi"])
	t.Logf("✓ Avg metrics: sharpe=%.2f, roi=%.2f",
		metrics["avg_sharpe_ratio"], metrics["avg_roi"])
}

func TestEmptyResults(t *testing.T) {
	aggregated, err := aggregateMLFeatures([]*models.BacktestResult{})
	if err != nil {
		t.Fatalf("Should not error on empty results: %v", err)
	}
	if len(aggregated) != 0 {
		t.Errorf("Expected empty aggregated features, got %d", len(aggregated))
	}

	metrics := extractTopMetrics([]*models.BacktestResult{})
	if len(metrics) != 0 {
		t.Errorf("Expected empty metrics, got %d", len(metrics))
	}

	t.Log("✓ Empty results handled correctly")
}
