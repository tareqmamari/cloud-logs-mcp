package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// testCtx returns a context with a mock client and isolated session injected.
func testCtx(mock *client.MockClient) context.Context {
	ctx := context.Background()
	ctx = WithClient(ctx, mock)
	ctx = WithSession(ctx, NewSessionContext("test-user", "test-instance"))
	return ctx
}

// --- GetAlertTool Execute tests ---

func TestGetAlertTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":        "alert-123",
		"name":      "High CPU Alert",
		"is_active": true,
	})

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "alert-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success, got error result")
	}

	// Verify the correct API request was made
	req := mock.LastRequest()
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/v1/alerts/alert-123" {
		t.Errorf("Path = %q, want /v1/alerts/alert-123", req.Path)
	}
}

func TestGetAlertTool_Execute_MissingID(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result for missing ID")
	}

	// No API call should have been made
	if mock.RequestCount() != 0 {
		t.Errorf("No API request should be made when ID is missing, got %d", mock.RequestCount())
	}
}

func TestGetAlertTool_Execute_APIError(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(404, map[string]interface{}{
		"error": "alert not found",
	})

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "nonexistent"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Tool should handle API errors gracefully and return an error result
	if !result.IsError {
		t.Error("Expected error result for 404 response")
	}
}

// --- ListAlertsTool Execute tests ---

func TestListAlertsTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"alerts": []map[string]interface{}{
			{"id": "alert-1", "name": "Alert One"},
			{"id": "alert-2", "name": "Alert Two"},
		},
	})

	tool := NewListAlertsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/v1/alerts" {
		t.Errorf("Path = %q, want /v1/alerts", req.Path)
	}
}

// --- CreateAlertTool Execute tests ---

func TestCreateAlertTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(201, map[string]interface{}{
		"id":   "new-alert-id",
		"name": "New Alert",
	})

	tool := NewCreateAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	alertData := map[string]interface{}{
		"name":      "New Alert",
		"is_active": true,
		"alert": map[string]interface{}{
			"name": "New Alert",
		},
	}

	result, err := tool.Execute(ctx, alertData)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success, got error result")
	}

	req := mock.LastRequest()
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/v1/alerts" {
		t.Errorf("Path = %q, want /v1/alerts", req.Path)
	}
}

// --- DeleteAlertTool Execute tests ---

func TestDeleteAlertTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 204, Body: []byte("")}

	tool := NewDeleteAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "alert-123", "confirm": true})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", req.Method)
	}
	if req.Path != "/v1/alerts/alert-123" {
		t.Errorf("Path = %q, want /v1/alerts/alert-123", req.Path)
	}
}

func TestDeleteAlertTool_Execute_MissingID(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewDeleteAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result when ID is missing")
	}
	if mock.RequestCount() != 0 {
		t.Error("No API call should be made without ID")
	}
}

// --- Dashboard tool Execute tests ---

func TestGetDashboardTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":   "dash-123",
		"name": "My Dashboard",
	})

	tool := NewGetDashboardTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"dashboard_id": "dash-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != nil && result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/dashboards/dash-123" {
		t.Errorf("Path = %q, want /v1/dashboards/dash-123", req.Path)
	}
}

// --- Policy tool Execute tests ---

func TestListPoliciesTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"policies": []map[string]interface{}{
			{"id": "policy-1", "name": "Archive Policy"},
		},
	})

	tool := NewListPoliciesTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/policies" {
		t.Errorf("Path = %q, want /v1/policies", req.Path)
	}
}

// --- Cross-cutting concerns ---

func TestTool_Execute_WithCancelledContext(t *testing.T) {
	mock := client.NewMockClient()
	mock.DoFunc = func(ctx context.Context, _ *client.Request) (*client.Response, error) {
		return nil, ctx.Err()
	}

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)
	ctx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "alert-123"})
	if err != nil {
		t.Fatalf("Execute should not return Go error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result for cancelled context")
	}
}

func TestTool_Execute_ServerError(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(500, map[string]interface{}{
		"error": "internal server error",
	})

	tool := NewListAlertsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute should not return Go error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result for 500 response")
	}
}

// --- Query tool Execute test ---

func TestQueryTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	// Query endpoint returns SSE data
	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte("data: {\"result\":{\"message\":\"test log\"}}\n"),
	}

	tool := NewQueryTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{
		"query":      "source logs | limit 10",
		"start_date": "2024-01-01T00:00:00Z",
		"end_date":   "2024-01-02T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success, got error: %v", result.Content)
	}

	req := mock.LastRequest()
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if !req.AcceptSSE {
		t.Error("Query requests should use AcceptSSE=true")
	}
}

func TestQueryTool_Execute_MissingRequiredParams(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewQueryTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	// Missing query, start_date, end_date
	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result for missing required params")
	}
	if mock.RequestCount() != 0 {
		t.Error("No API call should be made with missing params")
	}
}

// --- Validate query tool Execute test ---

func TestValidateQueryTool_Execute(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewValidateQueryTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "source logs | filter $m.severity >= 4",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// ValidateQuery doesn't make API calls — it validates locally
	if result.IsError {
		t.Errorf("Expected success for valid query")
	}
}

// --- E2M tool Execute test ---

func TestGetE2MTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":   "e2m-123",
		"name": "Error Rate Metric",
	})

	tool := NewGetE2MTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "e2m-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/events2metrics/e2m-123" {
		t.Errorf("Path = %q, want /v1/events2metrics/e2m-123", req.Path)
	}
}

// --- View tool Execute test ---

func TestListViewsTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"views": []map[string]interface{}{
			{"id": "view-1", "name": "Error View"},
		},
	})

	tool := NewListViewsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/views" {
		t.Errorf("Path = %q, want /v1/views", req.Path)
	}
}

// --- Stream tool Execute test ---

func TestListStreamsTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"streams": []map[string]interface{}{
			{"id": "stream-1", "name": "Production Stream"},
		},
	})

	tool := NewListStreamsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/streams" {
		t.Errorf("Path = %q, want /v1/streams", req.Path)
	}
}

// --- Enrichment tool Execute test ---

func TestListEnrichmentsTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"enrichments": []map[string]interface{}{
			{"id": "enrich-1", "name": "GeoIP Enrichment"},
		},
	})

	tool := NewListEnrichmentsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}
}

// --- Response content verification ---

func TestGetAlertTool_Execute_ResponseContainsData(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":        "alert-123",
		"name":      "CPU Alert",
		"is_active": true,
	})

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "alert-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Result content should contain the response data
	if len(result.Content) == 0 {
		t.Fatal("Expected non-empty content")
	}

	// Extract text content and verify it contains the alert data
	var found bool
	for _, c := range result.Content {
		data, _ := json.Marshal(c)
		text := string(data)
		if strings.Contains(text, "alert-123") || strings.Contains(text, "CPU Alert") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Response content should contain the alert data")
	}
}

// --- Rule Group tool Execute tests ---

func TestGetRuleGroupTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":   "rg-123",
		"name": "Parse JSON Logs",
	})

	tool := NewGetRuleGroupTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "rg-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/rule_groups/rg-123" {
		t.Errorf("Path = %q, want /v1/rule_groups/rg-123", req.Path)
	}
}

// --- Outgoing Webhook tool Execute tests ---

func TestGetOutgoingWebhookTool_Execute_Success(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{
		"id":   "wh-123",
		"name": "Slack Webhook",
		"type": "slack",
	})

	tool := NewGetOutgoingWebhookTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "wh-123"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success result")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/outgoing_webhooks/wh-123" {
		t.Errorf("Path = %q, want /v1/outgoing_webhooks/wh-123", req.Path)
	}
}
