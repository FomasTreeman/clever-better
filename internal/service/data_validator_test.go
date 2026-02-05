package service

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/models"
)

const (
	validatorPrefix   = "validator: "
	expectedErrorsMsg = "expected validation errors"
	errorContainsMsg  = "expected error containing %q, got %v"
	runnerName        = "Lightning Strike"
)

func newTestValidator() *DataValidator {
	logger := log.New(os.Stderr, validatorPrefix, log.LstdFlags)
	return NewDataValidator(logger)
}

// TestRaceDataValidation tests race data validation rules using production validator
func TestRaceDataValidation(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name        string
		race        *models.Race
		expectValid bool
		shouldHave  string // error message substring to check
	}{
		{
			name: "Valid race data",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       1600,
				Grade:          "Group 1",
				Conditions:     "Good",
				Status:         "scheduled",
				NumberOfRunners: 8,
			},
			expectValid: true,
		},
		{
			name: "Missing track",
			race: &models.Race{
				ID:             uuid.New(),
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       1600,
			},
			expectValid: false,
			shouldHave:  "track is required",
		},
		{
			name: "Missing race type",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       1600,
			},
			expectValid: false,
			shouldHave:  "race_type is required",
		},
		{
			name: "Missing scheduled start",
			race: &models.Race{
				ID:        uuid.New(),
				Track:     "Ascot",
				RaceType:  "Flat",
				Distance:  1600,
			},
			expectValid: false,
			shouldHave:  "scheduled_start is required",
		},
		{
			name: "Invalid distance - zero",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       0,
			},
			expectValid: false,
			shouldHave:  "distance must be positive",
		},
		{
			name: "Invalid distance - out of range (too small)",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       50,
			},
			expectValid: false,
			shouldHave:  "distance out of range",
		},
		{
			name: "Invalid distance - out of range (too large)",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(24 * time.Hour),
				Distance:       2000,
			},
			expectValid: false,
			shouldHave:  "distance out of range",
		},
		{
			name: "Race scheduled in past",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(-2 * 24 * time.Hour),
				Distance:       1600,
			},
			expectValid: false,
			shouldHave:  "race scheduled in past",
		},
		{
			name: "Race scheduled too far in future",
			race: &models.Race{
				ID:             uuid.New(),
				Track:          "Ascot",
				RaceType:       "Flat",
				ScheduledStart: time.Now().Add(400 * 24 * time.Hour),
				Distance:       1600,
			},
			expectValid: false,
			shouldHave:  "race scheduled more than 1 year in future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateRace(tt.race)
			assertValidationErrors(t, errors, tt.expectValid, tt.shouldHave)
		})
	}
}

// TestRunnerDataValidation tests runner validation using production validator
func TestRunnerDataValidation(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name        string
		runner      *models.Runner
		expectValid bool
		shouldHave  string
	}{
		{
			name: "Valid runner",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 3,
			},
			expectValid: true,
		},
		{
			name: "Missing runner name",
			runner: &models.Runner{
				TrapNumber: 3,
			},
			expectValid: false,
			shouldHave:  "runner name is required",
		},
		{
			name: "Invalid trap number - zero",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 0,
			},
			expectValid: false,
			shouldHave:  "trap_number must be 1-8",
		},
		{
			name: "Invalid trap number - too high",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 9,
			},
			expectValid: false,
			shouldHave:  "trap_number must be 1-8",
		},
		{
			name: "Valid with optional fields",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 3,
				Age:        ptr(4),
				Sex:        ptr("M"),
			},
			expectValid: true,
		},
		{
			name: "Invalid age - negative",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 3,
				Age:        ptr(-1),
			},
			expectValid: false,
			shouldHave:  "age must be positive",
		},
		{
			name: "Invalid sex value",
			runner: &models.Runner{
				Name:       runnerName,
				TrapNumber: 3,
				Sex:        ptr("X"),
			},
			expectValid: false,
			shouldHave:  "sex must be M or F",
		},
		{
			name: "Invalid days since last run - negative",
			runner: &models.Runner{
				Name:              runnerName,
				TrapNumber:        3,
				DaysSinceLastRun:  ptr(-1),
			},
			expectValid: false,
			shouldHave:  "days_since_last_run cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateRunner(tt.runner)
			assertValidationErrors(t, errors, tt.expectValid, tt.shouldHave)
		})
	}
}

