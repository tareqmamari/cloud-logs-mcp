---
name: ibm-cloud-logs-access-control
description: >
  Configure data access rules and audit logging for IBM Cloud Logs. Activate
  when restricting log access by team/role, setting up compliance controls,
  or reviewing audit trails.
license: Apache-2.0
compatibility: Works with any agent that can read markdown. No runtime dependencies.
metadata:
  category: observability
  platform: ibm-cloud
  domain: security
  version: "0.10.0" # x-release-please-version
---

# IBM Cloud Logs Access Control Skill

## When to Activate

Use this skill when the user:
- Needs to restrict which logs specific teams or roles can see
- Wants to create, update, or manage data access rules
- Is setting up multi-tenant log isolation
- Asks about compliance controls (GDPR, PII filtering, audit trails)
- Wants to organize saved views for scoped access by team
- Needs to review audit logs for tool executions and operations
- Is investigating who accessed sensitive data or changed configurations
- Wants to set up security monitoring queries (auth failures, privilege escalation)

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

## Data Access Rules

Data access rules control which logs users can view based on filter expressions. They are the primary mechanism for enforcing data isolation and access control in IBM Cloud Logs.

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/data_access_rules` | List all data access rules |
| `GET` | `/v1/data_access_rules/{id}` | Get a specific rule |
| `POST` | `/v1/data_access_rules` | Create a new rule |
| `PUT` | `/v1/data_access_rules/{id}` | Update a rule |
| `DELETE` | `/v1/data_access_rules/{id}` | Delete a rule |

### Rule Structure

A data access rule consists of:
- **display_name** (required): Human-readable name for the rule
- **description**: Purpose of the access restriction
- **default_expression**: Default filter expression applied to all users (e.g., `NOT subsystemName:'pii-service'`)
- **filters**: Array of filter configurations specifying entity type and filter expression

### Filter Expressions

Filters use log field expressions to scope visibility:
- `applicationName.startsWith('production')` -- restrict to production app logs
- `NOT subsystemName:'pii-service'` -- exclude PII service logs
- `applicationName == 'payment-service'` -- limit to a single service

### Common Use Cases

- **Team isolation**: Restrict each team to see only their own application logs
- **Multi-tenant separation**: Ensure tenant A cannot see tenant B's logs
- **PII exclusion**: Filter out logs from services that handle personal data
- **Environment scoping**: Limit access to production, staging, or dev logs

### Dry-Run Validation

Use `dry_run: true` when creating rules to validate configuration without applying changes. The validator checks for:
- Missing required fields (display_name)
- Rules with no filters or default_expression (which would not restrict any data)
- Estimated impact assessment (medium risk level for access rule changes)

## Data Access Policies

Data access policies provide a higher-level abstraction for managing access control. They work alongside data access rules to define who can see what data.

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/data_access_policies` | List all data access policies |
| `GET` | `/v1/data_access_policies/{id}` | Get a specific policy |
| `POST` | `/v1/data_access_policies` | Create a new policy |
| `PUT` | `/v1/data_access_policies/{id}` | Update a policy |
| `DELETE` | `/v1/data_access_policies/{id}` | Delete a policy |

### Workflow

1. List current policies to review existing access boundaries
2. Inspect a specific policy to understand its scope
3. Create new policies to define access boundaries for teams or roles
4. Update policies as requirements change

## Views for Scoped Access

Views are saved log queries with predefined filters that provide scoped access to specific subsets of logs. Combined with view folders, they enable organized, team-specific log access.

### View API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/views` | List all saved views |
| `GET` | `/v1/views/{id}` | Get a specific view |
| `POST` | `/v1/views` | Create a saved view |
| `PUT` | `/v1/views/{id}` | Replace a view |
| `DELETE` | `/v1/views/{id}` | Delete a view |

### View Folder API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/view_folders` | List all view folders |
| `GET` | `/v1/view_folders/{id}` | Get a view folder |
| `POST` | `/v1/view_folders` | Create a folder |
| `PUT` | `/v1/view_folders/{id}` | Replace a folder |
| `DELETE` | `/v1/view_folders/{id}` | Delete a folder |

### View Configuration

A view includes:
- **name**: Display name for the view (e.g., "Production Errors")
- **search_query**: Query configuration with a `query` string (e.g., `application:production AND level:error`)
- **time_selection**: Time range selection for the view
- **filters**: Additional filter configuration
- **folder_id**: Optional folder ID to organize the view

### Access Control Patterns with Views

- Create team-specific view folders (e.g., "Platform Team", "Security Team")
- Save commonly used log queries for quick, consistent access
- Set up debugging views with specific severity and application filters
- Share standardized views across the organization for consistent log access

## Audit Logging

IBM Cloud Logs captures audit log entries showing recent operations and API calls.

### What is Captured

Each audit log entry includes:
- **timestamp**: When the operation occurred
- **trace_id**: Correlation ID for tracing related operations
- **tool**: Which operation was executed (e.g., `query`, `data-access-rule-create`)
- **operation**: Type of operation performed
- **success**: Whether the operation completed successfully
- **duration**: How long the operation took

### Tool Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Maximum entries to return (default: 50, max: 1000) |
| `tool` | string | Filter by specific tool name |
| `trace_id` | string | Filter by trace ID to see all operations in a trace |

### Enabling Verbose Audit Logging

Set `LOG_LEVEL=debug` in the environment to enable detailed audit entries. Look for log entries with logger name `"audit"` in server logs.

### Compliance Use Cases

- **Change tracking**: Review all create, update, and delete operations on access rules
- **Operation verification**: Confirm that security policy changes were applied successfully
- **Incident timeline**: Use trace IDs to reconstruct the sequence of operations during an incident
- **Access review**: Audit which tools were used and when, for periodic security reviews

