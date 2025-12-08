// Package prompts provides pre-built prompts for common IBM Cloud Logs operations.
package prompts

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func TestNewRegistry(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}

	prompts := registry.GetPrompts()
	if len(prompts) == 0 {
		t.Error("Expected prompts to be registered")
	}
}

func TestGetPrompts(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	prompts := registry.GetPrompts()

	// Verify expected number of prompts
	expectedCount := 7
	if len(prompts) != expectedCount {
		t.Errorf("Expected %d prompts, got %d", expectedCount, len(prompts))
	}

	// Verify all prompts have required fields
	for _, p := range prompts {
		if p.Prompt == nil {
			t.Error("Prompt definition is nil")
			continue
		}
		if p.Prompt.Name == "" {
			t.Error("Prompt name is empty")
		}
		if p.Prompt.Description == "" {
			t.Errorf("Prompt %s has empty description", p.Prompt.Name)
		}
		if p.Handler == nil {
			t.Errorf("Prompt %s has nil handler", p.Prompt.Name)
		}
	}
}

func TestPromptNames(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	expectedNames := map[string]bool{
		"investigate_errors":        true,
		"setup_monitoring":          true,
		"compare_environments":      true,
		"debugging_workflow":        true,
		"optimize_retention":        true,
		"test_log_ingestion":        true,
		"create_dashboard_workflow": true,
	}

	prompts := registry.GetPrompts()
	for _, p := range prompts {
		if _, ok := expectedNames[p.Prompt.Name]; !ok {
			t.Errorf("Unexpected prompt name: %s", p.Prompt.Name)
		}
		delete(expectedNames, p.Prompt.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected prompt: %s", name)
	}
}

func TestInvestigateErrorsPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "investigate_errors" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("investigate_errors prompt not found")
	}

	tests := []struct {
		name          string
		args          map[string]string
		wantInContent string
	}{
		{
			name:          "default time range",
			args:          nil,
			wantInContent: "last 1h",
		},
		{
			name:          "custom time range",
			args:          map[string]string{"time_range": "24h"},
			wantInContent: "last 24h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			result, err := prompt.Handler(context.Background(), req)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Handler returned nil result")
			}

			if len(result.Messages) == 0 {
				t.Fatal("Result has no messages")
			}

			content, ok := result.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatal("Message content is not TextContent")
			}

			if content.Text == "" {
				t.Error("Content text is empty")
			}

			if tt.wantInContent != "" && !containsString(content.Text, tt.wantInContent) {
				t.Errorf("Content does not contain expected string %q", tt.wantInContent)
			}
		})
	}
}

func TestSetupMonitoringPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "setup_monitoring" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("setup_monitoring prompt not found")
	}

	tests := []struct {
		name          string
		args          map[string]string
		wantInContent string
	}{
		{
			name:          "default service name",
			args:          nil,
			wantInContent: "your-service",
		},
		{
			name:          "custom service name",
			args:          map[string]string{"service_name": "my-api"},
			wantInContent: "my-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			result, err := prompt.Handler(context.Background(), req)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			content, ok := result.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatal("Message content is not TextContent")
			}

			if !containsString(content.Text, tt.wantInContent) {
				t.Errorf("Content does not contain expected string %q", tt.wantInContent)
			}
		})
	}
}

func TestCompareEnvironmentsPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "compare_environments" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("compare_environments prompt not found")
	}

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{"time_range": "6h"},
		},
	}

	result, err := prompt.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Message content is not TextContent")
	}

	expectedStrings := []string{"production", "staging", "6h"}
	for _, s := range expectedStrings {
		if !containsString(content.Text, s) {
			t.Errorf("Content does not contain expected string %q", s)
		}
	}
}

func TestDebuggingWorkflowPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "debugging_workflow" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("debugging_workflow prompt not found")
	}

	tests := []struct {
		name          string
		args          map[string]string
		wantInContent string
	}{
		{
			name:          "default error message",
			args:          nil,
			wantInContent: "your error message",
		},
		{
			name:          "custom error message",
			args:          map[string]string{"error_message": "NullPointerException"},
			wantInContent: "NullPointerException",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			result, err := prompt.Handler(context.Background(), req)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			content, ok := result.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatal("Message content is not TextContent")
			}

			if !containsString(content.Text, tt.wantInContent) {
				t.Errorf("Content does not contain expected string %q", tt.wantInContent)
			}
		})
	}
}

func TestOptimizeRetentionPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "optimize_retention" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("optimize_retention prompt not found")
	}

	// This prompt has no arguments
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: nil,
		},
	}

	result, err := prompt.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if result.Description == "" {
		t.Error("Result description is empty")
	}

	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Message content is not TextContent")
	}

	// Verify it mentions key concepts
	expectedStrings := []string{"retention", "E2M", "policies", "cost"}
	for _, s := range expectedStrings {
		if !containsString(content.Text, s) {
			t.Errorf("Content does not contain expected string %q", s)
		}
	}
}

func TestTestLogIngestionPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "test_log_ingestion" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("test_log_ingestion prompt not found")
	}

	tests := []struct {
		name          string
		args          map[string]string
		wantInContent string
	}{
		{
			name:          "default application name",
			args:          nil,
			wantInContent: "test-app",
		},
		{
			name:          "custom application name",
			args:          map[string]string{"application_name": "my-service"},
			wantInContent: "my-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			result, err := prompt.Handler(context.Background(), req)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			content, ok := result.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatal("Message content is not TextContent")
			}

			if !containsString(content.Text, tt.wantInContent) {
				t.Errorf("Content does not contain expected string %q", tt.wantInContent)
			}

			// Verify ingestion-specific content
			if !containsString(content.Text, "severity") {
				t.Error("Content does not mention severity")
			}
			if !containsString(content.Text, "ingress") {
				t.Error("Content does not mention ingress endpoint")
			}
		})
	}
}

func TestCreateDashboardWorkflowPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "create_dashboard_workflow" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("create_dashboard_workflow prompt not found")
	}

	tests := []struct {
		name          string
		args          map[string]string
		wantInContent string
	}{
		{
			name:          "default dashboard name",
			args:          nil,
			wantInContent: "Custom Dashboard",
		},
		{
			name:          "custom dashboard name",
			args:          map[string]string{"dashboard_name": "Production Metrics"},
			wantInContent: "Production Metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			result, err := prompt.Handler(context.Background(), req)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			content, ok := result.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatal("Message content is not TextContent")
			}

			if !containsString(content.Text, tt.wantInContent) {
				t.Errorf("Content does not contain expected string %q", tt.wantInContent)
			}

			// Verify dashboard-specific content
			widgetTypes := []string{"line_chart", "bar_chart", "data_table"}
			for _, wt := range widgetTypes {
				if !containsString(content.Text, wt) {
					t.Errorf("Content does not mention widget type %q", wt)
				}
			}
		})
	}
}

func TestGetStringArg(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]string
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "key exists with value",
			args:       map[string]string{"foo": "bar"},
			key:        "foo",
			defaultVal: "default",
			want:       "bar",
		},
		{
			name:       "key does not exist",
			args:       map[string]string{"other": "value"},
			key:        "foo",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "key exists but empty",
			args:       map[string]string{"foo": ""},
			key:        "foo",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "nil args",
			args:       nil,
			key:        "foo",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringArg(tt.args, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getStringArg() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreatePromptResult(t *testing.T) {
	description := "Test description"
	content := "Test content"

	result := createPromptResult(description, content)

	if result.Description != description {
		t.Errorf("Description = %q, want %q", result.Description, description)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result.Messages))
	}

	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}

	textContent, ok := msg.Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Content is not TextContent")
	}

	if textContent.Text != content {
		t.Errorf("Text = %q, want %q", textContent.Text, content)
	}
}

func TestPromptArgumentsDefinition(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	expectedArgs := map[string][]string{
		"investigate_errors":        {"time_range"},
		"setup_monitoring":          {"service_name"},
		"compare_environments":      {"time_range"},
		"debugging_workflow":        {"error_message"},
		"optimize_retention":        {},
		"test_log_ingestion":        {"application_name"},
		"create_dashboard_workflow": {"dashboard_name"},
	}

	for _, p := range registry.GetPrompts() {
		expected, ok := expectedArgs[p.Prompt.Name]
		if !ok {
			t.Errorf("Unexpected prompt: %s", p.Prompt.Name)
			continue
		}

		if len(p.Prompt.Arguments) != len(expected) {
			t.Errorf("Prompt %s: expected %d arguments, got %d",
				p.Prompt.Name, len(expected), len(p.Prompt.Arguments))
			continue
		}

		for i, argName := range expected {
			if p.Prompt.Arguments[i].Name != argName {
				t.Errorf("Prompt %s: argument %d expected name %q, got %q",
					p.Prompt.Name, i, argName, p.Prompt.Arguments[i].Name)
			}
		}
	}
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
