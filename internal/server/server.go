// Package server provides the MCP server implementation for the logs service.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/audit"
	"github.com/tareqmamari/cloud-logs-mcp/internal/auth"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/health"
	"github.com/tareqmamari/cloud-logs-mcp/internal/metrics"
	"github.com/tareqmamari/cloud-logs-mcp/internal/prompts"
	"github.com/tareqmamari/cloud-logs-mcp/internal/resources"
	"github.com/tareqmamari/cloud-logs-mcp/internal/security"
	"github.com/tareqmamari/cloud-logs-mcp/internal/tools"
)

// Authenticator defines the authentication interface needed by the server.
// *auth.Authenticator satisfies this interface.
type Authenticator interface {
	health.TokenValidator
	GetUserIdentity() (string, error)
}

// defaultStartupTimeout bounds startup I/O (user-identity lookup, TCO
// policy fetch) when cfg.Timeout is not positive. Config.Validate requires
// Timeout > 0 in production; this fallback only matters for callers that
// construct a Config without validating it (e.g. tests).
const defaultStartupTimeout = 30 * time.Second

// defaultShutdownTimeout mirrors config.Load()'s ShutdownTimeout default,
// used as a fallback when cfg.ShutdownTimeout is not positive.
const defaultShutdownTimeout = 30 * time.Second

// startupTimeout returns the bound to apply to startup I/O calls.
func startupTimeout(cfg *config.Config) time.Duration {
	if cfg.Timeout > 0 {
		return cfg.Timeout
	}
	return defaultStartupTimeout
}

// shutdownTimeout returns the bound to apply when shutting down the health
// server.
func shutdownTimeout(cfg *config.Config) time.Duration {
	if cfg.ShutdownTimeout > 0 {
		return cfg.ShutdownTimeout
	}
	return defaultShutdownTimeout
}

