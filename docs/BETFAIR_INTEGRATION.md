# Betfair Integration Guide

## Overview

This guide covers the integration of Betfair's API-NG platform with the Clever Better trading system. The integration enables real-time market data collection, automated bet placement, and order lifecycle management for greyhound racing markets.

## Betfair API Architecture

### Components

1. **REST API (JSON-RPC 2.0)**: Used for market catalog queries, bet placement, and order management
2. **Streaming API (WebSocket)**: Provides real-time market price updates with minimal latency
3. **Certificate Authentication**: Secure non-interactive login using client certificates

### Authentication Flow

```
1. Load Client Certificate (PEM format)
2. Send TLS-encrypted login request to identitysso-cert.betfair.com
3. Receive session token (valid for ~12 hours)
4. Include token in X-Authentication header for all API requests
5. Monitor token expiry and refresh before expiration
```

## Prerequisites

### Certificates

Betfair requires client certificates for non-interactive authentication:

1. **Obtain Certificates**:
   - Log in to your Betfair account
   - Navigate to My Account > Security > API Settings
   - Request a certificate package
   - Betfair will provide: certificate.crt and certificate.key

2. **Certificate Storage**:
   - Place in a secure location (e.g., `/etc/betfair/` or `/var/secrets/betfair/`)
   - Restrict file permissions: `chmod 600 certificate.*`
   - Never commit to version control

3. **Certificate Format**:
   - Certificates must be in PEM format
   - Both client certificate and private key are required
   - Concatenate them into a single file if needed:
     ```bash
     cat certificate.crt certificate.key > client-cert.pem
     ```

### Betfair Account Requirements

- Active Betfair account with API access enabled
- Funds available for betting (if placing bets)
- Correct API Application Key assigned in account settings

## Configuration

### Environment Setup

Add the following environment variables to your `.env` file or deployment configuration:

```bash
# Betfair API Settings
BETFAIR_CERT_PATH=/etc/betfair/client-cert.pem
BETFAIR_APP_KEY=your-application-key-here
BETFAIR_USERNAME=your-betfair-username

# Optional: Custom Betfair endpoints (defaults provided)
BETFAIR_API_URL=https://api.betfair.com/exchange/betting/json-rpc/v1/
BETFAIR_STREAM_URL=stream-api.betfair.com:443
BETFAIR_IDENTITY_URL=https://identitysso-cert.betfair.com/api/
```

### YAML Configuration

Update your `config.yaml` with:

```yaml
betfair:
  # Certificate authentication
  certificate:
    certPath: /etc/betfair/client-cert.pem
    # Certificate is automatically reloaded on login attempts
  
  # API Configuration
  api:
    appKey: "${BETFAIR_APP_KEY}"
    username: "${BETFAIR_USERNAME}"
    baseUrl: "https://api.betfair.com/exchange/betting/json-rpc/v1/"
    requestTimeout: "30s"
    
    # Rate limiting (requests per second)
    rateLimit: 50
    
    # Retry configuration
    maxRetries: 3
    retryBackoff: "100ms"
  
  # Streaming Configuration
  stream:
    url: "stream-api.betfair.com:443"
    heartbeatMs: 5000
    conflateMs: 1000  # Conflate updates to 1 second
    
    # Reconnection settings
    maxReconnectAttempts: 10
    initialReconnectDelay: "1s"
    maxReconnectDelay: "30s"
  
  # Market Data Collection
  marketData:
    # Buffer settings for batch inserts
    bufferSize: 1000
    flushIntervalSeconds: 5
    
    # Market collection settings
    eventTypeId: 4339  # Greyhound racing
    marketTypes:
      - WIN
      - PLACE
    
    # Time range for market queries
    inPlayDelay: "2m"
    marketWindow: "7d"
  
  # Betting Configuration
  betting:
    # Stake limits
    minStake: 0.50
    maxStake: 500.00
    defaultOrderType: "LAPSE"  # LAPSE, PERSIST, or EXECUTE_LAPSE
    
    # Commission (percentage)
    commission: 5.0
    
    # Order monitoring
    pollIntervalSeconds: 30
    maxOrderAge: "24h"
  
  # Error Handling
  errorHandling:
    # Log level for API errors
    logLevel: "INFO"
    
    # Maximum errors before circuit breaker opens
    circuitBreakerThreshold: 10
    circuitBreakerTimeout: "30s"
```

## Development Setup

### Local Testing

