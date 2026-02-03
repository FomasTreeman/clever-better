# ALB Module Outputs

output "alb_id" {
  description = "ALB ID"
  value       = aws_lb.this.id
}

output "alb_arn" {
  description = "ALB ARN"
  value       = aws_lb.this.arn
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = aws_lb.this.dns_name
}

output "alb_zone_id" {
  description = "ALB Route53 zone ID"
  value       = aws_lb.this.zone_id
}

output "http_listener_arn" {
  description = "HTTP listener ARN"
  value       = aws_lb_listener.http.arn
}

output "https_listener_arn" {
  description = "HTTPS listener ARN"
  value       = aws_lb_listener.https.arn
}

output "ml_http_target_group_arn" {
  description = "ML HTTP target group ARN"
  value       = aws_lb_target_group.ml_http.arn
}

output "ml_http_target_group_name" {
  description = "ML HTTP target group name"
  value       = aws_lb_target_group.ml_http.name
}

output "ml_grpc_target_group_arn" {
  description = "ML gRPC target group ARN"
  value       = aws_lb_target_group.ml_grpc.arn
}

output "ml_grpc_target_group_name" {
  description = "ML gRPC target group name"
  value       = aws_lb_target_group.ml_grpc.name
}

output "access_logs_bucket_name" {
  description = "S3 bucket name for access logs (if enabled)"
  value       = var.enable_access_logs ? (var.access_logs_bucket != "" ? var.access_logs_bucket : aws_s3_bucket.access_logs[0].id) : ""
}
