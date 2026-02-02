package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

// PostgresStrategyPerformanceRepository implements StrategyPerformanceRepository for PostgreSQL
type PostgresStrategyPerformanceRepository struct {
	db *database.DB
}

// NewPostgresStrategyPerformanceRepository creates a new strategy performance repository
func NewPostgresStrategyPerformanceRepository(db *database.DB) StrategyPerformanceRepository {
	return &PostgresStrategyPerformanceRepository{db: db}
}

// Insert inserts a new strategy performance record
func (sp *PostgresStrategyPerformanceRepository) Insert(ctx context.Context, perf *models.StrategyPerformance) error {
	query := `
		INSERT INTO strategy_performance (time, strategy_id, total_bets, winning_bets, losing_bets, 
		                                    gross_profit, gross_loss, net_profit, roi, sharpe_ratio, max_drawdown)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := sp.db.GetPool().Exec(ctx, query,
		perf.Time, perf.StrategyID, perf.TotalBets, perf.WinningBets, perf.LosingBets,
		perf.GrossProfit, perf.GrossLoss, perf.NetProfit, perf.ROI, perf.SharpeRatio, perf.MaxDrawdown,
	)
	if err != nil {
		return fmt.Errorf("failed to insert strategy performance: %w", err)
	}

	return nil
}

// GetByStrategyID retrieves performance data for a specific strategy
func (sp *PostgresStrategyPerformanceRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.StrategyPerformance, error) {
	query := `
		SELECT time, strategy_id, total_bets, winning_bets, losing_bets,
		       gross_profit, gross_loss, net_profit, roi, sharpe_ratio, max_drawdown
		FROM strategy_performance
		WHERE strategy_id = $1 AND time >= $2 AND time <= $3
		ORDER BY time DESC
	`

	rows, err := sp.db.GetPool().Query(ctx, query, strategyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategy performance: %w", err)
	}
	defer rows.Close()

	var performances []*models.StrategyPerformance
	for rows.Next() {
		perf := &models.StrategyPerformance{}
		err := rows.Scan(
			&perf.Time, &perf.StrategyID, &perf.TotalBets, &perf.WinningBets, &perf.LosingBets,
			&perf.GrossProfit, &perf.GrossLoss, &perf.NetProfit, &perf.ROI, &perf.SharpeRatio, &perf.MaxDrawdown,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performance: %w", err)
		}
		performances = append(performances, perf)
	}

	return performances, rows.Err()
}

// GetDailyRollup retrieves daily aggregated performance from the continuous aggregate
func (sp *PostgresStrategyPerformanceRepository) GetDailyRollup(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.StrategyPerformance, error) {
	// Using the continuous aggregate view for daily rollups
	query := `
		SELECT day, strategy_id, total_bets, winning_bets, losing_bets,
		       gross_profit, gross_loss, net_profit, avg_roi, avg_sharpe_ratio, min_max_drawdown
		FROM strategy_performance_daily
		WHERE strategy_id = $1 AND day >= $2 AND day <= $3
		ORDER BY day DESC
	`

	rows, err := sp.db.GetPool().Query(ctx, query, strategyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily rollup: %w", err)
	}
	defer rows.Close()

	var performances []*models.StrategyPerformance
	for rows.Next() {
		perf := &models.StrategyPerformance{}
		err := rows.Scan(
			&perf.Time, &perf.StrategyID, &perf.TotalBets, &perf.WinningBets, &perf.LosingBets,
			&perf.GrossProfit, &perf.GrossLoss, &perf.NetProfit, &perf.ROI, &perf.SharpeRatio, &perf.MaxDrawdown,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily rollup: %w", err)
		}
		performances = append(performances, perf)
	}

	return performances, rows.Err()
}
