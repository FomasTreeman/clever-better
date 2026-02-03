// Package ml provides cached ML client implementation.
package ml

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/models"
)

// CachedMLClient wraps MLClient with prediction caching
type CachedMLClient struct {
	client *MLClient
	cache  *PredictionCache
	logger *logrus.Logger
}

// NewCachedMLClient creates a new cached ML client
func NewCachedMLClient(cfg *config.MLServiceConfig, logger *logrus.Logger) (*CachedMLClient, error) {
	client, err := NewMLClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	cache := NewPredictionCache(
		time.Duration(cfg.CacheTTLSeconds)*time.Second,
		cfg.CacheMaxSize,
	)

	return &CachedMLClient{
		client: client,
		cache:  cache,
		logger: logger,
	}, nil
}

// GetPrediction retrieves prediction with caching
func (c *CachedMLClient) GetPrediction(ctx context.Context, raceID, runnerID, strategyID uuid.UUID, features []float64, modelVersion string) (*PredictionResult, error) {
	// Check cache first
	cacheKey := CacheKey{
		RaceID:       raceID,
		RunnerID:     runnerID,
		StrategyID:   strategyID,
		ModelVersion: modelVersion,
	}

	if cached := c.cache.Get(ctx, cacheKey); cached != nil {
		c.logger.WithField("cache_key", cacheKey.String()).Debug("Cache hit for prediction")
		MLPredictionsTotal.WithLabelValues("cached", "true").Inc()
		return cached, nil
	}

	// Cache miss, call ML service
	c.logger.WithField("cache_key", cacheKey.String()).Debug("Cache miss, fetching from ML service")
	result, err := c.client.GetPrediction(ctx, raceID, runnerID, strategyID, features, modelVersion)
	if err != nil {
		return nil, err
	}

	// Store in cache
	result.RunnerID = runnerID
	result.StrategyID = strategyID
	result.ModelVersion = modelVersion
	c.cache.Set(ctx, cacheKey, result)

	MLPredictionsTotal.WithLabelValues("grpc", "false").Inc()
	return result, nil
}

// EvaluateStrategy evaluates a strategy (not cached)
func (c *CachedMLClient) EvaluateStrategy(ctx context.Context, strategyID uuid.UUID) (float64, string, error) {
	return c.client.EvaluateStrategy(ctx, strategyID)
}

// SubmitBacktestFeedback submits feedback and invalidates cache for strategy
func (c *CachedMLClient) SubmitBacktestFeedback(ctx context.Context, result *models.BacktestResult) error {
	err := c.client.SubmitBacktestFeedback(ctx, result)
	if err != nil {
		return err
	}

	// Invalidate cache for this strategy since model may be retrained
	c.cache.Invalidate(ctx, result.StrategyID)
	c.logger.WithField("strategy_id", result.StrategyID).Debug("Invalidated cache for strategy")

	return nil
}

// GenerateStrategy generates strategies (not cached)
func (c *CachedMLClient) GenerateStrategy(ctx context.Context, constraints StrategyConstraints) ([]*GeneratedStrategy, error) {
	return c.client.GenerateStrategy(ctx, constraints)
}

// BatchPredict performs batch predictions with partial caching
func (c *CachedMLClient) BatchPredict(ctx context.Context, requests []PredictionRequest) ([]*PredictionResult, error) {
	results := make([]*PredictionResult, len(requests))
	uncachedRequests := make([]PredictionRequest, 0)
	uncachedIndices := make([]int, 0)

	// Check cache for each request
	for i, req := range requests {
		cacheKey := CacheKey{
			RaceID:       req.RaceID,
			RunnerID:     req.RunnerID,
			StrategyID:   req.StrategyID,
			ModelVersion: req.ModelVersion,
		}

		if cached := c.cache.Get(ctx, cacheKey); cached != nil {
			results[i] = cached
		} else {
			uncachedRequests = append(uncachedRequests, req)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Fetch uncached predictions
	if len(uncachedRequests) > 0 {
		c.logger.WithFields(logrus.Fields{
			"total_requests":    len(requests),
			"cached":            len(requests) - len(uncachedRequests),
			"uncached":          len(uncachedRequests),
		}).Debug("Batch prediction with partial cache")

		uncachedResults, err := c.client.BatchPredict(ctx, uncachedRequests)
		if err != nil {
			return nil, err
		}

		// Store in cache and populate results
		for i, result := range uncachedResults {
			idx := uncachedIndices[i]
			req := uncachedRequests[i]

			cacheKey := CacheKey{
				RaceID:       req.RaceID,
				RunnerID:     req.RunnerID,
				StrategyID:   req.StrategyID,
				ModelVersion: req.ModelVersion,
			}
			c.cache.Set(ctx, cacheKey, result)
			results[idx] = result
		}
	}

	return results, nil
}

// InvalidateStrategyCache invalidates cache for a specific strategy
func (c *CachedMLClient) InvalidateStrategyCache(ctx context.Context, strategyID uuid.UUID) {
	c.cache.Invalidate(ctx, strategyID)
}

// ClearCache clears all cached predictions
func (c *CachedMLClient) ClearCache() {
	c.cache.Clear()
}

// GetCacheStats returns cache statistics
func (c *CachedMLClient) GetCacheStats() (hits, misses uint64, hitRatio float64) {
	return c.cache.Stats()
}

// Close closes the underlying ML client
func (c *CachedMLClient) Close() error {
	return c.client.Close()
}
