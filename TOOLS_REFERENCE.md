# IBM Cloud Logs MCP Tools Reference

This document provides comprehensive documentation for all 92 tools available in the IBM Cloud Logs MCP Server. Tools are organized by category with best practices and usage guidelines.

---

## Table of Contents

- [Quick Reference](#quick-reference)
- [Query Operations](#query-operations)
- [Log Ingestion](#log-ingestion)
- [Alert Management](#alert-management)
- [Alert Definitions](#alert-definitions)
- [Dashboard Management](#dashboard-management)
- [Dashboard Folders](#dashboard-folders)
- [Rule Groups](#rule-groups)
- [Webhooks](#webhooks)
- [Policies](#policies)
- [Events to Metrics (E2M)](#events-to-metrics-e2m)
- [Data Access Rules](#data-access-rules)
- [Enrichments](#enrichments)
- [Views](#views)
- [View Folders](#view-folders)
- [Streams](#streams)
- [Data Usage](#data-usage)
- [Event Stream Targets](#event-stream-targets)
- [AI Helpers](#ai-helpers)
- [Query Intelligence](#query-intelligence)
- [Workflow Automation](#workflow-automation)
- [Root Cause Analysis](#root-cause-analysis)
- [Meta Tools](#meta-tools)
- [Best Practices](#best-practices)

---

## Quick Reference

| Category | Tools | Primary Use |
|----------|-------|-------------|
| Query | 7 | Searching and analyzing logs |
| Ingestion | 1 | Sending logs to IBM Cloud Logs |
| Alerts | 5 | Managing alert instances |
| Alert Definitions | 5 | Creating alert templates |
| Dashboards | 5 | Visualization management |
| Dashboard Folders | 9 | Dashboard organization |
| Rule Groups | 5 | Log parsing rules |
| Webhooks | 5 | Alert notifications |
| Policies | 5 | Retention and routing policies |
| E2M | 5 | Events to metrics conversion |
| Data Access | 5 | Access control rules |
| Enrichments | 5 | Log enrichment rules |
| Views | 5 | Saved query views |
| View Folders | 5 | View organization |
| Streams | 5 | Data streaming configuration |
| Data Usage | 2 | Usage metrics export |
| Event Streams | 4 | Event stream targets |
| AI Helpers | 3 | AI-powered analysis |
| Query Intelligence | 3 | Query building assistance |
| Workflows | 2 | Automated investigation |
| Root Cause Analysis | 4 | Advanced RCA with causal discovery |
| Meta | 4 | Tool discovery and session |

---

## Query Operations

### query_logs

Execute synchronous queries against IBM Cloud Logs.

**When to use:** Default method for querying logs. Results stream via SSE.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | DataPrime or Lucene query (max 4096 chars) |
| `tier` | string | No | `archive` (default), `frequent_search`, or `unspecified` |
| `syntax` | string | No | `dataprime` (default), `lucene`, or encoded variants |
| `start_date` | string | No | RFC3339 timestamp for query start |
| `end_date` | string | No | RFC3339 timestamp for query end |
| `limit` | integer | No | Maximum results to return |

**Example:**
```
query: "source logs | filter $m.severity >= 5 | limit 100"
tier: "archive"
```

**Best Practices:**
- Use `frequent_search` tier for real-time monitoring (last 24h)
- Use `archive` for historical analysis
- Include time bounds to reduce query cost
- Use `limit` to control response size

**Related:** `build_query`, `submit_background_query`, `get_dataprime_reference`

---

### build_query

Construct queries without knowing DataPrime/Lucene syntax.

**When to use:** You want to search logs but don't know query syntax.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `text_search` | string | Free text to search for |
| `applications` | array | Filter by application names |
| `subsystems` | array | Filter by subsystem/component names |
| `severities` | array | Filter by severity levels |
| `min_severity` | string | Minimum severity to include |
| `fields` | array | Field-value filters |
| `exclude_text` | string | Text to exclude from results |
| `output_format` | string | `lucene`, `dataprime`, or `both` |

**Example:**
```json
{
  "text_search": "connection timeout",
  "applications": ["api-gateway", "auth-service"],
  "min_severity": "error"
}
```

**Output:** Returns both Lucene and DataPrime query strings.

---

### submit_background_query

Submit queries that run asynchronously for large datasets.

**When to use:** Queries that may timeout or scan large amounts of data.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Query string to execute |
| `tier` | string | No | Data tier to query |
| `syntax` | string | No | Query syntax |
| `start_date` | string | No | Query start time |
| `end_date` | string | No | Query end time |

**Returns:** Query ID for status checking

---

### get_background_query_status

Check the status of a background query.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query_id` | string | Yes | ID from submit_background_query |

**Returns:** Status (`running`, `completed`, `failed`) and progress info

---

### get_background_query_data

Retrieve results from a completed background query.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query_id` | string | Yes | ID from submit_background_query |
| `offset` | integer | No | Pagination offset |
| `limit` | integer | No | Results per page |

---

### cancel_background_query

Cancel a running background query.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query_id` | string | Yes | ID to cancel |

---

### get_dataprime_reference

Get DataPrime syntax documentation.

**When to use:** You need to understand query syntax, operators, or functions.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `topic` | string | `basics`, `operators`, `functions`, `aggregations`, `examples` |

---

## Log Ingestion

### ingest_logs

Send log entries to IBM Cloud Logs.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `logs` | array | Yes | Array of log entries |
| `application_name` | string | No | Application identifier |
| `subsystem_name` | string | No | Component identifier |

**Log Entry Structure:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "severity": "ERROR",
  "message": "Connection timeout",
  "metadata": {
    "user_id": "123",
    "request_id": "abc"
  }
}
```

**Best Practices:**
- Batch multiple log entries for efficiency
- Use consistent application/subsystem naming
- Include structured metadata for filtering

---

## Alert Management

### list_alerts

List all alert instances.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Max results |
| `offset` | integer | Pagination offset |

### get_alert

Get details of a specific alert.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Alert ID |

### create_alert

Create a new alert.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Alert name |
| `description` | string | No | Alert description |
| `is_active` | boolean | No | Whether alert is enabled |
| `severity` | string | Yes | `info`, `warning`, `error`, `critical` |
| `condition` | object | Yes | Alert trigger conditions |
| `notification_groups` | array | No | Where to send notifications |

### update_alert

Update an existing alert.

**Parameters:** Same as `create_alert` plus `id` (required).

### delete_alert

Delete an alert.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Alert ID to delete |

---

## Alert Definitions

Alert definitions are reusable templates for creating alerts.

### list_alert_definitions

List all alert definition templates.

### get_alert_definition

Get a specific alert definition.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Definition ID |

### create_alert_definition

Create an alert definition template.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Definition name |
| `priority` | string | Yes | `P1`, `P2`, `P3`, `P4`, `P5` |
| `type` | string | Yes | Alert type |
| `condition` | object | Yes | Alert conditions |

### update_alert_definition

Update an alert definition.

### delete_alert_definition

Delete an alert definition.

---

## Dashboard Management

### list_dashboards

List all dashboards.

### get_dashboard

Get dashboard details including widgets.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Dashboard ID |

### create_dashboard

Create a new dashboard.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Dashboard name |
| `description` | string | No | Description |
| `layout` | object | No | Widget layout configuration |
| `widgets` | array | No | Dashboard widgets |
| `folder_id` | string | No | Parent folder |

### update_dashboard

Update an existing dashboard.

### delete_dashboard

Delete a dashboard.

---

## Dashboard Folders

### list_dashboard_folders

List all dashboard folders.

### get_dashboard_folder

Get folder details.

### create_dashboard_folder

Create a new folder.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Folder name |
| `parent_id` | string | No | Parent folder ID |

### update_dashboard_folder

Update a folder.

### delete_dashboard_folder

Delete a folder.

### move_dashboard_to_folder

Move a dashboard to a different folder.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `dashboard_id` | string | Yes | Dashboard to move |
| `folder_id` | string | Yes | Target folder |

### pin_dashboard

Pin a dashboard for quick access.

### unpin_dashboard

Unpin a dashboard.

### set_default_dashboard

Set a dashboard as the default view.

---

## Rule Groups

Rule groups define log parsing and transformation rules.

### list_rule_groups

List all rule groups.

### get_rule_group

Get rule group details.

### create_rule_group

Create a new rule group.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Rule group name |
| `description` | string | No | Description |
| `rules` | array | Yes | Array of parsing rules |
| `enabled` | boolean | No | Whether enabled |
| `order` | integer | No | Processing order |

### update_rule_group

Update a rule group.

### delete_rule_group

Delete a rule group.

---

## Webhooks

Outgoing webhooks for alert notifications.

### list_outgoing_webhooks

List all webhooks.

### get_outgoing_webhook

Get webhook details.

### create_outgoing_webhook

Create a new webhook.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Webhook name |
| `type` | string | Yes | `slack`, `pagerduty`, `generic`, etc. |
| `url` | string | Yes | Webhook URL |

### update_outgoing_webhook

Update a webhook.

### delete_outgoing_webhook

Delete a webhook.

---

## Policies

Retention and routing policies.

### list_policies

List all policies.

### get_policy

Get policy details.

### create_policy

Create a new policy.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Policy name |
| `priority` | string | Yes | Processing priority |
| `application_rule` | object | No | Application matching rules |
| `subsystem_rule` | object | No | Subsystem matching rules |
| `archive_retention` | object | No | Retention settings |

### update_policy

Update a policy.

### delete_policy

Delete a policy.

---

## Events to Metrics (E2M)

Convert log events to metrics.

### list_e2m

List all E2M definitions.

### get_e2m

Get E2M definition details.

### create_e2m

Create an E2M definition.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | E2M name |
| `description` | string | No | Description |
| `logs_query` | string | Yes | Query to match logs |
| `type` | string | Yes | Metric type |
| `metric_fields` | object | No | Fields to extract |

### replace_e2m

Replace an E2M definition entirely.

### delete_e2m

Delete an E2M definition.

---

## Data Access Rules

Control access to log data.

### list_data_access_rules

List all data access rules.

### get_data_access_rule

Get rule details.

### create_data_access_rule

Create a new access rule.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `display_name` | string | Yes | Rule display name |
| `filters` | array | Yes | Data filters |
| `default_expression` | string | No | Default filter expression |

### update_data_access_rule

Update an access rule.

### delete_data_access_rule

Delete an access rule.

---

## Enrichments

Log enrichment rules.

### list_enrichments

List all enrichments.

### get_enrichments

Get enrichment details.

### create_enrichment

Create an enrichment.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Enrichment name |
| `description` | string | No | Description |
| `source_type` | string | Yes | Enrichment source type |
| `enrichment_type` | string | Yes | Type of enrichment |

### update_enrichment

Update an enrichment.

### delete_enrichment

Delete an enrichment.

---

## Views

Saved query views.

### list_views

List all views.

### get_view

Get view details.

### create_view

Create a new view.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | View name |
| `search_query` | string | Yes | Saved query |
| `time_selection` | object | No | Time range settings |
| `filters` | object | No | Applied filters |
| `folder_id` | string | No | Parent folder |

### replace_view

Replace a view entirely.

### delete_view

Delete a view.

---

## View Folders

### list_view_folders

List all view folders.

### get_view_folder

Get folder details.

### create_view_folder

Create a new folder.

### replace_view_folder

Replace a folder.

### delete_view_folder

Delete a folder.

---

## Streams

Data streaming configuration.

### list_streams

List all streams.

### get_stream

Get stream details.

### create_stream

Create a new stream.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Stream name |
| `is_active` | boolean | No | Whether enabled |
| `dpxl_expression` | string | Yes | Filter expression |
| `compression_type` | string | No | Compression type |

### update_stream

Update a stream.

### delete_stream

Delete a stream.

---

## Data Usage

### export_data_usage

Export data usage metrics.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `from_date` | string | Start date |
| `to_date` | string | End date |
| `granularity` | string | `daily`, `weekly`, `monthly` |

### update_data_usage_metrics_export_status

Update export settings.

---

## Event Stream Targets

### get_event_stream_targets

List event stream targets.

### create_event_stream_target

Create a new target.

**Key Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Target name |
| `type` | string | Yes | Target type |
| `configuration` | object | Yes | Target configuration |

### update_event_stream_target

Update a target.

### delete_event_stream_target

Delete a target.

---

## AI Helpers

### explain_query

Explain what a DataPrime/Lucene query does.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Query to explain |

### suggest_alert

**SRE-grade alert recommendations** based on industry best practices.

**Key Features:**
- Automatic methodology selection (RED for services, USE for resources)
- Multi-window burn rate alerting for SLO-based monitoring
- Severity classification based on user impact (P1/P2/P3)

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `service_type` | string | `web_service`, `database`, `cache`, `message_queue`, etc. |
| `slo_target` | number | SLO target (e.g., 0.999 for 99.9%) |
| `slo_window_days` | integer | SLO window (default: 30) |
| `criticality_tier` | string | `tier1_critical`, `tier2_important`, `tier3_standard` |
| `is_user_facing` | boolean | Whether service affects end users |
| `use_case` | string | Natural language description |
| `enable_burn_rate` | boolean | Enable burn rate alerting |

**Example:**
```json
{
  "service_type": "web_service",
  "slo_target": 0.999,
  "is_user_facing": true,
  "use_case": "high error rate on checkout API"
}
```

**Output:** Alert configurations with burn rate thresholds, severity, runbook templates.

### get_audit_log

Get audit log entries for the account.

---

## Query Intelligence

### get_query_templates

Get pre-built query templates for common use cases.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `category` | string | Template category |
| `use_case` | string | Specific use case |

### validate_query

Validate query syntax before execution.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Query to validate |
| `syntax` | string | No | Expected syntax type |

### estimate_query_cost

Estimate query resource consumption.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Query to estimate |
| `start_date` | string | No | Time range start |
| `end_date` | string | No | Time range end |

---

## Workflow Automation

### investigate_incident

Guided incident investigation workflow.

**What it does:**
1. Queries recent error logs
2. Analyzes error patterns and trends
3. Identifies top error sources
4. Provides root cause hypotheses
5. Suggests remediation actions

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `application` | string | Application to investigate |
| `time_range` | string | `15m`, `1h`, `6h`, `24h`, `7d` |
| `severity` | string | `warning`, `error`, `critical` |
| `keyword` | string | Additional search term |

**Example:**
```json
{
  "application": "api-gateway",
  "time_range": "1h",
  "severity": "error"
}
```

### smart_investigate

Autonomous incident investigation that thinks like a senior SRE.

**Investigation Modes:**
- **global**: System-wide health scan
- **component**: Deep dive into a specific service
- **flow**: Trace a request across service boundaries

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `application` | string | Target application |
| `trace_id` | string | For flow-mode investigation |
| `correlation_id` | string | Alternative to trace_id |
| `time_range` | string | Investigation window |
| `generate_assets` | boolean | Generate alerts/dashboards |
| `max_queries` | integer | Max queries to execute (1-10) |

### health_check

Quick system health overview with log clustering.

**What it does:**
1. Queries recent error and warning logs
2. Clusters similar log patterns using semantic templates
3. Identifies root cause categories (TIMEOUT, MEMORY_PRESSURE, etc.)
4. Provides system health summary

---

## Root Cause Analysis

Advanced RCA tools implementing SOTA 2025 patterns for causal discovery and trace-log cohesion.

### analyze_log_delta

Analyze log volume and pattern changes between time windows.

**When to use:** Identify what changed between healthy and unhealthy states. Implements LogAssist/LogBatcher 2025 patterns for semantic log clustering.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | DataPrime query to filter logs |
| `baseline_start` | string | Yes | RFC3339 start of baseline window |
| `baseline_end` | string | Yes | RFC3339 end of baseline window |
| `comparison_start` | string | Yes | RFC3339 start of comparison window |
| `comparison_end` | string | Yes | RFC3339 end of comparison window |
| `min_change_percent` | number | No | Minimum change threshold (default: 20%) |

**Example:**
```json
{
  "query": "source logs | filter $l.applicationname == 'api-gateway'",
  "baseline_start": "2024-01-15T09:00:00Z",
  "baseline_end": "2024-01-15T10:00:00Z",
  "comparison_start": "2024-01-15T10:00:00Z",
  "comparison_end": "2024-01-15T11:00:00Z"
}
```

**Output:**
- Clustered log patterns with semantic templates
- Volume change percentages per cluster
- Root cause categories (TIMEOUT, MEMORY_PRESSURE, NETWORK_FAILURE, etc.)
- New patterns that emerged in comparison window
- Disappeared patterns from baseline

**Key Features:**
- Log template extraction (normalizes UUIDs, IPs, timestamps, etc.)
- Automatic root cause inference from templates
- Verification traces for reasoning models (System 2)
- sync.Pool optimization for high-throughput scenarios

---

### get_trace_context

Retrieve and analyze all logs associated with a trace ID.

**When to use:** Understand the full context of a distributed request flow. Provides trace-log cohesion by correlating spans across services.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `trace_id` | string | Yes | Trace ID to look up |
| `start_date` | string | No | RFC3339 start time |
| `end_date` | string | No | RFC3339 end time |

**Example:**
```json
{
  "trace_id": "abc123def456789012345678",
  "start_date": "2024-01-15T10:00:00Z",
  "end_date": "2024-01-15T11:00:00Z"
}
```

**Output:**
- Timeline of events sorted by timestamp
- Services involved (in order of appearance)
- Error summary with counts per service
- Total trace duration
- Span relationships

**Key Features:**
- Automatic service extraction from various field formats
- Severity-based error tracking
- Cross-service correlation

---

### analyze_causal_chain

Build causal graph from clustered log patterns to identify root causes.

**When to use:** Determine the likely root cause from a set of error patterns. Implements AURORA/RCD 2025 causal discovery algorithms.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | DataPrime query to filter logs |
| `start_date` | string | Yes | RFC3339 start time |
| `end_date` | string | Yes | RFC3339 end time |
| `include_context` | boolean | No | Include additional timeline context |

**Example:**
```json
{
  "query": "source logs | filter $m.severity >= 5",
  "start_date": "2024-01-15T10:00:00Z",
  "end_date": "2024-01-15T11:00:00Z",
  "include_context": true
}
```

**Output:**
- Causal graph with nodes (log clusters) and edges (relationships)
- Root cause candidates ranked by confidence score
- Error propagation path across services
- Timeline of cluster appearances

**Confidence Scoring:**
- **Temporal ordering**: Earlier clusters score higher as potential causes
- **Fundamental causes**: MEMORY_PRESSURE, NETWORK_FAILURE, STORAGE_FAILURE score higher than symptoms like TIMEOUT
- **Service spread**: Issues affecting fewer services (more localized) score higher

**Root Cause Categories:**
| Category | Typical Patterns |
|----------|------------------|
| MEMORY_PRESSURE | OOM, killed process, heap exhausted |
| TIMEOUT | Connection timeout, request timeout |
| NETWORK_FAILURE | Connection refused, socket error |
| AUTH_FAILURE | Permission denied, unauthorized |
| STORAGE_FAILURE | Disk full, no space left |
| RATE_LIMITED | Rate limit exceeded, throttled |
| DNS_FAILURE | DNS resolution failed |
| TLS_FAILURE | Certificate expired, TLS handshake |
| DATABASE_FAILURE | Deadlock, connection pool exhausted |
| K8S_ORCHESTRATION | Pod evicted, node pressure |

---

### generate_rca_document

Generate a structured Root Cause Analysis document template.

**When to use:** After completing incident investigation to create a formal RCA report for post-mortem review and stakeholder communication.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_title` | string | Yes | Short title describing the incident |
| `incident_id` | string | No | Incident tracking ID (auto-generated if not provided) |
| `incident_start` | string | Yes | When the incident started (RFC3339) |
| `incident_end` | string | No | When resolved (omit if ongoing) |
| `severity` | string | No | SEV1, SEV2, SEV3, SEV4 (default: SEV2) |
| `affected_services` | array | No | List of affected services |
| `root_cause_category` | string | No | Primary root cause from analysis |
| `root_cause_description` | string | No | Detailed root cause description |
| `error_patterns` | array | No | Error patterns from analyze_log_delta |
| `timeline_events` | array | No | Timeline events from investigation |
| `include_template_sections` | boolean | No | Include blank sections to fill (default: true) |

**Example:**
```json
{
  "incident_title": "API Gateway Timeout Spike",
  "incident_start": "2024-01-15T10:00:00Z",
  "incident_end": "2024-01-15T11:30:00Z",
  "severity": "SEV2",
  "affected_services": ["api-gateway", "auth-service"],
  "root_cause_category": "NETWORK_FAILURE",
  "root_cause_description": "Network partition between api-gateway and auth-service due to misconfigured network policy"
}
```

**Output:** Markdown document with sections:
1. Executive Summary
2. Affected Services
3. Impact Assessment
4. Incident Timeline
5. Root Cause Analysis (5 Whys)
6. Log Evidence
7. Contributing Factors
8. Corrective Actions
9. Prevention Measures
10. Lessons Learned

**Key Features:**
- Pre-filled with data from RCA tools
- Template sections with guidance for manual completion
- Industry-standard 5 Whys methodology
- Checklist format for action items

---

## Meta Tools

### discover_tools

Comprehensive tool discovery with semantic search.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `intent` | string | What you want to accomplish |
| `category` | string | Filter by category |

### search_tools

Token-efficient tool search returning minimal info.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | string | Natural language search |
| `category` | string | Filter by category |
| `limit` | integer | Max results (default: 10) |

### describe_tools

Get full schema for specific tools.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tools` | array | Yes | Tool names to describe |

### session_context

Manage session state and preferences.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `action` | string | `get`, `set`, `clear` |
| `preferences` | object | User preferences to set |

### list_tool_categories_brief

Get a brief overview of all tool categories.

---

## Best Practices

### Query Best Practices

1. **Use appropriate tiers:**
   - `frequent_search` for real-time monitoring (last 24h)
   - `archive` for historical analysis

2. **Include time bounds:**
   ```
   source logs
   | filter $m.timestamp >= now() - 1h
   | filter $m.severity >= 5
   ```

3. **Use limits to control costs:**
   ```
   source logs | filter ... | limit 1000
   ```

4. **Build queries incrementally:**
   Use `build_query` → `validate_query` → `query_logs`

### Alert Best Practices

1. **Use `suggest_alert` for SRE-grade recommendations:**
   - Provides burn rate alerting for SLO-based monitoring
   - Auto-selects RED (services) vs USE (resources) methodology
   - Includes runbook templates

2. **Define SLOs before alerts:**
   ```json
   {
     "slo_target": 0.999,
     "is_user_facing": true
   }
   ```

3. **Severity classification:**
   - **P1**: User-facing + high burn rate (14.4x)
   - **P2**: Important + moderate burn rate (3x)
   - **P3**: Non-critical, ticket-based response

4. **Always include runbooks:**
   Alerts without runbooks lead to slower incident response.

### Investigation Best Practices

1. **Start with `smart_investigate` for complex issues:**
   - Uses multi-phase query strategy
   - Applies heuristic rules automatically
   - Suggests next steps

2. **Use `investigate_incident` for guided approach:**
   - Good for learning the system
   - Provides step-by-step analysis

3. **Set appropriate time ranges:**
   - `15m` for active incidents
   - `1h` for recent issues
   - `24h` for pattern analysis

### Performance Best Practices

1. **Use `estimate_query_cost` before large queries**

2. **Use background queries for:**
   - Queries spanning > 24 hours
   - Queries without limits on large datasets
   - Queries during peak usage times

3. **Leverage session context:**
   - Set preferred time ranges
   - Configure default applications

### Security Best Practices

1. **Use data access rules** to control log visibility

2. **Audit webhook configurations** regularly

3. **Rotate API keys** every 90 days

4. **Use service IDs** instead of personal API keys in production

---

## Tool Categories Reference

| Category | Description |
|----------|-------------|
| `query` | Log querying and search |
| `alert` | Alert instance management |
| `alerting` | Alert definitions and templates |
| `dashboard` | Dashboard management |
| `policy` | Retention and routing policies |
| `webhook` | Outgoing webhook notifications |
| `e2m` | Events to metrics conversion |
| `rule_group` | Log parsing rules |
| `data_access` | Access control rules |
| `enrichment` | Log enrichment |
| `view` | Saved query views |
| `stream` | Data streaming |
| `ingestion` | Log ingestion |
| `workflow` | Automated workflows |
| `data_usage` | Usage metrics |
| `ai_helper` | AI-powered analysis |
| `rca` | Root cause analysis |
| `meta` | Tool discovery and session |

---

## DataPrime Quick Reference

### Field Prefixes

| Prefix | Description | Examples |
|--------|-------------|----------|
| `$l.` | Labels | `$l.applicationname`, `$l.subsystemname` |
| `$m.` | Metadata | `$m.severity`, `$m.timestamp` |
| `$d.` | Data (JSON fields) | `$d.status_code`, `$d.user_id` |

### Severity Levels

| Level | Value | Description |
|-------|-------|-------------|
| DEBUG | 1 | Debug information |
| VERBOSE | 2 | Verbose output |
| INFO | 3 | Informational |
| WARNING | 4 | Warning |
| ERROR | 5 | Error |
| CRITICAL | 6 | Critical error |

### Common Operators

```dataprime
# Filtering
filter $m.severity >= 5
filter $l.applicationname == 'api-gateway'
filter $d.message.contains('timeout')

# Aggregation
groupby $l.applicationname aggregate count() as error_count
sortby error_count desc

# Limiting
limit 100
```

---

## Support

- **Issues**: [GitHub Issues](https://github.com/tareqmamari/cloud-logs-mcp/issues)
- **IBM Cloud Logs Docs**: https://cloud.ibm.com/docs/cloud-logs
- **IBM Cloud Logs API**: https://cloud.ibm.com/apidocs/logs-service-api
