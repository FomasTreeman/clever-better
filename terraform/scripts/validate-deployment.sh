#!/usr/bin/env bash
# =============================================================================
# Validate Deployment Script
# Validates the health and status of ECS services
# Usage: ./validate-deployment.sh <environment>
# =============================================================================
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; }

# Validate arguments
if [ $# -lt 1 ]; then
  echo "Usage: $0 <environment>"
  echo ""
  echo "Arguments:"
  echo "  environment  - dev, staging, or production"
  echo ""
  echo "Examples:"
  echo "  $0 dev"
  echo "  $0 production"
  exit 1
fi

ENVIRONMENT=$1

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|staging|production)$ ]]; then
  log_error "Invalid environment: $ENVIRONMENT"
  exit 1
fi

# Get AWS configuration
AWS_REGION=${AWS_REGION:-us-east-1}
PROJECT_NAME="clever-better"
CLUSTER_NAME="${PROJECT_NAME}-${ENVIRONMENT}"

ERRORS=0

echo "=============================================="
echo "Deployment Validation Report"
echo "Environment: $ENVIRONMENT"
echo "Date: $(date)"
echo "=============================================="
echo ""

# Check ECS Cluster
log_step "Checking ECS Cluster..."
CLUSTER_STATUS=$(aws ecs describe-clusters \
  --clusters "$CLUSTER_NAME" \
  --query 'clusters[0].status' \
  --output text \
  --region "$AWS_REGION" 2>/dev/null || echo "NOT_FOUND")

if [ "$CLUSTER_STATUS" = "ACTIVE" ]; then
  log_pass "ECS Cluster: $CLUSTER_NAME is ACTIVE"
else
  log_fail "ECS Cluster: $CLUSTER_NAME status is $CLUSTER_STATUS"
  ERRORS=$((ERRORS + 1))
fi

echo ""

# Check Services
log_step "Checking ECS Services..."
SERVICES=("bot" "ml-service")

for svc in "${SERVICES[@]}"; do
  SERVICE_NAME="${PROJECT_NAME}-${ENVIRONMENT}-${svc}"

  SERVICE_INFO=$(aws ecs describe-services \
    --cluster "$CLUSTER_NAME" \
    --services "$SERVICE_NAME" \
    --query 'services[0].{status: status, running: runningCount, desired: desiredCount, pending: pendingCount}' \
    --output json \
    --region "$AWS_REGION" 2>/dev/null || echo '{"status": "NOT_FOUND"}')

  STATUS=$(echo "$SERVICE_INFO" | jq -r '.status')
  RUNNING=$(echo "$SERVICE_INFO" | jq -r '.running // 0')
  DESIRED=$(echo "$SERVICE_INFO" | jq -r '.desired // 0')

  if [ "$STATUS" = "ACTIVE" ] && [ "$RUNNING" = "$DESIRED" ] && [ "$RUNNING" != "0" ]; then
    log_pass "Service $svc: HEALTHY ($RUNNING/$DESIRED tasks running)"
  elif [ "$STATUS" = "ACTIVE" ]; then
    log_warn "Service $svc: DEGRADED ($RUNNING/$DESIRED tasks running)"
    ERRORS=$((ERRORS + 1))
  else
    log_fail "Service $svc: $STATUS"
    ERRORS=$((ERRORS + 1))
  fi
done

echo ""

# Check Task Definitions
log_step "Checking Task Definitions..."
TASK_FAMILIES=("bot" "ml-service" "data-ingestion")

for family in "${TASK_FAMILIES[@]}"; do
  TASK_FAMILY="${PROJECT_NAME}-${ENVIRONMENT}-${family}"

  TASK_STATUS=$(aws ecs describe-task-definition \
    --task-definition "$TASK_FAMILY" \
    --query 'taskDefinition.status' \
    --output text \
    --region "$AWS_REGION" 2>/dev/null || echo "NOT_FOUND")

  if [ "$TASK_STATUS" = "ACTIVE" ]; then
    REVISION=$(aws ecs describe-task-definition \
      --task-definition "$TASK_FAMILY" \
      --query 'taskDefinition.revision' \
      --output text \
      --region "$AWS_REGION")
    log_pass "Task Definition $family: revision $REVISION"
  else
    log_fail "Task Definition $family: $TASK_STATUS"
    ERRORS=$((ERRORS + 1))
  fi
done

echo ""

# Check ECR Repositories
log_step "Checking ECR Repositories..."
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPOS=("bot" "ml-service" "data-ingestion")

for repo in "${ECR_REPOS[@]}"; do
  REPO_NAME="${PROJECT_NAME}-${ENVIRONMENT}-${repo}"

  REPO_URI=$(aws ecr describe-repositories \
    --repository-names "$REPO_NAME" \
    --query 'repositories[0].repositoryUri' \
    --output text \
    --region "$AWS_REGION" 2>/dev/null || echo "NOT_FOUND")

  if [ "$REPO_URI" != "NOT_FOUND" ]; then
    IMAGE_COUNT=$(aws ecr list-images \
      --repository-name "$REPO_NAME" \
      --query 'length(imageIds)' \
      --output text \
      --region "$AWS_REGION" 2>/dev/null || echo "0")
    log_pass "ECR Repository $repo: $IMAGE_COUNT images"
  else
    log_fail "ECR Repository $repo: NOT_FOUND"
    ERRORS=$((ERRORS + 1))
  fi
done

echo ""

# Check CloudWatch Logs
log_step "Checking CloudWatch Log Groups..."
LOG_GROUPS=("bot" "ml-service" "data-ingestion")

for lg in "${LOG_GROUPS[@]}"; do
  LOG_GROUP_NAME="/ecs/${PROJECT_NAME}-${ENVIRONMENT}/${lg}"

  LOG_STATUS=$(aws logs describe-log-groups \
    --log-group-name-prefix "$LOG_GROUP_NAME" \
    --query 'logGroups[0].logGroupName' \
    --output text \
    --region "$AWS_REGION" 2>/dev/null || echo "NOT_FOUND")

  if [ "$LOG_STATUS" != "NOT_FOUND" ] && [ "$LOG_STATUS" != "None" ]; then
    log_pass "Log Group $lg: exists"
  else
    log_warn "Log Group $lg: NOT_FOUND"
  fi
done

echo ""

# Summary
echo "=============================================="
echo "Validation Summary"
echo "=============================================="

if [ $ERRORS -eq 0 ]; then
  log_info "All checks passed!"
  exit 0
else
  log_error "$ERRORS check(s) failed"
  exit 1
fi
