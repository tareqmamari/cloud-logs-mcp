# API Reference Guide

> Domain guide for IBM Cloud Logs API. For inline essentials (authentication,
> DataPrime syntax), see [SKILL.md](../SKILL.md).

## Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_IAM_URL` | `https://iam.cloud.ibm.com/identity/token` | Custom IAM endpoint |
| `LOGS_TIMEOUT` | `30s` | HTTP request timeout |
| `LOGS_QUERY_TIMEOUT` | `60s` | Sync query timeout |
| `LOGS_MAX_RETRIES` | `3` | Maximum retry attempts |
| `LOGS_ENABLE_RATE_LIMIT` | `true` | Enable rate limiting |
| `LOGS_RATE_LIMIT` | `100` | Requests per second |
| `LOGS_RATE_LIMIT_BURST` | `20` | Burst size |
| `LOGS_HEALTH_PORT` | `8080` | Health/metrics HTTP port |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json` or `console` |

## Authentication

### IAM Token Exchange Flow

1. Provide API key via `LOGS_API_KEY`
2. Call IAM at `https://iam.cloud.ibm.com/identity/token` to exchange for bearer token
3. SDK caches and auto-refreshes before expiry
4. Every request includes `Authorization: Bearer <token>`

### Endpoint Construction

| Purpose | URL Pattern |
|---------|-------------|
| **Management API** | `https://{instance-id}.api.{region}.logs.cloud.ibm.com` |
| **Ingress (ingestion)** | `https://{instance-id}.ingress.{region}.logs.cloud.ibm.com` |

### Standard Request Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/json` |
| `Accept` | `application/json` (or `text/event-stream` for SSE queries) |
| `X-Request-ID` | Client-provided request ID (optional) |
| `Idempotency-Key` | Same as request ID on POST/PUT (optional) |

## API Categories (87 operations, 20 categories)

### Query and Analysis (7 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `query_logs` | POST | `/v1/query` |
| `submit_background_query` | POST | `/v1/background_query` |
| `get_background_query_status` | GET | `/v1/background_query/{id}/status` |
| `get_background_query_data` | GET | `/v1/background_query/{id}/data` |
| `cancel_background_query` | DELETE | `/v1/background_query/{id}` |
| `build_query` | -- | local |
| `get_dataprime_reference` | -- | local |

### Alert Management (5 operations)

`GET/POST /v1/alerts`, `GET/PUT/DELETE /v1/alerts/{id}`

### Alert Definitions (5 operations)

`GET/POST /v1/alert_definitions`, `GET/PUT/DELETE /v1/alert_definitions/{id}`

### Dashboard Management (5 operations)

`GET/POST /v1/dashboards`, `GET/PUT/DELETE /v1/dashboards/{id}`

### Dashboard Folders and Organization (9 operations)

`GET/POST /v1/dashboard_folders`, `GET/PUT/DELETE /v1/dashboard_folders/{id}`
Plus: `move_dashboard_to_folder`, `pin_dashboard`, `unpin_dashboard`, `set_default_dashboard`

### Rule Groups (5 operations)

`GET/POST /v1/rule_groups`, `GET/PUT/DELETE /v1/rule_groups/{id}`

### Rule Helpers (2 operations)

`discover_log_fields` (POST), `test_rule_group` (POST)

### Outgoing Webhooks (5 operations)

`GET/POST /v1/outgoing_webhooks`, `GET/PUT/DELETE /v1/outgoing_webhooks/{id}`

### Policies (5 operations)

`GET/POST /v1/policies`, `GET/PUT/DELETE /v1/policies/{id}`

### Events to Metrics (5 operations)

`GET/POST /v1/events2metrics`, `GET/PUT/DELETE /v1/events2metrics/{id}`

### Data Access Rules (5 operations)

`GET/POST /v1/data_access_rules`, `GET/PUT/DELETE /v1/data_access_rules/{id}`

### Enrichments (5 operations)

`GET/POST /v1/enrichments`, `GET/PUT/DELETE /v1/enrichments/{id}`

### Views (5 operations)

`GET/POST /v1/views`, `GET/PUT/DELETE /v1/views/{id}`

### View Folders (5 operations)

`GET/POST /v1/view_folders`, `GET/PUT/DELETE /v1/view_folders/{id}`

### Streams (5 operations)

`GET/POST /v1/streams`, `GET/PUT/DELETE /v1/streams/{id}`

### Data Usage (2 operations)

`GET /v1/data_usage`, `PUT /v1/data_usage/metrics_export`

### Event Stream Targets (4 operations)

