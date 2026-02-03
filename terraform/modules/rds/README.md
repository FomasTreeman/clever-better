# RDS Module

This module creates an Amazon RDS PostgreSQL 15 instance optimized for TimescaleDB time-series workloads.

## Features

- **PostgreSQL 15** with TimescaleDB extension support
- **Multi-AZ deployment** for high availability (configurable)
- **Encryption at rest** using KMS with automatic key rotation
- **Encryption in transit** via SSL/TLS
- **Automated backups** with configurable retention (up to 35 days)
- **Performance Insights** for query performance monitoring
- **Enhanced monitoring** with 60-second granularity
- **CloudWatch Logs** for PostgreSQL logs and upgrade logs
- **Auto-scaling storage** to prevent out-of-space issues
- **Automated minor version upgrades**
- **Deletion protection** for production environments
- **Secrets Manager integration** for credential management

## Architecture

```
┌─────────────────────────────────────────┐
│         VPC Private Subnets             │
│  ┌──────────────┐    ┌──────────────┐   │
│  │   AZ-1       │    │   AZ-2       │   │
│  │ ┌──────────┐ │    │ ┌──────────┐ │   │
│  │ │ Primary  │◄┼────┼►│ Standby  │ │   │
│  │ │ RDS      │ │    │ │ RDS      │ │   │
│  │ │(Multi-AZ)│ │    │ │          │ │   │
│  │ └──────────┘ │    │ └──────────┘ │   │
│  └──────────────┘    └──────────────┘   │
└─────────────────────────────────────────┘
           │                  │
           ▼                  ▼
    ┌─────────────┐    ┌──────────────┐
    │  KMS Key    │    │   Secrets    │
    │ (Encrypted) │    │   Manager    │
    └─────────────┘    └──────────────┘
```

## Usage

```hcl
module "rds" {
  source = "../../modules/rds"

  environment         = "production"
  project_name        = "clever-better"
  
  # Instance configuration
  instance_class      = "db.r6g.large"
  allocated_storage   = 100
  max_allocated_storage = 500
  
  # High availability
  multi_az            = true
  deletion_protection = true
  
  # Network configuration
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_data_subnet_ids
  security_group_ids  = [module.security.database_security_group_id]
  
  # Monitoring
  monitoring_role_arn = module.iam.rds_monitoring_role_arn
  enable_performance_insights = true
  performance_insights_retention_period = 7
  
  # Backups
  backup_retention_period = 30
  
  tags = {
    Project   = "clever-better"
    ManagedBy = "terraform"
  }
}
```

## Post-Deployment Steps

### 1. Install TimescaleDB Extension

After the RDS instance is created, you must manually install the TimescaleDB extension:

```sql
-- Connect to the database as the master user
psql -h <rds_endpoint> -U admin -d clever_better

-- Install TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Verify installation
SELECT default_version FROM pg_available_extensions WHERE name = 'timescaledb';

-- Show TimescaleDB version
SELECT extname, extversion FROM pg_extension WHERE extname = 'timescaledb';
```

### 2. Retrieve Database Credentials

Credentials are stored in AWS Secrets Manager:

```bash
# Get secret ARN from Terraform output
SECRET_ARN=$(terraform output -raw rds_secret_arn)

# Retrieve credentials
aws secretsmanager get-secret-value \
  --secret-id $SECRET_ARN \
  --query SecretString \
  --output text | jq .
```

### 3. Connection String Format

```bash
# Standard PostgreSQL connection string
postgresql://admin:<password>@<rds_endpoint>:5432/clever_better?sslmode=require

# Using environment variables
export PGHOST=<rds_endpoint>
export PGPORT=5432
export PGDATABASE=clever_better
export PGUSER=admin
export PGPASSWORD=<password>
export PGSSLMODE=require
```

### 4. Run Database Migrations

```bash
# Run migrations from the migrations/ directory
cd migrations
./run-migrations.sh
```

## TimescaleDB Configuration

The module creates a parameter group optimized for TimescaleDB:

- **shared_preload_libraries**: `timescaledb` (required for extension)
- **max_connections**: 200
- **shared_buffers**: 25% of instance memory
- **effective_cache_size**: 75% of instance memory
- **work_mem**: 10 MB
- **maintenance_work_mem**: 1 GB
- **timescaledb.max_background_workers**: 8

## Monitoring and Alerting

### CloudWatch Logs

The following log streams are available:
- PostgreSQL error logs
- PostgreSQL slow query logs
- Database upgrade logs

### Performance Insights

Enabled by default with 7-day retention (free tier). Shows:
- Top SQL queries by execution time
- Database load metrics
- Wait events analysis

### Enhanced Monitoring

60-second granularity metrics for:
- CPU utilization
- Memory usage
- Disk I/O
- Network throughput
- Database connections
- Replication lag (Multi-AZ)

### Recommended CloudWatch Alarms

Create alarms for:
- CPU utilization > 80%
- Free storage space < 10 GB
- Database connections > 80% of max_connections
- Replication lag > 1 second (Multi-AZ)
- Read/write IOPS approaching limits

