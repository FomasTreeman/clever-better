// Package service provides ML orchestration functionality.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// MLOrchestratorService orchestrates ML-driven workflows
type MLOrchestratorService struct {
	strategyGenerator *StrategyGeneratorService
	mlFeedback        *MLFeedbackService
	strategyEvaluator *StrategyEvaluatorService
	mlClient          *ml.CachedMLClient
	predictionRepo    repository.PredictionRepository
	logger            *logrus.Logger
}

// NewMLOrchestratorService creates a new ML orchestrator
func NewMLOrchestratorService(
	strategyGenerator *StrategyGeneratorService,
	mlFeedback *MLFeedbackService,
	strategyEvaluator *StrategyEvaluatorService,
	mlClient *ml.CachedMLClient,
	predictionRepo repository.PredictionRepository,
	logger *logrus.Logger,
) *MLOrchestratorService {
	return &MLOrchestratorService{
		strategyGenerator: strategyGenerator,
		mlFeedback:        mlFeedback,
		strategyEvaluator: strategyEvaluator,
		mlClient:          mlClient,
		predictionRepo:    predictionRepo,
		logger:            logger,
	}
}

// PipelineReport represents the result of a discovery pipeline run
type PipelineReport struct {
	RunID              uuid.UUID
	GeneratedCount     int
	ActivatedCount     int
	DeactivatedCount   int
	FeedbackSubmitted  int
	RetrainingTriggered bool
	TopStrategies      []*StrategyEvaluation
	Duration           time.Duration
	CompletedAt        time.Time
}

// DiscoveryConfig configures the strategy discovery pipeline
type DiscoveryConfig struct {
	GenerateCount       int
	RiskLevel           string
	TargetReturn        float64
	MinCompositeScore   float64
	DeactivateThreshold float64
	SubmitFeedback      bool
	TriggerRetraining   bool
}

// RunStrategyDiscoveryPipeline executes full ML-driven strategy discovery
func (o *MLOrchestratorService) RunStrategyDiscoveryPipeline(ctx context.Context, config DiscoveryConfig) (*PipelineReport, error) {
	start := time.Now()
	runID := uuid.New()

	o.logger.WithFields(logrus.Fields{
		"run_id":         runID,
		"generate_count": config.GenerateCount,
		"risk_level":     config.RiskLevel,
	}).Info("Starting strategy discovery pipeline")

	report := &PipelineReport{
		RunID: runID,
	}

	// Step 1: Submit backtest feedback
	var feedbackCount int
	var err error
	if config.SubmitFeedback {
		feedbackCount, err = o.mlFeedback.SubmitBatch(ctx, 100)
		if err != nil {
			o.logger.WithError(err).Warn("Failed to submit feedback, continuing pipeline")
		}
		report.FeedbackSubmitted = feedbackCount
		o.logger.WithField("feedback_count", feedbackCount).Info("Submitted backtest feedback")
	}

	// Step 2: Trigger retraining if sufficient feedback
	if config.TriggerRetraining && feedbackCount >= 20 {
		trainingConfig := ml.TrainingConfig{
			ModelType:            "ensemble",
			Epochs:               50,
			BatchSize:            32,
			LearningRate:         0.001,
			HyperparameterSearch: true,
		}

		status, err := o.mlFeedback.TriggerRetraining(ctx, trainingConfig)
		if err != nil {
			o.logger.WithError(err).Warn("Failed to trigger retraining, continuing pipeline")
		} else {
			report.RetrainingTriggered = true
			o.logger.WithField("job_id", status.JobID).Info("Triggered model retraining")
		}
	}

	// Step 3: Generate new strategies
	constraints := ml.StrategyConstraints{
		RiskLevel:         config.RiskLevel,
		TargetReturn:      config.TargetReturn,
		MaxDrawdownLimit:  0.15,
		MinWinRate:        0.55,
		MaxCandidates:     config.GenerateCount,
	}

	generatedStrategies, err := o.strategyGenerator.GenerateFromBacktestResults(ctx, 50, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to generate strategies: %w", err)
	}
	report.GeneratedCount = len(generatedStrategies)
	o.logger.WithField("generated_count", len(generatedStrategies)).Info("Generated new strategies")

	// Step 4: Evaluate and activate top performers
	activatedIDs, err := o.strategyGenerator.ActivateTopStrategies(ctx, generatedStrategies)
	if err != nil {
		o.logger.WithError(err).Warn("Failed to activate strategies")
	}
	report.ActivatedCount = len(activatedIDs)
	o.logger.WithField("activated_count", len(activatedIDs)).Info("Activated top strategies")

	// Step 5: Deactivate underperformers
	deactivatedIDs, err := o.strategyEvaluator.DeactivateUnderperformers(ctx, config.DeactivateThreshold)
	if err != nil {
		o.logger.WithError(err).Warn("Failed to deactivate underperformers")
	}
	report.DeactivatedCount = len(deactivatedIDs)
	o.logger.WithField("deactivated_count", len(deactivatedIDs)).Info("Deactivated underperformers")

	// Step 6: Get final rankings
	topStrategies, err := o.strategyEvaluator.GetTopPerformers(ctx, 10)
	if err != nil {
		o.logger.WithError(err).Warn("Failed to get top performers")
	}
	report.TopStrategies = topStrategies

	report.Duration = time.Since(start)
	report.CompletedAt = time.Now()

	o.logger.WithFields(logrus.Fields{
		"run_id":             runID,
		"generated":          report.GeneratedCount,
		"activated":          report.ActivatedCount,
		"deactivated":        report.DeactivatedCount,
		"duration":           report.Duration,
		"retraining_triggered": report.RetrainingTriggered,
	}).Info("Strategy discovery pipeline complete")

	return report, nil
}

