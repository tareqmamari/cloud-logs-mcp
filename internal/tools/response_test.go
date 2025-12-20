package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformLogEntry_ArrayFormat(t *testing.T) {
	// Array format: labels/metadata as arrays with key/value pairs
	entry := map[string]interface{}{
		"labels": []interface{}{
			map[string]interface{}{"key": "applicationname", "value": "myapp"},
			map[string]interface{}{"key": "subsystemname", "value": "api"},
		},
		"metadata": []interface{}{
			map[string]interface{}{"key": "timestamp", "value": "2024-01-01T12:00:00Z"},
			map[string]interface{}{"key": "severity", "value": "ERROR"},
		},
		"user_data": `{"message": "Test error message", "level": "error"}`,
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "myapp", result["app"])
	assert.Equal(t, "api", result["subsystem"])
	assert.Equal(t, "2024-01-01T12:00:00Z", result["time"])
	assert.Equal(t, "ERROR", result["severity"])
	assert.Equal(t, "Test error message", result["message"])
}

func TestTransformLogEntry_MapFormat(t *testing.T) {
	// Map format: labels/metadata as maps with direct fields
	entry := map[string]interface{}{
		"labels": map[string]interface{}{
			"applicationname": "webapp",
			"subsystemname":   "auth",
		},
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-02T15:30:00Z",
			"severity":  "WARNING",
		},
		"user_data": map[string]interface{}{
			"message": "User data as map",
			"level":   "warn",
		},
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "webapp", result["app"])
	assert.Equal(t, "auth", result["subsystem"])
	assert.Equal(t, "2024-01-02T15:30:00Z", result["time"])
	assert.Equal(t, "WARNING", result["severity"])
	assert.Equal(t, "User data as map", result["message"])
}

func TestTransformLogEntry_FlatFormat(t *testing.T) {
	// Flat format: direct fields at root level (aggregation results)
	entry := map[string]interface{}{
		"timestamp":       "2024-01-03T10:00:00Z",
		"message":         "Direct message in flat format",
		"severity":        "Info",
		"applicationname": "service-a",
		"subsystemname":   "worker",
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "2024-01-03T10:00:00Z", result["time"])
	assert.Equal(t, "Direct message in flat format", result["message"])
	assert.Equal(t, "Info", result["severity"])
	assert.Equal(t, "service-a", result["app"])
	assert.Equal(t, "worker", result["subsystem"])
}

func TestTransformLogEntry_AggregationResult(t *testing.T) {
	// Aggregation result: count/sum/etc fields without standard log structure
	entry := map[string]interface{}{
		"count":       float64(42),
		"avg_latency": float64(125.5),
	}

	result := transformLogEntry(entry)

	// Should create a message from the available fields
	assert.NotEmpty(t, result["message"])
	msg := result["message"].(string)
	assert.Contains(t, msg, "count=42")
	assert.Contains(t, msg, "avg_latency=126") // Rounded from 125.5
}

func TestTransformLogEntry_AggregationResultWithApp(t *testing.T) {
	// Aggregation result with application field
	entry := map[string]interface{}{
		"count":       float64(10),
		"application": "metrics-service",
	}

	result := transformLogEntry(entry)

	// Application should be extracted to app
	assert.Equal(t, "metrics-service", result["app"])
	// Count should be in message since no standard message field
	assert.NotEmpty(t, result["message"])
}

func TestTransformLogEntry_NumericSeverity(t *testing.T) {
	// Numeric severity in metadata map format
	entry := map[string]interface{}{
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-04T08:00:00Z",
			"severity":  float64(5), // 5 = Error
		},
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "Error", result["severity"])
}

func TestTransformLogEntry_EmptyEntry(t *testing.T) {
	entry := map[string]interface{}{}
	result := transformLogEntry(entry)

	// Should return empty but not panic
	assert.Empty(t, result)
}

func TestTransformLogEntry_AlternativeFieldNames(t *testing.T) {
	// Test alternative timestamp field names
	tests := []struct {
		name     string
		entry    map[string]interface{}
		expected string
	}{
		{
			name:     "@timestamp",
			entry:    map[string]interface{}{"@timestamp": "2024-01-01T00:00:00Z"},
			expected: "2024-01-01T00:00:00Z",
		},
		{
			name:     "_time",
			entry:    map[string]interface{}{"_time": "2024-01-02T00:00:00Z"},
			expected: "2024-01-02T00:00:00Z",
		},
		{
			name:     "ts",
			entry:    map[string]interface{}{"ts": "2024-01-03T00:00:00Z"},
			expected: "2024-01-03T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformLogEntry(tt.entry)
			assert.Equal(t, tt.expected, result["time"])
		})
	}
}

