---
name: ibm-cloud-logs-ingestion
description: >
  Ingest, parse, and enrich logs in IBM Cloud Logs. Activate when sending
  logs, setting up pipelines, configuring parsing rules, or testing
  ingestion. Covers log entry format, parsing rule groups, enrichments,
  and event streaming.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: ingestion
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Ingestion Skill

## When to Activate

Use this skill when the user:
- Wants to send or ingest log entries into IBM Cloud Logs
- Needs to configure parsing rules or rule groups to transform incoming logs
- Asks about log entry format, required fields, or severity levels
- Wants to set up enrichments (geo-IP, custom lookups) on ingested logs
- Needs to create or manage event stream targets (Kafka / IBM Event Streams)
- Is testing an ingestion pipeline end-to-end
- Asks about the `.ingress.` endpoint or batch ingestion limits
- Wants to discover available fields in their log data before writing parsing rules

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

**Ingestion note:** Log ingestion uses a different endpoint: `https://{instance-id}.ingress.{region}.logs.cloud.ibm.com` (replace `.api.` with `.ingress.` in the service URL).

## Log Entry Format

Each log entry sent to the ingestion endpoint is a JSON object with the following fields:

### Required Fields

| Field | Type | Description |
|---|---|---|
| `applicationName` | string | Name of the application generating the log. Aliases: `namespace`, `app`, `application`, `service`, `app_name`, `application_name` |
| `subsystemName` | string | Name of the subsystem within the application. Aliases: `component`, `resource`, `subsystem`, `module`, `component_name`, `subsystem_name`, `resource_name` |
| `severity` | integer | Log severity level (see table below) |
| `text` | string | The log message text |

### Optional Fields

| Field | Type | Description |
|---|---|---|
| `timestamp` | number | Unix timestamp with nanosecond precision (e.g., `1699564800.123456789`). Auto-generated as current time if omitted. |
| `json` | object | Arbitrary structured JSON data attached to the log entry |

### Severity Levels

| Value | Name | Description |
|---|---|---|
| 1 | Debug | Detailed debugging information |
| 2 | Verbose | Verbose output for troubleshooting |
| 3 | Info | Informational messages (default) |
| 4 | Warning | Warning conditions |
| 5 | Error | Error conditions |
| 6 | Critical | Critical/fatal conditions |

> **Note:** The ingestion API uses numeric severity 1-6. Once ingested, logs are queried in DataPrime using named values on a 0-5 scale: `$m.severity >= ERROR`. See the [Query Skill](../ibm-cloud-logs-query/SKILL.md) for DataPrime severity handling.

For a complete JSON schema, see [references/log-format.md](references/log-format.md).

## Ingestion Endpoint

Logs are sent to the **ingress** subdomain, which is separate from the management API:

```
POST https://{instance-id}.ingress.{region}.logs.cloud.ibm.com/logs/v1/singles
```

The client converts the standard API URL by replacing `.api.` with `.ingress.`:

```
https://{instance-id}.api.{region}.logs.cloud.ibm.com
  -->
https://{instance-id}.ingress.{region}.logs.cloud.ibm.com
```

**Key details:**
- Content-Type: `application/json`
- Authentication: IAM bearer token (same credentials as the management API)
- The request body is a JSON **array** of log entry objects
- Maximum batch size: **1000** entries per request (`MaxIngestionBatchSize`)
- Requests exceeding the batch limit are rejected; split into smaller batches
- Timestamps are automatically added for entries that omit the `timestamp` field
- Alias field names (e.g., `namespace` for `applicationName`) are resolved and normalized before sending

## Parsing Rule Groups

Rule groups transform incoming logs during ingestion. Each group contains ordered subgroups, and each subgroup contains ordered rules.

### Rule Group Structure

```
Rule Group (name, enabled, order, rule_matchers)
  -> Rule Subgroup (enabled, order)
    -> Rule (name, source_field, parameters, enabled, order)
```

- **name**: Display name (1-255 characters)
- **description**: Purpose description (up to 4096 characters)
- **enabled**: Boolean toggle
- **order**: Execution order (lower runs first, 0-4294967295)
- **rule_matchers**: Optional filters to limit which logs the group processes

