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

## 5-Step Workflow for Task Execution

When helping users with IBM Cloud Logs tasks, follow this structured workflow:

### Step 1 — Understand Intent
Before providing guidance, determine:
- **What is the user trying to accomplish?** (create, troubleshoot, configure, optimize)
- **What type of task?** (alert, dashboard, query, parsing, TCO policy)
- **Is this creation or troubleshooting?**
- **What's the expected vs actual behavior?**

Ask ONE focused question if critical information is missing.

### Step 2 — Gather Context
Collect relevant information based on task type:

**For Alerts:**
- Alert type needed or current configuration
- Query or condition
- Expected vs actual behavior
- TCO policy configuration
- Notification channels

**For Dashboards:**
- Widget types needed
- Data sources and queries
- Expected vs actual visualization
- TCO policy configuration
- Time range settings

**For Queries:**
- Sample logs or log structure
- Desired output or fields
- Current query (if troubleshooting)
- Time range and filters

**For Parsing:**
- Sample log messages (raw format)
- Current parsing rules (if any)
- Expected vs actual parsed output
- Rule order/priority

**For TCO Policies:**
- Current policies (if any)
- Log volume and applications
- Cost concerns or functional issues
- Alert/dashboard requirements

### Step 3 — Provide Targeted Guidance
Based on the task type, provide specific guidance:

**For Creation Tasks:**
- Recommend appropriate configuration
- Guide through setup steps
- Explain best practices
- Provide working examples

**For Troubleshooting:**
- Verify TCO policy first (if applicable)
- Validate syntax and configuration
- Check common issues
- Test with sample data
- Provide systematic diagnosis

### Step 4 — Present Solution with Examples
Always provide:
- Clear step-by-step instructions
- Working configuration examples
- Sample queries or patterns
- Expected behavior after changes
- Validation steps

### Step 5 — Offer Follow-up Guidance
Suggest:
- Optimization tips
- Additional monitoring recommendations
- Best practices for their use case
- Related features or capabilities

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

## TCO Policy Impact (CRITICAL for Alerts & Dashboards)

### Data Pipelines Overview

IBM Cloud Logs has three data pipelines with different costs and capabilities:

| Pipeline | Priority | Cost | Search Speed | Alerts | Dashboards | Retention |
|----------|----------|------|--------------|--------|------------|-----------|
| **Priority insights** | High | Highest | Fastest | ✅ Yes | ✅ Yes | Configurable (default: 7 days) |
| **Analyze & alert** | Medium | Medium | Fast | ✅ Yes | ✅ Yes | 30 days |
| **Store & search** | Low | Lowest | Slower | ❌ No | ❌ No | Long-term (COS) |

### ⚠️ CRITICAL RULES

**Alerts:**
- **Alerts ONLY trigger on High and Medium priority logs**
- If logs are routed to Low priority (Store & search), alerts will NOT work
- This is the #1 cause of "alert not triggering" issues (80% of cases)

**Dashboards:**
- **Dashboards ONLY show High and Medium priority logs**
- If logs are routed to Low priority (Store & search), dashboards will NOT display them
- This is the #1 cause of "dashboard showing no data" issues (80% of cases)

### Quick TCO Check

When troubleshooting alerts or dashboards:
```
1. Go to Explore Logs
2. Search for logs that should trigger alert/appear in dashboard
3. Check which pipeline they're in:
   - Priority insights (High) → ✅ Alerts & Dashboards work
   - Analyze & alert (Medium) → ✅ Alerts & Dashboards work
   - Store & search (Low) → ❌ Alerts & Dashboards DON'T work
```

### TCO Policy Structure

Policies route logs to pipelines based on:
- **Application name**: The source application
- **Subsystem name**: Component or service within the application
- **Severity level**: DEBUG, VERBOSE, INFO, WARNING, ERROR, CRITICAL

**Policy Priority:**
- Policies are evaluated in order (priority 1, 2, 3...)
- First matching policy wins
- Default policy (if no match): All logs go to Priority insights (High)

