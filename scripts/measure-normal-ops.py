#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = ["requests"]
# ///
"""
Measure token costs for normal CRUD operations (alerts, dashboards, views)
against the cxint IBM Cloud Logs instance. Measures both Skills+CLI and MCP
approaches side-by-side.

Requires: LOGS_SERVICE_URL, LOGS_API_KEY environment variables.
"""

import json
import os
import subprocess
import sys
import time

import requests

# ── Config ────────────────────────────────────────────────────────

BYTES_PER_TOKEN = 3.79
IAM_TOKEN_URL = "https://iam.cloud.ibm.com/identity/token"
OUT_DIR = "/tmp/iteration-tax/normal-ops"
SKILLS_DIR = os.path.join(os.path.dirname(__file__), "..", ".agents", "skills")
MCP_BINARY = os.path.join(os.path.dirname(__file__), "..", "bin", "logs-mcp-server")

for var in ("LOGS_SERVICE_URL", "LOGS_API_KEY"):
    if not os.environ.get(var):
        print(f"Error: {var} must be set", file=sys.stderr)
        sys.exit(1)

SERVICE_URL = os.environ["LOGS_SERVICE_URL"]
API_KEY = os.environ["LOGS_API_KEY"]

os.makedirs(OUT_DIR, exist_ok=True)
os.makedirs(f"{OUT_DIR}/skills", exist_ok=True)
os.makedirs(f"{OUT_DIR}/mcp", exist_ok=True)

# ── Helpers ───────────────────────────────────────────────────────

def tokens(byte_count):
    return round(byte_count / BYTES_PER_TOKEN)


def file_size(path):
    return os.path.getsize(path)


def log_entry(label, byte_count, category):
    t = tokens(byte_count)
    marker = "✓" if category not in ("error_retry",) else "✗"
    print(f"  {marker} {label:<55s} {t:>6,} tokens ({byte_count:,} bytes)")
    return {"label": label, "bytes": byte_count, "tokens": t, "category": category}


# ── IAM Auth ──────────────────────────────────────────────────────

print("Authenticating...")
resp = requests.post(IAM_TOKEN_URL, data={
    "grant_type": "urn:ibm:params:oauth:grant-type:apikey",
    "apikey": API_KEY,
}, headers={"Content-Type": "application/x-www-form-urlencoded"}, timeout=30)
resp.raise_for_status()
TOKEN = resp.json()["access_token"]
auth_bytes = len(resp.content)
print(f"  Auth response: {tokens(auth_bytes):,} tokens ({auth_bytes:,} bytes)")
print()


def api_call(method, path, body=None):
    """Make an API call and return (response_json, response_bytes)."""
    url = f"{SERVICE_URL}{path}"
    headers = {
        "Authorization": f"Bearer {TOKEN}",
        "Content-Type": "application/json",
    }
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


# ══════════════════════════════════════════════════════════════════
# PART 1: SKILLS + CLI MEASUREMENT
# ══════════════════════════════════════════════════════════════════

print("═══════════════════════════════════════════════════════════════")
print("  SKILLS + CLI: Normal Operations")
print("═══════════════════════════════════════════════════════════════")
print()

skills_ledger = []

# ── 4a: Alert CRUD ───────────────────────────────────────────────

print("── 4a: Alert Lifecycle (Skills + CLI) ──")
print()

# Agent reads skill files to learn how alerts work
alert_skill = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "SKILL.md")
skills_ledger.append(log_entry("Read ibm-cloud-logs/SKILL.md", file_size(alert_skill), "skill_read"))

alerting_guide = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "alerting-guide.md")
skills_ledger.append(log_entry("Read alerting-guide.md", file_size(alerting_guide), "skill_read"))

strategy_ref = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "strategy-matrix.md")
skills_ledger.append(log_entry("Read strategy-matrix.md", file_size(strategy_ref), "skill_read"))

