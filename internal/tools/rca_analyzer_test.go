package tools

import (
	"testing"
	"time"
)

func TestExtractLogTemplate(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		wantTemplate   string
		wantConsistent bool // template should be consistent for similar messages
	}{
		{
			name:           "UUID extraction",
			message:        "Failed to process request 550e8400-e29b-41d4-a716-446655440000",
			wantTemplate:   "Failed to process request <UUID>",
			wantConsistent: true,
		},
		{
			name:           "IP address extraction",
			message:        "Connection from 192.168.1.100 refused",
			wantTemplate:   "Connection from <IP> refused",
			wantConsistent: true,
		},
		{
			name:           "Timestamp extraction",
			message:        "Event at 2024-01-15T10:30:00Z processed",
			wantTemplate:   "Event at <TIME> processed",
			wantConsistent: true,
		},
		{
			name:           "Duration extraction",
			message:        "Request took 150ms to complete",
			wantTemplate:   "Request took <DUR> to complete",
			wantConsistent: true,
		},
		{
			name:           "Number extraction",
			message:        "Processed 1234 records in batch 5678",
			wantTemplate:   "Processed <NUM> records in batch <NUM>",
			wantConsistent: true,
		},
		{
			name:           "Quoted string extraction",
			message:        `Error: "connection refused" for host "db-server"`,
			wantTemplate:   "Error: <STR> for host <STR>",
			wantConsistent: true,
		},
		{
			name:           "File path extraction",
			message:        "Failed to read /var/log/app/error.log",
			wantTemplate:   "Failed to read <PATH>",
			wantConsistent: true,
		},
		{
			name:           "Hex ID extraction",
			message:        "Trace ID: abc123def456789012345678 not found",
			wantTemplate:   "Trace ID: <HEX> not found",
			wantConsistent: true,
		},
		{
			name:           "Multiple patterns",
			message:        "User 12345 from 10.0.0.1 requested /api/v1/users at 2024-01-15T10:30:00Z",
			wantTemplate:   "User <NUM> from <IP> requested <PATH> at <TIME>",
			wantConsistent: true,
		},
		{
			name:           "Plain text unchanged",
			message:        "Database connection established successfully",
			wantTemplate:   "Database connection established successfully",
			wantConsistent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, templateID := ExtractLogTemplate(tt.message)

			if template != tt.wantTemplate {
				t.Errorf("ExtractLogTemplate() template = %q, want %q", template, tt.wantTemplate)
			}

			if templateID == "" {
				t.Error("ExtractLogTemplate() templateID should not be empty")
			}

			// Verify consistency - same template should produce same ID
			if tt.wantConsistent {
				_, templateID2 := ExtractLogTemplate(tt.message)
				if templateID != templateID2 {
					t.Errorf("ExtractLogTemplate() templateID not consistent: %s != %s", templateID, templateID2)
				}
			}
		})
	}
}

