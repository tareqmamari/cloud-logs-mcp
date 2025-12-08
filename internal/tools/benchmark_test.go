package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// testLogger creates a no-op logger for tests
func testLogger() *zap.Logger {
	return zap.NewNop()
}

// BenchmarkFormatResponse benchmarks response formatting
func BenchmarkFormatResponse(b *testing.B) {
	result := generateTestEvents(100)

	bt := &BaseTool{logger: testLogger()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bt.FormatResponse(result)
	}
}

// BenchmarkFormatResponseLarge benchmarks response formatting with large data
func BenchmarkFormatResponseLarge(b *testing.B) {
	result := generateTestEvents(1000)

	bt := &BaseTool{logger: testLogger()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bt.FormatResponse(result)
	}
}

// BenchmarkParseSSEResponseLarge benchmarks SSE parsing with large data
func BenchmarkParseSSEResponseLarge(b *testing.B) {
	sseData := generateTestSSEData(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseSSEResponse(sseData)
	}
}

// BenchmarkAnalyzeQueryResults benchmarks query result analysis
func BenchmarkAnalyzeQueryResults(b *testing.B) {
	result := generateTestEvents(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AnalyzeQueryResults(result)
	}
}

// BenchmarkGetProactiveSuggestions benchmarks suggestion generation
func BenchmarkGetProactiveSuggestions(b *testing.B) {
	result := generateTestEvents(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetProactiveSuggestions("query_logs", result, false)
	}
}

// BenchmarkValidateDataPrimeQuery benchmarks query validation
func BenchmarkValidateDataPrimeQuery(b *testing.B) {
	query := "source logs | filter $l.applicationname == 'myapp' && $m.severity >= 5 | groupby $l.subsystemname calculate count() as errors | sortby -errors | limit 20"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateDataPrimeQuery(query)
	}
}

// BenchmarkExplainDataPrimeQuery benchmarks query explanation
func BenchmarkExplainDataPrimeQuery(b *testing.B) {
	query := "source logs | filter $l.applicationname == 'myapp' && $m.severity >= 5 | groupby $l.subsystemname calculate count() as errors | sortby -errors | limit 20"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = explainDataPrimeQuery(query)
	}
}

// generateTestEvents generates test event data
func generateTestEvents(count int) map[string]interface{} {
	events := make([]interface{}, count)
	apps := []string{"payment-service", "user-service", "api-gateway", "auth-service", "notification-service"}
	subsystems := []string{"api", "worker", "scheduler", "db", "cache"}
	severities := []float64{1, 2, 3, 4, 5, 6}

	for i := 0; i < count; i++ {
		events[i] = map[string]interface{}{
			"timestamp":       "2024-01-15T10:30:00Z",
			"applicationname": apps[i%len(apps)],
			"subsystemname":   subsystems[i%len(subsystems)],
			"severity":        severities[i%len(severities)],
			"message":         "Test log message " + string(rune(i)),
			"labels": map[string]interface{}{
				"applicationname": apps[i%len(apps)],
				"subsystemname":   subsystems[i%len(subsystems)],
			},
			"metadata": map[string]interface{}{
				"severity":  severities[i%len(severities)],
				"timestamp": "2024-01-15T10:30:00Z",
			},
		}
	}

	return map[string]interface{}{
		"events": events,
	}
}

// generateTestSSEData generates test SSE data
func generateTestSSEData(count int) []byte {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		event := map[string]interface{}{
			"timestamp":       "2024-01-15T10:30:00Z",
			"applicationname": "test-app",
			"severity":        3,
			"message":         "Test message",
		}
		data, _ := json.Marshal(event)
		builder.WriteString("data: ")
		builder.Write(data)
		builder.WriteString("\n\n")
	}
	return []byte(builder.String())
}

// Edge Case Tests

func TestEmptyResultsHandling(t *testing.T) {
	bt := &BaseTool{logger: testLogger()}

	// Test nil result
	result, err := bt.FormatResponse(nil)
	if err != nil {
		t.Errorf("FormatResponse(nil) returned error: %v", err)
	}
	if result == nil {
		t.Error("FormatResponse(nil) returned nil result")
	}

	// Test empty map
	result, err = bt.FormatResponse(map[string]interface{}{})
	if err != nil {
		t.Errorf("FormatResponse({}) returned error: %v", err)
	}
	if result == nil {
		t.Error("FormatResponse({}) returned nil result")
	}

	// Test empty events array
	result, err = bt.FormatResponse(map[string]interface{}{
		"events": []interface{}{},
	})
	if err != nil {
		t.Errorf("FormatResponse(empty events) returned error: %v", err)
	}
	if result == nil {
		t.Error("FormatResponse(empty events) returned nil result")
	}
}

func TestLargeResultTruncation(t *testing.T) {
	// Generate result larger than MaxResultSize
	largeResult := generateTestEvents(2000)

	bt := &BaseTool{logger: testLogger()}
	result, err := bt.FormatResponse(largeResult)
	if err != nil {
		t.Errorf("FormatResponse(large) returned error: %v", err)
	}
	if result == nil {
		t.Error("FormatResponse(large) returned nil result")
	}

	// Check that result is truncated
	if len(result.Content) == 0 {
		t.Error("Result has no content")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Error("Result content is not TextContent")
	}
	if len(textContent.Text) > FinalResponseLimit {
		t.Errorf("Result exceeds FinalResponseLimit: %d > %d", len(textContent.Text), FinalResponseLimit)
	}
}

func TestSSEParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		minCount int
	}{
		{
			name:    "empty input",
			input:   "",
			wantNil: true,
		},
		{
			name:    "no data prefix",
			input:   "event: test\n\n",
			wantNil: true,
		},
		{
			name:     "single event",
			input:    "data: {\"test\": \"value\"}\n\n",
			wantNil:  false,
			minCount: 1,
		},
		{
			name:    "invalid json",
			input:   "data: not json\n\n",
			wantNil: true,
		},
		{
			name:     "mixed valid and invalid",
			input:    "data: {\"a\": 1}\ndata: invalid\ndata: {\"b\": 2}\n",
			wantNil:  false,
			minCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSSEResponse([]byte(tt.input))
			if tt.wantNil && result != nil {
				t.Errorf("Expected nil, got result with %v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected result, got nil")
			}
			if !tt.wantNil && result != nil {
				events, ok := result["events"].([]interface{})
				if !ok {
					t.Error("Result missing events array")
				}
				if len(events) < tt.minCount {
					t.Errorf("Expected at least %d events, got %d", tt.minCount, len(events))
				}
			}
		})
	}
}

func TestQueryTemplateSubstitution(t *testing.T) {
	template := QueryTemplate{
		Query: "source logs | filter $l.applicationname == '{APPLICATION}'",
	}

	// Test substitution
	result := substituteTemplateParams(template, "my-app", "")
	if !strings.Contains(result.Query, "my-app") {
		t.Error("Application not substituted")
	}
	if strings.Contains(result.Query, "{APPLICATION}") {
		t.Error("Placeholder not replaced")
	}
}

func TestAnalyzeQueryResultsEmpty(t *testing.T) {
	// Test with empty events
	analysis := AnalyzeQueryResults(map[string]interface{}{
		"events": []interface{}{},
	})

	if analysis == nil {
		t.Fatal("Expected analysis, got nil")
	}
	if analysis.Summary == "" {
		t.Error("Expected summary for empty results")
	}
	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations for empty results")
	}
}

func TestAnalyzeQueryResultsWithErrors(t *testing.T) {
	// Generate results with high error rate
	result := map[string]interface{}{
		"events": []interface{}{
			map[string]interface{}{"severity": float64(5), "applicationname": "app1"},
			map[string]interface{}{"severity": float64(5), "applicationname": "app1"},
			map[string]interface{}{"severity": float64(6), "applicationname": "app1"},
			map[string]interface{}{"severity": float64(3), "applicationname": "app2"},
		},
	}

	analysis := AnalyzeQueryResults(result)

	if analysis == nil {
		t.Fatal("Expected analysis, got nil")
	}
	if analysis.Statistics == nil {
		t.Fatal("Expected statistics")
	}
	if analysis.Statistics.ErrorRate == 0 {
		t.Error("Expected non-zero error rate")
	}
}

func TestValidateDataPrimeQueryCases(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid query",
			query:   "source logs | filter $l.applicationname == 'myapp'",
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
		{
			name:    "query with typo",
			query:   "source logs | filter $l.applicationame == 'myapp'",
			wantErr: true,
		},
		{
			name:    "query with wrong quotes",
			query:   "source logs | filter $l.applicationname == \"myapp\"",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateDataPrimeQuery(tt.query)
			if tt.wantErr && result.Valid {
				t.Error("Expected invalid, got valid")
			}
			if !tt.wantErr && !result.Valid {
				t.Errorf("Expected valid, got errors: %v", result.Errors)
			}
		})
	}
}

func TestGetQueryTemplates(t *testing.T) {
	templates := getQueryTemplates()

	if len(templates) == 0 {
		t.Fatal("Expected templates, got none")
	}

	// Check each template has required fields
	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("Template missing name")
		}
		if tmpl.Category == "" {
			t.Errorf("Template %s missing category", tmpl.Name)
		}
		if tmpl.Query == "" {
			t.Errorf("Template %s missing query", tmpl.Name)
		}
		if len(tmpl.UseCases) == 0 {
			t.Errorf("Template %s missing use cases", tmpl.Name)
		}
	}

	// Check categories are valid
	validCategories := map[string]bool{
		"error": true, "performance": true, "security": true,
		"health": true, "usage": true, "audit": true,
	}
	for _, tmpl := range templates {
		if !validCategories[tmpl.Category] {
			t.Errorf("Template %s has invalid category: %s", tmpl.Name, tmpl.Category)
		}
	}
}

// Concurrent access test
func TestConcurrentResponseFormatting(t *testing.T) {
	result := generateTestEvents(100)
	bt := &BaseTool{logger: testLogger()}

	// Run multiple goroutines formatting concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, err := bt.FormatResponse(result)
				if err != nil {
					t.Errorf("Concurrent FormatResponse error: %v", err)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
