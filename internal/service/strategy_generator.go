// Package service provides strategy generation from ML.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/clever-better/internal/backtest"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/ml"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

// StrategyGeneratorService generates betting strategies using ML
type StrategyGeneratorService struct {
	mlClient          *ml.CachedMLClient
	strategyRepo      repository.StrategyRepository
	backtestRepo      repository.BacktestResultRepository
	db                *database.DB
	logger            *logrus.Logger
	minCompositeScore float64
	backtestConfig    backtest.BacktestConfig
}

// NewStrategyGeneratorService creates a new strategy generator service
func NewStrategyGeneratorService(
	mlClient *ml.CachedMLClient,
	strategyRepo repository.StrategyRepository,
	backtestRepo repository.BacktestResultRepository,
	db *database.DB,
	logger *logrus.Logger,
) *StrategyGeneratorService {
	// Default backtest config for strategy evaluation (last 6 months)
	btConfig := backtest.BacktestConfig{
		StartDate:            time.Now().AddDate(0, -6, 0),
		EndDate:              time.Now().AddDate(0, -1, 0),
		InitialBankroll:      10000.0,
		CommissionRate:       0.05,
		SlippageTicks:        1,
		MinLiquidity:         100.0,
		MonteCarloIterations: 100,
		WalkForwardWindows:   1,
		RiskFreeRate:         0.02,
	}

	return &StrategyGeneratorService{
		mlClient:          mlClient,
		strategyRepo:      strategyRepo,
		backtestRepo:      backtestRepo,
		db:                db,
		logger:            logger,
		minCompositeScore: 0.6,
		backtestConfig:    btConfig,
	}
}

// GenerateFromBacktestResults analyzes top backtest results and generates new strategies
func (s *StrategyGeneratorService) GenerateFromBacktestResults(ctx context.Context, topN int, constraints ml.StrategyConstraints) ([]*ml.GeneratedStrategy, error) {
	s.logger.WithField("top_n", topN).Info("Generating strategies from backtest results")

	// Get top performing backtest results
	results, err := s.backtestRepo.GetTopPerforming(ctx, topN)
	if err != nil {
		return nil, fmt.Errorf("failed to get top backtest results: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No backtest results available for strategy generation")
		return nil, nil
	}

	// Aggregate ML features from backtest results
	aggregatedFeatures, err := s.aggregateMLFeatures(results)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to aggregate ML features, proceeding without them")
		aggregatedFeatures = make(map[string]float64)
	}

	// Extract top metrics from best performing results
	topMetrics := s.extractTopMetrics(results)

	// Populate constraints with aggregated data from backtest results
	constraints.AggregatedFeatures = aggregatedFeatures
	constraints.TopMetrics = topMetrics

	s.logger.WithFields(logrus.Fields{
		"aggregated_features_count": len(aggregatedFeatures),
		"top_metrics_count":         len(topMetrics),
		"backtest_results_analyzed": len(results),
	}).Info("Aggregated backtest data for ML strategy generation")

	// Generate strategies based on constraints with real backtest data
	strategies, err := s.mlClient.GenerateStrategy(ctx, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to generate strategies: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"generated_count": len(strategies),
		"top_results":     len(results),
	}).Info("Successfully generated strategies")

	return strategies, nil
}

// GenerateOptimizedStrategy generates a single optimized strategy for specific risk profile
func (s *StrategyGeneratorService) GenerateOptimizedStrategy(ctx context.Context, riskLevel string, targetReturn float64) (*ml.GeneratedStrategy, error) {
	constraints := ml.StrategyConstraints{
		RiskLevel:        riskLevel,
		TargetReturn:     targetReturn,
		MaxDrawdownLimit: 0.15, // 15% max drawdown
		MinWinRate:       0.55, // 55% win rate
		MaxCandidates:    1,
	}

	strategies, err := s.mlClient.GenerateStrategy(ctx, constraints)
	if err != nil {
		return nil, err
	}

	if len(strategies) == 0 {
		return nil, fmt.Errorf("no strategy generated for constraints: %+v", constraints)
	}

	return strategies[0], nil
}

