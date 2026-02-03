# ECS Module

This module creates an Amazon ECS Fargate cluster for running containerized applications.

## Features

- **AWS Fargate** for serverless container orchestration
- **Fargate Spot** support for cost optimization (optional)
- **Container Insights** for comprehensive monitoring
- **CloudWatch Logs integration** for centralized logging
- **Multiple capacity provider strategies** for flexibility
- **Auto-scaling capabilities** (configured at service level)

## Architecture

```
┌──────────────────────────────────────────┐
│        ECS Fargate Cluster               │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │   Capacity Providers              │  │
│  │  ┌──────────┐    ┌──────────────┐ │  │
│  │  │ FARGATE  │    │ FARGATE_SPOT │ │  │
│  │  │ (Base)   │    │  (Optional)  │ │  │
│  │  └──────────┘    └──────────────┘ │  │
│  └────────────────────────────────────┘  │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │    Container Insights              │  │
│  │   (CloudWatch Monitoring)          │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
              │
              ▼
      ┌──────────────┐
      │  CloudWatch  │
      │  Log Groups  │
      └──────────────┘
```

## Usage

```hcl
module "ecs" {
  source = "../../modules/ecs"

  environment             = "production"
  project_name            = "clever-better"
  
  # Monitoring
  enable_container_insights = true
  log_retention_days        = 90
  
  # Cost optimization (disable in production)
  enable_fargate_spot = false
  
  tags = {
    Project   = "clever-better"
    ManagedBy = "terraform"
  }
}
```

## Container Insights

When enabled, Container Insights provides:

- **Cluster-level metrics**: CPU, memory, network, storage
- **Service-level metrics**: Running tasks, pending tasks, deployments
- **Task-level metrics**: Per-task resource utilization
- **Performance monitoring dashboard** in CloudWatch
- **Automatic alarms** for resource utilization

### Viewing Container Insights

```bash
# Open CloudWatch Container Insights in AWS Console
# Navigate to: CloudWatch > Container Insights > Performance monitoring

# Query metrics via CLI
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ClusterName,Value=<cluster_name> \
  --start-time <start> \
  --end-time <end> \
  --period 300 \
  --statistics Average
```

## Capacity Providers

### FARGATE (Default)

- **On-demand pricing**: Pay for vCPU and memory per second
- **Guaranteed availability**: No spot interruptions
- **Base capacity**: Minimum number of tasks always on Fargate
- **Use cases**: Production workloads, critical services

### FARGATE_SPOT (Optional)

- **Spot pricing**: Up to 70% cost savings
- **Interruption possible**: Tasks may be stopped with 2-minute notice
- **Weight-based strategy**: 80% of tasks on Spot when enabled (weight=4)
- **Use cases**: Dev/staging, fault-tolerant workloads, batch jobs

### Capacity Provider Strategy

When Fargate Spot is enabled:
```hcl
default_capacity_provider_strategy {
  capacity_provider = "FARGATE"
  weight            = 1
  base              = 1  # At least 1 task on on-demand
}

default_capacity_provider_strategy {
  capacity_provider = "FARGATE_SPOT"
  weight            = 4  # 80% of remaining tasks on Spot
  base              = 0
}
```

## CloudWatch Logs

### Log Groups

- **Cluster logs**: `/ecs/<project>-<environment>`
- **Task logs**: Configured at task definition level
- **Retention**: Configurable (default 90 days)

### Log Aggregation

```bash
# View logs for a specific task
aws logs tail /ecs/clever-better-production \
  --follow \
  --filter-pattern "ERROR"

# Query logs across all tasks
aws logs start-query \
  --log-group-name /ecs/clever-better-production \
  --start-time $(date -u -d '1 hour ago' +%s) \
  --end-time $(date -u +%s) \
  --query-string 'fields @timestamp, @message | filter @message like /ERROR/ | sort @timestamp desc'
```

## Task Definitions and Services

**Note**: This module creates only the ECS cluster. Task definitions and services should be created separately using the `ecs-tasks` module (future implementation) or directly in your application-specific Terraform configurations.

### Example Task Definition

