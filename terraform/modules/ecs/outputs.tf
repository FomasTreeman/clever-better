# ECS Module Outputs

output "cluster_id" {
  description = "ECS cluster ID"
  value       = aws_ecs_cluster.this.id
}

output "cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.this.name
}

output "cluster_arn" {
  description = "ECS cluster ARN"
  value       = aws_ecs_cluster.this.arn
}

output "log_group_name" {
  description = "CloudWatch log group name for ECS cluster"
  value       = aws_cloudwatch_log_group.cluster.name
}

output "log_group_arn" {
  description = "CloudWatch log group ARN for ECS cluster"
  value       = aws_cloudwatch_log_group.cluster.arn
}
