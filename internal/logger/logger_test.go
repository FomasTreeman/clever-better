package logger

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLogger() (*logrus.Logger, *bytes.Buffer) {
	log := logrus.New()
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.DebugLevel)
	return log, buf
}

func parseLogOutput(buf *bytes.Buffer) map[string]interface{} {
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		return nil
	}
	return logEntry
}

func TestStrategyLoggerEvaluation(t *testing.T) {
	log, buf := setupTestLogger()
	strategyLogger := NewStrategyLogger(log)
	
	strategyLogger.LogStrategyEvaluation(
		"strategy_001",
		"TestStrategy",
		"race_123",
		100,
		2,
		500.0,
	)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "strategy_001", logEntry["strategy_id"])
	assert.Equal(t, "strategy", logEntry["component"])
}

func TestStrategyLoggerDecision(t *testing.T) {
	log, buf := setupTestLogger()
	strategyLogger := NewStrategyLogger(log)
	
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
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "PLACE_BET", logEntry["decision"])
}

func TestStrategyLoggerActivation(t *testing.T) {
	log, buf := setupTestLogger()
	strategyLogger := NewStrategyLogger(log)
	
	strategyLogger.LogStrategyActivation("strategy_001", "TestStrategy", "backtest_approval")
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "strategy_001", logEntry["strategy_id"])
	assert.Equal(t, "activation", logEntry["event_type"])
}

func TestStrategyLoggerDeactivation(t *testing.T) {
	log, buf := setupTestLogger()
	strategyLogger := NewStrategyLogger(log)
	
	strategyLogger.LogStrategyDeactivation("strategy_001", "TestStrategy", "circuit_breaker_trip")
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "deactivation", logEntry["event_type"])
	assert.Equal(t, "circuit_breaker_trip", logEntry["reason"])
}

func TestMLLoggerPredictionRequest(t *testing.T) {
	log, buf := setupTestLogger()
	mlLogger := NewMLLogger(log)
	
	mlLogger.LogMLPredictionRequest("strategy_predictor", 50, true, 45)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "strategy_predictor", logEntry["model_type"])
}

func TestMLLoggerStrategyGeneration(t *testing.T) {
	log, buf := setupTestLogger()
	mlLogger := NewMLLogger(log)
	
	mlLogger.LogStrategyGeneration(
		map[string]interface{}{"type": "value"},
		10,
		0.856,
		42,
	)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, float64(10), logEntry["candidates_evaluated"])
}

func TestMLLoggerModelTraining(t *testing.T) {
	log, buf := setupTestLogger()
	mlLogger := NewMLLogger(log)
	
	mlLogger.LogModelTraining(
		"strategy_predictor_v2",
		120.5,
		map[string]float64{"accuracy": 0.847, "precision": 0.821},
		map[string]interface{}{"learning_rate": 0.001, "epochs": 100},
	)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "strategy_predictor_v2", logEntry["model_name"])
}

func TestAuditLoggerBetPlacement(t *testing.T) {
	log, buf := setupTestLogger()
	auditLogger := NewAuditLogger(log)
	
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
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "bet_123", logEntry["bet_id"])
	assert.Equal(t, false, logEntry["paper_trading"])
}

func TestAuditLoggerStrategyParameterChange(t *testing.T) {
	log, buf := setupTestLogger()
	auditLogger := NewAuditLogger(log)
	
	auditLogger.LogStrategyParameterChange(
		"strategy_001",
		"max_stake",
		100,
		150,
		"user@example.com",
	)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "max_stake", logEntry["parameter_name"])
}

func TestAuditLoggerCircuitBreakerEvent(t *testing.T) {
	log, buf := setupTestLogger()
	auditLogger := NewAuditLogger(log)
	
	auditLogger.LogCircuitBreakerEvent(
		"OPENED",
		"max_daily_loss_exceeded",
		map[string]interface{}{"daily_loss": -500, "threshold": -500},
		"PAUSE_TRADING",
	)
	
	logEntry := parseLogOutput(buf)
	require.NotNil(t, logEntry)
	assert.Equal(t, "OPENED", logEntry["event_type"])
}

func TestLoggerJSONFormat(t *testing.T) {
	log, buf := setupTestLogger()
	strategyLogger := NewStrategyLogger(log)
	
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
	
	// Verify output is valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, logEntry)
}

func BenchmarkStrategyLoggerDecision(b *testing.B) {
	log := logrus.New()
	log.SetOutput(&bytes.Buffer{})
	strategyLogger := NewStrategyLogger(log)
	
	for i := 0; i < b.N; i++ {
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
	}
}

func BenchmarkAuditLoggerBetPlacement(b *testing.B) {
	log := logrus.New()
	log.SetOutput(&bytes.Buffer{})
	auditLogger := NewAuditLogger(log)
	
	for i := 0; i < b.N; i++ {
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
	}
}
