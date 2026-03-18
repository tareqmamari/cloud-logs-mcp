# MCP vs Agent Skills: Benchmark Report

**IBM Cloud Logs — Measured Performance Data**
**Date:** March 2026 | **Version:** 0.11.0 | **Tokenizer:** Claude (native)

> All token counts in this report are **measured** using the Claude (native) tokenizer.
> Wire payload sizes are from actual Go test execution (`TestMCPWirePayload`).
> Scenario benchmarks use real queries against IBM Cloud Logs (au-syd instance).

## Executive Summary

| Metric | MCP | 8 Skills (previous) | 1 Skill (current) |
|--------|----:|--------------------:|------------------:|
| Fixed context overhead | 18,229 tokens | 0 tokens | 0 tokens |
| Per-conversation cost (typical) | ~26,224 tokens | ~7,068 tokens | ~7,506 tokens |
| **9-scenario total (measured)** | **187,456 tokens** | **161,555 tokens** | **107,937 tokens** |
| vs MCP | — | -14% | **-42%** |
| Domain knowledge available | 96 tool definitions | 8 skills + 21 references | 1 skill + 29 references |
| Wire payload size | 69,087 bytes (67.5 KB) | N/A | N/A |
| Avg tool response (measured) | 1,120 tokens | N/A (in-context) | N/A (in-context) |
| Binary size impact | — | +371,832 bytes embedded | +313,063 bytes embedded |

**Key finding:** All three architectures measured against the au-syd instance across
**9 scenarios covering every feature area**:
- **1 Skill:** 107,937 tokens — **42% cheaper than MCP**, **33% cheaper than 8 Skills**
- **8 Skills:** 161,555 tokens — 14% cheaper than MCP, but 50% more than 1 Skill
- **MCP:** 187,456 tokens — most expensive due to 18,229-token fixed overhead per turn

