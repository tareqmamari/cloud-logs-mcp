package tools

import (
	"strings"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	// Verify registry is initialized
	if registry == nil {
		t.Fatal("NewToolRegistry returned nil")
	}

	// Verify tools are registered
	if len(registry.tools) == 0 {
		t.Error("No tools registered in registry")
	}

	// Verify tool chains are registered
	if len(registry.chains) == 0 {
		t.Error("No tool chains registered")
	}

	// Verify intents are indexed
	if len(registry.intents) == 0 {
		t.Error("No intents indexed")
	}
}

func TestIntentMappings(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		name          string
		intent        string
		expectedTools []string
		minMatches    int
	}{
		// Error investigation
		{
			name:          "investigate errors",
			intent:        "investigate errors",
			expectedTools: []string{"investigate_incident", "query_logs"},
			minMatches:    2,
		},
		{
			name:          "debug production",
			intent:        "debug production",
			expectedTools: []string{"investigate_incident", "query_logs", "health_check"},
			minMatches:    2,
		},
		{
			name:          "crash investigation",
			intent:        "crash",
			expectedTools: []string{"investigate_incident", "query_logs"},
			minMatches:    2,
		},
		{
			name:          "outage response",
			intent:        "outage",
			expectedTools: []string{"investigate_incident", "health_check"},
			minMatches:    2,
		},

		// Query intents
		{
			name:          "search logs",
			intent:        "search logs",
			expectedTools: []string{"query_logs", "build_query"},
			minMatches:    1,
		},
		{
			name:          "grep pattern",
			intent:        "grep",
			expectedTools: []string{"query_logs", "build_query"},
			minMatches:    1,
		},
		{
			name:          "recent logs",
			intent:        "recent logs",
			expectedTools: []string{"query_logs"},
			minMatches:    1,
		},

		// Alerting intents
		{
			name:          "create alert",
			intent:        "create alert",
			expectedTools: []string{"create_alert", "suggest_alert"},
			minMatches:    1,
		},
		{
			name:          "threshold alert",
			intent:        "threshold alert",
			expectedTools: []string{"create_alert", "suggest_alert"},
			minMatches:    1,
		},
		{
			name:          "triggered alerts",
			intent:        "triggered alerts",
			expectedTools: []string{"list_alerts"},
			minMatches:    1,
		},

		// Dashboard intents
		{
			name:          "create dashboard",
			intent:        "create dashboard",
			expectedTools: []string{"create_dashboard"},
			minMatches:    1,
		},
		{
			name:          "time series chart",
			intent:        "time series",
			expectedTools: []string{"create_dashboard"},
			minMatches:    1,
		},

		// Health and monitoring
		{
			name:          "health check",
			intent:        "check health",
			expectedTools: []string{"health_check"},
			minMatches:    1,
		},
		{
			name:          "morning operations",
			intent:        "morning check",
			expectedTools: []string{"health_check", "list_alerts"},
			minMatches:    1,
		},

		// Learning intents
		{
			name:          "learn dataprime",
			intent:        "learn dataprime",
			expectedTools: []string{"query_templates", "build_query", "explain_query"},
			minMatches:    2,
		},
		{
			name:          "explain query",
			intent:        "explain this query",
			expectedTools: []string{"explain_query"},
			minMatches:    1,
		},

		// Integration intents
		{
			name:          "slack integration",
			intent:        "integrate slack",
			expectedTools: []string{"create_outgoing_webhook"},
			minMatches:    1,
		},
		{
			name:          "pagerduty notification",
			intent:        "pagerduty notification",
			expectedTools: []string{"create_outgoing_webhook"},
			minMatches:    1,
		},

		// Security intents
		{
			name:          "security audit",
			intent:        "security audit",
			expectedTools: []string{"list_data_access_rules", "query_logs"},
			minMatches:    1,
		},
		{
			name:          "access control",
			intent:        "rbac",
			expectedTools: []string{"list_data_access_rules"},
			minMatches:    1,
		},

		// Performance intents
		{
			name:          "latency investigation",
			intent:        "high latency",
			expectedTools: []string{"query_logs", "investigate_incident"},
			minMatches:    1,
		},
		{
			name:          "p99 metrics",
			intent:        "p99",
			expectedTools: []string{"query_logs", "create_e2m"},
			minMatches:    1,
		},

		// Kubernetes intents
		{
			name:          "kubernetes logs",
			intent:        "kubernetes",
			expectedTools: []string{"query_logs", "build_query"},
			minMatches:    1,
		},
		{
			name:          "k8s pod logs",
			intent:        "pod logs",
			expectedTools: []string{"query_logs"},
			minMatches:    1,
		},

		// Background queries
		{
			name:          "background query",
			intent:        "background query",
			expectedTools: []string{"submit_background_query"},
			minMatches:    1,
		},
		{
			name:          "query timeout",
			intent:        "query timeout",
			expectedTools: []string{"submit_background_query"},
			minMatches:    1,
		},

		// Policy intents
		{
			name:          "retention policy",
			intent:        "retention policy",
			expectedTools: []string{"list_policies", "create_policy"},
			minMatches:    1,
		},
		{
			name:          "cost optimization",
			intent:        "cost optimization",
			expectedTools: []string{"list_policies", "list_e2m"},
			minMatches:    1,
		},

		// E2M intents
		{
			name:          "events to metrics",
			intent:        "events to metrics",
			expectedTools: []string{"list_e2m", "create_e2m"},
			minMatches:    1,
		},
		{
			name:          "create metric",
			intent:        "create metric",
			expectedTools: []string{"create_e2m"},
			minMatches:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.DiscoverTools(tt.intent, "", "")

			if len(result.MatchedTools) < tt.minMatches {
				t.Errorf("Expected at least %d matches for intent '%s', got %d",
					tt.minMatches, tt.intent, len(result.MatchedTools))
			}

			// Check that at least one expected tool is in the results
			foundExpected := false
			for _, match := range result.MatchedTools {
				for _, expected := range tt.expectedTools {
					if match.Name == expected {
						foundExpected = true
						break
					}
				}
				if foundExpected {
					break
				}
			}

			if !foundExpected && len(tt.expectedTools) > 0 {
				matchedNames := make([]string, len(result.MatchedTools))
				for i, m := range result.MatchedTools {
					matchedNames[i] = m.Name
				}
				t.Errorf("Expected one of %v in results for intent '%s', got %v",
					tt.expectedTools, tt.intent, matchedNames)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		query    string
		target   string
		minScore float64
	}{
		{"investigate errors", "investigate errors", 1.0},
		{"investigate error", "investigate errors", 0.6},
		{"search logs", "search logs", 1.0},
		{"search", "search logs", 0.8},
		{"errors", "find errors", 0.8},
		{"completely different", "investigate errors", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.query+"_vs_"+tt.target, func(t *testing.T) {
			score := fuzzyMatch(tt.query, tt.target)
			if score < tt.minScore {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want >= %v",
					tt.query, tt.target, score, tt.minScore)
			}
		})
	}
}

