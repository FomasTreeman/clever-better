# =============================================================================
# ECS Services Module - Service Definitions and Auto-Scaling
# Provides ECS Fargate services with deployment and scaling configuration
# =============================================================================

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    var.tags,
    {
      Environment = var.environment
      ManagedBy   = "terraform"
      Module      = "ecs-services"
    }
  )
}

# Data sources
data "aws_region" "current" {}

# =============================================================================
# Bot Service
# =============================================================================

resource "aws_ecs_service" "bot" {
  name             = "${local.name_prefix}-bot"
  cluster          = var.ecs_cluster_id
  task_definition  = var.bot_task_definition_arn
  desired_count    = var.bot_desired_count
  launch_type      = "FARGATE"
  platform_version = "LATEST"

  deployment_maximum_percent         = 200
  deployment_minimum_healthy_percent = 100
  health_check_grace_period_seconds  = 60

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = var.application_security_group_ids
    assign_public_ip = false
  }

  enable_ecs_managed_tags = true
  propagate_tags          = "SERVICE"
  enable_execute_command  = var.enable_execute_command

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-service"
      Service = "bot"
    }
  )

  lifecycle {
    ignore_changes = [desired_count]
  }
}

# =============================================================================
# ML Service
# =============================================================================

resource "aws_ecs_service" "ml_service" {
  name             = "${local.name_prefix}-ml-service"
  cluster          = var.ecs_cluster_id
  task_definition  = var.ml_service_task_definition_arn
  desired_count    = var.ml_desired_count
  launch_type      = "FARGATE"
  platform_version = "LATEST"

  deployment_maximum_percent         = 200
  deployment_minimum_healthy_percent = 100
  health_check_grace_period_seconds  = 120

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = var.application_security_group_ids
    assign_public_ip = false
  }

  # HTTP target group
  load_balancer {
    target_group_arn = var.ml_http_target_group_arn
    container_name   = "ml-service"
    container_port   = 8000
  }

  # gRPC target group
  load_balancer {
    target_group_arn = var.ml_grpc_target_group_arn
    container_name   = "ml-service"
    container_port   = 50051
  }

  enable_ecs_managed_tags = true
  propagate_tags          = "SERVICE"
  enable_execute_command  = var.enable_execute_command

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-service"
      Service = "ml-service"
    }
  )

  lifecycle {
    ignore_changes = [desired_count]
  }
}

# =============================================================================
# Auto Scaling - Bot Service
# =============================================================================

resource "aws_appautoscaling_target" "bot" {
  service_namespace  = "ecs"
  resource_id        = "service/${var.ecs_cluster_name}/${aws_ecs_service.bot.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.bot_min_capacity
  max_capacity       = var.bot_max_capacity

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-scaling"
      Service = "bot"
    }
  )
}

resource "aws_appautoscaling_policy" "bot_cpu" {
  name               = "${local.name_prefix}-bot-cpu"
  service_namespace  = aws_appautoscaling_target.bot.service_namespace
  resource_id        = aws_appautoscaling_target.bot.resource_id
  scalable_dimension = aws_appautoscaling_target.bot.scalable_dimension
  policy_type        = "TargetTrackingScaling"

  target_tracking_scaling_policy_configuration {
    target_value       = var.bot_cpu_target
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

resource "aws_appautoscaling_policy" "bot_memory" {
  name               = "${local.name_prefix}-bot-memory"
  service_namespace  = aws_appautoscaling_target.bot.service_namespace
  resource_id        = aws_appautoscaling_target.bot.resource_id
  scalable_dimension = aws_appautoscaling_target.bot.scalable_dimension
  policy_type        = "TargetTrackingScaling"

  target_tracking_scaling_policy_configuration {
    target_value       = var.bot_memory_target
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
  }
}

# =============================================================================
# Auto Scaling - ML Service
# =============================================================================

resource "aws_appautoscaling_target" "ml_service" {
  service_namespace  = "ecs"
  resource_id        = "service/${var.ecs_cluster_name}/${aws_ecs_service.ml_service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.ml_min_capacity
  max_capacity       = var.ml_max_capacity

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-scaling"
      Service = "ml-service"
    }
  )
}

resource "aws_appautoscaling_policy" "ml_cpu" {
  name               = "${local.name_prefix}-ml-cpu"
  service_namespace  = aws_appautoscaling_target.ml_service.service_namespace
  resource_id        = aws_appautoscaling_target.ml_service.resource_id
  scalable_dimension = aws_appautoscaling_target.ml_service.scalable_dimension
  policy_type        = "TargetTrackingScaling"

  target_tracking_scaling_policy_configuration {
    target_value       = var.ml_cpu_target
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

resource "aws_appautoscaling_policy" "ml_memory" {
  name               = "${local.name_prefix}-ml-memory"
  service_namespace  = aws_appautoscaling_target.ml_service.service_namespace
  resource_id        = aws_appautoscaling_target.ml_service.resource_id
  scalable_dimension = aws_appautoscaling_target.ml_service.scalable_dimension
  policy_type        = "TargetTrackingScaling"

  target_tracking_scaling_policy_configuration {
    target_value       = var.ml_memory_target
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
  }
}

# Request count based scaling for ML service
resource "aws_appautoscaling_policy" "ml_request_count" {
  count = var.ml_http_target_group_arn != "" ? 1 : 0

  name               = "${local.name_prefix}-ml-requests"
  service_namespace  = aws_appautoscaling_target.ml_service.service_namespace
  resource_id        = aws_appautoscaling_target.ml_service.resource_id
  scalable_dimension = aws_appautoscaling_target.ml_service.scalable_dimension
  policy_type        = "TargetTrackingScaling"

  target_tracking_scaling_policy_configuration {
    target_value       = var.ml_request_count_target
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ALBRequestCountPerTarget"
      resource_label         = var.alb_arn_suffix != "" ? "${var.alb_arn_suffix}/${var.ml_target_group_arn_suffix}" : ""
    }
  }
}

# =============================================================================
# EventBridge Rule for Data Ingestion
# =============================================================================

resource "aws_cloudwatch_event_rule" "data_ingestion" {
  count = var.enable_data_ingestion ? 1 : 0

  name                = "${local.name_prefix}-data-ingestion"
  description         = "Scheduled data ingestion task"
  schedule_expression = var.data_ingestion_schedule
  state               = "ENABLED"

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-data-ingestion-rule"
      Service = "data-ingestion"
    }
  )
}

resource "aws_cloudwatch_event_target" "data_ingestion" {
  count = var.enable_data_ingestion ? 1 : 0

  rule      = aws_cloudwatch_event_rule.data_ingestion[0].name
  target_id = "${local.name_prefix}-data-ingestion"
  arn       = var.ecs_cluster_arn
  role_arn  = var.cloudwatch_events_role_arn

  ecs_target {
    task_definition_arn = var.data_ingestion_task_definition_arn
    task_count          = 1
    launch_type         = "FARGATE"
    platform_version    = "LATEST"

    network_configuration {
      subnets          = var.private_subnet_ids
      security_groups  = var.application_security_group_ids
      assign_public_ip = false
    }
  }
}
