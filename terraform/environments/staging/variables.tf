variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region"
}

variable "environment" {
  type        = string
  default     = "staging"
  description = "Environment name"
}

variable "vpc_cidr" {
  type        = string
  default     = "10.1.0.0/16"
  description = "VPC CIDR"
}

variable "enable_nat_gateway" {
  type        = bool
  default     = true
  description = "Enable NAT gateways"
}

variable "enable_waf_logging" {
  type        = bool
  default     = true
  description = "Enable WAF logging"
}

variable "waf_rate_limit" {
  type        = number
  default     = 3000
  description = "WAF rate limit"
}

variable "alarm_email" {
  type        = string
  description = "Email for alarms"
}

variable "rds_instance_class" {
  type        = string
  default     = "db.r6g.large"
  description = "RDS instance class"
}

variable "rds_multi_az" {
  type        = bool
  default     = true
  description = "Enable Multi-AZ for RDS"
}

variable "acm_certificate_arn" {
  type        = string
  description = "ACM certificate ARN for HTTPS"
}
