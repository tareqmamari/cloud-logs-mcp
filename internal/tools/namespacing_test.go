package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetToolNamespace(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected ToolNamespace
	}{
		{"query tool", "query_logs", NamespaceQuery},
		{"build query", "build_query", NamespaceQuery},
		{"alert tool", "list_alerts", NamespaceAlert},
		{"create alert", "create_alert", NamespaceAlert},
		{"dashboard tool", "list_dashboards", NamespaceDashboard},
		{"policy tool", "list_policies", NamespacePolicy},
		{"webhook tool", "create_outgoing_webhook", NamespaceWebhook},
		{"e2m tool", "create_e2m", NamespaceE2M},
		{"stream tool", "list_streams", NamespaceStream},
		{"view tool", "list_views", NamespaceView},
		{"workflow tool", "investigate_incident", NamespaceWorkflow},
		{"meta tool", "search_tools", NamespaceMeta},
		{"unknown tool defaults to meta", "unknown_tool", NamespaceMeta},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetToolNamespace(tt.toolName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetToolsByNamespace(t *testing.T) {
	t.Run("query namespace has tools", func(t *testing.T) {
		tools := GetToolsByNamespace(NamespaceQuery)
		assert.Greater(t, len(tools), 0)
		assert.Contains(t, tools, "query_logs")
	})

	t.Run("alert namespace has tools", func(t *testing.T) {
		tools := GetToolsByNamespace(NamespaceAlert)
		assert.Greater(t, len(tools), 0)
		assert.Contains(t, tools, "list_alerts")
	})

	t.Run("workflow namespace has tools", func(t *testing.T) {
		tools := GetToolsByNamespace(NamespaceWorkflow)
		assert.Greater(t, len(tools), 0)
		assert.Contains(t, tools, "investigate_incident")
	})
}

func TestGetAllNamespaces(t *testing.T) {
	namespaces := GetAllNamespaces()

	t.Run("has expected namespaces", func(t *testing.T) {
		assert.Contains(t, namespaces, NamespaceQuery)
		assert.Contains(t, namespaces, NamespaceAlert)
		assert.Contains(t, namespaces, NamespaceDashboard)
		assert.Contains(t, namespaces, NamespaceWorkflow)
		assert.Contains(t, namespaces, NamespaceMeta)
	})

	t.Run("counts are positive", func(t *testing.T) {
		for ns, count := range namespaces {
			assert.Greater(t, count, 0, "Namespace %s should have tools", ns)
		}
	})
}

func TestGetNamespaceInfo(t *testing.T) {
	t.Run("returns info without tools", func(t *testing.T) {
		info := GetNamespaceInfo(NamespaceQuery, false)
		assert.Equal(t, NamespaceQuery, info.Name)
		assert.NotEmpty(t, info.Description)
		assert.Greater(t, info.ToolCount, 0)
		assert.Nil(t, info.Tools)
	})

	t.Run("returns info with tools", func(t *testing.T) {
		info := GetNamespaceInfo(NamespaceQuery, true)
		assert.Equal(t, NamespaceQuery, info.Name)
		assert.NotEmpty(t, info.Description)
		assert.Greater(t, info.ToolCount, 0)
		assert.NotNil(t, info.Tools)
		assert.Equal(t, info.ToolCount, len(info.Tools))
	})
}

func TestGetAllNamespaceInfo(t *testing.T) {
	t.Run("returns all namespace info", func(t *testing.T) {
		infos := GetAllNamespaceInfo(false)
		assert.Greater(t, len(infos), 0)

		// All infos should have descriptions
		for _, info := range infos {
			assert.NotEmpty(t, info.Description, "Namespace %s should have description", info.Name)
		}
	})
}

func TestParseNamespacedTool(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedNS   ToolNamespace
		expectedTool string
	}{
		{"namespaced tool", "queries/query_logs", "queries", "query_logs"},
		{"alerts namespace", "alerts/create_alert", "alerts", "create_alert"},
		{"non-namespaced tool", "query_logs", "", "query_logs"},
		{"single segment", "health_check", "", "health_check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, tool := ParseNamespacedTool(tt.input)
			assert.Equal(t, tt.expectedNS, ns)
			assert.Equal(t, tt.expectedTool, tool)
		})
	}
}

func TestFormatNamespacedTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected string
	}{
		{"query tool", "query_logs", "queries/query_logs"},
		{"alert tool", "list_alerts", "alerts/list_alerts"},
		{"workflow tool", "investigate_incident", "workflows/investigate_incident"},
		{"meta tool", "search_tools", "meta/search_tools"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNamespacedTool(tt.toolName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
