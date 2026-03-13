---
name: ibm-cloud-logs-api-reference
description: >
  IBM Cloud Logs API reference covering authentication, endpoints, and
  all tool categories. Activate when understanding API endpoints,
  authentication, error handling, or making direct API calls to
  IBM Cloud Logs.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: api
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs API Reference Skill

## When to Activate

Use this skill when the user:
- Asks about IBM Cloud Logs API endpoints, URL structure, or HTTP methods
- Needs to authenticate with IBM Cloud Logs (IAM tokens, API keys)
- Wants to understand which tool or endpoint handles a specific operation
- Is troubleshooting API errors (401, 403, 429, 500)
- Needs to understand rate limits, cost implications, or execution speed of operations
- Wants a map of all available API categories and their CRUD operations
- Asks about the ingress endpoint for log ingestion vs the API endpoint for management
- Needs to understand background query lifecycle or SSE streaming behavior

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

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_IAM_URL` | `https://iam.cloud.ibm.com/identity/token` | Custom IAM endpoint (for staging: `https://iam.test.cloud.ibm.com/identity/token`) |
| `LOGS_TIMEOUT` | `30s` | HTTP request timeout |
| `LOGS_QUERY_TIMEOUT` | `60s` | Sync query timeout |
| `LOGS_MAX_RETRIES` | `3` | Maximum retry attempts |
| `LOGS_ENABLE_RATE_LIMIT` | `true` | Enable rate limiting |
| `LOGS_RATE_LIMIT` | `100` | Requests per second |
| `LOGS_RATE_LIMIT_BURST` | `20` | Burst size |
| `LOGS_HEALTH_PORT` | `8080` | Health/metrics HTTP port (set to `0` to disable) |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json` or `console` |

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

## Authentication

### IAM API Key and Bearer Token

All requests to IBM Cloud Logs require a valid IAM bearer token obtained by
exchanging an IBM Cloud API key. The CLI handles this automatically after
`ibmcloud login`. For curl, exchange the API key manually (see Prerequisites).

**Token exchange flow:**

1. Provide your IBM Cloud API key via the `LOGS_API_KEY` environment variable.
2. The authenticator calls IAM at `https://iam.cloud.ibm.com/identity/token`
   (or a custom `LOGS_IAM_URL` for staging environments) to exchange the key for a
   bearer token.
3. The SDK caches and auto-refreshes the token before expiry.
4. Every outgoing request includes the `Authorization: Bearer <token>` header.

### Endpoint Construction

IBM Cloud Logs uses instance-scoped endpoints:

| Purpose | URL Pattern |
|---------|-------------|
| **Management API** | `https://{instance-id}.api.{region}.logs.cloud.ibm.com` |
| **Ingress (ingestion)** | `https://{instance-id}.ingress.{region}.logs.cloud.ibm.com` |

Request URLs are constructed as `{ServiceURL}{Path}` where `ServiceURL` is the
management API base. For log ingestion, replace `.api.` with `.ingress.` in the URL.

### Standard Request Headers

Every request includes:

| Header | Value |
|--------|-------|
| `Content-Type` | `application/json` |
| `Accept` | `application/json` (or `text/event-stream` for SSE queries) |
| `X-Request-ID` | Client-provided request ID (optional, for tracing) |
| `Idempotency-Key` | Same as request ID on POST/PUT (optional, for safe retries) |

## API Categories

IBM Cloud Logs provides 87 API operations organized into 20 categories. Each
category maps to a set of REST endpoints.

### Query and Analysis (7 operations)

| Operation | Method | Endpoint | Description |
|------|--------|----------|-------------|
| `query_logs` | POST | `/v1/query` | Execute synchronous log queries (SSE) |
| `build_query` | -- | local | Construct DataPrime/Lucene queries locally |
| `get_dataprime_reference` | -- | local | DataPrime syntax documentation |
| `submit_background_query` | POST | `/v1/background_query` | Submit async query |
| `get_background_query_status` | GET | `/v1/background_query/{id}/status` | Poll async query status |
| `get_background_query_data` | GET | `/v1/background_query/{id}/data` | Retrieve async query results |
| `cancel_background_query` | DELETE | `/v1/background_query/{id}` | Cancel a running async query |

### Alert Management (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_alerts` | GET | `/v1/alerts` |
| `get_alert` | GET | `/v1/alerts/{id}` |
| `create_alert` | POST | `/v1/alerts` |
| `update_alert` | PUT | `/v1/alerts/{id}` |
| `delete_alert` | DELETE | `/v1/alerts/{id}` |

### Alert Definitions (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_alert_definitions` | GET | `/v1/alert_definitions` |
| `get_alert_definition` | GET | `/v1/alert_definitions/{id}` |
| `create_alert_definition` | POST | `/v1/alert_definitions` |
| `update_alert_definition` | PUT | `/v1/alert_definitions/{id}` |
| `delete_alert_definition` | DELETE | `/v1/alert_definitions/{id}` |

