#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = ["requests"]
# ///
"""
Comprehensive benchmark: Skills vs MCP full-conversation cost model.

Models complete LLM conversations with proper token accounting:
- Context accumulation (each API call re-sends full conversation history)
- Per-turn fixed overhead (system prompt + tool definitions on every call)
- Output tokens (assistant reasoning + tool call JSON, billed at higher rate)
- Dollar costs at Claude Sonnet and Opus pricing
- Wall-clock time for tool execution
- Error/retry costs

Usage:
    export LOGS_SERVICE_URL="https://<instance>.api.<region>.logs.cloud.ibm.com"
    export LOGS_API_KEY="your-api-key"  # pragma: allowlist secret

    # Run all scenarios
    python3 scripts/measure-benchmark.py

    # Run specific scenarios
    python3 scripts/measure-benchmark.py --scenarios 1,4,5

    # Output results to file
    python3 scripts/measure-benchmark.py --output-file /tmp/benchmark.json

License: Apache-2.0
"""

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from datetime import datetime, timedelta, timezone

import requests

# ── Constants ────────────────────────────────────────────────────

BYTES_PER_TOKEN = 3.79
IAM_TOKEN_URL = "https://iam.cloud.ibm.com/identity/token"
SKILLS_DIR = os.path.join(os.path.dirname(__file__), "..", ".agents", "skills")
MCP_BINARY = os.path.join(os.path.dirname(__file__), "..", "bin", "logs-mcp-server")

RUN_ID = uuid.uuid4().hex[:8]

# Time range for queries
_now = datetime.now(timezone.utc)
_start = _now - timedelta(hours=24)
START_DATE = _start.strftime("%Y-%m-%dT%H:%M:%S.000Z")
END_DATE = _now.strftime("%Y-%m-%dT%H:%M:%S.000Z")

# ── LLM conversation cost model constants ────────────────────────
# These are charged on EVERY API call (every assistant turn).
SYSTEM_PROMPT_TOKENS = 4_000      # Claude Code base system prompt
BUILTIN_TOOL_TOKENS = 3_000       # Read, Write, Edit, Bash, Glob, Grep, Agent, etc.
MCP_TOOL_TOKENS = 18_229          # Measured from tools/list (96 MCP tools)

SKILLS_OVERHEAD = SYSTEM_PROMPT_TOKENS + BUILTIN_TOOL_TOKENS          # 7,000 /turn
MCP_OVERHEAD = SYSTEM_PROMPT_TOKENS + BUILTIN_TOOL_TOKENS + MCP_TOOL_TOKENS  # 25,229 /turn

# Claude API pricing ($ per million tokens)
SONNET_INPUT = 3.0
SONNET_OUTPUT = 15.0
OPUS_INPUT = 15.0
OPUS_OUTPUT = 75.0


# ── Helpers ──────────────────────────────────────────────────────

def tok(byte_count):
    return round(byte_count / BYTES_PER_TOKEN)


def fsize(path):
    try:
        return os.path.getsize(path)
    except FileNotFoundError:
        print(f"  WARN: not found: {path}", file=sys.stderr)
        return 0


def skill_path(*parts):
    return os.path.join(SKILLS_DIR, *parts)


# ── Conversation Model ──────────────────────────────────────────

class Conversation:
    """Models a complete LLM conversation with proper cost accounting.

    Key insight: each assistant turn is an LLM API call that bills for the
    ENTIRE context (system prompt + tool definitions + all previous messages).
    So a 10-turn conversation doesn't just cost sum(responses) — it costs
    sum(context_at_turn_i) where context grows with every turn.
    """

    def __init__(self, label, approach, fixed_overhead):
        self.label = label
        self.approach = approach          # "skills" or "mcp"
        self.fixed_overhead = fixed_overhead  # tokens billed per turn
        self._history_tokens = 0          # accumulated context from messages
        self.turns = []                   # list of Turn dicts

    def turn(self, desc, output_tokens, tool_results, user_tokens=0):
        """Record one assistant turn (= one LLM API call).

        Args:
            desc: human-readable description of what the assistant does
            output_tokens: estimated tokens the assistant generates
                (reasoning text + tool call JSON). These are billed as output
                AND enter context for future turns.
            tool_results: list of (label, content_bytes, is_error, wall_ms)
                tuples. Content bytes are converted to tokens and enter context.
            user_tokens: tokens from user message preceding this turn (if any)
        """
        # Add user message to history first (if any)
        if user_tokens:
            self._history_tokens += user_tokens

        # Billed input = fixed overhead + all history so far
        billed_input = self.fixed_overhead + self._history_tokens

        # Process tool results
        results = []
        total_content_tokens = 0
        total_wall_ms = 0
        errors = 0
        for label, content_bytes, is_error, wall_ms in tool_results:
            content_tokens = tok(content_bytes)
            total_content_tokens += content_tokens
            total_wall_ms += wall_ms
            if is_error:
                errors += 1
            marker = "✗" if is_error else "✓"
            print(f"  {marker} {label:<55s} {content_tokens:>6,} tok ({content_bytes:,} B)")
            results.append({
                "label": label,
                "bytes": content_bytes,
                "tokens": content_tokens,
                "is_error": is_error,
                "wall_ms": wall_ms,
            })

        turn_data = {
            "turn": len(self.turns) + 1,
            "description": desc,
            "billed_input": billed_input,
            "output_tokens": output_tokens,
            "tool_results": results,
            "content_tokens": total_content_tokens,
            "wall_ms": total_wall_ms,
            "errors": errors,
        }
        self.turns.append(turn_data)

        # Update history: assistant output + tool results enter context
        self._history_tokens += output_tokens + total_content_tokens

    # ── Computed metrics ──

    @property
    def total_billed_input(self):
        """Sum of billed input across all turns. This is what you pay for."""
        return sum(t["billed_input"] for t in self.turns)

    @property
    def total_output_tokens(self):
        return sum(t["output_tokens"] for t in self.turns)

    @property
    def total_content_tokens(self):
        """Total tokens from tool results (skill reads + API responses)."""
        return sum(t["content_tokens"] for t in self.turns)

    @property
    def num_turns(self):
        return len(self.turns)

    @property
    def peak_context(self):
        """Largest context sent to the model (last turn is typically the peak)."""
        if not self.turns:
            return 0
        return max(t["billed_input"] + t["output_tokens"] + t["content_tokens"]
                   for t in self.turns)

    @property
    def num_errors(self):
        return sum(t["errors"] for t in self.turns)

    @property
    def total_wall_ms(self):
        return sum(t["wall_ms"] for t in self.turns)

    def cost(self, input_price, output_price):
        return (self.total_billed_input * input_price / 1_000_000
                + self.total_output_tokens * output_price / 1_000_000)

    @property
    def cost_sonnet(self):
        return self.cost(SONNET_INPUT, SONNET_OUTPUT)

    @property
    def cost_opus(self):
        return self.cost(OPUS_INPUT, OPUS_OUTPUT)

    def to_dict(self):
        return {
            "label": self.label,
            "approach": self.approach,
            "fixed_overhead_per_turn": self.fixed_overhead,
            "turns": self.turns,
            "metrics": {
                "total_billed_input": self.total_billed_input,
                "total_output_tokens": self.total_output_tokens,
                "total_content_tokens": self.total_content_tokens,
                "num_turns": self.num_turns,
                "peak_context": self.peak_context,
                "num_errors": self.num_errors,
                "total_wall_ms": self.total_wall_ms,
                "cost_sonnet_usd": round(self.cost_sonnet, 4),
                "cost_opus_usd": round(self.cost_opus, 4),
            },
        }


# ── REST API Client ──────────────────────────────────────────────

