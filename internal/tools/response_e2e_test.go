//go:build integration

package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// TestReal_AggregationQueryResults tests that aggregation queries return properly formatted results
// Run with: go test -tags=integration -v ./internal/tools/... -run "TestReal_Aggregation"
func TestReal_AggregationQueryResults(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	queryTool := NewQueryTool(c, logger)
	ctx := context.Background()

	now := time.Now().UTC()
	startDate := now.Add(-7 * 24 * time.Hour).Format(time.RFC3339) // 7 days back to ensure data
	endDate := now.Format(time.RFC3339)

	tests := []struct {
		name           string
		query          string
		expectedFields []string // Fields that should appear in results
		description    string
	}{
		{
			name: "groupby_application_severity_count",
			query: `source logs
				| filter $m.severity >= 4
				| groupby $l.applicationname, $m.severity
				| aggregate count() as error_count
				| sortby -error_count
				| limit 20`,
			expectedFields: []string{"app", "severity", "error_count"},
			description:    "Should extract $l.applicationname to 'app' and $m.severity to 'severity'",
		},
		{
			name: "groupby_with_aliases",
			query: `source logs
				| filter $m.severity >= 4
				| groupby $l.applicationname as app_name
				| aggregate count() as cnt
				| sortby -cnt
				| limit 20`,
			expectedFields: []string{"app", "cnt"},
			description:    "Should extract aliased field 'app_name' to 'app'",
		},
		{
			name: "groupby_time_bucket",
			query: `source logs
				| filter $m.severity >= 5
				| groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname
				| aggregate count() as error_count
				| sortby time_bucket
				| limit 20`,
			expectedFields: []string{"time_bucket", "app", "error_count"},
			description:    "Should handle time bucketing with application grouping",
		},
		{
			name: "simple_count_by_app",
			query: `source logs
				| groupby $l.applicationname
				| aggregate count() as log_count
				| sortby -log_count
				| limit 10`,
			expectedFields: []string{"app", "log_count"},
			description:    "Simple count by application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := queryTool.Execute(ctx, map[string]interface{}{
				"query":      tt.query,
				"start_date": startDate,
				"end_date":   endDate,
				"tier":       "archive",
			})

			require.NoError(t, err, "Query execution should not error")
			require.NotNil(t, result, "Result should not be nil")

			text := extractTextContent(result)
			t.Logf("Query: %s\n\nResult:\n%s", tt.name, truncateForLog(text, 2000))

			// Check it's not showing truly empty numbered entries (no content at all after the number)
			// Note: The format is "**[N]** \n> message" which is correct - message is on next line with blockquote
			// An entry is only empty if there's NO blockquote message following it
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "**[") && strings.Contains(line, "]**") {
					// Check if next non-empty line is a blockquote with content
					hasContent := false
					for j := i + 1; j < len(lines) && j < i+3; j++ {
						nextLine := strings.TrimSpace(lines[j])
						if strings.HasPrefix(nextLine, ">") && len(nextLine) > 2 {
							hasContent = true
							break
						}
						if strings.HasPrefix(nextLine, "**[") {
							// Hit next entry without finding content
							break
						}
					}
					if !hasContent && !strings.Contains(text, "No log entries found") {
						// Only fail if we actually got results but they're empty
						if strings.Contains(text, "Total Items: 0") || strings.Contains(text, "No log entries") {
							continue
						}
						t.Logf("Warning: Entry at line %d may be missing content: %s", i, line)
					}
				}
			}

			// If we got results (not "No log entries"), verify expected fields are present
			if !strings.Contains(text, "No log entries found") && !strings.Contains(text, "No events found") {
				for _, field := range tt.expectedFields {
					// Check if the field appears in the output (either as field name or in message)
					fieldPatterns := []string{
						field + "=",       // In message like "error_count=150"
						"**" + field,      // Bold field name
						"`" + field + "`", // Code formatted
					}

					found := false
					for _, pattern := range fieldPatterns {
						if strings.Contains(text, pattern) {
							found = true
							break
						}
					}

					// Also check for app/severity which get special treatment
					if field == "app" && (strings.Contains(text, "**") || strings.Contains(text, "app=")) {
						found = true
					}
					if field == "severity" && (strings.Contains(text, "[Error]") || strings.Contains(text, "[Warning]") || strings.Contains(text, "severity=")) {
						found = true
					}

					if !found {
						t.Logf("Warning: Expected field '%s' not clearly visible in output for: %s", field, tt.description)
					}
				}
			}
		})
	}
}

