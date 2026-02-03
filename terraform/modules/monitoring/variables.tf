variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarm notifications"
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.alarm_email))
    error_message = "Alarm email must be a valid email address."
  }
}

variable "log_retention_days" {
  type        = number
  default     = 90
  description = "CloudWatch log retention in days"
}

variable "rds_instance_id" {
  type        = string
  default     = ""
  description = "RDS instance identifier for alarms"
}

variable "ecs_cluster_name" {
  type        = string
  default     = ""
  description = "ECS cluster name for alarms"
}

variable "alb_arn_suffix" {
  type        = string
  default     = ""
  description = "ALB ARN suffix for alarms"
}

variable "cloudtrail_log_group_name" {
  type        = string
  default     = ""
  description = "CloudTrail log group name for metric filters (from audit module)"
}

variable "enable_rds_alarms" {
  type        = bool
  default     = true
  description = "Enable RDS alarms"
}

variable "enable_ecs_alarms" {
  type        = bool
  default     = true
  description = "Enable ECS alarms"
}

variable "enable_alb_alarms" {
  type        = bool
  default     = true
  description = "Enable ALB alarms"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
