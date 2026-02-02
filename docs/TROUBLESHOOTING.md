# Troubleshooting Guide

This document provides solutions to common issues encountered when developing, deploying, or operating Clever Better.

## Table of Contents

- [Development Issues](#development-issues)
- [Build Issues](#build-issues)
- [Runtime Issues](#runtime-issues)
- [Database Issues](#database-issues)
- [Betfair API Issues](#betfair-api-issues)
- [ML Service Issues](#ml-service-issues)
- [Deployment Issues](#deployment-issues)

## Development Issues

### Go module issues

**Problem**: `go mod tidy` fails or dependencies not resolving

**Solution**:
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify modules
go mod verify

# If still failing, check go.sum
rm go.sum
go mod tidy
```

### Python virtual environment issues

**Problem**: Python packages not found or wrong versions

**Solution**:
```bash
# Recreate virtual environment
cd ml-service
rm -rf venv
python3.11 -m venv venv
source venv/bin/activate
pip install --upgrade pip
pip install -r requirements-dev.txt
```

### Docker containers not starting

**Problem**: `make docker-up` fails

**Solution**:
```bash
# Check Docker daemon is running
docker info

# Clean up old containers/networks
docker-compose down -v
docker system prune -f

# Check port conflicts
lsof -i :5432  # PostgreSQL
lsof -i :8000  # ML service

# Rebuild images
make docker-build
make docker-up
```

### Port already in use

**Problem**: `address already in use` error

**Solution**:
```bash
# Find process using port
lsof -i :8000

# Kill process
kill -9 <PID>

# Or use different port in config
export ML_SERVICE_PORT=8001
```

## Build Issues

### Go build fails

**Problem**: Compilation errors

**Common Causes**:
1. Missing dependencies
2. Go version mismatch
3. CGO issues

**Solution**:
```bash
# Check Go version
go version  # Should be 1.22+

# Update dependencies
go mod tidy

# If CGO issues (e.g., sqlite)
CGO_ENABLED=0 go build ./cmd/bot

# For more verbose output
go build -v ./...
```

### Docker build fails

**Problem**: Dockerfile build errors

**Solution**:
```bash
# Build with no cache
docker build --no-cache -t clever-better-bot .

# Check for syntax errors
docker build --progress=plain -t clever-better-bot .

# Multi-stage build issues - build specific stage
docker build --target builder -t clever-better-builder .
```

### Python package installation fails

**Problem**: `pip install` fails

**Solution**:
```bash
# Upgrade pip
pip install --upgrade pip setuptools wheel

# Install with verbose output
pip install -r requirements.txt -v

# For packages with native extensions
pip install --no-binary :all: <package>

# If SSL issues
pip install --trusted-host pypi.org --trusted-host files.pythonhosted.org <package>
```

## Runtime Issues

### Application crashes on startup

**Problem**: Service exits immediately

**Diagnosis**:
```bash
# Check logs
docker logs clever-better-bot

# Run interactively for better error messages
go run ./cmd/bot

# Check configuration
cat config/config.yaml
```

**Common Causes**:
1. Missing configuration file
2. Invalid configuration values
3. Database connection failure
4. Missing secrets/environment variables

### Out of memory errors

**Problem**: `OOM killed` or memory allocation failures

**Solution**:
```bash
# Increase Docker memory limit
docker-compose.yml:
  services:
    bot:
      deploy:
        resources:
          limits:
            memory: 2G

# For Go, check for goroutine leaks
curl http://localhost:6060/debug/pprof/goroutine?debug=2

# For Python, profile memory
pip install memory_profiler
python -m memory_profiler app/main.py
```

### High CPU usage

**Problem**: Service consuming excessive CPU

**Diagnosis**:
```bash
# Go CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Python CPU profile
python -m cProfile -o output.prof app/main.py
```

**Common Causes**:
1. Infinite loops
2. Excessive polling without sleep
3. Unoptimized database queries
4. ML model inference in tight loop

## Database Issues

### Connection refused

**Problem**: Cannot connect to database

**Solution**:
```bash
# Check if database is running
docker-compose ps

# Check connectivity
pg_isready -h localhost -p 5432

# Check credentials
psql postgresql://postgres:postgres@localhost:5432/clever_better

# Check security groups (AWS)
aws ec2 describe-security-groups --group-ids sg-xxxxx
```

### Slow queries

**Problem**: Database operations taking too long

**Diagnosis**:
```sql
-- Check running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active';

-- Explain slow query
EXPLAIN (ANALYZE, BUFFERS) SELECT ...;

-- Check for missing indexes
SELECT schemaname, relname, seq_scan, seq_tup_read, idx_scan
FROM pg_stat_user_tables
WHERE seq_scan > 100;
```

**Solution**:
```sql
-- Add index
CREATE INDEX CONCURRENTLY idx_odds_race_time
ON odds_snapshots (race_id, time DESC);

-- Update statistics
ANALYZE odds_snapshots;

-- Check table bloat and vacuum
VACUUM ANALYZE odds_snapshots;
```

### Migration failures

**Problem**: Database migrations not applying

**Solution**:
```bash
# Check current version
migrate -path migrations -database "$DB_URL" version

# Force to specific version (dangerous!)
migrate -path migrations -database "$DB_URL" force <version>

# Run migrations with verbose output
migrate -path migrations -database "$DB_URL" up -verbose
```

## Betfair API Issues

### Authentication failures

**Problem**: `INVALID_SESSION_INFORMATION` or certificate errors

**Solution**:
```bash
# Verify certificate files exist
ls -la config/certs/

# Test certificate validity
openssl x509 -in config/certs/client.crt -text -noout

# Test connection manually
curl -X POST \
  -H "X-Application: YOUR_APP_KEY" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --cert config/certs/client.crt \
  --key config/certs/client.key \
  -d "username=YOUR_USER&password=YOUR_PASS" \
  https://identitysso-cert.betfair.com/api/certlogin
```

### Rate limiting

**Problem**: `TOO_MANY_REQUESTS` errors

**Solution**:
```go
// Implement rate limiter
limiter := rate.NewLimiter(rate.Every(time.Second/5), 1)  // 5 req/sec

func (c *Client) Request(ctx context.Context, req *Request) (*Response, error) {
    if err := limiter.Wait(ctx); err != nil {
        return nil, err
    }
    return c.doRequest(ctx, req)
}
```

### Market data not updating

**Problem**: Streaming connection drops or stale data

**Solution**:
```go
// Implement reconnection logic
func (s *Stream) reconnect() {
    backoff := time.Second
    maxBackoff := time.Minute

    for {
        err := s.connect()
        if err == nil {
            return
        }

        log.Warn("stream connection failed, retrying",
            "error", err,
            "backoff", backoff)

        time.Sleep(backoff)
        backoff = min(backoff*2, maxBackoff)
    }
}
```

## ML Service Issues

### Model not loading

**Problem**: `ModelNotLoadedError` or model file not found

**Solution**:
```bash
# Check model file exists
ls -la ml-service/models/

# Check model file integrity
python -c "import joblib; joblib.load('ml-service/models/model.pkl')"

# Check permissions
chmod 644 ml-service/models/*.pkl
```

### Prediction timeout

**Problem**: ML predictions taking too long

**Solution**:
```python
# Profile inference
import time

start = time.time()
prediction = model.predict(features)
print(f"Inference time: {time.time() - start:.3f}s")

# Use lighter model for real-time
# Or batch predictions
predictions = model.predict(batch_features)
```

### Feature mismatch

**Problem**: `ValueError: X has N features, but model expects M features`

**Solution**:
```python
# Check expected features
print(f"Model features: {model.feature_names_in_}")

# Ensure feature order matches
features = features[model.feature_names_in_]

# Handle missing features
for feature in model.feature_names_in_:
    if feature not in features:
        features[feature] = 0.0  # Or appropriate default
```

## Deployment Issues

### ECS task not starting

**Problem**: Tasks in STOPPED state

**Diagnosis**:
```bash
# Get stopped reason
aws ecs describe-tasks \
  --cluster clever-better \
  --tasks <task-arn> \
  --query 'tasks[0].stoppedReason'

# Check CloudWatch logs
aws logs tail /ecs/clever-better-bot --since 1h
```

**Common Causes**:
1. **CannotPullContainerError**: ECR authentication or image not found
2. **ResourceInitializationError**: ENI allocation failure
3. **OutOfMemoryError**: Task memory limit too low
4. **Essential container exited**: Application crash

### Terraform apply fails

**Problem**: Terraform state conflicts or resource errors

**Solution**:
```bash
# Refresh state
terraform refresh

# Import existing resource
terraform import aws_ecs_service.bot arn:aws:ecs:...

# State surgery (careful!)
terraform state rm <resource>
terraform import <resource> <id>

# Unlock state (if stuck)
terraform force-unlock <lock-id>
```

### Secrets not available

**Problem**: Application cannot read secrets

**Solution**:
```bash
# Verify secret exists
aws secretsmanager get-secret-value --secret-id clever-better/prod/betfair

# Check IAM permissions
aws iam simulate-principal-policy \
  --policy-source-arn <task-role-arn> \
  --action-names secretsmanager:GetSecretValue \
  --resource-arns <secret-arn>

# Verify ECS task role
aws ecs describe-task-definition --task-definition clever-better-bot \
  --query 'taskDefinition.taskRoleArn'
```

### Health check failures

**Problem**: ALB health checks failing, tasks draining

**Solution**:
```bash
# Check health endpoint manually
curl -v http://localhost:8000/health

# Check ALB target group health
aws elbv2 describe-target-health \
  --target-group-arn <tg-arn>

# Common fixes:
# 1. Increase health check grace period
# 2. Adjust health check path/port
# 3. Increase deregistration delay
```