### Rule Matchers

Control which logs a rule group applies to:

| Matcher | Field | Description |
|---|---|---|
| `application_name` | `value` (string) | Match by application name |
| `subsystem_name` | `value` (string) | Match by subsystem name |
| `severity` | `value` (enum) | Match by severity: `debug_or_unspecified`, `verbose`, `info`, `warning`, `error`, `critical` |

### Rule Types

Each rule has a `parameters` object containing exactly one of these keys:

| Rule Type | Purpose |
|---|---|
| `extract_parameters` | Extract fields using regex, keeping the original log |
| `parse_parameters` | Parse log into JSON fields. Set `destination_field = source_field` for in-place transform |
| `json_extract_parameters` | Extract a JSON field to metadata |
| `replace_parameters` | Replace text matching a regex pattern |
| `allow_parameters` | Allow logs matching a regex (drop non-matching) |
| `block_parameters` | Block logs matching a regex (keep non-matching) |
| `extract_timestamp_parameters` | Extract timestamp from log content |
| `remove_fields_parameters` | Remove specific JSON fields |
| `json_stringify_parameters` | Convert JSON object to string |
| `json_parse_parameters` | Parse string into JSON object |

### Valid Source Fields

The `source_field` in each rule must start with a valid prefix:

- `text` -- Main log message field
- `text.log` -- Nested log field from JSON logs (common for Kubernetes)
- `text.<fieldname>` -- Any nested field under text
- `json.<fieldname>` -- Custom JSON fields
- `kubernetes.<fieldname>` -- Kubernetes metadata
- `log.<fieldname>` -- Log metadata

For full rule configuration details, see [references/parsing-rules.md](references/parsing-rules.md).

### API Endpoints

| Operation | Method | Path |
|---|---|---|
| List rule groups | GET | `/v1/rule_groups` |
| Get rule group | GET | `/v1/rule_groups/{id}` |
| Create rule group | POST | `/v1/rule_groups` |
| Update rule group | PUT | `/v1/rule_groups/{id}` |
| Delete rule group | DELETE | `/v1/rule_groups/{id}` |

## Enrichment Rules

Enrichments add context to incoming logs automatically. Two enrichment types are supported.

### Enrichment Types

**geo_ip** -- Add geographic information based on IP address fields:
- Resolves IP addresses to country, city, latitude, longitude
- Configure `field_name` to point at the source IP field (e.g., `json.client_ip`)

**custom_enrichment** -- Add fields from lookup tables:
- Maps a log field value to additional context from a lookup table
- Configure `field_name` (source), `enrichment_type: custom_enrichment`, and `custom_enrichment_config.lookup_table_id`

### Enrichment Configuration

```json
{
  "name": "Client IP Geolocation",
  "description": "Add geographic data from client IP addresses",
  "field_name": "json.client_ip",
  "enrichment_type": "geo_ip"
}
```

```json
{
  "name": "Customer Tier Lookup",
  "description": "Enrich logs with customer tier information",
  "field_name": "json.customer_id",
  "enrichment_type": "custom_enrichment",
  "custom_enrichment_config": {
    "lookup_table_id": "customer-tiers-table"
  }
}
```

### API Endpoints

| Operation | Method | Path |
|---|---|---|
| List enrichments | GET | `/v1/enrichments` |
| Get enrichments | GET | `/v1/enrichments` |
| Create enrichment | POST | `/v1/enrichments` |
| Update enrichment | PUT | `/v1/enrichments/{id}` |
| Delete enrichment | DELETE | `/v1/enrichments/{id}` |

For all enrichment types and options, see [references/enrichment-types.md](references/enrichment-types.md).

## Event Streams

Event streams forward ingested logs to external systems in real time, primarily IBM Event Streams (Kafka).

### Stream Configuration

Each event stream target requires:
- **name**: Display name (1-4096 characters)
- **dpxl_expression**: DPXL filter expression (e.g., `<v1>contains(kubernetes.labels.app, "frontend")`)
- **is_active**: Boolean toggle (default: true)
- **compression_type**: One of `gzip`, `snappy`, `lz4`, `zstd`, or `unspecified`
- **ibm_event_streams**: Kafka configuration object with `brokers` (comma-separated) and `topic`

