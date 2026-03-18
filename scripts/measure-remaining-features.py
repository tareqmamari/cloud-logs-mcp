#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = ["requests"]
# ///
"""
Measure token costs for ALL remaining features not covered in Scenarios 1-4.
Covers: query authoring, ingestion pipeline, data governance, E2M & streaming,
and API discovery/meta tools.

Requires: LOGS_SERVICE_URL, LOGS_API_KEY environment variables.
"""

import json
import os
import subprocess
import sys
import time
import uuid

import requests

# ── Config ────────────────────────────────────────────────────────

BYTES_PER_TOKEN = 3.79
IAM_TOKEN_URL = "https://iam.cloud.ibm.com/identity/token"
OUT_DIR = "/tmp/iteration-tax/remaining"
SKILLS_DIR = os.path.join(os.path.dirname(__file__), "..", ".agents", "skills")
MCP_BINARY = os.path.join(os.path.dirname(__file__), "..", "bin", "logs-mcp-server")

for var in ("LOGS_SERVICE_URL", "LOGS_API_KEY"):
    if not os.environ.get(var):
        print(f"Error: {var} must be set", file=sys.stderr)
        sys.exit(1)

SERVICE_URL = os.environ["LOGS_SERVICE_URL"]
API_KEY = os.environ["LOGS_API_KEY"]

os.makedirs(f"{OUT_DIR}/skills", exist_ok=True)
os.makedirs(f"{OUT_DIR}/mcp", exist_ok=True)

run_id = uuid.uuid4().hex[:8]

# ── Helpers ───────────────────────────────────────────────────────

def tokens(byte_count):
    return round(byte_count / BYTES_PER_TOKEN)


def file_size(path):
    try:
        return os.path.getsize(path)
    except FileNotFoundError:
        print(f"  WARN: file not found: {path}", file=sys.stderr)
        return 0


def log_entry(label, byte_count, category):
    t = tokens(byte_count)
    marker = "✗" if category == "error_retry" else "✓"
    print(f"  {marker} {label:<60s} {t:>6,} tokens ({byte_count:,} bytes)")
    return {"label": label, "bytes": byte_count, "tokens": t, "category": category}


# ── IAM Auth ──────────────────────────────────────────────────────

print("Authenticating...")
resp = requests.post(IAM_TOKEN_URL, data={
    "grant_type": "urn:ibm:params:oauth:grant-type:apikey",
    "apikey": API_KEY,
}, headers={"Content-Type": "application/x-www-form-urlencoded"}, timeout=30)
resp.raise_for_status()
TOKEN = resp.json()["access_token"]
print(f"  Auth OK ({tokens(len(resp.content)):,} tokens)")
print()


def api_call(method, path, body=None):
    url = f"{SERVICE_URL}{path}"
    headers = {"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"}
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
    raw_len = len(r.text.encode("utf-8")) if r.text else 0
    if r.status_code >= 400:
        print(f"    API {method} {path} → HTTP {r.status_code}: {r.text[:200]}", file=sys.stderr)
    try:
        return r.json() if r.text else {}, raw_len
    except Exception:
        return {}, raw_len


# ── Scenario ledgers ──────────────────────────────────────────────

scenarios = {}


def start_scenario(name):
    scenarios[name] = {"skills": [], "mcp": []}
    return scenarios[name]


def save_file(scenario, side, name, data):
    path = f"{OUT_DIR}/{side}/{scenario}-{name}.json"
    with open(path, "w") as f:
        json.dump(data, f, indent=2)


# ══════════════════════════════════════════════════════════════════
# PART 1: SKILLS MEASUREMENT (all remaining scenarios)
# ══════════════════════════════════════════════════════════════════

print("═══════════════════════════════════════════════════════════════")
print("  SKILLS + CLI: Remaining Features")
print("═══════════════════════════════════════════════════════════════")
print()

# ── S5: Query Authoring & Validation ─────────────────────────────

print("── S5: Query Authoring & Validation (Skills) ──")
print()
s5 = start_scenario("s5")

