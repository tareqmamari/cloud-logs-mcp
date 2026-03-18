# Remediation Asset Generation

When `generate_assets: true` is passed to `smart_investigate` and findings
exist, the `RemediationGenerator` creates alert and dashboard
configurations from the investigation context.

## Asset Generation Pipeline

The generator receives an `IncidentContext` containing:

- **RootCause** -- synthesized root cause statement from evidence
- **AffectedServices** -- list of impacted service names

It produces an `IncidentResponseAssets` structure with:

1. Alert configuration (Terraform HCL + IBM Cloud Logs JSON)
2. Dashboard configuration (widget definitions + IBM Cloud Logs JSON)
3. SOP recommendations matched to the root cause

## Alert Configuration

### Alert Naming

- Single service: `<service>-error-rate-alert`
- Multiple services: `multiple-services-error-rate-alert`
- Names are sanitized: lowercased, hyphens/spaces/slashes replaced with underscores

### Alert Condition

The condition is built from affected services:

```
(<service_1_filter> || <service_2_filter>) && $m.severity >= 5
```

Each service filter: `$l.applicationname == '<service>'`

If no services are specified, falls back to: `$m.severity >= 5`

### Alert Type

Logs ratio threshold alert:
- **Numerator (query_1):** Error events matching the condition
- **Denominator (query_2):** All events with `$m.severity >= 1`
- **Ratio:** `query_1 / query_2 * 100`
- **Condition:** `MORE_THAN`
- **Threshold:** 5 (5% error rate)
- **Time window:** 5 minutes
- **Notifications:** On triggered and on resolved, every 10 minutes

### Terraform Output

The generator produces a complete `ibm_logs_alert` Terraform resource:

```hcl
resource "ibm_logs_alert" "<resource_name>" {
  name        = "<alert_name>"
  description = "Alert for: <root_cause>"
  severity    = "<severity>"
  is_active   = true

  condition {
    logs_ratio_threshold {
      rules {
        condition {
          condition_type = "MORE_THAN"
          threshold      = 5
          time_window    = "FIVE_MINUTES"
        }
        override {
          priority = "P2"
        }
      }

      query_1 {
        search_query {
          query = "<condition>"
        }
      }

      query_2 {
        search_query {
          query = "$m.severity >= 1"
        }
      }
    }
  }

  notification_groups {
    notifications {
      notify_on       = ["Triggered"]
      integration_id  = var.notification_integration_id
    }
  }
}
```

Severity mapping:

| Input severity | Terraform severity |
|----------------|--------------------|
| critical       | CRITICAL           |
| high           | ERROR              |
| medium         | WARNING            |
| low            | INFO               |

A `notification_integration_id` variable and an alert ID output are
also generated.

### IBM Cloud Logs Alert JSON

The generator also produces a JSON object suitable for direct API use:

```json
{
  "name": "<alert_name>",
  "description": "<description>",
  "is_active": true,
  "severity": "<SEVERITY>",
  "condition": {
    "type": "logs_ratio_threshold",
    "parameters": {
      "threshold": 5,
      "time_window": "5m",
      "query_1": "<condition>",
      "query_2": "$m.severity >= 1",
      "ratio": "query_1 / query_2 * 100",
      "condition": "MORE_THAN",
      "group_by": [],
      "manage_undetected_values": {
        "enable_triggering_on_undetected_values": false
      }
    }
  },
  "notifications": {
    "on_triggered": true,
    "on_resolved": true,
    "notify_every": "10m",
    "channels": []
  }
}
```

## Dashboard Configuration

### Dashboard Naming

- Single service: `<service> - Incident Analysis`
- Multiple services: `System - Incident Analysis`

### Generated Widgets (5)

| # | Title                | Type        | Description                                           |
|---|----------------------|-------------|-------------------------------------------------------|
| 1 | Error Rate Over Time | line_chart  | Errors per minute with severity >= 5, grouped by time |
| 2 | Errors by Service    | pie_chart   | Error distribution across services                    |
| 3 | Latency Distribution | line_chart  | avg, p95, p99 latency per minute (requires `$d.duration_ms`) |
| 4 | Top Error Messages   | data_table  | Top 10 error messages by occurrence count             |
| 5 | Errors by Subsystem  | bar_chart   | Top 10 subsystems by error count                      |

All widgets include a service filter when affected services are known:
`&& ($l.applicationname == '<svc1>' || $l.applicationname == '<svc2>')`

### Widget Queries

**Error Rate Over Time:**
```dataprime
source logs
| filter $m.severity >= 5 && (<service_filter>)
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate count() as errors
```

**Errors by Service:**
```dataprime
source logs
| filter $m.severity >= 5 && (<service_filter>)
| groupby $l.applicationname
| calculate count() as errors
```

**Latency Distribution:**
```dataprime
source logs
| filter $d.duration_ms.exists() && (<service_filter>)
| groupby roundTime($m.timestamp, 1m) as time_bucket
| aggregate
    avg($d.duration_ms) as avg_latency,
    percentile($d.duration_ms, 95) as p95,
    percentile($d.duration_ms, 99) as p99
```

**Top Error Messages:**
```dataprime
source logs
| filter $m.severity >= 5 && (<service_filter>)
| groupby $d.message
| calculate count() as occurrences
| sortby -occurrences
| limit 10
```

**Errors by Subsystem:**
```dataprime
source logs
| filter $m.severity >= 5 && (<service_filter>)
| groupby $l.subsystemname
| calculate count() as errors
| sortby -errors
| limit 10
```

### IBM Cloud Logs Dashboard JSON

The dashboard JSON includes widget positioning on a 12-column grid:

- Widgets are arranged in a 2-column layout (width: 6 each)
- Height: 4 units per widget
- Position calculated as: `x = (index % 2) * 6`, `y = (index / 2) * 4`
- Each widget includes query syntax set to `dataprime`
- Time frame defaults to `last_1_hour`

## SOP Recommendations

The remediation generator also matches the root cause text against
keyword patterns to select relevant SOPs:

| Root cause contains | SOP trigger                      |
|---------------------|----------------------------------|
| `timeout`           | Timeout errors detected          |
| `memory`            | Memory pressure detected         |
| `connection`        | Connection errors detected       |
| `auth`              | Authentication failures detected |
| `rate`              | Rate limiting detected           |

If no pattern matches, a generic SOP is provided covering: review error
logs, check recent deployments, verify infrastructure health, and review
dependent service status.
