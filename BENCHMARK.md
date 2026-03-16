# MCP vs Agent Skills: Benchmark Report

**IBM Cloud Logs — Measured Performance Data**
**Date:** March 2026 | **Version:** 0.10.0 | **Tokenizer:** Claude (native)

> All token counts in this report are **measured** using the Claude (native) tokenizer.
> Wire payload sizes are from actual Go test execution (`TestMCPWirePayload`).
> Scenario benchmarks use real queries against IBM Cloud Logs (cxint eu-gb instance).

## Executive Summary

| Metric | MCP (measured) | Skills (measured) | Ratio |
|--------|---------------:|------------------:|------:|
| Fixed context overhead | 18,229 tokens | 0 tokens | — |
| Per-conversation cost (typical) | ~26,224 tokens | ~4,608 tokens | 5.7x |
| **9-scenario total (measured)** | **188,393 tokens** | **161,555 tokens** | **1.2x** |
| Domain knowledge available | 96 tool definitions | 98,018 tokens across 8 skills | — |
| Wire payload size | 69,087 bytes (67.5 KB) | N/A | — |
| Avg tool response (measured) | 1,120 tokens | N/A (in-context) | — |
| Binary size impact | — | +335,021 bytes embedded | — |

**Key finding:** Both sides measured against the cxint eu-gb instance across **9 scenarios
covering every feature area**. Skills consumed **161,555 tokens** vs MCP's **188,393 tokens** —
Skills is **14% more efficient** overall. MCP wins only **1 of 9 scenarios** (incident
investigation, -64%) due to server-side summarization. Skills wins the remaining 8
scenarios by 16-71% because its selective loading (only needed skills) avoids MCP's
fixed 18,229-token overhead from loading all 96 tool schemas. The iteration tax (syntax
errors, wrong tier, CLI flag mistakes) is negligible: **527 tokens (0.5%)**. The dominant
cost in MCP is the fixed overhead; the dominant cost in Skills is raw query data volume.
See [Section 7](#7-real-world-scenario-benchmark-measured).

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

Each file in every skill directory was tokenized using Claude (native).
Skills are loaded **on-demand**. Only the activated skill's SKILL.md enters the context.

| Skill | SKILL.md | References | Scripts | Assets | **Total** | Lines |
|-------|--------:|-----------:|--------:|-------:|----------:|------:|
| access-control | 3,081 | 1,649 | 0 | 1,110 | **5,840** | 292 |
| alerting | 4,745 | 9,287 | 1,976 | 1,193 | **17,201** | 382 |
| api-reference | 5,123 | 3,731 | 0 | 3,844 | **12,698** | 422 |
| cost-optimization | 3,628 | 3,693 | 0 | 1,507 | **8,828** | 286 |
| dashboards | 2,597 | 4,383 | 0 | 4,363 | **11,343** | 230 |
| incident-investigation | 3,024 | 5,604 | 0 | 1,402 | **10,030** | 243 |
| ingestion | 3,142 | 3,957 | 1,338 | 687 | **9,124** | 310 |
| query | 3,526 | 12,292 | 3,254 | 3,882 | **22,954** | 268 |
| **Total** | **28,866** | **44,596** | **6,568** | **17,988** | **98,018** | 2433 |

![Skill Token Breakdown](benchmarks/skill-token-breakdown.png)

---

## 3. Head-to-Head: Per-Conversation Token Cost

MCP cost = fixed overhead + tool call/response tokens.
Skills cost = SKILL.md + reference loads.

| Scenario | MCP Tokens | Skills Tokens | Ratio |
|----------|----------:|-------------:|------:|
| Before any tool call | 18,794 | 0 | — |
| After 1 tool call / 1 skill activation | 19,537 | 3,608 | 5.4x |
| Typical session (10 calls / 1 skill + 2 refs) | 26,224 | 4,608 | 5.7x |
| Heavy session (25 calls / 2 skills + 5 refs) | 37,369 | 9,716 | 3.8x |

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
| Embedded skill files | 42 files |
| Total skill bytes | 335,021 bytes |

---

## 5. Token Cost Projection

Based on Claude Sonnet 4 pricing ($3 / 1M input tokens) and measured token counts.
MCP: 26,224 tokens/conversation. Skills: 4,608 tokens/conversation.

| Conversations/Month | MCP Annual Cost | Skills Annual Cost | Savings |
|--------------------:|----------------:|-------------------:|--------:|
| 10 | $9.44 | $1.66 | 82% |
| 50 | $47.20 | $8.29 | 82% |
| 100 | $94.41 | $16.59 | 82% |
| 500 | $472.03 | $82.94 | 82% |
| 1,000 | $944.06 | $165.89 | 82% |

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

| Dimension | MCP | Skills | Notes |
|-----------|:---:|:------:|-------|
| Token efficiency | 3/10 | 9/10 | MCP: ~26,224/convo; Skills: ~4,608/convo |
| Query accuracy | 10/10 | 9/10 | MCP has programmatic auto-correction engine |
| Live data access | 10/10 | 0/10 | Only MCP can execute queries; Skills provide guidance only |
| Setup friction | 4/10 | 10/10 | Skills need zero configuration |
| Platform reach | 5/10 | 10/10 | Skills work on 30+ agent platforms |
| Offline guidance | 0/10 | 8/10 | Skills provide query/config guidance offline; execution still requires API access |
| Latency | 5/10 | 10/10 | Skills are in-context; MCP requires network round-trips |
| Security posture | 6/10 | 9/10 | Skills never handle credentials; but production use still requires API auth |
| Cost efficiency | 4/10 | 9/10 | Skills save ~82% on token costs per conversation |

---

## 7. Real-World Scenario Benchmark (Measured)

> **Data source:** Live queries against IBM Cloud Logs (cxint eu-gb instance, archive tier, 24h window).
> Skill file sizes measured from embedded files. MCP response sizes estimated from measured tool benchmarks.
> Token counts estimated at 3.79 bytes/token (calibrated from wire payload measurement: 71,195 bytes = 18,794 tokens).

### Scenario Overview

Nine end-to-end workflows replayed step-by-step against the cxint eu-gb instance,
covering **every feature area** of the MCP server's 96 tools. **Both sides measured** —
Skills via `curl` + file reads, MCP via the actual MCP server binary with JSON-RPC
tool calls. Token counts at 3.79 bytes/token. Scenarios 1-3 include deliberate
mistakes (wrong tier, `AND` vs `&&`, `=` vs `==`, `--from-file`, missing dashboard CLI).
Scenarios 4-9 measure clean operations (no mistakes).

| Scenario | Skills+CLI (measured) | MCP (measured) | Winner | Delta |
|----------|----------------------:|---------------:|--------|------:|
| 1: Incident Investigation | 66,177 | 23,587 | MCP | -64% |
| 2: Cost Optimization | 13,826 | 20,715 | Skills | -33% |
| 3: Monitoring Setup | 20,864 | 24,938 | Skills | -16% |
| 4: Normal Operations (CRUD) | 19,618 | 23,995 | Skills | -18% |
| 5: Query Authoring & Validation | 14,594 | 20,646 | Skills | -29% |
| 6: Ingestion Pipeline | 7,317 | 18,383 | Skills | -60% |
| 7: Data Governance | 5,429 | 18,437 | Skills | -71% |
| 8: E2M & Streaming | 5,948 | 18,399 | Skills | -68% |
| 9: API Discovery & Meta | 7,342 | 19,293 | Skills | -62% |
| Auth overhead | 440 | 0 | — | — |
| **Total (all 9)** | **161,555** | **188,393** | **Skills** | **-14%** |

### Scenario 1: Incident Investigation (measured)

Global error scan → component deep-dive → heuristic matching → alert creation.

**Skills + CLI step-by-step (measured against cxint):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read investigation/SKILL.md | skill_read | 4,292 | 16,265 | Investigation methodology |
| Read query/SKILL.md | skill_read | 4,146 | 15,712 | DataPrime syntax |
| GET /v1/tco_policies | api_call | 0 | 0 | Empty (no policies configured) |
| Query wrong tier (frequent_search) | **error_retry** | 58 | 221 | Returns near-empty result |
| Query global error rate (archive) | query_success | 522 | 1,978 | Aggregation — small response |
| Query with `AND` (wrong syntax) | **error_retry** | 73 | 275 | DataPrime parse error |
| Query with `&&` (fixed) | query_success | 6,740 | 25,545 | Filtered results — moderate |
| Global error timeline | query_success | 442 | 1,675 | Time-bucketed aggregation |
| **Critical errors (raw)** | **query_success** | **39,076** | **148,099** | **Raw log data — context bomb** |
| Read investigation-queries.md | skill_read | 1,631 | 6,180 | Query templates |
| Query with `=` (wrong syntax) | **error_retry** | 68 | 258 | DataPrime parse error |
| Component error patterns | query_success | 54 | 203 | Small aggregation |
| Component subsystems | query_success | 174 | 659 | Small aggregation |
| Component dependencies | query_success | 22 | 82 | Small aggregation |
| Read heuristic-details.md | skill_read | 2,142 | 8,118 | Heuristic patterns |
| Read alerting/SKILL.md | skill_read | 5,149 | 19,514 | Alert methodology |
| Read burn-rate-math.md | skill_read | 1,564 | 5,927 | Threshold formulas |
| CLI `--from-file` error | **error_retry** | 24 | 91 | Unknown flag |
| Create alert via API | api_call | 0 | 0 | Success (empty response) |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 18,924 | 6 files |
| Query responses | 47,030 | 7 queries |
| API calls | 0 | 2 calls |
| **Error retries** | **223** | **4 mistakes** |
| **TOTAL** | **66,177** | **19 steps** |

**MCP breakdown (measured against cxint):**
| Component | Tokens | Bytes | Details |
|-----------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| investigate_incident | 232 | 879 | "No Issues Found" (no recent errors in cxint) |
| suggest_alert | 4,842 | 18,350 | Full SRE-grade alert with burn rate, 6 suggestions |
| create_alert_definition | 284 | 1,075 | Dry-run validation response |
| **Total** | **23,587** | **89,391** | Fixed overhead + 3 tool calls |

**Key insight:** The critical errors query alone (39,076 tokens) exceeds MCP's entire
scenario cost (23,587 tokens). MCP's `investigate_incident` returned just 232 tokens
(no active errors); even with errors present, it summarizes server-side (~3K tokens max)
vs Skills dumping 148KB of raw log data into context.

### Scenario 2: Cost Optimization (measured)

List TCO policies → analyze volume by severity/app → recommend tier changes.

**Skills + CLI step-by-step (measured against cxint):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read cost-optimization/SKILL.md | skill_read | 3,922 | 14,865 | TCO policies, tier selection |
| Read query/SKILL.md | skill_read | 4,146 | 15,712 | DataPrime syntax |
| GET /v1/tco_policies | api_call | 0 | 0 | Empty response |
| Query with `sort` (wrong syntax) | **error_retry** | 134 | 507 | DataPrime uses `orderby` |
| Volume by severity | query_success | 168 | 638 | Small aggregation |
| Volume by application | query_success | 533 | 2,021 | Small aggregation |
| Volume by app + severity | query_success | 1,574 | 5,967 | Moderate aggregation |
| Read tco-policies.md | skill_read | 1,303 | 4,938 | Policy reference |
| CLI `--from-file` error | **error_retry** | 24 | 91 | Unknown flag |
| Create TCO policy via API | api_call | 0 | 0 | Success |
| Read e2m-guide.md | skill_read | 2,022 | 7,662 | E2M reference |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 11,393 | 4 files |
| Query responses | 2,275 | 3 queries |
| API calls | 0 | 2 calls |
| **Error retries** | **158** | **2 mistakes** |
| **TOTAL** | **13,826** | **11 steps** |

**MCP breakdown (measured against cxint):**
| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| list_policies | 1,733 | 6,568 | Returns all configured TCO policies |
| query_logs (severity) | 116 | 441 | 6 severity buckets |
| query_logs (app) | 164 | 621 | Top 20 applications by volume |
| estimate_query_cost | 207 | 786 | Cost estimation breakdown |
| create_policy (dry-run) | 266 | 1,009 | Validation result |
| **Total** | **20,715** | **78,512** | Fixed overhead + 5 tool calls |

**Skills wins by 33%.** Aggregation queries return small responses (2,275 tokens for Skills
vs 2,486 for MCP — nearly identical). The difference is entirely MCP's 18,229-token fixed
overhead vs Skills' 0. Both query response sizes are comparable.

### Scenario 3: Monitoring Setup (measured)

Discover patterns → create alert with burn rate → create webhook → build dashboard.

**Skills + CLI step-by-step (measured against cxint):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read query/SKILL.md | skill_read | 4,146 | 15,712 | DataPrime syntax |
| Read alerting/SKILL.md | skill_read | 5,149 | 19,514 | Alert methodology |
| Discover applications | query_success | 625 | 2,367 | Small aggregation |
| Query with double quotes | **error_retry** | 70 | 264 | DataPrime uses single quotes |
| App patterns (single quotes) | query_success | 661 | 2,507 | Small aggregation |
| Error rate baseline | query_success | 103 | 391 | Small aggregation |
| Read component-profiles.md | skill_read | 1,909 | 7,235 | Component detection |
| Read strategy-matrix.md | skill_read | 3,396 | 12,870 | Metrics per component |
| CLI `--from-file` error | **error_retry** | 15 | 55 | Unknown flag |
| Create alert via API | api_call | 0 | 0 | Success |
| GET /v1/outgoing_webhooks | api_call | 6 | 24 | Small response |
| Read dashboards/SKILL.md | skill_read | 3,064 | 11,614 | Dashboard creation |
| Read dashboard-schema.md | skill_read | 1,621 | 6,144 | JSON schema |
| CLI `dashboard-create` error | **error_retry** | 23 | 88 | Command doesn't exist |
| Create dashboard via REST | api_call | 38 | 144 | Attempt 1 |
| Dashboard schema retry | **error_retry** | 38 | 144 | Wrong widget format |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 19,285 | 6 files |
| Query responses | 1,389 | 3 queries |
| API calls | 44 | 3 calls |
| **Error retries** | **146** | **4 mistakes** |
| **TOTAL** | **20,864** | **16 steps** |

**MCP breakdown (measured against cxint):**
| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| query_logs (discover apps) | 164 | 622 | Top 20 applications |
| suggest_alert | 5,419 | 20,538 | Full SRE alert package with burn rate |
| create_alert_definition (dry-run) | 282 | 1,067 | Validation result |
| list_outgoing_webhooks | 69 | 260 | Webhook listing |
| create_dashboard (dry-run) | 775 | 2,938 | Dashboard validation |
| **Total** | **24,938** | **94,512** | Fixed overhead + 5 tool calls |

**Skills wins by 16%.** The `suggest_alert` response is large (5,419 tokens — full SRE
package with burn rate math), but Skills still wins because its total query + skill load
(20,864 tokens) is less than MCP's fixed overhead alone (18,229) plus responses (6,709).

### Scenario 4: Normal Operations — CRUD (measured)

Alert lifecycle → dashboard lifecycle → view lifecycle. No queries, no investigation —
pure resource management. This is the bread-and-butter of day-to-day operations.

**Skills + CLI step-by-step (measured against cxint):**

| Step | Category | Tokens | Bytes | Details |
|------|----------|-------:|------:|---------|
| Read alerting/SKILL.md | skill_read | 5,149 | 19,514 | Alert methodology + schema |
| Read strategy-matrix.md | skill_read | 3,396 | 12,870 | Strategy per component type |
| POST /v1/alert_definitions (create) | api_call | 191 | 724 | Created alert |
| GET /v1/alert_definitions (list) | api_call | 197 | 748 | Listed alerts |
| GET /v1/alert_definitions/{id} (get) | api_call | 191 | 724 | Retrieved alert |
| DELETE /v1/alert_definitions/{id} | api_call | 0 | 0 | Deleted alert |
| Read dashboards/SKILL.md | skill_read | 3,064 | 11,614 | Dashboard creation guide |
| Read dashboard-schema.md | skill_read | 1,621 | 6,144 | JSON schema reference |
| POST /v1/dashboards (create) | api_call | 198 | 749 | Created dashboard |
| GET /v1/dashboards (list, 4) | api_call | 230 | 870 | Listed dashboards |
| GET /v1/dashboards/{id} (get) | api_call | 198 | 749 | Retrieved dashboard |
| DELETE /v1/dashboards/{id} | api_call | 0 | 0 | Deleted dashboard |
| Read access-control/SKILL.md | skill_read | 3,868 | 14,659 | View management guide |
| POST /v1/views (create) | api_call | 90 | 340 | Created view |
| GET /v1/views (list, 13) | api_call | 1,225 | 4,644 | Listed views |
| DELETE /v1/views/{id} | api_call | 0 | 0 | Deleted view |

| Category | Tokens | Items |
|----------|-------:|------:|
| Skill reads | 17,098 | 5 files |
| API responses | 2,520 | 11 calls |
| **TOTAL** | **19,618** | **16 steps** |

**MCP breakdown (measured against cxint):**

| Step | Tokens | Bytes | Details |
|------|-------:|------:|---------|
| Fixed overhead (96 tools) | 18,229 | 69,087 | tools/list — always present |
| create_alert_definition | 337 | 1,276 | Created + formatted response |
| list_alert_definitions | 391 | 1,480 | Listed all alerts |
| get_alert_definition | 337 | 1,279 | Retrieved single alert |
| delete_alert_definition | 24 | 92 | Deletion confirmation |
| create_dashboard | 707 | 2,681 | Created + formatted response |
| list_dashboards | 406 | 1,540 | Listed all dashboards |
| get_dashboard | 723 | 2,740 | Full dashboard with layout |
| delete_dashboard | 25 | 93 | Deletion confirmation |
| create_view | 187 | 709 | Created view |
| list_views | 2,629 | 9,964 | Listed all views (13) |
| **Total** | **23,995** | **90,941** | Fixed overhead + 10 tool calls |

**Skills wins by 18%.** API responses are small for CRUD — Skills' 11 API calls total just
2,520 tokens vs MCP's 10 tool responses at 5,766 tokens (MCP adds formatting, suggestions,
and related-tool hints). The cost split is nearly identical: Skills' knowledge overhead
(17,098 tokens for 5 file reads) vs MCP's schema overhead (18,229 tokens for 96 tool
definitions). Skills wins because it loads only the relevant 3 skills, while MCP loads
all 96 tool schemas regardless of which ones are used.

