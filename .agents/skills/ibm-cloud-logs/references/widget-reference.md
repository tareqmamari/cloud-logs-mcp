# Widget Reference

Detailed configuration for each IBM Cloud Logs dashboard widget type.

For DataPrime query syntax used within widgets, see [query-guide.md](query-guide.md).

## Widget Structure

Every widget follows this structure inside a row:

```json
{
  "id": { "value": "widget-uuid" },
  "title": "Widget Title",
  "appearance": { "width": 0 },
  "definition": {
    "<widget_type>": { ... }
  }
}
```

- `id.value` must be unique within the dashboard (use UUIDs).
- `appearance.width` of `0` means auto-width (fills available space). Set explicit values to control column layout.
- `definition` contains exactly one key matching the widget type.

## Line Chart

Best for time-series data -- error rates, request counts, latency over time.

```json
{
  "definition": {
    "line_chart": {
      "legend": {
        "is_visible": true,
        "group_by_query": true
      },
      "tooltip": {
        "show_labels": false,
        "type": "all"
      },
      "stacked_line": "unspecified",
      "query_definitions": [
        {
          "id": "query-uuid-1",
          "is_visible": true,
          "name": "Query1",
          "color_scheme": "classic",
          "data_mode_type": "high_unspecified",
          "scale_type": "linear",
          "unit": "unspecified",
          "series_count_limit": "20",
          "resolution": {
            "buckets_presented": 96
          },
          "query": {
            "logs": {
              "dataprime_query": {
                "text": "source logs | filter $m.severity >= 5 | groupby roundTime($m.timestamp, 1m) as time_bucket | aggregate count() as errors"
              },
              "filters": []
            }
          }
        }
      ]
    }
  }
}
```

### Key fields

| Field | Type | Description |
|-------|------|-------------|
| `legend.is_visible` | boolean | Show legend below chart |
| `legend.group_by_query` | boolean | Group legend entries by query |
| `tooltip.type` | string | `"all"` or `"single"` |
| `stacked_line` | string | `"unspecified"`, `"absolute"`, `"relative"` |
| `scale_type` | string | `"linear"` or `"logarithmic"` |
| `resolution.buckets_presented` | integer | Number of time buckets (default: 96) |
| `series_count_limit` | string | Max series to display (default: `"20"`) |

### Typical queries

- Error rate: `groupby roundTime($m.timestamp, 1m) | aggregate count() as errors`
- Latency percentiles: `groupby roundTime($m.timestamp, 1m) | aggregate percentile($d.duration_ms, 95) as p95`
- Request volume: `groupby roundTime($m.timestamp, 5m) | aggregate count() as requests`

## Bar Chart

Best for ranked categorical comparisons.

```json
{
  "definition": {
    "bar_chart": {
      "data_mode_type": "high_unspecified",
      "query": {
        "logs": {
          "dataprime_query": {
            "text": "source logs | filter $m.severity >= 5 | groupby $l.subsystemname | calculate count() as errors | sortby -errors | limit 10"
          },
          "filters": []
        }
      }
    }
  }
}
```

### Key fields

| Field | Type | Description |
|-------|------|-------------|
| `data_mode_type` | string | `"high_unspecified"` (default) |

### Typical queries

- Errors by subsystem: `groupby $l.subsystemname | calculate count() as errors | sortby -errors | limit 10`
- Errors by application: `groupby $l.applicationname | calculate count() as cnt | sortby -cnt | limit 10`
- Status codes distribution: `groupby $d.status_code | calculate count() as cnt | sortby -cnt`

## Pie Chart

Best for proportional breakdowns across a single categorical dimension.

```json
{
  "definition": {
    "pie_chart": {
      "data_mode_type": "high_unspecified",
      "query": {
        "logs": {
          "dataprime_query": {
            "text": "source logs | filter $m.severity >= 5 | groupby $l.applicationname | calculate count() as errors"
          },
          "filters": []
        }
      }
    }
  }
}
```

