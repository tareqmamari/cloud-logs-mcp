---
name: ibm-cloud-logs-query
description: >
  Write and validate DataPrime and Lucene queries for IBM Cloud Logs.
  Activate when the user needs to search logs, build analytical queries,
  filter by severity/application/subsystem, aggregate metrics, compute
  percentiles, detect anomalies, or troubleshoot query syntax errors.
  Covers DataPrime piped syntax, field prefixes ($l, $m, $d), 30+
  commands, 25+ ready-to-use query templates, auto-correction rules,
  and common mistakes to avoid.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: log-analytics
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Query Skill

## When to Activate

Use this skill when the user:
- Wants to search, filter, or analyze logs in IBM Cloud Logs
- Asks about DataPrime or Lucene query syntax
- Needs to build queries for error investigation, performance analysis, or security auditing
- Is troubleshooting query syntax errors or unexpected results
- Wants to aggregate, group, or compute statistics on log data
- Asks about log field names, severity levels, or data prefixes

## Prerequisites

### Required Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LOGS_API_KEY` | Yes | IBM Cloud API key ([create one](https://cloud.ibm.com/iam/apikeys)) |
| `LOGS_SERVICE_URL` | Yes* | Instance endpoint: `https://{instance-id}.api.{region}.logs.cloud.ibm.com` |
| `LOGS_INSTANCE_ID` | Alt* | Instance UUID (alternative to full URL) |
| `LOGS_REGION` | Alt* | Region code: `us-south`, `eu-de`, `eu-gb`, `au-syd`, `us-east`, `jp-tok` |

*Either `LOGS_SERVICE_URL` or both `LOGS_INSTANCE_ID` + `LOGS_REGION` must be set. The URL is auto-constructed as `https://{LOGS_INSTANCE_ID}.api.{LOGS_REGION}.logs.cloud.ibm.com`.

### Authentication

- **CLI** (recommended): Run `ibmcloud login --apikey $LOGS_API_KEY -r $LOGS_REGION`, then pass `--service-url $LOGS_SERVICE_URL` to each `ibmcloud logs` command.
- **curl**: Exchange API key for a bearer token first:
  ```bash
  TOKEN=$(curl -s -X POST "https://iam.cloud.ibm.com/identity/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=$LOGS_API_KEY" \
    | jq -r .access_token)
  ```
  Then use `Authorization: Bearer $TOKEN` in subsequent requests to `$LOGS_SERVICE_URL`.

## Core Concepts

### Query Structure
DataPrime uses piped syntax. Queries flow left to right:
```
source logs | filter <condition> | groupby <field> aggregate <func> | orderby <expr> | limit <n>
```

### Field Access Prefixes
| Prefix | Layer | Description | Examples |
|--------|-------|-------------|----------|
| `$d.` | User Data | Log payload (default, can omit) | `$d.status_code`, `$d.message`, `status_code` |
| `$l.` | Labels | Application metadata | `$l.applicationname`, `$l.subsystemname` |
| `$m.` | Metadata | System metadata | `$m.severity`, `$m.timestamp` |
| `$p.` | Parameters | Dashboard template variables | `$p.myVariable` |

### Valid Label Fields ($l.)
`applicationname`, `subsystemname`, `computername`, `ipaddress`, `threadid`, `processid`, `classname`, `methodname`, `category`

### Valid Metadata Fields ($m.)
`severity`, `timestamp`, `priority`

### Severity Levels
```
VERBOSE (0) < DEBUG (1) < INFO (2) < WARNING (3) < ERROR (4) < CRITICAL (5)
```
Use named values, not numbers: `filter $m.severity >= WARNING`

## CRITICAL: Common Mistakes to Avoid

These are the most frequent errors. Get these wrong and your query will fail:

1. **Use `&&` not `AND`** — DataPrime uses `&&` for logical AND
   ```
   WRONG: filter $l.applicationname == 'myapp' AND $m.severity >= ERROR
   RIGHT: filter $l.applicationname == 'myapp' && $m.severity >= ERROR
   ```

