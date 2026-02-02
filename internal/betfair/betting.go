package betfair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// BettingService handles bet placement and management
type BettingService struct {
	client        *BetfairClient
	betRepository repository.BetRepository
	config        BettingConfig
	logger        *log.Logger
}

// BettingConfig contains betting service configuration
type BettingConfig struct {
	MaxStake          float64
	MinStake          float64
	MaxBetsPerDay     int
	CommissionRate    float64
	DefaultOrderType  string
}

// PlaceInstruction represents a single bet placement instruction
type PlaceInstruction struct {
	OrderType      string     `json:"orderType"`
	SelectionID    uint64     `json:"selectionId"`
	Handicap       float64    `json:"handicap,omitempty"`
	Side           string     `json:"side"`
	LimitOrder     *LimitOrder `json:"limitOrder,omitempty"`
	LimitOnClose   *LimitOnClose `json:"limitOnCloseOrder,omitempty"`
}

// LimitOrder represents a limit order
type LimitOrder struct {
	Size            float64 `json:"size"`
	Price           float64 `json:"price"`
	PersistenceType string  `json:"persistenceType,omitempty"`
	MinFillSize     float64 `json:"minFillSize,omitempty"`
	BetTargetType   string  `json:"betTargetType,omitempty"`
	BetTargetSize   float64 `json:"betTargetSize,omitempty"`
}

// LimitOnClose represents a limit on close order
type LimitOnClose struct {
	Liability float64 `json:"liability"`
	Price     float64 `json:"price"`
}

// PlaceOrdersRequest represents bet placement request
type PlaceOrdersRequest struct {
	MarketID         string              `json:"marketId"`
	Instructions     []PlaceInstruction  `json:"instructions"`
	OrderMode        string              `json:"orderMode,omitempty"`
	MarginBetMode    string              `json:"marginBetMode,omitempty"`
	CustomerOrderRef string              `json:"customerOrderRef,omitempty"`
}

// PlaceOrdersResponse represents bet placement response
type PlaceOrdersResponse struct {
	MarketID      string            `json:"marketId"`
	Status        string            `json:"status"`
	InstructionReports []InstructionReport `json:"instructionReports"`
	PlaceOrdersErrors []string `json:"placeOrdersErrors,omitempty"`
}

// InstructionReport represents result of a single bet placement
type InstructionReport struct {
	Status          string          `json:"status"`
	OrderStatus     string          `json:"orderStatus"`
	BetID           string          `json:"betId"`
	PlacedDate      *time.Time      `json:"placedDate"`
	AveragePriceMatched float64     `json:"averagePriceMatched"`
	SizeMatched     float64         `json:"sizeMatched"`
	OrderRejects    []OrderReject   `json:"orderRejects,omitempty"`
}

// OrderReject represents a rejected order
type OrderReject struct {
	Status string `json:"status"`
	Reason string `json:"rejectReason"`
}

// NewBettingService creates a new betting service
func NewBettingService(
	client *BetfairClient,
	betRepository repository.BetRepository,
	config BettingConfig,
	logger *log.Logger,
) *BettingService {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	if config.MaxStake <= 0 {
		config.MaxStake = 10.0
	}

	if config.MinStake <= 0 {
		config.MinStake = 0.10
	}

	if config.CommissionRate <= 0 {
		config.CommissionRate = 0.05
	}

	if config.DefaultOrderType == "" {
		config.DefaultOrderType = "LIMIT"
	}

	return &BettingService{
		client:        client,
		betRepository: betRepository,
		config:        config,
		logger:        logger,
	}
}