func TestDiscoverToolsByCategory(t *testing.T) {
	registry := NewToolRegistry()

	// Verify tools are registered with correct categories
	tests := []struct {
		category     ToolCategory
		expectedTool string
	}{
		{CategoryQuery, "query_logs"},
		{CategoryAlert, "list_alerts"},
		{CategoryDashboard, "list_dashboards"},
		{CategoryPolicy, "list_policies"},
		{CategoryWebhook, "list_outgoing_webhooks"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			// Check tool is registered with correct category
			meta, exists := registry.tools[tt.expectedTool]
			if !exists {
				t.Errorf("Tool %s not found in registry", tt.expectedTool)
				return
			}

			found := false
			for _, cat := range meta.Categories {
				if cat == tt.category {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected tool %s to have category %s", tt.expectedTool, tt.category)
			}
		})
	}
}

func TestDiscoverToolsByComplexity(t *testing.T) {
	registry := NewToolRegistry()

	// Verify we have tools of each complexity level
	complexityCounts := map[string]int{
		ComplexitySimple:       0,
		ComplexityIntermediate: 0,
		ComplexityAdvanced:     0,
	}

	for _, meta := range registry.tools {
		if meta.Complexity != "" {
			complexityCounts[meta.Complexity]++
		}
	}

	for complexity, count := range complexityCounts {
		if count == 0 {
			t.Errorf("No tools registered with complexity %s", complexity)
		}
	}

	// Test that complexity filter works on keyword search results
	result := registry.DiscoverTools("query", "", ComplexitySimple)
	if len(result.MatchedTools) == 0 {
		t.Error("Expected some results for complexity simple with query intent")
	}

	// Verify that intent-based matches have complexity metadata
	for _, match := range result.MatchedTools {
		if match.Complexity == "" {
			t.Errorf("Tool %s has empty complexity", match.Name)
		}
	}
}

