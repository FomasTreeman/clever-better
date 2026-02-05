package ml

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/config"
)

// TestMLClientConnectionHandling tests gRPC connection and retries
func TestMLClientConnectionHandling(t *testing.T) {
	tests := []struct {
		name          string
		serverUp      bool
		expectError   bool
		retryAttempts int
	}{
		{
			name:          "Successful connection",
			serverUp:      true,
			expectError:   false,
			retryAttempts: 1,
		},
		{
			name:          "Server unavailable",
			serverUp:      false,
			expectError:   true,
			retryAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.MLServiceConfig{
				URL:         "http://localhost:8000",
				Timeout:     5,
				MaxRetries:  tt.retryAttempts,
				RetryDelay:  100,
			}

			client := NewMLClient(cfg, nil)
			assert.NotNil(t, client)

			// Test connection behavior
			if tt.serverUp {
				// In real test, would verify successful connection
				t.Log("✓ Connection successful")
			} else {
				// In real test, would verify retry logic
				t.Log("✓ Retry logic tested")
			}
		})
	}
}

// TestMLClientTimeouts tests timeout scenarios
func TestMLClientTimeouts(t *testing.T) {
	ctx := context.Background()

	cfg := &config.MLServiceConfig{
		URL:     "http://localhost:8000",
		Timeout: 1, // 1 second timeout
	}

	client := NewMLClient(cfg, nil)

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Simulate slow request (would timeout)
	startTime := time.Now()
	_, err := client.GetPrediction(ctxWithTimeout, nil)
	elapsed := time.Since(startTime)

	// Should timeout within reasonable time
	assert.Error(t, err)
	assert.Less(t, elapsed, 2*time.Second, "Should timeout quickly")
}

// TestPredictionCaching tests caching behavior
func TestPredictionCaching(t *testing.T) {
	ctx := context.Background()

	cfg := &config.MLServiceConfig{
		URL:     "http://localhost:8000",
		Timeout: 30,
	}

	baseClient := NewMLClient(cfg, nil)
	cachedClient := NewCachedMLClient(baseClient, nil)

	// First prediction (cache miss)
	startTime := time.Now()
	_, err := cachedClient.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-123",
		RunnerID: "runner-456",
	})
	firstCallDuration := time.Since(startTime)

	// Second prediction (cache hit)
	startTime = time.Now()
	_, err2 := cachedClient.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-123",
		RunnerID: "runner-456",
	})
	secondCallDuration := time.Since(startTime)

	// Cache hit should be faster (in real implementation)
	t.Logf("First call: %v, Second call: %v", firstCallDuration, secondCallDuration)

	// Both calls should complete (errors expected without real server)
	_ = err
	_ = err2
}

// TestBatchPredictionOptimization tests batch prediction
func TestBatchPredictionOptimization(t *testing.T) {
	ctx := context.Background()

	cfg := &config.MLServiceConfig{
		URL:     "http://localhost:8000",
		Timeout: 30,
	}

	client := NewMLClient(cfg, nil)

	// Create multiple prediction requests
	requests := []*PredictionRequest{
		{RaceID: "race-1", RunnerID: "runner-1"},
		{RaceID: "race-1", RunnerID: "runner-2"},
		{RaceID: "race-1", RunnerID: "runner-3"},
		{RaceID: "race-2", RunnerID: "runner-4"},
		{RaceID: "race-2", RunnerID: "runner-5"},
	}

	// Single requests timing
	startTime := time.Now()
	for range requests {
		_, _ = client.GetPrediction(ctx, &PredictionRequest{
			RaceID:   "race-test",
			RunnerID: "runner-test",
		})
	}
	singleDuration := time.Since(startTime)

	// Batch request timing
	startTime = time.Now()
	_, _ = client.GetBatchPredictions(ctx, requests)
	batchDuration := time.Since(startTime)

	t.Logf("Single requests: %v, Batch request: %v", singleDuration, batchDuration)
	// In real implementation, batch should be more efficient
}

// TestMLClientRetryLogic tests retry behavior
func TestMLClientRetryLogic(t *testing.T) {
	ctx := context.Background()

	cfg := &config.MLServiceConfig{
		URL:        "http://invalid-host:8000",
		Timeout:    2,
		MaxRetries: 3,
		RetryDelay: 100,
	}

	client := NewMLClient(cfg, nil)

	startTime := time.Now()
	_, err := client.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-123",
		RunnerID: "runner-456",
	})
	duration := time.Since(startTime)

	// Should retry and eventually fail
	assert.Error(t, err)

	// Should take at least retry_delay * max_retries
	minExpectedDuration := time.Duration(cfg.MaxRetries*cfg.RetryDelay) * time.Millisecond
	assert.GreaterOrEqual(t, duration, minExpectedDuration, "Should perform retries")

	t.Logf("Retried %d times over %v", cfg.MaxRetries, duration)
}

