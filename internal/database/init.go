package database

import (
	"context"
	"fmt"

	"github.com/yourusername/clever-better/internal/config"
)

// Initialize creates a database connection pool and verifies TimescaleDB installation
func Initialize(ctx context.Context, cfg *config.Config) (*DB, error) {
	// Create connection pool
	db, err := NewDB(ctx, &cfg.Database)
	if err != nil {
		return nil, err
	}

	// Verify TimescaleDB extension is installed
	var extName string
	err = db.pool.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'timescaledb'").Scan(&extName)
	if err != nil {
		closeErr := db.Close(ctx)
		if closeErr != nil {
			return nil, fmt.Errorf("TimescaleDB extension not found and close failed: close=%w, ext=%w", closeErr, err)
		}
		return nil, fmt.Errorf(
			"TimescaleDB extension not found. Please install TimescaleDB: " +
			"https://docs.timescale.com/getting-started/latest/installation/",
		)
	}

	// Verify migrations are applied by checking schema_migrations table
	var migrationCount int
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
	if err != nil {
		// Table might not exist yet, which is OK for initial setup
		return db, nil
	}

	if migrationCount == 0 {
		fmt.Println("Warning: No migrations have been applied. Please run database migrations.")
	}

	return db, nil
}
