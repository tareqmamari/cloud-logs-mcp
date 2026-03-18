# Cost Optimization Guide

> Domain guide for IBM Cloud Logs cost optimization. For inline essentials
> (DataPrime syntax, common mistakes), see [SKILL.md](../SKILL.md).

## Data Tier Model

| Tier | Alias | Query Speed | Cost | Retention | Use Case |
|------|-------|-------------|------|-----------|----------|
| `frequent_search` | Hot / Priority Insights | Fast | Highest | Days to weeks | Production errors, real-time debugging |
| `monitoring` | Warm | Moderate | Medium | Weeks to months | Dashboards, trend analysis, SLO tracking |
| `archive` | Cold / COS | Slow | Lowest | Months to years | Compliance, audit trails, historical forensics |

Key behaviors:
- When **no TCO policies** exist, logs go to **both** tiers.
- `type_high` routes to `frequent_search`. `type_medium` routes to archive only. `type_low` drops logs.
- Archive tier is always available by default.

## TCO Policy Design

### Priority Levels

| Priority | Constant | Routing Behavior | Cost Impact |
|----------|----------|-----------------|-------------|
| High | `type_high` | frequent_search AND archive | Highest |
| Medium | `type_medium` | archive ONLY | Moderate |
| Low | `type_low` | Blocked/dropped | Zero |

### Matching Rules

Rule types: `is` (exact), `is_not` (negation), `starts_with` (prefix), `includes` (substring).
Both application_rule and subsystem_rule must match when specified together (AND logic).

### Policy Evaluation

```
For each incoming log:
  1. Walk policies in order (first match wins)
  2. If disabled, skip
  3. If application_rule set, must match
  4. If subsystem_rule set, must match
  5. Apply the policy's priority
  6. If no match, fall back to default tier
```

### CLI Commands

```bash
ibmcloud logs policies --output json
ibmcloud logs policy-create --prototype @policy.json
ibmcloud logs policy-update --id <id> --prototype @policy.json
ibmcloud logs policy-delete --id <id>
```

## Tier Selection Strategy

### By Log Type

| Log Type | Recommended Tier | Priority | Rationale |
|----------|-----------------|----------|-----------|
| Production errors | `frequent_search` | `type_high` | Need fast query for real-time debugging |
| Application info/warning | `archive` | `type_medium` | Reference only |
| Debug/verbose | Drop | `type_low` | Rarely needed, high volume |
| Security audit | `archive` | `type_medium` | Compliance requirement |
| SLO/SLI source | `frequent_search` | `type_high` | Real-time dashboards and alerts |
| Health check | Drop | `type_low` | Repetitive, low value |

### Cost Reduction Patterns

1. **Drop health checks**: `type_low` policy matching health-check subsystems
2. **Archive debug logs**: Route debug/trace severity to `type_medium`
3. **Tier by environment**: Use `starts_with` to route non-production to archive
4. **Combine with E2M**: Convert high-volume info logs to metrics, then drop raw

## Events-to-Metrics (E2M)

### When to Use E2M

- Logs are high-volume but only aggregated values matter
- You need SLI/SLO metrics from log data
- Dashboard panels only display aggregated statistics

### Metric Types

| Type | Description | Use Case |
|------|-------------|----------|
| `counter` | Counts occurrences | Error counts, request counts |
| `gauge` | Samples numeric values | Queue depth, connections |
| `histogram` | Value distribution buckets | Response times, payload sizes |

### E2M Configuration

```json
{
  "name": "error_count_by_service",
  "description": "Count errors per service for SLO tracking",
  "type": "logs2metrics",
  "logs_query": {
    "lucene": "level:error",
    "severity_filters": ["error", "critical"]
  },
  "metric_labels": [
    {"target_label": "service", "source_field": "applicationName"},
    {"target_label": "component", "source_field": "subsystemName"}
  ],
  "permutations_limit": 30000
}
```

**Permutation control**: Use fewer labels, filter narrowly, avoid high-cardinality fields.

```bash
ibmcloud logs e2m-list --output json
ibmcloud logs e2m-create --prototype @e2m.json
```

## Query Cost Awareness

Before running expensive queries, estimate cost using four factors (each 1-25 points, total 1-100):

- **Time Range**: Short (<=1h) = 5, full day = 15, >7 days = 25
- **Filter Efficiency**: Highly specific = 5, no filters = 25
- **Aggregation Complexity**: None = 5, window/subquery = 22
- **Sorting Cost**: None = 5, multi-key = 20

Total: <=30 = low, <=50 = medium, <=75 = high, >75 = very_high.

## Cost Optimization Checklist

### 1. Review TCO Policy Coverage
- [ ] Run `ibmcloud logs policies --output json`
- [ ] Verify every high-volume application has a policy
- [ ] Confirm debug/verbose logs routed to `type_low` or `type_medium`

### 2. Evaluate Tier Assignments
- [ ] Production error logs in `frequent_search`
- [ ] Non-critical operational logs in `archive`
- [ ] Health checks and debug logs dropped

### 3. Audit E2M Configurations
- [ ] Run `ibmcloud logs e2m-list --output json`
- [ ] Convert high-volume aggregation-only logs to E2M
- [ ] Check permutation limits not exceeded

### 4. Check Data Usage
- [ ] Run `ibmcloud logs data-usage --output json`
- [ ] Compare usage against plan limits

### 5. Optimize Query Patterns
- [ ] Prefer tight time ranges and specific filters
- [ ] Use `ibmcloud logs bgq-create` for queries spanning >24 hours

### 6. Analyze Volume by Source
- [ ] Identify top applications by volume (see [cost-analysis-queries.json](../assets/cost-analysis-queries.json))
- [ ] Review severity distribution

## Deep References

- [TCO Policies Reference](tco-policies.md) -- Policy configuration details
- [E2M Guide](e2m-guide.md) -- Events-to-Metrics best practices
- [Cost Analysis Queries](../assets/cost-analysis-queries.json) -- Ready-to-use volume analysis queries
