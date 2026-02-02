package strategy

import (
	"fmt"
	"math"
	"time"

	"github.com/yourusername/clever-better/internal/models"
)

// BaseStrategy provides shared functionality for strategies
type BaseStrategy struct {
	MinOdds          float64
	MaxOdds          float64
	MinLiquidity     float64
	KellyFraction    float64
	MinEdgeThreshold float64
}

// ValidateOdds ensures odds are within acceptable bounds
func (b *BaseStrategy) ValidateOdds(odds float64) error {
	if odds <= 1.0 {
		return fmt.Errorf("odds must be greater than 1.0")
	}
	if b.MinOdds > 0 && odds < b.MinOdds {
		return fmt.Errorf("odds below minimum")
	}
	if b.MaxOdds > 0 && odds > b.MaxOdds {
		return fmt.Errorf("odds above maximum")
	}
	return nil
}

// CheckLiquidity ensures the odds snapshot has sufficient liquidity
func (b *BaseStrategy) CheckLiquidity(snapshot *models.OddsSnapshot) bool {
	if snapshot == nil {
		return false
	}
	if b.MinLiquidity <= 0 {
		return true
	}
	if snapshot.BackSize != nil && *snapshot.BackSize >= b.MinLiquidity {
		return true
	}
	if snapshot.LaySize != nil && *snapshot.LaySize >= b.MinLiquidity {
		return true
	}
	return false
}

// ApplyKellyCriterion calculates stake based on the Kelly criterion
func (b *BaseStrategy) ApplyKellyCriterion(probability float64, odds float64, bankroll float64) float64 {
	if probability <= 0 || odds <= 1 || bankroll <= 0 {
		return 0
	}
	p := probability
	q := 1.0 - p
	bOdds := odds - 1.0
	kelly := (bOdds*p - q) / bOdds
	if kelly <= 0 {
		return 0
	}
	fraction := b.KellyFraction
	if fraction <= 0 {
		fraction = 0.5
	}
	return bankroll * kelly * fraction
}

// CalculateExpectedValue calculates expected value for a bet
func (b *BaseStrategy) CalculateExpectedValue(probability float64, odds float64, stake float64) float64 {
	if probability <= 0 || odds <= 1 || stake <= 0 {
		return 0
	}
	winProfit := (odds - 1.0) * stake
	loss := stake
	return probability*winProfit - (1.0-probability)*loss
}

// ValidateTemporalSafety ensures data is not from the future
func (b *BaseStrategy) ValidateTemporalSafety(currentTime time.Time, oddsHistory []*models.OddsSnapshot) error {
	for _, snapshot := range oddsHistory {
		if snapshot.Time.After(currentTime) {
			return fmt.Errorf("temporal safety violation: odds timestamp %s after %s", snapshot.Time, currentTime)
		}
	}
	return nil
}

// NormalizeProbability ensures probability in [0,1]
func (b *BaseStrategy) NormalizeProbability(p float64) float64 {
	if math.IsNaN(p) || math.IsInf(p, 0) {
		return 0
	}
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}
