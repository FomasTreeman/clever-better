# =============================================================================
# Clever Better - Multi-stage Go Application Dockerfile
# Builds all Go binaries for the trading bot and related services
# =============================================================================

# Build arguments for versioning
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Re-declare ARGs after FROM
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy entire source code
COPY . .

# Define ldflags for version embedding
ENV LDFLAGS="-w -s -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}"

# Build all binaries with static linking, optimized flags, and version info
# CGO_ENABLED=0 for static binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/bot ./cmd/bot

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/data-ingestion ./cmd/data-ingestion

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/backtest ./cmd/backtest

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/ml-feedback ./cmd/ml-feedback

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/strategy-discovery ./cmd/strategy-discovery

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o /build/bin/ml-status ./cmd/ml-status

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM alpine:latest

# Re-declare ARGs for labels
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE

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

# Add labels for image metadata
LABEL version="${VERSION}" \
      git.commit="${GIT_COMMIT}" \
      build.date="${BUILD_DATE}" \
      maintainer="Clever Better Team" \
      description="Clever Better Trading Bot"

# Expose health check port
EXPOSE 8080

# Switch to non-root user
USER appuser

# Environment variable for health check port (can be overridden)
ENV HEALTH_PORT=8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${HEALTH_PORT}/health || exit 1

# Default command runs the bot
CMD ["/app/bin/bot"]
