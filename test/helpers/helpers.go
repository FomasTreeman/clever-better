package helpers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// SetupTestDB creates a test database connection and runs migrations.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://test:test@localhost:5432/clever_better_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "failed to connect to test database")

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	require.NoError(t, err, "failed to ping test database")

	// Run migrations
	err = runMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	return db
}

// TeardownTestDB closes the database connection and cleans up test data.
func TeardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clean up all tables
	tables := []string{
		"bets",
		"race_results",
		"odds_snapshots",
		"races",
		"strategies",
		"backtest_results",
		"ml_predictions",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}

	err := db.Close()
	require.NoError(t, err, "failed to close database connection")
}

// runMigrations applies database migrations.
func runMigrations(db *sql.DB) error {
	// This is a simplified version - in production, use a proper migration tool
	// like golang-migrate or goose
	return nil
}

// LoadFixture loads test data from a JSON fixture file.
func LoadFixture(t *testing.T, filename string, target interface{}) {
	t.Helper()

	fixturePath := filepath.Join("test", "fixtures", filename)
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "failed to read fixture file: %s", filename)

	err = json.Unmarshal(data, target)
	require.NoError(t, err, "failed to unmarshal fixture: %s", filename)
}

// LoadRaceFixtures loads race test data from fixtures.
func LoadRaceFixtures(t *testing.T) []map[string]interface{} {
	t.Helper()

	var races []map[string]interface{}
	LoadFixture(t, "races.json", &races)
	return races
}

// LoadOddsFixtures loads odds snapshot test data.
func LoadOddsFixtures(t *testing.T) []map[string]interface{} {
	t.Helper()

	var odds []map[string]interface{}
	LoadFixture(t, "odds_snapshots.json", &odds)
	return odds
}

// LoadBacktestFixtures loads backtest result test data.
func LoadBacktestFixtures(t *testing.T) []map[string]interface{} {
	t.Helper()

	var results []map[string]interface{}
	LoadFixture(t, "backtest_results.json", &results)
	return results
}

// LoadMLPredictionFixtures loads ML prediction test data.
func LoadMLPredictionFixtures(t *testing.T) []map[string]interface{} {
	t.Helper()

	var predictions []map[string]interface{}
	LoadFixture(t, "ml_predictions.json", &predictions)
	return predictions
}

// MockBetfairServer creates a mock HTTP server for Betfair API testing.
func MockBetfairServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sessionToken": "mock-session-token",
				"loginStatus":  "SUCCESS",
			})

		case "/api/listMarketCatalogue":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"marketId":   "1.198765433",
					"marketName": "Win Market",
					"eventName":  "Ascot Race 1",
				},
			})

		case "/api/listMarketBook":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"marketId": "1.198765433",
					"runners": []map[string]interface{}{
						{
							"selectionId": 12345678,
							"status":      "ACTIVE",
							"ex": map[string]interface{}{
								"availableToBack": []map[string]float64{
									{"price": 3.5, "size": 1500.0},
								},
								"availableToLay": []map[string]float64{
									{"price": 3.6, "size": 1200.0},
								},
							},
						},
					},
				},
			})

		case "/api/placeOrders":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "SUCCESS",
				"instructionReports": []map[string]interface{}{
					{
						"status":      "SUCCESS",
						"betId":       "12345",
						"placedDate":  time.Now().Format(time.RFC3339),
						"averagePrice": 3.5,
						"sizeMatched": 50.0,
					},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	return httptest.NewServer(handler)
}

// MockMLServiceServer creates a mock HTTP server for ML service testing.
func MockMLServiceServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":        "healthy",
				"models_loaded": 1,
			})

		case "/predict":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"prediction_id": "pred-test-001",
				"confidence":    0.85,
				"probability":   0.78,
				"model_id":      "model-v1.0.0",
			})

		case "/predict/batch":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			predictions := req["predictions"].([]interface{})
			results := make([]map[string]interface{}, len(predictions))

			for i := range predictions {
				results[i] = map[string]interface{}{
					"confidence":  0.80,
					"probability": 0.70,
				}
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"predictions": results,
				"model_id":    "model-v1.0.0",
			})

		case "/train":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"job_id": "train-test-001",
				"status": "queued",
			})

		case "/feedback":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":      "success",
				"feedback_id": "fb-test-001",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	return httptest.NewServer(handler)
}

// WaitForCondition waits for a condition to become true or times out.
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.Fail(t, "condition not met within timeout", message)
}

// AssertEventuallyTrue retries an assertion until it passes or times out.
func AssertEventuallyTrue(t *testing.T, timeout time.Duration, assertion func() bool, message string) {
	t.Helper()

	WaitForCondition(t, timeout, assertion, message)
}

// CleanupDatabase truncates all test tables.
func CleanupDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{
		"bets",
		"race_results",
		"odds_snapshots",
		"races",
		"strategies",
		"backtest_results",
		"ml_predictions",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// CreateTestContext creates a context with a timeout for testing.
func CreateTestContext(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
}

// InsertTestRace inserts a test race into the database.
func InsertTestRace(t *testing.T, db *sql.DB, race map[string]interface{}) int64 {
	t.Helper()

	query := `
		INSERT INTO races (event_id, market_id, start_time, venue, distance, going, race_class)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := db.QueryRow(
		query,
		race["event_id"],
		race["market_id"],
		race["start_time"],
		race["venue"],
		race["distance"],
		race["going"],
		race["race_class"],
	).Scan(&id)

	require.NoError(t, err, "failed to insert test race")
	return id
}

// InsertTestBet inserts a test bet into the database.
func InsertTestBet(t *testing.T, db *sql.DB, bet map[string]interface{}) int64 {
	t.Helper()

	query := `
		INSERT INTO bets (race_id, runner_id, stake, odds, bet_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int64
	err := db.QueryRow(
		query,
		bet["race_id"],
		bet["runner_id"],
		bet["stake"],
		bet["odds"],
		bet["bet_type"],
		bet["timestamp"],
	).Scan(&id)

	require.NoError(t, err, "failed to insert test bet")
	return id
}

// GetEnvOrDefault returns environment variable value or a default.
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SkipIfShort skips test if running in short mode.
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// SkipIfNoDocker skips test if Docker is not available.
func SkipIfNoDocker(t *testing.T) {
	// Simple check - try to ping Docker
	// In a real implementation, you would check if Docker daemon is running
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("skipping test - Docker not available")
	}
}
