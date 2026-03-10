package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Realistic SSE data modeled after IBM Cloud Logs (cxint eu-gb instance) responses.
// Each test fixture represents actual API wire format.

func TestParseSSEResponse_ResultMessages(t *testing.T) {
	// Simulate a real /v1/query SSE response with result messages containing log entries
	sseBody := strings.Join([]string{
		`data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"api-gateway"},{"key":"subsystemname","value":"nginx"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:15:30.123Z"},{"key":"severity","value":"5"}],"user_data":"{\"message\":\"Connection timeout to upstream service payments-svc after 30s\",\"level\":\"ERROR\",\"trace_id\":\"abc123def456\",\"span_id\":\"span-789\",\"logger_name\":\"com.ibm.cloud.gateway.ProxyHandler\"}"}]}}`,
		`data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"payments-svc"},{"key":"subsystemname","value":"spring-boot"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:15:29.800Z"},{"key":"severity","value":"5"}],"user_data":"{\"message\":\"Database connection pool exhausted: max connections (50) reached\",\"level\":\"ERROR\",\"trace_id\":\"abc123def456\",\"span_id\":\"span-456\",\"event\":{\"_message\":\"Pool exhausted\",\"sql\":\"SELECT * FROM transactions WHERE status='PENDING'\",\"execMs\":30250}}"}]}}`,
		`data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"order-svc"},{"key":"subsystemname","value":"java"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:15:28.000Z"},{"key":"severity","value":"3"}],"user_data":"{\"message\":\"Processing order #12345 for customer C-9876\",\"level\":\"INFO\"}"}]}}`,
		"",
	}, "\n")

	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil for valid SSE body")
	}

	events, ok := result["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in result")
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Verify first event is properly flattened
	first, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected first event to be map")
	}

	// Labels should be flattened to top-level keys
	if first["applicationname"] != "api-gateway" {
		t.Errorf("applicationname = %v, want api-gateway", first["applicationname"])
	}
	if first["subsystemname"] != "nginx" {
		t.Errorf("subsystemname = %v, want nginx", first["subsystemname"])
	}

	// Metadata should be flattened
	if first["timestamp"] != "2026-03-09T10:15:30.123Z" {
		t.Errorf("timestamp = %v, want 2026-03-09T10:15:30.123Z", first["timestamp"])
	}
	if first["severity"] != "5" {
		t.Errorf("severity = %v, want 5", first["severity"])
	}

	// user_data should be parsed into an object, not a raw JSON string
	userData, ok := first["user_data"].(map[string]interface{})
	if !ok {
		t.Fatalf("user_data should be parsed map, got %T", first["user_data"])
	}
	if userData["message"] != "Connection timeout to upstream service payments-svc after 30s" {
		t.Errorf("user_data.message = %v", userData["message"])
	}
	if userData["trace_id"] != "abc123def456" {
		t.Errorf("user_data.trace_id = %v", userData["trace_id"])
	}

	// Verify second event has nested event structure in user_data
	second := events[1].(map[string]interface{})
	ud2 := second["user_data"].(map[string]interface{})
	event, ok := ud2["event"].(map[string]interface{})
	if !ok {
		t.Fatal("expected event sub-object in second entry user_data")
	}
	if event["sql"] != "SELECT * FROM transactions WHERE status='PENDING'" {
		t.Errorf("event.sql = %v", event["sql"])
	}
}

func TestParseSSEResponse_WarningMessages(t *testing.T) {
	tests := []struct {
		name        string
		sseBody     string
		wantWarning string
	}{
		{
			name: "compile warning",
			sseBody: `data: {"warning":{"compile_warning":{"warning_message":"keypath does not exist '$d.foo' in line 0 at column 21"}}}
data: {"result":{"results":[{"labels":[],"metadata":[],"user_data":"{}"}]}}
`,
			wantWarning: "keypath does not exist '$d.foo'",
		},
		{
			name: "time range warning",
			sseBody: `data: {"warning":{"time_range_warning":{"warning_message":"end of time range is set to: 2026-03-09 15:00:00.000","start_date":"2021-01-01T00:00:00.000Z","end_date":"2021-01-01T00:00:00.000Z"}}}
data: {"result":{"results":[]}}
`,
			wantWarning: "end of time range is set to",
		},
		{
			name: "number of results limit warning",
			sseBody: `data: {"warning":{"number_of_results_limit_warning":{"number_of_results_limit":10000}}}
data: {"result":{"results":[{"labels":[],"metadata":[],"user_data":"{}"}]}}
`,
			wantWarning: "Results capped at 10000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSSEResponse([]byte(tt.sseBody))
			if result == nil {
				t.Fatal("parseSSEResponse returned nil")
			}

			warnings, ok := result["_warnings"].([]string)
			if !ok || len(warnings) == 0 {
				t.Fatal("expected _warnings in result")
			}

			found := false
			for _, w := range warnings {
				if strings.Contains(w, tt.wantWarning) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected warning containing %q, got %v", tt.wantWarning, warnings)
			}
		})
	}
}

