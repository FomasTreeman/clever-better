package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Runner represents a runner (horse) in a race
type Runner struct {
	ID                  uuid.UUID       `db:"id" json:"id" validate:"required,uuid4"`
	RaceID              uuid.UUID       `db:"race_id" json:"race_id" validate:"required,uuid4"`
	TrapNumber          int             `db:"trap_number" json:"trap_number" validate:"required,gt=0,lt=15"`
	Name                string          `db:"name" json:"name" validate:"required"`
	FormRating          *float64        `db:"form_rating" json:"form_rating"`
	Weight              *float64        `db:"weight" json:"weight"`
	Trainer             string          `db:"trainer" json:"trainer"`
	DaysSinceLastRace   *int            `db:"days_since_last_race" json:"days_since_last_race"`
	Metadata            json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt           time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at" json:"updated_at"`
	Race                *Race           `db:"-" json:"race,omitempty"`
}

// GetFormRating returns the form rating or 0 if nil
func (r *Runner) GetFormRating() float64 {
	if r.FormRating == nil {
		return 0
	}
	return *r.FormRating
}

// GetRecentForm returns the days since last race or returns high number if nil
func (r *Runner) GetRecentForm() int {
	if r.DaysSinceLastRace == nil {
		return 999
	}
	return *r.DaysSinceLastRace
}
