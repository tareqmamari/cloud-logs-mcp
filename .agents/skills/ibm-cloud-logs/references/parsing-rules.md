# Parsing Rules Reference

## Rule Group Structure

A rule group is the top-level container for parsing rules. It is submitted as a JSON object to the `/v1/rule_groups` endpoint.

```json
{
  "name": "string (1-255 chars, required)",
  "description": "string (1-4096 chars, optional)",
  "enabled": true,
  "order": 1,
  "rule_matchers": [],
  "rule_subgroups": []
}
```

### Required fields

- `name` -- Display name for the rule group
- `rule_subgroups` -- Array of subgroups (at least one)

### Optional fields

- `description` -- Purpose description (max 4096 chars)
- `enabled` -- Toggle on/off (default: true)
- `order` -- Execution priority (0-4294967295, lower runs first)
- `rule_matchers` -- Array of matchers to filter which logs this group processes

## Rule Matchers

Each matcher is an object with exactly one key. Multiple matchers in the array are ANDed together.

### application_name

```json
{ "application_name": { "value": "nginx" } }
```

### subsystem_name

```json
{ "subsystem_name": { "value": "access-log" } }
```

### severity

```json
{ "severity": { "value": "error" } }
```

Valid severity values: `debug_or_unspecified`, `verbose`, `info`, `warning`, `error`, `critical`.

## Rule Subgroups

Each subgroup contains an ordered list of rules:

```json
{
  "enabled": true,
  "order": 1,
  "rules": []
}
```

Required fields: `rules` (at least one rule), `order`.

## Rules

Each rule specifies a source field and a parameters object:

```json
{
  "name": "string (1-4096 chars)",
  "description": "string (1-4096 chars)",
  "source_field": "text.log",
  "enabled": true,
  "order": 1,
  "parameters": {
    "<rule_type>": { ... }
  }
}
```

Required fields: `source_field`, `parameters`, `enabled`, `order`.

### Valid Source Fields

Source fields must start with one of these prefixes:

| Prefix | Description | Examples |
|---|---|---|
| `text` | Main log message | `text`, `text.log`, `text.message` |
| `json` | Custom JSON fields | `json.request_id`, `json.status_code` |
| `kubernetes` | Kubernetes metadata | `kubernetes.labels.app`, `kubernetes.namespace` |
| `log` | Log metadata | `log.file.path` |

The `text.log` field is especially common for Kubernetes environments where container logs are nested inside a JSON wrapper.

## Rule Types

### extract_parameters

Extract fields using a regex pattern while keeping the original log intact.

```json
{
  "parameters": {
    "extract_parameters": {
      "rule": "(?P<method>GET|POST|PUT|DELETE) (?P<path>/\\S+) (?P<status>\\d{3})"
    }
  }
}
```

- `rule` -- Regex pattern with named capture groups (`(?P<name>...)`)

### parse_parameters

Parse a log field into structured JSON. Set `destination_field` equal to `source_field` for in-place transformation.

```json
{
  "parameters": {
    "parse_parameters": {
      "destination_field": "text.log",
      "rule": "(?P<timestamp>\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}) \\[(?P<level>\\w+)\\] (?P<pid>\\d+)#(?P<tid>\\d+): \\*(?P<cid>\\d+) (?P<message>.*), client: (?P<client_ip>[\\d\\.]+), server: (?P<server>[^,]+)"
    }
  }
}
```

- `destination_field` -- Where to write parsed output
- `rule` -- Regex pattern with named capture groups

### json_extract_parameters

Extract a specific JSON field to metadata.

```json
{
  "parameters": {
    "json_extract_parameters": {
      "destination_field": "metadata.request_id",
      "rule": "request_id"
    }
  }
}
```

### replace_parameters

Replace text matching a regex pattern.

```json
{
  "parameters": {
    "replace_parameters": {
      "destination_field": "text",
      "rule": "\\b\\d{4}-\\d{4}-\\d{4}-\\d{4}\\b",
      "replace_new_val": "[REDACTED]"
    }
  }
}
```

- `rule` -- Regex pattern to match
- `replace_new_val` -- Replacement string
- `destination_field` -- Field to write the result to

### allow_parameters

Allow only logs that match a regex; drop all non-matching logs.

```json
{
  "parameters": {
    "allow_parameters": {
      "rule": "^(GET|POST|PUT|DELETE)"
    }
  }
}
```

### block_parameters

Block (drop) logs that match a regex; keep all non-matching logs.

```json
{
  "parameters": {
    "block_parameters": {
      "rule": "health_check|readiness_probe"
    }
  }
}
```

### extract_timestamp_parameters

Extract a timestamp from log content and use it as the log entry timestamp.

```json
{
  "parameters": {
    "extract_timestamp_parameters": {
      "standard": "strftime",
      "format": "%Y-%m-%dT%H:%M:%S.%fZ"
    }
  }
}
```

### remove_fields_parameters

Remove specific JSON fields from the log entry.

```json
{
  "parameters": {
    "remove_fields_parameters": {
      "fields": ["json.sensitive_data", "json.internal_id"]
    }
  }
}
```

### json_stringify_parameters

Convert a JSON object field to its string representation.

```json
{
  "parameters": {
    "json_stringify_parameters": {
      "destination_field": "text.payload_string"
    }
  }
}
```

### json_parse_parameters

Parse a string field into a JSON object.

```json
{
  "parameters": {
    "json_parse_parameters": {
      "destination_field": "json.parsed_payload"
    }
  }
}
```

## Complete Example

This rule group parses nginx error logs from Kubernetes, matching only the `nginx` subsystem:

```json
{
  "rule_group": {
    "name": "Nginx Error Log Parser",
    "description": "Parse nginx error logs into structured JSON",
    "enabled": true,
    "order": 1,
    "rule_matchers": [
      {
        "subsystem_name": {
          "value": "nginx"
        }
      }
    ],
    "rule_subgroups": [
      {
        "enabled": true,
        "order": 1,
        "rules": [
          {
            "name": "Parse Nginx Error from text.log",
            "description": "Extract fields from nginx error log in text.log field (common for Kubernetes logs)",
            "source_field": "text.log",
            "enabled": true,
            "order": 1,
            "parameters": {
              "parse_parameters": {
                "destination_field": "text.log",
                "rule": "(?P<timestamp>\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}) \\[(?P<level>\\w+)\\] (?P<pid>\\d+)#(?P<tid>\\d+): \\*(?P<cid>\\d+) (?P<message>.*), client: (?P<client_ip>[\\d\\.]+), server: (?P<server>[^,]+)"
              }
            }
          }
        ]
      }
    ]
  }
}
```

## Field Discovery

Before writing parsing rules, use the `discover_log_fields` tool to see what fields exist in your logs. It queries recent log entries and reports available field paths grouped by prefix (text, json, kubernetes, other).

You can also use `test_rule_pattern` to validate regex patterns against sample log text before creating a rule group.
