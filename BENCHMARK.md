# MCP vs Agent Skills: Benchmark Report

**IBM Cloud Logs — Measured Performance Data**
**Date:** March 2026 | **Version:** 0.10.0 | **Tokenizer:** Claude (native)

> All token counts in this report are **measured** using the Claude (native) tokenizer.
> Wire payload sizes are from actual Go test execution (`TestMCPWirePayload`).
> Scenario benchmarks use real queries against IBM Cloud Logs (cxint eu-gb instance).

## Executive Summary

| Metric | MCP (measured) | Skills (measured) | Ratio |
|--------|---------------:|------------------:|------:|
| Fixed context overhead | 18,794 tokens | 0 tokens | — |
| Per-conversation cost (typical) | ~26,224 tokens | ~4,608 tokens | 5.7x |
| 3-scenario total (happy-path) | 69,382 tokens | 48,990 tokens | 1.4x |
| **3-scenario total (realistic)** | **71,382 tokens** | **71,190 tokens** | **~1.0x** |
| Domain knowledge available | 98 tool definitions | 98,018 tokens across 8 skills | — |
| Wire payload size | 71,195 bytes (69.5 KB) | N/A | — |
| Avg tool response size | 593 tokens | N/A (in-context) | — |
| Binary size impact | — | +335,021 bytes embedded | — |