func TestTransformLogEntry_AlternativeMessageFields(t *testing.T) {
	tests := []struct {
		name     string
		entry    map[string]interface{}
		expected string
	}{
		{
			name:     "msg field",
			entry:    map[string]interface{}{"msg": "Message via msg"},
			expected: "Message via msg",
		},
		{
			name:     "text field",
			entry:    map[string]interface{}{"text": "Message via text"},
			expected: "Message via text",
		},
		{
			name:     "log field",
			entry:    map[string]interface{}{"log": "Message via log"},
			expected: "Message via log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformLogEntry(tt.entry)
			assert.Equal(t, tt.expected, result["message"])
		})
	}
}

func TestCleanQueryResults_WithDifferentFormats(t *testing.T) {
	t.Run("nested result structure", func(t *testing.T) {
		// SSE format with nested result.results
		result := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"result": map[string]interface{}{
						"results": []interface{}{
							map[string]interface{}{
								"user_data": `{"message": "nested message", "level": "info"}`,
							},
						},
					},
				},
			},
		}

		cleaned := CleanQueryResults(result)

		logs, ok := cleaned["logs"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, logs, 1)

		log := logs[0].(map[string]interface{})
		assert.Equal(t, "nested message", log["message"])
	})

	t.Run("direct event format", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"message":   "direct format message",
					"timestamp": "2024-01-01T00:00:00Z",
				},
			},
		}

		cleaned := CleanQueryResults(result)

		logs, ok := cleaned["logs"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, logs, 1)

		log := logs[0].(map[string]interface{})
		assert.Equal(t, "direct format message", log["message"])
		assert.Equal(t, "2024-01-01T00:00:00Z", log["time"])
	})

	t.Run("aggregation query results", func(t *testing.T) {
		// This is what groupby ... aggregate queries return
		result := map[string]interface{}{
			"events": []interface{}{
				// First event is usually query_id metadata - should be skipped
				map[string]interface{}{
					"query_id": map[string]interface{}{
						"id": "some-query-id",
					},
				},
				// Actual aggregation results
				map[string]interface{}{
					"result": map[string]interface{}{
						"results": []interface{}{
							map[string]interface{}{
								"applicationname": "api-gateway",
								"severity":        float64(5),
								"error_count":     float64(150),
							},
							map[string]interface{}{
								"applicationname": "auth-service",
								"severity":        float64(4),
								"error_count":     float64(75),
							},
						},
					},
				},
			},
		}

		cleaned := CleanQueryResults(result)

		logs, ok := cleaned["logs"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, logs, 2, "Should have 2 aggregation results")

		// First result
		log1 := logs[0].(map[string]interface{})
		assert.Equal(t, "api-gateway", log1["app"])
		assert.NotEmpty(t, log1["message"], "Aggregation result should have a message")
		assert.Contains(t, log1["message"].(string), "error_count=150")

		// Second result
		log2 := logs[1].(map[string]interface{})
		assert.Equal(t, "auth-service", log2["app"])
		assert.Contains(t, log2["message"].(string), "error_count=75")
	})

	t.Run("skip query_id only events", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"query_id": "abc123",
				},
				map[string]interface{}{
					"message": "actual log entry",
				},
			},
		}

		cleaned := CleanQueryResults(result)

		logs, ok := cleaned["logs"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, logs, 1, "Should skip query_id only events")
		assert.Equal(t, "actual log entry", logs[0].(map[string]interface{})["message"])
	})
}

