variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
}

variable "rate_limit_threshold" {
  type        = number
  default     = 2000
  description = "Rate limit threshold per 5 minutes"
}

variable "enable_geo_blocking" {
  type        = bool
  default     = false
  description = "Enable geo blocking"
}

variable "allowed_countries" {
  type        = list(string)
  default     = ["GB", "IE"]
  description = "Allowed countries if geo blocking is enabled"
}

variable "ip_allowlist" {
  type        = list(string)
  default     = []
  description = "Allowlisted IPs"
}

variable "ip_blocklist" {
  type        = list(string)
  default     = []
  description = "Blocked IPs"
}

variable "enable_logging" {
  type        = bool
  default     = true
  description = "Enable WAF logging"
}

variable "log_retention_days" {
  type        = number
  default     = 90
  description = "Log retention in days"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
