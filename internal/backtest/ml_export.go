package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
)

// MLExport represents ML-oriented export data
type MLExport struct {
	StrategyMetadata  strategy.StrategyMetadata `json:"strategy_metadata"`
	BacktestSummary   BacktestSummary           `json:"backtest_summary"`
	Metrics           map[string]any            `json:"metrics"`
	BetHistory        []models.Bet              `json:"bet_history"`
	EquityCurve       EquityCurve               `json:"equity_curve"`
	FeatureImportance map[string]float64        `json:"feature_importance,omitempty"`
	ValidationResults WalkForwardResult         `json:"validation_results"`
	RiskProfile       RiskProfile               `json:"risk_profile"`
	MLFeatures        map[string]float64        `json:"ml_features"`
	Recommendation    string                    `json:"recommendation"`
	CompositeScore    float64                   `json:"composite_score"`
}

// BacktestSummary summarizes a backtest run
type BacktestSummary struct {
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	InitialCapital float64  `json:"initial_capital"`
	FinalCapital   float64  `json:"final_capital"`
	TotalBets     int      `json:"total_bets"`
}

// RiskProfile summarizes risk metrics
type RiskProfile struct {
	VaR95        float64 `json:"var_95"`
	VaR99        float64 `json:"var_99"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	TailRisk     float64 `json:"tail_risk"`
}

// ExportDBParams groups parameters for database export
type ExportDBParams struct {
	StrategyID     uuid.UUID
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital float64
	FinalCapital   float64
}

// ExportToJSON writes export data to JSON file
func ExportToJSON(export MLExport, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}
	return os.WriteFile(outputPath, data, 0o644)
}

// ExportToDatabase persists backtest result to database
func ExportToDatabase(ctx context.Context, result AggregatedResult, repo repository.BacktestResultRepository, params ExportDBParams) error {
	if repo == nil {
		return fmt.Errorf("backtest result repository is required")
	}
	model := models.BacktestResult{
		ID:             uuid.New(),
		StrategyID:     params.StrategyID,
		RunDate:        time.Now().UTC(),
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
		InitialCapital: params.InitialCapital,
		FinalCapital:   params.FinalCapital,
		TotalReturn:    result.HistoricalReplayMetrics.TotalReturn,
		SharpeRatio:    result.HistoricalReplayMetrics.SharpeRatio,
		MaxDrawdown:    result.HistoricalReplayMetrics.MaxDrawdown,
		TotalBets:      result.HistoricalReplayMetrics.TotalBets,
		WinRate:        result.HistoricalReplayMetrics.WinRate,
		ProfitFactor:   result.HistoricalReplayMetrics.ProfitFactor,
		Method:         "aggregated",
		CompositeScore: result.CompositeScore,
		Recommendation: result.Recommendation,
		MLFeatures:     mustMarshalJSON(result.MLFeatures),
		FullResults:    mustMarshalJSON(result),
		CreatedAt:      time.Now().UTC(),
	}
	return repo.SaveResult(ctx, &model)
}

// GenerateMLFeatures extracts features for ML training
func GenerateMLFeatures(result AggregatedResult) map[string]float64 {
	features := map[string]float64{}
	for key, value := range result.MLFeatures {
		features[key] = value
	}
	features["composite_score"] = result.CompositeScore
	return features
}

func mustMarshalJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
