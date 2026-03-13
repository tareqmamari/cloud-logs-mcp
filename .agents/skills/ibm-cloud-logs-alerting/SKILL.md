---
name: ibm-cloud-logs-alerting
description: >
  Design SRE-grade alerts using RED/USE methodologies, multi-window burn rate
  alerting, and SLO-based monitoring for IBM Cloud Logs. Activate when creating
  alerts, setting up monitoring, configuring SLOs, or reducing alert noise.
  Covers 15 component types with strategy matrices and Terraform generation.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: alerting
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Alerting Skill

## When to Activate

Use this skill when the user:
- Wants to create or improve alerts for IBM Cloud Logs
- Asks about SLO-based alerting or burn rate monitoring
- Needs RED or USE methodology guidance for a specific component type
- Wants to reduce alert noise or improve alert quality
- Asks about multi-window burn rate alerting
- Needs Terraform or JSON configuration for IBM Cloud Logs alerts
- Wants to set up monitoring for a specific component (database, cache, Kubernetes, etc.)
- Asks about severity classification (P1/P2/P3)
- Needs runbook templates for alert response
- Wants to detect component types from log patterns
- Asks about dynamic baselines for seasonal metrics

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

## Methodology Selection

Choose the alerting methodology based on component type:

### RED Method (Request-Driven Services)
Use for components that process requests:
- **Rate** -- requests per second (traffic volume)
- **Errors** -- failed requests as a percentage of total
- **Duration** -- latency distribution (P50/P90/P99)

Applies to: `web_service`, `api_gateway`, `worker`, `microservice`, `monolith`, `serverless`

### USE Method (Resource-Driven Infrastructure)
Use for components that provide resources:
- **Utilization** -- percentage of resource capacity in use
- **Saturation** -- degree of queuing or backpressure
- **Errors** -- operation failures

Applies to: `database`, `cache`, `message_queue`, `load_balancer`, `storage`, `network`, `kubernetes`

### Decision Tree
1. Does the component serve requests directly? --> RED
2. Does the component provide a shared resource (connections, memory, CPU)? --> USE
3. Unknown? --> Default to RED, then refine after reviewing log patterns

If the component type is unknown, the engine inspects the query and use-case text for keywords
(see [Component Profiles](references/component-profiles.md)) and selects automatically.

## Severity Classification

Severity is determined by `ClassifySeverity(isUserFacing, burnRate, componentType)`:

| Level | Name | Action | Criteria |
|-------|------|--------|----------|
| **P1** | Critical | Page on-call immediately | User-facing service AND burn rate >= 6.0x |
| **P1** | Critical | Page on-call immediately | Critical infrastructure (database, message_queue, api_gateway) AND burn rate >= 10.0x |
| **P2** | Warning | Create ticket | Burn rate >= 1.0x OR user-facing service (lower burn rate) |
| **P3** | Info | Monitor/trend | Everything else (burn rate < 1.0x, non-user-facing) |

Key principle: **Only page humans for user-facing symptoms with high burn rates.** Low burn rates
on non-user-facing components create tickets, not pages.

## Multi-Window Burn Rate Alerting

This is the most important alerting pattern for SLO-based monitoring. Based on the Google SRE
Workbook, Chapter 5.

### Core Concepts

**Error Budget** = `1 - SLO target`
- SLO 99.9% --> Error budget = 0.1% (43.2 minutes/month in a 30-day window)
- SLO 99.99% --> Error budget = 0.01% (4.32 minutes/month)

**Burn Rate** = multiple of sustainable error rate that would exactly exhaust the budget over the SLO window.
- 1x burn rate = budget exhausted at end of window (sustainable)
- 14.4x burn rate = budget exhausted in ~2 days instead of 30

### Formula: Error Rate Threshold

```
error_rate_threshold = error_budget * burn_rate
                     = (1 - slo_target) * burn_rate
```

For `CalculateErrorThreshold()` with budget consumption percentage:
```
threshold = (budget_consumption_% / 100) * error_budget * (slo_window_hours / alert_window_hours)
```

### Multi-Window Table (30-Day SLO Window)

| Window | Burn Rate | Budget Consumed | Severity | Alert Type | Action |
|--------|-----------|-----------------|----------|------------|--------|
| 1h | 14.4x | 2% in 1h | P1 | fast_burn | Page on-call |
| 6h | 6.0x | 5% in 6h | P1 | fast_burn | Page on-call |
| 24h | 3.0x | 10% in 24h | P2 | slow_burn | Create ticket |
| 72h | 1.0x | 10% in 72h | P3 | slow_burn | Informational |