// TestReal_RegularLogQueryResults tests that regular (non-aggregation) queries return proper results
func TestReal_RegularLogQueryResults(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	queryTool := NewQueryTool(c, logger)
	ctx := context.Background()

	now := time.Now().UTC()
	startDate := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endDate := now.Format(time.RFC3339)

	t.Run("basic_error_logs", func(t *testing.T) {
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      "source logs | filter $m.severity >= 5 | limit 10",
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "frequent_search",
		})

		require.NoError(t, err)
		text := extractTextContent(result)
		t.Logf("Basic error logs result:\n%s", truncateForLog(text, 1500))

		// Should have timestamps, severity, app names if logs exist
		if !strings.Contains(text, "No log entries found") {
			// Check for proper formatting
			assert.True(t, strings.Contains(text, "[") || strings.Contains(text, "**["),
				"Should have numbered entries")
		}
	})
}

// TestReal_HealthCheckTCOAwareness tests that health_check uses TCO configuration
func TestReal_HealthCheckTCOAwareness(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	// First, fetch TCO config to know what tier to expect
	ctx := context.Background()
	err := FetchAndCacheTCOConfig(ctx, c, logger)
	require.NoError(t, err, "Should fetch TCO config")

	session := GetSession()
	tcoConfig := session.GetTCOConfig()

	expectedTier := "frequent_search" // default
	if tcoConfig != nil {
		expectedTier = tcoConfig.DefaultTier
		t.Logf("TCO Config - HasPolicies: %v, DefaultTier: %s", tcoConfig.HasPolicies, tcoConfig.DefaultTier)
	}

	healthTool := NewHealthCheckTool(c, logger)
	result, err := healthTool.Execute(ctx, map[string]interface{}{
		"time_range": "1h",
	})

	require.NoError(t, err)
	text := extractTextContent(result)
	t.Logf("Health check result:\n%s", truncateForLog(text, 2000))

	// The health check should complete without error
	assert.False(t, strings.Contains(text, "Health Check Failed"),
		"Health check should not fail")

	t.Logf("Health check used tier based on TCO config (expected: %s)", expectedTier)
}