## Backup and Recovery

### Automated Backups

- **Retention**: Configurable (default 30 days)
- **Backup window**: 03:00-04:00 UTC (daily)
- **Point-in-time recovery**: Available for entire retention period
- **Final snapshot**: Created automatically on instance deletion

### Manual Snapshots

```bash
# Create manual snapshot
aws rds create-db-snapshot \
  --db-instance-identifier <instance_id> \
  --db-snapshot-identifier <snapshot_name>

# Restore from snapshot
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier <new_instance_id> \
  --db-snapshot-identifier <snapshot_name>
```

## Security

### Encryption

- **At rest**: AES-256 encryption using KMS with automatic key rotation
- **In transit**: SSL/TLS required (sslmode=require)
- **Performance Insights**: Encrypted with same KMS key

### Network Security

- Deployed in **private subnets** only
- **No public accessibility**
- Access controlled via security groups
- VPC endpoints recommended for AWS service access

### Credentials Management

- Passwords generated with 32-character complexity
- Stored in **AWS Secrets Manager** with encryption
- Automatic rotation supported (configure separately)
- IAM authentication supported (enable separately)

## Cost Optimization

### Development/Staging

```hcl
instance_class        = "db.t4g.medium"    # ~$50/month
allocated_storage     = 50
multi_az              = false
deletion_protection   = false
backup_retention_period = 7
performance_insights_retention_period = 7  # Free tier
```

### Production

```hcl
instance_class        = "db.r6g.large"     # ~$350/month Multi-AZ
allocated_storage     = 100
multi_az              = true
deletion_protection   = true
backup_retention_period = 30
performance_insights_retention_period = 7  # Free tier
```

### Cost-Saving Tips

1. Use **Graviton2 instances** (r6g, t4g) for 20% cost savings
2. Enable **storage autoscaling** instead of over-provisioning
3. Use **gp3 storage** instead of io1/io2 for better cost/performance
4. Leverage **Reserved Instances** for production (up to 60% savings)
5. Keep Performance Insights at **7-day retention** (free)

## Maintenance

### Maintenance Window

- **Scheduled**: Monday 04:00-05:00 UTC
- **Auto minor upgrades**: Enabled
- **Apply immediately**: Only in non-production environments

### Major Version Upgrades

```bash
# Upgrade to PostgreSQL 16 (when TimescaleDB supports it)
aws rds modify-db-instance \
  --db-instance-identifier <instance_id> \
  --engine-version 16.x \
  --allow-major-version-upgrade \
  --apply-immediately
```

## Troubleshooting

### Connection Issues

1. Verify security group allows inbound on port 5432
2. Check NACLs on private subnets
3. Ensure client is in VPC or using VPN/bastion
4. Verify SSL/TLS configuration

### Performance Issues

1. Check Performance Insights for slow queries
2. Review parameter group settings
3. Analyze CloudWatch metrics (CPU, IOPS, connections)
4. Consider scaling instance class or enabling read replicas

### Storage Issues

1. Monitor free storage space (should auto-scale if configured)
2. Check storage IOPS limits
3. Review log retention settings

## Inputs

| Name | Type | Default | Description |
|------|------|---------|-------------|
| environment | string | required | Environment name (dev, staging, production) |
| project_name | string | "clever-better" | Project name for resource naming |
| instance_class | string | "db.r6g.large" | RDS instance class |
| allocated_storage | number | 100 | Initial storage in GB |
| max_allocated_storage | number | 500 | Maximum storage for autoscaling |
| database_name | string | "clever_better" | Initial database name |
| master_username | string | "admin" | Master username |
| multi_az | bool | true | Enable Multi-AZ deployment |
| backup_retention_period | number | 30 | Backup retention in days |
| deletion_protection | bool | true | Enable deletion protection |
| enable_performance_insights | bool | true | Enable Performance Insights |
| performance_insights_retention_period | number | 7 | PI retention in days |
| vpc_id | string | required | VPC ID |
| subnet_ids | list(string) | required | Private data subnet IDs |
| security_group_ids | list(string) | required | Security group IDs |
| monitoring_role_arn | string | required | IAM role for enhanced monitoring |
| tags | map(string) | {} | Additional resource tags |

## Outputs

| Name | Description |
|------|-------------|
| db_instance_id | RDS instance identifier |
| db_instance_endpoint | RDS endpoint (includes port) |
| db_instance_address | RDS hostname |
| db_instance_port | RDS port |
| db_instance_arn | RDS instance ARN |
| db_subnet_group_name | DB subnet group name |
| db_parameter_group_name | DB parameter group name |
| kms_key_id | KMS key ID |
| kms_key_arn | KMS key ARN |
| secret_arn | Secrets Manager secret ARN |
| secret_name | Secrets Manager secret name |

## References

- [Amazon RDS for PostgreSQL](https://aws.amazon.com/rds/postgresql/)
- [TimescaleDB Documentation](https://docs.timescale.com/)
- [PostgreSQL 15 Release Notes](https://www.postgresql.org/docs/15/release-15.html)
- [AWS RDS Best Practices](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_BestPractices.html)
