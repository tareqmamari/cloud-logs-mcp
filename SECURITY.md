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
export LOGS_API_KEY="<your-api-key>"  # pragma: allowlist secret
export LOGS_SERVICE_URL="https://[your-instance-id].api.us-south.logs.cloud.ibm.com"

# ❌ BAD - Never hardcode in scripts
LOGS_API_KEY="<example-key>" ./logs-mcp-server  # pragma: allowlist secret
```

### Configuration Files

**Example Secure Config** (`config.json`):

```json
{
  "service_url": "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
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

## Static Application Security Testing (SAST)

This project uses multiple SAST tools to identify security vulnerabilities before they reach production.

### Automated SAST Scans

**GitHub Actions Integration:**
- SAST scans run automatically on push to main and pull requests
- Scans run weekly on Sunday at midnight UTC
- Results are uploaded to GitHub Security tab (Code Scanning)

**Tools Integrated:**
1. **Gosec** - Go security checker for common security issues
   - Detects SQL injection, command injection, weak crypto, etc.
   - Outputs: SARIF format for GitHub Security integration

2. **govulncheck** - Official Go vulnerability scanner
   - Scans for known vulnerabilities in dependencies
   - Checks if vulnerable code paths are actually reachable
   - Outputs: SARIF format for GitHub Security integration

3. **Trivy** - Comprehensive vulnerability scanner
   - Scans filesystem, dependencies, and configurations
   - Detects CRITICAL, HIGH, and MEDIUM severity issues
   - Outputs: SARIF format for GitHub Security integration

4. **Semgrep** - Pattern-based static analysis
   - Detects security anti-patterns and best practice violations
   - Configurable rules for Go-specific security issues
   - Outputs: SARIF format for GitHub Security integration

### Running SAST Locally

**Install SAST tools:**
```bash
# Gosec and govulncheck (auto-installed by make targets)
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Trivy
brew install trivy              # macOS
# Linux: https://aquasecurity.github.io/trivy/latest/getting-started/installation/

# Semgrep
brew install semgrep            # macOS
pip install semgrep             # Linux/Windows
```

**Run SAST scans:**
```bash
# Run all SAST scans
make sast

# Run individual scanners
make sast-gosec         # Gosec only
make sast-govulncheck   # govulncheck only
make sast-trivy         # Trivy only
make sast-semgrep       # Semgrep only

# Clean reports
make sast-clean
```

**Reports Location:**
- Local reports: `.sast-reports/` directory (ignored by git)
- GitHub Security: Code scanning alerts in repository Security tab

### Testing GitHub Actions Locally

Use `act` to run GitHub Actions workflows locally before pushing:

```bash
# Install act
brew install act                # macOS
# Linux/Windows: https://github.com/nektos/act#installation

# List all workflows
make act-list

# Run specific workflows
make act-ci        # Run CI workflow locally
make act-sast      # Run SAST workflow locally
make act-lint      # Run commitlint workflow locally
```

### SAST Configuration

**Workflow File:** [`.github/workflows/sast.yaml`](.github/workflows/sast.yaml)

**act Configuration:** [`.actrc`](.actrc) - Configuration for running actions locally

### Interpreting Results

**Severity Levels:**
- **CRITICAL**: Immediate action required (e.g., RCE, SQL injection)
- **HIGH**: Important vulnerabilities (e.g., XSS, path traversal)
- **MEDIUM**: Security improvements (e.g., weak crypto, information disclosure)
- **LOW**: Best practice violations

**Where to View Results:**
1. **GitHub Security Tab**: Navigate to repository → Security → Code scanning alerts
2. **Pull Request Checks**: SAST results appear as PR check status
3. **Local Reports**: JSON and SARIF files in `.sast-reports/`

### SAST Best Practices

1. **Fix Issues Early**: Address CRITICAL and HIGH severity issues before merging
2. **Review Dependencies**: Run `make sast-govulncheck` after dependency updates
3. **False Positives**: Document false positives with inline comments
4. **Continuous Scanning**: SAST runs automatically on every push and weekly
5. **Local Testing**: Run `make sast` before committing significant changes

### Common Security Issues Detected

**Gosec:**
- G101: Hardcoded credentials
- G104: Unhandled errors
- G204: Command injection
- G401: Weak crypto (MD5, SHA1)

**govulncheck:**
- Known CVEs in dependencies
- Vulnerable code paths in use

**Trivy:**
- Dependency vulnerabilities
- Misconfigurations
- License issues

**Semgrep:**
- OWASP Top 10 violations
- Security anti-patterns
- Best practice violations

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
