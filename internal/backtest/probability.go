package backtest

import (
	"math"

	"github.com/yourusername/clever-better/internal/models"
)

// EstimateProbability estimates win probability using implied odds and historical calibration
func EstimateProbability(runner *models.Runner, odds float64, historicalResults []*models.RaceResult) float64 {
	_ = runner
	if odds <= 0 {
		return 0
	}
	implied := 1.0 / odds
	calibration := historicalCalibration(odds, historicalResults)
	return math.Min(1.0, math.Max(0.0, implied*calibration))
}

func historicalCalibration(odds float64, results []*models.RaceResult) float64 {
	if len(results) == 0 {
		return 1.0
	}
	_ = odds
	// Placeholder calibration: return neutral factor. Implement buckets as data grows.
	return 1.0
}