### Example TCO Policy for Alerts/Dashboards

```
Policy 1 (Priority 1):
  Application: payment-service, auth-service
  Subsystem: *
  Severity: ERROR, CRITICAL
  Pipeline: Priority insights (High)
  Reason: Critical alerts needed

Policy 2 (Priority 2):
  Application: api-gateway, web-app
  Subsystem: *
  Severity: WARNING, ERROR, CRITICAL
  Pipeline: Analyze & alert (Medium)
  Reason: Standard monitoring alerts

Policy 3 (Priority 3):
  Application: *
  Subsystem: *
  Severity: DEBUG, VERBOSE
  Pipeline: Store & search (Low)
  Reason: Archive only, no alerts needed
```

## Alert Types and Configuration

### 7 Alert Types

#### 1. Standard Alert
**Use Case**: Trigger when log count exceeds threshold  
**Example**: Alert when error count > 10 in 5 minutes

**Configuration**:
```
Type: Standard
Query: source logs | filter $m.severity == ERROR
Condition: More than 10 results
Time Window: 5 minutes
Group By: (optional) $l.applicationname
```

#### 2. Ratio Alert
**Use Case**: Trigger when ratio between two queries exceeds threshold  
**Example**: Alert when error rate > 5% of total requests

**Configuration**:
```
Type: Ratio
Query 1 (Numerator): source logs | filter $m.severity == ERROR | count
Query 2 (Denominator): source logs | count
Condition: Ratio > 0.05 (5%)
Time Window: 10 minutes
```

#### 3. New Value Alert
**Use Case**: Trigger when a new unique value appears  
**Example**: Alert on new error message or new failing endpoint

**Configuration**:
```
Type: New Value
Query: source logs | filter $m.severity == ERROR
Key to Track: error_message
Time Window: Look back 24 hours
```

#### 4. Unique Count Alert
**Use Case**: Trigger when unique value count exceeds threshold  
**Example**: Alert when more than 5 different services are failing

**Configuration**:
```
Type: Unique Count
Query: source logs | filter $m.severity == ERROR
Key to Count: $l.applicationname
Condition: More than 5 unique values
Time Window: 15 minutes
```

#### 5. Time Relative Alert
**Use Case**: Trigger when current value differs from historical baseline  
**Example**: Alert when error count is 2x higher than last week

**Configuration**:
```
Type: Time Relative
Query: source logs | filter $m.severity == ERROR | count
Condition: More than 2x compared to same time last week
Time Window: 5 minutes
Comparison Period: 1 week ago
```

#### 6. Metric Alert
**Use Case**: Trigger based on metric values  
**Example**: Alert when CPU usage > 80%

**Configuration**:
```
Type: Metric
Metric Query: avg(cpu_usage)
Condition: > 80
Time Window: 5 minutes
```

#### 7. Flow Alert
**Use Case**: Trigger when log flow stops or resumes  
**Example**: Alert when no logs received from critical service

**Configuration**:
```
Type: Flow
Query: source logs | filter $l.applicationname == 'payment-service'
Condition: No logs for 10 minutes
```

### Alert Query Best Practices

#### 1. Use Specific Filters
```
❌ Bad:
source logs | filter $m.severity == ERROR

✅ Good:
source logs
| filter $m.severity == ERROR
| filter $l.applicationname == 'payment-service'
| filter $l.subsystemname == 'transaction'
```

#### 2. Test Queries First
```
Always test in Explore Logs before creating alert:
1. Run query
2. Verify results match expectations
3. Check field names are correct
4. Validate time range
```

#### 3. Use Appropriate Aggregations
```
For count-based alerts:
source logs | filter condition | count

For grouped counts:
source logs
| filter condition
| groupby $l.applicationname aggregate count() as error_count

For metrics:
source logs
| filter condition
| groupby $l.applicationname aggregate avg(response_time) as avg_time
```

### Alert Troubleshooting Checklist

