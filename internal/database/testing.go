package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/clever-better/internal/config"
)

// SetupTestDB creates a test database connection and verifies it
func SetupTestDB(t *testing.T) *DB {
	// Load config for test database
	cfg, err := config.Load("../../config/config.yaml.test")
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	// Create context for connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := NewDB(ctx, &cfg.Database)
	if err != nil {
		t.Fatalf("failed to create test database connection: %v", err)
	}

	// Verify connection
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer verifyCancel()

	if err := db.Ping(verifyCtx); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	return db
}

// TeardownTestDB closes the database connection cleanly
func TeardownTestDB(t *testing.T, db *DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Close(ctx); err != nil {
		t.Logf("warning: failed to close test database: %v", err)
	}
}

// RunMigrations runs database migrations from the migrations directory
// Uses golang-migrate CLI for test execution
func RunMigrations(ctx context.Context, db *DB) error {
	// Note: In tests, migrations should be run with golang-migrate CLI:
	// migrate -path migrations -database "postgres://..." up
	//
	// This is a placeholder for programmatic migration in tests if needed.
	// For most cases, use migrate CLI before running tests.
	return fmt.Errorf("use migrate CLI: migrate -path migrations -database \"your_dsn\" up")
}
