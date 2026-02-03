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

variable "rds_instance_class" {
  type        = string
  default     = "db.t4g.medium"  # Smaller instance for dev
  description = "RDS instance class"
}

variable "rds_multi_az" {
  type        = bool
  default     = false  # Single AZ for dev to save costs
  description = "Enable Multi-AZ for RDS"
}

variable "acm_certificate_arn" {
  type        = string
  description = "ACM certificate ARN for HTTPS"
}
