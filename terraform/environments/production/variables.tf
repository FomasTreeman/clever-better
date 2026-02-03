variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region"
}

variable "environment" {
  type        = string
  default     = "production"
  description = "Environment name"
}

variable "vpc_cidr" {
  type        = string
  default     = "10.2.0.0/16"
  description = "VPC CIDR"
}

variable "enable_nat_gateway" {
  type        = bool
  default     = true
  description = "Enable NAT gateways"
}

variable "enable_flow_logs" {
  type        = bool
  default     = true
  description = "Enable VPC flow logs"
}

variable "flow_logs_retention_days" {
  type        = number
  default     = 90
  description = "Flow logs retention"
}

variable "enable_waf_logging" {
  type        = bool
  default     = true
  description = "Enable WAF logging"
}

variable "waf_rate_limit" {
  type        = number
  default     = 2000
  description = "WAF rate limit"
}

variable "enable_geo_blocking" {
  type        = bool
  default     = false
  description = "Enable geo blocking"
}

variable "allowed_countries" {
  type        = list(string)
  default     = ["GB", "IE"]
  description = "Allowed countries"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarms"
}
