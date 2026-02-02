# Configuration Management Guide

## Overview

This document explains how to configure the Clever Better application. The configuration system supports multiple sources with a clear priority order, validation at startup, and AWS Secrets Manager integration for sensitive credentials.

## Configuration File Structure

The application uses a hierarchical YAML configuration file with the following main sections:

- **app** - Application metadata and logging settings
- **database** - PostgreSQL connection configuration
- **betfair** - Betfair API authentication and URLs
- **ml_service** - ML service connectivity
- **trading** - Risk management and strategy parameters
- **backtest** - Backtesting parameters
- **data_ingestion** - Data source configuration
- **metrics** - Monitoring and observability settings
- **features** - Feature flags for different trading modes

### Environment Variable Placeholder Expansion

Configuration files support environment variable placeholders using the `${VAR_NAME}` syntax. These placeholders are automatically expanded when the configuration is loaded, **before** validation and before AWS Secrets Manager overlay.

**Example:**
```yaml
database:
  password: ${DB_PASSWORD}
```

When `DB_PASSWORD=my_secret` is set in the shell environment, it will be expanded to:
```yaml
database:
  password: my_secret
```

**Important:**
- Placeholders are expanded using `os.ExpandEnv()` which uses Unix-style `${VAR}` syntax
- If a variable is not set, the placeholder remains as a literal string: `${NONEXISTENT}` becomes `${NONEXISTENT}`
- Expansion happens **before** validation, so validation will catch invalid placeholder values
- This is especially useful for development configurations; production should use AWS Secrets Manager

### Example Configuration Files

- `config/config.yaml.example` - Template with all available options
- `config/config.development.yaml` - Development-optimized defaults
- `config/config.production.yaml` - Production-optimized defaults

## Loading Configuration

### Default Behavior

```go
cfg, err := config.Load("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### Configuration Loading Priority (Highest to Lowest)

The configuration system applies values in this order, with later sources overriding earlier ones:

1. **Hardcoded defaults** - Built-in defaults (from LoadWithDefaults)
2. **Configuration file** - YAML file (with `${VAR}` expansion)
3. **Environment variables** - `CLEVER_BETTER_*` variables
4. **AWS Secrets Manager** - Sensitive credentials (if enabled)

### Placeholder Expansion Flow

When loading a configuration file, the system performs placeholder expansion **before** parsing:

```
1. Read YAML file
   ↓
2. Expand ${VAR_NAME} placeholders using environment variables
   ↓
3. Parse expanded YAML
   ↓
4. Apply environment variable overrides (CLEVER_BETTER_*)
   ↓
5. Validate configuration
   ↓
6. (Optional) Overlay AWS Secrets Manager values
```

**Example of expansion priority:**
```yaml
# config.yaml
database:
  password: ${DB_PASS}  # Will use shell environment DB_PASS
