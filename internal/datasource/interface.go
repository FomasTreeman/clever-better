package datasource

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DataSource defines the interface for fetching racing data from external providers
type DataSource interface {
	// FetchRaces retrieves races within the specified date range
	FetchRaces(ctx context.Context, startDate, endDate time.Time) ([]RaceData, error)

	// FetchRaceDetails retrieves detailed information for a specific race
	FetchRaceDetails(ctx context.Context, raceID string) (*RaceData, error)

	// Name returns the name of the data source
	Name() string

	// IsEnabled returns whether this data source is currently enabled
	IsEnabled() bool
}

// RaceData represents normalized race data from any data source
type RaceData struct {
	SourceID          string              `json:"source_id"`           // Provider's unique race ID
	Track             string              `json:"track"`               // Track name (e.g., "Romford")
	ScheduledStartTime time.Time          `json:"scheduled_start_time"` // Race start time UTC
	RaceType          string              `json:"race_type"`           // Race classification (e.g., "A1")
	Distance          int                 `json:"distance"`            // Distance in meters
	RaceNumber        int                 `json:"race_number"`         // Race number at track
	GoingDescription  *string             `json:"going_description"`   // Track condition
	WeatherCode       *string             `json:"weather_code"`        // Weather condition
	Grade             *string             `json:"grade"`               // Race grade
	NumberOfRunners   int                 `json:"number_of_runners"`   // Expected number of runners
	Runners           []RunnerData        `json:"runners"`             // Runner details
	CreatedAt         time.Time           `json:"created_at"`          // When data was fetched
}

// RunnerData represents normalized runner/dog data from any data source
type RunnerData struct {
	SourceID        string           `json:"source_id"`        // Provider's unique runner ID
	TrackRunnerID   *string          `json:"track_runner_id"`  // Track-specific identifier
	TrapNumber      int              `json:"trap_number"`      // Trap/draw number (1-8)
	DogName         string           `json:"dog_name"`         // Dog's racing name
	Trainer         *string          `json:"trainer"`          // Trainer name
	Odds            *decimal.Decimal `json:"odds"`             // Decimal odds if available
	Form            *string          `json:"form"`             // Recent form string
	DaysSinceLastRun *int            `json:"days_since_last_run"` // Days since last race
	Weight          *decimal.Decimal `json:"weight"`           // Dog weight in kg
	BreedCode       *string          `json:"breed_code"`       // Breed designation
	Age             *int             `json:"age"`              // Age in months
	Sex             *string          `json:"sex"`              // M (male) or F (female)
	Color           *string          `json:"color"`            // Color/markings
	Pedigree        *string          `json:"pedigree"`         // Sire/dam information
}

// DataSourceError represents errors from data source operations
type DataSourceError struct {
	Source  string // Data source name
	Code    string // Error code (e.g., "rate_limit_exceeded")
	Message string // Error message
	Err     error  // Underlying error
}

func (e DataSourceError) Error() string {
	if e.Err != nil {
		return e.Source + ": " + e.Code + ": " + e.Message + " (" + e.Err.Error() + ")"
	}
	return e.Source + ": " + e.Code + ": " + e.Message
}

// Common error codes
const (
	ErrCodeRateLimitExceeded = "rate_limit_exceeded"
	ErrCodeAuthenticationFailed = "authentication_failed"
	ErrCodeNotFound = "not_found"
	ErrCodeInvalidData = "invalid_data"
	ErrCodeNetworkError = "network_error"
	ErrCodeServerError = "server_error"
	ErrCodeUnknown = "unknown"
)

// Error constructors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrNotFound = errors.New("data not found")
	ErrInvalidData = errors.New("invalid data format")
	ErrNetworkError = errors.New("network error")
	ErrServerError = errors.New("server error")
)

// NewDataSourceError creates a new data source error
func NewDataSourceError(source, code, message string, err error) DataSourceError {
	return DataSourceError{
		Source:  source,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