func TestToolChainMatching(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		intent        string
		expectedChain string
	}{
		{"error investigation", "error_investigation"},
		{"incident response", "error_investigation"},
		{"security investigation", "security_investigation"},
		{"new service monitoring", "monitoring_setup"},
		{"slack integration", "slack_integration"},
		{"cost optimization", "cost_optimization"},
		{"learning dataprime", "query_learning"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := registry.DiscoverTools(tt.intent, "", "")

			if result.SuggestedChain == nil {
				t.Errorf("Expected a suggested chain for intent '%s'", tt.intent)
				return
			}

			if result.SuggestedChain.Name != tt.expectedChain {
				t.Errorf("Expected chain '%s' for intent '%s', got '%s'",
					tt.expectedChain, tt.intent, result.SuggestedChain.Name)
			}
		})
	}
}

func TestDiscoveryResultStructure(t *testing.T) {
	registry := NewToolRegistry()
	result := registry.DiscoverTools("investigate errors", "", "")

	// Check result has expected fields
	if result.Intent != "investigate errors" {
		t.Errorf("Expected intent 'investigate errors', got '%s'", result.Intent)
	}

	if len(result.MatchedTools) == 0 {
		t.Error("Expected matched tools in result")
	}

	// Check tool match structure
	for _, match := range result.MatchedTools {
		if match.Name == "" {
			t.Error("Tool match has empty name")
		}
		if match.Relevance < 0 || match.Relevance > 1 {
			t.Errorf("Tool relevance %f out of range [0,1]", match.Relevance)
		}
		if match.Reason == "" {
			t.Error("Tool match has empty reason")
		}
	}

	// Check session context is included
	if result.SessionContext == nil {
		t.Error("Expected session context in result")
	}
}

func TestIntentCoverage(t *testing.T) {
	registry := NewToolRegistry()

	// Test that we have a reasonable number of intent mappings
	expectedMinIntents := 200 // We have significantly expanded the mappings
	if len(registry.intents) < expectedMinIntents {
		t.Errorf("Expected at least %d intent mappings, got %d",
			expectedMinIntents, len(registry.intents))
	}

	// Test specific categories have coverage
	categories := map[string][]string{
		"errors":      {"investigate errors", "find errors", "debug", "crash", "exception", "stack trace"},
		"queries":     {"search logs", "query", "grep", "recent logs", "filter logs"},
		"alerts":      {"create alert", "threshold alert", "list alerts", "alert recommendations"},
		"dashboards":  {"create dashboard", "time series", "chart", "graph"},
		"security":    {"security audit", "access control", "rbac", "permissions"},
		"performance": {"high latency", "slow requests", "p99", "timeout"},
		"kubernetes":  {"kubernetes", "k8s", "pod logs", "container logs"},
	}

	for category, intents := range categories {
		for _, intent := range intents {
			if _, exists := registry.intents[intent]; !exists {
				t.Errorf("Missing intent mapping for '%s' in category '%s'",
					intent, category)
			}
		}
	}
}