2. **Use `||` not `OR`** — DataPrime uses `||` for logical OR
   ```
   WRONG: filter severity == ERROR OR severity == CRITICAL
   RIGHT: filter $m.severity == ERROR || $m.severity == CRITICAL
   ```

3. **Use `==` not `=`** — Single `=` is not valid for comparison
   ```
   WRONG: filter $l.applicationname = 'myapp'
   RIGHT: filter $l.applicationname == 'myapp'
   ```

4. **Use single quotes, not double quotes** for string values
   ```
   WRONG: filter $l.applicationname == "myapp"
   RIGHT: filter $l.applicationname == 'myapp'
   ```

5. **`~~` is NOT supported** — Use `matches()` for regex, `contains()` for substring
   ```
   WRONG: filter $d.message ~~ 'error.*timeout'
   RIGHT: filter $d.message:string.matches(/error.*timeout/)
   RIGHT: filter $d.message:string.contains('error')
   ```

6. **Cast mixed-type fields** — Fields like `message`, `msg`, `log`, `error` often have `object|string` type. Add `:string` before calling string methods:
   ```
   WRONG: filter $d.message.contains('error')
   RIGHT: filter $d.message:string.contains('error')
   ```

7. **`$l.namespace` does not exist** — Use `$l.applicationname` (K8s namespace maps to applicationname)

8. **`$m.level` does not exist** — Use `$m.severity` with named values (ERROR, not 5)

9. **`LIKE` and `IN` are not valid** — Use `contains()`/`matches()` and `||` chains instead

10. **Use `orderby` not `sort`** — DataPrime uses `orderby` (alias: `sortby`), not `sort`

11. **Use `desc` not `-` for timestamp ordering** — `-` prefix only works for numeric fields
   ```
   WRONG: orderby -$m.timestamp
   RIGHT: orderby $m.timestamp desc
   ```

## DataPrime Quick Reference

### Essential Commands
| Command | Aliases | Syntax | Example |
|---------|---------|--------|---------|
| `source` | — | `source <name>` | `source logs` |
| `filter` | `f`, `where` | `filter <condition>` | `filter $m.severity >= ERROR` |
| `groupby` | — | `groupby <expr> aggregate <func>` | `groupby $l.applicationname aggregate count() as cnt` (note: `calculate` can be used as an alias for `aggregate` within `groupby` pipelines) |
| `aggregate` | `agg` | `aggregate <func> as <alias>` | `aggregate avg(duration) as avg_dur` |
| `orderby` | `sortby` | `orderby <expr> [asc\|desc]` | `orderby error_count desc` |
| `limit` | `l` | `limit <n>` | `limit 100` |
| `create` | `add`, `c`, `a` | `create <field> from <expr>` | `create is_error from status_code >= 400` |
| `extract` | — | `extract <field> into <target> using <extractor>` | `extract message into fields using regexp(e=/(?<user>\w+)/)` |
| `choose` | — | `choose <fields>` | `choose timestamp, message, severity` |
| `remove` | — | `remove <fields>` | `remove sensitive_data` |
| `distinct` | — | `distinct <expr>` | `distinct $l.applicationname` |
| `countby` | — | `countby <expr>` | `countby status_code` |
| `top` | — | `top <n> <expr> by <order>` | `top 5 endpoint by request_count` |
| `lucene` | — | `lucene '<query>'` | `lucene 'error AND timeout'` |
| `find` | `text` | `find '<text>' [in <field>]` | `find 'error' in message` |
| `join` | — | `join (<subquery>) on left=><f> == right=><f> into <k>` | See references |
| `roundTime` | — | `roundTime(<ts>, <interval>)` | `roundTime($m.timestamp, 5m)` |

### Key Functions
**Aggregation:** `count()`, `sum(f)`, `avg(f)`, `min(f)`, `max(f)`, `percentile(f, p)`, `approx_count_distinct(f)`, `stddev(f)`