### Worked Example: SLO 99.9% (30-day window)

Error budget = 0.1% = 0.001

| Window | Burn Rate | Error Rate Threshold | Meaning |
|--------|-----------|---------------------|---------|
| 1h fast burn | 14.4x | 0.001 * 14.4 = 1.44% | If >1.44% of requests fail over 1h, page |
| 6h fast burn | 6.0x | 0.001 * 6.0 = 0.60% | If >0.60% fail over 6h, page |
| 24h slow burn | 3.0x | 0.001 * 3.0 = 0.30% | If >0.30% fail over 24h, ticket |
| 72h slow burn | 1.0x | 0.001 * 1.0 = 0.10% | If >0.10% fail over 72h, info |

### Multi-Window Confirmation

Each burn rate alert uses two windows to reduce false positives:
- **Fast burn**: long window (1h) + short confirmation window (5m)
- **Slow burn**: long window (24h) + short confirmation window (6h)

Both windows must fire simultaneously. This prevents alerting on brief spikes.

For detailed formulas and more examples, see [Burn Rate Math](references/burn-rate-math.md).

## Component Quick Reference

### web_service (RED)
- **Methodology**: RED
- **Tier**: service
- **Key Metrics**:
  - `request_rate` (counter/rate) -- threshold: dynamic baseline
    ```
    source logs | filter $d.type == 'http_request' | stats count() as requests by bin(1m)
    ```
  - `error_rate` (counter/errors) -- threshold: 1% (0.01)
    ```
    source logs | filter $m.severity >= 5 OR $d.status_code >= 500 | stats count() as errors by bin(1m)
    ```
  - `latency_p99` (histogram/duration) -- threshold: 500ms
    ```
    source logs | filter $d.response_time_ms exists | stats percentile($d.response_time_ms, 99) as p99_latency by bin(5m)
    ```

### api_gateway (RED)
- **Methodology**: RED
- **Tier**: infrastructure
- **Key Metrics**:
  - `upstream_error_rate` (counter/errors) -- threshold: 0.5% (0.005)
    ```
    source logs | filter $d.component == 'api_gateway' AND $d.upstream_status >= 500 | stats count() as errors by bin(1m)
    ```
  - `gateway_latency` (histogram/duration) -- threshold: 100ms
    ```
    source logs | filter $d.component == 'api_gateway' | stats percentile($d.latency_ms, 99) as p99 by bin(1m)
    ```
  - `rate_limit_triggers` (counter/saturation) -- threshold: 100
    ```
    source logs | filter $d.rate_limited == true | stats count() by bin(5m)
    ```

### database (USE)
- **Methodology**: USE
- **Tier**: data
- **Key Metrics**:
  - `connection_utilization` (gauge/utilization) -- threshold: 80%
    ```
    source logs | filter $d.component == 'database' | stats avg($d.connections_active / $d.connections_max * 100) as utilization by bin(1m)
    ```
  - `query_queue_depth` (gauge/saturation) -- threshold: 10 queries
    ```
    source logs | filter $d.component == 'database' | stats max($d.query_queue_length) as queue_depth by bin(1m)
    ```
  - `replication_lag` (gauge/saturation) -- threshold: 5s
    ```
    source logs | filter $d.component == 'database' AND $d.role == 'replica' | stats max($d.replication_lag_seconds) as lag by bin(1m)
    ```
  - `slow_queries` (counter/errors) -- threshold: 10/5min
    ```
    source logs | filter $d.component == 'database' AND $d.query_duration_ms > 1000 | stats count() as slow_queries by bin(5m)
    ```

### cache (USE)
- **Methodology**: USE
- **Tier**: caching
- **Key Metrics**:
  - `memory_utilization` (gauge/utilization) -- threshold: 85%
  - `hit_rate` (gauge/utilization) -- threshold: 90% (alert below)
  - `eviction_rate` (counter/saturation) -- threshold: 100 keys/5min

### message_queue (USE)
- **Methodology**: USE
- **Tier**: messaging
- **Key Metrics**:
  - `queue_depth` (gauge/saturation) -- threshold: 1000 messages
  - `consumer_lag` (gauge/saturation) -- threshold: 10000 messages
  - `dead_letter_queue` (counter/errors) -- threshold: 1 (any DLQ activity)
  - `publish_errors` (counter/errors) -- threshold: 5/min

### worker (RED)
- **Methodology**: RED
- **Tier**: background
- **Key Metrics**:
  - `job_success_rate` (counter/errors) -- threshold: 99%
  - `job_duration` (histogram/duration) -- threshold: 60000ms
  - `retry_rate` (counter/errors) -- threshold: 10/5min

