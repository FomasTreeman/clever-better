# RDS Module Variables

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

variable "instance_class" {
  type        = string
  default     = "db.r6g.large"
  description = "RDS instance class (e.g., db.r6g.large, db.t4g.medium)"
}

variable "allocated_storage" {
  type        = number
  default     = 100
  description = "Initial allocated storage in GB"
  
  validation {
    condition     = var.allocated_storage >= 20 && var.allocated_storage <= 65536
    error_message = "Allocated storage must be between 20 and 65536 GB."
  }
}

variable "max_allocated_storage" {
  type        = number
  default     = 500
  description = "Maximum storage for autoscaling in GB"
  
  validation {
    condition     = var.max_allocated_storage >= 0
    error_message = "Max allocated storage must be non-negative (0 disables autoscaling)."
  }
}

variable "database_name" {
  type        = string
  default     = "clever_better"
  description = "Initial database name"
  
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z0-9_]*$", var.database_name))
    error_message = "Database name must start with a letter and contain only alphanumeric characters and underscores."
  }
}

variable "master_username" {
  type        = string
  default     = "admin"
  description = "Master username for database"
  
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z0-9_]*$", var.master_username))
    error_message = "Master username must start with a letter and contain only alphanumeric characters and underscores."
  }
}

variable "multi_az" {
  type        = bool
  default     = true
  description = "Enable Multi-AZ deployment for high availability"
}

variable "backup_retention_period" {
  type        = number
  default     = 30
  description = "Number of days to retain automated backups"
  
  validation {
    condition     = var.backup_retention_period >= 0 && var.backup_retention_period <= 35
    error_message = "Backup retention period must be between 0 and 35 days."
  }
}

variable "deletion_protection" {
  type        = bool
  default     = true
  description = "Enable deletion protection"
}

variable "enable_performance_insights" {
  type        = bool
  default     = true
  description = "Enable Performance Insights"
}

variable "performance_insights_retention_period" {
  type        = number
  default     = 7
  description = "Performance Insights retention period in days (7, 731, or month * 31)"
  
  validation {
    condition     = var.performance_insights_retention_period == 7 || var.performance_insights_retention_period == 731 || (var.performance_insights_retention_period >= 31 && var.performance_insights_retention_period <= 731)
    error_message = "Performance Insights retention must be 7 days (free tier) or between 31-731 days."
  }
}

variable "vpc_id" {
  type        = string
  description = "VPC ID where RDS will be deployed"
}

variable "subnet_ids" {
  type        = list(string)
  description = "List of private data subnet IDs for DB subnet group"
  
  validation {
    condition     = length(var.subnet_ids) >= 2
    error_message = "At least 2 subnets are required for Multi-AZ deployment."
  }
}

variable "security_group_ids" {
  type        = list(string)
  description = "List of security group IDs to attach to RDS instance"
  
  validation {
    condition     = length(var.security_group_ids) > 0
    error_message = "At least one security group is required."
  }
}

variable "monitoring_role_arn" {
  type        = string
  description = "IAM role ARN for enhanced monitoring"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for all resources"
}
