variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region"
}

variable "environment" {
  type        = string
  default     = "dev"
  description = "Environment name"
}

variable "vpc_cidr" {
  type        = string
  default     = "10.0.0.0/16"
  description = "VPC CIDR"
}

variable "enable_nat_gateway" {
  type        = bool
  default     = true
  description = "Enable NAT gateways"
}

variable "enable_waf_logging" {
  type        = bool
  default     = false
  description = "Enable WAF logging in dev"
}

variable "waf_rate_limit" {
  type        = number
  default     = 5000
  description = "WAF rate limit"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarms"
}
