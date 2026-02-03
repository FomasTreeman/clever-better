# =============================================================================
# Clever Better - Multi-stage Go Application Dockerfile
# Builds all Go binaries for the trading bot and related services
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy entire source code
COPY . .

# Build all binaries with static linking and optimized flags
# CGO_ENABLED=0 for static binaries, -ldflags="-w -s" for smaller size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/bot ./cmd/bot

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/data-ingestion ./cmd/data-ingestion

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/backtest ./cmd/backtest

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/ml-feedback ./cmd/ml-feedback

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/strategy-discovery ./cmd/strategy-discovery

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/ml-status ./cmd/ml-status

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /build/bin/ /app/bin/

# Copy configuration directory
COPY --from=builder /build/config/ /app/config/

# Ensure binaries are executable
RUN chmod +x /app/bin/*

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Expose health check port
EXPOSE 8080

# Switch to non-root user
USER appuser

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command runs the bot
CMD ["/app/bin/bot"]
