package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

// PostgresRunnerRepository implements RunnerRepository for PostgreSQL
type PostgresRunnerRepository struct {
	db *database.DB
}

// NewPostgresRunnerRepository creates a new runner repository
func NewPostgresRunnerRepository(db *database.DB) RunnerRepository {
	return &PostgresRunnerRepository{db: db}
}

// Create inserts a new runner
func (r *PostgresRunnerRepository) Create(ctx context.Context, runner *models.Runner) error {
	query := `
		INSERT INTO runners (id, race_id, trap_number, name, form_rating, weight, trainer, days_since_last_race, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.GetPool().Exec(ctx, query,
		runner.ID, runner.RaceID, runner.TrapNumber, runner.Name, runner.FormRating,
		runner.Weight, runner.Trainer, runner.DaysSinceLastRace, runner.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	return nil
}

// GetByID retrieves a runner by ID
func (r *PostgresRunnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Runner, error) {
	query := `
		SELECT id, race_id, trap_number, name, form_rating, weight, trainer, 
		       days_since_last_race, metadata, created_at, updated_at
		FROM runners WHERE id = $1
	`

	runner := &models.Runner{}
	err := r.db.GetPool().QueryRow(ctx, query, id).Scan(
		&runner.ID, &runner.RaceID, &runner.TrapNumber, &runner.Name, &runner.FormRating,
		&runner.Weight, &runner.Trainer, &runner.DaysSinceLastRace, &runner.Metadata, &runner.CreatedAt, &runner.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get runner: %w", err)
	}

	return runner, nil
}

// GetByRaceID retrieves all runners for a race
func (r *PostgresRunnerRepository) GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Runner, error) {
	query := `
		SELECT id, race_id, trap_number, name, form_rating, weight, trainer,
		       days_since_last_race, metadata, created_at, updated_at
		FROM runners
		WHERE race_id = $1
		ORDER BY trap_number ASC
	`

	rows, err := r.db.GetPool().Query(ctx, query, raceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query runners by race: %w", err)
	}
	defer rows.Close()

	var runners []*models.Runner
	for rows.Next() {
		runner := &models.Runner{}
		err := rows.Scan(
			&runner.ID, &runner.RaceID, &runner.TrapNumber, &runner.Name, &runner.FormRating,
			&runner.Weight, &runner.Trainer, &runner.DaysSinceLastRace, &runner.Metadata, &runner.CreatedAt, &runner.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan runner: %w", err)
		}
		runners = append(runners, runner)
	}

	return runners, rows.Err()
}

// Update updates an existing runner
func (r *PostgresRunnerRepository) Update(ctx context.Context, runner *models.Runner) error {
	query := `
		UPDATE runners SET
			trap_number = $2, name = $3, form_rating = $4, weight = $5,
			trainer = $6, days_since_last_race = $7, metadata = $8, updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := r.db.GetPool().Exec(ctx, query,
		runner.ID, runner.TrapNumber, runner.Name, runner.FormRating,
		runner.Weight, runner.Trainer, runner.DaysSinceLastRace, runner.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update runner: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Delete deletes a runner
func (r *PostgresRunnerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM runners WHERE id = $1"

	commandTag, err := r.db.GetPool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete runner: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}
