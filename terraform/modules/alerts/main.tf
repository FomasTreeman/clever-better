terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# SNS topics for different alarm severities
resource "aws_sns_topic" "critical_alarms" {
  name = "${var.project_name}-${var.environment}-critical-alarms"
  tags = merge(
    var.tags,
    {
      Name        = "${var.project_name}-critical-alarms"
      Environment = var.environment
    }
  )
}

resource "aws_sns_topic" "warning_alarms" {
  name = "${var.project_name}-${var.environment}-warning-alarms"
  tags = merge(
    var.tags,
    {
      Name        = "${var.project_name}-warning-alarms"
      Environment = var.environment
    }
  )
}

resource "aws_sns_topic" "info_alarms" {
  name = "${var.project_name}-${var.environment}-info-alarms"
  tags = merge(
    var.tags,
    {
      Name        = "${var.project_name}-info-alarms"
      Environment = var.environment
    }
  )
}

# SNS topic subscriptions for critical alarms
resource "aws_sns_topic_subscription" "critical_email" {
  count           = var.critical_email != "" ? 1 : 0
  topic_arn       = aws_sns_topic.critical_alarms.arn
  protocol        = "email"
  endpoint        = var.critical_email
}

# SNS topic subscriptions for warning alarms
resource "aws_sns_topic_subscription" "warning_email" {
  count           = var.warning_email != "" ? 1 : 0
  topic_arn       = aws_sns_topic.warning_alarms.arn
  protocol        = "email"
  endpoint        = var.warning_email
}

# SNS topic subscriptions for info alarms
resource "aws_sns_topic_subscription" "info_email" {
  count           = var.info_email != "" ? 1 : 0
  topic_arn       = aws_sns_topic.info_alarms.arn
  protocol        = "email"
  endpoint        = var.info_email
}

# Trading Performance Alarms
resource "aws_cloudwatch_metric_alarm" "daily_loss_limit_exceeded" {
  alarm_name          = "${var.project_name}-${var.environment}-daily-loss-exceeded"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 1
  metric_name         = "daily_pnl"
  namespace           = "clever_better"
  period              = 300
  statistic           = "Average"
  threshold           = var.daily_loss_threshold
  alarm_description   = "Daily P&L has fallen below threshold"
  alarm_actions       = [aws_sns_topic.critical_alarms.arn]
  treat_missing_data  = "notBreaching"

  tags = merge(var.tags, { Alarm = "daily_loss_limit" })
}

resource "aws_cloudwatch_metric_alarm" "exposure_limit_exceeded" {
  alarm_name          = "${var.project_name}-${var.environment}-exposure-exceeded"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "total_exposure"
  namespace           = "clever_better"
  period              = 60
  statistic           = "Average"
  threshold           = var.exposure_limit
  alarm_description   = "Total exposure has exceeded threshold"
  alarm_actions       = [aws_sns_topic.critical_alarms.arn]
  treat_missing_data  = "notBreaching"

  tags = merge(var.tags, { Alarm = "exposure_limit" })
}

resource "aws_cloudwatch_metric_alarm" "bankroll_critical_low" {
  alarm_name          = "${var.project_name}-${var.environment}-bankroll-critical"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 1
  metric_name         = "current_bankroll"
  namespace           = "clever_better"
  period              = 300
  statistic           = "Average"
  threshold           = var.critical_bankroll_threshold
  alarm_description   = "Bankroll has fallen to critical level"
  alarm_actions       = [aws_sns_topic.critical_alarms.arn]
  treat_missing_data  = "notBreaching"

  tags = merge(var.tags, { Alarm = "bankroll_critical" })
}

# Strategy Alarms
resource "aws_cloudwatch_metric_alarm" "no_active_strategies" {
  alarm_name          = "${var.project_name}-${var.environment}-no-active-strategies"
  comparison_operator = "LessThanOrEqualToThreshold"
  evaluation_periods  = 2
  metric_name         = "active_strategies"
  namespace           = "clever_better"
  period              = 60
  statistic           = "Average"
  threshold           = 0
  alarm_description   = "No active strategies running"
  alarm_actions       = [aws_sns_topic.warning_alarms.arn]
  treat_missing_data  = "breaching"

  tags = merge(var.tags, { Alarm = "no_active_strategies" })
}

resource "aws_cloudwatch_metric_alarm" "circuit_breaker_trips" {
  alarm_name          = "${var.project_name}-${var.environment}-circuit-breaker-active"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "circuit_breaker_trips_total"
  namespace           = "clever_better"
  period              = 300
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "Circuit breaker has tripped"
  alarm_actions       = [aws_sns_topic.critical_alarms.arn]
  treat_missing_data  = "notBreaching"

  tags = merge(var.tags, { Alarm = "circuit_breaker_trips" })
}
