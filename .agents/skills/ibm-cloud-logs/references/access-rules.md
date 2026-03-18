# Data Access Rules API Reference

## Overview

Data access rules control log visibility in IBM Cloud Logs by applying filter expressions that restrict which logs specific users or groups can see. Rules are managed through the `/v1/data_access_rules` API endpoint.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/data_access_rules` | List all data access rules |
| `GET` | `/v1/data_access_rules/{id}` | Get a specific rule by ID |
| `POST` | `/v1/data_access_rules` | Create a new data access rule |
| `PUT` | `/v1/data_access_rules/{id}` | Update an existing rule |
| `DELETE` | `/v1/data_access_rules/{id}` | Delete a rule |

## Rule Schema

```json
{
  "display_name": "string (required)",
  "description": "string (optional)",
  "default_expression": "string (optional)",
  "filters": [
    {
      "entity_type": "logs",
      "expression": "string"
    }
  ]
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `display_name` | string | Yes | Human-readable name for the rule |
| `description` | string | No | Explanation of the rule's purpose |
| `default_expression` | string | No | Default filter expression applied to all users |
| `filters` | array | No | Array of filter configurations |
| `filters[].entity_type` | string | Yes (if filters used) | Entity type to filter, typically `"logs"` |
| `filters[].expression` | string | Yes (if filters used) | Filter expression using log field syntax |

## Filter Expression Syntax

Filter expressions use log field names and operators to define visibility boundaries.

### Field References

| Field | Description | Example |
|-------|-------------|---------|
| `applicationName` | Application name label | `applicationName == 'payment-service'` |
| `subsystemName` | Subsystem name label | `subsystemName:'api-gateway'` |
| `computername` | Compute host name | `computername.startsWith('prod-')` |

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Exact match | `applicationName == 'my-app'` |
| `!=` | Not equal | `applicationName != 'test-app'` |
| `:` | Contains / matches | `subsystemName:'api'` |
| `.startsWith()` | Prefix match | `applicationName.startsWith('production')` |
| `.contains()` | Substring match | `applicationName.contains('service')` |
| `NOT` | Negation | `NOT subsystemName:'pii-service'` |
| `AND` | Conjunction | `applicationName:'prod' AND subsystemName:'api'` |
| `OR` | Disjunction | `applicationName:'app-a' OR applicationName:'app-b'` |

### Expression Examples

**Restrict to a single application:**
```
applicationName == 'payment-service'
```

**Restrict to production environment:**
```
applicationName.startsWith('production')
```

**Exclude sensitive subsystems:**
```
NOT subsystemName:'pii-service'
```

**Multi-condition filter:**
```
applicationName.startsWith('prod') AND NOT subsystemName:'internal-audit'
```

**Multiple application access:**
```
applicationName:'frontend' OR applicationName:'api-gateway' OR applicationName:'cdn'
```

## Dry-Run Validation

Pass `dry_run: true` alongside the `rule` parameter when calling `create_data_access_rule` to validate the configuration without creating it.

### Validation Checks

1. **Required fields**: Verifies `display_name` is present
2. **Filter coverage**: Warns if no `filters` or `default_expression` is specified (rule would not restrict any data)
3. **Impact assessment**: Returns an estimated risk level (`medium` for access rule changes)

### Dry-Run Response

```json
{
  "valid": true,
  "summary": {
    "display_name": "Production Team Access"
  },
  "warnings": [],
  "errors": [],
  "suggestions": [
    "Data access rule configuration is valid",
    "Remove dry_run parameter to create the rule"
  ],
  "estimated_impact": {
    "risk_level": "medium"
  }
}
```

## Tool Workflows

### Creating a New Rule

1. Call `list_data_access_rules` to see existing rules and avoid conflicts
2. Call `create_data_access_rule` with `dry_run: true` to validate
3. Call `create_data_access_rule` without `dry_run` to create the rule
4. Call `list_data_access_rules` to verify the new rule appears

### Updating a Rule

1. Call `get_data_access_rule` with the rule ID to get the current configuration
2. Modify the desired fields in the rule object
3. Call `update_data_access_rule` with the ID and updated rule object
4. Call `get_data_access_rule` to verify the changes

### Deleting a Rule

1. Call `get_data_access_rule` to confirm the rule to delete
2. Call `delete_data_access_rule` with the rule ID
3. Call `list_data_access_rules` to confirm removal

## Related Tools

- **Views**: Use `create_view` with scoped queries to complement data access rules with pre-built filtered views
- **View Folders**: Use `create_view_folder` to organize views by team or access level
- **Security Templates**: Use `get_query_templates` with `category: "security"` for monitoring queries
- **Audit Log**: Use `get_audit_log` to review access rule changes and operations

## Security Intents

The discovery system maps the following intents to data access rule tools:

| Intent | Tools Suggested |
|--------|----------------|
| "security audit" | `list_data_access_rules`, `query_logs` |
| "access control" | `list_data_access_rules` |
| "permissions" | `list_data_access_rules` |
| "who can access" | `list_data_access_rules` |
| "sensitive data" | `list_data_access_rules`, `create_policy` |
| "pii" | `list_data_access_rules`, `create_policy` |
| "gdpr" | `list_data_access_rules`, `list_policies` |
| "audit logs" | `query_logs`, `list_data_access_rules` |
| "authorization" | `query_logs`, `list_data_access_rules` |
