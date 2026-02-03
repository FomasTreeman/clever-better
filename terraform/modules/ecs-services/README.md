# ECS Services Module

This module creates and manages ECS Fargate services with auto-scaling and deployment configuration.

## Features

- ECS Fargate services for bot and ML service
- Deployment circuit breaker with automatic rollback
- Auto-scaling based on CPU, memory, and request count
- Scheduled data ingestion via EventBridge
- CloudWatch alarms for deployment monitoring
- SNS notifications for deployment events

## Usage

```hcl
module "ecs_services" {
  source = "../../modules/ecs-services"

  environment      = var.environment
  ecs_cluster_id   = module.ecs.cluster_id
  ecs_cluster_name = module.ecs.cluster_name
  ecs_cluster_arn  = module.ecs.cluster_arn

  # Task definitions
  bot_task_definition_arn          = module.ecs_tasks.bot_task_definition_arn
  ml_service_task_definition_arn   = module.ecs_tasks.ml_service_task_definition_arn
  data_ingestion_task_definition_arn = module.ecs_tasks.data_ingestion_task_definition_arn

  # Network
  private_subnet_ids             = module.vpc.private_app_subnet_ids
  application_security_group_ids = [module.security.application_security_group_id]

  # Load balancer
  ml_http_target_group_arn = module.alb.ml_http_target_group_arn
  ml_grpc_target_group_arn = module.alb.ml_grpc_target_group_arn

  # Scheduled tasks
  cloudwatch_events_role_arn = module.iam.cloudwatch_events_role_arn
  enable_data_ingestion      = true
  data_ingestion_schedule    = "cron(0 */6 * * ? *)"

  # Scaling
  bot_desired_count = 1
  bot_max_capacity  = 2
  ml_desired_count  = 1
  ml_max_capacity   = 3

  # Debugging
  enable_execute_command = true

  tags = local.tags
}
```

## Blue-Green Deployment

The module uses ECS deployment circuit breaker for automatic rollback:

1. ECS starts new tasks with updated task definition
2. New tasks must pass health checks
3. If health checks fail, circuit breaker triggers rollback
4. Old tasks are drained and stopped after successful deployment

### Deployment Configuration

| Setting | Value |
|---------|-------|
| Maximum percent | 200% |
| Minimum healthy | 100% |
| Health check grace | 60-120s |
| Circuit breaker | Enabled |
| Auto rollback | Enabled |

## Auto-Scaling Policies

### Bot Service

| Metric | Target | Scale Out | Scale In |
|--------|--------|-----------|----------|
| CPU | 70% | 60s cooldown | 300s cooldown |
| Memory | 80% | 60s cooldown | 300s cooldown |

### ML Service

| Metric | Target | Scale Out | Scale In |
|--------|--------|-----------|----------|
| CPU | 60% | 60s cooldown | 300s cooldown |
| Memory | 75% | 60s cooldown | 300s cooldown |
| Requests | 100/target | 60s cooldown | 300s cooldown |

## Environment Configuration

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| **Bot Service** |
| Desired | 1 | 1 | 2 |
| Min | 1 | 1 | 2 |
| Max | 2 | 3 | 5 |
| **ML Service** |
| Desired | 1 | 1 | 2 |
| Min | 1 | 1 | 2 |
| Max | 3 | 5 | 10 |
| **Options** |
| Execute Command | Yes | Yes | No |
| Data Ingestion | Yes | Yes | Yes |

## CloudWatch Alarms

The module creates the following alarms:

- `{env}-bot-high-cpu` - Bot CPU > 90%
- `{env}-bot-high-memory` - Bot memory > 90%
- `{env}-bot-no-running-tasks` - Bot has 0 tasks
- `{env}-ml-high-cpu` - ML CPU > 85%
- `{env}-ml-high-memory` - ML memory > 85%
- `{env}-ml-no-running-tasks` - ML has 0 tasks

## Inputs

| Name | Description | Type | Default |
|------|-------------|------|---------|
| environment | Environment name | string | - |
| ecs_cluster_id | ECS cluster ID | string | - |
| ecs_cluster_name | ECS cluster name | string | - |
| bot_task_definition_arn | Bot task ARN | string | - |
| ml_service_task_definition_arn | ML task ARN | string | - |
| private_subnet_ids | Subnet IDs | list(string) | - |
| bot_desired_count | Bot desired count | number | 1 |
| bot_max_capacity | Bot max capacity | number | 2 |
| ml_desired_count | ML desired count | number | 1 |
| ml_max_capacity | ML max capacity | number | 3 |
| enable_execute_command | Enable ECS Exec | bool | false |

## Outputs

| Name | Description |
|------|-------------|
| bot_service_id | Bot service ID |
| bot_service_name | Bot service name |
| ml_service_id | ML service ID |
| ml_service_name | ML service name |
| bot_autoscaling_target_id | Bot scaling target |
| ml_autoscaling_target_id | ML scaling target |
| data_ingestion_rule_arn | EventBridge rule ARN |

## Troubleshooting

### Service Won't Start

1. Check task definition configuration
2. Verify ECR image exists
3. Check security group rules
4. Review CloudWatch logs

### Deployment Rollback

1. Check deployment events in ECS console
2. Review task stopped reasons
3. Verify health check endpoint
4. Check container exit codes

### Scaling Issues

1. Verify auto-scaling target is registered
2. Check CloudWatch metrics
3. Review scaling policy configuration
4. Check service quotas
