package tools

import (
	"testing"
	"time"
)

func TestTraceAnalysis(t *testing.T) {
	tool := &GetTraceContextTool{}

	events := []interface{}{
		map[string]interface{}{
			"timestamp":       "2024-01-15T10:00:00Z",
			"severity":        "INFO",
			"message":         "Request received",
			"applicationname": "api-gateway",
			"user_data":       map[string]interface{}{"span_id": "span1"},
		},
		map[string]interface{}{
			"timestamp":       "2024-01-15T10:00:01Z",
			"severity":        "INFO",
			"message":         "Calling auth service",
			"applicationname": "api-gateway",
			"user_data":       map[string]interface{}{"span_id": "span2"},
		},
		map[string]interface{}{
			"timestamp":       "2024-01-15T10:00:02Z",
			"severity":        "ERROR",
			"message":         "Auth service timeout",
			"applicationname": "auth-service",
			"user_data":       map[string]interface{}{"span_id": "span3"},
		},
		map[string]interface{}{
			"timestamp":       "2024-01-15T10:00:03Z",
			"severity":        "ERROR",
			"message":         "Request failed",
			"applicationname": "api-gateway",
			"user_data":       map[string]interface{}{"span_id": "span4"},
		},
	}

	analysis := tool.analyzeTrace(events)

	// Check services are in order of appearance
	if len(analysis.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(analysis.Services))
	}
	if len(analysis.Services) >= 1 && analysis.Services[0] != "api-gateway" {
		t.Errorf("Expected first service to be 'api-gateway', got %q", analysis.Services[0])
	}

	// Check errors are tracked
	if len(analysis.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(analysis.Errors))
	}

	// Check error counts by service
	if analysis.ServiceErrors["auth-service"] != 1 {
		t.Errorf("Expected 1 error for auth-service, got %d", analysis.ServiceErrors["auth-service"])
	}
	if analysis.ServiceErrors["api-gateway"] != 1 {
		t.Errorf("Expected 1 error for api-gateway, got %d", analysis.ServiceErrors["api-gateway"])
	}

	// Check timeline is sorted
	if len(analysis.Timeline) != 4 {
		t.Errorf("Expected 4 timeline entries, got %d", len(analysis.Timeline))
	}
	for i := 1; i < len(analysis.Timeline); i++ {
		if analysis.Timeline[i].Timestamp.Before(analysis.Timeline[i-1].Timestamp) {
			t.Error("Timeline should be sorted by timestamp")
		}
	}

	// Check duration is calculated
	if analysis.Duration == "unknown" || analysis.Duration == "" {
		t.Error("Duration should be calculated")
	}
}

func TestCausalGraph(t *testing.T) {
	tool := &AnalyzeCausalChainTool{}

	// Create clusters with different timestamps and root causes
	clusters := []*LogCluster{
		{
			TemplateID: "template1",
			Template:   "Connection refused to database",
			RootCause:  "NETWORK_FAILURE",
			Count:      10,
			FirstSeen:  time.Now().Add(-5 * time.Minute),
			Apps:       []string{"db-client"},
		},
		{
			TemplateID: "template2",
			Template:   "Request timeout waiting for response",
			RootCause:  "TIMEOUT",
			Count:      50,
			FirstSeen:  time.Now().Add(-3 * time.Minute),
			Apps:       []string{"api-gateway"},
		},
		{
			TemplateID: "template3",
			Template:   "Service unavailable",
			RootCause:  "UNKNOWN",
			Count:      100,
			FirstSeen:  time.Now().Add(-1 * time.Minute),
			Apps:       []string{"frontend"},
		},
	}

	graph := tool.buildCausalGraph(clusters, nil)

	// Should have nodes for all clusters
	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}

	// First cluster (earliest + fundamental cause) should be root cause
	if len(graph.RootCauses) == 0 {
		t.Fatal("Expected at least one root cause candidate")
	}

	// The network failure (earliest and fundamental) should be highest confidence
	foundNetworkAsRoot := false
	for _, rc := range graph.RootCauses {
		if rc.Cluster.RootCause == "NETWORK_FAILURE" {
			foundNetworkAsRoot = true
			if rc.Confidence < 0.5 {
				t.Error("Network failure should have high confidence as root cause")
			}
			break
		}
	}
	if !foundNetworkAsRoot {
		t.Error("NETWORK_FAILURE should be identified as potential root cause")
	}

	// Check propagation path includes all apps
	if len(graph.PropagationPath) == 0 {
		t.Error("Propagation path should not be empty")
	}
}

