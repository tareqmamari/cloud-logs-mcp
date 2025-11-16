package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"github.com/tareqmamari/logs-mcp-server/internal/tools"
)

// Server represents the MCP server
type Server struct {
	mcpServer *server.MCPServer
	apiClient *client.Client
	config    *config.Config
	logger    *zap.Logger
}

// New creates a new MCP server
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Create API client with IBM Cloud authentication
	apiClient, err := client.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"IBM Cloud Logs MCP Server",
		"0.2.0",
	)

	s := &Server{
		mcpServer: mcpServer,
		apiClient: apiClient,
		config:    cfg,
		logger:    logger,
	}

	// Register all tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

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

	s.logger.Info("Registered all MCP tools")
	return nil
}

// registerTool is a helper to register a tool with proper error handling
func (s *Server) registerTool(toolInterface interface {
	Name() string
	Description() string
	InputSchema() mcp.ToolInputSchema
	Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}) {
	// Create tool definition
	tool := mcp.Tool{
		Name:        toolInterface.Name(),
		Description: toolInterface.Description(),
		InputSchema: toolInterface.InputSchema(),
	}

	// Create handler that calls the tool's Execute method
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolInterface.Execute(ctx, request.GetArguments())
	}

	// Register tool with MCP server
	s.mcpServer.AddTool(tool, handler)
	s.logger.Debug("Registered tool", zap.String("tool", tool.Name))
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP server")

	defer func() {
		if err := s.apiClient.Close(); err != nil {
			s.logger.Error("Failed to close API client", zap.Error(err))
		}
	}()

	// Start serving using stdio transport
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
