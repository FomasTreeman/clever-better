package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// RacingPostClient implements DataSource for Racing Post API
type RacingPostClient struct {
	httpClient *RateLimitedHTTPClient
	baseURL    string
	apiKey     string
	enabled    bool
	logger     *log.Logger
}

// RacingPostRace represents a race from Racing Post API
type RacingPostRace struct {
	ID              string                  `json:"id"`
	Date            string                  `json:"date"`
	Track           string                  `json:"venueName"`
	ScheduledTime   string                  `json:"scheduledTime"`
	RaceType        string                  `json:"raceType"`
	Distance        int                     `json:"distance"`
	RaceNumber      int                     `json:"raceNumber"`
	Grade           *string                 `json:"grade"`
	Going           *string                 `json:"going"`
	Weather         *string                 `json:"weather"`
	NumberOfRunners int                     `json:"numberOfRunners"`
	Runners         []RacingPostRunnerEntry `json:"runners"`
}

// RacingPostRunnerEntry represents a runner entry from Racing Post API
type RacingPostRunnerEntry struct {
	ID             string           `json:"id"`
	TrapNumber     int              `json:"trapNumber"`
	DogName        string           `json:"name"`
	Trainer        *string          `json:"trainer"`
	Odds           *string          `json:"odds"`
	Win            *string          `json:"winOdds"`
	Place          *string          `json:"placeOdds"`
	Form           *string          `json:"form"`
	DaysLastRun    *int             `json:"daysSinceLastRun"`
	Weight         *string          `json:"weight"`
	Breed          *string          `json:"breed"`
	Age            *int             `json:"age"`
	Sex            *string          `json:"sex"`
	Color          *string          `json:"color"`
	Sire           *string          `json:"sire"`
	Dam            *string          `json:"dam"`
	BreedingInfo   *string          `json:"breeding"`
	FormFigures    *string          `json:"formFigures"`
	Rating         *int             `json:"rating"`
	HandicapWeight *string          `json:"handicapWeight"`
}

// NewRacingPostClient creates a new Racing Post API client
func NewRacingPostClient(httpClient *RateLimitedHTTPClient, apiKey string, enabled bool, logger *log.Logger) *RacingPostClient {
	return &RacingPostClient{
		httpClient: httpClient,
		baseURL:    "https://api.racingpost.com/v1",
		apiKey:     apiKey,
		enabled:    enabled,
		logger:     logger,
	}
}

// FetchRaces retrieves races within the specified date range
func (c *RacingPostClient) FetchRaces(ctx context.Context, startDate, endDate time.Time) ([]RaceData, error) {
	if !c.enabled {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, dataSourceDisabledMsg, nil)
	}

	url := fmt.Sprintf("%s/races?from=%s&to=%s", c.baseURL, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, "failed to create request", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, "failed to fetch races", err)
	}
	defer resp.Body.Close()

	// Handle authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewDataSourceError("racing_post", ErrCodeAuthenticationFailed, "invalid API key", nil)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, NewDataSourceError("racing_post", ErrCodeRateLimitExceeded, "rate limit exceeded", nil)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, NewDataSourceError("racing_post", ErrCodeServerError, fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	var rpRaces []RacingPostRace
	if err := json.NewDecoder(resp.Body).Decode(&rpRaces); err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeInvalidData, "failed to parse response", err)
	}

	// Convert to RaceData
	races := make([]RaceData, len(rpRaces))
	for i, rpRace := range rpRaces {
		race, err := c.convertRace(&rpRace)
		if err != nil {
			c.logger.Printf("Failed to convert race %s: %v", rpRace.ID, err)
			continue
		}
		races[i] = *race
	}

	return races, nil
}

// FetchRaceDetails retrieves detailed information for a specific race
func (c *RacingPostClient) FetchRaceDetails(ctx context.Context, raceID string) (*RaceData, error) {
	if !c.enabled {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, dataSourceDisabledMsg, nil)
	}

	url := fmt.Sprintf("%s/races/%s", c.baseURL, raceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, "failed to create request", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeNetworkError, "failed to fetch race details", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewDataSourceError("racing_post", ErrCodeNotFound, "race not found", nil)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewDataSourceError("racing_post", ErrCodeAuthenticationFailed, "invalid API key", nil)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewDataSourceError("racing_post", ErrCodeServerError, fmt.Sprintf("unexpected status %d", resp.StatusCode), nil)
	}

	var rpRace RacingPostRace
	if err := json.NewDecoder(resp.Body).Decode(&rpRace); err != nil {
		return nil, NewDataSourceError("racing_post", ErrCodeInvalidData, "failed to parse response", err)
	}

	return c.convertRace(&rpRace)
}

// Name returns the data source name
func (c *RacingPostClient) Name() string {
	return "racing_post"
}

// IsEnabled returns whether this data source is enabled
func (c *RacingPostClient) IsEnabled() bool {
	return c.enabled
}

// convertRace converts Racing Post race format to RaceData
func (c *RacingPostClient) convertRace(rpRace *RacingPostRace) (*RaceData, error) {
	scheduledTime, err := time.Parse(time.RFC3339, rpRace.ScheduledTime)
	if err != nil {
		scheduledTime = time.Now()
	}

	race := &RaceData{
		SourceID:           rpRace.ID,
		Track:              rpRace.Track,
		ScheduledStartTime: scheduledTime,
		RaceType:           rpRace.RaceType,
		Distance:           rpRace.Distance,
		RaceNumber:         rpRace.RaceNumber,
		GoingDescription:   rpRace.Going,
		WeatherCode:        rpRace.Weather,
		Grade:              rpRace.Grade,
		NumberOfRunners:    rpRace.NumberOfRunners,
		Runners:            make([]RunnerData, len(rpRace.Runners)),
		CreatedAt:          time.Now(),
	}

	// Convert runners
	for i, rpRunner := range rpRace.Runners {
		runner := RunnerData{
			SourceID:        rpRunner.ID,
			TrapNumber:      rpRunner.TrapNumber,
			DogName:         rpRunner.DogName,
			Trainer:         rpRunner.Trainer,
			Form:            rpRunner.Form,
			DaysSinceLastRun: rpRunner.DaysLastRun,
			Weight:          parseDecimal(rpRunner.Weight),
			BreedCode:       rpRunner.Breed,
			Age:             rpRunner.Age,
			Sex:             rpRunner.Sex,
			Color:           rpRunner.Color,
			Pedigree:        rpRunner.Sire,
		}

		// Parse odds if available
		if rpRunner.Odds != nil {
			if odds, err := parseDecimalOdds(*rpRunner.Odds); odds != nil {
				runner.Odds = odds
			} else if c.logger != nil && err != nil {
				c.logger.Printf("Failed to parse odds for runner %s: %v", rpRunner.DogName, err)
			}
		}

		race.Runners[i] = runner
	}

	return race, nil
}

// parseDecimal parses a string to decimal.Decimal, returning nil if invalid
func parseDecimal(s *string) *decimal.Decimal {
	if s == nil || *s == "" {
		return nil
	}
	d, err := decimal.NewFromString(*s)
	if err != nil {
		return nil
	}
	return &d
}

// parseDecimalOdds parses odds in various formats to decimal
func parseDecimalOdds(oddsStr string) (*decimal.Decimal, error) {
	// Try parsing as decimal directly
	d, err := decimal.NewFromString(oddsStr)
	if err == nil {
		return &d, nil
	}

	// Could add support for fractional (e.g., "2/1") or American odds formats here
	return nil, fmt.Errorf("invalid odds format: %s", oddsStr)
}
