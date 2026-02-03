terraform {
  # backend "s3" {
  #   bucket         = "clever-better-terraform-state-staging"
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
  source             = "../../modules/vpc"
  environment        = var.environment
  vpc_cidr           = var.vpc_cidr
  enable_nat_gateway = var.enable_nat_gateway
  flow_logs_role_arn = module.iam.vpc_flow_logs_role_arn
  tags               = local.tags
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
  tags                 = local.tags
}

module "secrets" {
  source          = "../../modules/secrets"
  environment     = var.environment
  enable_rotation = false # Set to true and provide rotation_lambda_arn when Lambda is available
  tags            = local.tags
}

module "rds" {
  source                = "../../modules/rds"
  environment           = var.environment
  vpc_id                = module.vpc.vpc_id
  subnet_ids            = module.vpc.private_data_subnet_ids
  security_group_ids    = [module.security.database_security_group_id]
  monitoring_role_arn   = module.iam.rds_monitoring_role_arn
  instance_class        = var.rds_instance_class
  multi_az              = var.rds_multi_az
  deletion_protection   = true
  tags                  = local.tags
}

module "ecs" {
  source                  = "../../modules/ecs"
  environment             = var.environment
  enable_fargate_spot     = false
  tags                    = local.tags
}

module "alb" {
  source              = "../../modules/alb"
  environment         = var.environment
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.public_subnet_ids
  security_group_ids  = [module.security.alb_security_group_id]
  certificate_arn     = var.acm_certificate_arn
  waf_web_acl_arn     = module.waf.web_acl_arn
  enable_deletion_protection = true
  enable_access_logs  = true
  tags                = local.tags
}

module "audit" {
  source                        = "../../modules/audit"
  environment                   = var.environment
  alert_email                   = var.alarm_email
  enable_cloudtrail_insights    = false
  enable_s3_data_events         = false
  enable_guardduty_malware_protection = false
  tags                          = local.tags
}

module "monitoring" {
  source      = "../../modules/monitoring"
  environment = var.environment
  alarm_email = var.alarm_email
  rds_instance_id  = module.rds.db_instance_id
  ecs_cluster_name = module.ecs.cluster_name
  alb_arn_suffix   = join("/", slice(split("/", module.alb.alb_arn), 1, 4))
  cloudtrail_log_group_name = module.audit.cloudtrail_log_group_name
  tags        = local.tags
}
