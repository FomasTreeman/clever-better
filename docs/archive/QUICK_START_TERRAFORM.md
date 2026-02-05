# Quick Start Guide - Terraform Deployment

## Prerequisites

```bash
# Install tools
brew install terraform awscli pre-commit

# Configure AWS credentials
aws configure
# Enter: Access Key, Secret Key, Region (us-east-1), Output Format (json)

# Verify
terraform version     # Should be ≥1.5.0
aws sts get-caller-identity  # Should show your AWS account
```

## Step 1: Backend Setup (One Time)

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/scripts

# Create S3 bucket and DynamoDB table for state management
./setup-backend.sh

# Output will show bucket name and table name
# Copy these values - you'll need them next
```

## Step 2: Configure Backend (Per Environment)

For each environment (dev, staging, production):

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Copy the example file
cp backend.tf.example backend.tf

# Edit backend.tf and update:
#   - bucket = "clever-better-terraform-state-dev-ACCOUNT_ID"  
#   - dynamodb_table = "clever-better-terraform-lock-dev"
#   - region = "us-east-1"
nano backend.tf
```

## Step 3: Copy Variable Files (Per Environment)

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Copy example to actual file
cp terraform.tfvars.example terraform.tfvars

# Edit with your values (most should work as-is)
nano terraform.tfvars
```

## Step 4: Initialize Terraform (Per Environment)

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Download required providers and modules
terraform init

# Verify - should show "Terraform has been successfully configured!"
```

## Step 5: Validate & Plan (Per Environment)

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Format check
terraform fmt -check .

# Syntax validation
terraform validate

# Show what will be created
terraform plan -out=tfplan

# Review the output carefully!
```

## Step 6: Apply (Per Environment)

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Deploy infrastructure
terraform apply tfplan

# This will take 5-10 minutes
# Watch for errors - note any failed resources
```

## Step 7: Verify Outputs

```bash
cd /Users/tom/Personal/DevSecOps/clever-better/terraform/environments/dev

# Show all created resource IDs
terraform output

# Save these - needed for RDS and ECS modules:
# - vpc_id
# - public_subnet_ids
# - app_subnet_ids  
# - data_subnet_ids
# - alb_security_group_id
# - application_security_group_id
# - database_security_group_id
```

## Step 8: Populate Secrets

Once VPC is deployed, add credentials to Secrets Manager:

```bash
# Database credentials
aws secretsmanager put-secret-value \
  --secret-id clever-better/dev/database \
  --secret-string '{
    "username": "postgres",
    "password": "SECURE_PASSWORD",
    "host": "RDS_ENDPOINT",
    "port": 5432,
    "dbname": "clever_better"
  }'

# Betfair credentials
aws secretsmanager put-secret-value \
  --secret-id clever-better/dev/betfair \
  --secret-string '{
    "username": "USERNAME",
    "password": "PASSWORD",
    "app_key": "APP_KEY",
    "certificate_path": "/app/certs/betfair.pem"
  }'

# API Keys
aws secretsmanager put-secret-value \
  --secret-id clever-better/dev/api-keys \
  --secret-string '{
    "racing_post_api_key": "YOUR_KEY"
  }'
```

## Makefile Shortcuts

Instead of `cd terraform/environments/dev && terraform ...`, use:

```bash
cd /Users/tom/Personal/DevSecOps/clever-better

# Initialize all environments at once
make tf-init-all

# Validate all environments
make tf-validate-all

# Plan specific environment
make tf-plan-dev       # Dev
make tf-plan-staging   # Staging
make tf-plan-production # Production

# Apply specific environment
make tf-apply-dev
make tf-apply-staging
make tf-apply-production

# Show outputs
make tf-output-dev
make tf-output-staging
make tf-output-production

# Setup backend
make tf-setup-backend
```

## Troubleshooting

### "Backend bucket not found"
**Fix**: Run `./scripts/setup-backend.sh` first

### "Provider requirements not met"
**Fix**: Run `terraform init` in the environment directory

### "Security group creation failed - limits"
**Fix**: Check AWS quotas (EC2 → Limits → Security groups per VPC)

### "State lock timeout"
**Fix**: Check DynamoDB table exists and has read/write capacity

### "Cannot assume role"
**Fix**: Verify IAM user has permissions in AWS account

## Important Files to Know

| File | Purpose |
|------|---------|
| `terraform/versions.tf` | Provider versions |
| `terraform/backend.tf.example` | Backend template |
| `terraform/environments/dev/main.tf` | Module references |
| `terraform/environments/dev/terraform.tfvars` | Variable values |
| `terraform/modules/vpc/main.tf` | Network resources |
| `terraform/modules/security/main.tf` | Security groups |
| `terraform/scripts/setup-backend.sh` | Create S3 + DynamoDB |

## Environment Differences

| Setting | Dev | Staging | Prod |
|---------|-----|---------|------|
| VPC CIDR | 10.0.0.0/16 | 10.1.0.0/16 | 10.2.0.0/16 |
| WAF Limit | 5000 req/5min | 3000 req/5min | 2000 req/5min |
| WAF Logging | Off | On | On |
| Secrets Rotation | Off | On | On |

## Getting Help

1. **Validation errors**: Check `.tflint.hcl` rules and AWS Terraform provider docs
2. **Plan changes**: Review `terraform plan` output carefully before apply
3. **Resource not found**: Verify module outputs in `outputs.tf` are exported
4. **State issues**: See `terraform/ROLLBACK.md` for recovery procedures

## When Done

After all environments deployed:
1. Verify all outputs captured
2. Populate all secrets in Secrets Manager
3. Test VPC connectivity (bastion host, RDS port 5432)
4. Move to RDS module implementation
5. Then ECS module implementation

---

**Status**: ✅ All infrastructure ready for deployment
**Total Deploy Time**: ~30-45 minutes (all 3 environments)