When alerts aren't triggering:
```
1. ✅ Check TCO Policy (MOST COMMON - 80% of cases)
   - Are logs in High or Medium priority?
   - If Low priority → That's the problem!

2. ✅ Verify Alert Query
   - Run query in Explore Logs
   - Does it return expected results?
   - Check field names are correct

3. ✅ Check Alert Conditions
   - Is threshold appropriate?
   - Is time window correct?
   - Are group-by fields valid?

4. ✅ Verify Alert is Enabled
   - Check alert status
   - Ensure not in maintenance window

5. ✅ Check Notification Channel
   - Is channel configured?
   - Are recipients correct?
   - Check spam/junk folders

6. ✅ Validate Query Syntax
   - Test for syntax errors
   - Verify field prefixes ($l., $m., $d.)
   - Check operators (&&, ||, ==)
```

## Dashboard Widget Types

### 6 Widget Types

#### 1. Line Chart
**Use Case**: Show trends over time  
**Example**: Error rate over the last 24 hours

**Configuration**:
```
Widget Type: Line Chart
Query: source logs | filter $m.severity == ERROR | groupby $m.timestamp aggregate count()
Time Range: Last 24 hours
Aggregation: Count
Group By: timestamp
```

#### 2. Bar Chart
**Use Case**: Compare values across categories  
**Example**: Error count by application

**Configuration**:
```
Widget Type: Bar Chart
Query: source logs | filter $m.severity == ERROR | groupby $l.applicationname aggregate count()
Aggregation: Count
Group By: applicationname
Sort: Descending by count
```

#### 3. Pie Chart
**Use Case**: Show distribution/proportions  
**Example**: Log distribution by severity

**Configuration**:
```
Widget Type: Pie Chart
Query: source logs | groupby $m.severity aggregate count()
Aggregation: Count
Group By: severity
```

#### 4. Data Table
**Use Case**: Display detailed log entries  
**Example**: Recent error logs with details

**Configuration**:
```
Widget Type: Data Table
Query: source logs | filter $m.severity == ERROR | limit 100
Columns: timestamp, applicationname, subsystemname, message
Sort: timestamp descending
```

#### 5. Gauge
**Use Case**: Show single metric value  
**Example**: Current error rate

**Configuration**:
```
Widget Type: Gauge
Query: source logs | filter $m.severity == ERROR | count
Aggregation: Count
Thresholds: 
  - Green: 0-10
  - Yellow: 11-50
  - Red: 51+
```

#### 6. Markdown
**Use Case**: Add text, documentation, or instructions  
**Example**: Dashboard description or runbook links

### Dashboard Layout Best Practices

#### 1. Organize by Priority
```
Top Row: Most critical metrics (gauges, key numbers)
Middle Rows: Trend charts (line charts, bar charts)
Bottom Rows: Detailed data (tables, logs)
```

#### 2. Use Consistent Sizing
```
Full Width (12 cols): Important trend charts
Half Width (6 cols): Comparison charts
Quarter Width (3 cols): Gauges and single metrics
```

#### 3. Group Related Widgets
```
Section 1: Error Monitoring
  - Error count gauge
  - Error trend line chart
  - Error distribution pie chart

Section 2: Performance
  - Response time line chart
  - Throughput gauge
  - Slow requests table
```

## Parsing Rules (CRITICAL: Rule Order)

### ⚠️ Rule Order is MOST IMPORTANT

**70% of parsing issues are caused by incorrect rule order:**
- **Parsing rules execute in ORDER (top to bottom)**
- **First matching rule wins - subsequent rules are skipped**
- **Rule order determines which pattern gets applied**

**Rule Execution Flow**:
```
Log arrives → Rule 1 matches? → YES → Apply Rule 1 → STOP
                              ↓ NO
              Rule 2 matches? → YES → Apply Rule 2 → STOP
                              ↓ NO
              Rule 3 matches? → YES → Apply Rule 3 → STOP
                              ↓ NO
              No rules match → Log remains unparsed
```

### 4 Parsing Rule Types

