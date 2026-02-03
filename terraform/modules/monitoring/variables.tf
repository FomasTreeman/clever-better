variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "enable_cloudtrail" {
  type        = bool
  default     = true
  description = "Enable CloudTrail"
}

variable "enable_guardduty" {
  type        = bool
  default     = true
  description = "Enable GuardDuty"
}

variable "cloudtrail_retention_days" {
  type        = number
  default     = 90
  description = "CloudTrail log retention"
}

variable "guardduty_finding_frequency" {
  type        = string
  default     = "FIFTEEN_MINUTES"
  description = "GuardDuty finding frequency"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarm notifications"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