func TestExtractLogTemplate_Consistency(t *testing.T) {
	// Similar messages should produce the same template
	messages := []string{
		"Failed to process request 550e8400-e29b-41d4-a716-446655440000",
		"Failed to process request 123e4567-e89b-12d3-a456-426614174000",
		"Failed to process request aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}

	var firstTemplate, firstID string
	for i, msg := range messages {
		template, id := ExtractLogTemplate(msg)
		if i == 0 {
			firstTemplate = template
			firstID = id
		} else {
			if template != firstTemplate {
				t.Errorf("Template mismatch: %q != %q", template, firstTemplate)
			}
			if id != firstID {
				t.Errorf("Template ID mismatch: %s != %s", id, firstID)
			}
		}
	}
}

func TestClusterLogs(t *testing.T) {
	events := []interface{}{
		map[string]interface{}{
			"message":  "Connection timeout to 192.168.1.1 after 30s",
			"severity": "ERROR",
			"labels":   map[string]interface{}{"applicationname": "api-gateway"},
		},
		map[string]interface{}{
			"message":  "Connection timeout to 192.168.1.2 after 30s",
			"severity": "ERROR",
			"labels":   map[string]interface{}{"applicationname": "api-gateway"},
		},
		map[string]interface{}{
			"message":  "Connection timeout to 10.0.0.1 after 30s",
			"severity": "ERROR",
			"labels":   map[string]interface{}{"applicationname": "payment-service"},
		},
		map[string]interface{}{
			"message":  "Out of memory: killed process 12345",
			"severity": "CRITICAL",
			"labels":   map[string]interface{}{"applicationname": "worker"},
		},
		map[string]interface{}{
			"message":  "Out of memory: killed process 67890",
			"severity": "CRITICAL",
			"labels":   map[string]interface{}{"applicationname": "worker"},
		},
	}

	clusters := ClusterLogs(events)

	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(clusters))
	}

	// Verify clusters are sorted by severity then count
	if len(clusters) >= 2 {
		// CRITICAL should come before ERROR (higher severity)
		if clusters[0].SeverityNum < clusters[1].SeverityNum {
			t.Error("Clusters should be sorted by severity (highest first)")
		}
	}

	// Find the timeout cluster
	var timeoutCluster *LogCluster
	for _, c := range clusters {
		if c.RootCause == "TIMEOUT" {
			timeoutCluster = c
			break
		}
	}

	if timeoutCluster == nil {
		t.Fatal("Expected to find TIMEOUT cluster")
	}

	if timeoutCluster.Count != 3 {
		t.Errorf("Expected timeout cluster count = 3, got %d", timeoutCluster.Count)
	}

	if len(timeoutCluster.Apps) != 2 {
		t.Errorf("Expected 2 apps in timeout cluster, got %d", len(timeoutCluster.Apps))
	}
}

func TestInferRootCause(t *testing.T) {
	tests := []struct {
		template string
		want     string
	}{
		{"Connection timeout to <IP> after <DUR>", "TIMEOUT"},
		{"Out of memory: killed process <NUM>", "MEMORY_PRESSURE"},
		{"Connection refused to <IP>:<NUM>", "NETWORK_FAILURE"},
		{"Permission denied for user <STR>", "AUTH_FAILURE"},
		{"Disk full: no space left on device", "STORAGE_FAILURE"},
		{"Rate limit exceeded for client <UUID>", "RATE_LIMITED"},
		{"DNS resolution failed for <STR>", "DNS_FAILURE"},
		{"Certificate expired for <STR>", "TLS_FAILURE"},
		{"Database deadlock detected", "DATABASE_FAILURE"},
		{"Pod evicted due to resource pressure", "K8S_ORCHESTRATION"},
		{"Random unknown error occurred", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			got := inferRootCause(tt.template)
			if got != tt.want {
				t.Errorf("inferRootCause(%q) = %q, want %q", tt.template, got, tt.want)
			}
		})
	}
}

func TestLogEntryPool(t *testing.T) {
	// Acquire entry from pool
	entry := AcquireLogEntry()
	if entry == nil {
		t.Fatal("AcquireLogEntry() returned nil")
	}

	// Populate entry
	entry.Timestamp = time.Now()
	entry.Severity = "ERROR"
	entry.Message = "Test message"
	entry.Labels["app"] = "test"

	// Release back to pool
	ReleaseLogEntry(entry)

	// Acquire again - should get a clean entry
	entry2 := AcquireLogEntry()
	if entry2.Severity != "" {
		t.Error("Entry from pool should be reset")
	}
	if len(entry2.Labels) != 0 {
		t.Error("Entry labels should be cleared")
	}

	ReleaseLogEntry(entry2)
}

func TestVerificationTrace(t *testing.T) {
	startTime := time.Now().Add(-100 * time.Millisecond)
	trace := NewVerificationTrace("test_tool", startTime)

	if trace.ToolName != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", trace.ToolName)
	}

	if trace.ExecutionMs < 100 {
		t.Errorf("Expected execution time >= 100ms, got %d", trace.ExecutionMs)
	}

	if trace.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	if !trace.RetryAllowed {
		t.Error("RetryAllowed should default to true")
	}
}

