package backtest

import (
	"encoding/json"
	"math"
)

// AggregatedResult represents combined backtest outcomes
type AggregatedResult struct {
	StrategyID              string            `json:"strategy_id"`
	HistoricalReplayMetrics Metrics           `json:"historical_replay_metrics"`
	MonteCarloResult        MonteCarloResult  `json:"monte_carlo_result"`
	WalkForwardResult       WalkForwardResult `json:"walk_forward_result"`
	CompositeScore          float64           `json:"composite_score"`
	Weights                 AggregationWeights `json:"weights"`
	Recommendation          string            `json:"recommendation"`
	MLFeatures              map[string]float64 `json:"ml_features"`
}

// AggregationWeights define weighting per method
type AggregationWeights struct {
	HistoricalReplay float64 `json:"historical_replay"`
	MonteCarlo       float64 `json:"monte_carlo"`
	WalkForward      float64 `json:"walk_forward"`
}

// AggregateResults aggregates results with weights
func AggregateResults(historical Metrics, monteCarlo MonteCarloResult, walkForward WalkForwardResult, weights AggregationWeights) AggregatedResult {
	historicalScore := CalculateCompositeScore(historical, weights)
	monteCarloScore := normalize(monteCarlo.MeanReturn, -0.5, 1.0)
	walkForwardScore := normalize(walkForward.AggregatedMetrics.TotalReturn, -0.5, 1.0)
	composite := historicalScore*weights.HistoricalReplay + monteCarloScore*weights.MonteCarlo + walkForwardScore*weights.WalkForward
	features := extractFeatures(historical, monteCarlo, walkForward)
	recommendation := GenerateRecommendation(composite, walkForward.ConsistencyScore, historical.TotalReturn, walkForward.AggregatedMetrics.TotalReturn)

	return AggregatedResult{
		HistoricalReplayMetrics: historical,
		MonteCarloResult:        monteCarlo,
		WalkForwardResult:       walkForward,
		CompositeScore:          composite,
		Weights:                 weights,
		Recommendation:          recommendation,
		MLFeatures:              features,
	}
}

// CalculateCompositeScore calculates weighted score from metrics
func CalculateCompositeScore(metrics Metrics, weights AggregationWeights) float64 {
	sharpeScore := normalize(metrics.SharpeRatio, -2, 3)
	roiScore := normalize(metrics.TotalReturn, -0.5, 1.0)
	profitFactorScore := normalize(metrics.ProfitFactor, 0, 3)
	drawdownPenalty := 1.0 - normalize(metrics.MaxDrawdown, 0, 0.5)
	winRateScore := normalize(metrics.WinRate, 0, 1)

	weighted := 0.0
	weighted += sharpeScore * 0.30
	weighted += roiScore * 0.20
	weighted += profitFactorScore * 0.20
	weighted += drawdownPenalty * 0.15
	weighted += winRateScore * 0.15

	_ = weights
	return weighted
}

// GenerateRecommendation determines if strategy is acceptable
func GenerateRecommendation(score float64, consistency float64, historicalReturn float64, walkForwardReturn float64) string {
	if score > 0.7 && historicalReturn > 0 && walkForwardReturn > 0 && consistency > 0.6 {
		return "ACCEPT"
	}
	if score < 0.4 || historicalReturn < 0 || walkForwardReturn < 0 || consistency < 0.4 {
		return "REJECT"
	}
	return "NEEDS_REVIEW"
}

// ExportForML exports aggregated result for ML consumption
func (a AggregatedResult) ExportForML() string {
	data, _ := json.Marshal(a)
	return string(data)
}

func extractFeatures(h Metrics, mc MonteCarloResult, wf WalkForwardResult) map[string]float64 {
	return map[string]float64{
		"total_return":       h.TotalReturn,
		"sharpe_ratio":       h.SharpeRatio,
		"max_drawdown":       h.MaxDrawdown,
		"profit_factor":      h.ProfitFactor,
		"win_rate":           h.WinRate,
		"monte_carlo_var95":  mc.VaR95,
		"monte_carlo_var99":  mc.VaR99,
		"consistency_score":  wf.ConsistencyScore,
		"overfit_score":      wf.OverfitScore,
	}
}

func normalize(value, min, max float64) float64 {
	if max-min == 0 {
		return 0
	}
	v := (value - min) / (max - min)
	return math.Max(0, math.Min(1, v))
}
