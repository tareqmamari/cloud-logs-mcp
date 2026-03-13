---
name: ibm-cloud-logs-cost-optimization
description: >
  Optimize IBM Cloud Logs costs through TCO policies, data tier selection,
  and Events-to-Metrics conversion. Activate when asking about costs,
  retention, storage tiers, or reducing logging expenses.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: cost-optimization
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Cost Optimization Skill

## When to Activate

Use this skill when the user:
- Asks about reducing IBM Cloud Logs costs or storage expenses
- Wants to configure TCO (Total Cost of Ownership) policies
- Needs to choose between frequent_search, monitoring, or archive tiers
- Wants to convert high-volume logs into aggregated metrics (E2M)
- Asks about data retention, log routing, or tier selection
- Wants to understand query cost implications before running queries
- Needs a cost optimization audit or checklist for their logging setup

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

## Data Tier Model

IBM Cloud Logs uses a tiered storage model where cost and query speed are inversely related:

| Tier | Alias | Query Speed | Cost | Retention | Use Case |
|------|-------|-------------|------|-----------|----------|
| `frequent_search` | Hot / Priority Insights | Fast | Highest | Days to weeks | Production errors, real-time debugging, active investigations |
| `monitoring` | Warm | Moderate | Medium | Weeks to months | Operational dashboards, trend analysis, SLO tracking |
| `archive` | Cold / COS | Slow | Lowest | Months to years | Compliance, audit trails, historical forensics |

Key behaviors:
- When **no TCO policies** are configured, logs go to **both** tiers (frequent_search and archive). The system defaults to querying `frequent_search` for faster results.
- When policies exist, they determine which tier receives each log. Only `type_high` priority routes logs to `frequent_search`. `type_medium` routes to archive only. `type_low` blocks/drops logs entirely.
- Archive tier is always available by default (`HasArchive: true`).
- The `frequent_search` tier is only confirmed available when policies explicitly route logs there, or when no policies exist (logs go to both).

## TCO Policy Design

TCO policies control how logs are routed to storage tiers. Policies are evaluated in order -- first match wins.

### Priority Levels

| Priority | Constant | Routing Behavior | Cost Impact |
|----------|----------|-----------------|-------------|
| High | `type_high` | Logs go to Priority Insights (frequent_search) AND archive | Highest -- stored in both tiers |
| Medium | `type_medium` | Logs go to archive ONLY (not frequent_search) | Moderate -- archive storage only |
| Low | `type_low` | Logs are blocked/dropped (not stored) | Zero -- no storage |
| Unspecified | `type_unspecified` | Treated as archive | Moderate |

### Matching Rules

Each policy can match on application name, subsystem name, or both. Both conditions must match when specified together (AND logic).

Rule types for matching:
- `is` -- exact match (e.g., application_rule name "api-gateway" matches only "api-gateway")
- `is_not` -- negation match
- `starts_with` -- prefix match (e.g., name "prod" matches "production-api", "prod-worker")
- `includes` -- substring match

### Policy Evaluation

```
For each incoming log:
  1. Walk policies in order (first match wins)
  2. If policy is disabled (enabled: false), skip it
  3. If application_rule is set, it must match the log's application name
  4. If subsystem_rule is set, it must match the log's subsystem name
  5. If both rules match (or no rules are set), apply the policy's priority
  6. If no policy matches, fall back to default tier
```

### API Endpoints

- `GET /v1/policies` -- List all TCO policies
- `POST /v1/policies` -- Create a new policy
- `GET /v1/policies/{id}` -- Get a specific policy
- `PUT /v1/policies/{id}` -- Update a policy
- `DELETE /v1/policies/{id}` -- Delete a policy (ImpactLevel: critical)

### CLI Commands

```bash
ibmcloud logs policies --output json              # List all policies
ibmcloud logs policy --id <id> --output json       # Get a specific policy
ibmcloud logs policy-create --prototype @policy.json # Create a policy
ibmcloud logs policy-update --id <id> --prototype @policy.json # Update
ibmcloud logs policy-delete --id <id>              # Delete (destructive)
```