func TestParseSSEResponse_ErrorMessages(t *testing.T) {
	sseBody := `data: {"error":{"message":"Failed to run the query 1/archive/3424/4rwoNx1XNcc","code":{"rate_limit_reached":{}}}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil for error SSE")
	}

	errors, ok := result["_errors"].([]string)
	if !ok || len(errors) == 0 {
		t.Fatal("expected _errors in result")
	}
	if !strings.Contains(errors[0], "Failed to run the query") {
		t.Errorf("expected error message about failed query, got %v", errors[0])
	}
}

func TestParseSSEResponse_QueryID(t *testing.T) {
	sseBody := `data: {"query_id":{"query_id":"4rwoNx1XNcc"}}
data: {"result":{"results":[{"labels":[],"metadata":[],"user_data":"{}"}]}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	queryID, ok := result["_query_id"].(string)
	if !ok || queryID != "4rwoNx1XNcc" {
		t.Errorf("expected _query_id=4rwoNx1XNcc, got %v", result["_query_id"])
	}
}

func TestParseSSEResponse_MixedMessageTypes(t *testing.T) {
	// Realistic full response: query_id → warning → results → results
	sseBody := `data: {"query_id":{"query_id":"test-query-123"}}
data: {"warning":{"compile_warning":{"warning_message":"field '$d.nonexistent' not found"}}}
data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"frontend"}],"metadata":[{"key":"timestamp","value":"2026-03-09T12:00:00Z"},{"key":"severity","value":"4"}],"user_data":"{\"message\":\"Slow render detected: 2500ms for /dashboard\"}"}]}}
data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"frontend"}],"metadata":[{"key":"timestamp","value":"2026-03-09T12:00:01Z"},{"key":"severity","value":"3"}],"user_data":"{\"message\":\"Page loaded successfully\"}"}]}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	// Should have query_id
	if result["_query_id"] != "test-query-123" {
		t.Errorf("_query_id = %v", result["_query_id"])
	}

	// Should have warning
	warnings := result["_warnings"].([]string)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}

	// Should have 2 events
	events := result["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Events should be flattened
	first := events[0].(map[string]interface{})
	if first["applicationname"] != "frontend" {
		t.Errorf("first event applicationname = %v", first["applicationname"])
	}
}

func TestParseSSEResponse_NotSSE(t *testing.T) {
	// Regular JSON should return nil
	jsonBody := `{"alerts": [{"id": "123", "name": "test"}]}`
	result := parseSSEResponse([]byte(jsonBody))
	if result != nil {
		t.Error("expected nil for non-SSE body")
	}

	// Empty body
	result = parseSSEResponse([]byte{})
	if result != nil {
		t.Error("expected nil for empty body")
	}
}

func TestParseSSEResponse_Truncation(t *testing.T) {
	// Build SSE body with more events than MaxSSEEvents
	var lines []string
	for i := 0; i < MaxSSEEvents+100; i++ {
		line := fmt.Sprintf(`data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"app-%d"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:00:%02d.000Z"}],"user_data":"{\"message\":\"Log entry %d\"}"}]}}`, i%10, i%60, i)
		lines = append(lines, line)
	}
	sseBody := strings.Join(lines, "\n") + "\n"

	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	if len(events) != MaxSSEEvents {
		t.Errorf("expected %d events (MaxSSEEvents), got %d", MaxSSEEvents, len(events))
	}

	if result["_truncated"] != true {
		t.Error("expected _truncated=true")
	}
	if result["_total_events"] != MaxSSEEvents+100 {
		t.Errorf("_total_events = %v, want %d", result["_total_events"], MaxSSEEvents+100)
	}
	if result["_shown_events"] != MaxSSEEvents {
		t.Errorf("_shown_events = %v, want %d", result["_shown_events"], MaxSSEEvents)
	}
}

func TestParseSSEResponse_MultipleResultsPerMessage(t *testing.T) {
	// A single SSE data line can contain multiple results in the results array
	sseBody := `data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"batch-job"}],"metadata":[{"key":"timestamp","value":"2026-03-09T08:00:00Z"},{"key":"severity","value":"3"}],"user_data":"{\"message\":\"Processing batch item 1\"}"},{"labels":[{"key":"applicationname","value":"batch-job"}],"metadata":[{"key":"timestamp","value":"2026-03-09T08:00:01Z"},{"key":"severity","value":"3"}],"user_data":"{\"message\":\"Processing batch item 2\"}"},{"labels":[{"key":"applicationname","value":"batch-job"}],"metadata":[{"key":"timestamp","value":"2026-03-09T08:00:02Z"},{"key":"severity","value":"5"}],"user_data":"{\"message\":\"Batch item 3 failed: invalid input\"}"}]}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	if len(events) != 3 {
		t.Errorf("expected 3 events from single SSE line with 3 results, got %d", len(events))
	}

	// Third event should be the error
	third := events[2].(map[string]interface{})
	if third["severity"] != "5" {
		t.Errorf("third event severity = %v, want 5", third["severity"])
	}
	ud := third["user_data"].(map[string]interface{})
	if !strings.Contains(ud["message"].(string), "failed") {
		t.Errorf("expected failure message in third event")
	}
}

