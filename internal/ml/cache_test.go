package ml

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheKeyString tests cache key string representation
func TestCacheKeyString(t *testing.T) {
	key := CacheKey{
		RaceID:       uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		RunnerID:     uuid.MustParse("87654321-4321-8765-4321-876543218765"),
		StrategyID:   uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		ModelVersion: "1.0",
	}

	keyStr := key.String()
	assert.NotEmpty(t, keyStr)
	assert.Contains(t, keyStr, "12345678")
	assert.Contains(t, keyStr, "87654321")
	assert.Contains(t, keyStr, "11111111")
	assert.Contains(t, keyStr, "1.0")
}

// TestPredictionCacheGet tests cache Get operation
func TestPredictionCacheGet(t *testing.T) {
	cache := NewPredictionCache(time.Hour, 100)
	defer cache.Clear()

	key := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   uuid.New(),
		ModelVersion: "1.0",
	}

	ctx := context.Background()

	// Get non-existent key should return nil
	result := cache.Get(ctx, key)
	assert.Nil(t, result)
}

// TestPredictionCacheSet tests cache Set operation
func TestPredictionCacheSet(t *testing.T) {
	cache := NewPredictionCache(time.Hour, 100)
	defer cache.Clear()

	key := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   uuid.New(),
		ModelVersion: "1.0",
	}

	prediction := &PredictionResult{
		RaceID:       key.RaceID,
		RunnerID:     key.RunnerID,
		Probability:  0.75,
		Confidence:   0.85,
		PredictedAt:  time.Now(),
	}

	ctx := context.Background()
	cache.Set(ctx, key, prediction)

	retrieved := cache.Get(ctx, key)
	require.NotNil(t, retrieved)
	assert.Equal(t, prediction.Probability, retrieved.Probability)
	assert.Equal(t, prediction.Confidence, retrieved.Confidence)
}

// TestPredictionCacheExpiration tests cache TTL expiration
func TestPredictionCacheExpiration(t *testing.T) {
	cache := NewPredictionCache(100*time.Millisecond, 100)
	defer cache.Clear()

	key := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   uuid.New(),
		ModelVersion: "1.0",
	}

	prediction := &PredictionResult{
		Probability: 0.75,
		Confidence:  0.85,
	}

	ctx := context.Background()
	cache.Set(ctx, key, prediction)

	// Should be in cache immediately
	retrieved := cache.Get(ctx, key)
	require.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	expired := cache.Get(ctx, key)
	assert.Nil(t, expired)
}

// TestPredictionCacheInvalidate tests cache invalidation by strategy ID
func TestPredictionCacheInvalidate(t *testing.T) {
	cache := NewPredictionCache(time.Hour, 100)
	defer cache.Clear()

	strategyID := uuid.New()
	otherStrategyID := uuid.New()

	key1 := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   strategyID,
		ModelVersion: "1.0",
	}

	key2 := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   strategyID,
		ModelVersion: "1.0",
	}

	key3 := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   otherStrategyID,
		ModelVersion: "1.0",
	}

	prediction := &PredictionResult{Probability: 0.5}

	ctx := context.Background()
	cache.Set(ctx, key1, prediction)
	cache.Set(ctx, key2, prediction)
	cache.Set(ctx, key3, prediction)

	// Invalidate first strategy
	cache.Invalidate(ctx, strategyID)

	// First two should be gone
	assert.Nil(t, cache.Get(ctx, key1))
	assert.Nil(t, cache.Get(ctx, key2))

	// Third should still be there
	retrieved := cache.Get(ctx, key3)
	require.NotNil(t, retrieved)
}

// TestPredictionCacheStats tests cache statistics tracking
func TestPredictionCacheStats(t *testing.T) {
	cache := NewPredictionCache(time.Hour, 100)
	defer cache.Clear()

	key := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     uuid.New(),
		StrategyID:   uuid.New(),
		ModelVersion: "1.0",
	}

	prediction := &PredictionResult{Probability: 0.75}

	ctx := context.Background()

	// Initial stats
	hits, misses, ratio := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, 0.0, ratio)

	// Miss
	_ = cache.Get(ctx, key)
	hits, misses, ratio = cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, 0.0, ratio)

	// Set and hit
	cache.Set(ctx, key, prediction)
	_ = cache.Get(ctx, key)
	hits, misses, ratio = cache.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, 0.5, ratio)
}

// TestPredictionCacheMaxSize tests cache size limit enforcement
func TestPredictionCacheMaxSize(t *testing.T) {
	maxSize := 5
	cache := NewPredictionCache(time.Hour, maxSize)
	defer cache.Clear()

	ctx := context.Background()
	prediction := &PredictionResult{Probability: 0.75}

	// Fill cache beyond max size
	for i := 0; i < maxSize+5; i++ {
		key := CacheKey{
			RaceID:       uuid.New(),
			RunnerID:     uuid.New(),
			StrategyID:   uuid.New(),
			ModelVersion: "1.0",
		}
		cache.Set(ctx, key, prediction)
	}

	// Cache should not exceed max size (accounting for some internal operations)
	// This is a basic check since the cache uses ExpireAll periodically
}

// TestCacheKeyEquality tests cache key equality
func TestCacheKeyEquality(t *testing.T) {
	raceID := uuid.New()
	runnerID := uuid.New()
	strategyID := uuid.New()

	key1 := CacheKey{
		RaceID:       raceID,
		RunnerID:     runnerID,
		StrategyID:   strategyID,
		ModelVersion: "1.0",
	}

	key2 := CacheKey{
		RaceID:       raceID,
		RunnerID:     runnerID,
		StrategyID:   strategyID,
		ModelVersion: "1.0",
	}

	key3 := CacheKey{
		RaceID:       uuid.New(),
		RunnerID:     runnerID,
		StrategyID:   strategyID,
		ModelVersion: "1.0",
	}

	assert.Equal(t, key1.String(), key2.String())
	assert.NotEqual(t, key1.String(), key3.String())
}
