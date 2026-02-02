package betfair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/clever-better/internal/datasource"
)

// StreamClient handles WebSocket connection to Betfair Stream API
type StreamClient struct {
	conn            *websocket.Conn
	sessionToken    string
	appKey          string
	baseURL         string
	mu              sync.RWMutex
	isConnected     bool
	handlers        []MessageHandler
	reconnectConfig ReconnectConfig
	lastMessageTime time.Time
	logger          *log.Logger
}

// ReconnectConfig controls reconnection behavior
type ReconnectConfig struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiplier float64
}

// MessageHandler is called when a message is received from the stream
type MessageHandler func(msg interface{}) error

// StreamMessage represents a message from Betfair Stream API
type StreamMessage struct {
	Op            string        `json:"op"`
	ID            int           `json:"id,omitempty"`
	Status        int           `json:"status,omitempty"`
	ConnectionID  string        `json:"connectionId,omitempty"`
	ConnectionClosed bool        `json:"connectionClosed,omitempty"`
	MarketChanges []MarketChange `json:"mc,omitempty"`
}

// MarketChange represents market change in stream message
type MarketChange struct {
	MarketID    string         `json:"id"`
	FullImage   bool           `json:"img"`
	Runners     []RunnerChange `json:"rc"`
	TotalMatched float64       `json:"tm"`
	Conflated   bool           `json:"con"`
	Heartbeat   bool           `json:"heartbeat"`
}

// RunnerChange represents runner change in stream message
type RunnerChange struct {
	SelectionID uint64       `json:"id"`
	PriceChanges []PriceChange `json:"ltp"`
	BackPrices  []PriceChange `json:"b,omitempty"`
	LayPrices   []PriceChange `json:"l,omitempty"`
	TradeVolume []PriceChange `json:"tv,omitempty"`
}

// PriceChange represents a price level change
type PriceChange struct {
	Price float64 `json:"p"`
	Size  float64 `json:"s"`
}

// SubscriptionMessage for subscribing to markets
type SubscriptionMessage struct {
	Op                string        `json:"op"`
	ID                int           `json:"id"`
	AuthToken         string        `json:"authToken"`
	AppKey            string        `json:"appKey"`
	Clk               string        `json:"clk,omitempty"`
	MarketIDs         []string      `json:"marketIds,omitempty"`
	ConflateMs        int           `json:"conflateMs,omitempty"`
	PriceProjection   []string      `json:"priceProjection,omitempty"`
	Heartbeat         bool          `json:"heartbeat,omitempty"`
	InitialClk        string        `json:"initialClk,omitempty"`
	From              int64         `json:"from,omitempty"`
	To                int64         `json:"to,omitempty"`
}

// DefaultReconnectConfig returns default reconnection configuration
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		MaxRetries:        10,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 1.5,
	}
}

// NewStreamClient creates a new stream client
func NewStreamClient(
	sessionToken string,
	appKey string,
	streamURL string,
	logger *log.Logger,
) *StreamClient {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	return &StreamClient{
		sessionToken:    sessionToken,
		appKey:          appKey,
		baseURL:         streamURL,
		handlers:        make([]MessageHandler, 0),
		reconnectConfig: DefaultReconnectConfig(),
		logger:          logger,
	}
}

// Connect establishes connection to Betfair Stream API
func (s *StreamClient) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isConnected {
		return fmt.Errorf("already connected")
	}

	// Use wss:// protocol for secure WebSocket
	wsURL := fmt.Sprintf("wss://%s/stream", s.baseURL)

	s.logger.Printf("Connecting to stream: %s", wsURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to stream: %w", err)
	}

	s.conn = conn
	s.isConnected = true
	s.lastMessageTime = time.Now()

	s.logger.Printf("Connected to stream successfully")

	// Start message reading loop
	go s.readMessages()

	return nil
}

// Authenticate sends authentication message
func (s *StreamClient) Authenticate(ctx context.Context) error {
	s.mu.RLock()
	if !s.isConnected || s.conn == nil {
		s.mu.RUnlock()
		return fmt.Errorf("not connected to stream")
	}
	s.mu.RUnlock()

	authMsg := map[string]interface{}{
		"op":        "connection",
		"authToken": s.sessionToken,
		"appKey":    s.appKey,
	}

	return s.sendMessage(authMsg)
}

// SubscribeToMarkets subscribes to market data for specified market IDs
func (s *StreamClient) SubscribeToMarkets(
	ctx context.Context,
	marketIDs []string,
) error {
	s.mu.RLock()
	if !s.isConnected || s.conn == nil {
		s.mu.RUnlock()
		return fmt.Errorf("not connected to stream")
	}
	s.mu.RUnlock()

	subMsg := map[string]interface{}{
		"op":               "mcm",
		"authToken":        s.sessionToken,
		"appKey":           s.appKey,
		"marketIds":        marketIDs,
		"conflateMs":       1000,
		"priceProjection":  []string{"EX_BEST_OFFERS", "EX_TRADED"},
		"heartbeat":        true,
	}

	s.logger.Printf("Subscribing to %d markets", len(marketIDs))
	return s.sendMessage(subMsg)
}

// AddHandler registers a message handler
func (s *StreamClient) AddHandler(handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// readMessages reads messages from WebSocket connection
func (s *StreamClient) readMessages() {
	defer s.Close()

	for {
		var msg json.RawMessage
		err := s.conn.ReadJSON(&msg)
		if err != nil {
			s.logger.Printf("Error reading message: %v", err)
			s.mu.Lock()
			s.isConnected = false
			s.mu.Unlock()
			return
		}

		s.mu.Lock()
		s.lastMessageTime = time.Now()
		s.mu.Unlock()

		// Call registered handlers
		s.mu.RLock()
		handlers := s.handlers
		s.mu.RUnlock()

		for _, handler := range handlers {
			if err := handler(msg); err != nil {
				s.logger.Printf("Handler error: %v", err)
			}
		}
	}
}

// sendMessage sends a JSON message to the stream
func (s *StreamClient) sendMessage(msg interface{}) error {
	s.mu.RLock()
	if !s.isConnected || s.conn == nil {
		s.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	conn := s.conn
	s.mu.RUnlock()

	return conn.WriteJSON(msg)
}

// IsConnected returns whether the stream is connected
func (s *StreamClient) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isConnected
}

// LastMessageTime returns the time of the last received message
func (s *StreamClient) LastMessageTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastMessageTime
}

// Close closes the stream connection
func (s *StreamClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil
	}

	s.isConnected = false
	return s.conn.Close()
}

// Ping sends a ping message to keep connection alive
func (s *StreamClient) Ping() error {
	return s.sendMessage(map[string]interface{}{
		"op": "ping",
	})
}
