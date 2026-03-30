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
  version: "0.11.0" # x-release-please-version
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

*Either `LOGS_SERVICE_URL` or both `LOGS_INSTANCE_ID` + `LOGS_REGION` must be set.

### Authentication

- **CLI**: Run `ibmcloud login --apikey $LOGS_API_KEY -r $LOGS_REGION`, then pass `--service-url $LOGS_SERVICE_URL` to commands.
- **curl**: Exchange API key for bearer token via `https://iam.cloud.ibm.com/identity/token`, then use `Authorization: Bearer $TOKEN`.

## 5-Step Workflow for Task Execution

### Step 1 — Understand Intent
Determine: What is the user trying to accomplish? (create, troubleshoot, configure, optimize). Ask ONE focused question if critical information is missing.

### Step 2 — Gather Context
Collect relevant information: alert type/query, dashboard widgets, log samples, TCO policies, notification channels, time ranges.

### Step 3 — Provide Targeted Guidance
For creation: recommend configuration, guide setup, explain best practices. For troubleshooting: verify TCO policy first, validate syntax, check common issues, test with sample data.

### Step 4 — Present Solution with Examples
Provide: step-by-step instructions, working configurations, sample queries, expected behavior, validation steps.

### Step 5 — Offer Follow-up Guidance
Suggest: optimization tips, additional monitoring, best practices, related features.

## Domain Routing

Load the relevant guide from `references/` based on the user's task. **Do not load guides eagerly** — only when the task requires domain-specific detail.

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
```
source logs | filter <condition> | groupby <field> aggregate <func> | orderby <expr> | limit <n>
```

### Field Access Prefixes
| Prefix | Layer | Examples |
|--------|-------|----------|
| `$d.` | User Data (default) | `$d.status_code`, `$d.message`, `status_code` |
| `$l.` | Labels | `$l.applicationname`, `$l.subsystemname` |
| `$m.` | Metadata | `$m.severity`, `$m.timestamp` |

**Valid Labels**: `applicationname`, `subsystemname`, `computername`, `ipaddress`, `threadid`, `processid`, `classname`, `methodname`, `category`

**Valid Metadata**: `severity`, `timestamp`, `priority`

**Severity Levels**: `VERBOSE (0) < DEBUG (1) < INFO (2) < WARNING (3) < ERROR (4) < CRITICAL (5)`

### Common Mistakes to Avoid

1. **Use `&&` not `AND`**, `||` not `OR`
2. **Use `==` not `=`** for comparison
3. **Use single quotes** for strings: `'myapp'` not `"myapp"`
4. **`~~` is NOT supported** — Use `matches()` or `contains()`
5. **Cast mixed-type fields**: `$d.message:string.contains('error')`
6. **`$l.namespace` does not exist** — Use `$l.applicationname`
7. **`$m.level` does not exist** — Use `$m.severity`
8. **Use `orderby` not `sort`**

### Essential Commands
`source`, `filter` (aliases: `f`, `where`), `groupby`, `aggregate` (`agg`), `orderby` (`sortby`), `limit` (`l`), `create` (`add`), `extract`, `choose`, `distinct`, `countby`, `lucene`, `find` (`text`)

**Key Functions**: `count()`, `sum()`, `avg()`, `min()`, `max()`, `percentile()`, `contains()`, `startsWith()`, `matches()`, `now()`, `roundTime()`, `if()`, `coalesce()`

**Full reference**: [dataprime-commands.md](references/dataprime-commands.md), [dataprime-functions.md](references/dataprime-functions.md)

## Top Query Patterns

