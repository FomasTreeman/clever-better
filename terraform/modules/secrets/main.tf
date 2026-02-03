locals {
  tags = merge({
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Sensitive   = "true"
  }, var.tags)
}

resource "aws_kms_key" "secrets" {
  count               = var.use_custom_kms_key ? 1 : 0
  description         = "KMS key for ${var.project_name} secrets"
  enable_key_rotation = true
  tags                = local.tags
}

resource "aws_kms_alias" "secrets" {
  count         = var.use_custom_kms_key ? 1 : 0
  name          = "alias/${var.project_name}-secrets-${var.environment}"
  target_key_id = aws_kms_key.secrets[0].key_id
}

locals {
  kms_key_id = var.use_custom_kms_key ? aws_kms_key.secrets[0].key_id : null
}

resource "aws_secretsmanager_secret" "database" {
  name                    = "${var.project_name}/${var.environment}/database"
  description             = "Database credentials for TimescaleDB"
  recovery_window_in_days = var.recovery_window_days
  kms_key_id              = local.kms_key_id
  tags                    = merge(local.tags, { Component = "database" })
}

resource "aws_secretsmanager_secret" "betfair" {
  name                    = "${var.project_name}/${var.environment}/betfair"
  description             = "Betfair API credentials and certificates"
  recovery_window_in_days = var.recovery_window_days
  kms_key_id              = local.kms_key_id
  tags                    = merge(local.tags, { Component = "betfair" })
}

resource "aws_secretsmanager_secret" "api_keys" {
  name                    = "${var.project_name}/${var.environment}/api-keys"
  description             = "Internal service API keys"
  recovery_window_in_days = var.recovery_window_days
  kms_key_id              = local.kms_key_id
  tags                    = merge(local.tags, { Component = "internal" })
}

resource "aws_secretsmanager_secret" "racing_post" {
  count                   = var.enable_racing_post ? 1 : 0
  name                    = "${var.project_name}/${var.environment}/racing-post"
  description             = "Racing Post API credentials"
  recovery_window_in_days = var.recovery_window_days
  kms_key_id              = local.kms_key_id
  tags                    = merge(local.tags, { Component = "data-source" })
}

resource "aws_secretsmanager_secret_rotation" "database" {
  count               = var.enable_rotation && var.rotation_lambda_arn != "" ? 1 : 0
  secret_id           = aws_secretsmanager_secret.database.id
  rotation_lambda_arn = var.rotation_lambda_arn

  rotation_rules {
    automatically_after_days = var.rotation_days
  }
}
