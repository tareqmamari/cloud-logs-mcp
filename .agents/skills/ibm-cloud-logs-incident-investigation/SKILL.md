---
name: ibm-cloud-logs-incident-investigation
description: >
  Systematic incident investigation using IBM Cloud Logs. Activate when
  debugging production issues, responding to incidents, or asking "why are
  we seeing errors/latency/failures." Provides three investigation modes
  (global scan, component deep-dive, request tracing), heuristic pattern
  matching, and remediation generation.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: incident-response
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Incident Investigation Skill

## When to Activate

Use this skill when the user:

- Is responding to an active incident ("users are seeing 500 errors")
- Is debugging production issues ("why is this service slow?")
- Is performing proactive health scans ("what's the system health?")
- Is tracing a specific request across service boundaries ("why did this request fail?")
- Asks about errors, latency, failures, timeouts, or outages
- Requests root cause analysis for any production anomaly

This skill provides a systematic 5-phase investigation methodology:
scope determination, query execution, result analysis, heuristic pattern
matching, and evidence synthesis. It can be executed manually using the
CLI or curl, following the investigation modes and query patterns below.

For DataPrime syntax, commands, and functions, see [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md).

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

## Investigation Modes

The tool supports three modes. Mode selection is automatic based on
the parameters provided.

### Mode Selection Logic

| Parameter supplied       | Mode selected  |
|--------------------------|----------------|
| `trace_id`               | **flow**       |
| `correlation_id`         | **flow**       |
| `application`            | **component**  |
| _(none of the above)_    | **global**     |

Priority: `trace_id` / `correlation_id` takes precedence over `application`.
If both `application` and `trace_id` are provided, flow mode is used.

### Tool Parameters

| Parameter         | Type    | Default | Description                                              |
|-------------------|---------|---------|----------------------------------------------------------|
| `application`     | string  | --      | Target service for component mode                        |
| `trace_id`        | string  | --      | Trace ID for flow mode                                   |
| `correlation_id`  | string  | --      | Correlation ID for flow mode (alternative to trace_id)   |
| `time_range`      | enum    | `1h`    | Investigation window: `15m`, `1h`, `6h`, `24h`           |
| `generate_assets` | boolean | false   | Generate Terraform/JSON for alerts and dashboards        |
| `max_queries`     | integer | 5       | Maximum queries to execute (1-10)                        |

## Global Mode

System-wide health scan. Aggregates errors across all services to identify
the most impacted components and detect anomalies.

### Initial Queries (3)

1. **global-error-rate** -- Error count per application, severity >= ERROR,
   sorted by volume descending, top 20
2. **global-error-timeline** -- Error distribution over time using 1-minute
   buckets, severity >= WARNING, sorted chronologically
3. **global-critical-errors** -- Raw CRITICAL severity events, up to 50

### Analysis Logic

- **Error rate analysis:** Applications with > 10 errors are flagged.
  Severity is categorized by count thresholds.
- **Spike detection:** Computes the average error rate across all time
  buckets. Any bucket exceeding **3x the average** AND > 10 errors is
  flagged as a spike with HIGH severity and 85% confidence.
- **Critical error grouping:** Critical events are grouped by normalized
  message pattern. Patterns with >= 3 occurrences are flagged as CRITICAL
  with 95% confidence.

### Next Actions

For each affected service discovered, global mode suggests a drill-down
query filtering that service's errors, prompting component-mode
investigation.

See [references/investigation-queries.md](references/investigation-queries.md) for full query text.

## Component Mode

Deep dive into a single service and its dependencies. Requires the
`application` parameter.

### Initial Queries (4)

1. **component-errors** -- All errors from the target service (limit 200)
2. **component-error-patterns** -- Errors grouped by message with
   occurrence counts, top 20
3. **component-subsystems** -- Error distribution by subsystem
   (`$l.subsystemname`), severity >= WARNING
4. **component-dependencies** -- Downstream failures: messages containing
   `connection`, `timeout`, or `refused` (limit 100)

### Analysis Logic

- **Error pattern analysis:** Top 5 recurring patterns with > 5
  occurrences are flagged. Severity scales with count.
