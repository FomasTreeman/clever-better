package backtest

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

func TestCalculateMetrics(t *testing.T) {
	state := NewBacktestState(100)
	start := time.Now().Add(-10 * 24 * time.Hour)
	end := time.Now()
	state.EquityCurve = EquityCurve{
		{Time: start, Value: 100},
		{Time: end, Value: 120},
	}

	pnl := 10.0
	bet := &models.Bet{ID: uuid.New(), Stake: 10, ProfitLoss: &pnl}
	state.Bets = []*models.Bet{bet}

	cfg := BacktestConfig{StartDate: start, EndDate: end, RiskFreeRate: 0.01}
	metrics := CalculateMetrics(state, cfg)
	if metrics.TotalReturn <= 0 {
		t.Fatalf("expected positive return")
	}
	if metrics.TotalBets != 1 {
		t.Fatalf("expected total bets 1, got %d", metrics.TotalBets)
	}
}

func TestSharpeRatio(t *testing.T) {
	returns := []float64{0.01, 0.02, -0.01, 0.03}
	sharpe := calculateSharpeRatio(returns, 0)
	if sharpe == 0 {
		t.Fatalf("expected non-zero sharpe ratio")
	}
}
