variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region"
}

variable "environment" {
  type        = string
  default     = "dev"
  description = "Environment name"
}

variable "vpc_cidr" {
  type        = string
  default     = "10.0.0.0/16"
  description = "VPC CIDR"
}

variable "enable_nat_gateway" {
  type        = bool
  default     = true
  description = "Enable NAT gateways"
}

variable "enable_waf_logging" {
  type        = bool
  default     = false
  description = "Enable WAF logging in dev"
}

variable "waf_rate_limit" {
  type        = number
  default     = 5000
  description = "WAF rate limit"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarms"
}

variable "rds_instance_class" {
  type        = string
  default     = "db.t4g.medium" # Smaller instance for dev
  description = "RDS instance class"
}

variable "rds_multi_az" {
  type        = bool
  default     = false # Single AZ for dev to save costs
  description = "Enable Multi-AZ for RDS"
}

variable "acm_certificate_arn" {
  type        = string
  description = "ACM certificate ARN for HTTPS"
}

# =============================================================================
# Container Image Tags
# =============================================================================

variable "bot_image_tag" {
  type        = string
  default     = "latest"
  description = "Bot container image tag"
}

variable "ml_image_tag" {
  type        = string
  default     = "latest"
  description = "ML service container image tag"
}

variable "data_ingestion_image_tag" {
  type        = string
  default     = "latest"
  description = "Data ingestion container image tag"
}

# =============================================================================
# Container Resource Allocation
# =============================================================================

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


  # =============================================================================
  # Alert Configuration
  # =============================================================================

  variable "daily_loss_threshold" {
    type        = number
    default     = -500
    description = "Daily loss threshold for alerting"
  }

  variable "exposure_limit" {
    type        = number
    default     = 50000
    description = "Total market exposure limit"
  }

  variable "critical_bankroll_threshold" {
    type        = number
    default     = 1000
    description = "Critical minimum bankroll threshold"
  }

  variable "critical_alert_email" {
    type        = string
    description = "Email for critical alerts"
  }

  variable "warning_alert_email" {
    type        = string
    description = "Email for warning alerts"
  }

  variable "info_alert_email" {
    type        = string
    description = "Email for info alerts"
  }