class APIClient:
    def __init__(self, service_url, api_key):
        self.service_url = service_url
        resp = requests.post(IAM_TOKEN_URL, data={
            "grant_type": "urn:ibm:params:oauth:grant-type:apikey",
            "apikey": api_key,
        }, headers={"Content-Type": "application/x-www-form-urlencoded"}, timeout=30)
        resp.raise_for_status()
        self.token = resp.json()["access_token"]

    def call(self, method, path, body=None):
        """Returns (json_data, byte_count, wall_ms)."""
        url = f"{self.service_url}{path}"
        headers = {"Authorization": f"Bearer {self.token}", "Content-Type": "application/json"}
        t0 = time.time()
        if method == "GET":
            r = requests.get(url, headers=headers, timeout=60)
        elif method == "POST":
            r = requests.post(url, headers=headers, json=body, timeout=60)
        elif method == "PUT":
            r = requests.put(url, headers=headers, json=body, timeout=60)
        elif method == "DELETE":
            r = requests.delete(url, headers=headers, timeout=60)
        else:
            raise ValueError(f"Unknown method: {method}")
        wall_ms = (time.time() - t0) * 1000
        raw_len = len(r.text.encode("utf-8")) if r.text else 0
        if r.status_code >= 400:
            print(f"    HTTP {r.status_code}: {r.text[:200]}", file=sys.stderr)
        try:
            return r.json() if r.text else {}, raw_len, wall_ms
        except Exception:
            return {}, raw_len, wall_ms

    def query(self, query_str, tier="archive"):
        """Returns (text, byte_count, wall_ms)."""
        payload = {
            "query": query_str,
            "metadata": {
                "startDate": START_DATE, "endDate": END_DATE,
                "defaultSource": "logs", "tier": tier,
                "syntax": "dataprime", "limit": 200,
            }
        }
        t0 = time.time()
        r = requests.post(
            f"{self.service_url}/v1/query",
            headers={"Authorization": f"Bearer {self.token}", "Content-Type": "application/json"},
            json=payload, timeout=120,
        )
        wall_ms = (time.time() - t0) * 1000
        raw_len = len(r.text.encode("utf-8")) if r.text else 0
        return r.text, raw_len, wall_ms


# ── MCP Client ───────────────────────────────────────────────────

class MCPClient:
    def __init__(self, binary_path, env):
        self.proc = subprocess.Popen(
            [binary_path], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env=env,
        )
        time.sleep(2)
        self.msg_id = 0
        self.tools_bytes = 0

    def _make_request(self, method, params=None):
        self.msg_id += 1
        req = {"jsonrpc": "2.0", "id": self.msg_id, "method": method}
        if params:
            req["params"] = params
        return req

    def _send_recv(self, request, label, timeout=120):
        target_id = request.get("id")
        self.proc.stdin.write((json.dumps(request) + "\n").encode())
        self.proc.stdin.flush()
        start_t = time.time()
        while True:
            if time.time() - start_t > timeout:
                print(f"  TIMEOUT: {label}")
                return None, b"", (time.time() - start_t) * 1000
            line = self.proc.stdout.readline()
            if not line:
                time.sleep(0.05)
                continue
            try:
                msg = json.loads(line.decode())
            except json.JSONDecodeError:
                continue
            if "id" not in msg:
                continue
            if msg["id"] == target_id:
                return msg, line, (time.time() - start_t) * 1000

    def initialize(self):
        req = self._make_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "benchmark", "version": "1.0"},
        })
        resp, _, _ = self._send_recv(req, "initialize", timeout=10)
        if not resp:
            raise RuntimeError("Failed to initialize MCP server")
        notif = {"jsonrpc": "2.0", "method": "notifications/initialized"}
        self.proc.stdin.write((json.dumps(notif) + "\n").encode())
        self.proc.stdin.flush()
        time.sleep(1)

        req = self._make_request("tools/list", {})
        resp, raw, _ = self._send_recv(req, "tools/list", timeout=15)
        if not resp:
            raise RuntimeError("Failed to get tools/list")
        self.tools_bytes = len(raw)
        tool_count = len(resp.get("result", {}).get("tools", []))
        print(f"  MCP server ready: {tool_count} tools, {tok(self.tools_bytes):,} tokens schema overhead")

    def call_tool(self, name, arguments, label, timeout=120):
        """Returns (resp, content_bytes, is_error, wall_ms)."""
        req = self._make_request("tools/call", {"name": name, "arguments": arguments})
        resp, raw, wall_ms = self._send_recv(req, label, timeout=timeout)
        is_error = False
        if resp:
            result = resp.get("result", {})
            if result.get("isError"):
                is_error = True
            for c in result.get("content", []):
                text = c.get("text", "")
                if "error" in text.lower()[:100] and len(text) < 500:
                    is_error = True
                    break
        else:
            is_error = True
            raw = b""
        return resp, len(raw), is_error, wall_ms

    def extract_id(self, resp, field="id"):
        if not resp:
            return None
        for c in resp.get("result", {}).get("content", []):
            text = c.get("text", "")
            try:
                parsed = json.loads(text)
                if field in parsed:
                    return parsed[field]
            except (json.JSONDecodeError, TypeError):
                import re
                m = re.search(rf'"{field}":\s*"([^"]+)"', text)
                if m:
                    return m.group(1)
        return None

    def shutdown(self):
        self.proc.stdin.close()
        self.proc.terminate()
        try:
            self.proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.proc.kill()


# ══════════════════════════════════════════════════════════════════
# SCENARIO DEFINITIONS
#
# Each scenario builds a Conversation with realistic turn grouping.
# Turn boundaries reflect how Claude Code actually batches tool calls.
#
# Output token estimates per turn type:
#   - Read files + plan:       200 (reasoning + read tool calls)
#   - API call with reasoning: 150 (brief analysis + tool call JSON)
#   - Error recovery + retry:  180 (error analysis + corrected call)
#   - Multi-tool batch:        250 (reasoning + 2-3 tool calls)
#   - Analysis/summary:        400 (detailed analysis, no tools)
#   - Final report:            600 (comprehensive summary)
# ══════════════════════════════════════════════════════════════════


