# IBM Cloud Logs MCP Server

[![CI](https://github.com/tareqmamari/cloud-logs-mcp/actions/workflows/ci.yaml/badge.svg)](https://github.com/tareqmamari/cloud-logs-mcp/actions/workflows/ci.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tareqmamari/cloud-logs-mcp)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/tareqmamari/cloud-logs-mcp)](https://goreportcard.com/report/github.com/tareqmamari/cloud-logs-mcp)
[![Release](https://img.shields.io/github/v/release/tareqmamari/cloud-logs-mcp)](https://github.com/tareqmamari/cloud-logs-mcp/releases)
[![License](https://img.shields.io/github/license/tareqmamari/cloud-logs-mcp)](LICENSE)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/tareqmamari/cloud-logs-mcp/badge)](https://scorecard.dev/viewer/?uri=github.com/tareqmamari/cloud-logs-mcp)

Model Context Protocol (MCP) server for IBM Cloud Logs, enabling AI assistants to interact with IBM Cloud Logs instances. Includes 8 portable Agent Skills for use with Claude Code, Cursor, Gemini CLI, GitHub Copilot, and 30+ other agents.

---

## Overview

This project provides two complementary ways to work with IBM Cloud Logs through AI agents:

| | MCP Server | Agent Skills |
|---|---|---|
| **What** | Running Go server with 88 tools via JSON-RPC | 8 portable instruction bundles (markdown + JSON) |
| **When** | Real-time log queries, CRUD operations, live monitoring | Query writing, architecture guidance, offline reference |
| **Requires** | Binary + API key + network | Nothing — loaded on-demand by your agent |
| **Works with** | Claude Desktop, any MCP client | Claude Code, Cursor, Gemini CLI, GitHub Copilot, 30+ agents |

**Key Features:**
- Complete IBM Cloud Logs API coverage (88 tools)
- 8 embedded Agent Skills following the [agentskills.io](https://agentskills.io) open standard
- IBM Cloud IAM authentication with automatic token refresh
- Retry logic with exponential backoff
- Configurable rate limiting
- Health checks and metrics tracking
- Structured logging

---

## Quick Start

### Installation

**Option 1: Homebrew (Recommended)**

```bash
brew tap tareqmamari/tap
brew install logs-mcp-server
logs-mcp-server --version
```

Agent Skills are automatically installed to `~/.agents/skills/` after Homebrew install.

**Option 2: Build from Source**

```bash
git clone https://github.com/tareqmamari/cloud-logs-mcp.git
cd cloud-logs-mcp
make deps && make build
```

### Configuration

#### Prerequisites

1. **Get IBM Cloud credentials**:
   - API Key: https://cloud.ibm.com/iam/apikeys
   - Service URL: `https://[instance-id].api.[region].logs.cloud.ibm.com`
   - Region: `us-south`, `eu-de`, `au-syd`, etc.

#### For Claude Desktop

**Configuration file**: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://[your-instance-id].api.[region].logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ibm-cloud-api-key"
      }
    }
  }
}
```

**After setup**: Restart Claude Desktop and ask "List my IBM Cloud Logs alerts"

#### For Microsoft 365 Copilot

**Prerequisites**:
- Microsoft 365 Copilot subscription (Enterprise or Business)
- Windows 11 or Windows 10 with Microsoft 365 apps
- Admin access to configure organization settings (for enterprise deployment)

**Configuration Options:**

**Option 1: Personal Setup (Environment Variables)**

```powershell
# PowerShell (Windows)
$env:LOGS_SERVICE_URL = "https://[your-instance-id].api.[region].logs.cloud.ibm.com"
$env:LOGS_API_KEY = "your-ibm-cloud-api-key"

# Run the server
logs-mcp-server
```

**Option 2: Enterprise Deployment (Microsoft 365 Admin Center)**

For organization-wide deployment, configure through Microsoft 365 Admin Center:

1. Navigate to **Settings** > **Integrated apps** > **Copilot extensions**
2. Add custom MCP server with the following configuration:
   - **Name**: IBM Cloud Logs
   - **Command**: `logs-mcp-server`
   - **Environment Variables**:
     - `LOGS_SERVICE_URL`: Your IBM Cloud Logs endpoint
     - `LOGS_API_KEY`: Service ID API key (recommended for enterprise)

**After setup**:
- In Microsoft 365 Copilot, ask "Query my IBM Cloud Logs for errors"
- In Microsoft Teams, use "@Copilot list my dashboards in IBM Cloud Logs"
- In Outlook, ask "Show me alerts from IBM Cloud Logs"

**Note**: Microsoft 365 Copilot MCP support is currently in preview. Configuration steps may vary by tenant settings. See [CONTRIBUTING.md](CONTRIBUTING.md) for alternative setup methods.

#### For Other MCP Clients

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions for Cline, programmatic usage, and other MCP-compatible clients.

---

## API Reference

### Tools

88 tools organized by functionality:

#### Query Operations (5 tools)
- `query_logs`, `submit_background_query`, `get_background_query_status`, `get_background_query_data`, `cancel_background_query`

#### Log Ingestion (1 tool)
- `ingest_logs`

#### Alert Management (11 tools)
- `list_alerts`, `get_alert`, `create_alert`, `update_alert`, `delete_alert`
- `list_alert_definitions`, `get_alert_definition`, `create_alert_definition`, `update_alert_definition`, `delete_alert_definition`
- `suggest_alert` - **SRE-grade alert recommendations** (see [Alert Intelligence](#alert-intelligence) below)

#### Dashboard Management (14 tools)
- `list_dashboards`, `get_dashboard`, `create_dashboard`, `update_dashboard`, `delete_dashboard`
- `list_dashboard_folders`, `get_dashboard_folder`, `create_dashboard_folder`, `update_dashboard_folder`, `delete_dashboard_folder`
- `move_dashboard_to_folder`, `pin_dashboard`, `unpin_dashboard`, `set_default_dashboard`

#### Policies (5 tools)
- `list_policies`, `get_policy`, `create_policy`, `update_policy`, `delete_policy`

#### Webhooks (5 tools)
- `list_outgoing_webhooks`, `get_outgoing_webhook`, `create_outgoing_webhook`, `update_outgoing_webhook`, `delete_outgoing_webhook`

#### Events to Metrics - E2M (5 tools)
- `list_e2m`, `get_e2m`, `create_e2m`, `replace_e2m`, `delete_e2m`

#### Rule Groups (5 tools)
- `list_rule_groups`, `get_rule_group`, `create_rule_group`, `update_rule_group`, `delete_rule_group`

#### Data Access Rules (5 tools)
- `list_data_access_rules`, `get_data_access_rule`, `create_data_access_rule`, `update_data_access_rule`, `delete_data_access_rule`

#### Enrichments (5 tools)
- `list_enrichments`, `get_enrichment`, `create_enrichment`, `update_enrichment`, `delete_enrichment`

#### Streams (5 tools)
- `list_streams`, `get_stream`, `create_stream`, `update_stream`, `delete_stream`

#### Views (5 tools)
- `list_views`, `get_view`, `create_view`, `replace_view`, `delete_view`

### Resources

| Resource URI | Description |
|--------------|-------------|
| `config://current` | Server configuration |
| `metrics://server` | Server metrics |
| `health://status` | Health check status |

### Prompts

| Prompt | Description |
|--------|-------------|
| `investigate_errors` | Guide for investigating error spikes |
| `setup_monitoring` | Setup monitoring for a service |
| `test_log_ingestion` | Test log ingestion workflow |
| `create_dashboard_workflow` | Dashboard creation wizard |
| `compare_environments` | Compare different environments |
| `debugging_workflow` | Systematic debugging approach |
| `optimize_retention` | Optimize log retention costs |

---

## Alert Intelligence

The `suggest_alert` tool provides **SRE-grade alert recommendations** based on industry best practices from Google SRE, the RED/USE methodologies, and academic research on alerting.

### Why Use `suggest_alert`?

| Problem | How `suggest_alert` Helps |
|---------|---------------------------|
| **Alert fatigue** | Uses burn rate alerting to reduce noise by 90%+ |
| **False positives** | Multi-window validation prevents flapping alerts |
| **Static thresholds** | Suggests dynamic baselines for seasonal metrics |
| **Missing context** | Auto-generates runbook templates and actions |
| **Wrong methodology** | Auto-selects RED (services) vs USE (resources) |

### Quick Start

```
"Suggest an alert for high error rate on my API service with 99.9% SLO"
"What alerts should I create for my Kafka cluster?"
"Help me set up monitoring for my PostgreSQL database"
```

### Key Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `service_type` | Component type (auto-detected if not provided) | `web_service`, `database`, `message_queue` |
| `slo_target` | SLO target (enables burn rate alerting) | `0.999` (99.9%) |
| `is_user_facing` | Affects severity classification | `true` → P1 eligible |
| `use_case` | Natural language description | `"high latency on checkout"` |

### Supported Service Types

**RED Method** (Rate, Errors, Duration) for services:
- `web_service`, `api_gateway`, `worker`, `microservice`, `serverless`

**USE Method** (Utilization, Saturation, Errors) for resources:
- `database`, `cache`, `message_queue`, `kubernetes`, `storage`

### Example Output

```json
{
  "suggestions": [{
    "name": "API Error Rate - Fast Burn (Page)",
    "severity": "P1",
    "methodology": "RED",
    "burn_rate_condition": {
      "slo_target": 0.999,
      "burn_rate": 14.4,
      "window_duration": "1h",
      "consumption_percent": 2.0
    },
    "suggested_actions": [
      "1. Check error logs for patterns",
      "2. Review recent deployments",
      "3. Verify dependent service health"
    ],
    "runbook_url": "/runbooks/web_service/error-rate"
  }]
}
```

### References

The alerting engine implements recommendations from:
- [Google SRE Handbook - Alerting](https://sre.google/sre-book/monitoring-distributed-systems/)
- ["My Philosophy on Alerting"](https://docs.google.com/document/d/199PqyG3UsyXlwieHaqbGiWVa8eMWi8zzAn0YfcApr8Q) by Rob Ewaschuk
- [SRE Workbook - Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/)

---

## Agent Skills

The binary embeds 8 Agent Skills following the [agentskills.io](https://agentskills.io) open standard. Skills are portable instruction bundles that work across 30+ AI agents — no runtime, authentication, or network required.

### Available Skills

| Skill | Description |
|-------|-------------|
| `ibm-cloud-logs-query` | DataPrime and Lucene query writing, validation, and auto-correction |
| `ibm-cloud-logs-alerting` | SRE-grade alerting with RED/USE methodologies and burn rate math |
| `ibm-cloud-logs-incident-investigation` | Systematic incident investigation with heuristic pattern matching |
| `ibm-cloud-logs-dashboards` | Dashboard design with DataPrime-powered widgets |
| `ibm-cloud-logs-cost-optimization` | TCO policies, data tier selection, and Events-to-Metrics |
| `ibm-cloud-logs-ingestion` | Log ingestion, parsing rules, and enrichments |
| `ibm-cloud-logs-access-control` | Data access rules, audit logging, and compliance patterns |
| `ibm-cloud-logs-api-reference` | Full API reference for all 88 tool endpoints |

### Installing Skills

Skills are embedded in the binary. Use the `skills` subcommand to manage them.

```bash
# Install to ~/.agents/skills/ (user-level, available to all projects)
logs-mcp-server skills install

# Install to ./.agents/skills/ (project-level, current project only)
logs-mcp-server skills install --project

# List all available skills
logs-mcp-server skills list

# Remove installed skills
logs-mcp-server skills remove
```

If you installed via Homebrew, skills are automatically installed to `~/.agents/skills/` on first install.

### How Skills Work

Skills use a **progressive disclosure** model to minimize context window usage:

1. **Catalog** (~200 tokens) — your agent sees skill names and descriptions
2. **SKILL.md** (~300 lines) — loaded on-demand when a skill activates
3. **References** — detailed docs loaded only when deeper information is needed

This means skills consume **~2K tokens on-demand** compared to the MCP server's **~25K fixed overhead** per conversation. See [BENCHMARK.md](BENCHMARK.md) for a detailed comparison.

### Compatible Agents

Skills work with any agent that can read markdown files from `~/.agents/skills/` or `./.agents/skills/`:

- **Claude Code** — auto-discovers skills from `~/.agents/skills/`
- **Cursor** — reads project-level `.agents/skills/`
- **Gemini CLI** — reads `~/.agents/skills/`
- **GitHub Copilot** — reads project-level skills
- **Bob, Windsurf, Cline, Aider** — and 20+ more via the agentskills.io standard

### When to Use Skills vs MCP

| Scenario | Use |
|----------|-----|
| Query real-time logs | MCP |
| Write a DataPrime query (no execution) | Skills |
| Create or manage alerts | MCP |
| Design an alerting strategy | Skills |
| Debug a live production incident | MCP |
| Learn investigation methodology | Skills |
| Set up dashboards | MCP |
| Plan dashboard layout and queries | Skills |
| Offline query/config guidance (no execution) | Skills |
| Any CRUD operation on IBM Cloud Logs | MCP |

For maximum effectiveness, use both together — skills provide the domain knowledge, MCP executes the actions.

---

## Usage Examples

```
"Search logs for errors in the last hour"
"Create an alert when error rate exceeds 100 per minute"
"Suggest alerts for my web service with 99.9% SLO"
"What monitoring should I set up for my Redis cache?"
"List all my dashboards"
"Ingest a test log message for my-app"
"Show me all retention policies"
```

---

## Configuration

### Environment Variables

#### Required
| Variable | Description |
|----------|-------------|
| `LOGS_API_KEY` | IBM Cloud API key |

**Plus one of the following:**
- `LOGS_SERVICE_URL` - Full service endpoint URL (region and instance ID are auto-extracted), OR
- `LOGS_REGION` + `LOGS_INSTANCE_ID` - Region and instance ID (service URL is constructed)

#### Optional
| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_TIMEOUT` | `30s` | HTTP request timeout |
| `LOGS_MAX_RETRIES` | `3` | Maximum retry attempts |
| `LOGS_RATE_LIMIT` | `100` | Requests per second |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | Log format (json/console) |

### Multiple Instances

Configure multiple IBM Cloud Logs instances:

```json
{
  "mcpServers": {
    "logs-production": {
      "command": "logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "prod-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Production US"
      }
    },
    "logs-staging": {
      "command": "logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://stage-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "staging-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Staging US"
      }
    }
  }
}
```

---

## Security

**MCP Server Best Practices:**
- Use environment variables for API keys (never hardcode)
- Use service IDs instead of personal API keys for production
- Enable TLS verification (`LOGS_TLS_VERIFY=true`)
- Rotate API keys regularly (recommended: 90 days)
- Apply principle of least privilege (Viewer/Operator roles)
- Use data access rules to restrict log visibility by team or role
- Enable audit logging (`LOG_LEVEL=debug`) for compliance tracking

**Agent Skills Security:**
- Skills contain no credentials, tokens, or connection strings
- Embedded in the binary via `go:embed` — immutable after build
- Zero network attack surface — no API calls, no data access
- The `ibm-cloud-logs-access-control` skill includes security query templates (auth failures, privilege escalation, sensitive data access) and compliance patterns (GDPR, SOC 2, multi-tenant isolation)

See [BENCHMARK.md](BENCHMARK.md#7-security-analysis) for a detailed MCP vs Skills security comparison.

**See [SECURITY.md](SECURITY.md)** for comprehensive security guidance.

---

## Development

### Prerequisites
- Go 1.23.2+
- Make

### Building

```bash
make deps      # Download dependencies
make build     # Build binary
make test      # Run tests
make lint      # Run linters
```

### Project Structure

```
├── main.go                    # Entry point with skills subcommand
├── embed_skills.go            # Embeds .agents/skills/ into binary
├── .agents/skills/            # 8 Agent Skills (agentskills.io format)
│   ├── ibm-cloud-logs-query/         # DataPrime query writing & validation
│   ├── ibm-cloud-logs-alerting/      # SRE-grade alerting with burn rate math
│   ├── ibm-cloud-logs-incident-investigation/
│   ├── ibm-cloud-logs-dashboards/
│   ├── ibm-cloud-logs-cost-optimization/
│   ├── ibm-cloud-logs-ingestion/
│   ├── ibm-cloud-logs-access-control/
│   └── ibm-cloud-logs-api-reference/
├── internal/
│   ├── auth/                  # IBM Cloud IAM authentication
│   ├── client/                # HTTP client with retry/rate limiting
│   ├── config/                # Configuration management
│   ├── tools/                 # MCP tool implementations (88 tools)
│   ├── skills/                # Skill installer
│   ├── server/                # MCP server
│   ├── health/                # Health checks
│   └── metrics/               # Metrics tracking
├── Makefile                   # Build automation
└── .goreleaser.yaml           # Release configuration
```

**See [CONTRIBUTING.md](CONTRIBUTING.md)** for detailed development guide including:
- Setting up other MCP clients (GitHub Copilot, Cline, etc.)
- Updating API definitions
- Adding new tools
- Testing strategies
- Release process

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Tools not showing | Use absolute binary path, restart MCP client completely |
| `401 Unauthorized` | Check API key validity and IAM permissions |
| `429 Too Many Requests` | Reduce `LOGS_RATE_LIMIT` or request quota increase |
| Connection timeout | Increase `LOGS_TIMEOUT` or check network |

**Debug mode**:

```json
{
  "env": {
    "LOG_LEVEL": "debug",
    "LOG_FORMAT": "console"
  }
}
```

---

## Support

- **Issues**: [GitHub Issues](https://github.com/tareqmamari/cloud-logs-mcp/issues)
- **IBM Cloud Logs Docs**: https://cloud.ibm.com/docs/cloud-logs
- **IBM Cloud Logs API**: https://cloud.ibm.com/apidocs/logs-service-api

---

## License

Copyright (c) 2025. All rights reserved.
