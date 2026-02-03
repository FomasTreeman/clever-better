#!/usr/bin/env bash
# =============================================================================
# Wait for ECS Deployment Script
# =============================================================================
# Wait for ECS service deployment to complete and verify health
#
# Usage:
#   ./scripts/wait-for-deployment.sh <cluster> <service> [timeout]
#
# Arguments:
#   cluster - ECS cluster name
#   service - ECS service name
#   timeout - Maximum wait time in seconds (default: 600)
#
# Exit Codes:
#   0 - Deployment successful
#   1 - Deployment failed or timed out
#   2 - Invalid arguments
#
# Examples:
#   ./scripts/wait-for-deployment.sh clever-better-staging bot
#   ./scripts/wait-for-deployment.sh clever-better-production ml-service 900
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# Default values
DEFAULT_TIMEOUT=600
CHECK_INTERVAL=15

# Parse arguments
if [[ $# -lt 2 ]]; then
    log_error "Usage: $0 <cluster> <service> [timeout]"
    log_error "  cluster - ECS cluster name"
    log_error "  service - ECS service name"
    log_error "  timeout - Maximum wait time in seconds (default: ${DEFAULT_TIMEOUT})"
    exit 2
fi

CLUSTER="$1"
SERVICE="$2"
TIMEOUT="${3:-$DEFAULT_TIMEOUT}"

# AWS Region
AWS_REGION="${AWS_REGION:-eu-west-1}"

log_info "Waiting for ECS deployment"
log_info "  Cluster: $CLUSTER"
log_info "  Service: $SERVICE"
log_info "  Timeout: ${TIMEOUT}s"
log_info "  Region:  $AWS_REGION"
log_info ""

# Check if service exists
check_service_exists() {
    aws ecs describe-services \
        --cluster "$CLUSTER" \
        --services "$SERVICE" \
        --region "$AWS_REGION" \
        --query 'services[0].serviceName' \
        --output text 2>/dev/null
}

# Get service status
get_service_status() {
    aws ecs describe-services \
        --cluster "$CLUSTER" \
        --services "$SERVICE" \
        --region "$AWS_REGION" \
        --query 'services[0]' \
        --output json 2>/dev/null
}

# Get deployment status
get_deployment_status() {
    local service_json="$1"

    local running_count=$(echo "$service_json" | jq -r '.runningCount // 0')
    local desired_count=$(echo "$service_json" | jq -r '.desiredCount // 0')
    local pending_count=$(echo "$service_json" | jq -r '.pendingCount // 0')

    local primary_status=$(echo "$service_json" | jq -r '.deployments[] | select(.status == "PRIMARY") | .rolloutState // "UNKNOWN"')
    local primary_running=$(echo "$service_json" | jq -r '.deployments[] | select(.status == "PRIMARY") | .runningCount // 0')
    local primary_desired=$(echo "$service_json" | jq -r '.deployments[] | select(.status == "PRIMARY") | .desiredCount // 0')

    echo "running=$running_count desired=$desired_count pending=$pending_count status=$primary_status primary_running=$primary_running primary_desired=$primary_desired"
}

# Check if deployment is complete
is_deployment_complete() {
    local service_json="$1"

    # Check if there's only one deployment (PRIMARY)
    local deployment_count=$(echo "$service_json" | jq '.deployments | length')

    if [[ "$deployment_count" -gt 1 ]]; then
        return 1
    fi

    # Check if PRIMARY deployment is complete
    local primary_status=$(echo "$service_json" | jq -r '.deployments[] | select(.status == "PRIMARY") | .rolloutState // "UNKNOWN"')
    local running_count=$(echo "$service_json" | jq -r '.runningCount // 0')
    local desired_count=$(echo "$service_json" | jq -r '.desiredCount // 0')

    if [[ "$primary_status" == "COMPLETED" ]] && [[ "$running_count" -eq "$desired_count" ]] && [[ "$desired_count" -gt 0 ]]; then
        return 0
    fi

    return 1
}

# Check if deployment failed
is_deployment_failed() {
    local service_json="$1"

    local primary_status=$(echo "$service_json" | jq -r '.deployments[] | select(.status == "PRIMARY") | .rolloutState // "UNKNOWN"')

    if [[ "$primary_status" == "FAILED" ]]; then
        return 0
    fi

    return 1
}

# Verify service exists
SERVICE_NAME=$(check_service_exists)
if [[ -z "$SERVICE_NAME" || "$SERVICE_NAME" == "None" ]]; then
    log_error "Service '$SERVICE' not found in cluster '$CLUSTER'"
    exit 1
fi

log_info "Service found: $SERVICE_NAME"
log_info ""

# Main wait loop
start_time=$(date +%s)
last_status=""

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))

    if [[ $elapsed -ge $TIMEOUT ]]; then
        log_error ""
        log_error "============================================="
        log_error "Deployment TIMED OUT after ${TIMEOUT}s"
        log_error "============================================="
        exit 1
    fi

    remaining=$((TIMEOUT - elapsed))

    # Get current status
    SERVICE_JSON=$(get_service_status)

    if [[ -z "$SERVICE_JSON" ]]; then
        log_warn "Failed to get service status, retrying..."
        sleep "$CHECK_INTERVAL"
        continue
    fi

    # Get deployment status
    status=$(get_deployment_status "$SERVICE_JSON")

    # Only log if status changed
    if [[ "$status" != "$last_status" ]]; then
        log_info "[${elapsed}s] $status"
        last_status="$status"
    else
        log_debug "[${elapsed}s] Waiting... (${remaining}s remaining)"
    fi

    # Check if deployment failed
    if is_deployment_failed "$SERVICE_JSON"; then
        log_error ""
        log_error "============================================="
        log_error "Deployment FAILED"
        log_error "============================================="

        # Get failure reason
        EVENTS=$(echo "$SERVICE_JSON" | jq -r '.events[:5][] | "\(.createdAt): \(.message)"')
        log_error "Recent events:"
        echo "$EVENTS" | while read -r event; do
            log_error "  $event"
        done

        exit 1
    fi

    # Check if deployment is complete
    if is_deployment_complete "$SERVICE_JSON"; then
        log_info ""
        log_info "============================================="
        log_info "Deployment SUCCESSFUL"
        log_info "============================================="

        # Get task info
        TASK_DEF=$(echo "$SERVICE_JSON" | jq -r '.taskDefinition')
        RUNNING=$(echo "$SERVICE_JSON" | jq -r '.runningCount')

        log_info "Task Definition: $(basename "$TASK_DEF")"
        log_info "Running Tasks:   $RUNNING"
        log_info "Duration:        ${elapsed}s"

        exit 0
    fi

    sleep "$CHECK_INTERVAL"
done