### Dashboard Management (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_dashboards` | GET | `/v1/dashboards` |
| `get_dashboard` | GET | `/v1/dashboards/{id}` |
| `create_dashboard` | POST | `/v1/dashboards` |
| `update_dashboard` | PUT | `/v1/dashboards/{id}` |
| `delete_dashboard` | DELETE | `/v1/dashboards/{id}` |

### Dashboard Folders and Organization (9 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_dashboard_folders` | GET | `/v1/dashboard_folders` |
| `get_dashboard_folder` | GET | `/v1/dashboard_folders/{id}` |
| `create_dashboard_folder` | POST | `/v1/dashboard_folders` |
| `update_dashboard_folder` | PUT | `/v1/dashboard_folders/{id}` |
| `delete_dashboard_folder` | DELETE | `/v1/dashboard_folders/{id}` |
| `move_dashboard_to_folder` | PUT | `/v1/dashboards/{id}` (folder field) |
| `pin_dashboard` | PUT | `/v1/dashboards/{id}/pinned` |
| `unpin_dashboard` | DELETE | `/v1/dashboards/{id}/pinned` |
| `set_default_dashboard` | PUT | `/v1/dashboards/{id}/default` |

### Rule Groups (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_rule_groups` | GET | `/v1/rule_groups` |
| `get_rule_group` | GET | `/v1/rule_groups/{id}` |
| `create_rule_group` | POST | `/v1/rule_groups` |
| `update_rule_group` | PUT | `/v1/rule_groups/{id}` |
| `delete_rule_group` | DELETE | `/v1/rule_groups/{id}` |

### Rule Helpers (2 operations)

| Tool | Method | Description |
|------|--------|-------------|
| `discover_log_fields` | POST | Discover available log fields |
| `test_rule_group` | POST | Test rule group against sample data |

### Outgoing Webhooks (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_outgoing_webhooks` | GET | `/v1/outgoing_webhooks` |
| `get_outgoing_webhook` | GET | `/v1/outgoing_webhooks/{id}` |
| `create_outgoing_webhook` | POST | `/v1/outgoing_webhooks` |
| `update_outgoing_webhook` | PUT | `/v1/outgoing_webhooks/{id}` |
| `delete_outgoing_webhook` | DELETE | `/v1/outgoing_webhooks/{id}` |

### Policies (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_policies` | GET | `/v1/policies` |
| `get_policy` | GET | `/v1/policies/{id}` |
| `create_policy` | POST | `/v1/policies` |
| `update_policy` | PUT | `/v1/policies/{id}` |
| `delete_policy` | DELETE | `/v1/policies/{id}` |

### Events to Metrics (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_e2m` | GET | `/v1/events2metrics` |
| `get_e2m` | GET | `/v1/events2metrics/{id}` |
| `create_e2m` | POST | `/v1/events2metrics` |
| `replace_e2m` | PUT | `/v1/events2metrics/{id}` |
| `delete_e2m` | DELETE | `/v1/events2metrics/{id}` |

### Data Access Rules (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_data_access_rules` | GET | `/v1/data_access_rules` |
| `get_data_access_rule` | GET | `/v1/data_access_rules/{id}` |
| `create_data_access_rule` | POST | `/v1/data_access_rules` |
| `update_data_access_rule` | PUT | `/v1/data_access_rules/{id}` |
| `delete_data_access_rule` | DELETE | `/v1/data_access_rules/{id}` |

### Enrichments (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_enrichments` | GET | `/v1/enrichments` |
| `get_enrichments` | GET | `/v1/enrichments/{id}` |
| `create_enrichment` | POST | `/v1/enrichments` |
| `update_enrichment` | PUT | `/v1/enrichments/{id}` |
| `delete_enrichment` | DELETE | `/v1/enrichments/{id}` |

### Views (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_views` | GET | `/v1/views` |
| `get_view` | GET | `/v1/views/{id}` |
| `create_view` | POST | `/v1/views` |
| `replace_view` | PUT | `/v1/views/{id}` |
| `delete_view` | DELETE | `/v1/views/{id}` |

### View Folders (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_view_folders` | GET | `/v1/view_folders` |
| `get_view_folder` | GET | `/v1/view_folders/{id}` |
| `create_view_folder` | POST | `/v1/view_folders` |
| `replace_view_folder` | PUT | `/v1/view_folders/{id}` |
| `delete_view_folder` | DELETE | `/v1/view_folders/{id}` |

### Streams (5 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `list_streams` | GET | `/v1/streams` |
| `get_stream` | GET | `/v1/streams/{id}` |
| `create_stream` | POST | `/v1/streams` |
| `update_stream` | PUT | `/v1/streams/{id}` |
| `delete_stream` | DELETE | `/v1/streams/{id}` |

