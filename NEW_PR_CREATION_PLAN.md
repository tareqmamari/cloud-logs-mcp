# Plan: Replace PR #114 with Clean PR

## Objective
Close PR #114 and create a new PR from the same branch (`fix/code-review-findings`) with a clean description (no Claude attribution).

## Prerequisites ✅
- [x] Clean PR description prepared: `PR_114_CLEAN_DESCRIPTION.md`
- [x] Verified code has no Claude references (3,847 lines checked)
- [x] GitHub CLI (`gh`) is available
- [x] Branch `fix/code-review-findings` exists and is ready

## Steps to Execute

### Step 1: Close PR #114
```bash
gh pr close 114 \
  --repo tareqmamari/cloud-logs-mcp \
  --comment "Closing to recreate with clean description (no tool attribution)"
```

**Expected Result:** PR #114 status changes to "Closed"

### Step 2: Create New PR
```bash
gh pr create \
  --repo tareqmamari/cloud-logs-mcp \
  --base main \
  --head fix/code-review-findings \
  --title "Fix the Issues Turned Up by the Code Review" \
  --body-file PR_114_CLEAN_DESCRIPTION.md
```

**Expected Result:** New PR created (likely #115) with clean description

### Step 3: Verify New PR
```bash
# Get the new PR number (will be in the output of step 2)
NEW_PR_NUMBER=$(gh pr list --repo tareqmamari/cloud-logs-mcp --head fix/code-review-findings --json number -q '.[0].number')

# Verify description has no Claude reference
gh pr view $NEW_PR_NUMBER --repo tareqmamari/cloud-logs-mcp --json body -q '.body' | grep -i claude
```

**Expected Result:** Empty output (no Claude references found)

## What Gets Preserved
- ✅ All 79 commits (same branch)
- ✅ All code changes
- ✅ Git history
- ✅ Branch name

## What Gets Lost
- ❌ PR number (#114 → #115 or higher)
- ❌ Any review comments on PR #114
- ❌ PR #114 discussion thread

## Rollback Plan
If something goes wrong:
```bash
# Reopen PR #114
gh pr reopen 114 --repo tareqmamari/cloud-logs-mcp

# Close the new PR if needed
gh pr close <NEW_PR_NUMBER> --repo tareqmamari/cloud-logs-mcp
```

## Ready to Execute?
All prerequisites are met. The commands above can be run in sequence to complete the PR replacement.

**Estimated time:** 2-3 minutes
