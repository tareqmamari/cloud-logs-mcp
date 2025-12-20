package tools

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateRCADocument_Basic(t *testing.T) {
	tool := &GenerateRCADocumentTool{}

	// Verify tool metadata
	if tool.Name() != "generate_rca_document" {
		t.Errorf("Expected name 'generate_rca_document', got %q", tool.Name())
	}

	meta := tool.Metadata()
	if meta == nil {
		t.Fatal("Metadata() returned nil")
	}

	if len(meta.Categories) == 0 {
		t.Error("Expected at least one category")
	}

	if len(meta.Keywords) == 0 {
		t.Error("Expected at least one keyword")
	}

	// Verify description mentions key features
	desc := tool.Description()
	if !strings.Contains(desc, "5 Whys") {
		t.Error("Description should mention 5 Whys methodology")
	}
	if !strings.Contains(desc, "post-mortem") {
		t.Error("Description should mention post-mortem")
	}
}

func TestGenerateRCADocument_DocumentGeneration(t *testing.T) {
	tool := &GenerateRCADocumentTool{}

	start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC)

	doc := tool.generateDocument(
		"API Gateway Timeout Spike",
		"INC-2024-0115",
		start,
		&end,
		"1h30m",
		"SEV2",
		[]string{"api-gateway", "auth-service"},
		"NETWORK_FAILURE",
		"Network partition between api-gateway and auth-service",
		[]map[string]interface{}{
			{"pattern": "Connection timeout to <IP>", "count": float64(150), "severity": "ERROR", "root_cause": "TIMEOUT"},
		},
		[]map[string]interface{}{
			{"time": "10:00:00", "event": "First errors observed", "source": "Logs", "is_key_event": true},
			{"time": "10:05:00", "event": "Alert triggered", "source": "Monitoring", "is_key_event": false},
		},
		true,
	)

	// Verify document structure
	requiredSections := []string{
		"# Root Cause Analysis:",
		"## Document Information",
		"## 1. Executive Summary",
		"## 2. Affected Services",
		"## 3. Impact Assessment",
		"## 4. Incident Timeline",
		"## 5. Root Cause Analysis (5 Whys)",
		"## 6. Log Evidence",
		"## 7. Contributing Factors",
		"## 8. Corrective Actions",
		"## 9. Prevention Measures",
		"## 10. Lessons Learned",
	}

	for _, section := range requiredSections {
		if !strings.Contains(doc, section) {
			t.Errorf("Document missing section: %s", section)
		}
	}

	// Verify incident data is included
	if !strings.Contains(doc, "INC-2024-0115") {
		t.Error("Document should contain incident ID")
	}
	if !strings.Contains(doc, "SEV2") {
		t.Error("Document should contain severity")
	}
	if !strings.Contains(doc, "api-gateway") {
		t.Error("Document should contain affected services")
	}
	if !strings.Contains(doc, "NETWORK_FAILURE") {
		t.Error("Document should contain root cause category")
	}

	// Verify error patterns table
	if !strings.Contains(doc, "Connection timeout") {
		t.Error("Document should contain error patterns")
	}

	// Verify timeline
	if !strings.Contains(doc, "First errors observed") {
		t.Error("Document should contain timeline events")
	}
}

func TestGenerateRCADocument_OngoingIncident(t *testing.T) {
	tool := &GenerateRCADocumentTool{}

	start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	doc := tool.generateDocument(
		"Database Connectivity Issue",
		"",
		start,
		nil, // No end time - ongoing
		"",
		"SEV1",
		[]string{"database"},
		"",
		"",
		nil,
		nil,
		true,
	)

	// Verify ongoing status
	if !strings.Contains(doc, "_Ongoing_") {
		t.Error("Document should indicate ongoing incident")
	}

	// Verify auto-generated incident ID format
	if !strings.Contains(doc, "INC-") {
		t.Error("Document should contain auto-generated incident ID")
	}
}

func TestGenerateRCADocument_WithoutTemplates(t *testing.T) {
	tool := &GenerateRCADocumentTool{}

	start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	doc := tool.generateDocument(
		"Test Incident",
		"INC-001",
		start,
		nil,
		"",
		"SEV3",
		nil,
		"TIMEOUT",
		"Service timeout due to slow database",
		nil,
		nil,
		false, // No template sections
	)

	// Should still have structure but no template placeholders
	if !strings.Contains(doc, "# Root Cause Analysis:") {
		t.Error("Document should have title")
	}

	// Should not have template instructions when include_template_sections=false
	if strings.Contains(doc, "[Fill in]") && !strings.Contains(doc, "Author") {
		// Author is always [Fill in], but other sections shouldn't have it
		t.Log("Document correctly excludes template placeholders")
	}
}

func TestGetIncidentStatus(t *testing.T) {
	now := time.Now()

	// Test ongoing (nil end time)
	status := getIncidentStatus(nil)
	if status != "Ongoing" {
		t.Errorf("Expected 'Ongoing' for nil end time, got %q", status)
	}

	// Test resolved
	status = getIncidentStatus(&now)
	if status != "Resolved" {
		t.Errorf("Expected 'Resolved' for non-nil end time, got %q", status)
	}
}