## Tier Selection Strategy

Use this decision framework to assign logs to the appropriate tier:

### By Log Type

| Log Type | Recommended Tier | Priority | Rationale |
|----------|-----------------|----------|-----------|
| Production errors (severity: error, critical) | `frequent_search` | `type_high` | Need fast query for real-time debugging |
| Application info/warning logs | `archive` | `type_medium` | Reference only, not real-time critical |
| Debug/verbose logs | Drop | `type_low` | Rarely needed in production, high volume |
| Security audit logs | `archive` | `type_medium` | Compliance requirement, infrequent access |
| Access/request logs | `archive` | `type_medium` | High volume, query only for investigations |
| SLO/SLI source logs | `frequent_search` | `type_high` | Needed for real-time dashboards and alerts |
| Health check logs | Drop | `type_low` | Repetitive, low value, very high volume |

### By Application Role

| Application Pattern | Rule Type | Recommended Tier |
|-------------------|-----------|-----------------|
| `production-*` (customer-facing) | `starts_with` | `frequent_search` for errors, `archive` for info |
| `staging-*`, `dev-*` | `starts_with` | `archive` or drop |
| `batch-worker`, `cron-*` | `is` / `starts_with` | `archive` |
| `api-gateway` + subsystem `auth` | `is` + `is` | `frequent_search` (security-critical) |

### Cost Reduction Patterns

1. **Drop health checks**: Create a `type_low` policy matching health-check subsystems
2. **Archive debug logs**: Route debug/trace severity to `type_medium`
3. **Tier by environment**: Use `starts_with` rules to route non-production to archive
4. **Combine with E2M**: Convert high-volume info logs to metrics, then drop or archive the raw logs

## Events-to-Metrics (E2M)

E2M converts high-volume log events into compact metric aggregations, dramatically reducing storage costs while preserving analytical value.

### When to Use E2M

- Logs are high-volume but only aggregated values matter (counts, averages, percentiles)
- You need SLI/SLO metrics derived from log data
- Dashboard panels only display aggregated statistics, not raw log lines
- You want to reduce storage cost without losing analytical capability

### Metric Types

| Type | Description | Use Case |
|------|-------------|----------|
| `counter` | Counts occurrences of matching events | Error counts, request counts per service |
| `gauge` | Samples numeric values from log fields | Current queue depth, active connections |
| `histogram` | Creates value distribution buckets | Response time distribution, payload sizes |

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

### Permutation Limits

The `permutations_limit` (default: 30000) caps the number of unique label combinations. Each unique combination of label values counts as one permutation. Exceeding the limit causes metric samples to be dropped silently.

To control permutations:
- Use fewer labels (each additional label multiplies permutations)
- Filter logs narrowly before metric extraction
- Avoid high-cardinality fields (user IDs, request IDs) as labels

### E2M Types

- `logs2metrics` -- Convert log events to metrics
- `spans2metrics` -- Convert trace spans to metrics

### API Endpoints and CLI

- API: `GET/POST /v1/events2metrics`, `GET/PUT/DELETE /v1/events2metrics/{id}`

```bash
ibmcloud logs e2m-list --output json                # List all E2M configs
ibmcloud logs e2m --id <id> --output json            # Get a specific E2M
ibmcloud logs e2m-create --prototype @e2m.json        # Create an E2M mapping
ibmcloud logs e2m-update --id <id> --prototype @e2m.json # Update
ibmcloud logs e2m-delete --id <id>                   # Delete
```

## Query Cost Awareness

API operations have varying cost and latency characteristics that you should consider:

### Operation Cost Levels

| Level | Description | Examples |
|-------|-------------|---------|
| `low` | Single simple API call | List policies, list E2M configs |
| `medium` | Multiple calls or moderate computation | Query logs, health checks |
| `high` | Complex queries, large data processing | Background queries, multi-step investigations |

