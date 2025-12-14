package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchToolsTool(t *testing.T) {
	tool := NewSearchToolsTool(nil, nil)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "search_tools", tool.Name())
	})

	t.Run("Description is concise", func(t *testing.T) {
		desc := tool.Description()
		assert.Less(t, len(desc), 200, "Description should be under 200 chars for token efficiency")
	})

	t.Run("InputSchema valid", func(t *testing.T) {
		schema := tool.InputSchema()
		assert.NotNil(t, schema)
	})

	t.Run("Execute without params returns error", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("Execute with query returns brief results", func(t *testing.T) {
		// First register some tools
		RegisterToolForDynamic(&mockTool{name: "query_logs", desc: "Execute queries against IBM Cloud Logs."})
		RegisterToolForDynamic(&mockTool{name: "list_alerts", desc: "List all alerts configured in the system."})

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "error",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Verify response contains brief info, not full schemas
		text := result.Content[0].(*mcp.TextContent).Text
		var response map[string]interface{}
		err = json.Unmarshal([]byte(text), &response)
		require.NoError(t, err)

		tools, ok := response["tools"].([]interface{})
		assert.True(t, ok)
		// Should have hint about describe_tools
		hint, ok := response["hint"].(string)
		assert.True(t, ok)
		assert.Contains(t, hint, "describe_tools")

		// Brief results should not include full inputSchema
		if len(tools) > 0 {
			firstTool := tools[0].(map[string]interface{})
			_, hasInputSchema := firstTool["input_schema"]
			assert.False(t, hasInputSchema, "Brief results should not include input_schema")
		}
	})

	t.Run("Execute with category filter", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"category": "alert",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
	})

	t.Run("Execute with limit", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "logs",
			"limit": 5,
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		text := result.Content[0].(*mcp.TextContent).Text
		var response map[string]interface{}
		err = json.Unmarshal([]byte(text), &response)
		require.NoError(t, err)

		showing, _ := response["showing"].(float64)
		assert.LessOrEqual(t, int(showing), 5)
	})
}

func TestDescribeToolsTool(t *testing.T) {
	tool := NewDescribeToolsTool(nil, nil)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "describe_tools", tool.Name())
	})

	t.Run("Description is concise", func(t *testing.T) {
		desc := tool.Description()
		assert.Less(t, len(desc), 200, "Description should be under 200 chars")
	})

	t.Run("Execute without names returns error", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("Execute with empty names returns error", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"names": []interface{}{},
		})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("Execute with valid names returns full schemas", func(t *testing.T) {
		// Register a mock tool
		mockDesc := "This is a detailed description with examples and documentation."
		RegisterToolForDynamic(&mockTool{
			name: "test_tool",
			desc: mockDesc,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{"type": "string"},
				},
			},
		})

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"names": []interface{}{"test_tool"},
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		text := result.Content[0].(*mcp.TextContent).Text
		var response map[string]interface{}
		err = json.Unmarshal([]byte(text), &response)
		require.NoError(t, err)

		tools, ok := response["tools"].(map[string]interface{})
		assert.True(t, ok)

		testTool, ok := tools["test_tool"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, mockDesc, testTool["description"])

		// Full schemas should include input_schema
		_, hasInputSchema := testTool["input_schema"]
		assert.True(t, hasInputSchema, "Full schemas should include input_schema")
	})

	t.Run("Execute with unknown tool returns not_found", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"names": []interface{}{"nonexistent_tool"},
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		text := result.Content[0].(*mcp.TextContent).Text
		var response map[string]interface{}
		err = json.Unmarshal([]byte(text), &response)
		require.NoError(t, err)

		notFound, ok := response["not_found"].([]interface{})
		assert.True(t, ok)
		assert.Contains(t, notFound, "nonexistent_tool")
	})

	t.Run("Execute limits to 5 tools", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"names": []interface{}{"t1", "t2", "t3", "t4", "t5", "t6", "t7"},
		})
		require.NoError(t, err)
		// Should not error, just process first 5
		assert.False(t, result.IsError)
	})
}

func TestListToolCategoriesBrief(t *testing.T) {
	tool := NewListToolCategoriesBrief(nil, nil)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "list_tool_categories", tool.Name())
	})

	t.Run("Execute returns categories with counts", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		text := result.Content[0].(*mcp.TextContent).Text
		var response map[string]interface{}
		err = json.Unmarshal([]byte(text), &response)
		require.NoError(t, err)

		categories, ok := response["categories"].(map[string]interface{})
		assert.True(t, ok)
		assert.Greater(t, len(categories), 0)

		// Check namespaces are included (new feature)
		namespaces, ok := response["namespaces"].(map[string]interface{})
		assert.True(t, ok)
		assert.Greater(t, len(namespaces), 0)

		// Check usage hints are included with new keys
		usage, ok := response["usage"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, usage, "by_namespace")
		assert.Contains(t, usage, "by_intent")
		assert.Contains(t, usage, "get_details")
	})
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "Short description",
			maxLen:   100,
			expected: "Short description",
		},
		{
			name:     "long string truncated at word boundary",
			input:    "This is a very long description that should be truncated at a word boundary",
			maxLen:   30,
			expected: "This is a very long...",
		},
		{
			name:     "markdown removed",
			input:    "**Bold** and `code` text",
			maxLen:   100,
			expected: "Bold and code text",
		},
		{
			name:     "multiline takes first line",
			input:    "First line\nSecond line\nThird line",
			maxLen:   100,
			expected: "First line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDescription(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDynamicToolRegistry(t *testing.T) {
	t.Run("RegisterToolForDynamic adds tool", func(t *testing.T) {
		mock := &mockTool{name: "registry_test_tool", desc: "Test"}
		RegisterToolForDynamic(mock)

		retrieved := GetRegisteredTool("registry_test_tool")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "registry_test_tool", retrieved.Name())
	})

	t.Run("GetRegisteredTool returns nil for unknown", func(t *testing.T) {
		retrieved := GetRegisteredTool("definitely_not_registered")
		assert.Nil(t, retrieved)
	})

	t.Run("GetAllToolNames returns sorted list", func(t *testing.T) {
		names := GetAllToolNames()
		assert.Greater(t, len(names), 0)

		// Verify sorted
		for i := 1; i < len(names); i++ {
			assert.LessOrEqual(t, names[i-1], names[i], "Names should be sorted")
		}
	})
}

// mockTool implements Tool interface for testing
type mockTool struct {
	name   string
	desc   string
	schema interface{}
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.desc }
func (m *mockTool) InputSchema() interface{} {
	if m.schema != nil {
		return m.schema
	}
	return map[string]interface{}{"type": "object"}
}
func (m *mockTool) Execute(_ context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "mock result"}},
	}, nil
}
func (m *mockTool) Annotations() *mcp.ToolAnnotations { return nil }
func (m *mockTool) DefaultTimeout() time.Duration     { return 0 }