def s1_skills(api):
    """Scenario 1: Incident Investigation (Skills)"""
    c = Conversation("Incident Investigation", "skills", SKILLS_OVERHEAD)

    # Turn 1: User asks → Claude reads consolidated skill + incident guide + checks TCO
    _, tco_bytes, tco_ms = api.call("GET", "/v1/tco_policies")
    c.turn("Read skill files, check TCO policies", output_tokens=200,
        user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read incident-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "incident-guide.md")), False, 1),
        ("GET /v1/tco_policies", tco_bytes, False, tco_ms),
    ])

    # Turn 2: First query attempt — wrong tier (mistake)
    _, q_bytes, q_ms = api.query(
        "source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as error_count | orderby -error_count | limit 20",
        "frequent_search")
    c.turn("Query global errors (wrong tier)", output_tokens=150, tool_results=[
        ("Query frequent_search tier (error)", q_bytes, True, q_ms),
    ])

    # Turn 3: Fix tier + run correct query, also try AND (batched)
    _, q1_bytes, q1_ms = api.query(
        "source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as error_count | orderby -error_count | limit 20",
        "archive")
    _, q2_bytes, q2_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' AND $m.severity >= ERROR | limit 10",
        "archive")
    c.turn("Fix tier, run global error rate + AND query (error)", output_tokens=250, tool_results=[
        ("Query global error rate (archive)", q1_bytes, False, q1_ms),
        ("Query with AND (wrong syntax)", q2_bytes, True, q2_ms),
    ])

    # Turn 4: Fix AND→&& + run timeline + critical
    _, q3_bytes, q3_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | limit 10",
        "archive")
    _, q4_bytes, q4_ms = api.query(
        "source logs | filter $m.severity >= WARNING | groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as errors | orderby time_bucket",
        "archive")
    _, q5_bytes, q5_ms = api.query(
        "source logs | filter $m.severity == CRITICAL | limit 50",
        "archive")
    c.turn("Fix &&, run timeline + critical queries", output_tokens=250, tool_results=[
        ("Query with && (fixed)", q3_bytes, False, q3_ms),
        ("Global error timeline", q4_bytes, False, q4_ms),
        ("Critical errors (raw)", q5_bytes, False, q5_ms),
    ])

    # Turn 5: Read more refs, start component analysis with = mistake
    _, q6_bytes, q6_ms = api.query(
        "source logs | filter $l.applicationname = 'radiant' && $m.severity >= ERROR | groupby $d.message:string aggregate count() as occurrences | orderby -occurrences | limit 20",
        "archive")
    c.turn("Read investigation-queries.md, run component query (= error)", output_tokens=200, tool_results=[
        ("Read investigation-queries.md",
            fsize(skill_path("ibm-cloud-logs", "references", "investigation-queries.md")), False, 1),
        ("Query with = (wrong syntax)", q6_bytes, True, q6_ms),
    ])

    # Turn 6: Fix ==, run component patterns + subsystems + dependencies
    _, q7_bytes, q7_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | groupby $d.message:string aggregate count() as occurrences | orderby -occurrences | limit 20",
        "archive")
    _, q8_bytes, q8_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' && $m.severity >= WARNING | groupby $l.subsystemname aggregate count() as error_count | orderby -error_count",
        "archive")
    _, q9_bytes, q9_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' && ($d.message:string.contains('connection') || $d.message:string.contains('timeout') || $d.message:string.contains('refused')) | limit 100",
        "archive")
    c.turn("Fix ==, run patterns + subsystems + dependencies", output_tokens=250, tool_results=[
        ("Component error patterns (fixed)", q7_bytes, False, q7_ms),
        ("Component subsystems", q8_bytes, False, q8_ms),
        ("Component dependencies", q9_bytes, False, q9_ms),
    ])

    # Turn 7: Read heuristic + alerting refs
    c.turn("Read heuristic details + alerting skill", output_tokens=200, tool_results=[
        ("Read heuristic-details.md",
            fsize(skill_path("ibm-cloud-logs", "references", "heuristic-details.md")), False, 1),
        ("Read alerting-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "alerting-guide.md")), False, 1),
        ("Read burn-rate-math.md",
            fsize(skill_path("ibm-cloud-logs", "references", "burn-rate-math.md")), False, 1),
    ])

    # Turn 8: CLI error then create alert via API
    alert_def = {
        "name": "radiant-error-rate", "description": "Alert on radiant errors",
        "is_active": True, "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 100, "time_window": {"logs_time_window_specific_value": "minutes_10"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 10, "notify_on": "triggered_only_unspecified"},
    }
    data, alert_bytes, alert_ms = api.call("POST", "/v1/alert_definitions", alert_def)
    alert_id = data.get("id", data.get("unique_identifier"))
    c.turn("CLI error → create alert via REST", output_tokens=180, tool_results=[
        ("CLI --from-file error", 91, True, 0),
        ("Create alert via API", alert_bytes, False, alert_ms),
    ])
    if alert_id:
        api.call("DELETE", f"/v1/alert_definitions/{alert_id}")

    # Turn 9: Final investigation report
    c.turn("Generate investigation report + recommendations", output_tokens=600, tool_results=[])

    return c


def s1_mcp(mcp):
    """Scenario 1: Incident Investigation (MCP)"""
    c = Conversation("Incident Investigation", "mcp", MCP_OVERHEAD)

    # Turn 1: User asks → Claude calls investigate_incident
    _, b, err, ms = mcp.call_tool("investigate_incident",
        {"time_range": "24h", "severity": "error"},
        "investigate_incident (global)", timeout=180)
    c.turn("Call investigate_incident", output_tokens=100,
        user_tokens=25, tool_results=[
        ("investigate_incident (global)", b, err, ms),
    ])

    # Turn 2: Claude calls suggest_alert based on findings
    _, b, err, ms = mcp.call_tool("suggest_alert",
        {"service_type": "web_service", "slo_target": 0.999,
         "use_case": "high error rate on radiant",
         "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR"},
        "suggest_alert")
    c.turn("Call suggest_alert", output_tokens=100, tool_results=[
        ("suggest_alert", b, err, ms),
    ])

    # Turn 3: Create alert (dry-run) + generate report
    _, b, err, ms = mcp.call_tool("create_alert_definition", {"definition": {
        "name": f"benchmark-s1-{RUN_ID}", "description": "Benchmark", "is_active": True,
        "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 100, "time_window": {"logs_time_window_specific_value": "minutes_10"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 10, "notify_on": "triggered_only_unspecified"},
    }, "dry_run": True}, "create_alert_definition (dry-run)")
    c.turn("Create alert + final report", output_tokens=400, tool_results=[
        ("create_alert_definition (dry-run)", b, err, ms),
    ])

    return c


def s2_skills(api):
    """Scenario 2: Cost Optimization (Skills)"""
    c = Conversation("Cost Optimization", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read consolidated skill + cost guide + check TCO policies
    _, tco_bytes, tco_ms = api.call("GET", "/v1/tco_policies")
    c.turn("Read skill files, check TCO", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read cost-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "cost-guide.md")), False, 1),
        ("GET /v1/tco_policies", tco_bytes, False, tco_ms),
    ])

    # Turn 2: Query with 'sort' mistake
    _, q1_bytes, q1_ms = api.query(
        "source logs | groupby $m.severity aggregate count() as volume | sort -volume", "archive")
    c.turn("Query volume by severity (sort error)", output_tokens=150, tool_results=[
        ("Query with sort (wrong syntax)", q1_bytes, True, q1_ms),
    ])

    # Turn 3: Fix to orderby + run more volume queries
    _, q2_bytes, q2_ms = api.query(
        "source logs | groupby $m.severity aggregate count() as volume | orderby -volume", "archive")
    _, q3_bytes, q3_ms = api.query(
        "source logs | groupby $l.applicationname aggregate count() as volume | orderby -volume | limit 20", "archive")
    _, q4_bytes, q4_ms = api.query(
        "source logs | groupby $l.applicationname, $m.severity aggregate count() as volume | orderby -volume | limit 50", "archive")
    c.turn("Fix orderby, run volume analysis queries", output_tokens=250, tool_results=[
        ("Volume by severity (fixed)", q2_bytes, False, q2_ms),
        ("Volume by application", q3_bytes, False, q3_ms),
        ("Volume by app + severity", q4_bytes, False, q4_ms),
    ])

    # Turn 4: Read TCO ref + CLI error + create policy
    _, pol_bytes, pol_ms = api.call("POST", "/v1/tco_policies", {
        "name": f"benchmark-s2-{RUN_ID}", "description": "Benchmark",
        "priority": "type_medium", "application_rule": {"name": "*", "rule_type_id": "is"},
        "subsystem_rule": {"name": "*", "rule_type_id": "is"},
        "log_rules": {"severities": ["info"]}, "enabled": True,
    })
    c.turn("Read TCO ref, CLI error, create policy via REST", output_tokens=200, tool_results=[
        ("Read tco-policies.md",
            fsize(skill_path("ibm-cloud-logs", "references", "tco-policies.md")), False, 1),
        ("CLI --from-file error", 91, True, 0),
        ("Create TCO policy", pol_bytes, False, pol_ms),
    ])

    # Turn 5: Read E2M guide + summary
    c.turn("Read E2M guide, generate cost report", output_tokens=400, tool_results=[
        ("Read e2m-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "e2m-guide.md")), False, 1),
    ])

    return c


def s2_mcp(mcp):
    """Scenario 2: Cost Optimization (MCP)"""
    c = Conversation("Cost Optimization", "mcp", MCP_OVERHEAD)

    # Turn 1: List policies + query severity distribution
    _, b1, e1, ms1 = mcp.call_tool("list_policies", {}, "list_policies")
    _, b2, e2, ms2 = mcp.call_tool("query_logs", {
        "query": "source logs | groupby $m.severity aggregate count() as volume | orderby -volume",
        "tier": "archive", "start_date": START_DATE, "end_date": END_DATE},
        "query_logs (severity)")
    c.turn("List policies + query severity", output_tokens=150, user_tokens=25, tool_results=[
        ("list_policies", b1, e1, ms1),
        ("query_logs (severity)", b2, e2, ms2),
    ])

    # Turn 2: Query by app + estimate cost
    _, b3, e3, ms3 = mcp.call_tool("query_logs", {
        "query": "source logs | groupby $l.applicationname aggregate count() as volume | orderby -volume | limit 20",
        "tier": "archive", "start_date": START_DATE, "end_date": END_DATE},
        "query_logs (app)")
    _, b4, e4, ms4 = mcp.call_tool("estimate_query_cost", {
        "query": "source logs | groupby $l.applicationname, $m.severity aggregate count() as volume | orderby -volume | limit 50",
        "tier": "archive"}, "estimate_query_cost")
    c.turn("Query by app + estimate cost", output_tokens=150, tool_results=[
        ("query_logs (app)", b3, e3, ms3),
        ("estimate_query_cost", b4, e4, ms4),
    ])

    # Turn 3: Create policy + summary
    _, b5, e5, ms5 = mcp.call_tool("create_policy", {"policy": {
        "name": f"benchmark-s2-mcp-{RUN_ID}", "description": "Benchmark",
        "priority": "type_medium", "application_rule": {"name": "*", "rule_type_id": "is"},
        "subsystem_rule": {"name": "*", "rule_type_id": "is"},
        "log_rules": {"severities": ["info"]}, "enabled": True,
    }, "dry_run": True}, "create_policy (dry-run)")
    c.turn("Create policy + cost report", output_tokens=400, tool_results=[
        ("create_policy (dry-run)", b5, e5, ms5),
    ])

    return c


def s3_skills(api):
    """Scenario 3: Monitoring Setup (Skills)"""
    c = Conversation("Monitoring Setup", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read skills + discover apps
    _, q1_bytes, q1_ms = api.query(
        "source logs | groupby $l.applicationname aggregate count() as volume, approx_count_distinct($l.subsystemname) as components | orderby -volume | limit 20",
        "archive")
    c.turn("Read query+alerting skills, discover apps", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read alerting-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "alerting-guide.md")), False, 1),
        ("Discover applications", q1_bytes, False, q1_ms),
    ])

    # Turn 2: Query with double quotes (mistake)
    _, q2_bytes, q2_ms = api.query(
        'source logs | filter $l.applicationname == "radiant" | groupby $l.subsystemname, $m.severity aggregate count() as volume | orderby -volume | limit 20',
        "archive")
    c.turn("Query app patterns (double quotes error)", output_tokens=150, tool_results=[
        ("Query with double quotes (wrong)", q2_bytes, True, q2_ms),
    ])

    # Turn 3: Fix quotes + baseline query
    _, q3_bytes, q3_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' | groupby $l.subsystemname, $m.severity aggregate count() as volume | orderby -volume | limit 20",
        "archive")
    _, q4_bytes, q4_ms = api.query(
        "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | groupby roundTime($m.timestamp, 5m) as bucket aggregate count() as errors | orderby bucket",
        "archive")
    c.turn("Fix quotes, run pattern + baseline queries", output_tokens=250, tool_results=[
        ("App patterns (single quotes)", q3_bytes, False, q3_ms),
        ("Error rate baseline", q4_bytes, False, q4_ms),
    ])

    # Turn 4: Read alerting refs + CLI error + create alert
    alert_def = {
        "name": f"benchmark-s3-{RUN_ID}", "description": "Benchmark",
        "is_active": True, "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 50, "time_window": {"logs_time_window_specific_value": "minutes_5"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 5, "notify_on": "triggered_only_unspecified"},
    }
    data, alert_bytes, alert_ms = api.call("POST", "/v1/alert_definitions", alert_def)
    alert_id = data.get("id", data.get("unique_identifier"))
    _, wh_bytes, wh_ms = api.call("GET", "/v1/outgoing_webhooks")
    c.turn("Read alerting refs, create alert, list webhooks", output_tokens=250, tool_results=[
        ("Read component-profiles.md",
            fsize(skill_path("ibm-cloud-logs", "references", "component-profiles.md")), False, 1),
        ("Read strategy-matrix.md",
            fsize(skill_path("ibm-cloud-logs", "references", "strategy-matrix.md")), False, 1),
        ("CLI --from-file error", 55, True, 0),
        ("Create alert via API", alert_bytes, False, alert_ms),
        ("GET /v1/outgoing_webhooks", wh_bytes, False, wh_ms),
    ])
    if alert_id:
        api.call("DELETE", f"/v1/alert_definitions/{alert_id}")

    # Turn 5: Read dashboard skill + CLI error + create dashboard
    dash_def = {
        "name": f"Benchmark S3 {RUN_ID}", "description": "Benchmark",
        "layout": {"sections": [{"id": {"value": str(uuid.uuid4())}, "rows": [{"id": {"value": str(uuid.uuid4())},
            "appearance": {"height": 19}, "widgets": [{"id": {"value": str(uuid.uuid4())}, "title": "Error Rate",
            "definition": {"line_chart": {"query_definitions": [{"id": str(uuid.uuid4()),
                "query": {"logs": {"aggregations": [{"count": {}}], "filters": [], "group_by": [], "group_bys": []}}}]}}}]}]}]},
        "filters": [], "annotations": [],
    }
    data, dash_bytes, dash_ms = api.call("POST", "/v1/dashboards", dash_def)
    dash_id = data.get("id")
    c.turn("Read dashboard skill, create dashboard", output_tokens=200, tool_results=[
        ("Read dashboards-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dashboards-guide.md")), False, 1),
        ("Read dashboard-schema.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dashboard-schema.md")), False, 1),
        ("CLI dashboard-create error", 88, True, 0),
        ("Create dashboard via REST", dash_bytes, False, dash_ms),
    ])
    if dash_id:
        api.call("DELETE", f"/v1/dashboards/{dash_id}")

    # Turn 6: Summary
    c.turn("Generate monitoring setup report", output_tokens=500, tool_results=[])

    return c