func TestOrDefault(t *testing.T) {
	tests := []struct {
		s    string
		def  string
		want string
	}{
		{"value", "default", "value"},
		{"", "default", "default"},
		{"  ", "default", "  "}, // whitespace is not empty
	}

	for _, tt := range tests {
		got := orDefault(tt.s, tt.def)
		if got != tt.want {
			t.Errorf("orDefault(%q, %q) = %q, want %q", tt.s, tt.def, got, tt.want)
		}
	}
}

func BenchmarkExtractLogTemplate(b *testing.B) {
	message := "Failed to process request 550e8400-e29b-41d4-a716-446655440000 from 192.168.1.100 at 2024-01-15T10:30:00Z"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractLogTemplate(message)
	}
}

func BenchmarkClusterLogs(b *testing.B) {
	// Create 100 events with 10 patterns
	events := make([]interface{}, 100)
	patterns := []string{
		"Connection timeout to 192.168.1.%d after 30s",
		"Out of memory: killed process %d",
		"Permission denied for user user%d",
		"Rate limit exceeded for client %d",
		"Database connection failed: error %d",
	}

	for i := 0; i < 100; i++ {
		pattern := patterns[i%len(patterns)] //nolint:gosec // safe: modulo ensures in-bounds
		events[i] = map[string]interface{}{
			"message":  pattern,
			"severity": "ERROR",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClusterLogs(events)
	}
}

func BenchmarkLogEntryPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := AcquireLogEntry()
		entry.Severity = "ERROR"
		entry.Message = "Test message"
		ReleaseLogEntry(entry)
	}
}

// ============================================================================
// SOTA 2025 Tests: Comprehensive RCA Analyzer Testing
// ============================================================================

func TestParseEventIntoEntry(t *testing.T) {
	tests := []struct {
		name      string
		event     map[string]interface{}
		wantMsg   string
		wantSev   string
		wantApp   string
		wantTrace string
	}{
		{
			name: "complete event with all fields",
			event: map[string]interface{}{
				"message":  "Connection failed to database",
				"severity": "ERROR",
				"labels": map[string]interface{}{
					"applicationname": "payment-service",
					"subsystemname":   "db-connector",
				},
				"user_data": map[string]interface{}{
					"trace_id": "abc123def456",
					"span_id":  "span789",
				},
			},
			wantMsg:   "Connection failed to database",
			wantSev:   "ERROR",
			wantApp:   "payment-service",
			wantTrace: "abc123def456", // pragma: allowlist secret
		},
		{
			name: "event with nested user_data message",
			event: map[string]interface{}{
				"user_data": map[string]interface{}{
					"message": "Nested message content",
				},
				"severity": "WARNING",
			},
			wantMsg:   "Nested message content",
			wantSev:   "WARNING",
			wantApp:   "",
			wantTrace: "",
		},
		{
			name: "event with numeric severity",
			event: map[string]interface{}{
				"message":  "Numeric severity test",
				"severity": float64(5), // ERROR
			},
			wantMsg:   "Numeric severity test",
			wantSev:   "ERROR",
			wantApp:   "",
			wantTrace: "",
		},
		{
			name: "event with traceId (camelCase)",
			event: map[string]interface{}{
				"message": "Trace test",
				"user_data": map[string]interface{}{
					"traceId": "camelCaseTraceId",
				},
			},
			wantMsg:   "Trace test",
			wantTrace: "camelCaseTraceId",
		},
		{
			name: "event with direct trace_id field",
			event: map[string]interface{}{
				"message":  "Direct trace field",
				"trace_id": "directTraceId",
			},
			wantMsg:   "Direct trace field",
			wantTrace: "directTraceId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := AcquireLogEntry()
			defer ReleaseLogEntry(entry)

			parseEventIntoEntry(tt.event, entry)

			if entry.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMsg)
			}
			if tt.wantSev != "" && entry.Severity != tt.wantSev {
				t.Errorf("Severity = %q, want %q", entry.Severity, tt.wantSev)
			}
			if entry.App != tt.wantApp {
				t.Errorf("App = %q, want %q", entry.App, tt.wantApp)
			}
			if tt.wantTrace != "" && entry.TraceID != tt.wantTrace {
				t.Errorf("TraceID = %q, want %q", entry.TraceID, tt.wantTrace)
			}
		})
	}
}