**Key finding:** On the happy path, Skills + CLI consumed **48,990 tokens** vs MCP's
**69,382 tokens** (29% less). But factoring in real-world iteration costs (query retries,
CLI flag discovery, tier misses, SSE parsing), Skills rises to **71,190 tokens** — nearly
identical to MCP's **71,382 tokens**. Skills pays a **45% iteration tax**; MCP pays only
**3%** because it handles retries, auth, and formatting server-side. MCP is also **4x
faster** (53s vs 213s). See [Section 7](#7-real-world-scenario-benchmark-measured).

---

## 1. MCP Wire Payload (Measured)

The MCP server registers **98 tools**. On connection, every tool's name, description,
and input schema is sent to the agent via `tools/list`. These tokens are **always present**
in the context window, whether or not the tools are used.

> **Measured via Go test:** `go test -run TestMCPWirePayload ./internal/tools/`
> The actual `tools/list` JSON-RPC response is **71,195 bytes** (69.5 KB) on the wire.
> Token count: **18,794 tokens** (measured via Claude (native)).

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

Three end-to-end workflows from the blog, executed both ways:

| Scenario | Skills+CLI Tokens | MCP Tokens | Winner | Delta |
|----------|------------------:|-----------:|--------|------:|
| 1: Incident Investigation (agg-only) | 18,053 | 24,644 | Skills | -27% |
| 2: Cost Optimization | 13,669 | 22,294 | Skills | -39% |
| 3: Monitoring Setup | 17,268 | 22,444 | Skills | -23% |
| **Total (all 3)** | **48,990** | **69,382** | **Skills** | **-29%** |

### Scenario 1: Incident Investigation

Global error scan → component deep-dive → heuristic matching → alert creation.

**Skills + CLI breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| investigation SKILL.md | 4,292 | Investigation methodology, modes, heuristics |
| query SKILL.md | 4,146 | DataPrime syntax for writing CLI queries |
| investigation-queries.md | 1,631 | Full query text for all 3 investigation modes |
| alerting SKILL.md | 5,149 | RED/USE methodology, burn rate math |
| burn-rate-math.md | 1,564 | Threshold calculation formulas |
| CLI query responses (5 agg) | 1,272 | error-rate, timeline, patterns, subsystems, deps |
| **Total** | **18,053** | 5 skill files + 5 aggregation queries |

**MCP breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| Fixed overhead (98 tools) | 18,794 | Always present in context |
| investigate_incident response | 3,000 | Server-side summarization of 4-7 queries |
| suggest_alert response | 1,500 | SRE-grade alert with burn rate |
| Other tool responses (5) | 1,350 | discover, describe, session, create alert/webhook |
| **Total** | **24,644** | Fixed overhead + 7 tool calls |

**Critical finding:** If the agent fetches raw logs (`limit 200`) instead of aggregation queries,
Skills + CLI jumps to **103,088 tokens** — a 4.2x increase. MCP's `investigate_incident`
avoids this by executing queries server-side and returning only the summary. **Aggregation-only
queries are essential for Skills efficiency.**

### Scenario 2: Cost Optimization

List TCO policies → analyze volume by severity/app → recommend tier changes.

**Skills + CLI breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| cost-optimization SKILL.md | 3,922 | TCO policies, tier selection, E2M |
| query SKILL.md | 4,146 | DataPrime syntax |
| tco-policies.md | 1,303 | Policy configuration reference |
| e2m-guide.md | 2,022 | Events-to-Metrics guide |
| CLI query responses (4) | 2,277 | policies, severity volume, app volume, app×severity |
| **Total** | **13,669** | 4 skill files + 4 queries |

**MCP breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| Fixed overhead (98 tools) | 18,794 | Always present |
| Tool responses (7) | 3,500 | list_policies, 2×query_logs, estimate_cost, 2×create_policy |
| **Total** | **22,294** | Fixed overhead + 7 tool calls |

**Skills wins by 39%.** Cost optimization uses only aggregation queries (small responses),
so Skills' lower fixed overhead dominates.

### Scenario 3: Monitoring Setup

Discover patterns → create alert with burn rate → create webhook → build dashboard.

**Skills + CLI breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| query SKILL.md | 4,146 | DataPrime syntax |
| alerting SKILL.md | 5,149 | Alert design methodology |
| strategy-matrix.md | 3,396 | Component type detection + metrics |
| dashboards SKILL.md | 3,064 | Widget types, dashboard JSON schema |
| CLI responses (6) | 1,513 | apps, patterns, error-rate, webhooks, alerts, dashboards |
| **Total** | **17,268** | 4 skill files + 6 queries/API calls |

**MCP breakdown:**
| Component | Tokens | Details |
|-----------|-------:|---------|
| Fixed overhead (98 tools) | 18,794 | Always present |
| Tool responses (6) | 3,650 | query_logs, suggest_alert, create_alert, webhook, dashboard, pin |
| **Total** | **22,444** | Fixed overhead + 6 tool calls |

### Realistic Cost: The Iteration Tax

The happy-path numbers above assume every query is correct on the first try, every CLI flag
is known, and every response is parsed cleanly. In practice, Skills + CLI workflows involve
**trial-and-error loops** that MCP avoids entirely because it handles queries, auth, tier
selection, and response formatting server-side.

**Iteration cost sources (observed during testing):**

| Cost Source | Tokens per Occurrence | Typical Frequency |
|-------------|----------------------:|:-----------------:|
| DataPrime syntax retry (AND→&&, =→==, quotes) | ~1,000 | 1-3 per scenario |
| CLI flag discovery (--prototype, no dashboard CLI) | ~1,000 | 1-2 per scenario |
| Tier miss (frequent_search empty → retry archive) | ~1,200 | 0-1 per scenario |
| SSE response format parsing | ~500 | First query only |
| Auth setup (IAM token exchange) | ~400 | Once per session |
| Agent reasoning per step | ~200 | Every step |
| Response data extraction per query | ~300 | Every query |

**With iteration costs included:**

| Scenario | Skills (happy) | Skills (realistic) | MCP | Winner |
|----------|---------------:|-------------------:|----:|--------|
| 1: Incident Investigation | 18,053 | 27,253 (+51%) | 25,244 | **MCP** |
| 2: Cost Optimization | 13,669 | 18,269 (+34%) | 23,094 | **Skills** |
| 3: Monitoring Setup | 17,268 | 25,668 (+49%) | 23,044 | **MCP** |
| **Total** | **48,990** | **71,190** (+45%) | **71,382** | **Tied** |

Skills iteration overhead: **+22,200 tokens** (45% of happy-path).
MCP iteration overhead: **+2,000 tokens** (3% of happy-path).

### Time Cost

Skills workflows take **4x longer** due to sequential query execution and retry loops:

| Scenario | Skills Time | MCP Time | Skills Steps | MCP Steps |
|----------|:----------:|:--------:|:-----------:|:---------:|
| 1: Incident Investigation | ~82s | ~27s | 14 | 7 |
| 2: Cost Optimization | ~51s | ~14s | 9 | 7 |
| 3: Monitoring Setup | ~80s | ~12s | 13 | 6 |
| **Total** | **~213s** | **~53s** | **36** | **20** |

MCP's compound tools (`investigate_incident`, `suggest_alert`) execute multiple queries
in a single server-side call, avoiding per-query LLM reasoning and network round-trips.

### Why MCP Wins for Investigation

MCP's `investigate_incident` tool is uniquely efficient because it:
1. Executes 4-7 queries **server-side** (zero token cost for intermediate results)
2. Applies heuristic pattern matching server-side
3. Returns only a summarized report (~3K tokens vs ~80K for raw query results)
4. Handles auth, tier selection, and retry logic internally (zero iteration tax)

If a Skills-based agent fetches raw logs, MCP is **4.2x more efficient**.
If it uses only aggregation queries with zero retries, Skills is **27% more efficient**.
With realistic retries factored in, they are **roughly equivalent** on tokens.

### Data-Driven Decision Matrix

| Scenario Type | Recommended | Why |
|---------------|:-----------:|-----|
| Incident investigation | **MCP** | Server-side summarization + zero iteration tax |
| Cost/policy analysis | **Skills** | Aggregation queries, still wins with retries |
| Monitoring setup (execution) | **MCP** | Dashboard/alert creation needs API, fewer retries |
| Monitoring setup (planning) | **Skills** | Architecture guidance needs no execution |
| Query writing (no execution) | **Skills** | Zero data needed, pure syntax knowledge |
| Live debugging with raw logs | **MCP** | `summary_only` flag reduces token blast |
| Architecture/design guidance | **Skills** | Zero overhead, pure knowledge |
| **Hybrid (plan + execute)** | **Both** | Skills for design, MCP for execution |

---

## 8. Methodology

### Scenario Benchmark
Queries executed via `curl` against IBM Cloud Logs REST API (cxint eu-gb instance,
archive tier, 24h window). Response sizes measured as raw SSE stream bytes.
Skill file sizes measured from the embedded `.agents/skills/` directory.
Token estimates use the calibrated ratio of 3.79 bytes/token.
Scripts: `scripts/scenario-benchmark.sh` (data collection), `scripts/scenario-token-analysis.py` (analysis).

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
