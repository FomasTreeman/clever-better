package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/logger"
	"github.com/yourusername/clever-better/internal/metrics"
	"github.com/yourusername/clever-better/internal/tracing"
)

func TestObservabilityIntegration(t *testing.T) {
	// Initialize all observability components
	metrics.InitRegistry()
	
	// Set up logger with buffer to capture output
	appLog := logrus.New()
	logBuf := &bytes.Buffer{}
	appLog.SetOutput(logBuf)
	appLog.SetFormatter(&logrus.JSONFormatter{})
	appLog.SetLevel(logrus.DebugLevel)
	
	// Create specialized loggers
	strategyLogger := logger.NewStrategyLogger(appLog)
	mlLogger := logger.NewMLLogger(appLog)
	auditLogger := logger.NewAuditLogger(appLog)
	
	// Initialize X-Ray tracing (disabled for test)
	tracing.Initialize(tracing.Config{
		ServiceName: "test-bot",
		Enabled:     false, // Disable for unit tests
	}, appLog)
	
	// Test complete observability flow
	t.Run("metrics collection", func(t *testing.T) {
		// Record bet placement
		metrics.RecordBetPlaced()
		
		// Record strategy evaluation
		metrics.RecordStrategyEvaluation(0.5)
		
		// Update bankroll
		metrics.UpdateBankroll(10500)
		
		// Update exposure
		metrics.UpdateExposure(5000)
		
		// All operations should complete without panic
		assert.True(t, true)
	})
	
	t.Run("strategy logging", func(t *testing.T) {
		logBuf.Reset()
		
		// Log strategy decision
		strategyLogger.LogStrategyDecision(
			"strategy_001",
			"TestStrategy",
			"PLACE_BET",
			0.87,
			0.045,
			0.05,
			100,
			123456,
			3.5,
		)
		
		// Verify log output
		var logEntry map[string]interface{}
		err := json.Unmarshal(logBuf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "PLACE_BET", logEntry["decision"])
	})
	
	t.Run("ml logging", func(t *testing.T) {
		logBuf.Reset()
		
		// Log ML prediction
		mlLogger.LogMLPredictionRequest("strategy_predictor", 50, true, 45)
		
		// Verify log output
		var logEntry map[string]interface{}
		err := json.Unmarshal(logBuf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "strategy_predictor", logEntry["model_type"])
	})
	
	t.Run("audit logging", func(t *testing.T) {
		logBuf.Reset()
		
		// Log bet placement
		auditLogger.LogBetPlacement(
			"bet_123",
			"strategy_001",
			"1.123456",
			456789,
			"BACK",
			100,
			3.5,
			time.Date(2024, 2, 3, 12, 0, 0, 0, time.UTC),
			false,
		)
		
		// Verify log output
		var logEntry map[string]interface{}
		err := json.Unmarshal(logBuf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "bet_123", logEntry["bet_id"])
	})
	
	t.Run("prometheus metrics endpoint", func(t *testing.T) {
		registry := metrics.GetRegistry()
		assert.NotNil(t, registry)
		
		// Create test server with metrics handler
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		server := httptest.NewServer(handler)
		defer server.Close()
		
		// Make request to metrics endpoint
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
		
		// Verify metrics are present in response
		body := &bytes.Buffer{}
		_, err = body.ReadFrom(resp.Body)
		require.NoError(t, err)
		
		metricsText := body.String()
		assert.Contains(t, metricsText, "clever_better_")
	})
	
	t.Run("end-to-end trading workflow", func(t *testing.T) {
		logBuf.Reset()
		
		// Simulate complete trading workflow with observability
		
		// 1. Strategy evaluation
		strategyLogger.LogStrategyEvaluation(
			"strategy_001",
			"TestStrategy",
			"race_123",
			100,
			2,
			500.0,
		)
		metrics.RecordStrategyEvaluation(0.5)
		
		// 2. ML prediction
		mlLogger.LogMLPredictionRequest("strategy_predictor", 50, false, 150)
		metrics.RecordStrategyDecision("strategy_001", "TestStrategy", "PLACE_BET", "success")
		
		// 3. Strategy decision
		strategyLogger.LogStrategyDecision(
			"strategy_001",
			"TestStrategy",
			"PLACE_BET",
			0.87,
			0.045,
			0.05,
			100,
			123456,
			3.5,
		)
		
		// 4. Bet placement
		auditLogger.LogBetPlacement(
			"bet_123",
			"strategy_001",
			"1.123456",
			456789,
			"BACK",
			100,
			3.5,
			time.Now(),
			false,
		)
		metrics.RecordBetPlaced()
		
		// 5. P&L update
		strategyLogger.LogStrategyPnLUpdate(
			"strategy_001",
			"TestStrategy",
			50,
			10550,
			1,
			0,
		)
		metrics.UpdateBankroll(10550)
		
		// Verify workflow completed successfully
		assert.True(t, true)
	})
	
	t.Run("concurrent metrics recording", func(t *testing.T) {
		// Test concurrent metric recording (race condition detection)
		done := make(chan bool)
		
		for i := 0; i < 10; i++ {
			go func(idx int) {
				_ = fmt.Sprintf("strategy_%03d", idx)
				metrics.RecordBetPlaced()
				metrics.RecordStrategyEvaluation(0.5)
				metrics.UpdateBankroll(10000.0 + float64(idx*100))
				done <- true
			}(i)
		}
		
		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
		
		assert.True(t, true)
	})
	
	t.Run("error handling", func(t *testing.T) {
		logBuf.Reset()
		
		// Log ML error
		mlLogger.LogMLPredictionError("strategy_predictor", "timeout")
		
		// Verify error is logged
		var logEntry map[string]interface{}
		err := json.Unmarshal(logBuf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "strategy_predictor", logEntry["model_type"])
	})
	
	t.Run("circuit breaker events", func(t *testing.T) {
		logBuf.Reset()
		
		// Log circuit breaker trip
		metrics.RecordCircuitBreakerTrip()
		auditLogger.LogCircuitBreakerEvent(
			"OPENED",
			"max_daily_loss_exceeded",
			map[string]interface{}{"daily_loss": -500, "threshold": -500},
			"PAUSE_TRADING",
		)
		
		// Verify event is logged
		var logEntry map[string]interface{}
		err := json.Unmarshal(logBuf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "OPENED", logEntry["event_type"])
	})
}

