terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Data source for current region
data "aws_region" "current" {}

# CloudWatch Dashboard for monitoring trading performance and ML metrics
resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "${var.project_name}-${var.environment}-main"

  dashboard_body = templatefile("${path.module}/dashboard.json.tpl", {
    environment      = var.environment
    project_name     = var.project_name
    ecs_cluster_name = var.ecs_cluster_name
    rds_instance_id  = var.rds_instance_id
    alb_arn_suffix   = var.alb_arn_suffix
    log_group_names  = jsonencode(var.log_group_names)
    region           = data.aws_region.current.name
  })
}
