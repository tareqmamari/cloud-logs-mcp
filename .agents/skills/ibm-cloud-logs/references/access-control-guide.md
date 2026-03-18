# Access Control & Security Guide

> Domain guide for IBM Cloud Logs access control. For inline essentials
> (DataPrime syntax, common mistakes), see [SKILL.md](../SKILL.md).

## Data Access Rules

Data access rules control which logs users can view based on filter expressions.

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/data_access_rules` | List all rules |
| `GET` | `/v1/data_access_rules/{id}` | Get a specific rule |
| `POST` | `/v1/data_access_rules` | Create a rule |
| `PUT` | `/v1/data_access_rules/{id}` | Update a rule |
| `DELETE` | `/v1/data_access_rules/{id}` | Delete a rule |

### Rule Structure

- **display_name** (required): Human-readable name
- **description**: Purpose of the restriction
- **default_expression**: Default filter applied to all users (e.g., `NOT subsystemName:'pii-service'`)
- **filters**: Array of filter configurations

### Filter Expressions

- `applicationName.startsWith('production')` -- restrict to production
- `NOT subsystemName:'pii-service'` -- exclude PII
- `applicationName == 'payment-service'` -- single service

### Dry-Run Validation

Use `dry_run: true` to validate without applying changes.

## Data Access Policies

Higher-level abstraction for managing access control, working alongside rules.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/data_access_policies` | List all policies |
| `POST` | `/v1/data_access_policies` | Create a policy |
| `PUT` | `/v1/data_access_policies/{id}` | Update a policy |
| `DELETE` | `/v1/data_access_policies/{id}` | Delete a policy |

## Views for Scoped Access

Saved log queries with predefined filters for team-specific access.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/views` | List all views |
| `POST` | `/v1/views` | Create a view |
| `PUT` | `/v1/views/{id}` | Replace a view |
| `DELETE` | `/v1/views/{id}` | Delete a view |

View folders: `GET/POST/PUT/DELETE /v1/view_folders[/{id}]`

### Access Control Patterns with Views

- Create team-specific view folders
- Save commonly used log queries for consistent access
- Set up debugging views with severity and application filters

## Audit Logging

Each audit log entry includes: timestamp, trace_id, tool, operation, success, duration.

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Max entries (default 50, max 1000) |
| `tool` | string | Filter by tool name |
| `trace_id` | string | Filter by trace ID |

Set `LOG_LEVEL=debug` for verbose audit entries.

## Security Query Templates

### Authentication Failures
```
source logs
  | filter $d.event_type == 'auth_failure'
       || $d.message.contains('authentication failed')
       || $d.message.contains('invalid credentials')
  | groupby $d.source_ip, $d.username
  | aggregate count() as failures
  | filter failures > 3
  | sortby -failures
```

### Privilege Escalation
```
source logs
  | filter $d.event_type.contains('privilege')
       || $d.message.contains('sudo')
       || $d.message.contains('root access')
  | select $m.timestamp, $l.applicationname, $d.username, $d.action, $d.message
  | sortby -$m.timestamp | limit 100
```

### Sensitive Data Access
```
source logs
  | filter $d.resource.contains('pii')
       || $d.resource.contains('secrets')
       || $d.endpoint.contains('/admin')
  | select $m.timestamp, $d.username, $d.resource, $d.action, $d.source_ip
  | sortby -$m.timestamp | limit 100
```

### Configuration Changes
```
source logs
  | filter $d.event_type.contains('config')
       || $d.message.contains('configuration changed')
  | select $m.timestamp, $d.username, $d.resource, $d.old_value, $d.new_value
  | sortby -$m.timestamp | limit 100
```

### Data Exports
```
source logs
  | filter $d.action.contains('export') || $d.action.contains('download')
  | select $m.timestamp, $d.username, $d.resource, $d.record_count
  | sortby -$m.timestamp | limit 100
```

### API Key Usage
```
source logs
  | filter $d.api_key_id != '' || $d.auth_type == 'api_key'
  | groupby $d.api_key_id, $l.applicationname
  | aggregate count() as calls, approx_count_distinct($d.source_ip) as unique_ips
  | sortby -calls | limit 50
```

## Compliance Patterns

### GDPR / PII Protection
1. Create rule with `default_expression: "NOT subsystemName:'pii-service'"`
2. Create dedicated view for privacy team
3. Use `sensitive_data_access` query for auditing
4. Review audit logs periodically

### Multi-Tenant Isolation
1. Per-tenant data access rules by `applicationName` prefix
2. Organize views into tenant-specific folders
3. Monitor `auth_failures` and `privilege_escalation` templates

### SOC 2 / Audit Readiness
1. Enable verbose audit logging (`LOG_LEVEL=debug`)
2. Run `config_changes` and `data_exports` templates regularly
3. Export query results for audit reports

## CLI Commands

```bash
ibmcloud logs data-access-rules --output json
ibmcloud logs data-access-rule-create --prototype @access-rule.json
ibmcloud logs views --output json
ibmcloud logs view-create --prototype @view.json
ibmcloud logs view-folders --output json
```

## Deep References

- [Access Rules API Reference](access-rules.md) -- Full API details, filter syntax, examples
- [Access Rule Template](../assets/access-rule-template.json) -- JSON template
