---
name: ibm-cloud-logs
description: >
  Comprehensive IBM Cloud Logs skill covering query authoring, alerting,
  incident investigation, dashboards, cost optimization, ingestion,
  access control, and API reference. Activate for any IBM Cloud Logs task
  including DataPrime queries, SLO-based alerting, incident debugging,
  dashboard design, TCO policies, parsing rules, data access rules, or
  API operations. Domain-specific details are loaded on demand from
  references/*-guide.md files.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: log-analytics
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Skill

## When to Activate

Use this skill when the user works with IBM Cloud Logs in **any** capacity:

**Query & Analysis** — search/filter/analyze logs, DataPrime or Lucene syntax, aggregation, field names, severity levels, query troubleshooting

**Alerting & Monitoring** — create alerts, SLO/burn-rate monitoring, RED/USE methodology, alert noise reduction, Terraform/JSON alert config, runbooks

**Incident Investigation** — debug production issues, respond to incidents, root-cause analysis, trace requests across services, proactive health scans

**Dashboards** — create/update dashboards, widget types, chart configuration, monitoring views, dashboard folders

**Cost Optimization** — reduce costs, TCO policies, tier selection, Events-to-Metrics (E2M), data retention, query cost estimation

**Ingestion** — send logs, parsing rules, enrichments, event streams, log entry format, ingestion testing

**Access Control & Security** — data access rules, multi-tenant isolation, compliance (GDPR/PII), audit logging, security monitoring queries

**API & Operations** — API endpoints, authentication, error handling, rate limits, CLI command mapping, background queries

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

## Domain Routing

Load the relevant guide from `references/` based on the user's task. **Do not load guides eagerly** — only when the task requires domain-specific detail beyond what this file provides.

| Domain | Load | When |
|--------|------|------|
| Query authoring | [query-guide.md](references/query-guide.md) | Full command reference, all query patterns, auto-correction, tier selection |
| Alerting | [alerting-guide.md](references/alerting-guide.md) | RED/USE methodology, burn rate tables, component profiles, alert output |
| Incidents | [incident-guide.md](references/incident-guide.md) | 3 investigation modes, heuristic patterns, SOPs, remediation |
| Dashboards | [dashboards-guide.md](references/dashboards-guide.md) | Widget types, standard patterns, JSON structure, REST API |
| Cost | [cost-guide.md](references/cost-guide.md) | TCO policy design, tier selection strategy, E2M, cost checklist |
| Ingestion | [ingestion-guide.md](references/ingestion-guide.md) | Log format, parsing rules, enrichments, event streams, testing |
| Access control | [access-control-guide.md](references/access-control-guide.md) | Data access rules, views, audit, security queries, compliance |
| API | [api-guide.md](references/api-guide.md) | 87 operations by category, cost/rate limits, error handling |

## DataPrime Quick Reference

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

1. **Use `&&` not `AND`** — DataPrime uses `&&` for logical AND
   ```
   WRONG: filter $l.applicationname == 'myapp' AND $m.severity >= ERROR
   RIGHT: filter $l.applicationname == 'myapp' && $m.severity >= ERROR
   ```

2. **Use `||` not `OR`** — DataPrime uses `||` for logical OR

3. **Use `==` not `=`** — Single `=` is not valid for comparison

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

9. **`LIKE` and `IN` are not valid** — Use `contains()`/`matches()` and `||` chains

10. **Use `orderby` not `sort`** — DataPrime uses `orderby` (alias: `sortby`)

11. **Use `desc` not `-` for timestamp ordering**
    ```
    WRONG: orderby -$m.timestamp
    RIGHT: orderby $m.timestamp desc
    ```

## Essential Commands
| Command | Aliases | Syntax | Example |
|---------|---------|--------|---------|
| `source` | — | `source <name>` | `source logs` |
| `filter` | `f`, `where` | `filter <condition>` | `filter $m.severity >= ERROR` |
| `groupby` | — | `groupby <expr> aggregate <func>` | `groupby $l.applicationname aggregate count() as cnt` |
| `aggregate` | `agg` | `aggregate <func> as <alias>` | `aggregate avg(duration) as avg_dur` |
| `orderby` | `sortby` | `orderby <expr> [asc\|desc]` | `orderby error_count desc` |
| `limit` | `l` | `limit <n>` | `limit 100` |
| `create` | `add`, `c`, `a` | `create <field> from <expr>` | `create is_error from status_code >= 400` |
| `extract` | — | `extract <field> into <target> using <extractor>` | `extract message into fields using regexp(e=/(?<user>\w+)/)` |
| `choose` | — | `choose <fields>` | `choose timestamp, message, severity` |
| `distinct` | — | `distinct <expr>` | `distinct $l.applicationname` |
| `countby` | — | `countby <expr>` | `countby status_code` |
| `lucene` | — | `lucene '<query>'` | `lucene 'error AND timeout'` |
| `find` | `text` | `find '<text>' [in <field>]` | `find 'error' in message` |
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

## Top 5 Query Patterns

### 1. Error Hotspots (Start here during incidents)
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname, $l.subsystemname
| aggregate count() as error_count
| orderby -error_count | limit 20
```

### 2. Error Timeline
```
source logs | filter $m.severity >= ERROR
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as errors
| orderby time_bucket
```

### 3. Top Error Messages
```
source logs | filter $m.severity >= ERROR
| groupby $d.message:string
| aggregate count() as occurrences, min($m.timestamp) as first_seen, max($m.timestamp) as last_seen
| filter occurrences >= 3
| orderby -occurrences | limit 30
```

### 4. Latency Percentiles by Endpoint
```
source logs | filter $d.response_time_ms > 0
| groupby $d.endpoint
| aggregate percentile($d.response_time_ms, 50) as p50,
          percentile($d.response_time_ms, 95) as p95,
          percentile($d.response_time_ms, 99) as p99,
          count() as requests
| orderby -requests | limit 20
```

### 5. Log Volume by Application Over Time
```
source logs
| groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname
| aggregate count() as logs
| orderby time_bucket
```

## CRITICAL: Query Execution Strategy

These rules prevent raw log data from flooding the context window. A single
unfiltered query can return 148KB+ (39,000 tokens). Follow these rules strictly.

### Rule 1: Always Use Aggregation Before Raw Logs

Before ANY raw log query (`limit` without `groupby`), run an aggregation first:
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname aggregate count() as error_count
| orderby -error_count | limit 20
```
Only fetch raw logs for a SPECIFIC application/pattern identified by the aggregation.

### Rule 2: Use Companion Scripts for Execution

For incident investigation:
```bash
python3 scripts/investigate.py --application api-gateway --time-range 1h --output-file /tmp/report.md
```

For any query, use the compactor to avoid raw SSE in context:
```bash
python3 scripts/query-compact.py \
  --query "source logs | filter $m.severity >= ERROR | limit 100" \
  --output-file /tmp/results.md
```

### Rule 3: Aggregation-First Query Ladder

Follow this order for any investigation:
1. **Scope** (aggregation): `groupby $l.applicationname aggregate count() as errors`
2. **Patterns** (aggregation): `groupby $d.message:string aggregate count() as occurrences`
3. **Timeline** (aggregation): `groupby roundTime($m.timestamp, 5m) aggregate count() as errors`
4. **Details** (raw, targeted): `filter $l.applicationname == 'specific-app' | limit 10`

Never skip to step 4.

### Rule 4: Always Limit Raw Queries

Never run `| limit 50` or higher on raw log queries. Use `| limit 10` maximum.
For larger datasets, use aggregation queries or the query-compact script.

## Using the IBM Cloud CLI

```bash
# Prerequisites (one-time setup)
ibmcloud plugin install logs
ibmcloud login --apikey <your-api-key> -r <region>

# Run a DataPrime query
ibmcloud logs query \
  --query 'source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as errors | orderby -errors | limit 20' \
  --output json

# Background query for large time ranges
ibmcloud logs bgq-create \
  --query 'source logs | filter $m.severity >= ERROR' \
  --output json
```

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides. Use the Domain Routing table above.
- **Use scripts with `--output-file`** to write large results to disk instead of stdout.
- **Write generated configs to files** (alert JSON, Terraform, dashboard JSON, rule groups, etc.) instead of pasting inline.
- **Prefer `--json` output** from scripts — it's structured and agents can extract only needed fields.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Resource Index

### Domain Guides (load on demand via Domain Routing table)
- [query-guide.md](references/query-guide.md) — Full query authoring reference
- [alerting-guide.md](references/alerting-guide.md) — Alerting & monitoring
- [incident-guide.md](references/incident-guide.md) — Incident investigation
- [dashboards-guide.md](references/dashboards-guide.md) — Dashboard design
- [cost-guide.md](references/cost-guide.md) — Cost optimization
- [ingestion-guide.md](references/ingestion-guide.md) — Ingestion pipelines
- [access-control-guide.md](references/access-control-guide.md) — Access control & security
- [api-guide.md](references/api-guide.md) — API reference

### Deep References (21 files)
- [dataprime-commands.md](references/dataprime-commands.md) — Full 30+ command catalog
- [dataprime-functions.md](references/dataprime-functions.md) — Aggregation, string, time, conditional functions
- [lucene-integration.md](references/lucene-integration.md) — Lucene syntax and DataPrime integration
- [query-templates.md](references/query-templates.md) — 25+ templates across 7 categories
- [burn-rate-math.md](references/burn-rate-math.md) — Detailed burn rate formulas
- [component-profiles.md](references/component-profiles.md) — Detection keywords, labels, tiers
- [runbook-templates.md](references/runbook-templates.md) — Per-component runbook templates
- [strategy-matrix.md](references/strategy-matrix.md) — Full metric recommendations
- [heuristic-details.md](references/heuristic-details.md) — Pattern lists, SOPs, escalation
- [investigation-queries.md](references/investigation-queries.md) — All investigation DataPrime queries
- [remediation-assets.md](references/remediation-assets.md) — Alert and dashboard generation
- [dashboard-schema.md](references/dashboard-schema.md) — Full JSON schema
- [widget-reference.md](references/widget-reference.md) — Widget configuration per type
- [e2m-guide.md](references/e2m-guide.md) — Events-to-Metrics configuration
- [tco-policies.md](references/tco-policies.md) — TCO policy details
- [enrichment-types.md](references/enrichment-types.md) — Enrichment types and config
- [log-format.md](references/log-format.md) — Log entry JSON schema
- [parsing-rules.md](references/parsing-rules.md) — Rule types and config
- [access-rules.md](references/access-rules.md) — Access rules API details
- [authentication.md](references/authentication.md) — IAM token exchange
- [endpoints.md](references/endpoints.md) — All endpoints by category

### Assets (10 files)
- [alert-config.json](assets/alert-config.json), [alert-terraform.tf](assets/alert-terraform.tf)
- [incident-dashboard.json](assets/incident-dashboard.json), [service-health-dashboard.json](assets/service-health-dashboard.json)
- [access-rule-template.json](assets/access-rule-template.json)
- [api-endpoints.json](assets/api-endpoints.json)
- [cost-analysis-queries.json](assets/cost-analysis-queries.json)
- [query-templates.json](assets/query-templates.json)
- [sample-logs.json](assets/sample-logs.json)
- [investigation-checklist.md](assets/investigation-checklist.md)

### Scripts (3 files)
- [validate-query.sh](scripts/validate-query.sh) — Offline DataPrime query validator
- [calculate-burn-rate.sh](scripts/calculate-burn-rate.sh) — Burn rate table calculator
- [send-test-logs.sh](scripts/send-test-logs.sh) — Ingestion test script

### Companion Scripts (in project root)
- [Query Compactor](../../scripts/query-compact.py) — SSE parsing and result compaction
- [Investigation Script](../../scripts/investigate.py) — Full incident investigation pipeline

> **Windows note:** Bash scripts require bash (available via [Git for Windows](https://gitforwindows.org/) or WSL). Python scripts require Python 3.9+ and `pip install requests`.
