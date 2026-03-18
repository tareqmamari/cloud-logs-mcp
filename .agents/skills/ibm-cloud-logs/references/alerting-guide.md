# Alerting & Monitoring Guide

> Domain guide for IBM Cloud Logs alerting. For inline essentials (DataPrime syntax,
> common mistakes), see [SKILL.md](../SKILL.md).

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
(see [Component Profiles](component-profiles.md)) and selects automatically.

## Severity Classification

Severity is determined by `ClassifySeverity(isUserFacing, burnRate, componentType)`:

| Level | Name | Action | Criteria |
|-------|------|--------|----------|
| **P1** | Critical | Page on-call immediately | User-facing service AND burn rate >= 6.0x |
| **P1** | Critical | Page on-call immediately | Critical infrastructure (database, message_queue, api_gateway) AND burn rate >= 10.0x |
| **P2** | Warning | Create ticket | Burn rate >= 1.0x OR user-facing service (lower burn rate) |
| **P3** | Info | Monitor/trend | Everything else (burn rate < 1.0x, non-user-facing) |

Key principle: **Only page humans for user-facing symptoms with high burn rates.**

## Multi-Window Burn Rate Alerting

Based on the Google SRE Workbook, Chapter 5.

### Core Concepts

**Error Budget** = `1 - SLO target`
- SLO 99.9% --> Error budget = 0.1% (43.2 minutes/month in a 30-day window)
- SLO 99.99% --> Error budget = 0.01% (4.32 minutes/month)

**Burn Rate** = multiple of sustainable error rate that would exactly exhaust the budget over the SLO window.

### Formula: Error Rate Threshold

```
error_rate_threshold = error_budget * burn_rate
                     = (1 - slo_target) * burn_rate
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

For detailed formulas and more examples, see [Burn Rate Math](burn-rate-math.md).

## Component Quick Reference

### web_service (RED)
- **Key Metrics**: `request_rate` (dynamic baseline), `error_rate` (1%), `latency_p99` (500ms)

### api_gateway (RED)
- **Key Metrics**: `upstream_error_rate` (0.5%), `gateway_latency` (100ms), `rate_limit_triggers` (100)

### database (USE)
- **Key Metrics**: `connection_utilization` (80%), `query_queue_depth` (10), `replication_lag` (5s), `slow_queries` (10/5min)

### cache (USE)
- **Key Metrics**: `memory_utilization` (85%), `hit_rate` (90%, alert below), `eviction_rate` (100 keys/5min)

### message_queue (USE)
- **Key Metrics**: `queue_depth` (1000), `consumer_lag` (10000), `dead_letter_queue` (any activity), `publish_errors` (5/min)

### worker (RED)
- **Key Metrics**: `job_success_rate` (99%), `job_duration` (60000ms), `retry_rate` (10/5min)

### kubernetes (USE)
- **Key Metrics**: `pod_restarts` (3/5min), `cpu_throttling` (25%), `memory_utilization` (90%), `pending_pods` (5)

For complete strategy matrix with all metrics and queries, see [Strategy Matrix](strategy-matrix.md).

## Dynamic Baselines

| Seasonality Type | Group By | Format | Use Case |
|-----------------|----------|--------|----------|
| `hourly` | `hour_of_day` | `HH` | Traffic that varies by hour |
| `daily` | `day_of_week` | `E` | Patterns that differ weekday vs weekend |
| `weekly` | `week_of_year` | `w` | Monthly/seasonal business cycles |

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
See [assets/alert-terraform.tf](../assets/alert-terraform.tf) for the full template.

## Step-by-Step: Creating an Alert

1. **Identify the component type** -- Determine what is being monitored.
2. **Select methodology** -- RED for request-driven, USE for resource-driven.
3. **Choose metrics** -- Select from recommended metrics for your component type.
4. **Define an SLO** -- Set a target (e.g., 99.9%) and window (e.g., 30 days).
5. **Calculate burn rate thresholds** -- Use formulas or [calculate-burn-rate.sh](../scripts/calculate-burn-rate.sh).
6. **Classify severity** -- Apply `ClassifySeverity()` rules.
7. **Generate configuration** -- Produce JSON and/or Terraform HCL.
8. **Write a runbook** -- Use [Runbook Templates](runbook-templates.md).
9. **Test the query** -- Use `ibmcloud logs query` to verify.
10. **Set up notifications** -- Link alert definitions to webhook targets.

## Best Practices

1. **Alert on symptoms, not causes.** Alert on user-visible impact (error rate, latency).
2. **Every alert must be actionable.** If the responder cannot take action, remove the alert.
3. **Use burn rate alerting over static thresholds.**
4. **Require runbooks.** Document triage, investigation, escalation, and known failure modes.
5. **Classify severity correctly.** Only P1 should page. P2 creates tickets. P3 is informational.
6. **Use multi-window confirmation.** Require both a long and short window to fire.
7. **Differentiate client vs server errors.** 4xx are client problems; 5xx are server problems.
8. **Use dynamic baselines for traffic alerts.**
9. **Label alerts consistently.** Include methodology, signal, team, service, environment.
10. **Review and tune regularly.** Target < 5 pages per on-call shift.

## CLI Commands

```bash
ibmcloud logs alert-definitions --output json
ibmcloud logs alert-definition-create --prototype @alert-def.json
ibmcloud logs alerts --output json
ibmcloud logs alert-create --prototype @alert.json
ibmcloud logs outgoing-webhooks --output json
```

## Deep References

- [Strategy Matrix](strategy-matrix.md) -- Full metric recommendations for all component types
- [Burn Rate Math](burn-rate-math.md) -- Detailed formulas with worked examples
- [Runbook Templates](runbook-templates.md) -- Per-component runbook templates
- [Component Profiles](component-profiles.md) -- Detection keywords, labels, and tier assignments
- [Alert Config JSON](../assets/alert-config.json) -- IBM Cloud Logs alert JSON template
- [Terraform Template](../assets/alert-terraform.tf) -- ibm_logs_alert resource template

### External References
- Google SRE Handbook -- Chapter 5: Alerting
- Google SRE Handbook -- Chapter 6: Monitoring Distributed Systems
- Google SRE Workbook -- Chapter 5: Alerting on SLOs
- "My Philosophy on Alerting" by Rob Ewaschuk (Google)
- "USE Method" by Brendan Gregg
