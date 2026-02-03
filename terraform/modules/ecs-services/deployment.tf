# =============================================================================
# ECS Services Module - Deployment Configuration
# Deployment monitoring, alarms, and notifications
# =============================================================================

# -----------------------------------------------------------------------------
# SNS Topic for Deployment Notifications
# -----------------------------------------------------------------------------

resource "aws_sns_topic" "deployment_notifications" {
  name = "${local.name_prefix}-deployment-notifications"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-deployment-notifications"
    }
  )
}

# -----------------------------------------------------------------------------
# Deployment Failure Alarms
# -----------------------------------------------------------------------------

resource "aws_cloudwatch_metric_alarm" "bot_high_cpu" {
  alarm_name          = "${local.name_prefix}-bot-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = 90
  alarm_description   = "Bot service CPU utilization is above 90%"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.bot.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-high-cpu"
      Service = "bot"
    }
  )
}

resource "aws_cloudwatch_metric_alarm" "bot_high_memory" {
  alarm_name          = "${local.name_prefix}-bot-high-memory"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = 90
  alarm_description   = "Bot service memory utilization is above 90%"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.bot.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-high-memory"
      Service = "bot"
    }
  )
}

resource "aws_cloudwatch_metric_alarm" "ml_high_cpu" {
  alarm_name          = "${local.name_prefix}-ml-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = 85
  alarm_description   = "ML service CPU utilization is above 85%"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.ml_service.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-high-cpu"
      Service = "ml-service"
    }
  )
}

resource "aws_cloudwatch_metric_alarm" "ml_high_memory" {
  alarm_name          = "${local.name_prefix}-ml-high-memory"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = 85
  alarm_description   = "ML service memory utilization is above 85%"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.ml_service.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-high-memory"
      Service = "ml-service"
    }
  )
}

# -----------------------------------------------------------------------------
# Running Task Count Alarms
# -----------------------------------------------------------------------------

resource "aws_cloudwatch_metric_alarm" "bot_running_tasks" {
  alarm_name          = "${local.name_prefix}-bot-no-running-tasks"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  metric_name         = "RunningTaskCount"
  namespace           = "ECS/ContainerInsights"
  period              = 60
  statistic           = "Average"
  threshold           = 1
  alarm_description   = "Bot service has no running tasks"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.bot.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-bot-no-running-tasks"
      Service = "bot"
    }
  )
}

resource "aws_cloudwatch_metric_alarm" "ml_running_tasks" {
  alarm_name          = "${local.name_prefix}-ml-no-running-tasks"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  metric_name         = "RunningTaskCount"
  namespace           = "ECS/ContainerInsights"
  period              = 60
  statistic           = "Average"
  threshold           = 1
  alarm_description   = "ML service has no running tasks"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = aws_ecs_service.ml_service.name
  }

  alarm_actions = [aws_sns_topic.deployment_notifications.arn]
  ok_actions    = [aws_sns_topic.deployment_notifications.arn]

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-ml-no-running-tasks"
      Service = "ml-service"
    }
  )
}
