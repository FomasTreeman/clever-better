package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

// PostgresStrategyRepository implements StrategyRepository for PostgreSQL
type PostgresStrategyRepository struct {
	db *database.DB
}

// NewPostgresStrategyRepository creates a new strategy repository
func NewPostgresStrategyRepository(db *database.DB) StrategyRepository {
	return &PostgresStrategyRepository{db: db}
}

// Create inserts a new strategy
func (s *PostgresStrategyRepository) Create(ctx context.Context, strategy *models.Strategy) error {
	query := `
		INSERT INTO strategies (id, name, description, parameters, active)
		VALUES ($1, $2, $3, $4, $5)
	`

	if strategy.Name == "" {
		return models.ErrStrategyNameRequired
	}

	_, err := s.db.GetPool().Exec(ctx, query,
		strategy.ID, strategy.Name, strategy.Description, strategy.Parameters, strategy.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create strategy: %w", err)
	}

	return nil
}

// GetByID retrieves a strategy by ID
func (s *PostgresStrategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Strategy, error) {
	query := `
		SELECT id, name, description, parameters, active, created_at, updated_at
		FROM strategies WHERE id = $1
	`

	strategy := &models.Strategy{}
	err := s.db.GetPool().QueryRow(ctx, query, id).Scan(
		&strategy.ID, &strategy.Name, &strategy.Description, &strategy.Parameters,
		&strategy.Active, &strategy.CreatedAt, &strategy.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	return strategy, nil
}

// GetByName retrieves a strategy by name
func (s *PostgresStrategyRepository) GetByName(ctx context.Context, name string) (*models.Strategy, error) {
	query := `
		SELECT id, name, description, parameters, active, created_at, updated_at
		FROM strategies
		WHERE name = $1
		LIMIT 1
	`

	strategy := &models.Strategy{}
	err := s.db.GetPool().QueryRow(ctx, query, name).Scan(
		&strategy.ID, &strategy.Name, &strategy.Description, &strategy.Parameters,
		&strategy.Active, &strategy.CreatedAt, &strategy.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy by name: %w", err)
	}

	return strategy, nil
}

// GetActive retrieves all active strategies
func (s *PostgresStrategyRepository) GetActive(ctx context.Context) ([]*models.Strategy, error) {
	query := `
		SELECT id, name, description, parameters, active, created_at, updated_at
		FROM strategies
		WHERE active = true
		ORDER BY name ASC
	`

	rows, err := s.db.GetPool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active strategies: %w", err)
	}
	defer rows.Close()

	var strategies []*models.Strategy
	for rows.Next() {
		strategy := &models.Strategy{}
		err := rows.Scan(
			&strategy.ID, &strategy.Name, &strategy.Description, &strategy.Parameters,
			&strategy.Active, &strategy.CreatedAt, &strategy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}
		strategies = append(strategies, strategy)
	}

	return strategies, rows.Err()
}

// Update updates an existing strategy
func (s *PostgresStrategyRepository) Update(ctx context.Context, strategy *models.Strategy) error {
	query := `
		UPDATE strategies SET
			name = $2, description = $3, parameters = $4, active = $5, updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := s.db.GetPool().Exec(ctx, query,
		strategy.ID, strategy.Name, strategy.Description, strategy.Parameters, strategy.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to update strategy: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Delete deletes a strategy
func (s *PostgresStrategyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM strategies WHERE id = $1"

	commandTag, err := s.db.GetPool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete strategy: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}