// boundedGetUserIdentity calls authenticator.GetUserIdentity() but does not
// wait past ctx's deadline. GetUserIdentity has no context parameter of its
// own (it's a small, stable interface implemented by *auth.Authenticator),
// so boundedness is enforced here by racing the call against ctx.Done() in
// a separate goroutine. If ctx expires first, this returns ctx.Err() and the
// goroutine is left to finish in the background. That leaked goroutine is
// now itself bounded rather than potentially-forever: auth.New gives the
// underlying IAM authenticator's HTTP client an explicit timeout
// (defaultIAMClientTimeout), so the token round trip - and therefore this
// goroutine - always returns within that timeout even if IAM hangs.
func boundedGetUserIdentity(ctx context.Context, authenticator Authenticator) (string, error) {
	type result struct {
		userID string
		err    error
	}

	resultCh := make(chan result, 1)
	go func() {
		userID, err := authenticator.GetUserIdentity()
		resultCh <- result{userID: userID, err: err}
	}()

	select {
	case r := <-resultCh:
		return r.userID, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Server represents the MCP server
type Server struct {
	mcpServer     *mcp.Server
	apiClient     client.Doer
	config        *config.Config
	logger        *zap.Logger
	metrics       *metrics.Metrics
	version       string
	healthServer  *health.Server
	authenticator Authenticator
	auditLogger   *audit.Logger
}

// New creates a new MCP server instance using real IBM Cloud credentials.
// ctx bounds startup I/O (user-identity lookup, TCO policy fetch) — it is
// not retained beyond NewWithDeps returning.
func New(ctx context.Context, cfg *config.Config, logger *zap.Logger, version string) (*Server, error) {
	// Create a single authenticator shared by the API client and health
	// checks, so there is only one IAM token cache per process.
	authenticator, err := auth.New(cfg.APIKey, cfg.IAMURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	// Create IBM Cloud Logs API client, injecting the shared authenticator.
	apiClient, err := client.NewWithAuthenticator(cfg, logger, version, authenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return NewWithDeps(ctx, cfg, apiClient, authenticator, logger, version)
}

// NewWithDeps creates a new MCP server instance with injectable dependencies.
// This constructor enables testing with mock clients and authenticators.
// ctx bounds startup I/O (user-identity lookup, TCO policy fetch) by
// cfg.Timeout; it is not retained beyond this call returning.
func NewWithDeps(ctx context.Context, cfg *config.Config, apiClient client.Doer, authenticator Authenticator, logger *zap.Logger, version string) (*Server, error) {
	// Create MCP server with tools, prompts, and resources capabilities
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "IBM Cloud Logs MCP Server",
		Version: version,
	}, &mcp.ServerOptions{
		HasTools:     true,
		HasPrompts:   true,
		HasResources: true,
	})

	metricsTracker := metrics.New(logger)

	// Initialize user-specific session using JWT subject from IAM token.
	// The subject uniquely identifies the user/service across sessions.
	// Bounded by cfg.Timeout so a hanging IAM call can't stall startup.
	identityCtx, cancelIdentity := context.WithTimeout(ctx, startupTimeout(cfg))
	userID, err := boundedGetUserIdentity(identityCtx, authenticator)
	cancelIdentity()
	if err != nil {
		// Fall back to API key hash if token retrieval fails
		logger.Warn("Could not get user identity from token, using API key hash",
			zap.String("error", security.SanitizeError(err)),
		)
		tools.SetCurrentUser(cfg.APIKey, cfg.InstanceID)
	} else {
		tools.SetCurrentUserFromJWT(userID, cfg.InstanceID)
		logger.Debug("Initialized user session from JWT",
			zap.String("user_id", userID),
			zap.String("instance_id", cfg.InstanceID),
		)
	}

	s := &Server{
		mcpServer:     mcpServer,
		apiClient:     apiClient,
		config:        cfg,
		logger:        logger,
		metrics:       metricsTracker,
		version:       version,
		authenticator: authenticator,
		auditLogger:   audit.NewLogger(logger, cfg.EnableAuditLog),
	}

	// Create health server if port is configured (port > 0)
	if cfg.HealthPort > 0 {
		healthChecker := health.New(apiClient, authenticator, logger)
		s.healthServer = health.NewServer(healthChecker, logger, cfg.HealthPort, cfg.HealthBindAddr, cfg.MetricsEndpoint)
	}

	// Fetch and cache TCO policies for tier selection. This helps tools
	// determine which tier (archive vs frequent_search) to query. Bounded
	// by cfg.Timeout so a hanging API call can't stall startup; failure
	// here remains non-fatal (tools fall back to defaults).
	tcoCtx, cancelTCO := context.WithTimeout(ctx, startupTimeout(cfg))
	err = tools.FetchAndCacheTCOConfig(tcoCtx, apiClient, logger)
	cancelTCO()
	if err != nil {
		logger.Warn("Failed to fetch TCO policies, will use defaults", zap.String("error", security.SanitizeError(err)))
	}

	// Register all tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register all prompts
	s.registerPrompts()

	// Register all resources
	s.registerResources()

	return s, nil
}

// registerTools registers all available MCP tools by iterating the single
// descriptor table in the tools package (the source of truth for which tools
// the server exposes). Adding or removing a tool is done there, not here.
func (s *Server) registerTools() error {
	descriptors := tools.Descriptors()
	for _, d := range descriptors {
		s.registerTool(d.New(s.apiClient, s.logger))
	}

	s.logger.Info("Registered all MCP tools", zap.Int("count", len(descriptors)))
	return nil
}

// registerTool is a helper to register a tool with proper error handling.
// It accepts any type that implements the tools.Tool interface.
func (s *Server) registerTool(t tools.Tool) {
	toolName := t.Name()

	// Register in dynamic registry for search_tools/describe_tools pattern
	tools.RegisterToolForDynamic(t)

	// Create tool definition with annotations
	mcpTool := &mcp.Tool{
		Name:        toolName,
		Description: t.Description(),
		InputSchema: t.InputSchema(),
		Annotations: t.Annotations(),
	}

	// Create handler that calls the tool's Execute method with metrics
	// tracking. Every exit path funnels through s.finishToolExecution — the
	// single chokepoint (shared by all 96 registered tools, since this
	// function body is written once and reused per registration) that
	// sanitizes credentials out of errors/results and records the audit
	// entry. Do not bypass it with a direct return of a tool's raw error or
	// result.
	handler := func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// Add client to context for tool execution
		// This enables per-request client injection for future HTTP transport
		ctx = tools.WithClient(ctx, s.apiClient)

		// Add session to context for tool execution
		// This enables per-request session injection for better testability
		ctx = tools.WithSession(ctx, tools.GetSession())
		ctx = tools.WithSessionProvider(ctx, tools.GetSessionManager())

		// Add the audit logger to context so GetAuditLogTool can read
		// recent entries without a constructor-level dependency change.
		ctx = tools.WithAuditLogger(ctx, s.auditLogger)

		var args map[string]interface{}
		if len(request.Params.Arguments) > 0 {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return s.finishToolExecution(ctx, toolName, args, start, nil, fmt.Errorf("failed to unmarshal arguments: %w", err))
			}
		}

		// Estimate input tokens from arguments
		inputTokens := tools.EstimateJSONTokens(args)

		result, err := t.Execute(ctx, args)
		result, err = s.finishToolExecution(ctx, toolName, args, start, result, err)

		// Estimate output tokens from result and record budget usage
		outputTokens := 0
		if result != nil && len(result.Content) > 0 {
			for _, content := range result.Content {
				if textContent, ok := content.(*mcp.TextContent); ok {
					outputTokens += tools.EstimateTokens(textContent.Text)
				}
			}
		}
		tools.GetBudgetContext().RecordToolExecution(inputTokens, outputTokens)

		return result, err
	}

	// Register tool with MCP server
	s.mcpServer.AddTool(mcpTool, handler)
	s.logger.Debug("Registered tool", zap.String("tool", mcpTool.Name))
}

