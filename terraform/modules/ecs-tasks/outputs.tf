# =============================================================================
# ECS Tasks Module Outputs
# =============================================================================

# -----------------------------------------------------------------------------
# Task Definition ARNs
# -----------------------------------------------------------------------------

output "bot_task_definition_arn" {
  description = "Bot task definition ARN"
  value       = aws_ecs_task_definition.bot.arn
}

output "ml_service_task_definition_arn" {
  description = "ML service task definition ARN"
  value       = aws_ecs_task_definition.ml_service.arn
}

output "data_ingestion_task_definition_arn" {
  description = "Data ingestion task definition ARN"
  value       = aws_ecs_task_definition.data_ingestion.arn
}

# -----------------------------------------------------------------------------
# Task Definition Families
# -----------------------------------------------------------------------------

output "bot_task_family" {
  description = "Bot task definition family"
  value       = aws_ecs_task_definition.bot.family
}

output "ml_service_task_family" {
  description = "ML service task definition family"
  value       = aws_ecs_task_definition.ml_service.family
}

output "data_ingestion_task_family" {
  description = "Data ingestion task definition family"
  value       = aws_ecs_task_definition.data_ingestion.family
}

# -----------------------------------------------------------------------------
# Task Definition Revisions
# -----------------------------------------------------------------------------

output "bot_task_revision" {
  description = "Bot task definition revision"
  value       = aws_ecs_task_definition.bot.revision
}

output "ml_service_task_revision" {
  description = "ML service task definition revision"
  value       = aws_ecs_task_definition.ml_service.revision
}

output "data_ingestion_task_revision" {
  description = "Data ingestion task definition revision"
  value       = aws_ecs_task_definition.data_ingestion.revision
}

# -----------------------------------------------------------------------------
# CloudWatch Log Groups
# -----------------------------------------------------------------------------

output "bot_log_group_name" {
  description = "Bot CloudWatch log group name"
  value       = aws_cloudwatch_log_group.bot.name
}

output "bot_log_group_arn" {
  description = "Bot CloudWatch log group ARN"
  value       = aws_cloudwatch_log_group.bot.arn
}

output "ml_service_log_group_name" {
  description = "ML service CloudWatch log group name"
  value       = aws_cloudwatch_log_group.ml_service.name
}

output "ml_service_log_group_arn" {
  description = "ML service CloudWatch log group ARN"
  value       = aws_cloudwatch_log_group.ml_service.arn
}

output "data_ingestion_log_group_name" {
  description = "Data ingestion CloudWatch log group name"
  value       = aws_cloudwatch_log_group.data_ingestion.name
}

output "data_ingestion_log_group_arn" {
  description = "Data ingestion CloudWatch log group ARN"
  value       = aws_cloudwatch_log_group.data_ingestion.arn
}
