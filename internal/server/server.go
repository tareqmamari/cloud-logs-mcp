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

// registerTools registers all available MCP tools
func (s *Server) registerTools() error {
	// Alert tools
	s.registerTool(tools.NewGetAlertTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListAlertsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateAlertTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateAlertTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteAlertTool(s.apiClient, s.logger))

	// Alert Definition tools
	s.registerTool(tools.NewGetAlertDefinitionTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListAlertDefinitionsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateAlertDefinitionTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateAlertDefinitionTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteAlertDefinitionTool(s.apiClient, s.logger))

	// Rule Group tools
	s.registerTool(tools.NewGetRuleGroupTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListRuleGroupsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateRuleGroupTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateRuleGroupTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteRuleGroupTool(s.apiClient, s.logger))

	// Outgoing Webhook tools
	s.registerTool(tools.NewGetOutgoingWebhookTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListOutgoingWebhooksTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateOutgoingWebhookTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateOutgoingWebhookTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteOutgoingWebhookTool(s.apiClient, s.logger))

	// Policy tools
	s.registerTool(tools.NewGetPolicyTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListPoliciesTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreatePolicyTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdatePolicyTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeletePolicyTool(s.apiClient, s.logger))

	// Events to Metrics (E2M) tools
	s.registerTool(tools.NewGetE2MTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListE2MTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateE2MTool(s.apiClient, s.logger))
	s.registerTool(tools.NewReplaceE2MTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteE2MTool(s.apiClient, s.logger))

	// Query tools
	s.registerTool(tools.NewQueryTool(s.apiClient, s.logger))
	s.registerTool(tools.NewBuildQueryTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDataPrimeReferenceTool(s.apiClient, s.logger))
	s.registerTool(tools.NewSubmitBackgroundQueryTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetBackgroundQueryStatusTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetBackgroundQueryDataTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCancelBackgroundQueryTool(s.apiClient, s.logger))

	// Log Ingestion tools
	s.registerTool(tools.NewIngestLogsTool(s.apiClient, s.logger))

	// Data Access Rule tools
	s.registerTool(tools.NewListDataAccessRulesTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetDataAccessRuleTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateDataAccessRuleTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateDataAccessRuleTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteDataAccessRuleTool(s.apiClient, s.logger))

	// Enrichment tools
	s.registerTool(tools.NewListEnrichmentsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetEnrichmentsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateEnrichmentTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateEnrichmentTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteEnrichmentTool(s.apiClient, s.logger))

	// View tools
	s.registerTool(tools.NewListViewsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateViewTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetViewTool(s.apiClient, s.logger))
	s.registerTool(tools.NewReplaceViewTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteViewTool(s.apiClient, s.logger))

	// View Folder tools
	s.registerTool(tools.NewListViewFoldersTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateViewFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetViewFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewReplaceViewFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteViewFolderTool(s.apiClient, s.logger))

	// Data Usage tools
	s.registerTool(tools.NewExportDataUsageTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateDataUsageMetricsExportStatusTool(s.apiClient, s.logger))

	// Event Stream Target tools
	s.registerTool(tools.NewGetEventStreamTargetsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateEventStreamTargetTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateEventStreamTargetTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteEventStreamTargetTool(s.apiClient, s.logger))

	// Dashboard tools
	s.registerTool(tools.NewListDashboardsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetDashboardTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateDashboardTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateDashboardTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteDashboardTool(s.apiClient, s.logger))

	// Dashboard Folder and Management tools
	s.registerTool(tools.NewListDashboardFoldersTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetDashboardFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateDashboardFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateDashboardFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteDashboardFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewMoveDashboardToFolderTool(s.apiClient, s.logger))
	s.registerTool(tools.NewPinDashboardTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUnpinDashboardTool(s.apiClient, s.logger))
	s.registerTool(tools.NewSetDefaultDashboardTool(s.apiClient, s.logger))

	// Stream tools
	s.registerTool(tools.NewListStreamsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetStreamTool(s.apiClient, s.logger))
	s.registerTool(tools.NewCreateStreamTool(s.apiClient, s.logger))
	s.registerTool(tools.NewUpdateStreamTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDeleteStreamTool(s.apiClient, s.logger))

	// AI Helper tools
	s.registerTool(tools.NewExplainQueryTool(s.apiClient, s.logger))
	s.registerTool(tools.NewSuggestAlertTool(s.apiClient, s.logger))
	s.registerTool(tools.NewGetAuditLogTool(s.apiClient, s.logger))

	// Query Intelligence tools
	s.registerTool(tools.NewQueryTemplatesTool(s.apiClient, s.logger))
	s.registerTool(tools.NewValidateQueryTool(s.apiClient, s.logger))
	s.registerTool(tools.NewQueryCostEstimateTool(s.apiClient, s.logger))

	// Pattern Discovery tools (Investigation Mode)
	s.registerTool(tools.NewScoutLogsTool(s.apiClient, s.logger))

	// Workflow Automation tools
	s.registerTool(tools.NewInvestigateIncidentTool(s.apiClient, s.logger))
	s.registerTool(tools.NewHealthCheckTool(s.apiClient, s.logger))

	// Meta tools (discovery and session management)
	s.registerTool(tools.NewDiscoverToolsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewSessionContextTool(s.apiClient, s.logger))

	// Dynamic toolset meta-tools (token-efficient discovery pattern)
	// These enable: search_tools → describe_tools → execute workflow
	s.registerTool(tools.NewSearchToolsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewDescribeToolsTool(s.apiClient, s.logger))
	s.registerTool(tools.NewListToolCategoriesBrief(s.apiClient, s.logger))

	s.logger.Info("Registered all MCP tools")
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