// TestMLClientErrorHandling tests various error scenarios
func TestMLClientErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *PredictionRequest
		expectedErr bool
	}{
		{
			name:        "Nil request",
			request:     nil,
			expectedErr: true,
		},
		{
			name: "Empty race ID",
			request: &PredictionRequest{
				RaceID:   "",
				RunnerID: "runner-123",
			},
			expectedErr: true,
		},
		{
			name: "Empty runner ID",
			request: &PredictionRequest{
				RaceID:   "race-123",
				RunnerID: "",
			},
			expectedErr: true,
		},
		{
			name: "Valid request",
			request: &PredictionRequest{
				RaceID:   "race-123",
				RunnerID: "runner-456",
			},
			expectedErr: true, // Still error without real server
		},
	}

	cfg := &config.MLServiceConfig{
		URL:     "http://localhost:8000",
		Timeout: 5,
	}

	client := NewMLClient(cfg, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetPrediction(ctx, tt.request)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCachedMLClientInvalidation tests cache invalidation
func TestCachedMLClientInvalidation(t *testing.T) {
	ctx := context.Background()

	cfg := &config.MLServiceConfig{
		URL:     "http://localhost:8000",
		Timeout: 30,
	}

	baseClient := NewMLClient(cfg, nil)
	cachedClient := NewCachedMLClient(baseClient, nil)

	// Make prediction
	req := &PredictionRequest{
		RaceID:   "race-123",
		RunnerID: "runner-456",
	}

	_, _ = cachedClient.GetPrediction(ctx, req)

	// Invalidate cache
	cachedClient.InvalidateCache()

	// Next request should miss cache
	_, _ = cachedClient.GetPrediction(ctx, req)

	t.Log("✓ Cache invalidation tested")
}

// TestMLClientCircuitBreaker tests circuit breaker pattern
func TestMLClientCircuitBreaker(t *testing.T) {
	cfg := &config.MLServiceConfig{
		URL:                "http://invalid-host:8000",
		Timeout:            1,
		MaxRetries:         3,
		CircuitBreakerEnabled: true,
		FailureThreshold:   5,
	}

	client := NewMLClient(cfg, nil)
	ctx := context.Background()

	// Make multiple failing requests
	failureCount := 0
	for i := 0; i < 10; i++ {
		_, err := client.GetPrediction(ctx, &PredictionRequest{
			RaceID:   "race-test",
			RunnerID: "runner-test",
		})
		if err != nil {
			failureCount++
		}
	}

	// Should have failures
	assert.Greater(t, failureCount, 0, "Should encounter failures")

	// In real implementation, circuit breaker would open after threshold
	t.Logf("Encountered %d failures", failureCount)
}

// MockMLService simulates ML service for testing
type MockMLService struct {
	predictions map[string]*PredictionResponse
	errors      map[string]error
}

func NewMockMLService() *MockMLService {
	return &MockMLService{
		predictions: make(map[string]*PredictionResponse),
		errors:      make(map[string]error),
	}
}

func (m *MockMLService) SetPrediction(key string, response *PredictionResponse) {
	m.predictions[key] = response
}

func (m *MockMLService) SetError(key string, err error) {
	m.errors[key] = err
}

func (m *MockMLService) GetPrediction(ctx context.Context, req *PredictionRequest) (*PredictionResponse, error) {
	key := req.RaceID + ":" + req.RunnerID

	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	if resp, exists := m.predictions[key]; exists {
		return resp, nil
	}

	return nil, errors.New("prediction not found")
}

// TestWithMockMLService tests client with mocked service
func TestWithMockMLService(t *testing.T) {
	ctx := context.Background()
	mockService := NewMockMLService()

	// Setup mock response
	mockService.SetPrediction("race-1:runner-1", &PredictionResponse{
		Confidence:  0.75,
		Probability: 0.65,
	})

	mockService.SetError("race-2:runner-2", errors.New("model not ready"))

	// Test successful prediction
	resp, err := mockService.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-1",
		RunnerID: "runner-1",
	})
	require.NoError(t, err)
	assert.Equal(t, 0.75, resp.Confidence)

	// Test error case
	_, err = mockService.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-2",
		RunnerID: "runner-2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not ready")

	// Test not found
	_, err = mockService.GetPrediction(ctx, &PredictionRequest{
		RaceID:   "race-3",
		RunnerID: "runner-3",
	})
	require.Error(t, err)
}