def s3_mcp(mcp):
    """Scenario 3: Monitoring Setup (MCP)"""
    c = Conversation("Monitoring Setup", "mcp", MCP_OVERHEAD)

    # Turn 1: Discover apps
    _, b1, e1, ms1 = mcp.call_tool("query_logs", {
        "query": "source logs | groupby $l.applicationname aggregate count() as volume | orderby -volume | limit 20",
        "tier": "archive", "start_date": START_DATE, "end_date": END_DATE},
        "query_logs (discover apps)")
    c.turn("Discover applications", output_tokens=100, user_tokens=25, tool_results=[
        ("query_logs (discover apps)", b1, e1, ms1),
    ])

    # Turn 2: Suggest alert + create
    _, b2, e2, ms2 = mcp.call_tool("suggest_alert", {
        "service_type": "web_service", "slo_target": 0.999,
        "use_case": "monitor radiant error rate",
        "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR"},
        "suggest_alert")
    _, b3, e3, ms3 = mcp.call_tool("create_alert_definition", {"definition": {
        "name": f"benchmark-s3-mcp-{RUN_ID}", "description": "Benchmark", "is_active": True,
        "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 50, "time_window": {"logs_time_window_specific_value": "minutes_5"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 5, "notify_on": "triggered_only_unspecified"},
    }, "dry_run": True}, "create_alert_definition (dry-run)")
    c.turn("Suggest + create alert", output_tokens=150, tool_results=[
        ("suggest_alert", b2, e2, ms2),
        ("create_alert_definition (dry-run)", b3, e3, ms3),
    ])

    # Turn 3: Webhooks + dashboard + summary
    _, b4, e4, ms4 = mcp.call_tool("list_outgoing_webhooks", {}, "list_outgoing_webhooks")
    _, b5, e5, ms5 = mcp.call_tool("create_dashboard", {
        "name": f"Benchmark S3 MCP {RUN_ID}", "description": "Benchmark",
        "layout": {"sections": [{"id": {"value": str(uuid.uuid4())}, "rows": [{"id": {"value": str(uuid.uuid4())},
            "appearance": {"height": 19}, "widgets": [{"id": {"value": str(uuid.uuid4())}, "title": "Error Rate",
            "definition": {"line_chart": {"query_definitions": [{"id": str(uuid.uuid4()),
                "query": {"logs": {"aggregations": [{"count": {}}], "filters": [], "group_by": [], "group_bys": []}}}]}}}]}]}]},
    }, "create_dashboard (dry-run)")
    c.turn("Webhooks + dashboard + report", output_tokens=400, tool_results=[
        ("list_outgoing_webhooks", b4, e4, ms4),
        ("create_dashboard (dry-run)", b5, e5, ms5),
    ])

    return c


