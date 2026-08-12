package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/audit"
)

// getAuditLogToolResponse mirrors auditLogResponse for test assertions
// without depending on its unexported fields directly.
type getAuditLogToolResponse struct {
	Enabled bool   `json:"audit_logging_enabled"`
	Count   int    `json:"count"`
	Note    string `json:"note,omitempty"`
	Entries []struct {
		Tool      string `json:"tool"`
		Operation string `json:"operation"`
		Success   bool   `json:"success"`
		ErrorMsg  string `json:"error_message,omitempty"`
		TraceID   string `json:"trace_id,omitempty"`
	} `json:"entries"`
}

func decodeAuditLogResult(t *testing.T, result *mcp.CallToolResult) getAuditLogToolResponse {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("expected non-empty result content")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("result content is not TextContent: %T", result.Content[0])
	}
	var resp getAuditLogToolResponse
	if err := json.Unmarshal([]byte(textContent.Text), &resp); err != nil {
		t.Fatalf("failed to unmarshal tool output as JSON: %v\noutput: %s", err, textContent.Text)
	}
	return resp
}

// TestGetAuditLogTool_Execute_ReturnsRealEntries verifies Task 10's Fix 5:
// GetAuditLogTool.Execute now returns the server's real recent audit
// entries (via the audit logger injected into context, the same dependency
// path other context-scoped resources like the API client use) instead of
// always returning static placeholder help text regardless of arguments.
func TestGetAuditLogTool_Execute_ReturnsRealEntries(t *testing.T) {
	logger := audit.NewLogger(zap.NewNop(), true)

	logger.LogToolExecution(context.Background(), "query_logs", "query", "", "", true, 10*time.Millisecond, nil, "hash1")
	logger.LogToolExecution(context.Background(), "create_alert", "create", "", "", false, 5*time.Millisecond, errors.New("boom"), "hash2")

	tool := NewGetAuditLogTool(nil, zap.NewNop())
	ctx := WithAuditLogger(context.Background(), logger)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	resp := decodeAuditLogResult(t, result)
	if !resp.Enabled {
		t.Fatal("expected audit_logging_enabled=true")
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2 entries, got %d", resp.Count)
	}

	toolNames := map[string]bool{}
	for _, e := range resp.Entries {
		toolNames[e.Tool] = true
	}
	if !toolNames["query_logs"] || !toolNames["create_alert"] {
		t.Errorf("expected both logged tool names to appear, got entries: %+v", resp.Entries)
	}
}

// TestGetAuditLogTool_Execute_FiltersByTool verifies the "tool" argument
// (part of the tool's pre-existing, unchanged input schema) actually filters
// results now that Execute is wired to the real audit logger.
func TestGetAuditLogTool_Execute_FiltersByTool(t *testing.T) {
	logger := audit.NewLogger(zap.NewNop(), true)
	logger.LogToolExecution(context.Background(), "query_logs", "query", "", "", true, time.Millisecond, nil, "h1")
	logger.LogToolExecution(context.Background(), "create_alert", "create", "", "", true, time.Millisecond, nil, "h2")

	tool := NewGetAuditLogTool(nil, zap.NewNop())
	ctx := WithAuditLogger(context.Background(), logger)

	result, err := tool.Execute(ctx, map[string]interface{}{"tool": "create_alert"})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	resp := decodeAuditLogResult(t, result)
	if resp.Count != 1 {
		t.Fatalf("expected 1 filtered entry, got %d: %+v", resp.Count, resp.Entries)
	}
	if resp.Entries[0].Tool != "create_alert" {
		t.Errorf("expected filtered entry to be create_alert, got %q", resp.Entries[0].Tool)
	}
}

// TestGetAuditLogTool_Execute_MasksErrors guards the masking requirement
// from Task 10 Fix 5: any error text carried by an audit entry must be
// sanitized before it reaches the tool's output, never leaking a raw
// credential-shaped value even if it somehow bypassed masking upstream.
func TestGetAuditLogTool_Execute_MasksErrors(t *testing.T) {
	logger := audit.NewLogger(zap.NewNop(), true)
	secretErr := errors.New("request failed: api_key=abcdefghij1234567890secretvalue") // pragma: allowlist secret
	logger.LogToolExecution(context.Background(), "query_logs", "query", "", "", false, time.Millisecond, secretErr, "h1")

	tool := NewGetAuditLogTool(nil, zap.NewNop())
	ctx := WithAuditLogger(context.Background(), logger)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	resp := decodeAuditLogResult(t, result)
	if resp.Count != 1 {
		t.Fatalf("expected 1 entry, got %d", resp.Count)
	}
	if strings.Contains(resp.Entries[0].ErrorMsg, "abcdefghij1234567890secretvalue") { // pragma: allowlist secret
		t.Errorf("audit log entry leaks raw secret: %q", resp.Entries[0].ErrorMsg)
	}
	if !strings.Contains(resp.Entries[0].ErrorMsg, "REDACTED") {
		t.Errorf("expected masked error message, got: %q", resp.Entries[0].ErrorMsg)
	}
}

