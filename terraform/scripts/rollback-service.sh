#!/usr/bin/env bash
# =============================================================================
# Rollback Service Script
# Rolls back a service to the previous task definition
# Usage: ./rollback-service.sh <environment> <service>
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

# Validate arguments
if [ $# -lt 2 ]; then
  echo "Usage: $0 <environment> <service>"
  echo ""
  echo "Arguments:"
  echo "  environment  - dev, staging, or production"
  echo "  service      - bot, ml-service, or data-ingestion"
  echo ""
  echo "Examples:"
  echo "  $0 dev bot"
  echo "  $0 production ml-service"
  exit 1
fi

ENVIRONMENT=$1
SERVICE=$2

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|staging|production)$ ]]; then
  log_error "Invalid environment: $ENVIRONMENT"
  exit 1
fi

# Validate service
if [[ ! "$SERVICE" =~ ^(bot|ml-service|data-ingestion)$ ]]; then
  log_error "Invalid service: $SERVICE"
  exit 1
fi

# Get AWS configuration
AWS_REGION=${AWS_REGION:-us-east-1}
PROJECT_NAME="clever-better"
TASK_FAMILY="${PROJECT_NAME}-${ENVIRONMENT}-${SERVICE}"
SERVICE_NAME="${PROJECT_NAME}-${ENVIRONMENT}-${SERVICE}"
CLUSTER_NAME="${PROJECT_NAME}-${ENVIRONMENT}"

log_info "Rollback Configuration:"
log_info "  Environment: $ENVIRONMENT"
log_info "  Service: $SERVICE"
log_info "  Task Family: $TASK_FAMILY"
echo ""

# Get current task definition
log_step "Getting current task definition..."
CURRENT_TASK_DEF=$(aws ecs describe-services \
  --cluster "$CLUSTER_NAME" \
  --services "$SERVICE_NAME" \
  --query 'services[0].taskDefinition' \
  --output text \
  --region "$AWS_REGION" 2>/dev/null || echo "")

if [ -z "$CURRENT_TASK_DEF" ] || [ "$CURRENT_TASK_DEF" = "None" ]; then
  log_error "Service not found or has no task definition"
  exit 1
fi

CURRENT_REVISION=$(echo "$CURRENT_TASK_DEF" | grep -oE '[0-9]+$')
log_info "Current task definition: $TASK_FAMILY:$CURRENT_REVISION"

# Get previous revision
PREVIOUS_REVISION=$((CURRENT_REVISION - 1))

if [ "$PREVIOUS_REVISION" -lt 1 ]; then
  log_error "No previous revision available (current is revision 1)"
  exit 1
fi

PREVIOUS_TASK_DEF="${TASK_FAMILY}:${PREVIOUS_REVISION}"

# Verify previous revision exists
log_step "Verifying previous task definition exists..."
PREVIOUS_STATUS=$(aws ecs describe-task-definition \
  --task-definition "$PREVIOUS_TASK_DEF" \
  --query 'taskDefinition.status' \
  --output text \
  --region "$AWS_REGION" 2>/dev/null || echo "INACTIVE")

if [ "$PREVIOUS_STATUS" != "ACTIVE" ]; then
  log_error "Previous task definition $PREVIOUS_TASK_DEF is not active"
  log_error "Status: $PREVIOUS_STATUS"
  exit 1
fi

log_info "Previous task definition: $PREVIOUS_TASK_DEF (status: $PREVIOUS_STATUS)"

# Get image information for both versions
CURRENT_IMAGE=$(aws ecs describe-task-definition \
  --task-definition "$CURRENT_TASK_DEF" \
  --query 'taskDefinition.containerDefinitions[0].image' \
  --output text \
  --region "$AWS_REGION")

PREVIOUS_IMAGE=$(aws ecs describe-task-definition \
  --task-definition "$PREVIOUS_TASK_DEF" \
  --query 'taskDefinition.containerDefinitions[0].image' \
  --output text \
  --region "$AWS_REGION")

echo ""
log_info "Current image:  $CURRENT_IMAGE"
log_info "Previous image: $PREVIOUS_IMAGE"
echo ""

# Confirm rollback
read -rp "Proceed with rollback? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
  log_warn "Rollback cancelled"
  exit 0
fi

# Perform rollback
if [ "$SERVICE" != "data-ingestion" ]; then
  log_step "Rolling back service to previous task definition..."
  aws ecs update-service \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-definition "$PREVIOUS_TASK_DEF" \
    --region "$AWS_REGION" > /dev/null

  # Wait for rollback to stabilize
  log_info "Waiting for rollback to stabilize..."
  aws ecs wait services-stable \
    --cluster "$CLUSTER_NAME" \
    --services "$SERVICE_NAME" \
    --region "$AWS_REGION"

  log_info "Rollback complete!"
else
  log_info "Data ingestion is a scheduled task. Update complete."
  log_info "Next scheduled run will use: $PREVIOUS_TASK_DEF"
fi

# Verify service health
log_step "Verifying service health..."
SERVICE_STATUS=$(aws ecs describe-services \
  --cluster "$CLUSTER_NAME" \
  --services "$SERVICE_NAME" \
  --query 'services[0].{running: runningCount, desired: desiredCount, status: status}' \
  --output json \
  --region "$AWS_REGION")

log_info "Service status:"
echo "$SERVICE_STATUS" | jq .

echo ""
log_info "Rollback completed successfully!"
