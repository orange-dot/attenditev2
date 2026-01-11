# Multi-stage build for Go application
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Ensure go.mod is tidy
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/platform \
    ./cmd/platform

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 platform && \
    adduser -u 1000 -G platform -s /bin/sh -D platform

# Set timezone
ENV TZ=Europe/Belgrade

# Copy binary from builder
COPY --from=builder /app/platform /usr/local/bin/platform

# Copy migrations if needed
COPY --from=builder /app/internal/shared/database/migrations /app/migrations

# Set ownership
RUN chown -R platform:platform /app

# Switch to non-root user
USER platform

# Working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["platform"]
