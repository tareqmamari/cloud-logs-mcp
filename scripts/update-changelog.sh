#!/bin/bash
# Update CHANGELOG.md with the current release notes
# This script is called by GoReleaser before each release

set -e

VERSION="${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")}"
DATE=$(date +%Y-%m-%d)

# Get the previous tag for comparison
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")

# Generate changelog for this version using git log
if [ -n "$PREV_TAG" ]; then
    CHANGES=$(git log --pretty=format:"- %s" "${PREV_TAG}..HEAD" | grep -E "^- (feat|fix|docs|perf|refactor):" || echo "")
else
    CHANGES=$(git log --pretty=format:"- %s" HEAD | grep -E "^- (feat|fix|docs|perf|refactor):" || echo "")
fi

# Categorize changes
FEATURES=$(echo "$CHANGES" | grep "^- feat:" | sed 's/^- feat: /- /' || echo "")
FIXES=$(echo "$CHANGES" | grep "^- fix:" | sed 's/^- fix: /- /' || echo "")
DOCS=$(echo "$CHANGES" | grep "^- docs:" | sed 's/^- docs: /- /' || echo "")
PERF=$(echo "$CHANGES" | grep "^- perf:" | sed 's/^- perf: /- /' || echo "")

# Create the new changelog entry
NEW_ENTRY="## [${VERSION}] - ${DATE}"

if [ -n "$FEATURES" ]; then
    NEW_ENTRY="${NEW_ENTRY}
### Features
${FEATURES}"
fi

if [ -n "$FIXES" ]; then
    NEW_ENTRY="${NEW_ENTRY}
### Bug Fixes
${FIXES}"
fi

if [ -n "$DOCS" ]; then
    NEW_ENTRY="${NEW_ENTRY}
### Documentation
${DOCS}"
fi

if [ -n "$PERF" ]; then
    NEW_ENTRY="${NEW_ENTRY}
### Performance
${PERF}"
fi

# Insert the new entry into CHANGELOG.md after the header
if [ -f "CHANGELOG.md" ]; then
    # Create a temporary file
    TMP_FILE=$(mktemp)

    # Write header
    head -n 7 CHANGELOG.md > "$TMP_FILE"

    # Add blank line
    echo "" >> "$TMP_FILE"

    # Add new entry
    echo "$NEW_ENTRY" >> "$TMP_FILE"
    echo "" >> "$TMP_FILE"

    # Add the rest of the file (skip header)
    tail -n +8 CHANGELOG.md >> "$TMP_FILE"

    # Replace original file
    mv "$TMP_FILE" CHANGELOG.md

    # Add compare link at the bottom if it doesn't exist
    if ! grep -q "\[${VERSION}\]:" CHANGELOG.md; then
        if [ -n "$PREV_TAG" ]; then
            echo "[${VERSION}]: https://github.com/tareqmamari/logs-mcp-server/compare/${PREV_TAG}...${VERSION}" >> CHANGELOG.md
        fi
    fi

    echo "✓ Updated CHANGELOG.md with ${VERSION}"
else
    echo "✗ CHANGELOG.md not found"
    exit 1
fi
