# Incident Investigation Guide

> Domain guide for IBM Cloud Logs incident investigation. For inline essentials
> (DataPrime syntax, query patterns), see [SKILL.md](../SKILL.md).

## Investigation Modes

The tool supports three modes. Mode selection is automatic based on parameters.

### Mode Selection Logic

| Parameter supplied       | Mode selected  |
|--------------------------|----------------|
| `trace_id`               | **flow**       |
| `correlation_id`         | **flow**       |
| `application`            | **component**  |
| _(none of the above)_    | **global**     |

Priority: `trace_id` / `correlation_id` takes precedence over `application`.

### Tool Parameters

| Parameter         | Type    | Default | Description                                              |
|-------------------|---------|---------|----------------------------------------------------------|
| `application`     | string  | --      | Target service for component mode                        |
| `trace_id`        | string  | --      | Trace ID for flow mode                                   |
| `correlation_id`  | string  | --      | Correlation ID for flow mode                             |
| `time_range`      | enum    | `1h`    | Investigation window: `15m`, `1h`, `6h`, `24h`           |
| `generate_assets` | boolean | false   | Generate Terraform/JSON for alerts and dashboards        |
| `max_queries`     | integer | 5       | Maximum queries to execute (1-10)                        |

## Global Mode

System-wide health scan. Aggregates errors across all services.

### Initial Queries (3)

1. **global-error-rate** -- Error count per application, severity >= ERROR, top 20
2. **global-error-timeline** -- Error distribution over time using 1-minute buckets
3. **global-critical-errors** -- Raw CRITICAL severity events, up to 50

### Analysis Logic

- **Error rate analysis:** Applications with > 10 errors are flagged.
- **Spike detection:** Any bucket exceeding **3x the average** AND > 10 errors is flagged as a spike with HIGH severity and 85% confidence.
- **Critical error grouping:** Patterns with >= 3 occurrences are flagged as CRITICAL with 95% confidence.

## Component Mode

Deep dive into a single service. Requires `application` parameter.

### Initial Queries (4)

1. **component-errors** -- All errors from the target service (limit 200)
2. **component-error-patterns** -- Errors grouped by message, top 20
3. **component-subsystems** -- Error distribution by subsystem
4. **component-dependencies** -- Downstream failures: `connection`, `timeout`, `refused`

### Analysis Logic

- **Error pattern analysis:** Top 5 recurring patterns with > 5 occurrences are flagged.
- **Dependency detection:** 7 patterns: `connection refused`, `timeout`, `econnreset`, `etimedout`, `pool exhausted`, `deadlock`, `too many connections`.
- **Subsystem analysis:** Subsystems with > 20 errors are flagged.

## Flow Mode

Traces a single request across service boundaries. Requires `trace_id` or `correlation_id`.

### Initial Queries (1-2)

1. **flow-by-trace** -- All events matching `$d.trace_id`, sorted by timestamp
2. **flow-by-correlation** -- All events matching `$d.correlation_id`

### Analysis Logic

- Builds a service-traversal timeline from events
- Identifies the first error event in the request flow
- Reports which service the request failed at and the traversal path

See [investigation-queries.md](investigation-queries.md) for full query text.

## Heuristic Pattern Matching

After queries execute, six pattern matchers scan findings:

| Heuristic      | Trigger patterns | Suggested action |
|----------------|------------------|------------------|
| **timeout**    | `timeout`, `timed out`, `deadline exceeded`, `context deadline`, `504` | Check downstream service health and network latency |
| **memory**     | `out of memory`, `oom`, `heap space`, `memory limit`, `oomkilled` | Check container resource limits and memory trends |
| **database**   | `connection pool`, `too many connections`, `deadlock`, `slow query` | Analyze slow database queries |
| **auth**       | `unauthorized`, `forbidden`, `401`, `403`, `invalid token` | Investigate authentication failures |
| **rate_limit** | `rate limit`, `429`, `too many requests`, `throttled` | Analyze request patterns and rate limits |
| **network**    | `connection refused`, `connection reset`, `dns`, `502`, `503` | Check network connectivity and DNS resolution |

See [heuristic-details.md](heuristic-details.md) for full pattern lists, SOPs, and escalation paths.

## Standard Operating Procedures

| SOP trigger | Key steps | Escalation target |
|-------------|-----------|-------------------|
| Timeout errors | Check downstream health, review network latency, verify connection pools | Platform team (15 min) |
| Memory pressure | Check container limits, review JVM heap, analyze heap dumps | Development team (OOMKill) |
| Database issues | Check pool settings, review slow queries, verify max_connections | DBA team |
| Auth failures | Verify credentials, check IAM policies, review token expiration | Security team (immediate) |
| Rate limiting | Identify request source, review limits, implement backoff | Engineering lead |
| Network issues | Verify DNS, check network policies, verify endpoints, check TLS | Platform/Network team |

## Step-by-Step Investigation Flow

1. **Scope Determination** -- Determine mode from parameters (flow > component > global).
2. **Query Execution** -- Execute initial queries. Check TCO policies for correct tier.
3. **Result Analysis** -- Strategy-specific analysis: error rate, spike detection, pattern grouping.
4. **Heuristic Matching** -- All six matchers run against findings and raw events.
5. **Evidence Synthesis** -- Root cause statement, confidence score, affected services, impact summary.

## Remediation Asset Generation

When `generate_assets: true` and findings exist:
- **Alert configuration** -- Terraform HCL + raw JSON
- **Dashboard configuration** -- 5 widgets + IBM Cloud Logs dashboard JSON

See [remediation-assets.md](remediation-assets.md) for details.

## CRITICAL: Use the Investigation Script

For ANY incident investigation, ALWAYS start with the companion script:

```bash
# Global scan
python3 scripts/investigate.py --time-range 1h --output-file /tmp/report.md

# Component deep-dive
python3 scripts/investigate.py --application api-gateway --time-range 1h --output-file /tmp/report.md

# Request tracing
python3 scripts/investigate.py --trace-id abc123 --output-file /tmp/report.md
```

**Saves 99% of tokens** compared to manual multi-step queries.

## Deep References

- [Investigation Queries](investigation-queries.md) -- All DataPrime queries from all 3 strategies
- [Heuristic Details](heuristic-details.md) -- Full pattern lists, SOPs, and escalation paths
- [Remediation Assets](remediation-assets.md) -- Alert and dashboard generation details
- [Investigation Checklist](../assets/investigation-checklist.md) -- Printable incident checklist
