output "database_secret_arn" {
  value = aws_secretsmanager_secret.database.arn
}

output "database_secret_name" {
  value = aws_secretsmanager_secret.database.name
}

output "betfair_secret_arn" {
  value = aws_secretsmanager_secret.betfair.arn
}

output "betfair_secret_name" {
  value = aws_secretsmanager_secret.betfair.name
}

output "api_keys_secret_arn" {
  value = aws_secretsmanager_secret.api_keys.arn
}

output "api_keys_secret_name" {
  value = aws_secretsmanager_secret.api_keys.name
}

output "racing_post_secret_arn" {
  value = var.enable_racing_post ? aws_secretsmanager_secret.racing_post[0].arn : ""
}

output "kms_key_id" {
  value = var.use_custom_kms_key ? aws_kms_key.secrets[0].key_id : ""
}