func TestExtractTraceContext(t *testing.T) {
	tests := []struct {
		name        string
		event       map[string]interface{}
		wantTraceID string
		wantSpanID  string
	}{
		{
			name: "trace_id in user_data",
			event: map[string]interface{}{
				"user_data": map[string]interface{}{
					"trace_id": "user-data-trace",
					"span_id":  "user-data-span",
				},
			},
			wantTraceID: "user-data-trace",
			wantSpanID:  "user-data-span",
		},
		{
			name: "traceId camelCase",
			event: map[string]interface{}{
				"user_data": map[string]interface{}{
					"traceId": "camel-trace",
					"spanId":  "camel-span",
				},
			},
			wantTraceID: "camel-trace",
			wantSpanID:  "camel-span",
		},
		{
			name: "traceID uppercase ID",
			event: map[string]interface{}{
				"user_data": map[string]interface{}{
					"traceID": "upper-trace",
					"spanID":  "upper-span",
				},
			},
			wantTraceID: "upper-trace",
			wantSpanID:  "upper-span",
		},
		{
			name: "direct fields fallback",
			event: map[string]interface{}{
				"trace_id": "direct-trace",
				"span_id":  "direct-span",
			},
			wantTraceID: "direct-trace",
			wantSpanID:  "direct-span",
		},
		{
			name: "mixed: user_data trace, direct span",
			event: map[string]interface{}{
				"user_data": map[string]interface{}{
					"trace_id": "user-data-trace",
				},
				"span_id": "direct-span",
			},
			wantTraceID: "user-data-trace",
			wantSpanID:  "direct-span",
		},
		{
			name:        "empty event",
			event:       map[string]interface{}{},
			wantTraceID: "",
			wantSpanID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traceID, spanID := extractTraceContext(tt.event)
			if traceID != tt.wantTraceID {
				t.Errorf("traceID = %q, want %q", traceID, tt.wantTraceID)
			}
			if spanID != tt.wantSpanID {
				t.Errorf("spanID = %q, want %q", spanID, tt.wantSpanID)
			}
		})
	}
}