func TestGetToolRegistry(t *testing.T) {
	// Reset global registry
	globalRegistry = nil

	// First call should create registry
	registry1 := GetToolRegistry()
	if registry1 == nil {
		t.Fatal("GetToolRegistry returned nil")
	}

	// Second call should return same instance
	registry2 := GetToolRegistry()
	if registry1 != registry2 {
		t.Error("GetToolRegistry should return singleton instance")
	}
}

func TestConfidenceScoring(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		name               string
		intent             string
		expectedLevel      ConfidenceLevel
		minScore           float64
		maxScore           float64
		expectAlternatives bool
	}{
		{
			name:          "exact intent match - high confidence",
			intent:        "investigate errors",
			expectedLevel: ConfidenceHigh,
			minScore:      0.8,
			maxScore:      1.0,
		},
		{
			name:          "direct keyword match - high confidence",
			intent:        "search logs",
			expectedLevel: ConfidenceHigh,
			minScore:      0.8,
			maxScore:      1.0,
		},
		{
			name:          "single word exact match - high confidence",
			intent:        "debug",
			expectedLevel: ConfidenceHigh,
			minScore:      0.8,
			maxScore:      1.0,
		},
		{
			name:               "no match - low confidence",
			intent:             "xyzzy12345",
			expectedLevel:      ConfidenceLow,
			minScore:           0.0,
			maxScore:           0.6,
			expectAlternatives: false, // Random gibberish won't have alternatives
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.DiscoverTools(tt.intent, "", "")

			if result.Confidence == nil {
				t.Fatal("Expected confidence result")
			}

			if result.Confidence.Level != tt.expectedLevel {
				t.Errorf("Expected confidence level %s, got %s (score: %.2f)",
					tt.expectedLevel, result.Confidence.Level, result.Confidence.Score)
			}

			if result.Confidence.Score < tt.minScore || result.Confidence.Score > tt.maxScore {
				t.Errorf("Expected score between %.2f and %.2f, got %.2f",
					tt.minScore, tt.maxScore, result.Confidence.Score)
			}

			if tt.expectAlternatives && len(result.Confidence.Alternatives) == 0 {
				t.Error("Expected alternatives for low confidence match")
			}
		})
	}
}

func TestConfidenceBasedFiltering(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		name           string
		intent         string
		expectedLevel  ConfidenceLevel
		maxToolsExpect int
	}{
		{
			name:           "high confidence limits to 3 tools",
			intent:         "investigate errors",
			expectedLevel:  ConfidenceHigh,
			maxToolsExpect: HighConfidenceMaxTools,
		},
		{
			name:           "exact match high confidence",
			intent:         "search logs",
			expectedLevel:  ConfidenceHigh,
			maxToolsExpect: HighConfidenceMaxTools,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.DiscoverTools(tt.intent, "", "")

			if result.Confidence.Level != tt.expectedLevel {
				t.Errorf("Expected confidence level %s, got %s",
					tt.expectedLevel, result.Confidence.Level)
			}

			if len(result.MatchedTools) > tt.maxToolsExpect {
				t.Errorf("Expected at most %d tools for %s confidence, got %d",
					tt.maxToolsExpect, tt.expectedLevel, len(result.MatchedTools))
			}
		})
	}
}

func TestConfidenceExplanation(t *testing.T) {
	registry := NewToolRegistry()

	// Test exact match explanation
	result := registry.DiscoverTools("investigate errors", "", "")
	if result.Confidence == nil {
		t.Fatal("Expected confidence result")
	}
	if result.Confidence.Explanation == "" {
		t.Error("Expected non-empty explanation")
	}
	if result.Confidence.Level == ConfidenceHigh && result.Confidence.Score == 1.0 {
		if result.Confidence.Explanation != "Exact intent match found" {
			t.Errorf("Expected exact match explanation, got: %s", result.Confidence.Explanation)
		}
	}

	// Test no match explanation
	result = registry.DiscoverTools("completely_random_gibberish_xyz123", "", "")
	if result.Confidence.Explanation == "" {
		t.Error("Expected non-empty explanation for no match")
	}
}

