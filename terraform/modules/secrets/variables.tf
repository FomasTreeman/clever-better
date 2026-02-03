variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "enable_rotation" {
  type        = bool
  default     = true
  description = "Enable secrets rotation"
}

variable "rotation_days" {
  type        = number
  default     = 90
  description = "Rotation interval in days"
}

variable "rotation_lambda_arn" {
  type        = string
  default     = ""
  description = "Lambda ARN for secrets rotation"
}

variable "recovery_window_days" {
  type        = number
  default     = 30
  description = "Secret recovery window"
}

variable "use_custom_kms_key" {
  type        = bool
  default     = false
  description = "Use a custom KMS key"
}

variable "enable_racing_post" {
  type        = bool
  default     = false
  description = "Create Racing Post secret"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
