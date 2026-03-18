# IBM Cloud Logs Authentication Guide

## Overview

IBM Cloud Logs uses IBM Cloud IAM (Identity and Access Management) for
authentication. All API requests require a valid bearer token obtained by
exchanging an IBM Cloud API key with the IAM token service.

## IAM Token Exchange Flow

### Step 1: Obtain an IBM Cloud API Key

Create an API key in the IBM Cloud console or CLI:

```bash
ibmcloud iam api-key-create logs-mcp-key --description "MCP Server for Cloud Logs"
```

Store the key securely. Set it as the `LOGS_API_KEY` environment variable
for the MCP server.

### Step 2: Exchange API Key for Bearer Token

The MCP server uses the IBM Go SDK `core.IamAuthenticator` which handles
the token exchange automatically. Under the hood, it sends:

```
POST https://iam.cloud.ibm.com/identity/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey={API_KEY}
```

The response contains:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "expiration": 1700000000
}
```

### Step 3: Use Bearer Token

Every request to IBM Cloud Logs includes the token:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

The SDK caches the token and refreshes it automatically before expiration.

## IAM Endpoints

| Environment | IAM URL |
|-------------|---------|
| Production | `https://iam.cloud.ibm.com` (default) |
| Staging | `https://iam.test.cloud.ibm.com` |

Set a custom IAM URL via the `IAM_URL` environment variable for non-production
environments.

## Service Endpoint Construction

### Management API

Used for all CRUD operations, queries, and configuration:

```
https://{instance-id}.api.{region}.logs.cloud.ibm.com
```

Examples:
- `https://abc123.api.us-south.logs.cloud.ibm.com`
- `https://def456.api.eu-de.logs.cloud.ibm.com`

### Ingress (Log Ingestion)

Used exclusively for sending logs via the `ingest_logs` tool:

```
https://{instance-id}.ingress.{region}.logs.cloud.ibm.com
```

The client converts `.api.` to `.ingress.` automatically when the request
has `UseIngressHost` set. The ingestion path is `/logs/v1/singles`.

## JWT Token Claims

The IAM bearer token is a JWT containing these claims:

| Claim | Field | Description |
|-------|-------|-------------|
| Subject | `sub` | User or Service ID (e.g., `iam-ServiceId-...`) |
| IAM ID | `iam_id` | IAM identifier |
| Account | `account` | IBM Cloud account ID |
| Realm | `realmid` | Realm identifier |
| Name | `name` | Human-readable name |
| Email | `email` | User email (if applicable) |
| Issued At | `iat` | Token issue timestamp |
| Expires At | `exp` | Token expiration timestamp |

## API Key Management Best Practices

1. **Use service IDs** instead of personal API keys in production.
2. **Rotate API keys** every 90 days.
3. **Scope permissions** to the minimum required IAM roles.
4. **Never commit API keys** to source control. Use environment variables
   or secret managers.
5. **Monitor API key usage** via IBM Cloud audit logs.

## Configuration Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LOGS_API_KEY` | Yes | IBM Cloud API key for authentication |
| `LOGS_SERVICE_URL` | Yes | Management API endpoint URL |
| `LOGS_REGION` | No | IBM Cloud region (e.g., `us-south`, `eu-de`) |
| `LOGS_INSTANCE_NAME` | No | Human-readable instance name |
| `IAM_URL` | No | Custom IAM endpoint (for staging) |

## Troubleshooting

### 401 Unauthorized

- API key is invalid, revoked, or the token has expired.
- Verify `LOGS_API_KEY` is set and valid.
- Check IAM service status at https://cloud.ibm.com/status.

### 403 Forbidden

- The API key's associated service ID or user lacks the required IAM role.
- Verify the resource group and service access policies in IBM Cloud IAM.

### Token Refresh Failures

- Network connectivity issues to IAM endpoint.
- IAM service outage (check https://cloud.ibm.com/status).
- The SDK retries token refresh automatically with backoff.
