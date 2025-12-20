# AGENTS.md - IBM Cloud Logs MCP Server Agent Card

<!-- x-release-please-start-version -->
> **Version**: 0.8.0
<!-- x-release-please-end -->
> **Last Updated**: 2025-12-18
> **MCP Spec Compliance**: 2025-11-25
> **A2A Discovery**: Supported via `/.well-known/agent.json`

## Agent Identity

```yaml
name: ibm-cloud-logs-mcp
display_name: IBM Cloud Logs MCP Server
version: 0.8.0  # x-release-please-version
description: >
  Model Context Protocol server for IBM Cloud Logs observability platform.
  Provides intelligent log querying, alert management, dashboard creation,
  and incident investigation capabilities with budget-aware progressive disclosure.
vendor: IBM / Community
license: Apache-2.0
```

## Capabilities Matrix

### Core Capabilities

| Capability | Status | Description |
|------------|--------|-------------|
| `tools` | ✅ Full | 50+ tools across 12 namespaces |
| `resources` | ❌ None | Not implemented |
| `prompts` | ❌ None | Not implemented |
| `sampling` | ❌ None | Not implemented |
| `elicitation` | 🔶 Partial | Intent verification via tools |

### Transport Support

| Transport | Status | Notes |
|-----------|--------|-------|
| `stdio` | ✅ Primary | Default transport |
| `sse` | ✅ Supported | For streaming queries |
| `http` | 🔶 Planned | Streamable HTTP per MCP 2025-11-25 |

## Tool Namespaces & IBM Cloud IAM Mapping

### IAM Service Roles

IBM Cloud Logs uses four service roles with granular actions:

| Role | Description | Typical Use Case |
|------|-------------|------------------|
| **Sender** | Data ingestion only | Log shipping agents |
| **Reader** | Read-only access | Monitoring dashboards, read-only queries |
| **Writer** | Read + limited write | Alert management, view/dashboard creation |
| **Manager** | Full administrative access | Policy management, enrichments, full config |
| **Service Configuration Reader** | Configuration read-only | Compliance auditing |

### Namespace → IAM Action Mapping

#### 1. Queries Namespace (`queries.*`)

Tools: `query_logs`, `build_query`, `submit_background_query`, `get_background_query_status`, `get_background_query_data`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `query_logs` | `logs.logs-data-api-high.read`, `logs.logs-data-api-low.read` | Reader |
| `build_query` | None (local validation) | None |
| `submit_background_query` | `logs.logs-data-api-high.read`, `logs.legacy-archive-query.execute` | Reader |
| `get_background_query_*` | `logs.logs-data-api-high.read` | Reader |

#### 2. Alerts Namespace (`alerts.*`)

Tools: `list_alerts`, `get_alert`, `create_alert`, `update_alert`, `delete_alert`, `suggest_alert`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_alerts` | `logs.alert-config.read`, `logs.logs-alert.read` | Reader |
| `get_alert` | `logs.alert-config.read` | Reader |
| `create_alert` | `logs.alert-config.manage`, `logs.logs-alert.manage` | Writer |
| `update_alert` | `logs.alert-config.manage` | Writer |
| `delete_alert` | `logs.alert-config.manage` | Writer |
| `suggest_alert` | `logs.logs-data-api-high.read` | Reader |

#### 3. Dashboards Namespace (`dashboards.*`)

Tools: `list_dashboards`, `get_dashboard`, `create_dashboard`, `update_dashboard`, `delete_dashboard`, `list_dashboard_folders`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_dashboards` | `logs.shared-dashboard.read` | Reader |
| `get_dashboard` | `logs.shared-dashboard.read` | Reader |
| `create_dashboard` | `logs.shared-dashboard.manage` | Writer |
| `update_dashboard` | `logs.shared-dashboard.manage` | Writer |
| `delete_dashboard` | `logs.shared-dashboard.manage` | Writer |

#### 4. Policies Namespace (`policies.*`)

Tools: `list_policies`, `get_policy`, `create_policy`, `update_policy`, `delete_policy`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_policies` | `logs.logs-tco-policy.read` | Service Configuration Reader |
| `get_policy` | `logs.logs-tco-policy.read` | Service Configuration Reader |
| `create_policy` | `logs.logs-tco-policy.manage` | Manager |
| `update_policy` | `logs.logs-tco-policy.manage` | Manager |
| `delete_policy` | `logs.logs-tco-policy.manage` | Manager |

#### 5. Webhooks Namespace (`webhooks.*`)

Tools: `list_outgoing_webhooks`, `get_outgoing_webhook`, `create_outgoing_webhook`, `update_outgoing_webhook`, `delete_outgoing_webhook`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_outgoing_webhooks` | `logs.webhook.read` | Reader |
| `get_outgoing_webhook` | `logs.webhook.read` | Reader |
| `create_outgoing_webhook` | `logs.webhook.manage` | Writer |
| `update_outgoing_webhook` | `logs.webhook.manage` | Writer |
| `delete_outgoing_webhook` | `logs.webhook.manage` | Writer |