// TestValidateRunnerInRace tests runner-in-race validation
func TestValidateRunnerInRace(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name        string
		race        *models.Race
		runner      *models.Runner
		expectValid bool
		shouldHave  string
	}{
		{
			name: "Runner trap matches race",
			race: &models.Race{
				NumberOfRunners: 8,
			},
			runner: &models.Runner{
				Name:       "Lightning",
				TrapNumber: 5,
			},
			expectValid: true,
		},
		{
			name: "Runner trap exceeds race runners",
			race: &models.Race{
				NumberOfRunners: 6,
			},
			runner: &models.Runner{
				Name:       "Lightning",
				TrapNumber: 7,
			},
			expectValid: false,
			shouldHave:  "trap_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateRunnerInRace(tt.runner, tt.race)
			assertValidationErrors(t, errors, tt.expectValid, tt.shouldHave)
		})
	}
}

// TestValidateRaceUniqueness tests race uniqueness validation
func TestValidateRaceUniqueness(t *testing.T) {
	validator := newTestValidator()

	existingRace := &models.Race{
		Track:          "Ascot",
		ScheduledStart: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		RaceNumber:     3,
	}

	tests := []struct {
		name        string
		race        *models.Race
		existing    []*models.Race
		expectValid bool
	}{
		{
			name: "New race is unique",
			race: &models.Race{
				Track:          "Cheltenham",
				ScheduledStart: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
				RaceNumber:     3,
			},
			existing:    []*models.Race{existingRace},
			expectValid: true,
		},
		{
			name: "Duplicate race detected",
			race: &models.Race{
				Track:          "Ascot",
				ScheduledStart: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
				RaceNumber:     3,
			},
			existing:    []*models.Race{existingRace},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRaceUniqueness(tt.race, tt.existing)

			if tt.expectValid {
				require.NoError(t, err, "expected no uniqueness error")
			} else {
				require.Error(t, err, "expected uniqueness error")
			}
		})
	}
}

// TestTrackValidation tests track name validation
func TestTrackValidation(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name    string
		track   string
		isValid bool
	}{
		{"Valid track", "Ascot", true},
		{"Valid track", "Cheltenham", true},
		{"Empty track", "", false},
		{"Very long track name", string(make([]byte, 200)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validator.IsValidTrackName(tt.track)
			assert.Equal(t, tt.isValid, valid)
		})
	}
}

// TestRaceTypeValidation tests race type validation
func TestRaceTypeValidation(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name      string
		raceType  string
		isValid   bool
	}{
		{"Valid A1", "A1", true},
		{"Valid A5", "A5", true},
		{"Valid Maiden", "Maiden", true},
		{"Valid Chase", "Chase", true},
		{"Invalid type", "Invalid", false},
		{"Empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validator.IsValidRaceType(tt.raceType)
			assert.Equal(t, tt.isValid, valid)
		})
	}
}

// TestDistanceValidation tests distance validation
func TestDistanceValidation(t *testing.T) {
	validator := newTestValidator()

	tests := []struct {
		name       string
		distance   int
		isValid    bool
	}{
		{"Valid 400m", 400, true},
		{"Valid 500m", 500, true},
		{"Valid 800m", 800, true},
		{"Invalid 200m", 200, false},
		{"Invalid 1000m", 1000, false},
		{"Invalid negative", -100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validator.IsValidDistance(tt.distance)
			assert.Equal(t, tt.isValid, valid)
		})
	}
}

// Helper functions
func ptr[T any](v T) *T {
	return &v
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertValidationErrors(t *testing.T, errors []string, expectValid bool, shouldHave string) {
	if expectValid {
		require.Empty(t, errors, "expected no validation errors for valid input")
		return
	}

	require.NotEmpty(t, errors, expectedErrorsMsg)
	if shouldHave == "" {
		return
	}

	found := false
	for _, err := range errors {
		if err == shouldHave || contains(err, shouldHave) {
			found = true
			break
		}
	}
	require.True(t, found, errorContainsMsg, shouldHave, errors)
}
