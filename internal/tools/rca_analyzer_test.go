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
