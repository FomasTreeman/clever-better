output "operational_alarms_topic_arn" {
  value       = aws_sns_topic.operational_alarms.arn
  description = "SNS topic ARN for operational alarms"
}

output "bot_log_group_name" {
  value       = aws_cloudwatch_log_group.bot.name
  description = "CloudWatch log group name for bot service"
}

output "ml_service_log_group_name" {
  value       = aws_cloudwatch_log_group.ml_service.name
  description = "CloudWatch log group name for ML service"
}
