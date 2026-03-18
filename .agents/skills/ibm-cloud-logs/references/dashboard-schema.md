# Dashboard JSON Schema

Complete schema reference for IBM Cloud Logs dashboard configuration. Derived from the IBM Cloud Logs API (`/v1/dashboards`).

## Top-Level Structure

```json
{
  "name": "string (required)",
  "description": "string (optional)",
  "folder_id": "string | null (optional)",
  "layout": {
    "sections": [ ... ]
  },
  "widgets": [ ... ],
  "variables": [ ... ],
  "filters": [ ... ],
  "time_frame": {
    "relative": "string"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name for the dashboard |
| `description` | string | No | Purpose or context for the dashboard |
| `folder_id` | string or null | No | Target folder ID; `null` for root |
| `layout` | object | Yes | Contains sections, rows, and widgets |
| `variables` | array | No | Template variable definitions |
| `filters` | array | No | Global dashboard filters |
| `time_frame` | object | No | Default time range for all widgets |

## Layout: Sections

Sections are logical groupings within a dashboard. Most dashboards use a single section.

```json
{
  "sections": [
    {
      "id": { "value": "section-1" },
      "rows": [ ... ]
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id.value` | string | Unique section identifier |
| `rows` | array | Horizontal row containers within this section |

## Layout: Rows

Rows are horizontal containers that hold widgets. Each row has a fixed height.

```json
{
  "id": { "value": "row-1" },
  "appearance": { "height": 19 },
  "widgets": [ ... ]
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id.value` | string | -- | Unique row identifier |
| `appearance.height` | integer | 19 | Row height in grid units |
| `widgets` | array | -- | Widgets displayed in this row |

## Layout: Widgets

Widgets are the individual visualizations within a row.

```json
{
  "id": { "value": "widget-1" },
  "title": "Error Rate Over Time",
  "appearance": { "width": 0 },
  "definition": {
    "line_chart": { ... }
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id.value` | string | -- | Unique widget identifier (UUID recommended) |
| `title` | string | -- | Display title shown above the widget |
| `appearance.width` | integer | 0 | Widget width; `0` = auto (fills available space) |
| `definition` | object | -- | Contains exactly one key matching the widget type |

### Definition Keys

The `definition` object must contain exactly one of these keys:

| Key | Widget Type |
|-----|-------------|
| `line_chart` | Time-series line chart |
| `bar_chart` | Categorical bar chart |
| `pie_chart` | Proportional pie chart |
| `data_table` | Tabular data display |
| `gauge` | Single-value KPI gauge |
| `markdown` | Static markdown text |

## Position Grid (Auto-Generated)

When using `generateIBMCloudLogsDashboardJSON()`, widgets are automatically positioned on a 12-column grid:

```json
{
  "position": {
    "x": 0,
    "y": 0,
    "width": 6,
    "height": 4
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `x` | integer | Horizontal position (0-11, in a 12-column grid) |
| `y` | integer | Vertical position (row offset) |
| `width` | integer | Widget width in grid columns (1-12) |
| `height` | integer | Widget height in grid rows |

**Layout algorithm:** Widgets are arranged in a 2-column layout:
- Even-indexed widgets: `x = 0`, `y = (index / 2) * 4`
- Odd-indexed widgets: `x = 6`, `y = (index / 2) * 4`
- Each widget defaults to `width: 6`, `height: 4`

## Time Frame

The `time_frame` object controls the default time range for the dashboard.

### Relative Time

```json
{
  "time_frame": {
    "relative": "last_1_hour"
  }
}
```

Valid values for `relative`:

| Value | Duration |
|-------|----------|
| `last_5_minutes` | 5 minutes |
| `last_15_minutes` | 15 minutes |
| `last_1_hour` | 1 hour |
| `last_6_hours` | 6 hours |
| `last_12_hours` | 12 hours |
| `last_24_hours` | 24 hours |
| `last_2_days` | 2 days |
| `last_3_days` | 3 days |
| `last_7_days` | 7 days |
| `last_14_days` | 14 days |
| `last_30_days` | 30 days |

## Variables

Template variables enable dynamic filtering in dashboards. Users can change values via dropdown selectors in the UI.

```json
{
  "variables": [
    {
      "name": "service_name",
      "display_name": "Service",
      "type": "query",
      "query": "source logs | groupby $l.applicationname | limit 50"
    }
  ]
}
```

Variables are referenced in widget queries with the `$p.` prefix:
```
source logs | filter $l.applicationname == $p.service_name
```

## Filters

Global filters apply to all widgets in the dashboard.

```json
{
  "filters": [
    {
      "source": "logs",
      "enabled": true,
      "field": "$l.applicationname",
      "operator": "equals",
      "value": "my-service"
    }
  ]
}
```

## API Endpoints

| Operation | Method | Path |
|-----------|--------|------|
| List dashboards | `GET` | `/v1/dashboards` |
| Get dashboard | `GET` | `/v1/dashboards/{id}` |
| Create dashboard | `POST` | `/v1/dashboards` |
| Update dashboard | `PUT` | `/v1/dashboards/{id}` |
| Delete dashboard | `DELETE` | `/v1/dashboards/{id}` |
| Move to folder | `PUT` | `/v1/dashboards/{id}/folder/{folder_id}` |
| Pin dashboard | `PUT` | `/v1/dashboards/{id}/pinned` |
| Unpin dashboard | `DELETE` | `/v1/dashboards/{id}/pinned` |
| Set default | `PUT` | `/v1/dashboards/{id}/default` |
| List folders | `GET` | `/v1/folders` |
| Get folder | `GET` | `/v1/folders/{id}` |
| Create folder | `POST` | `/v1/folders` |
| Update folder | `PUT` | `/v1/folders/{id}` |
| Delete folder | `DELETE` | `/v1/folders/{id}` |

## Validation

The `create_dashboard` tool validates all queries before creating the dashboard:
- Queries are extracted from all widget definitions (DataPrime and Lucene).
- Each query is executed against the API with a 1-minute time window and `limit 1` to verify syntax.
- If any query fails validation, the dashboard is not created and errors are returned.
- Use `dry_run: true` to validate without creating.