def s4_skills(api):
    """Scenario 4: Normal Operations / CRUD (Skills)"""
    c = Conversation("Normal Operations (CRUD)", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read alerting skill + create alert
    alert_def = {
        "name": f"benchmark-s4-{RUN_ID}", "description": "Benchmark", "is_active": True,
        "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 100, "time_window": {"logs_time_window_specific_value": "minutes_10"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 10, "notify_on": "triggered_only_unspecified"},
    }
    data, create_bytes, create_ms = api.call("POST", "/v1/alert_definitions", alert_def)
    alert_id = data.get("id", data.get("unique_identifier"))
    c.turn("Read skill + alerting guide, create alert", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read alerting-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "alerting-guide.md")), False, 1),
        ("Read strategy-matrix.md",
            fsize(skill_path("ibm-cloud-logs", "references", "strategy-matrix.md")), False, 1),
        ("POST /v1/alert_definitions (create)", create_bytes, False, create_ms),
    ])

    # Turn 2: List + get + delete alert
    _, list_bytes, list_ms = api.call("GET", "/v1/alert_definitions")
    results = [("GET /v1/alert_definitions (list)", list_bytes, False, list_ms)]
    if alert_id:
        _, get_bytes, get_ms = api.call("GET", f"/v1/alert_definitions/{alert_id}")
        _, del_bytes, del_ms = api.call("DELETE", f"/v1/alert_definitions/{alert_id}")
        results.append(("GET /v1/alert_definitions/{id}", get_bytes, False, get_ms))
        results.append(("DELETE /v1/alert_definitions/{id}", del_bytes, False, del_ms))
    c.turn("List, get, delete alert", output_tokens=200, tool_results=results)

    # Turn 3: Read dashboard skill + create dashboard
    dash_def = {
        "name": f"Benchmark S4 {RUN_ID}", "description": "Benchmark",
        "layout": {"sections": [{"id": {"value": str(uuid.uuid4())}, "rows": [{"id": {"value": str(uuid.uuid4())},
            "appearance": {"height": 19}, "widgets": [{"id": {"value": str(uuid.uuid4())}, "title": "Errors",
            "definition": {"line_chart": {"query_definitions": [{"id": str(uuid.uuid4()),
                "query": {"logs": {"aggregations": [{"count": {}}], "filters": [], "group_by": [], "group_bys": []}}}]}}}]}]}]},
        "filters": [], "annotations": [],
    }
    data, create_bytes, create_ms = api.call("POST", "/v1/dashboards", dash_def)
    dash_id = data.get("id")
    c.turn("Read dashboard skill, create dashboard", output_tokens=200, tool_results=[
        ("Read dashboards-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dashboards-guide.md")), False, 1),
        ("Read dashboard-schema.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dashboard-schema.md")), False, 1),
        ("POST /v1/dashboards (create)", create_bytes, False, create_ms),
    ])

    # Turn 4: List + get + delete dashboard
    _, list_bytes, list_ms = api.call("GET", "/v1/dashboards")
    results = [("GET /v1/dashboards (list)", list_bytes, False, list_ms)]
    if dash_id:
        _, get_bytes, get_ms = api.call("GET", f"/v1/dashboards/{dash_id}")
        _, del_bytes, del_ms = api.call("DELETE", f"/v1/dashboards/{dash_id}")
        results.append(("GET /v1/dashboards/{id}", get_bytes, False, get_ms))
        results.append(("DELETE /v1/dashboards/{id}", del_bytes, False, del_ms))
    c.turn("List, get, delete dashboard", output_tokens=200, tool_results=results)

    # Turn 5: Read access-control + view CRUD
    data, create_bytes, create_ms = api.call("POST", "/v1/views", {
        "name": f"Benchmark S4 {RUN_ID}",
        "search_query": {"query": "applicationname:radiant AND severity:error"},
        "time_selection": {"quick_selection": {"caption": "Last 1 hour", "seconds": 3600}},
        "filters": {"filters": [{"name": "applicationname", "selected_values": {"radiant": True}}]},
    })
    view_id = data.get("id")
    _, list_bytes, list_ms = api.call("GET", "/v1/views")
    results = [
        ("Read access-control-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "access-control-guide.md")), False, 1),
        ("POST /v1/views (create)", create_bytes, False, create_ms),
        ("GET /v1/views (list)", list_bytes, False, list_ms),
    ]
    if view_id:
        _, del_bytes, del_ms = api.call("DELETE", f"/v1/views/{view_id}")
        results.append(("DELETE /v1/views/{id}", del_bytes, False, del_ms))
    c.turn("Read access-control, view CRUD", output_tokens=200, tool_results=results)

    # Turn 6: Summary
    c.turn("Summarize CRUD results", output_tokens=300, tool_results=[])

    return c


def s4_mcp(mcp):
    """Scenario 4: Normal Operations / CRUD (MCP)"""
    c = Conversation("Normal Operations (CRUD)", "mcp", MCP_OVERHEAD)

    # Turn 1: Create alert
    resp, b, err, ms = mcp.call_tool("create_alert_definition", {"definition": {
        "name": f"benchmark-s4-mcp-{RUN_ID}", "description": "Benchmark", "is_active": True,
        "severity": "error_or_unspecified", "type": "logs_threshold",
        "logs_threshold": {"condition_type": "more_than_or_unspecified",
            "logs_filter": {"simple_filter": {"lucene_query": "applicationname:radiant AND severity:error"}},
            "rules": [{"condition": {"threshold": 100, "time_window": {"logs_time_window_specific_value": "minutes_10"}}, "override": {"priority": "p2"}}]},
        "incidents_settings": {"minutes": 10, "notify_on": "triggered_only_unspecified"},
    }}, "create_alert_definition")
    alert_id = mcp.extract_id(resp)
    c.turn("Create alert", output_tokens=120, user_tokens=25, tool_results=[
        ("create_alert_definition", b, err, ms),
    ])

    # Turn 2: List + get + delete alert
    _, b1, e1, ms1 = mcp.call_tool("list_alert_definitions", {}, "list_alert_definitions")
    results = [("list_alert_definitions", b1, e1, ms1)]
    if alert_id:
        _, b2, e2, ms2 = mcp.call_tool("get_alert_definition", {"id": alert_id}, "get_alert_definition")
        _, b3, e3, ms3 = mcp.call_tool("delete_alert_definition", {"id": alert_id}, "delete_alert_definition")
        results.append(("get_alert_definition", b2, e2, ms2))
        results.append(("delete_alert_definition", b3, e3, ms3))
    c.turn("List, get, delete alert", output_tokens=150, tool_results=results)

    # Turn 3: Create dashboard
    resp, b, err, ms = mcp.call_tool("create_dashboard", {
        "name": f"Benchmark S4 MCP {RUN_ID}", "description": "Benchmark",
        "layout": {"sections": [{"id": {"value": str(uuid.uuid4())}, "rows": [{"id": {"value": str(uuid.uuid4())},
            "appearance": {"height": 19}, "widgets": [{"id": {"value": str(uuid.uuid4())}, "title": "Errors",
            "definition": {"line_chart": {"query_definitions": [{"id": str(uuid.uuid4()),
                "query": {"logs": {"aggregations": [{"count": {}}], "filters": [], "group_by": [], "group_bys": []}}}]}}}]}]}]},
    }, "create_dashboard")
    dash_id = mcp.extract_id(resp)
    c.turn("Create dashboard", output_tokens=120, tool_results=[
        ("create_dashboard", b, err, ms),
    ])

    # Turn 4: List + get + delete dashboard
    _, b1, e1, ms1 = mcp.call_tool("list_dashboards", {}, "list_dashboards")
    results = [("list_dashboards", b1, e1, ms1)]
    if dash_id:
        _, b2, e2, ms2 = mcp.call_tool("get_dashboard", {"dashboard_id": dash_id}, "get_dashboard")
        _, b3, e3, ms3 = mcp.call_tool("delete_dashboard", {"dashboard_id": dash_id}, "delete_dashboard")
        results.append(("get_dashboard", b2, e2, ms2))
        results.append(("delete_dashboard", b3, e3, ms3))
    c.turn("List, get, delete dashboard", output_tokens=150, tool_results=results)

    # Turn 5: View CRUD
    resp, b1, e1, ms1 = mcp.call_tool("create_view", {"view": {
        "name": f"Benchmark S4 MCP {RUN_ID}",
        "search_query": {"query": "applicationname:radiant AND severity:error"},
        "time_selection": {"quick_selection": {"caption": "Last 1 hour", "seconds": 3600}},
        "filters": {"filters": [{"name": "applicationname", "selected_values": {"radiant": True}}]},
    }}, "create_view")
    view_id = mcp.extract_id(resp)
    _, b2, e2, ms2 = mcp.call_tool("list_views", {}, "list_views")
    results = [("create_view", b1, e1, ms1), ("list_views", b2, e2, ms2)]
    if view_id:
        _, b3, e3, ms3 = mcp.call_tool("delete_view", {"id": view_id}, "delete_view")
        results.append(("delete_view", b3, e3, ms3))
    c.turn("View CRUD", output_tokens=150, tool_results=results)

    # Turn 6: Summary
    c.turn("Summarize CRUD results", output_tokens=300, tool_results=[])

    return c


def s5_skills(api):
    """Scenario 5: Query Authoring & Validation (Skills)"""
    c = Conversation("Query Authoring & Validation", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read all query reference materials
    c.turn("Read query skill + all references", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read dataprime-commands.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dataprime-commands.md")), False, 1),
        ("Read query-templates.md",
            fsize(skill_path("ibm-cloud-logs", "references", "query-templates.md")), False, 1),
        ("Read dataprime-functions.md",
            fsize(skill_path("ibm-cloud-logs", "references", "dataprime-functions.md")), False, 1),
    ])

    # Turn 2: Generate query with explanation
    c.turn("Write query + explain syntax", output_tokens=400, tool_results=[])

    return c


def s5_mcp(mcp):
    """Scenario 5: Query Authoring & Validation (MCP)"""
    c = Conversation("Query Authoring & Validation", "mcp", MCP_OVERHEAD)

    # Turn 1: Build + validate queries
    _, b1, e1, ms1 = mcp.call_tool("build_query", {
        "intent": "find error logs from radiant service",
        "filters": {"application": "radiant", "severity": "error"}}, "build_query")
    _, b2, e2, ms2 = mcp.call_tool("validate_query", {
        "query": "source logs | filter $l.applicationname == 'radiant' AND $m.severity >= ERROR | limit 10"},
        "validate_query (AND mistake)")
    _, b3, e3, ms3 = mcp.call_tool("validate_query", {
        "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | limit 10"},
        "validate_query (correct)")
    c.turn("Build + validate queries", output_tokens=150, user_tokens=25, tool_results=[
        ("build_query", b1, e1, ms1),
        ("validate_query (AND mistake)", b2, e2, ms2),
        ("validate_query (correct)", b3, e3, ms3),
    ])

    # Turn 2: Explain + get references
    _, b4, e4, ms4 = mcp.call_tool("explain_query", {
        "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | groupby $l.subsystemname aggregate count() as errors | orderby -errors | limit 10"},
        "explain_query")
    _, b5, e5, ms5 = mcp.call_tool("get_dataprime_reference", {}, "get_dataprime_reference")
    _, b6, e6, ms6 = mcp.call_tool("get_query_templates", {}, "get_query_templates")
    _, b7, e7, ms7 = mcp.call_tool("estimate_query_cost", {
        "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | limit 100",
        "tier": "archive"}, "estimate_query_cost")
    c.turn("Explain + references + cost estimate", output_tokens=300, tool_results=[
        ("explain_query", b4, e4, ms4),
        ("get_dataprime_reference", b5, e5, ms5),
        ("get_query_templates", b6, e6, ms6),
        ("estimate_query_cost", b7, e7, ms7),
    ])

    return c


def s6_skills(api):
    """Scenario 6: Ingestion Pipeline (Skills)"""
    c = Conversation("Ingestion Pipeline", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read skill files + list existing config
    _, rg_bytes, rg_ms = api.call("GET", "/v1/rule_groups")
    _, en_bytes, en_ms = api.call("GET", "/v1/enrichments")
    c.turn("Read skill + ingestion guide + list config", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read ingestion-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "ingestion-guide.md")), False, 1),
        ("Read parsing-rules.md",
            fsize(skill_path("ibm-cloud-logs", "references", "parsing-rules.md")), False, 1),
        ("Read enrichment-types.md",
            fsize(skill_path("ibm-cloud-logs", "references", "enrichment-types.md")), False, 1),
        ("Read log-format.md",
            fsize(skill_path("ibm-cloud-logs", "references", "log-format.md")), False, 1),
        ("GET /v1/rule_groups (list)", rg_bytes, False, rg_ms),
        ("GET /v1/enrichments (list)", en_bytes, False, en_ms),
    ])

    # Turn 2: Explain pipeline + recommendations
    c.turn("Explain pipeline, recommend config", output_tokens=400, tool_results=[])

    return c


