package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/yourusername/clever-better/internal/models"
)

// MonteCarloConfig configures monte carlo simulation
type MonteCarloConfig struct {
	Iterations       int
	ConfidenceLevel  float64
	Seed             int64
	CommissionRate   float64
	InitialBankroll  float64
}

// MonteCarloResult represents monte carlo outcomes
type MonteCarloResult struct {
	Iterations           int              `json:"iterations"`
	MeanReturn           float64          `json:"mean_return"`
	StdReturn            float64          `json:"std_return"`
	VaR95                float64          `json:"var_95"`
	VaR99                float64          `json:"var_99"`
	ProbabilityOfProfit  float64          `json:"probability_of_profit"`
	ProbabilityOfRuin    float64          `json:"probability_of_ruin"`
	ConfidenceIntervals  map[string]float64 `json:"confidence_intervals"`
	Distribution         []float64        `json:"distribution"`
}

// RunMonteCarlo runs monte carlo simulation for bet outcomes
func RunMonteCarlo(ctx context.Context, bets []*models.Bet, probabilities map[string]float64, cfg MonteCarloConfig) (MonteCarloResult, error) {
	_ = ctx
	if cfg.Iterations <= 0 {
		cfg.Iterations = 1000
	}
	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	rng := rand.New(rand.NewSource(seed))
	distribution := make([]float64, cfg.Iterations)

	for i := 0; i < cfg.Iterations; i++ {
		bankroll := cfg.InitialBankroll
		for _, bet := range bets {
			prob := probabilities[bet.ID.String()]
			if prob <= 0 {
				prob = 0.5
			}
			win := rng.Float64() < prob
			pnl := calculatePnL(bet, win)
			if pnl > 0 && cfg.CommissionRate > 0 {
				pnl -= pnl * cfg.CommissionRate
			}
			bankroll += pnl
			if bankroll <= 0 {
				bankroll = 0
				break
			}
		}
		distribution[i] = bankroll
	}

	mean, std := meanStd(distribution)
	var95 := percentile(distribution, 0.05)
	var99 := percentile(distribution, 0.01)
	profitProb := probabilityAbove(distribution, cfg.InitialBankroll)
	ruinProb := probabilityAtOrBelow(distribution, 0)

	result := MonteCarloResult{
		Iterations:          cfg.Iterations,
		MeanReturn:          (mean - cfg.InitialBankroll) / cfg.InitialBankroll,
		StdReturn:           std / cfg.InitialBankroll,
		VaR95:               (var95 - cfg.InitialBankroll) / cfg.InitialBankroll,
		VaR99:               (var99 - cfg.InitialBankroll) / cfg.InitialBankroll,
		ProbabilityOfProfit: profitProb,
		ProbabilityOfRuin:   ruinProb,
		ConfidenceIntervals: CalculateConfidenceIntervals(distribution, []float64{0.9, 0.95, 0.99}),
		Distribution:        distribution,
	}

	return result, nil
}

// CalculateConfidenceIntervals computes confidence intervals for distribution
func CalculateConfidenceIntervals(distribution []float64, levels []float64) map[string]float64 {
	results := make(map[string]float64)
	for _, level := range levels {
		p := (1.0 - level) / 2.0
		low := percentile(distribution, p)
		high := percentile(distribution, 1.0-p)
		results[formatPercent(level)] = high - low
	}
	return results
}

// ExportForML exports monte carlo result for ML consumption
func (m MonteCarloResult) ExportForML() string {
	data, _ := json.Marshal(m)
	return string(data)
}

func meanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	valuesCopy := append([]float64{}, values...)
	sortFloats(valuesCopy)
	idx := int(math.Floor(p * float64(len(valuesCopy)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(valuesCopy) {
		idx = len(valuesCopy) - 1
	}
	return valuesCopy[idx]
}

func probabilityAbove(values []float64, threshold float64) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0
	for _, v := range values {
		if v > threshold {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

func probabilityAtOrBelow(values []float64, threshold float64) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0
	for _, v := range values {
		if v <= threshold {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

func formatPercent(level float64) string {
	return fmt.Sprintf("%.0f%%", level*100)
}