**String:** `contains(s, sub)`, `startsWith(s, prefix)`, `endsWith(s, suffix)`, `matches(s, /regex/)`, `toLowerCase(s)`, `concat(s1, s2)`, `length(s)`

**Time:** `now()`, `roundTime(ts, interval)`, `formatTimestamp(ts, fmt)`, `parseTimestamp(s, fmt)`, `diffTime(ts1, ts2)`

**Conditional:** `if(cond, then, else)`, `coalesce(v1, v2)`, `case { cond1 -> val1, cond2 -> val2 }`

### Operators
| Type | Operators | Notes |
|------|-----------|-------|
| Comparison | `==`, `!=`, `>`, `<`, `>=`, `<=` | Use `==` not `=` |
| Logical | `&&`, `\|\|`, `!` | NOT `AND`/`OR` |
| Text search | `~`, `!~` | Contains / does not contain |

## Top 10 Query Patterns

### 1. Error Hotspots (Start here during incidents)
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname, $l.subsystemname
| aggregate count() as error_count
| orderby -error_count | limit 20
```

### 2. Error Timeline (Visualize when errors started)
```
source logs | filter $m.severity >= ERROR
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as errors
| orderby time_bucket
```

### 3. Top Error Messages (Find most impactful errors)
```
source logs | filter $m.severity >= ERROR
| groupby $d.message:string
| aggregate count() as occurrences, min($m.timestamp) as first_seen, max($m.timestamp) as last_seen
| filter occurrences >= 3
| orderby -occurrences | limit 30
```

### 4. Error Details for Specific App
```
source logs | filter $l.applicationname == '{APP_NAME}' && $m.severity >= ERROR
| choose $m.timestamp, $m.severity, $l.subsystemname, $d.message, $d.error, $d.stack_trace
| orderby $m.timestamp desc | limit 100
```

### 5. Latency Percentiles by Endpoint
```
source logs | filter $d.response_time_ms > 0
| groupby $d.endpoint
| aggregate percentile($d.response_time_ms, 50) as p50,
          percentile($d.response_time_ms, 95) as p95,
          percentile($d.response_time_ms, 99) as p99,
          count() as requests
| orderby -requests | limit 20
```

### 6. Authentication Failures
```
source logs
| filter $d.event_type == 'auth_failure'
   || $d.message:string.contains('authentication failed')
   || $d.message:string.contains('invalid credentials')
| groupby $d.source_ip, $d.username
| aggregate count() as failures
| filter failures > 3 | orderby -failures
```

### 7. Service Health Overview
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname
| aggregate count() as error_count
| orderby -error_count | limit 20
```

### 8. Traffic Overview by Service
```
source logs
| groupby $l.applicationname
| aggregate count() as volume, approx_count_distinct($l.subsystemname) as components
| orderby -volume | limit 20
```

### 9. Restart / Crash Detection
```
source logs
| filter $d.message:string.contains('starting')
   || $d.message:string.contains('shutdown')
   || $d.message:string.contains('OOMKilled')
   || $d.message:string.contains('terminated')
| choose $m.timestamp, $l.applicationname, $l.subsystemname, $d.message
| orderby $m.timestamp desc | limit 50
```

### 10. Log Volume by Application Over Time
```
source logs
| groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname
| aggregate count() as logs
| orderby time_bucket
```

## Auto-Correction Rules

These common issues are auto-corrected by the IBM Cloud Logs API. When writing queries manually, be aware:

| What You Write | What It Becomes | Why |
|---------------|-----------------|-----|
| `$m.severity >= 5` | `$m.severity >= CRITICAL` | Numeric severity not supported; use named values |
| `$d.message.contains('x')` | `$d.message:string.contains('x')` | Mixed-type fields need `:string` cast |
| `$d.level == 'ERROR'` | `$m.severity == ERROR` | Level field has mixed types; use $m.severity |
| `\| sort -field` | `\| orderby field desc` | `sort` is not valid DataPrime; use `orderby` |
| `count_distinct(f)` | `approx_count_distinct(f)` | Only approximate variant supported |
| `bucket($m.timestamp, 5m)` | `roundTime($m.timestamp, 5m)` | `bucket()` not valid; use `roundTime()` |
| `aggregate x = count()` | `aggregate count() as x` | Use `as` for aliases, not `=` |
| `orderby -$m.timestamp` | `orderby $m.timestamp desc` | `-` prefix only works for numeric fields |

