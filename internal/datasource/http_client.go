package datasource

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

// HTTPClientConfig holds configuration for HTTP clients
type HTTPClientConfig struct {
	Timeout           time.Duration
	MaxRetries        int
	RetryWaitMin      time.Duration
	RetryWaitMax      time.Duration
	RateLimit         float64 // requests per second
	CircuitBreakerMax int     // max consecutive failures before circuit break
}

// DefaultHTTPClientConfig returns recommended defaults
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:           30 * time.Second,
		MaxRetries:        5,
		RetryWaitMin:      100 * time.Millisecond,
		RetryWaitMax:      10 * time.Second,
		RateLimit:         10.0, // 10 requests per second by default
		CircuitBreakerMax: 5,
	}
}

// RateLimitedHTTPClient wraps retryablehttp.Client with rate limiting and circuit breaker
type RateLimitedHTTPClient struct {
	client            *retryablehttp.Client
	limiter           *rate.Limiter
	circuitBreakerMax int
	consecutiveErrors int
	isOpen            bool
	lastError         error
	logger            *log.Logger
}

// NewRateLimitedHTTPClient creates a new rate-limited HTTP client
func NewRateLimitedHTTPClient(cfg HTTPClientConfig, logger *log.Logger) *RateLimitedHTTPClient {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Timeout = cfg.Timeout
	retryClient.RetryMax = cfg.MaxRetries
	retryClient.RetryWaitMin = cfg.RetryWaitMin
	retryClient.RetryWaitMax = cfg.RetryWaitMax
	retryClient.CheckRetry = customRetryPolicy()

	// Don't log verbose retry info
	retryClient.Logger = logger

	return &RateLimitedHTTPClient{
		client:            retryClient,
		limiter:           rate.NewLimiter(rate.Limit(cfg.RateLimit), 1),
		circuitBreakerMax: cfg.CircuitBreakerMax,
		logger:            logger,
	}
}

// Do executes an HTTP request with rate limiting and circuit breaker
func (c *RateLimitedHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Check circuit breaker status
	if c.isOpen {
		return nil, fmt.Errorf("circuit breaker open: %v", c.lastError)
	}

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Execute request
	resp, err := c.client.DoWithContext(ctx, req)

	// Update circuit breaker state
	if err != nil {
		c.consecutiveErrors++
		c.lastError = err
		if c.consecutiveErrors >= c.circuitBreakerMax {
			c.isOpen = true
			c.logger.Printf("Circuit breaker opened after %d consecutive errors: %v", c.consecutiveErrors, err)
		}
		return nil, err
	}

	// Reset circuit breaker on success
	if resp.StatusCode < 500 {
		c.consecutiveErrors = 0
		c.isOpen = false
	}

	return resp, nil
}

// Get executes a GET request
func (c *RateLimitedHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post executes a POST request
func (c *RateLimitedHTTPClient) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// Close closes any resources held by the client
func (c *RateLimitedHTTPClient) Close() error {
	c.client.HTTPClient.CloseIdleConnections()
	return nil
}

// customRetryPolicy defines which HTTP responses should trigger a retry
func customRetryPolicy() retryablehttp.CheckRetry {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			// Retry on network errors
			return true, err
		}

		// Retry on rate limit (429), server errors (500, 502, 503, 504), and gateway errors
		if resp.StatusCode == 429 || resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			return true, nil
		}

		// Don't retry on client errors (4xx) except 429
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return false, nil
		}

		return false, nil
	}
}
