package backtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

func TestRunMonteCarloDeterministic(t *testing.T) {
	bet := &models.Bet{ID: uuid.New(), Odds: 2.0, Stake: 10}
	probabilities := map[string]float64{bet.ID.String(): 0.6}

	result, err := RunMonteCarlo(context.Background(), []*models.Bet{bet}, probabilities, MonteCarloConfig{
		Iterations:      1000,
		Seed:            42,
		CommissionRate:  0.05,
		InitialBankroll: 100,
	})
	if err != nil {
		t.Fatalf("RunMonteCarlo failed: %v", err)
	}
	if result.Iterations != 1000 {
		t.Fatalf("expected 1000 iterations")
	}
	if len(result.Distribution) != 1000 {
		t.Fatalf("expected distribution length 1000")
	}
}
