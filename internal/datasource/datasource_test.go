package datasource

import (
	"context"
	"testing"
	"time"
)

// TestDataValidatorValid tests validation of correct data
func TestDataValidatorValid(t *testing.T) {
	validator := NewDataValidator(nil)

	record := &Record{
		EventID:    "12345",
		EventName:  "Test Race",
		Track:      "Wimbledon",
		StartTime:  time.Now().Add(1 * time.Hour),
		Selection: &Selection{
			ID:     "54321",
			Name:   "Test Horse",
			Price:  3.50,
			Volume: 1000.0,
		},
	}

	err := validator.Validate(record)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestDataValidatorInvalidPrice tests validation of invalid prices
func TestDataValidatorInvalidPrice(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name  string
		price float64
		valid bool
	}{
		{"Valid minimum", 1.01, true},
		{"Valid maximum", 1000.0, true},
		{"Below minimum", 1.00, false},
		{"Above maximum", 1001.0, false},
		{"Zero", 0.0, false},
		{"Negative", -1.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &Record{
				EventID:   "12345",
				EventName: "Test Race",
				Track:     "Wimbledon",
				StartTime: time.Now().Add(1 * time.Hour),
				Selection: &Selection{
					ID:     "54321",
					Name:   "Test Horse",
					Price:  tt.price,
					Volume: 1000.0,
				},
			}

			err := validator.Validate(record)
			if (err == nil) != tt.valid {
				t.Errorf("Expected valid=%v, got error=%v", tt.valid, err)
			}
		})
	}
}

// TestDataValidatorInvalidStartTime tests start time validation
func TestDataValidatorInvalidStartTime(t *testing.T) {
	validator := NewDataValidator(nil)

	// Test past event (more than 30 days old)
	record := &Record{
		EventID:    "12345",
		EventName:  "Test Race",
		Track:      "Wimbledon",
		StartTime:  time.Now().Add(-31 * 24 * time.Hour),
		Selection: &Selection{
			ID:     "54321",
			Name:   "Test Horse",
			Price:  3.50,
			Volume: 1000.0,
		},
	}

	err := validator.Validate(record)
	if err == nil {
		t.Errorf("Expected error for old event, got nil")
	}
}

// TestDataNormalizerPrice tests price normalization to decimal format
func TestDataNormalizerPrice(t *testing.T) {
	normalizer := NewDataNormalizer(nil)

	// Test case: prices in different formats
	tests := []struct {
		name      string
		input     float64
		expected  float64
		tolerance float64
	}{
		{"Integer odds", 3.0, 3.0, 0.01},
		{"Fractional equivalent", 5.5, 5.5, 0.01},
		{"Small odds", 1.05, 1.05, 0.001},
		{"Large odds", 999.99, 999.99, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Normalizer typically ensures consistent format
			normalized := normalizer.normalizePrice(tt.input)
			if normalized < tt.expected-tt.tolerance || normalized > tt.expected+tt.tolerance {
				t.Errorf("Expected ~%f, got %f", tt.expected, normalized)
			}
		})
	}
}

// TestHTTPClientRateLimit tests rate limiting functionality
func TestHTTPClientRateLimit(t *testing.T) {
	client := NewHTTPClient(10, 20, nil) // 10 req/s, burst of 20

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make 15 quick requests - should succeed with burst
	for i := 0; i < 15; i++ {
		err := client.rateLimiter.Wait(ctx)
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// Measure time for next 10 sequential requests
	start := time.Now()
	for i := 0; i < 10; i++ {
		_ = client.rateLimiter.Wait(ctx)
	}
	elapsed := time.Since(start)

	// Should take approximately 1 second (10 requests at 10 req/s)
	expectedMin := time.Duration(800) * time.Millisecond
	expectedMax := time.Duration(1200) * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("Expected duration ~1s, got %v", elapsed)
	}
}

// TestIngestionMetrics tests metrics collection
func TestIngestionMetrics(t *testing.T) {
	metrics := NewIngestionMetrics(nil)

	// Simulate ingestion
	metrics.RecordStart()
	metrics.IncrementRecordsProcessed(100)
	metrics.IncrementValidationErrors(5)
	metrics.RecordEnd()

	if metrics.RecordsProcessed != 100 {
		t.Errorf("Expected 100 records processed, got %d", metrics.RecordsProcessed)
	}

	if metrics.ValidationErrors != 5 {
		t.Errorf("Expected 5 validation errors, got %d", metrics.ValidationErrors)
	}

	if metrics.Duration == 0 {
		t.Errorf("Expected non-zero duration")
	}
}

// TestBetfairCSVParserValidFormat tests CSV parsing with valid format
func TestBetfairCSVParserValidFormat(t *testing.T) {
	parser := NewBetfairCSVParser(nil)

	csvData := `event_id,event_name,track,start_time,selection_id,selection_name,price,volume
12345,Test Race,Wimbledon,2024-01-01T14:00:00Z,54321,Test Horse,3.50,1000.0`

	records, err := parser.Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if records[0].Selection.Price != 3.50 {
		t.Errorf("Expected price 3.50, got %f", records[0].Selection.Price)
	}
}

// TestBetfairCSVParserInvalidFormat tests CSV parsing with invalid format
func TestBetfairCSVParserInvalidFormat(t *testing.T) {
	parser := NewBetfairCSVParser(nil)

	// Missing required columns
	csvData := `event_id,event_name,track
12345,Test Race,Wimbledon`

	_, err := parser.Parse([]byte(csvData))
	if err == nil {
		t.Errorf("Expected error for missing columns, got nil")
	}
}

// TestDataSourceFactory tests factory creation
func TestDataSourceFactoryCreate(t *testing.T) {
	factory := NewFactory(nil, nil)

	tests := []struct {
		name        string
		sourceType  SourceType
		shouldError bool
	}{
		{"Betfair", BetfairSourceType, true},  // Will error without config
		{"Racing Post", RacingPostSourceType, true},  // Will error without config
		{"CSV", CSVSourceType, true},  // Will error without config
		{"Unknown", SourceType("unknown"), true},  // Will error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.Create(tt.sourceType)
			if (err == nil) != !tt.shouldError {
				t.Errorf("Expected error=%v, got error=%v", tt.shouldError, err)
			}
		})
	}
}

// BenchmarkDataValidator benchmarks validation performance
func BenchmarkDataValidator(b *testing.B) {
	validator := NewDataValidator(nil)

	record := &Record{
		EventID:    "12345",
		EventName:  "Test Race",
		Track:      "Wimbledon",
		StartTime:  time.Now().Add(1 * time.Hour),
		Selection: &Selection{
			ID:     "54321",
			Name:   "Test Horse",
			Price:  3.50,
			Volume: 1000.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(record)
	}
}

// BenchmarkDataNormalizer benchmarks normalization performance
func BenchmarkDataNormalizer(b *testing.B) {
	normalizer := NewDataNormalizer(nil)

	record := &Record{
		EventID:    "12345",
		EventName:  "Test Race",
		Track:      "Wimbledon",
		StartTime:  time.Now().Add(1 * time.Hour),
		Selection: &Selection{
			ID:     "54321",
			Name:   "Test Horse",
			Price:  3.50,
			Volume: 1000.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.Normalize(record)
	}
}