## Choosing the Right Tier

Before running a query, determine which storage tier to target. The tier affects both cost and data availability.

### Step 1: Check TCO Policies

```bash
ibmcloud logs policies --output json --service-url $LOGS_SERVICE_URL
```

Review the output for policies matching your target application. Key fields:
- `priority: "type_high"` → logs go to **both** `frequent_search` AND `archive`
- `priority: "type_medium"` → logs go to `archive` **only**
- `priority: "type_low"` or `"type_block"` → logs are **dropped** (not stored)

### Step 2: Select Tier

| Situation | Use `--tier` | Why |
|-----------|-------------|-----|
| No TCO policies exist | `frequent_search` | Logs go to both tiers by default; frequent_search is faster |
| Target app has `type_high` policy | `frequent_search` | Logs are in both tiers; frequent_search is faster |
| Target app has `type_medium` policy | `archive` | Logs are only in archive |
| Querying > 24h of data | `archive` | Archive has longer retention |
| Unsure | `archive` | Archive always has the data (safest default) |

Pass the tier flag to the CLI: `ibmcloud logs query --tier archive ...`

## Step-by-Step: Building a Query

1. **Start with source:** `source logs`
2. **Add time filter if needed:** `| last 1h` or `| between @'2024-01-01' and @'2024-01-02'`
3. **Filter to relevant data:** `| filter $l.applicationname == 'myapp' && $m.severity >= WARNING`
4. **Aggregate or select:** `| groupby $l.subsystemname aggregate count() as errors` or `| choose $m.timestamp, $d.message`
5. **Order results:** `| orderby -errors` (prefix with `-` for descending)
6. **Limit output:** `| limit 20`

## Using the IBM Cloud CLI

All queries can be executed directly via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# Prerequisites (one-time setup)
ibmcloud plugin install logs
ibmcloud login --apikey <your-api-key> -r <region>

# Run a DataPrime query
ibmcloud logs query \
  --query 'source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as errors | orderby -errors | limit 20' \
  --output json

# Submit a background query for large time ranges
ibmcloud logs bgq-create \
  --query 'source logs | filter $m.severity >= ERROR' \
  --output json

# Check background query status and retrieve results
ibmcloud logs bgq-status --id <query-id>
ibmcloud logs bgq-data --id <query-id> --output json
```

The CLI handles authentication, SSE parsing, and output formatting automatically — no manual token exchange or header management required.

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Use scripts with `--output-file`** to write large results to disk instead of stdout:
  ```
  ./scripts/validate-query.sh --json --output-file /tmp/validation.json 'source logs | ...'
  ```
  Then read the summary line from stdout and only load the file if the user needs details.
- **Write generated queries to files** when producing multiple queries or long outputs:
  ```
  # Write to file, show user the path
  echo "source logs | filter ..." > /tmp/query.dpl
  ```
- **Prefer `--json` output** from scripts — it's structured and agents can extract only needed fields.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [DataPrime Command Reference](references/dataprime-commands.md) — Full 30+ command catalog with syntax, aliases, examples
- [Query Templates Catalog](references/query-templates.md) — All 25+ templates across 7 categories
- [DataPrime Functions Reference](references/dataprime-functions.md) — Aggregation, string, time, conditional, array functions
- [Lucene Integration](references/lucene-integration.md) — Lucene syntax and combining with DataPrime
- [Query Validation Script](scripts/validate-query.sh) — Offline DataPrime query validator (supports `--json`, `--output-file`)

> **Windows note:** Scripts require bash (available via [Git for Windows](https://gitforwindows.org/) or WSL).
