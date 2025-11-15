# IBM Cloud Logs MCP Server

A production-ready Model Context Protocol (MCP) server for IBM Cloud Logs, enabling AI assistants to interact with your cloud logging infrastructure securely and efficiently.

**Version**: 0.1.0 | **Go**: 1.22+ | **Status**: Production Ready ✅

---

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Single Instance](#single-instance-setup)
  - [Multiple Instances](#multiple-instances-setup)
  - [Environment Variables](#environment-variables)
- [Available Tools](#available-tools)
- [Usage Examples](#usage-examples)
- [Architecture](#architecture)
- [Security](#security)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
  - [Updating API Definitions](#updating-api-definitions)
- [Contributing](#contributing)

---

## Features

### Production-Ready Capabilities
- ✅ **Complete API Coverage** - 70+ tools for all IBM Cloud Logs operations
- ✅ **IBM Cloud IAM** - Automatic bearer token generation and refresh
- ✅ **Retry Logic** - Exponential backoff with configurable attempts
- ✅ **Rate Limiting** - Configurable requests/second with burst support
- ✅ **Health Checks** - Authentication and API connectivity validation
- ✅ **Metrics Tracking** - Request counts, latency, errors, tool usage
- ✅ **Structured Logging** - JSON/console formats with configurable levels
- ✅ **TLS Verification** - Secure HTTPS with certificate validation
- ✅ **Connection Pooling** - Efficient resource management
- ✅ **Graceful Shutdown** - Proper cleanup and connection draining

### API Operations
- **Queries**: Synchronous and asynchronous (background) log queries
- **Log Ingestion**: Push, add, or ingest log entries in real-time
- **Alerts**: Create, read, update, delete, and manage alert definitions
- **Dashboards**: Create, manage, and organize dashboards with widgets and visualizations
- **Policies**: Data retention and access policies
- **Webhooks**: Outgoing webhook integrations
- **Events-to-Metrics (E2M)**: Convert log events to metrics
- **Rule Groups**: Log processing and transformation rules
- **Data Access Rules**: Fine-grained access control
- **Enrichments**: Log enrichment configurations
- **Views**: Saved query views

---

## Quick Start

### Installation

#### Option 1: Homebrew (Recommended)

```bash
# Add the tap
brew tap tareqmamari/tap

# Install
brew install logs-mcp-server

# Verify installation
logs-mcp-server --version
```

The binary will be installed at `/usr/local/bin/logs-mcp-server` (Intel) or `/opt/homebrew/bin/logs-mcp-server` (Apple Silicon).

To find the exact path:
```bash
which logs-mcp-server
```

#### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/tareqmamari/logs-mcp-server.git
cd logs-mcp-server

# Download dependencies and build
make deps
make build
```

Binary will be at `./bin/logs-mcp-server` - note this absolute path for step 3.

### Step 2: Get Your IBM Cloud Credentials

1. **API Key**:
   - Go to [IBM Cloud Console](https://cloud.ibm.com/iam/apikeys)
   - Click **Create** → Give it a name → **Copy the key immediately** (you won't see it again)

2. **Service URL** (format: `https://{instance-id}.api.{region}.logs.cloud.ibm.com`):
   - Go to [Resource List](https://cloud.ibm.com/resources) → Cloud Logs
   - Click your instance → Copy the instance ID from the details
   - Construct URL: `https://<instance-id>.api.<region>.logs.cloud.ibm.com`

3. **Region**: `us-south`, `us-east`, `eu-de`, `eu-gb`, `au-syd`, `jp-tok`, etc.

### Step 3: Configure Claude Desktop

Edit your Claude Desktop config file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

Add this configuration:

**If installed via Homebrew:**

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ibm-cloud-api-key-here",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

**If built from source:**

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/absolute/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ibm-cloud-api-key-here",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

**Tip**: If installed via Homebrew, you can use just `logs-mcp-server` as the command (no path needed). To verify the path, run `which logs-mcp-server`.

**Important**:
- Replace `/absolute/path/to/logs-mcp-server` with the actual full path from step 1
- Replace `your-instance-id` with your actual instance ID
- Replace `your-ibm-cloud-api-key-here` with the API key from step 2
- Replace `us-south` with your actual region

### Step 4: Restart Claude Desktop

**Completely quit and restart Claude Desktop** for changes to take effect.

### Step 5: Verify It Works

In Claude Desktop, try asking:
- "List my IBM Cloud Logs alerts"
- "Query logs from the last hour"

You should see the MCP server tools being used in Claude's responses.

---

## Using with Different AI Platforms

The IBM Cloud Logs MCP Server works with any MCP-compatible AI assistant. Here's how to set it up for popular platforms:

### Claude Desktop

**Configuration file locations:**
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

**Configuration:**

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/absolute/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ibm-cloud-api-key",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

**After configuration:**
1. Completely quit Claude Desktop (Cmd+Q on macOS)
2. Restart the application
3. Start a new conversation and ask: "List my IBM Cloud Logs alerts"

### GitHub Copilot

GitHub Copilot with MCP support (available in VS Code Insiders):

**Prerequisites:**
- VS Code Insiders
- GitHub Copilot Chat extension

**Setup:**

1. Open VS Code Settings (JSON)
2. Add MCP server configuration:

```json
{
  "github.copilot.chat.mcp.servers": {
    "ibm-cloud-logs": {
      "command": "/absolute/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ibm-cloud-api-key",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

3. Restart VS Code
4. Open Copilot Chat and ask: "Query my IBM Cloud Logs for errors in the last hour"

### Cline (VS Code Extension)

Cline is a VS Code extension that supports MCP servers:

**Setup:**

1. Install Cline extension from VS Code Marketplace
2. Open Cline settings
3. Add MCP server in the MCP Servers section:

```json
{
  "ibm-cloud-logs": {
    "command": "/absolute/path/to/logs-mcp-server/bin/logs-mcp-server",
    "env": {
      "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
      "LOGS_API_KEY": "your-ibm-cloud-api-key",
      "LOGS_REGION": "us-south"
    }
  }
}
```

4. Restart Cline
5. Start using IBM Cloud Logs tools in your conversations

### Other MCP-Compatible Clients

The server follows the official [Model Context Protocol specification](https://modelcontextprotocol.io/), so it works with any MCP-compatible client.

**Generic setup pattern:**

1. **Identify your client's MCP configuration file/location**
2. **Add server configuration with:**
   - Command: Full path to `logs-mcp-server` binary
   - Environment variables: `LOGS_SERVICE_URL`, `LOGS_API_KEY`, `LOGS_REGION`
3. **Restart your MCP client**
4. **Verify** by asking the AI to list alerts or query logs

**Common configuration structure:**

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/path/to/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "...",
        "LOGS_API_KEY": "...",
        "LOGS_REGION": "..."
      }
    }
  }
}
```

### Programmatic Usage (Python)

You can also interact with the MCP server programmatically:

```python
#!/usr/bin/env python3
import json
import subprocess

# Start MCP server
env = {
    "LOGS_SERVICE_URL": "https://instance-id.api.us-south.logs.cloud.ibm.com",
    "LOGS_API_KEY": "your-api-key",
    "LOGS_REGION": "us-south"
}

proc = subprocess.Popen(
    ["/path/to/logs-mcp-server"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    env=env,
    text=True
)

# Send MCP initialize request
initialize = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "my-client", "version": "1.0.0"}
    }
}

proc.stdin.write(json.dumps(initialize) + '\n')
proc.stdin.flush()
response = proc.stdout.readline()
print(json.loads(response))

# Call query_logs tool
query_request = {
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "query_logs",
        "arguments": {
            "query": "source logs | limit 10",
            "tier": "frequent_search",
            "syntax": "dataprime"
        }
    }
}

proc.stdin.write(json.dumps(query_request) + '\n')
proc.stdin.flush()
response = proc.stdout.readline()
print(json.loads(response))
```

### Security Notes for All Platforms

**⚠️ Important:**
- Never commit configuration files with API keys to version control
- Use environment variables or secret managers for API keys
- Regularly rotate your IBM Cloud API keys (recommended: every 90 days)
- Use service IDs instead of personal API keys for production
- See [SECURITY.md](SECURITY.md) for comprehensive security guidance

### Setup Pre-Commit Hooks (Developers)

To prevent accidentally committing secrets:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run on all files (first time)
pre-commit run --all-files
```

This will automatically scan for:
- IBM Cloud API keys
- AWS keys, GitHub tokens, private keys
- High-entropy strings that might be secrets
- Code formatting issues

---

## Configuration

### Multiple Instances Setup

Configure multiple IBM Cloud Logs instances for different environments/regions:

```json
{
  "mcpServers": {
    "logs-production": {
      "command": "/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-abc.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "prod-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Production US"
      }
    },
    "logs-staging": {
      "command": "/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://stage-def.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "staging-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Staging US"
      }
    },
    "logs-eu-production": {
      "command": "/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://eu-ghi.api.eu-de.logs.cloud.ibm.com",
        "LOGS_API_KEY": "eu-api-key",
        "LOGS_REGION": "eu-de",
        "LOGS_INSTANCE_NAME": "Production EU"
      }
    }
  }
}
```

**Use cases**:
- Multiple environments (production, staging, development)
- Multiple regions (US, EU, Asia Pacific)
- Team-based separation
- Compliance zones (PCI, HIPAA, etc.)

**Query specific instances**:
```
"Show me production alerts"           → Uses logs-production
"Query EU logs for errors"            → Uses logs-eu-production
"Compare staging vs production"       → Uses both instances
```

### Alternative Configuration Methods

#### Method 1: Terminal Environment Variables

```bash
# Export variables
export LOGS_SERVICE_URL="https://instance-id.api.us-south.logs.cloud.ibm.com"
export LOGS_API_KEY="your-api-key"
export LOGS_REGION="us-south"

# Run server
./bin/logs-mcp-server
```

#### Method 2: One-Line Export

```bash
LOGS_SERVICE_URL="..." LOGS_API_KEY="..." LOGS_REGION="us-south" ./bin/logs-mcp-server
```

#### Method 3: .env File (Development)

```bash
# Create .env file
cp .env.example .env
# Edit with your credentials
./bin/logs-mcp-server  # Automatically loads .env
```

### Environment Variables

#### Required
| Variable | Description | Example |
|----------|-------------|---------|
| `LOGS_SERVICE_URL` | IBM Cloud Logs endpoint | `https://abc123.api.us-south.logs.cloud.ibm.com` |
| `LOGS_API_KEY` | IBM Cloud API key | `your-api-key` |

#### Recommended
| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_REGION` | - | IBM Cloud region (`us-south`, `eu-de`, etc.) |
| `LOGS_INSTANCE_NAME` | - | Friendly name for logs (e.g., "Production US") |
| `LOGS_IAM_URL` | - | Custom IAM endpoint URL (optional) |

#### Optional - Performance
| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_TIMEOUT` | `30s` | HTTP request timeout |
| `LOGS_MAX_RETRIES` | `3` | Maximum retry attempts |
| `LOGS_RATE_LIMIT` | `100` | Requests per second |
| `LOGS_RATE_LIMIT_BURST` | `20` | Burst capacity |
| `LOGS_ENABLE_RATE_LIMIT` | `true` | Enable rate limiting |

#### Optional - Logging & Security
| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `json` | Log format (`json`, `console`) |
| `LOGS_TLS_VERIFY` | `true` | Verify TLS certificates |
| `ENVIRONMENT` | - | Environment name (`production`, `development`) |

**Configuration Priority** (highest to lowest):
1. Environment variables
2. JSON config file (if `CONFIG_FILE` is set)
3. Default values

---

## Available Tools

The MCP server provides **70+ tools** covering all IBM Cloud Logs operations:

### Query Operations (5 tools)
- `query_logs` - Execute synchronous queries
- `submit_background_query` - Submit long-running queries
- `get_background_query_status` - Check query status
- `get_background_query_data` - Retrieve query results
- `cancel_background_query` - Cancel running query

### Log Ingestion (1 tool)
- `ingest_logs` - Ingest, push, or add log entries to IBM Cloud Logs for real-time log ingestion

### Alert Management (10 tools)
- `get_alert`, `list_alerts`, `create_alert`, `update_alert`, `delete_alert`
- `get_alert_definition`, `list_alert_definitions`, `create_alert_definition`, `update_alert_definition`, `delete_alert_definition`

### Dashboard Management (10 tools)
- `list_dashboards` - List all dashboards in the catalog
- `get_dashboard` - Get a specific dashboard by ID
- `create_dashboard` - Create a new dashboard with widgets and layout
- `update_dashboard` - Update an existing dashboard (replaces entire dashboard)
- `delete_dashboard` - Delete a dashboard
- `list_dashboard_folders` - List all dashboard folders
- `move_dashboard_to_folder` - Move a dashboard to a specific folder
- `pin_dashboard` - Pin a dashboard for quick access
- `unpin_dashboard` - Unpin a dashboard
- `set_default_dashboard` - Set a dashboard as the default

### Rule Groups (5 tools)
- `get_rule_group`, `list_rule_groups`, `create_rule_group`, `update_rule_group`, `delete_rule_group`

### Policies (5 tools)
- `get_policy`, `list_policies`, `create_policy`, `update_policy`, `delete_policy`

### Webhooks (5 tools)
- `get_outgoing_webhook`, `list_outgoing_webhooks`, `create_outgoing_webhook`, `update_outgoing_webhook`, `delete_outgoing_webhook`

### Events to Metrics - E2M (5 tools)
- `get_e2m`, `list_e2m`, `create_e2m`, `replace_e2m`, `delete_e2m`

### Data Access Rules (5 tools)
- `list_data_access_rules`, `get_data_access_rule`, `create_data_access_rule`, `update_data_access_rule`, `delete_data_access_rule`

### Enrichments (5 tools)
- `list_enrichments`, `get_enrichment`, `create_enrichment`, `update_enrichment`, `delete_enrichment`

### Views (5 tools)
- `list_views`, `get_view`, `create_view`, `replace_view`, `delete_view`

---

## Usage Examples

Once configured, you can ask your AI assistant:

### Querying Logs
```
"Search logs for errors in the last hour"
"Query production logs where severity is CRITICAL"
"Run a background query for all warnings from yesterday"
"What's the status of query abc-123?"
```

### Managing Alerts
```
"Show me all active alerts"
"Create an alert when error rate exceeds 100 per minute"
"Update alert xyz-456 to increase the threshold"
"Delete alert abc-123"
```

### Working with Dashboards
```
"List all my dashboards"
"Create a dashboard showing error rates by severity"
"Show me the dashboard with ID abc-123"
"Pin the production overview dashboard"
"Move this dashboard to the monitoring folder"
"Set dashboard xyz as the default"
```

### Ingesting Logs
```
"Ingest a test log message for my-app"
"Push these error logs to IBM Cloud Logs"
"Add application logs with severity level 5"
```

### Working with Policies
```
"List all retention policies"
"Create a policy to keep audit logs for 90 days"
"Show me all data access rules"
```

### Webhooks
```
"List all my webhooks"
"Create a webhook to send alerts to Slack"
"Update webhook to change the URL"
```

### Multi-Instance Queries
```
"Show me production alerts"
"Query staging logs for API errors"
"Compare error rates between US and EU production"
"List all alerts across all my log instances"
```

### Workflow Prompts (Advanced)

The server includes pre-built workflow prompts for common scenarios:

**Investigate Error Spikes:**
```
"I'm seeing high error rates - help me investigate"
→ Uses investigate_errors prompt
→ Guides you through: query errors, check alerts, review definitions, analyze policies
```

**Setup Monitoring:**
```
"Help me set up monitoring for my-service"
→ Uses setup_monitoring prompt
→ Walks through: create alert definition, webhook, alert, and policy
```

**Test Log Ingestion:**
```
"Help me test log ingestion for my-app"
→ Uses test_log_ingestion prompt
→ Guides through: ingest test logs, verify ingestion, check details
```

**Create Dashboard:**
```
"Help me create a dashboard for monitoring errors"
→ Uses create_dashboard_workflow prompt
→ Walks through: design layout, add widgets, configure queries
```

**Compare Environments:**
```
"Compare production and staging logs"
→ Uses compare_environments prompt
→ Analyzes: error patterns, alert configs, policy differences
```

**Debug Issues:**
```
"Debug error: 'database connection timeout'"
→ Uses debugging_workflow prompt
→ Systematic approach: search logs, analyze context, check resources, correlate
```

**Optimize Costs:**
```
"How can I optimize my log retention costs?"
→ Uses optimize_retention prompt
→ Reviews: policies, E2M conversions, access rules, enrichments
```

---

## Architecture

### System Overview

```
┌─────────────────┐
│  MCP Client     │
│  (e.g., Claude) │
└────────┬────────┘
         │ MCP Protocol
         ▼
┌─────────────────────────────────────┐
│     MCP Server (Go)                 │
│  ┌───────────────────────────────┐  │
│  │  Tool Registry (45+ tools)    │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  HTTP Client                  │  │
│  │  - Rate Limiting              │  │
│  │  - Retry Logic                │  │
│  │  - Connection Pooling         │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  IBM Cloud IAM Auth           │  │
│  │  - Token Generation           │  │
│  │  - Auto Refresh               │  │
│  └───────────────────────────────┘  │
└─────────────┬───────────────────────┘
              │ HTTPS
              ▼
     ┌────────────────────┐
     │  IBM Cloud Logs    │
     │  REST API          │
     └────────────────────┘
```

### Components

| Component | File | Purpose |
|-----------|------|---------|
| **Main** | `main.go` | Entry point, initialization, graceful shutdown |
| **MCP Server** | `internal/server/` | MCP protocol handler, tool registration |
| **API Client** | `internal/client/` | HTTP client with retry and rate limiting |
| **Authenticator** | `internal/auth/` | IBM Cloud IAM token management |
| **Config** | `internal/config/` | Configuration loading and validation |
| **Tools** | `internal/tools/` | 45+ MCP tool implementations |
| **Health** | `internal/health/` | Health checks for auth and API |
| **Metrics** | `internal/metrics/` | Operational metrics tracking |

### Request Flow

1. **AI assistant** sends MCP tool request
2. **MCP server** validates request, extracts parameters
3. **Authenticator** adds IBM Cloud bearer token
4. **Rate limiter** enforces request limits
5. **HTTP client** executes request with retry logic
6. **Response** parsed and formatted as MCP result
7. **Result** returned to AI assistant

### Technology Stack

- **Language**: Go 1.22+
- **MCP SDK**: `github.com/mark3labs/mcp-go` v0.8.0
- **IBM SDK**: `github.com/IBM/go-sdk-core/v5` v5.17.4
- **Logging**: `go.uber.org/zap` v1.27.0
- **Rate Limiting**: `golang.org/x/time/rate`
- **Config**: `github.com/joho/godotenv` v1.5.1

---

## Security

### Best Practices Implemented

✅ **Credential Management**
- API keys only in environment variables or secure vaults
- Never logged or exposed in output
- Automatic redaction in logs and errors

✅ **Network Security**
- TLS 1.2+ for all connections
- Certificate verification enabled by default
- Support for private endpoints

✅ **Input Validation**
- All tool parameters validated
- Type checking on user inputs
- Required vs optional enforcement

✅ **Error Handling**
- No sensitive data in error messages
- Proper error wrapping with context
- Secure logging practices

✅ **Runtime Security**
- Runs as non-root in Docker
- Read-only root filesystem support
- Minimal attack surface

### IAM Permissions

**Recommended IAM roles**:
- **Viewer** - Read-only access (queries, listing)
- **Operator** - Read + limited write operations
- **Editor** - Full management (use sparingly)

**Create service ID** (recommended over personal API keys):
```bash
ibmcloud iam service-id-create logs-mcp-server "MCP access to Cloud Logs"
ibmcloud iam service-policy-create logs-mcp-server \
  --roles Viewer,Operator \
  --service-name logs
```

### Security Checklist

For production deployment:
- [ ] API keys in secret manager (not .env files)
- [ ] `LOGS_TLS_VERIFY=true` enforced
- [ ] Running as non-root user
- [ ] Resource limits configured
- [ ] Rate limiting enabled
- [ ] Structured logging enabled
- [ ] Access controls configured
- [ ] Network policies restricting traffic
- [ ] Regular API key rotation (90 days)
- [ ] Audit logging enabled

**For detailed security guidance**, see [SECURITY.md](SECURITY.md).

---

## Performance

### Optimization Features

**Connection Pooling**
- Max idle connections: 10 (configurable)
- Idle timeout: 90s (configurable)
- Persistent connections reused

**Rate Limiting**
- Default: 100 requests/second
- Burst: 20 requests
- Prevents API quota exhaustion

**Retry Strategy**
- Exponential backoff with jitter
- Initial wait: 1s → Max wait: 30s
- Retries on: 429, 500, 502, 503, 504
- Max attempts: 3 (configurable)

**Resource Usage**
- Memory: ~50MB baseline, <200MB under load
- CPU: Minimal (I/O bound)
- Binary size: 8.8MB (optimized build)

### Performance Tuning

```json
{
  "env": {
    "LOGS_TIMEOUT": "60s",           // Longer timeout for slow queries
    "LOGS_MAX_RETRIES": "5",         // More retries for flaky networks
    "LOGS_RATE_LIMIT": "200",        // Higher throughput
    "LOGS_RATE_LIMIT_BURST": "50"    // Larger burst capacity
  }
}
```

---

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| `LOGS_SERVICE_URL is required` | Set environment variable with your instance endpoint |
| `401 Unauthorized` | Check API key validity and IAM permissions |
| `429 Too Many Requests` | Reduce `LOGS_RATE_LIMIT` or request quota increase |
| `Connection timeout` | Increase `LOGS_TIMEOUT` or check network |
| Tools not showing | Use absolute binary path, restart MCP client completely |

### Debug Mode

Enable detailed logging:

```bash
export LOG_LEVEL=debug
export LOG_FORMAT=console
./bin/logs-mcp-server
```

Or in MCP client config:
```json
{
  "env": {
    "LOG_LEVEL": "debug",
    "LOG_FORMAT": "console"
  }
}
```

### Health Checks

The server performs automatic health checks:
- **Authentication**: Validates IBM Cloud IAM token
- **API Connectivity**: Tests connection to Cloud Logs API

Check logs on startup for health status.

### Log Locations

**MCP Client logs**:
- macOS: `~/Library/Logs/Claude/mcp*.log`
- Linux: `~/.config/Claude/logs/`
- Windows: `%APPDATA%\Claude\logs\`

Look for:
- "Starting IBM Cloud Logs MCP Server"
- "IBM Cloud IAM authenticator initialized successfully"
- "Registered all MCP tools"

---

## Development

### Project Structure

```
logs-mcp-server/
├── main.go                          # Application entry point
├── go.mod, go.sum                   # Go dependencies
├── Makefile                         # Build automation (20+ targets)
├── Dockerfile                       # Multi-stage container build
├── internal/
│   ├── auth/                        # IBM Cloud IAM authentication
│   │   ├── authenticator.go
│   │   └── authenticator_test.go
│   ├── client/                      # HTTP client
│   │   └── client.go
│   ├── config/                      # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── health/                      # Health checks
│   │   └── health.go
│   ├── metrics/                     # Metrics tracking
│   │   └── metrics.go
│   ├── server/                      # MCP server
│   │   └── server.go
│   └── tools/                       # MCP tool implementations
│       ├── base.go, base_test.go
│       ├── alerts.go
│       ├── queries.go
│       ├── alert_definitions.go
│       └── all_tools.go
├── README.md                        # This file
├── SECURITY.md                      # Security best practices
├── CONTRIBUTING.md                  # Contribution guidelines
└── .env.example                     # Example configuration
```

### Available Make Targets

```bash
make help              # Show all targets
make build             # Build for current platform
make build-all         # Build for all platforms
make test              # Run tests with coverage
make lint              # Run linters
make check             # Run all quality checks
make docker-build      # Build Docker image
make clean             # Clean build artifacts
make release           # Create release artifacts

# API update helpers
make backup-api        # Backup current API definition
make compare-api       # Compare old and new API versions
make list-operations   # List all API operations
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run specific package
go test ./internal/config -v
```

### Building

```bash
# Standard build
make build

# Optimized production build
CGO_ENABLED=0 go build -ldflags="-s -w" -o logs-mcp-server .

# Cross-platform builds
make build-all  # Creates binaries for Linux, macOS, Windows (AMD64, ARM64)
```

### Docker Deployment

```bash
# Build image
make docker-build

# Run container
docker run --env-file .env logs-mcp-server:latest
```

### Updating API Definitions

When the IBM Cloud Logs API (`logs-service-api.json`) is updated with new endpoints or changes:

#### 1. Backup and Compare

```bash
# Backup current API definition
make backup-api

# Get new API definition
cp /path/to/new-api.json logs-service-api.json

# Compare changes
make compare-api
```

The comparison script will show:
- New operations (need to implement)
- Removed operations (need to deprecate/remove)
- Changed endpoints
- API version changes

#### 2. List Current Operations

```bash
# See all current operations
make list-operations
```

#### 3. Implement Changes

For **new operations**, add tools in `internal/tools/`:

```go
type NewFeatureTool struct {
    BaseTool
}

func NewNewFeatureTool(client *client.Client, logger *zap.Logger) *NewFeatureTool {
    return &NewFeatureTool{BaseTool: BaseTool{client: client, logger: logger}}
}

func (t *NewFeatureTool) Name() string { return "new_feature" }

func (t *NewFeatureTool) Description() string {
    return "Description from API spec"
}

func (t *NewFeatureTool) InputSchema() mcp.ToolInputSchema {
    return mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "param": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        Required: []string{"param"},
    }
}

func (t *NewFeatureTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
    param, err := GetStringParam(arguments, "param", true)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    req := &client.Request{
        Method: "POST",
        Path:   "/v1/new-feature",
        Body:   map[string]interface{}{"param": param},
    }

    result, err := t.ExecuteRequest(ctx, req)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    return t.FormatResponse(result)
}
```

For **modified operations**, update the existing tool's InputSchema and Execute methods.

#### 4. Register New Tools

Add to `internal/server/server.go`:

```go
func (s *Server) RegisterTools() {
    // ... existing tools ...
    s.registerTool(tools.NewNewFeatureTool(s.apiClient, s.logger))
}
```

#### 5. Add Tests

Create or update `*_test.go` files:

```go
func TestNewFeatureTool(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    tool := NewNewFeatureTool(nil, logger)

    if tool.Name() != "new_feature" {
        t.Errorf("Expected 'new_feature', got %s", tool.Name())
    }

    schema := tool.InputSchema()
    if len(schema.Required) == 0 {
        t.Error("Expected required parameters")
    }
}
```

#### 6. Update Documentation

Add the new tool to this README's [Available Tools](#available-tools) section.

#### 7. Test and Build

```bash
make test
make build
```

#### Tool Organization

- `internal/tools/alerts.go` - Alert management
- `internal/tools/queries.go` - Query execution
- `internal/tools/all_tools.go` - Policies, webhooks, E2M, enrichments, views, etc.
- `internal/tools/base.go` - Base tool helpers

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for new functionality
4. Ensure all tests pass (`make check`)
5. Commit with clear messages
6. Submit a pull request

**See [CONTRIBUTING.md](CONTRIBUTING.md)** for detailed guidelines.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/tareqmamari/logs-mcp-server/issues)
- **IBM Cloud Logs Docs**: https://cloud.ibm.com/docs/cloud-logs
- **IBM Cloud Logs API**: https://cloud.ibm.com/apidocs/logs-service-api

---

## License

Copyright (c) 2025. All rights reserved.

---

## Acknowledgments

Built with production-grade best practices for:
- Security and credential management
- Reliability and error handling
- Performance and resource efficiency
- Observability and debugging
- Developer experience and documentation

**Stats**: ~3,500 lines of Go code | 70+ MCP tools | 9.2MB binary | Production ready ✅