```hcl
resource "aws_ecs_task_definition" "app" {
  family                   = "my-app"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = <ecs_execution_role_arn>
  task_role_arn            = <ecs_task_role_arn>

  container_definitions = jsonencode([{
    name  = "app"
    image = "my-app:latest"
    
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = module.ecs.log_group_name
        awslogs-region        = "us-east-1"
        awslogs-stream-prefix = "app"
      }
    }
  }])
}
```

### Example ECS Service

```hcl
resource "aws_ecs_service" "app" {
  name            = "my-app-service"
  cluster         = module.ecs.cluster_id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = 2
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = <private_subnet_ids>
    security_groups  = <security_group_ids>
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = <target_group_arn>
    container_name   = "app"
    container_port   = 8080
  }
}
```

## Cost Optimization

### Development/Staging

```hcl
enable_container_insights = false  # Save on CloudWatch costs
enable_fargate_spot       = true   # 70% savings on compute
log_retention_days        = 30     # Shorter retention
```

**Estimated Cost**: ~$50/month for 2 small tasks

### Production

```hcl
enable_container_insights = true   # Full monitoring
enable_fargate_spot       = false  # Guaranteed availability
log_retention_days        = 90     # Longer retention for compliance
```

**Estimated Cost**: ~$150/month for 4 medium tasks

### Cost-Saving Tips

1. **Right-size tasks**: Don't over-provision CPU/memory
2. **Use Fargate Spot**: For non-critical workloads (70% savings)
3. **Optimize log retention**: Balance compliance with storage costs
4. **Schedule tasks**: Stop dev/staging services overnight
5. **Use service auto-scaling**: Scale down during low traffic

## Monitoring and Alerting

### Recommended CloudWatch Alarms

```hcl
# High CPU utilization
resource "aws_cloudwatch_metric_alarm" "ecs_cpu_high" {
  alarm_name          = "ecs-cpu-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "This metric monitors ECS CPU utilization"

  dimensions = {
    ClusterName = module.ecs.cluster_name
  }
}

# High memory utilization
resource "aws_cloudwatch_metric_alarm" "ecs_memory_high" {
  alarm_name          = "ecs-memory-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 85
  alarm_description   = "This metric monitors ECS memory utilization"

  dimensions = {
    ClusterName = module.ecs.cluster_name
  }
}
```

## Security Best Practices

1. **Use task roles**: Grant minimal IAM permissions per task
2. **Use secrets**: Store sensitive data in Secrets Manager/Parameter Store
3. **Network isolation**: Deploy tasks in private subnets
4. **Security groups**: Restrict inbound/outbound traffic
5. **Image scanning**: Enable ECR vulnerability scanning
6. **Read-only root filesystem**: When possible
7. **Non-root user**: Run containers as non-root

## Troubleshooting

### Task Fails to Start

1. Check CloudWatch Logs for errors
2. Verify task execution role has ECR pull permissions
3. Ensure security groups allow required traffic
4. Check subnet has available IP addresses
5. Verify task resource limits (CPU/memory)

### High Costs

1. Review task sizing (CPU/memory allocation)
2. Check for idle tasks (scale down if not needed)
3. Enable Fargate Spot for non-production
4. Optimize log retention periods
5. Review Container Insights necessity

### Performance Issues

1. Check Container Insights for resource bottlenecks
2. Review task placement strategy
3. Analyze CloudWatch Logs for application errors
4. Consider increasing task resources
5. Enable auto-scaling for dynamic load

## Inputs

| Name | Type | Default | Description |
|------|------|---------|-------------|
| environment | string | required | Environment name (dev, staging, production) |
| project_name | string | "clever-better" | Project name for resource naming |
| enable_container_insights | bool | true | Enable CloudWatch Container Insights |
| log_retention_days | number | 90 | CloudWatch logs retention in days |
| enable_fargate_spot | bool | false | Enable Fargate Spot capacity provider |
| tags | map(string) | {} | Additional resource tags |

## Outputs

| Name | Description |
|------|-------------|
| cluster_id | ECS cluster ID |
| cluster_name | ECS cluster name |
| cluster_arn | ECS cluster ARN |
| log_group_name | CloudWatch log group name |
| log_group_arn | CloudWatch log group ARN |

## References

- [Amazon ECS](https://aws.amazon.com/ecs/)
- [AWS Fargate](https://aws.amazon.com/fargate/)
- [Fargate Spot](https://aws.amazon.com/fargate/pricing/)
- [Container Insights](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html)
- [ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/intro.html)
