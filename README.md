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
- [Contributing](#contributing)

---

## Features

### Production-Ready Capabilities
- ✅ **Complete API Coverage** - 45+ tools for all IBM Cloud Logs operations
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
- **Alerts**: Create, read, update, delete, and manage alert definitions
- **Policies**: Data retention and access policies
- **Webhooks**: Outgoing webhook integrations
- **Events-to-Metrics (E2M)**: Convert log events to metrics
- **Rule Groups**: Log processing and transformation rules
- **Data Access Rules**: Fine-grained access control
- **Enrichments**: Log enrichment configurations
- **Views**: Saved query views

---

## Quick Start

### Prerequisites

- Go 1.22+ installed ([download](https://go.dev/dl/))
- IBM Cloud Logs service instance
- IBM Cloud API key with appropriate permissions

### Installation

```bash
# Clone repository
git clone https://github.com/observability-c/logs-mcp-server.git
cd logs-mcp-server

# Download dependencies
make deps

# Build
make build
```

Binary will be at `./bin/logs-mcp-server`

### Get Your IBM Cloud Credentials

1. **Instance ID**:
   - Go to [IBM Cloud Console](https://cloud.ibm.com) → Resource List → Cloud Logs
   - Click your instance → Note the instance ID
   - Format: `https://{instance-id}.api.{region}.logs.cloud.ibm.com`

2. **API Key**:
   - Go to [IAM API Keys](https://cloud.ibm.com/iam/apikeys)
   - Click **Create** → Give it a name → Copy the key immediately

3. **Region**: `us-south`, `us-east`, `eu-de`, `eu-gb`, `au-syd`, `jp-tok`, etc.

---

## Configuration

### Single Instance Setup

**Recommended**: Configure directly in MCP client (e.g., Claude Desktop):

**macOS/Linux**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

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

**Important**:
- Use **absolute path** to the binary
- Replace `your-instance-id` with actual instance ID
- Replace `your-ibm-cloud-api-key` with your API key
- **Restart** your MCP client completely after editing

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

The MCP server provides **45+ tools** covering all IBM Cloud Logs operations:

### Query Operations (5 tools)
- `query_logs` - Execute synchronous queries
- `submit_background_query` - Submit long-running queries
- `get_background_query_status` - Check query status
- `get_background_query_data` - Retrieve query results
- `cancel_background_query` - Cancel running query

### Alert Management (10 tools)
- `get_alert`, `list_alerts`, `create_alert`, `update_alert`, `delete_alert`
- `get_alert_definition`, `list_alert_definitions`, `create_alert_definition`, `update_alert_definition`, `delete_alert_definition`

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

- **Issues**: [GitHub Issues](https://github.com/observability-c/logs-mcp-server/issues)
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

**Stats**: ~3,000 lines of Go code | 45+ MCP tools | 8.8MB binary | Production ready ✅