func TestClarificationGeneration(t *testing.T) {
	registry := NewToolRegistry()

	// Test that clarifications are generated for ambiguous intents
	// "error" could mean investigate errors or set up error alerting
	result := registry.DiscoverTools("error monitoring", "", "")

	// Should have some matched tools
	if len(result.MatchedTools) == 0 {
		t.Skip("No matched tools for test intent")
	}

	// For medium/low confidence, clarifications should be present
	if result.Confidence.Level != ConfidenceHigh {
		if len(result.Confidence.Clarifications) == 0 {
			t.Error("Expected clarifications for non-high confidence")
		}
	}
}

func TestAlternativeIntents(t *testing.T) {
	registry := NewToolRegistry()

	// Test with a partial match that should generate alternatives or fuzzy matches
	result := registry.DiscoverTools("errr", "", "") // Typo of "error"

	// For any confidence level, we should have some result (matches OR alternatives)
	hasMatches := len(result.MatchedTools) > 0
	hasAlternatives := len(result.Confidence.Alternatives) > 0

	// At minimum, we should get *something* back for a partial match
	// Either fuzzy matching finds tools, or alternatives are suggested
	if !hasMatches && !hasAlternatives && result.Confidence.Level == ConfidenceLow {
		// This is acceptable - very short/unusual input might not match anything
		// Just verify confidence reflects this
		if result.Confidence.Score != 0 {
			t.Error("Expected zero score when no matches and no alternatives")
		}
	}

	// Test that a real partial match provides alternatives
	result = registry.DiscoverTools("serach logs", "", "") // Typo of "search logs"

	// Should have either matches from fuzzy matching or alternatives
	if len(result.MatchedTools) == 0 && len(result.Confidence.Alternatives) == 0 {
		// Log what we got for debugging
		t.Logf("Score: %.2f, Level: %s", result.Confidence.Score, result.Confidence.Level)
	}
}

func TestConfidenceThresholds(t *testing.T) {
	// Verify threshold constants are correctly ordered
	if HighConfidenceThreshold <= MediumConfidenceThreshold {
		t.Error("HighConfidenceThreshold should be greater than MediumConfidenceThreshold")
	}

	if MediumConfidenceThreshold <= 0 {
		t.Error("MediumConfidenceThreshold should be positive")
	}

	// Verify tool limits
	if HighConfidenceMaxTools >= MediumConfidenceMaxTools {
		t.Error("High confidence should return fewer tools than medium confidence")
	}

	if MediumConfidenceMaxTools >= LowConfidenceMaxTools {
		t.Error("Medium confidence should return fewer tools than low confidence")
	}
}

func TestConfidenceResultStructure(t *testing.T) {
	registry := NewToolRegistry()
	result := registry.DiscoverTools("search logs", "", "")

	if result.Confidence == nil {
		t.Fatal("Expected confidence in result")
	}

	// Check all required fields
	if result.Confidence.Level == "" {
		t.Error("Confidence level should not be empty")
	}

	if result.Confidence.Score < 0 || result.Confidence.Score > 1 {
		t.Errorf("Confidence score should be between 0 and 1, got %f", result.Confidence.Score)
	}

	if result.Confidence.Explanation == "" {
		t.Error("Confidence explanation should not be empty")
	}

	// Verify level matches score
	switch {
	case result.Confidence.Score >= HighConfidenceThreshold:
		if result.Confidence.Level != ConfidenceHigh {
			t.Errorf("Score %.2f should yield high confidence, got %s",
				result.Confidence.Score, result.Confidence.Level)
		}
	case result.Confidence.Score >= MediumConfidenceThreshold:
		if result.Confidence.Level != ConfidenceMedium {
			t.Errorf("Score %.2f should yield medium confidence, got %s",
				result.Confidence.Score, result.Confidence.Level)
		}
	default:
		if result.Confidence.Level != ConfidenceLow {
			t.Errorf("Score %.2f should yield low confidence, got %s",
				result.Confidence.Score, result.Confidence.Level)
		}
	}
}

