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

# =============================================================================
# Container Image Tags
# =============================================================================

variable "bot_image_tag" {
  type        = string
  default     = "latest"
  description = "Bot container image tag"
}

variable "ml_image_tag" {
  type        = string
  default     = "latest"
  description = "ML service container image tag"
}

variable "data_ingestion_image_tag" {
  type        = string
  default     = "latest"
  description = "Data ingestion container image tag"
}

# =============================================================================
# Container Resource Allocation (Production - Higher)
# =============================================================================

variable "bot_cpu" {
  type        = number
  default     = 1024
  description = "Bot task CPU units"
}

variable "bot_memory" {
  type        = number
  default     = 2048
  description = "Bot task memory (MB)"
}

variable "ml_cpu" {
  type        = number
  default     = 2048
  description = "ML service task CPU units"
}

variable "ml_memory" {
  type        = number
  default     = 4096
  description = "ML service task memory (MB)"
}
