// Package ml provides gRPC client for ML service.
package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/models"
	mlpb "github.com/yourusername/clever-better/internal/ml/mlpb"
)

// MLClient provides gRPC client for ML service
type MLClient struct {
	conn   *grpc.ClientConn
	client mlpb.MLServiceClient
	config *config.MLServiceConfig
	logger *logrus.Logger
}

// NewMLClient creates a new ML service client
func NewMLClient(cfg *config.MLServiceConfig, logger *logrus.Logger) (*MLClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if strings.HasPrefix(cfg.URL, "https://") {
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}

	connectParams := grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  1 * time.Second,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   5 * time.Second,
		},
		MinConnectTimeout: 10 * time.Second,
	}

	keepAlive := keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	}

	// Establish gRPC connection with retry
	conn, err := grpc.DialContext(ctx, cfg.GRPCAddress,
		creds,
		grpc.WithBlock(),
		grpc.WithConnectParams(connectParams),
		grpc.WithKeepaliveParams(keepAlive),
	)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to ML service")
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	client := &MLClient{
		conn:   conn,
		client: mlpb.NewMLServiceClient(conn),
		config: cfg,
		logger: logger,
	}

	logger.WithField("address", cfg.GRPCAddress).Info("Connected to ML service")
	return client, nil
}

// GetPrediction gets a prediction from the ML service
func (c *MLClient) GetPrediction(ctx context.Context, raceID, runnerID, strategyID uuid.UUID, features []float64, modelVersion string) (*PredictionResult, error) {
	start := time.Now()
	defer func() {
		MLPredictionLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	if modelVersion == "" {
		modelVersion = "latest"
	}

	// Create gRPC request with proper field population
	req := &mlpb.PredictionRequest{
		RaceId:       raceID.String(),
		RunnerId:     runnerID.String(),
		StrategyId:   strategyID.String(),
		Features:     features,
		ModelVersion: modelVersion,
	}

	// Make actual RPC call
	resp, err := c.client.GetPrediction(ctx, req)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("GetPrediction", "rpc_failed").Inc()
		c.logger.WithError(err).Error("Failed to get prediction from ML service")
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrediction, err)
	}

	// Map response to internal type
	result := &PredictionResult{
		RaceID:         raceID,
		RunnerID:       runnerID,
		StrategyID:     strategyID,
		Probability:    resp.PredictedProbability,
		Confidence:     resp.Confidence,
		Recommendation: resp.Recommendation,
		PredictedAt:    time.Now(),
		ModelVersion:   resp.ModelVersion,
	}

	if resp.RaceId != "" {
		if parsed, err := uuid.Parse(resp.RaceId); err == nil {
			result.RaceID = parsed
		}
	}
	if resp.RunnerId != "" {
		if parsed, err := uuid.Parse(resp.RunnerId); err == nil {
			result.RunnerID = parsed
		}
	}

	MLPredictionsTotal.WithLabelValues("grpc", "false").Inc()
	return result, nil
}

// EvaluateStrategy evaluates a strategy using ML service
func (c *MLClient) EvaluateStrategy(ctx context.Context, strategyID uuid.UUID) (float64, string, error) {
	start := time.Now()
	defer func() {
		MLPredictionLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	req := &mlpb.StrategyRequest{
		StrategyId: strategyID.String(),
	}

	resp, err := c.client.EvaluateStrategy(ctx, req)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("EvaluateStrategy", "rpc_failed").Inc()
		c.logger.WithError(err).Error("Failed to evaluate strategy from ML service")
		return 0, "", fmt.Errorf("%w: %v", ErrInvalidPrediction, err)
	}

	return resp.CompositeScore, resp.Recommendation, nil
}

// SubmitBacktestFeedback submits backtest results as feedback to ML service
func (c *MLClient) SubmitBacktestFeedback(ctx context.Context, result *models.BacktestResult) error {
	start := time.Now()
	defer func() {
		MLPredictionLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	mlFeatures := parseMLFeatures(result.MLFeatures)

	// Populate request with backtest result fields
	req := &mlpb.BacktestFeedbackRequest{
		StrategyId:     result.StrategyID.String(),
		CompositeScore: result.CompositeScore,
		SharpeRatio:    result.SharpeRatio,
		Roi:            result.TotalReturn,
		MaxDrawdown:    result.MaxDrawdown,
		WinRate:        result.WinRate,
		ProfitFactor:   result.ProfitFactor,
		TotalBets:      int32(result.TotalBets),
		Method:         result.Method,
		MlFeatures:     mlFeatures,
	}

	resp, err := c.client.SubmitBacktestFeedback(ctx, req)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("SubmitBacktestFeedback", "rpc_failed").Inc()
		c.logger.WithError(err).Error("Failed to submit backtest feedback")
		return fmt.Errorf("%w: %v", ErrFeedbackSubmissionFailed, err)
	}

	if !resp.Success {
		MLGRPCErrorsTotal.WithLabelValues("SubmitBacktestFeedback", "failed").Inc()
		return fmt.Errorf("%w: %s", ErrFeedbackSubmissionFailed, resp.Message)
	}

	MLFeedbackSubmittedTotal.Inc()
	c.logger.WithField("strategy_id", result.StrategyID).Debug("Successfully submitted feedback")
	return nil
}

