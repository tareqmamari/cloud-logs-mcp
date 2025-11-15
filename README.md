# IBM Cloud Logs MCP Server

Model Context Protocol (MCP) server for IBM Cloud Logs, enabling AI assistants to interact with IBM Cloud Logs instances.

**Version**: 0.2.0 | **Go**: 1.23.2+

---

## Overview

This MCP server provides comprehensive access to IBM Cloud Logs through 70+ tools covering queries, alerts, dashboards, policies, webhooks, and more.

**Key Features:**
- Complete IBM Cloud Logs API coverage (70+ tools)
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

**Option 2: Build from Source**

```bash
git clone https://github.com/tareqmamari/logs-mcp-server.git
cd logs-mcp-server
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
        "LOGS_API_KEY": "your-ibm-cloud-api-key",
        "LOGS_REGION": "us-south"
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
$env:LOGS_REGION = "us-south"

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
     - `LOGS_REGION`: Your IBM Cloud region

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

70+ tools organized by functionality:

#### Query Operations (5 tools)
- `query_logs`, `submit_background_query`, `get_background_query_status`, `get_background_query_data`, `cancel_background_query`

#### Log Ingestion (1 tool)
- `ingest_logs`

#### Alert Management (10 tools)
- `list_alerts`, `get_alert`, `create_alert`, `update_alert`, `delete_alert`
- `list_alert_definitions`, `get_alert_definition`, `create_alert_definition`, `update_alert_definition`, `delete_alert_definition`

#### Dashboard Management (10 tools)
- `list_dashboards`, `get_dashboard`, `create_dashboard`, `update_dashboard`, `delete_dashboard`
- `list_dashboard_folders`, `move_dashboard_to_folder`, `pin_dashboard`, `unpin_dashboard`, `set_default_dashboard`

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

## Usage Examples

```
"Search logs for errors in the last hour"
"Create an alert when error rate exceeds 100 per minute"
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
| `LOGS_SERVICE_URL` | IBM Cloud Logs endpoint |
| `LOGS_API_KEY` | IBM Cloud API key |
| `LOGS_REGION` | IBM Cloud region |

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

**Best Practices:**
- Use environment variables for API keys (never hardcode)
- Use service IDs instead of personal API keys for production
- Enable TLS verification (`LOGS_TLS_VERIFY=true`)
- Rotate API keys regularly (recommended: 90 days)
- Apply principle of least privilege (Viewer/Operator roles)

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
├── main.go                    # Entry point
├── internal/
│   ├── auth/                  # IBM Cloud IAM authentication
│   ├── client/                # HTTP client with retry/rate limiting
│   ├── config/                # Configuration management
│   ├── tools/                 # MCP tool implementations (70+ tools)
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

- **Issues**: [GitHub Issues](https://github.com/tareqmamari/logs-mcp-server/issues)
- **IBM Cloud Logs Docs**: https://cloud.ibm.com/docs/cloud-logs
- **IBM Cloud Logs API**: https://cloud.ibm.com/apidocs/logs-service-api

---

## License

Copyright (c) 2025. All rights reserved.
