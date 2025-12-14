// Package server provides the MCP server implementation for the logs service.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/auth"
	"github.com/tareqmamari/logs-mcp-server/internal/client"
	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"github.com/tareqmamari/logs-mcp-server/internal/health"
	"github.com/tareqmamari/logs-mcp-server/internal/metrics"
	"github.com/tareqmamari/logs-mcp-server/internal/prompts"
	"github.com/tareqmamari/logs-mcp-server/internal/resources"
	"github.com/tareqmamari/logs-mcp-server/internal/tools"
)

// Server represents the MCP server
type Server struct {
	mcpServer     *mcp.Server
	apiClient     *client.Client
	config        *config.Config
	logger        *zap.Logger
	metrics       *metrics.Metrics
	version       string
	healthServer  *health.Server
	authenticator *auth.Authenticator
}

// New creates a new MCP server instance.
func New(cfg *config.Config, logger *zap.Logger, version string) (*Server, error) {
	// Create IBM Cloud Logs API client
	apiClient, err := client.New(cfg, logger, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create authenticator for health checks
	authenticator, err := auth.New(cfg.APIKey, cfg.IAMURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

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

	// Initialize user-specific session using JWT subject from IAM token
	// The subject uniquely identifies the user/service across sessions
	userID, err := authenticator.GetUserIdentity()
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

// registerTools registers all available MCP tools using the centralized registry.
func (s *Server) registerTools() error {
	allTools := tools.GetAllTools(s.apiClient, s.logger)
	for _, t := range allTools {
		s.registerTool(t)
	}
	s.logger.Info("Registered all MCP tools", zap.Int("count", len(allTools)))
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

		// Apply tool-specific timeout if defined
		// This allows different tool categories (queries, workflows, etc.)
		// to have appropriate timeout values based on their expected execution time
		if timeout := t.DefaultTimeout(); timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		// Add client to context for tool execution
		// This enables per-request client injection for future HTTP transport
		ctx = tools.WithClient(ctx, s.apiClient)

		// Add session to context for tool execution
		// This enables per-request session injection for better testability
		ctx = tools.WithSession(ctx, tools.GetSession())

		var args map[string]interface{}
		if len(request.Params.Arguments) > 0 {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				s.metrics.RecordToolExecution(toolName, false, time.Since(start))
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}
		}

		result, err := t.Execute(ctx, args)
		success := err == nil && (result == nil || !result.IsError)
		s.metrics.RecordToolExecution(toolName, success, time.Since(start))

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

	// Start health HTTP server in background if configured
	if s.healthServer != nil {
		go func() {
			if err := s.healthServer.Start(); err != nil {
				s.logger.Error("Health server error", zap.Error(err))
			}
		}()
		// Mark as ready once server is starting
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
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
