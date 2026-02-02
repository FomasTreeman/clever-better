package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Race represents a race event in the system
type Race struct {
	ID              uuid.UUID           `db:"id" json:"id" validate:"required,uuid4"`
	ScheduledStart  time.Time           `db:"scheduled_start" json:"scheduled_start" validate:"required"`
	ActualStart     *time.Time          `db:"actual_start" json:"actual_start"`
	Track           string              `db:"track" json:"track" validate:"required"`
	RaceType        string              `db:"race_type" json:"race_type" validate:"required"`
	Distance        int                 `db:"distance" json:"distance" validate:"required,gt=0"`
	Grade           string              `db:"grade" json:"grade"`
	Conditions      json.RawMessage     `db:"conditions" json:"conditions"`
	Status          string              `db:"status" json:"status" validate:"oneof=scheduled started finished cancelled"`
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at" json:"updated_at"`
}

// IsUpcoming checks if the race hasn't started yet
func (r *Race) IsUpcoming() bool {
	return r.ActualStart == nil && r.Status == "scheduled"
}

// IsFinished checks if the race has completed
func (r *Race) IsFinished() bool {
	return r.Status == "finished" && r.ActualStart != nil
}

// TimeToStart returns the duration until race start
func (r *Race) TimeToStart() time.Duration {
	return time.Until(r.ScheduledStart)
}
