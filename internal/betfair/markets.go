package betfair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/models"
)

// MarketCatalogue represents market catalog information from Betfair
type MarketCatalogue struct {
	MarketID    string                 `json:"marketId"`
	MarketName  string                 `json:"marketName"`
	Description MarketDescription      `json:"description"`
	Runners     []RunnerCatalog        `json:"runners"`
	TotalMatched float64               `json:"totalMatched"`
	Status      string                 `json:"status"`
}

// MarketDescription contains market metadata
type MarketDescription struct {
	PersistenceType string    `json:"persistenceType"`
	MarketType      string    `json:"marketType"`
	EventID         string    `json:"eventId"`
	EventTypeID     string    `json:"eventTypeId"`
	CompetitionID   string    `json:"competitionId"`
	ScheduledTime   time.Time `json:"scheduledTime"`
	BspMarket       bool      `json:"bspMarket"`
	TurnInPlayEnabled bool    `json:"turnInPlayEnabled"`
	PriceLadderDefinition *PriceLadder `json:"priceLadderDefinition"`
}

// PriceLadder represents price ladder configuration
type PriceLadder struct {
	Type string `json:"type"`
}

// RunnerCatalog represents a runner in the market catalog
type RunnerCatalog struct {
	SelectionID uint64      `json:"selectionId"`
	RunnerName  string      `json:"runnerName"`
	Status      string      `json:"status"`
	Metadata    map[string]string `json:"metadata"`
}

// MarketBook represents current market state and odds
type MarketBook struct {
	MarketID         string        `json:"marketId"`
	IsMarketDataOnly bool          `json:"isMarketDataOnly"`
	Status           string        `json:"status"`
	BetDelay         int           `json:"betDelay"`
	BSPReconciled    bool          `json:"bspReconciled"`
	Complete         bool          `json:"complete"`
	Runners          []Runner      `json:"runners"`
	TotalMatched     float64       `json:"totalMatched"`
	TotalAvailable   float64       `json:"totalAvailable"`
	LastMatchTime    *time.Time    `json:"lastMatchTime"`
	CountdownMillis  int           `json:"countdownMillis"`
	OpenDate         *time.Time    `json:"openDate"`
	PersistenceType  string        `json:"persistenceType"`
	MarketType       string        `json:"marketType"`
	Regulators       []string      `json:"regulators"`
	BetCount         int64         `json:"betCount"`
	CommissionRange  []float64     `json:"commissionRange"`
	PriceRange       []float64     `json:"priceRange"`
	Version          int64         `json:"version"`
}

// Runner represents a runner in the market with current odds
type Runner struct {
	SelectionID      uint64         `json:"selectionId"`
	Handicap         float64        `json:"handicap"`
	Status           string         `json:"status"`
	LastPriceTraded  float64        `json:"lastPriceTraded"`
	AdjustmentFactor float64        `json:"adjustmentFactor"`
	TotalMatched     float64        `json:"totalMatched"`
	TotalAvailable   float64        `json:"totalAvailable"`
	ExchangePrices   ExchangePrices `json:"ex"`
	SpareBool        bool           `json:"sp"`
	StartingPrice    *StartingPrice `json:"startingPrice"`
}

// ExchangePrices represents back/lay prices on the exchange
type ExchangePrices struct {
	AvailableToBack []PriceSize `json:"availableToBack"`
	AvailableToLay  []PriceSize `json:"availableToLay"`
	TradedVolume    []PriceSize `json:"tradedVolume"`
}

// PriceSize represents a price level with size
type PriceSize struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// StartingPrice represents BSP starting price
type StartingPrice struct {
	NearPrice     float64 `json:"nearPrice"`
	FarPrice      float64 `json:"farPrice"`
	BackStakeTaken []PriceSize `json:"backStakeTaken"`
	LayLiabilityTaken []PriceSize `json:"layLiabilityTaken"`
}

// ListMarketCatalogParams parameters for listing market catalog
type ListMarketCatalogParams struct {
	Filter          MarketFilter `json:"filter"`
	MarketProjection []string    `json:"marketProjection"`
	Sort            string       `json:"sort"`
	MaxResults      int          `json:"maxResults"`
}

// MarketFilter for filtering market catalog
type MarketFilter struct {
	EventTypeIDs  []string `json:"eventTypeIds,omitempty"`
	EventIDs      []string `json:"eventIds,omitempty"`
	CompetitionIDs []string `json:"competitionIds,omitempty"`
	MarketIDs     []string `json:"marketIds,omitempty"`
	Venues        []string `json:"venues,omitempty"`
	BspOnly       *bool    `json:"bspOnly,omitempty"`
	TurnInPlayOnly *bool   `json:"turnInPlayOnly,omitempty"`
	PersistenceType string `json:"persistenceType,omitempty"`
	MarketTypes   []string `json:"marketTypes,omitempty"`
	WithOrders    []string `json:"withOrders,omitempty"`
}