# Create alert (IBM Cloud Logs native API format)
alert_def = {
    "name": "benchmark-normal-ops-alert",
    "description": "Benchmark: Normal ops alert for radiant errors",
    "is_active": True,
    "severity": "error_or_unspecified",
    "type": "logs_threshold",
    "logs_threshold": {
        "condition_type": "more_than_or_unspecified",
        "logs_filter": {
            "simple_filter": {
                "lucene_query": "applicationname:radiant AND severity:error"
            }
        },
        "rules": [{
            "condition": {
                "threshold": 100,
                "time_window": {
                    "logs_time_window_specific_value": "minutes_10"
                }
            },
            "override": {"priority": "p2"}
        }]
    },
    "incidents_settings": {
        "minutes": 10,
        "notify_on": "triggered_only_unspecified"
    }
}
data, nbytes = api_call("POST", "/v1/alert_definitions", alert_def)
with open(f"{OUT_DIR}/skills/create-alert.json", "w") as f:
    json.dump(data, f, indent=2)
skills_ledger.append(log_entry("POST /v1/alert_definitions (create)", nbytes, "api_call"))
alert_id = data.get("id", data.get("unique_identifier"))

# List alerts
data, nbytes = api_call("GET", "/v1/alert_definitions")
with open(f"{OUT_DIR}/skills/list-alerts.json", "w") as f:
    json.dump(data, f, indent=2)
alert_count = len(data) if isinstance(data, list) else len(data.get("alert_defs", data.get("alerts", [])))
skills_ledger.append(log_entry(f"GET /v1/alert_definitions (list, {alert_count} alerts)", nbytes, "api_call"))

# Get alert
if alert_id:
    data, nbytes = api_call("GET", f"/v1/alert_definitions/{alert_id}")
    with open(f"{OUT_DIR}/skills/get-alert.json", "w") as f:
        json.dump(data, f, indent=2)
    skills_ledger.append(log_entry("GET /v1/alert_definitions/{id} (get)", nbytes, "api_call"))

    # Delete alert (cleanup)
    _, nbytes = api_call("DELETE", f"/v1/alert_definitions/{alert_id}")
    skills_ledger.append(log_entry("DELETE /v1/alert_definitions/{id} (delete)", nbytes, "api_call"))

print()

# ── 4b: Dashboard CRUD ──────────────────────────────────────────

print("── 4b: Dashboard Lifecycle (Skills + CLI) ──")
print()

dash_guide = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "dashboards-guide.md")
skills_ledger.append(log_entry("Read dashboards-guide.md", file_size(dash_guide), "skill_read"))

dash_schema_ref = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "dashboard-schema.md")
skills_ledger.append(log_entry("Read dashboard-schema.md", file_size(dash_schema_ref), "skill_read"))

# Create dashboard (format matching IBM Cloud Logs API)
import uuid
run_id = uuid.uuid4().hex[:8]
dash_def = {
    "name": f"Benchmark Normal Ops Dashboard {run_id}",
    "description": "Benchmark: RED monitoring dashboard",
    "layout": {
        "sections": [{
            "id": {"value": str(uuid.uuid4())},
            "rows": [{
                "id": {"value": str(uuid.uuid4())},
                "appearance": {"height": 19},
                "widgets": [{
                    "id": {"value": str(uuid.uuid4())},
                    "title": "Error Rate Over Time",
                    "definition": {
                        "line_chart": {
                            "query_definitions": [{
                                "id": str(uuid.uuid4()),
                                "query": {
                                    "logs": {
                                        "aggregations": [{"count": {}}],
                                        "filters": [],
                                        "group_by": [],
                                        "group_bys": []
                                    }
                                }
                            }]
                        }
                    }
                }]
            }]
        }]
    },
    "filters": [],
    "annotations": []
}
data, nbytes = api_call("POST", "/v1/dashboards", dash_def)
with open(f"{OUT_DIR}/skills/create-dashboard.json", "w") as f:
    json.dump(data, f, indent=2)
skills_ledger.append(log_entry("POST /v1/dashboards (create)", nbytes, "api_call"))
dash_id = data.get("id")

# List dashboards
data, nbytes = api_call("GET", "/v1/dashboards")
with open(f"{OUT_DIR}/skills/list-dashboards.json", "w") as f:
    json.dump(data, f, indent=2)
dash_count = len(data) if isinstance(data, list) else len(data.get("dashboards", []))
skills_ledger.append(log_entry(f"GET /v1/dashboards (list, {dash_count} dashboards)", nbytes, "api_call"))

# Get dashboard
if dash_id:
    data, nbytes = api_call("GET", f"/v1/dashboards/{dash_id}")
    with open(f"{OUT_DIR}/skills/get-dashboard.json", "w") as f:
        json.dump(data, f, indent=2)
    skills_ledger.append(log_entry("GET /v1/dashboards/{id} (get)", nbytes, "api_call"))

    # Delete dashboard
    _, nbytes = api_call("DELETE", f"/v1/dashboards/{dash_id}")
    skills_ledger.append(log_entry("DELETE /v1/dashboards/{id} (delete)", nbytes, "api_call"))

