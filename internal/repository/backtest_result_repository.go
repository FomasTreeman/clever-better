package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

const errScanBacktestResult = "failed to scan backtest result: %w"

// PostgresBacktestResultRepository implements BacktestResultRepository for PostgreSQL
type PostgresBacktestResultRepository struct {
	db *database.DB
}

// NewPostgresBacktestResultRepository creates a new backtest result repository
func NewPostgresBacktestResultRepository(db *database.DB) BacktestResultRepository {
	return &PostgresBacktestResultRepository{db: db}
}

// SaveResult inserts a backtest result
func (r *PostgresBacktestResultRepository) SaveResult(ctx context.Context, result *models.BacktestResult) error {
	query := `
		INSERT INTO backtest_results (
			id, strategy_id, run_date, start_date, end_date,
			initial_capital, final_capital, total_return, sharpe_ratio, max_drawdown,
			total_bets, win_rate, profit_factor, method, composite_score, recommendation,
			ml_features, full_results, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	`

	_, err := r.db.GetPool().Exec(ctx, query,
		result.ID, result.StrategyID, result.RunDate, result.StartDate, result.EndDate,
		result.InitialCapital, result.FinalCapital, result.TotalReturn, result.SharpeRatio, result.MaxDrawdown,
		result.TotalBets, result.WinRate, result.ProfitFactor, result.Method, result.CompositeScore, result.Recommendation,
		result.MLFeatures, result.FullResults, result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save backtest result: %w", err)
	}
	return nil
}

// GetByStrategyID retrieves backtest results by strategy ID
func (r *PostgresBacktestResultRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results WHERE strategy_id = $1 ORDER BY run_date DESC
	`
	rows, err := r.db.GetPool().Query(ctx, query, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backtest results: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// GetLatest retrieves latest backtest results
func (r *PostgresBacktestResultRepository) GetLatest(ctx context.Context, limit int) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results ORDER BY run_date DESC LIMIT $1
	`
	rows, err := r.db.GetPool().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest backtest results: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// GetByDateRange retrieves backtest results within a date range
func (r *PostgresBacktestResultRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results WHERE run_date >= $1 AND run_date <= $2 ORDER BY run_date DESC
	`
	rows, err := r.db.GetPool().Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query backtest results by date range: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// GetTopPerforming retrieves top N backtest results by composite score
func (r *PostgresBacktestResultRepository) GetTopPerforming(ctx context.Context, limit int) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results 
		ORDER BY composite_score DESC, run_date DESC 
		LIMIT $1
	`
	rows, err := r.db.GetPool().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top performing backtest results: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// GetRecentUnprocessed retrieves recent unprocessed backtest results
func (r *PostgresBacktestResultRepository) GetRecentUnprocessed(ctx context.Context, limit int) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results 
		WHERE ml_feedback_submitted = FALSE OR ml_feedback_submitted IS NULL
		ORDER BY run_date DESC 
		LIMIT $1
	`
	rows, err := r.db.GetPool().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed backtest results: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// MarkAsProcessed marks a backtest result as having submitted feedback
func (r *PostgresBacktestResultRepository) MarkAsProcessed(ctx context.Context, resultID uuid.UUID) error {
	query := `
		UPDATE backtest_results 
		SET ml_feedback_submitted = TRUE 
		WHERE id = $1
	`
	_, err := r.db.GetPool().Exec(ctx, query, resultID)
	if err != nil {
		return fmt.Errorf("failed to mark backtest result as processed: %w", err)
	}
	return nil
}

// GetByCompositeScoreRange retrieves backtest results within a score range
func (r *PostgresBacktestResultRepository) GetByCompositeScoreRange(ctx context.Context, minScore, maxScore float64, limit int) ([]*models.BacktestResult, error) {
	query := `
		SELECT id, strategy_id, run_date, start_date, end_date, initial_capital, final_capital,
			total_return, sharpe_ratio, max_drawdown, total_bets, win_rate, profit_factor,
			method, composite_score, recommendation, ml_features, full_results, created_at
		FROM backtest_results 
		WHERE composite_score >= $1 AND composite_score <= $2
		ORDER BY composite_score DESC, run_date DESC 
		LIMIT $3
	`
	rows, err := r.db.GetPool().Query(ctx, query, minScore, maxScore, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query backtest results by score range: %w", err)
	}
	defer rows.Close()

	var results []*models.BacktestResult
	for rows.Next() {
		result := &models.BacktestResult{}
		if err := rows.Scan(
			&result.ID, &result.StrategyID, &result.RunDate, &result.StartDate, &result.EndDate,
			&result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.SharpeRatio, &result.MaxDrawdown,
			&result.TotalBets, &result.WinRate, &result.ProfitFactor, &result.Method, &result.CompositeScore, &result.Recommendation,
			&result.MLFeatures, &result.FullResults, &result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(errScanBacktestResult, err)
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