#### 6. Events2Metrics Namespace (`e2m.*`)

Tools: `list_e2m`, `get_e2m`, `create_e2m`, `update_e2m`, `delete_e2m`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_e2m` | `logs.events2metrics.read` | Service Configuration Reader |
| `get_e2m` | `logs.events2metrics.read` | Service Configuration Reader |
| `create_e2m` | `logs.events2metrics.manage` | Manager |
| `update_e2m` | `logs.events2metrics.manage` | Manager |
| `delete_e2m` | `logs.events2metrics.manage` | Manager |

#### 7. Enrichments Namespace (`enrichments.*`)

Tools: `list_enrichments`, `get_enrichment`, `create_enrichment`, `update_enrichment`, `delete_enrichment`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_enrichments` | `logs.custom-enrichment.read`, `logs.geo-enrichment.read`, `logs.security-enrichment.read` | Service Configuration Reader |
| `create_enrichment` | `logs.custom-enrichment.manage`, `logs.geo-enrichment.manage`, `logs.security-enrichment.manage` | Manager |
| `update_enrichment` | `logs.custom-enrichment.manage` | Manager |
| `delete_enrichment` | `logs.custom-enrichment.manage` | Manager |

#### 8. Views Namespace (`views.*`)

Tools: `list_views`, `get_view`, `create_view`, `update_view`, `delete_view`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_views` | `logs.shared-view.read`, `logs.private-view.read` | Reader |
| `get_view` | `logs.shared-view.read` | Reader |
| `create_view` | `logs.shared-view.manage`, `logs.private-view.manage` | Reader (private), Writer (shared) |
| `update_view` | `logs.shared-view.manage` | Writer |
| `delete_view` | `logs.shared-view.manage` | Writer |

#### 9. Data Access Namespace (`data-access.*`)

Tools: `list_data_access_rules`, `get_data_access_rule`, `create_data_access_rule`, `update_data_access_rule`, `delete_data_access_rule`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_data_access_rules` | `logs.data-access-rule.read` | Service Configuration Reader |
| `create_data_access_rule` | `logs.data-access-rule.manage` | Manager |
| `update_data_access_rule` | `logs.data-access-rule.manage` | Manager |
| `delete_data_access_rule` | `logs.data-access-rule.manage` | Manager |

#### 10. Streams Namespace (`streams.*`)

Tools: `list_streams`, `get_stream`, `create_stream`, `update_stream`, `delete_stream`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `list_streams` | `logs.logs-stream-setup.read` | Service Configuration Reader |
| `create_stream` | `logs.logs-stream-setup.manage` | Manager |

#### 11. Workflows Namespace (`workflows.*`)

Tools: `investigate_incident`, `health_check`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `investigate_incident` | `logs.logs-data-api-high.read`, `logs.incident.read` | Reader |
| `health_check` | `logs.logs-data-api-high.read`, `logs.alert-config.read` | Reader |

#### 12. Meta Namespace (`meta.*`)

Tools: `search_tools`, `describe_tools`, `list_tool_categories`, `discover_tools`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| All meta tools | None (local operations) | None |

#### 13. Ingestion Namespace (`ingestion.*`)

Tools: `ingest_logs`

| Tool | Required IAM Actions | Minimum Role |
|------|---------------------|--------------|
| `ingest_logs` | `logs.data-ingress.send` | Sender |

## Authentication Configuration

### Environment Variables

```bash
# Required
LOGS_API_KEY="<ibm-cloud-api-key>"           # IAM API key with appropriate roles
LOGS_SERVICE_URL="https://<instance>.api.<region>.logs.cloud.ibm.com"

# Optional
LOGS_REGION="us-south"                        # Default region
LOGS_TLS_VERIFY="true"                        # Always true in production
LOGS_RATE_LIMIT="100"                         # Requests per second
LOGS_RATE_LIMIT_BURST="20"                    # Burst allowance
```

### IAM Role CRN Format

When assigning roles via API, use these CRN formats:

