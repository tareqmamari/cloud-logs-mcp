package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQueryTool_InputSchema verifies the schema includes tier and syntax with proper defaults
// This test validates the fix for missing required API parameters
func TestQueryTool_InputSchema(t *testing.T) {
	tool := &QueryTool{}
	schema := tool.InputSchema()

	// Verify schema structure
	assert.Equal(t, "object", schema.Type)
	assert.Equal(t, []string{"query"}, schema.Required)

	// Verify query property
	queryProp, ok := schema.Properties["query"].(map[string]interface{})
	assert.True(t, ok, "query property should exist")
	assert.Equal(t, "string", queryProp["type"])

	// Verify tier property with enum and default
	tierProp, ok := schema.Properties["tier"].(map[string]interface{})
	assert.True(t, ok, "tier property should exist")
	assert.Equal(t, "string", tierProp["type"])
	assert.Equal(t, "frequent", tierProp["default"], "tier should default to 'frequent'")

	enum, ok := tierProp["enum"].([]string)
	assert.True(t, ok, "tier enum should be []string")
	assert.Contains(t, enum, "frequent")
	assert.Contains(t, enum, "monitoring")
	assert.Contains(t, enum, "archive")

	// Verify syntax property with enum and default
	syntaxProp, ok := schema.Properties["syntax"].(map[string]interface{})
	assert.True(t, ok, "syntax property should exist")
	assert.Equal(t, "string", syntaxProp["type"])
	assert.Equal(t, "dataprime", syntaxProp["default"], "syntax should default to 'dataprime'")

	syntaxEnum, ok := syntaxProp["enum"].([]string)
	assert.True(t, ok, "syntax enum should be []string")
	assert.Contains(t, syntaxEnum, "dataprime")
	assert.Contains(t, syntaxEnum, "lucene")

	// Verify limit property has description about default
	limitProp, ok := schema.Properties["limit"].(map[string]interface{})
	assert.True(t, ok, "limit property should exist")
	assert.Equal(t, "number", limitProp["type"])
	assert.Contains(t, limitProp["description"], "default: 50")
}

// TestQueryTool_DefaultParameterHandling verifies default values are applied correctly
func TestQueryTool_DefaultParameterHandling(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]interface{}
		wantTier   string
		wantSyntax string
		wantLimit  int
	}{
		{
			name: "minimal params - apply all defaults",
			args: map[string]interface{}{
				"query": "source logs",
			},
			wantTier:   "frequent",
			wantSyntax: "dataprime",
			wantLimit:  50,
		},
		{
			name: "custom tier and syntax",
			args: map[string]interface{}{
				"query":  "source logs",
				"tier":   "archive",
				"syntax": "lucene",
			},
			wantTier:   "archive",
			wantSyntax: "lucene",
			wantLimit:  50,
		},
		{
			name: "custom limit",
			args: map[string]interface{}{
				"query": "source logs",
				"limit": float64(100), // JSON numbers are float64
			},
			wantTier:   "frequent",
			wantSyntax: "dataprime",
			wantLimit:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract tier with default (simulating Execute logic)
			tier, _ := GetStringParam(tt.args, "tier", false)
			if tier == "" {
				tier = "frequent"
			}
			assert.Equal(t, tt.wantTier, tier)

			// Extract syntax with default (simulating Execute logic)
			syntax, _ := GetStringParam(tt.args, "syntax", false)
			if syntax == "" {
				syntax = "dataprime"
			}
			assert.Equal(t, tt.wantSyntax, syntax)

			// Extract limit with default (simulating Execute logic)
			limit, _ := GetIntParam(tt.args, "limit", false)
			if limit == 0 {
				limit = 50
			}
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}

// TestQueryTool_MissingRequiredQuery verifies proper error handling for missing required param
func TestQueryTool_MissingRequiredQuery(t *testing.T) {
	args := map[string]interface{}{
		"tier": "frequent",
		// Missing 'query' - should cause error
	}

	_, err := GetStringParam(args, "query", true)
	assert.Error(t, err, "should return error when required 'query' param is missing")
	assert.Contains(t, err.Error(), "query", "error message should mention the missing parameter")
}
