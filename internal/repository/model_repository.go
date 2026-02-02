package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

// PostgresModelRepository implements ModelRepository for PostgreSQL
type PostgresModelRepository struct {
	db *database.DB
}

// NewPostgresModelRepository creates a new model repository
func NewPostgresModelRepository(db *database.DB) ModelRepository {
	return &PostgresModelRepository{db: db}
}

// Create inserts a new ML model
func (m *PostgresModelRepository) Create(ctx context.Context, model *models.Model) error {
	query := `
		INSERT INTO models (id, name, version, model_type, path, hyperparameters, metrics, trained_at, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := m.db.GetPool().Exec(ctx, query,
		model.ID, model.Name, model.Version, model.ModelType, model.Path, model.Hyperparameters, model.Metrics, model.TrainedAt, model.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	return nil
}

// GetByID retrieves a model by ID
func (m *PostgresModelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Model, error) {
	query := `
		SELECT id, name, version, model_type, path, hyperparameters, metrics, trained_at, active, created_at, updated_at
		FROM models WHERE id = $1
	`

	model := &models.Model{}
	err := m.db.GetPool().QueryRow(ctx, query, id).Scan(
		&model.ID, &model.Name, &model.Version, &model.ModelType, &model.Path, &model.Hyperparameters,
		&model.Metrics, &model.TrainedAt, &model.Active, &model.CreatedAt, &model.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return model, nil
}

// GetActive retrieves all active models
func (m *PostgresModelRepository) GetActive(ctx context.Context) ([]*models.Model, error) {
	query := `
		SELECT id, name, version, model_type, path, hyperparameters, metrics, trained_at, active, created_at, updated_at
		FROM models
		WHERE active = true
		ORDER BY name ASC, version DESC
	`

	rows, err := m.db.GetPool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active models: %w", err)
	}
	defer rows.Close()

	var models []*models.Model
	for rows.Next() {
		model := &models.Model{}
		err := rows.Scan(
			&model.ID, &model.Name, &model.Version, &model.ModelType, &model.Path, &model.Hyperparameters,
			&model.Metrics, &model.TrainedAt, &model.Active, &model.CreatedAt, &model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, model)
	}

	return models, rows.Err()
}

// GetByVersion retrieves a specific model version
func (m *PostgresModelRepository) GetByVersion(ctx context.Context, name, version string) (*models.Model, error) {
	query := `
		SELECT id, name, version, model_type, path, hyperparameters, metrics, trained_at, active, created_at, updated_at
		FROM models
		WHERE name = $1 AND version = $2
	`

	model := &models.Model{}
	err := m.db.GetPool().QueryRow(ctx, query, name, version).Scan(
		&model.ID, &model.Name, &model.Version, &model.ModelType, &model.Path, &model.Hyperparameters,
		&model.Metrics, &model.TrainedAt, &model.Active, &model.CreatedAt, &model.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model by version: %w", err)
	}

	return model, nil
}

// Update updates an existing model
func (m *PostgresModelRepository) Update(ctx context.Context, model *models.Model) error {
	query := `
		UPDATE models SET
			path = $2, hyperparameters = $3, metrics = $4, trained_at = $5, active = $6, updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := m.db.GetPool().Exec(ctx, query, model.ID, model.Path, model.Hyperparameters, model.Metrics, model.TrainedAt, model.Active)
	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}

// SetActive sets a model as active and deactivates other versions
func (m *PostgresModelRepository) SetActive(ctx context.Context, id uuid.UUID) error {
	// First get the model to find its name
	model, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := m.db.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deactivate all other versions of this model
	_, err = tx.Exec(ctx, "UPDATE models SET active = false WHERE name = $1 AND id != $2", model.Name, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate other versions: %w", err)
	}

	// Activate this version
	_, err = tx.Exec(ctx, "UPDATE models SET active = true, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to activate model: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
