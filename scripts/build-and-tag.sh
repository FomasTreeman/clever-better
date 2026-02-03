#!/usr/bin/env bash
# =============================================================================
# Build and Tag Docker Images
# =============================================================================
# Script to build Docker images with proper versioning for CI/CD
#
# Usage:
#   ./scripts/build-and-tag.sh <service> <environment> [version]
#
# Arguments:
#   service     - Service name: bot, ml-service, data-ingestion
#   environment - Environment: dev, staging, production
#   version     - Optional version tag (defaults to git describe)
#
# Examples:
#   ./scripts/build-and-tag.sh bot staging v1.2.3
#   ./scripts/build-and-tag.sh ml-service production
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Validate arguments
if [[ $# -lt 2 ]]; then
    log_error "Usage: $0 <service> <environment> [version]"
    log_error "  service: bot, ml-service, data-ingestion"
    log_error "  environment: dev, staging, production"
    exit 1
fi

SERVICE="$1"
ENVIRONMENT="$2"
VERSION="${3:-}"

# Validate service
case "$SERVICE" in
    bot|ml-service|data-ingestion)
        ;;
    *)
        log_error "Invalid service: $SERVICE"
        log_error "Valid services: bot, ml-service, data-ingestion"
        exit 1
        ;;
esac

# Validate environment
case "$ENVIRONMENT" in
    dev|staging|production)
        ;;
    *)
        log_error "Invalid environment: $ENVIRONMENT"
        log_error "Valid environments: dev, staging, production"
        exit 1
        ;;
esac

# Get version info
if [[ -z "$VERSION" ]]; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev-$(git rev-parse --short HEAD)")
fi

GIT_SHA=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Project configuration
PROJECT_NAME="clever-better"
AWS_REGION="${AWS_REGION:-eu-west-1}"
AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "")}"

if [[ -z "$AWS_ACCOUNT_ID" ]]; then
    log_warn "AWS_ACCOUNT_ID not set and unable to detect. Using local build only."
    ECR_REGISTRY=""
else
    ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
fi

# Determine Dockerfile and context
case "$SERVICE" in
    bot|data-ingestion)
        DOCKERFILE="Dockerfile"
        CONTEXT="."
        ;;
    ml-service)
        DOCKERFILE="ml-service/Dockerfile"
        CONTEXT="ml-service"
        ;;
esac

# Image naming
IMAGE_NAME="${PROJECT_NAME}-${ENVIRONMENT}-${SERVICE}"
LOCAL_IMAGE="${IMAGE_NAME}:${VERSION}"

log_info "Building Docker image for ${SERVICE}"
log_info "  Environment: ${ENVIRONMENT}"
log_info "  Version: ${VERSION}"
log_info "  Git SHA: ${GIT_SHA}"
log_info "  Build Date: ${BUILD_DATE}"

# Build the image
log_info "Building image: ${LOCAL_IMAGE}"

docker build \
    --build-arg VERSION="${VERSION}" \
    --build-arg GIT_COMMIT="${GIT_SHA}" \
    --build-arg BUILD_DATE="${BUILD_DATE}" \
    --label "version=${VERSION}" \
    --label "git.commit=${GIT_SHA}" \
    --label "git.branch=${BRANCH}" \
    --label "build.date=${BUILD_DATE}" \
    --label "service=${SERVICE}" \
    --label "environment=${ENVIRONMENT}" \
    -t "${LOCAL_IMAGE}" \
    -f "${DOCKERFILE}" \
    "${CONTEXT}"

log_info "Successfully built: ${LOCAL_IMAGE}"

# Generate all tags
TAGS=(
    "${VERSION}"
    "${GIT_SHA}"
    "latest-${ENVIRONMENT}"
)

# Add semantic version tags for tagged releases
if [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    MAJOR=$(echo "$VERSION" | cut -d. -f1)
    MINOR=$(echo "$VERSION" | cut -d. -f1-2)
    TAGS+=("${MAJOR}" "${MINOR}")
fi

# Tag the image locally
log_info "Tagging image with:"
for TAG in "${TAGS[@]}"; do
    docker tag "${LOCAL_IMAGE}" "${IMAGE_NAME}:${TAG}"
    log_info "  - ${IMAGE_NAME}:${TAG}"
done

# Push to ECR if registry is available
if [[ -n "$ECR_REGISTRY" ]]; then
    log_info "Tagging for ECR:"

    for TAG in "${TAGS[@]}"; do
        ECR_IMAGE="${ECR_REGISTRY}/${IMAGE_NAME}:${TAG}"
        docker tag "${LOCAL_IMAGE}" "${ECR_IMAGE}"
        log_info "  - ${ECR_IMAGE}"
    done

    # Get image digest
    DIGEST=$(docker inspect --format='{{index .RepoDigests 0}}' "${LOCAL_IMAGE}" 2>/dev/null || echo "")

    if [[ -n "$DIGEST" ]]; then
        log_info "Image digest: ${DIGEST}"
    fi
fi

# Output summary
echo ""
log_info "============================================="
log_info "Build Summary"
log_info "============================================="
log_info "Service:     ${SERVICE}"
log_info "Environment: ${ENVIRONMENT}"
log_info "Version:     ${VERSION}"
log_info "Git SHA:     ${GIT_SHA}"
log_info "Local Image: ${LOCAL_IMAGE}"
if [[ -n "$ECR_REGISTRY" ]]; then
    log_info "ECR Image:   ${ECR_REGISTRY}/${IMAGE_NAME}:${VERSION}"
fi
log_info "============================================="

# Export variables for use in CI
if [[ "${CI:-}" == "true" ]]; then
    echo "IMAGE_NAME=${IMAGE_NAME}" >> "${GITHUB_ENV:-/dev/null}"
    echo "IMAGE_TAG=${VERSION}" >> "${GITHUB_ENV:-/dev/null}"
    echo "GIT_SHA=${GIT_SHA}" >> "${GITHUB_ENV:-/dev/null}"
fi