### kubernetes (USE)
- **Methodology**: USE
- **Tier**: platform
- **Key Metrics**:
  - `pod_restarts` (counter/errors) -- threshold: 3 restarts/5min
    ```
    source logs | filter $d.kubernetes exists AND $d.event_type == 'container_restart' | stats count() by $d.kubernetes.pod_name, bin(5m)
    ```
  - `cpu_throttling` (gauge/saturation) -- threshold: 25%
  - `memory_utilization` (gauge/utilization) -- threshold: 90%
  - `pending_pods` (gauge/saturation) -- threshold: 5 pods

For the complete strategy matrix with all metrics, queries, and best practices, see
[Strategy Matrix](references/strategy-matrix.md).

## Dynamic Baselines

Use dynamic baselines for metrics with natural seasonality (traffic patterns, batch jobs, etc.).

| Seasonality Type | Group By | Format | Use Case |
|-----------------|----------|--------|----------|
| `hourly` | `hour_of_day` | `HH` | Traffic that varies by hour |
| `daily` | `day_of_week` | `E` | Patterns that differ weekday vs weekend |
| `weekly` | `week_of_year` | `w` | Monthly/seasonal business cycles |

**Configuration**: Standard deviation multiplier (default: 2.0-3.0). Minimum 30 data points required.

Baseline query pattern:
```
source logs
| filter $d.<metric_field> exists
| extend hour_of_day = formatTimestamp($m.timestamp, 'HH')
| stats
    avg($d.<metric_field>) as baseline_mean,
    stddev($d.<metric_field>) as baseline_stddev,
    count() as sample_count
  by hour_of_day
| filter sample_count >= 30
```

Alert threshold = `baseline_mean + (std_dev_multiplier * baseline_stddev)`

## Alert Output Format

### IBM Cloud Logs Alert Definition JSON

Alert definitions are created via `POST /v1/alert_definitions`. The type-specific config is a top-level key matching the `type` field.

```json
{
  "name": "<alert_name>",
  "description": "<description>",
  "enabled": true,
  "type": "logs_threshold",
  "logs_threshold": {
    "condition_type": "more_than_or_unspecified",
    "logs_filter": {
      "simple_filter": {
        "lucene_query": "<lucene_filter>"
      }
    },
    "rules": [
      {
        "condition": {
          "threshold": 100,
          "time_window": {
            "logs_time_window_specific_value": "minutes_10"
          }
        },
        "override": {
          "priority": "p2"
        }
      }
    ]
  },
  "incidents_settings": {
    "minutes": 10,
    "notify_on": "triggered_only_unspecified"
  }
}
```

**Key points:**
- Do NOT set top-level `priority` when rules have `override.priority` (API rejects the conflict)
- Priority values are lowercase: `p1`, `p2`, `p3`, `p4`, `p5_or_unspecified`
- `logs_filter.simple_filter.lucene_query` uses Lucene syntax (not DataPrime)
- Time windows: `minutes_1`, `minutes_5`, `minutes_10`, `minutes_30`, `hours_1`, `hours_6`, `hours_24`

### Terraform Resource
See [assets/alert-terraform.tf](assets/alert-terraform.tf) for the full template.

The `ibm_logs_alert` resource maps severity strings to Terraform values:
- `critical` --> `CRITICAL`
- `high` --> `ERROR`
- `medium` --> `WARNING`
- `low` --> `INFO`

## Step-by-Step: Creating an Alert

1. **Identify the component type** -- Determine what is being monitored (web service, database,
   Kubernetes, etc.). If unknown, the engine detects it from query keywords.

2. **Select methodology** -- RED for request-driven services, USE for resource-driven infrastructure.
   The strategy matrix auto-selects based on component type.

3. **Choose metrics** -- Select from the recommended metrics for your component type.
   Each metric includes a DataPrime query template, default threshold, and best practices.

4. **Define an SLO** -- Set a target (e.g., 99.9%) and window (e.g., 30 days).
   This enables burn rate alerting instead of static thresholds.

5. **Calculate burn rate thresholds** -- Use the formulas or the
   [burn rate calculator script](scripts/calculate-burn-rate.sh) to compute
   error rate thresholds for each alerting window.

6. **Classify severity** -- Apply `ClassifySeverity()` rules:
   user-facing + high burn rate = P1, medium burn rate = P2, low = P3.

7. **Generate configuration** -- Produce IBM Cloud Logs JSON and/or Terraform HCL.
   Deploy with `ibmcloud logs alert-definition-create --prototype @alert-def.json`, or apply Terraform directly.