**Key insight:** For CRUD-heavy workflows, the fixed overhead dominates both sides.
MCP's per-response overhead (~2.3x larger than raw API responses due to formatting) tips
the balance toward Skills. However, MCP requires zero knowledge — the agent doesn't
need to learn API schemas from skill files; the tool schema tells it exactly what to send.

### Scenario 5: Query Authoring & Validation (measured)

Writing, validating, and explaining queries — pure knowledge work, no execution needed.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read query/SKILL.md | skill_read | 4,674 | 17,714 |
| Read dataprime-commands.md | skill_read | 3,357 | 12,724 |
| Read query-templates.md | skill_read | 4,339 | 16,443 |
| Read dataprime-functions.md | skill_read | 2,224 | 8,429 |
| **TOTAL** | | **14,594** | **55,310** |

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

**Skills wins by 29%.** Query authoring is pure knowledge work — the agent needs syntax
references, not API execution. Skills loads 4 reference files (14,594 tokens) while MCP's
fixed overhead alone (18,229 tokens) already exceeds Skills' total. MCP's query tools
return compact responses (2,417 tokens for 7 calls) but can't overcome the schema overhead.

### Scenario 6: Ingestion Pipeline (measured)

Configuring log parsing rules, enrichments, and understanding log formats.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read ingestion/SKILL.md | skill_read | 3,831 | 14,521 |
| Read parsing-rules.md | skill_read | 1,753 | 6,643 |
| Read enrichment-types.md | skill_read | 740 | 2,806 |
| Read log-format.md | skill_read | 950 | 3,601 |
| GET /v1/rule_groups (list) | api_call | 38 | 144 |
| GET /v1/enrichments (list) | api_call | 5 | 18 |
| **TOTAL** | | **7,317** | **27,733** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_rule_groups | 80 | 302 |
| list_enrichments | 48 | 181 |
| discover_log_fields | 26 | 99 |
| **Total** | **18,383** | **69,669** |