// GenerateStrategy generates a new strategy using ML
func (c *MLClient) GenerateStrategy(ctx context.Context, constraints StrategyConstraints) ([]*GeneratedStrategy, error) {
	start := time.Now()
	defer func() {
		MLPredictionLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	req := &mlpb.StrategyGenerationRequest{
		RiskLevel:          constraints.RiskLevel,
		TargetReturn:       constraints.TargetReturn,
		MaxDrawdownLimit:   constraints.MaxDrawdownLimit,
		MinWinRate:         constraints.MinWinRate,
		MaxCandidates:      int32(constraints.MaxCandidates),
		AggregatedFeatures: constraints.AggregatedFeatures,
		TopMetrics:         constraints.TopMetrics,
	}

	resp, err := c.client.GenerateStrategy(ctx, req)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("GenerateStrategy", "rpc_failed").Inc()
		c.logger.WithError(err).Error("Failed to generate strategy from ML service")
		return nil, fmt.Errorf("%w: %v", ErrStrategyGenerationFailed, err)
	}

	// Map proto responses to internal types
	strategies := make([]*GeneratedStrategy, len(resp.Strategies))
	for i, protoStrat := range resp.Strategies {
		stratID, _ := uuid.Parse(protoStrat.StrategyId)
		strategies[i] = &GeneratedStrategy{
			StrategyID:      stratID,
			Parameters:      protoStrat.Parameters,
			Confidence:      protoStrat.Confidence,
			ExpectedReturn:  protoStrat.ExpectedReturn,
			ExpectedSharpe:  protoStrat.ExpectedSharpe,
			ExpectedWinRate: protoStrat.ExpectedWinRate,
			GeneratedAt:     time.Now(),
		}
	}

	MLStrategyGenerationTotal.WithLabelValues("success").Inc()
	c.logger.WithField("count", len(strategies)).Debug("Generated strategies from ML service")
	return strategies, nil
}

// BatchPredict performs bulk predictions
func (c *MLClient) BatchPredict(ctx context.Context, requests []PredictionRequest) ([]*PredictionResult, error) {
	start := time.Now()
	defer func() {
		MLPredictionLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	// Convert to proto requests
	protoRequests := make([]*mlpb.SinglePredictionRequest, len(requests))
	for i, req := range requests {
		protoRequests[i] = &mlpb.SinglePredictionRequest{
			RaceId:     req.RaceID.String(),
			RunnerId:   req.RunnerID.String(),
			StrategyId: req.StrategyID.String(),
			Features:   req.Features,
		}
	}

	batchReq := &mlpb.BatchPredictionRequest{
		Predictions: protoRequests,
	}

	resp, err := c.client.BatchPredict(ctx, batchReq)
	if err != nil {
		MLGRPCErrorsTotal.WithLabelValues("BatchPredict", "rpc_failed").Inc()
		c.logger.WithError(err).Error("Failed to batch predict from ML service")
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrediction, err)
	}

	// Map proto responses to internal types
	results := make([]*PredictionResult, len(resp.Predictions))
	for i, protoResult := range resp.Predictions {
		req := requests[i]
		raceID := req.RaceID
		runnerID := req.RunnerID
		if protoResult.RaceId != "" {
			if parsed, err := uuid.Parse(protoResult.RaceId); err == nil {
				raceID = parsed
			}
		}
		if protoResult.RunnerId != "" {
			if parsed, err := uuid.Parse(protoResult.RunnerId); err == nil {
				runnerID = parsed
			}
		}

		results[i] = &PredictionResult{
			RaceID:         raceID,
			RunnerID:       runnerID,
			StrategyID:     req.StrategyID,
			Probability:    protoResult.PredictedProbability,
			Confidence:     protoResult.Confidence,
			Recommendation: protoResult.Recommendation,
			PredictedAt:    time.Now(),
			ModelVersion:   req.ModelVersion,
		}
	}

	MLPredictionsTotal.WithLabelValues("grpc", "false").Add(float64(len(results)))
	c.logger.WithField("count", len(results)).Debug("Batch predictions completed")
	return results, nil
}

// Close closes the gRPC connection
func (c *MLClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Helper functions for type conversion between gRPC and internal types

func parseMLFeatures(raw json.RawMessage) map[string]float64 {
	if len(raw) == 0 {
		return nil
	}
	features := make(map[string]float64)
	if err := json.Unmarshal(raw, &features); err == nil {
		return features
	}
	return nil
}
