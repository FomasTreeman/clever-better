# Deployment Guide

This document describes the deployment procedures for Clever Better, including environment setup, deployment steps, and operational runbooks.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Deployment Procedures](#deployment-procedures)
- [Rollback Procedures](#rollback-procedures)
- [Operational Runbooks](#operational-runbooks)

## Overview

Clever Better is deployed on AWS using the following stack:

- **Compute**: ECS Fargate
- **Database**: RDS (TimescaleDB)
- **Infrastructure**: Terraform
- **CI/CD**: GitHub Actions

### Deployment Environments

| Environment | Purpose | AWS Account |
|-------------|---------|-------------|
| Development | Local development, feature testing | N/A (local) |
| Staging | Integration testing, UAT | staging-account |
| Production | Live trading | production-account |

## Prerequisites

### Required Tools

```bash
# AWS CLI v2
aws --version  # >= 2.0.0

# Terraform
terraform --version  # >= 1.5.0

# Docker
docker --version  # >= 24.0.0

# kubectl (optional, for debugging)
kubectl version --client
```

### AWS Permissions

Deployer needs the following IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:*",
        "ecr:*",
        "rds:*",
        "ec2:*",
        "elasticloadbalancing:*",
        "secretsmanager:*",
        "logs:*",
        "cloudwatch:*",
        "iam:PassRole"
      ],
      "Resource": "*"
    }
  ]
}
```

### Environment Variables

```bash
export AWS_PROFILE=clever-better-deploy
export AWS_REGION=eu-west-1
export TF_VAR_environment=staging
```

## Environment Configuration

### Staging Environment

```hcl
# terraform/environments/staging/terraform.tfvars

environment = "staging"
aws_region  = "eu-west-1"

# ECS Configuration
bot_cpu    = 512
bot_memory = 1024
bot_count  = 1

ml_cpu    = 1024
ml_memory = 2048
ml_count  = 1

# RDS Configuration
db_instance_class    = "db.t3.medium"
db_allocated_storage = 50
db_multi_az         = false

# Feature Flags
live_trading_enabled = false
```

### Production Environment

```hcl
# terraform/environments/production/terraform.tfvars

environment = "production"
aws_region  = "eu-west-1"

# ECS Configuration
bot_cpu    = 512
bot_memory = 1024
bot_count  = 1

ml_cpu    = 2048
ml_memory = 4096
ml_count  = 2

# RDS Configuration
db_instance_class    = "db.r6g.large"
db_allocated_storage = 100
db_multi_az         = true

# Feature Flags
live_trading_enabled = true
```

## Deployment Procedures

### Initial Infrastructure Setup

First-time setup for a new environment:

```bash
# 1. Initialize Terraform
cd terraform/environments/staging
terraform init

# 2. Review the plan
terraform plan -out=tfplan

# 3. Apply infrastructure
terraform apply tfplan

# 4. Store outputs
terraform output -json > ../../../config/terraform-outputs.json
```

### Application Deployment

#### Step 1: Build and Push Docker Images

```bash
# Authenticate with ECR
aws ecr get-login-password --region eu-west-1 | \
  docker login --username AWS --password-stdin 123456789.dkr.ecr.eu-west-1.amazonaws.com

# Build images
make docker-build

# Tag images
docker tag clever-better-bot:latest \
  123456789.dkr.ecr.eu-west-1.amazonaws.com/clever-better-bot:v1.2.3

docker tag clever-better-ml:latest \
  123456789.dkr.ecr.eu-west-1.amazonaws.com/clever-better-ml:v1.2.3

# Push images
docker push 123456789.dkr.ecr.eu-west-1.amazonaws.com/clever-better-bot:v1.2.3
docker push 123456789.dkr.ecr.eu-west-1.amazonaws.com/clever-better-ml:v1.2.3
```

#### Step 2: Update Task Definitions

```bash
# Update bot task definition
aws ecs register-task-definition \
  --cli-input-json file://deployments/ecs/bot-task-definition.json

# Update ML service task definition
aws ecs register-task-definition \
  --cli-input-json file://deployments/ecs/ml-task-definition.json
```

#### Step 3: Deploy Services

```bash
# Deploy bot service
aws ecs update-service \
  --cluster clever-better-staging \
  --service bot \
  --task-definition clever-better-bot:latest \
  --force-new-deployment

# Deploy ML service
aws ecs update-service \
  --cluster clever-better-staging \
  --service ml-service \
  --task-definition clever-better-ml:latest \
  --force-new-deployment
```

#### Step 4: Monitor Deployment

```bash
# Watch service deployment
aws ecs wait services-stable \
  --cluster clever-better-staging \
  --services bot ml-service

# Check task status
aws ecs describe-tasks \
  --cluster clever-better-staging \
  --tasks $(aws ecs list-tasks --cluster clever-better-staging --query 'taskArns[*]' --output text)
