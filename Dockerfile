# Multi-stage Dockerfile for production-ready deployment

# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=0.1.0" \
    -trimpath \
    -o logs-mcp-server \
    .

# Stage 2: Runtime
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/logs-mcp-server /logs-mcp-server

# Set environment
ENV TZ=UTC
ENV ENVIRONMENT=production
ENV LOG_FORMAT=json
ENV LOG_LEVEL=info

# Run as non-root user (nobody)
USER 65534:65534

# Set entrypoint
ENTRYPOINT ["/logs-mcp-server"]

# Metadata
LABEL maintainer="IBM Cloud Logs Team" \
      description="MCP Server for IBM Cloud Logs" \
      version="0.1.0"
