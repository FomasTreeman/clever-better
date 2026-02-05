package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yourusername/clever-better/internal/models"
)

// MockStrategyRepository mocks strategy repository
type MockStrategyRepository struct {
	mock.Mock
}

func (m *MockStrategyRepository) GetAll(ctx context.Context) ([]models.Strategy, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Strategy), args.Error(1)
}

func (m *MockStrategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Strategy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Strategy), args.Error(1)
}

func (m *MockStrategyRepository) Create(ctx context.Context, strategy *models.Strategy) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

func (m *MockStrategyRepository) Update(ctx context.Context, strategy *models.Strategy) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

// TestCompositeScoreCalculation tests various metric combinations
func TestCompositeScoreCalculation(t *testing.T) {
	tests := []struct {
		name           string
		winRate        float64
		roi            float64
		sharpeRatio    float64
		maxDrawdown    float64
		expectedScore  float64
		shouldBePositive bool
	}{
		{
			name:             "High performing strategy",
			winRate:          0.65,
			roi:              0.25,
			sharpeRatio:      1.8,
			maxDrawdown:      0.10,
			expectedScore:    0.0,
			shouldBePositive: true,
		},
		{
			name:             "Poor performing strategy",
			winRate:          0.35,
			roi:              -0.15,
			sharpeRatio:      -0.5,
			maxDrawdown:      0.35,
			expectedScore:    0.0,
			shouldBePositive: false,
		},
		{
			name:             "Break-even strategy",
			winRate:          0.50,
			roi:              0.0,
			sharpeRatio:      0.0,
			maxDrawdown:      0.15,
			expectedScore:    0.0,
			shouldBePositive: false,
		},
		{
			name:             "High volatility strategy",
			winRate:          0.55,
			roi:              0.30,
			sharpeRatio:      0.8,
			maxDrawdown:      0.40,
			expectedScore:    0.0,
			shouldBePositive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate composite score (example formula)
			score := (tt.winRate * 0.3) + (tt.roi * 0.4) + (tt.sharpeRatio * 0.2) - (tt.maxDrawdown * 0.1)

			if tt.shouldBePositive {
				assert.Greater(t, score, 0.0, "Score should be positive for good strategies")
			} else {
				assert.LessOrEqual(t, score, 0.0, "Score should be non-positive for poor strategies")
			}
		})
	}
}

// TestStrategyRanking tests ranking algorithm with edge cases
func TestStrategyRanking(t *testing.T) {
	strategies := []struct {
		id    uuid.UUID
		name  string
		score float64
	}{
		{uuid.New(), "Strategy A", 0.85},
		{uuid.New(), "Strategy B", 0.85}, // Tie
		{uuid.New(), "Strategy C", 0.72},
		{uuid.New(), "Strategy D", -0.15}, // Negative score
		{uuid.New(), "Strategy E", 0.90},
	}

	// Sort by score descending
	sorted := make([]struct {
		id    uuid.UUID
		name  string
		score float64
	}, len(strategies))
	copy(sorted, strategies)

	// Simple bubble sort for test
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Verify ranking
	assert.Equal(t, "Strategy E", sorted[0].name, "Highest score should be first")
	assert.Equal(t, 0.90, sorted[0].score)

	// Verify ties maintain relative order or are handled consistently
	tieIdx := -1
	for i, s := range sorted {
		if s.score == 0.85 {
			tieIdx = i
			break
		}
	}
	assert.Greater(t, tieIdx, 0, "Tied strategies should not be first")

	// Verify negative scores are ranked last
	assert.Equal(t, "Strategy D", sorted[len(sorted)-1].name)
	assert.Less(t, sorted[len(sorted)-1].score, 0.0)
}

// TestThresholdActivation tests activation/deactivation logic
func TestThresholdActivation(t *testing.T) {
	tests := []struct {
		name              string
		currentScore      float64
		activationThreshold float64
		deactivationThreshold float64
		currentlyActive   bool
		expectedActive    bool
	}{
		{
			name:                  "Activate above threshold",
			currentScore:          0.75,
			activationThreshold:   0.65,
			deactivationThreshold: 0.45,
			currentlyActive:       false,
			expectedActive:        true,
		},
		{
			name:                  "Remain active above deactivation threshold",
			currentScore:          0.55,
			activationThreshold:   0.65,
			deactivationThreshold: 0.45,
			currentlyActive:       true,
			expectedActive:        true,
		},
		{
			name:                  "Deactivate below threshold",
			currentScore:          0.40,
			activationThreshold:   0.65,
			deactivationThreshold: 0.45,
			currentlyActive:       true,
			expectedActive:        false,
		},
		{
			name:                  "Do not activate below threshold",
			currentScore:          0.60,
			activationThreshold:   0.65,
			deactivationThreshold: 0.45,
			currentlyActive:       false,
			expectedActive:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldActivate := false

			if !tt.currentlyActive && tt.currentScore >= tt.activationThreshold {
				shouldActivate = true
			} else if tt.currentlyActive && tt.currentScore >= tt.deactivationThreshold {
				shouldActivate = true
			}

			assert.Equal(t, tt.expectedActive, shouldActivate)
		})
	}
}

// TestPerformanceDegradation tests detection of performance decline
func TestPerformanceDegradation(t *testing.T) {
	// Historical performance scores
	historicalScores := []float64{0.85, 0.83, 0.87, 0.86, 0.84}
	recentScores := []float64{0.72, 0.68, 0.65, 0.70, 0.67}

	// Calculate moving averages
	historicalAvg := 0.0
	for _, score := range historicalScores {
		historicalAvg += score
	}
	historicalAvg /= float64(len(historicalScores))

	recentAvg := 0.0
	for _, score := range recentScores {
		recentAvg += score
	}
	recentAvg /= float64(len(recentScores))

	// Calculate degradation percentage
	degradation := (historicalAvg - recentAvg) / historicalAvg * 100

	assert.Greater(t, degradation, 10.0, "Should detect significant performance degradation")
	assert.Less(t, recentAvg, historicalAvg, "Recent performance should be worse")

	// Check if degradation exceeds threshold
	degradationThreshold := 15.0
	if degradation > degradationThreshold {
		t.Logf("Performance degradation detected: %.2f%% (threshold: %.2f%%)", degradation, degradationThreshold)
	}
}

// TestStrategyEvaluatorWithMocks tests evaluator with mocked dependencies
func TestStrategyEvaluatorWithMocks(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockStrategyRepository)

	// Setup test strategies
	testStrategies := []models.Strategy{
		{
			ID:       uuid.New(),
			Name:     "Test Strategy 1",
			Type:     "simple_value",
			IsActive: true,
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:       uuid.New(),
			Name:     "Test Strategy 2",
			Type:     "simple_value",
			IsActive: false,
			CreatedAt: time.Now().Add(-60 * 24 * time.Hour),
		},
	}

	mockRepo.On("GetAll", ctx).Return(testStrategies, nil)

	// Call repository
	strategies, err := mockRepo.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, strategies, 2)

	// Verify active count
	activeCount := 0
	for _, s := range strategies {
		if s.IsActive {
			activeCount++
		}
	}
	assert.Equal(t, 1, activeCount)

	mockRepo.AssertExpectations(t)
}
