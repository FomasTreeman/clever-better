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

// PostgresRaceResultRepository implements RaceResultRepository for PostgreSQL
type PostgresRaceResultRepository struct {
	db *database.DB
}

// NewPostgresRaceResultRepository creates a new race result repository
func NewPostgresRaceResultRepository(db *database.DB) RaceResultRepository {
	return &PostgresRaceResultRepository{db: db}
}

// Insert inserts a single race result
func (r *PostgresRaceResultRepository) Insert(ctx context.Context, result *models.RaceResult) error {
	query := `
		INSERT INTO race_results (time, race_id, winner_trap, positions, total_payouts, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.GetPool().Exec(ctx, query,
		result.Time, result.RaceID, result.WinnerTrap, result.Positions, result.TotalPayouts, result.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to insert race result: %w", err)
	}

	return nil
}

// InsertBatch inserts multiple race results using high-performance batch insert
func (r *PostgresRaceResultRepository) InsertBatch(ctx context.Context, results []*models.RaceResult) error {
	if len(results) == 0 {
		return nil
	}

	// Use COPY for high-performance bulk insert
	columns := []string{"time", "race_id", "winner_trap", "positions", "total_payouts", "status"}

	copyFromSource := make([][]interface{}, len(results))
	for i, res := range results {
		copyFromSource[i] = []interface{}{
			res.Time, res.RaceID, res.WinnerTrap, res.Positions, res.TotalPayouts, res.Status,
		}
	}

	copyCount, err := r.db.GetPool().CopyFrom(
		ctx,
		pgx.Identifier{"race_results"},
		columns,
		pgx.CopyFromRows(copyFromSource),
	)
	if err != nil {
		return fmt.Errorf("failed to batch insert race results: %w", err)
	}

	if copyCount != int64(len(results)) {
		return fmt.Errorf("inserted %d rows, expected %d", copyCount, len(results))
	}

	return nil
}

// GetByRaceID retrieves the result for a specific race
func (r *PostgresRaceResultRepository) GetByRaceID(ctx context.Context, raceID uuid.UUID) (*models.RaceResult, error) {
	query := `
		SELECT time, race_id, winner_trap, positions, total_payouts, status, created_at, updated_at
		FROM race_results
		WHERE race_id = $1
		ORDER BY time DESC
		LIMIT 1
	`

	result := &models.RaceResult{}
	err := r.db.GetPool().QueryRow(ctx, query, raceID).Scan(
		&result.Time, &result.RaceID, &result.WinnerTrap, &result.Positions,
		&result.TotalPayouts, &result.Status, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrRaceResultNotFound
		}
		return nil, fmt.Errorf("failed to query race result: %w", err)
	}

	return result, nil
}

// GetByTimeRange retrieves race results within a time range
func (r *PostgresRaceResultRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.RaceResult, error) {
	query := `
		SELECT time, race_id, winner_trap, positions, total_payouts, status, created_at, updated_at
		FROM race_results
		WHERE time >= $1 AND time <= $2
		ORDER BY time DESC
	`

	rows, err := r.db.GetPool().Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query race results by time range: %w", err)
	}
	defer rows.Close()

	var results []*models.RaceResult
	for rows.Next() {
		result := &models.RaceResult{}
		err := rows.Scan(
			&result.Time, &result.RaceID, &result.WinnerTrap, &result.Positions,
			&result.TotalPayouts, &result.Status, &result.CreatedAt, &result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan race result: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating race results: %w", err)
	}

	return results, nil
}

// GetByStatus retrieves race results with a specific status
func (r *PostgresRaceResultRepository) GetByStatus(ctx context.Context, status string, limit int) ([]*models.RaceResult, error) {
	query := `
		SELECT time, race_id, winner_trap, positions, total_payouts, status, created_at, updated_at
		FROM race_results
		WHERE status = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := r.db.GetPool().Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query race results by status: %w", err)
	}
	defer rows.Close()

	var results []*models.RaceResult
	for rows.Next() {
		result := &models.RaceResult{}
		err := rows.Scan(
			&result.Time, &result.RaceID, &result.WinnerTrap, &result.Positions,
			&result.TotalPayouts, &result.Status, &result.CreatedAt, &result.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan race result: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating race results: %w", err)
	}

	return results, nil
}

// GetDailySummary retrieves aggregated daily results from the continuous aggregate
func (r *PostgresRaceResultRepository) GetDailySummary(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.RaceResultSummary, error) {
	query := `
		SELECT day, race_id, total_races, winners, total_payouts_sum, status_count, last_updated
		FROM race_results_daily
		WHERE race_id = $1 AND day >= $2 AND day <= $3
		ORDER BY day DESC
	`

	rows, err := r.db.GetPool().Query(ctx, query, raceID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query race results daily summary: %w", err)
	}
	defer rows.Close()

	var summaries []*models.RaceResultSummary
	for rows.Next() {
		summary := &models.RaceResultSummary{}
		err := rows.Scan(
			&summary.Day, &summary.RaceID, &summary.TotalRaces, &summary.Winners,
			&summary.TotalPayoutsSum, &summary.StatusCount, &summary.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan race result summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating race result summaries: %w", err)
	}

	return summaries, nil
}

// Update updates an existing race result
func (r *PostgresRaceResultRepository) Update(ctx context.Context, result *models.RaceResult) error {
	query := `
		UPDATE race_results SET
			winner_trap = $2, positions = $3, total_payouts = $4, status = $5, updated_at = NOW()
		WHERE race_id = $6 AND time = $7
	`

	commandTag, err := r.db.GetPool().Exec(ctx, query,
		result.WinnerTrap, result.Positions, result.TotalPayouts, result.Status,
		result.RaceID, result.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to update race result: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrRaceResultNotFound
	}

	return nil
}

// Delete deletes a race result
func (r *PostgresRaceResultRepository) Delete(ctx context.Context, raceID uuid.UUID, resultTime time.Time) error {
	query := `
		DELETE FROM race_results
		WHERE race_id = $1 AND time = $2
	`

	commandTag, err := r.db.GetPool().Exec(ctx, query, raceID, resultTime)
	if err != nil {
		return fmt.Errorf("failed to delete race result: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrRaceResultNotFound
	}

	return nil
}