**Error Hotspots** (start here):
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname, $l.subsystemname aggregate count() as error_count
| orderby -error_count | limit 20
```

**Error Timeline**:
```
source logs | filter $m.severity >= ERROR
| groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as errors
| orderby time_bucket
```

**More patterns**: [query-templates.md](references/query-templates.md) — 25+ templates across 7 categories

## CRITICAL: Query Execution Strategy

**Rule 1**: Always use aggregation before raw logs. Never fetch raw logs without first identifying specific patterns via aggregation.

**Rule 2**: Use companion scripts:
- Investigation: `python3 scripts/investigate.py --application api-gateway --time-range 1h`
- Query compaction: `python3 scripts/query-compact.py --query "..." --output-file /tmp/results.md`

**Rule 3**: Aggregation-first ladder: Scope → Patterns → Timeline → Details (never skip to details)

**Rule 4**: Limit raw queries to `| limit 10` maximum.

## TCO Policy Impact (CRITICAL for Alerts & Dashboards)

### Data Pipelines

| Pipeline | Priority | Alerts | Dashboards | Cost |
|----------|----------|--------|------------|------|
| **Priority insights** | High | ✅ Yes | ✅ Yes | Highest |
| **Analyze & alert** | Medium | ✅ Yes | ✅ Yes | Medium |
| **Store & search** | Low | ❌ No | ❌ No | Lowest |

### ⚠️ CRITICAL RULES

**Alerts ONLY trigger on High and Medium priority logs.** If logs are routed to Low priority (Store & search), alerts will NOT work. This is the #1 cause of "alert not triggering" issues (80% of cases).

**Dashboards ONLY show High and Medium priority logs.** If logs are routed to Low priority, dashboards will NOT display them.

### Quick TCO Check

When troubleshooting alerts or dashboards:
1. Go to Explore Logs
2. Search for logs that should trigger alert/appear in dashboard
3. Check pipeline: Priority insights (High) or Analyze & alert (Medium) → ✅ Works | Store & search (Low) → ❌ Doesn't work

### TCO Policy Structure

Policies route logs based on: Application name, Subsystem name, Severity level. Policies are evaluated in order (priority 1, 2, 3...). First matching policy wins.

**Example for Alerts**:
```
Policy 1: payment-service, ERROR/CRITICAL → Priority insights (High)
Policy 2: api-gateway, WARNING/ERROR/CRITICAL → Analyze & alert (Medium)
Policy 3: *, DEBUG/VERBOSE → Store & search (Low)
```

**Full TCO guide**: [cost-guide.md](references/cost-guide.md), [tco-policies.md](references/tco-policies.md)

## Extended Topics

**Alert Types** — 7 types (Standard, Ratio, New Value, Unique Count, Time Relative, Metric, Flow). See [alerting-guide.md](references/alerting-guide.md)

**Dashboard Widgets** — 6 types (Line Chart, Bar Chart, Pie Chart, Data Table, Gauge, Markdown). See [dashboards-guide.md](references/dashboards-guide.md)

**Parsing Rules** — 8 rule types for log transformation. See [parsing-rules.md](references/parsing-rules.md)

**CLI Troubleshooting** — Empty query results diagnosis. See [query-guide.md](references/query-guide.md#cli-troubleshooting)

## Resource Index

### Domain Guides (8 files)
[query-guide.md](references/query-guide.md), [alerting-guide.md](references/alerting-guide.md), [incident-guide.md](references/incident-guide.md), [dashboards-guide.md](references/dashboards-guide.md), [cost-guide.md](references/cost-guide.md), [ingestion-guide.md](references/ingestion-guide.md), [access-control-guide.md](references/access-control-guide.md), [api-guide.md](references/api-guide.md)

### Deep References (21 files)
[dataprime-commands.md](references/dataprime-commands.md), [dataprime-functions.md](references/dataprime-functions.md), [lucene-integration.md](references/lucene-integration.md), [query-templates.md](references/query-templates.md), [burn-rate-math.md](references/burn-rate-math.md), [component-profiles.md](references/component-profiles.md), [runbook-templates.md](references/runbook-templates.md), [strategy-matrix.md](references/strategy-matrix.md), [heuristic-details.md](references/heuristic-details.md), [investigation-queries.md](references/investigation-queries.md), [remediation-assets.md](references/remediation-assets.md), [dashboard-schema.md](references/dashboard-schema.md), [widget-reference.md](references/widget-reference.md), [e2m-guide.md](references/e2m-guide.md), [tco-policies.md](references/tco-policies.md), [enrichment-types.md](references/enrichment-types.md), [log-format.md](references/log-format.md), [parsing-rules.md](references/parsing-rules.md), [access-rules.md](references/access-rules.md), [authentication.md](references/authentication.md), [endpoints.md](references/endpoints.md)

### Assets (10 files)
[alert-config.json](assets/alert-config.json), [alert-terraform.tf](assets/alert-terraform.tf), [incident-dashboard.json](assets/incident-dashboard.json), [service-health-dashboard.json](assets/service-health-dashboard.json), [access-rule-template.json](assets/access-rule-template.json), [api-endpoints.json](assets/api-endpoints.json), [cost-analysis-queries.json](assets/cost-analysis-queries.json), [query-templates.json](assets/query-templates.json), [sample-logs.json](assets/sample-logs.json), [investigation-checklist.md](assets/investigation-checklist.md)

### Scripts (3 files)
[validate-query.sh](scripts/validate-query.sh), [calculate-burn-rate.sh](scripts/calculate-burn-rate.sh), [send-test-logs.sh](scripts/send-test-logs.sh)

### Companion Scripts
[Query Compactor](../../scripts/query-compact.py), [Investigation Script](../../scripts/investigate.py)

> **Windows note:** Bash scripts require bash (Git for Windows or WSL). Python scripts require Python 3.9+ and `pip install requests`.
