//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/datasource"
)

const (
	skipIntegration = "Skipping integration test in short mode"
	sampleMarketID  = "1.198765432"
)

type betfairHandler func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse)

func setupBetfairClient(t *testing.T, handler betfairHandler) (*betfair.BetfairClient, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReq betfair.JSONRPCRequest
		err := json.NewDecoder(r.Body).Decode(&rpcReq)
		require.NoError(t, err)

		status, resp := handler(t, rpcReq)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	httpClient := datasource.NewRateLimitedHTTPClient(datasource.HTTPClientConfig{
		Timeout:           2 * time.Second,
		MaxRetries:        0,
		RetryWaitMin:      10 * time.Millisecond,
		RetryWaitMax:      20 * time.Millisecond,
		RateLimit:         100,
		CircuitBreakerMax: 10,
	}, nil)

	cfg := &config.BetfairConfig{
		APIURL:    server.URL,
		StreamURL: server.URL,
		AppKey:    "test-key",
		Username:  "test-user",
		Password:  "test-pass",
	}

	client := betfair.NewBetfairClient(cfg, httpClient, nil)
	client.SetSessionToken("test-session", time.Now().Add(1*time.Hour))

	return client, server
}

func jsonResult(t *testing.T, value interface{}) json.RawMessage {
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}

// TestMarketDataRetrieval tests market data retrieval using ListMarketCatalog
func TestMarketDataRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	client, server := setupBetfairClient(t, func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse) {
		require.Equal(t, "listMarketCatalogue", req.Method)

		result := []map[string]interface{}{
			{
				"marketId":   sampleMarketID,
				"marketName": "Test Race",
				"description": map[string]interface{}{
					"marketType":    "WIN",
					"scheduledTime": time.Now().Format(time.RFC3339),
				},
				"runners": []map[string]interface{}{
					{
						"selectionId": 12345678,
						"runnerName":  "Test Runner",
					},
				},
			},
		}

		return http.StatusOK, betfair.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  jsonResult(t, result),
		}
	})
	defer server.Close()

	ctx := context.Background()
	catalogs, err := client.ListMarketCatalog(ctx, "4339", []string{"WIN"}, []string{"Ascot"}, 1)
	require.NoError(t, err)
	require.Len(t, catalogs, 1)

	assert.Equal(t, sampleMarketID, catalogs[0].MarketID)
	require.Len(t, catalogs[0].Runners, 1)
	assert.Equal(t, uint64(12345678), catalogs[0].Runners[0].SelectionID)
}

// TestBetPlacement tests bet placement using BettingService.PlaceBet
func TestBetPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	client, server := setupBetfairClient(t, func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse) {
		require.Equal(t, "placeOrders", req.Method)

		result := betfair.PlaceOrdersResponse{
			MarketID: sampleMarketID,
			Status:   "SUCCESS",
			InstructionReports: []betfair.InstructionReport{
				{
					Status:              "SUCCESS",
					OrderStatus:         "EXECUTION_COMPLETE",
					BetID:               "123456789",
					AveragePriceMatched: 3.5,
					SizeMatched:         100.0,
				},
			},
		}

		return http.StatusOK, betfair.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  jsonResult(t, result),
		}
	})
	defer server.Close()

	bettingService := betfair.NewBettingService(client, nil, betfair.BettingConfig{
		MaxStake:       500.0,
		MinStake:       10.0,
		CommissionRate: 0.05,
	}, nil)

	ctx := context.Background()
	betID, err := bettingService.PlaceBet(ctx, sampleMarketID, 12345678, 3.5, 100.0, "BACK")
	require.NoError(t, err)
	assert.Equal(t, "123456789", betID)
}

// TestMarketBookRetrieval tests market book retrieval using ListMarketBook
func TestMarketBookRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	client, server := setupBetfairClient(t, func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse) {
		require.Equal(t, "listMarketBook", req.Method)

		result := []map[string]interface{}{
			{
				"marketId": sampleMarketID,
				"status":   "OPEN",
				"runners": []map[string]interface{}{
					{
						"selectionId":     12345678,
						"lastPriceTraded": 3.4,
						"ex": map[string]interface{}{
							"availableToBack": []map[string]interface{}{{"price": 3.4, "size": 1000}},
							"availableToLay":  []map[string]interface{}{{"price": 3.45, "size": 800}},
						},
					},
				},
			},
		}

		return http.StatusOK, betfair.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  jsonResult(t, result),
		}
	})
	defer server.Close()

	ctx := context.Background()
	books, err := client.ListMarketBook(ctx, []string{sampleMarketID}, []string{"EX_BEST_OFFERS"})
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Len(t, books[0].Runners, 1)
	assert.Equal(t, uint64(12345678), books[0].Runners[0].SelectionID)
}

// TestRateLimiting tests rate limiting behavior on JSON-RPC requests
func TestRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	requestCount := 0
	client, server := setupBetfairClient(t, func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse) {
		require.Equal(t, "listMarketCatalogue", req.Method)
		requestCount++

		if requestCount > 5 {
			return http.StatusTooManyRequests, betfair.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &betfair.JSONRPCError{
					Code:    429,
					Message: "RATE_LIMIT_EXCEEDED",
				},
			}
		}

		result := []map[string]interface{}{{"marketId": sampleMarketID}}
		return http.StatusOK, betfair.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  jsonResult(t, result),
		}
	})
	defer server.Close()

	ctx := context.Background()
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 10; i++ {
		_, err := client.ListMarketCatalog(ctx, "4339", []string{"WIN"}, nil, 1)
		if err != nil {
			rateLimitedCount++
		} else {
			successCount++
		}
	}

	assert.Greater(t, rateLimitedCount, 0, "Should encounter rate limiting")
	assert.Greater(t, successCount, 0, "Some requests should succeed")
}

// TestBetfairErrorHandling tests JSON-RPC error handling
func TestBetfairErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegration)
	}

	tests := []struct {
		name          string
		responseError *betfair.JSONRPCError
		expectError   bool
	}{
		{
			name:          "Successful response",
			responseError: nil,
			expectError:   false,
		},
		{
			name: "Insufficient funds",
			responseError: &betfair.JSONRPCError{
				Code:    400,
				Message: betfair.ErrorInsufficientFunds,
			},
			expectError: true,
		},
		{
			name: "Invalid session",
			responseError: &betfair.JSONRPCError{
				Code:    400,
				Message: betfair.ErrorInvalidSessionInformation,
			},
			expectError: true,
		},
		{
			name: "Server error",
			responseError: &betfair.JSONRPCError{
				Code:    500,
				Message: "INTERNAL_ERROR",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := setupBetfairClient(t, func(t *testing.T, req betfair.JSONRPCRequest) (int, betfair.JSONRPCResponse) {
				if tt.responseError != nil {
					return http.StatusBadRequest, betfair.JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      req.ID,
						Error:   tt.responseError,
					}
				}

				result := []map[string]interface{}{{"marketId": sampleMarketID}}
				return http.StatusOK, betfair.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  jsonResult(t, result),
				}
			})
			defer server.Close()

			_, err := client.ListMarketCatalog(context.Background(), "4339", []string{"WIN"}, nil, 1)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
