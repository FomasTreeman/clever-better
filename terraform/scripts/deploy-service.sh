#!/usr/bin/env bash
# =============================================================================
# Deploy Service Script
# Builds, pushes, and deploys a service to the specified environment
# Usage: ./deploy-service.sh <environment> <service> <image-tag>
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
if [ $# -lt 3 ]; then
  echo "Usage: $0 <environment> <service> <image-tag>"
  echo ""
  echo "Arguments:"
  echo "  environment  - dev, staging, or production"
  echo "  service      - bot, ml-service, data-ingestion, or all"
  echo "  image-tag    - Docker image tag (e.g., v1.0.0, latest)"
  echo ""
  echo "Examples:"
  echo "  $0 dev bot latest"
  echo "  $0 staging ml-service v1.2.3"
  echo "  $0 production all v1.0.0"
  exit 1
fi

ENVIRONMENT=$1
SERVICE=$2
IMAGE_TAG=$3

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|staging|production)$ ]]; then
  log_error "Invalid environment: $ENVIRONMENT"
  log_error "Must be one of: dev, staging, production"
  exit 1
fi

# Validate service
if [[ ! "$SERVICE" =~ ^(bot|ml-service|data-ingestion|all)$ ]]; then
  log_error "Invalid service: $SERVICE"
  log_error "Must be one of: bot, ml-service, data-ingestion, all"
  exit 1
fi

# Get AWS account and region
AWS_REGION=${AWS_REGION:-us-east-1}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
PROJECT_NAME="clever-better"

log_info "Deployment Configuration:"
log_info "  Environment: $ENVIRONMENT"
log_info "  Service: $SERVICE"
log_info "  Image Tag: $IMAGE_TAG"
log_info "  ECR Registry: $ECR_REGISTRY"
echo ""

# Function to deploy a single service
deploy_service() {
  local svc=$1
  local tag=$2
  local repo_name="${PROJECT_NAME}-${ENVIRONMENT}-${svc}"
  local image_uri="${ECR_REGISTRY}/${repo_name}:${tag}"

  log_step "Deploying $svc..."

  # Determine Dockerfile and context
  local dockerfile=""
  local context=""
  case $svc in
    bot|data-ingestion)
      dockerfile="Dockerfile"
      context="."
      ;;
    ml-service)
      dockerfile="ml-service/Dockerfile"
      context="ml-service"
      ;;
  esac

  # Build Docker image
  log_info "Building Docker image for $svc..."
  docker build -t "${repo_name}:${tag}" -f "$dockerfile" "$context"

  # Tag for ECR
  log_info "Tagging image for ECR..."
  docker tag "${repo_name}:${tag}" "$image_uri"

  # Push to ECR
  log_info "Pushing image to ECR..."
  docker push "$image_uri"

  # Get current task definition
  local task_family="${PROJECT_NAME}-${ENVIRONMENT}-${svc}"
  log_info "Getting current task definition for $task_family..."

  local task_def
  task_def=$(aws ecs describe-task-definition \
    --task-definition "$task_family" \
    --query 'taskDefinition' \
    --region "$AWS_REGION" 2>/dev/null || echo "")

  if [ -z "$task_def" ]; then
    log_warn "Task definition not found. Run terraform apply first."
    return 1
  fi

  # Update task definition with new image
  log_info "Creating new task definition revision..."
  local new_task_def
  new_task_def=$(echo "$task_def" | jq --arg IMAGE "$image_uri" '
    .containerDefinitions[0].image = $IMAGE |
    del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)
  ')

  local new_task_arn
  new_task_arn=$(aws ecs register-task-definition \
    --cli-input-json "$new_task_def" \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text \
    --region "$AWS_REGION")

  log_info "New task definition: $new_task_arn"

  # Update ECS service (skip for data-ingestion as it's scheduled)
  if [ "$svc" != "data-ingestion" ]; then
    local service_name="${PROJECT_NAME}-${ENVIRONMENT}-${svc}"
    local cluster_name="${PROJECT_NAME}-${ENVIRONMENT}"

    log_info "Updating ECS service $service_name..."
    aws ecs update-service \
      --cluster "$cluster_name" \
      --service "$service_name" \
      --task-definition "$new_task_arn" \
      --region "$AWS_REGION" > /dev/null

    # Wait for deployment to stabilize
    log_info "Waiting for deployment to stabilize..."
    aws ecs wait services-stable \
      --cluster "$cluster_name" \
      --services "$service_name" \
      --region "$AWS_REGION"

    log_info "Service $svc deployed successfully!"
  else
    log_info "Data ingestion task definition updated. Next scheduled run will use the new image."
  fi

  echo ""
}

# Login to ECR
log_step "Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "$ECR_REGISTRY"
echo ""

# Deploy services
if [ "$SERVICE" = "all" ]; then
  deploy_service "bot" "$IMAGE_TAG"
  deploy_service "ml-service" "$IMAGE_TAG"
  deploy_service "data-ingestion" "$IMAGE_TAG"
else
  deploy_service "$SERVICE" "$IMAGE_TAG"
fi

log_info "Deployment complete!"
