# IBM Cloud Logs API Endpoints

Complete endpoint reference organized by category. All paths are relative to
the management API base URL unless noted otherwise.

**Management API base:** `https://{instance-id}.api.{region}.logs.cloud.ibm.com`
**Ingress base:** `https://{instance-id}.ingress.{region}.logs.cloud.ibm.com`

---

## Query and Analysis

| Method | Path | Tool | Notes |
|--------|------|------|-------|
| POST | `/v1/query` | `query_logs` | SSE streaming response (`Accept: text/event-stream`) |
| POST | `/v1/background_query` | `submit_background_query` | Returns query ID for async polling |
| GET | `/v1/background_query/{id}/status` | `get_background_query_status` | Status: `running`, `completed`, `failed` |
| GET | `/v1/background_query/{id}/data` | `get_background_query_data` | Supports `offset` and `limit` for pagination |
| DELETE | `/v1/background_query/{id}` | `cancel_background_query` | Cancels a running background query |
| -- | local | `build_query` | Constructs queries locally, no API call |
| -- | local | `get_dataprime_reference` | Returns syntax docs locally, no API call |

## Alert Management

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/alerts` | `list_alerts` |
| GET | `/v1/alerts/{id}` | `get_alert` |
| POST | `/v1/alerts` | `create_alert` |
| PUT | `/v1/alerts/{id}` | `update_alert` |
| DELETE | `/v1/alerts/{id}` | `delete_alert` |

## Alert Definitions

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/alert_definitions` | `list_alert_definitions` |
| GET | `/v1/alert_definitions/{id}` | `get_alert_definition` |
| POST | `/v1/alert_definitions` | `create_alert_definition` |
| PUT | `/v1/alert_definitions/{id}` | `update_alert_definition` |
| DELETE | `/v1/alert_definitions/{id}` | `delete_alert_definition` |

## Dashboards

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/dashboards` | `list_dashboards` |
| GET | `/v1/dashboards/{id}` | `get_dashboard` |
| POST | `/v1/dashboards` | `create_dashboard` |
| PUT | `/v1/dashboards/{id}` | `update_dashboard` |
| DELETE | `/v1/dashboards/{id}` | `delete_dashboard` |

## Dashboard Folders and Organization

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/dashboard_folders` | `list_dashboard_folders` |
| GET | `/v1/dashboard_folders/{id}` | `get_dashboard_folder` |
| POST | `/v1/dashboard_folders` | `create_dashboard_folder` |
| PUT | `/v1/dashboard_folders/{id}` | `update_dashboard_folder` |
| DELETE | `/v1/dashboard_folders/{id}` | `delete_dashboard_folder` |
| PUT | `/v1/dashboards/{id}` | `move_dashboard_to_folder` (updates folder field) |
| PUT | `/v1/dashboards/{id}/pinned` | `pin_dashboard` |
| DELETE | `/v1/dashboards/{id}/pinned` | `unpin_dashboard` |
| PUT | `/v1/dashboards/{id}/default` | `set_default_dashboard` |

## Rule Groups

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/rule_groups` | `list_rule_groups` |
| GET | `/v1/rule_groups/{id}` | `get_rule_group` |
| POST | `/v1/rule_groups` | `create_rule_group` |
| PUT | `/v1/rule_groups/{id}` | `update_rule_group` |
| DELETE | `/v1/rule_groups/{id}` | `delete_rule_group` |

## Rule Helpers

| Method | Path | Tool | Notes |
|--------|------|------|-------|
| POST | -- | `discover_log_fields` | Discovers available log fields |
| POST | -- | `test_rule_group` | Tests rule group against sample data |

## Outgoing Webhooks

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/outgoing_webhooks` | `list_outgoing_webhooks` |
| GET | `/v1/outgoing_webhooks/{id}` | `get_outgoing_webhook` |
| POST | `/v1/outgoing_webhooks` | `create_outgoing_webhook` |
| PUT | `/v1/outgoing_webhooks/{id}` | `update_outgoing_webhook` |
| DELETE | `/v1/outgoing_webhooks/{id}` | `delete_outgoing_webhook` |

## Policies

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/policies` | `list_policies` |
| GET | `/v1/policies/{id}` | `get_policy` |
| POST | `/v1/policies` | `create_policy` |
| PUT | `/v1/policies/{id}` | `update_policy` |
| DELETE | `/v1/policies/{id}` | `delete_policy` |

## Events to Metrics (E2M)

Uses replace semantics (PUT fully replaces the resource).

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/events2metrics` | `list_e2m` |
| GET | `/v1/events2metrics/{id}` | `get_e2m` |
| POST | `/v1/events2metrics` | `create_e2m` |
| PUT | `/v1/events2metrics/{id}` | `replace_e2m` |
| DELETE | `/v1/events2metrics/{id}` | `delete_e2m` |

## Data Access Rules

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/data_access_rules` | `list_data_access_rules` |
| GET | `/v1/data_access_rules/{id}` | `get_data_access_rule` |
| POST | `/v1/data_access_rules` | `create_data_access_rule` |
| PUT | `/v1/data_access_rules/{id}` | `update_data_access_rule` |
| DELETE | `/v1/data_access_rules/{id}` | `delete_data_access_rule` |

## Enrichments

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/enrichments` | `list_enrichments` |
| GET | `/v1/enrichments/{id}` | `get_enrichments` |
| POST | `/v1/enrichments` | `create_enrichment` |
| PUT | `/v1/enrichments/{id}` | `update_enrichment` |
| DELETE | `/v1/enrichments/{id}` | `delete_enrichment` |

## Views

Uses replace semantics (PUT fully replaces the resource).

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/views` | `list_views` |
| GET | `/v1/views/{id}` | `get_view` |
| POST | `/v1/views` | `create_view` |
| PUT | `/v1/views/{id}` | `replace_view` |
| DELETE | `/v1/views/{id}` | `delete_view` |

## View Folders

Uses replace semantics (PUT fully replaces the resource).

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/view_folders` | `list_view_folders` |
| GET | `/v1/view_folders/{id}` | `get_view_folder` |
| POST | `/v1/view_folders` | `create_view_folder` |
| PUT | `/v1/view_folders/{id}` | `replace_view_folder` |
| DELETE | `/v1/view_folders/{id}` | `delete_view_folder` |

## Streams

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/streams` | `list_streams` |
| GET | `/v1/streams/{id}` | `get_stream` |
| POST | `/v1/streams` | `create_stream` |
| PUT | `/v1/streams/{id}` | `update_stream` |
| DELETE | `/v1/streams/{id}` | `delete_stream` |

## Data Usage

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/data_usage` | `export_data_usage` |
| PUT | `/v1/data_usage/metrics_export` | `update_data_usage_metrics_export_status` |

## Event Stream Targets

| Method | Path | Tool |
|--------|------|------|
| GET | `/v1/event_stream_targets` | `get_event_stream_targets` |
| POST | `/v1/event_stream_targets` | `create_event_stream_target` |
| PUT | `/v1/event_stream_targets/{id}` | `update_event_stream_target` |
| DELETE | `/v1/event_stream_targets/{id}` | `delete_event_stream_target` |

## Log Ingestion

Uses the **ingress** host, not the management API host.

| Method | Path | Tool | Host |
|--------|------|------|------|
| POST | `/logs/v1/singles` | `ingest_logs` | `{instance-id}.ingress.{region}.logs.cloud.ibm.com` |
