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
