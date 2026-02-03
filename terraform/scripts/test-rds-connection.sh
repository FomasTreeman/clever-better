#!/usr/bin/env bash
# Test RDS connection and TimescaleDB extension
# Usage: ./test-rds-connection.sh <environment>

set -euo pipefail

ENVIRONMENT=${1:-}
if [[ -z "$ENVIRONMENT" ]]; then
  echo "Usage: $0 <environment>"
  exit 1
fi

ENV_DIR="$(cd "$(dirname "$0")/.." && pwd)/environments/$ENVIRONMENT"
if [[ ! -d "$ENV_DIR" ]]; then
  echo "Environment directory not found: $ENV_DIR"
  exit 1
fi

pushd "$ENV_DIR" >/dev/null

SECRET_ARN=$(terraform output -raw rds_secret_arn)
if [[ -z "$SECRET_ARN" ]]; then
  echo "rds_secret_arn output is empty. Ensure Terraform apply has been run."
  exit 1
fi

CREDENTIALS=$(aws secretsmanager get-secret-value --secret-id "$SECRET_ARN" --query SecretString --output text)

DB_HOST=$(echo "$CREDENTIALS" | jq -r '.host')
DB_PORT=$(echo "$CREDENTIALS" | jq -r '.port')
DB_NAME=$(echo "$CREDENTIALS" | jq -r '.dbname')
DB_USER=$(echo "$CREDENTIALS" | jq -r '.username')
DB_PASS=$(echo "$CREDENTIALS" | jq -r '.password')

if [[ -z "$DB_HOST" || -z "$DB_NAME" || -z "$DB_USER" || -z "$DB_PASS" ]]; then
  echo "Failed to parse database credentials from Secrets Manager."
  exit 1
fi

export PGHOST="$DB_HOST"
export PGPORT="$DB_PORT"
export PGDATABASE="$DB_NAME"
export PGUSER="$DB_USER"
export PGPASSWORD="$DB_PASS"
export PGSSLMODE=require

echo "Testing connection to $DB_HOST:$DB_PORT/$DB_NAME"
psql -c "SELECT 1;"

echo "Checking TimescaleDB extension"
psql -c "SELECT default_version FROM pg_available_extensions WHERE name = 'timescaledb';"

popd >/dev/null

echo "RDS connection test completed successfully."
