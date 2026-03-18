# Query Authoring Guide

> Domain guide for IBM Cloud Logs query authoring. For inline essentials (field prefixes,
> common mistakes, top patterns), see [SKILL.md](../SKILL.md).

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

## Additional Query Patterns

### Error Details for Specific App
```
source logs | filter $l.applicationname == '{APP_NAME}' && $m.severity >= ERROR
| choose $m.timestamp, $m.severity, $l.subsystemname, $d.message, $d.error, $d.stack_trace
| orderby $m.timestamp desc | limit 100
```

### Authentication Failures
```
source logs
| filter $d.event_type == 'auth_failure'
   || $d.message:string.contains('authentication failed')
   || $d.message:string.contains('invalid credentials')
| groupby $d.source_ip, $d.username
| aggregate count() as failures
| filter failures > 3 | orderby -failures
```

### Service Health Overview
```
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname
| aggregate count() as error_count
| orderby -error_count | limit 20
```

### Traffic Overview by Service
```
source logs
| groupby $l.applicationname
| aggregate count() as volume, approx_count_distinct($l.subsystemname) as components
| orderby -volume | limit 20
```

### Restart / Crash Detection
```
source logs
| filter $d.message:string.contains('starting')
   || $d.message:string.contains('shutdown')
   || $d.message:string.contains('OOMKilled')
   || $d.message:string.contains('terminated')
| choose $m.timestamp, $l.applicationname, $l.subsystemname, $d.message
| orderby $m.timestamp desc | limit 50
```

## Step-by-Step: Building a Query

1. **Start with source:** `source logs`
2. **Add time filter if needed:** `| last 1h` or `| between @'2024-01-01' and @'2024-01-02'`
3. **Filter to relevant data:** `| filter $l.applicationname == 'myapp' && $m.severity >= WARNING`
4. **Aggregate or select:** `| groupby $l.subsystemname aggregate count() as errors` or `| choose $m.timestamp, $d.message`
5. **Order results:** `| orderby -errors` (prefix with `-` for descending)
6. **Limit output:** `| limit 20`

## Using the IBM Cloud CLI

```bash
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

The CLI handles authentication, SSE parsing, and output formatting automatically.

## Deep References

- [DataPrime Command Reference](dataprime-commands.md) — Full 30+ command catalog with syntax, aliases, examples
- [Query Templates Catalog](query-templates.md) — All 25+ templates across 7 categories
- [DataPrime Functions Reference](dataprime-functions.md) — Aggregation, string, time, conditional, array functions
- [Lucene Integration](lucene-integration.md) — Lucene syntax and combining with DataPrime
- [Query Validation Script](../scripts/validate-query.sh) — Offline DataPrime query validator
