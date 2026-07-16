package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	mcperrors "github.com/tareqmamari/cloud-logs-mcp/internal/errors"
)

// resultText extracts the text content from a CallToolResult for assertions.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("result content is not TextContent: %T", result.Content[0])
	}
	return textContent.Text
}

// TestGetTool_NotFound_SurfacesListSuggestion is a regression test for the
// tools error path after client.Do started returning classified errors for
// 4xx statuses. Before that change, ExecuteRequest saw (resp, nil) and built
// a *tools.APIError itself; afterwards Do returns
// (resp, *mcperrors.StructuredError) and ExecuteRequest must translate the
// classified error back into *tools.APIError, or downstream
// errors.As(err, &apiErr) matching in HandleGetError/HandleQueryError
// silently stops working: IsNotFound()/IsTimeout() handling, the
// "use list_* to see available IDs" agent suggestion, and request-ID
// extraction disappear across every get_* tool.
//
// The DoFunc below reproduces the real client's current contract exactly:
// a non-nil response returned alongside a classified error.
func TestGetTool_NotFound_SurfacesListSuggestion(t *testing.T) {
	mock := client.NewMockClient()
	mock.DoFunc = func(_ context.Context, _ *client.Request) (*client.Response, error) {
		resp := &client.Response{
			StatusCode: 404,
			Body:       []byte(`{"error":"alert not found"}`),
		}
		return resp, mcperrors.FromHTTPStatus(404, `{"error":"alert not found"}`)
	}

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "nonexistent"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Fatal("Expected error result for 404 response")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "not found") {
		t.Errorf("404 result should contain not-found guidance, got: %s", text)
	}
	if !strings.Contains(text, "list_alerts") {
		t.Errorf("404 result should suggest list_alerts to find valid IDs, got: %s", text)
	}
}

// TestExecuteRequest_TranslatesClassifiedErrorToAPIError verifies the
// translation directly: a classified *mcperrors.StructuredError from the
// client becomes an errors.As-matchable *tools.APIError carrying the status
// code and the request ID from the response headers.
func TestExecuteRequest_TranslatesClassifiedErrorToAPIError(t *testing.T) {
	mock := client.NewMockClient()
	mock.DoFunc = func(_ context.Context, _ *client.Request) (*client.Response, error) {
		resp := &client.Response{
			StatusCode: 404,
			Body:       []byte(`{"error":"not found"}`),
			Headers:    map[string][]string{"X-Request-Id": {"req-abc-123"}},
		}
		return resp, mcperrors.FromHTTPStatus(404, `{"error":"not found"}`)
	}

	base := NewBaseTool(mock, zap.NewNop())
	_, err := base.ExecuteRequest(testCtx(mock), &client.Request{Method: "GET", Path: "/v1/alerts/x"})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to match *tools.APIError via errors.As, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("APIError.IsNotFound() = false, want true (StatusCode = %d)", apiErr.StatusCode)
	}
	if apiErr.RequestID != "req-abc-123" {
		t.Errorf("APIError.RequestID = %q, want %q (should be extracted from response headers)", apiErr.RequestID, "req-abc-123")
	}
}

// TestGetTool_NotFound_ViaQueuedMockResponse verifies the same guidance works
// through MockClient's queued-response path, which now classifies 4xx
// statuses the same way the real client does (so mock-based tests exercise
// the real contract instead of the removed (resp, nil)-for-4xx one).
func TestGetTool_NotFound_ViaQueuedMockResponse(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(404, map[string]interface{}{"error": "alert not found"})

	tool := NewGetAlertTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	result, err := tool.Execute(ctx, map[string]interface{}{"id": "nonexistent"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Fatal("Expected error result for 404 response")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "list_alerts") {
		t.Errorf("404 result should suggest list_alerts to find valid IDs, got: %s", text)
	}
}
