package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/models"
)

// DataNormalizer normalizes data from various sources to standard format
type DataNormalizer struct {
	trackNameMap map[string]string // Maps provider track names to canonical names
	logger       *log.Logger
}

// NewDataNormalizer creates a new data normalizer
func NewDataNormalizer(logger *log.Logger) *DataNormalizer {
	return &DataNormalizer{
		trackNameMap: buildTrackNameMap(),
		logger:       logger,
	}
}

// NormalizeRace converts RaceData from any source to internal Race model
func (n *DataNormalizer) NormalizeRace(sourceRace *datasource.RaceData) (*models.Race, error) {
	if sourceRace == nil {
		return nil, fmt.Errorf("source race is nil")
	}

	race := &models.Race{
		ID:                 uuid.New(),
		SourceID:           sourceRace.SourceID,
		Track:              n.normalizeTrackName(sourceRace.Track),
		ScheduledStart:     sourceRace.ScheduledStartTime,
		RaceType:           n.normalizeRaceType(sourceRace.RaceType),
		Distance:           sourceRace.Distance,
		RaceNumber:         sourceRace.RaceNumber,
		GoingDescription:   sourceRace.GoingDescription,
		WeatherCode:        sourceRace.WeatherCode,
		Grade:              sourceRace.Grade,
		NumberOfRunners:    sourceRace.NumberOfRunners,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Convert runners
	race.Runners = make([]*models.Runner, len(sourceRace.Runners))
	for i, sourceRunner := range sourceRace.Runners {
		runner, err := n.NormalizeRunner(&sourceRunner, race.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize runner: %w", err)
		}
		race.Runners[i] = runner
	}

	return race, nil
}

// NormalizeRunner converts RunnerData from any source to internal Runner model
func (n *DataNormalizer) NormalizeRunner(sourceRunner *datasource.RunnerData, raceID uuid.UUID) (*models.Runner, error) {
	if sourceRunner == nil {
		return nil, fmt.Errorf("source runner is nil")
	}

	runner := &models.Runner{
		ID:             uuid.New(),
		RaceID:         raceID,
		SourceID:       sourceRunner.SourceID,
		TrapNumber:     sourceRunner.TrapNumber,
		Name:           sanitizeName(sourceRunner.DogName),
		Trainer:        sanitizeName(getStringPtr(sourceRunner.Trainer)),
		Odds:           sourceRunner.Odds,
		Form:           sourceRunner.Form,
		DaysSinceLastRun: sourceRunner.DaysSinceLastRun,
		Weight:         sourceRunner.Weight,
		BreedCode:      sourceRunner.BreedCode,
		Age:            sourceRunner.Age,
		Sex:            sourceRunner.Sex,
		Color:          sourceRunner.Color,
		Pedigree:       sourceRunner.Pedigree,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return runner, nil
}

// normalizeTrackName converts provider-specific track names to canonical format
func (n *DataNormalizer) normalizeTrackName(track string) string {
	if track == "" {
		return ""
	}

	// Try exact match first
	if canonical, ok := n.trackNameMap[strings.ToUpper(track)]; ok {
		return canonical
	}

	// Try case-insensitive match
	upper := strings.ToUpper(track)
	for key, canonical := range n.trackNameMap {
		if strings.EqualFold(key, upper) {
			return canonical
		}
	}

	// Return normalized version (title case)
	return strings.Title(strings.ToLower(track))
}

// normalizeRaceType converts provider-specific race types to canonical format
func (n *DataNormalizer) normalizeRaceType(raceType string) string {
	if raceType == "" {
		return ""
	}

	// Normalize case
	normalized := strings.ToUpper(strings.TrimSpace(raceType))

	// Map common variations
	raceTypeMap := map[string]string{
		"A1":            "A1",
		"A2":            "A2",
		"A3":            "A3",
		"A4":            "A4",
		"A5":            "A5",
		"A6":            "A6",
		"A7":            "A7",
		"A8":            "A8",
		"A9":            "A9",
		"OPEN RACE":     "Open Race",
		"OPEN":          "Open Race",
		"MAIDEN":        "Maiden",
		"JUVENILE":      "Juvenile",
		"RESTRICTED":    "Restricted",
		"CHASE":         "Chase",
		"MATCH":         "Match",
		"TRIAL":         "Trial",
	}

	if mapped, ok := raceTypeMap[normalized]; ok {
		return mapped
	}

	return raceType
}

// NormalizeOdds converts odds from various formats to decimal
func (n *DataNormalizer) NormalizeOdds(oddsStr string) *decimal.Decimal {
	if oddsStr == "" {
		return nil
	}

	// Try parsing as decimal (e.g., "2.5")
	d, err := decimal.NewFromString(oddsStr)
	if err == nil && d.GreaterThan(decimal.Zero) {
		return &d
	}

	// Could add support for fractional (2/1) or American odds here
	return nil
}

// NormalizeDistance ensures distance is in meters
func (n *DataNormalizer) NormalizeDistance(distance int) int {
	// Distances should already be in meters from sources
	// This is a validation pass
	if distance < 100 {
		n.logger.Printf("Warning: distance %d seems too small, may need conversion", distance)
	}
	return distance
}

// NormalizeScheduledTime ensures time is in UTC
func (n *DataNormalizer) NormalizeScheduledTime(t time.Time) time.Time {
	return t.UTC()
}

// sanitizeName removes extra whitespace and normalizes names
func sanitizeName(name *string) string {
	if name == nil || *name == "" {
		return ""
	}

	// Trim whitespace and normalize case
	trimmed := strings.TrimSpace(*name)

	// Title case for names
	return strings.Title(strings.ToLower(trimmed))
}

// getStringPtr safely dereferences string pointer
func getStringPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// buildTrackNameMap returns mapping of track name variations to canonical names
func buildTrackNameMap() map[string]string {
	return map[string]string{
		// UK Tracks (canonical format: Title Case)
		"ROMFORD":          "Romford",
		"ROMFORD STADIUM":  "Romford",
		"CRAYFORD":         "Crayford",
		"CRAYFORD STADIUM": "Crayford",
		"PERRY BARR":       "Perry Barr",
		"PERRY BARR STADIUM": "Perry Barr",
		"BELLE VUE":        "Belle Vue",
		"BELLE VUE AKELA":  "Belle Vue",
		"WIMBLEDON":        "Wimbledon",
		"WIMBLEDON STADIUM": "Wimbledon",
		"WALTHAMSTOW":      "Walthamstow",
		"WALTHAMSTOW STADIUM": "Walthamstow",
		"HARRINGAY":        "Harringay",
		"HARRINGAY STADIUM": "Harringay",
		"WEST HAM":         "West Ham",
		"WEST HAM STADIUM": "West Ham",
		"HACKNEY":          "Hackney",
		"HACKNEY STADIUM":  "Hackney",
		"CATFORD":          "Catford",
		"CATFORD STADIUM":  "Catford",
		"SHEFFIELD":        "Sheffield",
		"SHEFFIELD STADIUM": "Sheffield",
		"COVENTRY":         "Coventry",
		"COVENTRY STADIUM": "Coventry",
		"BRIGHTON":         "Brighton",
		"BRIGHTON STADIUM": "Brighton",
		"MONMORE GREEN":    "Monmore Green",
		"WOLVERHAMPTON":    "Wolverhampton",
		"SWINDON":          "Swindon",
		"SWINDON STADIUM":  "Swindon",
		"OXFORD":           "Oxford",
		"OXFORD STADIUM":   "Oxford",
		"TAUNTON":          "Taunton",
		"TAUNTON STADIUM":  "Taunton",
		"WESTON SUPER MARE": "Weston Super Mare",
		"POOLE":            "Poole",
		"POOLE STADIUM":    "Poole",
		"BOURNEMOUTH":      "Bournemouth",
		"BOURNEMOUTH STADIUM": "Bournemouth",
		// Scottish tracks
		"SHAWFIELD":       "Shawfield",
		"SHAWFIELD STADIUM": "Shawfield",
		"POWDERHALL":      "Powderhall",
		// Irish tracks
		"SHELBOURNE PARK": "Shelbourne Park",
		"HAROLD'S CROSS":  "Harold's Cross",
		"DUNMORE PARK":    "Dunmore Park",
	}
}
