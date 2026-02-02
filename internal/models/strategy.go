package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Strategy represents a trading strategy
type Strategy struct {
	ID          uuid.UUID       `db:"id" json:"id" validate:"required,uuid4"`
	Name        string          `db:"name" json:"name" validate:"required,min=1,max=255"`
	Description string          `db:"description" json:"description"`
	Parameters  json.RawMessage `db:"parameters" json:"parameters"`
	Active      bool            `db:"active" json:"active"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// GetParameter retrieves a parameter value from the Parameters JSON
func (s *Strategy) GetParameter(key string) (interface{}, error) {
	if s.Parameters == nil {
		return nil, nil
	}

	var params map[string]interface{}
	if err := json.Unmarshal(s.Parameters, &params); err != nil {
		return nil, err
	}

	return params[key], nil
}

// Validate performs basic validation on the strategy
func (s *Strategy) Validate() error {
	if s.Name == "" {
		return ErrStrategyNameRequired
	}
	return nil
}