#### 1. Extract (Regex)
**Use Case**: Extract specific fields using regex patterns  
**Example**: Extract timestamp, level, message from structured logs

**Configuration**:
```
Rule Type: Extract
Source Field: text
Regex Pattern: (?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}) \[(?P<level>\w+)\] (?P<message>.*)
Destination: Create new fields (timestamp, level, message)
```

#### 2. Parse (JSON)
**Use Case**: Parse JSON-formatted log messages  
**Example**: Extract fields from JSON logs

**Configuration**:
```
Rule Type: Parse
Source Field: text
Format: JSON
Destination: Extract all JSON fields
```

#### 3. Replace
**Use Case**: Replace text patterns in logs  
**Example**: Mask sensitive data, normalize formats

**Configuration**:
```
Rule Type: Replace
Source Field: text
Regex Pattern: \b\d{16}\b
Replacement: [REDACTED-CARD]
```

#### 4. Block
**Use Case**: Drop logs matching pattern  
**Example**: Filter out health check logs

**Configuration**:
```
Rule Type: Block
Source Field: text
Regex Pattern: /health
Action: Drop log
```

### Rule Ordering Strategy

```
Order 1-10: Blocking rules (drop unwanted logs)
Order 11-20: Data masking rules (security)
Order 21-30: Specific parsing rules (exact patterns)
Order 31-40: General parsing rules (catch-all patterns)
Order 41+: Fallback rules
```

### Valid Source Fields

- `text` - Main log message field
- `text.log` - Nested log field from JSON logs (common for Kubernetes logs)
- `text.<fieldname>` - Any nested field under text
- `json.<fieldname>` - Custom JSON fields

**Note**: Use lowercase for field names.

### Common Parsing Mistakes

#### Mistake 1: Wrong Rule Order (70% of cases)
```
Wrong Order:
  Rule 1: .* (matches everything) → Catches all logs
  Rule 2: ERROR.* (specific pattern) → Never reached

Right Order:
  Rule 1: ERROR.* (specific pattern) → Matches errors first
  Rule 2: .* (matches everything) → Catches remaining logs
```

#### Mistake 2: Incorrect Regex Pattern
```
Wrong: (?P<level>\w+)
Right: \[(?P<level>\w+)\]

Test with sample log:
Log: [ERROR] Connection failed
Pattern: \[(?P<level>\w+)\] (?P<message>.*)
Result: level=ERROR, message=Connection failed
```

#### Mistake 3: Wrong Source Field
```
Common fields:
- text: Main log message
- text.log: Nested log field (Kubernetes)
- json.message: JSON field

Check your log structure first!
```

### Regex Pattern Library

**Timestamp (ISO 8601)**:
```
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z?)
```

**Log Level**:
```
(?P<level>DEBUG|INFO|WARN|WARNING|ERROR|CRITICAL|FATAL)
```

**IP Address**:
```
(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})
```

**HTTP Status Code**:
```
(?P<status>[1-5]\d{2})
```

**Duration (milliseconds)**:
```
(?P<duration_ms>\d+)ms
```

**UUID**:
```
(?P<uuid>[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})
```

## CLI Query Troubleshooting

### Common CLI Query Issues

When `ibmcloud cloud-logs query` returns empty results:

#### 1. Query Syntax Validation

**Check if query works in UI first:**
- Open IBM Cloud Logs UI
- Navigate to the same instance
- Test the exact same query
- Verify logs are returned in UI

**Common query issues:**
- Lucene syntax requires field names: `applicationName:value` or `subsystemName:value`
- Plain text search may not match: `"value"` (with quotes)
- Wildcard searches: `value*` or `*value*`

**Correct query formats:**
```bash
# Field-specific search (recommended)
--query "applicationName:bm-server-manager" --syntax lucene

# Wildcard search
--query "bm-server-manager*" --syntax lucene

# Full text search with quotes
--query "\"bm-server-manager\"" --syntax lucene

# DataPrime syntax alternative
--query "source logs | filter \$l.applicationname == 'bm-server-manager'" --syntax dataprime
```

