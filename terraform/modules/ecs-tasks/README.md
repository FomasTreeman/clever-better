# ECS Tasks Module

This module creates ECS Fargate task definitions for all Clever Better services.

## Services

| Service | Description | Ports |
|---------|-------------|-------|
| bot | Trading bot execution engine | 8080 (health) |
| ml-service | Machine learning prediction service | 8000 (HTTP), 50051 (gRPC) |
| data-ingestion | Scheduled data collection | None (batch job) |

## Usage

```hcl
module "ecs_tasks" {
  source = "../../modules/ecs-tasks"

  environment            = var.environment
  ecs_execution_role_arn = module.iam.ecs_task_execution_role_arn
  bot_task_role_arn      = module.iam.bot_task_role_arn
  ml_task_role_arn       = module.iam.ml_service_task_role_arn

  # Image configuration
  bot_image_url            = module.ecr.repository_urls["bot"]
  ml_image_url             = module.ecr.repository_urls["ml-service"]
  data_ingestion_image_url = module.ecr.repository_urls["data-ingestion"]
  bot_image_tag            = var.bot_image_tag
  ml_image_tag             = var.ml_image_tag

  # Secrets
  database_secret_arn = module.secrets.database_secret_arn
  betfair_secret_arn  = module.secrets.betfair_secret_arn

  log_retention_days = 30
  tags               = local.tags
}
```

## Task Definition Structure

### Container Resources

| Service | Dev | Staging | Production |
|---------|-----|---------|------------|
| **bot** |
| CPU | 512 | 512 | 1024 |
| Memory | 1024 MB | 1024 MB | 2048 MB |
| **ml-service** |
| CPU | 1024 | 1024 | 2048 |
| Memory | 2048 MB | 2048 MB | 4096 MB |
| **data-ingestion** |
| CPU | 512 | 512 | 512 |
| Memory | 1024 MB | 1024 MB | 1024 MB |

### Secrets Injection

Secrets are injected from AWS Secrets Manager at container startup:

- `DATABASE_URL` - Full database connection string
- `BETFAIR_API_KEY` - Betfair API authentication
- `BETFAIR_USERNAME` - Betfair account username
- `BETFAIR_PASSWORD` - Betfair account password

### Health Checks

All services implement health check endpoints:

- **bot**: `GET http://localhost:8080/health`
- **ml-service**: `GET http://localhost:8000/health`
- **data-ingestion**: No health check (batch job)

Health check configuration:
- Interval: 30 seconds
- Timeout: 5 seconds
- Retries: 3
- Start period: 60 seconds (120 for ML service)

## Inputs

| Name | Description | Type | Default |
|------|-------------|------|---------|
| environment | Environment name | string | - |
| ecs_execution_role_arn | ECS execution role | string | - |
| bot_task_role_arn | Bot task role | string | - |
| ml_task_role_arn | ML task role | string | - |
| bot_image_url | Bot image URL | string | - |
| ml_image_url | ML image URL | string | - |
| bot_cpu | Bot CPU units | number | 512 |
| bot_memory | Bot memory (MB) | number | 1024 |
| ml_cpu | ML CPU units | number | 1024 |
| ml_memory | ML memory (MB) | number | 2048 |
| log_retention_days | Log retention | number | 30 |

## Outputs

| Name | Description |
|------|-------------|
| bot_task_definition_arn | Bot task definition ARN |
| ml_service_task_definition_arn | ML service task definition ARN |
| data_ingestion_task_definition_arn | Data ingestion task definition ARN |
| bot_task_family | Bot task family name |
| bot_log_group_name | Bot log group name |

## Troubleshooting

### Task Fails to Start

1. Check CloudWatch logs for container errors
2. Verify secrets exist and have correct format
3. Ensure ECR image exists and is accessible
4. Check task role permissions

### Health Check Failures

1. Verify health endpoint responds correctly
2. Increase start period for slow-starting containers
3. Check container resource limits

### Secrets Access Errors

1. Verify execution role has `secretsmanager:GetSecretValue` permission
2. Check secret ARN format in task definition
3. Ensure secrets exist in the correct region
