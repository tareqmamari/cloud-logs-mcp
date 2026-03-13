---
name: ibm-cloud-logs-dashboards
description: >
  Design IBM Cloud Logs dashboards with DataPrime-powered widgets. Activate
  when the user wants to create dashboards, add visualizations, or build
  monitoring views. Covers widget types, layout grid, time frames, and
  template variables.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: dashboards
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Dashboards Skill

## When to Activate

Use this skill when the user:
- Wants to create, update, or organize dashboards in IBM Cloud Logs
- Asks about widget types, chart configurations, or visualization options
- Needs to build monitoring views for services, incidents, or operational health
- Wants to add DataPrime-powered widgets to a dashboard layout
- Asks about dashboard folders, pinning, or default dashboard settings
- Needs to design a multi-widget dashboard for incident analysis or RED metrics

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

## Widget Types

IBM Cloud Logs supports six widget types. Choose based on the data pattern:

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
- Use `pie_chart` when the query groups by a single categorical field (service, region).
- Use `bar_chart` when comparing ranked categories (`sortby -field | limit N`).
- Use `data_table` when showing individual records or top-N listings.
- Use `gauge` for single aggregate values (count, average, percentile).

## Standard Dashboard Patterns

### Incident Analysis Dashboard (5 widgets)

These five widgets form a complete incident analysis view. All queries use DataPrime syntax.

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
| aggregate
    avg($d.duration_ms) as avg_latency,
    percentile($d.duration_ms, 95) as p95,
    percentile($d.duration_ms, 99) as p99
```

**4. Top Error Messages** (`data_table`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby $d.message
| calculate count() as occurrences
| sortby -occurrences
| limit 10
```

**5. Errors by Subsystem** (`bar_chart`)
```
source logs
| filter $m.severity >= 5 && ($l.applicationname == 'my-service')
| groupby $l.subsystemname
| calculate count() as errors
| sortby -errors
| limit 10
```

### Service Health (RED) Dashboard (3 widgets)

Rate, Error, Duration -- the RED method for service monitoring.

**Rate** (`line_chart`)
```
source logs
| filter $l.applicationname == 'my-service'
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as request_rate
```

**Errors** (`line_chart`)
```
source logs
| filter $m.severity >= 5 && $l.applicationname == 'my-service'
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as error_count
```

**Duration** (`line_chart`)
```
source logs
| filter $d.duration_ms.exists() && $l.applicationname == 'my-service'
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate
    avg($d.duration_ms) as avg_ms,
    percentile($d.duration_ms, 50) as p50,
    percentile($d.duration_ms, 95) as p95,
    percentile($d.duration_ms, 99) as p99
```

## Dashboard JSON Structure

Dashboards are created via `POST /v1/dashboards` with this structure:

```json
{
  "name": "Dashboard Name",
  "description": "Purpose of this dashboard",
  "folder_id": null,
  "layout": {
    "sections": [
      {
        "id": { "value": "section-1" },
        "rows": [
          {
            "id": { "value": "row-1" },
            "appearance": { "height": 19 },
            "widgets": [ ]
          }
        ]
      }
    ]
  },
  "variables": [],
  "filters": [],
  "time_frame": {
    "relative": "last_1_hour"
  }
}
```

Key points:
- **sections** group rows logically. Most dashboards use one section.
- **rows** are horizontal containers. Each row has a `height` (default 19 units) and holds widgets.
- **widgets** sit inside rows. Each widget has an `id`, `title`, `appearance.width` (0 = auto), and a `definition` block keyed by widget type.
- **time_frame.relative** accepts: `last_5_minutes`, `last_15_minutes`, `last_1_hour`, `last_6_hours`, `last_12_hours`, `last_24_hours`, `last_2_days`, `last_3_days`, `last_7_days`, `last_14_days`, `last_30_days`.

For the complete JSON schema and widget definition format, see [references/dashboard-schema.md](references/dashboard-schema.md).

## Dashboard Variables

Template variables let users filter dashboards dynamically without editing queries.

- Variables are referenced in queries with the `$p.` prefix (e.g., `$p.service_name`).
- Define variables in the dashboard's `variables` array.
- Use them in widget queries: `| filter $l.applicationname == $p.service_name`.
- Variables appear as dropdown selectors in the dashboard UI.
- Multiple variables can be combined: `$p.service_name`, `$p.environment`, `$p.region`.

## Folder Organization

Dashboards can be organized into folders via the `/v1/folders` API.

| Operation | Method | Path |
|-----------|--------|------|
| List folders | `GET` | `/v1/folders` |
| Create folder | `POST` | `/v1/folders` |
| Move dashboard | `PUT` | `/v1/dashboards/{id}/folder/{folder_id}` |
| Pin dashboard | `PUT` | `/v1/dashboards/{id}/pinned` |
| Unpin dashboard | `DELETE` | `/v1/dashboards/{id}/pinned` |
| Set default | `PUT` | `/v1/dashboards/{id}/default` |

Folders support nesting via `parent_id`. Use folders to group dashboards by team, environment, or domain.

## Step-by-Step: Creating a Dashboard

1. **Plan widgets.** Decide which metrics matter. Use the incident analysis or RED patterns above as starting points.
2. **Write queries.** Build each widget query using DataPrime. Test queries individually with `ibmcloud logs query` before embedding in a dashboard. For DataPrime syntax, commands, and functions, see [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md).
3. **Create the dashboard.** Write the full layout JSON to a file and create via the REST API (see below).
4. **Organize.** Move the dashboard to a folder and pin it for quick access.

## Using the REST API

**Important:** The IBM Cloud Logs CLI plugin does **not** have dashboard commands. Dashboard operations require the REST API directly.

```bash
# Authenticate first
TOKEN=$(curl -s -X POST "https://iam.cloud.ibm.com/identity/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=$LOGS_API_KEY" \
  | jq -r .access_token)

# Create a dashboard from a JSON file
curl -s -X POST "$LOGS_SERVICE_URL/v1/dashboards" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @dashboard.json

# List all dashboards
curl -s -X GET "$LOGS_SERVICE_URL/v1/dashboards" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Update an existing dashboard
curl -s -X PUT "$LOGS_SERVICE_URL/v1/dashboards/<dashboard-id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @dashboard.json

# Delete a dashboard
curl -s -X DELETE "$LOGS_SERVICE_URL/v1/dashboards/<dashboard-id>" \
  -H "Authorization: Bearer $TOKEN"

# Test widget queries before creating a dashboard (CLI works for queries)
ibmcloud logs query \
  --query 'source logs | filter $m.severity >= ERROR | groupby roundTime($m.timestamp, 1m) as bucket aggregate count() as errors | orderby bucket'
```

Write the dashboard layout JSON to a file using the structure in [Dashboard JSON Structure](#dashboard-json-structure), then pass it via `curl -d @dashboard.json`.

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Write dashboard JSON configs to files.** Dashboard configurations are large (2K+ tokens each). Always write them to a file (e.g., `dashboard.json`) instead of pasting inline.
- **Load widget and schema references selectively.** Only read `references/widget-reference.md` or `references/dashboard-schema.md` when the user asks about specific widget types or schema details -- do not load them preemptively.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [Widget Reference](references/widget-reference.md) -- detailed widget configuration and query patterns per type.
- [Dashboard Schema](references/dashboard-schema.md) -- full JSON schema with position grid, sections, and time frame options.
- [Incident Dashboard Example](assets/incident-dashboard.json) -- complete 5-widget incident analysis dashboard.
- [Service Health Dashboard Example](assets/service-health-dashboard.json) -- RED-based service health dashboard.
- [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md) -- DataPrime syntax, commands, and functions
