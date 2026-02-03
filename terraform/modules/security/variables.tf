variable "vpc_id" {
  type        = string
  description = "VPC ID"
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

variable "allowed_cidr_blocks" {
  type        = list(string)
  default     = ["0.0.0.0/0"]
  description = "Allowed CIDR blocks for ALB ingress"
}

variable "enable_ssh_access" {
  type        = bool
  default     = false
  description = "Enable SSH access (not recommended)"
}

variable "ssh_cidr_blocks" {
  type        = list(string)
  default     = []
  description = "CIDR blocks allowed for SSH"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags"
}
