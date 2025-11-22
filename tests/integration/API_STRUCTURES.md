# IBM Cloud Logs API Structures Reference

This document provides the correct API structures based on the IBM Terraform provider examples and actual API responses.

## Alerts (`/v1/alerts`)

### Create Alert Structure
```json
{
  "name": "alert-name",
  "description": "Alert description",
  "is_active": true,
  "severity": "info_or_unspecified",
  "condition": {
    "immediate": {}
  },
  "notification_groups": [
    {
      "group_by_fields": ["coralogix.metadata.applicationName"],
      "notifications": [
        {
          "retriggering_period_seconds": 60,
          "notify_on": "triggered_and_resolved",
          "integration_id": 123
        }
      ]
    }
  ],
  "filters": {
    "filter_type": "text_or_unspecified",
    "severities": ["info"],
    "text": "_exists_:\"field_name\"",
    "metadata": {
      "applications": ["app1", "app2"],
      "subsystems": ["subsystem1"]
    }
  },
  "expiration": {
    "year": 2025,
    "month": 12,
    "day": 31
  },
  "incident_settings": {
    "retriggering_period_seconds": 300,
    "notify_on": "triggered_only",
    "use_as_notification_settings": true
  },
  "meta_labels": [
    {
      "key": "env",
      "value": "production"
    }
  ],
  "active_when": {
    "timeframes": [
      {
        "days_of_week": ["monday_or_unspecified", "tuesday"],
        "range": {
          "start": {"hours": 9, "minutes": 0, "seconds": 0},
          "end": {"hours": 17, "minutes": 0, "seconds": 0}
        }
      }
    ]
  }
}
```

### Severity Values
- `debug_or_unspecified`
- `verbose`
- `info_or_unspecified`
- `warning_or_unspecified`
- `error_or_unspecified`
- `critical`

## Policies (`/v1/policies`)

### Create Policy Structure
```json
{
  "name": "policy-name",
  "description": "Policy description",
  "priority": "type_high",
  "enabled": true,
  "application_rule": {
    "rule_type_id": "is",
    "name": "application-name"
  },
  "subsystem_rule": {
    "rule_type_id": "is",
    "name": "subsystem-name"
  },
  "log_rules": {
    "severities": ["info", "warning", "error"]
  },
  "archive_retention": {
    "id": "3d9a5b88-f344-47f2-893a-580e50d4f7b8"
  }
}
```

### Rule Type IDs
- `is` - Exact match
- `is_not` - Not equal
- `starts_with` - Prefix match
- `ends_with` - Suffix match
- `includes` - Contains
- `includes_regex` - Regex match

### Priority Values
- `type_low`
- `type_medium`
- `type_high`

## E2M (Events to Metrics) (`/v1/events2metrics`)

### Create E2M Structure
```json
{
  "name": "e2m-name",
  "description": "E2M description",
  "type": "logs2metrics",
  "logs_query": {
    "lucene": "severity:error",
    "applicationname_filters": ["app1"],
    "subsystemname_filters": ["subsystem1"],
    "severity_filters": ["error", "critical"]
  },
  "metric_labels": [
    {
      "target_label": "service",
      "source_field": "log_obj.service_name"
    }
  ],
  "metric_fields": [
    {
      "target_base_metric_name": "error_count",
      "source_field": "log_obj.count",
      "aggregations": [
        {
          "enabled": true,
          "agg_type": "count",
          "target_metric_name": "total_errors"
        }
      ]
    }
  ]
}
```

### Aggregation Types
- `count` - Count aggregation
- `samples` - Sample aggregation (requires `samples` object with `sample_type`)
- `histogram` - Histogram aggregation (requires `histogram` object with `buckets`)
- `min` - Minimum value
- `max` - Maximum value
- `avg` - Average value
- `sum` - Sum aggregation

### Sample Types (for `samples` aggregation)
- `min`
- `max`
- `avg`
- `sum`
- `unspecified`

## Views (`/v1/views`)

### Create View Structure
```json
{
  "name": "view-name",
  "folder_id": "folder-uuid-optional",
  "search_query": {
    "query": "error"
  },
  "time_selection": {
    "quick_selection": {
      "caption": "Last hour",
      "seconds": 3600
    }
  },
  "filters": {
    "filters": [
      {
        "name": "applicationName",
        "selected_values": {
          "app1": true,
          "app2": true
        }
      }
    ]
  }
}
```

### Quick Selection Presets
- 15 minutes: 900 seconds
- 1 hour: 3600 seconds
- 6 hours: 21600 seconds
- 1 day: 86400 seconds

### Custom Time Selection
```json
{
  "time_selection": {
    "custom_selection": {
      "from_time": "2024-01-01T00:00:00Z",
      "to_time": "2024-01-01T23:59:59Z"
    }
  }
}
```

