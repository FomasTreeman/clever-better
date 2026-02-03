# ECS Module - Fargate Cluster
# Provides ECS cluster with Fargate capacity providers and container insights

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
      Name        = "${local.name_prefix}-ecs"
      Environment = var.environment
      ManagedBy   = "terraform"
      Module      = "ecs"
    }
  )
}

# ECS Cluster
resource "aws_ecs_cluster" "this" {
  name = "${local.name_prefix}"

  setting {
    name  = "containerInsights"
    value = var.enable_container_insights ? "enabled" : "disabled"
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-cluster"
    }
  )
}

# ECS Cluster Capacity Providers
resource "aws_ecs_cluster_capacity_providers" "this" {
  cluster_name = aws_ecs_cluster.this.name

  capacity_providers = var.enable_fargate_spot ? ["FARGATE", "FARGATE_SPOT"] : ["FARGATE"]

  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
    base              = 1
  }

  dynamic "default_capacity_provider_strategy" {
    for_each = var.enable_fargate_spot ? [1] : []
    content {
      capacity_provider = "FARGATE_SPOT"
      weight            = 4
      base              = 0
    }
  }
}

# CloudWatch Log Group for ECS Cluster
resource "aws_cloudwatch_log_group" "cluster" {
  name              = "/ecs/${local.name_prefix}"
  retention_in_days = var.log_retention_days

  tags = merge(
    local.common_tags,
    {
      Name = "/ecs/${local.name_prefix}"
    }
  )
}
