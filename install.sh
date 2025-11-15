#!/bin/bash
# IBM Cloud Logs MCP Server Installation Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/observability-c/logs-mcp-server/main/install.sh | bash
#   OR
#   ./install.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="observability-c/logs-mcp-server"
BINARY_NAME="logs-mcp-server"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo -e "${GREEN}IBM Cloud Logs MCP Server Installer${NC}"
echo "======================================"
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        PLATFORM="linux"
        ;;
    darwin)
        PLATFORM="darwin"
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

BINARY_FILE="${BINARY_NAME}-${PLATFORM}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_FILE="${BINARY_FILE}.exe"
fi

echo "Detected platform: ${PLATFORM}-${ARCH}"
echo ""

# Get latest release version
echo "Fetching latest release..."
LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Error: Could not fetch latest version${NC}"
    exit 1
fi

echo "Latest version: v${LATEST_VERSION}"
echo ""

# Download binary
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${LATEST_VERSION}/${BINARY_FILE}"
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

echo "Downloading from: ${DOWNLOAD_URL}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
    echo -e "${RED}Error: Download failed${NC}"
    echo "URL: ${DOWNLOAD_URL}"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Make binary executable
chmod +x "$TMP_FILE"

# Verify binary works
echo ""
echo "Verifying binary..."
if ! "$TMP_FILE" --version > /dev/null 2>&1; then
    # Binary might not have --version flag yet, just check it runs
    echo -e "${YELLOW}Warning: Could not verify binary version${NC}"
fi

# Install binary
echo ""
echo "Installing to ${INSTALL_DIR}..."

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Installing to ${INSTALL_DIR} requires sudo"
    sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

# Cleanup
rm -rf "$TMP_DIR"

echo ""
echo -e "${GREEN}✓ Installation successful!${NC}"
echo ""
echo "Binary installed to: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
echo "Next steps:"
echo "  1. Get your IBM Cloud API key: https://cloud.ibm.com/iam/apikeys"
echo "  2. Set environment variables:"
echo "     export LOGS_API_KEY='your-api-key'"
echo "     export LOGS_SERVICE_URL='https://instance-id.api.region.logs.cloud.ibm.com'"
echo "     export LOGS_REGION='us-south'"
echo "  3. Configure in Claude Desktop (see README for details)"
echo ""
echo "Documentation: https://github.com/${REPO}#readme"
echo ""
