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

// PostgresOddsRepository implements OddsRepository for PostgreSQL
type PostgresOddsRepository struct {
	db *database.DB
}

// NewPostgresOddsRepository creates a new odds repository
func NewPostgresOddsRepository(db *database.DB) OddsRepository {
	return &PostgresOddsRepository{db: db}
}

// Insert inserts a single odds snapshot
func (o *PostgresOddsRepository) Insert(ctx context.Context, odds *models.OddsSnapshot) error {
	query := `
		INSERT INTO odds_snapshots (time, race_id, runner_id, back_price, back_size, lay_price, lay_size, ltp, total_volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := o.db.GetPool().Exec(ctx, query,
		odds.Time, odds.RaceID, odds.RunnerID, odds.BackPrice, odds.BackSize,
		odds.LayPrice, odds.LaySize, odds.LTP, odds.TotalVolume,
	)
	if err != nil {
		return fmt.Errorf("failed to insert odds snapshot: %w", err)
	}

	return nil
}

// InsertBatch inserts multiple odds snapshots using high-performance batch insert
func (o *PostgresOddsRepository) InsertBatch(ctx context.Context, odds []*models.OddsSnapshot) error {
	if len(odds) == 0 {
		return nil
	}

	// Use COPY for high-performance bulk insert
	columns := []string{"time", "race_id", "runner_id", "back_price", "back_size", "lay_price", "lay_size", "ltp", "total_volume"}
	
	copyFromSource := make([][]interface{}, len(odds))
	for i, o := range odds {
		copyFromSource[i] = []interface{}{
			o.Time, o.RaceID, o.RunnerID, o.BackPrice, o.BackSize,
			o.LayPrice, o.LaySize, o.LTP, o.TotalVolume,
		}
	}

	count, err := o.db.GetPool().CopyFrom(ctx, "odds_snapshots", columns, pgx.CopyFromRows(copyFromSource))
	if err != nil {
		return fmt.Errorf("failed to batch insert odds snapshots: %w", err)
	}

	if count != int64(len(odds)) {
		return fmt.Errorf("inserted %d rows, expected %d", count, len(odds))
	}

	return nil
}

// GetByRaceID retrieves odds snapshots for a specific race within a time range
func (o *PostgresOddsRepository) GetByRaceID(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error) {
	query := `
		SELECT time, race_id, runner_id, back_price, back_size, lay_price, lay_size, ltp, total_volume
		FROM odds_snapshots
		WHERE race_id = $1 AND time >= $2 AND time <= $3
		ORDER BY time ASC
	`

	rows, err := o.db.GetPool().Query(ctx, query, raceID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query odds by race: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.OddsSnapshot
	for rows.Next() {
		snapshot := &models.OddsSnapshot{}
		err := rows.Scan(
			&snapshot.Time, &snapshot.RaceID, &snapshot.RunnerID, &snapshot.BackPrice, &snapshot.BackSize,
			&snapshot.LayPrice, &snapshot.LaySize, &snapshot.LTP, &snapshot.TotalVolume,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan odds: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// GetLatest retrieves the most recent odds snapshot for a runner in a race
func (o *PostgresOddsRepository) GetLatest(ctx context.Context, raceID, runnerID uuid.UUID) (*models.OddsSnapshot, error) {
	query := `
		SELECT time, race_id, runner_id, back_price, back_size, lay_price, lay_size, ltp, total_volume
		FROM odds_snapshots
		WHERE race_id = $1 AND runner_id = $2
		ORDER BY time DESC
		LIMIT 1
	`

	snapshot := &models.OddsSnapshot{}
	err := o.db.GetPool().QueryRow(ctx, query, raceID, runnerID).Scan(
		&snapshot.Time, &snapshot.RaceID, &snapshot.RunnerID, &snapshot.BackPrice, &snapshot.BackSize,
		&snapshot.LayPrice, &snapshot.LaySize, &snapshot.LTP, &snapshot.TotalVolume,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest odds: %w", err)
	}

	return snapshot, nil
}

// GetTimeSeriesForRunner retrieves time-series odds data for a specific runner
func (o *PostgresOddsRepository) GetTimeSeriesForRunner(ctx context.Context, runnerID uuid.UUID, start, end time.Time) ([]*models.OddsSnapshot, error) {
	query := `
		SELECT time, race_id, runner_id, back_price, back_size, lay_price, lay_size, ltp, total_volume
		FROM odds_snapshots
		WHERE runner_id = $1 AND time >= $2 AND time <= $3
		ORDER BY time ASC
	`

	rows, err := o.db.GetPool().Query(ctx, query, runnerID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.OddsSnapshot
	for rows.Next() {
		snapshot := &models.OddsSnapshot{}
		err := rows.Scan(
			&snapshot.Time, &snapshot.RaceID, &snapshot.RunnerID, &snapshot.BackPrice, &snapshot.BackSize,
			&snapshot.LayPrice, &snapshot.LaySize, &snapshot.LTP, &snapshot.TotalVolume,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan odds: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}
