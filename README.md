# IBM Cloud Logs MCP Server

A production-ready Model Context Protocol (MCP) server for IBM Cloud Logs, enabling AI assistants to interact with your cloud logging infrastructure securely and efficiently.

## Features

- **Complete API Coverage**: All IBM Cloud Logs API operations including alerts, queries, policies, webhooks, and more
- **Production-Ready**: Comprehensive error handling, retry logic, rate limiting, and observability
- **Secure**: TLS verification, API key authentication, configurable security controls
- **High Performance**: Connection pooling, concurrent request handling, efficient resource management
- **Flexible Configuration**: Environment variables, config files, or hybrid approaches
- **Observable**: Structured logging with configurable levels and formats
- **Reliable**: Automatic retries with exponential backoff, graceful shutdown handling

## Quick Start

### Prerequisites

- Go 1.22 or higher
- IBM Cloud Logs service instance
- IBM Cloud API key with appropriate permissions

### Installation

1. Clone the repository:
```bash
git clone https://github.com/observability-c/logs-mcp-server.git
cd logs-mcp-server
```

2. Install dependencies:
```bash
go mod download
```

3. Build the server:
```bash
go build -o logs-mcp-server .
```

### Configuration

Create a `.env` file in the project root:

```bash
# Required
LOGS_SERVICE_URL=https://your-instance-id.api.us-south.logs.cloud.ibm.com
LOGS_API_KEY=your-api-key-here
LOGS_REGION=us-south

# Optional - Performance Tuning
LOGS_TIMEOUT=30s
LOGS_MAX_RETRIES=3
LOGS_RATE_LIMIT=100
LOGS_RATE_LIMIT_BURST=20

# Optional - Security
LOGS_TLS_VERIFY=true

# Optional - Logging
LOG_LEVEL=info
LOG_FORMAT=json
ENVIRONMENT=production
```

### Running the Server

Start the MCP server:

```bash
./logs-mcp-server
```

Or with Go:

```bash
go run main.go
```

### Integrating with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/path/to/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-api-key-here",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

## Available Tools

The MCP server exposes the following tools for AI assistants:

### Query Operations

- `query_logs` - Execute synchronous queries against log data
- `submit_background_query` - Submit long-running background queries
- `get_background_query_status` - Check background query status
- `get_background_query_data` - Retrieve background query results
- `cancel_background_query` - Cancel a running background query

### Alert Management

- `get_alert` - Retrieve a specific alert
- `list_alerts` - List all alerts
- `create_alert` - Create a new alert
- `update_alert` - Update an existing alert
- `delete_alert` - Delete an alert

### Alert Definitions

- `get_alert_definition` - Retrieve alert definition
- `list_alert_definitions` - List all alert definitions
- `create_alert_definition` - Create new alert definition
- `update_alert_definition` - Update alert definition
- `delete_alert_definition` - Delete alert definition

### Rule Groups

- `get_rule_group` - Retrieve a rule group
- `list_rule_groups` - List all rule groups
- `create_rule_group` - Create a new rule group
- `update_rule_group` - Update a rule group
- `delete_rule_group` - Delete a rule group

### Policies

- `get_policy` - Retrieve a policy
- `list_policies` - List all policies
- `create_policy` - Create a new policy
- `update_policy` - Update a policy
- `delete_policy` - Delete a policy

### Webhooks

- `get_outgoing_webhook` - Retrieve a webhook
- `list_outgoing_webhooks` - List all webhooks
- `create_outgoing_webhook` - Create a new webhook
- `update_outgoing_webhook` - Update a webhook
- `delete_outgoing_webhook` - Delete a webhook

### Events to Metrics (E2M)

- `get_e2m` - Retrieve E2M configuration
- `list_e2m` - List all E2M configurations
- `create_e2m` - Create new E2M configuration
- `replace_e2m` - Replace E2M configuration
- `delete_e2m` - Delete E2M configuration

### Data Access Rules

- `list_data_access_rules` - List all data access rules
- `get_data_access_rule` - Retrieve a data access rule
- `create_data_access_rule` - Create a new data access rule
- `update_data_access_rule` - Update a data access rule
- `delete_data_access_rule` - Delete a data access rule

### Enrichments

- `list_enrichments` - List all enrichments
- `get_enrichment` - Retrieve an enrichment
- `create_enrichment` - Create a new enrichment
- `update_enrichment` - Update an enrichment
- `delete_enrichment` - Delete an enrichment

### Views

