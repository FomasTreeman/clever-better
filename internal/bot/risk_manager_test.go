package bot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/models"
)

// MockBetRepository is a mock implementation of BetRepository
type MockBetRepository struct {
	mock.Mock
}

func (m *MockBetRepository) Create(ctx context.Context, bet *models.Bet) error {
	args := m.Called(ctx, bet)
	return args.Error(0)
}

func (m *MockBetRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Bet, error) {
	args := m.Called(ctx, raceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.Bet, error) {
	args := m.Called(ctx, strategyID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

func (m *MockBetRepository) Update(ctx context.Context, bet *models.Bet) error {
	if bet != nil {
		_ = bet.UpdatedAt
	}
	args := m.Called(ctx, bet)
	return args.Error(0)
}

func (m *MockBetRepository) GetPendingBets(ctx context.Context) ([]*models.Bet, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetSettledBets(ctx context.Context, start, end time.Time) ([]*models.Bet, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetByBetfairBetID(ctx context.Context, betID string) (*models.Bet, error) {
	args := m.Called(ctx, betID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bet), args.Error(1)
}

func TestCalculatePositionSize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	tests := []struct {
		name           string
		odds           float64
		bankroll       float64
		confidence     float64
		edgeEstimate   float64
		expectedStake  float64
		expectZero     bool
	}{
		{
			name:          "High confidence positive edge",
			odds:          3.0,
			bankroll:      1000.0,
			confidence:    0.8,
			edgeEstimate:  0.1,
			expectedStake: 50.0, // Will be capped by max stake
		},
		{
			name:          "Low confidence",
			odds:          2.0,
			bankroll:      1000.0,
			confidence:    0.3,
			edgeEstimate:  0.05,
			expectedStake: 0,
			expectZero:    true,
		},
		{
			name:          "Negative edge",
			odds:          1.5,
			bankroll:      1000.0,
			confidence:    0.4,
			edgeEstimate:  -0.1,
			expectedStake: 0,
			expectZero:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stake, err := rm.CalculatePositionSize(tt.odds, tt.bankroll, tt.confidence, tt.edgeEstimate)
			assert.NoError(t, err)

			if tt.expectZero {
				assert.Equal(t, 0.0, stake, "Expected zero stake")
			} else {
				assert.Greater(t, stake, 0.0, "Expected positive stake")
				assert.LessOrEqual(t, stake, cfg.MaxStakePerBet, "Stake should not exceed max")
			}
		})
	}
}

func TestCheckRiskLimits(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()

	// Test max stake exceeded
	err := rm.CheckRiskLimits(ctx, 150.0)
	assert.Error(t, err, "Should reject stake exceeding max")

	// Test exposure limit
	rm.currentExposure = 480.0
	err = rm.CheckRiskLimits(ctx, 30.0)
	assert.Error(t, err, "Should reject stake exceeding max exposure")

	// Test daily loss limit
	rm.currentExposure = 0
	rm.dailyLoss = 250.0
	err = rm.CheckRiskLimits(ctx, 10.0)
	assert.Error(t, err, "Should reject when daily loss limit reached")

	// Test valid stake
	rm.dailyLoss = 0
	err = rm.CheckRiskLimits(ctx, 50.0)
	assert.NoError(t, err, "Should accept valid stake")
}

func TestUpdateExposure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()

	// Mock pending bets
	pendingBets := []*models.Bet{
		{ID: uuid.New(), Stake: 50.0, Status: models.BetStatusPending},
		{ID: uuid.New(), Stake: 75.0, Status: models.BetStatusPending},
		{ID: uuid.New(), Stake: 100.0, Status: models.BetStatusPending},
	}

	mockRepo.On("GetPendingBets", ctx).Return(pendingBets, nil)

	err := rm.UpdateExposure(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 225.0, rm.currentExposure, "Exposure should be sum of pending bets")

	mockRepo.AssertExpectations(t)
}

func TestUpdateDailyLoss(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()

	// Mock settled bets with losses
	pl1 := -50.0
	pl2 := -75.0
	pl3 := 30.0 // Win

	settledBets := []*models.Bet{
		{ID: uuid.New(), ProfitLoss: &pl1, Status: models.BetStatusSettled},
		{ID: uuid.New(), ProfitLoss: &pl2, Status: models.BetStatusSettled},
		{ID: uuid.New(), ProfitLoss: &pl3, Status: models.BetStatusSettled},
	}

	mockRepo.On("GetSettledBets", ctx, mock.Anything, mock.Anything).Return(settledBets, nil)

	err := rm.UpdateDailyLoss(ctx)
	assert.NoError(t, err)

	// Total P&L = -50 - 75 + 30 = -95
	// Daily loss = 95
	assert.Equal(t, 95.0, rm.dailyLoss, "Daily loss should be absolute value of negative P&L")

	mockRepo.AssertExpectations(t)
}

func TestDailyLossReset(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	// Set daily loss and reset time in the past
	rm.dailyLoss = 100.0
	rm.dailyLossResetTime = time.Now().Add(-1 * time.Hour)

	// Mock empty settled bets
	mockRepo.On("GetSettledBets", mock.Anything, mock.Anything, mock.Anything).Return([]*models.Bet{}, nil)

	// Check risk limits should trigger reset
	ctx := context.Background()
	_ = rm.CheckRiskLimits(ctx, 10.0)

	// Verify reset time was updated
	assert.True(t, rm.dailyLossResetTime.After(time.Now()), "Reset time should be in the future")
}

func TestIsWithinLimits(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	// Within limits
	assert.True(t, rm.IsWithinLimits())

	// Exposure limit reached
	rm.currentExposure = 500.0
	assert.False(t, rm.IsWithinLimits())

	// Daily loss limit reached
	rm.currentExposure = 0
	rm.dailyLoss = 200.0
	assert.False(t, rm.IsWithinLimits())

	// Back within limits
	rm.dailyLoss = 0
	assert.True(t, rm.IsWithinLimits())
}

func TestGetRiskMetrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	rm.currentExposure = 250.0
	rm.dailyLoss = 50.0

	metrics := rm.GetRiskMetrics()

	assert.Equal(t, 250.0, metrics.CurrentExposure)
	assert.Equal(t, 50.0, metrics.DailyLoss)
	assert.Equal(t, 500.0, metrics.MaxExposure)
	assert.Equal(t, 200.0, metrics.MaxDailyLoss)
	assert.Equal(t, 250.0, metrics.RemainingCapacity) // 500 - 250
}

func TestCheckRiskLimitsTable(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)
	rm.dailyLossResetTime = time.Now().Add(1 * time.Hour)

	tests := []struct {
		name          string
		currentExposure float64
		dailyLoss     float64
		stake         float64
		expectError   bool
	}{
		{
			name:            "Exceeds max stake",
			currentExposure: 0,
			dailyLoss:       0,
			stake:           150.0,
			expectError:     true,
		},
		{
			name:            "Exceeds max exposure",
			currentExposure: 480.0,
			dailyLoss:       0,
			stake:           30.0,
			expectError:     true,
		},
		{
			name:            "Exceeds daily loss",
			currentExposure: 0,
			dailyLoss:       250.0,
			stake:           10.0,
			expectError:     true,
		},
		{
			name:            "Within limits",
			currentExposure: 100.0,
			dailyLoss:       50.0,
			stake:           25.0,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm.currentExposure = tt.currentExposure
			rm.dailyLoss = tt.dailyLoss

			err := rm.CheckRiskLimits(context.Background(), tt.stake)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatePositionSizeMinimumStake(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	stake, err := rm.CalculatePositionSize(2.0, 50.0, 0.55, 0.1)
	require.NoError(t, err)
	assert.Equal(t, 0.0, stake, "stake below minimum should be zero")
}

func TestUpdateExposureError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()
	mockRepo.On("GetPendingBets", ctx).Return(nil, assert.AnError)

	err := rm.UpdateExposure(ctx)
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateDailyLossError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()
	mockRepo.On("GetSettledBets", ctx, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	err := rm.UpdateDailyLoss(ctx)
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCheckRiskLimitsResetsDailyLoss(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.TradingConfig{
		MaxStakePerBet: 100.0,
		MaxExposure:    500.0,
		MaxDailyLoss:   200.0,
	}

	mockRepo := new(MockBetRepository)
	rm := NewRiskManager(cfg, mockRepo, logger)

	ctx := context.Background()
	rm.dailyLossResetTime = time.Now().Add(-1 * time.Hour)

	mockRepo.On("GetSettledBets", ctx, mock.Anything, mock.Anything).Return([]*models.Bet{}, nil)

	err := rm.CheckRiskLimits(ctx, 10.0)
	assert.NoError(t, err)
	assert.True(t, rm.dailyLossResetTime.After(time.Now()))
	mockRepo.AssertExpectations(t)
}
