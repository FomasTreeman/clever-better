// Package service provides strategy evaluation functionality.
package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// StrategyEvaluatorService evaluates and ranks betting strategies
type StrategyEvaluatorService struct {
	mlClient     *ml.CachedMLClient
	strategyRepo repository.StrategyRepository
	backtestRepo repository.BacktestResultRepository
	logger       *logrus.Logger
}

// NewStrategyEvaluatorService creates a new strategy evaluator service
func NewStrategyEvaluatorService(
	mlClient *ml.CachedMLClient,
	strategyRepo repository.StrategyRepository,
	backtestRepo repository.BacktestResultRepository,
	logger *logrus.Logger,
) *StrategyEvaluatorService {
	return &StrategyEvaluatorService{
		mlClient:     mlClient,
		strategyRepo: strategyRepo,
		backtestRepo: backtestRepo,
		logger:       logger,
	}
}

// StrategyEvaluation represents evaluation result for a strategy
type StrategyEvaluation struct {
	StrategyID      uuid.UUID
	StrategyName    string
	CompositeScore  float64
	MLConfidence    float64
	BacktestMetrics *models.BacktestResult
	Rank            int
	Recommendation  string
	EvaluatedAt     time.Time
}

// EvaluateStrategy performs comprehensive evaluation of a strategy
func (s *StrategyEvaluatorService) EvaluateStrategy(ctx context.Context, strategyID uuid.UUID) (*StrategyEvaluation, error) {
	s.logger.WithField("strategy_id", strategyID).Info("Evaluating strategy")

	// Get strategy from database
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	// Get ML evaluation
	mlScore, recommendation, err := s.mlClient.EvaluateStrategy(ctx, strategyID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get ML evaluation, using backtest only")
		mlScore = 0.0
		recommendation = "UNKNOWN"
	}

	// Get latest backtest results
	backtestResults, err := s.backtestRepo.GetByStrategyID(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backtest results: %w", err)
	}

	var latestBacktest *models.BacktestResult
	if len(backtestResults) > 0 {
		latestBacktest = backtestResults[0]
	} else {
		// No backtest results exist - log recommendation to run backtest
		s.logger.WithField("strategy_id", strategyID).Warn("No backtest results found - strategy needs real backtest evaluation")
	}

	// Calculate composite score
	compositeScore := s.calculateCompositeScore(mlScore, latestBacktest)

	evaluation := &StrategyEvaluation{
		StrategyID:      strategyID,
		StrategyName:    strategy.Name,
		CompositeScore:  compositeScore,
		MLConfidence:    mlScore,
		BacktestMetrics: latestBacktest,
		Recommendation:  recommendation,
		EvaluatedAt:     time.Now(),
	}

	s.logger.WithFields(logrus.Fields{
		"strategy_id":     strategyID,
		"composite_score": compositeScore,
		"ml_confidence":   mlScore,
		"recommendation":  recommendation,
	}).Info("Strategy evaluation complete")

	return evaluation, nil
}

// CompareStrategies compares multiple strategies and returns ranked list
func (s *StrategyEvaluatorService) CompareStrategies(ctx context.Context, strategyIDs []uuid.UUID) ([]*StrategyEvaluation, error) {
	s.logger.WithField("count", len(strategyIDs)).Info("Comparing strategies")

	evaluations := make([]*StrategyEvaluation, 0, len(strategyIDs))

	for _, strategyID := range strategyIDs {
		eval, err := s.EvaluateStrategy(ctx, strategyID)
		if err != nil {
			s.logger.WithError(err).WithField("strategy_id", strategyID).Error("Failed to evaluate strategy")
			continue
		}
		evaluations = append(evaluations, eval)
	}

	// Sort by composite score (descending)
	sort.Slice(evaluations, func(i, j int) bool {
		return evaluations[i].CompositeScore > evaluations[j].CompositeScore
	})

	// Assign ranks
	for i, eval := range evaluations {
		eval.Rank = i + 1
	}

	s.logger.WithField("evaluated_count", len(evaluations)).Info("Strategy comparison complete")
	return evaluations, nil
}

// RankActiveStrategies ranks all active strategies by performance
func (s *StrategyEvaluatorService) RankActiveStrategies(ctx context.Context) ([]*StrategyEvaluation, error) {
	s.logger.Info("Ranking active strategies")

	// Get all active strategies
	strategies, err := s.strategyRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategies: %w", err)
	}

	activeIDs := make([]uuid.UUID, 0)
	for _, strategy := range strategies {
		if strategy.IsActive {
			activeIDs = append(activeIDs, strategy.ID)
		}
	}

	if len(activeIDs) == 0 {
		s.logger.Info("No active strategies to rank")
		return []*StrategyEvaluation{}, nil
	}

	return s.CompareStrategies(ctx, activeIDs)
}

// calculateCompositeScore combines ML confidence and backtest metrics
func (s *StrategyEvaluatorService) calculateCompositeScore(mlScore float64, backtest *models.BacktestResult) float64 {
	// If no backtest, use ML score only
	if backtest == nil {
		return mlScore * 0.5 // Reduce confidence without backtest validation
	}

	// Weight ML score and backtest composite score equally
	return (mlScore + backtest.CompositeScore) / 2.0
}

// GetTopPerformers returns top N strategies by composite score
func (s *StrategyEvaluatorService) GetTopPerformers(ctx context.Context, topN int) ([]*StrategyEvaluation, error) {
	rankings, err := s.RankActiveStrategies(ctx)
	if err != nil {
		return nil, err
	}

	if len(rankings) <= topN {
		return rankings, nil
	}

	return rankings[:topN], nil
}

// DeactivateUnderperformers deactivates strategies below threshold
func (s *StrategyEvaluatorService) DeactivateUnderperformers(ctx context.Context, minCompositeScore float64) ([]uuid.UUID, error) {
	s.logger.WithField("threshold", minCompositeScore).Info("Deactivating underperformers")

	rankings, err := s.RankActiveStrategies(ctx)
	if err != nil {
		return nil, err
	}

	deactivatedIDs := make([]uuid.UUID, 0)

	for _, eval := range rankings {
		if eval.CompositeScore < minCompositeScore {
			strategy, err := s.strategyRepo.GetByID(ctx, eval.StrategyID)
			if err != nil {
				s.logger.WithError(err).WithField("strategy_id", eval.StrategyID).Error("Failed to get strategy")
				continue
			}

			strategy.IsActive = false
			strategy.UpdatedAt = time.Now()

			if err := s.strategyRepo.Update(ctx, strategy); err != nil {
				s.logger.WithError(err).WithField("strategy_id", eval.StrategyID).Error("Failed to deactivate strategy")
				continue
			}

			deactivatedIDs = append(deactivatedIDs, eval.StrategyID)
			s.logger.WithFields(logrus.Fields{
				"strategy_id":     eval.StrategyID,
				"composite_score": eval.CompositeScore,
			}).Info("Deactivated underperforming strategy")
		}
	}

	return deactivatedIDs, nil
}
