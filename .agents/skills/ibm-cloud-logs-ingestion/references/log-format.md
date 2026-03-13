# Log Entry Format Reference

## JSON Schema

Each log entry in the ingestion request body conforms to the following schema:

```json
{
  "type": "object",
  "properties": {
    "applicationName": {
      "type": "string",
      "description": "Name of the application generating the log. Aliases accepted: namespace, app, application, service, app_name, application_name. Aliases are resolved and normalized to applicationName before sending."
    },
    "subsystemName": {
      "type": "string",
      "description": "Name of the subsystem within the application. Aliases accepted: component, resource, subsystem, module, component_name, subsystem_name, resource_name. Aliases are resolved and normalized to subsystemName before sending."
    },
    "severity": {
      "type": "integer",
      "description": "Log severity level",
      "minimum": 1,
      "maximum": 6,
      "enum_descriptions": {
        "1": "Debug - Detailed debugging information",
        "2": "Verbose - Verbose output for troubleshooting",
        "3": "Info - Informational messages (default)",
        "4": "Warning - Warning conditions",
        "5": "Error - Error conditions",
        "6": "Critical - Critical/fatal conditions"
      }
    },
    "text": {
      "type": "string",
      "description": "The log message text"
    },
    "timestamp": {
      "type": "number",
      "description": "Unix timestamp with nanosecond precision (e.g., 1699564800.123456789). If not provided, the server generates the current time as float64(unix_seconds) + float64(nanoseconds)/1e9."
    },
    "json": {
      "type": "object",
      "description": "Optional JSON object containing arbitrary structured log data. Any valid JSON object is accepted.",
      "additionalProperties": true
    }
  },
  "required": ["applicationName", "subsystemName", "severity", "text"]
}
```

## Request Body Schema

The ingestion endpoint accepts an **array** of log entry objects:

```json
{
  "type": "array",
  "items": { "$ref": "#/log-entry" },
  "minItems": 1,
  "maxItems": 1000
}
```

Maximum batch size is **1000** entries per request (enforced by `MaxIngestionBatchSize`).

## Field Alias Resolution

The ingestion tool resolves field name aliases before sending the request to the API. Aliases are checked in priority order; the first match wins:

### applicationName aliases (checked in order)

1. `namespace`
2. `app`
3. `application`
4. `service`
5. `app_name`
6. `application_name`

### subsystemName aliases (checked in order)

1. `component`
2. `resource`
3. `subsystem`
4. `module`
5. `component_name`
6. `subsystem_name`
7. `resource_name`

When an alias is matched, it is renamed to the canonical field name and the alias key is removed from the entry to avoid duplicate fields.

## Endpoint

```
POST https://{instance-id}.ingress.{region}.logs.cloud.ibm.com/logs/v1/singles
```

The URL is derived from the management API URL by replacing `.api.` with `.ingress.`.

## Complete Example

```json
[
  {
    "applicationName": "api-gateway",
    "subsystemName": "auth",
    "severity": 5,
    "timestamp": 1699564800.123456789,
    "text": "Authentication failed for user john@example.com",
    "json": {
      "user_id": "12345",
      "ip_address": "192.168.1.100",
      "error_code": "AUTH_FAILED"
    }
  },
  {
    "applicationName": "api-gateway",
    "subsystemName": "auth",
    "severity": 3,
    "text": "User login successful",
    "json": {
      "user_id": "67890",
      "ip_address": "10.0.0.50"
    }
  }
]
```

Note: the second entry omits `timestamp`, which will be auto-generated at ingestion time.
