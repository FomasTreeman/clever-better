# TimescaleDB Setup Guide

This guide walks through installing and validating the TimescaleDB extension on the provisioned RDS PostgreSQL instance.

## Prerequisites

- RDS instance is provisioned and available
- Security group allows access from your client or bastion host
- Credentials retrieved from AWS Secrets Manager
- `psql` or pgAdmin installed on your workstation

## 1. Retrieve Database Credentials

```bash
# Get secret ARN from Terraform output
SECRET_ARN=$(terraform output -raw rds_secret_arn)

# Retrieve credentials
aws secretsmanager get-secret-value \
  --secret-id $SECRET_ARN \
  --query SecretString \
  --output text | jq .
```

The secret JSON contains:
- `username`
- `password`
- `host`
- `port`
- `dbname`

## 2. Connect to the Database

### Using psql

```bash
export PGHOST=<host>
export PGPORT=5432
export PGDATABASE=clever_better
export PGUSER=admin
export PGPASSWORD=<password>
export PGSSLMODE=require

psql
```

### Using pgAdmin

1. Create a new server connection
2. Host: `<host>`
3. Port: `5432`
4. Maintenance DB: `clever_better`
5. Username: `admin`
6. Password: `<password>`
7. SSL Mode: `require`

## 3. Install TimescaleDB Extension

Run the following SQL as the master user:

```sql
-- Install TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Verify installation
SELECT default_version FROM pg_available_extensions WHERE name = 'timescaledb';

-- Confirm extension is installed
SELECT extname, extversion FROM pg_extension WHERE extname = 'timescaledb';
```

## 4. Convert Tables to Hypertables

Identify time-series tables and convert them using TimescaleDB:

```sql
-- Example: Convert bets table to hypertable
SELECT create_hypertable('bets', 'created_at');

-- Example: Convert odds table to hypertable
SELECT create_hypertable('odds', 'recorded_at');
```

### Use Migration Files

Refer to the migration files in [migrations/](../../../../migrations/) for table definitions. For example:

```sql
-- Example migration logic
CREATE TABLE odds (
  id UUID PRIMARY KEY,
  runner_id UUID NOT NULL,
  price NUMERIC NOT NULL,
  recorded_at TIMESTAMP NOT NULL
);

SELECT create_hypertable('odds', 'recorded_at');
```

## 5. Configure Compression Policies

Compression reduces storage costs for historical data:

```sql
-- Enable compression on hypertable
ALTER TABLE odds SET (
  timescaledb.compress,
  timescaledb.compress_segmentby = 'runner_id'
);

-- Add compression policy (compress data older than 7 days)
SELECT add_compression_policy('odds', INTERVAL '7 days');
```

## 6. Set Up Continuous Aggregates

Continuous aggregates improve query performance for dashboards:

```sql
-- Create continuous aggregate
CREATE MATERIALIZED VIEW odds_daily_stats
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 day', recorded_at) AS day,
  runner_id,
  avg(price) as avg_price,
  min(price) as min_price,
  max(price) as max_price
FROM odds
GROUP BY day, runner_id;

-- Refresh policy (every hour)
SELECT add_continuous_aggregate_policy('odds_daily_stats',
  start_offset => INTERVAL '7 days',
  end_offset   => INTERVAL '1 hour',
  schedule_interval => INTERVAL '1 hour');
```

## 7. Verify Performance

### Check Hypertables

```sql
SELECT * FROM timescaledb_information.hypertables;
```

### Check Compression

```sql
SELECT * FROM timescaledb_information.compressed_hypertables;
```

### Check Continuous Aggregates

```sql
SELECT * FROM timescaledb_information.continuous_aggregates;
```

## 8. Performance Tuning Recommendations

### Indexing

- Add indexes on time and filtering columns
- Use composite indexes for common query patterns

```sql
CREATE INDEX ON odds (runner_id, recorded_at DESC);
```

### Chunk Size

Adjust chunk size for large tables:

```sql
SELECT set_chunk_time_interval('odds', INTERVAL '1 day');
```

### Retention Policies

Remove old data automatically:

```sql
SELECT add_retention_policy('odds', INTERVAL '90 days');
```

## 9. Troubleshooting

### Extension Not Found

If `CREATE EXTENSION timescaledb` fails, verify:

1. Parameter group has `shared_preload_libraries = 'timescaledb'`
2. RDS instance is using the custom parameter group
3. The instance was rebooted after parameter group update

```sql
SHOW shared_preload_libraries;
```

### Permission Errors

Ensure you are connected as the master user (admin).

### Connection Issues

Check:
- Security group inbound rule for port 5432
- Network ACLs
- VPN or bastion host access

## References

- [TimescaleDB Documentation](https://docs.timescale.com/)
- [TimescaleDB on RDS](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Appendix.PostgreSQL.CommonDBATasks.html#Appendix.PostgreSQL.CommonDBATasks.TimescaleDB)
- [PostgreSQL Extensions](https://www.postgresql.org/docs/current/extend-extensions.html)