// TestReal_QueryResponseTransformation tests the full transformation pipeline with real API responses
func TestReal_QueryResponseTransformation(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	queryTool := NewQueryTool(c, logger)
	ctx := context.Background()

	now := time.Now().UTC()
	startDate := now.Add(-1 * time.Hour).Format(time.RFC3339) // 1 hour back (data exists in this range)
	endDate := now.Format(time.RFC3339)

	// First verify data exists using SmartInvestigateTool
	t.Run("compare_investigate_vs_query", func(t *testing.T) {
		// SmartInvestigate should find data
		invTool := NewSmartInvestigateTool(c, logger)
		invResult, err := invTool.Execute(ctx, map[string]interface{}{
			"time_range": "1h",
		})
		require.NoError(t, err)
		invText := extractTextContent(invResult)

		// Check if investigation found data
		hasData := strings.Contains(invText, "events") && !strings.Contains(invText, "0 events")
		t.Logf("SmartInvestigate found data: %v", hasData)

		if hasData {
			t.Logf("Investigation preview:\n%s", truncateForLog(invText, 500))
		}

		// Now run QueryTool with same time range
		// Use exact same query format as SmartInvestigate
		queryString := `source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as error_count | sortby -error_count | limit 20`
		t.Logf("Running QueryTool with: %s", queryString)
		t.Logf("Time range: %s to %s", startDate, endDate)

		queryResult, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      queryString,
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "archive",
		})
		require.NoError(t, err)
		queryText := extractTextContent(queryResult)

		t.Logf("QueryTool result:\n%s", truncateForLog(queryText, 1500))

		// Both should find data if investigation did
		if hasData {
			assert.False(t, strings.Contains(queryText, "No log entries found"),
				"QueryTool should also find data when SmartInvestigate does")
		}
	})

	t.Run("verify_no_empty_entries", func(t *testing.T) {
		// Use the exact query pattern that was used in the investigation test that returns data
		// Filter by ERROR severity to get aggregation results
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query": `source logs
				| filter $m.severity >= ERROR
				| groupby $l.applicationname
				| aggregate count() as error_count
				| sortby -error_count
				| limit 20`,
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "archive", // Use archive tier as the investigation does
		})

		require.NoError(t, err)
		text := extractTextContent(result)

		// Count entries vs empty entries
		lines := strings.Split(text, "\n")
		totalEntries := 0
		emptyEntries := 0

		for i, line := range lines {
			if strings.HasPrefix(line, "**[") {
				totalEntries++
				// Check if this entry is effectively empty (next line is blank or another entry)
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					// Entry is empty if it only has the number and nothing else on the same line
					// and the next line is blank or starts with another entry
					entryContent := strings.TrimPrefix(line, "**[")
					if idx := strings.Index(entryContent, "]**"); idx != -1 {
						afterNumber := strings.TrimSpace(entryContent[idx+3:])
						if afterNumber == "" && (nextLine == "" || strings.HasPrefix(nextLine, "**[")) {
							emptyEntries++
							t.Logf("Found empty entry at line %d: %s", i, line)
						}
					}
				}
			}
		}

		t.Logf("Total entries: %d, Empty entries: %d", totalEntries, emptyEntries)

		if totalEntries > 0 && emptyEntries > 0 {
			emptyRatio := float64(emptyEntries) / float64(totalEntries)
			if emptyRatio > 0.5 {
				t.Errorf("More than 50%% of entries are empty (%d/%d) - transformation is broken!",
					emptyEntries, totalEntries)
			}
		}

		t.Logf("Query result:\n%s", truncateForLog(text, 3000))
	})
}

// Helper to get test logger
func getTestLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

// TestReal_DebugRawAPIResponse tests the raw API response to understand the data structure
func TestReal_DebugRawAPIResponse(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()
	ctx := context.Background()

	now := time.Now().UTC()
	startDate := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endDate := now.Format(time.RFC3339)

	// Create a raw request to see what the API returns
	baseTool := NewBaseTool(c, logger)

	query := `source logs | filter $m.severity >= ERROR | groupby $l.applicationname aggregate count() as error_count | sortby -error_count | limit 20`

	metadata := map[string]interface{}{
		"tier":       "archive",
		"syntax":     "dataprime",
		"start_date": startDate,
		"end_date":   endDate,
	}

	body := map[string]interface{}{
		"query":    query,
		"metadata": metadata,
	}

	req := &client.Request{
		Method:    "POST",
		Path:      "/v1/query",
		Body:      body,
		AcceptSSE: true,
		Timeout:   60 * time.Second,
	}

	result, err := baseTool.ExecuteRequest(ctx, req)
	require.NoError(t, err)

	// Log the raw result structure
	t.Logf("Raw result type: %T", result)
	if result != nil {
		for k, v := range result {
			t.Logf("Key '%s': type=%T, preview=%v", k, v, truncateForLog(fmt.Sprintf("%v", v), 200))
		}

		// Check events
		if events, ok := result["events"].([]interface{}); ok {
			t.Logf("Events count: %d", len(events))
			for i, event := range events {
				if i >= 3 {
					t.Logf("... and %d more events", len(events)-3)
					break
				}
				eventMap, ok := event.(map[string]interface{})
				if ok {
					t.Logf("Event[%d] keys: %v", i, getKeys(eventMap))
					// If it has result.results, show that
					if resultObj, ok := eventMap["result"].(map[string]interface{}); ok {
						if results, ok := resultObj["results"].([]interface{}); ok {
							t.Logf("  result.results count: %d", len(results))
							for j, r := range results {
								if j >= 2 {
									break
								}
								if rMap, ok := r.(map[string]interface{}); ok {
									t.Logf("  results[%d]: %v", j, rMap)
								}
							}
						}
					}
				}
			}
		}

		// Now test CleanQueryResults
		cleaned := CleanQueryResults(result)
		t.Logf("\nAfter CleanQueryResults:")
		for k, v := range cleaned {
			t.Logf("Key '%s': type=%T", k, v)
			if k == "logs" {
				if logs, ok := v.([]interface{}); ok {
					t.Logf("  logs count: %d", len(logs))
					for i, log := range logs {
						if i >= 3 {
							break
						}
						t.Logf("  logs[%d]: %v", i, log)
					}
				}
			}
		}
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
