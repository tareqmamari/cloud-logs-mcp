# Investigation Queries Reference

All DataPrime queries used by the `smart_investigate` tool across its three
investigation modes. Queries are executed against the IBM Cloud Logs
`/v1/query` endpoint with `syntax: dataprime`.

For DataPrime syntax, commands, and functions, see [IBM Cloud Logs Query Skill](../../ibm-cloud-logs-query/SKILL.md).

## Global Mode Queries

### global-error-rate

**Purpose:** Calculate error rate per application across the entire system.

```dataprime
source logs
| filter $m.severity >= ERROR
| groupby $l.applicationname aggregate count() as error_count
| sortby -error_count
| limit 20
```

**Tier:** archive
**Analysis:** Applications with `error_count > 10` are flagged as findings.
Severity is categorized by count thresholds using `categorizeSeverityByCount()`.

### global-error-timeline

**Purpose:** Error distribution over time to detect spikes.

```dataprime
source logs
| filter $m.severity >= WARNING
| groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as errors
| sortby time_bucket
```

**Tier:** archive
**Analysis:**
1. Compute average error count across all time buckets.
2. Flag any bucket where `errors > average * 3` AND `errors > 10` as a spike.
3. Spike findings are HIGH severity with 85% confidence.
4. Evidence includes the average rate and the spike value.

### global-critical-errors

**Purpose:** Retrieve raw CRITICAL severity events for pattern analysis.

```dataprime
source logs
| filter $m.severity == CRITICAL
| limit 50
```

**Tier:** archive
**Analysis:**
1. Normalize messages to patterns (strip variable parts).
2. Group by normalized pattern.
3. Patterns with >= 3 occurrences generate CRITICAL findings with 95% confidence.

## Component Mode Queries

All component mode queries filter on `$l.applicationname == '<service>'`
where `<service>` is the `application` parameter value.

### component-errors

**Purpose:** Retrieve all errors from the target service.

```dataprime
source logs
| filter $l.applicationname == '<service>' && $m.severity >= ERROR
| limit 200
```

**Tier:** archive (TCO-aware; uses session tier recommendation for the application)
**Analysis:** Raw events are passed to heuristic engine for pattern matching.

### component-error-patterns

**Purpose:** Group errors by message to find recurring patterns.

```dataprime
source logs
| filter $l.applicationname == '<service>' && $m.severity >= ERROR
| groupby $d.message aggregate count() as occurrences
| sortby -occurrences
| limit 20
```

**Tier:** archive
**Analysis:**
1. Top 5 patterns are examined.
2. Patterns with `occurrences > 5` generate findings.
3. Severity scales with occurrence count.
4. Confidence: 90%.

### component-subsystems

**Purpose:** Error distribution by subsystem within the target service.

```dataprime
source logs
| filter $l.applicationname == '<service>' && $m.severity >= WARNING
| groupby $l.subsystemname aggregate count() as errors
| sortby -errors
```

**Tier:** archive
**Analysis:** Subsystems with `errors > 20` are flagged. Finding service
is reported as `<application>/<subsystem>`.

### component-dependencies

**Purpose:** Identify downstream failures and connectivity issues.

```dataprime
source logs
| filter $l.applicationname == '<service>'
  && ($d.message.contains('connection')
      || $d.message.contains('timeout')
      || $d.message.contains('refused'))
| limit 100
```

**Tier:** archive
**Analysis:** Scans messages for 7 dependency failure patterns:

| Pattern               | Description                          |
|-----------------------|--------------------------------------|
| `connection refused`  | Network/service connectivity failure |
| `timeout`             | Downstream service not responding    |
| `econnreset`          | Connection reset by peer             |
| `etimedout`           | Connection timed out                 |
| `pool exhausted`      | Connection pool exhaustion           |
| `deadlock`            | Database deadlock detected           |
| `too many connections`| Connection limit exceeded            |

Patterns with >= 3 occurrences generate HIGH severity dependency findings.

## Flow Mode Queries

### flow-by-trace

**Purpose:** Trace a request flow by trace_id across service boundaries.

```dataprime
source logs
| filter $d.trace_id == '<trace_id>'
| sortby $m.timestamp asc
| limit 500
```

**Tier:** archive
**Analysis:**
1. Build ordered service traversal list.
2. Identify the first event with severity >= 5 (ERROR).
3. Report which service the request failed at.
4. Include full traversal path as evidence (e.g., `svc-a -> svc-b -> svc-c`).

### flow-by-correlation

**Purpose:** Trace a request flow by correlation_id.

```dataprime
source logs
| filter $d.correlation_id == '<correlation_id>'
| sortby $m.timestamp asc
| limit 500
```

**Tier:** archive
**Analysis:** Same as `flow-by-trace`.

## Heuristic Follow-up Queries

When heuristics match, they may suggest follow-up queries. These are not
executed automatically but presented as suggested next actions.

### Timeout Heuristic -- Latency Analysis

```dataprime
source logs
| filter $l.applicationname == '<service>'
| filter $d.duration_ms.exists()
| calculate
    avg($d.duration_ms) as avg_latency,
    percentile($d.duration_ms, 95) as p95_latency,
    percentile($d.duration_ms, 99) as p99_latency
| limit 1
```

### Database Heuristic -- Slow Query Analysis

```dataprime
source logs
| filter $l.applicationname == '<service>' && $d.sql.exists()
| groupby $d.sql
| calculate
    avg($d.exec_ms) as avg_time,
    max($d.exec_ms) as max_time,
    count() as query_count
| sortby -avg_time
| limit 10
```

### Auth Heuristic -- Failure Analysis

```dataprime
source logs
| filter $l.applicationname == '<service>'
| filter $d.message.contains('401') || $d.message.contains('403') || $d.message.contains('auth')
| groupby $d.user_id, $d.endpoint
| calculate count() as failures
| sortby -failures
| limit 20
```

### Global/Flow Drill-down

After global or flow mode identifies an affected service, the suggested
drill-down query is:

```dataprime
source logs
| filter $l.applicationname == '<service>' && $m.severity >= ERROR
| limit 100
```
