package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// BetfairHistoricalClient implements DataSource for Betfair historical data
type BetfairHistoricalClient struct {
	httpClient *RateLimitedHTTPClient
	baseURL    string
	apiKey     string
	sessionID  string
	enabled    bool
	logger     *log.Logger
}

// BetfairStreamMessage represents a Betfair Stream API JSON message
type BetfairStreamMessage struct {
	Op            string                 `json:"op"`
	ClosedMarkets []BetfairClosedMarket  `json:"mc"`
	MarketUpdates []BetfairMarketUpdate  `json:"mc"`
}

// BetfairClosedMarket represents a closed market in Betfair Stream format
type BetfairClosedMarket struct {
	MarketID   string         `json:"id"`
	IsMarketDataOnly bool      `json:"omM"`
	Runners    []BetfairRunner `json:"rc"`
	Con        BetfairMarketCatalog `json:"con"`
}

// BetfairMarketUpdate represents a market update
type BetfairMarketUpdate struct {
	MarketID string         `json:"id"`
	Runners  []BetfairRunner `json:"rc"`
}

// BetfairRunner represents a runner/dog
type BetfairRunner struct {
	SelectionID int64              `json:"id"`
	Name        string             `json:"name"`
	Status      string             `json:"status"`
	Metadata    map[string]string  `json:"metadata"`
}

// BetfairMarketCatalog contains market metadata
type BetfairMarketCatalog struct {
	Market BetfairMarketMeta `json:"market"`
}

// BetfairMarketMeta contains metadata
type BetfairMarketMeta struct {
	Name          string            `json:"name"`
	Description   BetfairMarketDesc `json:"description"`
	RunnersCount  int               `json:"numberOfRunners"`
	ScheduledTime string            `json:"bspReconciled"`
}

// BetfairMarketDesc contains market description
type BetfairMarketDesc struct {
	ScheduledTime string `json:"scheduledDate"`
	Distance      int    `json:"distance"`
	Track         string `json:"venue"`
	Type          string `json:"raceType"`
}

// NewBetfairHistoricalClient creates a new Betfair historical data client
func NewBetfairHistoricalClient(httpClient *RateLimitedHTTPClient, apiKey string, enabled bool, logger *log.Logger) *BetfairHistoricalClient {
	return &BetfairHistoricalClient{
		httpClient: httpClient,
		baseURL:    "https://historicaldata.betfair.com/api",
		apiKey:     apiKey,
		enabled:    enabled,
		logger:     logger,
	}
}

// FetchRaces retrieves races within the specified date range
func (c *BetfairHistoricalClient) FetchRaces(ctx context.Context, startDate, endDate time.Time) ([]RaceData, error) {
	if !c.enabled {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "data source disabled", nil)
	}

	// Get available files in the date range
	files, err := c.GetAvailableFiles(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		// No data available - return empty list without error
		return []RaceData{}, nil
	}

	var allRaces []RaceData
	parser := NewBetfairCSVParser(c.logger)

	// Download and parse each file
	for _, filename := range files {
		fileReader, err := c.DownloadHistoricalFile(ctx, filename)
		if err != nil {
			if c.logger != nil {
				c.logger.Printf("Failed to download file %s: %v", filename, err)
			}
			continue
		}

		races, err := parser.ParseCSVReader(fileReader)
		fileReader.Close()
		if err != nil {
			if c.logger != nil {
				c.logger.Printf("Failed to parse file %s: %v", filename, err)
			}
			continue
		}

		allRaces = append(allRaces, races...)
	}

	if c.logger != nil {
		c.logger.Printf("Fetched %d races from Betfair historical data", len(allRaces))
	}

	return allRaces, nil
}

// FetchRaceDetails retrieves detailed information for a specific race
func (c *BetfairHistoricalClient) FetchRaceDetails(ctx context.Context, raceID string) (*RaceData, error) {
	if !c.enabled {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "data source disabled", nil)
	}

	// Race details would be fetched from the available files
	// For now, return a not-found error indicating the method is not yet fully implemented
	// In production, this would query the available historical data or API
	return nil, NewDataSourceError("betfair_historical", ErrCodeNotFound, fmt.Sprintf("race details not found or method not fully implemented: %s", raceID), nil)
}

// Name returns the data source name
func (c *BetfairHistoricalClient) Name() string {
	return "betfair_historical"
}

// IsEnabled returns whether this data source is enabled
func (c *BetfairHistoricalClient) IsEnabled() bool {
	return c.enabled
}

// parseStreamMarket converts a Betfair Stream market to RaceData
func (c *BetfairHistoricalClient) parseStreamMarket(market BetfairClosedMarket) (*RaceData, error) {
	// Extract market metadata to determine track and race info
	// This would involve parsing Betfair's market names and descriptions
	// Betfair market names typically follow format: "HH:MM Trackname DistanceClass"

	race := &RaceData{
		SourceID:    market.MarketID,
		Runners:     make([]RunnerData, 0, len(market.Runners)),
		CreatedAt:   time.Now(),
	}

	// Parse runners
	for _, runner := range market.Runners {
		runnerData := RunnerData{
			SourceID:   fmt.Sprintf("%d", runner.SelectionID),
			DogName:    runner.Name,
		}
		race.Runners = append(race.Runners, runnerData)
	}

	return race, nil
}

// BetfairCSVParser handles CSV format parsing
type BetfairCSVParser struct {
	logger *log.Logger
}

// NewBetfairCSVParser creates a new CSV parser
func NewBetfairCSVParser(logger *log.Logger) *BetfairCSVParser {
	return &BetfairCSVParser{logger: logger}
}

// ParseCSVReader parses Betfair CSV historical data
func (p *BetfairCSVParser) ParseCSVReader(reader io.Reader) ([]RaceData, error) {
	// Read and parse CSV format
	// Betfair CSV historical format typically includes:
	// - Track name
	// - Race time
	// - Distance
	// - Race type
	// - Runner information (trap, name, odds, result)

	var races []RaceData

	// Read CSV content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("CSV file too small: %d lines", len(lines))
	}

	// Parse header and data rows
	// Implementation depends on specific CSV schema
	// This is a placeholder for actual parsing logic

	return races, nil
}

// DownloadHistoricalFile downloads a historical data file from Betfair
func (c *BetfairHistoricalClient) DownloadHistoricalFile(ctx context.Context, filename string) (io.ReadCloser, error) {
	if !c.enabled {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "data source disabled", nil)
	}

	url := fmt.Sprintf("%s/files/%s", c.baseURL, filename)

	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "failed to download file", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, NewDataSourceError("betfair_historical", ErrCodeNotFound, fmt.Sprintf("file not found: %s", filename), nil)
	}

	return resp.Body, nil
}

// GetAvailableFiles lists available historical data files
func (c *BetfairHistoricalClient) GetAvailableFiles(ctx context.Context, startDate, endDate time.Time) ([]string, error) {
	if !c.enabled {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "data source disabled", nil)
	}

	url := fmt.Sprintf("%s/files?from=%s&to=%s", c.baseURL, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, NewDataSourceError("betfair_historical", ErrCodeNetworkError, "failed to list files", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewDataSourceError("betfair_historical", ErrCodeServerError, fmt.Sprintf("unexpected status: %d", resp.StatusCode), nil)
	}

	var files []string
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, NewDataSourceError("betfair_historical", ErrCodeInvalidData, "invalid response format", err)
	}

	return files, nil
}
