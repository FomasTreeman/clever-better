# Audit Module Outputs

output "cloudtrail_id" {
  description = "CloudTrail ID"
  value       = var.enable_cloudtrail ? aws_cloudtrail.main[0].id : ""
}

output "cloudtrail_arn" {
  description = "CloudTrail ARN"
  value       = var.enable_cloudtrail ? aws_cloudtrail.main[0].arn : ""
}

output "cloudtrail_bucket_name" {
  description = "S3 bucket name for CloudTrail logs"
  value       = var.enable_cloudtrail ? aws_s3_bucket.cloudtrail[0].id : ""
}

output "cloudtrail_log_group_name" {
  description = "CloudWatch log group name for CloudTrail"
  value       = var.enable_cloudtrail ? aws_cloudwatch_log_group.cloudtrail[0].name : ""
}

output "guardduty_detector_id" {
  description = "GuardDuty detector ID"
  value       = var.enable_guardduty ? aws_guardduty_detector.main[0].id : ""
}

output "security_alerts_topic_arn" {
  description = "SNS topic ARN for security alerts"
  value       = aws_sns_topic.security_alerts.arn
}

output "security_alerts_topic_name" {
  description = "SNS topic name for security alerts"
  value       = aws_sns_topic.security_alerts.name
}
