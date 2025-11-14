#!/bin/bash
# Compare two versions of the IBM Cloud Logs API definition
# Usage: ./scripts/compare-api-changes.sh [old-api.json] [new-api.json]

set -e

OLD_API="${1:-logs-service-api.json.backup}"
NEW_API="${2:-logs-service-api.json}"

if [ ! -f "$OLD_API" ]; then
    echo "Error: Old API file not found: $OLD_API"
    echo "Usage: $0 [old-api.json] [new-api.json]"
    exit 1
fi

if [ ! -f "$NEW_API" ]; then
    echo "Error: New API file not found: $NEW_API"
    echo "Usage: $0 [old-api.json] [new-api.json]"
    exit 1
fi

echo "=== Comparing API Definitions ==="
echo "Old: $OLD_API"
echo "New: $NEW_API"
echo ""

# Extract operation IDs
echo "Extracting operation IDs..."
grep -o '"operationId": "[^"]*"' "$OLD_API" | sed 's/"operationId": "\([^"]*\)"/\1/' | sort > /tmp/old-operations.txt
grep -o '"operationId": "[^"]*"' "$NEW_API" | sed 's/"operationId": "\([^"]*\)"/\1/' | sort > /tmp/new-operations.txt

# Compare operations
echo "=== Operation Changes ==="
echo ""

NEW_OPS=$(comm -13 /tmp/old-operations.txt /tmp/new-operations.txt | wc -l | tr -d ' ')
REMOVED_OPS=$(comm -23 /tmp/old-operations.txt /tmp/new-operations.txt | wc -l | tr -d ' ')
COMMON_OPS=$(comm -12 /tmp/old-operations.txt /tmp/new-operations.txt | wc -l | tr -d ' ')

echo "Summary:"
echo "  Total operations in old API: $(wc -l < /tmp/old-operations.txt | tr -d ' ')"
echo "  Total operations in new API: $(wc -l < /tmp/new-operations.txt | tr -d ' ')"
echo "  Common operations: $COMMON_OPS"
echo "  New operations: $NEW_OPS"
echo "  Removed operations: $REMOVED_OPS"
echo ""

if [ "$NEW_OPS" -gt 0 ]; then
    echo "New operations (need to implement):"
    comm -13 /tmp/old-operations.txt /tmp/new-operations.txt | sed 's/^/  - /'
    echo ""
fi

if [ "$REMOVED_OPS" -gt 0 ]; then
    echo "Removed operations (need to deprecate/remove):"
    comm -23 /tmp/old-operations.txt /tmp/new-operations.txt | sed 's/^/  - /'
    echo ""
fi

# Extract and compare paths
echo "=== Endpoint Path Changes ==="
echo ""
grep -o '"/v1/[^"]*"' "$OLD_API" | sort -u > /tmp/old-paths.txt
grep -o '"/v1/[^"]*"' "$NEW_API" | sort -u > /tmp/new-paths.txt

NEW_PATHS=$(comm -13 /tmp/old-paths.txt /tmp/new-paths.txt | wc -l | tr -d ' ')
REMOVED_PATHS=$(comm -23 /tmp/old-paths.txt /tmp/new-paths.txt | wc -l | tr -d ' ')

if [ "$NEW_PATHS" -gt 0 ]; then
    echo "New paths:"
    comm -13 /tmp/old-paths.txt /tmp/new-paths.txt | sed 's/^/  /'
    echo ""
fi

if [ "$REMOVED_PATHS" -gt 0 ]; then
    echo "Removed paths:"
    comm -23 /tmp/old-paths.txt /tmp/new-paths.txt | sed 's/^/  /'
    echo ""
fi

# Check API version
echo "=== API Version ==="
OLD_VERSION=$(grep -o '"version": "[^"]*"' "$OLD_API" | head -1 | sed 's/"version": "\([^"]*\)"/\1/')
NEW_VERSION=$(grep -o '"version": "[^"]*"' "$NEW_API" | head -1 | sed 's/"version": "\([^"]*\)"/\1/')
echo "Old version: $OLD_VERSION"
echo "New version: $NEW_VERSION"
echo ""

# Check last updated date
OLD_DATE=$(grep -o '"x-last-updated": "[^"]*"' "$OLD_API" | sed 's/"x-last-updated": "\([^"]*\)"/\1/' || echo "N/A")
NEW_DATE=$(grep -o '"x-last-updated": "[^"]*"' "$NEW_API" | sed 's/"x-last-updated": "\([^"]*\)"/\1/' || echo "N/A")
echo "Old last updated: $OLD_DATE"
echo "New last updated: $NEW_DATE"
echo ""

# Generate detailed diff file
echo "=== Generating Detailed Diff ==="
DIFF_FILE="api-changes-$(date +%Y%m%d-%H%M%S).diff"
if diff -u "$OLD_API" "$NEW_API" > "$DIFF_FILE" 2>&1; then
    echo "No changes detected in API definition."
    rm "$DIFF_FILE"
else
    echo "Detailed diff saved to: $DIFF_FILE"
    echo ""
    echo "Next steps:"
    echo "  1. Review the diff file for detailed changes"
    echo "  2. Update tool implementations in internal/tools/"
    echo "  3. Register new tools in internal/server/server.go"
    echo "  4. Add tests for new/modified tools"
    echo "  5. Update README.md documentation"
    echo "  6. Run 'make test' to verify changes"
    echo "  7. Run 'make build' to ensure everything compiles"
fi

# Cleanup
rm -f /tmp/old-operations.txt /tmp/new-operations.txt /tmp/old-paths.txt /tmp/new-paths.txt

echo ""
echo "For more details, see UPDATE_API.md"