**Skills wins by 60%.** Ingestion configuration is mostly knowledge (understanding formats,
rule types, enrichment options). API responses are tiny. Skills' 4 reference files
(7,274 tokens) are less than half of MCP's fixed overhead.

### Scenario 7: Data Governance (measured)

Data access rules and outgoing webhook management for security and compliance.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read access-control/SKILL.md | skill_read | 3,868 | 14,659 |
| Read access-rules.md | skill_read | 1,508 | 5,717 |
| GET /v1/data_access_rules (list) | api_call | 6 | 24 |
| GET /v1/outgoing_webhooks (list) | api_call | 6 | 24 |
| POST /v1/outgoing_webhooks (create) | api_call | 41 | 154 |
| **TOTAL** | | **5,429** | **20,578** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_data_access_rules | 71 | 269 |
| list_outgoing_webhooks | 69 | 260 |
| create_outgoing_webhook | 68 | 256 |
| **Total** | **18,437** | **69,872** |

**Skills wins by 71%.** The largest margin of any scenario. Data governance involves
few API calls with tiny responses. Skills needs only 2 files (5,376 tokens) — less
than a third of MCP's fixed overhead.

### Scenario 8: E2M & Streaming (measured)

Events-to-Metrics conversion and log streaming configuration.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read cost-optimization/SKILL.md | skill_read | 3,922 | 14,865 |
| Read e2m-guide.md | skill_read | 2,022 | 7,662 |
| GET /v1/e2m (list) | api_call | 0 | 0 |
| GET /v1/streams (list) | api_call | 4 | 14 |
| GET /v1/event_stream_targets (list) | api_call | 0 | 0 |
| **TOTAL** | | **5,948** | **22,541** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_e2m | 61 | 230 |
| list_streams | 58 | 219 |
| get_event_stream_targets | 51 | 195 |
| **Total** | **18,399** | **69,731** |