## View Folders (`/v1/view_folders`)

### Create View Folder Structure
```json
{
  "name": "folder-name"
}
```

## Outgoing Webhooks (`/v1/outgoing_webhooks`)

### Generic Webhook
```json
{
  "type": "generic",
  "name": "webhook-name",
  "url": "https://example.com/webhook",
  "generic_webhook": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "method": "post",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token"
    },
    "payload": "{\"message\": \"{{message}}\"}"
  }
}
```

### IBM Event Notifications Webhook
```json
{
  "type": "ibm_event_notifications",
  "name": "ibm-en-webhook",
  "url": "https://event-notifications-endpoint",
  "ibm_event_notifications": {
    "event_notifications_instance_id": "9fab83da-98cb-4f18-a7ba-b6f0435c9673",
    "region_id": "eu-es",
    "source_id": "crn:v1:staging:public:logs:eu-gb:a/account:instance-id::",
    "source_name": "IBM Cloud Event Notifications",
    "endpoint_type": "private"
  }
}
```

### Slack Webhook
```json
{
  "type": "slack",
  "name": "slack-webhook",
  "url": "https://hooks.slack.com/services/T00000000/B00000000/XXXX",
  "slack": {
    "digests": [
      {
        "type": "error_and_critical_logs",
        "is_active": true
      }
    ],
    "attachments": [
      {
        "type": "metric_snapshot",
        "is_active": true
      }
    ]
  }
}
```

### PagerDuty Webhook
```json
{
  "type": "pager_duty",
  "name": "pagerduty-webhook",
  "url": "https://events.pagerduty.com/integration/key/enqueue",
  "pager_duty": {
    "service_key": "your-pagerduty-service-key"
  }
}
```

## Dashboards (`/v1/dashboards`)

### Create Dashboard Structure
```json
{
  "name": "dashboard-name",
  "description": "Dashboard description",
  "layout": {
    "sections": [
      {
        "id": "section-id",
        "rows": [
          {
            "id": "row-id",
            "appearance": {
              "height": 400
            },
            "widgets": [
              {
                "id": "widget-id",
                "title": "Widget Title",
                "definition": {
                  "line_chart": {
                    "legend": {
                      "is_visible": true,
                      "columns": ["avg", "sum", "last"]
                    },
                    "tooltip": {
                      "show_labels": true,
                      "type": "all"
                    },
                    "query_definitions": [
                      {
                        "id": "query-id",
                        "query": {
                          "logs": {
                            "lucene_query": {
                              "value": "*"
                            },
                            "group_by": ["applicationName"],
                            "aggregations": [
                              {
                                "count": {}
                              }
                            ],
                            "filters": []
                          }
                        }
                      }
                    ]
                  }
                }
              }
            ]
          }
        ]
      }
    ]
  },
  "filters": [
    {
      "source": {
        "logs": {
          "operator": {
            "equals": {
              "selection": {
                "all": {}
              }
            }
          },
          "observation_field": {
            "keypath": ["applicationName"],
            "scope": "metadata"
          }
        }
      },
      "enabled": true,
      "collapsed": false
    }
  ],
  "relative_time_frame": "900s"
}
```

## Common Patterns

### Field Naming Conventions
- Use `snake_case` for all field names
- Boolean fields: `is_active`, `is_visible`, `enabled`
- ID fields: `id`, `integration_id`, `instance_id`
- Enum fields often have `_or_unspecified` suffix

### Metadata Fields
- Applications: `coralogix.metadata.applicationName`
- Subsystems: `coralogix.metadata.subsystemName`
- Severity: `severity`

### Observation Fields
```json
{
  "keypath": ["fieldName"],
  "scope": "metadata"
}
```

Scopes: `metadata`, `labels`, `user_data`

### Date/Time Formats
- ISO 8601: `"2024-01-01T00:00:00Z"`
- Time ranges: `"900s"`, `"1h"`, `"24h"`
- Expiration: `{"year": 2025, "month": 12, "day": 31}`

### Error Response Format
```json
{
  "errors": [
    {
      "code": "bad_request_or_unspecified",
      "message": "Error message"
    }
  ],
  "status_code": 400,
  "trace": "trace-id"
}
```

## Testing Tips

1. **Use GET before PUT**: Always fetch the current resource before updating to ensure all required fields are present
2. **Field names are case-sensitive**: Use exact snake_case as shown
3. **Enum values**: Use the full enum name including `_or_unspecified` suffix where applicable
4. **Nested objects**: Some fields require empty objects `{}` even when not configured
5. **Arrays vs single values**: Some fields that appear singular actually expect arrays
6. **UUIDs**: Must be valid UUID format for ID references
7. **Timestamps**: Use RFC3339 format with timezone
