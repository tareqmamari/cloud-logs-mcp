#!/usr/bin/env bash
# Install IBM Cloud Logs agent skills to user-level directory and optionally
# inject routing instructions into your AI assistant's project config.
#
# Usage:
#   ./scripts/install-skills.sh                      # skills only
#   ./scripts/install-skills.sh --ai claude           # skills + CLAUDE.md
#   ./scripts/install-skills.sh --ai cursor           # skills + .cursor/rules/
#   ./scripts/install-skills.sh --ai all              # skills + all supported AI tools
#   ./scripts/install-skills.sh --ai claude,copilot   # skills + multiple tools
#
# Supported AI tools:
#   claude    → CLAUDE.md
#   cursor    → .cursor/rules/ibm-cloud-logs.md
#   copilot   → .github/copilot-instructions.md
#   windsurf  → .windsurfrules
#   gemini    → GEMINI.md
#   cline     → .clinerules
#   amazonq   → .amazonq/rules/ibm-cloud-logs.md
#   bob       → BOB.md
#   aider     → CONVENTIONS.md

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SKILLS_SRC="$PROJECT_DIR/.agents/skills"
SKILLS_DEST="${HOME}/.agents/skills"
TEMPLATE="$SKILLS_SRC/AI_INSTRUCTIONS.md"

# Section markers for safe append/update
MARKER_START="<!-- IBM_CLOUD_LOGS_SKILLS_START -->"
MARKER_END="<!-- IBM_CLOUD_LOGS_SKILLS_END -->"

AI_TARGETS=""

usage() {
  echo "Usage: $0 [--ai <tool>[,<tool>...]]"
  echo ""
  echo "Install IBM Cloud Logs agent skills and optionally configure AI tools."
  echo ""
  echo "Options:"
  echo "  --ai <tool>  Configure AI assistant project instructions."
  echo "               Supported: claude, cursor, copilot, windsurf, gemini,"
  echo "               cline, amazonq, bob, aider, all"
  echo "               Comma-separate for multiple: --ai claude,copilot"
  echo ""
  echo "Examples:"
  echo "  $0                        # Install skills only"
  echo "  $0 --ai claude            # Install skills + CLAUDE.md"
  echo "  $0 --ai all               # Install skills + all AI tool configs"
  exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --ai)
      shift
      AI_TARGETS="${1:-}"
      if [[ -z "$AI_TARGETS" ]]; then
        echo "Error: --ai requires a tool name" >&2
        exit 1
      fi
      shift
      ;;
    --help|-h)
      usage
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

# ── Install skills ──────────────────────────────────────────────

echo "Installing skills to $SKILLS_DEST ..."
mkdir -p "$SKILLS_DEST"

count=0
for skill in "$SKILLS_SRC"/ibm-cloud-logs*; do
  [ -d "$skill" ] || continue
  name="$(basename "$skill")"
  cp -r "$skill" "$SKILLS_DEST/"
  echo "  ✓ $name"
  count=$((count + 1))
done

echo "Installed $count skills."
echo ""

# ── AI tool configuration ──────────────────────────────────────

if [[ -z "$AI_TARGETS" ]]; then
  exit 0
fi

# Resolve file path for each AI tool
resolve_ai_path() {
  local tool="$1"
  case "$tool" in
    claude)   echo "CLAUDE.md" ;;
    cursor)   echo ".cursor/rules/ibm-cloud-logs.md" ;;
    copilot)  echo ".github/copilot-instructions.md" ;;
    windsurf) echo ".windsurfrules" ;;
    gemini)   echo "GEMINI.md" ;;
    cline)    echo ".clinerules" ;;
    amazonq)  echo ".amazonq/rules/ibm-cloud-logs.md" ;;
    bob)      echo "BOB.md" ;;
    aider)    echo "CONVENTIONS.md" ;;
    *)
      echo "Error: unknown AI tool '$tool'" >&2
      echo "Supported: claude, cursor, copilot, windsurf, gemini, cline, amazonq, bob, aider" >&2
      return 1
      ;;
  esac
}

# Inject routing content into a file (append or update)
inject_routing() {
  local target="$1"
  local tool_name="$2"
  local dir
  dir="$(dirname "$target")"

  # Ensure parent directory exists
  if [[ "$dir" != "." ]]; then
    mkdir -p "$dir"
  fi

  # Read template content
  local content
  content="$(cat "$TEMPLATE")"

  if [[ -f "$target" ]]; then
    # File exists — check if we already injected
    if grep -qF "$MARKER_START" "$target" 2>/dev/null; then
      # Update existing section: strip old content between markers, insert new
      local tmp
      tmp="$(mktemp)"
      local skip=0
      while IFS= read -r line; do
        if [[ "$line" == *"$MARKER_START"* ]]; then
          skip=1
          echo "$line" >> "$tmp"
          cat "$TEMPLATE" >> "$tmp"
          continue
        fi
        if [[ "$line" == *"$MARKER_END"* ]]; then
          skip=0
        fi
        if [[ $skip -eq 0 ]]; then
          echo "$line" >> "$tmp"
        fi
      done < "$target"
      mv "$tmp" "$target"
      echo "  ↻ Updated existing section in $target"
    else
      # Append with markers
      {
        echo ""
        echo "$MARKER_START"
        cat "$TEMPLATE"
        echo "$MARKER_END"
      } >> "$target"
      echo "  + Appended to existing $target"
    fi
  else
    # Create new file with markers
    {
      echo "$MARKER_START"
      cat "$TEMPLATE"
      echo "$MARKER_END"
    } > "$target"
    echo "  ✓ Created $target"
  fi
}

# Expand "all" to full list
ALL_TOOLS="claude cursor copilot windsurf gemini cline amazonq bob aider"

if [[ "$AI_TARGETS" == "all" ]]; then
  AI_TARGETS="$ALL_TOOLS"
else
  # Support comma-separated: --ai claude,copilot
  AI_TARGETS="${AI_TARGETS//,/ }"
fi

echo "Configuring AI tool instructions ..."
for tool in $AI_TARGETS; do
  target="$(resolve_ai_path "$tool")" || continue
  inject_routing "$target" "$tool"
done

echo ""
echo "Done. Routing instructions help your AI assistant choose between"
echo "Skills (cheaper for 7/9 tasks) and MCP (better for investigation)."
