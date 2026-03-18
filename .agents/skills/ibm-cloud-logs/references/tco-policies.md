# TCO Policies Reference

## Overview

TCO (Total Cost of Ownership) policies control how IBM Cloud Logs routes incoming log data to storage tiers. By assigning the correct priority to each policy, you determine the cost and query performance characteristics for each log stream.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/policies` | List all TCO policies |
| `POST` | `/v1/policies` | Create a new policy |
| `GET` | `/v1/policies/{id}` | Get a specific policy |
| `PUT` | `/v1/policies/{id}` | Update an existing policy |
| `DELETE` | `/v1/policies/{id}` | Delete a policy |

## Policy Structure

A TCO policy consists of:

```json
{
  "name": "Production API Errors",
  "description": "Route production API errors to Priority Insights",
  "enabled": true,
  "priority": "type_high",
  "application_rule": {
    "name": "production-api",
    "rule_type_id": "is"
  },
  "subsystem_rule": {
    "name": "auth",
    "rule_type_id": "is"
  },
  "archive_retention": {
    "id": "retention-policy-id"
  }
}
```

## Priority Levels

| Priority | Constant | Tier Routing | Description |
|----------|----------|-------------|-------------|
| High | `type_high` | frequent_search + archive | Logs go to Priority Insights for fast queries AND to archive for long-term storage |
| Medium | `type_medium` | archive only | Logs go to archive (COS) only -- not available for fast queries |
| Low | `type_low` | dropped | Logs are blocked and not stored anywhere |
| Unspecified | `type_unspecified` | archive | Treated as archive-only routing |

## Matching Rules

### Rule Types

| Rule Type ID | Behavior | Example |
|-------------|----------|---------|
| `is` | Exact string match | `"name": "api-gateway"` matches only `api-gateway` |
| `is_not` | Negation match | `"name": "debug-service"` matches everything except `debug-service` |
| `starts_with` | Prefix match | `"name": "prod"` matches `production-api`, `prod-worker`, etc. |
| `includes` | Substring match | `"name": "api"` matches `api-gateway`, `my-api-service`, etc. |

### Combined Rules

When both `application_rule` and `subsystem_rule` are specified, both must match (AND logic). If only one rule is specified, only that dimension is checked.

## Policy Evaluation Order

Policies are evaluated in the order they are returned by the API. **First match wins**. Design policy order carefully:

1. Place most specific rules first (exact app + subsystem match)
2. Place broader rules after (prefix/substring matches)
3. Use a catch-all policy last if needed

## Example Policies

### Route Production Errors to Priority Insights

```json
{
  "name": "Production Errors",
  "priority": "type_high",
  "application_rule": {
    "name": "production",
    "rule_type_id": "starts_with"
  }
}
```

### Archive Standard Operational Logs

```json
{
  "name": "Standard Operations",
  "priority": "type_medium",
  "application_rule": {
    "name": "operations",
    "rule_type_id": "includes"
  }
}
```

### Drop Health Check Noise

```json
{
  "name": "Drop Health Checks",
  "priority": "type_low",
  "subsystem_rule": {
    "name": "health-check",
    "rule_type_id": "is"
  }
}
```

### Selective Subsystem Routing

```json
{
  "name": "API Gateway Auth to Priority Insights",
  "priority": "type_high",
  "application_rule": {
    "name": "api-gateway",
    "rule_type_id": "is"
  },
  "subsystem_rule": {
    "name": "auth",
    "rule_type_id": "is"
  }
}
```

### Drop Debug Logs from Non-Production

```json
{
  "name": "Drop Dev Debug",
  "priority": "type_low",
  "application_rule": {
    "name": "dev-",
    "rule_type_id": "starts_with"
  },
  "subsystem_rule": {
    "name": "debug",
    "rule_type_id": "includes"
  }
}
```

## MCP Tool Cost Hints

| Tool | APICost | ExecutionSpeed | Impact | RateLimitImpact | RequiresConfirm |
|------|---------|---------------|--------|-----------------|-----------------|
| `list_policies` | low | fast | none | minimal | no |
| `get_policy` | low | fast | none | minimal | no |
| `create_policy` | low | fast | high | minimal | no |
| `update_policy` | low | fast | high | minimal | no |
| `delete_policy` | low | fast | critical | minimal | yes |

## Important Notes

- Deleting a policy can cause logs to fall through to the default tier, potentially increasing costs or losing data. Always review the policy list before deleting.
- Disabled policies (`"enabled": false`) are skipped during evaluation but still count toward the policy count.
- When no policies exist, all logs go to both tiers (frequent_search and archive). This is the most expensive configuration but provides the fastest queries.
- TCO configuration is cached in the MCP session for up to 1 hour. Changes may not be reflected immediately in tier-aware query routing.
- The session method `IsTCOConfigStale()` returns true if the cached config is older than 1 hour, triggering a refresh on the next operation.
