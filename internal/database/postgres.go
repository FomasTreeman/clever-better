package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/clever-better/internal/config"
)

// DB wraps the pgxpool.Pool to provide database operations
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new database connection pool from configuration
func NewDB(ctx context.Context, cfg *config.DatabaseConfig) (*DB, error) {
	// Create connection string from configuration
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Apply pool settings from configuration
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 1 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	// Create the pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Ping verifies database connectivity
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close gracefully closes the connection pool
func (db *DB) Close(ctx context.Context) error {
	if db.pool != nil {
		return db.pool.Close(ctx)
	}
	return nil
}

// QueryRow executes a query that returns at most one row
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) interface{} {
	return db.pool.QueryRow(ctx, query, args...)
}

// Query executes a query that returns multiple rows
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) interface{} {
	return db.pool.Query(ctx, query, args...)
}

// Exec executes a command
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) interface{} {
	return db.pool.Exec(ctx, query, args...)
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (interface{}, error) {
	return db.pool.Begin(ctx)
}

// WithTransaction provides transaction support with automatic rollback on error
func (db *DB) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute function within transaction context
	txCtx := context.WithValue(ctx, "tx", tx)
	if err := fn(txCtx); err != nil {
		// Rollback on error
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %w", err, rollbackErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// HealthCheck performs a simple health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, "SELECT 1")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// GetPool returns the underlying connection pool for advanced operations
func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}