func TestSeverityNumToName(t *testing.T) {
	tests := []struct {
		severity int
		expected string
	}{
		{1, "Debug"},
		{2, "Verbose"},
		{3, "Info"},
		{4, "Warning"},
		{5, "Error"},
		{6, "Critical"},
		{7, "Level 7"}, // Unknown level
		{0, "Level 0"}, // Unknown level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := severityNumToName(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformLogEntry_DataPrimeGroupByFields(t *testing.T) {
	// Test DataPrime groupby results with $l. and $m. prefixed field names
	// This is what happens when you run: groupby $l.applicationname, $m.severity aggregate count() as error_count
	entry := map[string]interface{}{
		"$l.applicationname": "api-gateway",
		"$m.severity":        float64(5),
		"error_count":        float64(150),
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "api-gateway", result["app"])
	assert.Equal(t, "Error", result["severity"])
	assert.NotEmpty(t, result["message"])
	assert.Contains(t, result["message"].(string), "error_count=150")
}

func TestTransformLogEntry_DataPrimeGroupByWithAlias(t *testing.T) {
	// Test DataPrime groupby with 'as' alias: groupby $l.applicationname as app_name
	entry := map[string]interface{}{
		"app_name":    "auth-service",
		"error_count": float64(75),
	}

	result := transformLogEntry(entry)

	assert.Equal(t, "auth-service", result["app"])
	assert.Contains(t, result["message"].(string), "error_count=75")
}

func TestCleanQueryResults_DataPrimeAggregation(t *testing.T) {
	// Test the full flow with DataPrime aggregation query results
	// This is what the API returns for: groupby $l.applicationname, $m.severity aggregate count() as error_count
	result := map[string]interface{}{
		"events": []interface{}{
			// First event is query_id metadata - should be skipped
			map[string]interface{}{
				"query_id": map[string]interface{}{
					"id": "some-query-id",
				},
			},
			// Second event contains the actual results
			map[string]interface{}{
				"result": map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"$l.applicationname": "api-gateway",
							"$m.severity":        float64(5),
							"error_count":        float64(150),
						},
						map[string]interface{}{
							"$l.applicationname": "auth-service",
							"$m.severity":        float64(4),
							"error_count":        float64(75),
						},
					},
				},
			},
		},
	}

	cleaned := CleanQueryResults(result)

	logs, ok := cleaned["logs"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, logs, 2, "Should have 2 aggregation results")

	// First result
	log1 := logs[0].(map[string]interface{})
	assert.Equal(t, "api-gateway", log1["app"])
	assert.Equal(t, "Error", log1["severity"])
	assert.Contains(t, log1["message"].(string), "error_count=150")

	// Second result
	log2 := logs[1].(map[string]interface{})
	assert.Equal(t, "auth-service", log2["app"])
	assert.Equal(t, "Warning", log2["severity"])
	assert.Contains(t, log2["message"].(string), "error_count=75")
}

func TestCleanQueryResults_UserDataJSONAggregation(t *testing.T) {
	// Test the real API format where aggregation results are in user_data JSON string
	// This is the exact format returned by IBM Cloud Logs for groupby queries
	result := map[string]interface{}{
		"events": []interface{}{
			// First event is query_id metadata - should be skipped
			map[string]interface{}{
				"query_id": map[string]interface{}{
					"query_id": "test-query-id",
				},
			},
			// Second event contains the actual results in user_data JSON
			map[string]interface{}{
				"result": map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"labels":    []interface{}{},
							"metadata":  []interface{}{},
							"user_data": `{"applicationname":"content-structure","error_count":3185448}`,
						},
						map[string]interface{}{
							"labels":    []interface{}{},
							"metadata":  []interface{}{},
							"user_data": `{"applicationname":"veni-vici","error_count":139617}`,
						},
						map[string]interface{}{
							"labels":    []interface{}{},
							"metadata":  []interface{}{},
							"user_data": `{"applicationname":"auth-service","severity":5,"error_count":117}`,
						},
					},
				},
			},
		},
	}

	cleaned := CleanQueryResults(result)

	logs, ok := cleaned["logs"].([]interface{})
	assert.True(t, ok, "Should have logs array")
	assert.Len(t, logs, 3, "Should have 3 aggregation results")

	// First result - content-structure
	log1 := logs[0].(map[string]interface{})
	assert.Equal(t, "content-structure", log1["app"], "Should extract applicationname from user_data")
	assert.NotEmpty(t, log1["message"], "Should have message with error_count")
	assert.Contains(t, log1["message"].(string), "error_count=3185448")

	// Second result - veni-vici
	log2 := logs[1].(map[string]interface{})
	assert.Equal(t, "veni-vici", log2["app"])
	assert.Contains(t, log2["message"].(string), "error_count=139617")

	// Third result - auth-service with severity
	log3 := logs[2].(map[string]interface{})
	assert.Equal(t, "auth-service", log3["app"])
	assert.Equal(t, "Error", log3["severity"], "Should convert numeric severity to name")
	assert.Contains(t, log3["message"].(string), "error_count=117")
}
