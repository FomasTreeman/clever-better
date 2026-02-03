output "vpc_id" { value = module.vpc.vpc_id }
output "public_subnet_ids" { value = module.vpc.public_subnet_ids }
output "private_app_subnet_ids" { value = module.vpc.private_app_subnet_ids }
output "private_data_subnet_ids" { value = module.vpc.private_data_subnet_ids }
output "alb_security_group_id" { value = module.security.alb_security_group_id }
output "application_security_group_id" { value = module.security.application_security_group_id }
output "database_security_group_id" { value = module.security.database_security_group_id }
output "waf_web_acl_arn" { value = module.waf.web_acl_arn }
output "bot_task_role_arn" { value = module.iam.bot_task_role_arn }
output "ml_task_role_arn" { value = module.iam.ml_service_task_role_arn }
output "database_secret_arn" { value = module.secrets.database_secret_arn }

output "rds_endpoint" {
  value       = module.rds.db_instance_endpoint
  description = "RDS endpoint"
}

output "rds_secret_arn" {
  value       = module.rds.secret_arn
  description = "RDS credentials secret ARN"
  sensitive   = true
}

output "ecs_cluster_name" {
  value       = module.ecs.cluster_name
  description = "ECS cluster name"
}

output "alb_dns_name" {
  value       = module.alb.alb_dns_name
  description = "ALB DNS name"
}

output "ml_http_target_group_arn" {
  value       = module.alb.ml_http_target_group_arn
  description = "ML HTTP target group ARN"
}

output "ml_grpc_target_group_arn" {
  value       = module.alb.ml_grpc_target_group_arn
  description = "ML gRPC target group ARN"
}

output "cloudtrail_bucket" {
  value       = module.audit.cloudtrail_bucket_name
  description = "CloudTrail S3 bucket"
}

output "guardduty_detector_id" {
  value       = module.audit.guardduty_detector_id
  description = "GuardDuty detector ID"
}

# =============================================================================
# Dashboard Outputs
# =============================================================================

output "cloudwatch_dashboard_arn" {
  value       = module.dashboards.dashboard_arn
  description = "CloudWatch dashboard ARN"
}

output "cloudwatch_dashboard_url" {
  value       = module.dashboards.dashboard_url
  description = "CloudWatch dashboard URL"
}

# =============================================================================
# Alerts Outputs
# =============================================================================

output "critical_alarms_topic_arn" {
  value       = module.alerts.critical_topic_arn
  description = "Critical alerts SNS topic ARN"
}

output "warning_alarms_topic_arn" {
  value       = module.alerts.warning_topic_arn
  description = "Warning alerts SNS topic ARN"
}

output "info_alarms_topic_arn" {
  value       = module.alerts.info_topic_arn
  description = "Info alerts SNS topic ARN"
}

output "alarm_arns" {
  value       = module.alerts.alarm_arns
  description = "Map of alarm names to ARNs"
}

# =============================================================================
# ECR Outputs
# =============================================================================

output "ecr_repository_urls" {
  value       = module.ecr.repository_urls
  description = "ECR repository URLs"
}

output "ecr_registry_url" {
  value       = module.ecr.registry_url
  description = "ECR registry URL"
}

# =============================================================================
# ECS Task Definition Outputs
# =============================================================================

output "bot_task_definition_arn" {
  value       = module.ecs_tasks.bot_task_definition_arn
  description = "Bot task definition ARN"
}

output "ml_service_task_definition_arn" {
  value       = module.ecs_tasks.ml_service_task_definition_arn
  description = "ML service task definition ARN"
}

output "data_ingestion_task_definition_arn" {
  value       = module.ecs_tasks.data_ingestion_task_definition_arn
  description = "Data ingestion task definition ARN"
}

# =============================================================================
# ECS Service Outputs
# =============================================================================

output "bot_service_name" {
  value       = module.ecs_services.bot_service_name
  description = "Bot ECS service name"
}

output "ml_service_name" {
  value       = module.ecs_services.ml_service_name
  description = "ML ECS service name"
}

output "bot_autoscaling_target_id" {
  value       = module.ecs_services.bot_autoscaling_target_id
  description = "Bot auto-scaling target ID"
}

output "ml_autoscaling_target_id" {
  value       = module.ecs_services.ml_autoscaling_target_id
  description = "ML service auto-scaling target ID"
}

output "data_ingestion_rule_arn" {
  value       = module.ecs_services.data_ingestion_rule_arn
  description = "Data ingestion EventBridge rule ARN"
}