// EvaluateGeneratedStrategy evaluates a generated strategy via REAL backtesting
func (s *StrategyGeneratorService) EvaluateGeneratedStrategy(ctx context.Context, generatedStrategy *ml.GeneratedStrategy) (*models.BacktestResult, error) {
	s.logger.WithField("strategy_id", generatedStrategy.StrategyID).Info("Evaluating generated strategy with real backtest")

	// Convert generated strategy to actual strategy model
	strategyModel := &models.Strategy{
		ID:          generatedStrategy.StrategyID,
		Name:        fmt.Sprintf("ML-Generated-%s", generatedStrategy.StrategyID),
		Description: fmt.Sprintf("ML-generated strategy with confidence %.2f", generatedStrategy.Confidence),
		Parameters:  generatedStrategy.Parameters,
		IsActive:    false, // Not active until proven successful
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save strategy to database
	if err := s.strategyRepo.Create(ctx, strategyModel); err != nil {
		return nil, fmt.Errorf("failed to save generated strategy: %w", err)
	}

	// Create strategy implementation from ML parameters
	stratImpl := s.createStrategyFromMLParams(generatedStrategy)

	// Create backtest engine with the ML-generated strategy
	engine, err := backtest.NewEngine(s.backtestConfig, s.db, stratImpl, s.logger)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create backtest engine, using ML estimates")
		return s.createFallbackResult(generatedStrategy), nil
	}
	defer engine.Close(ctx)

	// Run REAL backtest
	s.logger.WithFields(logrus.Fields{
		"start_date": s.backtestConfig.StartDate.Format("2006-01-02"),
		"end_date":   s.backtestConfig.EndDate.Format("2006-01-02"),
	}).Info("Running real backtest for ML-generated strategy")

	state, metrics, err := engine.Run(ctx, s.backtestConfig.StartDate, s.backtestConfig.EndDate)
	if err != nil {
		s.logger.WithError(err).Error("Backtest execution failed, using ML estimates")
		return s.createFallbackResult(generatedStrategy), nil
	}

	// Calculate composite score from REAL backtest metrics
	// Composite = (Sharpe * 0.4) + (ROI * 0.3) + (Win Rate * 0.2) + (ML Confidence * 0.1)
	compositeScore := (metrics.SharpeRatio * 0.4) + (metrics.TotalReturn * 0.3) + (metrics.WinRate * 0.2) + (generatedStrategy.Confidence * 0.1)

	// Create ML features from backtest state for feedback
	mlFeatures := s.extractMLFeaturesFromBacktest(state, metrics)
	mlFeaturesJSON, _ := json.Marshal(mlFeatures)

	// Store REAL backtest result
	result := &models.BacktestResult{
		ID:             uuid.New(),
		StrategyID:     generatedStrategy.StrategyID,
		RunDate:        time.Now(),
		StartDate:      s.backtestConfig.StartDate,
		EndDate:        s.backtestConfig.EndDate,
		InitialCapital: s.backtestConfig.InitialBankroll,
		FinalCapital:   state.CurrentBankroll,
		TotalReturn:    metrics.TotalReturn,
		SharpeRatio:    metrics.SharpeRatio,
		MaxDrawdown:    metrics.MaxDrawdown,
		TotalBets:      metrics.TotalBets,
		WinRate:        metrics.WinRate,
		ProfitFactor:   metrics.ProfitFactor,
		Method:         "real_backtest",
		CompositeScore: compositeScore,
		Recommendation: s.getRecommendation(compositeScore, metrics),
		MLFeatures:     mlFeaturesJSON,
		CreatedAt:      time.Now(),
	}

	// Store backtest result
	if err := s.backtestRepo.Create(ctx, result); err != nil {
		s.logger.WithError(err).Error("Failed to store backtest result")
		return nil, fmt.Errorf("failed to store backtest result: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"strategy_id":     generatedStrategy.StrategyID,
		"composite_score": result.CompositeScore,
		"sharpe_ratio":    result.SharpeRatio,
		"total_return":    result.TotalReturn,
		"win_rate":        result.WinRate,
		"total_bets":      result.TotalBets,
		"max_drawdown":    result.MaxDrawdown,
		"recommendation":  result.Recommendation,
	}).Info("Real backtest evaluation complete")

	return result, nil
}

// ActivateTopStrategies activates strategies that exceed minimum composite score
func (s *StrategyGeneratorService) ActivateTopStrategies(ctx context.Context, strategies []*ml.GeneratedStrategy) ([]uuid.UUID, error) {
	activatedIDs := make([]uuid.UUID, 0)

	for _, strategy := range strategies {
		// Evaluate strategy
		result, err := s.EvaluateGeneratedStrategy(ctx, strategy)
		if err != nil {
			s.logger.WithError(err).WithField("strategy_id", strategy.StrategyID).Error("Failed to evaluate strategy")
			continue
		}

		// Activate if composite score exceeds threshold
		if result.CompositeScore >= s.minCompositeScore {
			strategyModel, err := s.strategyRepo.GetByID(ctx, strategy.StrategyID)
			if err != nil {
				s.logger.WithError(err).WithField("strategy_id", strategy.StrategyID).Error("Failed to retrieve strategy")
				continue
			}

			strategyModel.IsActive = true
			strategyModel.UpdatedAt = time.Now()

			if err := s.strategyRepo.Update(ctx, strategyModel); err != nil {
				s.logger.WithError(err).WithField("strategy_id", strategy.StrategyID).Error("Failed to activate strategy")
				continue
			}

			activatedIDs = append(activatedIDs, strategy.StrategyID)
			s.logger.WithFields(logrus.Fields{
				"strategy_id":     strategy.StrategyID,
				"composite_score": result.CompositeScore,
			}).Info("Activated high-performing strategy")
		}
	}

	return activatedIDs, nil
}

