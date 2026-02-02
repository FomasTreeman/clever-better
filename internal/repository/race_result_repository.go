package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/models"
)

// RaceResultRepository defines operations for race results
type RaceResultRepository interface {
	// Insert inserts a single race result
	Insert(ctx context.Context, result *models.RaceResult) error

	// InsertBatch inserts multiple race results efficiently
	InsertBatch(ctx context.Context, results []*models.RaceResult) error

	// GetByRaceID retrieves all results for a specific race
	GetByRaceID(ctx context.Context, raceID uuid.UUID) (*models.RaceResult, error)

	// GetByTimeRange retrieves race results within a time range
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.RaceResult, error)

	// GetByStatus retrieves race results with a specific status
	GetByStatus(ctx context.Context, status string, limit int) ([]*models.RaceResult, error)

	// GetDailySummary retrieves aggregated daily results from the continuous aggregate
	GetDailySummary(ctx context.Context, raceID uuid.UUID, start, end time.Time) ([]*models.RaceResultSummary, error)

	// Update updates an existing race result
	Update(ctx context.Context, result *models.RaceResult) error

	// Delete deletes a race result
	Delete(ctx context.Context, raceID uuid.UUID, resultTime time.Time) error
}
