package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RaceResult represents the outcome and results of a completed race
type RaceResult struct {
	Time         time.Time       `db:"time" json:"time"`
	RaceID       uuid.UUID       `db:"race_id" json:"race_id" validate:"required"`
	WinnerTrap   *int            `db:"winner_trap" json:"winner_trap"`
	Positions    json.RawMessage `db:"positions" json:"positions"` // JSON array of runner positions
	TotalPayouts decimal.Decimal `db:"total_payouts" json:"total_payouts"`
	Status       string          `db:"status" json:"status" validate:"required,oneof=pending completed cancelled"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// RaceResultSummary represents daily aggregated race results (from continuous aggregate)
type RaceResultSummary struct {
	Day             time.Time       `db:"day" json:"day"`
	RaceID          uuid.UUID       `db:"race_id" json:"race_id"`
	TotalRaces      int64           `db:"total_races" json:"total_races"`
	Winners         int64           `db:"winners" json:"winners"`
	TotalPayoutsSum decimal.Decimal `db:"total_payouts_sum" json:"total_payouts_sum"`
	StatusCount     int64           `db:"status_count" json:"status_count"`
	LastUpdated     time.Time       `db:"last_updated" json:"last_updated"`
}

// PositionsData represents the structured positions data
type PositionsData struct {
	Runners []RunnerPosition `json:"runners"`
}

// RunnerPosition represents a runner's final position in the race
type RunnerPosition struct {
	RunnerID    uuid.UUID       `json:"runner_id"`
	TrapNumber  int             `json:"trap_number"`
	Position    int             `json:"position"`
	Timeform    *string         `json:"timeform,omitempty"`
	SP          decimal.Decimal `json:"sp"`
	PlacePayout decimal.Decimal `json:"place_payout"`
}

// ParsePositions parses the positions JSON data
func (rr *RaceResult) ParsePositions() (*PositionsData, error) {
	if rr.Positions == nil {
		return nil, ErrInvalidRaceResult
	}

	var posData PositionsData
	if err := json.Unmarshal(rr.Positions, &posData); err != nil {
		return nil, err
	}

	return &posData, nil
}

// Errors
var (
	ErrRaceResultNotFound    = NewValidationError("race_result_not_found", "race result not found")
	ErrInvalidRaceResult     = NewValidationError("invalid_race_result", "invalid race result data")
	ErrRaceResultDuplicate   = NewValidationError("race_result_duplicate", "race result already exists for this race")
)
