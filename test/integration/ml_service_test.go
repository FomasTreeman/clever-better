//go:build integration

package integration

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/ml"
	mlpb "github.com/yourusername/clever-better/internal/ml/mlpb"
	"github.com/yourusername/clever-better/internal/models"
	"google.golang.org/grpc"
)

const skipIntegration = "Skipping integration test in short mode"

type mockMLService struct {
	mlpb.UnimplementedMLServiceServer

	getPrediction       func(context.Context, *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error)
	evaluateStrategy    func(context.Context, *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error)
	submitBacktest      func(context.Context, *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error)
	generateStrategy    func(context.Context, *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error)
	batchPredict        func(context.Context, *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error)
}

func (m *mockMLService) GetPrediction(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
	return m.getPrediction(ctx, req)
}

func (m *mockMLService) EvaluateStrategy(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
	return m.evaluateStrategy(ctx, req)
}

func (m *mockMLService) SubmitBacktestFeedback(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
	return m.submitBacktest(ctx, req)
}

func (m *mockMLService) GenerateStrategy(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
	return m.generateStrategy(ctx, req)
}

func (m *mockMLService) BatchPredict(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
	return m.batchPredict(ctx, req)
}

func startMockMLServer(t *testing.T, service *mockMLService) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	mlpb.RegisterMLServiceServer(grpcServer, service)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	cleanup := func() {
		grpcServer.Stop()
		_ = listener.Close()
	}

	return listener.Addr().String(), cleanup
}

func newMLClient(t *testing.T, addr string) *ml.MLClient {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.MLServiceConfig{
		URL:         "http://localhost",
		GRPCAddress: addr,
	}

	client, err := ml.NewMLClient(cfg, logger)
	require.NoError(t, err)
	return client
}

// TestMLPredictionEndpoints tests prediction endpoint via gRPC
func TestMLPredictionEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			require.NotEmpty(t, req.RaceId)
			require.NotEmpty(t, req.RunnerId)
			return &mlpb.PredictionResponse{
				RaceId:               req.RaceId,
				RunnerId:             req.RunnerId,
				PredictedProbability: 0.65,
				Confidence:           0.75,
				Recommendation:       "BACK",
				ModelVersion:         "v1",
			}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{CompositeScore: 0.8, Recommendation: "INCREASE"}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true, Message: "ok"}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{
				Strategies: []*mlpb.GeneratedStrategy{{
					StrategyId:     uuid.New().String(),
					Parameters:     map[string]float64{"alpha": 0.1},
					Confidence:     0.7,
					ExpectedReturn: 0.2,
					ExpectedSharpe: 1.1,
					ExpectedWinRate: 0.55,
				}},
			}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			responses := make([]*mlpb.SinglePredictionResponse, 0, len(req.Predictions))
			for _, pred := range req.Predictions {
				responses = append(responses, &mlpb.SinglePredictionResponse{
					RaceId:               pred.RaceId,
					RunnerId:             pred.RunnerId,
					PredictedProbability: 0.6,
					Confidence:           0.7,
					Recommendation:       "BACK",
					ModelVersion:         "v1",
				})
			}
			return &mlpb.BatchPredictionResponse{Predictions: responses}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)
	ctx := context.Background()

	raceID := uuid.New()
	runnerID := uuid.New()
	strategyID := uuid.New()

	result, err := client.GetPrediction(ctx, raceID, runnerID, strategyID, []float64{1.0, 2.0}, "")
	require.NoError(t, err)
	assert.Equal(t, raceID, result.RaceID)
	assert.Equal(t, runnerID, result.RunnerID)
	assert.Equal(t, 0.65, result.Probability)
	assert.Equal(t, 0.75, result.Confidence)
	assert.Equal(t, "BACK", result.Recommendation)
}

// TestMLEvaluateStrategy tests strategy evaluation over gRPC
func TestMLEvaluateStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			return &mlpb.PredictionResponse{}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{CompositeScore: 0.9, Recommendation: "INCREASE"}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			return &mlpb.BatchPredictionResponse{}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)

	score, recommendation, err := client.EvaluateStrategy(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 0.9, score)
	assert.Equal(t, "INCREASE", recommendation)
}

