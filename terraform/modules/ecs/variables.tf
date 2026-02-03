# ECS Module Variables

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
  description = "Project name used in resource naming"
}

variable "enable_container_insights" {
  type        = bool
  default     = true
  description = "Enable CloudWatch Container Insights for the cluster"
}

variable "log_retention_days" {
  type        = number
  default     = 90
  description = "Number of days to retain CloudWatch logs"
  
  validation {
    condition = contains([
      1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653
    ], var.log_retention_days)
    error_message = "Log retention days must be a valid CloudWatch Logs retention value."
  }
}

variable "enable_fargate_spot" {
  type        = bool
  default     = false
  description = "Enable Fargate Spot capacity provider for cost optimization (recommended for dev/staging)"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for all resources"
}