def s6_mcp(mcp):
    """Scenario 6: Ingestion Pipeline (MCP)"""
    c = Conversation("Ingestion Pipeline", "mcp", MCP_OVERHEAD)

    # Turn 1: List rule groups + enrichments + discover fields
    _, b1, e1, ms1 = mcp.call_tool("list_rule_groups", {}, "list_rule_groups")
    _, b2, e2, ms2 = mcp.call_tool("list_enrichments", {}, "list_enrichments")
    _, b3, e3, ms3 = mcp.call_tool("discover_log_fields", {"application": "radiant"}, "discover_log_fields")
    c.turn("List configs + discover fields", output_tokens=150, user_tokens=25, tool_results=[
        ("list_rule_groups", b1, e1, ms1),
        ("list_enrichments", b2, e2, ms2),
        ("discover_log_fields", b3, e3, ms3),
    ])

    # Turn 2: Recommendations
    c.turn("Recommend pipeline config", output_tokens=400, tool_results=[])

    return c


def s7_skills(api):
    """Scenario 7: Data Governance (Skills)"""
    c = Conversation("Data Governance", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read skills + list access rules + webhooks
    _, dar_bytes, dar_ms = api.call("GET", "/v1/data_access_rules")
    _, wh_bytes, wh_ms = api.call("GET", "/v1/outgoing_webhooks")
    c.turn("Read skill + access-control guide, list rules + webhooks", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read access-control-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "access-control-guide.md")), False, 1),
        ("Read access-rules.md",
            fsize(skill_path("ibm-cloud-logs", "references", "access-rules.md")), False, 1),
        ("GET /v1/data_access_rules (list)", dar_bytes, False, dar_ms),
        ("GET /v1/outgoing_webhooks (list)", wh_bytes, False, wh_ms),
    ])

    # Turn 2: Create webhook + get + delete
    data, create_bytes, create_ms = api.call("POST", "/v1/outgoing_webhooks", {
        "type": "generic", "name": f"benchmark-s7-{RUN_ID}", "url": "https://example.com/webhook"})
    wh_id = data.get("id")
    results = [("POST /v1/outgoing_webhooks (create)", create_bytes, False, create_ms)]
    if wh_id:
        _, get_bytes, get_ms = api.call("GET", f"/v1/outgoing_webhooks/{wh_id}")
        _, del_bytes, del_ms = api.call("DELETE", f"/v1/outgoing_webhooks/{wh_id}")
        results.append(("GET /v1/outgoing_webhooks/{id}", get_bytes, False, get_ms))
        results.append(("DELETE /v1/outgoing_webhooks/{id}", del_bytes, False, del_ms))
    c.turn("Webhook CRUD", output_tokens=200, tool_results=results)

    # Turn 3: Summary
    c.turn("Governance report", output_tokens=300, tool_results=[])

    return c