```
# Service Roles
crn:v1:bluemix:public:logs::::serviceRole:Reader
crn:v1:bluemix:public:logs::::serviceRole:Writer
crn:v1:bluemix:public:logs::::serviceRole:Manager
crn:v1:bluemix:public:logs::::serviceRole:Sender
crn:v1:bluemix:public:logs::::serviceRole:DataAccessReader

# Platform Roles (for instance management)
crn:v1:bluemix:public:iam::::role:Viewer
crn:v1:bluemix:public:iam::::role:Operator
crn:v1:bluemix:public:iam::::role:Editor
crn:v1:bluemix:public:iam::::role:Administrator
```

### Recommended Role Assignments by Use Case

| Use Case | Recommended Role | Rationale |
|----------|------------------|-----------|
| Read-only monitoring | Reader | Query logs, view alerts/dashboards |
| Alert management | Writer | Create/modify alerts and webhooks |
| Full administration | Manager | Policy, enrichment, and stream management |
| Log shipping agent | Sender | Data ingestion only |
| Compliance auditing | Service Configuration Reader | View-only on all configurations |

## Behavioral Characteristics

### Budget-Aware Execution

The server implements progressive disclosure based on token budgets:

```json
{
  "compression_levels": {
    "none": "Full data, no compression",
    "summary": "Counts + top patterns only",
    "insights": "Summary + actionable insights",
    "samples": "Insights + representative samples",
    "full": "Complete dataset"
  },
  "default_budget": {
    "max_tokens": 4000,
    "compression": "insights"
  }
}
```

### Tool Annotations (MCP Hints)

| Annotation | Tools | Meaning |
|------------|-------|---------|
| `ReadOnlyHint: true` | All `list_*`, `get_*`, `query_*` | No side effects |
| `IdempotentHint: true` | `create_*` with dry_run | Safe to retry |
| `DestructiveHint: true` | `delete_*` | Irreversible action |
| `OpenWorldHint: false` | All tools | Closed-world assumption |

### Cost Hints

Tools provide execution cost metadata:

```json
{
  "api_cost": "low|medium|high|very_high",
  "token_cost": 100,
  "execution_speed": "instant|fast|medium|slow|async",
  "rate_limit_impact": "none|minimal|moderate|high|burst",
  "requires_confirm": true
}
```

## Security Considerations

### Query Injection Protection

The server validates all DataPrime queries against injection patterns:

- SQL injection attempts (UNION SELECT, DROP, DELETE)
- Command injection via backticks or $()
- Path traversal attempts
- Encoded attack payloads

### Sensitive Actions

Tools marked with `RequiresConfirm: true`:

| Tool | Risk Level | Impact |
|------|------------|--------|
| `delete_alert` | High | Removes monitoring |
| `delete_policy` | Critical | Affects data routing |
| `delete_dashboard` | Medium | Removes visualization |
| `delete_e2m` | High | Stops metric generation |

### Rate Limiting

Built-in rate limiting protects against:
- Quota exhaustion
- Runaway automation
- Denial of service

```yaml
rate_limit:
  requests_per_second: 100
  burst: 20
  per_tool_limits:
    query_logs: 10/s
    investigate_incident: 2/s
```

## Inter-Agent Discovery (A2A)

### Discovery Endpoint

The server exposes `/.well-known/agent.json` for A2A discovery:

```bash
curl https://your-mcp-server/.well-known/agent.json
```

### Negotiation Protocol

1. Agent requests `/.well-known/agent.json`
2. Inspects `capabilities` and `authentication`
3. Calls `search_tools` or `list_tool_categories` to discover tools
4. Uses `describe_tools` to get schemas for needed tools
5. Executes tools with appropriate authentication

### Integration Examples

```python
# Python SDK integration
from mcp import Client

async with Client("stdio://logs-mcp-server") as client:
    # Discover available tools
    tools = await client.call_tool("search_tools", {"query": "investigate errors"})

    # Get schema for specific tool
    schema = await client.call_tool("describe_tools", {"names": ["query_logs"]})

    # Execute with budget
    result = await client.call_tool("query_logs", {
        "query": "severity == 'error'",
        "time_range": "1h",
        "budget": {"max_tokens": 2000}
    })
```

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2025-11 | Initial release |
| 1.0.0 | 2025-12 | Added IAM action mapping, A2A discovery |

## References

- [IBM Cloud Logs Documentation](https://cloud.ibm.com/docs/cloud-logs)
- [IBM Cloud Logs IAM Actions](https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-iam-actions)
- [Granting Access to IBM Cloud Logs](https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-iam-assign-access)
- [MCP Specification](https://spec.modelcontextprotocol.io)
- [DataPrime Query Language](https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-dataprime-reference)
