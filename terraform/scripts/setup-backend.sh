#!/usr/bin/env bash
# =============================================================================
# Setup Terraform Backend
# Creates S3 bucket and DynamoDB table for Terraform state management
# =============================================================================
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
if ! command -v aws >/dev/null 2>&1; then
  log_error "AWS CLI not found. Install AWS CLI and configure credentials."
  exit 1
fi

# Get inputs
read -rp "Environment (dev|staging|production): " ENV
if [[ ! "$ENV" =~ ^(dev|staging|production)$ ]]; then
  log_error "Invalid environment. Must be dev, staging, or production."
  exit 1
fi

read -rp "AWS region (e.g., us-east-1): " REGION

# Resource names
BUCKET="clever-better-terraform-state-${ENV}"
TABLE="clever-better-terraform-locks"

log_info "Creating Terraform backend resources for ${ENV} environment..."
log_info "  S3 Bucket: ${BUCKET}"
log_info "  DynamoDB Table: ${TABLE}"
echo ""

# Create S3 bucket
log_info "Creating S3 bucket..."
if aws s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
  log_warn "Bucket ${BUCKET} already exists, skipping creation."
else
  # Note: us-east-1 requires special handling - no LocationConstraint
  if [ "$REGION" = "us-east-1" ]; then
    aws s3api create-bucket \
      --bucket "$BUCKET" \
      --region "$REGION"
  else
    aws s3api create-bucket \
      --bucket "$BUCKET" \
      --region "$REGION" \
      --create-bucket-configuration LocationConstraint="$REGION"
  fi
  log_info "Bucket ${BUCKET} created."
fi

# Enable versioning
log_info "Enabling bucket versioning..."
aws s3api put-bucket-versioning \
  --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

# Enable encryption
log_info "Enabling bucket encryption..."
aws s3api put-bucket-encryption \
  --bucket "$BUCKET" \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "AES256"
      },
      "BucketKeyEnabled": true
    }]
  }'

# Block public access
log_info "Blocking public access..."
aws s3api put-public-access-block \
  --bucket "$BUCKET" \
  --public-access-block-configuration '{
    "BlockPublicAcls": true,
    "IgnorePublicAcls": true,
    "BlockPublicPolicy": true,
    "RestrictPublicBuckets": true
  }'

# Create DynamoDB table
log_info "Creating DynamoDB table..."
if aws dynamodb describe-table --table-name "$TABLE" --region "$REGION" >/dev/null 2>&1; then
  log_warn "Table ${TABLE} already exists, skipping creation."
else
  aws dynamodb create-table \
    --table-name "$TABLE" \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region "$REGION" \
    --tags Key=Project,Value=clever-better Key=Environment,Value="${ENV}" Key=ManagedBy,Value=terraform

  log_info "Waiting for table to be active..."
  aws dynamodb wait table-exists --table-name "$TABLE" --region "$REGION"
  log_info "Table ${TABLE} created."
fi

# Verify setup
log_info "Verifying backend setup..."

BUCKET_EXISTS=$(aws s3api head-bucket --bucket "$BUCKET" 2>&1 || echo "error")
if [[ "$BUCKET_EXISTS" == *"error"* ]]; then
  log_error "Failed to verify S3 bucket."
  exit 1
fi

TABLE_STATUS=$(aws dynamodb describe-table --table-name "$TABLE" --region "$REGION" --query 'Table.TableStatus' --output text 2>/dev/null || echo "error")
if [ "$TABLE_STATUS" != "ACTIVE" ]; then
  log_error "DynamoDB table is not active. Status: ${TABLE_STATUS}"
  exit 1
fi

echo ""
log_info "Backend setup complete!"
echo ""
echo "Add the following to your terraform/environments/${ENV}/main.tf:"
echo ""
echo "terraform {"
echo "  backend \"s3\" {"
echo "    bucket         = \"${BUCKET}\""
echo "    key            = \"terraform.tfstate\""
echo "    region         = \"${REGION}\""
echo "    dynamodb_table = \"${TABLE}\""
echo "    encrypt        = true"
echo "  }"
echo "}"
echo ""
log_info "Then run: cd terraform/environments/${ENV} && terraform init"
