# Multi-Instance Configuration Guide

This guide shows you how to configure multiple IBM Cloud Logs instances with the MCP server.

## Why Multiple Instances?

You might need multiple instances for:
- **Multiple environments**: Production, Staging, Development
- **Multiple regions**: US, EU, Asia Pacific
- **Multiple teams**: Different business units or projects
- **Compliance**: Separate instances for different data classifications

## Configuration Examples

### Example 1: Multiple Environments (Same Region)

```json
{
  "mcpServers": {
    "logs-production": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-abc123.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-prod-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Production US"
      }
    },
    "logs-staging": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://staging-def456.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-staging-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Staging US"
      }
    },
    "logs-development": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://dev-ghi789.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-dev-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Development US"
      }
    }
  }
}
```

**Usage:**
```
"Show me production alerts"
"Query staging logs for errors in the last hour"
"List all alerts in development"
"Compare error rates between production and staging"
```

### Example 2: Multiple Regions

```json
{
  "mcpServers": {
    "logs-us-production": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-us.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-us-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Production US South"
      }
    },
    "logs-eu-production": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-eu.api.eu-de.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-eu-api-key",
        "LOGS_REGION": "eu-de",
        "LOGS_INSTANCE_NAME": "Production EU Frankfurt"
      }
    },
    "logs-ap-production": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://prod-ap.api.au-syd.logs.cloud.ibm.com",
        "LOGS_API_KEY": "your-ap-api-key",
        "LOGS_REGION": "au-syd",
        "LOGS_INSTANCE_NAME": "Production AP Sydney"
      }
    }
  }
}
```

**Usage:**
```
"Show alerts from EU production"
"Query US production logs for API errors"
"Compare metrics across all regions"
"What's the error rate in Asia Pacific?"
```

### Example 3: Mixed Environments and Regions

```json
{
  "mcpServers": {
    "logs-prod-us": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://p1.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "key-prod-us",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Production US"
      }
    },
    "logs-prod-eu": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://p2.api.eu-de.logs.cloud.ibm.com",
        "LOGS_API_KEY": "key-prod-eu",
        "LOGS_REGION": "eu-de",
        "LOGS_INSTANCE_NAME": "Production EU"
      }
    },
    "logs-stage-us": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://s1.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "key-stage-us",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Staging US"
      }
    },
    "logs-dev": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://d1.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "key-dev",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Development"
      }
    }
  }
}
```

### Example 4: Team-Based Configuration

```json
{
  "mcpServers": {
    "logs-backend-team": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://backend.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "backend-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Backend Services"
      }
    },
    "logs-frontend-team": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://frontend.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "frontend-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Frontend Applications"
      }
    },
    "logs-platform-team": {
      "command": "/Users/tareq/workspace/logs-mcp/bin/logs-mcp-server",
      "env": {
        "LOGS_SERVICE_URL": "https://platform.api.us-south.logs.cloud.ibm.com",
        "LOGS_API_KEY": "platform-api-key",
        "LOGS_REGION": "us-south",
        "LOGS_INSTANCE_NAME": "Platform Infrastructure"
      }
    }
  }
}
```

## Best Practices

### 1. Naming Convention

Use clear, descriptive names that indicate:
- **Environment**: prod, staging, dev
- **Region**: us, eu, ap
- **Purpose**: team name, service type, compliance zone

**Good examples:**
- `logs-prod-us-east`
- `logs-staging-eu-frankfurt`
- `logs-dev-team-backend`
- `logs-compliance-pci`

**Avoid:**
- `logs1`, `logs2` (not descriptive)
- `my-logs` (too vague)
- `test` (ambiguous)

### 2. Instance Names

Use the `LOGS_INSTANCE_NAME` variable to add a friendly name that appears in logs:

```json
"env": {
  "LOGS_SERVICE_URL": "https://abc123.api.us-south.logs.cloud.ibm.com",
  "LOGS_API_KEY": "...",
  "LOGS_REGION": "us-south",
  "LOGS_INSTANCE_NAME": "Production US South - E-Commerce"
}
```

This helps when debugging or reviewing logs to know which instance is being used.

### 3. API Key Management

**Security recommendations:**
- Use **separate API keys** for each instance
- Apply **principle of least privilege** (minimal permissions)
- **Rotate keys regularly** (every 90 days)
- Use **service IDs** instead of personal API keys
- Consider using **trusted profiles** for enhanced security

**Example IAM setup:**
```bash
# Create service IDs for each environment
ibmcloud iam service-id-create logs-prod-mcp "MCP access to production logs"
ibmcloud iam service-id-create logs-staging-mcp "MCP access to staging logs"

# Assign appropriate roles
ibmcloud iam service-policy-create logs-prod-mcp \
  --roles Viewer \
  --service-name logs
```

### 4. Environment Isolation

For strict separation:
- **Different API keys** per environment
- **Separate IAM policies** with specific permissions
- **Different log levels** (debug for dev, info for prod)