s5["skills"].append(log_entry("Read ibm-cloud-logs/SKILL.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")), "skill_read"))
s5["skills"].append(log_entry("Read query-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "query-guide.md")), "skill_read"))
s5["skills"].append(log_entry("Read dataprime-commands.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "dataprime-commands.md")), "skill_read"))
s5["skills"].append(log_entry("Read query-templates.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "query-templates.md")), "skill_read"))
s5["skills"].append(log_entry("Read dataprime-functions.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "dataprime-functions.md")), "skill_read"))
print()

# ── S6: Ingestion Pipeline ───────────────────────────────────────

print("── S6: Ingestion Pipeline (Skills) ──")
print()
s6 = start_scenario("s6")

s6["skills"].append(log_entry("Read ibm-cloud-logs/SKILL.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")), "skill_read"))
s6["skills"].append(log_entry("Read ingestion-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "ingestion-guide.md")), "skill_read"))
s6["skills"].append(log_entry("Read parsing-rules.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "parsing-rules.md")), "skill_read"))
s6["skills"].append(log_entry("Read enrichment-types.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "enrichment-types.md")), "skill_read"))
s6["skills"].append(log_entry("Read log-format.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "log-format.md")), "skill_read"))

# Rule groups
data, nbytes = api_call("GET", "/v1/rule_groups")
save_file("s6", "skills", "list-rule-groups", data)
rg_count = len(data) if isinstance(data, list) else len(data.get("rule_groups", []))
s6["skills"].append(log_entry(f"GET /v1/rule_groups (list, {rg_count})", nbytes, "api_call"))

# Enrichments
data, nbytes = api_call("GET", "/v1/enrichments")
save_file("s6", "skills", "list-enrichments", data)
s6["skills"].append(log_entry("GET /v1/enrichments (list)", nbytes, "api_call"))

print()

# ── S7: Data Governance ──────────────────────────────────────────

print("── S7: Data Governance (Skills) ──")
print()
s7 = start_scenario("s7")

s7["skills"].append(log_entry("Read ibm-cloud-logs/SKILL.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")), "skill_read"))
s7["skills"].append(log_entry("Read access-control-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "access-control-guide.md")), "skill_read"))
s7["skills"].append(log_entry("Read access-rules.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "access-rules.md")), "skill_read"))

# Data access rules
data, nbytes = api_call("GET", "/v1/data_access_rules")
save_file("s7", "skills", "list-access-rules", data)
s7["skills"].append(log_entry("GET /v1/data_access_rules (list)", nbytes, "api_call"))

# Outgoing webhooks (full CRUD — S3 only did list)
data, nbytes = api_call("GET", "/v1/outgoing_webhooks")
save_file("s7", "skills", "list-webhooks", data)
wh_count = len(data) if isinstance(data, list) else len(data.get("outgoing_webhooks", data.get("webhooks", [])))
s7["skills"].append(log_entry(f"GET /v1/outgoing_webhooks (list, {wh_count})", nbytes, "api_call"))

# Create webhook
wh_def = {
    "type": "generic",
    "name": f"benchmark-webhook-{run_id}",
    "url": "https://example.com/webhook",
}
data, nbytes = api_call("POST", "/v1/outgoing_webhooks", wh_def)
save_file("s7", "skills", "create-webhook", data)
s7["skills"].append(log_entry("POST /v1/outgoing_webhooks (create)", nbytes, "api_call"))
wh_id = data.get("id")

if wh_id:
    data, nbytes = api_call("GET", f"/v1/outgoing_webhooks/{wh_id}")
    s7["skills"].append(log_entry("GET /v1/outgoing_webhooks/{id} (get)", nbytes, "api_call"))
    _, nbytes = api_call("DELETE", f"/v1/outgoing_webhooks/{wh_id}")
    s7["skills"].append(log_entry("DELETE /v1/outgoing_webhooks/{id}", nbytes, "api_call"))

print()

# ── S8: E2M & Streaming ─────────────────────────────────────────

print("── S8: E2M & Streaming (Skills) ──")
print()
s8 = start_scenario("s8")

s8["skills"].append(log_entry("Read ibm-cloud-logs/SKILL.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")), "skill_read"))
s8["skills"].append(log_entry("Read cost-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "cost-guide.md")), "skill_read"))
s8["skills"].append(log_entry("Read e2m-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "e2m-guide.md")), "skill_read"))

# E2M
data, nbytes = api_call("GET", "/v1/e2m")
save_file("s8", "skills", "list-e2m", data)
s8["skills"].append(log_entry("GET /v1/e2m (list)", nbytes, "api_call"))

# Streams
data, nbytes = api_call("GET", "/v1/streams")
save_file("s8", "skills", "list-streams", data)
s8["skills"].append(log_entry("GET /v1/streams (list)", nbytes, "api_call"))

# Event stream targets
data, nbytes = api_call("GET", "/v1/event_stream_targets")
save_file("s8", "skills", "list-event-targets", data)
s8["skills"].append(log_entry("GET /v1/event_stream_targets (list)", nbytes, "api_call"))

print()

# ── S9: API Discovery & Meta ────────────────────────────────────

print("── S9: API Discovery & Meta (Skills) ──")
print()
s9 = start_scenario("s9")

s9["skills"].append(log_entry("Read ibm-cloud-logs/SKILL.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")), "skill_read"))
s9["skills"].append(log_entry("Read api-guide.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "api-guide.md")), "skill_read"))
s9["skills"].append(log_entry("Read endpoints.md",
    file_size(os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "endpoints.md")), "skill_read"))

print()


# ══════════════════════════════════════════════════════════════════
# PART 2: MCP MEASUREMENT
# ══════════════════════════════════════════════════════════════════

print("═══════════════════════════════════════════════════════════════")
print("  MCP: Remaining Features")
print("═══════════════════════════════════════════════════════════════")
print()

msg_id = 0


def make_request(method, params=None):
    global msg_id
    msg_id += 1
    req = {"jsonrpc": "2.0", "id": msg_id, "method": method}
    if params:
        req["params"] = params
    return req


def send_recv(proc, request, label, timeout=120):
    target_id = request.get("id")
    req_line = json.dumps(request) + "\n"
    proc.stdin.write(req_line.encode())
    proc.stdin.flush()
    start_t = time.time()
    while True:
        if time.time() - start_t > timeout:
            print(f"  TIMEOUT: {label} (id={target_id})")
            return None, b""
        line = proc.stdout.readline()
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
            return msg, line
        continue


def call_tool(proc, name, arguments, scenario_name, label, timeout=120):
    req = make_request("tools/call", {"name": name, "arguments": arguments})
    resp, raw = send_recv(proc, req, label, timeout=timeout)
    category = "tool_response"
    if resp:
        result = resp.get("result", {})
        if result.get("isError"):
            category = "error_retry"
        content = result.get("content", [])
        for c in content:
            text = c.get("text", "")
            if "error" in text.lower()[:100] and len(text) < 500:
                category = "error_retry"
                break
        fname = f"{OUT_DIR}/mcp/{scenario_name}-{name}-{msg_id}.json"
        with open(fname, "w") as f:
            json.dump(resp, f, indent=2)
    else:
        category = "error_retry"
        raw = b""
    entry = log_entry(label, len(raw), category)
    scenarios[scenario_name]["mcp"].append(entry)
    return resp, raw


def extract_id(resp, field="id"):
    """Extract resource ID from MCP tool response."""
    if not resp:
        return None
    content = resp.get("result", {}).get("content", [])
    for c in content:
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


# Start MCP server
print("Starting MCP server binary...")
env = {**os.environ, "ENVIRONMENT": "production", "LOGS_HEALTH_PORT": "0"}
proc = subprocess.Popen(
    [MCP_BINARY], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
    stderr=subprocess.PIPE, env=env,
)
time.sleep(2)

# Initialize
init_req = make_request("initialize", {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {"name": "benchmark-remaining", "version": "1.0"},
})
init_resp, _ = send_recv(proc, init_req, "initialize", timeout=10)
if not init_resp:
    print("  FAILED to initialize MCP server!")
    proc.terminate()
    sys.exit(1)

notif = {"jsonrpc": "2.0", "method": "notifications/initialized"}
proc.stdin.write((json.dumps(notif) + "\n").encode())
proc.stdin.flush()
time.sleep(1)

# tools/list
list_req = make_request("tools/list", {})
list_resp, list_raw = send_recv(proc, list_req, "tools/list", timeout=15)
if not list_resp:
    print("  FAILED to get tools/list!")
    proc.terminate()
    sys.exit(1)

tool_count = len(list_resp.get("result", {}).get("tools", []))
fixed_overhead = {"label": f"tools/list ({tool_count} tools)", "bytes": len(list_raw),
                  "tokens": tokens(len(list_raw)), "category": "fixed_overhead"}
print(f"  ✓ tools/list ({tool_count} tools)                                   {fixed_overhead['tokens']:>6,} tokens ({fixed_overhead['bytes']:,} bytes)")
print()


# ── S5: Query Authoring (MCP) ────────────────────────────────────

print("── S5: Query Authoring & Validation (MCP) ──")
print()

call_tool(proc, "build_query", {
    "intent": "find error logs from radiant service in the last hour",
    "filters": {"application": "radiant", "severity": "error"},
}, "s5", "build_query")

call_tool(proc, "validate_query", {
    "query": "source logs | filter $l.applicationname == 'radiant' AND $m.severity >= ERROR | limit 10",
}, "s5", "validate_query (with AND mistake)")

call_tool(proc, "validate_query", {
    "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | limit 10",
}, "s5", "validate_query (correct)")

call_tool(proc, "explain_query", {
    "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | groupby $l.subsystemname aggregate count() as errors | orderby -errors | limit 10",
}, "s5", "explain_query")

call_tool(proc, "get_dataprime_reference", {}, "s5", "get_dataprime_reference")

call_tool(proc, "get_query_templates", {}, "s5", "get_query_templates")

call_tool(proc, "estimate_query_cost", {
    "query": "source logs | filter $l.applicationname == 'radiant' && $m.severity >= ERROR | limit 100",
    "tier": "archive",
}, "s5", "estimate_query_cost")

print()

# ── S6: Ingestion Pipeline (MCP) ─────────────────────────────────

print("── S6: Ingestion Pipeline (MCP) ──")
print()

call_tool(proc, "list_rule_groups", {}, "s6", "list_rule_groups")

call_tool(proc, "list_enrichments", {}, "s6", "list_enrichments")

call_tool(proc, "discover_log_fields", {
    "application": "radiant",
}, "s6", "discover_log_fields")

print()

# ── S7: Data Governance (MCP) ────────────────────────────────────

print("── S7: Data Governance (MCP) ──")
print()

call_tool(proc, "list_data_access_rules", {}, "s7", "list_data_access_rules")

call_tool(proc, "list_outgoing_webhooks", {}, "s7", "list_outgoing_webhooks")

resp, _ = call_tool(proc, "create_outgoing_webhook", {
    "webhook": {
        "type": "generic",
        "name": f"benchmark-mcp-webhook-{run_id}",
        "url": "https://example.com/webhook",
    },
}, "s7", "create_outgoing_webhook")
mcp_wh_id = extract_id(resp)

if mcp_wh_id:
    call_tool(proc, "get_outgoing_webhook", {"id": mcp_wh_id}, "s7", "get_outgoing_webhook")
    call_tool(proc, "delete_outgoing_webhook", {"id": mcp_wh_id}, "s7", "delete_outgoing_webhook")

print()

# ── S8: E2M & Streaming (MCP) ────────────────────────────────────

print("── S8: E2M & Streaming (MCP) ──")
print()

call_tool(proc, "list_e2m", {}, "s8", "list_e2m")

call_tool(proc, "list_streams", {}, "s8", "list_streams")

call_tool(proc, "get_event_stream_targets", {}, "s8", "get_event_stream_targets")

print()

# ── S9: API Discovery & Meta (MCP) ───────────────────────────────

print("── S9: API Discovery & Meta (MCP) ──")
print()

call_tool(proc, "list_tool_categories", {}, "s9", "list_tool_categories")

call_tool(proc, "search_tools", {"query": "alert"}, "s9", "search_tools (alert)")

call_tool(proc, "session_context", {}, "s9", "session_context")

call_tool(proc, "health_check", {}, "s9", "health_check", timeout=30)

print()

# Shutdown MCP
proc.stdin.close()
proc.terminate()
try:
    proc.wait(timeout=5)
except subprocess.TimeoutExpired:
    proc.kill()


# ══════════════════════════════════════════════════════════════════
# ANALYSIS
# ══════════════════════════════════════════════════════════════════

print()
print("═══════════════════════════════════════════════════════════════")
print("  RESULTS: Remaining Features (S5-S9)")
print("═══════════════════════════════════════════════════════════════")
print()

results = {}

for sid, label in [
    ("s5", "S5: Query Authoring & Validation"),
    ("s6", "S6: Ingestion Pipeline"),
    ("s7", "S7: Data Governance"),
    ("s8", "S8: E2M & Streaming"),
    ("s9", "S9: API Discovery & Meta"),
]:
    s = scenarios[sid]
    skills_tokens = sum(e["tokens"] for e in s["skills"])
    mcp_response_tokens = sum(e["tokens"] for e in s["mcp"])
    mcp_total = fixed_overhead["tokens"] + mcp_response_tokens

    winner = "Skills" if skills_tokens < mcp_total else "MCP"
    bigger = max(skills_tokens, mcp_total)
    delta = abs(skills_tokens - mcp_total)
    pct = round(delta / bigger * 100) if bigger > 0 else 0

    results[sid] = {
        "label": label,
        "skills_tokens": skills_tokens,
        "mcp_response_tokens": mcp_response_tokens,
        "mcp_total": mcp_total,
        "winner": winner,
        "delta": delta,
        "delta_pct": pct,
    }

    print(f"━━━ {label} ━━━")
    print(f"  Skills: {skills_tokens:>6,} tokens")
    print(f"  MCP:    {mcp_total:>6,} tokens (overhead {fixed_overhead['tokens']:,} + responses {mcp_response_tokens:,})")
    print(f"  Winner: {winner} (by {delta:,} tokens, {pct}%)")
    print()

# Summary table
print("━━━ SUMMARY TABLE ━━━")
print()
print(f"  {'Scenario':<40s} {'Skills':>8s} {'MCP':>8s} {'Winner':>8s} {'Delta':>8s}")
print(f"  {'─' * 40} {'─' * 8} {'─' * 8} {'─' * 8} {'─' * 8}")
for sid in ["s5", "s6", "s7", "s8", "s9"]:
    r = results[sid]
    print(f"  {r['label']:<40s} {r['skills_tokens']:>7,} {r['mcp_total']:>7,} {r['winner']:>8s} {r['delta_pct']:>6}%")

print()

# Write JSON
output = {
    "fixed_overhead_tokens": fixed_overhead["tokens"],
    "fixed_overhead_bytes": fixed_overhead["bytes"],
    "scenarios": {},
}

for sid in ["s5", "s6", "s7", "s8", "s9"]:
    s = scenarios[sid]
    r = results[sid]
    output["scenarios"][sid] = {
        "label": r["label"],
        "skills": {
            "steps": s["skills"],
            "total_tokens": r["skills_tokens"],
        },
        "mcp": {
            "steps": s["mcp"],
            "response_tokens": r["mcp_response_tokens"],
            "total_tokens": r["mcp_total"],
        },
        "winner": r["winner"],
        "delta_tokens": r["delta"],
        "delta_pct": r["delta_pct"],
    }

result_file = f"{OUT_DIR}/remaining-measured.json"
with open(result_file, "w") as f:
    json.dump(output, f, indent=2)
    f.write("\n")

print(f"✓ Results: {result_file}")