### Data Usage (2 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `export_data_usage` | GET | `/v1/data_usage` |
| `update_data_usage_metrics_export_status` | PUT | `/v1/data_usage/metrics_export` |

### Event Stream Targets (4 operations)

| Operation | Method | Endpoint |
|------|--------|----------|
| `get_event_stream_targets` | GET | `/v1/event_stream_targets` |
| `create_event_stream_target` | POST | `/v1/event_stream_targets` |
| `update_event_stream_target` | PUT | `/v1/event_stream_targets/{id}` |
| `delete_event_stream_target` | DELETE | `/v1/event_stream_targets/{id}` |

### Log Ingestion (1 tool)

| Operation | Method | Endpoint | Host |
|------|--------|----------|------|
| `ingest_logs` | POST | `/logs/v1/singles` | Ingress endpoint |

### AI Helpers (3 operations)

| Operation | Description |
|------|-------------|
| `explain_query` | Explain what a DataPrime/Lucene query does |
| `suggest_alert` | SRE-grade alert recommendations with burn rate alerting |
| `get_audit_log` | Retrieve audit log entries |

### Query Intelligence (3 operations)

| Operation | Description |
|------|-------------|
| `get_query_templates` | Pre-built query templates for common use cases |
| `validate_query` | Validate query syntax before execution |
| `estimate_query_cost` | Heuristic-based query complexity estimation |

### Workflow Automation (2 operations)

| Operation | Description |
|------|-------------|
| `investigate_incident` | Guided multi-query incident investigation |
| `health_check` | Quick system health overview |

### Meta / Discovery (5 operations)

| Operation | Description |
|------|-------------|
| `discover_tools` | Semantic tool discovery |
| `session_context` | Session state and preferences management |
| `search_tools` | Token-efficient tool search (brief results) |
| `describe_tools` | Full schema retrieval for specific tools |
| `list_tool_categories` | Category overview with tool counts |

## Key API Patterns

### CRUD Consistency

Most resource categories follow a uniform REST pattern:

```
GET    /v1/{resource}          -- List all
GET    /v1/{resource}/{id}     -- Get one
POST   /v1/{resource}          -- Create
PUT    /v1/{resource}/{id}     -- Update (or replace)
DELETE /v1/{resource}/{id}     -- Delete
```

Some resources use "replace" semantics where PUT fully replaces the resource
(E2M, Views, View Folders). All others use standard update semantics.

### Request and Response Format

- All request and response bodies are JSON (`Content-Type: application/json`).
- List endpoints return results under a resource-specific key
  (e.g., `{"alerts": [...]}`, `{"dashboards": [...]}`).
- Create/Update endpoints return the full resource object.
- Delete endpoints return `204 No Content` on success.

### Ingestion vs Management

The API uses two distinct host types. Management operations (CRUD, queries)
go to the `.api.` host. Log ingestion uses the `.ingress.` host with
the `/logs/v1/singles` path. The client handles this routing transparently.

## Cost and Rate Limits

API operations have varying cost, latency, and rate-limit characteristics.

### Cost Levels

| Level | Description | Examples |
|-------|-------------|---------------|
| `free` | No API calls, local only | `build_query`, `discover_tools` |
| `low` | Single simple API call | `list_alerts`, `create_dashboard` |
| `medium` | Multiple calls or moderate computation | `query_logs`, `health_check`, `ingest_logs` |
| `high` | Complex queries, large data processing | `submit_background_query`, `investigate_incident` |
| `very_high` | Bulk operations, full scans | Bulk operations |

### Execution Speed

| Speed | Latency | Examples |
|-------|---------|---------------|
| `instant` | < 100ms | `build_query`, `discover_tools` |
| `fast` | < 1s | `list_alerts`, `create_policy`, `ingest_logs` |
| `medium` | 1-10s | `query_logs`, `suggest_alert`, `investigate_incident` |
| `slow` | 10-60s | Complex operations |
| `async` | > 60s | `submit_background_query` |

### Rate Limit Impact

| Level | API Calls | Examples |
|-------|-----------|---------------|
| `none` | 0 | `build_query`, `discover_tools` |
| `minimal` | 1 | `list_alerts`, `create_alert`, `delete_policy` |
| `moderate` | 2-5 | `query_logs`, `health_check`, `ingest_logs` |
| `high` | 6-20 | `investigate_incident` |
| `burst` | 20+ | Bulk operations (may throttle) |

### System Impact

| Level | Description | Confirmation Required |
|-------|-------------|----------------------|
| `none` | Read-only, no changes | No |
| `low` | Minor, easily reversible | No |
| `medium` | Affects workflows (alerts, webhooks, E2M) | No |
| `high` | Significant (alert deletion, policy changes) | Yes |
| `critical` | Destructive / system-wide (`delete_policy`) | Yes |