1. **Configure test environment**:
   ```bash
   export BETFAIR_CERT_PATH=/path/to/test/certificate.pem
   export BETFAIR_APP_KEY=your-test-app-key
   export BETFAIR_USERNAME=your-test-username
   ```

2. **Run unit tests**:
   ```bash
   go test ./internal/betfair/... -v
   ```

3. **Run integration tests**:
   ```bash
   go test ./test/integration/... -v -tags=integration
   ```

### Production Deployment

1. **Secure Certificate Management**:
   - Use environment-specific secret management (Kubernetes Secrets, AWS Secrets Manager, etc.)
   - Never store certificates in code or configuration files
   - Rotate certificates regularly

2. **Rate Limiting**:
   - Respect Betfair's rate limits (typically 500 requests/second for accounts with sufficient turnover)
   - Use the built-in rate limiter: `NewRateLimitedHTTPClient()`
   - Monitor `Metrics.APIRequestLatency` for performance

3. **Monitoring**:
   - Monitor connection health: `Metrics.StreamConnections` and `StreamReconnections`
   - Track error rates: `Metrics.APIRequestsFailure`
   - Monitor market data flow: `Metrics.OddsSnapshotsStored`

## API Usage Examples

### Market Data Collection

```go
// Initialize components
config := loadBetfairConfig()
httpClient := NewRateLimitedHTTPClient(config.RateLimit)
betfairClient := NewBetfairClient(config, httpClient)
authService := NewAuthService(betfairClient, config)

// Authenticate
session, err := authService.Login(config.Certificate.CertPath)
if err != nil {
    log.Fatal("Failed to login:", err)
}

// Fetch greyhound race markets
markets, err := betfairClient.ListGreyhoundRaceMarkets(ctx, &MarketFilter{
    EventTypeIDs: []string{"4339"},
    MarketTypes:  []string{"WIN"},
})

// Collect streaming data
collector := NewMarketDataCollector(betfairClient, oddsRepository)
err = collector.Start(ctx, []string{marketID})
if err != nil {
    log.Fatal("Failed to start collection:", err)
}

// Data flows to database automatically via buffering
```

### Bet Placement

```go
// Create betting service
bettingService := NewBettingService(betfairClient, betRepository)

// Place a bet
bet, err := bettingService.PlaceBet(ctx, &BetRequest{
    MarketID:      "1.123456789",
    SelectionID:   987654,
    Side:          "BACK",
    Stake:         10.00,
    Price:         2.50,
})
if err != nil {
    if err.Error() == "insufficient_funds" {
        log.Println("Account has insufficient funds")
    }
    return err
}

// Monitor order status automatically
orderManager := NewOrderManager(betfairClient, betRepository)
go orderManager.MonitorOrders(ctx)
```

### Historical Data Storage

```go
// Store market data for backtesting
dataService := NewMarketDataService(
    betfairClient,
    raceRepository,
    runnerRepository,
    oddsRepository,
)

// Backfill data for date range
err := dataService.BackfillMarketData(ctx, 
    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    time.Now(),
)
if err != nil {
    log.Fatal("Failed to backfill data:", err)
}
```

## Troubleshooting

### Common Issues

#### 1. Certificate Authentication Failures

**Error**: `CERTIFICATE_AUTH_FAILED` or `INVALID_CERTIFICATE`

**Solutions**:
- Verify certificate file exists and is readable: `ls -la /etc/betfair/`
- Confirm certificate is in PEM format (not DER)
- Check certificate expiry: `openssl x509 -in cert.pem -text -noout | grep -A2 Validity`
- Ensure private key is included in the certificate file
- Verify certificate is uploaded to Betfair account settings

#### 2. Session Token Expiration

**Error**: `INVALID_SESSION_INFORMATION` or `SESSION_TOKEN_INVALID`

**Solutions**:
- Check that `IsAuthenticated()` returns true
- Verify `NeedsRefresh()` is being checked periodically
- Call `AuthService.RefreshSession()` before token expires (typically ~12 hours)
- Log shows token refresh events for debugging

#### 3. Market Data Stream Disconnections

**Error**: WebSocket closes unexpectedly

**Solutions**:
- Check network connectivity and firewall rules
- Verify heartbeat messages are being received (logs show heartbeat timestamps)
- Inspect `Metrics.StreamReconnections` for reconnection attempts
- Enable debug logging to see stream message details
- Check Betfair service status page for platform issues

#### 4. Insufficient Funds Error

**Error**: `INSUFFICIENT_FUNDS` when placing bets

**Solutions**:
- Check account balance in Betfair dashboard
- Verify stake amount doesn't exceed available funds
- Consider account liability (potential loss on lay bets)
- Use `PlaceBet()` which performs validation before submission

