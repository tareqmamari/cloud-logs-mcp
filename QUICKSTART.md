# Quick Start Guide

Get up and running with IBM Cloud Logs MCP Server in 5 minutes.

## Prerequisites

- Go 1.22+ installed ([download](https://go.dev/dl/))
- IBM Cloud account with Logs service
- API key with appropriate permissions

## Installation Steps

### 1. Get Your IBM Cloud Credentials

1. Log in to [IBM Cloud](https://cloud.ibm.com)
2. Navigate to **Manage > Access (IAM) > API keys**
3. Click **Create** to create a new API key
4. Copy and save the API key securely
5. Find your Logs service endpoint URL:
   - Go to **Resource List** → **Cloud Logs**
   - Click your instance
   - Note the instance ID from the details
   - Your endpoint: `https://{instance-id}.api.{region}.logs.cloud.ibm.com`

### 2. Build the Server

```bash
# Clone the repository
git clone https://github.com/observability-c/logs-mcp-server.git
cd logs-mcp-server

# Install dependencies and build
make deps
make build
```

The binary will be at `./bin/logs-mcp-server`

### 3. Configure Claude Desktop

**No `.env` files needed!** Configure directly in Claude Desktop.

Find your Claude config file:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

Edit it and add:

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/full/path/to/logs-mcp-server/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-actual-api-key-here",
        "LOGS_REGION": "us-south"
      }
    }
  }
}
```

**Important**:
- Use the **absolute path** to the binary (e.g., `/Users/yourusername/...` on Mac)
- Replace `your-instance-id` with your actual instance ID
- Replace `your-actual-api-key-here` with your IBM Cloud API key

### 4. Restart Claude Desktop

**Completely quit and restart** Claude Desktop for the changes to take effect.

That's it! No `.env` files needed.

## First Commands

Once connected, you can ask Claude:

```
"Show me all alerts in IBM Cloud Logs"
"Query logs from the last hour for errors"
"Create an alert when error rate exceeds 100 per minute"
"List all retention policies"
"Show me the status of background queries"
```

## Verify Installation

### Test the Build

```bash
make test
```

### Check Configuration

```bash
# View redacted config
./bin/logs-mcp-server --help
```

### Run Health Checks

The server includes built-in health checks that verify:
- IBM Cloud authentication
- API connectivity
- Service availability

## Common Issues

### "LOGS_API_KEY is required"

**Solution**: Ensure you've set `LOGS_API_KEY` in your `.env` file or environment.

### "401 Unauthorized"

**Solution**:
- Verify your API key is correct
- Check that the API key has appropriate IAM permissions
- Ensure the key hasn't expired

### "Connection timeout"

**Solution**:
- Check your internet connection
- Verify the `LOGS_SERVICE_URL` is correct
- Try increasing the timeout: `LOGS_TIMEOUT=60s`

### Claude Desktop doesn't show the tools

**Solution**:
- Verify the path to the binary is absolute (not relative)
- Check Claude Desktop logs for errors
- Ensure the binary has execute permissions: `chmod +x bin/logs-mcp-server`
- Restart Claude Desktop completely

## Development Mode

For development with debug logging:

```bash
make dev
```

This runs with:
- `LOG_LEVEL=debug` - Detailed logs
- `LOG_FORMAT=console` - Human-readable logs
- `ENVIRONMENT=development` - Development config

## Production Deployment

### Docker

```bash
# Build Docker image
make docker-build

# Run with Docker
docker run --env-file .env logs-mcp-server:latest
```

### Binary Deployment

```bash
# Build optimized binary
make build

# Install system-wide
sudo make install

# Run as service (systemd example)
sudo cp logs-mcp-server.service /etc/systemd/system/
sudo systemctl enable logs-mcp-server
sudo systemctl start logs-mcp-server
```

## Next Steps

- Read the full [README.md](README.md) for comprehensive documentation
- Review [SECURITY.md](SECURITY.md) for production security best practices
- Check [CONTRIBUTING.md](CONTRIBUTING.md) if you want to contribute
- Explore available MCP tools in the README

## Getting Help

- **Documentation**: See [README.md](README.md)
- **Issues**: https://github.com/observability-c/logs-mcp-server/issues
- **IBM Cloud Support**: https://cloud.ibm.com/unifiedsupport
- **IBM Cloud Logs Docs**: https://cloud.ibm.com/docs/cloud-logs

## Useful Commands

```bash
# Build and run
make build && make run

# Run all checks
make check

# View coverage
make test-coverage

# Clean build artifacts
make clean

# Build for all platforms
make build-all

# View all available commands
make help
```

Enjoy using IBM Cloud Logs with AI assistance! 🚀
