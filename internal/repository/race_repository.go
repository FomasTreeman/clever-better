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

const errScanRace = "failed to scan race: %w"

// PostgresRaceRepository implements RaceRepository for PostgreSQL
type PostgresRaceRepository struct {
	db *database.DB
}

// NewPostgresRaceRepository creates a new race repository
func NewPostgresRaceRepository(db *database.DB) RaceRepository {
	return &PostgresRaceRepository{db: db}
}

// Create inserts a new race
func (r *PostgresRaceRepository) Create(ctx context.Context, race *models.Race) error {
	query := `
		INSERT INTO races (id, scheduled_start, track, race_type, distance, grade, conditions, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.GetPool().Exec(ctx, query,
		race.ID, race.ScheduledStart, race.Track, race.RaceType, race.Distance,
		race.Grade, race.Conditions, race.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create race: %w", err)
	}

	return nil
}

// CreateWithTx inserts a new race using a provided transaction
func (r *PostgresRaceRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, race *models.Race) error {
	query := `
		INSERT INTO races (id, scheduled_start, track, race_type, distance, grade, conditions, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		race.ID, race.ScheduledStart, race.Track, race.RaceType, race.Distance,
		race.Grade, race.Conditions, race.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create race within transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a race by ID
func (r *PostgresRaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Race, error) {
	query := `
		SELECT id, scheduled_start, actual_start, track, race_type, distance, grade, 
		       conditions, status, created_at, updated_at
		FROM races WHERE id = $1
	`

	race := &models.Race{}
	err := r.db.GetPool().QueryRow(ctx, query, id).Scan(
		&race.ID, &race.ScheduledStart, &race.ActualStart, &race.Track, &race.RaceType,
		&race.Distance, &race.Grade, &race.Conditions, &race.Status, &race.CreatedAt, &race.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get race: %w", err)
	}

	return race, nil
}

// GetUpcoming retrieves upcoming races ordered by scheduled start time
func (r *PostgresRaceRepository) GetUpcoming(ctx context.Context, limit int) ([]*models.Race, error) {
	query := `
		SELECT id, scheduled_start, actual_start, track, race_type, distance, grade,
		       conditions, status, created_at, updated_at
		FROM races
		WHERE status = 'scheduled' AND scheduled_start > NOW()
		ORDER BY scheduled_start ASC
		LIMIT $1
	`

	rows, err := r.db.GetPool().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming races: %w", err)
	}
	defer rows.Close()

	var races []*models.Race
	for rows.Next() {
		race := &models.Race{}
		err := rows.Scan(
			&race.ID, &race.ScheduledStart, &race.ActualStart, &race.Track, &race.RaceType,
			&race.Distance, &race.Grade, &race.Conditions, &race.Status, &race.CreatedAt, &race.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(errScanRace, err)
		}
		races = append(races, race)
	}

	return races, rows.Err()
}

// GetByDateRange retrieves races within a date range
func (r *PostgresRaceRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]*models.Race, error) {
	query := `
		SELECT id, scheduled_start, actual_start, track, race_type, distance, grade,
		       conditions, status, created_at, updated_at
		FROM races
		WHERE scheduled_start >= $1 AND scheduled_start <= $2
		ORDER BY scheduled_start ASC
	`

	rows, err := r.db.GetPool().Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query races by date range: %w", err)
	}
	defer rows.Close()

	var races []*models.Race
	for rows.Next() {
		race := &models.Race{}
		err := rows.Scan(
			&race.ID, &race.ScheduledStart, &race.ActualStart, &race.Track, &race.RaceType,
			&race.Distance, &race.Grade, &race.Conditions, &race.Status, &race.CreatedAt, &race.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(errScanRace, err)
		}
		races = append(races, race)
	}

	return races, rows.Err()
}

// GetByTrackAndDate retrieves races by track and date for deduplication
func (r *PostgresRaceRepository) GetByTrackAndDate(ctx context.Context, track string, date time.Time) ([]*models.Race, error) {
	// Find races on the same track on the same day (within 24 hours)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, scheduled_start, actual_start, track, race_type, distance, grade,
		       conditions, status, created_at, updated_at
		FROM races
		WHERE track = $1 AND scheduled_start >= $2 AND scheduled_start < $3
		ORDER BY scheduled_start ASC
	`

	rows, err := r.db.GetPool().Query(ctx, query, track, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query races by track and date: %w", err)
	}
	defer rows.Close()

	var races []*models.Race
	for rows.Next() {
		race := &models.Race{}
		err := rows.Scan(
			&race.ID, &race.ScheduledStart, &race.ActualStart, &race.Track, &race.RaceType,
			&race.Distance, &race.Grade, &race.Conditions, &race.Status, &race.CreatedAt, &race.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(errScanRace, err)
		}
		races = append(races, race)
	}

	return races, rows.Err()
}

// Update updates an existing race
func (r *PostgresRaceRepository) Update(ctx context.Context, race *models.Race) error {
	query := `
		UPDATE races SET
			actual_start = $2, status = $3, updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := r.db.GetPool().Exec(ctx, query, race.ID, race.ActualStart, race.Status)
	if err != nil {
		return fmt.Errorf("failed to update race: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Delete deletes a race
func (r *PostgresRaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM races WHERE id = $1"

	commandTag, err := r.db.GetPool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete race: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}