```json
{
  "mcpServers": {
    "logs-production": {
      "env": {
        "LOG_LEVEL": "info",
        "LOGS_RATE_LIMIT": "100"
      }
    },
    "logs-development": {
      "env": {
        "LOG_LEVEL": "debug",
        "LOGS_RATE_LIMIT": "200"
      }
    }
  }
}
```

### 5. Performance Tuning per Instance

Adjust settings based on instance usage:

```json
{
  "mcpServers": {
    "logs-high-volume": {
      "env": {
        "LOGS_RATE_LIMIT": "200",
        "LOGS_RATE_LIMIT_BURST": "50",
        "LOGS_TIMEOUT": "60s"
      }
    },
    "logs-low-volume": {
      "env": {
        "LOGS_RATE_LIMIT": "50",
        "LOGS_RATE_LIMIT_BURST": "10",
        "LOGS_TIMEOUT": "30s"
      }
    }
  }
}
```

## Working with Multiple Instances

### Querying Specific Instances

The AI can distinguish between instances based on context:

```
"Show me production alerts" → Uses logs-production
"What errors are in staging?" → Uses logs-staging
"List EU alerts" → Uses logs-eu-production
```

### Cross-Instance Analysis

You can ask questions that span multiple instances:

```
"Compare error rates between production and staging"
"Show me all critical alerts across all regions"
"What's the total query volume across all instances?"
```

### Targeting by Name

Reference instances explicitly:

```
"Show alerts from the Backend Services instance"
"Query the Production US logs for API errors"
"List policies in the EU Frankfurt instance"
```

## Troubleshooting Multi-Instance Setup

### Issue: Can't distinguish between instances

**Solution:** Use clear naming and add `LOGS_INSTANCE_NAME`:
```json
"LOGS_INSTANCE_NAME": "Production US South - API Services"
```

### Issue: Wrong instance being queried

**Solution:** Be more specific in your questions:
```
Instead of: "Show me alerts"
Use: "Show me alerts from production US"
```

### Issue: Too many MCP servers slowing down startup

**Solution:** Comment out unused instances:
```json
{
  "mcpServers": {
    "logs-production": { ... },
    // "logs-old-dev": { ... }  // Commented out
  }
}
```

### Issue: API key conflicts

**Solution:** Ensure each instance has its own API key:
- Create separate service IDs in IBM Cloud IAM
- Generate unique API key for each
- Use appropriate IAM policies per instance

## Environment Variables Reference

Available per-instance configuration:

| Variable | Description | Example |
|----------|-------------|---------|
| `LOGS_SERVICE_URL` | Instance endpoint (required) | `https://abc.api.us-south.logs.cloud.ibm.com` |
| `LOGS_API_KEY` | IBM Cloud API key (required) | `your-api-key` |
| `LOGS_REGION` | Region (recommended) | `us-south`, `eu-de`, `au-syd` |
| `LOGS_INSTANCE_NAME` | Friendly name (optional) | `Production US - API Gateway` |
| `LOGS_TIMEOUT` | Request timeout | `30s`, `60s` |
| `LOGS_MAX_RETRIES` | Max retries | `3`, `5` |
| `LOGS_RATE_LIMIT` | Requests/second | `50`, `100`, `200` |
| `LOG_LEVEL` | Logging level | `debug`, `info`, `warn` |

## Examples by Use Case

### Multi-Region Deployment

For global applications with logs in multiple regions:

```json
{
  "mcpServers": {
    "logs-americas": {
      "env": {
        "LOGS_SERVICE_URL": "https://na.api.us-south.logs.cloud.ibm.com",
        "LOGS_INSTANCE_NAME": "Americas Production"
      }
    },
    "logs-emea": {
      "env": {
        "LOGS_SERVICE_URL": "https://emea.api.eu-de.logs.cloud.ibm.com",
        "LOGS_INSTANCE_NAME": "EMEA Production"
      }
    },
    "logs-apac": {
      "env": {
        "LOGS_SERVICE_URL": "https://apac.api.au-syd.logs.cloud.ibm.com",
        "LOGS_INSTANCE_NAME": "APAC Production"
      }
    }
  }
}
```

### Compliance Separation

For environments with different compliance requirements:

```json
{
  "mcpServers": {
    "logs-pci-compliant": {
      "env": {
        "LOGS_SERVICE_URL": "https://pci.api.us-south.logs.cloud.ibm.com",
        "LOGS_INSTANCE_NAME": "PCI Compliant Workloads"
      }
    },
    "logs-general": {
      "env": {
        "LOGS_SERVICE_URL": "https://general.api.us-south.logs.cloud.ibm.com",
        "LOGS_INSTANCE_NAME": "General Workloads"
      }
    }
  }
}
```

---

**Questions?** See the main [README.md](README.md) or [USAGE.md](USAGE.md) for more information.