print()

# ── 4c: View CRUD ───────────────────────────────────────────────

print("── 4c: View Lifecycle (Skills + CLI) ──")
print()

access_guide = os.path.join(SKILLS_DIR, "ibm-cloud-logs", "references", "access-control-guide.md")
skills_ledger.append(log_entry("Read access-control-guide.md", file_size(access_guide), "skill_read"))

# Create view
view_def = {
    "name": f"Benchmark Normal Ops View {run_id}",
    "search_query": {"query": "applicationname:radiant AND severity:error"},
    "time_selection": {"quick_selection": {"caption": "Last 1 hour", "seconds": 3600}},
    "filters": {
        "filters": [
            {"name": "applicationname", "selected_values": {"radiant": True}}
        ]
    }
}
data, nbytes = api_call("POST", "/v1/views", view_def)
with open(f"{OUT_DIR}/skills/create-view.json", "w") as f:
    json.dump(data, f, indent=2)
skills_ledger.append(log_entry("POST /v1/views (create)", nbytes, "api_call"))
view_id = data.get("id")

# List views
data, nbytes = api_call("GET", "/v1/views")
with open(f"{OUT_DIR}/skills/list-views.json", "w") as f:
    json.dump(data, f, indent=2)
view_count = len(data) if isinstance(data, list) else len(data.get("views", []))
skills_ledger.append(log_entry(f"GET /v1/views (list, {view_count} views)", nbytes, "api_call"))

# Delete view
if view_id:
    _, nbytes = api_call("DELETE", f"/v1/views/{view_id}")
    skills_ledger.append(log_entry("DELETE /v1/views/{id} (delete)", nbytes, "api_call"))

print()

# ── Skills Summary ────────────────────────────────────────────────

print("── Skills Summary ──")
print()

skills_by_cat = {}
for e in skills_ledger:
    cat = e["category"]
    if cat not in skills_by_cat:
        skills_by_cat[cat] = {"tokens": 0, "bytes": 0, "count": 0}
    skills_by_cat[cat]["tokens"] += e["tokens"]
    skills_by_cat[cat]["bytes"] += e["bytes"]
    skills_by_cat[cat]["count"] += 1

skills_total = sum(e["tokens"] for e in skills_ledger)

for cat in ("skill_read", "api_call"):
    if cat in skills_by_cat:
        c = skills_by_cat[cat]
        print(f"  {cat:<20s} {c['tokens']:>6,} tokens ({c['count']} items, {c['bytes']:,} bytes)")
print(f"  {'TOTAL':<20s} {skills_total:>6,} tokens")
print()


# ══════════════════════════════════════════════════════════════════
# PART 2: MCP MEASUREMENT
# ══════════════════════════════════════════════════════════════════

print("═══════════════════════════════════════════════════════════════")
print("  MCP: Normal Operations")
print("═══════════════════════════════════════════════════════════════")
print()

mcp_ledger = []
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


def call_tool(proc, name, arguments, label, timeout=120):
    req = make_request("tools/call", {"name": name, "arguments": arguments})
    resp, raw = send_recv(proc, req, label, timeout=timeout)
    category = "tool_response"
    if resp:
        result = resp.get("result", {})
        if result.get("isError"):
            category = "error_retry"
        fname = f"{OUT_DIR}/mcp/{name}-{msg_id}.json"
        with open(fname, "w") as f:
            json.dump(resp, f, indent=2)
    else:
        category = "error_retry"
        raw = b""
    entry = log_entry(label, len(raw), category)
    mcp_ledger.append(entry)
    return resp, raw


# Start MCP server
print("Starting MCP server binary...")
env = {
    **os.environ,
    "ENVIRONMENT": "production",
    "LOGS_HEALTH_PORT": "0",
}
proc = subprocess.Popen(
    [MCP_BINARY],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    env=env,
)
time.sleep(2)

# Initialize
init_req = make_request("initialize", {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {"name": "benchmark-normal-ops", "version": "1.0"},
})
init_resp, init_raw = send_recv(proc, init_req, "initialize", timeout=10)
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
if list_resp:
    tool_count = len(list_resp.get("result", {}).get("tools", []))
    mcp_ledger.append(log_entry(f"tools/list ({tool_count} tools)", len(list_raw), "fixed_overhead"))