func TestParseSSEResponse_UserDataNotJSON(t *testing.T) {
	// user_data can sometimes be a plain string, not JSON
	sseBody := `data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"legacy-app"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:00:00Z"}],"user_data":"This is just a plain text log message, not JSON"}]}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	first := events[0].(map[string]interface{})

	// user_data should be kept as string since it's not valid JSON
	if _, ok := first["user_data"].(string); !ok {
		t.Errorf("expected user_data to remain as string for non-JSON content, got %T", first["user_data"])
	}
}

func TestParseSSEResponse_CarriageReturns(t *testing.T) {
	// Some servers send \r\n line endings
	sseBody := "data: {\"result\":{\"results\":[{\"labels\":[],\"metadata\":[],\"user_data\":\"{}\"}]}}\r\ndata: {\"result\":{\"results\":[{\"labels\":[],\"metadata\":[],\"user_data\":\"{}\"}]}}\r\n"

	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("expected 2 events with \\r\\n line endings, got %d", len(events))
	}
}

func TestParseSSEResponse_EmptyResults(t *testing.T) {
	// Query that matches nothing — empty results array
	sseBody := `data: {"query_id":{"query_id":"empty-query"}}
data: {"result":{"results":[]}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	if result["_query_id"] != "empty-query" {
		t.Errorf("_query_id = %v", result["_query_id"])
	}
}

func TestFlattenLogEntry(t *testing.T) {
	entry := map[string]interface{}{
		"labels": []interface{}{
			map[string]interface{}{"key": "applicationname", "value": "my-service"},
			map[string]interface{}{"key": "subsystemname", "value": "worker"},
			map[string]interface{}{"key": "computername", "value": "node-01"},
		},
		"metadata": []interface{}{
			map[string]interface{}{"key": "timestamp", "value": "2026-03-09T10:00:00Z"},
			map[string]interface{}{"key": "severity", "value": "5"},
			map[string]interface{}{"key": "logid", "value": "abc-should-be-kept-at-flatten"},
		},
		"user_data": `{"message":"Connection refused","level":"ERROR","trace_id":"t-123"}`,
	}

	flat := flattenLogEntry(entry)

	// Labels are flattened
	if flat["applicationname"] != "my-service" {
		t.Errorf("applicationname = %v", flat["applicationname"])
	}
	if flat["subsystemname"] != "worker" {
		t.Errorf("subsystemname = %v", flat["subsystemname"])
	}
	if flat["computername"] != "node-01" {
		t.Errorf("computername = %v", flat["computername"])
	}

	// Metadata is flattened
	if flat["timestamp"] != "2026-03-09T10:00:00Z" {
		t.Errorf("timestamp = %v", flat["timestamp"])
	}
	if flat["severity"] != "5" {
		t.Errorf("severity = %v", flat["severity"])
	}

	// user_data is parsed into a map
	ud, ok := flat["user_data"].(map[string]interface{})
	if !ok {
		t.Fatalf("user_data should be map, got %T", flat["user_data"])
	}
	if ud["message"] != "Connection refused" {
		t.Errorf("user_data.message = %v", ud["message"])
	}
	if ud["trace_id"] != "t-123" {
		t.Errorf("user_data.trace_id = %v", ud["trace_id"])
	}

	// Original array keys should not be present
	if _, ok := flat["labels"]; ok {
		t.Error("labels array should not be in flattened output")
	}
	if _, ok := flat["metadata"]; ok {
		t.Error("metadata array should not be in flattened output")
	}
}

func TestFlattenLogEntry_EmptyLabelsAndMetadata(t *testing.T) {
	entry := map[string]interface{}{
		"labels":    []interface{}{},
		"metadata":  []interface{}{},
		"user_data": `{"message":"bare log"}`,
	}

	flat := flattenLogEntry(entry)
	ud := flat["user_data"].(map[string]interface{})
	if ud["message"] != "bare log" {
		t.Errorf("message = %v", ud["message"])
	}
}

func TestFlattenLogEntry_PreservesExtraFields(t *testing.T) {
	entry := map[string]interface{}{
		"labels":        []interface{}{},
		"metadata":      []interface{}{},
		"user_data":     `{}`,
		"custom_field":  "should-be-preserved",
		"another_field": float64(42),
	}

	flat := flattenLogEntry(entry)
	if flat["custom_field"] != "should-be-preserved" {
		t.Errorf("custom_field = %v", flat["custom_field"])
	}
	if flat["another_field"] != float64(42) {
		t.Errorf("another_field = %v", flat["another_field"])
	}
}

func TestTransformLogEntry_FlattenedFormat(t *testing.T) {
	// Test that transformLogEntry works with the new flattened format
	// (output of flattenLogEntry)
	flattened := map[string]interface{}{
		"applicationname": "api-gateway",
		"subsystemname":   "nginx",
		"timestamp":       "2026-03-09T10:15:30.123Z",
		"severity":        "5",
		"user_data": map[string]interface{}{
			"message":     "Connection timeout to upstream service",
			"level":       "ERROR",
			"trace_id":    "abc123",
			"logger_name": "com.ibm.cloud.gateway.ProxyHandler",
		},
	}

	compact := transformLogEntry(flattened)

	if compact["app"] != "api-gateway" {
		t.Errorf("app = %v, want api-gateway", compact["app"])
	}
	if compact["subsystem"] != "nginx" {
		t.Errorf("subsystem = %v, want nginx", compact["subsystem"])
	}
	if compact["time"] != "2026-03-09T10:15:30.123Z" {
		t.Errorf("time = %v", compact["time"])
	}
	if compact["severity"] != "5" {
		t.Errorf("severity = %v", compact["severity"])
	}
	if compact["message"] != "Connection timeout to upstream service" {
		t.Errorf("message = %v", compact["message"])
	}
	if compact["trace_id"] != "abc123" {
		t.Errorf("trace_id = %v", compact["trace_id"])
	}
	// Logger should be shortened
	if compact["logger"] != "gateway.ProxyHandler" {
		t.Errorf("logger = %v, want gateway.ProxyHandler", compact["logger"])
	}
}

func TestTransformLogEntry_LegacyFormat(t *testing.T) {
	// Test that transformLogEntry still works with the legacy array format
	legacy := map[string]interface{}{
		"labels": []interface{}{
			map[string]interface{}{"key": "applicationname", "value": "my-app"},
			map[string]interface{}{"key": "subsystemname", "value": "worker"},
		},
		"metadata": []interface{}{
			map[string]interface{}{"key": "timestamp", "value": "2026-03-09T10:00:00Z"},
			map[string]interface{}{"key": "severity", "value": "3"},
		},
		"user_data": `{"message":"Processing complete","level":"INFO"}`,
	}

	compact := transformLogEntry(legacy)
	if compact["app"] != "my-app" {
		t.Errorf("app = %v", compact["app"])
	}
	if compact["time"] != "2026-03-09T10:00:00Z" {
		t.Errorf("time = %v", compact["time"])
	}
	if compact["message"] != "Processing complete" {
		t.Errorf("message = %v", compact["message"])
	}
}

func TestParseSSEMessages_ClassifiesCorrectly(t *testing.T) {
	body := `data: {"query_id":{"query_id":"q1"}}
data: {"warning":{"compile_warning":{"warning_message":"test warning"}}}
data: {"error":{"message":"test error"}}
data: {"result":{"results":[{"labels":[],"metadata":[],"user_data":"{}"}]}}
data: {"unknown_type":"should be treated as raw event"}
`
	parsed := parseSSEMessages(body, 100)

	if parsed.QueryID != "q1" {
		t.Errorf("QueryID = %v", parsed.QueryID)
	}
	if len(parsed.Warnings) != 1 || parsed.Warnings[0] != "test warning" {
		t.Errorf("Warnings = %v", parsed.Warnings)
	}
	if len(parsed.Errors) != 1 || parsed.Errors[0] != "test error" {
		t.Errorf("Errors = %v", parsed.Errors)
	}
	// 1 result entry + 1 unknown type = 2 events
	if len(parsed.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(parsed.Events))
	}
	if parsed.Total != 2 {
		t.Errorf("Total = %d, want 2", parsed.Total)
	}
}

