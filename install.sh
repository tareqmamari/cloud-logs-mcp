#!/bin/bash
# IBM Cloud Logs MCP Server Installation Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/tareqmamari/cloud-logs-mcp/main/install.sh | bash
#   OR
#   ./install.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="tareqmamari/cloud-logs-mcp"
# GoReleaser project name — the prefix of every published archive
# (see .goreleaser.yaml archives.name_template `{{ .ProjectName }}`).
PROJECT_NAME="cloud-logs-mcp"
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

# Build the release archive name to match goreleaser's name_template:
#   {{ .ProjectName }}_{{ .Version }}_{{ title .Os }}_{{ arch }}.{{ ext }}
# (see .goreleaser.yaml). Os is title-cased; arch maps amd64->x86_64,
# 386->i386, arm64->arm64. No format_overrides are configured, so every
# platform (including Windows) ships as tar.gz.
case "$PLATFORM" in
    linux)   OS_TITLE="Linux" ;;
    darwin)  OS_TITLE="Darwin" ;;
    windows) OS_TITLE="Windows" ;;
esac

case "$ARCH" in
    amd64) ARCH_LABEL="x86_64" ;;
    386)   ARCH_LABEL="i386" ;;
    arm64) ARCH_LABEL="arm64" ;;
    *)     ARCH_LABEL="$ARCH" ;;
esac

ARCHIVE_EXT="tar.gz"

# The binary's name inside the archive (goreleaser builds.binary).
BINARY_IN_ARCHIVE="${BINARY_NAME}"
if [ "$OS" = "windows" ]; then
    BINARY_IN_ARCHIVE="${BINARY_NAME}.exe"
fi

echo "Detected platform: ${PLATFORM}-${ARCH}"
echo ""

# Get latest release version
echo "Fetching latest release..."
LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/') || true

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Error: Could not fetch latest version${NC}"
    exit 1
fi

echo "Latest version: v${LATEST_VERSION}"
echo ""

# Download release archive
ARCHIVE_NAME="${PROJECT_NAME}_${LATEST_VERSION}_${OS_TITLE}_${ARCH_LABEL}.${ARCHIVE_EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${LATEST_VERSION}/${ARCHIVE_NAME}"
TMP_DIR=$(mktemp -d)
ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"

echo "Downloading from: ${DOWNLOAD_URL}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
    echo -e "${RED}Error: Download failed${NC}"
    echo "URL: ${DOWNLOAD_URL}"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Download checksums.txt from the same release and verify the archive's integrity
CHECKSUMS_FILE="${TMP_DIR}/checksums.txt"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${LATEST_VERSION}/checksums.txt"

echo ""
echo "Downloading checksums..."
if ! curl -fsSL "$CHECKSUMS_URL" -o "$CHECKSUMS_FILE"; then
    echo -e "${RED}Error: Failed to download checksums.txt${NC}"
    echo "URL: ${CHECKSUMS_URL}"
    rm -rf "$TMP_DIR"
    exit 1
fi

echo "Verifying checksum..."
# checksums.txt lists archive filenames; keep only the entry for the archive
# we downloaded so the check tools don't fail on the other (absent) files.
#
# This is a literal (non-regex) match on purpose: ARCHIVE_NAME contains "."
# (e.g. between the version and the ".tar.gz"/".zip" extension), and "."
# is a wildcard in grep -E, so a plain `grep -E "  ${ARCHIVE_NAME}\$"` could
# match a checksums.txt line for a DIFFERENT archive that merely has some
# other character in that position. A bash `case` glob pattern treats "."
# as an ordinary character (only *, ?, [...] are special), so it matches
# ARCHIVE_NAME exactly while still anchoring on the trailing "  <name>".
CHECKSUM_LINE=""
while IFS= read -r line; do
    case "$line" in
        *"  ${ARCHIVE_NAME}")
            CHECKSUM_LINE="$line"
            break
            ;;
    esac
done <"$CHECKSUMS_FILE"
if [ -z "$CHECKSUM_LINE" ]; then
    echo -e "${RED}Error: No checksum entry for ${ARCHIVE_NAME} found in checksums.txt${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

# The archive was saved under its real name in TMP_DIR, so the checksum line's
# filename already matches; verify from within TMP_DIR so the tools find it.
VERIFY_FILE="${TMP_DIR}/checksums.verify.txt"
printf '%s\n' "$CHECKSUM_LINE" > "$VERIFY_FILE"