func TestCausalGraphScoring(t *testing.T) {
	tool := &AnalyzeCausalChainTool{}

	now := time.Now()

	// Memory error at T-10min (fundamental cause, early)
	// Timeout at T-5min (symptom, later)
	clusters := []*LogCluster{
		{
			TemplateID: "mem",
			Template:   "Out of memory error",
			RootCause:  "MEMORY_PRESSURE",
			Count:      5,
			FirstSeen:  now.Add(-10 * time.Minute),
			Apps:       []string{"worker"},
		},
		{
			TemplateID: "timeout",
			Template:   "Request timeout",
			RootCause:  "TIMEOUT",
			Count:      50,
			FirstSeen:  now.Add(-5 * time.Minute),
			Apps:       []string{"api"},
		},
	}

	graph := tool.buildCausalGraph(clusters, nil)

	// Memory error should be identified as more likely root cause
	// because it's earlier AND it's a fundamental cause type
	if len(graph.RootCauses) == 0 {
		t.Fatal("Expected root causes")
	}

	topCause := graph.RootCauses[0]
	if topCause.Cluster.RootCause != "MEMORY_PRESSURE" {
		t.Errorf("Expected MEMORY_PRESSURE as top root cause, got %s", topCause.Cluster.RootCause)
	}
}

func TestTraceEntryExtraction(t *testing.T) {
	// Test various event formats
	events := []interface{}{
		// Standard format
		map[string]interface{}{
			"timestamp":       "2024-01-15T10:00:00Z",
			"severity":        "ERROR",
			"message":         "Error message",
			"applicationname": "app1",
		},
		// Nested user_data
		map[string]interface{}{
			"timestamp": "2024-01-15T10:00:01Z",
			"metadata":  map[string]interface{}{"severity": "WARNING"},
			"user_data": map[string]interface{}{"message": "User data message"},
			"labels":    map[string]interface{}{"applicationname": "app2"},
		},
		// Alternative field names
		map[string]interface{}{
			"@timestamp": "2024-01-15T10:00:02Z",
			"severity":   float64(5), // numeric severity
			"msg":        "Msg field",
			"app":        "app3",
		},
	}

	tool := &GetTraceContextTool{}
	analysis := tool.analyzeTrace(events)

	if len(analysis.Timeline) != 3 {
		t.Errorf("Expected 3 timeline entries, got %d", len(analysis.Timeline))
	}

	// Verify services were extracted
	if len(analysis.Services) < 1 {
		t.Error("Expected at least 1 service extracted")
	}
}

func TestEmptyTraceAnalysis(t *testing.T) {
	tool := &GetTraceContextTool{}

	analysis := tool.analyzeTrace([]interface{}{})

	if len(analysis.Services) != 0 {
		t.Error("Empty events should produce empty services")
	}
	if len(analysis.Timeline) != 0 {
		t.Error("Empty events should produce empty timeline")
	}
	if analysis.Duration != "unknown" {
		t.Errorf("Empty events should have unknown duration, got %s", analysis.Duration)
	}
}

func TestCausalGraphEmptyClusters(t *testing.T) {
	tool := &AnalyzeCausalChainTool{}

	graph := tool.buildCausalGraph([]*LogCluster{}, nil)

	if len(graph.Nodes) != 0 {
		t.Error("Empty clusters should produce empty graph")
	}
	if len(graph.RootCauses) != 0 {
		t.Error("Empty clusters should produce no root causes")
	}
}

func TestToolMetadata(t *testing.T) {
	tests := []struct {
		name     string
		tool     interface{ Metadata() *ToolMetadata }
		wantCats []ToolCategory
	}{
		{
			name:     "GetTraceContextTool",
			tool:     &GetTraceContextTool{},
			wantCats: []ToolCategory{CategoryQuery, CategoryObservability},
		},
		{
			name:     "AnalyzeCausalChainTool",
			tool:     &AnalyzeCausalChainTool{},
			wantCats: []ToolCategory{CategoryWorkflow, CategoryObservability, CategoryAIHelper},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := tt.tool.Metadata()
			if meta == nil {
				t.Fatal("Metadata() returned nil")
			}

			if len(meta.Categories) != len(tt.wantCats) {
				t.Errorf("Expected %d categories, got %d", len(tt.wantCats), len(meta.Categories))
			}

			if len(meta.Keywords) == 0 {
				t.Error("Keywords should not be empty")
			}

			if len(meta.UseCases) == 0 {
				t.Error("UseCases should not be empty")
			}

			if len(meta.RelatedTools) == 0 {
				t.Error("RelatedTools should not be empty")
			}
		})
	}
}