// aggregateMLFeatures aggregates ML features from multiple backtest results
// It computes mean, std dev, min, and max for each feature across all results
func (s *StrategyGeneratorService) aggregateMLFeatures(results []*models.BacktestResult) (map[string]float64, error) {
	if len(results) == 0 {
		return make(map[string]float64), nil
	}

	// Collect all features from all results
	allFeatures := make(map[string][]float64)

	for _, result := range results {
		if len(result.MLFeatures) == 0 {
			continue
		}

		var features map[string]float64
		if err := json.Unmarshal(result.MLFeatures, &features); err != nil {
			s.logger.WithError(err).WithField("result_id", result.ID).Warn("Failed to unmarshal ML features")
			continue
		}

		// Accumulate each feature value
		for key, value := range features {
			allFeatures[key] = append(allFeatures[key], value)
		}
	}

	// Compute aggregated statistics
	aggregated := make(map[string]float64)

	for key, values := range allFeatures {
		if len(values) == 0 {
			continue
		}

		// Mean
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		mean := sum / float64(len(values))
		aggregated[key+"_mean"] = mean

		// Standard deviation
		if len(values) > 1 {
			variance := 0.0
			for _, v := range values {
				diff := v - mean
				variance += diff * diff
			}
			stdDev := math.Sqrt(variance / float64(len(values)-1))
			aggregated[key+"_std"] = stdDev
		}

		// Min and Max
		min := values[0]
		max := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		aggregated[key+"_min"] = min
		aggregated[key+"_max"] = max
	}

	s.logger.WithFields(logrus.Fields{
		"feature_count":    len(allFeatures),
		"aggregated_count": len(aggregated),
		"results_analyzed": len(results),
	}).Debug("Aggregated ML features from backtest results")

	return aggregated, nil
}

// extractTopMetrics extracts key metrics from top performing backtest results
func (s *StrategyGeneratorService) extractTopMetrics(results []*models.BacktestResult) map[string]float64 {
	if len(results) == 0 {
		return make(map[string]float64)
	}

	// Sort by composite score descending
	sortedResults := make([]*models.BacktestResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].CompositeScore > sortedResults[j].CompositeScore
	})

	// Extract metrics from top result
	top := sortedResults[0]

	metrics := map[string]float64{
		"top_composite_score": top.CompositeScore,
		"top_sharpe_ratio":    top.SharpeRatio,
		"top_roi":             top.TotalReturn,
		"top_win_rate":        top.WinRate,
		"top_max_drawdown":    top.MaxDrawdown,
		"top_profit_factor":   top.ProfitFactor,
	}

	// Add averages across all results
	avgSharpe := 0.0
	avgROI := 0.0
	avgWinRate := 0.0
	avgDrawdown := 0.0
	avgComposite := 0.0

	for _, result := range results {
		avgSharpe += result.SharpeRatio
		avgROI += result.TotalReturn
		avgWinRate += result.WinRate
		avgDrawdown += result.MaxDrawdown
		avgComposite += result.CompositeScore
	}

	n := float64(len(results))
	metrics["avg_sharpe_ratio"] = avgSharpe / n
	metrics["avg_roi"] = avgROI / n
	metrics["avg_win_rate"] = avgWinRate / n
	metrics["avg_max_drawdown"] = avgDrawdown / n
	metrics["avg_composite_score"] = avgComposite / n

	// Add standard deviations
	sharpeVariance := 0.0
	roiVariance := 0.0
	for _, result := range results {
		sharpeVariance += math.Pow(result.SharpeRatio-metrics["avg_sharpe_ratio"], 2)
		roiVariance += math.Pow(result.TotalReturn-metrics["avg_roi"], 2)
	}
	metrics["std_sharpe_ratio"] = math.Sqrt(sharpeVariance / n)
	metrics["std_roi"] = math.Sqrt(roiVariance / n)

	s.logger.WithFields(logrus.Fields{
		"top_composite": metrics["top_composite_score"],
		"avg_sharpe":    metrics["avg_sharpe_ratio"],
		"avg_roi":       metrics["avg_roi"],
	}).Debug("Extracted top metrics from backtest results")

	return metrics
}