func TestHandleSSEError_StringError(t *testing.T) {
	parsed := &SSEParseResult{}
	handleSSEError("simple error string", parsed)
	if len(parsed.Errors) != 1 || parsed.Errors[0] != "simple error string" {
		t.Errorf("Errors = %v", parsed.Errors)
	}
}

func TestHandleSSEError_ObjectError(t *testing.T) {
	parsed := &SSEParseResult{}
	handleSSEError(map[string]interface{}{
		"message": "Rate limit exceeded",
		"code":    map[string]interface{}{"rate_limit_reached": map[string]interface{}{}},
	}, parsed)
	if len(parsed.Errors) != 1 || parsed.Errors[0] != "Rate limit exceeded" {
		t.Errorf("Errors = %v", parsed.Errors)
	}
}

func TestHandleSSEError_ObjectWithoutMessage(t *testing.T) {
	parsed := &SSEParseResult{}
	handleSSEError(map[string]interface{}{
		"code": "UNKNOWN_ERROR",
	}, parsed)
	if len(parsed.Errors) != 1 {
		t.Fatal("expected 1 error")
	}
	// Should serialize the whole object
	if !strings.Contains(parsed.Errors[0], "UNKNOWN_ERROR") {
		t.Errorf("expected serialized error, got %v", parsed.Errors[0])
	}
}