def s7_mcp(mcp):
    """Scenario 7: Data Governance (MCP)"""
    c = Conversation("Data Governance", "mcp", MCP_OVERHEAD)

    # Turn 1: List access rules + webhooks
    _, b1, e1, ms1 = mcp.call_tool("list_data_access_rules", {}, "list_data_access_rules")
    _, b2, e2, ms2 = mcp.call_tool("list_outgoing_webhooks", {}, "list_outgoing_webhooks")
    c.turn("List access rules + webhooks", output_tokens=100, user_tokens=25, tool_results=[
        ("list_data_access_rules", b1, e1, ms1),
        ("list_outgoing_webhooks", b2, e2, ms2),
    ])

    # Turn 2: Create + get + delete webhook
    resp, b3, e3, ms3 = mcp.call_tool("create_outgoing_webhook", {"webhook": {
        "type": "generic", "name": f"benchmark-s7-mcp-{RUN_ID}", "url": "https://example.com/webhook"}},
        "create_outgoing_webhook")
    wh_id = mcp.extract_id(resp)
    results = [("create_outgoing_webhook", b3, e3, ms3)]
    if wh_id:
        _, b4, e4, ms4 = mcp.call_tool("get_outgoing_webhook", {"id": wh_id}, "get_outgoing_webhook")
        _, b5, e5, ms5 = mcp.call_tool("delete_outgoing_webhook", {"id": wh_id}, "delete_outgoing_webhook")
        results.append(("get_outgoing_webhook", b4, e4, ms4))
        results.append(("delete_outgoing_webhook", b5, e5, ms5))
    c.turn("Webhook CRUD", output_tokens=150, tool_results=results)

    # Turn 3: Summary
    c.turn("Governance report", output_tokens=300, tool_results=[])

    return c


def s8_skills(api):
    """Scenario 8: E2M & Streaming (Skills)"""
    c = Conversation("E2M & Streaming", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read skills + list E2M + streams + targets
    _, e2m_bytes, e2m_ms = api.call("GET", "/v1/e2m")
    _, str_bytes, str_ms = api.call("GET", "/v1/streams")
    _, est_bytes, est_ms = api.call("GET", "/v1/event_stream_targets")
    c.turn("Read skill + cost guide + list E2M/streams", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read cost-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "cost-guide.md")), False, 1),
        ("Read e2m-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "e2m-guide.md")), False, 1),
        ("GET /v1/e2m (list)", e2m_bytes, False, e2m_ms),
        ("GET /v1/streams (list)", str_bytes, False, str_ms),
        ("GET /v1/event_stream_targets (list)", est_bytes, False, est_ms),
    ])

    # Turn 2: Recommendations
    c.turn("E2M + streaming recommendations", output_tokens=400, tool_results=[])

    return c


def s8_mcp(mcp):
    """Scenario 8: E2M & Streaming (MCP)"""
    c = Conversation("E2M & Streaming", "mcp", MCP_OVERHEAD)

    # Turn 1: List E2M + streams + targets
    _, b1, e1, ms1 = mcp.call_tool("list_e2m", {}, "list_e2m")
    _, b2, e2, ms2 = mcp.call_tool("list_streams", {}, "list_streams")
    _, b3, e3, ms3 = mcp.call_tool("get_event_stream_targets", {}, "get_event_stream_targets")
    c.turn("List E2M + streams + targets", output_tokens=100, user_tokens=25, tool_results=[
        ("list_e2m", b1, e1, ms1),
        ("list_streams", b2, e2, ms2),
        ("get_event_stream_targets", b3, e3, ms3),
    ])

    # Turn 2: Recommendations
    c.turn("E2M + streaming recommendations", output_tokens=400, tool_results=[])

    return c


def s9_skills(api):
    """Scenario 9: API Discovery & Meta (Skills)"""
    c = Conversation("API Discovery & Meta", "skills", SKILLS_OVERHEAD)

    # Turn 1: Read API reference skill
    c.turn("Read skill + api-guide + endpoints", output_tokens=200, user_tokens=25, tool_results=[
        ("Read ibm-cloud-logs/SKILL.md",
            fsize(skill_path("ibm-cloud-logs", "SKILL.md")), False, 1),
        ("Read api-guide.md",
            fsize(skill_path("ibm-cloud-logs", "references", "api-guide.md")), False, 1),
        ("Read endpoints.md",
            fsize(skill_path("ibm-cloud-logs", "references", "endpoints.md")), False, 1),
    ])

    # Turn 2: Answer with explanation
    c.turn("Explain API structure", output_tokens=300, tool_results=[])

    return c


def s9_mcp(mcp):
    """Scenario 9: API Discovery & Meta (MCP)"""
    c = Conversation("API Discovery & Meta", "mcp", MCP_OVERHEAD)

    # Turn 1: List categories + search
    _, b1, e1, ms1 = mcp.call_tool("list_tool_categories", {}, "list_tool_categories")
    _, b2, e2, ms2 = mcp.call_tool("search_tools", {"query": "alert"}, "search_tools")
    c.turn("List categories + search tools", output_tokens=100, user_tokens=25, tool_results=[
        ("list_tool_categories", b1, e1, ms1),
        ("search_tools", b2, e2, ms2),
    ])

    # Turn 2: Session context + health + explanation
    _, b3, e3, ms3 = mcp.call_tool("session_context", {}, "session_context")
    _, b4, e4, ms4 = mcp.call_tool("health_check", {}, "health_check", timeout=30)
    c.turn("Session context + health + explain API", output_tokens=300, tool_results=[
        ("session_context", b3, e3, ms3),
        ("health_check", b4, e4, ms4),
    ])

    return c


# ══════════════════════════════════════════════════════════════════
# MAIN
# ══════════════════════════════════════════════════════════════════

SCENARIO_MAP = {
    1: ("Incident Investigation", s1_skills, s1_mcp),
    2: ("Cost Optimization", s2_skills, s2_mcp),
    3: ("Monitoring Setup", s3_skills, s3_mcp),
    4: ("Normal Operations (CRUD)", s4_skills, s4_mcp),
    5: ("Query Authoring & Validation", s5_skills, s5_mcp),
    6: ("Ingestion Pipeline", s6_skills, s6_mcp),
    7: ("Data Governance", s7_skills, s7_mcp),
    8: ("E2M & Streaming", s8_skills, s8_mcp),
    9: ("API Discovery & Meta", s9_skills, s9_mcp),
}


