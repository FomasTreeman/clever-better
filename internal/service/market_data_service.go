package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

const dateLayout = "2006-01-02"

// MarketDataService handles storing historical market data for backtesting
type MarketDataService struct {
	betfairClient    *betfair.BetfairClient
	raceRepository   repository.RaceRepository
	runnerRepository repository.RunnerRepository
	oddsRepository   repository.OddsRepository
	logger           *log.Logger
}

// NewMarketDataService creates a new market data service
func NewMarketDataService(
	betfairClient *betfair.BetfairClient,
	raceRepository repository.RaceRepository,
	runnerRepository repository.RunnerRepository,
	oddsRepository repository.OddsRepository,
	logger *log.Logger,
) *MarketDataService {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	return &MarketDataService{
		betfairClient:    betfairClient,
		raceRepository:   raceRepository,
		runnerRepository: runnerRepository,
		oddsRepository:   oddsRepository,
		logger:           logger,
	}
}

// FetchAndStoreMarketData fetches market data for a date range and stores it
func (m *MarketDataService) FetchAndStoreMarketData(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
) error {
	m.logger.Printf("Fetching market data from %s to %s", startDate.Format(dateLayout), endDate.Format(dateLayout))

	// Get greyhound racing markets
	catalogs, err := m.betfairClient.ListGreyhoundRaceMarkets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch market catalog: %w", err)
	}

	m.logger.Printf("Found %d markets for greyhound racing", len(catalogs))

	// Process each market
	for _, catalog := range catalogs {
		if err := m.storeMarketCatalog(ctx, &catalog); err != nil {
			m.logger.Printf("Error storing market catalog: %v", err)
			continue
		}
	}

	return nil
}