func BenchmarkObservabilitySystem(b *testing.B) {
	metrics.InitRegistry()
	
	appLog := logrus.New()
	appLog.SetOutput(&bytes.Buffer{})
	strategyLogger := logger.NewStrategyLogger(appLog)
	auditLogger := logger.NewAuditLogger(appLog)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		metrics.RecordBetPlaced()
		metrics.UpdateBankroll(10000.0)
		
		strategyLogger.LogStrategyDecision(
			"strategy_001", "TestStrategy", "PLACE_BET",
			0.87, 0.045, 0.05, 100, 123456, 3.5,
		)
		
		auditLogger.LogBetPlacement(
			"bet_123", "strategy_001", "1.123456", 456789,
			"BACK", 100, 3.5, time.Now(), false,
		)
	}
}

func TestMetricsRegistryRace(t *testing.T) {
	// Test for race conditions in metrics registry
	metrics.InitRegistry()
	
	done := make(chan bool)
	
	// Concurrent reads and writes
	for i := 0; i < 100; i++ {
		go func(idx int) {
			strategyID := fmt.Sprintf("strategy_%d", idx%10)
			metrics.RecordBetPlaced()
			metrics.UpdateBankroll(10000.0)
			metrics.RecordCircuitBreakerTrip()
			done <- true
		}(i)
	}
	
	for i := 0; i < 100; i++ {
		<-done
	}
	
	assert.True(t, true)
}

func TestTraceSegmentIntegration(t *testing.T) {
	appLog := logrus.New()
	appLog.SetOutput(&bytes.Buffer{})
	
	// Initialize X-Ray (disabled)
	tracing.Initialize(tracing.Config{
		ServiceName: "test-bot",
		Enabled:     false,
	}, appLog)
	
	// Create segment (should not panic even with X-Ray disabled)
	ctx, seg := tracing.StartSegment(context.Background(), "test-segment")
	assert.NotNil(t, ctx)
	assert.NotNil(t, seg)
	
	// Add metadata (should not panic)
	tracing.AddAnnotation(ctx, "test_key", "test_value")
	tracing.AddMetadata(ctx, "test_meta", map[string]string{"key": "value"})
	
	// Close segment
	seg.Close(nil)
	
	assert.True(t, true)
}