`GET/POST /v1/event_stream_targets`, `PUT/DELETE /v1/event_stream_targets/{id}`

### Log Ingestion (1 operation)

`POST /logs/v1/singles` (Ingress endpoint)

### AI Helpers (3 operations)

`explain_query`, `suggest_alert`, `get_audit_log`

### Query Intelligence (3 operations)

`get_query_templates`, `validate_query`, `estimate_query_cost`

### Workflow Automation (2 operations)

`investigate_incident`, `health_check`

### Meta / Discovery (5 operations)

`discover_tools`, `session_context`, `search_tools`, `describe_tools`, `list_tool_categories`

## Key API Patterns

### CRUD Consistency

```
GET    /v1/{resource}          -- List all
GET    /v1/{resource}/{id}     -- Get one
POST   /v1/{resource}          -- Create
PUT    /v1/{resource}/{id}     -- Update (or replace)
DELETE /v1/{resource}/{id}     -- Delete
```

Some resources use "replace" semantics (E2M, Views, View Folders).

### Ingestion vs Management

Management operations use `.api.` host. Log ingestion uses `.ingress.` host with `/logs/v1/singles`.

## Cost and Rate Limits

### Cost Levels

| Level | Examples |
|-------|---------|
| `free` | `build_query`, `discover_tools` |
| `low` | `list_alerts`, `create_dashboard` |
| `medium` | `query_logs`, `health_check`, `ingest_logs` |
| `high` | `submit_background_query`, `investigate_incident` |

### Execution Speed

| Speed | Latency | Examples |
|-------|---------|---------|
| `instant` | < 100ms | `build_query`, `discover_tools` |
| `fast` | < 1s | `list_alerts`, `create_policy` |
| `medium` | 1-10s | `query_logs`, `suggest_alert` |
| `slow` | 10-60s | Complex operations |
| `async` | > 60s | `submit_background_query` |

### System Impact

| Level | Confirmation Required |
|-------|----------------------|
| `none` (read-only) | No |
| `low` (easily reversible) | No |
| `medium` (affects workflows) | No |
| `high` (alert/policy deletion) | Yes |
| `critical` (system-wide) | Yes |

## Query Execution

### Synchronous Queries (SSE)

POST to `/v1/query` with `Accept: text/event-stream`. Server responds with SSE events.

### Background Queries

1. `submit_background_query` → `query_id`
2. `get_background_query_status` → poll until `completed`
3. `get_background_query_data` → retrieve results
4. `cancel_background_query` → cancel if unneeded

## Error Handling

| Status | Meaning | Action |
|--------|---------|--------|
| 400 | Bad Request | Check body, syntax, parameters |
| 401 | Unauthorized | Re-authenticate |
| 403 | Forbidden | Insufficient IAM permissions |
| 404 | Not Found | Resource ID doesn't exist |
| 429 | Too Many Requests | Wait per `Retry-After` header |
| 500 | Internal Error | Retry with exponential backoff |
| 502/503/504 | Gateway errors | Automatic retry with backoff |

### CLI Command Mapping

| API Category | CLI Commands |
|---|---|
| Queries | `query`, `bgq-create`, `bgq-status`, `bgq-data`, `bgq-cancel` |
| Alert Definitions | `alert-definitions`, `alert-definition-create`, `-update`, `-delete` |
| Alerts | `alerts`, `alert-create`, `-update`, `-delete` |
| Policies | `policies`, `policy-create`, `-update`, `-delete` |
| Rule Groups | `rule-groups`, `rule-group-create`, `-update`, `-delete` |
| Enrichments | `enrichments`, `enrichment-create`, `-delete` |
| Views | `views`, `view-create`, `-update`, `-delete` |
| Webhooks | `outgoing-webhooks`, `-create`, `-update`, `-delete` |
| E2M | `e2m-list`, `e2m-create`, `-update`, `-delete` |
| Data Access | `data-access-rules`, `data-access-rule-create`, `-update`, `-delete` |
| Event Streams | `event-stream-targets`, `event-stream-target-create`, `-update`, `-delete` |
| Data Usage | `data-usage` |

**Note:** Dashboard operations have no CLI support — use REST API directly.

## Deep References

- [Endpoint Reference](endpoints.md) -- All endpoints by category
- [Authentication Guide](authentication.md) -- IAM token exchange details
- [Endpoint Catalog (JSON)](../assets/api-endpoints.json) -- Machine-readable endpoint data
- [IBM Cloud Logs Docs](https://cloud.ibm.com/docs/cloud-logs)
- [IBM Cloud Logs API](https://cloud.ibm.com/apidocs/logs-service-api)