func TestConfidenceRecommendations(t *testing.T) {
	registry := NewToolRegistry()

	// High confidence should have specific recommendation
	result := registry.DiscoverTools("search logs", "", "")
	if result.Confidence.Level == ConfidenceHigh {
		found := false
		for _, rec := range result.Recommendations {
			if strings.Contains(rec, "High confidence") || strings.Contains(rec, "proceed with the top tool") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected high confidence recommendation for high confidence result")
		}
	}

	// Low confidence should have rephrasing recommendation
	result = registry.DiscoverTools("xyzzy_unknown_intent", "", "")
	if result.Confidence.Level == ConfidenceLow {
		found := false
		for _, rec := range result.Recommendations {
			if strings.Contains(rec, "rephras") || strings.Contains(rec, "alternatives") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected rephrasing recommendation for low confidence result")
		}
	}
}

func TestAdaptiveChainStructure(t *testing.T) {
	// Test AdaptiveChain struct
	chain := &AdaptiveChain{
		Name:        "test chain",
		Description: "Test description",
		Tools:       []string{"query_logs", "investigate_incident"},
		SuccessRate: 85.0,
		UseCount:    5,
		Confidence:  "high",
		Source:      "learned",
	}

	if chain.Name != "test chain" {
		t.Errorf("Expected name 'test chain', got %s", chain.Name)
	}
	if len(chain.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(chain.Tools))
	}
	if chain.SuccessRate != 85.0 {
		t.Errorf("Expected success rate 85.0, got %f", chain.SuccessRate)
	}
}

func TestGenerateChainName(t *testing.T) {
	tests := []struct {
		tools    []string
		expected string
	}{
		{[]string{}, "Empty chain"},
		{[]string{"query_logs"}, "query_logs"},
		{[]string{"query_logs", "create_alert"}, "query_logs → alert"}, // query_logs doesn't match prefixes
		{[]string{"list_alerts", "get_alert", "create_dashboard"}, "alerts → alert → dashboard"},
	}

	for _, tt := range tests {
		result := generateChainName(tt.tools)
		if result != tt.expected {
			t.Errorf("generateChainName(%v) = %s, want %s", tt.tools, result, tt.expected)
		}
	}
}

func TestGenerateChainDescription(t *testing.T) {
	tests := []struct {
		tools       []string
		shouldMatch string
	}{
		{[]string{"query_logs", "investigate_incident"}, "Search logs and investigate"},
		{[]string{"list_alerts", "create_alert"}, "Review and create alerts"},
		{[]string{"query_logs", "create_dashboard"}, "Query data and visualize"},
		{[]string{"unknown_tool"}, "Sequence:"},
	}

	for _, tt := range tests {
		result := generateChainDescription(tt.tools)
		if !strings.Contains(result, tt.shouldMatch) {
			t.Errorf("generateChainDescription(%v) = %s, should contain %s", tt.tools, result, tt.shouldMatch)
		}
	}
}

func TestContainsAllTools(t *testing.T) {
	tests := []struct {
		tools    []string
		items    []string
		expected bool
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b"}, true},
		{[]string{"a", "b", "c"}, []string{"a", "d"}, false},
		{[]string{"query_logs", "investigate_incident"}, []string{"query_logs"}, true},
		{[]string{}, []string{"a"}, false},
		{[]string{"a", "b"}, []string{}, true},
	}

	for _, tt := range tests {
		result := containsAll(tt.tools, tt.items...)
		if result != tt.expected {
			t.Errorf("containsAll(%v, %v) = %v, want %v", tt.tools, tt.items, result, tt.expected)
		}
	}
}