#### 5. Rate Limit Exceeded

**Error**: `TOO_MANY_REQUESTS` or `ORDER_LIMIT_EXCEEDED`

**Solutions**:
- Check `Metrics.APIRequestLatency` to identify slow requests
- Verify `RateLimitedHTTPClient` is configured with appropriate limit
- Reduce concurrent market subscriptions
- Check for burst traffic from multiple components
- Contact Betfair support if limit is insufficient for your use case

### Debug Logging

Enable debug logging for detailed troubleshooting:

```go
import "log"

// Enable verbose logging
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Check metrics periodically
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        metrics := GetMetrics()
        log.Printf("API: %d requests, %d failures\n", 
            metrics.APIRequestsTotal, 
            metrics.APIRequestsFailure)
        log.Printf("Stream: %d messages, %d errors\n",
            metrics.MessagesReceived,
            metrics.MessageProcessErrors)
        log.Printf("Betting: %d placed, %d matched, %d settled\n",
            metrics.BetsPlaced,
            metrics.BetsMatched,
            metrics.BetsSettled)
    }
}()
```

## Performance Considerations

### Market Data Collection

- **Buffer Size**: Default 1000 snapshots per flush
  - Increase for high-volume markets to reduce database load
  - Decrease for low-latency requirements
  
- **Flush Interval**: Default 5 seconds
  - Shorter intervals improve data freshness
  - Longer intervals reduce database writes

- **Message Conflation**: Set to 1000ms (1 second)
  - Reduces message volume without losing accuracy
  - Adjust based on strategy requirements

### Bet Placement

- **Rate Limit**: Default 50 requests/second
  - Adjust based on Betfair API allowance
  - Higher limits may require account requirements with Betfair

- **Order Polling**: Default 30 second interval
  - Shorter intervals detect fills faster
  - Longer intervals reduce API overhead

## Integration with Backtesting

Historical market data collected via `MarketDataService` is stored in TimescaleDB with the following structure:

```sql
-- Race information
CREATE TABLE races (
    id BIGSERIAL PRIMARY KEY,
    track VARCHAR(255) NOT NULL,
    race_date DATE NOT NULL,
    race_time TIME NOT NULL,
    betfair_market_id VARCHAR(255) UNIQUE,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Runner information
CREATE TABLE runners (
    id BIGSERIAL PRIMARY KEY,
    race_id BIGINT REFERENCES races(id),
    trap_number INT,
    betfair_selection_id BIGINT,
    runner_name VARCHAR(255),
    odds DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Odds snapshots (hypertable)
CREATE TABLE odds_snapshots (
    id BIGSERIAL,
    runner_id BIGINT REFERENCES runners(id),
    timestamp TIMESTAMP NOT NULL,
    back_price DECIMAL(10, 2),
    lay_price DECIMAL(10, 2),
    back_size DECIMAL(15, 2),
    lay_size DECIMAL(15, 2),
    PRIMARY KEY (id, timestamp)
);

SELECT create_hypertable('odds_snapshots', 'timestamp');
```

Query historical data for backtesting:

```go
// Get odds data for a specific race
odds, err := oddsRepository.GetByRunner(ctx, runnerID, startTime, endTime)

// Get market-wide data
odds, err := oddsRepository.GetByMarket(ctx, marketID, startTime, endTime)

// Aggregate into OHLC candles
ohlc, err := oddsRepository.GetOHLC(ctx, runnerID, 
    startTime, endTime, 5*time.Minute)
```

## Security Best Practices

1. **Certificate Management**:
   - Store certificates outside the application codebase
   - Use environment-specific secret management
   - Rotate certificates annually or upon expiration

2. **API Keys**:
   - Never log API keys or session tokens
   - Use environment variables, not hardcoded values
   - Restrict scope of application keys (betting-only vs. full access)

3. **Network Security**:
   - Enforce TLS 1.2+ for all connections
   - Use VPN or IP whitelisting for production systems
   - Monitor connection logs for unauthorized access

4. **Account Security**:
   - Enable two-factor authentication on Betfair account
   - Use separate accounts for testing and production
   - Audit API access regularly through Betfair dashboard

## Support and Resources

- **Betfair API Documentation**: https://developer.betfair.com/
- **Betfair Forum**: https://forum.betfair.com/
- **API Status Page**: https://status.betfair.com/
- **Support Email**: support@betfair.com

## Version History

- **v1.0** (2024-01): Initial Betfair integration with REST and Stream API support
