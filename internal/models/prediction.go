package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Prediction represents a model prediction for a runner in a race
type Prediction struct {
	ID         uuid.UUID       `db:"id" json:"id" validate:"required,uuid4"`
	ModelID    uuid.UUID       `db:"model_id" json:"model_id" validate:"required,uuid4"`
	RaceID     uuid.UUID       `db:"race_id" json:"race_id" validate:"required,uuid4"`
	RunnerID   uuid.UUID       `db:"runner_id" json:"runner_id" validate:"required,uuid4"`
	Probability float64        `db:"probability" json:"probability" validate:"required,gte=0,lte=1"`
	Confidence float64        `db:"confidence" json:"confidence" validate:"required,gte=0,lte=1"`
	Features   json.RawMessage `db:"features" json:"features"`
	PredictedAt time.Time      `db:"predicted_at" json:"predicted_at" validate:"required"`
}

// GetFeature retrieves a feature value from the Features JSON
func (p *Prediction) GetFeature(name string) (interface{}, error) {
	if p.Features == nil {
		return nil, nil
	}

	var features map[string]interface{}
	if err := json.Unmarshal(p.Features, &features); err != nil {
		return nil, err
	}

	return features[name], nil
}

// MeetsThreshold checks if the confidence meets the given threshold
func (p *Prediction) MeetsThreshold(threshold float64) bool {
	return p.Confidence >= threshold
}
