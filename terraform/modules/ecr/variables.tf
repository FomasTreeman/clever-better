# =============================================================================
# ECR Module Variables
# =============================================================================

variable "environment" {
  type        = string
  description = "Environment name (dev, staging, production)"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be dev, staging, or production."
  }
}

variable "project_name" {
  type        = string
  default     = "clever-better"
  description = "Project name for resource naming"
}

variable "repository_names" {
  type        = list(string)
  default     = ["bot", "ml-service", "data-ingestion"]
  description = "List of repository names to create"
}

variable "image_tag_mutability" {
  type        = string
  default     = "MUTABLE"
  description = "Image tag mutability (MUTABLE or IMMUTABLE)"

  validation {
    condition     = contains(["MUTABLE", "IMMUTABLE"], var.image_tag_mutability)
    error_message = "Image tag mutability must be MUTABLE or IMMUTABLE."
  }
}

variable "enable_scan_on_push" {
  type        = bool
  default     = true
  description = "Enable image scanning on push"
}

variable "encryption_type" {
  type        = string
  default     = "AES256"
  description = "Encryption type (AES256 or KMS)"

  validation {
    condition     = contains(["AES256", "KMS"], var.encryption_type)
    error_message = "Encryption type must be AES256 or KMS."
  }
}

variable "kms_key_arn" {
  type        = string
  default     = null
  description = "KMS key ARN for encryption (required if encryption_type is KMS)"
}

variable "enable_lifecycle_policy" {
  type        = bool
  default     = true
  description = "Enable lifecycle policy for image cleanup"
}

variable "max_image_count" {
  type        = number
  default     = 10
  description = "Maximum number of tagged images to retain"
}

variable "untagged_expiration_days" {
  type        = number
  default     = 7
  description = "Days to retain untagged images before expiration"
}

variable "latest_expiration_days" {
  type        = number
  default     = 30
  description = "Days to retain images tagged with 'latest'"
}

variable "ecs_execution_role_arn" {
  type        = string
  default     = ""
  description = "ECS task execution role ARN to grant pull permissions"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Additional tags for resources"
}
