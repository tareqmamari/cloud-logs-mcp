package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func TestBuildQueryTool_Name(t *testing.T) {
	tool := NewBuildQueryTool(nil, zap.NewNop())
	if tool.Name() != "build_query" {
		t.Errorf("Expected name 'build_query', got '%s'", tool.Name())
	}
}

func TestBuildQueryTool_Execute(t *testing.T) {
	tool := NewBuildQueryTool(nil, zap.NewNop())

	tests := []struct {
		name         string
		args         map[string]interface{}
		wantInLucene []string
		wantInDP     []string
		notWantIn    []string
	}{
		{
			name: "text search",
			args: map[string]interface{}{
				"text_search": "connection timeout",
			},
			wantInLucene: []string{`"connection timeout"`},
			wantInDP:     []string{"$d.text ~~ 'connection timeout'"},
		},
		{
			name: "single application filter",
			args: map[string]interface{}{
				"applications": []interface{}{"api-gateway"},
			},
			wantInLucene: []string{"applicationname:api-gateway"},
			wantInDP:     []string{"$l.applicationname == 'api-gateway'"},
		},
		{
			name: "multiple applications",
			args: map[string]interface{}{
				"applications": []interface{}{"api-gateway", "auth-service"},
			},
			wantInLucene: []string{"applicationname:api-gateway", "applicationname:auth-service", "OR"},
			wantInDP:     []string{"$l.applicationname == 'api-gateway'", "$l.applicationname == 'auth-service'", "||"},
		},
		{
			name: "min severity",
			args: map[string]interface{}{
				"min_severity": "error",
			},
			wantInLucene: []string{"severity:>=5"},
			wantInDP:     []string{"$m.severity >= 5"},
		},
		{
			name: "specific severities",
			args: map[string]interface{}{
				"severities": []interface{}{"warning", "error"},
			},
			wantInLucene: []string{"severity:4", "severity:5", "OR"},
			wantInDP:     []string{"$m.severity == 4", "$m.severity == 5", "||"},
		},
		{
			name: "field equals filter",
			args: map[string]interface{}{
				"fields": []interface{}{
					map[string]interface{}{
						"field":    "json.status_code",
						"operator": "equals",
						"value":    "500",
					},
				},
			},
			wantInLucene: []string{"json.status_code:500"},
			wantInDP:     []string{"$d.status_code == '500'"},
		},
		{
			name: "field exists filter",
			args: map[string]interface{}{
				"fields": []interface{}{
					map[string]interface{}{
						"field":    "json.error_code",
						"operator": "exists",
					},
				},
			},
			wantInLucene: []string{"json.error_code:*"},
			wantInDP:     []string{"$d.error_code != null"},
		},
		{
			name: "exclude text",
			args: map[string]interface{}{
				"text_search":  "error",
				"exclude_text": "health check",
			},
			wantInLucene: []string{"error", `NOT "health check"`},
			wantInDP:     []string{"$d.text ~~ 'error'", "$d.text !~~ 'health check'"},
		},
		{
			name: "combined filters",
			args: map[string]interface{}{
				"text_search":  "failed",
				"applications": []interface{}{"payment-service"},
				"min_severity": "warning",
			},
			wantInLucene: []string{"failed", "applicationname:payment-service", "severity:>=4"},
			wantInDP:     []string{"$d.text ~~ 'failed'", "$l.applicationname == 'payment-service'", "$m.severity >= 4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}

			if result == nil || len(result.Content) == 0 {
				t.Fatal("Result is nil or empty")
			}

			// Get the text content
			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("Result content is not mcp.TextContent")
			}

			text := textContent.Text

			// Check Lucene query contains expected parts
			for _, want := range tt.wantInLucene {
				if !strings.Contains(text, want) {
					t.Errorf("Expected Lucene query to contain '%s', but got:\n%s", want, text)
				}
			}

			// Check DataPrime query contains expected parts
			for _, want := range tt.wantInDP {
				if !strings.Contains(text, want) {
					t.Errorf("Expected DataPrime query to contain '%s', but got:\n%s", want, text)
				}
			}

			// Check for things that shouldn't be there
			for _, notWant := range tt.notWantIn {
				if strings.Contains(text, notWant) {
					t.Errorf("Expected result to NOT contain '%s', but it did", notWant)
				}
			}
		})
	}
}

func TestBuildQueryTool_EmptyArgs(t *testing.T) {
	tool := NewBuildQueryTool(nil, zap.NewNop())

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Result content is not mcp.TextContent")
	}

	// Should contain help text
	if !strings.Contains(textContent.Text, "No filters specified") {
		t.Error("Expected help text for empty args")
	}
}

func TestSeverityToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"debug", 1},
		{"verbose", 2},
		{"info", 3},
		{"warning", 4},
		{"warn", 4},
		{"error", 5},
		{"critical", 6},
		{"fatal", 6},
		{"unknown", 0},
		{"DEBUG", 1}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := severityToInt(tt.input)
			if got != tt.expected {
				t.Errorf("severityToInt(%s) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToDataPrimeField(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"json.status_code", "$d.status_code"},
		{"applicationname", "$l.applicationname"},
		{"severity", "$m.severity"},
		{"custom_field", "$d.custom_field"},
		{"$d.already_prefixed", "$d.already_prefixed"},
		{"$l.label_field", "$l.label_field"},
		{"$m.meta_field", "$m.meta_field"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toDataPrimeField(tt.input)
			if got != tt.expected {
				t.Errorf("toDataPrimeField(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}