- **Dependency detection:** Scans for 7 dependency failure patterns:
  `connection refused`, `timeout`, `econnreset`, `etimedout`,
  `pool exhausted`, `deadlock`, `too many connections`. Patterns with
  >= 3 occurrences generate HIGH severity findings.
- **Subsystem analysis:** Subsystems with > 20 errors are flagged
  with service path `application/subsystem`.

### Next Actions

- If dependency issues are found: suggest checking downstream service health
- Always: suggest checking recent deployment correlations

See [references/investigation-queries.md](references/investigation-queries.md) for full query text.

## Flow Mode

Traces a single request across service boundaries. Requires `trace_id`
or `correlation_id`.

### Initial Queries (1-2)

1. **flow-by-trace** -- All events matching `$d.trace_id`, sorted by
   timestamp ascending, limit 500 (only if `trace_id` provided)
2. **flow-by-correlation** -- All events matching `$d.correlation_id`,
   sorted by timestamp ascending, limit 500 (only if `correlation_id`
   provided)

### Analysis Logic

- Builds a service-traversal timeline from events
- Identifies the first error event (severity >= 5) in the request flow
- Reports which service the request failed at and the traversal path
  (e.g., `api-gateway -> auth-service -> user-db`)
- If no errors found, reports a successful trace with LOW severity

### Next Actions

For each service where an error occurred, suggests a component-mode
drill-down query.

See [references/investigation-queries.md](references/investigation-queries.md) for full query text.

## Heuristic Pattern Matching

After queries execute, a heuristic engine scans findings and raw events
against six pattern matchers. Each matcher triggers on specific text
patterns in finding summaries and suggests targeted follow-up actions.

### Heuristic Summary

| Heuristic      | Trigger patterns                                                                                          | Suggested action                                    |
|----------------|-----------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
| **timeout**    | `timeout`, `timed out`, `deadline exceeded`, `context deadline`, `read timeout`, `write timeout`, `connection timeout`, `request timeout`, `504` | Check downstream service health and network latency |
| **memory**     | `out of memory`, `oom`, `heap space`, `memory limit`, `gc overhead`, `allocation failure`, `java.lang.outofmemory`, `fatal error: runtime: out of memory`, `oomkilled`, `memory pressure`, `memory leak` | Check container resource limits and memory trends   |
| **database**   | `connection pool`, `too many connections`, `deadlock`, `lock wait timeout`, `cannot acquire`, `database`, `sql`, `query failed`, `transaction`, `postgres`, `mysql`, `mongodb`, `redis`, `connection refused`, `max_connections`, `slow query`, `query timeout` | Analyze slow database queries                       |
| **auth**       | `unauthorized`, `forbidden`, `401`, `403`, `authentication failed`, `invalid token`, `expired token`, `access denied`, `permission denied`, `invalid credentials`, `jwt`, `oauth`, `saml` | Investigate authentication failures                 |
| **rate_limit** | `rate limit`, `429`, `too many requests`, `throttled`, `quota exceeded`, `limit exceeded`, `backoff`       | Analyze request patterns and rate limits            |
| **network**    | `connection refused`, `connection reset`, `no route to host`, `network unreachable`, `dns`, `econnrefused`, `econnreset`, `socket`, `tcp`, `ssl`, `tls`, `certificate`, `502`, `503`, `bad gateway`, `service unavailable` | Check network connectivity and DNS resolution       |

See [references/heuristic-details.md](references/heuristic-details.md) for full pattern lists, SOPs, and escalation paths.

## Standard Operating Procedures

Each heuristic provides a standard operating procedure (SOP) that is
included in the investigation report when the corresponding pattern matches.

| SOP trigger                              | Key steps                                                                                  | Escalation target          |
|------------------------------------------|--------------------------------------------------------------------------------------------|----------------------------|
| Timeout errors detected                  | Check downstream health, review network latency, verify connection pools, check resources  | Platform team (15 min)     |
| Memory pressure detected                 | Check container limits, review JVM heap, analyze heap dumps, check for leaks               | Development team (OOMKill) |
| Database connection/query issues         | Check pool settings, review slow queries, verify max_connections, check locks              | DBA team                   |
| Authentication/Authorization failures    | Verify credentials, check IAM policies, review token expiration, check certs               | Security team (immediate)  |
| Rate limiting detected                   | Identify request source, review limits, check retry storms, implement backoff              | Engineering lead           |
| Network connectivity issues              | Verify DNS, check network policies, verify endpoints, check LB health, check TLS certs     | Platform/Network team      |

