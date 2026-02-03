#!/usr/bin/env bash
# =============================================================================
# Health Check Script
# =============================================================================
# Verify health endpoints are responding correctly for CI/CD pipelines
#
# Usage:
#   ./scripts/health-check.sh <url> [timeout] [retries]
#
# Arguments:
#   url     - Health check URL (e.g., http://localhost:8080/health)
#   timeout - Request timeout in seconds (default: 5)
#   retries - Number of retry attempts (default: 3)
#
# Exit Codes:
#   0 - Health check passed
#   1 - Health check failed
#   2 - Invalid arguments
#
# Examples:
#   ./scripts/health-check.sh http://localhost:8080/health
#   ./scripts/health-check.sh https://api.example.com/health 10 5
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

# Default values
DEFAULT_TIMEOUT=5
DEFAULT_RETRIES=3
RETRY_DELAY=5

# Parse arguments
if [[ $# -lt 1 ]]; then
    log_error "Usage: $0 <url> [timeout] [retries]"
    log_error "  url     - Health check URL"
    log_error "  timeout - Request timeout in seconds (default: ${DEFAULT_TIMEOUT})"
    log_error "  retries - Number of retry attempts (default: ${DEFAULT_RETRIES})"
    exit 2
fi

URL="$1"
TIMEOUT="${2:-$DEFAULT_TIMEOUT}"
RETRIES="${3:-$DEFAULT_RETRIES}"

# Validate URL format
if ! [[ "$URL" =~ ^https?:// ]]; then
    log_error "Invalid URL format: $URL"
    log_error "URL must start with http:// or https://"
    exit 2
fi

log_info "Health check configuration:"
log_info "  URL:     $URL"
log_info "  Timeout: ${TIMEOUT}s"
log_info "  Retries: $RETRIES"
log_info ""

# Health check function
check_health() {
    local attempt=$1

    log_info "Attempt $attempt/$RETRIES: Checking $URL"

    # Make the request
    HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" \
        --connect-timeout "$TIMEOUT" \
        --max-time "$TIMEOUT" \
        "$URL" 2>/dev/null || echo "")

    if [[ -z "$HTTP_RESPONSE" ]]; then
        log_error "Connection failed or timed out"
        return 1
    fi

    # Extract status code and body
    HTTP_CODE=$(echo "$HTTP_RESPONSE" | tail -n1)
    HTTP_BODY=$(echo "$HTTP_RESPONSE" | sed '$d')

    log_info "  Response code: $HTTP_CODE"

    # Check for successful response
    if [[ "$HTTP_CODE" == "200" ]]; then
        # Validate JSON response if possible
        if command -v jq &> /dev/null; then
            if echo "$HTTP_BODY" | jq -e '.status' &> /dev/null; then
                STATUS=$(echo "$HTTP_BODY" | jq -r '.status')
                SERVICE=$(echo "$HTTP_BODY" | jq -r '.service // "unknown"')
                log_info "  Status: $STATUS"
                log_info "  Service: $SERVICE"

                if [[ "$STATUS" == "ok" || "$STATUS" == "healthy" ]]; then
                    return 0
                else
                    log_warn "  Health status is not 'ok': $STATUS"
                    return 1
                fi
            fi
        fi

        # No jq or invalid JSON, just check HTTP status
        log_info "  Response body: $HTTP_BODY"
        return 0
    fi

    log_error "  Unexpected status code: $HTTP_CODE"
    log_error "  Response: $HTTP_BODY"
    return 1
}

# Main retry loop
attempt=1
while [[ $attempt -le $RETRIES ]]; do
    if check_health $attempt; then
        log_info ""
        log_info "============================================="
        log_info "Health check PASSED"
        log_info "============================================="
        exit 0
    fi

    if [[ $attempt -lt $RETRIES ]]; then
        log_warn "Retrying in ${RETRY_DELAY}s..."
        sleep "$RETRY_DELAY"
    fi

    ((attempt++))
done

# All retries exhausted
log_error ""
log_error "============================================="
log_error "Health check FAILED after $RETRIES attempts"
log_error "============================================="
exit 1
