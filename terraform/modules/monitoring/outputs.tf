output "cloudtrail_arn" {
  value = var.enable_cloudtrail ? aws_cloudtrail.this[0].arn : ""
}

output "cloudtrail_bucket_name" {
  value = var.enable_cloudtrail ? aws_s3_bucket.cloudtrail[0].bucket : ""
}

output "guardduty_detector_id" {
  value = var.enable_guardduty ? aws_guardduty_detector.this[0].id : ""
}

output "security_alarms_topic_arn" {
  value = aws_sns_topic.security.arn
}
