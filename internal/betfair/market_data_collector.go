package betfair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
)

// MarketDataCollector collects streaming market data and stores it
type MarketDataCollector struct {
	streamClient    *StreamClient
	oddsRepository  repository.OddsRepository
	buffer          []*models.OddsSnapshot
	bufferSize      int
	flushInterval   time.Duration
	mu              sync.Mutex
	done            chan struct{}
	metrics         *CollectorMetrics
	logger          *log.Logger
}

// CollectorMetrics tracks collector performance
type CollectorMetrics struct {
	MessagesProcessed int64
	BufferFlushes     int64
	SnapshotsStored   int64
	Errors            int64
	LastFlushTime     time.Time
	BufferSize        int
}

// NewMarketDataCollector creates a new market data collector
func NewMarketDataCollector(
	streamClient *StreamClient,
	oddsRepository repository.OddsRepository,
	bufferSize int,
	flushInterval time.Duration,
	logger *log.Logger,
) *MarketDataCollector {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	if bufferSize <= 0 {
		bufferSize = 1000
	}

	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	collector := &MarketDataCollector{
		streamClient:   streamClient,
		oddsRepository: oddsRepository,
		buffer:         make([]*models.OddsSnapshot, 0, bufferSize),
		bufferSize:     bufferSize,
		flushInterval:  flushInterval,
		done:           make(chan struct{}),
		metrics:        &CollectorMetrics{},
		logger:         logger,
	}

	return collector
}

// Start begins collecting market data
func (c *MarketDataCollector) Start(ctx context.Context, marketIDs []string) error {
	if len(marketIDs) == 0 {
		return fmt.Errorf("at least one market ID required")
	}

	c.logger.Printf("Starting market data collector for %d markets", len(marketIDs))

	// Register message handler
	c.streamClient.AddHandler(c.onMessage)

	// Subscribe to markets
	if err := c.streamClient.SubscribeToMarkets(ctx, marketIDs); err != nil {
		return fmt.Errorf("failed to subscribe to markets: %w", err)
	}

	// Start periodic flush
	go c.flushLoop()

	c.logger.Printf("Market data collector started")
	return nil
}

// onMessage processes incoming stream messages
func (c *MarketDataCollector) onMessage(msg interface{}) error {
	c.metrics.MessagesProcessed++

	data, ok := msg.(json.RawMessage)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	var streamMsg StreamMessage
	if err := json.Unmarshal(data, &streamMsg); err != nil {
		c.logger.Printf("Failed to unmarshal stream message: %v", err)
		c.metrics.Errors++
		return err
	}

	// Handle connection messages
	if streamMsg.Op == "connection" {
		if streamMsg.Status == 0 {
			c.logger.Printf("Stream connection: %s", streamMsg.ConnectionID)
		}
		return nil
	}

	// Handle status messages
	if streamMsg.Op == "status" {
		c.logger.Printf("Stream status: %d", streamMsg.Status)
		return nil
	}

	// Process market changes
	if streamMsg.Op == "mcm" {
		return c.processMarketChanges(streamMsg.MarketChanges)
	}

	return nil
}

// extractMarketChangePrices extracts prices from a runner change and updates the snapshot
func (c *MarketDataCollector) extractMarketChangePrices(snapshot *models.OddsSnapshot, runner *Runner) {
	// Extract back prices
	if len(runner.BackPrices) > 0 {
		snapshot.BackPrice = runner.BackPrices[0].Price
		snapshot.BackSize = runner.BackPrices[0].Size
	}

	// Extract lay prices
	if len(runner.LayPrices) > 0 {
		snapshot.LayPrice = runner.LayPrices[0].Price
		snapshot.LaySize = runner.LayPrices[0].Size
	}

	// Extract traded volume
	if len(runner.TradeVolume) > 0 {
		snapshot.TradedVolume = runner.TradeVolume[0].Size
	}
}

// processMarketChanges converts market change messages to odds snapshots
func (c *MarketDataCollector) processMarketChanges(changes []MarketChange) error {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, change := range changes {
		for _, runner := range change.Runners {
			snapshot := &models.OddsSnapshot{
				MarketID:    change.MarketID,
				SelectionID: runner.SelectionID,
				Timestamp:   now,
			}

			// Extract prices
			c.extractMarketChangePrices(snapshot, &runner)

			c.buffer = append(c.buffer, snapshot)

			// Flush if buffer is full
			if len(c.buffer) >= c.bufferSize {
				if err := c.flushBuffer(); err != nil {
					c.logger.Printf("Error flushing buffer: %v", err)
					c.metrics.Errors++
				}
			}
		}
	}

	c.metrics.BufferSize = len(c.buffer)
	return nil
}

// flushBuffer writes buffered snapshots to database
func (c *MarketDataCollector) flushBuffer() error {
	if len(c.buffer) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c.logger.Printf("Flushing buffer: %d snapshots", len(c.buffer))

	if err := c.oddsRepository.InsertBatch(ctx, c.buffer); err != nil {
		c.logger.Printf("Failed to insert batch: %v", err)
		c.metrics.Errors++
		return err
	}

	count := len(c.buffer)
	c.buffer = make([]*models.OddsSnapshot, 0, c.bufferSize)
	c.metrics.SnapshotsStored += int64(count)
	c.metrics.BufferFlushes++
	c.metrics.LastFlushTime = time.Now()

	c.logger.Printf("Buffer flushed: %d snapshots stored", count)
	return nil
}

// flushLoop periodically flushes the buffer
func (c *MarketDataCollector) flushLoop() {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			if err := c.flushBuffer(); err != nil {
				c.logger.Printf("Periodic flush failed: %v", err)
			}
			c.mu.Unlock()

		case <-c.done:
			c.logger.Printf("Flush loop stopped")
			return
		}
	}
}

// Stop gracefully shuts down the collector
func (c *MarketDataCollector) Stop() error {
	c.logger.Printf("Stopping market data collector")

	close(c.done)

	// Final flush
	c.mu.Lock()
	if len(c.buffer) > 0 {
		c.logger.Printf("Performing final flush: %d snapshots", len(c.buffer))
		if err := c.flushBuffer(); err != nil {
			c.logger.Printf("Final flush failed: %v", err)
		}
	}
	c.mu.Unlock()

	// Close stream
	if err := c.streamClient.Close(); err != nil {
		c.logger.Printf("Error closing stream: %v", err)
	}

	c.logger.Printf("Market data collector stopped")
	return nil
}

// GetMetrics returns current collector metrics
func (c *MarketDataCollector) GetMetrics() CollectorMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return *c.metrics
}

// ResetMetrics resets collector metrics
func (c *MarketDataCollector) ResetMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = &CollectorMetrics{}
}
