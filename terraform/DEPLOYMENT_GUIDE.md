# Deployment Guide

This guide covers the deployment workflow for the Clever Better trading system using ECS Fargate.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Infrastructure Setup](#infrastructure-setup)
3. [Container Image Management](#container-image-management)
4. [Deployment Workflow](#deployment-workflow)
5. [Blue-Green Deployment](#blue-green-deployment)
6. [Rollback Procedures](#rollback-procedures)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

Before deploying, ensure you have:

- AWS CLI configured with appropriate credentials
- Docker installed and running
- Terraform >= 1.0.0
- jq for JSON processing
- Access to the target AWS account

## Infrastructure Setup

### 1. Setup Terraform Backend

```bash
# Create S3 bucket and DynamoDB table for state management
./terraform/scripts/setup-backend.sh
```

Follow the prompts to select environment and region.

### 2. Initialize Terraform

```bash
# For dev environment
cd terraform/environments/dev
terraform init
```

### 3. Apply Infrastructure

```bash
# Copy and configure tfvars
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Plan changes
terraform plan

# Apply changes
terraform apply
```

## Container Image Management

### ECR Repositories

Three repositories are created per environment:
- `clever-better-{env}-bot` - Trading bot
- `clever-better-{env}-ml-service` - ML prediction service
- `clever-better-{env}-data-ingestion` - Data collection

### Image Tagging Strategy

| Environment | Tag Pattern | Mutability |
|-------------|-------------|------------|
| dev | `latest`, feature branches | MUTABLE |
| staging | `latest`, release candidates | MUTABLE |
| production | Semantic versions (`v1.2.3`) | IMMUTABLE |

### Building and Pushing Images

```bash
# Login to ECR
make ecr-login

# Build and push all services
make ecr-push-all ENV=dev TAG=latest

# Build and push specific service
make ecr-push-bot ENV=dev TAG=v1.0.0
make ecr-push-ml ENV=staging TAG=v1.0.0
```

## Deployment Workflow

### Standard Deployment

```bash
# Deploy all services to dev
make deploy-dev TAG=latest

# Deploy specific service
make deploy-service ENV=staging SERVICE=bot TAG=v1.2.3

# Deploy to production
make deploy-prod SERVICE=all TAG=v1.0.0
```

### Deployment Process

1. **Build Phase**
   - Docker image built from Dockerfile
   - Image tagged with ECR repository URL
   - Image pushed to ECR

2. **Task Definition Update**
   - New task definition revision created
   - Container image URI updated

3. **Service Update**
   - ECS service updated with new task definition
   - Rolling deployment starts

4. **Health Checks**
   - New tasks start and pass health checks
   - Old tasks drained and stopped

### Validating Deployments

```bash
# Validate dev deployment
make validate-dev

# Validate production
make validate-prod
```

## Blue-Green Deployment

The system uses ECS deployment circuit breaker for automatic rollback capability.

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Deployment Process                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. New Task Definition Created                              │
│     └─> Container image updated                              │
│                                                              │
│  2. ECS Starts New Tasks (Blue)                              │
│     └─> Running alongside existing tasks (Green)             │
│                                                              │
│  3. Health Check Phase                                       │
│     ├─> ALB health checks pass                               │
│     └─> Container health checks pass                         │
│                                                              │
│  4. Traffic Shift                                            │
│     └─> ALB routes traffic to new tasks                      │
│                                                              │
│  5. Old Tasks Drained (Green)                                │
│     └─> Connections drained, tasks stopped                   │
│                                                              │
│  [If health checks fail]                                     │
│     └─> Circuit breaker triggers automatic rollback          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Deployment Configuration

| Setting | Value |
|---------|-------|
| Maximum Percent | 200% |
| Minimum Healthy Percent | 100% |
| Health Check Grace Period | 60-120s |
| Deployment Circuit Breaker | Enabled |
| Auto Rollback | Enabled |

## Rollback Procedures

### Automatic Rollback

The deployment circuit breaker automatically rolls back when:
- Task health checks fail
- Tasks fail to start
- Application health endpoints fail

### Manual Rollback

```bash
# Rollback specific service
make rollback-dev SERVICE=bot
make rollback-staging SERVICE=ml-service
make rollback-prod SERVICE=bot
```

### Manual Rollback Steps

1. Identify the previous task definition revision:
   ```bash
   aws ecs describe-services \
     --cluster clever-better-dev \
     --services clever-better-dev-bot \
     --query 'services[0].taskDefinition'
   ```

2. Update service to previous revision:
   ```bash
   aws ecs update-service \
     --cluster clever-better-dev \
     --service clever-better-dev-bot \
     --task-definition clever-better-dev-bot:N-1
   ```

3. Wait for stabilization:
   ```bash
   aws ecs wait services-stable \
     --cluster clever-better-dev \
     --services clever-better-dev-bot
   ```

## Troubleshooting

### Task Fails to Start

**Symptoms:** Tasks stuck in PENDING or STOPPED state

**Check:**
1. CloudWatch Logs for container errors
   ```bash
   aws logs tail /ecs/clever-better-dev/bot --follow
   ```

2. Task stopped reason
   ```bash
   aws ecs describe-tasks \
     --cluster clever-better-dev \
     --tasks <task-arn> \
     --query 'tasks[0].stoppedReason'
   ```

3. Common causes:
   - ECR image not found
   - Secrets Manager access denied
   - Resource limits exceeded
   - Container exit code non-zero

### Health Check Failures

**Symptoms:** Service deployment fails, tasks unhealthy

**Check:**
1. Verify health endpoint works locally
2. Check security group allows health check port
3. Increase health check grace period
4. Review container startup time

### Secrets Access Denied

**Symptoms:** Container fails to start with secrets error

**Check:**
1. Verify execution role has `secretsmanager:GetSecretValue`
2. Check secret ARN format in task definition
3. Ensure secrets exist in correct region

### Out of Memory

**Symptoms:** Tasks killed with OOM error

**Resolution:**
1. Increase memory in task definition
2. Monitor memory usage in CloudWatch
3. Optimize application memory usage

## Environment-Specific Notes

### Dev
- Fargate Spot enabled for cost savings
- ECS Exec enabled for debugging
- Single task count for services

### Staging
- Standard Fargate for reliability
- ECS Exec enabled for debugging
- Production-like configuration

### Production
- Standard Fargate
- ECS Exec disabled for security
- Multi-task deployment for HA
- Immutable image tags enforced
