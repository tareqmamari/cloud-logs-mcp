# Security Best Practices

## Overview

This document outlines security best practices for deploying and operating the IBM Cloud Logs MCP Server in production environments.

## Authentication & Authorization

### API Key Management

**Critical Security Requirements:**

1. **Never Commit API Keys**
   - NEVER commit API keys to version control
   - Use `.env` files (add to `.gitignore`)
   - Use environment variables in production
   - Use secret management systems (HashiCorp Vault, AWS Secrets Manager, etc.)

2. **API Key Rotation**
   - Rotate API keys every 90 days minimum
   - Use IBM Cloud IAM for centralized key management
   - Implement automated rotation where possible
   - Revoke old keys immediately after rotation

3. **Least Privilege Principle**
   - Request only the minimum required IBM Cloud IAM permissions
   - Use service-specific API keys, not personal keys
   - Recommended IAM roles:
     - `Viewer` - Read-only access for queries and listing
     - `Operator` - Read and limited write operations
     - `Editor` - Full management capabilities (use sparingly)

### Example IAM Policy

```json
{
  "roles": [
    {
      "role_id": "crn:v1:bluemix:public:iam::::role:Viewer",
      "description": "Read-only access to logs"
    },
    {
      "role_id": "crn:v1:bluemix:public:logs::::serviceRole:Reader",
      "description": "Query logs and view configurations"
    }
  ]
}
```

## Network Security

### TLS/SSL Configuration

1. **Always Use TLS**
   ```bash
   LOGS_TLS_VERIFY=true  # MUST be true in production
   ```

2. **Certificate Validation**
   - Never disable certificate verification in production
   - Keep system CA certificates up to date
   - Pin certificates for additional security (advanced)

3. **Service Endpoints**
   - Use private endpoints when available
   - Prefer region-specific endpoints over global
   - Example: `https://your-instance-id.api.private.us-south.logs.cloud.ibm.com`

### Network Isolation

1. **Firewall Rules**
   - Restrict outbound connections to IBM Cloud Logs service endpoints only
   - Whitelist specific IP ranges if required
   - Use IBM Cloud private networking (Service Endpoints) when possible

2. **VPC Integration**
   ```bash
   # Use IBM Cloud private endpoint
   LOGS_SERVICE_URL=https://your-instance-id.api.private.us-south.logs.cloud.ibm.com
   ```

## Data Protection

### Data in Transit

- All communication uses TLS 1.2+ encryption
- API requests include authentication headers
- Sensitive data is never logged or cached

### Data at Rest

- API keys stored only in environment variables or secure secret managers
- No persistent storage of credentials
- Configuration files must never contain API keys

### Data Handling

1. **Query Results**
   - Be aware that query results may contain sensitive log data
   - Implement appropriate data classification
   - Use data access rules to restrict sensitive data

2. **PII/PHI Handling**
   - Configure enrichments to mask PII
   - Use IBM Cloud Logs data access rules
   - Implement additional filtering if processing regulated data

## Configuration Security

### Environment Variables

**Secure Configuration:**

```bash
# ✅ GOOD - Use environment variables
export LOGS_API_KEY="your-secret-key"
export LOGS_SERVICE_URL="https://your-instance-id.api.us-south.logs.cloud.ibm.com"

# ❌ BAD - Never hardcode in scripts
LOGS_API_KEY="hardcoded-key" ./logs-mcp-server
```

### Configuration Files

**Example Secure Config** (`config.json`):

```json
{
  "service_url": "https://your-instance-id.api.us-south.logs.cloud.ibm.com",
  "region": "us-south",
  "tls_verify": true,
  "rate_limit": 100,
  "rate_limit_burst": 20
}
```

**Note**: Never include `api_key` in configuration files!

#### File Permissions

```bash
# Restrict permissions on config files
chmod 600 .env
chmod 600 config.json

# Verify no secrets in git
git secrets --scan
```

## Pre-Commit Hooks

### Setup

To prevent accidental commits of secrets and credentials, this project uses pre-commit hooks:

```bash
# Install pre-commit (if not already installed)
pip install pre-commit

# Install the git hooks
pre-commit install

# Run manually on all files
pre-commit run --all-files
```

### What Gets Checked

The pre-commit hooks automatically check for:

1. **Secrets Detection** (detect-secrets)
   - IBM Cloud API keys
   - AWS keys
   - GitHub tokens
   - Private keys
   - JWT tokens
   - High-entropy strings

2. **Code Quality** (Go-specific)
   - Go formatting (gofmt)
   - Go vet checks
   - Go mod tidy

3. **File Quality**
   - Trailing whitespace
   - End-of-file fixers
   - YAML syntax
   - Large files
   - Merge conflicts
   - Private keys

### Bypassing Hooks (Emergency Only)

```bash
# Only use in emergencies - requires justification
git commit --no-verify -m "Emergency fix"
```

**Warning**: Bypassing hooks may introduce security vulnerabilities. Always review changes carefully.

## Runtime Security

### Process Isolation

1. **Run as Non-Root User**
   ```dockerfile
   # Dockerfile example
   USER nobody:nogroup
   CMD ["./logs-mcp-server"]
   ```

2. **Container Security**
   - Use minimal base images (distroless, alpine)
   - Scan images for vulnerabilities
   - Run with read-only root filesystem
   - Drop unnecessary capabilities

### Resource Limits

```yaml
# Kubernetes example
resources:
  limits:
    cpu: "1000m"
    memory: "512Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
```

### Rate Limiting

