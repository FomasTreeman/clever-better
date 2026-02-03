# RDS Module - PostgreSQL 15 with TimescaleDB support
# Provides Multi-AZ RDS instance with encryption, automated backups, and monitoring

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  
  common_tags = merge(
    var.tags,
    {
      Name        = "${local.name_prefix}-rds"
      Environment = var.environment
      ManagedBy   = "terraform"
      Module      = "rds"
    }
  )
}

# DB Subnet Group for private data subnets
resource "aws_db_subnet_group" "this" {
  name       = "${local.name_prefix}-db-subnet-group"
  subnet_ids = var.subnet_ids

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-db-subnet-group"
    }
  )
}

# Parameter Group for PostgreSQL 15 with TimescaleDB optimization
resource "aws_db_parameter_group" "this" {
  name   = "${local.name_prefix}-postgres15-timescaledb"
  family = "postgres15"

  # TimescaleDB extension
  parameter {
    name  = "shared_preload_libraries"
    value = "timescaledb"
  }

  # Connection settings
  parameter {
    name  = "max_connections"
    value = "200"
  }

  # Memory settings - optimized for db.r6g.large (16 GB RAM)
  parameter {
    name  = "shared_buffers"
    value = "{DBInstanceClassMemory/4096}"  # ~4 GB for r6g.large
  }

  parameter {
    name  = "effective_cache_size"
    value = "{DBInstanceClassMemory*3/4096}"  # ~12 GB for r6g.large
  }

  parameter {
    name  = "work_mem"
    value = "10485"  # 10 MB
  }

  parameter {
    name  = "maintenance_work_mem"
    value = "1048576"  # 1 GB
  }

  # TimescaleDB specific
  parameter {
    name  = "timescaledb.max_background_workers"
    value = "8"
  }

  # Write-ahead log settings for better performance
  parameter {
    name  = "wal_buffers"
    value = "16384"  # 16 MB
  }

  parameter {
    name  = "checkpoint_completion_target"
    value = "0.9"
  }

  parameter {
    name  = "random_page_cost"
    value = "1.1"  # For SSD storage
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-postgres15-timescaledb"
    }
  )
}

# KMS key for RDS encryption
resource "aws_kms_key" "rds" {
  description             = "KMS key for ${local.name_prefix} RDS encryption"
  deletion_window_in_days = 10
  enable_key_rotation     = true

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-rds-kms"
    }
  )
}

resource "aws_kms_alias" "rds" {
  name          = "alias/${local.name_prefix}-rds"
  target_key_id = aws_kms_key.rds.key_id
}

# Generate secure random password for database
resource "random_password" "db_password" {
  length  = 32
  special = true
  # Avoid characters that might cause issues in connection strings
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# RDS PostgreSQL Instance
resource "aws_db_instance" "this" {
  identifier = "${local.name_prefix}-postgres"

  # Engine configuration
  engine               = "postgres"
  engine_version       = "15.5"
  instance_class       = var.instance_class
  
  # Storage configuration
  allocated_storage     = var.allocated_storage
  max_allocated_storage = var.max_allocated_storage
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.rds.arn
  iops                  = 3000
  storage_throughput    = 125

  # Database configuration
  db_name  = var.database_name
  username = var.master_username
  password = random_password.db_password.result
  port     = 5432

  # Network configuration
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = var.security_group_ids
  publicly_accessible    = false
  multi_az               = var.multi_az

  # Parameter and option groups
  parameter_group_name = aws_db_parameter_group.this.name

  # Backup configuration
  backup_retention_period = var.backup_retention_period
  backup_window           = "03:00-04:00"
  maintenance_window      = "Mon:04:00-Mon:05:00"
  
  # Enhanced monitoring
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  monitoring_interval             = 60
  monitoring_role_arn             = var.monitoring_role_arn

  # Performance Insights
  performance_insights_enabled          = var.enable_performance_insights
  performance_insights_retention_period = var.enable_performance_insights ? var.performance_insights_retention_period : null
  performance_insights_kms_key_id       = var.enable_performance_insights ? aws_kms_key.rds.arn : null

  # Deletion protection
  deletion_protection       = var.deletion_protection
  skip_final_snapshot       = false
  final_snapshot_identifier = "${local.name_prefix}-postgres-final-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  copy_tags_to_snapshot     = true

  # Auto minor version upgrade
  auto_minor_version_upgrade = true

  # Apply changes immediately in non-production
  apply_immediately = var.environment != "production"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-postgres"
    }
  )

  lifecycle {
    ignore_changes = [
      final_snapshot_identifier,  # Timestamp will always change
      password,                   # Managed by secrets manager after initial creation
    ]
  }
}

# Secrets Manager secret for database credentials
resource "aws_secretsmanager_secret" "db_credentials" {
  name                    = "${local.name_prefix}-db-credentials"
  description             = "Database credentials for ${local.name_prefix} PostgreSQL"
  recovery_window_in_days = 7

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-db-credentials"
    }
  )
}

# Store database credentials in Secrets Manager
resource "aws_secretsmanager_secret_version" "db_credentials" {
  secret_id = aws_secretsmanager_secret.db_credentials.id
  
  secret_string = jsonencode({
    username = var.master_username
    password = random_password.db_password.result
    engine   = "postgres"
    host     = aws_db_instance.this.address
    port     = aws_db_instance.this.port
    dbname   = var.database_name
    dbInstanceIdentifier = aws_db_instance.this.identifier
  })

  lifecycle {
    ignore_changes = [
      secret_string,  # Allow manual rotation without Terraform changes
    ]
  }
}
