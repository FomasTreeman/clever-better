# ALB Module Variables

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

variable "vpc_id" {
  type        = string
  description = "VPC ID where ALB will be deployed"
}

variable "subnet_ids" {
  type        = list(string)
  description = "List of public subnet IDs for ALB"
  
  validation {
    condition     = length(var.subnet_ids) >= 2
    error_message = "At least 2 subnets in different AZs are required for ALB."
  }
}

variable "security_group_ids" {
  type        = list(string)
  description = "List of security group IDs for ALB"
  
  validation {
    condition     = length(var.security_group_ids) > 0
    error_message = "At least one security group is required."
  }
}

variable "certificate_arn" {
  type        = string
  description = "ACM certificate ARN for HTTPS listener"
  
  validation {
    condition     = can(regex("^arn:aws:acm:", var.certificate_arn))
    error_message = "Certificate ARN must be a valid ACM certificate ARN."
  }
}

variable "enable_deletion_protection" {
  type        = bool
  default     = true
  description = "Enable deletion protection for ALB"
}

variable "enable_access_logs" {
  type        = bool
  default     = false
  description = "Enable access logs to S3"
}

variable "access_logs_bucket" {
  type        = string
  default     = ""
  description = "S3 bucket name for access logs (created automatically if empty and enable_access_logs is true)"
}

variable "enable_stickiness" {
  type        = bool
  default     = false
  description = "Enable session stickiness"
}

variable "health_check_path" {
  type        = string
  default     = "/health"
  description = "Health check path for target groups"
}

variable "health_check_interval" {
  type        = number
  default     = 30
  description = "Health check interval in seconds"
  
  validation {
    condition     = var.health_check_interval >= 5 && var.health_check_interval <= 300
    error_message = "Health check interval must be between 5 and 300 seconds."
  }
}

variable "health_check_timeout" {
  type        = number
  default     = 5
  description = "Health check timeout in seconds"
  
  validation {
    condition     = var.health_check_timeout >= 2 && var.health_check_timeout <= 120
    error_message = "Health check timeout must be between 2 and 120 seconds."
  }
}

variable "deregistration_delay" {
  type        = number
  default     = 30
  description = "Time to wait before deregistering targets in seconds"
  
  validation {
    condition     = var.deregistration_delay >= 0 && var.deregistration_delay <= 3600
    error_message = "Deregistration delay must be between 0 and 3600 seconds."
  }
}

variable "waf_web_acl_arn" {
  type        = string
  default     = ""
  description = "WAF Web ACL ARN to associate with ALB (optional)"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for all resources"
}