### Client-Side Rate Limiting

When the server returns HTTP 429, honor the `Retry-After` header. Add jitter
to avoid thundering herd effects. The `LOGS_RATE_LIMIT` and `LOGS_RATE_LIMIT_BURST`
environment variables configure client-side rate limiting when using SDK clients.

## Query Execution

### Synchronous Queries (SSE)

`query_logs` sends a POST to `/v1/query` with `Accept: text/event-stream`.
The server responds with Server-Sent Events where each `data:` line contains
a JSON payload. The client collects all events and returns a unified result.

- Use `frequent_search` tier for real-time data (last 24h).
- Use `archive` tier for historical analysis.
- Include time bounds (`start_date`, `end_date`) to reduce cost.
- Set `limit` to control response size (recommended: 100).

### Background Queries

For large or long-running queries, use the background query lifecycle:

1. `submit_background_query` -- returns a `query_id`.
2. `get_background_query_status` -- poll until status is `completed`.
3. `get_background_query_data` -- retrieve results (supports pagination).
4. `cancel_background_query` -- cancel if no longer needed.

Background queries are best for scans spanning more than 24 hours or expecting
more than 10,000 results.

## Error Handling

### Common HTTP Errors

| Status | Meaning | Action |
|--------|---------|--------|
| 400 | Bad Request | Check request body, query syntax, or parameters |
| 401 | Unauthorized | API key invalid or token expired; re-authenticate |
| 403 | Forbidden | Insufficient IAM permissions for the operation |
| 404 | Not Found | Resource ID does not exist |
| 429 | Too Many Requests | Rate limited; wait per `Retry-After` header |
| 500 | Internal Error | Retry with exponential backoff |
| 502/503/504 | Gateway errors | Transient; automatic retry with backoff |

### Retry Strategy

The client retries on 429, 500, 502, 503, and 504 with exponential backoff
plus jitter. Network errors (connection reset, DNS failure, timeout) are also
retried. Maximum retries and backoff bounds are configurable. The client
parses both delta-seconds and HTTP-date formats in `Retry-After` headers.

## Using the IBM Cloud CLI

All API operations are also available via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# Prerequisites (one-time setup)
ibmcloud plugin install logs
ibmcloud login --apikey <your-api-key> -r <region>
```

### CLI Command Mapping

| API Category | CLI Commands |
|---|---|
| Queries | `query`, `bgq-create`, `bgq-status`, `bgq-data`, `bgq-cancel` |
| Alert Definitions | `alert-definitions`, `alert-definition`, `alert-definition-create`, `-update`, `-delete` |
| Alerts | `alerts`, `alert`, `alert-create`, `-update`, `-delete` |
| Policies | `policies`, `policy`, `policy-create`, `-update`, `-delete` |
| Rule Groups | `rule-groups`, `rule-group`, `rule-group-create`, `-update`, `-delete` |
| Enrichments | `enrichments`, `enrichment-create`, `-delete` |
| Views | `views`, `view`, `view-create`, `-update`, `-delete` |
| View Folders | `view-folders`, `view-folder`, `view-folder-create`, `-update`, `-delete` |
| Webhooks | `outgoing-webhooks`, `outgoing-webhook`, `-create`, `-update`, `-delete` |
| E2M | `e2m-list`, `e2m`, `e2m-create`, `-update`, `-delete` |
| Data Access Rules | `data-access-rules`, `data-access-rule-create`, `-update`, `-delete` |
| Event Streams | `event-stream-targets`, `event-stream-target-create`, `-update`, `-delete` |
| Data Usage | `data-usage`, `data-usage-metrics-export-status`, `-update` |
| Configuration | `config` |

All commands support `--output json` for machine-readable output. Create/update commands accept JSON input via `--prototype @file.json`. **Note:** Dashboard operations have no CLI support — use the REST API directly (see [Dashboards Skill](../ibm-cloud-logs-dashboards/SKILL.md)).

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Search the endpoint catalog selectively.** The `assets/api-endpoints.json` file contains the full endpoint catalog and is large. Search it for specific endpoints by name or path rather than loading the entire file.
- **Write API call examples to files.** When generating curl commands or API call examples, write them to a file (e.g., `api-example.sh`) instead of pasting them inline.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [Endpoint Reference](references/endpoints.md) -- all endpoints by category
- [Authentication Guide](references/authentication.md) -- IAM token exchange details
- [Endpoint Catalog (JSON)](assets/api-endpoints.json) -- machine-readable endpoint data
- [IBM Cloud Logs Docs](https://cloud.ibm.com/docs/cloud-logs)
- [IBM Cloud Logs API](https://cloud.ibm.com/apidocs/logs-service-api)