- `list_views` - List all views
- `get_view` - Retrieve a view
- `create_view` - Create a new view
- `replace_view` - Replace a view
- `delete_view` - Delete a view

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOGS_SERVICE_URL` | Yes | - | IBM Cloud Logs service endpoint URL |
| `LOGS_API_KEY` | Yes | - | IBM Cloud API key for authentication |
| `LOGS_REGION` | No | - | IBM Cloud region |
| `LOGS_TIMEOUT` | No | `30s` | HTTP request timeout |
| `LOGS_MAX_RETRIES` | No | `3` | Maximum number of retry attempts |
| `LOGS_RETRY_WAIT_MIN` | No | `1s` | Minimum wait time between retries |
| `LOGS_RETRY_WAIT_MAX` | No | `30s` | Maximum wait time between retries |
| `LOGS_RATE_LIMIT` | No | `100` | Requests per second limit |
| `LOGS_RATE_LIMIT_BURST` | No | `20` | Burst size for rate limiting |
| `LOGS_ENABLE_RATE_LIMIT` | No | `true` | Enable/disable rate limiting |
| `LOGS_TLS_VERIFY` | No | `true` | Verify TLS certificates |
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | No | `json` | Log format (json, console) |
| `ENVIRONMENT` | No | - | Environment (production, development) |

### Configuration File

Alternatively, use a JSON configuration file:

```json
{
  "service_url": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
  "region": "us-south",
  "timeout": "30s",
  "max_retries": 3,
  "retry_wait_min": "1s",
  "retry_wait_max": "30s",
  "rate_limit": 100,
  "rate_limit_burst": 20,
  "enable_rate_limit": true,
  "tls_verify": true,
  "log_level": "info",
  "log_format": "json"
}
```

Specify the config file path:

```bash
CONFIG_FILE=/path/to/config.json ./logs-mcp-server
```

**Note**: Environment variables take precedence over config file values.

## Usage Examples

### Example 1: Query Logs

```
User: Show me all error logs from the last hour
Claude: [Uses query_logs tool with appropriate time range and filter]
```

### Example 2: Create an Alert

```
User: Create an alert for when error rate exceeds 100 per minute
Claude: [Uses create_alert tool with the specified threshold]
```

### Example 3: List Active Policies

```
User: What retention policies do we have?
Claude: [Uses list_policies tool to retrieve all policies]
```

## Architecture

### Components

- **Main Server** ([main.go](main.go)): Entry point, initialization, graceful shutdown
- **MCP Server** ([internal/server/server.go](internal/server/server.go)): MCP protocol handler, tool registration
- **API Client** ([internal/client/client.go](internal/client/client.go)): HTTP client with retry logic and rate limiting
- **Configuration** ([internal/config/config.go](internal/config/config.go)): Configuration management
- **Tools** ([internal/tools/](internal/tools/)): Individual MCP tool implementations

### Request Flow

1. AI assistant sends MCP tool request
2. MCP server validates request and extracts parameters
3. API client constructs HTTP request with authentication
4. Rate limiter enforces request limits
5. HTTP client executes request with retry logic
6. Response is parsed and formatted
7. Result is returned to AI assistant

## Security

See [SECURITY.md](SECURITY.md) for comprehensive security guidelines and best practices.

### Key Security Features

- **API Key Security**: Never log or expose API keys
- **TLS Verification**: Enabled by default, validate all certificates
- **Rate Limiting**: Protect against abuse and excessive usage
- **Input Validation**: All tool parameters are validated
- **Error Handling**: Secure error messages, no sensitive data exposure
- **Least Privilege**: Request minimal required permissions

## Performance Tuning

### Connection Pooling

The server maintains a pool of persistent HTTP connections:

- `MaxIdleConns`: 10 (configurable)
- `IdleConnTimeout`: 90s (configurable)

### Rate Limiting

Rate limiting prevents API quota exhaustion:

```bash
LOGS_RATE_LIMIT=100          # 100 requests/second
LOGS_RATE_LIMIT_BURST=20     # Allow bursts up to 20
```

### Retry Strategy

Exponential backoff with jitter:

- Initial wait: 1s
- Maximum wait: 30s
- Max retries: 3
- Retryable errors: 429, 500, 502, 503, 504

## Monitoring and Observability

### Structured Logging

All operations are logged with structured fields:

```json
{
  "level": "info",
  "ts": "2025-11-14T22:00:00Z",
  "msg": "HTTP request completed",
  "method": "GET",
  "url": "https://api.../v1/alerts",
  "status": 200,
  "duration": "234ms",
  "response_size": 1024
}
```

### Log Levels

- **debug**: Detailed request/response information
- **info**: Normal operations, startup, shutdown
- **warn**: Retries, degraded performance
- **error**: Failed requests, configuration errors

### Metrics

Key metrics to monitor:

- Request latency (p50, p95, p99)
- Error rate by status code
- Retry rate
- Rate limit hits
- Active connections

## Troubleshooting

### Common Issues

**Issue**: `LOGS_SERVICE_URL is required`
**Solution**: Set the environment variable or add to config file

**Issue**: `401 Unauthorized`
**Solution**: Verify API key has correct permissions and is not expired

**Issue**: `429 Too Many Requests`
**Solution**: Reduce rate limit or request quota increase

**Issue**: `Connection timeout`
**Solution**: Increase `LOGS_TIMEOUT` or check network connectivity

### Debug Mode

Enable debug logging:

```bash
LOG_LEVEL=debug ./logs-mcp-server
```

## Development

### Project Structure

```
logs-mcp-server/
├── main.go                 # Entry point
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── internal/
│   ├── config/            # Configuration management
│   │   └── config.go
│   ├── server/            # MCP server implementation
│   │   └── server.go
│   ├── client/            # HTTP API client
│   │   └── client.go
│   └── tools/             # MCP tool implementations
│       ├── base.go
│       ├── alerts.go
│       ├── queries.go
│       ├── alert_definitions.go
│       └── all_tools.go
├── README.md              # This file
├── SECURITY.md           # Security documentation
└── .env.example          # Example environment file
```

### Running Tests

```bash
go test ./...
```

### Building for Production

```bash
# Build with optimizations
CGO_ENABLED=0 go build -ldflags="-s -w" -o logs-mcp-server .

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o logs-mcp-server-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o logs-mcp-server-darwin-arm64 .
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

Copyright (c) 2025. All rights reserved.

## Support

For issues, questions, or feature requests:

- GitHub Issues: https://github.com/observability-c/logs-mcp-server/issues
- IBM Cloud Logs Documentation: https://cloud.ibm.com/docs/cloud-logs

## Version History

- **0.1.0** (2025-11-14): Initial release
  - Complete IBM Cloud Logs API coverage
  - Production-ready features: retry logic, rate limiting, structured logging
  - Comprehensive documentation and examples
