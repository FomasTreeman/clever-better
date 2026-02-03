# =============================================================================
# ECS Tasks Module Variables
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
# IAM Role ARNs
# -----------------------------------------------------------------------------

variable "ecs_execution_role_arn" {
  type        = string
  description = "ECS task execution role ARN"
}

variable "bot_task_role_arn" {
  type        = string
  description = "Bot task role ARN"
}

variable "ml_task_role_arn" {
  type        = string
  description = "ML service task role ARN"
}

# -----------------------------------------------------------------------------
# Image Configuration
# -----------------------------------------------------------------------------

variable "bot_image_url" {
  type        = string
  description = "Bot container image URL (ECR repository URL)"
}

variable "bot_image_tag" {
  type        = string
  default     = "latest"
  description = "Bot container image tag"
}

variable "ml_image_url" {
  type        = string
  description = "ML service container image URL (ECR repository URL)"
}

variable "ml_image_tag" {
  type        = string
  default     = "latest"
  description = "ML service container image tag"
}

variable "data_ingestion_image_url" {
  type        = string
  description = "Data ingestion container image URL (ECR repository URL)"
}

variable "data_ingestion_image_tag" {
  type        = string
  default     = "latest"
  description = "Data ingestion container image tag"
}

# -----------------------------------------------------------------------------
# Bot Service Resources
# -----------------------------------------------------------------------------

variable "bot_cpu" {
  type        = number
  default     = 512
  description = "Bot task CPU units"
}

variable "bot_memory" {
  type        = number
  default     = 1024
  description = "Bot task memory (MB)"
}

# -----------------------------------------------------------------------------
# ML Service Resources
# -----------------------------------------------------------------------------

variable "ml_cpu" {
  type        = number
  default     = 1024
  description = "ML service task CPU units"
}

variable "ml_memory" {
  type        = number
  default     = 2048
  description = "ML service task memory (MB)"
}

variable "mlflow_tracking_uri" {
  type        = string
  default     = ""
  description = "MLflow tracking server URI"
}

# -----------------------------------------------------------------------------
# Data Ingestion Resources
# -----------------------------------------------------------------------------

variable "data_ingestion_cpu" {
  type        = number
  default     = 512
  description = "Data ingestion task CPU units"
}

variable "data_ingestion_memory" {
  type        = number
  default     = 1024
  description = "Data ingestion task memory (MB)"
}

# -----------------------------------------------------------------------------
# Secrets Manager ARNs
# -----------------------------------------------------------------------------

variable "database_secret_arn" {
  type        = string
  description = "Database credentials secret ARN"
}

variable "betfair_secret_arn" {
  type        = string
  description = "Betfair API credentials secret ARN"
}

# -----------------------------------------------------------------------------
# Logging Configuration
# -----------------------------------------------------------------------------

variable "log_retention_days" {
  type        = number
  default     = 30
  description = "CloudWatch log retention in days"
}

# -----------------------------------------------------------------------------
# Health Check Configuration
# -----------------------------------------------------------------------------

variable "health_check_interval" {
  type        = number
  default     = 30
  description = "Health check interval in seconds"
}

variable "health_check_timeout" {
  type        = number
  default     = 5
  description = "Health check timeout in seconds"
}

variable "health_check_retries" {
  type        = number
  default     = 3
  description = "Number of health check retries"
}

variable "health_check_start_period" {
  type        = number
  default     = 60
  description = "Health check start period in seconds"
}
