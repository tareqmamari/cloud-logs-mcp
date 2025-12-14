package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllTools(t *testing.T) {
	tools := GetAllTools(nil, nil)

	t.Run("returns non-empty list", func(t *testing.T) {
		assert.NotEmpty(t, tools, "GetAllTools should return tools")
	})

	t.Run("all tools have required methods", func(t *testing.T) {
		for _, tool := range tools {
			assert.NotEmpty(t, tool.Name(), "Tool should have a name")
			assert.NotEmpty(t, tool.Description(), "Tool %s should have a description", tool.Name())
			assert.NotNil(t, tool.InputSchema(), "Tool %s should have an input schema", tool.Name())
			// DefaultTimeout can be 0 (use default) or positive
			assert.GreaterOrEqual(t, tool.DefaultTimeout(), time.Duration(0),
				"Tool %s should have non-negative timeout", tool.Name())
		}
	})

	t.Run("no duplicate tool names", func(t *testing.T) {
		names := make(map[string]bool)
		for _, tool := range tools {
			name := tool.Name()
			assert.False(t, names[name], "Duplicate tool name: %s", name)
			names[name] = true
		}
	})

	t.Run("expected tool count", func(t *testing.T) {
		// Update this number when adding/removing tools
		expectedMin := 80 // At least 80 tools
		assert.GreaterOrEqual(t, len(tools), expectedMin,
			"Expected at least %d tools, got %d", expectedMin, len(tools))
	})
}

func TestToolTimeouts(t *testing.T) {
	t.Run("QueryTool has custom timeout", func(t *testing.T) {
		tool := NewQueryTool(nil, nil)
		assert.Equal(t, DefaultQueryTimeout, tool.DefaultTimeout(),
			"QueryTool should have 60s timeout")
	})

	t.Run("InvestigateIncidentTool has workflow timeout", func(t *testing.T) {
		tool := NewInvestigateIncidentTool(nil, nil)
		assert.Equal(t, DefaultWorkflowTimeout, tool.DefaultTimeout(),
			"InvestigateIncidentTool should have 90s timeout")
	})

	t.Run("HealthCheckTool has health check timeout", func(t *testing.T) {
		tool := NewHealthCheckTool(nil, nil)
		assert.Equal(t, DefaultHealthCheckTimeout, tool.DefaultTimeout(),
			"HealthCheckTool should have 45s timeout")
	})

	t.Run("BaseTool has zero timeout (use default)", func(t *testing.T) {
		tool := NewBaseTool(nil, nil)
		assert.Equal(t, time.Duration(0), tool.DefaultTimeout(),
			"BaseTool should return 0 (use client default)")
	})

	t.Run("ListAlertsTool inherits BaseTool timeout", func(t *testing.T) {
		tool := NewListAlertsTool(nil, nil)
		assert.Equal(t, time.Duration(0), tool.DefaultTimeout(),
			"ListAlertsTool should inherit 0 timeout from BaseTool")
	})
}

func TestTimeoutConstants(t *testing.T) {
	t.Run("timeout values are sensible", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, DefaultListTimeout)
		assert.Equal(t, 15*time.Second, DefaultGetTimeout)
		assert.Equal(t, 30*time.Second, DefaultCreateTimeout)
		assert.Equal(t, 15*time.Second, DefaultDeleteTimeout)
		assert.Equal(t, 90*time.Second, DefaultWorkflowTimeout)
		assert.Equal(t, 45*time.Second, DefaultHealthCheckTimeout)
		assert.Equal(t, 60*time.Second, DefaultQueryTimeout)
	})

	t.Run("workflow timeout is longer than query timeout", func(t *testing.T) {
		assert.Greater(t, DefaultWorkflowTimeout, DefaultQueryTimeout,
			"Workflow timeout should be longer than query timeout")
	})
}

func TestRegistryToolCategories(t *testing.T) {
	tools := GetAllTools(nil, nil)

	// Map tool names to verify we have expected categories
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	t.Run("has alert tools", func(t *testing.T) {
		require.True(t, toolNames["get_alert"])
		require.True(t, toolNames["list_alerts"])
		require.True(t, toolNames["create_alert"])
	})

	t.Run("has query tools", func(t *testing.T) {
		require.True(t, toolNames["query_logs"])
		require.True(t, toolNames["build_query"])
	})

	t.Run("has workflow tools", func(t *testing.T) {
		require.True(t, toolNames["investigate_incident"])
		require.True(t, toolNames["health_check"])
	})

	t.Run("has discovery tools", func(t *testing.T) {
		require.True(t, toolNames["discover_tools"])
		require.True(t, toolNames["search_tools"])
		require.True(t, toolNames["describe_tools"])
	})
}
