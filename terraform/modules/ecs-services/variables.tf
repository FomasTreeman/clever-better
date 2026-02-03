# =============================================================================
# ECS Services Module Variables
# =============================================================================

# -----------------------------------------------------------------------------
# General Configuration
# -----------------------------------------------------------------------------

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
  description = "Project name for resource naming"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for resources"
}

# -----------------------------------------------------------------------------
# ECS Cluster
# -----------------------------------------------------------------------------

variable "ecs_cluster_id" {
  type        = string
  description = "ECS cluster ID"
}

variable "ecs_cluster_name" {
  type        = string
  description = "ECS cluster name"
}

variable "ecs_cluster_arn" {
  type        = string
  description = "ECS cluster ARN"
}

# -----------------------------------------------------------------------------
# Task Definitions
# -----------------------------------------------------------------------------

variable "bot_task_definition_arn" {
  type        = string
  description = "Bot task definition ARN"
}

variable "ml_service_task_definition_arn" {
  type        = string
  description = "ML service task definition ARN"
}

variable "data_ingestion_task_definition_arn" {
  type        = string
  description = "Data ingestion task definition ARN"
}

# -----------------------------------------------------------------------------
# Network Configuration
# -----------------------------------------------------------------------------

variable "private_subnet_ids" {
  type        = list(string)
  description = "Private subnet IDs for ECS tasks"
}

variable "application_security_group_ids" {
  type        = list(string)
  description = "Security group IDs for application"
}

# -----------------------------------------------------------------------------
# Load Balancer Configuration
# -----------------------------------------------------------------------------

variable "ml_http_target_group_arn" {
  type        = string
  description = "ML service HTTP target group ARN"
}

variable "ml_grpc_target_group_arn" {
  type        = string
  description = "ML service gRPC target group ARN"
}

variable "alb_arn_suffix" {
  type        = string
  default     = ""
  description = "ALB ARN suffix for request count scaling"
}

variable "ml_target_group_arn_suffix" {
  type        = string
  default     = ""
  description = "ML target group ARN suffix for request count scaling"
}

# -----------------------------------------------------------------------------
# Bot Service Configuration
# -----------------------------------------------------------------------------

variable "bot_desired_count" {
  type        = number
  default     = 1
  description = "Bot service desired task count"
}

variable "bot_min_capacity" {
  type        = number
  default     = 1
  description = "Bot service minimum task count"
}

variable "bot_max_capacity" {
  type        = number
  default     = 2
  description = "Bot service maximum task count"
}

variable "bot_cpu_target" {
  type        = number
  default     = 70
  description = "Bot CPU utilization target for auto-scaling (%)"
}

variable "bot_memory_target" {
  type        = number
  default     = 80
  description = "Bot memory utilization target for auto-scaling (%)"
}

# -----------------------------------------------------------------------------
# ML Service Configuration
# -----------------------------------------------------------------------------

variable "ml_desired_count" {
  type        = number
  default     = 1
  description = "ML service desired task count"
}

variable "ml_min_capacity" {
  type        = number
  default     = 1
  description = "ML service minimum task count"
}

variable "ml_max_capacity" {
  type        = number
  default     = 3
  description = "ML service maximum task count"
}

variable "ml_cpu_target" {
  type        = number
  default     = 60
  description = "ML CPU utilization target for auto-scaling (%)"
}

variable "ml_memory_target" {
  type        = number
  default     = 75
  description = "ML memory utilization target for auto-scaling (%)"
}

variable "ml_request_count_target" {
  type        = number
  default     = 100
  description = "ML request count per target for auto-scaling"
}

# -----------------------------------------------------------------------------
# Data Ingestion Configuration
# -----------------------------------------------------------------------------

variable "enable_data_ingestion" {
  type        = bool
  default     = true
  description = "Enable scheduled data ingestion"
}

variable "data_ingestion_schedule" {
  type        = string
  default     = "cron(0 */6 * * ? *)"
  description = "Data ingestion schedule expression (default: every 6 hours)"
}

variable "cloudwatch_events_role_arn" {
  type        = string
  description = "CloudWatch Events role ARN for scheduled tasks"
}

# -----------------------------------------------------------------------------
# Service Configuration
# -----------------------------------------------------------------------------

variable "enable_execute_command" {
  type        = bool
  default     = false
  description = "Enable ECS Exec for debugging"
}
