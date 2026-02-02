package betfair

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
)

// BetfairClient implements Betfair API-NG REST client
type BetfairClient struct {
	httpClient    *datasource.RateLimitedHTTPClient
	config        *config.BetfairConfig
	baseURL       string
	streamURL     string
	sessionToken  string
	appKey        string
	tokenExpiry   time.Time
	mu            sync.RWMutex
	logger        *log.Logger
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      int               `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Error codes for Betfair API
const (
	ErrorInvalidSessionInformation = "INVALID_SESSION_INFORMATION"
	ErrorInsufficientFunds         = "INSUFFICIENT_FUNDS"
	ErrorMarketSuspended           = "MARKET_SUSPENDED"
	ErrorOrderLimitExceeded        = "ORDER_LIMIT_EXCEEDED"
	ErrorPersistenceQuotaExceeded  = "PERSISTENCE_QUOTA_EXCEEDED"
	ErrorInvalidBetSize            = "INVALID_BET_SIZE"
	ErrorOperationNotAllowed       = "OPERATION_NOT_ALLOWED"
)

// NewBetfairClient creates a new Betfair API client
func NewBetfairClient(
	cfg *config.BetfairConfig,
	httpClient *datasource.RateLimitedHTTPClient,
	logger *log.Logger,
) *BetfairClient {
	if logger == nil {
		logger = log.New(nil, "", 0)
	}

	return &BetfairClient{
		httpClient: httpClient,
		config:     cfg,
		baseURL:    cfg.APIURL,
		streamURL:  cfg.StreamURL,
		appKey:     cfg.AppKey,
		logger:     logger,
	}
}

// makeRequest performs a JSON-RPC request to Betfair API
func (c *BetfairClient) makeRequest(
	ctx context.Context,
	method string,
	params map[string]interface{},
) (json.RawMessage, error) {
	c.mu.RLock()
	sessionToken := c.sessionToken
	c.mu.RUnlock()

	if sessionToken == "" {
		return nil, NewAuthenticationError("no active session token", nil)
	}

	// Build JSON-RPC request
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Application", c.appKey)
	req.Header.Set("X-Authentication", sessionToken)

	c.logger.Printf("Making Betfair API request: %s", method)

	// Execute request
	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		c.logger.Printf("Failed to make request: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var jsonResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for JSON-RPC error
	if jsonResp.Error != nil {
		return nil, NewBetfairAPIError(jsonResp.Error.Message, jsonResp.Error.Data, nil)
	}

	// Check for HTTP error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Printf("API request successful: %s", method)
	return jsonResp.Result, nil
}

// SetSessionToken sets the session token for API requests
func (c *BetfairClient) SetSessionToken(token string, expiry time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionToken = token
	c.tokenExpiry = expiry
	c.logger.Printf("Session token updated, expiry: %v", expiry)
}

// GetSessionToken returns the current session token
func (c *BetfairClient) GetSessionToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionToken
}

// IsAuthenticated checks if the client has an active session
func (c *BetfairClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionToken != "" && time.Now().Before(c.tokenExpiry)
}

// NeedsRefresh checks if the session token needs refreshing
func (c *BetfairClient) NeedsRefresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Refresh if expiry is within 5 minutes
	return time.Now().Add(5 * time.Minute).After(c.tokenExpiry)
}

// GetStreamURL returns the stream API URL
func (c *BetfairClient) GetStreamURL() string {
	return c.streamURL
}

// GetAppKey returns the app key
func (c *BetfairClient) GetAppKey() string {
	return c.appKey
}

// GetConfig returns the Betfair configuration
func (c *BetfairClient) GetConfig() *config.BetfairConfig {
	return c.config
}