// finishToolExecution is the single chokepoint every tool call passes
// through on its way out, regardless of which registered tool ran or how it
// failed. It is responsible for everything that must happen exactly once,
// for every tool, on every call:
//
//  1. Sanitizing credentials/secrets out of both possible error shapes a
//     tool can produce: a Go error returned directly (rare — most tools
//     convert failures into a result with IsError=true before returning,
//     but a few, plus the arguments-unmarshal failure below, return a raw
//     error), and an IsError result's text content (the common case — see
//     tools.NewToolResultError / HandleGetError / HandleQueryError, which
//     embed err.Error() as the result text). Both paths can carry a
//     response-body snippet folded in by the API client's classified
//     errors (client.go's classifyResponseError / internal/errors'
//     FromHTTPStatus), which is attacker/API-controlled and must never
//     reach the MCP client or a log line unmasked.
//  2. Recording an audit.Entry for the call (tool, duration, success, the
//     masked error message, and a SHA-256 hash of the arguments) via the
//     server's audit.Logger, which itself both keeps an in-memory ring
//     buffer and emits a structured zap log line — the "errors logged via
//     zap" sink required alongside the MCP response.
//  3. Recording tool-execution metrics.
//
// Do not sanitize or audit-log per-tool; add new failure modes here instead.
func (s *Server) finishToolExecution(ctx context.Context, toolName string, args map[string]interface{}, start time.Time, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
	duration := time.Since(start)
	success := err == nil && (result == nil || !result.IsError)

	// Sanitize the Go error, if any: this is what a raw (non-IsError-result)
	// tool failure returns, and the MCP SDK serializes it directly into the
	// error surfaced to the client (see mcp.Server.callTool) — never masking
	// it here would leak whatever the error message embeds.
	if err != nil {
		err = errors.New(security.SanitizeError(err))
	}

	// Sanitize an IsError result's text content in place: this is the
	// common path (tools convert failures into text before returning), and
	// that text is returned to the MCP client verbatim.
	maskErrorResultContent(result)

	s.metrics.RecordToolExecution(toolName, success, duration)

	if s.auditLogger != nil {
		auditErr := err
		if auditErr == nil && result != nil && result.IsError {
			// The failure lives in the (already-masked) result content, not
			// in err. Synthesize an error carrying that text so the audit
			// entry still records what went wrong. LogToolExecution masks
			// again via security.SanitizeError, which is idempotent on
			// already-masked text.
			auditErr = errors.New(resultErrorText(result))
		}
		s.auditLogger.LogToolExecution(ctx, toolName, "execute", "", "", success, duration, auditErr, hashArgs(args))
	}

	return result, err
}

