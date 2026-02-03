locals {
  tags = merge({
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }, var.tags)
}

# SNS topic for operational alarms
resource "aws_sns_topic" "operational_alarms" {
  name = "${var.project_name}-${var.environment}-operational-alarms"
  tags = local.tags
}

# SNS topic subscription for email notifications
resource "aws_sns_topic_subscription" "operational_alarms_email" {
  topic_arn = aws_sns_topic.operational_alarms.arn
  protocol  = "email"
  endpoint  = var.alarm_email
}

# Metric filter for unauthorized API calls from CloudTrail logs
resource "aws_cloudwatch_log_metric_filter" "unauthorized_api_calls" {
  count = var.cloudtrail_log_group_name != "" ? 1 : 0
  
  name           = "${var.project_name}-${var.environment}-unauthorized-api-calls"
  log_group_name = var.cloudtrail_log_group_name
  pattern        = "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") }"

  metric_transformation {
    name      = "UnauthorizedAPICalls"
    namespace = "CloudTrail/${var.project_name}-${var.environment}"
    value     = "1"
  }
}

# Alarm for unauthorized API calls (operational visibility)
resource "aws_cloudwatch_metric_alarm" "unauthorized_calls" {
  count = var.cloudtrail_log_group_name != "" ? 1 : 0
  
  alarm_name          = "${var.project_name}-${var.environment}-unauthorized-api-calls"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "UnauthorizedAPICalls"
  namespace           = "CloudTrail/${var.project_name}-${var.environment}"
  period              = 300
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "Multiple unauthorized API calls detected"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]
  treat_missing_data  = "notBreaching"

  depends_on = [aws_cloudwatch_log_metric_filter.unauthorized_api_calls]
}

# Application log groups
resource "aws_cloudwatch_log_group" "bot" {
  name              = "/ecs/${var.project_name}-${var.environment}/bot"
  retention_in_days = var.log_retention_days
  tags              = local.tags
}

resource "aws_cloudwatch_log_group" "ml_service" {
  name              = "/ecs/${var.project_name}-${var.environment}/ml-service"
  retention_in_days = var.log_retention_days
  tags              = local.tags
}

# RDS Alarms
resource "aws_cloudwatch_metric_alarm" "rds_cpu_high" {
  count = var.enable_rds_alarms && var.rds_instance_id != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-rds-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "RDS CPU utilization > 80%"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    DBInstanceIdentifier = var.rds_instance_id
  }
}

resource "aws_cloudwatch_metric_alarm" "rds_connections_high" {
  count = var.enable_rds_alarms && var.rds_instance_id != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-rds-connections-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "DatabaseConnections"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 160
  alarm_description   = "RDS database connections > 80% of max_connections (200)"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    DBInstanceIdentifier = var.rds_instance_id
  }
}

resource "aws_cloudwatch_metric_alarm" "rds_storage_low" {
  count = var.enable_rds_alarms && var.rds_instance_id != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-rds-storage-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  metric_name         = "FreeStorageSpace"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 10737418240
  alarm_description   = "RDS free storage space < 10 GB"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    DBInstanceIdentifier = var.rds_instance_id
  }
}

# ECS Alarms
resource "aws_cloudwatch_metric_alarm" "ecs_cpu_high" {
  count = var.enable_ecs_alarms && var.ecs_cluster_name != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-ecs-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "ECS cluster CPU utilization > 80%"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    ClusterName = var.ecs_cluster_name
  }
}

resource "aws_cloudwatch_metric_alarm" "ecs_memory_high" {
  count = var.enable_ecs_alarms && var.ecs_cluster_name != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-ecs-memory-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 85
  alarm_description   = "ECS cluster memory utilization > 85%"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    ClusterName = var.ecs_cluster_name
  }
}

# ALB Alarms
resource "aws_cloudwatch_metric_alarm" "alb_response_time_high" {
  count = var.enable_alb_alarms && var.alb_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-alb-response-time-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "TargetResponseTime"
  namespace           = "AWS/ApplicationELB"
  period              = 300
  statistic           = "Average"
  threshold           = 1
  alarm_description   = "ALB target response time > 1 second"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
  }
}

resource "aws_cloudwatch_metric_alarm" "alb_5xx_high" {
  count = var.enable_alb_alarms && var.alb_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-alb-5xx-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "HTTPCode_Target_5XX_Count"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Sum"
  threshold           = 10
  alarm_description   = "ALB 5xx errors > 10 per minute"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
  }
}

resource "aws_cloudwatch_metric_alarm" "alb_unhealthy_targets" {
  count = var.enable_alb_alarms && var.alb_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.environment}-alb-unhealthy-targets"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "UnHealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_description   = "ALB unhealthy target count > 0"
  alarm_actions       = [aws_sns_topic.operational_alarms.arn]

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
  }
}