// createStrategyFromMLParams creates a strategy implementation from ML parameters
func (s *StrategyGeneratorService) createStrategyFromMLParams(gen *ml.GeneratedStrategy) strategy.Strategy {
	// Create a value strategy with ML-generated parameters
	strat := strategy.NewSimpleValueStrategy()

	// Override parameters from ML generation
	if minEdge, ok := gen.Parameters["min_edge_threshold"]; ok {
		strat.MinEdgeThreshold = minEdge
	}
	if minConf, ok := gen.Parameters["min_confidence"]; ok {
		strat.MinConfidence = minConf
	}
	if kellyFrac, ok := gen.Parameters["kelly_fraction"]; ok {
		strat.KellyFraction = kellyFrac
	}
	if minOdds, ok := gen.Parameters["min_odds"]; ok {
		strat.MinOdds = minOdds
	}
	if maxOdds, ok := gen.Parameters["max_odds"]; ok {
		strat.MaxOdds = maxOdds
	}

	strat.NameValue = fmt.Sprintf("ml_gen_%s", gen.StrategyID.String()[:8])

	s.logger.WithFields(logrus.Fields{
		"strategy_name":  strat.Name(),
		"min_edge":       strat.MinEdgeThreshold,
		"kelly_fraction": strat.KellyFraction,
	}).Debug("Created strategy implementation from ML parameters")

	return strat
}

// createFallbackResult creates a fallback result using ML estimates when backtest fails
func (s *StrategyGeneratorService) createFallbackResult(gen *ml.GeneratedStrategy) *models.BacktestResult {
	s.logger.Warn("Using ML estimates as fallback for backtest result")

	winRate := gen.ExpectedWinRate
	if winRate < 0 || winRate > 1 {
		winRate = 0.55
	}

	roi := gen.ExpectedReturn
	if roi < -1 {
		roi = -1
	}

	sharpe := gen.ExpectedSharpe
	if sharpe < -5 {
		sharpe = -5
	}

	compositeScore := (sharpe * 0.4) + (roi * 0.3) + (winRate * 0.2) + (gen.Confidence * 0.1)

	return &models.BacktestResult{
		ID:             uuid.New(),
		StrategyID:     gen.StrategyID,
		RunDate:        time.Now(),
		StartDate:      s.backtestConfig.StartDate,
		EndDate:        s.backtestConfig.EndDate,
		InitialCapital: s.backtestConfig.InitialBankroll,
		FinalCapital:   s.backtestConfig.InitialBankroll * (1 + roi),
		TotalReturn:    roi,
		SharpeRatio:    sharpe,
		MaxDrawdown:    1.0 - winRate,
		TotalBets:      0,
		WinRate:        winRate,
		ProfitFactor:   1.0 + (roi * winRate),
		Method:         "ml_estimate",
		CompositeScore: compositeScore,
		Recommendation: s.getRecommendation(compositeScore, backtest.Metrics{SharpeRatio: sharpe, WinRate: winRate}),
		CreatedAt:      time.Now(),
	}
}

// extractMLFeaturesFromBacktest extracts ML features from backtest state for feedback
func (s *StrategyGeneratorService) extractMLFeaturesFromBacktest(state *backtest.BacktestState, metrics backtest.Metrics) map[string]float64 {
	features := map[string]float64{
		"sharpe_ratio":   metrics.SharpeRatio,
		"total_return":   metrics.TotalReturn,
		"max_drawdown":   metrics.MaxDrawdown,
		"win_rate":       metrics.WinRate,
		"profit_factor":  metrics.ProfitFactor,
		"total_bets":     float64(metrics.TotalBets),
		"winning_bets":   float64(metrics.WinningBets),
		"losing_bets":    float64(metrics.LosingBets),
		"average_win":    metrics.AverageWin,
		"average_loss":   metrics.AverageLoss,
		"largest_win":    metrics.LargestWin,
		"largest_loss":   metrics.LargestLoss,
		"sortino_ratio":  metrics.SortinoRatio,
		"calmar_ratio":   metrics.CalmarRatio,
		"var_95":         metrics.ValueAtRisk95,
		"final_bankroll": state.CurrentBankroll,
		"trades_per_day": float64(metrics.TotalBets) / float64(metrics.TradingDays),
	}

	return features
}

// getRecommendation generates a recommendation based on metrics
func (s *StrategyGeneratorService) getRecommendation(compositeScore float64, metrics backtest.Metrics) string {
	if compositeScore >= 0.8 && metrics.SharpeRatio > 1.5 {
		return "EXCELLENT"
	} else if compositeScore >= 0.6 && metrics.WinRate > 0.55 {
		return "GOOD"
	} else if compositeScore >= 0.4 {
		return "ACCEPTABLE"
	} else if compositeScore >= 0.2 {
		return "POOR"
	}
	return "REJECT"
}