CHECKSUM_OK=0
if command -v sha256sum > /dev/null 2>&1; then
    if (cd "$TMP_DIR" && sha256sum -c "$(basename "$VERIFY_FILE")" --status); then
        CHECKSUM_OK=1
    fi
elif command -v shasum > /dev/null 2>&1; then
    if (cd "$TMP_DIR" && shasum -a 256 -c "$(basename "$VERIFY_FILE")" --status); then
        CHECKSUM_OK=1
    fi
else
    echo -e "${RED}Error: Neither sha256sum nor shasum found; cannot verify checksum${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

if [ "$CHECKSUM_OK" -ne 1 ]; then
    echo -e "${RED}Error: Checksum verification failed for ${ARCHIVE_NAME}${NC}"
    echo "The downloaded archive does not match the published checksum. Aborting."
    rm -rf "$TMP_DIR"
    exit 1
fi

echo -e "${GREEN}✓ Checksum verified${NC}"

# Optional: verify checksums.txt's cosign bundle (keyless OIDC signing) if
# cosign is installed. This confirms checksums.txt itself was published by
# the repo's release workflow, not just that the binary matches checksums.txt.
if command -v cosign > /dev/null 2>&1; then
    BUNDLE_FILE="${TMP_DIR}/checksums.txt.bundle"
    BUNDLE_URL="https://github.com/${REPO}/releases/download/v${LATEST_VERSION}/checksums.txt.bundle"

    echo ""
    echo "Verifying checksums.txt signature with cosign..."
    if ! curl -fsSL "$BUNDLE_URL" -o "$BUNDLE_FILE"; then
        echo -e "${RED}Error: Failed to download checksums.txt.bundle${NC}"
        echo "URL: ${BUNDLE_URL}"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    if ! cosign verify-blob \
        --bundle "$BUNDLE_FILE" \
        --certificate-identity-regexp "^https://github\\.com/${REPO}/\\.github/workflows/release\\.yaml@.+" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "$CHECKSUMS_FILE"; then
        echo -e "${RED}Error: cosign verification of checksums.txt failed${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    echo -e "${GREEN}✓ Cosign signature verified${NC}"
else
    echo -e "${YELLOW}Notice: cosign not found; skipping signature verification of checksums.txt${NC}"
    echo "  Install cosign (https://docs.sigstore.dev/cosign/installation/) for full supply-chain verification."
fi

# Extract the binary from the verified archive
echo ""
echo "Extracting binary..."
TMP_FILE="${TMP_DIR}/${BINARY_IN_ARCHIVE}"
case "$ARCHIVE_EXT" in
    tar.gz|tgz)
        if ! tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR" "$BINARY_IN_ARCHIVE"; then
            echo -e "${RED}Error: Failed to extract ${BINARY_IN_ARCHIVE} from ${ARCHIVE_NAME}${NC}"
            rm -rf "$TMP_DIR"
            exit 1
        fi
        ;;
    zip)
        if ! command -v unzip > /dev/null 2>&1; then
            echo -e "${RED}Error: unzip is required to extract ${ARCHIVE_NAME} but was not found${NC}"
            rm -rf "$TMP_DIR"
            exit 1
        fi
        if ! unzip -o -q "$ARCHIVE_PATH" "$BINARY_IN_ARCHIVE" -d "$TMP_DIR"; then
            echo -e "${RED}Error: Failed to extract ${BINARY_IN_ARCHIVE} from ${ARCHIVE_NAME}${NC}"
            rm -rf "$TMP_DIR"
            exit 1
        fi
        ;;
    *)
        echo -e "${RED}Error: Unsupported archive format: ${ARCHIVE_EXT}${NC}"
        rm -rf "$TMP_DIR"
        exit 1
        ;;
esac

if [ ! -f "$TMP_FILE" ]; then
    echo -e "${RED}Error: ${BINARY_IN_ARCHIVE} not found after extracting ${ARCHIVE_NAME}${NC}"
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
echo "     export LOGS_SERVICE_URL='https://[instance-id].api.[region].logs.cloud.ibm.com'"
echo "     export LOGS_REGION='us-south'"
echo "  3. Configure in Claude Desktop (see README for details)"
echo ""
echo "Documentation: https://github.com/${REPO}#readme"
echo ""