8. **Write a runbook** -- Every alert must have a runbook. Use the templates in
   [Runbook Templates](references/runbook-templates.md) as a starting point.

9. **Test the query** -- Use `ibmcloud logs query` to verify the DataPrime query
   returns expected results before activating the alert.

10. **Set up notifications** -- List webhook targets with `ibmcloud logs outgoing-webhooks --output json`,
    then create an alert linking the definition to notifications with `ibmcloud logs alert-create --prototype @alert.json`.

## Best Practices

1. **Alert on symptoms, not causes.** Alert on user-visible impact (error rate, latency),
   not infrastructure metrics (CPU, disk). High CPU does not always cause user impact.

2. **Every alert must be actionable.** If the responder cannot take action, remove the alert.
   Each alert requires: runbook URL, suggested actions, clear severity, escalation path.

3. **Use burn rate alerting over static thresholds.** Burn rates account for error budget
   consumption over time, reducing noise from brief spikes while catching sustained degradation.

4. **Require runbooks.** A runbook must document: initial triage steps, investigation procedures,
   escalation paths, and known failure modes with mitigations.

5. **Classify severity correctly.** Only P1 should page. P2 creates tickets. P3 is informational.
   Over-paging causes alert fatigue and on-call burnout.

6. **Use multi-window confirmation.** Require both a long and short window to fire before alerting.
   This eliminates false positives from transient spikes.

7. **Differentiate client vs server errors.** 4xx errors are client problems; 5xx are server
   problems. Alert on 5xx for service health; track 4xx for API usability.

8. **Use dynamic baselines for traffic alerts.** Static thresholds fail for traffic metrics
   that naturally vary by time of day or day of week.

9. **Label alerts consistently.** Include: methodology, signal, team, service, environment,
   and criticality tier in all alert labels.

10. **Review and tune regularly.** Alerts that never fire may have thresholds too high.
    Alerts that fire constantly have thresholds too low. Target < 5 pages per on-call shift.

## Using the IBM Cloud CLI

Alert definitions and alerts can be managed via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# List existing alert definitions
ibmcloud logs alert-definitions --output json

# Create an alert definition from a JSON file
ibmcloud logs alert-definition-create --prototype @alert-def.json

# List alerts (alert definitions linked to notifications)
ibmcloud logs alerts --output json

# Create an alert linking a definition to webhooks
ibmcloud logs alert-create --prototype @alert.json

# List outgoing webhooks for notification targets
ibmcloud logs outgoing-webhooks --output json

# Test the alert query first
ibmcloud logs query --query 'source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as errors'
```

Use [assets/alert-config.json](assets/alert-config.json) as a starting template. Pass JSON files with `--prototype @filename.json`.

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** The strategy matrix and component profiles are large. Only read them when the user specifies a component type or asks for full metric details.
- **Use the burn rate calculator with `--output-file`** to avoid flooding context:
  ```
  ./scripts/calculate-burn-rate.sh --slo 0.999 --window 30 --json --output-file /tmp/burn-rate.json
  ```
  Then read the summary line and only load the file if the user needs the full table.
- **Write generated Terraform/JSON configs to files**, not inline:
  ```
  # Write alert config to file, tell user the path
  cat > /tmp/alert-config.tf << 'EOF'
  ...
  EOF
  ```
- **Prefer showing 1-2 relevant component profiles** from the strategy matrix rather than loading the entire reference.

## Additional Resources

- [Strategy Matrix](references/strategy-matrix.md) -- Full metric recommendations for all component types
- [Burn Rate Math](references/burn-rate-math.md) -- Detailed formulas with worked examples
- [Runbook Templates](references/runbook-templates.md) -- Per-component runbook templates
- [Component Profiles](references/component-profiles.md) -- Detection keywords, labels, and tier assignments
- [Burn Rate Calculator](scripts/calculate-burn-rate.sh) -- CLI tool for computing burn rate tables (supports `--json`, `--output-file`)
- [Terraform Template](assets/alert-terraform.tf) -- ibm_logs_alert resource template
- [Alert Config JSON](assets/alert-config.json) -- IBM Cloud Logs alert JSON template
- [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md) -- DataPrime syntax, commands, and functions

> **Windows note:** Scripts require bash (available via [Git for Windows](https://gitforwindows.org/) or WSL).

### External References
- Google SRE Handbook -- Chapter 5: Alerting
- Google SRE Handbook -- Chapter 6: Monitoring Distributed Systems
- Google SRE Workbook -- Chapter 5: Alerting on SLOs
- "My Philosophy on Alerting" by Rob Ewaschuk (Google)
- "USE Method" by Brendan Gregg