else:
    print("  FAILED to get tools/list!")
    proc.terminate()
    sys.exit(1)

print()

# ── 4a: Alert CRUD (MCP) ─────────────────────────────────────────

print("── 4a: Alert Lifecycle (MCP) ──")
print()

resp, _ = call_tool(proc, "create_alert_definition", {
    "definition": {
        "name": "benchmark-normal-ops-mcp-alert",
        "description": "Benchmark: Normal ops alert via MCP",
        "is_active": True,
        "severity": "error_or_unspecified",
        "type": "logs_threshold",
        "logs_threshold": {
            "condition_type": "more_than_or_unspecified",
            "logs_filter": {
                "simple_filter": {
                    "lucene_query": "applicationname:radiant AND severity:error"
                }
            },
            "rules": [{
                "condition": {
                    "threshold": 100,
                    "time_window": {
                        "logs_time_window_specific_value": "minutes_10"
                    }
                },
                "override": {"priority": "p2"}
            }]
        },
        "incidents_settings": {
            "minutes": 10,
            "notify_on": "triggered_only_unspecified"
        }
    },
}, "create_alert_definition")

# Extract alert ID from response
mcp_alert_id = None
if resp:
    content = resp.get("result", {}).get("content", [])
    for c in content:
        text = c.get("text", "")
        try:
            parsed = json.loads(text)
            mcp_alert_id = parsed.get("id", parsed.get("unique_identifier"))
        except (json.JSONDecodeError, TypeError):
            # Try to find ID in text
            if "id" in text:
                import re
                m = re.search(r'"(?:id|unique_identifier)":\s*"([^"]+)"', text)
                if m:
                    mcp_alert_id = m.group(1)

call_tool(proc, "list_alert_definitions", {}, "list_alert_definitions")

if mcp_alert_id:
    call_tool(proc, "get_alert_definition", {"id": mcp_alert_id}, "get_alert_definition")
    call_tool(proc, "delete_alert_definition", {"id": mcp_alert_id}, "delete_alert_definition")

print()

# ── 4b: Dashboard CRUD (MCP) ─────────────────────────────────────

print("── 4b: Dashboard Lifecycle (MCP) ──")
print()

import uuid as _uuid
mcp_run_id = _uuid.uuid4().hex[:8]
resp, _ = call_tool(proc, "create_dashboard", {
    "name": f"Benchmark Normal Ops MCP Dashboard {mcp_run_id}",
    "description": "Benchmark: RED monitoring via MCP",
    "layout": {
        "sections": [{
            "id": {"value": str(_uuid.uuid4())},
            "rows": [{
                "id": {"value": str(_uuid.uuid4())},
                "appearance": {"height": 19},
                "widgets": [{
                    "id": {"value": str(_uuid.uuid4())},
                    "title": "Error Rate Over Time",
                    "definition": {
                        "line_chart": {
                            "query_definitions": [{
                                "id": str(_uuid.uuid4()),
                                "query": {
                                    "logs": {
                                        "aggregations": [{"count": {}}],
                                        "filters": [],
                                        "group_by": [],
                                        "group_bys": []
                                    }
                                }
                            }]
                        }
                    }
                }]
            }]
        }]
    },
}, "create_dashboard")

mcp_dash_id = None
if resp:
    content = resp.get("result", {}).get("content", [])
    for c in content:
        text = c.get("text", "")
        try:
            parsed = json.loads(text)
            mcp_dash_id = parsed.get("id")
        except (json.JSONDecodeError, TypeError):
            import re
            m = re.search(r'"id":\s*"([^"]+)"', text)
            if m:
                mcp_dash_id = m.group(1)

call_tool(proc, "list_dashboards", {}, "list_dashboards")

if mcp_dash_id:
    call_tool(proc, "get_dashboard", {"dashboard_id": mcp_dash_id}, "get_dashboard")
    call_tool(proc, "delete_dashboard", {"dashboard_id": mcp_dash_id}, "delete_dashboard")

print()

# ── 4c: View CRUD (MCP) ──────────────────────────────────────────

print("── 4c: View Lifecycle (MCP) ──")
print()

