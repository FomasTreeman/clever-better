package backtest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// Metrics represents backtest performance metrics
type Metrics struct {
	TotalReturn      float64   `json:"total_return"`
	AnnualizedReturn float64   `json:"annualized_return"`
	CAGR             float64   `json:"cagr"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	SharpeRatio      float64   `json:"sharpe_ratio"`
	SortinoRatio     float64   `json:"sortino_ratio"`
	CalmarRatio      float64   `json:"calmar_ratio"`
	ValueAtRisk95    float64   `json:"var_95"`
	ValueAtRisk99    float64   `json:"var_99"`
	TotalBets        int       `json:"total_bets"`
	WinningBets      int       `json:"winning_bets"`
	LosingBets       int       `json:"losing_bets"`
	WinRate          float64   `json:"win_rate"`
	ProfitFactor     float64   `json:"profit_factor"`
	AverageWin       float64   `json:"average_win"`
	AverageLoss      float64   `json:"average_loss"`
	Expectancy       float64   `json:"expectancy"`
	LargestWin       float64   `json:"largest_win"`
	LargestLoss      float64   `json:"largest_loss"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	TradingDays      int       `json:"trading_days"`
	StrategyID       uuid.UUID `json:"strategy_id"`
	ParameterHash    string    `json:"parameter_hash"`
	ValidationScore  float64   `json:"validation_score"`
}

// CalculateMetrics calculates metrics from backtest state
func CalculateMetrics(state *BacktestState, cfg BacktestConfig) Metrics {
	metrics := Metrics{
		StartDate:   cfg.StartDate,
		EndDate:     cfg.EndDate,
		TradingDays: int(cfg.EndDate.Sub(cfg.StartDate).Hours()/24) + 1,
	}

	if state == nil || len(state.EquityCurve) == 0 {
		return metrics
	}

	initial := state.EquityCurve[0].Value
	final := state.EquityCurve[len(state.EquityCurve)-1].Value
	if initial > 0 {
		metrics.TotalReturn = (final - initial) / initial
		metrics.CAGR = calculateCAGR(initial, final, metrics.TradingDays)
		metrics.AnnualizedReturn = metrics.CAGR
	}

	metrics.MaxDrawdown = calculateMaxDrawdown(state.EquityCurve)
	returns := state.EquityCurve.GetReturns()
	metrics.SharpeRatio = calculateSharpeRatio(returns, cfg.RiskFreeRate)
	metrics.SortinoRatio = calculateSortinoRatio(returns, cfg.RiskFreeRate)
	if metrics.MaxDrawdown > 0 {
		metrics.CalmarRatio = metrics.AnnualizedReturn / metrics.MaxDrawdown
	}
	metrics.ValueAtRisk95 = calculateVaR(returns, 0.95)
	metrics.ValueAtRisk99 = calculateVaR(returns, 0.99)

	metrics.TotalBets = len(state.Bets)
	metrics.WinningBets, metrics.LosingBets, metrics.AverageWin, metrics.AverageLoss, metrics.LargestWin, metrics.LargestLoss = calculateBetStats(state.Bets)
	metrics.WinRate = calculateWinRate(metrics.WinningBets, metrics.TotalBets)
	metrics.ProfitFactor = calculateProfitFactor(state.Bets)
	metrics.Expectancy = calculateExpectancy(state.Bets)

	return metrics
}

// ToJSON exports metrics to JSON
func (m Metrics) ToJSON() string {
	data, _ := json.Marshal(m)
	return string(data)
}

// ToDB converts metrics to StrategyPerformance for persistence
func (m Metrics) ToDB(strategyID uuid.UUID) *models.StrategyPerformance {
	perf := &models.StrategyPerformance{
		Time:        time.Now(),
		StrategyID:  strategyID,
		TotalBets:   m.TotalBets,
		WinningBets: m.WinningBets,
		LosingBets:  m.LosingBets,
		GrossProfit: calculateGrossProfit(m),
		GrossLoss:   calculateGrossLoss(m),
		NetProfit:   calculateNetProfit(m),
		ROI:         m.TotalReturn * 100,
	}

	sharpe := m.SharpeRatio
	perf.SharpeRatio = &sharpe
	maxDD := m.MaxDrawdown
	perf.MaxDrawdown = &maxDD
	return perf
}

func calculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	mean := average(returns)
	std := stddev(returns)
	if std == 0 {
		return 0
	}
	return (mean - riskFreeRate/252.0) / std * math.Sqrt(252)
}

func calculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	mean := average(returns)
	std := downsideStddev(returns)
	if std == 0 {
		return 0
	}
	return (mean - riskFreeRate/252.0) / std * math.Sqrt(252)
}

func calculateMaxDrawdown(curve EquityCurve) float64 {
	maxDD := 0.0
	peak := 0.0
	for _, p := range curve {
		if p.Value > peak {
			peak = p.Value
		}
		if peak == 0 {
			continue
		}
		drawdown := (peak - p.Value) / peak
		if drawdown > maxDD {
			maxDD = drawdown
		}
	}
	return maxDD
}

func calculateProfitFactor(bets []*models.Bet) float64 {
	grossProfit := 0.0
	grossLoss := 0.0
	for _, bet := range bets {
		if bet.ProfitLoss == nil {
			continue
		}
		if *bet.ProfitLoss > 0 {
			grossProfit += *bet.ProfitLoss
		} else {
			grossLoss += math.Abs(*bet.ProfitLoss)
		}
	}
	if grossLoss == 0 {
		if grossProfit > 0 {
			return 999
		}
		return 0
	}
	return grossProfit / grossLoss
}

func calculateExpectancy(bets []*models.Bet) float64 {
	if len(bets) == 0 {
		return 0
	}
	net := 0.0
	for _, bet := range bets {
		if bet.ProfitLoss != nil {
			net += *bet.ProfitLoss
		}
	}
	return net / float64(len(bets))
}

func calculateCAGR(initial, final float64, days int) float64 {
	if initial <= 0 || days <= 0 {
		return 0
	}
	years := float64(days) / 365.0
	return math.Pow(final/initial, 1.0/years) - 1.0
}

func calculateVaR(returns []float64, level float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	sorted := append([]float64{}, returns...)
	sortFloats(sorted)
	index := int(math.Floor((1.0-level)*float64(len(sorted))))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func calculateBetStats(bets []*models.Bet) (int, int, float64, float64, float64, float64) {
	wins := 0
	losses := 0
	winSum := 0.0
	lossSum := 0.0
	largestWin := 0.0
	largestLoss := 0.0
	for _, bet := range bets {
		if bet.ProfitLoss == nil {
			continue
		}
		pl := *bet.ProfitLoss
		if pl > 0 {
			wins++
			winSum += pl
			if pl > largestWin {
				largestWin = pl
			}
		} else if pl < 0 {
			losses++
			lossSum += pl
			if pl < largestLoss {
				largestLoss = pl
			}
		}
	}

	avgWin := 0.0
	avgLoss := 0.0
	if wins > 0 {
		avgWin = winSum / float64(wins)
	}
	if losses > 0 {
		avgLoss = lossSum / float64(losses)
	}
	return wins, losses, avgWin, avgLoss, largestWin, largestLoss
}

func calculateWinRate(wins, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total)
}

func calculateGrossProfit(m Metrics) float64 {
	return m.AverageWin * float64(m.WinningBets)
}

func calculateGrossLoss(m Metrics) float64 {
	return math.Abs(m.AverageLoss) * float64(m.LosingBets)
}

func calculateNetProfit(m Metrics) float64 {
	return calculateGrossProfit(m) - calculateGrossLoss(m)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	return mean / float64(len(values))
}

func stddev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := average(values)
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func downsideStddev(values []float64) float64 {
	negatives := make([]float64, 0)
	for _, v := range values {
		if v < 0 {
			negatives = append(negatives, v)
		}
	}
	return stddev(negatives)
}

func sortFloats(values []float64) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

// HashParameters creates a stable hash for parameter maps
func HashParameters(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}
