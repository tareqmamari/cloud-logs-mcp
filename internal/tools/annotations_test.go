package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetToolIcon(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		expectedIcon ToolIcon
	}{
		// Query tools
		{"query_logs", "query_logs", IconQuery},
		{"build_query", "build_query", IconQuery},
		{"submit_background_query", "submit_background_query", IconQuery},
		{"get_background_query_status", "get_background_query_status", IconQuery},
		{"validate_query", "validate_query", IconQuery},

		// Alert tools
		{"list_alerts", "list_alerts", IconAlert},
		{"get_alert", "get_alert", IconAlert},
		{"create_alert", "create_alert", IconCreate},
		{"update_alert", "update_alert", IconUpdate},
		{"delete_alert", "delete_alert", IconDelete},

		// Dashboard tools
		{"list_dashboards", "list_dashboards", IconDashboard},
		{"get_dashboard", "get_dashboard", IconDashboard},
		{"create_dashboard", "create_dashboard", IconCreate},
		{"delete_dashboard", "delete_dashboard", IconDelete},

		// Policy tools
		{"list_policies", "list_policies", IconPolicy},
		{"get_policy", "get_policy", IconPolicy},
		{"create_policy", "create_policy", IconCreate},

		// Webhook tools
		{"list_outgoing_webhooks", "list_outgoing_webhooks", IconWebhook},
		{"create_outgoing_webhook", "create_outgoing_webhook", IconCreate},

		// E2M tools
		{"list_e2m", "list_e2m", IconE2M},
		{"create_e2m", "create_e2m", IconCreate},

		// Stream tools
		{"list_streams", "list_streams", IconStream},
		{"create_stream", "create_stream", IconCreate},

		// Ingestion
		{"ingest_logs", "ingest_logs", IconIngestion},

		// Workflow tools
		{"investigate_incident", "investigate_incident", IconInvestigate},
		{"health_check", "health_check", IconHealth},

		// Meta tools
		{"search_tools", "search_tools", IconMeta},
		{"describe_tools", "describe_tools", IconMeta},
		{"list_tool_categories", "list_tool_categories", IconMeta},

		// Default fallback
		{"unknown_tool", "unknown_tool", IconWorkflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetToolIcon(tt.toolName)
			assert.Equal(t, tt.expectedIcon, icon, "Icon mismatch for tool: %s", tt.toolName)
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		expected bool
	}{
		{"match", "query_logs", "query_", true},
		{"no match", "list_alerts", "query_", false},
		{"exact match", "query_logs", "query_logs", true},
		{"prefix longer than string", "ab", "abc", false},
		{"empty prefix", "query_logs", "", true},
		{"empty string", "", "query_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPrefix(tt.input, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnnotationFunctions(t *testing.T) {
	t.Run("ReadOnlyAnnotations", func(t *testing.T) {
		ann := ReadOnlyAnnotations("Test Read")
		assert.Equal(t, "Test Read", ann.Title)
		assert.True(t, ann.ReadOnlyHint)
		assert.True(t, ann.IdempotentHint)
		assert.NotNil(t, ann.OpenWorldHint)
		assert.False(t, *ann.OpenWorldHint)
	})

	t.Run("CreateAnnotations", func(t *testing.T) {
		ann := CreateAnnotations("Test Create")
		assert.Equal(t, "Test Create", ann.Title)
		assert.False(t, ann.ReadOnlyHint)
		assert.False(t, ann.IdempotentHint)
		assert.NotNil(t, ann.DestructiveHint)
		assert.False(t, *ann.DestructiveHint)
	})

	t.Run("UpdateAnnotations", func(t *testing.T) {
		ann := UpdateAnnotations("Test Update")
		assert.Equal(t, "Test Update", ann.Title)
		assert.False(t, ann.ReadOnlyHint)
		assert.True(t, ann.IdempotentHint)
		assert.NotNil(t, ann.DestructiveHint)
		assert.False(t, *ann.DestructiveHint)
	})

	t.Run("DeleteAnnotations", func(t *testing.T) {
		ann := DeleteAnnotations("Test Delete")
		assert.Equal(t, "Test Delete", ann.Title)
		assert.False(t, ann.ReadOnlyHint)
		assert.True(t, ann.IdempotentHint)
		assert.NotNil(t, ann.DestructiveHint)
		assert.True(t, *ann.DestructiveHint)
	})

	t.Run("QueryAnnotations", func(t *testing.T) {
		ann := QueryAnnotations("Test Query")
		assert.Equal(t, "Test Query", ann.Title)
		assert.True(t, ann.ReadOnlyHint)
		assert.True(t, ann.IdempotentHint)
	})

	t.Run("WorkflowAnnotations", func(t *testing.T) {
		ann := WorkflowAnnotations("Test Workflow")
		assert.Equal(t, "Test Workflow", ann.Title)
		assert.True(t, ann.ReadOnlyHint)
		assert.True(t, ann.IdempotentHint)
	})

	t.Run("IngestionAnnotations", func(t *testing.T) {
		ann := IngestionAnnotations("Test Ingestion")
		assert.Equal(t, "Test Ingestion", ann.Title)
		assert.False(t, ann.ReadOnlyHint)
		assert.False(t, ann.IdempotentHint)
		assert.NotNil(t, ann.DestructiveHint)
		assert.False(t, *ann.DestructiveHint)
	})

	t.Run("DefaultAnnotations", func(t *testing.T) {
		ann := DefaultAnnotations("Test Default")
		assert.Equal(t, "Test Default", ann.Title)
		assert.NotNil(t, ann.OpenWorldHint)
		assert.False(t, *ann.OpenWorldHint)
	})
}

func TestIconConstants(t *testing.T) {
	// Verify icon constants are defined and non-empty
	icons := []ToolIcon{
		IconQuery,
		IconAlert,
		IconDashboard,
		IconPolicy,
		IconWebhook,
		IconE2M,
		IconEnrichment,
		IconView,
		IconDataAccess,
		IconStream,
		IconIngestion,
		IconWorkflow,
		IconMeta,
		IconCreate,
		IconUpdate,
		IconDelete,
		IconInvestigate,
		IconHealth,
	}

	for _, icon := range icons {
		assert.NotEmpty(t, string(icon), "Icon should not be empty")
	}
}
