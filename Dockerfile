# Multi-stage Dockerfile for IBM Cloud Logs MCP Server
# Uses Red Hat Universal Base Image (UBI) for enterprise-grade security
# Best practices: multi-stage build, non-root user, minimal runtime image

# ==============================================================================
# Build Arguments - centralized for easy version management and automation
# Renovate/Dependabot can automatically update these values
# ==============================================================================
# renovate: datasource=docker depName=registry.access.redhat.com/ubi9/go-toolset
ARG GO_TOOLSET_VERSION=1.25.3-1765311584
# renovate: datasource=docker depName=registry.access.redhat.com/ubi9/ubi-micro
ARG UBI_MICRO_VERSION=9.7-1762965531

# ==============================================================================
# Stage 1: Build environment using Red Hat UBI Go Toolset
# ==============================================================================
FROM registry.access.redhat.com/ubi9/go-toolset:${GO_TOOLSET_VERSION} AS builder

# Switch to root for build operations (go-toolset runs as user 1001 by default)
USER root

# Set working directory
WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with security and optimization flags
# -trimpath: Remove file system paths from the binary
# -ldflags: Strip debug info (-s -w) and set version
# CGO_ENABLED=0: Static binary, no C dependencies
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o logs-mcp-server \
    .

# Verify the binary is statically linked
RUN file logs-mcp-server | grep -q "statically linked" || \
    (echo "Binary is not statically linked" && exit 1)

# ==============================================================================
# Stage 2: Minimal runtime using UBI Micro
# UBI Micro is the smallest UBI image (~26MB), ideal for static Go binaries
# ==============================================================================
FROM registry.access.redhat.com/ubi9/ubi-micro:${UBI_MICRO_VERSION}

# Copy CA certificates for HTTPS connections (UBI uses pki path)
COPY --from=builder /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
COPY --from=builder /etc/pki/tls/certs/ca-bundle.crt /etc/pki/tls/certs/ca-bundle.crt

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/logs-mcp-server /logs-mcp-server

# Environment configuration
ENV TZ=UTC \
    ENVIRONMENT=production \
    LOG_FORMAT=json \
    LOG_LEVEL=info

# Use non-root user for security (CIS Docker Benchmark 4.1)
# UBI images use UID 1001 as the default non-root user
USER 1001:0

# Expose no ports - MCP uses stdio transport
# EXPOSE is documentation only, not functional

# Set entrypoint
ENTRYPOINT ["/logs-mcp-server"]

# ==============================================================================
# OCI Image Labels (following OCI Image Spec)
# https://github.com/opencontainers/image-spec/blob/main/annotations.md
# ==============================================================================
LABEL org.opencontainers.image.title="IBM Cloud Logs MCP Server" \
      org.opencontainers.image.description="Model Context Protocol server for IBM Cloud Logs service" \
      org.opencontainers.image.vendor="IBM" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/tareqmamari/cloud-logs-mcp" \
      org.opencontainers.image.base.name="registry.access.redhat.com/ubi9/ubi-micro"