When no specific pattern matches, a generic SOP is provided:
review error logs, check recent deployments, verify infrastructure health,
review dependent service status.

## Step-by-Step Investigation Flow

A systematic investigation follows a 5-phase pipeline:

1. **Scope Determination** -- Determine mode from parameters (flow >
   component > global). Parse time range. Extract mode-specific context
   (target service, trace ID, correlation ID).

2. **Query Execution** -- Execute initial queries from the selected
   strategy. Before querying, determine the correct tier by checking
   TCO policies (`ibmcloud logs policies --output json`). If the target
   application has `type_medium` priority, use `--tier archive`. If
   `type_high`, use `--tier frequent_search` for speed. If unsure, use
   `--tier archive` as the safe default. See the
   [Query Skill's Choosing the Right Tier](../ibm-cloud-logs-query/SKILL.md#choosing-the-right-tier)
   section for details.

3. **Result Analysis** -- Strategy-specific analysis: error rate
   computation, spike detection, pattern grouping, dependency
   identification, or request flow tracing.

4. **Heuristic Matching** -- All six heuristic matchers run against
   findings and raw events. Matching heuristics contribute follow-up
   actions and SOPs. Actions are deduplicated and sorted by priority.

5. **Evidence Synthesis** -- Root cause statement, confidence score,
   affected services list, impact summary. Optionally generates
   Terraform and JSON configurations for alerts and dashboards
   (when `generate_assets: true`).

## Remediation Asset Generation

When `generate_assets` is set to `true` and findings exist, the tool
generates:

- **Alert configuration** -- Terraform HCL for `ibm_logs_alert` resource
  using logs ratio threshold (error rate > 5% over 5-minute window),
  plus raw IBM Cloud Logs alert JSON
- **Dashboard configuration** -- 5 widgets (error rate over time, errors
  by service, latency distribution, top error messages, errors by
  subsystem), plus IBM Cloud Logs dashboard JSON

See [references/remediation-assets.md](references/remediation-assets.md) for details on generated assets.

## Using the IBM Cloud CLI

Investigation queries can be run directly via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# Phase 1: Error reconnaissance
ibmcloud logs query \
  --query 'source logs | filter $m.severity >= ERROR | groupby $l.applicationname, $l.subsystemname aggregate count() as error_count | orderby -error_count | limit 20' \
  --output json

# Phase 2: Timeline analysis for a specific component
ibmcloud logs query \
  --query 'source logs | filter $l.applicationname == '\''payment-service'\'' && $m.severity >= ERROR | groupby roundTime($m.timestamp, 5m) as bucket aggregate count() as errors | orderby bucket' \
  --output json

# Background query for large historical investigations
ibmcloud logs bgq-create \
  --query 'source logs | filter $m.severity >= ERROR' \
  --output json
```

For full automation, the investigation phases can be scripted by chaining multiple `ibmcloud logs query` calls and applying the heuristic patterns described above.

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Write investigation query plans to files.** Investigations generate many queries. Write the full query plan to a file (e.g., `investigation-plan.md`) instead of listing all queries inline.
- **Load heuristic details on demand.** Only read `references/heuristic-details.md` when a specific heuristic pattern is matched during investigation -- do not load it preemptively.
- **Write remediation configs to files.** Generated alert and dashboard configurations should be written to files (e.g., `alert-config.json`, `remediation-dashboard.json`) rather than pasted into the response.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [Investigation Queries Reference](references/investigation-queries.md) -- all DataPrime queries from all 3 strategies
- [Heuristic Details](references/heuristic-details.md) -- full pattern lists, SOPs, and escalation paths
- [Remediation Assets](references/remediation-assets.md) -- alert and dashboard generation details
- [Investigation Checklist](assets/investigation-checklist.md) -- printable incident investigation checklist
- [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md) -- DataPrime syntax, commands, and functions reference