Protect against abuse and quota exhaustion:

```bash
LOGS_RATE_LIMIT=100           # Conservative limit
LOGS_RATE_LIMIT_BURST=20      # Small burst allowance
LOGS_ENABLE_RATE_LIMIT=true   # Always enabled
```

## Monitoring & Auditing

### Security Logging

1. **Enable Comprehensive Logging**
   ```bash
   LOG_LEVEL=info              # Production default
   LOG_FORMAT=json             # Structured logs
   ENVIRONMENT=production
   ```

2. **What Gets Logged**
   - All API requests (method, path, status, duration)
   - Authentication failures
   - Rate limit violations
   - Configuration errors
   - Startup/shutdown events

3. **What NEVER Gets Logged**
   - API keys or credentials
   - Full request/response bodies (may contain sensitive data)
   - Personally identifiable information (PII)

### Audit Trail

Monitor these security-relevant events:

- Failed authentication attempts
- Permission denied errors (403)
- Unusual query patterns
- Rate limit hits (potential abuse)
- Configuration changes

### Alerting

Set up alerts for:

```
- High error rates (> 5%)
- Repeated authentication failures
- Excessive rate limiting
- Unusual access patterns
- Service degradation
```

## Dependency Management

### Supply Chain Security

1. **Verify Dependencies**
   ```bash
   # Check for known vulnerabilities
   go list -json -m all | nancy sleuth

   # Or use govulncheck
   govulncheck ./...
   ```

2. **Dependency Updates**
   - Review dependencies quarterly
   - Subscribe to security advisories
   - Test updates in staging before production

3. **Lock Dependencies**
   - Commit `go.sum` to version control
   - Use specific versions, not `latest`
   - Verify checksums

### Trusted Sources

- Use only official Go modules
- Verify module authenticity
- Review third-party dependencies carefully

## Incident Response

### Security Incident Plan

1. **Immediate Actions**
   - Rotate compromised API keys immediately
   - Review access logs for unauthorized activity
   - Isolate affected systems
   - Notify security team

2. **Investigation**
   - Collect logs and metrics
   - Identify scope of compromise
   - Document timeline of events

3. **Remediation**
   - Apply security patches
   - Update compromised credentials
   - Implement additional controls
   - Review and update security policies

### API Key Compromise

If an API key is compromised:

1. **Immediately**:
   - Revoke the compromised key in IBM Cloud IAM
   - Generate a new API key
   - Update all systems using the old key

2. **Within 24 Hours**:
   - Review audit logs for unauthorized access
   - Assess potential data exposure
   - Document incident

3. **Follow-up**:
   - Implement additional monitoring
   - Review key management procedures
   - Update security training

## Deployment Security

### Production Checklist

- [ ] API keys stored in secret manager, not environment files
- [ ] `LOGS_TLS_VERIFY=true` enforced
- [ ] Running as non-root user
- [ ] Resource limits configured
- [ ] Rate limiting enabled
- [ ] Structured logging enabled (JSON format)
- [ ] Log aggregation configured
- [ ] Security monitoring enabled
- [ ] Dependencies scanned for vulnerabilities
- [ ] Access controls properly configured
- [ ] Incident response plan documented
- [ ] Backup credentials available
- [ ] Certificate validation enabled
- [ ] Network policies restricting traffic

### Kubernetes Deployment Example

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: logs-mcp-credentials
type: Opaque
stringData:
  api-key: "your-api-key"  # Use sealed secrets or external secrets operator
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logs-mcp-server
spec:
  replicas: 2
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: mcp-server
        image: logs-mcp-server:latest
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        env:
        - name: LOGS_API_KEY
          valueFrom:
            secretKeyRef:
              name: logs-mcp-credentials
              key: api-key
        - name: LOGS_SERVICE_URL
          value: "https://your-instance-id.api.private.us-south.logs.cloud.ibm.com"
        - name: LOGS_TLS_VERIFY
          value: "true"
        - name: LOG_LEVEL
          value: "info"
        - name: ENVIRONMENT
          value: "production"
        resources:
          limits:
            cpu: "1000m"
            memory: "512Mi"
          requests:
            cpu: "100m"
            memory: "128Mi"
        livenessProbe:
          exec:
            command: ["/bin/sh", "-c", "pgrep logs-mcp-server"]
          initialDelaySeconds: 10
          periodSeconds: 30
```

## Compliance Considerations

### GDPR Compliance

- Implement data retention policies via IBM Cloud Logs
- Use data access rules to restrict PII access
- Enable audit logging for all data access
- Implement right to erasure procedures

### HIPAA Compliance

- Use IBM Cloud BAA (Business Associate Agreement)
- Enable encryption in transit and at rest
- Implement comprehensive audit logging
- Restrict access via IAM policies
- Use private endpoints only

### SOC 2 Compliance

- Implement comprehensive logging and monitoring
- Enable access controls and authentication
- Document security procedures
- Conduct regular security reviews
- Maintain audit trails

## Security Contacts

- **IBM Cloud Security**: https://cloud.ibm.com/security
- **Vulnerability Reporting**: security@ibm.com
- **IBM Cloud Support**: https://cloud.ibm.com/unifiedsupport

## Additional Resources

- [IBM Cloud Security Best Practices](https://cloud.ibm.com/docs/security)
- [IBM Cloud IAM Documentation](https://cloud.ibm.com/docs/account?topic=account-iamoverview)
- [IBM Cloud Logs Security](https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-mng-data)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)

## Version History

- **0.1.0** (2025-11-14): Initial security documentation
