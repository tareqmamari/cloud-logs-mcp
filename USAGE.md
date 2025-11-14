# Usage Guide

This guide shows you how to use the IBM Cloud Logs MCP Server with Claude Desktop.

## Configuration (Choose One Method)

### Method 1: Claude Desktop Config (Recommended) ⭐

This is the **standard way** MCP servers are configured - no separate config files needed!

1. **Build the server**:
   ```bash
   cd /Users/tareq/workspace/logs-mcp
   make build
   ```

2. **Edit Claude Desktop config**:

   **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   ```bash
   nano ~/Library/Application\ Support/Claude/claude_desktop_config.json
   ```

   **Linux**: `~/.config/Claude/claude_desktop_config.json`
   ```bash
   nano ~/.config/Claude/claude_desktop_config.json
   ```

   **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
   ```bash
   notepad %APPDATA%\Claude\claude_desktop_config.json
   ```

3. **Add this configuration**:
   ```json
   {
     "mcpServers": {
       "ibm-cloud-logs": {
         "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
         "env": {
           "LOGS_SERVICE_URL": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
           "LOGS_API_KEY": "your-api-key-here",
           "LOGS_REGION": "us-south"
         }
       }
     }
   }
   ```

   **Important**:
   - Use **absolute path** to the binary
   - Replace `your-instance-id` with your actual IBM Cloud Logs instance ID
   - Replace `your-api-key-here` with your IBM Cloud API key

4. **Restart Claude Desktop** (quit completely and reopen)

That's it! No `.env` files, no copying configs.

### Method 2: Environment Variables (For Testing)

If you want to test the server standalone:

```bash
export LOGS_SERVICE_URL="https://your-instance-id.api.us-south.logs.cloud.ibm.com"
export LOGS_API_KEY="your-api-key"
export LOGS_REGION="us-south"

./bin/logs-mcp-server
```

### Method 3: .env File (For Development)

Only use this for local development:

```bash
cp .env.example .env
# Edit .env with your credentials
./bin/logs-mcp-server
```

## Finding Your Credentials

### Instance ID