## Security Query Templates

The following pre-built DataPrime query templates can be used for security monitoring. Run them with `ibmcloud logs query --query '<query>'`. For DataPrime syntax, commands, and functions, see [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md).

### Authentication Failures (`auth_failures`)

Detect brute force attempts and compromised accounts by finding authentication failures grouped by source IP and username.

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

**Tips**: Threshold of 3 filters noise. Group by IP to detect attacks. Follow up with IP geolocation.

### Privilege Escalation (`privilege_escalation`)

Detect unauthorized privilege changes and audit admin actions.

```
source logs
  | filter $d.event_type.contains('privilege')
       || $d.message.contains('sudo')
       || $d.message.contains('root access')
       || $d.message.contains('admin')
  | select $m.timestamp, $l.applicationname, $d.username, $d.action, $d.message
  | sortby -$m.timestamp
  | limit 100
```

**Tips**: Customize patterns for your environment. Correlate with known admin activities. Set up alerts for critical matches.

### Sensitive Data Access (`sensitive_data_access`)

Track access to PII, secrets, credentials, and admin endpoints for compliance evidence gathering.

```
source logs
  | filter $d.resource.contains('pii')
       || $d.resource.contains('secrets')
       || $d.resource.contains('credentials')
       || $d.endpoint.contains('/admin')
  | select $m.timestamp, $d.username, $d.resource, $d.action, $d.source_ip
  | sortby -$m.timestamp
  | limit 100
```

**Tips**: Adjust resource patterns for your data classification. Export results for compliance reports. Set up alerts for unusual access patterns.

### Configuration Changes (`config_changes`)

Track configuration changes for change management audit and troubleshooting.

```
source logs
  | filter $d.event_type.contains('config')
       || $d.message.contains('configuration changed')
       || $d.message.contains('settings updated')
  | select $m.timestamp, $d.username, $d.resource, $d.old_value, $d.new_value, $d.message
  | sortby -$m.timestamp
  | limit 100
```

### Data Exports (`data_exports`)

Track data export activities for data loss prevention and compliance auditing.

```
source logs
  | filter $d.action.contains('export')
       || $d.action.contains('download')
       || $d.message.contains('exported')
  | select $m.timestamp, $d.username, $d.resource, $d.record_count, $d.destination
  | sortby -$m.timestamp
  | limit 100
```

### API Key Usage (`api_key_usage`)

Track API key usage patterns to detect sharing, abuse, and plan key rotation.

```
source logs
  | filter $d.api_key_id != '' || $d.auth_type == 'api_key'
  | groupby $d.api_key_id, $l.applicationname
  | aggregate count() as calls, approx_count_distinct($d.source_ip) as unique_ips
  | sortby -calls
  | limit 50
```

**Tips**: Multiple IPs per key may indicate sharing. Inactive keys should be rotated.

## Compliance Patterns

### GDPR / PII Protection

1. Create a data access rule with `default_expression: "NOT subsystemName:'pii-service'"` to exclude PII logs from general access
2. Create a dedicated view for the privacy team with PII service logs only
3. Use the `sensitive_data_access` query template to audit who accesses PII-related endpoints
4. Review audit logs periodically to verify access rule enforcement

### Multi-Tenant Isolation

1. Create per-tenant data access rules scoping visibility by `applicationName` prefix
2. Organize views into tenant-specific folders
3. Review data access rules regularly (`ibmcloud logs data-access-rules --output json`) to verify no gaps in tenant isolation
4. Monitor `auth_failures` and `privilege_escalation` templates for cross-tenant access attempts

### SOC 2 / Audit Readiness

1. Enable verbose audit logging (`LOG_LEVEL=debug`) for comprehensive operation tracking
2. Review audit logs for security-sensitive operations
3. Run `config_changes` and `data_exports` templates on a schedule for compliance evidence
4. Export query results for inclusion in audit reports

## Using the IBM Cloud CLI

Data access rules and views can be managed via the [IBM Cloud Logs CLI plugin](https://cloud.ibm.com/docs/cloud-logs-cli-plugin):

```bash
# List data access rules
ibmcloud logs data-access-rules --output json

# Create a data access rule from a JSON file
ibmcloud logs data-access-rule-create --prototype @access-rule.json

# List views
ibmcloud logs views --output json

# Create a scoped view
ibmcloud logs view-create --prototype @view.json

# Manage view folders
ibmcloud logs view-folders --output json
ibmcloud logs view-folder-create --prototype @folder.json

# Run security audit queries
ibmcloud logs query \
  --query 'source logs | filter $d.event_type == '\''auth_failure'\'' | groupby $d.source_ip, $d.username aggregate count() as failures | filter failures > 3 | orderby -failures'
```

## Context Management

To minimize context window usage, follow these practices:

- **Do not load references eagerly.** Only read files from `references/` when the user's question requires deeper detail than what this SKILL.md provides.
- **Write access rule configs to files.** When generating data access rule JSON configurations, write them to a file (e.g., `access-rule.json`) instead of pasting inline.
- **Write audit query results to files.** Audit query output can be lengthy. Write results to a file (e.g., `audit-results.json`) rather than including them in the response.
- **Load access-rules.md on demand.** Only read `references/access-rules.md` when the user needs full API details, filter syntax, or advanced examples -- do not load it preemptively.
- **Do not paste full reference files** into responses. Summarize and link instead.

## Additional Resources

- [Access Rules API Reference](references/access-rules.md) -- full API details, filter syntax, and examples
- [Access Rule Template](assets/access-rule-template.json) -- JSON template for data access rule creation
- [IBM Cloud Logs Query Skill](../ibm-cloud-logs-query/SKILL.md) -- DataPrime syntax, commands, functions, and query templates