// TestToAuditLogEntryView_MasksResourceResourceIDAndMetadata guards Fix 4:
// toAuditLogEntryView previously only masked ErrorMsg, leaving Resource,
// ResourceID, and each Metadata value to pass through unmasked. No current
// writer (LogToolExecution) populates those fields, so this was inert
// today - but the code's own comment cites "masking as a hard boundary", so
// it must hold defensively for any future writer. InputHash is a hash, not
// sensitive free text, and must be left untouched; Metadata keys are field
// names and must also be left untouched (only values are masked).
func TestToAuditLogEntryView_MasksResourceResourceIDAndMetadata(t *testing.T) {
	const (
		secretPayload = "abcdefghij1234567890secretvalue" // pragma: allowlist secret
		secret        = "api_key=" + secretPayload        // pragma: allowlist secret
	)

	entry := audit.Entry{
		Timestamp:  time.Now(),
		Tool:       "query_logs",
		Operation:  "query",
		Resource:   secret,
		ResourceID: secret,
		Success:    true,
		InputHash:  "deadbeef",
		Metadata: map[string]interface{}{
			"note":      secret,
			"other":     "harmless value",
			"count":     42,
			"api_key":   secret, // key itself must be left alone; only the value is masked
			"api_token": secret,
		},
	}

	view := toAuditLogEntryView(entry)

	if strings.Contains(view.Resource, secretPayload) {
		t.Errorf("Resource leaks raw secret: %q", view.Resource)
	}
	if !strings.Contains(view.Resource, "REDACTED") {
		t.Errorf("expected Resource to be masked, got: %q", view.Resource)
	}

	if strings.Contains(view.ResourceID, secretPayload) {
		t.Errorf("ResourceID leaks raw secret: %q", view.ResourceID)
	}
	if !strings.Contains(view.ResourceID, "REDACTED") {
		t.Errorf("expected ResourceID to be masked, got: %q", view.ResourceID)
	}

	if view.InputHash != "deadbeef" {
		t.Errorf("InputHash should be left untouched, got: %q", view.InputHash)
	}

	for _, key := range []string{"note", "api_key", "api_token"} {
		v, ok := view.Metadata[key]
		if !ok {
			t.Fatalf("expected Metadata key %q to survive", key)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("Metadata[%q] is not a string: %T", key, v)
		}
		if strings.Contains(s, secretPayload) {
			t.Errorf("Metadata[%q] leaks raw secret: %q", key, s)
		}
		if !strings.Contains(s, "REDACTED") {
			t.Errorf("expected Metadata[%q] to be masked, got: %q", key, s)
		}
	}

	if _, ok := view.Metadata["api_key"]; !ok {
		t.Error("Metadata key 'api_key' should be left untouched even though its value is masked")
	}
	if view.Metadata["other"] != "harmless value" {
		t.Errorf("Metadata[\"other\"] = %v, want unchanged %q", view.Metadata["other"], "harmless value")
	}
	if view.Metadata["count"] != 42 {
		t.Errorf("Metadata[\"count\"] = %v, want unchanged non-string value 42", view.Metadata["count"])
	}
}

// TestGetAuditLogTool_Execute_NoLoggerInContext verifies the honest-fallback
// path: when no audit logger is available (e.g. audit logging disabled),
// Execute must say so rather than returning entries or the old always-on
// static placeholder text.
func TestGetAuditLogTool_Execute_NoLoggerInContext(t *testing.T) {
	tool := NewGetAuditLogTool(nil, zap.NewNop())

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}

	resp := decodeAuditLogResult(t, result)
	if resp.Enabled {
		t.Error("expected audit_logging_enabled=false when no audit logger is in context")
	}
	if resp.Count != 0 {
		t.Errorf("expected 0 entries, got %d", resp.Count)
	}
	if resp.Note == "" {
		t.Error("expected a note explaining why no entries are available")
	}
}

// TestGetAuditLogTool_NameAndSchemaUnchanged pins the tool's registered name
// and input schema, which the golden contract test in internal/server also
// asserts byte-identically. Fix 5 must wire real data through Execute WITHOUT
// touching either.
func TestGetAuditLogTool_NameAndSchemaUnchanged(t *testing.T) {
	tool := NewGetAuditLogTool(nil, nil)
	if tool.Name() != "get_audit_log" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "get_audit_log")
	}

	schema, ok := tool.InputSchema().(map[string]interface{})
	if !ok {
		t.Fatalf("InputSchema() is not a map: %T", tool.InputSchema())
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("InputSchema() properties is not a map: %T", schema["properties"])
	}
	for _, key := range []string{"limit", "tool", "trace_id"} {
		if _, ok := props[key]; !ok {
			t.Errorf("InputSchema() properties missing %q", key)
		}
	}
}
