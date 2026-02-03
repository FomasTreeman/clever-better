# =============================================================================
# ECS Tasks Module - Task Definitions
# Provides ECS Fargate task definitions for all services
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
      Module      = "ecs-tasks"
    }
  )

  # Common environment variables
  common_env_vars = [
    {
      name  = "APP_ENV"
      value = var.environment
    },
    {
      name  = "AWS_REGION"
      value = data.aws_region.current.name
    }
  ]
}

# Data sources
data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

# =============================================================================
# CloudWatch Log Groups
# =============================================================================

resource "aws_cloudwatch_log_group" "bot" {
  name              = "/ecs/${local.name_prefix}/bot"
  retention_in_days = var.log_retention_days

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-logs"
      Service = "bot"
    }
  )
}

resource "aws_cloudwatch_log_group" "ml_service" {
  name              = "/ecs/${local.name_prefix}/ml-service"
  retention_in_days = var.log_retention_days

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-service-logs"
      Service = "ml-service"
    }
  )
}

resource "aws_cloudwatch_log_group" "data_ingestion" {
  name              = "/ecs/${local.name_prefix}/data-ingestion"
  retention_in_days = var.log_retention_days

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-data-ingestion-logs"
      Service = "data-ingestion"
    }
  )
}

# =============================================================================
# Bot Task Definition
# =============================================================================

resource "aws_ecs_task_definition" "bot" {
  family                   = "${local.name_prefix}-bot"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.bot_cpu
  memory                   = var.bot_memory
  task_role_arn            = var.bot_task_role_arn
  execution_role_arn       = var.ecs_execution_role_arn

  container_definitions = jsonencode([
    {
      name      = "bot"
      image     = "${var.bot_image_url}:${var.bot_image_tag}"
      essential = true

      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
          name          = "health"
        }
      ]

      environment = concat(
        local.common_env_vars,
        [
          {
            name  = "SERVICE_NAME"
            value = "bot"
          },
          {
            name  = "HEALTH_PORT"
            value = "8080"
          }
        ]
      )

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = var.database_secret_arn
        },
        {
          name      = "BETFAIR_API_KEY"
          valueFrom = "${var.betfair_secret_arn}:api_key::"
        },
        {
          name      = "BETFAIR_USERNAME"
          valueFrom = "${var.betfair_secret_arn}:username::"
        },
        {
          name      = "BETFAIR_PASSWORD"
          valueFrom = "${var.betfair_secret_arn}:password::"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.bot.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "bot"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
        interval    = var.health_check_interval
        timeout     = var.health_check_timeout
        retries     = var.health_check_retries
        startPeriod = var.health_check_start_period
      }

      stopTimeout = 30
    }
  ])

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-task"
      Service = "bot"
    }
  )
}

# =============================================================================
# ML Service Task Definition
# =============================================================================

resource "aws_ecs_task_definition" "ml_service" {
  family                   = "${local.name_prefix}-ml-service"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.ml_cpu
  memory                   = var.ml_memory
  task_role_arn            = var.ml_task_role_arn
  execution_role_arn       = var.ecs_execution_role_arn

  container_definitions = jsonencode([
    {
      name      = "ml-service"
      image     = "${var.ml_image_url}:${var.ml_image_tag}"
      essential = true

      portMappings = [
        {
          containerPort = 8000
          protocol      = "tcp"
          name          = "http"
        },
        {
          containerPort = 50051
          protocol      = "tcp"
          name          = "grpc"
        }
      ]

      environment = concat(
        local.common_env_vars,
        [
          {
            name  = "ENVIRONMENT"
            value = var.environment
          },
          {
            name  = "SERVICE_NAME"
            value = "ml-service"
          },
          {
            name  = "HTTP_PORT"
            value = "8000"
          },
          {
            name  = "GRPC_PORT"
            value = "50051"
          },
          {
            name  = "MLFLOW_TRACKING_URI"
            value = var.mlflow_tracking_uri
          }
        ]
      )

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = var.database_secret_arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.ml_service.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "ml-service"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "curl -f http://localhost:8000/health || exit 1"]
        interval    = var.health_check_interval
        timeout     = var.health_check_timeout
        retries     = var.health_check_retries
        startPeriod = 120 # Allow more time for model loading
      }

      stopTimeout = 60 # Allow time for model saving
    }
  ])

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-service-task"
      Service = "ml-service"
    }
  )
}

# =============================================================================
# Data Ingestion Task Definition (Scheduled)
# =============================================================================

resource "aws_ecs_task_definition" "data_ingestion" {
  family                   = "${local.name_prefix}-data-ingestion"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.data_ingestion_cpu
  memory                   = var.data_ingestion_memory
  task_role_arn            = var.bot_task_role_arn # Reuses bot role
  execution_role_arn       = var.ecs_execution_role_arn

  container_definitions = jsonencode([
    {
      name      = "data-ingestion"
      image     = "${var.data_ingestion_image_url}:${var.data_ingestion_image_tag}"
      essential = true
      command   = ["/app/bin/data-ingestion"]

      environment = concat(
        local.common_env_vars,
        [
          {
            name  = "SERVICE_NAME"
            value = "data-ingestion"
          },
          {
            name  = "RUN_MODE"
            value = "scheduled"
          }
        ]
      )

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = var.database_secret_arn
        },
        {
          name      = "BETFAIR_API_KEY"
          valueFrom = "${var.betfair_secret_arn}:api_key::"
        },
        {
          name      = "BETFAIR_USERNAME"
          valueFrom = "${var.betfair_secret_arn}:username::"
        },
        {
          name      = "BETFAIR_PASSWORD"
          valueFrom = "${var.betfair_secret_arn}:password::"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.data_ingestion.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "data-ingestion"
        }
      }

      stopTimeout = 30
    }
  ])

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-data-ingestion-task"
      Service = "data-ingestion"
    }
  )
}