```

### Database Migrations

```bash
# Run migrations
migrate -path migrations \
  -database "postgresql://user:pass@hostname:5432/clever_better?sslmode=require" \
  up

# Verify migrations
migrate -path migrations \
  -database "postgresql://user:pass@hostname:5432/clever_better?sslmode=require" \
  version
```

### Secrets Update

```bash
# Update Betfair credentials
aws secretsmanager update-secret \
  --secret-id clever-better/staging/betfair \
  --secret-string '{"app_key":"xxx","username":"xxx","password":"xxx"}'

# Update database credentials
aws secretsmanager update-secret \
  --secret-id clever-better/staging/database \
  --secret-string '{"username":"xxx","password":"xxx"}'
```

## Rollback Procedures

### Quick Rollback (Previous Task Definition)

```bash
# Get previous task definition
PREVIOUS_TASK=$(aws ecs describe-services \
  --cluster clever-better-staging \
  --services bot \
  --query 'services[0].deployments[1].taskDefinition' \
  --output text)

# Rollback to previous version
aws ecs update-service \
  --cluster clever-better-staging \
  --service bot \
  --task-definition $PREVIOUS_TASK \
  --force-new-deployment
```

### Infrastructure Rollback

```bash
# Restore from Terraform state backup
cd terraform/environments/staging

# Get previous state version
aws s3 cp s3://terraform-state/clever-better/staging/terraform.tfstate.backup \
  terraform.tfstate

# Apply previous state
terraform apply
```

### Database Rollback

```bash
# Rollback last migration
migrate -path migrations \
  -database "postgresql://..." \
  down 1

# Or rollback to specific version
migrate -path migrations \
  -database "postgresql://..." \
  goto 20240115120000
```

## Operational Runbooks

### Runbook: Service Not Starting

**Symptoms:**
- ECS tasks failing to start
- Container health checks failing

**Investigation:**
```bash
# 1. Check task stopped reason
aws ecs describe-tasks \
  --cluster clever-better-staging \
  --tasks <task-arn> \
  --query 'tasks[0].stoppedReason'

# 2. Check CloudWatch logs
aws logs tail /ecs/clever-better-bot --follow

# 3. Check container exit code
aws ecs describe-tasks \
  --cluster clever-better-staging \
  --tasks <task-arn> \
  --query 'tasks[0].containers[*].exitCode'
```

**Common Solutions:**
1. **Memory exceeded**: Increase task memory allocation
2. **Secret not found**: Verify secrets exist in Secrets Manager
3. **Database connection failed**: Check security groups, DB status

### Runbook: High Latency

**Symptoms:**
- API response times > 1s
- ML prediction timeout

**Investigation:**
```bash
# 1. Check service metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=ml-service \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 300 \
  --statistics Average

# 2. Check database performance
# Connect to RDS and run:
# SELECT * FROM pg_stat_activity WHERE state = 'active';

# 3. Check ALB metrics for backend response time
```

**Common Solutions:**
1. **High CPU**: Scale out service (increase desired count)
2. **Slow queries**: Add database indexes, optimize queries
3. **Cold start**: Increase minimum task count

### Runbook: Trading Bot Stopped Trading

**Symptoms:**
- No new trades in database
- Bot logs show no market activity

**Investigation:**
```bash
# 1. Check bot logs
aws logs tail /ecs/clever-better-bot --filter-pattern "ERROR" --since 1h

# 2. Check Betfair API connectivity
# Look for authentication errors or rate limiting

# 3. Check if trading is disabled
aws secretsmanager get-secret-value \
  --secret-id clever-better/staging/config \
  --query 'SecretString' | jq '.live_trading_enabled'
```

**Common Solutions:**
1. **Betfair session expired**: Restart bot to re-authenticate
2. **Trading disabled**: Enable in configuration
3. **No markets**: Check market schedule, market filters

### Runbook: Emergency Stop

**Immediate actions to stop all trading:**

```bash
# 1. Scale bot to zero
aws ecs update-service \
  --cluster clever-better-production \
  --service bot \
  --desired-count 0

# 2. Disable trading flag
aws secretsmanager update-secret \
  --secret-id clever-better/production/config \
  --secret-string '{"live_trading_enabled": false}'

# 3. Cancel open bets (via Betfair API)
# Run emergency cancellation script
./scripts/emergency-cancel-bets.sh

# 4. Notify on-call team
# Automated via CloudWatch alarm
```

### Health Check Endpoints

| Service | Endpoint | Expected Response |
|---------|----------|-------------------|
| Bot | `/health` | `{"status": "healthy"}` |
| ML Service | `/health` | `{"status": "healthy"}` |
| ALB | `/health` | 200 OK |
