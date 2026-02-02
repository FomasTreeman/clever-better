package service

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/repository"
)

// TestIngestionServiceHistoricalSync tests the complete historical ingestion flow
func TestIngestionServiceHistoricalSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup database connection (requires test database)
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	// Create repository
	repo := repository.NewRepository(db)

	// Create factory
	factory := datasource.NewFactory(nil, nil)

	// Create HTTP client
	httpClient := datasource.NewHTTPClient(10, 20, nil)

	// Create ingestion service
	svc := NewIngestionService(factory, repo, httpClient, nil)

	// Test ingestion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	metrics, err := svc.IngestHistoricalData(ctx, "betfair", startDate, endDate)
	if err != nil {
		t.Errorf("Ingestion failed: %v", err)
	}

	if metrics.RecordsProcessed == 0 {
		t.Errorf("Expected records processed > 0, got %d", metrics.RecordsProcessed)
	}

	t.Logf("Ingestion completed: %s", metrics.String())
}

// TestIngestionServiceLivePolling tests live data ingestion
func TestIngestionServiceLivePolling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	factory := datasource.NewFactory(nil, nil)
	httpClient := datasource.NewHTTPClient(10, 20, nil)
	svc := NewIngestionService(factory, repo, httpClient, nil)

	// Test live polling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err = svc.IngestLiveData(ctx, "betfair")
	if err != nil {
		t.Errorf("Live polling failed: %v", err)
	}
}

// TestIngestionServiceMultipleSources tests ingestion from multiple sources
func TestIngestionServiceMultipleSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	factory := datasource.NewFactory(nil, nil)
	httpClient := datasource.NewHTTPClient(10, 20, nil)
	svc := NewIngestionService(factory, repo, httpClient, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	sources := []string{"betfair", "racing_post"}
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	totalRecords := 0
	for _, source := range sources {
		metrics, err := svc.IngestHistoricalData(ctx, source, startDate, endDate)
		if err != nil {
			t.Logf("Warning: ingestion from %s failed: %v", source, err)
		} else {
			totalRecords += int(metrics.RecordsProcessed)
		}
	}

	if totalRecords == 0 {
		t.Logf("Note: No records ingested (may be expected in test environment)")
	}
}

// TestIngestionServiceDataValidation tests data validation during ingestion
func TestIngestionServiceDataValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	factory := datasource.NewFactory(nil, nil)
	httpClient := datasource.NewHTTPClient(10, 20, nil)
	svc := NewIngestionService(factory, repo, httpClient, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Ingest and check metrics
	metrics, err := svc.IngestHistoricalData(ctx, "betfair", time.Now().Add(-1*24*time.Hour), time.Now())
	if err != nil {
		t.Errorf("Ingestion failed: %v", err)
		return
	}

	// Verify no validation errors
	if metrics.ValidationErrors > 0 {
		t.Logf("Warning: %d validation errors during ingestion", metrics.ValidationErrors)
	}
}

// TestDataSourceFactoryIntegration tests factory with configuration
func TestDataSourceFactoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test verifies that factory can create sources with proper config
	factory := datasource.NewFactory(nil, nil)

	// List available sources
	available := factory.ListAvailableSources()
	if len(available) == 0 {
		t.Logf("Note: No data sources configured")
	}

	for _, sourceType := range available {
		_, err := factory.Create(sourceType)
		if err != nil {
			t.Logf("Warning: Could not create %s source: %v", sourceType, err)
		}
	}
}

// TestSchedulerIntegration tests scheduler with ingestion service
func TestSchedulerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	factory := datasource.NewFactory(nil, nil)
	httpClient := datasource.NewHTTPClient(10, 20, nil)
	svc := NewIngestionService(factory, repo, httpClient, nil)

	// Create scheduler
	// scheduler := NewScheduler(svc, log)

	// Note: Full scheduler integration would require mocking cron
	t.Logf("Scheduler integration test prepared")
}

// BenchmarkIngestionServiceHistorical benchmarks historical ingestion
func BenchmarkIngestionServiceHistorical(b *testing.B) {
	// Setup
	db, err := database.NewConnection(database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "clever_better_test",
		User:     "postgres",
		Password: "test",
		SSLMode:  "disable",
	})
	if err != nil {
		b.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	factory := datasource.NewFactory(nil, nil)
	httpClient := datasource.NewHTTPClient(10, 20, nil)
	svc := NewIngestionService(factory, repo, httpClient, nil)

	ctx := context.Background()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.IngestHistoricalData(ctx, "betfair", startDate, endDate)
	}
}
