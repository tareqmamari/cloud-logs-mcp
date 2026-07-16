// Package tools provides the MCP tool implementations for IBM Cloud Logs.
//
// This file is the SINGLE SOURCE OF TRUTH for which tools the server exposes.
// Everything else is derived from the descriptor table below:
//
//   - GetAllTools()              -> constructs every descriptor's tool
//   - GetToolCount()             -> len(Descriptors())
//   - server.registerTools()     -> iterates Descriptors() (see internal/server)
//   - toolNamespaceMapping       -> name -> Namespace (see namespacing.go)
//   - discovery baseline metadata-> name -> Category  (see discovery.go)
//
// Historically these lived in four hand-maintained lists that drifted apart.
// Adding or removing a tool now means editing this one table.
package tools

import (
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// ToolDescriptor describes one registrable tool: how to construct it plus the
// static metadata (namespace, category) other subsystems derive from it.
type ToolDescriptor struct {
	// New constructs the tool with the given API client and logger. It must be
	// safe to call with nil arguments (name/schema introspection relies on it).
	New func(c client.Doer, logger *zap.Logger) Tool

	// Namespace is the hierarchical group the tool belongs to (see namespacing.go).
	Namespace ToolNamespace

	// Category is the primary functional category used as the discovery baseline
	// for tools that do not supply richer hand-written metadata (see discovery.go).
	Category ToolCategory
}

// desc adapts a concrete tool constructor (returning e.g. *GetAlertTool) into a
// ToolDescriptor whose New returns the Tool interface.
func desc[T Tool](ctor func(client.Doer, *zap.Logger) T, ns ToolNamespace, cat ToolCategory) ToolDescriptor {
	return ToolDescriptor{
		New:       func(c client.Doer, l *zap.Logger) Tool { return ctor(c, l) },
		Namespace: ns,
		Category:  cat,
	}
}

// Descriptors returns the canonical, ordered list of every tool the server
// registers. The ORDER is preserved for readability only; the served set is
// order-independent (the MCP layer and the golden contract both compare sets).
//
// NOTE: This list intentionally excludes the rule-helper tools
// (discover_log_fields, test_rule_pattern) and the smart-investigation tool,
// which the server has never registered.
func Descriptors() []ToolDescriptor {
	return []ToolDescriptor{
		// Alert tools
		desc(NewGetAlertTool, NamespaceAlert, CategoryAlert),
		desc(NewListAlertsTool, NamespaceAlert, CategoryAlert),
		desc(NewCreateAlertTool, NamespaceAlert, CategoryAlert),
		desc(NewUpdateAlertTool, NamespaceAlert, CategoryAlert),
		desc(NewDeleteAlertTool, NamespaceAlert, CategoryAlert),

		// Alert Definition tools
		desc(NewGetAlertDefinitionTool, NamespaceAlert, CategoryAlerting),
		desc(NewListAlertDefinitionsTool, NamespaceAlert, CategoryAlerting),
		desc(NewCreateAlertDefinitionTool, NamespaceAlert, CategoryAlerting),
		desc(NewUpdateAlertDefinitionTool, NamespaceAlert, CategoryAlerting),
		desc(NewDeleteAlertDefinitionTool, NamespaceAlert, CategoryAlerting),

		// Rule Group tools
		desc(NewGetRuleGroupTool, NamespaceRule, CategoryRuleGroup),
		desc(NewListRuleGroupsTool, NamespaceRule, CategoryRuleGroup),
		desc(NewCreateRuleGroupTool, NamespaceRule, CategoryRuleGroup),
		desc(NewUpdateRuleGroupTool, NamespaceRule, CategoryRuleGroup),
		desc(NewDeleteRuleGroupTool, NamespaceRule, CategoryRuleGroup),

		// Outgoing Webhook tools
		desc(NewGetOutgoingWebhookTool, NamespaceWebhook, CategoryWebhook),
		desc(NewListOutgoingWebhooksTool, NamespaceWebhook, CategoryWebhook),
		desc(NewCreateOutgoingWebhookTool, NamespaceWebhook, CategoryWebhook),
		desc(NewUpdateOutgoingWebhookTool, NamespaceWebhook, CategoryWebhook),
		desc(NewDeleteOutgoingWebhookTool, NamespaceWebhook, CategoryWebhook),

		// Policy tools
		desc(NewGetPolicyTool, NamespacePolicy, CategoryPolicy),
		desc(NewListPoliciesTool, NamespacePolicy, CategoryPolicy),
		desc(NewCreatePolicyTool, NamespacePolicy, CategoryPolicy),
		desc(NewUpdatePolicyTool, NamespacePolicy, CategoryPolicy),
		desc(NewDeletePolicyTool, NamespacePolicy, CategoryPolicy),

		// Events to Metrics (E2M) tools
		desc(NewGetE2MTool, NamespaceE2M, CategoryE2M),
		desc(NewListE2MTool, NamespaceE2M, CategoryE2M),
		desc(NewCreateE2MTool, NamespaceE2M, CategoryE2M),
		desc(NewReplaceE2MTool, NamespaceE2M, CategoryE2M),
		desc(NewDeleteE2MTool, NamespaceE2M, CategoryE2M),

		// Query tools
		desc(NewQueryTool, NamespaceQuery, CategoryQuery),
		desc(NewBuildQueryTool, NamespaceQuery, CategoryQuery),
		desc(NewDataPrimeReferenceTool, NamespaceQuery, CategoryQuery),
		desc(NewSubmitBackgroundQueryTool, NamespaceQuery, CategoryQuery),
		desc(NewGetBackgroundQueryStatusTool, NamespaceQuery, CategoryQuery),
		desc(NewGetBackgroundQueryDataTool, NamespaceQuery, CategoryQuery),
		desc(NewCancelBackgroundQueryTool, NamespaceQuery, CategoryQuery),

		// Log Ingestion tools
		desc(NewIngestLogsTool, NamespaceIngestion, CategoryIngestion),

		// Data Access Rule tools
		desc(NewListDataAccessRulesTool, NamespaceDataAccess, CategoryDataAccess),
		desc(NewGetDataAccessRuleTool, NamespaceDataAccess, CategoryDataAccess),
		desc(NewCreateDataAccessRuleTool, NamespaceDataAccess, CategoryDataAccess),
		desc(NewUpdateDataAccessRuleTool, NamespaceDataAccess, CategoryDataAccess),
		desc(NewDeleteDataAccessRuleTool, NamespaceDataAccess, CategoryDataAccess),

		// Enrichment tools
		desc(NewListEnrichmentsTool, NamespaceEnrichment, CategoryEnrichment),
		desc(NewGetEnrichmentsTool, NamespaceEnrichment, CategoryEnrichment),
		desc(NewCreateEnrichmentTool, NamespaceEnrichment, CategoryEnrichment),
		desc(NewUpdateEnrichmentTool, NamespaceEnrichment, CategoryEnrichment),
		desc(NewDeleteEnrichmentTool, NamespaceEnrichment, CategoryEnrichment),

		// View tools
		desc(NewListViewsTool, NamespaceView, CategoryView),
		desc(NewCreateViewTool, NamespaceView, CategoryView),
		desc(NewGetViewTool, NamespaceView, CategoryView),
		desc(NewReplaceViewTool, NamespaceView, CategoryView),
		desc(NewDeleteViewTool, NamespaceView, CategoryView),

		// View Folder tools
		desc(NewListViewFoldersTool, NamespaceView, CategoryView),
		desc(NewCreateViewFolderTool, NamespaceView, CategoryView),
		desc(NewGetViewFolderTool, NamespaceView, CategoryView),
		desc(NewReplaceViewFolderTool, NamespaceView, CategoryView),
		desc(NewDeleteViewFolderTool, NamespaceView, CategoryView),

		// Data Usage tools
		desc(NewExportDataUsageTool, NamespaceDataUsage, CategoryDataUsage),
		desc(NewUpdateDataUsageMetricsExportStatusTool, NamespaceDataUsage, CategoryDataUsage),

		// Event Stream Target tools
		desc(NewGetEventStreamTargetsTool, NamespaceEventStream, CategoryStream),
		desc(NewCreateEventStreamTargetTool, NamespaceEventStream, CategoryStream),
		desc(NewUpdateEventStreamTargetTool, NamespaceEventStream, CategoryStream),
		desc(NewDeleteEventStreamTargetTool, NamespaceEventStream, CategoryStream),

		// Dashboard tools
		desc(NewListDashboardsTool, NamespaceDashboard, CategoryDashboard),
		desc(NewGetDashboardTool, NamespaceDashboard, CategoryDashboard),
		desc(NewCreateDashboardTool, NamespaceDashboard, CategoryDashboard),
		desc(NewUpdateDashboardTool, NamespaceDashboard, CategoryDashboard),
		desc(NewDeleteDashboardTool, NamespaceDashboard, CategoryDashboard),

		// Dashboard Folder and Management tools
		desc(NewListDashboardFoldersTool, NamespaceDashboard, CategoryDashboard),
		desc(NewGetDashboardFolderTool, NamespaceDashboard, CategoryDashboard),
		desc(NewCreateDashboardFolderTool, NamespaceDashboard, CategoryDashboard),
		desc(NewUpdateDashboardFolderTool, NamespaceDashboard, CategoryDashboard),
		desc(NewDeleteDashboardFolderTool, NamespaceDashboard, CategoryDashboard),
		desc(NewMoveDashboardToFolderTool, NamespaceDashboard, CategoryDashboard),
		desc(NewPinDashboardTool, NamespaceDashboard, CategoryDashboard),
		desc(NewUnpinDashboardTool, NamespaceDashboard, CategoryDashboard),
		desc(NewSetDefaultDashboardTool, NamespaceDashboard, CategoryDashboard),

		// Stream tools
		desc(NewListStreamsTool, NamespaceStream, CategoryStream),
		desc(NewGetStreamTool, NamespaceStream, CategoryStream),
		desc(NewCreateStreamTool, NamespaceStream, CategoryStream),
		desc(NewUpdateStreamTool, NamespaceStream, CategoryStream),
		desc(NewDeleteStreamTool, NamespaceStream, CategoryStream),

		// AI Helper tools
		desc(NewExplainQueryTool, NamespaceQuery, CategoryAIHelper),
		desc(NewAdvancedSuggestAlertTool, NamespaceAlert, CategoryAIHelper), // SRE-grade alert recommendations
		desc(NewGetAuditLogTool, NamespaceQuery, CategoryQuery),

		// Query Intelligence tools
		desc(NewQueryTemplatesTool, NamespaceQuery, CategoryQuery),
		desc(NewValidateQueryTool, NamespaceQuery, CategoryQuery),
		desc(NewQueryCostEstimateTool, NamespaceQuery, CategoryQuery),

		// Workflow Automation tools
		desc(NewInvestigateIncidentTool, NamespaceWorkflow, CategoryWorkflow),
		desc(NewHealthCheckTool, NamespaceWorkflow, CategoryWorkflow),

		// Meta tools (discovery and session management)
		desc(NewDiscoverToolsTool, NamespaceMeta, CategoryMeta),
		desc(NewSessionContextTool, NamespaceMeta, CategoryMeta),

		// Dynamic toolset meta-tools (token-efficient discovery pattern)
		// These enable: search_tools -> describe_tools -> execute workflow
		desc(NewSearchToolsTool, NamespaceMeta, CategoryMeta),
		desc(NewDescribeToolsTool, NamespaceMeta, CategoryMeta),
		desc(NewListToolCategoriesBrief, NamespaceMeta, CategoryMeta),
	}
}