// ListMarketBookParams parameters for listing market book
type ListMarketBookParams struct {
	MarketIDs       []string      `json:"marketIds"`
	PriceProjection []string      `json:"priceProjection"`
	OrderProjection string        `json:"orderProjection,omitempty"`
	MatchProjection string        `json:"matchProjection,omitempty"`
	KeepAlive       bool          `json:"keepAlive"`
}

// ListMarketCatalog fetches market catalog for specified filters
func (c *BetfairClient) ListMarketCatalog(
	ctx context.Context,
	eventTypeID string,
	marketTypes []string,
	venues []string,
	maxResults int,
) ([]MarketCatalogue, error) {
	if maxResults <= 0 || maxResults > 1000 {
		maxResults = 100
	}

	filter := MarketFilter{
		EventTypeIDs: []string{eventTypeID},
		MarketTypes:  marketTypes,
		Venues:       venues,
	}

	if len(venues) > 0 {
		filter.Venues = venues
	}

	params := map[string]interface{}{
		"filter":          filter,
		"marketProjection": []string{"RUNNER_DESCRIPTION", "MARKET_DESCRIPTION", "EVENT", "COMPETITION", "EVENT_TYPE"},
		"sort":            "FIRST_TO_START",
		"maxResults":      maxResults,
	}

	result, err := c.makeRequest(ctx, "listMarketCatalogue", params)
	if err != nil {
		c.logger.Printf("Failed to list market catalog: %v", err)
		return nil, err
	}

	var catalogs []MarketCatalogue
	if err := json.Unmarshal(result, &catalogs); err != nil {
		return nil, fmt.Errorf("failed to parse market catalog response: %w", err)
	}

	c.logger.Printf("Retrieved %d markets", len(catalogs))
	return catalogs, nil
}

// ListGreyhoundRaceMarkets fetches greyhound racing markets for upcoming races
func (c *BetfairClient) ListGreyhoundRaceMarkets(ctx context.Context) ([]MarketCatalogue, error) {
	// Event type ID 4339 is greyhound racing
	eventTypeID := "4339"
	marketTypes := []string{"WIN", "PLACE"}

	return c.ListMarketCatalog(ctx, eventTypeID, marketTypes, nil, 100)
}

// ListMarketBook fetches current market state and prices
func (c *BetfairClient) ListMarketBook(
	ctx context.Context,
	marketIDs []string,
	priceProjection []string,
) ([]MarketBook, error) {
	if len(marketIDs) == 0 {
		return nil, fmt.Errorf("at least one market ID required")
	}

	if len(priceProjection) == 0 {
		priceProjection = []string{"EX_BEST_OFFERS", "EX_TRADED"}
	}

	params := map[string]interface{}{
		"marketIds":       marketIDs,
		"priceProjection": priceProjection,
		"keepAlive":       false,
	}

	result, err := c.makeRequest(ctx, "listMarketBook", params)
	if err != nil {
		c.logger.Printf("Failed to list market book: %v", err)
		return nil, err
	}

	var books []MarketBook
	if err := json.Unmarshal(result, &books); err != nil {
		return nil, fmt.Errorf("failed to parse market book response: %w", err)
	}

	c.logger.Printf("Retrieved market data for %d markets", len(books))
	return books, nil
}

// GetMarketPrices returns simplified price data for a market
func (c *BetfairClient) GetMarketPrices(ctx context.Context, marketID string) (map[uint64]*models.Price, error) {
	books, err := c.ListMarketBook(ctx, []string{marketID}, []string{"EX_BEST_OFFERS"})
	if err != nil {
		return nil, err
	}

	if len(books) == 0 {
		return nil, fmt.Errorf("no market book data returned")
	}

	prices := make(map[uint64]*models.Price)
	book := books[0]

	for _, runner := range book.Runners {
		backPrice := 0.0
		backSize := 0.0
		layPrice := 0.0
		laySize := 0.0

		// Get best back and lay prices
		if len(runner.ExchangePrices.AvailableToBack) > 0 {
			backPrice = runner.ExchangePrices.AvailableToBack[0].Price
			backSize = runner.ExchangePrices.AvailableToBack[0].Size
		}
		if len(runner.ExchangePrices.AvailableToLay) > 0 {
			layPrice = runner.ExchangePrices.AvailableToLay[0].Price
			laySize = runner.ExchangePrices.AvailableToLay[0].Size
		}

		prices[runner.SelectionID] = &models.Price{
			BackPrice: backPrice,
			BackSize:  backSize,
			LayPrice:  layPrice,
			LaySize:   laySize,
			Timestamp: time.Now(),
		}
	}

	return prices, nil
}