// TestMLBacktestFeedback tests feedback submission
func TestMLBacktestFeedback(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			return &mlpb.PredictionResponse{}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true, Message: "ok"}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			return &mlpb.BatchPredictionResponse{}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)

	result := &models.BacktestResult{
		StrategyID:     uuid.New(),
		CompositeScore: 0.8,
		SharpeRatio:    1.2,
		TotalReturn:    0.15,
		MaxDrawdown:    0.1,
		WinRate:        0.55,
		ProfitFactor:   1.3,
		TotalBets:      100,
		Method:         "walk_forward",
		MLFeatures:     []byte(`{"feature": 1.0}`),
	}

	err := client.SubmitBacktestFeedback(context.Background(), result)
	require.NoError(t, err)
}

// TestMLStrategyGeneration tests strategy generation via gRPC
func TestMLStrategyGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			return &mlpb.PredictionResponse{}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{
				Strategies: []*mlpb.GeneratedStrategy{{
					StrategyId:      uuid.New().String(),
					Parameters:      map[string]float64{"alpha": 0.2},
					Confidence:      0.8,
					ExpectedReturn:  0.25,
					ExpectedSharpe:  1.4,
					ExpectedWinRate: 0.6,
				}},
			}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			return &mlpb.BatchPredictionResponse{}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)

	strategies, err := client.GenerateStrategy(context.Background(), ml.StrategyConstraints{
		RiskLevel:    "medium",
		TargetReturn: 0.2,
		MaxDrawdownLimit: 0.1,
		MinWinRate:   0.5,
		MaxCandidates: 1,
	})
	require.NoError(t, err)
	require.Len(t, strategies, 1)
	assert.InDelta(t, 0.8, strategies[0].Confidence, 0.001)
}

// TestMLBatchPredict tests batch prediction handling
func TestMLBatchPredict(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			return &mlpb.PredictionResponse{}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			responses := make([]*mlpb.SinglePredictionResponse, 0, len(req.Predictions))
			for _, pred := range req.Predictions {
				responses = append(responses, &mlpb.SinglePredictionResponse{
					RaceId:               pred.RaceId,
					RunnerId:             pred.RunnerId,
					PredictedProbability: 0.6,
					Confidence:           0.7,
					Recommendation:       "BACK",
					ModelVersion:         "v1",
				})
			}
			return &mlpb.BatchPredictionResponse{Predictions: responses}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)

	requests := []ml.PredictionRequest{
		{RaceID: uuid.New(), RunnerID: uuid.New(), StrategyID: uuid.New(), Features: []float64{1, 2}},
		{RaceID: uuid.New(), RunnerID: uuid.New(), StrategyID: uuid.New(), Features: []float64{3, 4}},
	}

	results, err := client.BatchPredict(context.Background(), requests)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "BACK", results[0].Recommendation)
}

// TestMLGRPCErrorHandling tests gRPC error propagation
func TestMLGRPCErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			return nil, assert.AnError
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			return &mlpb.BatchPredictionResponse{}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)

	_, err := client.GetPrediction(context.Background(), uuid.New(), uuid.New(), uuid.New(), []float64{1.0}, "latest")
	assert.Error(t, err)
}

// TestConcurrentPredictions tests concurrent prediction requests
func TestConcurrentPredictions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var requestCount int32
	mu := sync.Mutex{}

	service := &mockMLService{
		getPrediction: func(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
			atomic.AddInt32(&requestCount, 1)
			mu.Lock()
			defer mu.Unlock()
			return &mlpb.PredictionResponse{
				RaceId:               req.RaceId,
				RunnerId:             req.RunnerId,
				PredictedProbability: 0.6,
				Confidence:           0.7,
				Recommendation:       "BACK",
				ModelVersion:         "v1",
			}, nil
		},
		evaluateStrategy: func(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
			return &mlpb.StrategyResponse{}, nil
		},
		submitBacktest: func(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
			return &mlpb.BacktestFeedbackResponse{Success: true}, nil
		},
		generateStrategy: func(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
			return &mlpb.StrategyGenerationResponse{}, nil
		},
		batchPredict: func(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
			return &mlpb.BatchPredictionResponse{}, nil
		},
	}

	addr, cleanup := startMockMLServer(t, service)
	defer cleanup()

	client := newMLClient(t, addr)
	ctx := context.Background()

	concurrency := 10
	results := make(chan error, concurrency)

	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := client.GetPrediction(ctx, uuid.New(), uuid.New(), uuid.New(), []float64{1.0}, "latest")
			results <- err
		}()
	}

	for i := 0; i < concurrency; i++ {
		require.NoError(t, <-results)
	}

	duration := time.Since(startTime)
	assert.Equal(t, int32(concurrency), atomic.LoadInt32(&requestCount))
	t.Logf("✓ Concurrent predictions validated (%d requests in %v)", concurrency, duration)
}
