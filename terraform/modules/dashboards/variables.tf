variable "environment" {
  type        = string
  description = "Environment name (dev, staging, production)"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be dev, staging, or production."
  }
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "ecs_cluster_name" {
  type        = string
  description = "ECS cluster name for dashboard metrics"
}

variable "rds_instance_id" {
  type        = string
  description = "RDS instance identifier for dashboard metrics"
}

variable "alb_arn_suffix" {
  type        = string
  description = "ALB ARN suffix for dashboard metrics"
}

variable "log_group_names" {
  type        = list(string)
  default     = []
  description = "List of CloudWatch log group names for dashboard queries"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for resources"
}