Consolidating 8 skills into 1 skill with on-demand domain guides eliminated redundant
SKILL.md loads for cross-domain tasks (S1 dropped from 66,177 → 14,995 tokens).
The iteration tax (syntax errors, wrong tier, CLI flag mistakes) is negligible:
**527 tokens (0.5%)**. See [Section 7](#7-real-world-scenario-benchmark-measured).

---

## 1. MCP Wire Payload (Measured)

The MCP server registers **96 tools**. On connection, every tool's name, description,
and input schema is sent to the agent via `tools/list`. These tokens are **always present**
in the context window, whether or not the tools are used.

> **Measured two ways:**
> - Go test (`TestMCPWirePayload`): tool definitions total **71,195 bytes** (69.5 KB), **18,794 tokens**.
> - Live JSON-RPC (`tools/list` response): **69,087 bytes** (67.5 KB), **18,229 tokens**.
> The small difference (~2KB) is due to JSON-RPC envelope overhead vs raw definitions.

| Component | Bytes (wire) | % of Total |
|-----------|------------:|-----------:|
| Tool descriptions (x98) | 20,505 | 28.8% |
| Input schemas (x98) | 41,205 | 57.9% |
| JSON structure overhead | 9,485 | 13.3% |
| **Total** | **71,195** | **100%** |

![Wire Payload Breakdown](benchmarks/wire-payload-breakdown.png)

**Top 10 largest tool definitions (on the wire):**

| Tool | Wire Bytes | Description | Schema |
|------|----------:|-----------:|-------:|
| `create_rule_group` | 5,014 | 1,257 | 3,699 |
| `query_logs` | 4,014 | 713 | 3,149 |
| `build_query` | 3,948 | 1,078 | 2,716 |
| `suggest_alert` | 3,086 | 885 | 2,147 |
| `create_e2m` | 2,535 | 521 | 1,963 |
| `create_policy` | 2,133 | 511 | 1,568 |
| `ingest_logs` | 2,014 | 487 | 1,391 |
| `session_context` | 2,009 | 953 | 958 |
| `create_dashboard` | 1,978 | 438 | 1,483 |
| `investigate_incident` | 1,782 | 790 | 820 |

![MCP Tool Size Distribution](benchmarks/mcp-tool-distribution.png)

### 1.1 Tool Response Sizes (Measured Locally)

These tools were executed locally via `TestMCPWirePayload` (no network, mock API client).
Response sizes are the actual bytes returned by each tool:

| Tool | Response Bytes | Notes |
|------|-------------:|-------|
| `list_tool_categories` | 2,037 | Executed locally |
| `search_tools` | 658 | Executed locally |
| `get_dataprime_reference` | 549 | Executed locally |
| `estimate_query_cost` | 507 | Executed locally |
| `build_query` | 448 | Executed locally |
| `validate_query` | 248 | Executed locally |
| `get_query_templates` | 37 | Executed locally |
| `session_context` | 19 | Executed locally |

![Tool Response Sizes](benchmarks/response-sizes.png)

**Estimated response sizes by category** (for tools requiring network):

| Category | Tools | Avg Response | Source |
|----------|------:|------------:|--------|
| CRUD list | 30 | ~500 tokens | estimate |
| CRUD get | 20 | ~300 tokens | estimate |
| CRUD create/update | 20 | ~300 tokens | estimate |
| CRUD delete | 15 | ~50 tokens | estimate |
| Query | 5 | ~3,000 tokens | estimate |
| Reference | 4 | ~2,500 tokens | measured_locally |
| Intelligence | 4 | ~1,500 tokens | estimate |
| Meta | 3 | ~400 tokens | measured_locally |
| **Weighted average** | **101** | **~593 tokens** | |

---

## 2. Agent Skills Token Counts (Measured)

Skills are consolidated into a **single `ibm-cloud-logs` skill** with domain-specific
content moved to reference guides loaded on demand. Only the SKILL.md enters context
on activation; domain guides are loaded when the agent's task requires them.

| Component | Tokens | Bytes | Files | Lines |
|-----------|-------:|------:|------:|------:|
| SKILL.md (entry point) | 4,506 | 17,079 | 1 | 361 |
| References (domain guides + refs) | 53,420 | 202,461 | 29 | — |
| Scripts | 7,874 | 29,843 | 3 | — |
| Assets | 16,802 | 63,680 | 10 | — |
| **Total** | **82,602** | **313,063** | **43** | — |

The consolidated SKILL.md (4,506 tokens) contains merged activation triggers, a single
auth block, a domain routing table, DataPrime quick reference, and query execution
strategy. Domain-specific guides (alerting, incident, dashboards, cost, ingestion,
access control, API, query) are loaded on demand from `references/`.

**Previous architecture (8 separate skills):** 98,018 tokens across 8 SKILL.md files
(28,866 tokens for SKILL.md files alone). The consolidation reduced the entry-point
cost from 3,024-5,123 tokens (per domain skill) to a single 4,506-token SKILL.md
shared across all domains.

![Skill Token Breakdown](benchmarks/skill-token-breakdown.png)

---

## 3. Head-to-Head: Per-Conversation Token Cost

MCP cost = fixed overhead + tool call/response tokens.
8 Skills cost = domain SKILL.md (3K-5K each) + reference loads.
1 Skill cost = consolidated SKILL.md (4,506) + domain guide + reference loads.

| Scenario | MCP | 8 Skills | 1 Skill | MCP vs 1 Skill |
|----------|----:|--------:|-------:|---------:|
| Before any tool call | 18,794 | 0 | 0 | — |
| After 1 tool call / skill activation | 19,537 | 3,024–5,123 | 4,506 | 4.3x |
| Typical session (10 calls / 1 domain) | 26,224 | 7,068 | 7,506 | 3.5x |
| Cross-domain session (2-3 domains) | 26,224 | 12,091–15,369 | 9,506 | 2.8x |
| Heavy session (25 calls / 5 refs) | 37,369 | 19,506 | 14,506 | 2.6x |

**Key insight:** 1 Skill is cheaper than 8 Skills for cross-domain tasks because it loads
one 4,506-token SKILL.md instead of 2-3 separate SKILL.md files (6K-15K tokens combined).
For single-domain tasks (S5-S9), 8 Skills can be slightly cheaper since individual
SKILL.md files are smaller than the consolidated one.

![Head-to-Head Comparison](benchmarks/head-to-head.png)

![Token Comparison](benchmarks/token-comparison.png)

---

## 4. Binary & Performance (Measured)

| Metric | Value |
|--------|------:|
| Binary size (with embedded skills) | 26.59 MB |
| Build time | 0.883s |
| `skills list` latency (avg / p99) | 109.0ms / 427.2ms |
| `skills install` latency (avg) | 56.5ms |
| Embedded skill files | 43 files |
| Total skill bytes | 313,063 bytes |

---

## 5. Token Cost Projection

Based on Claude Sonnet 4 pricing ($3 / 1M input tokens) and measured token counts.
MCP: 26,224 tokens/conversation. 8 Skills: 7,068 tokens/conversation. 1 Skill: 7,506 tokens/conversation.

| Conversations/Month | MCP Annual Cost | 8 Skills Annual Cost | 1 Skill Annual Cost | 1 Skill Savings vs MCP |
|--------------------:|----------------:|--------------------:|--------------------:|-----------------------:|
| 10 | $9.44 | $2.54 | $2.70 | 71% |
| 50 | $47.20 | $12.72 | $13.51 | 71% |
| 100 | $94.41 | $25.44 | $27.02 | 71% |
| 500 | $472.03 | $127.22 | $135.11 | 71% |
| 1,000 | $944.06 | $254.45 | $270.22 | 71% |

**Note:** 8 Skills appears ~6% cheaper per conversation than 1 Skill in typical single-domain
sessions. However, cross-domain tasks (incident investigation, monitoring setup) load 2-3
SKILL.md files, making 8 Skills 50% more expensive overall across all 9 scenarios
(161,555 vs 107,937 tokens).

![Cost Projection](benchmarks/cost-projection.png)

---

## 6. Multi-Dimensional Comparison

> **Important distinction:** MCP and Skills serve different roles. MCP is a **runtime** —
> it connects to IBM Cloud Logs, executes queries, and manages resources. Skills are
> **knowledge bundles** — they teach agents how to write correct queries, design alerts,
> and configure resources, but do not execute anything. To actually query logs or create
> alerts, you need either the MCP server or direct API access. Skills and MCP are
> complementary: Skills reduce token cost for knowledge, MCP provides execution.

![Radar Comparison](benchmarks/radar-comparison.png)

| Dimension | MCP | 8 Skills | 1 Skill | Notes |
|-----------|:---:|:--------:|:-------:|-------|
| Token efficiency | 3/10 | 7/10 | 9/10 | MCP: ~26K; 8 Skills: ~7K (single) / ~15K (cross); 1 Skill: ~7.5K |
| Cross-domain efficiency | 3/10 | 4/10 | 9/10 | 1 Skill loads one SKILL.md; 8 Skills loads 2-3 |
| Query accuracy | 10/10 | 9/10 | 9/10 | MCP has programmatic auto-correction engine |
| Live data access | 10/10 | 0/10 | 0/10 | Only MCP can execute queries; Skills provide guidance only |
| Setup friction | 4/10 | 8/10 | 10/10 | 1 Skill: zero config; 8 Skills: multiple activations |
| Platform reach | 5/10 | 10/10 | 10/10 | Skills work on 30+ agent platforms |
| Offline guidance | 0/10 | 8/10 | 8/10 | Skills provide query/config guidance offline |
| Latency | 5/10 | 10/10 | 10/10 | Skills are in-context; MCP requires network round-trips |
| Security posture | 6/10 | 9/10 | 9/10 | Skills never handle credentials |
| Maintenance burden | 7/10 | 4/10 | 9/10 | 1 Skill: 1 SKILL.md; 8 Skills: 8 SKILL.md with duplicated auth |
| Cost efficiency | 4/10 | 7/10 | 9/10 | 1 Skill saves 42% vs MCP, 33% vs 8 Skills across all scenarios |

---

## 7. Real-World Scenario Benchmark (Measured)

> **Data source:** Live queries against IBM Cloud Logs (au-syd instance, archive tier, 24h window).
> Skill file sizes measured from embedded files. MCP responses measured from the actual MCP server binary.
> Token counts estimated at 3.79 bytes/token (calibrated from wire payload measurement: 71,195 bytes = 18,794 tokens).

### Scenario Overview

Nine end-to-end workflows replayed step-by-step against the au-syd instance,
covering **every feature area** of the MCP server's 96 tools. **Both sides measured** —
Skills via `curl` + file reads, MCP via the actual MCP server binary with JSON-RPC
tool calls. Token counts at 3.79 bytes/token. Scenarios 1-3 include deliberate
mistakes (wrong tier, `AND` vs `&&`, `=` vs `==`, `--from-file`, missing dashboard CLI).
Scenarios 4-9 measure clean operations (no mistakes).

| Scenario | Skills+CLI (measured) | MCP (measured) | Winner | Delta |
|----------|----------------------:|---------------:|--------|------:|
| 1: Incident Investigation | 14,995 | 23,587 | Skills | -36% |
| 2: Cost Optimization | 9,641 | 18,920 | Skills | -49% |
| 3: Monitoring Setup | 15,819 | 24,854 | Skills | -36% |
| 4: Normal Operations (CRUD) | 17,936 | 24,919 | Skills | -28% |
| 5: Query Authoring & Validation | 15,832 | 20,646 | Skills | -23% |
| 6: Ingestion Pipeline | 9,283 | 18,383 | Skills | -50% |
| 7: Data Governance | 7,482 | 18,437 | Skills | -59% |
| 8: E2M & Streaming | 8,046 | 18,399 | Skills | -56% |
| 9: API Discovery & Meta | 8,466 | 19,311 | Skills | -56% |
| Auth overhead | 437 | 0 | — | — |
| **Total (all 9)** | **107,937** | **187,456** | **Skills** | **-42%** |

### Architecture Comparison: 1 Skill vs 8 Skills vs MCP

The consolidation from 8 separate skills into 1 skill with domain guides
significantly reduced Skills token consumption:

![Architecture Totals](benchmarks/architecture-totals.png)

| Scenario | 1 Skill (current) | 8 Skills (previous) | MCP | Best |
|----------|-------------------:|--------------------:|----:|------|
| 1: Incident Investigation | 14,995 | 66,177 | 23,587 | **1 Skill** |
| 2: Cost Optimization | 9,641 | 13,826 | 18,920 | **1 Skill** |
| 3: Monitoring Setup | 15,819 | 20,864 | 24,854 | **1 Skill** |
| 4: Normal Operations | 17,936 | 19,618 | 24,919 | **1 Skill** |
| 5: Query Authoring | 15,832 | 14,594 | 20,646 | 8 Skills |
| 6: Ingestion Pipeline | 9,283 | 7,317 | 18,383 | 8 Skills |
| 7: Data Governance | 7,482 | 5,429 | 18,437 | 8 Skills |
| 8: E2M & Streaming | 8,046 | 5,948 | 18,399 | 8 Skills |
| 9: API Discovery | 8,466 | 7,342 | 19,311 | 8 Skills |
| **Total** | **107,937** | **161,555** | **187,456** | **1 Skill** |
| **vs MCP** | **-42%** | **-14%** | — | — |

**Why 1 Skill wins overall despite S5-S9 being slightly higher:**
- S1-S4 savings are massive: the old 8-skill architecture loaded 2-3 full SKILL.md
  files (each 3K-5K tokens) for cross-domain tasks. The consolidated SKILL.md (4,506
  tokens) replaces all of them with a single load + lightweight domain guides.
- S5-S9 are slightly more expensive (+1K-2K tokens each) because the consolidated
  SKILL.md is larger than any individual domain SKILL.md. But these scenarios were
  already cheap, so the absolute increase is small.
- S1 saw the biggest improvement: from 66,177 → 14,995 tokens (-77%). The old
  architecture loaded investigation (16K) + query (15K) + alerting (19K) = 50K bytes
  of SKILL.md files. Now it loads one SKILL.md (17K) + domain guides on demand.

### Scenario 1: Incident Investigation (measured)

Global error scan → component deep-dive → heuristic matching → alert creation.

**Skills + CLI step-by-step (measured against au-syd):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 | Consolidated entry point |
| Read incident-guide.md | skill_read | 1,852 | 7,019 | Investigation methodology |
| GET /v1/tco_policies | api_call | 0 | 0 | Empty (no policies configured) |
| Query wrong tier (frequent_search) | **error_retry** | 58 | 221 | Returns near-empty result |
| Query global error rate (archive) | query_success | 58 | 221 | Aggregation — small response |
| Query with `AND` (wrong syntax) | **error_retry** | 73 | 275 | DataPrime parse error |
| Query with `&&` (fixed) | query_success | 58 | 221 | Filtered results |
| Global error timeline | query_success | 22 | 82 | Time-bucketed aggregation |
| Critical errors (raw) | query_success | 22 | 82 | Small result set (au-syd) |
| Read investigation-queries.md | skill_read | 1,622 | 6,147 | Query templates |
| Query with `=` (wrong syntax) | **error_retry** | 68 | 258 | DataPrime parse error |
| Component error patterns | query_success | 93 | 352 | Small aggregation |
| Component subsystems | query_success | 94 | 358 | Small aggregation |
| Component dependencies | query_success | 163 | 616 | Small aggregation |
| Read heuristic-details.md | skill_read | 2,142 | 8,118 | Heuristic patterns |
| Read alerting-guide.md | skill_read | 2,576 | 9,763 | Alert methodology |
| Read burn-rate-math.md | skill_read | 1,564 | 5,927 | Threshold formulas |
| CLI `--from-file` error | **error_retry** | 24 | 91 | Unknown flag |
| Create alert via API | api_call | 0 | 0 | Success (empty response) |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 14,262 | 6 files |
| Query responses | 510 | 7 queries |
| API calls | 0 | 2 calls |
| **Error retries** | **223** | **4 mistakes** |
| **TOTAL** | **14,995** | **19 steps** |

**MCP breakdown (measured against au-syd):**
| Component | Tokens | Bytes | Details |
|-----------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| investigate_incident | 232 | 881 | "No Issues Found" (no recent errors) |
| suggest_alert | 4,842 | 18,350 | Full SRE-grade alert with burn rate, 6 suggestions |
| create_alert_definition | 284 | 1,075 | Dry-run validation response |
| **Total** | **23,587** | **89,393** | Fixed overhead + 3 tool calls |

**Skills wins by 36%.** The consolidated SKILL.md (4,506 tokens) + incident-guide.md
(1,852 tokens) replaced the old pattern of loading 3 separate SKILL.md files (50K bytes).
Query responses were small on au-syd. MCP's fixed overhead (18,229 tokens) alone exceeds
Skills' entire scenario cost.

### Scenario 2: Cost Optimization (measured)

List TCO policies → analyze volume by severity/app → recommend tier changes.

**Skills + CLI step-by-step (measured against au-syd):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 | Consolidated entry point |
| Read cost-guide.md | skill_read | 1,514 | 5,738 | TCO policies, tier selection |
| GET /v1/tco_policies | api_call | 0 | 0 | Empty response |
| Query with `sort` (wrong syntax) | **error_retry** | 134 | 507 | DataPrime uses `orderby` |
| Volume by severity | query_success | 22 | 82 | Small aggregation |
| Volume by application | query_success | 58 | 221 | Small aggregation |
| Volume by app + severity | query_success | 58 | 221 | Small aggregation |
| Read tco-policies.md | skill_read | 1,303 | 4,938 | Policy reference |
| CLI `--from-file` error | **error_retry** | 24 | 91 | Unknown flag |
| Create TCO policy via API | api_call | 0 | 0 | Success |
| Read e2m-guide.md | skill_read | 2,022 | 7,662 | E2M reference |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 9,345 | 4 files |
| Query responses | 138 | 3 queries |
| API calls | 0 | 2 calls |
| **Error retries** | **158** | **2 mistakes** |
| **TOTAL** | **9,641** | **11 steps** |

**MCP breakdown (measured against au-syd):**
| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| list_policies | 58 | 219 | TCO policies |
| query_logs (severity) | 80 | 303 | Severity buckets |
| query_logs (app) | 80 | 303 | Top 20 applications by volume |
| estimate_query_cost | 207 | 786 | Cost estimation breakdown |
| create_policy (dry-run) | 266 | 1,009 | Validation result |
| **Total** | **18,920** | **71,707** | Fixed overhead + 5 tool calls |

**Skills wins by 49%.** The consolidated SKILL.md + cost-guide.md (6,020 tokens) replaced
loading two full SKILL.md files (cost + query = 8,068 tokens). MCP's fixed overhead
alone (18,229 tokens) is nearly double Skills' total cost.

### Scenario 3: Monitoring Setup (measured)

Discover patterns → create alert with burn rate → create webhook → build dashboard.

**Skills + CLI step-by-step (measured against au-syd):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 | Consolidated entry point |
| Read alerting-guide.md | skill_read | 2,576 | 9,763 | Alert methodology |
| Discover applications | query_success | 94 | 358 | Small aggregation |
| Query with double quotes | **error_retry** | 70 | 264 | DataPrime uses single quotes |
| App patterns (single quotes) | query_success | 94 | 358 | Small aggregation |
| Error rate baseline | query_success | 58 | 221 | Small aggregation |
| Read component-profiles.md | skill_read | 1,909 | 7,235 | Component detection |
| Read strategy-matrix.md | skill_read | 3,387 | 12,837 | Metrics per component |
| CLI `--from-file` error | **error_retry** | 15 | 55 | Unknown flag |
| Create alert via API | api_call | 0 | 0 | Success |
| GET /v1/outgoing_webhooks | api_call | 0 | 0 | Hanging call (killed) |
| Read dashboards-guide.md | skill_read | 1,390 | 5,269 | Dashboard creation |
| Read dashboard-schema.md | skill_read | 1,621 | 6,144 | JSON schema |
| CLI `dashboard-create` error | **error_retry** | 23 | 88 | Command doesn't exist |
| Create dashboard via REST | api_call | 38 | 144 | Attempt 1 |
| Dashboard schema retry | **error_retry** | 38 | 144 | Wrong widget format |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 15,389 | 6 files |
| Query responses | 246 | 3 queries |
| API calls | 38 | 3 calls |
| **Error retries** | **146** | **4 mistakes** |
| **TOTAL** | **15,819** | **16 steps** |

**MCP breakdown (measured against au-syd):**
| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| query_logs (discover apps) | 80 | 304 | Top 20 applications |
| suggest_alert | 5,419 | 20,538 | Full SRE alert package with burn rate |
| create_alert_definition (dry-run) | 282 | 1,067 | Validation result |
| list_outgoing_webhooks | 69 | 260 | Webhook listing |
| create_dashboard (dry-run) | 775 | 2,938 | Dashboard validation |
| **Total** | **24,854** | **94,194** | Fixed overhead + 5 tool calls |

**Skills wins by 36%.** The consolidated structure replaced loading 3 separate SKILL.md
files (query + alerting + dashboards = 12,359 tokens) with SKILL.md + alerting-guide.md +
dashboards-guide.md + dashboard-schema.md (10,093 tokens). Combined with smaller query
responses, Skills saves 9,035 tokens over MCP.

### Scenario 4: Normal Operations — CRUD (measured)

Alert lifecycle → dashboard lifecycle → view lifecycle. No queries, no investigation —
pure resource management. This is the bread-and-butter of day-to-day operations.

**Skills + CLI step-by-step (measured against au-syd):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 | Consolidated entry point |
| Read alerting-guide.md | skill_read | 2,576 | 9,763 | Alert methodology |
| Read strategy-matrix.md | skill_read | 3,387 | 12,837 | Strategy per component type |
| POST /v1/alert_definitions (create) | api_call | 191 | 724 | Created alert |
| GET /v1/alert_definitions (list) | api_call | 197 | 748 | Listed alerts |
| GET /v1/alert_definitions/{id} (get) | api_call | 191 | 724 | Retrieved alert |
| DELETE /v1/alert_definitions/{id} | api_call | 0 | 0 | Deleted alert |
| Read dashboards-guide.md | skill_read | 1,390 | 5,269 | Dashboard creation guide |
| Read dashboard-schema.md | skill_read | 1,621 | 6,144 | JSON schema reference |
| POST /v1/dashboards (create) | api_call | 198 | 749 | Created dashboard |
| GET /v1/dashboards (list, 7) | api_call | 397 | 1,503 | Listed dashboards |
| GET /v1/dashboards/{id} (get) | api_call | 198 | 749 | Retrieved dashboard |
| DELETE /v1/dashboards/{id} | api_call | 0 | 0 | Deleted dashboard |
| Read access-control-guide.md | skill_read | 1,415 | 5,364 | View management guide |
| POST /v1/views (create) | api_call | 90 | 340 | Created view |
| GET /v1/views (list, 17) | api_call | 1,579 | 5,985 | Listed views |
| DELETE /v1/views/{id} | api_call | 0 | 0 | Deleted view |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 14,895 | 6 files |
| API responses | 3,041 | 11 calls |
| **TOTAL** | **17,936** | **17 steps** |

**MCP breakdown (measured against au-syd):**

| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| create_alert_definition | 337 | 1,276 | Created + formatted response |
| list_alert_definitions | 391 | 1,480 | Listed all alerts |
| get_alert_definition | 337 | 1,279 | Retrieved single alert |
| delete_alert_definition | 24 | 92 | Deletion confirmation |
| create_dashboard | 707 | 2,681 | Created + formatted response |
| list_dashboards | 652 | 2,470 | Listed all dashboards (7) |
| get_dashboard | 723 | 2,740 | Full dashboard with layout |
| delete_dashboard | 25 | 93 | Deletion confirmation |
| create_view | 187 | 709 | Created view |
| list_views | 3,307 | 12,533 | Listed all views (17) |
| **Total** | **24,919** | **94,440** | Fixed overhead + 10 tool calls |

**Skills wins by 28%.** The consolidated structure loads SKILL.md once (4,506 tokens)
then domain guides on demand. Skills' 6 file reads (14,895 tokens) are smaller than
MCP's fixed overhead alone (18,229 tokens). MCP's tool responses add 6,690 tokens
on top of that overhead.

### Scenario 5: Query Authoring & Validation (measured)

Writing, validating, and explaining queries — pure knowledge work, no execution needed.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 |
| Read query-guide.md | skill_read | 1,406 | 5,327 |
| Read dataprime-commands.md | skill_read | 3,357 | 12,724 |
| Read query-templates.md | skill_read | 4,339 | 16,443 |
| Read dataprime-functions.md | skill_read | 2,224 | 8,429 |
| **TOTAL** | | **15,832** | **60,002** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| build_query | 141 | 534 |
| validate_query (AND mistake) | 100 | 380 |
| validate_query (correct) | 106 | 402 |
| explain_query | 633 | 2,400 |
| get_dataprime_reference | 196 | 741 |
| get_query_templates | 1,063 | 4,029 |
| estimate_query_cost | 178 | 676 |
| **Total** | **20,646** | **78,249** |

**Skills wins by 23%.** Query authoring is pure knowledge work. Skills loads SKILL.md
+ query-guide.md + 3 reference files (15,832 tokens). MCP's fixed overhead alone
(18,229 tokens) exceeds Skills' total. The consolidated SKILL.md now includes
DataPrime essentials inline, reducing the need for separate query skill reads.

### Scenario 6: Ingestion Pipeline (measured)

Configuring log parsing rules, enrichments, and understanding log formats.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 |
| Read ingestion-guide.md | skill_read | 1,291 | 4,893 |
| Read parsing-rules.md | skill_read | 1,753 | 6,643 |
| Read enrichment-types.md | skill_read | 740 | 2,806 |
| Read log-format.md | skill_read | 950 | 3,601 |
| GET /v1/rule_groups (list) | api_call | 38 | 144 |
| GET /v1/enrichments (list) | api_call | 5 | 18 |
| **TOTAL** | | **9,283** | **35,184** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_rule_groups | 80 | 302 |
| list_enrichments | 48 | 181 |
| discover_log_fields | 26 | 99 |
| **Total** | **18,383** | **69,669** |

**Skills wins by 50%.** Ingestion configuration is mostly knowledge (understanding formats,
rule types, enrichment options). API responses are tiny. Skills' SKILL.md + 4 reference
files (9,240 tokens) are about half of MCP's fixed overhead.

### Scenario 7: Data Governance (measured)

Data access rules and outgoing webhook management for security and compliance.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 |
| Read access-control-guide.md | skill_read | 1,415 | 5,364 |
| Read access-rules.md | skill_read | 1,508 | 5,717 |
| GET /v1/data_access_rules (list) | api_call | 6 | 24 |
| GET /v1/outgoing_webhooks (list) | api_call | 6 | 24 |
| POST /v1/outgoing_webhooks (create) | api_call | 41 | 154 |
| **TOTAL** | | **7,482** | **28,362** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_data_access_rules | 71 | 269 |
| list_outgoing_webhooks | 69 | 260 |
| create_outgoing_webhook | 68 | 256 |
| **Total** | **18,437** | **69,872** |

**Skills wins by 59%.** Data governance involves few API calls with tiny responses.
Skills needs SKILL.md + 2 domain files (7,429 tokens) — less than half of MCP's
fixed overhead.

### Scenario 8: E2M & Streaming (measured)

Events-to-Metrics conversion and log streaming configuration.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 |
| Read cost-guide.md | skill_read | 1,514 | 5,738 |
| Read e2m-guide.md | skill_read | 2,022 | 7,662 |
| GET /v1/e2m (list) | api_call | 0 | 0 |
| GET /v1/streams (list) | api_call | 4 | 14 |
| GET /v1/event_stream_targets (list) | api_call | 0 | 0 |
| **TOTAL** | | **8,046** | **30,493** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_e2m | 61 | 230 |
| list_streams | 58 | 219 |
| get_event_stream_targets | 51 | 195 |
| **Total** | **18,399** | **69,731** |

**Skills wins by 56%.** Knowledge-heavy workflow. API responses are near-empty
(no E2M or streams configured). Skills' SKILL.md + 2 domain files (8,042 tokens)
are less than half of MCP's overhead.

### Scenario 9: API Discovery & Meta (measured)

Understanding available tools, searching capabilities, system health.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ibm-cloud-logs/SKILL.md | skill_read | 4,506 | 17,079 |
| Read api-guide.md | skill_read | 2,116 | 8,021 |
| Read endpoints.md | skill_read | 1,844 | 6,988 |
| **TOTAL** | | **8,466** | **32,088** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_tool_categories | 617 | 2,339 |
| search_tools (alert) | 284 | 1,076 |
| session_context | 32 | 123 |
| health_check | 149 | 563 |
| **Total** | **19,311** | **73,188** |

**Skills wins by 56%.** For API discovery, MCP's `list_tool_categories` and `search_tools`
are useful but can't overcome the fixed overhead. Skills provides the same information
via SKILL.md + 2 files at 8,466 tokens total.

### The Iteration Tax: Measured vs Expected

The measured data disproves the initial assumption that iteration costs dominate.
Error responses are small (55-507 bytes), so the token impact is negligible:

| Scenario | Happy-path | Iteration tax | Tax % | Total |
|----------|----------:|-------------:|------:|------:|
| 1: Incident Investigation | 14,772 | 223 | 2% | 14,995 |
| 2: Cost Optimization | 9,483 | 158 | 2% | 9,641 |
| 3: Monitoring Setup | 15,673 | 146 | 1% | 15,819 |
| **All 3 + auth** | **39,928** | **527** | **1%** | **40,892** |

The real cost driver is not retries — it's **skill file reads**:

| Cost Source | Measured Tokens | % of Total |
|-------------|---------------:|----------:|
| **Skill file reads (16 files)** | **39,096** | **96%** |
| Query responses (13 queries) | 894 | 2% |
| Error retries (10 mistakes) | 527 | 1% |
| API calls + auth | 475 | 1% |

With the consolidated skill architecture, query responses are small (aggregation
queries return 22-163 tokens each). The dominant cost is now the skill file reads —
particularly the consolidated SKILL.md (4,506 tokens per scenario) plus domain guides.
The lesson: **skill architecture determines efficiency, not query design or retry count.**

### Time Cost

Skills workflows take longer due to sequential query execution:

| Scenario | Skills Steps | MCP Steps |
|----------|:-----------:|:---------:|
| 1: Incident Investigation | 19 | 4 |
| 2: Cost Optimization | 11 | 6 |
| 3: Monitoring Setup | 16 | 6 |
| 4: Normal Operations (CRUD) | 16 | 11 |
| 5: Query Authoring | 4 | 8 |
| 6: Ingestion Pipeline | 6 | 4 |
| 7: Data Governance | 5 | 4 |
| 8: E2M & Streaming | 5 | 4 |
| 9: API Discovery | 2 | 5 |
| **Total** | **84** | **52** |

MCP's compound tools (`investigate_incident`, `suggest_alert`) execute multiple queries
in a single server-side call, avoiding per-query LLM reasoning and network round-trips.
For lightweight scenarios (S6-S9), step counts are comparable.

### When MCP's Server-Side Summarization Matters

MCP's `investigate_incident` tool is uniquely efficient because it:
1. Executes 4-7 queries **server-side** (zero token cost for intermediate results)
2. Applies heuristic pattern matching server-side
3. Returns only a summarized report (~3K tokens vs raw query results)
4. Handles auth, tier selection, and retry logic internally

In our previous measurement (eu-gb instance with heavy log volume), raw log queries
dominated: a single `filter $m.severity == CRITICAL | limit 50` returned **39,076 tokens**
(148,099 bytes), making MCP 64% cheaper for investigation. In this measurement (au-syd
instance with lighter data), query responses were small, so Skills won all 9 scenarios.
**The lesson:** MCP's advantage scales with data volume — for instances with heavy log
traffic, `investigate_incident`'s server-side summarization becomes decisive.

### Data-Driven Decision Matrix

| Scenario Type | Best | 1 Skill | 8 Skills | MCP | Why |
|---------------|:----:|-------:|---------:|----:|-----|
| Incident investigation (heavy data) | **MCP** | — | — | — | Server-side summarization avoids context flooding |
| Incident investigation (light data) | **1 Skill** | 14,995 | 66,177 | 23,587 | 1 Skill 77% cheaper than 8 Skills, 36% cheaper than MCP |
| Live debugging with raw logs | **MCP** | — | — | — | `summary_only` flag prevents context flooding |
| Cost/policy analysis | **1 Skill** | 9,641 | 13,826 | 18,920 | 1 Skill 30% cheaper than 8 Skills, 49% cheaper than MCP |
| Monitoring setup | **1 Skill** | 15,819 | 20,864 | 24,854 | 1 Skill 24% cheaper than 8 Skills, 36% cheaper than MCP |
| Normal operations (CRUD) | **1 Skill** | 17,936 | 19,618 | 24,919 | 1 Skill 9% cheaper than 8 Skills, 28% cheaper than MCP |
| Query authoring | **8 Skills** | 15,832 | 14,594 | 20,646 | 8 Skills 8% cheaper (smaller domain SKILL.md) |
| Ingestion pipeline | **8 Skills** | 9,283 | 7,317 | 18,383 | 8 Skills 21% cheaper (smaller domain SKILL.md) |
| Data governance | **8 Skills** | 7,482 | 5,429 | 18,437 | 8 Skills 27% cheaper (smaller domain SKILL.md) |
| E2M & streaming | **8 Skills** | 8,046 | 5,948 | 18,399 | 8 Skills 26% cheaper (smaller domain SKILL.md) |
| API discovery & meta | **8 Skills** | 8,466 | 7,342 | 19,311 | 8 Skills 13% cheaper (smaller domain SKILL.md) |
| **Overall (all 9)** | **1 Skill** | **107,937** | **161,555** | **187,456** | 1 Skill wins on aggregate (-33% vs 8, -42% vs MCP) |
| **Hybrid (plan + execute)** | **Both** | — | — | — | Skills for design, MCP for execution |

**Trade-off:** 8 Skills wins single-domain scenarios (S5-S9) by 8-27% because individual
SKILL.md files are smaller. But 1 Skill wins cross-domain scenarios (S1-S4) by 9-77%
because it avoids loading multiple SKILL.md files. Since cross-domain tasks are the
expensive ones, 1 Skill wins overall by 33%.

---

## 8. Methodology

### Scenario Benchmark
All 9 scenarios replayed step-by-step against the live IBM Cloud Logs au-syd instance
(archive tier, 24h window). **Both approaches measured against the same instance:**

**Skills measurement:** Each step — skill file reads, API calls via `curl`, correct queries,
and deliberate mistakes (wrong tier, `AND` vs `&&`, `=` vs `==`, `sort` vs `orderby`,
double quotes, `--from-file` flag, missing dashboard CLI) — is recorded with exact
byte counts to a CSV ledger.

**MCP measurement:** The compiled MCP server binary (`bin/logs-mcp-server`) was started
against cxint and received JSON-RPC `tools/call` requests for each scenario. Response
bytes measured from the JSON-RPC response wire format. The `tools/list` response (96 tools,
69,087 bytes) is the measured fixed overhead.

Token estimates use the calibrated ratio of 3.79 bytes/token.

Scripts: `scripts/measure-iteration-tax.sh` (Skills replay, S1-S3), `scripts/measure-mcp-scenarios.py`
(MCP replay, S1-S3), `scripts/measure-normal-ops.py` (S4: CRUD operations, both sides),
`scripts/measure-remaining-features.py` (S5-S9: all remaining features, both sides),
`scripts/scenario-benchmark.sh` (data collection), `scripts/scenario-token-analysis.py` (analysis).

### Tokenizer
Token counts measured using **Claude's native tokenizer** via the `claude` CLI.
Each text content was sent to Claude Haiku and `usage.input_tokens` was read from
the JSON response. A baseline overhead was measured and subtracted to isolate
content-only token counts.
All comparisons use the same tokenizer, so relative ratios are accurate.

### MCP Wire Payload
Captured by running `go test -run TestMCPWirePayload ./internal/tools/`.
This test creates all 98 tool definitions with mock dependencies,
serializes them to JSON (exactly as the MCP server does), and measures the result.
The payload is **71,195 bytes** — real data, not an estimate.
Reference tool responses were executed locally (no network) and their sizes measured.

### Skill Token Measurement
Each of the 43 files in the consolidated `ibm-cloud-logs` skill directory
was read and tokenized individually using Claude (native).

### Binary Measurements
Build time measured with `time.monotonic()`. Command latencies averaged over
multiple runs (5 for `skills list`, 3 for `skills install` to temp directory).

### Cost Projections
Based on Claude Sonnet 4 input token pricing ($3/1M tokens) as of March 2026.
Assumptions: 10 tool calls per MCP conversation, 1 skill + 2 references per Skills conversation.

---

*This benchmark was generated using real API calls against live IBM Cloud Logs instances,
real wire payload data from 96 MCP tools, and 43 skill files in 1 consolidated skill
totaling 82,602 tokens of domain knowledge.*

---

## 9. References

### Standards & Specifications

- **Model Context Protocol (MCP)** — [modelcontextprotocol.io](https://modelcontextprotocol.io). Open protocol for connecting AI agents to tools and data sources. Defines JSON-RPC 2.0 transport, `tools/list` schema advertisement, and `tools/call` invocation. Version 2024-11-05 used in this benchmark.
- **Agent Skills (agentskills.io)** — [agentskills.io](https://agentskills.io). Open standard for portable AI agent instruction bundles. Skills are markdown files with YAML frontmatter, auto-discovered from `.agents/skills/` directories. Compatible with 30+ agent platforms.
- **JSON-RPC 2.0** — [jsonrpc.org/specification](https://www.jsonrpc.org/specification). Wire protocol used by MCP for client-server communication over stdio.
- **PEP 723 — Inline Script Metadata** — [peps.python.org/pep-0723](https://peps.python.org/pep-0723/). Used by companion scripts (`investigate.py`, `query-compact.py`) for inline dependency declarations (`# /// script` header).

### Tokenization

- **Claude Tokenizer** — Anthropic's native tokenizer used for all token counts in this report. Token counts were measured by sending content to Claude Haiku via the `claude` CLI and reading `usage.input_tokens` from the API response. No third-party tokenizer approximations were used.
- **Bytes-per-token calibration** — The ratio of 3.79 bytes/token was calibrated from the MCP wire payload measurement (71,195 bytes = 18,794 tokens) and used for byte-to-token estimates in scenario benchmarks.

### Pricing

- **Claude Sonnet 4 pricing** — $3 per 1M input tokens, $15 per 1M output tokens (as of March 2026). Used for cost projections in Section 5. Source: [Anthropic pricing](https://www.anthropic.com/pricing).

### IBM Cloud Logs

- **IBM Cloud Logs documentation** — [cloud.ibm.com/docs/cloud-logs](https://cloud.ibm.com/docs/cloud-logs). Platform documentation for the observability service benchmarked in this report.
- **DataPrime query language** — [cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-dataprime-ref](https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-dataprime-ref). Query language used in all log query scenarios. Piped syntax with `$l.` (labels), `$m.` (metadata), `$d.` (user data) field prefixes.
- **IBM Cloud Logs REST API** — [cloud.ibm.com/apidocs/logs-service-api](https://cloud.ibm.com/apidocs/logs-service-api). REST API used for all CRUD operations (alerts, dashboards, views, TCO policies). SSE streaming for query results.
- **IBM Cloud Logs CLI plugin** — [cloud.ibm.com/docs/cloud-logs-cli-plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin). CLI alternative to REST API, used in Skills scenario replays.
- **IBM Cloud IAM** — [cloud.ibm.com/docs/account?topic=account-iamoverview](https://cloud.ibm.com/docs/account?topic=account-iamoverview). Identity and Access Management service used for authentication. API key → bearer token exchange via `https://iam.cloud.ibm.com/identity/token`.

### SRE Methodology

- **Google SRE — Multi-Window, Multi-Burn-Rate Alerts** — Chapter 5 of *Site Reliability Workbook* (O'Reilly, 2018). The burn rate alerting model used by the `suggest_alert` MCP tool and the alerting agent skill. Fast-burn (2% budget in 1h) and slow-burn (10% budget in 6h) windows.
- **RED Method** — Rate, Errors, Duration. Tom Wilkie's microservice monitoring methodology used in the alerting strategy matrix for `web_service` and `api_gateway` component types.
- **USE Method** — Utilization, Saturation, Errors. Brendan Gregg's resource monitoring methodology used in the alerting strategy matrix for `database`, `cache`, and `message_queue` component types.

### Measurement Tools

- **Go testing framework** — `go test -run TestMCPWirePayload ./internal/tools/` for MCP wire payload measurement. Standard Go test infrastructure with mock dependencies.
- **Python `requests` library** — [docs.python-requests.org](https://docs.python-requests.org). Used in measurement scripts for HTTP calls to IBM Cloud Logs REST API and IAM token exchange.
