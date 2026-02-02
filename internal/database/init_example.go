package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/repository"
)

// initDatabase initializes the database connection and verifies migrations
func initDatabase(ctx context.Context, cfg *config.Config) (*DB, *repository.Repositories, error) {
	// Initialize database (creates connection, verifies TimescaleDB and migrations)
	db, err := database.Initialize(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	log.Println("✓ Database initialized and ready")

	// Create repositories
	repos, err := repository.NewRepositories(db)
	if err != nil {
		db.Close(ctx)
		return nil, nil, fmt.Errorf("failed to create repositories: %w", err)
	}

	log.Println("✓ Repositories created successfully")

	return db, repos, nil
}

// closeDatabase gracefully closes the database connection
func closeDatabase(ctx context.Context, db *database.DB) error {
	return db.Close(ctx)
}

// Example usage in main.go:
// 
// func main() {
//     // Load configuration
//     cfg, err := config.Load("config/config.yaml")
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     // Create context with cancellation for graceful shutdown
//     ctx, cancel := context.WithCancel(context.Background())
//     defer cancel()
//
//     // Initialize database and repositories
//     db, repos, err := initDatabase(ctx, cfg)
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer closeDatabase(ctx, db)
//
//     // Use repositories
//     races, err := repos.Race.GetUpcoming(ctx, 10)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     // Process races...
//
//     // Cleanup
//     if err := closeDatabase(repos.DB); err != nil {
//         log.Printf("warning: error closing database: %v", err)
//     }
// }
