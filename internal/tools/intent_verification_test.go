package tools

import (
	"strings"
	"testing"
)

// intentTestContainsSubstr is a helper function for tests
func intentTestContainsSubstr(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestVerifyIntent_Empty(t *testing.T) {
	result := VerifyIntent("")

	if result.IntentType != IntentUnknown {
		t.Errorf("Expected IntentUnknown for empty intent, got %s", result.IntentType)
	}
	if len(result.ClarifyingQuestions) == 0 {
		t.Error("Expected clarifying questions for empty intent")
	}
}

func TestVerifyIntent_QueryIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"search for recent logs", IntentQuery},
		{"find all timeout messages", IntentQuery},
		{"list logs from api-gateway", IntentQuery},
		{"filter logs by severity", IntentQuery},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestVerifyIntent_InvestigateIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"investigate the error spike", IntentInvestigate},
		{"debug the failing service", IntentInvestigate},
		{"troubleshoot the issue", IntentInvestigate},
		{"find root cause of the problem", IntentInvestigate},
		{"why is the api failing", IntentInvestigate},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestVerifyIntent_MonitorIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"create an alert for this pattern", IntentMonitor},
		{"set up monitoring for the service", IntentMonitor},
		{"notify me when this happens", IntentMonitor},
		{"track this metric with alerts", IntentMonitor},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestVerifyIntent_VisualizeIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"create a dashboard", IntentVisualize},
		{"visualize the data trends", IntentVisualize},
		{"make a chart of requests", IntentVisualize},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestVerifyIntent_LearnIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"how do I use this tool", IntentLearn},
		{"explain the syntax", IntentLearn},
		{"help me understand dataprime", IntentLearn},
		{"what is the dataprime format", IntentLearn},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestVerifyIntent_ExploreIntent(t *testing.T) {
	tests := []struct {
		intent   string
		expected IntentType
	}{
		{"what can I do with this tool", IntentExplore},
		{"show available capabilities", IntentExplore},
		{"what features are available", IntentExplore},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result := VerifyIntent(tt.intent)
			if result.IntentType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.IntentType)
			}
		})
	}
}

func TestExtractEntities_Services(t *testing.T) {
	tests := []struct {
		intent           string
		expectedServices []string
	}{
		{"errors in api-gateway", []string{"api-gateway"}},
		{"payment-service is failing", []string{"payment-service"}},
		{"check user-api and order-service", []string{"user-api", "order-service"}},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			entities := extractEntities(tt.intent)
			if len(entities.Services) != len(tt.expectedServices) {
				t.Errorf("Expected %d services, got %d: %v",
					len(tt.expectedServices), len(entities.Services), entities.Services)
			}
		})
	}
}

func TestExtractEntities_TimeRange(t *testing.T) {
	tests := []struct {
		intent       string
		expectedTime string
	}{
		{"errors in the last hour", "1h"},
		{"logs from last 24 hours", "24h"},
		{"what happened today", "today"},
		{"check yesterday's logs", "yesterday"},
		{"last week errors", "7d"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			entities := extractEntities(tt.intent)
			if entities.TimeRange != tt.expectedTime {
				t.Errorf("Expected TimeRange=%s, got %s", tt.expectedTime, entities.TimeRange)
			}
		})
	}
}

func TestExtractEntities_Severity(t *testing.T) {
	tests := []struct {
		intent           string
		expectedSeverity string
	}{
		{"show error logs", "ERROR"},
		{"critical alerts", "CRITICAL"},
		{"warning messages", "WARNING"},
		{"debug output", "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			entities := extractEntities(tt.intent)
			if entities.Severity != tt.expectedSeverity {
				t.Errorf("Expected Severity=%s, got %s", tt.expectedSeverity, entities.Severity)
			}
		})
	}
}

func TestExtractEntities_ErrorType(t *testing.T) {
	tests := []struct {
		intent            string
		expectedErrorType string
	}{
		{"timeout errors in api", "timeout"},
		{"connection failures", "connection"},
		{"authentication problems", "authentication"},
		{"500 errors", "500"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			entities := extractEntities(tt.intent)
			if entities.ErrorType != tt.expectedErrorType {
				t.Errorf("Expected ErrorType=%s, got %s", tt.expectedErrorType, entities.ErrorType)
			}
		})
	}
}

