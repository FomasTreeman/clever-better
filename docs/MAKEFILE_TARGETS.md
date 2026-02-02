# Makefile Database Targets Reference

## Overview

The Makefile includes database management targets for initializing, migrating, and managing the PostgreSQL + TimescaleDB database.

## Setup

### Initial Database Setup

```bash
# Create database and run all migrations (one command)
make db-setup

# Verify the database is accessible
make db-health-check
```

## Database Management

### Creating & Destroying

```bash
# Create empty database
make db-create

# Drop database (destructive!)
make db-drop

# Drop and recreate database
make db-reset
```

### Migrations

```bash
# Run all pending migrations
make db-migrate-up

# Rollback all migrations (one step)
make db-migrate-down

# Show current migration version
make db-migrate-status

# Create new migration pair
make db-migrate-create NAME=add_new_column

# Force migration to specific version (emergency only)
make db-migrate-force VERSION=3
```

## Configuration

Database connection uses these environment variables (from Makefile or .env):

```
DB_HOST       = localhost
DB_PORT       = 5432
DB_NAME       = clever_better
DB_USER       = postgres
DB_PASSWORD   = postgres
```

Override like:

```bash
make db-setup DB_HOST=prod-db.example.com DB_NAME=prod_clever_better
```

## Common Workflows

### Development Setup

```bash
# Initial setup
make db-setup
make db-health-check

# Start developing
make test
make build
```

### Testing

```bash
# Reset database to clean state
make db-reset
make db-migrate-up

# Run tests
make test
```

### Creating a New Migration

```bash
# Create migration files
make db-migrate-create NAME=add_runners_history_table

# Edit migrations/XXXXXX_add_runners_history_table.up.sql
# Edit migrations/XXXXXX_add_runners_history_table.down.sql

# Test the migration
make db-reset
make db-migrate-up

# Verify schema
psql -U postgres clever_better -c "\d"
```

### Production-Like Testing

```bash
# Reset to production state
make db-reset
make db-migrate-up

# Load test data
psql -U postgres clever_better < testdata/seed.sql

# Run integration tests
make test-integration
```

## Troubleshooting

### Database Connection Issues

```bash
# Check if database is accessible
make db-health-check

# Manual connection test
psql -h localhost -U postgres -d clever_better
```

### Migration Issues

```bash
# Check current migration version
make db-migrate-status

# View migration history
psql -U postgres clever_better -c "SELECT * FROM schema_migrations;"

# Rollback single migration
make db-migrate-down

# Force migration version if stuck
make db-migrate-force VERSION=3
```

### PostgreSQL Service Issues

```bash
# macOS - Start/Stop PostgreSQL
pg_ctl -D /usr/local/var/postgres start
pg_ctl -D /usr/local/var/postgres stop

# Ubuntu - Start/Stop PostgreSQL  
sudo systemctl start postgresql
sudo systemctl stop postgresql
```

## Advanced Usage

### Dump Database

```bash
pg_dump -U postgres clever_better > backup.sql
```

### Restore Database

```bash
psql -U postgres clever_better < backup.sql
```

### Connect to Database

```bash
psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME)
```

### Run SQL Script

```bash
psql -U postgres clever_better -f script.sql
```