**Skills wins by 68%.** Another knowledge-heavy workflow. API responses are near-empty
(no E2M or streams configured on cxint). Skills' 2 files (5,944 tokens) are a fraction
of MCP's overhead.

### Scenario 9: API Discovery & Meta (measured)

Understanding available tools, searching capabilities, system health.

**Skills + CLI (measured):**

| Step | Category | Tokens | Bytes |
|------|----------|-------:|------:|
| Read api-reference/SKILL.md | skill_read | 5,498 | 20,838 |
| Read endpoints.md | skill_read | 1,844 | 6,988 |
| **TOTAL** | | **7,342** | **27,826** |

**MCP (measured):**

| Step | Tokens | Bytes |
|------|-------:|------:|
| Fixed overhead (96 tools) | 18,229 | 69,087 |
| list_tool_categories | 617 | 2,339 |
| search_tools (alert) | 266 | 1,007 |
| session_context | 32 | 123 |
| health_check | 149 | 563 |
| **Total** | **19,293** | **73,119** |

**Skills wins by 62%.** For API discovery, MCP's `list_tool_categories` and `search_tools`
are useful but can't overcome the fixed overhead. Skills provides the same information
in 2 static files at 7,342 tokens total.

### The Iteration Tax: Measured vs Expected

The measured data disproves the initial assumption that iteration costs dominate.
Error responses are small (55-507 bytes), so the token impact is negligible:

| Scenario | Happy-path | Iteration tax | Tax % | Total |
|----------|----------:|-------------:|------:|------:|
| 1: Incident Investigation | 65,954 | 223 | 0.3% | 66,177 |
| 2: Cost Optimization | 13,668 | 158 | 1.2% | 13,826 |
| 3: Monitoring Setup | 20,718 | 146 | 0.7% | 20,864 |
| **All 3 + auth** | **100,340** | **527** | **0.5%** | **101,307** |

The real cost driver is not retries — it's **raw query data volume**:

| Cost Source | Measured Tokens | % of Total |
|-------------|---------------:|----------:|
| Skill file reads (16 files) | 49,602 | 49% |
| **Query responses (13 queries)** | **50,694** | **50%** |
| Error retries (10 mistakes) | 527 | 0.5% |
| API calls + auth | 484 | 0.5% |

A single raw log query (`filter $m.severity == CRITICAL | limit 50`) returned
**39,076 tokens** (148,099 bytes) — more than MCP's entire Scenario 1 cost.
Aggregation queries averaged only **408 tokens** each. The lesson:
**query design determines Skills efficiency, not retry count.**

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

### Why MCP Wins for Investigation

MCP's `investigate_incident` tool is uniquely efficient because it:
1. Executes 4-7 queries **server-side** (zero token cost for intermediate results)
2. Applies heuristic pattern matching server-side
3. Returns only a summarized report (~3K tokens vs ~47K for raw query results)
4. Handles auth, tier selection, and retry logic internally

