terraform {
  # backend "s3" {
  #   bucket         = "clever-better-terraform-state-production"
  #   key            = "terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "clever-better-terraform-locks"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region
}

locals {
  tags = {
    Project     = "clever-better"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

module "iam" {
  source      = "../../modules/iam"
  environment = var.environment
  tags        = local.tags
}

module "vpc" {
  source                   = "../../modules/vpc"
  environment              = var.environment
  vpc_cidr                 = var.vpc_cidr
  enable_nat_gateway       = var.enable_nat_gateway
  enable_flow_logs         = var.enable_flow_logs
  flow_logs_retention_days = var.flow_logs_retention_days
  flow_logs_role_arn       = module.iam.vpc_flow_logs_role_arn
  tags                     = local.tags
}

module "security" {
  source      = "../../modules/security"
  environment = var.environment
  vpc_id      = module.vpc.vpc_id
  tags        = local.tags
}

module "waf" {
  source               = "../../modules/waf"
  environment          = var.environment
  rate_limit_threshold = var.waf_rate_limit
  enable_logging       = var.enable_waf_logging
  enable_geo_blocking  = var.enable_geo_blocking
  allowed_countries    = var.allowed_countries
  tags                 = local.tags
}

module "secrets" {
  source          = "../../modules/secrets"
  environment     = var.environment
  enable_rotation = false # Set to true and provide rotation_lambda_arn when Lambda is available
  rotation_days   = 90
  tags            = local.tags
}

module "monitoring" {
  source      = "../../modules/monitoring"
  environment = var.environment
  alarm_email = var.alarm_email
  tags        = local.tags
}