def print_comparison(scenarios):
    """Print comprehensive comparison table."""
    # Header
    hdr = f"  {'Scenario':<35s} {'Approach':>8s} {'Input':>10s} {'Output':>8s} {'Turns':>6s} {'Peak':>8s} {'Err':>4s} {'Sonnet':>8s} {'Opus':>9s}"
    sep = f"  {'─'*35} {'─'*8} {'─'*10} {'─'*8} {'─'*6} {'─'*8} {'─'*4} {'─'*8} {'─'*9}"
    print(hdr)
    print(sep)

    grand = {"skills": {"input": 0, "output": 0, "turns": 0, "errors": 0, "sonnet": 0, "opus": 0},
             "mcp":    {"input": 0, "output": 0, "turns": 0, "errors": 0, "sonnet": 0, "opus": 0}}

    for num, (label, skills_conv, mcp_conv) in sorted(scenarios.items()):
        for conv in (skills_conv, mcp_conv):
            tag = "Skills" if conv.approach == "skills" else "MCP"
            name = f"S{num}: {label}" if conv.approach == "skills" else ""
            print(f"  {name:<35s} {tag:>8s} {conv.total_billed_input:>10,} {conv.total_output_tokens:>8,} {conv.num_turns:>6} {conv.peak_context:>8,} {conv.num_errors:>4} ${conv.cost_sonnet:>6.3f} ${conv.cost_opus:>7.3f}")
            grand[conv.approach]["input"] += conv.total_billed_input
            grand[conv.approach]["output"] += conv.total_output_tokens
            grand[conv.approach]["turns"] += conv.num_turns
            grand[conv.approach]["errors"] += conv.num_errors
            grand[conv.approach]["sonnet"] += conv.cost_sonnet
            grand[conv.approach]["opus"] += conv.cost_opus

        # Winner line
        s_cost = skills_conv.cost_sonnet
        m_cost = mcp_conv.cost_sonnet
        winner = "Skills" if s_cost < m_cost else "MCP"
        delta = abs(s_cost - m_cost) / max(s_cost, m_cost) * 100
        print(f"  {'':<35s} {'►'+winner:>8s} {'':>10s} {'':>8s} {'':>6s} {'':>8s} {'':>4s} {f'{delta:.0f}% less':>8s}")
        print()

    print(sep)
    for approach in ("skills", "mcp"):
        g = grand[approach]
        tag = "Skills" if approach == "skills" else "MCP"
        print(f"  {'TOTAL' if approach == 'skills' else '':<35s} {tag:>8s} {g['input']:>10,} {g['output']:>8,} {g['turns']:>6} {'':>8s} {g['errors']:>4} ${g['sonnet']:>6.3f} ${g['opus']:>7.3f}")

    # Overall winner
    s_total = grand["skills"]["sonnet"]
    m_total = grand["mcp"]["sonnet"]
    winner = "Skills" if s_total < m_total else "MCP"
    delta = abs(s_total - m_total) / max(s_total, m_total) * 100
    print()
    print(f"  Overall winner: {winner} ({delta:.0f}% less at Sonnet pricing)")
    print()

    # Assumptions box
    print("  Assumptions:")
    print(f"    System prompt: {SYSTEM_PROMPT_TOKENS:,} tokens/turn")
    print(f"    Built-in tools: {BUILTIN_TOOL_TOKENS:,} tokens/turn")
    print(f"    MCP tool schemas: {MCP_TOOL_TOKENS:,} tokens/turn (96 tools, measured)")
    print(f"    Skills overhead: {SKILLS_OVERHEAD:,} tokens/turn | MCP overhead: {MCP_OVERHEAD:,} tokens/turn")
    print(f"    Sonnet: ${SONNET_INPUT}/M input, ${SONNET_OUTPUT}/M output")
    print(f"    Opus: ${OPUS_INPUT}/M input, ${OPUS_OUTPUT}/M output")
    print(f"    Token ratio: {BYTES_PER_TOKEN} bytes/token (calibrated)")
    print()


def main():
    parser = argparse.ArgumentParser(
        description="Comprehensive benchmark: Skills vs MCP full-conversation cost model")
    parser.add_argument("--scenarios", default="1,2,3,4,5,6,7,8,9",
        help="Comma-separated scenario numbers to run (default: all)")
    parser.add_argument("--output-file", help="Write JSON results to file")
    args = parser.parse_args()

    selected = [int(s.strip()) for s in args.scenarios.split(",")]

    for var in ("LOGS_SERVICE_URL", "LOGS_API_KEY"):
        if not os.environ.get(var):
            print(f"Error: {var} must be set", file=sys.stderr)
            sys.exit(1)

    # Auth
    print("Authenticating...")
    api = APIClient(os.environ["LOGS_SERVICE_URL"], os.environ["LOGS_API_KEY"])
    print("  Auth OK")
    print()

    # ── Skills measurement ──
    print("═══════════════════════════════════════════════════════════════")
    print("  SKILLS + CLI  (overhead: {:,} tokens/turn)".format(SKILLS_OVERHEAD))
    print("═══════════════════════════════════════════════════════════════")
    print()

    scenarios = {}
    for num in selected:
        label, skills_fn, _ = SCENARIO_MAP[num]
        print(f"── S{num}: {label} (Skills) ──")
        skills_conv = skills_fn(api)
        print(f"  ▸ {skills_conv.num_turns} turns, {skills_conv.total_billed_input:,} billed input, "
              f"{skills_conv.total_output_tokens:,} output, ${skills_conv.cost_sonnet:.3f} Sonnet")
        print()
        scenarios[num] = (label, skills_conv, None)

    # ── MCP measurement ──
    print("═══════════════════════════════════════════════════════════════")
    print("  MCP  (overhead: {:,} tokens/turn)".format(MCP_OVERHEAD))
    print("═══════════════════════════════════════════════════════════════")
    print()

    print("Starting MCP server...")
    mcp = MCPClient(MCP_BINARY, {**os.environ, "ENVIRONMENT": "production", "LOGS_HEALTH_PORT": "0"})
    mcp.initialize()
    print()

    for num in selected:
        label, _, mcp_fn = SCENARIO_MAP[num]
        print(f"── S{num}: {label} (MCP) ──")
        mcp_conv = mcp_fn(mcp)
        print(f"  ▸ {mcp_conv.num_turns} turns, {mcp_conv.total_billed_input:,} billed input, "
              f"{mcp_conv.total_output_tokens:,} output, ${mcp_conv.cost_sonnet:.3f} Sonnet")
        print()
        label, skills_conv, _ = scenarios[num]
        scenarios[num] = (label, skills_conv, mcp_conv)

    mcp.shutdown()

    # ── Results ──
    print("═══════════════════════════════════════════════════════════════")
    print("  RESULTS — Full Conversation Cost Model")
    print("═══════════════════════════════════════════════════════════════")
    print()

    print_comparison(scenarios)

    # ── JSON output ──
    output = {
        "run_id": RUN_ID,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "model": {
            "system_prompt_tokens": SYSTEM_PROMPT_TOKENS,
            "builtin_tool_tokens": BUILTIN_TOOL_TOKENS,
            "mcp_tool_tokens": MCP_TOOL_TOKENS,
            "skills_overhead_per_turn": SKILLS_OVERHEAD,
            "mcp_overhead_per_turn": MCP_OVERHEAD,
            "bytes_per_token": BYTES_PER_TOKEN,
            "pricing": {
                "sonnet": {"input_per_m": SONNET_INPUT, "output_per_m": SONNET_OUTPUT},
                "opus": {"input_per_m": OPUS_INPUT, "output_per_m": OPUS_OUTPUT},
            },
        },
        "scenarios": {
            f"s{num}": {
                "label": label,
                "skills": skills_conv.to_dict(),
                "mcp": mcp_conv.to_dict(),
                "winner_sonnet": "skills" if skills_conv.cost_sonnet < mcp_conv.cost_sonnet else "mcp",
                "winner_opus": "skills" if skills_conv.cost_opus < mcp_conv.cost_opus else "mcp",
            }
            for num, (label, skills_conv, mcp_conv) in sorted(scenarios.items())
        },
        "totals": {
            "skills": {
                "billed_input": sum(s.total_billed_input for _, s, _ in scenarios.values()),
                "output_tokens": sum(s.total_output_tokens for _, s, _ in scenarios.values()),
                "turns": sum(s.num_turns for _, s, _ in scenarios.values()),
                "errors": sum(s.num_errors for _, s, _ in scenarios.values()),
                "cost_sonnet": round(sum(s.cost_sonnet for _, s, _ in scenarios.values()), 4),
                "cost_opus": round(sum(s.cost_opus for _, s, _ in scenarios.values()), 4),
            },
            "mcp": {
                "billed_input": sum(m.total_billed_input for _, _, m in scenarios.values()),
                "output_tokens": sum(m.total_output_tokens for _, _, m in scenarios.values()),
                "turns": sum(m.num_turns for _, _, m in scenarios.values()),
                "errors": sum(m.num_errors for _, _, m in scenarios.values()),
                "cost_sonnet": round(sum(m.cost_sonnet for _, _, m in scenarios.values()), 4),
                "cost_opus": round(sum(m.cost_opus for _, _, m in scenarios.values()), 4),
            },
        },
    }

    out_file = args.output_file or "/tmp/iteration-tax/benchmark-measured.json"
    os.makedirs(os.path.dirname(out_file), exist_ok=True)
    with open(out_file, "w") as f:
        json.dump(output, f, indent=2)
        f.write("\n")

    print(f"✓ Results: {out_file}")


if __name__ == "__main__":
    main()
