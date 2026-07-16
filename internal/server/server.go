// Package server provides the MCP server implementation for the logs service.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/auth"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/health"
	"github.com/tareqmamari/cloud-logs-mcp/internal/metrics"
	"github.com/tareqmamari/cloud-logs-mcp/internal/prompts"
	"github.com/tareqmamari/cloud-logs-mcp/internal/resources"
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
// goroutine is left to finish in the background — acceptable since the
// underlying call is a single bounded-by-nature HTTP round trip to IAM, not
// an unbounded operation.
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
			zap.Error(err),
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
		logger.Warn("Failed to fetch TCO policies, will use defaults", zap.Error(err))
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

	// Create handler that calls the tool's Execute method with metrics tracking
	handler := func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// Add client to context for tool execution
		// This enables per-request client injection for future HTTP transport
		ctx = tools.WithClient(ctx, s.apiClient)

		// Add session to context for tool execution
		// This enables per-request session injection for better testability
		ctx = tools.WithSession(ctx, tools.GetSession())
		ctx = tools.WithSessionProvider(ctx, tools.GetSessionManager())

		var args map[string]interface{}
		if len(request.Params.Arguments) > 0 {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				s.metrics.RecordToolExecution(toolName, false, time.Since(start))
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}
		}

		// Estimate input tokens from arguments
		inputTokens := tools.EstimateJSONTokens(args)

		result, err := t.Execute(ctx, args)
		success := err == nil && (result == nil || !result.IsError)
		s.metrics.RecordToolExecution(toolName, success, time.Since(start))

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
