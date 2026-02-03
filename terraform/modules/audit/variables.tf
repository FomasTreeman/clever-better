# Audit Module Variables

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

variable "enable_cloudtrail" {
  type        = bool
  default     = true
  description = "Enable AWS CloudTrail for API logging"
}

variable "enable_guardduty" {
  type        = bool
  default     = true
  description = "Enable AWS GuardDuty for threat detection"
}

variable "cloudtrail_retention_days" {
  type        = number
  default     = 90
  description = "Number of days to retain CloudTrail logs in CloudWatch"
  
  validation {
    condition = contains([
      1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653
    ], var.cloudtrail_retention_days)
    error_message = "CloudWatch retention days must be a valid value."
  }
}

variable "cloudtrail_s3_retention_days" {
  type        = number
  default     = 2555
  description = "Number of days to retain CloudTrail logs in S3 (default 7 years for compliance)"
  
  validation {
    condition     = var.cloudtrail_s3_retention_days >= 90
    error_message = "S3 retention must be at least 90 days for audit compliance."
  }
}

variable "enable_cloudtrail_insights" {
  type        = bool
  default     = false
  description = "Enable CloudTrail Insights for anomaly detection (additional cost)"
}

variable "enable_s3_data_events" {
  type        = bool
  default     = false
  description = "Enable CloudTrail data events for S3 (additional cost)"
}

variable "guardduty_finding_frequency" {
  type        = string
  default     = "FIFTEEN_MINUTES"
  description = "GuardDuty finding publishing frequency"
  
  validation {
    condition     = contains(["FIFTEEN_MINUTES", "ONE_HOUR", "SIX_HOURS"], var.guardduty_finding_frequency)
    error_message = "Finding frequency must be FIFTEEN_MINUTES, ONE_HOUR, or SIX_HOURS."
  }
}

variable "enable_guardduty_s3_protection" {
  type        = bool
  default     = true
  description = "Enable GuardDuty S3 protection"
}

variable "enable_guardduty_eks_protection" {
  type        = bool
  default     = false
  description = "Enable GuardDuty EKS protection (only needed if using EKS)"
}

variable "enable_guardduty_malware_protection" {
  type        = bool
  default     = false
  description = "Enable GuardDuty malware protection for EC2/EBS (additional cost)"
}

variable "alert_email" {
  type        = string
  description = "Email address for security alerts"
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.alert_email))
    error_message = "Alert email must be a valid email address."
  }
}

variable "enable_guardduty_export" {
  type        = bool
  default     = false
  description = "Enable GuardDuty findings export to S3"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for all resources"
}