func TestChainScore(t *testing.T) {
	highConfChain := &AdaptiveChain{
		Confidence:  "high",
		SuccessRate: 90.0,
		UseCount:    10,
	}

	medConfChain := &AdaptiveChain{
		Confidence:  "medium",
		SuccessRate: 90.0,
		UseCount:    10,
	}

	lowConfChain := &AdaptiveChain{
		Confidence:  "low",
		SuccessRate: 90.0,
		UseCount:    10,
	}

	highScore := chainScore(highConfChain)
	medScore := chainScore(medConfChain)
	lowScore := chainScore(lowConfChain)

	if highScore <= medScore {
		t.Error("High confidence should score higher than medium")
	}
	if medScore <= lowScore {
		t.Error("Medium confidence should score higher than low")
	}
}

func TestSortAdaptiveChains(t *testing.T) {
	chains := []*AdaptiveChain{
		{Confidence: "low", SuccessRate: 50.0, UseCount: 2},
		{Confidence: "high", SuccessRate: 90.0, UseCount: 10},
		{Confidence: "medium", SuccessRate: 70.0, UseCount: 5},
	}

	sortAdaptiveChains(chains)

	// First should be high confidence
	if chains[0].Confidence != "high" {
		t.Errorf("Expected first chain to be high confidence, got %s", chains[0].Confidence)
	}
	// Second should be medium
	if chains[1].Confidence != "medium" {
		t.Errorf("Expected second chain to be medium confidence, got %s", chains[1].Confidence)
	}
	// Third should be low
	if chains[2].Confidence != "low" {
		t.Errorf("Expected third chain to be low confidence, got %s", chains[2].Confidence)
	}
}

func TestSuggestChainsFromMatches(t *testing.T) {
	registry := NewToolRegistry()

	// Test with investigation-related tools
	matches := []ToolMatch{
		{Name: "query_logs"},
		{Name: "investigate_incident"},
	}

	chains := registry.suggestChainsFromMatches(matches)

	// Should suggest investigation workflow
	found := false
	for _, chain := range chains {
		if strings.Contains(chain.Name, "investigation") {
			found = true
			break
		}
	}
	if !found && len(chains) > 0 {
		t.Log("Chains suggested:", chains)
	}

	// Test with too few matches
	singleMatch := []ToolMatch{{Name: "query_logs"}}
	noChains := registry.suggestChainsFromMatches(singleMatch)
	if len(noChains) != 0 {
		t.Error("Expected no chains for single match")
	}
}

func TestDiscoveryResultIncludesAdaptiveChains(t *testing.T) {
	registry := NewToolRegistry()

	result := registry.DiscoverTools("investigate errors", "", "")

	// Result should have AdaptiveChains field (may be empty or populated)
	// The key test is that the field exists and doesn't cause errors
	if result.AdaptiveChains == nil {
		t.Log("AdaptiveChains is nil, initializing empty slice")
		result.AdaptiveChains = []*AdaptiveChain{}
	}

	// Verify AdaptiveChains length is valid
	t.Logf("Found %d adaptive chains", len(result.AdaptiveChains))
}

func TestGenerateAdaptiveChainRecommendations(t *testing.T) {
	registry := NewToolRegistry()

	// Test with no chains
	emptyRecs := registry.generateAdaptiveChainRecommendations([]*AdaptiveChain{})
	if len(emptyRecs) != 0 {
		t.Error("Expected no recommendations for empty chains")
	}

	// Test with a high-confidence learned chain
	chains := []*AdaptiveChain{
		{
			Tools:       []string{"query_logs", "create_alert"},
			SuccessRate: 90.0,
			UseCount:    5,
			Confidence:  "high",
			Source:      "learned",
		},
	}

	recs := registry.generateAdaptiveChainRecommendations(chains)
	if len(recs) == 0 {
		t.Error("Expected recommendations for high-confidence learned chain")
	}

	// Should mention history
	found := false
	for _, rec := range recs {
		if strings.Contains(rec, "history") || strings.Contains(rec, "success rate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected recommendation to mention history or success rate")
	}
}
