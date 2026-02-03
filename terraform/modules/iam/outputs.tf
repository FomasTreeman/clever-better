output "ecs_task_execution_role_arn" {
  value = aws_iam_role.ecs_execution.arn
}

output "ecs_task_execution_role_name" {
  value = aws_iam_role.ecs_execution.name
}

output "bot_task_role_arn" {
  value = aws_iam_role.bot_task.arn
}

output "bot_task_role_name" {
  value = aws_iam_role.bot_task.name
}

output "ml_service_task_role_arn" {
  value = aws_iam_role.ml_task.arn
}

output "ml_service_task_role_name" {
  value = aws_iam_role.ml_task.name
}

output "rds_monitoring_role_arn" {
  value = aws_iam_role.rds_monitoring.arn
}

output "vpc_flow_logs_role_arn" {
  value = aws_iam_role.vpc_flow_logs.arn
}

output "cloudwatch_events_role_arn" {
  value = aws_iam_role.cloudwatch_events.arn
}
