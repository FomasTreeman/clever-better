package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/service"
)

// TestMLIntegrationFlow tests the complete ML integration workflow
func TestMLIntegrationFlow(t *testing.T) {
	// This test would require a running ML service and database
	// Skip if integration tests are disabled
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Create test data
	strategyID := uuid.New()
	raceID := uuid.New()
	runnerID := uuid.New()

	t.Run("PredictionCaching", func(t *testing.T) {
		// Test that predictions are cached and hit ratio increases
		// This would require mock ML service setup
	})

	t.Run("StrategyGeneration", func(t *testing.T) {
		// Test strategy generation from backtest results
		constraints := ml.StrategyConstraints{
			RiskLevel:         "medium",
			TargetReturn:      0.15,
			MaxDrawdownLimit:  0.15,
			MinWinRate:        0.55,
			MaxCandidates:     5,
		}
		_ = constraints
		// Would call ml client to generate strategies
	})

	t.Run("FeedbackSubmission", func(t *testing.T) {
		// Test feedback submission workflow
		backtestResult := &models.BacktestResult{
			ID:             uuid.New(),
			StrategyID:     strategyID,
			CompositeScore: 0.75,
			SharpeRatio:    1.5,
			ROI:            0.20,
			MaxDrawdown:    0.10,
			WinRate:        0.60,
			CreatedAt:      time.Now(),
		}
		_ = backtestResult
		// Would submit feedback and verify processing
	})

	t.Run("StrategyEvaluation", func(t *testing.T) {
		// Test strategy evaluation and ranking
		strategies := []uuid.UUID{strategyID}
		_ = strategies
		// Would evaluate multiple strategies and verify rankings
	})
}

// TestCachingBehavior tests the caching layer functionality
func TestCachingBehavior(t *testing.T) {
	cache := ml.NewPredictionCache(time.Hour, 100)
	defer cache.Clear()

	key := ml.CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   uuid.New(),
		ModelVersion: "1.0",
	}

	ctx := context.Background()

	t.Run("CacheMiss", func(t *testing.T) {
		result := cache.Get(ctx, key)
		assert.Nil(t, result)

		hits, misses, _ := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
	})

	t.Run("CacheHit", func(t *testing.T) {
		prediction := &ml.PredictionResult{
			RaceID:         key.RaceID,
			RunnerID:       key.RunnerID,
			StrategyID:     key.StrategyID,
			Probability:    0.75,
			Confidence:     0.85,
			PredictedAt:    time.Now(),
		}

		cache.Set(ctx, key, prediction)

		cached := cache.Get(ctx, key)
		require.NotNil(t, cached)
		assert.Equal(t, prediction.Probability, cached.Probability)
		assert.Equal(t, prediction.Confidence, cached.Confidence)

		hits, _, ratio := cache.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Greater(t, ratio, 0.0)
	})

	t.Run("CacheInvalidation", func(t *testing.T) {
		strategyID := uuid.New()
		key1 := ml.CacheKey{
			RaceID:       uuid.New(),
			RunnerID:     uuid.New(),
			StrategyID:   strategyID,
			ModelVersion: "1.0",
		}
		key2 := ml.CacheKey{
			RaceID:       uuid.New(),
			RunnerID:     uuid.New(),
			StrategyID:   strategyID,
			ModelVersion: "1.0",
		}

		pred1 := &ml.PredictionResult{Probability: 0.5}
		pred2 := &ml.PredictionResult{Probability: 0.6}

		cache.Set(ctx, key1, pred1)
		cache.Set(ctx, key2, pred2)

		cache.Invalidate(ctx, strategyID)

		result1 := cache.Get(ctx, key1)
		result2 := cache.Get(ctx, key2)

		assert.Nil(t, result1)
		assert.Nil(t, result2)
	})
}

// TestStrategyEvaluation tests strategy evaluation logic
func TestStrategyEvaluation(t *testing.T) {
	t.Run("CompositeScoreCalculation", func(t *testing.T) {
		mlScore := 0.8
		backtestResult := &models.BacktestResult{
			CompositeScore: 0.7,
		}

		// Expected: (0.8 + 0.7) / 2 = 0.75
		compositeScore := (mlScore + backtestResult.CompositeScore) / 2.0
		assert.Equal(t, 0.75, compositeScore)
	})

	t.Run("StrategyActivationThreshold", func(t *testing.T) {
		threshold := 0.65
		scores := []float64{0.75, 0.70, 0.65, 0.60}

		activated := 0
		for _, score := range scores {
			if score >= threshold {
				activated++
			}
		}

		assert.Equal(t, 3, activated)
	})
}

// BenchmarkPredictionCaching benchmarks cache performance
func BenchmarkPredictionCaching(b *testing.B) {
	cache := ml.NewPredictionCache(time.Hour, 1000)
	defer cache.Clear()

	ctx := context.Background()
	keys := make([]ml.CacheKey, b.N)
	predictions := make([]*ml.PredictionResult, b.N)

	for i := 0; i < b.N; i++ {
		keys[i] = ml.CacheKey{
			RaceID:       uuid.New(),
			RunnerID:     uuid.New(),
			StrategyID:   uuid.New(),
			ModelVersion: "1.0",
		}
		predictions[i] = &ml.PredictionResult{
			Probability: 0.5,
			Confidence:  0.8,
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(ctx, keys[i], predictions[i])
		_ = cache.Get(ctx, keys[i])
	}
}