### API Endpoints

| Operation | Method | Path |
|---|---|---|
| List streams | GET | `/v1/streams` |
| Create stream | POST | `/v1/streams` |
| Update stream | PUT | `/v1/streams/{stream_id}` |
| Delete stream | DELETE | `/v1/streams/{stream_id}` |

### Example: Create a Stream Target

```json
{
  "name": "Frontend Logs to Kafka",
  "dpxl_expression": "<v1>contains(kubernetes.labels.app, \"frontend\")",
  "is_active": true,
  "compression_type": "gzip",
  "ibm_event_streams": {
    "brokers": "broker-1.kafka.svc:9093,broker-2.kafka.svc:9093",
    "topic": "frontend-logs"
  }
}
```

## Testing Ingestion

Follow this step-by-step procedure to verify your ingestion pipeline works end-to-end.

### Step 1: Send Test Logs

Send a small batch of test entries via curl to the ingress endpoint:

```json
{
  "logs": [
    {
      "applicationName": "ingestion-test",
      "subsystemName": "smoke-test",
      "severity": 3,
      "text": "Ingestion pipeline test entry",
      "json": {
        "test_id": "test-001",
        "environment": "development"
      }
    }
  ]
}
```

### Step 2: Verify Ingestion

Wait 10-30 seconds, then query for the test logs using `ibmcloud logs query`:

```
source logs | filter $l.applicationname == 'ingestion-test' | limit 10
```

### Step 3: Verify Parsing Rules

If parsing rules are configured, check that fields were extracted:

1. Run `ibmcloud logs rule-groups --output json` to see active rules
2. Query for specific extracted fields to confirm they exist
3. Query for specific extracted fields to confirm they exist

### Step 4: Verify Enrichments

If enrichments are configured:

1. Run `ibmcloud logs enrichments --output json` to confirm enrichments are active
2. Query for enriched fields (e.g., geo-IP data) on your test logs

### Step 5: Verify Event Streams

If event stream targets are configured:

1. Run `ibmcloud logs event-stream-targets --output json` to confirm streams are active
2. Check the downstream Kafka topic for your test log entries

See [scripts/send-test-logs.sh](scripts/send-test-logs.sh) for a standalone test script and [assets/sample-logs.json](assets/sample-logs.json) for sample payloads.

## Using the IBM Cloud CLI

Parsing rules, enrichments, and event streams can be managed via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# List parsing rule groups
ibmcloud logs rule-groups --output json

# Create a rule group from a JSON file
ibmcloud logs rule-group-create --prototype @rule-group.json

# List enrichments
ibmcloud logs enrichments --output json

# Create an enrichment
ibmcloud logs enrichment-create --prototype @enrichment.json

# List event stream targets
ibmcloud logs event-stream-targets --output json

# Verify ingested logs via query
ibmcloud logs query \
  --query 'source logs | filter $l.applicationname == '\''ingestion-test'\'' | limit 10'
```

For sending test logs, use the [send-test-logs.sh](scripts/send-test-logs.sh) script which handles the ingress endpoint and batch formatting.

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Use `--output-file` for test log results.** When using `send-test-logs.sh`, pass `--output-file` to capture request/response details to a file rather than dumping them into the conversation.
- **Write parsing rule and enrichment configs to files.** When generating rule group JSON or enrichment configurations, write them to a file (e.g., `rule-group.json`, `enrichment.json`) instead of pasting inline.
- **Write generated log batches to files.** Sample logs from `assets/` are small, but any generated log batches for testing should be written to a file (e.g., `test-batch.json`) to avoid cluttering the context.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [Log Format Reference](references/log-format.md) -- Complete log entry JSON schema
- [Parsing Rules Reference](references/parsing-rules.md) -- Rule types and configuration options
- [Enrichment Types Reference](references/enrichment-types.md) -- Enrichment types and configuration
- [Ingestion Test Script](scripts/send-test-logs.sh) -- Standalone ingestion test script
- [Sample Log Entries](assets/sample-logs.json) -- Sample log entries

> **Windows note:** Scripts require bash (available via [Git for Windows](https://gitforwindows.org/) or WSL).