// GetLivePredictions retrieves predictions for active races
func (o *MLOrchestratorService) GetLivePredictions(ctx context.Context, raceID uuid.UUID, runnerIDs []uuid.UUID, strategyID uuid.UUID) ([]*ml.PredictionResult, error) {
	o.logger.WithFields(logrus.Fields{
		"race_id":     raceID,
		"runner_count": len(runnerIDs),
		"strategy_id": strategyID,
	}).Info("Getting live predictions")

	requests := make([]ml.PredictionRequest, len(runnerIDs))
	for i, runnerID := range runnerIDs {
		requests[i] = ml.PredictionRequest{
			RaceID:       raceID,
			RunnerID:     runnerID,
			StrategyID:   strategyID,
			ModelVersion: "latest",
			Features:     []float64{}, // TODO: Extract features from race/runner data
		}
	}

	predictions, err := o.mlClient.BatchPredict(ctx, requests)
	if err != nil {
		return nil, fmt.Errorf("failed to get predictions: %w", err)
	}

	// Store predictions in database - batch operation for efficiency
	predictionsToStore := make([]*models.Prediction, len(predictions))
	for i, pred := range predictions {
		predictionsToStore[i] = &models.Prediction{
			ID:             uuid.New(),
			ModelID:        uuid.New(), // Note: ModelID should be determined from ML service response or strategy config
			RaceID:         pred.RaceID,
			RunnerID:       pred.RunnerID,
			Probability:    pred.Probability,
			Confidence:     pred.Confidence,
			Features:       nil, // TODO: Store features as JSON
			PredictedAt:    time.Now(),
		}
	}

	if err := o.predictionRepo.InsertBatch(ctx, predictionsToStore); err != nil {
		o.logger.WithError(err).Warn("Failed to store predictions")
	}

	o.logger.WithField("prediction_count", len(predictions)).Info("Live predictions retrieved")
	return predictions, nil
}

// UpdateStrategyRankings refreshes strategy rankings
func (o *MLOrchestratorService) UpdateStrategyRankings(ctx context.Context) ([]*StrategyEvaluation, error) {
	o.logger.Info("Updating strategy rankings")

	rankings, err := o.strategyEvaluator.RankActiveStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to rank strategies: %w", err)
	}

	o.logger.WithField("strategy_count", len(rankings)).Info("Strategy rankings updated")
	return rankings, nil
}

// MonitorMLService monitors ML service health
func (o *MLOrchestratorService) MonitorMLService(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	o.logger.WithField("interval", interval).Info("Starting ML service health monitor")

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Stopping ML service health monitor")
			return
		case <-ticker.C:
			// Get cache stats
			hits, misses, hitRatio := o.mlClient.GetCacheStats()
			o.logger.WithFields(logrus.Fields{
				"cache_hits":     hits,
				"cache_misses":   misses,
				"cache_hit_ratio": hitRatio,
			}).Debug("ML client cache stats")
		}
	}
}