1. Go to [IBM Cloud Console](https://cloud.ibm.com)
2. Navigate to **Resource List**
3. Find your **Cloud Logs** service
4. Click the instance
5. Copy the **instance ID** from the details section

Your endpoint format: `https://{instance-id}.api.{region}.logs.cloud.ibm.com`

### API Key

1. Go to [IBM Cloud IAM API Keys](https://cloud.ibm.com/iam/apikeys)
2. Click **Create**
3. Give it a name (e.g., "MCP Server Access")
4. Copy the API key immediately (you can't see it again!)
5. Store it securely

### Region

Common regions:
- `us-south` - Dallas
- `us-east` - Washington DC
- `eu-de` - Frankfurt
- `eu-gb` - London
- `au-syd` - Sydney
- `jp-tok` - Tokyo

## Verify It's Working

### Check Claude Desktop

After restarting Claude Desktop, you should be able to ask:

```
"List my IBM Cloud Logs alerts"
```

Claude will use the `list_alerts` tool and return your alerts.

### Check Logs (if having issues)

**macOS/Linux**:
```bash
tail -f ~/Library/Logs/Claude/mcp*.log
```

Look for:
- "Starting IBM Cloud Logs MCP Server"
- "IBM Cloud IAM authenticator initialized successfully"
- "Registered all MCP tools"

## Example Commands

Once configured, you can ask Claude:

### Queries
```
"Search my logs for errors in the last hour"
"Run a background query for all warnings from yesterday"
"What's the status of background query abc-123?"
"Cancel background query xyz-789"
```

### Alerts
```
"Show me all my alerts"
"Create an alert when error rate exceeds 100 per minute"
"Update alert abc-123 to change the threshold to 200"
"Delete alert xyz-456"
"List all alert definitions"
```

### Policies
```
"What retention policies do I have?"
"Create a retention policy to keep audit logs for 90 days"
"Show me all data access rules"
```

### Webhooks
```
"List all my webhooks"
"Create a webhook to send alerts to https://hooks.slack.com/..."
"Update webhook abc-123 to change the URL"
```

### Views & Enrichments
```
"Show all saved views"
"List my log enrichment rules"
"Create a new view for error analysis"
```

### Events to Metrics (E2M)
```
"List all events-to-metrics configurations"
"Create an E2M to track error counts"
```

## Configuration Reference

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `LOGS_SERVICE_URL` | Your IBM Cloud Logs endpoint | `https://abc123.api.us-south.logs.cloud.ibm.com` |
| `LOGS_API_KEY` | IBM Cloud API key | `your-api-key` |
| `LOGS_REGION` | IBM Cloud region (optional but recommended) | `us-south` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGS_TIMEOUT` | `30s` | HTTP request timeout |
| `LOGS_MAX_RETRIES` | `3` | Max retry attempts |
| `LOGS_RATE_LIMIT` | `100` | Requests per second |
| `LOGS_RATE_LIMIT_BURST` | `20` | Burst capacity |
| `LOGS_TLS_VERIFY` | `true` | Verify TLS certificates |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, console) |
| `ENVIRONMENT` | - | Environment name (production, development) |

### Example with All Options

```json
{
  "mcpServers": {
    "ibm-cloud-logs": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://abc123.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_TIMEOUT": "60s",
        "LOGS_MAX_RETRIES": "5",
        "LOGS_RATE_LIMIT": "50",
        "LOGS_RATE_LIMIT_BURST": "10",
        "LOG_LEVEL": "debug",
        "LOG_FORMAT": "json"
      }
    }
  }
}
```

## Troubleshooting

### "LOGS_SERVICE_URL is required"

**Solution**: Make sure you've set all three required variables in Claude Desktop config.

### "401 Unauthorized"

**Solutions**:
- Verify your API key is correct
- Check the API key has appropriate IAM permissions
- Ensure the API key hasn't expired
- Try creating a new API key

### "Connection timeout"

**Solutions**:
- Check your internet connection
- Verify the `LOGS_SERVICE_URL` is correct
- Increase timeout: `"LOGS_TIMEOUT": "60s"`
- Check if you need to use private endpoint

### Claude doesn't show the tools

**Solutions**:
1. Verify binary path is **absolute** (not relative)
2. Check execute permissions:
   ```bash
   chmod +x /Users/tareq/workspace/logs-mcp/bin/logs-mcp-server
   ```
3. **Completely quit** Claude Desktop (not just close window)
4. Restart Claude Desktop
5. Check Claude logs for errors

### Can't find instance ID

1. Go to https://cloud.ibm.com
2. Click **Resource List** (hamburger menu)
3. Expand **Logging and Monitoring**
4. Click your Cloud Logs instance
5. The instance ID is in the **Details** section

## Advanced Usage

### Development Mode

For development with debug logs:

```bash
export LOG_LEVEL=debug
export LOG_FORMAT=console
export ENVIRONMENT=development
./bin/logs-mcp-server
```

Or add to Claude config:
```json
"env": {
  ...
  "LOG_LEVEL": "debug",
  "LOG_FORMAT": "console"
}
```

### Using Private Endpoints

For enhanced security, use IBM Cloud private endpoints:

```json
"env": {
  "LOGS_SERVICE_URL": "https://your-instance-id.api.private.us-south.logs.cloud.ibm.com",
  ...
}
```

### Rate Limiting

Adjust rate limits based on your needs:

```json
"env": {
  ...
  "LOGS_RATE_LIMIT": "50",        // 50 requests/second
  "LOGS_RATE_LIMIT_BURST": "10"   // Allow 10-request bursts
}
```

### Increasing Timeout

For slow networks or large queries:

```json
"env": {
  ...
  "LOGS_TIMEOUT": "120s"  // 2 minutes
}
```

## Next Steps

- See [README.md](README.md) for complete documentation
- Check [SECURITY.md](SECURITY.md) for production deployment
- Review [QUICKSTART.md](QUICKSTART.md) for quick setup

---

**Questions?** Open an issue on GitHub or check the documentation.