// maskErrorResultContent masks the text content of an IsError tool result
// in place. No-op for a nil result or a successful (non-error) one, so
// ordinary tool output is never mangled by pattern matching.
func maskErrorResultContent(result *mcp.CallToolResult) {
	if result == nil || !result.IsError {
		return
	}
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			tc.Text = security.MaskSensitiveData(tc.Text)
		}
	}
}

// resultErrorText concatenates the text content of a tool result, used to
// recover an error-shaped string from a result whose failure was reported
// as IsError content rather than a Go error.
func resultErrorText(result *mcp.CallToolResult) string {
	var parts []string
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok && tc.Text != "" {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "; ")
}

// hashArgs returns a SHA-256 hex digest of the JSON-marshaled tool
// arguments, for audit.Entry.InputHash. This lets audit entries be
// correlated to a specific input without the audit log ever recording the
// raw (potentially sensitive) argument values. Returns "" if args is empty
// or unmarshalable (never fails the call over this).
func hashArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	data, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// registerPrompts registers all available MCP prompts
func (s *Server) registerPrompts() {
	registry := prompts.NewRegistry(s.logger)

	for _, p := range registry.GetPrompts() {
		s.mcpServer.AddPrompt(p.Prompt, p.Handler)
		s.logger.Debug("Registered prompt", zap.String("prompt", p.Prompt.Name))
	}

	s.logger.Info("Registered all MCP prompts", zap.Int("count", len(registry.GetPrompts())))
}

// registerResources registers all available MCP resources and resource templates
func (s *Server) registerResources() {
	registry := resources.NewRegistry(s.config, s.metrics, s.logger, s.version)

	// Register static resources
	for _, r := range registry.GetResources() {
		s.mcpServer.AddResource(r.Resource, r.Handler)
		s.logger.Debug("Registered resource", zap.String("uri", r.Resource.URI))
	}

	// Register resource templates for dynamic resource access
	// Templates allow LLMs to request configuration examples dynamically
	templateHandler := registry.GetTemplateHandler()
	for _, t := range registry.GetResourceTemplates() {
		s.mcpServer.AddResourceTemplate(&t, templateHandler)
		s.logger.Debug("Registered resource template", zap.String("uri_template", t.URITemplate))
	}

	s.logger.Info("Registered all MCP resources",
		zap.Int("static_count", len(registry.GetResources())),
		zap.Int("template_count", len(registry.GetResourceTemplates())),
	)
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP server")

	// Start health HTTP server if configured. Bind synchronously first, so a
	// bad port/bind address fails startup with a clear error instead of
	// being swallowed in a background goroutine; only mark the server ready
	// once the bind has actually succeeded.
	if s.healthServer != nil {
		ln, err := s.healthServer.Listen()
		if err != nil {
			return fmt.Errorf("failed to start health server: %w", err)
		}
		go func() {
			if err := s.healthServer.Serve(ln); err != nil {
				s.logger.Error("Health server error", zap.Error(err))
			}
		}()
		s.healthServer.SetReady(true)
	}

	defer func() {
		// Log final metrics on shutdown
		s.metrics.LogStats()

		// Save user session for persistence (learned patterns, preferences)
		if err := tools.SaveCurrentSession(); err != nil {
			s.logger.Error("Failed to save user session", zap.Error(err))
		} else {
			s.logger.Debug("User session saved successfully")
		}

		// Shutdown health server
		if s.healthServer != nil {
			s.healthServer.SetReady(false)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout(s.config))
			defer cancel()
			if err := s.healthServer.Shutdown(shutdownCtx); err != nil {
				s.logger.Error("Failed to shutdown health server", zap.Error(err))
			}
		}

		if err := s.apiClient.Close(); err != nil {
			s.logger.Error("Failed to close API client", zap.Error(err))
		}
	}()

	// Start serving using stdio transport
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// GetMetrics returns the server's metrics tracker for external access
func (s *Server) GetMetrics() *metrics.Metrics {
	return s.metrics
}

// MCPServer returns the underlying MCP server for testing.
func (s *Server) MCPServer() *mcp.Server {
	return s.mcpServer
}

// AuditLogger returns the server's audit logger, primarily for testing and
// for tools (e.g. a future get_audit_log implementation) that need to read
// back recent audit entries.
func (s *Server) AuditLogger() *audit.Logger {
	return s.auditLogger
}
