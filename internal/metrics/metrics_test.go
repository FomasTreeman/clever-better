package metrics

import (
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetricsRegistry(t *testing.T) {
	// Initialize the registry
	InitRegistry()
	registry := GetRegistry()
	
	assert.NotNil(t, registry)
	assert.IsType(t, &prometheus.Registry{}, registry)
}

func TestRecordBetPlaced(t *testing.T) {
	InitRegistry()

	assert.NotPanics(t, func() {
		RecordBetPlaced()
	})
}

func TestRecordStrategyEvaluation(t *testing.T) {
	InitRegistry()
	durationSeconds := 0.5

	assert.NotPanics(t, func() {
		RecordStrategyEvaluation(durationSeconds)
	})
}

func TestUpdateBankroll(t *testing.T) {
	InitRegistry()
	
	tests := []struct {
		name      string
		bankroll  float64
		shouldErr bool
	}{
		{
			name:      "positive bankroll",
			bankroll:  10000,
			shouldErr: false,
		},
		{
			name:      "zero bankroll",
			bankroll:  0,
			shouldErr: false,
		},
		{
			name:      "negative bankroll",
			bankroll:  -100,
			shouldErr: false, // Should still record
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				UpdateBankroll(tt.bankroll)
			})
		})
	}
}

func TestUpdateExposure(t *testing.T) {
	InitRegistry()
	
	tests := []struct {
		name      string
		exposure  float64
		shouldErr bool
	}{
		{
			name:      "normal exposure",
			exposure:  5000,
			shouldErr: false,
		},
		{
			name:      "high exposure",
			exposure:  50000,
			shouldErr: false,
		},
		{
			name:      "zero exposure",
			exposure:  0,
			shouldErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				UpdateExposure(tt.exposure)
			})
		})
	}
}

func TestRecordCircuitBreakerTrip(t *testing.T) {
	InitRegistry()

	assert.NotPanics(t, func() {
		RecordCircuitBreakerTrip()
	})
}

func TestMetricsHandler(t *testing.T) {
	InitRegistry()

	handler := Handler()
	assert.NotNil(t, handler)
	assert.Implements(t, (*http.Handler)(nil), handler)
}

func TestStrategyMetrics(t *testing.T) {
	InitRegistry()

	strategyID := "strategy_001"
	strategyName := "TestStrategy"

	assert.NotPanics(t, func() {
		RecordStrategyDecision(strategyID, strategyName, "PLACE_BET", "success")
	})

	assert.NotPanics(t, func() {
		RecordStrategyConfidence(strategyID, strategyName, 0.92)
	})

	assert.NotPanics(t, func() {
		UpdateStrategyActiveBets(strategyID, strategyName, 5)
	})
}

func TestBacktestMetrics(t *testing.T) {
	InitRegistry()

	strategyID := "strategy_001"
	method := "historical_replay"

	assert.NotPanics(t, func() {
		RecordBacktestRun(method, "success")
	})

	assert.NotPanics(t, func() {
		RecordCompositeScore(strategyID, method, 0.856)
	})

	assert.NotPanics(t, func() {
		UpdateAggregatedScore(strategyID, 0.856)
	})
}

func BenchmarkRecordBetPlaced(b *testing.B) {
	InitRegistry()
	
	for i := 0; i < b.N; i++ {
		RecordBetPlaced("strategy_001", "TestStrategy")
	}
}

func BenchmarkUpdateBankroll(b *testing.B) {
	InitRegistry()
	
	for i := 0; i < b.N; i++ {
		UpdateBankroll(10000.0)
	}
}

func BenchmarkRecordStrategyEvaluation(b *testing.B) {
	InitRegistry()
	
	for i := 0; i < b.N; i++ {
		RecordStrategyEvaluation("strategy_001", 0.5)
	}
}
