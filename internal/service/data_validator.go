package service

import (
	"fmt"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/models"
)

// DataValidator validates race and runner data
type DataValidator struct {
	logger *log.Logger
}

// NewDataValidator creates a new data validator
func NewDataValidator(logger *log.Logger) *DataValidator {
	return &DataValidator{logger: logger}
}

// ValidateRace validates race data for required fields and constraints
func (v *DataValidator) ValidateRace(race *models.Race) []string {
	var errors []string

	// Check required fields
	if race.Track == "" {
		errors = append(errors, "track is required")
	}

	if race.ScheduledStart.IsZero() {
		errors = append(errors, "scheduled_start is required")
	}

	if race.RaceType == "" {
		errors = append(errors, "race_type is required")
	}

	if race.Distance <= 0 {
		errors = append(errors, fmt.Sprintf("distance must be positive, got %d", race.Distance))
	}

	if race.Distance < 100 || race.Distance > 1000 {
		errors = append(errors, fmt.Sprintf("distance out of range (100-1000m), got %d", race.Distance))
	}

	// Check scheduled start is not too far in the past or future
	now := time.Now()
	if race.ScheduledStart.Before(now.Add(-24 * time.Hour)) {
		errors = append(errors, fmt.Sprintf("race scheduled in past by %v", now.Sub(race.ScheduledStart)))
	}

	if race.ScheduledStart.After(now.Add(365 * 24 * time.Hour)) {
		errors = append(errors, fmt.Sprintf("race scheduled more than 1 year in future"))
	}

	return errors
}

// ValidateRunner validates runner data for required fields and constraints
func (v *DataValidator) ValidateRunner(runner *models.Runner) []string {
	var errors []string

	// Check required fields
	if runner.Name == "" {
		errors = append(errors, "runner name is required")
	}

	if runner.TrapNumber < 1 || runner.TrapNumber > 8 {
		errors = append(errors, fmt.Sprintf("trap_number must be 1-8 for greyhounds, got %d", runner.TrapNumber))
	}

	// Validate optional fields if present
	if runner.DaysSinceLastRun != nil && *runner.DaysSinceLastRun < 0 {
		errors = append(errors, "days_since_last_run cannot be negative")
	}

	if runner.Age != nil && *runner.Age <= 0 {
		errors = append(errors, "age must be positive")
	}

	if runner.Sex != nil && *runner.Sex != "" {
		if *runner.Sex != "M" && *runner.Sex != "F" {
			errors = append(errors, fmt.Sprintf("sex must be M or F, got %s", *runner.Sex))
		}
	}

	return errors
}

// ValidateRaceUniqueness checks if race is unique by track and scheduled start
func (v *DataValidator) ValidateRaceUniqueness(race *models.Race, existingRaces []*models.Race) error {
	for _, existing := range existingRaces {
		if existing.Track == race.Track && 
		   existing.ScheduledStart.Equal(race.ScheduledStart) &&
		   existing.RaceNumber == race.RaceNumber {
			return fmt.Errorf("race already exists: %s on %s at %v", race.Track, existing.ScheduledStart.Format("2006-01-02"), race.RaceNumber)
		}
	}
	return nil
}

// ValidateRunnerInRace validates runner is appropriate for the race
func (v *DataValidator) ValidateRunnerInRace(runner *models.Runner, race *models.Race) []string {
	var errors []string

	// Trap number should be <= number of runners
	if runner.TrapNumber > race.NumberOfRunners {
		errors = append(errors, fmt.Sprintf("trap_number %d exceeds race runners %d", runner.TrapNumber, race.NumberOfRunners))
	}

	return errors
}

// IsValidTrackName checks if track name is in expected format
func (v *DataValidator) IsValidTrackName(track string) bool {
	// Simple validation: non-empty and reasonable length
	return len(track) > 0 && len(track) < 100
}

// IsValidRaceType checks if race type is valid
func (v *DataValidator) IsValidRaceType(raceType string) bool {
	// Common greyhound race types
	validTypes := map[string]bool{
		"A1": true, "A2": true, "A3": true, "A4": true, "A5": true, "A6": true,
		"A7": true, "A8": true, "A9": true,
		"Open Race": true,
		"Maiden": true,
		"Juvenile": true,
		"Restricted": true,
		"Chase": true,
		"Match": true,
		"Trial": true,
	}

	return validTypes[raceType]
}

// IsValidDistance checks if distance is reasonable for greyhound racing
func (v *DataValidator) IsValidDistance(distance int) bool {
	// Typical greyhound racing distances: 280m, 285m, 400m, 450m, 500m, 575m, 660m, 710m, 800m
	validDistances := map[int]bool{
		280: true, 285: true, 300: true, 400: true, 450: true, 460: true,
		500: true, 550: true, 575: true, 600: true, 660: true, 710: true, 800: true,
	}

	return validDistances[distance]
}