func TestExtractSubsystemFromEvent(t *testing.T) {
	tests := []struct {
		name  string
		event map[string]interface{}
		want  string
	}{
		{
			name: "subsystemname in labels",
			event: map[string]interface{}{
				"labels": map[string]interface{}{
					"subsystemname": "auth-service",
				},
			},
			want: "auth-service",
		},
		{
			name: "direct subsystemname",
			event: map[string]interface{}{
				"subsystemname": "direct-subsystem",
			},
			want: "direct-subsystem",
		},
		{
			name: "direct subsystem (short form)",
			event: map[string]interface{}{
				"subsystem": "short-subsystem",
			},
			want: "short-subsystem",
		},
		{
			name:  "no subsystem",
			event: map[string]interface{}{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSubsystemFromEvent(tt.event)
			if got != tt.want {
				t.Errorf("extractSubsystemFromEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClusterLogs_WithPooledEntries(t *testing.T) {
	// Test that ClusterLogs correctly uses pooled entries and produces correct results
	events := make([]interface{}, 1000)
	now := time.Now()

	for i := 0; i < 1000; i++ {
		severity := "INFO"
		if i%10 == 0 {
			severity = "ERROR"
		}
		events[i] = map[string]interface{}{
			"message":   "Request processed for user-" + string(rune('A'+(i%26))),
			"severity":  severity,
			"timestamp": now.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			"labels": map[string]interface{}{
				"applicationname": "test-app",
			},
		}
	}

	clusters := ClusterLogs(events)

	// Should have 26 clusters (one per letter A-Z)
	if len(clusters) != 26 {
		t.Errorf("Expected 26 clusters, got %d", len(clusters))
	}

	// Each cluster should have count of about 38-39 (1000/26)
	totalCount := 0
	for _, c := range clusters {
		totalCount += c.Count
	}
	if totalCount != 1000 {
		t.Errorf("Total cluster count = %d, want 1000", totalCount)
	}
}

func TestClusterLogs_EmptyInput(t *testing.T) {
	clusters := ClusterLogs(nil)
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters for nil input, got %d", len(clusters))
	}

	clusters = ClusterLogs([]interface{}{})
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters for empty input, got %d", len(clusters))
	}
}

func TestClusterLogs_InvalidEvents(t *testing.T) {
	events := []interface{}{
		"string event",                        // invalid type
		123,                                   // invalid type
		nil,                                   // nil
		map[string]interface{}{},              // empty map (no message)
		map[string]interface{}{"message": ""}, // empty message
	}

	clusters := ClusterLogs(events)
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters for invalid events, got %d", len(clusters))
	}
}

func TestClusterLogs_TimeTracking(t *testing.T) {
	now := time.Now()
	events := []interface{}{
		map[string]interface{}{
			"message":   "Test message A",
			"timestamp": now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		map[string]interface{}{
			"message":   "Test message A",
			"timestamp": now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		map[string]interface{}{
			"message":   "Test message A",
			"timestamp": now.Format(time.RFC3339),
		},
	}

	clusters := ClusterLogs(events)
	if len(clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	expectedFirst := now.Add(-10 * time.Minute)
	expectedLast := now

	// Allow 1 second tolerance for time parsing
	if cluster.FirstSeen.Sub(expectedFirst).Abs() > time.Second {
		t.Errorf("FirstSeen = %v, want ~%v", cluster.FirstSeen, expectedFirst)
	}
	if cluster.LastSeen.Sub(expectedLast).Abs() > time.Second {
		t.Errorf("LastSeen = %v, want ~%v", cluster.LastSeen, expectedLast)
	}
}

// ============================================================================
// Benchmarks for SOTA 2025 Performance Validation
// ============================================================================

func BenchmarkClusterLogs_1000Events(b *testing.B) {
	events := generateBenchmarkEvents(1000, 10)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ClusterLogs(events)
	}
}

func BenchmarkClusterLogs_10000Events(b *testing.B) {
	events := generateBenchmarkEvents(10000, 50)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ClusterLogs(events)
	}
}

func BenchmarkClusterLogs_Parallel(b *testing.B) {
	events := generateBenchmarkEvents(1000, 10)
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ClusterLogs(events)
		}
	})
}

func BenchmarkParseEventIntoEntry(b *testing.B) {
	event := map[string]interface{}{
		"message":  "Connection timeout to 192.168.1.100 after 30s",
		"severity": "ERROR",
		"labels": map[string]interface{}{
			"applicationname": "api-gateway",
			"subsystemname":   "http-handler",
		},
		"user_data": map[string]interface{}{
			"trace_id": "abc123def456",
			"span_id":  "span789",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry := AcquireLogEntry()
		parseEventIntoEntry(event, entry)
		ReleaseLogEntry(entry)
	}
}

func BenchmarkLogEntryPool_Parallel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := AcquireLogEntry()
			entry.Severity = "ERROR"
			entry.Message = "Test message for parallel benchmark"
			entry.App = "benchmark-app"
			ReleaseLogEntry(entry)
		}
	})
}

// generateBenchmarkEvents creates test events for benchmarks
func generateBenchmarkEvents(count int, patterns int) []interface{} {
	events := make([]interface{}, count)
	templatePatterns := []string{
		"Connection timeout to 192.168.1.%d after 30s",
		"Out of memory: killed process %d",
		"Permission denied for user user%d",
		"Rate limit exceeded for client %d",
		"Database connection failed: error code %d",
		"File not found: /var/log/app/%d.log",
		"Request took %dms to complete",
		"Retry attempt %d failed",
		"Queue depth exceeded threshold: %d",
		"CPU usage at %d percent",
	}

	severities := []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}

	for i := 0; i < count; i++ {
		patternIdx := i % min(patterns, len(templatePatterns))
		events[i] = map[string]interface{}{
			"message":  templatePatterns[patternIdx],
			"severity": severities[i%len(severities)],
			"labels": map[string]interface{}{
				"applicationname": "app-" + string(rune('A'+(i%5))),
			},
			"timestamp": time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
		}
	}

	return events
}
