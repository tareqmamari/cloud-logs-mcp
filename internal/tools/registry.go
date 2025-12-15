// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// GetAllTools returns all available MCP tools organized by category.
// This factory function centralizes tool creation and makes it easy to
// add new tools or modify tool registration.
func GetAllTools(c *client.Client, logger *zap.Logger) []Tool {
	return []Tool{
		// Alert tools
		NewGetAlertTool(c, logger),
		NewListAlertsTool(c, logger),
		NewCreateAlertTool(c, logger),
		NewUpdateAlertTool(c, logger),
		NewDeleteAlertTool(c, logger),

		// Alert Definition tools
		NewGetAlertDefinitionTool(c, logger),
		NewListAlertDefinitionsTool(c, logger),
		NewCreateAlertDefinitionTool(c, logger),
		NewUpdateAlertDefinitionTool(c, logger),
		NewDeleteAlertDefinitionTool(c, logger),

		// Rule Group tools
		NewGetRuleGroupTool(c, logger),
		NewListRuleGroupsTool(c, logger),
		NewCreateRuleGroupTool(c, logger),
		NewUpdateRuleGroupTool(c, logger),
		NewDeleteRuleGroupTool(c, logger),

		// Outgoing Webhook tools
		NewGetOutgoingWebhookTool(c, logger),
		NewListOutgoingWebhooksTool(c, logger),
		NewCreateOutgoingWebhookTool(c, logger),
		NewUpdateOutgoingWebhookTool(c, logger),
		NewDeleteOutgoingWebhookTool(c, logger),

		// Policy tools
		NewGetPolicyTool(c, logger),
		NewListPoliciesTool(c, logger),
		NewCreatePolicyTool(c, logger),
		NewUpdatePolicyTool(c, logger),
		NewDeletePolicyTool(c, logger),

		// Events to Metrics (E2M) tools
		NewGetE2MTool(c, logger),
		NewListE2MTool(c, logger),
		NewCreateE2MTool(c, logger),
		NewReplaceE2MTool(c, logger),
		NewDeleteE2MTool(c, logger),

		// Query tools
		NewQueryTool(c, logger),
		NewBuildQueryTool(c, logger),
		NewDataPrimeReferenceTool(c, logger),
		NewSubmitBackgroundQueryTool(c, logger),
		NewGetBackgroundQueryStatusTool(c, logger),
		NewGetBackgroundQueryDataTool(c, logger),
		NewCancelBackgroundQueryTool(c, logger),

		// Log Ingestion tools
		NewIngestLogsTool(c, logger),

		// Data Access Rule tools
		NewListDataAccessRulesTool(c, logger),
		NewGetDataAccessRuleTool(c, logger),
		NewCreateDataAccessRuleTool(c, logger),
		NewUpdateDataAccessRuleTool(c, logger),
		NewDeleteDataAccessRuleTool(c, logger),

		// Enrichment tools
		NewListEnrichmentsTool(c, logger),
		NewGetEnrichmentsTool(c, logger),
		NewCreateEnrichmentTool(c, logger),
		NewUpdateEnrichmentTool(c, logger),
		NewDeleteEnrichmentTool(c, logger),

		// View tools
		NewListViewsTool(c, logger),
		NewCreateViewTool(c, logger),
		NewGetViewTool(c, logger),
		NewReplaceViewTool(c, logger),
		NewDeleteViewTool(c, logger),

		// View Folder tools
		NewListViewFoldersTool(c, logger),
		NewCreateViewFolderTool(c, logger),
		NewGetViewFolderTool(c, logger),
		NewReplaceViewFolderTool(c, logger),
		NewDeleteViewFolderTool(c, logger),

		// Data Usage tools
		NewExportDataUsageTool(c, logger),
		NewUpdateDataUsageMetricsExportStatusTool(c, logger),

		// Event Stream Target tools
		NewGetEventStreamTargetsTool(c, logger),
		NewCreateEventStreamTargetTool(c, logger),
		NewUpdateEventStreamTargetTool(c, logger),
		NewDeleteEventStreamTargetTool(c, logger),

		// Dashboard tools
		NewListDashboardsTool(c, logger),
		NewGetDashboardTool(c, logger),
		NewCreateDashboardTool(c, logger),
		NewUpdateDashboardTool(c, logger),
		NewDeleteDashboardTool(c, logger),

		// Dashboard Folder and Management tools
		NewListDashboardFoldersTool(c, logger),
		NewGetDashboardFolderTool(c, logger),
		NewCreateDashboardFolderTool(c, logger),
		NewUpdateDashboardFolderTool(c, logger),
		NewDeleteDashboardFolderTool(c, logger),
		NewMoveDashboardToFolderTool(c, logger),
		NewPinDashboardTool(c, logger),
		NewUnpinDashboardTool(c, logger),
		NewSetDefaultDashboardTool(c, logger),

		// Stream tools
		NewListStreamsTool(c, logger),
		NewGetStreamTool(c, logger),
		NewCreateStreamTool(c, logger),
		NewUpdateStreamTool(c, logger),
		NewDeleteStreamTool(c, logger),

		// AI Helper tools
		NewExplainQueryTool(c, logger),
		NewSuggestAlertTool(c, logger),
		NewGetAuditLogTool(c, logger),

		// Query Intelligence tools
		NewQueryTemplatesTool(c, logger),
		NewValidateQueryTool(c, logger),
		NewQueryCostEstimateTool(c, logger),

		// Workflow Automation tools
		NewInvestigateIncidentTool(c, logger),
		NewHealthCheckTool(c, logger),

		// Meta tools (discovery and session management)
		NewDiscoverToolsTool(c, logger),
		NewSessionContextTool(c, logger),

		// Dynamic toolset meta-tools (token-efficient discovery pattern)
		// These enable: search_tools → describe_tools → execute workflow
		NewSearchToolsTool(c, logger),
		NewDescribeToolsTool(c, logger),
		NewListToolCategoriesBrief(c, logger),
	}
}

// GetToolCount returns the total number of registered tools.
// Useful for metrics and logging.
func GetToolCount() int {
	return 87 // Update this when adding new tools
}
