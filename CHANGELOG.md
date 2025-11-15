# Changelog

All notable changes to the IBM Cloud Logs MCP Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-11-14

### Added

#### Core Features
- **45+ MCP Tools** covering the complete IBM Cloud Logs API:
  - Alert Management (get, list, create, update, delete)
  - Alert Definitions (get, list, create, update, delete)
  - Query Execution (query logs, background queries, query status)
  - Rule Groups (get, list, create, update, delete)
  - Outgoing Webhooks (get, list, create, update, delete)
  - Policies (get, list, create, update, delete)
  - Events-to-Metrics (E2M) (get, list, create, replace, delete)
  - Data Access Rules (get, list, create, delete)
  - Enrichments (get, list, add)
  - Views (get, list, create, update, delete, folders)
  - Event Stream Targets (get, list, create, update, delete)

#### Authentication & Security
- IBM Cloud IAM authentication via official SDK
- Automatic bearer token generation and refresh
- Configurable IAM endpoints (production/staging support via `LOGS_IAM_URL`)
- TLS certificate verification (configurable)
- API key sanitization in logs and configuration output
- Service ID support for production deployments

#### Best Practices Implementation
- **Request ID Tracking**: Idempotency headers (`X-Request-ID`, `Idempotency-Key`)
- **Pagination Support**: Cursor-based pagination with configurable limits (default: 50, max: 100)
- **Structured Error Handling**: 12+ error types with categories (CLIENT_ERROR, SERVER_ERROR, EXTERNAL_ERROR)
- **Error Recovery Suggestions**: Actionable suggestions included in error responses
- **Workflow Prompts**: 5 pre-built workflow prompts:
  - `investigate_errors` - Debug error spikes systematically
  - `setup_monitoring` - Configure comprehensive monitoring
  - `compare_environments` - Compare production vs staging logs
  - `debugging_workflow` - Structured debugging guide
  - `optimize_retention` - Log retention cost optimization

#### Configuration & Deployment
- Environment variable-based configuration (12-factor app)
- Multi-instance support via `LOGS_INSTANCE_NAME`
- Docker support with multi-stage builds
- Cross-platform builds (Linux, macOS, Windows - AMD64, ARM64)
- Non-root container execution (UID 65534)
- Configuration via MCP client (recommended) or .env files

#### Performance & Reliability
- Connection pooling for HTTP client
- Configurable rate limiting (default: 100 req/s with burst=20)
- Exponential backoff retry logic (default: 3 retries)
- Request/response timeout configuration
- Health checks for authentication and API connectivity
- Metrics tracking (requests, latency, errors, tool usage)

#### Developer Experience
- Comprehensive Makefile with 20+ targets
- Unit tests for auth, config, tools, errors
- 100+ test cases with good coverage
- Structured logging with Uber Zap (JSON/console formats)
- API update workflow and helper scripts
- `make compare-api`, `make backup-api`, `make list-operations`

#### Documentation
- Single comprehensive README (consolidated all docs)
- Quick start guide
- Multi-instance configuration guide
- Security best practices (SECURITY.md)
- Contributing guidelines (CONTRIBUTING.md)
- API update workflow documentation
- Example configurations and usage examples
- Troubleshooting section

### Technical Details

**Dependencies:**
- Go 1.22+
- github.com/IBM/go-sdk-core/v5 v5.17.4
- github.com/mark3labs/mcp-go v0.8.0
- go.uber.org/zap v1.27.0
- golang.org/x/time v0.5.0
- github.com/joho/godotenv v1.5.1

**Supported Platforms:**
- Linux (AMD64, ARM64)
- macOS (AMD64, ARM64)
- Windows (AMD64)

**Binary Size:** ~8.8 MB (optimized with -ldflags "-s -w")

### Security

- All API keys stored in environment variables only
- Secrets never logged or exposed in responses
- TLS verification enabled by default
- IBM Cloud IAM authentication (OAuth 2.0 bearer tokens)
- Configurable allowed IP ranges
- Dockerfile runs as non-root user

### Known Limitations

- Stdio transport only (HTTP transport not yet implemented)
- MCP logging protocol not integrated (uses stderr logging)
- Progress notifications not implemented for long-running operations
- Circuit breaker pattern not implemented
- No built-in metrics export (Prometheus, etc.)

### Compatibility

- IBM Cloud Logs API version: 0.1.0
- MCP Protocol: Compatible with Claude Desktop and other MCP clients
- Tested with: Claude Desktop, MCP Inspector

---

## Release Notes Format (for future releases)

### [Unreleased]
- Features in development

### [X.Y.Z] - YYYY-MM-DD
#### Added
- New features

#### Changed
- Changes to existing functionality

#### Deprecated
- Soon-to-be removed features

#### Removed
- Removed features

#### Fixed
- Bug fixes

#### Security
- Security improvements

[0.1.0]: https://github.com/observability-c/logs-mcp-server/releases/tag/v0.1.0
