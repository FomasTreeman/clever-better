variable "vpc_cidr" {
  type        = string
  default     = "10.0.0.0/16"
  description = "VPC CIDR block"
}

variable "environment" {
  type        = string
  description = "Environment name"
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name"
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
  description = "Flow logs retention in days"
}

variable "availability_zones" {
  type        = list(string)
  default     = []
  description = "Availability zones (leave empty to auto-discover)"
}

variable "public_subnet_cidrs" {
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
  description = "Public subnet CIDRs"
}

variable "private_app_subnet_cidrs" {
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24"]
  description = "Private application subnet CIDRs"
}

variable "private_data_subnet_cidrs" {
  type        = list(string)
  default     = ["10.0.20.0/24", "10.0.21.0/24"]
  description = "Private data subnet CIDRs"
}

variable "flow_logs_role_arn" {
  type        = string
  default     = ""
  description = "IAM role ARN for VPC Flow Logs (required if enable_flow_logs is true)"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
