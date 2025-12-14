package tools

import (
	"testing"
	"time"
)

func TestScoutLogsTool_Name(t *testing.T) {
	tool := NewScoutLogsTool(nil, nil)
	if tool.Name() != "scout_logs" {
		t.Errorf("Expected name 'scout_logs', got '%s'", tool.Name())
	}
}

func TestScoutLogsTool_Annotations(t *testing.T) {
	tool := NewScoutLogsTool(nil, nil)
	ann := tool.Annotations()
	if ann == nil {
		t.Error("Expected non-nil annotations")
		return
	}
	if !ann.ReadOnlyHint {
		t.Error("Expected ReadOnlyHint to be true")
	}
}

func TestBuildNoiseExclusionFilter(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		want     string
	}{
		{
			name:     "empty patterns",
			patterns: []string{},
			want:     "",
		},
		{
			name:     "single pattern",
			patterns: []string{"health"},
			want:     "(!$d.message:string.toLowerCase().contains('health'))",
		},
		{
			name:     "multiple patterns",
			patterns: []string{"health", "ping"},
			want:     "(!$d.message:string.toLowerCase().contains('health') && !$d.message:string.toLowerCase().contains('ping'))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNoiseExclusionFilter(tt.patterns)
			if got != tt.want {
				t.Errorf("buildNoiseExclusionFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScoutLogsTool_TierParameter(t *testing.T) {
	tool := NewScoutLogsTool(nil, nil)
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})

	// Check tier property exists
	tierProp, ok := props["tier"].(map[string]interface{})
	if !ok {
		t.Error("tier property should exist in schema")
		return
	}

	// Verify default is archive (logs always land there unless TCO policy excludes)
	if tierProp["default"] != "archive" {
		t.Errorf("Expected tier default to be 'archive', got %v", tierProp["default"])
	}

	// Verify enum values include archive and frequent_search
	enum, ok := tierProp["enum"].([]string)
	if !ok {
		t.Error("tier enum should be []string")
		return
	}
	if len(enum) != 2 || enum[0] != "archive" || enum[1] != "frequent_search" {
		t.Errorf("Expected tier enum to be [archive, frequent_search], got %v", enum)
	}
}

func TestCalculateTimeRange(t *testing.T) {
	tests := []struct {
		name           string
		timeRange      string
		expectedOffset time.Duration
	}{
		{"15m", "15m", 15 * time.Minute},
		{"1h", "1h", 1 * time.Hour},
		{"6h", "6h", 6 * time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"default", "invalid", 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := calculateTimeRange(tt.timeRange)
			if start == "" || end == "" {
				t.Error("Expected non-empty start and end times")
				return
			}

			// Parse the times to verify format
			startTime, err := time.Parse(time.RFC3339, start)
			if err != nil {
				t.Errorf("Failed to parse start time: %v", err)
				return
			}

			endTime, err := time.Parse(time.RFC3339, end)
			if err != nil {
				t.Errorf("Failed to parse end time: %v", err)
				return
			}

			// Verify the duration is approximately correct (within 1 second tolerance)
			actualDuration := endTime.Sub(startTime)
			diff := actualDuration - tt.expectedOffset
			if diff < -time.Second || diff > time.Second {
				t.Errorf("Duration mismatch: got %v, expected %v", actualDuration, tt.expectedOffset)
			}
		})
	}
}

