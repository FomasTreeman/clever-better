output "critical_topic_arn" {
  description = "ARN of the critical alarms SNS topic"
  value       = aws_sns_topic.critical_alarms.arn
}

output "warning_topic_arn" {
  description = "ARN of the warning alarms SNS topic"
  value       = aws_sns_topic.warning_alarms.arn
}

output "info_topic_arn" {
  description = "ARN of the info alarms SNS topic"
  value       = aws_sns_topic.info_alarms.arn
}

output "alarm_arns" {
  description = "Map of alarm names to their ARNs"
  value = {
    daily_loss_exceeded    = aws_cloudwatch_metric_alarm.daily_loss_limit_exceeded.arn
    exposure_exceeded      = aws_cloudwatch_metric_alarm.exposure_limit_exceeded.arn
    bankroll_critical      = aws_cloudwatch_metric_alarm.bankroll_critical_low.arn
    no_active_strategies   = aws_cloudwatch_metric_alarm.no_active_strategies.arn
    circuit_breaker_trips  = aws_cloudwatch_metric_alarm.circuit_breaker_trips.arn
  }
}