### Query Cost Estimation

Before running expensive queries, estimate cost using four factors (each 1-25 points, total 1-100):

- **Time Range Cost**: Short (<=1h) = 5, full day = 15, >7 days = 25
- **Filter Efficiency**: Highly specific = 5, no filters (full scan) = 25
- **Aggregation Complexity**: None = 5, window/subquery = 22
- **Sorting Cost**: None = 5, multi-key sorting = 20

Total score mapping: <=30 = low, <=50 = medium, <=75 = high, >75 = very_high.

Optimization tips:
- Reduce time range when possible
- Add application/subsystem/severity filters to reduce data scan
- Add LIMIT to aggregation results
- Use `ibmcloud logs bgq-create` for large time ranges with weak filters

## Cost Optimization Checklist

Use this checklist when auditing an IBM Cloud Logs instance for cost savings:

### 1. Review TCO Policy Coverage
- [ ] Run `ibmcloud logs policies --output json` to see all policies
- [ ] Verify every high-volume application has a policy (not falling through to default)
- [ ] Confirm debug/verbose logs are routed to `type_low` (drop) or `type_medium` (archive)
- [ ] Check that disabled policies are intentional, not accidental

### 2. Evaluate Tier Assignments
- [ ] Production error logs in `frequent_search` (type_high)
- [ ] Non-critical operational logs in `archive` (type_medium)
- [ ] Health checks, heartbeats, and debug logs dropped (type_low)

### 3. Audit E2M Configurations
- [ ] Run `ibmcloud logs e2m-list --output json` to review active conversions
- [ ] Convert high-volume logs that only need aggregates into E2M metrics
- [ ] Check permutation limits are not being exceeded (monitor for silent drops)
- [ ] Remove unused E2M mappings to reduce metric cardinality

### 4. Check Data Usage
- [ ] Run `ibmcloud logs data-usage --output json` to get current consumption metrics
- [ ] Compare actual usage against plan limits

### 5. Optimize Query Patterns
- [ ] Estimate query cost before running expensive queries (see Query Cost Estimation above)
- [ ] Prefer tight time ranges and specific filters
- [ ] Use `ibmcloud logs bgq-create` for queries spanning >24 hours
- [ ] Add application and subsystem filters to every query

### 6. Analyze Volume by Source
- [ ] Identify top applications by log volume (see assets/cost-analysis-queries.json)
- [ ] Identify top subsystems generating the most data
- [ ] Review severity distribution -- high ratios of debug/info may indicate optimization opportunities

## Using the IBM Cloud CLI

TCO policies, E2M configurations, and data usage can be managed via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# List current TCO policies
ibmcloud logs policies --output json

# Create a TCO policy from a JSON file
ibmcloud logs policy-create --prototype @tco-policy.json

# List Events-to-Metrics configurations
ibmcloud logs e2m-list --output json

# Create an E2M configuration
ibmcloud logs e2m-create --prototype @e2m-config.json

# Check data usage
ibmcloud logs data-usage --output json

# Analyze volume by severity (query)
ibmcloud logs query \
  --query 'source logs | groupby $m.severity aggregate count() as volume | orderby -volume'
```

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Load cost analysis queries selectively.** Files in `assets/` such as `cost-analysis-queries.json` can be large. Load specific queries by name or category rather than reading the entire file.
- **Write TCO policy configs to files.** When generating TCO policy JSON configurations, write them to a file (e.g., `tco-policy.json`) instead of pasting inline.
- **Write E2M configs to files.** When generating Events-to-Metrics configuration JSON, write it to a file (e.g., `e2m-config.json`) rather than including it in the response.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [TCO Policies Reference](references/tco-policies.md) -- Policy configuration details and examples
- [E2M Guide](references/e2m-guide.md) -- Events-to-Metrics configuration and best practices
- [Cost Analysis Queries](assets/cost-analysis-queries.json) -- Ready-to-use DataPrime queries for volume analysis
- [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md) -- DataPrime syntax, commands, and functions