func TestFormatModeName(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"error_hotspots", "Error Hotspots"},
		{"severity_distribution", "Severity Distribution"},
		{"top_error_messages", "Top Error Messages"},
		{"anomaly_scan", "Anomaly Scan"},
		{"traffic_overview", "Traffic Overview"},
		{"recent_deployments", "Recent Deployments"},
		{"unknown_mode", "unknown_mode"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := formatModeName(tt.mode)
			if got != tt.expected {
				t.Errorf("formatModeName(%q) = %q, want %q", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestDefaultNoisePatterns(t *testing.T) {
	// Verify we have the expected default patterns
	if len(defaultNoisePatterns) == 0 {
		t.Error("Expected non-empty default noise patterns")
	}

	// Check for expected patterns
	expectedPatterns := []string{"health", "healthcheck", "ping", "metrics", "heartbeat"}
	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range defaultNoisePatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pattern %q not found in default noise patterns", expected)
		}
	}
}

func TestScoutLogsTool_BuildDiscoveryQuery(t *testing.T) {
	tool := NewScoutLogsTool(nil, nil)

	tests := []struct {
		name             string
		mode             string
		excludeNoise     bool
		customExclusions []string
		application      string
		minCount         int
		topN             int
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:         "error_hotspots basic",
			mode:         "error_hotspots",
			excludeNoise: false,
			minCount:     1,
			topN:         20,
			wantContains: []string{
				"source logs",
				"$m.severity >= ERROR",
				"groupby $l.applicationname",
				"orderby error_count desc",
			},
		},
		{
			name:         "error_hotspots with noise exclusion",
			mode:         "error_hotspots",
			excludeNoise: true,
			minCount:     1,
			topN:         20,
			wantContains: []string{
				"source logs",
				"$m.severity >= ERROR",
				"!$d.message:string.toLowerCase().contains('health')",
			},
		},
		{
			name:         "severity_distribution",
			mode:         "severity_distribution",
			excludeNoise: false,
			minCount:     1,
			topN:         20,
			wantContains: []string{
				"groupby $l.applicationname, $m.severity",
				"count() as log_count",
			},
		},
		{
			name:         "anomaly_scan",
			mode:         "anomaly_scan",
			excludeNoise: false,
			minCount:     5,
			topN:         10,
			wantContains: []string{
				"error_rate",
				"countif($m.severity >= ERROR)",
				"error_rate > 1",
			},
		},
		{
			name:             "with custom exclusions",
			mode:             "error_hotspots",
			excludeNoise:     false,
			customExclusions: []string{"debug", "test"},
			minCount:         1,
			topN:             20,
			wantContains: []string{
				"!$d.message:string.toLowerCase().contains('debug')",
				"!$d.message:string.toLowerCase().contains('test')",
			},
		},
		{
			name:         "with application filter",
			mode:         "error_hotspots",
			excludeNoise: false,
			application:  "my-app",
			minCount:     1,
			topN:         20,
			wantContains: []string{
				"$l.applicationname == 'my-app'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tool.buildDiscoveryQuery(tt.mode, tt.excludeNoise, tt.customExclusions, tt.application, tt.minCount, tt.topN)

			for _, want := range tt.wantContains {
				if !scoutContains(query, want) {
					t.Errorf("Query should contain %q\nGot: %s", want, query)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if scoutContains(query, notWant) {
					t.Errorf("Query should NOT contain %q\nGot: %s", notWant, query)
				}
			}
		})
	}
}

// Helper function for string contains check
func scoutContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && scoutContainsHelper(s, substr))
}

func scoutContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetStringFromMap(t *testing.T) {
	m := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}

	if got := getStringFromMap(m, "name", "default"); got != "test" {
		t.Errorf("Expected 'test', got '%s'", got)
	}

	if got := getStringFromMap(m, "missing", "default"); got != "default" {
		t.Errorf("Expected 'default', got '%s'", got)
	}

	if got := getStringFromMap(m, "count", "default"); got != "default" {
		t.Errorf("Expected 'default' for non-string, got '%s'", got)
	}
}

func TestGetNumberFromMap(t *testing.T) {
	m := map[string]interface{}{
		"float": 3.14,
		"int":   42,
		"str":   "not a number",
	}

	if got := getNumberFromMap(m, "float", 0); got != 3.14 {
		t.Errorf("Expected 3.14, got %v", got)
	}

	if got := getNumberFromMap(m, "int", 0); got != 42.0 {
		t.Errorf("Expected 42.0, got %v", got)
	}

	if got := getNumberFromMap(m, "missing", -1); got != -1 {
		t.Errorf("Expected -1, got %v", got)
	}

	if got := getNumberFromMap(m, "str", -1); got != -1 {
		t.Errorf("Expected -1 for non-number, got %v", got)
	}
}