func TestExtractEntities_TraceID(t *testing.T) {
	tests := []struct {
		intent          string
		expectedTraceID string
	}{
		{"trace abc123def456789012", "abc123def456789012"},     // pragma: allowlist secret
		{"find logs for 1234567890abcdef", "1234567890abcdef"}, // pragma: allowlist secret
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			entities := extractEntities(tt.intent)
			if entities.TraceID != tt.expectedTraceID {
				t.Errorf("Expected TraceID=%s, got %s", tt.expectedTraceID, entities.TraceID)
			}
		})
	}
}

func TestExtractEntities_Keywords(t *testing.T) {
	intent := "investigate spike in error rate, find the root cause of slow response"
	entities := extractEntities(intent)

	expectedKeywords := []string{"spike", "root cause", "slow"}
	for _, expected := range expectedKeywords {
		found := false
		for _, kw := range entities.Keywords {
			if kw == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected keyword %q not found in %v", expected, entities.Keywords)
		}
	}
}

func TestDetectAmbiguities(t *testing.T) {
	tests := []struct {
		intent            string
		expectAmbiguity   bool
		ambiguityContains string
	}{
		{"search logs and create alert", true, "searching logs OR setting up alerts"},
		{"find errors and make dashboard", true, "querying data OR creating visualizations"},
		{"search logs from api-service", true, "No time range"},         // No time mentioned
		{"search errors in my service last hour", true, "not specific"}, // Vague service
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			intentType, _ := classifyIntent(tt.intent)
			ambiguities := detectAmbiguities(tt.intent, intentType)

			if tt.expectAmbiguity && len(ambiguities) == 0 {
				t.Error("Expected ambiguity but none detected")
			}

			if tt.ambiguityContains != "" {
				found := false
				for _, a := range ambiguities {
					if intentTestContainsSubstr(a, tt.ambiguityContains) {
						found = true
						break
					}
				}
				if !found && tt.expectAmbiguity {
					t.Errorf("Expected ambiguity containing %q, got %v", tt.ambiguityContains, ambiguities)
				}
			}
		})
	}
}

func TestGenerateClarifyingQuestions(t *testing.T) {
	// Test with ambiguous intent
	v := &IntentVerification{
		IntentType:  IntentInvestigate,
		Ambiguities: []string{"No time range specified"},
		ExtractedEntities: &IntentEntities{
			Services: []string{},
		},
	}

	questions := generateClarifyingQuestions(v)

	if len(questions) == 0 {
		t.Error("Expected clarifying questions")
	}

	// Should ask about time range
	hasTimeQuestion := false
	for _, q := range questions {
		if intentTestContainsSubstr(q, "time") {
			hasTimeQuestion = true
			break
		}
	}
	if !hasTimeQuestion {
		t.Error("Expected question about time range")
	}
}

func TestGenerateAlternatives(t *testing.T) {
	tests := []struct {
		intentType  IntentType
		expectCount int
	}{
		{IntentQuery, 2},
		{IntentInvestigate, 2},
		{IntentMonitor, 2},
		{IntentVisualize, 2},
	}

	for _, tt := range tests {
		t.Run(string(tt.intentType), func(t *testing.T) {
			alternatives := generateAlternatives("test intent", tt.intentType)
			if len(alternatives) < tt.expectCount {
				t.Errorf("Expected at least %d alternatives, got %d", tt.expectCount, len(alternatives))
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"ABC123", true},
		{"0123456789abcdef", true},
		{"hello", false},
		{"abc-123", false},
		{"abc 123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isHexString(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVerifyIntent_Confidence(t *testing.T) {
	// High confidence: clear intent
	highConfidence := VerifyIntent("investigate the error spike in api-gateway")
	if highConfidence.Confidence < 0.5 {
		t.Errorf("Expected high confidence for clear intent, got %f", highConfidence.Confidence)
	}

	// Lower confidence: ambiguous intent
	lowConfidence := VerifyIntent("do something")
	if lowConfidence.Confidence > 0.5 {
		t.Errorf("Expected low confidence for vague intent, got %f", lowConfidence.Confidence)
	}
}

func TestGenerateParsedIntent(t *testing.T) {
	entities := &IntentEntities{
		Services:  []string{"api-gateway"},
		Severity:  "ERROR",
		TimeRange: "1h",
	}

	parsed := generateParsedIntent(IntentInvestigate, entities)

	if parsed == "" {
		t.Error("Expected non-empty parsed intent")
	}
	if !intentTestContainsSubstr(parsed, "Investigate") {
		t.Error("Expected parsed intent to contain action")
	}
	if !intentTestContainsSubstr(parsed, "api-gateway") {
		t.Error("Expected parsed intent to contain service name")
	}
}