### Key fields

| Field | Type | Description |
|-------|------|-------------|
| `data_mode_type` | string | `"high_unspecified"` (default) |

### Typical queries

- Errors by service: `groupby $l.applicationname | calculate count() as errors`
- Severity distribution: `groupby $m.severity | calculate count() as cnt`
- Errors by region: `groupby $d.region | calculate count() as cnt`

## Data Table

Best for detailed listings, top-N views, or raw event inspection.

```json
{
  "definition": {
    "data_table": {
      "data_mode_type": "high_unspecified",
      "query": {
        "logs": {
          "dataprime_query": {
            "text": "source logs | filter $m.severity >= 5 | groupby $d.message | calculate count() as occurrences | sortby -occurrences | limit 10"
          },
          "filters": []
        }
      }
    }
  }
}
```

### Key fields

| Field | Type | Description |
|-------|------|-------------|
| `data_mode_type` | string | `"high_unspecified"` (default) |

### Typical queries

- Top error messages: `groupby $d.message | calculate count() as occurrences | sortby -occurrences | limit 10`
- Recent errors: `filter $m.severity >= 5 | sortby -$m.timestamp | limit 20`
- Slow requests: `filter $d.duration_ms > 1000 | sortby -$d.duration_ms | limit 10`

## Gauge

Best for single-value KPIs and threshold indicators.

```json
{
  "definition": {
    "gauge": {
      "data_mode_type": "high_unspecified",
      "query": {
        "logs": {
          "dataprime_query": {
            "text": "source logs | filter $m.severity >= 5 | aggregate count() as total_errors"
          },
          "filters": []
        }
      }
    }
  }
}
```

### Typical queries

- Total error count: `filter $m.severity >= 5 | aggregate count() as total_errors`
- Average latency: `filter $d.duration_ms.exists() | aggregate avg($d.duration_ms) as avg_latency`
- Error rate percentage: `aggregate countif($m.severity >= 5) / count() * 100 as error_rate`

## Markdown

Static content widget for annotations, instructions, or links.

```json
{
  "definition": {
    "markdown": {
      "markdown_text": "## Service Health\nThis dashboard shows RED metrics for production services.\n\n**Escalation:** Page on-call if error rate > 5%."
    }
  }
}
```

### Key fields

| Field | Type | Description |
|-------|------|-------------|
| `markdown_text` | string | Markdown-formatted text content |

## Query Syntax Modes

Widgets support two query syntax modes:

### DataPrime (recommended)

```json
"query": {
  "logs": {
    "dataprime_query": {
      "text": "source logs | filter $m.severity >= 5 | limit 100"
    },
    "filters": []
  }
}
```

### Lucene

```json
"query": {
  "logs": {
    "lucene_query": {
      "value": "severity:>=5"
    },
    "aggregations": [
      { "count": {} }
    ],
    "filters": [],
    "group_bys": [
      { "keypath": ["severity"], "scope": "metadata" }
    ]
  }
}
```

Lucene queries require explicit `aggregations` and `group_bys` arrays. DataPrime handles these inline.

## Required Fields Auto-Fill

The `create_dashboard` tool automatically fills missing required fields:

| Field | Default Value | Applied To |
|-------|---------------|------------|
| `appearance.width` | `0` | All widgets |
| `legend` | `{ "is_visible": true, "group_by_query": true }` | `line_chart` |
| `tooltip` | `{ "show_labels": false, "type": "all" }` | `line_chart` |
| `stacked_line` | `"unspecified"` | `line_chart` |
| `data_mode_type` | `"high_unspecified"` | All chart types, data tables, gauges |
| `is_visible` | `true` | Query definitions |
| `scale_type` | `"linear"` | Query definitions |
| `unit` | `"unspecified"` | Query definitions |
| `resolution.buckets_presented` | `96` | Query definitions |
| `filters` | `[]` | All query blocks |