func TestHandleSSEWarning_UnknownSubType(t *testing.T) {
	parsed := &SSEParseResult{}
	handleSSEWarning(map[string]interface{}{
		"new_warning_type": map[string]interface{}{
			"detail": "some future warning format",
		},
	}, parsed)
	// Should serialize the whole warning as fallback
	if len(parsed.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(parsed.Warnings))
	}
	if !strings.Contains(parsed.Warnings[0], "new_warning_type") {
		t.Errorf("expected serialized warning, got %v", parsed.Warnings[0])
	}
}

func TestHandleSSEQueryID_StringFormat(t *testing.T) {
	parsed := &SSEParseResult{}
	handleSSEQueryID("direct-string-id", parsed)
	if parsed.QueryID != "direct-string-id" {
		t.Errorf("QueryID = %v", parsed.QueryID)
	}
}

func TestParseSSEResponse_InvalidJSON(t *testing.T) {
	// Lines with invalid JSON should be silently skipped
	sseBody := `data: {not valid json}
data: {"result":{"results":[{"labels":[],"metadata":[],"user_data":"{}"}]}}
data: another bad line
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil — should still parse valid lines")
	}

	events := result["events"].([]interface{})
	if len(events) != 1 {
		t.Errorf("expected 1 event (skipping invalid lines), got %d", len(events))
	}
}

func TestParseSSEResponse_ResultWithoutResults(t *testing.T) {
	// Result object without nested results[] — should be treated as a direct event
	sseBody := `data: {"result":{"message":"direct result format","count":42}}
`
	result := parseSSEResponse([]byte(sseBody))
	if result == nil {
		t.Fatal("parseSSEResponse returned nil")
	}

	events := result["events"].([]interface{})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0].(map[string]interface{})
	if event["message"] != "direct result format" {
		t.Errorf("message = %v", event["message"])
	}
}

func TestCleanQueryResults_StripsUserData(t *testing.T) {
	// Simulate what happens when raw_output is false (default):
	// CleanQueryResults → transformLogEntry → extractUserData discards unknown fields
	result := map[string]interface{}{
		"events": []interface{}{
			map[string]interface{}{
				"applicationname": "my-app",
				"subsystemname":   "worker",
				"timestamp":       "2026-03-09T10:00:00Z",
				"severity":        "3",
				"user_data": map[string]interface{}{
					"message":     "order processed",
					"order_id":    "ORD-12345",
					"amount":      99.99,
					"items":       []interface{}{"widget-a", "widget-b"},
					"customer":    map[string]interface{}{"id": "C-100", "tier": "premium"},
					"level":       "INFO",
					"trace_id":    "tr-abc",
					"logger_name": "com.example.orders.OrderProcessor",
				},
			},
		},
	}

	cleaned := CleanQueryResults(result)
	logs := cleaned["logs"].([]interface{})
	entry := logs[0].(map[string]interface{})

	// Known fields should be extracted
	assert.Equal(t, "order processed", entry["message"])
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "tr-abc", entry["trace_id"])

	// Custom fields from user_data are NOT preserved in compact mode
	assert.Nil(t, entry["order_id"], "compact mode should not include custom user_data fields")
	assert.Nil(t, entry["amount"], "compact mode should not include custom user_data fields")
	assert.Nil(t, entry["items"], "compact mode should not include custom user_data fields")
	assert.Nil(t, entry["customer"], "compact mode should not include custom user_data fields")
}

func TestRawOutput_PreservesFullUserData(t *testing.T) {
	// When raw_output is true, CleanQueryResults is skipped entirely.
	// The flattened events from the SSE parser should retain full user_data.
	sseBody := `data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"my-app"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:00:00Z"},{"key":"severity","value":"3"}],"user_data":"{\"message\":\"order processed\",\"order_id\":\"ORD-12345\",\"amount\":99.99,\"items\":[\"widget-a\",\"widget-b\"],\"customer\":{\"id\":\"C-100\",\"tier\":\"premium\"}}"}]}}`

	result := parseSSEResponse([]byte(sseBody))
	assert.NotNil(t, result)

	events := result["events"].([]interface{})
	assert.Len(t, events, 1)

	entry := events[0].(map[string]interface{})

	// In raw mode (no CleanQueryResults), user_data is a parsed map
	ud, ok := entry["user_data"].(map[string]interface{})
	assert.True(t, ok, "user_data should be a parsed map, not a string")
	assert.Equal(t, "order processed", ud["message"])
	assert.Equal(t, "ORD-12345", ud["order_id"])
	assert.Equal(t, 99.99, ud["amount"])

	items, ok := ud["items"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, items, 2)

	customer, ok := ud["customer"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "C-100", customer["id"])
	assert.Equal(t, "premium", customer["tier"])
}

func TestFormatLogsAsMarkdown_IncludesRawOutputHint(t *testing.T) {
	result := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"time":     "2026-03-09T10:00:00Z",
				"severity": "3",
				"app":      "my-app",
				"message":  "test log entry",
			},
		},
	}

	output := formatLogsAsMarkdown(result, "")
	assert.Contains(t, output, "raw_output: true",
		"compact log output should hint about raw_output for full JSON")
	assert.Contains(t, output, "user_data")
}

func TestFormatLogsAsMarkdownTruncated_IncludesRawOutputHint(t *testing.T) {
	result := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"time":     "2026-03-09T10:00:00Z",
				"severity": "3",
				"app":      "my-app",
				"message":  "test log entry",
			},
		},
	}

	output := formatLogsAsMarkdownTruncated(result, "", 100000)
	assert.Contains(t, output, "raw_output: true",
		"truncated log output should hint about raw_output for full JSON")
}

// Benchmark the new parser against realistic payloads
func BenchmarkParseSSEResponse_Realistic(b *testing.B) {
	// Build a realistic SSE response with 200 log entries
	var lines []string
	apps := []string{"api-gateway", "payments-svc", "order-svc", "auth-svc", "notification-svc"}
	severities := []string{"3", "4", "5"}
	for i := 0; i < 200; i++ {
		app := apps[i%len(apps)]
		sev := severities[i%len(severities)]
		ud := fmt.Sprintf(`{"message":"Request processed in %dms for endpoint /api/v2/resource/%d","level":"INFO","trace_id":"trace-%d","span_id":"span-%d"}`, i*10, i, i, i)
		udJSON, _ := json.Marshal(ud)
		// udJSON includes outer quotes since ud is already a string; we need the raw escaped version
		line := fmt.Sprintf(`data: {"result":{"results":[{"labels":[{"key":"applicationname","value":"%s"},{"key":"subsystemname","value":"spring-boot"}],"metadata":[{"key":"timestamp","value":"2026-03-09T10:%02d:%02d.000Z"},{"key":"severity","value":"%s"}],"user_data":%s}]}}`,
			app, i/60, i%60, sev, string(udJSON))
		lines = append(lines, line)
	}
	sseBody := []byte(strings.Join(lines, "\n") + "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseSSEResponse(sseBody)
	}
}
