variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "secrets_prefix" {
  type        = string
  default     = "clever-better"
  description = "Secrets Manager prefix"
}

variable "log_group_prefix" {
  type        = string
  default     = "/ecs/clever-better"
  description = "CloudWatch log group prefix"
}

variable "s3_bucket_prefix" {
  type        = string
  default     = "clever-better"
  description = "S3 bucket prefix"
}

variable "enable_enhanced_monitoring" {
  type        = bool
  default     = true
  description = "Enable RDS enhanced monitoring"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
