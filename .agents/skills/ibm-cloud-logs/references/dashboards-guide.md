# Dashboards Guide

> Domain guide for IBM Cloud Logs dashboards. For inline essentials (DataPrime syntax,
> common mistakes), see [SKILL.md](../SKILL.md).

## Widget Types

| Widget Type   | Key                | Best For                                           |
|---------------|--------------------|-----------------------------------------------------|
| `line_chart`  | `definition.line_chart`  | Time-series trends (error rates, latency over time) |
| `bar_chart`   | `definition.bar_chart`   | Categorical comparisons (errors by subsystem)        |
| `pie_chart`   | `definition.pie_chart`   | Proportional breakdowns (errors by service)          |
| `data_table`  | `definition.data_table`  | Detailed listings (top error messages, raw events)   |
| `gauge`       | `definition.gauge`       | Single-value KPIs (current error rate, SLO status)   |
| `markdown`    | `definition.markdown`    | Static text, links, instructions within dashboards   |

**Rules of thumb:**
- Use `line_chart` when the query contains `roundTime()` or `groupby` on a time bucket.
- Use `pie_chart` when the query groups by a single categorical field.
- Use `bar_chart` when comparing ranked categories.
- Use `data_table` when showing individual records or top-N listings.
- Use `gauge` for single aggregate values.

## Standard Dashboard Patterns

### Incident Analysis Dashboard (5 widgets)

**1. Error Rate Over Time** (`line_chart`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as errors
```

**2. Errors by Service** (`pie_chart`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby $l.applicationname
| calculate count() as errors
```

**3. Latency Distribution** (`line_chart`)
```
source logs
| filter $d.duration_ms.exists() && ($l.applicationname == 'my-service')
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate avg($d.duration_ms) as avg_latency, percentile($d.duration_ms, 95) as p95, percentile($d.duration_ms, 99) as p99
```

**4. Top Error Messages** (`data_table`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby $d.message
| calculate count() as occurrences
| sortby -occurrences | limit 10
```

**5. Errors by Subsystem** (`bar_chart`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby $l.subsystemname
| calculate count() as errors
| sortby -errors | limit 10
```

### Service Health (RED) Dashboard (3 widgets)

**Rate** (`line_chart`) — request count per minute
**Errors** (`line_chart`) — error count per minute
**Duration** (`line_chart`) — avg, p50, p95, p99 latency per minute

## Dashboard JSON Structure

Dashboards are created via `POST /v1/dashboards`:

```json
{
  "name": "Dashboard Name",
  "description": "Purpose of this dashboard",
  "folder_id": null,
  "layout": {
    "sections": [{
      "id": { "value": "section-1" },
      "rows": [{
        "id": { "value": "row-1" },
        "appearance": { "height": 19 },
        "widgets": []
      }]
    }]
  },
  "variables": [],
  "filters": [],
  "time_frame": { "relative": "last_1_hour" }
}
```

- **time_frame.relative** accepts: `last_5_minutes`, `last_15_minutes`, `last_1_hour`, `last_6_hours`, `last_12_hours`, `last_24_hours`, `last_2_days`, `last_3_days`, `last_7_days`, `last_14_days`, `last_30_days`.

For full JSON schema, see [dashboard-schema.md](dashboard-schema.md).

## Dashboard Variables

- Variables are referenced in queries with `$p.` prefix (e.g., `$p.service_name`).
- Use in widget queries: `| filter $l.applicationname == $p.service_name`.
- Variables appear as dropdown selectors in the dashboard UI.

## Folder Organization

| Operation | Method | Path |
|-----------|--------|------|
| List folders | `GET` | `/v1/folders` |
| Create folder | `POST` | `/v1/folders` |
| Move dashboard | `PUT` | `/v1/dashboards/{id}/folder/{folder_id}` |
| Pin dashboard | `PUT` | `/v1/dashboards/{id}/pinned` |
| Set default | `PUT` | `/v1/dashboards/{id}/default` |

## Using the REST API

**Important:** The IBM Cloud Logs CLI plugin does **not** have dashboard commands. Dashboard operations require the REST API directly.

```bash
# Create a dashboard from a JSON file
curl -s -X POST "$LOGS_SERVICE_URL/v1/dashboards" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @dashboard.json

# List all dashboards
curl -s -X GET "$LOGS_SERVICE_URL/v1/dashboards" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Update / Delete
curl -s -X PUT "$LOGS_SERVICE_URL/v1/dashboards/<id>" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d @dashboard.json
curl -s -X DELETE "$LOGS_SERVICE_URL/v1/dashboards/<id>" -H "Authorization: Bearer $TOKEN"
```

## Deep References

- [Widget Reference](widget-reference.md) -- Detailed widget configuration and query patterns
- [Dashboard Schema](dashboard-schema.md) -- Full JSON schema with position grid and time frames
- [Incident Dashboard Example](../assets/incident-dashboard.json) -- 5-widget incident analysis dashboard
- [Service Health Dashboard](../assets/service-health-dashboard.json) -- RED-based service health dashboard