// storeMarketCatalog converts and stores market catalog data
func (m *MarketDataService) storeMarketCatalog(ctx context.Context, catalog *betfair.MarketCatalogue) error {
	// Check if race already exists
	existingRaces, err := m.raceRepository.GetByDateRange(
		ctx,
		catalog.Description.ScheduledTime.Add(-1*time.Hour),
		catalog.Description.ScheduledTime.Add(1*time.Hour),
	)
	if err == nil && len(existingRaces) > 0 {
		m.logger.Printf("Race already exists for market %s, skipping", catalog.MarketID)
		return nil
	}

	// Create Race record
	race := &models.Race{
		ID:               uuid.New(),
		SourceID:         catalog.MarketID,
		Track:            m.extractTrack(catalog.MarketName),
		RaceType:         catalog.Description.MarketType,
		ScheduledStart:   catalog.Description.ScheduledTime,
		Status:           models.RaceStatusScheduled,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Try to extract other details from market name/description
	if catalog.Description.EventID != "" {
		race.EventID = catalog.Description.EventID
	}

	// Store race
	if err := m.raceRepository.Create(ctx, race); err != nil {
		return fmt.Errorf("failed to create race: %w", err)
	}

	m.logger.Printf("Stored race: %s (%s)", race.Track, catalog.MarketName)

	// Store runners
	for _, runner := range catalog.Runners {
		runnerModel := &models.Runner{
			ID:           uuid.New(),
			RaceID:       race.ID,
			SourceID:     fmt.Sprintf("%d", runner.SelectionID),
			Name:         runner.RunnerName,
			TrackNumber:  m.extractTrapNumber(runner.RunnerName),
			Status:       models.RunnerStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := m.runnerRepository.Create(ctx, runnerModel); err != nil {
			m.logger.Printf("Error storing runner %s: %v", runner.RunnerName, err)
			continue
		}
	}

	m.logger.Printf("Stored %d runners for race %s", len(catalog.Runners), race.Track)
	return nil
}

// storeHistoricalPrices stores historical odds data
// findRunnerIDBySourceID finds the runner ID matching the given selection ID
func (m *MarketDataService) findRunnerIDBySourceID(
	ctx context.Context,
	raceID uuid.UUID,
	selectionID int64,
) (uuid.UUID, error) {
	runners, err := m.runnerRepository.GetByRaceID(ctx, raceID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, r := range runners {
		if r.SourceID == fmt.Sprintf("%d", selectionID) {
			return r.ID, nil
		}
	}

	return uuid.Nil, nil
}

// extractPricesFromRunner extracts back and lay prices from a runner
func extractPricesFromRunner(runner *betfair.RunnerBook) (backPrice, backSize, layPrice, laySize, tradedVolume float64) {
	if len(runner.ExchangePrices.AvailableToBack) > 0 {
		backPrice = runner.ExchangePrices.AvailableToBack[0].Price
		backSize = runner.ExchangePrices.AvailableToBack[0].Size
	}

	if len(runner.ExchangePrices.AvailableToLay) > 0 {
		layPrice = runner.ExchangePrices.AvailableToLay[0].Price
		laySize = runner.ExchangePrices.AvailableToLay[0].Size
	}

	if len(runner.ExchangePrices.TradedVolume) > 0 {
		tradedVolume = runner.ExchangePrices.TradedVolume[0].Size
	}

	return
}

func (m *MarketDataService) storeHistoricalPrices(
	ctx context.Context,
	marketID string,
	raceID uuid.UUID,
) error {
	// Fetch market book for prices
	books, err := m.betfairClient.ListMarketBook(
		ctx,
		[]string{marketID},
		[]string{"EX_BEST_OFFERS", "EX_TRADED"},
	)
	if err != nil {
		return fmt.Errorf("failed to fetch market book: %w", err)
	}

	if len(books) == 0 {
		return fmt.Errorf("no market book data returned")
	}

	book := books[0]
	snapshots := make([]*models.OddsSnapshot, 0, len(book.Runners))

	for _, runner := range book.Runners {
		// Find runner ID from database
		runnerID, err := m.findRunnerIDBySourceID(ctx, raceID, runner.SelectionID)
		if err != nil || runnerID == uuid.Nil {
			continue
		}

		// Extract prices
		backPrice, backSize, layPrice, laySize, tradedVolume := extractPricesFromRunner(&runner)

		snapshot := &models.OddsSnapshot{
			RaceID:          raceID,
			RunnerID:        runnerID,
			MarketID:        marketID,
			SelectionID:     runner.SelectionID,
			BackPrice:       backPrice,
			BackSize:        backSize,
			LayPrice:        layPrice,
			LaySize:         laySize,
			TradedVolume:    tradedVolume,
			LastPriceTraded: runner.LastPriceTraded,
			Timestamp:       time.Now(),
		}

		snapshots = append(snapshots, snapshot)
	}

	// Batch insert snapshots
	if len(snapshots) > 0 {
		if err := m.oddsRepository.InsertBatch(ctx, snapshots); err != nil {
			return fmt.Errorf("failed to insert odds snapshots: %w", err)
		}
		m.logger.Printf("Stored %d odds snapshots for market %s", len(snapshots), marketID)
	}

	return nil
}

// BackfillMarketData performs bulk historical data import
func (m *MarketDataService) BackfillMarketData(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
) error {
	m.logger.Printf("Starting backfill of market data from %s to %s", startDate.Format(dateLayout), endDate.Format(dateLayout))

	currentDate := startDate
	for currentDate.Before(endDate) {
		m.logger.Printf("Processing date: %s", currentDate.Format(dateLayout))

		if err := m.FetchAndStoreMarketData(ctx, currentDate, currentDate.Add(24*time.Hour)); err != nil {
			m.logger.Printf("Error processing date %s: %v", currentDate.Format(dateLayout), err)
		}

		currentDate = currentDate.Add(24 * time.Hour)
	}

	m.logger.Printf("Backfill complete")
	return nil
}

// extractTrack extracts track name from market name
func (m *MarketDataService) extractTrack(marketName string) string {
	// Market names typically follow format: "HH:MM Trackname"
	// This is a simple extraction - in production, more sophisticated parsing would be needed
	return marketName
}

// extractTrapNumber extracts trap/lane number from runner name
func (m *MarketDataService) extractTrapNumber(runnerName string) int {
	// Runner names typically have trap numbers
	// This is a placeholder - actual parsing would depend on Betfair's naming convention
	return 0
}