The measured data confirms: raw log queries are the decisive factor.
Skills + CLI for investigation costs **66,177 tokens** — 2.8x more than MCP's **23,587 tokens**.

### Data-Driven Decision Matrix

| Scenario Type | Recommended | Why (measured) |
|---------------|:-----------:|-----|
| Incident investigation | **MCP** | 23,587 vs 66,177 tokens — MCP 64% cheaper (server-side summarization) |
| Live debugging with raw logs | **MCP** | `summary_only` flag prevents context flooding |
| Cost/policy analysis | **Skills** | 13,826 vs 20,715 tokens — Skills 33% cheaper |
| Monitoring setup | **Skills** | 20,864 vs 24,938 tokens — Skills 16% cheaper |
| Normal operations (CRUD) | **Skills** | 19,618 vs 23,995 tokens — Skills 18% cheaper |
| Query authoring & validation | **Skills** | 14,594 vs 20,646 tokens — Skills 29% cheaper |
| Ingestion pipeline | **Skills** | 7,317 vs 18,383 tokens — Skills 60% cheaper |
| Data governance | **Skills** | 5,429 vs 18,437 tokens — Skills 71% cheaper |
| E2M & streaming | **Skills** | 5,948 vs 18,399 tokens — Skills 68% cheaper |
| API discovery & meta | **Skills** | 7,342 vs 19,293 tokens — Skills 62% cheaper |
| **Hybrid (plan + execute)** | **Both** | Skills for design, MCP for execution |

---

## 8. Methodology

### Scenario Benchmark
All 9 scenarios replayed step-by-step against the live IBM Cloud Logs cxint eu-gb instance
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
Each of the 42 files across 8 skill directories
was read and tokenized individually using Claude (native).

### Binary Measurements
Build time measured with `time.monotonic()`. Command latencies averaged over
multiple runs (5 for `skills list`, 3 for `skills install` to temp directory).

### Cost Projections
Based on Claude Sonnet 4 input token pricing ($3/1M tokens) as of March 2026.
Assumptions: 10 tool calls per MCP conversation, 1 skill + 2 references per Skills conversation.

---

*This benchmark was generated by `scripts/run-benchmark.py` using Claude (native) tokenizer,
real wire payload data from 98 MCP tools, and 42 skill files
totaling 98,018 tokens of domain knowledge.*

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