#### 2. UTC Timezone Requirement

**All timestamps MUST be in UTC format:**
- Format: `YYYY-MM-DDTHH:MM:SSZ` (Z suffix is required)
- Verify your local time conversion to UTC

**Common time range issues:**
```bash
# ❌ Wrong: Missing Z suffix
--start-date 2026-03-08T00:00:00 --end-date 2026-03-09T11:41:30

# ✅ Correct: With Z suffix
--start-date 2026-03-08T00:00:00Z --end-date 2026-03-09T11:41:30Z

# ❌ Wrong: Local timezone
--start-date 2026-03-08T00:00:00+05:30

# ✅ Correct: UTC timezone
--start-date 2026-03-08T00:00:00Z
```

#### 3. Generate UTC Timestamps

```bash
# Current time in UTC
date -u +%Y-%m-%dT%H:%M:%SZ

# 1 hour ago
date -u -v-1H +%Y-%m-%dT%H:%M:%SZ

# 24 hours ago
date -u -v-24H +%Y-%m-%dT%H:%M:%SZ
```

#### 4. Systematic Troubleshooting Workflow

**Step 1: Validate Query in UI**
```
Ask user: "Can you verify this query returns logs in the IBM Cloud Logs UI?"
- If NO: Query syntax is wrong, help fix the query
- If YES: Continue to Step 2
```

**Step 2: Check Time Range**
```
Verify:
1. Timestamps are in UTC (Z suffix)
2. Format is correct: YYYY-MM-DDTHH:MM:SSZ
3. Time range contains logs (check UI)
4. end-date > start-date

Suggest: Try last 24 hours to test
```

**Step 3: Fix Query Syntax**
```
For Lucene queries:
1. Add field name: applicationName:value or subsystemName:value
2. Use wildcards if needed: value*
3. Quote exact phrases: "exact phrase"
4. Escape special characters

For DataPrime queries:
1. Start with: source logs | filter
2. Use field prefixes: $l. (labels), $m. (metadata), $d. (data)
3. Use correct operators: ==, !=, >=, contains()
```

**Step 4: Verify Service Configuration**
```
Check:
1. Service URL is correct (from instance details)
2. API key is valid (ibmcloud target)
3. User has access to the instance
4. Instance is in correct region
```

**Step 5: Test with Simple Query**
```
Start with simplest possible query:
ibmcloud cloud-logs query \
  --query "*" \
  --syntax lucene \
  --service-url <url> \
  --start-date $(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ) \
  --end-date $(date -u +%Y-%m-%dT%H:%M:%SZ)

If this works, gradually add filters
```

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

## Query Validation Checklist

Before creating alerts, dashboards, or running queries, validate:

```
✅ Query runs successfully in Explore Logs
✅ Query returns expected results
✅ Field names match actual log structure
✅ Filters are not too restrictive
✅ Time window is appropriate
✅ Aggregations are correct
✅ Group-by fields exist in logs
✅ Logs are in High or Medium priority (for alerts/dashboards)
✅ Query performance is acceptable
✅ Query handles missing fields gracefully
```

## Common Query Patterns for Specific Use Cases

### Pattern 1: Error Count by Service
```
source logs
| filter $m.severity == ERROR
| groupby $l.applicationname aggregate count() as error_count
| orderby -error_count
```

### Pattern 2: Slow Requests
```
source logs
| filter response_time > 2000
| filter $l.applicationname == 'api-gateway'
| count
```

### Pattern 3: Failed Authentication
```
source logs
| filter event_type == 'authentication'
| filter status == 'failed'
| groupby user_id aggregate count() as failed_attempts
| filter failed_attempts > 3
```

### Pattern 4: High Error Rate
```
source logs
| filter $l.applicationname == 'web-app'
| groupby $m.severity aggregate count() as log_count
| filter $m.severity == ERROR
```

### Pattern 5: Missing Expected Logs
```
source logs
| filter $l.applicationname == 'health-check'
| filter message ~ 'heartbeat'
| count
```

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
