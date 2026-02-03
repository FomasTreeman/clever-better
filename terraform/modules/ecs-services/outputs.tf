# =============================================================================
# ECS Services Module Outputs
# =============================================================================

# -----------------------------------------------------------------------------
# Bot Service
# -----------------------------------------------------------------------------

output "bot_service_id" {
  description = "Bot ECS service ID"
  value       = aws_ecs_service.bot.id
}

output "bot_service_name" {
  description = "Bot ECS service name"
  value       = aws_ecs_service.bot.name
}

output "bot_service_arn" {
  description = "Bot ECS service ARN"
  value       = aws_ecs_service.bot.id
}

# -----------------------------------------------------------------------------
# ML Service
# -----------------------------------------------------------------------------

output "ml_service_id" {
  description = "ML service ECS service ID"
  value       = aws_ecs_service.ml_service.id
}

output "ml_service_name" {
  description = "ML service ECS service name"
  value       = aws_ecs_service.ml_service.name
}

output "ml_service_arn" {
  description = "ML service ECS service ARN"
  value       = aws_ecs_service.ml_service.id
}

# -----------------------------------------------------------------------------
# Auto Scaling Targets
# -----------------------------------------------------------------------------

output "bot_autoscaling_target_id" {
  description = "Bot auto-scaling target ID"
  value       = aws_appautoscaling_target.bot.id
}

output "ml_autoscaling_target_id" {
  description = "ML service auto-scaling target ID"
  value       = aws_appautoscaling_target.ml_service.id
}

# -----------------------------------------------------------------------------
# EventBridge
# -----------------------------------------------------------------------------

output "data_ingestion_rule_arn" {
  description = "Data ingestion EventBridge rule ARN"
  value       = var.enable_data_ingestion ? aws_cloudwatch_event_rule.data_ingestion[0].arn : null
}

output "data_ingestion_rule_name" {
  description = "Data ingestion EventBridge rule name"
  value       = var.enable_data_ingestion ? aws_cloudwatch_event_rule.data_ingestion[0].name : null
}
