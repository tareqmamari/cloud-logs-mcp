# Ingestion Guide

> Domain guide for IBM Cloud Logs ingestion. For inline essentials
> (DataPrime syntax, common mistakes), see [SKILL.md](../SKILL.md).

## Log Entry Format

### Required Fields

| Field | Type | Description |
|---|---|---|
| `applicationName` | string | Application name. Aliases: `namespace`, `app`, `application`, `service` |
| `subsystemName` | string | Subsystem name. Aliases: `component`, `resource`, `subsystem`, `module` |
| `severity` | integer | Log severity level (1-6, see below) |
| `text` | string | Log message text |

### Optional Fields

| Field | Type | Description |
|---|---|---|
| `timestamp` | number | Unix timestamp with nanosecond precision. Auto-generated if omitted. |
| `json` | object | Arbitrary structured JSON data |

### Ingestion Severity Levels

| Value | Name |
|---|---|
| 1 | Debug |
| 2 | Verbose |
| 3 | Info (default) |
| 4 | Warning |
| 5 | Error |
| 6 | Critical |

> **Note:** Ingestion uses numeric 1-6. Queries use named values on 0-5 scale: `$m.severity >= ERROR`.

For complete JSON schema, see [log-format.md](log-format.md).

## Ingestion Endpoint

```
POST https://{instance-id}.ingress.{region}.logs.cloud.ibm.com/logs/v1/singles
```

Replace `.api.` with `.ingress.` in the service URL.

- Content-Type: `application/json`
- Body: JSON **array** of log entry objects
- Maximum batch size: **1000** entries per request
- Alias field names are resolved automatically

## Parsing Rule Groups

### Rule Group Structure

```
Rule Group (name, enabled, order, rule_matchers)
  -> Rule Subgroup (enabled, order)
    -> Rule (name, source_field, parameters, enabled, order)
```

### Rule Matchers

| Matcher | Field | Description |
|---|---|---|
| `application_name` | `value` | Match by application name |
| `subsystem_name` | `value` | Match by subsystem name |
| `severity` | `value` | Match by severity |

### Rule Types

| Rule Type | Purpose |
|---|---|
| `extract_parameters` | Extract fields using regex |
| `parse_parameters` | Parse log into JSON fields |
| `json_extract_parameters` | Extract JSON field to metadata |
| `replace_parameters` | Replace text matching regex |
| `allow_parameters` | Allow logs matching regex (drop non-matching) |
| `block_parameters` | Block logs matching regex |
| `extract_timestamp_parameters` | Extract timestamp from content |
| `remove_fields_parameters` | Remove specific JSON fields |
| `json_stringify_parameters` | Convert JSON to string |
| `json_parse_parameters` | Parse string into JSON |

### Valid Source Fields

- `text`, `text.log`, `text.<fieldname>`, `json.<fieldname>`, `kubernetes.<fieldname>`, `log.<fieldname>`

For full rule configuration, see [parsing-rules.md](parsing-rules.md).

### API Endpoints

| Operation | Method | Path |
|---|---|---|
| List rule groups | GET | `/v1/rule_groups` |
| Create rule group | POST | `/v1/rule_groups` |
| Update rule group | PUT | `/v1/rule_groups/{id}` |
| Delete rule group | DELETE | `/v1/rule_groups/{id}` |

## Enrichment Rules

### Enrichment Types

**geo_ip** -- Add geographic information based on IP address fields.

**custom_enrichment** -- Add fields from lookup tables.

```json
{
  "name": "Client IP Geolocation",
  "field_name": "json.client_ip",
  "enrichment_type": "geo_ip"
}
```

For all enrichment types, see [enrichment-types.md](enrichment-types.md).

## Event Streams

Event streams forward logs to external systems (Kafka / IBM Event Streams) in real time.

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

### Step 1: Send Test Logs
Send a small batch via curl or [send-test-logs.sh](../scripts/send-test-logs.sh).

### Step 2: Verify
Wait 10-30 seconds, then query:
```
source logs | filter $l.applicationname == 'ingestion-test' | limit 10
```

### Step 3-5: Verify Parsing Rules, Enrichments, Event Streams
Check with `ibmcloud logs rule-groups --output json`, `enrichments --output json`, `event-stream-targets --output json`.

## CLI Commands

```bash
ibmcloud logs rule-groups --output json
ibmcloud logs rule-group-create --prototype @rule-group.json
ibmcloud logs enrichments --output json
ibmcloud logs enrichment-create --prototype @enrichment.json
ibmcloud logs event-stream-targets --output json
```

## Deep References

- [Log Format Reference](log-format.md) -- Complete log entry JSON schema
- [Parsing Rules Reference](parsing-rules.md) -- Rule types and configuration
- [Enrichment Types Reference](enrichment-types.md) -- Enrichment types and config
- [Ingestion Test Script](../scripts/send-test-logs.sh) -- Standalone test script
- [Sample Log Entries](../assets/sample-logs.json) -- Sample log entries
