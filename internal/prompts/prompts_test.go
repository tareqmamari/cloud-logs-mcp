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

	// Verify expected number of prompts (11 original + 2 context-aware)
	expectedCount := 13
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
		"continue_investigation":    true,
		"dataprime_tutorial":        true,
		"quick_start":               true,
		"security_audit":            true,
		"context_aware_assist":      true,
		"smart_suggest":             true,
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
		"continue_investigation":    {},
		"dataprime_tutorial":        {"skill_level"},
		"quick_start":               {},
		"security_audit":            {"focus_area"},
		"context_aware_assist":      {},
		"smart_suggest":             {"goal"},
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

func TestContextAwarePromptWithoutProvider(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "context_aware_assist" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("context_aware_assist prompt not found")
	}

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: nil,
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

	// Without a provider, should show general guidance
	expectedStrings := []string{"Context-Aware Assistance", "health_check", "discover_tools"}
	for _, s := range expectedStrings {
		if !containsString(content.Text, s) {
			t.Errorf("Content does not contain expected string %q", s)
		}
	}
}

// MockContextProvider is a mock implementation of SessionContextProvider for testing
type MockContextProvider struct {
	LastQuery          string
	Filters            map[string]string
	RecentToolsList    []RecentToolInfo
	Investigation      *InvestigationInfo
	Preferences        *UserPreferencesInfo
	SuggestedNextTools []string
}

func (m *MockContextProvider) GetLastQuery() string {
	return m.LastQuery
}

func (m *MockContextProvider) GetAllFilters() map[string]string {
	return m.Filters
}

func (m *MockContextProvider) GetRecentTools(limit int) []RecentToolInfo {
	if limit > len(m.RecentToolsList) {
		return m.RecentToolsList
	}
	return m.RecentToolsList[:limit]
}

func (m *MockContextProvider) GetInvestigation() *InvestigationInfo {
	return m.Investigation
}

func (m *MockContextProvider) GetPreferences() *UserPreferencesInfo {
	return m.Preferences
}

func (m *MockContextProvider) GetSuggestedNextTools() []string {
	return m.SuggestedNextTools
}

func TestContextAwarePromptWithProvider(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	// Set up mock context provider
	mockProvider := &MockContextProvider{
		LastQuery: "source logs | filter $d.severity >= 4",
		Filters: map[string]string{
			"application": "my-app",
			"environment": "production",
		},
		Preferences: &UserPreferencesInfo{
			PreferredTimeRange:   "1h",
			FrequentApplications: []string{"my-app", "other-app"},
		},
		SuggestedNextTools: []string{"create_alert", "create_dashboard"},
	}
	registry.SetContextProvider(mockProvider)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "context_aware_assist" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("context_aware_assist prompt not found")
	}

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: nil,
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

	// Should show context-aware content
	expectedStrings := []string{
		"Context-Aware Assistance",
		"Active Filters",
		"my-app",
		"production",
		"Last Query",
		"source logs",
		"Learned Preferences",
		"1h",
	}
	for _, s := range expectedStrings {
		if !containsString(content.Text, s) {
			t.Errorf("Content does not contain expected string %q", s)
		}
	}
}

func TestContextAwarePromptWithActiveInvestigation(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	// Set up mock context provider with active investigation
	mockProvider := &MockContextProvider{
		Investigation: &InvestigationInfo{
			ID:            "20231215-120000",
			Application:   "payment-service",
			Hypothesis:    "Database connection timeout",
			FindingsCount: 3,
			ToolsUsed:     []string{"query_logs", "health_check"},
		},
	}
	registry.SetContextProvider(mockProvider)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "context_aware_assist" {
			prompt = p
			break
		}
	}

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: nil,
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

	// Should show investigation context
	expectedStrings := []string{
		"Active Investigation",
		"20231215-120000",
		"payment-service",
		"Database connection timeout",
		"3 recorded",
	}
	for _, s := range expectedStrings {
		if !containsString(content.Text, s) {
			t.Errorf("Content does not contain expected string %q", s)
		}
	}
}

func TestSmartSuggestPrompt(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "smart_suggest" {
			prompt = p
			break
		}
	}

	if prompt == nil {
		t.Fatal("smart_suggest prompt not found")
	}

	tests := []struct {
		name         string
		goal         string
		wantTools    []string
		wantWorkflow string
	}{
		{
			name:         "error debugging goal",
			goal:         "debug production errors",
			wantTools:    []string{"investigate_incident", "query_logs"},
			wantWorkflow: "error_investigation",
		},
		{
			name:         "alerting goal",
			goal:         "set up alert notifications",
			wantTools:    []string{"suggest_alert", "create_alert"},
			wantWorkflow: "monitoring_setup",
		},
		{
			name:         "dashboard goal",
			goal:         "create visualizations",
			wantTools:    []string{"create_dashboard", "list_dashboards"},
			wantWorkflow: "dashboard_creation",
		},
		{
			name:         "cost optimization goal",
			goal:         "reduce logging costs",
			wantTools:    []string{"list_policies", "list_e2m"},
			wantWorkflow: "cost_optimization",
		},
		{
			name:         "learning goal",
			goal:         "learn dataprime query syntax",
			wantTools:    []string{"query_templates", "build_query"},
			wantWorkflow: "query_learning",
		},
		{
			name:         "security goal",
			goal:         "audit access permissions",
			wantTools:    []string{"list_data_access_rules"},
			wantWorkflow: "security_investigation",
		},
		{
			name:         "unknown goal",
			goal:         "something random",
			wantTools:    []string{"discover_tools", "health_check"},
			wantWorkflow: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: map[string]string{"goal": tt.goal},
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

			// Check for expected tools
			for _, tool := range tt.wantTools {
				if !containsString(content.Text, tool) {
					t.Errorf("Content does not contain expected tool %q", tool)
				}
			}

			// Check for expected workflow
			if tt.wantWorkflow != "" {
				if !containsString(content.Text, tt.wantWorkflow) {
					t.Errorf("Content does not contain expected workflow %q", tt.wantWorkflow)
				}
			}
		})
	}
}

func TestSmartSuggestNoGoal(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	var prompt *PromptDefinition
	for _, p := range registry.GetPrompts() {
		if p.Prompt.Name == "smart_suggest" {
			prompt = p
			break
		}
	}

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{},
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

	if !containsString(content.Text, "provide a goal") {
		t.Error("Expected message about providing a goal")
	}
}

func TestSetContextProvider(t *testing.T) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)

	// Initially nil
	if registry.contextProvider != nil {
		t.Error("Expected nil context provider initially")
	}

	// Set a provider
	mockProvider := &MockContextProvider{}
	registry.SetContextProvider(mockProvider)

	if registry.contextProvider == nil {
		t.Error("Expected non-nil context provider after setting")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		want       bool
	}{
		{"debug production errors", []string{"error", "fail"}, true},
		{"setup monitoring", []string{"monitor", "alert"}, true},
		{"something else", []string{"error", "alert"}, false},
		{"", []string{"error"}, false},
		{"error handling", []string{}, false},
	}

	for _, tt := range tests {
		got := containsAny(tt.s, tt.substrings...)
		if got != tt.want {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrings, got, tt.want)
		}
	}
}
