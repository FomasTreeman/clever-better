package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Model represents an ML model
type Model struct {
	ID              uuid.UUID       `db:"id" json:"id" validate:"required,uuid4"`
	Name            string          `db:"name" json:"name" validate:"required"`
	Version         string          `db:"version" json:"version" validate:"required"`
	ModelType       string          `db:"model_type" json:"model_type" validate:"required"`
	Path            string          `db:"path" json:"path" validate:"required"`
	Metrics         json.RawMessage `db:"metrics" json:"metrics"`
	Hyperparameters json.RawMessage `db:"hyperparameters" json:"hyperparameters"`
	TrainedAt       time.Time       `db:"trained_at" json:"trained_at" validate:"required"`
	Active          bool            `db:"active" json:"active"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}

// IsActive checks if the model is currently active
func (m *Model) IsActive() bool {
	return m.Active
}

// GetMetric retrieves a metric value from the Metrics JSON
func (m *Model) GetMetric(name string) (interface{}, error) {
	if m.Metrics == nil {
		return nil, nil
	}

	var metrics map[string]interface{}
	if err := json.Unmarshal(m.Metrics, &metrics); err != nil {
		return nil, err
	}

	return metrics[name], nil
}
