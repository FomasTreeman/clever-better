// Package ml provides caching for ML predictions.
package ml

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
)

// CacheKey represents a unique key for caching predictions
type CacheKey struct {
	RaceID       uuid.UUID
	RunnerID     uuid.UUID
	StrategyID   uuid.UUID
	ModelVersion string
}

// String returns string representation of cache key
func (k CacheKey) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", k.RaceID, k.RunnerID, k.StrategyID, k.ModelVersion)
}

// PredictionCache provides in-memory caching for ML predictions
type PredictionCache struct {
	cache      *cache.Cache
	ttl        time.Duration
	maxSize    int
	mu         sync.RWMutex
	hitCount   uint64
	missCount  uint64
}

// NewPredictionCache creates a new prediction cache
func NewPredictionCache(ttl time.Duration, maxSize int) *PredictionCache {
	return &PredictionCache{
		cache:   cache.New(ttl, ttl*2),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves a cached prediction
func (pc *PredictionCache) Get(ctx context.Context, key CacheKey) *PredictionResult {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if result, found := pc.cache.Get(key.String()); found {
		pc.hitCount++
		pc.updateMetrics()
		if pred, ok := result.(*PredictionResult); ok {
			return pred
		}
	}

	pc.missCount++
	pc.updateMetrics()
	return nil
}

// Set stores a prediction in cache
func (pc *PredictionCache) Set(ctx context.Context, key CacheKey, prediction *PredictionResult) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Check size limit
	if pc.cache.ItemCount() >= pc.maxSize {
		// Remove expired items first
		pc.cache.DeleteExpired()
	}

	pc.cache.Set(key.String(), prediction, pc.ttl)
}

// Invalidate removes all cache entries for a specific strategy
func (pc *PredictionCache) Invalidate(ctx context.Context, strategyID uuid.UUID) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Remove only items matching the strategy ID
	// Cache key format: raceID:runnerID:strategyID:modelVersion
	items := pc.cache.Items()
	strategyIDStr := strategyID.String()
	
	for k := range items {
		// Parse the cache key to extract strategy ID
		parts := extractStrategyFromCacheKey(k)
		if parts == strategyIDStr {
			pc.cache.Delete(k)
		}
	}
}

// extractStrategyFromCacheKey parses the strategy ID from a cache key string
func extractStrategyFromCacheKey(keyStr string) string {
	// Cache key format: raceID:runnerID:strategyID:modelVersion
	// We need to extract the third UUID component
	parts := splitCacheKey(keyStr)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// splitCacheKey splits a cache key string into its components
// Handles UUID parsing: each UUID has 4 colons, so we split carefully
func splitCacheKey(keyStr string) []string {
	// Simple approach: split by ":" and reconstruct UUIDs
	// Cache key format: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx:modelversion"
	// Each UUID is 36 chars (32 hex + 4 hyphens), followed by a colon
	// We can split more intelligently:
	parts := make([]string, 0)
	currentPos := 0
	uuidCount := 0
	
	for i := 0; i < len(keyStr); i++ {
		if keyStr[i] == ':' {
			// Check if this is the end of a UUID (after 36 chars from last split or start)
			if uuidCount < 3 && (i == currentPos+36) {
				// This is a UUID separator
				parts = append(parts, keyStr[currentPos:i])
				currentPos = i + 1
				uuidCount++
			} else if uuidCount == 3 {
				// Everything after the third UUID is the model version
				parts = append(parts, keyStr[currentPos:])
				return parts
			}
		}
	}
	
	// Handle case where model version is at the end
	if uuidCount == 3 && currentPos < len(keyStr) {
		parts = append(parts, keyStr[currentPos:])
	}
	
	return parts
}

// Clear flushes the entire cache
func (pc *PredictionCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache.Flush()
	pc.hitCount = 0
	pc.missCount = 0
}

// Stats returns cache statistics
func (pc *PredictionCache) Stats() (hits, misses uint64, ratio float64) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	hits = pc.hitCount
	misses = pc.missCount
	total := hits + misses
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}
	return
}

// updateMetrics updates Prometheus metrics
func (pc *PredictionCache) updateMetrics() {
	hits, misses, ratio := pc.Stats()
	_ = hits
	_ = misses
	MLCacheHitRatio.Set(ratio)
}

// ItemCount returns the number of items in cache
func (pc *PredictionCache) ItemCount() int {
	return pc.cache.ItemCount()
}
