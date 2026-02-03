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
  enable_rotation = false
  tags            = local.tags
}

module "rds" {
  source              = "../../modules/rds"
  environment         = var.environment
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_data_subnet_ids
  security_group_ids  = [module.security.database_security_group_id]
  monitoring_role_arn = module.iam.rds_monitoring_role_arn
  instance_class      = var.rds_instance_class
  multi_az            = var.rds_multi_az
  deletion_protection = false # Allow deletion in dev
  tags                = local.tags
}

module "ecs" {
  source              = "../../modules/ecs"
  environment         = var.environment
  enable_fargate_spot = true # Cost savings in dev
  tags                = local.tags
}

module "alb" {
  source                     = "../../modules/alb"
  environment                = var.environment
  vpc_id                     = module.vpc.vpc_id
  subnet_ids                 = module.vpc.public_subnet_ids
  security_group_ids         = [module.security.alb_security_group_id]
  certificate_arn            = var.acm_certificate_arn
  waf_web_acl_arn            = module.waf.web_acl_arn
  enable_deletion_protection = false # Allow deletion in dev
  enable_access_logs         = false # Disable to save costs in dev
  tags                       = local.tags
}

module "audit" {
  source                              = "../../modules/audit"
  environment                         = var.environment
  alert_email                         = var.alarm_email
  enable_cloudtrail_insights          = false # Disable in dev to save costs
  enable_s3_data_events               = false
  enable_guardduty_malware_protection = false
  tags                                = local.tags
}

module "monitoring" {
  source                    = "../../modules/monitoring"
  environment               = var.environment
  alarm_email               = var.alarm_email
  rds_instance_id           = module.rds.db_instance_id
  ecs_cluster_name          = module.ecs.cluster_name
  alb_arn_suffix            = join("/", slice(split("/", module.alb.alb_arn), 1, 4))
  cloudtrail_log_group_name = module.audit.cloudtrail_log_group_name
  tags                      = local.tags
}

  # =============================================================================
  # CloudWatch Dashboards
  # =============================================================================

  module "dashboards" {
    source = "../../modules/dashboards"

    environment      = var.environment
    project_name     = "clever-better"
    ecs_cluster_name = module.ecs.cluster_name
    rds_instance_id  = module.rds.db_instance_id
    alb_arn_suffix   = join("/", slice(split("/", module.alb.alb_arn), 1, 4))
    log_group_names = [
      "/clever-better/strategy-decisions",
      "/clever-better/ml-operations",
      "/clever-better/audit-trail"
    ]

    tags = local.tags
  }

  # =============================================================================
  # CloudWatch Alerts
  # =============================================================================

  module "alerts" {
    source = "../../modules/alerts"

    environment                    = var.environment
    project_name                   = "clever-better"
    daily_loss_threshold           = var.daily_loss_threshold
    exposure_limit                 = var.exposure_limit
    critical_bankroll_threshold    = var.critical_bankroll_threshold
    critical_email                 = var.critical_alert_email
    warning_email                  = var.warning_alert_email
    info_email                      = var.info_alert_email

    tags = local.tags
  }
# =============================================================================
# ECR Repositories
# =============================================================================

module "ecr" {
  source = "../../modules/ecr"

  environment            = var.environment
  image_tag_mutability   = "MUTABLE"
  ecs_execution_role_arn = module.iam.ecs_task_execution_role_arn
  tags                   = local.tags
}

# =============================================================================
# ECS Task Definitions
# =============================================================================

module "ecs_tasks" {
  source = "../../modules/ecs-tasks"

  environment            = var.environment
  ecs_execution_role_arn = module.iam.ecs_task_execution_role_arn
  bot_task_role_arn      = module.iam.bot_task_role_arn
  ml_task_role_arn       = module.iam.ml_service_task_role_arn

  # Image configuration
  bot_image_url            = module.ecr.repository_urls["bot"]
  ml_image_url             = module.ecr.repository_urls["ml-service"]
  data_ingestion_image_url = module.ecr.repository_urls["data-ingestion"]
  bot_image_tag            = var.bot_image_tag
  ml_image_tag             = var.ml_image_tag
  data_ingestion_image_tag = var.data_ingestion_image_tag

  # Resource allocation (dev - smaller)
  bot_cpu    = var.bot_cpu
  bot_memory = var.bot_memory
  ml_cpu     = var.ml_cpu
  ml_memory  = var.ml_memory

  # Secrets
  database_secret_arn = module.secrets.database_secret_arn
  betfair_secret_arn  = module.secrets.betfair_secret_arn

  # Logging
  log_retention_days = 30

  tags = local.tags
}

# =============================================================================
# ECS Services
# =============================================================================

module "ecs_services" {
  source = "../../modules/ecs-services"

  environment      = var.environment
  ecs_cluster_id   = module.ecs.cluster_id
  ecs_cluster_name = module.ecs.cluster_name
  ecs_cluster_arn  = module.ecs.cluster_arn

  # Task definitions
  bot_task_definition_arn            = module.ecs_tasks.bot_task_definition_arn
  ml_service_task_definition_arn     = module.ecs_tasks.ml_service_task_definition_arn
  data_ingestion_task_definition_arn = module.ecs_tasks.data_ingestion_task_definition_arn

  # Network configuration
  private_subnet_ids             = module.vpc.private_app_subnet_ids
  application_security_group_ids = [module.security.application_security_group_id]

  # Load balancer
  ml_http_target_group_arn = module.alb.ml_http_target_group_arn
  ml_grpc_target_group_arn = module.alb.ml_grpc_target_group_arn

  # Service scaling (dev - minimal)
  bot_desired_count = 1
  bot_min_capacity  = 1
  bot_max_capacity  = 2
  ml_desired_count  = 1
  ml_min_capacity   = 1
  ml_max_capacity   = 3

  # Data ingestion
  cloudwatch_events_role_arn = module.iam.cloudwatch_events_role_arn
  enable_data_ingestion      = true
  data_ingestion_schedule    = "cron(0 */6 * * ? *)"

  # Dev options
  enable_execute_command = true

  tags = local.tags
}
