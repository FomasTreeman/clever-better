variable "environment" {
  type        = string
  description = "Environment name"
  
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

variable "daily_loss_threshold" {
  type        = number
  default     = -500
  description = "Daily loss threshold (in currency units) that triggers alarm"
}

variable "exposure_limit" {
  type        = number
  default     = 50000
  description = "Total exposure limit that triggers alarm"
}

variable "critical_bankroll_threshold" {
  type        = number
  description = "Critical bankroll threshold (20% of initial)"
}

variable "critical_email" {
  type        = string
  default     = ""
  description = "Email for critical alarm notifications"
}

variable "warning_email" {
  type        = string
  default     = ""
  description = "Email for warning alarm notifications"
}

variable "info_email" {
  type        = string
  default     = ""
  description = "Email for informational alarm notifications"
}

variable "slack_webhook_url" {
  type        = string
  default     = ""
  sensitive   = true
  description = "Slack webhook URL for alarm notifications"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for resources"
}
