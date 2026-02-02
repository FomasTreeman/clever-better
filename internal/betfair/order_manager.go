package betfair

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// OrderManager manages the lifecycle of bets
type OrderManager struct {
	bettingService  *BettingService
	betRepository   repository.BetRepository
	pollingInterval time.Duration
	done            chan struct{}
	mu              sync.Mutex
	metrics         *OrderMetrics
	logger          *log.Logger
}

// OrderMetrics tracks order management performance
type OrderMetrics struct {
	OrdersMonitored  int64
	OrdersMatched    int64
	OrdersSettled    int64
	OrdersCancelled   int64
	SyncErrors       int64
	LastSyncTime     time.Time
	AverageSyncTime  time.Duration
}

// NewOrderManager creates a new order manager
func NewOrderManager(
	bettingService *BettingService,
	betRepository repository.BetRepository,
	pollingInterval time.Duration,
	logger *log.Logger,
) *OrderManager {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	if pollingInterval <= 0 {
		pollingInterval = 30 * time.Second
	}

	return &OrderManager{
		bettingService:  bettingService,
		betRepository:   betRepository,
		pollingInterval: pollingInterval,
		done:            make(chan struct{}),
		metrics:         &OrderMetrics{},
		logger:          logger,
	}
}

// MonitorOrders starts monitoring pending bets
func (om *OrderManager) MonitorOrders(ctx context.Context) error {
	om.logger.Printf("Starting order monitoring with interval: %v", om.pollingInterval)

	ticker := time.NewTicker(om.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			startTime := time.Now()

			if err := om.syncOrderStatus(ctx); err != nil {
				om.logger.Printf("Error syncing order status: %v", err)
				om.mu.Lock()
				om.metrics.SyncErrors++
				om.mu.Unlock()
			}

			om.mu.Lock()
			om.metrics.LastSyncTime = time.Now()
			om.metrics.AverageSyncTime = time.Since(startTime)
			om.mu.Unlock()

		case <-ctx.Done():
			om.logger.Printf("Order monitoring stopped")
			return ctx.Err()

		case <-om.done:
			om.logger.Printf("Order monitoring terminated")
			return nil
		}
	}
}

// syncOrderStatus fetches current order status from Betfair and updates database
func (om *OrderManager) syncOrderStatus(ctx context.Context) error {
	// Get pending bets from database
	pendingBets, err := om.betRepository.GetPendingBets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending bets: %w", err)
	}

	if len(pendingBets) == 0 {
		om.logger.Printf("No pending bets to sync")
		return nil
	}

	om.logger.Printf("Syncing status for %d pending bets", len(pendingBets))

	// Group bets by market
	betsByMarket := make(map[string][]*models.Bet)
	for _, bet := range pendingBets {
		betsByMarket[bet.MarketID] = append(betsByMarket[bet.MarketID], bet)
	}

	// Fetch orders for each market
	marketIDs := make([]string, 0, len(betsByMarket))
	for marketID := range betsByMarket {
		marketIDs = append(marketIDs, marketID)
	}

	currentOrders, err := om.bettingService.ListCurrentOrders(ctx, marketIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch current orders: %w", err)
	}

	// Build map of orders by bet ID for faster lookup
	orderByBetID := make(map[string]*CurrentOrderResponse)
	for i := range currentOrders {
		orderByBetID[currentOrders[i].BetID] = &currentOrders[i]
	}

	// Update bet status based on current orders
	om.mu.Lock()
	defer om.mu.Unlock()

	for _, bet := range pendingBets {
		order, found := orderByBetID[bet.BetID]
		if !found {
			// Bet not found on Betfair - it may have been cancelled or settled
			om.handleMissingBet(ctx, bet)
			continue
		}

		switch order.Status {
		case "MATCHED":
			om.handleMatchedBet(ctx, bet, order)
		case "UNMATCHED":
			// Still unmatched, no action needed
		case "CANCELLED":
			om.handleCancelledBet(ctx, bet)
		}
	}

	om.metrics.OrdersMonitored += int64(len(pendingBets))
	return nil
}

// handleMatchedBet updates bet status to matched
func (om *OrderManager) handleMatchedBet(ctx context.Context, bet *models.Bet, order *CurrentOrderResponse) {
	bet.Status = models.BetStatusMatched
	bet.MatchedTime = time.Now()
	bet.MatchedPrice = order.AveragePriceMatched
	bet.MatchedSize = order.SizeMatched

	if err := om.bettingService.UpdateBetStatus(ctx, bet); err != nil {
		om.logger.Printf("Failed to update bet %s to matched: %v", bet.BetID, err)
	} else {
		om.logger.Printf("Bet %s matched at %.2f", bet.BetID, order.AveragePriceMatched)
		om.metrics.OrdersMatched++
	}
}

// handleSettledBet updates bet status to settled with profit/loss calculation
func (om *OrderManager) handleSettledBet(ctx context.Context, bet *models.Bet, result *BetResult) {
	bet.Status = models.BetStatusSettled
	bet.SettledTime = time.Now()

	// Calculate profit/loss
	if bet.Side == models.BetSideBack {
		if result.Won {
			bet.ProfitLoss = bet.Stake * (bet.MatchedPrice - 1)
		} else {
			bet.ProfitLoss = -bet.Stake
		}
	} else { // LAY
		if result.Won {
			bet.ProfitLoss = -bet.Stake * (bet.MatchedPrice - 1)
		} else {
			bet.ProfitLoss = bet.Stake
		}
	}

	// Deduct commission
	if bet.ProfitLoss > 0 {
		bet.ProfitLoss = bet.ProfitLoss * (1 - om.bettingService.config.CommissionRate)
	}

	if err := om.bettingService.UpdateBetStatus(ctx, bet); err != nil {
		om.logger.Printf("Failed to update bet %s to settled: %v", bet.BetID, err)
	} else {
		om.logger.Printf("Bet %s settled with P&L: %.2f", bet.BetID, bet.ProfitLoss)
		om.metrics.OrdersSettled++
	}
}

// handleCancelledBet updates bet status to cancelled
func (om *OrderManager) handleCancelledBet(ctx context.Context, bet *models.Bet) {
	bet.Status = models.BetStatusCancelled
	bet.CancelledTime = time.Now()

	if err := om.bettingService.UpdateBetStatus(ctx, bet); err != nil {
		om.logger.Printf("Failed to update bet %s to cancelled: %v", bet.BetID, err)
	} else {
		om.logger.Printf("Bet %s cancelled", bet.BetID)
		om.metrics.OrdersCancelled++
	}
}

// handleMissingBet marks bet as unknown if not found on Betfair
func (om *OrderManager) handleMissingBet(ctx context.Context, bet *models.Bet) {
	om.logger.Printf("Bet %s not found on Betfair, marking as unknown", bet.BetID)
	// This could indicate settlement or an error - log for investigation
}

// Stop gracefully stops order monitoring
func (om *OrderManager) Stop() error {
	om.logger.Printf("Stopping order manager")
	close(om.done)
	return nil
}

// GetMetrics returns current order manager metrics
func (om *OrderManager) GetMetrics() OrderMetrics {
	om.mu.Lock()
	defer om.mu.Unlock()
	return *om.metrics
}

// ResetMetrics resets order manager metrics
func (om *OrderManager) ResetMetrics() {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.metrics = &OrderMetrics{}
}

// BetResult represents settlement result of a bet
type BetResult struct {
	BetID   string
	Won     bool
	Result  string
	Settled bool
}