// PlaceBet places a single bet on Betfair
func (b *BettingService) PlaceBet(
	ctx context.Context,
	marketID string,
	selectionID uint64,
	price float64,
	stake float64,
	side string,
) (string, error) {
	// Validate parameters
	if err := b.validateBet(price, stake, side); err != nil {
		return "", err
	}

	instruction := PlaceInstruction{
		OrderType:   "LIMIT",
		SelectionID: selectionID,
		Side:        side,
		LimitOrder: &LimitOrder{
			Size:  stake,
			Price: price,
		},
	}

	request := PlaceOrdersRequest{
		MarketID:     marketID,
		Instructions: []PlaceInstruction{instruction},
		OrderMode:    "EXECUTE",
	}

	params := map[string]interface{}{
		"marketId":     marketID,
		"instructions": []PlaceInstruction{instruction},
		"orderMode":    "EXECUTE",
	}

	result, err := b.client.makeRequest(ctx, "placeOrders", params)
	if err != nil {
		b.logger.Printf("Failed to place bet: %v", err)
		return "", err
	}

	var resp PlaceOrdersResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("failed to parse place orders response: %w", err)
	}

	if resp.Status != "SUCCESS" {
		return "", fmt.Errorf("bet placement failed: status=%s, errors=%v", resp.Status, resp.PlaceOrdersErrors)
	}

	if len(resp.InstructionReports) == 0 {
		return "", fmt.Errorf("no instruction reports in response")
	}

	report := resp.InstructionReports[0]
	if report.Status != "SUCCESS" {
		return "", fmt.Errorf("instruction failed: %s", report.Status)
	}

	b.logger.Printf("Bet placed successfully: betId=%s, price=%.2f, stake=%.2f", report.BetID, price, stake)
	return report.BetID, nil
}

// ListCurrentOrders fetches current orders from Betfair
func (b *BettingService) ListCurrentOrders(ctx context.Context, marketIDs []string) ([]CurrentOrderResponse, error) {
	params := map[string]interface{}{
		"marketIds": marketIDs,
	}

	result, err := b.client.makeRequest(ctx, "listCurrentOrders", params)
	if err != nil {
		b.logger.Printf("Failed to list current orders: %v", err)
		return nil, err
	}

	var response struct {
		CurrentOrders []CurrentOrderResponse `json:"currentOrders"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse current orders response: %w", err)
	}

	return response.CurrentOrders, nil
}

// CurrentOrderResponse represents current order information from Betfair
type CurrentOrderResponse struct {
	BetID           string    `json:"betId"`
	MarketID        string    `json:"marketId"`
	SelectionID     uint64    `json:"selectionId"`
	Price           float64   `json:"price"`
	Size            float64   `json:"size"`
	SideMatched     string    `json:"bspLiability"`
	Status          string    `json:"status"`
	PlacedDate      time.Time `json:"placedDate"`
	AveragePriceMatched float64 `json:"averagePriceMatched"`
	SizeMatched     float64   `json:"sizeMatched"`
	SizeRemaining   float64   `json:"sizeRemaining"`
}

// CancelOrders cancels unmatched bets
func (b *BettingService) CancelOrders(
	ctx context.Context,
	marketID string,
	betIDs []string,
) error {
	if len(betIDs) == 0 {
		return fmt.Errorf("at least one bet ID required")
	}

	params := map[string]interface{}{
		"marketId": marketID,
		"betIds":   betIDs,
	}

	result, err := b.client.makeRequest(ctx, "cancelOrders", params)
	if err != nil {
		b.logger.Printf("Failed to cancel orders: %v", err)
		return err
	}

	var response struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse cancel response: %w", err)
	}

	if response.Status != "SUCCESS" {
		return fmt.Errorf("cancel failed: status=%s", response.Status)
	}

	b.logger.Printf("Cancelled %d bets on market %s", len(betIDs), marketID)
	return nil
}

// UpdateBetStatus updates bet status in database from Betfair
func (b *BettingService) UpdateBetStatus(ctx context.Context, bet *models.Bet) error {
	return b.betRepository.Update(ctx, bet)
}

// validateBet validates bet parameters
func (b *BettingService) validateBet(price, stake float64, side string) error {
	if price < 1.01 || price > 1000.0 {
		return fmt.Errorf("invalid price: %.2f (must be between 1.01 and 1000)", price)
	}

	if stake < b.config.MinStake || stake > b.config.MaxStake {
		return fmt.Errorf("invalid stake: %.2f (must be between %.2f and %.2f)", stake, b.config.MinStake, b.config.MaxStake)
	}

	if side != "BACK" && side != "LAY" {
		return fmt.Errorf("invalid side: %s (must be BACK or LAY)", side)
	}

	return nil
}