```

```bash
# Shell environment
export DB_PASS=shell_value
export CLEVER_BETTER_DATABASE_PASSWORD=override_value
```

Result: `database.password` = `override_value` (environment variable wins over placeholder)

### With Custom Path

```go
cfg, err := config.Load("/etc/clever-better/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### With Defaults

```go
cfg, err := config.LoadWithDefaults("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

## Environment Variable Mapping

### Convention

Environment variables override YAML configuration values. Use the prefix `CLEVER_BETTER_` followed by the field path with underscores:

```
CLEVER_BETTER_<SECTION>_<FIELD> = value
```

### Examples

| Environment Variable | Configuration Path |
|---|---|
| `CLEVER_BETTER_APP_ENVIRONMENT` | `app.environment` |
| `CLEVER_BETTER_DATABASE_HOST` | `database.host` |
| `CLEVER_BETTER_DATABASE_PORT` | `database.port` |
| `CLEVER_BETTER_BETFAIR_APP_KEY` | `betfair.app_key` |
| `CLEVER_BETTER_TRADING_MAX_STAKE_PER_BET` | `trading.max_stake_per_bet` |
| `CLEVER_BETTER_ML_SERVICE_URL` | `ml_service.url` |

### Setting Environment Variables

#### macOS/Linux

```bash
export CLEVER_BETTER_DATABASE_HOST=prod-db.example.com
export CLEVER_BETTER_APP_ENVIRONMENT=production
```

#### Docker

```dockerfile
ENV CLEVER_BETTER_DATABASE_HOST=db.service
ENV CLEVER_BETTER_APP_ENVIRONMENT=production
```

#### Kubernetes

```yaml
containers:
  - name: clever-better
    env:
      - name: CLEVER_BETTER_DATABASE_HOST
        value: "db.default.svc.cluster.local"
      - name: CLEVER_BETTER_APP_ENVIRONMENT
        value: "production"
```

## AWS Secrets Manager Integration

### Overview

For production deployments, sensitive credentials can be stored in AWS Secrets Manager rather than in configuration files or environment variables.

### Prerequisites

1. AWS SDK v2 credentials configured (IAM role, credentials file, etc.)
2. AWS Secrets Manager secret created with required credentials
3. Application has read access to the secret

### Secret Structure

Create a JSON secret in AWS Secrets Manager with the following structure:

```json
{
  "database_password": "your-database-password",
  "betfair_app_key": "your-betfair-app-key",
  "betfair_username": "your-betfair-username",
  "betfair_password": "your-betfair-password",
  "racing_post_api_key": "your-racing-post-api-key"
}
```

### Configuration

Set environment variables to enable AWS Secrets Manager integration:

```bash
export AWS_SECRETS_ENABLED=true
export AWS_REGION=us-east-1
export AWS_SECRET_NAME=clever-better/production/secrets
```

### Code Integration

```go
cfg, err := config.Load("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Load AWS secrets if enabled
if os.Getenv("AWS_SECRETS_ENABLED") == "true" {
    region := os.Getenv("AWS_REGION")
    secretName := os.Getenv("AWS_SECRET_NAME")
    if err := config.LoadSecretsFromAWS(cfg, region, secretName); err != nil {
        log.Fatalf("Failed to load secrets: %v", err)
    }
}

// Validate configuration
if err := config.Validate(cfg); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Retrieving Secrets Only

If you need to retrieve secrets without applying them to the configuration:

```go
secrets, err := config.GetSecretsFromAWS("us-east-1", "clever-better/production/secrets")
if err != nil {
    log.Fatalf("Failed to get secrets: %v", err)
}

password := secrets.DatabasePassword
```

## Configuration Validation

### Automatic Validation

Configuration is validated after loading:

```go
if err := config.Validate(cfg); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Valid Market Types

The `trading.markets` configuration field accepts the following Betfair market types:

- `WIN` - Win market (horse racing)
- `PLACE` - Place market (horse racing)
- `EW` - Each-way market

**Example:**
```yaml
trading:
  markets:
    - WIN
    - PLACE
```

**Invalid markets will fail validation:**
```yaml
trading:
  markets:
    - FOO    # ✗ Invalid - will fail validation
    - PLACE  # ✓ Valid
```

Validation errors for invalid markets:
```
configuration validation failed:
- Field 'Trading.Markets' validation failed: markets
```

### Validation Rules

#### Required Fields

All top-level configuration sections must be present:
- app
- database
- betfair
- ml_service
- trading
- backtest
- data_ingestion
- metrics
- features

#### Field Constraints

**Environment**
- Required field
- Must be one of: `development`, `staging`, `production`

**Log Level**
- Required field
- Must be one of: `debug`, `info`, `warn`, `error`

**Database**
- Host: Required, non-empty string
- Port: Required, 1-65535
- Name: Required, non-empty string
- User: Required, non-empty string
- Password: Required, non-empty string (can be empty string for local dev with trust auth)
- SSLMode: Required, one of `disable`, `require`, `verify-full`
- MaxConnections: Required, > 0
- MaxIdleConnections: Required, > 0, <= MaxConnections

**Betfair**
- APIURL: Required, valid URL
- StreamURL: Required, non-empty string
- AppKey: Required, non-empty string
- Username: Required, non-empty string
- Password: Required, non-empty string
- CertFile: Required, non-empty path
- KeyFile: Required, non-empty path

**Trading**
- MaxStakePerBet: Required, > 0
- MaxDailyLoss: Required, > 0
- MaxExposure: Required, > 0, >= MaxDailyLoss
- MinConfidenceThreshold: Required, 0-1
- MinExpectedValue: Required, >= 0
- Markets: Required, non-empty array, valid market types only (WIN, PLACE, EW)
- PreRaceWindowMinutes: Required, >= 0
- MinTimeToStartSeconds: Required, >= 0

**Backtest**
- StartDate: Required, valid date (YYYY-MM-DD)
- EndDate: Required, valid date (YYYY-MM-DD), > StartDate
- InitialBankroll: Required, > 0
- MonteCarloIterations: Required, > 0
- WalkForwardWindows: Required, > 0

**ML Service**
- URL: Required, valid URL
- GRPCAddress: Required, non-empty string
- TimeoutSeconds: Required, > 0
- RetryAttempts: Required, >= 0

**Data Ingestion**
- Sources: Required, at least one source
- Schedule.HistoricalSync: Required, valid cron expression
- Schedule.LivePollingIntervalSeconds: Required, > 0

**Metrics**
- Port: Required, 1-65535
- Path: Required, non-empty string

### Environment-Specific Validation

**Production**
- Database SSL mode must be `require` or `verify-full`
- At least one trading mode must be enabled
- Betfair credentials must not be test/placeholder values

**Development**
- Live trading should be disabled
- Paper trading should be enabled

### Validation Error Handling

Validation errors are returned with detailed field information:

```
configuration validation failed:
- Field 'App.Environment' must be one of: development, staging, production
- Field 'Database.Port' validation failed: numeric constraint min violated
- Field 'Trading.Markets' has invalid value '[]'
- backtest start_date must be before end_date
```

### Custom Validation

To add custom validation logic:

```go
cfg, err := config.Load("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Custom validation
if cfg.IsProduction() && cfg.Features.LiveTradingEnabled {
    log.Fatalf("Live trading must be disabled in production before manual enable")
}
```

## Local Development Configuration

### Quick Start

1. Copy the template:
```bash
cp config/config.yaml.example config/config.yaml
```

2. Edit `config/config.yaml` with your local settings

3. (Optional) Create a `.env` file for environment variables:
```bash
export CLEVER_BETTER_DATABASE_PASSWORD=your_local_password
export CLEVER_BETTER_BETFAIR_APP_KEY=your_app_key
export CLEVER_BETTER_BETFAIR_USERNAME=your_username
export CLEVER_BETTER_BETFAIR_PASSWORD=your_password
```

4. Source the environment:
```bash
source .env
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: clever_better_dev
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: dev_password
    ports:
      - "5432:5432"

  app:
    build: .
    environment:
      CLEVER_BETTER_DATABASE_HOST: postgres
      CLEVER_BETTER_DATABASE_PASSWORD: dev_password
      CLEVER_BETTER_APP_ENVIRONMENT: development
    depends_on:
      - postgres
```

## Production Configuration

### AWS Setup

1. Create an IAM user or role with Secrets Manager read access
2. Create a secret in Secrets Manager:
```bash
aws secretsmanager create-secret \
  --name clever-better/production/secrets \
  --secret-string '{"database_password":"...","betfair_app_key":"...",...}'
```

3. Set environment variables in your deployment:
```bash
AWS_SECRETS_ENABLED=true
AWS_REGION=us-east-1
AWS_SECRET_NAME=clever-better/production/secrets
CLEVER_BETTER_APP_ENVIRONMENT=production
CLEVER_BETTER_DATABASE_HOST=prod-db.example.com
```

### Terraform Example

```hcl
resource "aws_secretsmanager_secret" "app_secrets" {
  name = "clever-better/production/secrets"
}

resource "aws_secretsmanager_secret_version" "app_secrets" {
  secret_id = aws_secretsmanager_secret.app_secrets.id
  secret_string = jsonencode({
    database_password      = var.db_password
    betfair_app_key        = var.betfair_app_key
    betfair_username       = var.betfair_username
    betfair_password       = var.betfair_password
    racing_post_api_key    = var.racing_post_api_key
  })
}

resource "aws_ecs_task_definition" "app" {
  # ...
  container_definitions = jsonencode([{
    environment = [
      { name = "AWS_SECRETS_ENABLED", value = "true" },
      { name = "AWS_REGION", value = var.aws_region },
      { name = "AWS_SECRET_NAME", value = aws_secretsmanager_secret.app_secrets.name },
      { name = "CLEVER_BETTER_APP_ENVIRONMENT", value = "production" },
      { name = "CLEVER_BETTER_DATABASE_HOST", value = aws_db_instance.postgres.address }
    ]
  }])
}
```

## Security Best Practices

### Don't Do This

❌ Never commit `config/config.yaml` with real credentials to version control

❌ Never log configuration values, especially passwords

❌ Never store production credentials in environment variables in code

❌ Never use the same credentials in development and production

### Do This

✅ Use `config/config.yaml.example` as a template and document required fields

✅ Use `.gitignore` to prevent committing actual configuration files

✅ Use AWS Secrets Manager for production credentials

✅ Use environment variables or .env files for local development only

✅ Rotate credentials regularly

✅ Use different credentials for different environments

✅ Audit access to Secrets Manager

✅ Log which sources were loaded (secrets, files, env) for debugging

## Troubleshooting

### Configuration File Not Found

**Error:** `config file not found at config/config.yaml`

**Solution:**
1. Ensure the file exists: `ls -la config/config.yaml`
2. Check file permissions: `chmod 644 config/config.yaml`
3. Use absolute path: `CLEVER_BETTER_CONFIG_PATH=/etc/clever-better/config.yaml`

### Invalid YAML Format

**Error:** `failed to unmarshal configuration`

**Solution:**
1. Validate YAML syntax: `python -m yaml config/config.yaml`
2. Check indentation (must be spaces, not tabs)
3. Use quotes for string values with special characters

### Validation Errors

**Error:** `configuration validation failed: Field 'Database.Port' validation failed: numeric constraint min violated`

**Solution:**
1. Check field type: Port must be an integer, not a string
2. Check field range: Port must be 1-65535
3. Review constraint tags in config.go

### Environment Variable Not Applied

**Symptom:** Environment variable is set but not used

**Solution:**
1. Verify prefix: Must start with `CLEVER_BETTER_`
2. Verify naming: Use underscores, not dots or dashes
3. Verify export: Use `export VAR=value` not just `VAR=value`
4. Check priority: YAML file is loaded first, environment overrides

### AWS Secrets Manager Access Denied

**Error:** `failed to get secret from AWS Secrets Manager: AccessDenied`

**Solution:**
1. Verify IAM permissions: User/role needs `secretsmanager:GetSecretValue`
2. Verify secret exists: `aws secretsmanager describe-secret --secret-id <name>`
3. Verify region: Check `AWS_REGION` environment variable
4. Verify credentials: `aws sts get-caller-identity`

### SSL Certificate Errors

**Error:** `failed to load certificates: no such file or directory`

**Solution:**
1. Verify file paths in config: `cert_file` and `key_file`
2. Check file permissions: `ls -la` for cert/key files
3. Verify absolute or relative paths match your working directory
4. For Docker: Ensure volumes are mounted correctly

## Configuration Reference

### All Available Fields

See [CONFIG_REFERENCE.md](CONFIG_REFERENCE.md) for complete field documentation.

## Advanced Topics

### Configuration Hot Reload

Configuration is loaded once at startup. To reload:

```go
newCfg, err := config.Load("config/config.yaml")
if err != nil {
    log.Printf("Failed to reload config: %v", err)
}

// Use newCfg for subsequent operations
```

### Custom Configuration Paths

```go
configPath := os.Getenv("APP_CONFIG_PATH")
if configPath == "" {
    configPath = "config/config.yaml"
}

cfg, err := config.Load(configPath)
```

### Multiple Configuration Files

Currently not supported in a single load. Workaround:

```go
base, _ := config.Load("config/config.yaml")
overrides, _ := config.Load("config/config.local.yaml")

// Manually merge overrides into base
```

## Support

For issues, questions, or configuration help:
- Check logs for validation errors
- Review the examples in `config/` directory
- See the troubleshooting section above
- Examine the test fixtures in `internal/config/testdata/`
