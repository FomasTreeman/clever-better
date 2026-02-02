# Data Sources Documentation

This document describes the data sources integrated into the Clever Better system for greyhound racing data ingestion.

## Overview

The data ingestion system supports multiple data sources through a pluggable architecture:

- **Betfair**: Live and historical exchange data with market odds and selections
- **Racing Post**: Race information, form guides, and expert analysis
- **CSV**: Local file-based data for backtesting and manual uploads

## Betfair

### Configuration

```yaml
data_sources:
  betfair:
    enabled: true
    base_url: https://api.betfair.com
    auth:
      api_key: ${BETFAIR_APP_KEY}
      session_token: ${BETFAIR_SESSION_TOKEN}
      username: ${BETFAIR_USERNAME}
      password: ${BETFAIR_PASSWORD}
      cert_file: ${BETFAIR_CERT_FILE}
      key_file: ${BETFAIR_KEY_FILE}
    rate_limit:
      requests_per_second: 30
      burst_size: 50
    data_path: ./data/betfair/
```

### Authentication

Betfair requires:
1. **App Key**: Application token for API access
2. **Session Token**: Obtained through login (auto-refreshed by client)
3. **Username/Password**: Credentials for session renewal
4. **SSL Certificates**: Client certificate authentication for secure connections

### Data Access Methods

#### Historical Data via CSV
Access historical race data through Betfair's data export portal:
1. Download CSV files from Betfair's research tools
2. Parse using the BetfairCSVParser
3. Normalize and validate before storage

**CSV Format Example:**
```
Event ID, Event Name, Race Type, Track, Start Time, Selection ID, Selection Name, Price, Odds, Volume
```

#### Live Data via HTTP API
Stream real-time market data:
- Market prices and liquidity
- Runner selections and odds changes
- Race status updates

### Rate Limiting

- **Requests per second**: 30
- **Burst capacity**: 50 requests
- **Reset period**: 1 second

Exceeding these limits results in HTTP 429 responses with exponential backoff retry.

### Data Validation

Betfair data validation includes:
- Event start time validity
- Price range verification (1.01 - 1000)
- Volume sanity checks
- Selection name consistency
- Duplicate detection

## Racing Post

### Configuration

```yaml
data_sources:
  racing_post:
    enabled: false
    base_url: https://www.racingpost.com/api
    auth:
      api_key: ${RACING_POST_API_KEY}
    rate_limit:
      requests_per_second: 10
      burst_size: 20
    data_path: ./data/racing_post/
```

### Authentication

- **API Key**: Required for all requests
- Obtained from Racing Post developer portal

### API Endpoints

1. **Race Information**
   - Endpoint: `/races/{raceId}`
   - Returns: Race details, course, distance, conditions

2. **Horse Form Data**
   - Endpoint: `/selections/{selectionId}/form`
   - Returns: Historical runs, wins, place finishes

3. **Expert Analysis**
   - Endpoint: `/races/{raceId}/analysis`
   - Returns: Tips, ratings, confidence scores

### Rate Limiting

- **Requests per second**: 10
- **Burst capacity**: 20 requests
- Daily quota applies (check documentation)

### Data Format

Racing Post returns JSON with normalized field names:
```json
{
  "race_id": "12345",
  "race_name": "Wimbledon Apprentice Stakes",
  "course": "Wimbledon",
  "distance_meters": 1200,
  "race_type": "handicap",
  "selections": [
    {
      "selection_id": "54321",
      "horse_name": "Example Horse",
      "jockey": "J. Smith",
      "weight": 57.5,
      "form": "32145",
      "rating": 75
    }
  ]
}
```

## CSV

### Configuration

```yaml
data_sources:
  csv:
    enabled: false
    data_path: ./data/csv/
```

### File Format

Supports standard CSV with required columns:
- `event_id`: Unique race identifier
- `event_name`: Race name
- `track`: Course/venue
- `start_time`: Race start time (ISO 8601)
- `selection_id`: Horse/participant ID
- `selection_name`: Horse/participant name
- `price`: Betting odds (decimal format)
- `volume`: Matched volume in currency

### Directory Structure

```
data/csv/
├── 2024-01-01/
│   ├── races.csv
│   ├── selections.csv
│   └── markets.csv
├── 2024-01-02/
└── ...
```

### Parsing

The CSV parser:
1. Validates file format and required columns
2. Infers data types automatically
3. Handles missing values with defaults
4. Supports multiple file formats with configuration

## Data Normalization

All sources are normalized to a common schema before storage:

```go
type NormalizedRaceData struct {
    EventID        string
    EventName      string
    Track          string
    StartTime      time.Time
    RaceType       string
    Selections     []NormalizedSelection
    ProcessedAt    time.Time
}

type NormalizedSelection struct {
    SelectionID    string
    SelectionName  string
    Price          float64
    Volume         float64
    Status         string  // ACTIVE, INACTIVE, SCRATCHED
}
```

## Data Validation Rules

### Universal Rules
- Event start time must be in the future or within last 30 days
- Selection names must be non-empty
- Prices must be between 1.01 and 1000
- Volume must be non-negative

### Source-Specific Rules

**Betfair:**
- Market status validation
- Currency consistency
- Exchange platform rules

**Racing Post:**
- Course exists in database
- Distance within valid ranges
- Rating score 0-100

**CSV:**
- File encoding UTF-8
- Date format consistency
- No duplicate event IDs in single file

## Error Handling

### Validation Errors
- **Invalid**: Record rejected, logged, and skipped
- **Warning**: Data quality issue flagged but record kept
- **Critical**: Entire ingestion aborted

### Recovery

Failed ingestions can be:
1. Retried with exponential backoff
2. Manually re-triggered via API
3. Resumed from checkpoint

## Performance Characteristics

| Source | Latency | Volume | Reliability |
|--------|---------|--------|-------------|
| Betfair | Real-time (< 1s) | High (100K events/day) | 99.9% |
| Racing Post | 5-10s delay | Medium (50K events/day) | 99.5% |
| CSV | Batch upload | Custom | Dependent on source |

## Integration Example

```go
// Create factory
factory := datasource.NewFactory(config, logger)

// Create Betfair source
source, err := factory.Create(datasource.BetfairSourceType)
if err != nil {
    log.Fatal(err)
}

// Fetch data
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

data, metrics, err := source.FetchHistorical(ctx, startDate, endDate)
if err != nil {
    log.Printf("Fetch error: %v", err)
}

log.Printf("Ingested %d records", metrics.RecordsProcessed)
```

## Monitoring

Monitor ingestion health through metrics:

- `ingestion_records_processed`: Total records processed
- `ingestion_validation_errors`: Records failed validation
- `ingestion_http_requests`: API requests made
- `ingestion_duration_seconds`: Time to complete ingestion
- `ingestion_last_success`: Unix timestamp of last successful run

## Troubleshooting

### Betfair Connection Issues
- Verify session token is current
- Check SSL certificates exist and are valid
- Ensure app key has proper permissions
- Review rate limit compliance

### Racing Post API Errors
- Verify API key is valid and active
- Check daily quota usage
- Ensure course/distance data is up to date
- Review request format

### CSV Parsing Failures
- Validate file encoding (UTF-8)
- Check column headers match expected format
- Ensure date format is consistent (ISO 8601)
- Verify no special characters in event IDs

## Future Enhancements

1. **Additional Sources**: Timeform, Weather APIs
2. **Real-time Streaming**: WebSocket support for live data
3. **Data Deduplication**: Detect and merge duplicate records
4. **Advanced Caching**: Reduce API calls through smart caching
5. **Change Data Capture**: Track data modifications over time