resp, _ = call_tool(proc, "create_view", {
    "view": {
        "name": f"Benchmark Normal Ops MCP View {mcp_run_id}",
        "search_query": {"query": "applicationname:radiant AND severity:error"},
        "time_selection": {"quick_selection": {"caption": "Last 1 hour", "seconds": 3600}},
        "filters": {
            "filters": [
                {"name": "applicationname", "selected_values": {"radiant": True}}
            ]
        }
    },
}, "create_view")

mcp_view_id = None
if resp:
    content = resp.get("result", {}).get("content", [])
    for c in content:
        text = c.get("text", "")
        try:
            parsed = json.loads(text)
            mcp_view_id = parsed.get("id")
        except (json.JSONDecodeError, TypeError):
            import re
            m = re.search(r'"id":\s*"([^"]+)"', text)
            if m:
                mcp_view_id = m.group(1)

call_tool(proc, "list_views", {}, "list_views")

if mcp_view_id:
    call_tool(proc, "delete_view", {"id": mcp_view_id}, "delete_view")

print()

# Shutdown MCP
proc.stdin.close()
proc.terminate()
try:
    proc.wait(timeout=5)
except subprocess.TimeoutExpired:
    proc.kill()

# ── MCP Summary ───────────────────────────────────────────────────

print("── MCP Summary ──")
print()

mcp_by_cat = {}
for e in mcp_ledger:
    cat = e["category"]
    if cat not in mcp_by_cat:
        mcp_by_cat[cat] = {"tokens": 0, "bytes": 0, "count": 0}
    mcp_by_cat[cat]["tokens"] += e["tokens"]
    mcp_by_cat[cat]["bytes"] += e["bytes"]
    mcp_by_cat[cat]["count"] += 1

mcp_total = sum(e["tokens"] for e in mcp_ledger)
mcp_overhead = mcp_by_cat.get("fixed_overhead", {}).get("tokens", 0)
mcp_responses = mcp_by_cat.get("tool_response", {}).get("tokens", 0)

for cat in ("fixed_overhead", "tool_response", "error_retry"):
    if cat in mcp_by_cat:
        c = mcp_by_cat[cat]
        print(f"  {cat:<20s} {c['tokens']:>6,} tokens ({c['count']} items, {c['bytes']:,} bytes)")
print(f"  {'TOTAL':<20s} {mcp_total:>6,} tokens")
print()


# ══════════════════════════════════════════════════════════════════
# COMPARISON
# ══════════════════════════════════════════════════════════════════

print("═══════════════════════════════════════════════════════════════")
print("  COMPARISON: Normal Operations (Scenario 4)")
print("═══════════════════════════════════════════════════════════════")
print()

skills_skill_tokens = skills_by_cat.get("skill_read", {}).get("tokens", 0)
skills_api_tokens = skills_by_cat.get("api_call", {}).get("tokens", 0)

print(f"  Skills + CLI:")
print(f"    Skill file reads:  {skills_skill_tokens:>6,} tokens")
print(f"    API responses:     {skills_api_tokens:>6,} tokens")
print(f"    TOTAL:             {skills_total:>6,} tokens")
print()
print(f"  MCP:")
print(f"    Fixed overhead:    {mcp_overhead:>6,} tokens")
print(f"    Tool responses:    {mcp_responses:>6,} tokens")
print(f"    TOTAL:             {mcp_total:>6,} tokens")
print()

winner = "Skills" if skills_total < mcp_total else "MCP"
delta = abs(skills_total - mcp_total)
pct = round(delta / max(skills_total, mcp_total) * 100)
print(f"  Winner: {winner} (by {delta:,} tokens, {pct}%)")
print()

# ── Write results ─────────────────────────────────────────────────

output = {
    "scenario": "4-normal-operations",
    "skills": {
        "steps": skills_ledger,
        "by_category": {k: v for k, v in skills_by_cat.items()},
        "total_tokens": skills_total,
    },
    "mcp": {
        "steps": [e for e in mcp_ledger],
        "by_category": {k: v for k, v in mcp_by_cat.items()},
        "total_tokens": mcp_total,
    },
    "winner": winner,
    "delta_tokens": delta,
    "delta_pct": pct,
}

result_file = f"{OUT_DIR}/normal-ops-measured.json"
with open(result_file, "w") as f:
    json.dump(output, f, indent=2)
    f.write("\n")

print(f"✓ Results: {result_file}")
