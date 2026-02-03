// Package service provides ML feedback loop functionality.
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

// MLFeedbackService manages feedback submission to ML service
type MLFeedbackService struct {
	mlClient     *ml.CachedMLClient
	httpClient   *ml.HTTPClient
	backtestRepo repository.BacktestResultRepository
	logger       *logrus.Logger
}

// NewMLFeedbackService creates a new ML feedback service
func NewMLFeedbackService(
	mlClient *ml.CachedMLClient,
	httpClient *ml.HTTPClient,
	backtestRepo repository.BacktestResultRepository,
	logger *logrus.Logger,
) *MLFeedbackService {
	return &MLFeedbackService{
		mlClient:     mlClient,
		httpClient:   httpClient,
		backtestRepo: backtestRepo,
		logger:       logger,
	}
}

// SubmitBacktestResult submits a single backtest result as feedback
func (s *MLFeedbackService) SubmitBacktestResult(ctx context.Context, result *models.BacktestResult) error {
	s.logger.WithFields(logrus.Fields{
		"strategy_id":     result.StrategyID,
		"composite_score": result.CompositeScore,
	}).Info("Submitting backtest result as feedback")

	if err := s.mlClient.SubmitBacktestFeedback(ctx, result); err != nil {
		s.logger.WithError(err).Error("Failed to submit feedback")
		return fmt.Errorf("failed to submit backtest feedback: %w", err)
	}

	// Mark as processed in database
	if err := s.backtestRepo.MarkAsProcessed(ctx, result.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to mark backtest result as processed")
		// Don't fail the whole operation for this
	}

	s.logger.WithField("result_id", result.ID).Debug("Successfully submitted feedback")
	return nil
}

// SubmitBatch submits multiple backtest results in batch
func (s *MLFeedbackService) SubmitBatch(ctx context.Context, batchSize int) (int, error) {
	s.logger.WithField("batch_size", batchSize).Info("Submitting batch feedback")

	// Get recent unprocessed backtest results
	results, err := s.backtestRepo.GetRecentUnprocessed(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get unprocessed results: %w", err)
	}

	if len(results) == 0 {
		s.logger.Info("No unprocessed backtest results to submit")
		return 0, nil
	}

	successCount := 0
	for _, result := range results {
		if err := s.SubmitBacktestResult(ctx, result); err != nil {
			s.logger.WithError(err).WithField("result_id", result.ID).Error("Failed to submit result in batch")
			continue
		}
		successCount++
	}

	s.logger.WithFields(logrus.Fields{
		"total":   len(results),
		"success": successCount,
		"failed":  len(results) - successCount,
	}).Info("Batch feedback submission complete")

	return successCount, nil
}

// TriggerRetraining initiates model retraining with specified config
func (s *MLFeedbackService) TriggerRetraining(ctx context.Context, config ml.TrainingConfig) (*ml.TrainingStatus, error) {
	s.logger.WithFields(logrus.Fields{
		"model_type": config.ModelType,
		"epochs":     config.Epochs,
	}).Info("Triggering model retraining")

	status, err := s.httpClient.TrainModels(ctx, config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to trigger retraining")
		return nil, fmt.Errorf("failed to trigger retraining: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"job_id": status.JobID,
		"status": status.Status,
	}).Info("Retraining job submitted")

	return status, nil
}

// SchedulePeriodicRetraining schedules periodic model retraining
func (s *MLFeedbackService) SchedulePeriodicRetraining(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.WithField("interval", interval).Info("Starting periodic retraining scheduler")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping periodic retraining scheduler")
			return
		case <-ticker.C:
			s.logger.Info("Periodic retraining triggered")

			// Submit batch feedback first
			count, err := s.SubmitBatch(ctx, 100)
			if err != nil {
				s.logger.WithError(err).Error("Failed to submit batch feedback during periodic retraining")
				continue
			}

			s.logger.WithField("feedback_count", count).Info("Submitted feedback for retraining")

			// Only retrain if we have sufficient feedback
			if count >= 20 {
				configs := []ml.TrainingConfig{
					{
						ModelType:            "classifier",
						Epochs:               50,
						BatchSize:            32,
						LearningRate:         0.001,
						HyperparameterSearch: false,
					},
					{
						ModelType:            "ensemble",
						Epochs:               30,
						BatchSize:            64,
						LearningRate:         0.01,
						HyperparameterSearch: false,
					},
					{
						ModelType:            "rl_agent",
						Epochs:               100,
						BatchSize:            128,
						LearningRate:         0.0001,
						HyperparameterSearch: false,
					},
				}

				for _, config := range configs {
					status, err := s.TriggerRetraining(ctx, config)
					if err != nil {
						s.logger.WithError(err).WithField("model_type", config.ModelType).Error("Failed to trigger retraining")
						continue
					}
					s.logger.WithFields(logrus.Fields{
						"model_type": config.ModelType,
						"job_id":     status.JobID,
					}).Info("Retraining job submitted")
				}
			} else {
				s.logger.WithField("feedback_count", count).Info("Insufficient feedback for retraining, skipping")
			}
		}
	}
}

// GetRetrainingStatus checks status of a retraining job
func (s *MLFeedbackService) GetRetrainingStatus(ctx context.Context, jobID string) (*ml.TrainingStatus, error) {
	status, err := s.httpClient.GetTrainingStatus(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get retraining status: %w", err)
	}
	return status, nil
}
