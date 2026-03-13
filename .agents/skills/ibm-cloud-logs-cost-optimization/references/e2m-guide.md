# Events-to-Metrics (E2M) Guide

## Overview

Events-to-Metrics (E2M) converts high-volume log events into compact metric time series. Instead of storing every raw log line, E2M extracts numeric aggregations (counts, gauges, histograms) and stores them as metrics. This dramatically reduces storage costs while preserving the analytical value needed for dashboards, alerts, and SLO tracking.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/events2metrics` | List all E2M configurations |
| `POST` | `/v1/events2metrics` | Create a new E2M configuration |
| `GET` | `/v1/events2metrics/{id}` | Get a specific E2M configuration |
| `PUT` | `/v1/events2metrics/{id}` | Replace an E2M configuration |
| `DELETE` | `/v1/events2metrics/{id}` | Delete an E2M configuration |

Note: E2M uses PUT with replace semantics for updates (not PATCH).

## Metric Types

### Counter

Counts the number of log events matching the query filter. No numeric source field is needed.

**Use cases:**
- Error count per service
- Request count per endpoint
- Login attempt count per user

**Example:**
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
  ]
}
```

### Gauge

Samples a numeric value from a log field at each matching event. Represents the most recent value.

**Use cases:**
- Queue depth from worker logs
- Active connection count
- Memory usage reported in application logs

**Example:**
```json
{
  "name": "queue_depth",
  "type": "logs2metrics",
  "logs_query": {
    "lucene": "json.metric_type:queue_depth"
  },
  "metric_fields": [
    {"target_base_metric_name": "queue_depth", "source_field": "json.value"}
  ],
  "metric_labels": [
    {"target_label": "queue_name", "source_field": "json.queue_name"}
  ]
}
```

### Histogram

Creates a distribution of numeric values from a log field. Enables percentile calculations (p50, p95, p99) without storing raw data.

**Use cases:**
- Response time distribution
- Payload size distribution
- Processing duration percentiles

**Example:**
```json
{
  "name": "response_time_histogram",
  "description": "Response time distribution from API logs",
  "type": "logs2metrics",
  "logs_query": {
    "lucene": "json.endpoint:* AND json.response_time:*"
  },
  "metric_fields": [
    {"target_base_metric_name": "response_time_ms", "source_field": "json.response_time"}
  ],
  "metric_labels": [
    {"target_label": "endpoint", "source_field": "json.endpoint"},
    {"target_label": "method", "source_field": "json.method"}
  ]
}
```

## Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name for the E2M configuration |
| `description` | string | No | Description of what this metric measures |
| `type` | string | Yes | `logs2metrics` or `spans2metrics` |
| `logs_query` | object | For logs2metrics | Filter for which logs to convert |
| `logs_query.lucene` | string | No | Lucene query to match log events |
| `logs_query.severity_filters` | string[] | No | Severity levels to include |
| `metric_fields` | object[] | For gauge/histogram | Fields to extract as metric values |
| `metric_labels` | object[] | No | Fields to use as metric labels (dimensions) |
| `permutations_limit` | integer | No | Maximum unique label combinations (default: 30000) |

### metric_labels Structure

```json
{
  "target_label": "service",
  "source_field": "applicationName"
}
```

- `target_label`: The label name in the resulting metric
- `source_field`: The log field to extract the label value from

### metric_fields Structure

```json
{
  "target_base_metric_name": "response_time_ms",
  "source_field": "json.response_time"
}
```

- `target_base_metric_name`: Base name for the generated metric
- `source_field`: The log field containing the numeric value

## Permutation Limits

### What Are Permutations?

A permutation is a unique combination of label values. For example, with labels `service` and `endpoint`:

| service | endpoint | Permutation # |
|---------|----------|--------------|
| api | /users | 1 |
| api | /orders | 2 |
| worker | /process | 3 |

Three unique combinations = 3 permutations.

### Why Limits Matter

The default limit is **30,000** permutations. When exceeded:
- New label combinations are silently dropped
- Existing metrics continue to update
- No error is reported -- data loss is silent

### Controlling Permutations

1. **Reduce label count**: Each additional label multiplies the permutation space. Two labels with 100 values each = 10,000 permutations.
2. **Narrow the log filter**: Use specific Lucene queries to reduce the input event space.
3. **Avoid high-cardinality fields**: Never use user IDs, request IDs, trace IDs, or timestamps as labels.
4. **Use application/subsystem as primary labels**: These have naturally bounded cardinality.

### Estimating Permutations

```
permutations = unique_values(label_1) x unique_values(label_2) x ... x unique_values(label_n)
```

Example: 50 services x 200 endpoints x 5 methods = 50,000 permutations (exceeds default limit).

## E2M Types

| Type | Source | Description |
|------|--------|-------------|
| `logs2metrics` | Log events | Converts log entries matching a Lucene query into metrics |
| `spans2metrics` | Trace spans | Converts distributed tracing spans into metrics |

## Dry-Run Validation

The `create_e2m` tool supports a `dry_run` parameter. When set to `true`:
- Validates the configuration structure
- Checks required fields (`name`, `type`)
- Warns about missing `logs_query` for `logs2metrics` type
- Does not create the E2M configuration
- Returns validation results with errors, warnings, and suggestions

Always use dry-run before creating E2M configurations in production.

## MCP Tool Reference

| Tool | Purpose | APICost | Impact |
|------|---------|---------|--------|
| `list_e2m` | View all E2M configurations | low | none (read-only) |
| `get_e2m` | Get a specific E2M by ID | low | none (read-only) |
| `create_e2m` | Create a new E2M mapping (supports dry_run) | low | medium |
| `replace_e2m` | Replace an E2M (requires get_e2m first) | low | medium |
| `delete_e2m` | Remove an E2M mapping | low | medium |

## Cost Impact

### Before E2M
- 10,000 error logs/minute stored as raw log lines
- Each log ~500 bytes = ~7.2 GB/day of raw storage

### After E2M
- Same 10,000 events/minute converted to counter metrics
- With 50 service x 10 endpoint labels = 500 metric time series
- Storage: ~50 MB/day for metrics

**Estimated savings: ~99% storage reduction** for aggregated use cases.

## Best Practices

1. **Start with counters**: They require no source field and provide immediate value for error/request tracking.
2. **Validate with dry_run first**: Always test E2M configurations before creating them.
3. **Monitor permutation usage**: Review E2M configurations periodically to ensure limits are not being silently exceeded.
4. **Combine with TCO policies**: After creating E2M for a log stream, consider downgrading the raw logs to archive or dropping them entirely.
5. **Use descriptive names**: Name E2M configurations to reflect the metric they produce (e.g., `http_error_count_by_service`, not `e2m_config_1`).
6. **Keep labels minimal**: Start with 2-3 labels and add more only if dashboards require additional dimensions.
