package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

func TestNewLogsService(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		svc := NewLogsService(nil, logger, nil)
		if svc == nil {
			t.Fatal("NewLogsService returned nil")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		cfg := &Config{
			DefaultQueryTier:   "frequent_search",
			DefaultQuerySyntax: "lucene",
			DefaultQueryLimit:  50,
			MaxQueryLimit:      1000,
		}
		svc := NewLogsService(nil, logger, cfg)
		if svc == nil {
			t.Fatal("NewLogsService returned nil")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultQueryTier != "archive" {
		t.Errorf("DefaultQueryTier = %q, want %q", cfg.DefaultQueryTier, "archive")
	}
	if cfg.DefaultQuerySyntax != "dataprime" {
		t.Errorf("DefaultQuerySyntax = %q, want %q", cfg.DefaultQuerySyntax, "dataprime")
	}
	if cfg.DefaultQueryLimit != 200 {
		t.Errorf("DefaultQueryLimit = %d, want %d", cfg.DefaultQueryLimit, 200)
	}
	if cfg.MaxQueryLimit != 50000 {
		t.Errorf("MaxQueryLimit = %d, want %d", cfg.MaxQueryLimit, 50000)
	}
	if cfg.QueryTimeout != 60*time.Second {
		t.Errorf("QueryTimeout = %v, want 60s", cfg.QueryTimeout)
	}
	if !cfg.EnableAutoCorrect {
		t.Error("EnableAutoCorrect should default to true")
	}
}

func TestResourceConfig_AllResourceTypesConfigured(t *testing.T) {
	expectedTypes := []ResourceType{
		ResourceAlert, ResourceAlertDefinition, ResourceDashboard,
		ResourceDashboardFolder, ResourcePolicy, ResourceRuleGroup,
		ResourceOutgoingWebhook, ResourceE2M, ResourceEnrichment,
		ResourceView, ResourceViewFolder, ResourceDataAccessRule,
		ResourceStream, ResourceEventStream,
	}

	for _, rt := range expectedTypes {
		cfg, ok := resourceConfig[rt]
		if !ok {
			t.Errorf("Resource type %q not in resourceConfig", rt)
			continue
		}
		if cfg.basePath == "" {
			t.Errorf("Resource type %q has empty basePath", rt)
		}
		if cfg.listKey == "" {
			t.Errorf("Resource type %q has empty listKey", rt)
		}
	}
}

func TestGet_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Get(context.Background(), ResourceType("nonexistent"), "id-123")
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
	if !strings.Contains(err.Error(), "unknown resource type") {
		t.Errorf("Expected 'unknown resource type' error, got: %v", err)
	}
}

func TestList_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.List(context.Background(), ResourceType("nonexistent"), nil)
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestCreate_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Create(context.Background(), ResourceType("nonexistent"), map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestUpdate_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Update(context.Background(), ResourceType("nonexistent"), "id", map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestDelete_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	err := svc.Delete(context.Background(), ResourceType("nonexistent"), "id")
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestResourceConfig_HTTPMethodSelection(t *testing.T) {
	// Verify the resourceConfig map correctly flags which resources use PUT vs PATCH
	tests := []struct {
		resourceType ResourceType
		expectUsePUT bool
	}{
		{ResourceAlert, false},
		{ResourceAlertDefinition, false},
		{ResourceDashboard, false},
		{ResourceDashboardFolder, false},
		{ResourcePolicy, false},
		{ResourceRuleGroup, false},
		{ResourceOutgoingWebhook, false},
		{ResourceE2M, true}, // E2M uses PUT (replace semantics)
		{ResourceEnrichment, false},
		{ResourceView, true},       // Views use PUT (replace semantics)
		{ResourceViewFolder, true}, // View folders use PUT (replace semantics)
		{ResourceDataAccessRule, false},
		{ResourceStream, false},
		{ResourceEventStream, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			cfg, ok := resourceConfig[tt.resourceType]
			if !ok {
				t.Fatalf("Resource type %q not found in resourceConfig", tt.resourceType)
			}
			if cfg.usePUT != tt.expectUsePUT {
				t.Errorf("resourceConfig[%s].usePUT = %v, want %v", tt.resourceType, cfg.usePUT, tt.expectUsePUT)
			}
		})
	}
}

func TestResourceError(t *testing.T) {
	t.Run("with ID", func(t *testing.T) {
		err := &ResourceError{
			Type:      ResourceAlert,
			Operation: "get",
			ID:        "alert-123",
			Err:       errors.New("not found"),
		}

		msg := err.Error()
		if !strings.Contains(msg, "get") {
			t.Errorf("Error should contain operation: %s", msg)
		}
		if !strings.Contains(msg, "alert") {
			t.Errorf("Error should contain type: %s", msg)
		}
		if !strings.Contains(msg, "alert-123") {
			t.Errorf("Error should contain ID: %s", msg)
		}
		if !strings.Contains(msg, "not found") {
			t.Errorf("Error should contain cause: %s", msg)
		}
	})

	t.Run("without ID", func(t *testing.T) {
		err := &ResourceError{
			Type:      ResourceDashboard,
			Operation: "list",
			Err:       errors.New("timeout"),
		}

		msg := err.Error()
		if strings.Contains(msg, "id=") {
			t.Errorf("Error without ID should not contain 'id=': %s", msg)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &ResourceError{
			Type:      ResourceAlert,
			Operation: "delete",
			Err:       cause,
		}

		if err.Unwrap() != cause {
			t.Error("Unwrap should return original cause")
		}
	})
}

// --- Tests using MockClient (enabled by client.Doer interface) ---

func newTestService() (LogsService, *client.MockClient) {
	mock := client.NewMockClient()
	logger := zap.NewNop()
	svc := NewLogsService(mock, logger, nil)
	return svc, mock
}

func TestQuery_Success(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte(""),
	}

	resp, err := svc.Query(context.Background(), &QueryRequest{
		Query:     "source logs | limit 10",
		Tier:      "archive",
		Syntax:    "dataprime",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
	})

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Metadata == nil {
		t.Fatal("response metadata is nil")
	}
	if resp.Metadata.Tier != "archive" {
		t.Errorf("Metadata.Tier = %q, want %q", resp.Metadata.Tier, "archive")
	}
	if resp.Metadata.Syntax != "dataprime" {
		t.Errorf("Metadata.Syntax = %q, want %q", resp.Metadata.Syntax, "dataprime")
	}

	// Verify the request was sent correctly
	req := mock.LastRequest()
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/v1/query" {
		t.Errorf("Path = %q, want /v1/query", req.Path)
	}
	if !req.AcceptSSE {
		t.Error("AcceptSSE should be true for query requests")
	}
}

func TestQuery_AppliesDefaults(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte(""),
	}

	resp, err := svc.Query(context.Background(), &QueryRequest{
		Query:     "source logs | limit 10",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
		// No tier, syntax, or limit — should use defaults
	})

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Metadata.Tier != "archive" {
		t.Errorf("Default tier = %q, want %q", resp.Metadata.Tier, "archive")
	}
	if resp.Metadata.Syntax != "dataprime" {
		t.Errorf("Default syntax = %q, want %q", resp.Metadata.Syntax, "dataprime")
	}
	if resp.Metadata.Limit != 200 {
		t.Errorf("Default limit = %d, want 200", resp.Metadata.Limit)
	}
}

func TestQuery_ClampsLimit(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte(""),
	}

	resp, err := svc.Query(context.Background(), &QueryRequest{
		Query:     "source logs",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
		Limit:     999999, // Exceeds MaxQueryLimit (50000)
	})

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Metadata.Limit != 50000 {
		t.Errorf("Limit should be clamped to 50000, got %d", resp.Metadata.Limit)
	}
}

func TestQuery_ClientError(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultError = errors.New("connection refused")
	mock.DefaultResponse = nil

	_, err := svc.Query(context.Background(), &QueryRequest{
		Query:     "source logs | limit 10",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
	})

	if err == nil {
		t.Fatal("Expected error when client fails")
	}
	if !strings.Contains(err.Error(), "query execution failed") {
		t.Errorf("Error = %q, want to contain 'query execution failed'", err.Error())
	}
}

func TestGet_Success(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte(`{}`),
	}

	_, err := svc.Get(context.Background(), ResourceAlert, "alert-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/v1/alerts/alert-123" {
		t.Errorf("Path = %q, want /v1/alerts/alert-123", req.Path)
	}
}

func TestGet_ClientError(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultError = errors.New("network error")
	mock.DefaultResponse = nil

	_, err := svc.Get(context.Background(), ResourceAlert, "alert-123")
	if err == nil {
		t.Fatal("Expected error")
	}

	var resErr *ResourceError
	if !errors.As(err, &resErr) {
		t.Fatalf("Expected ResourceError, got %T", err)
	}
	if resErr.Type != ResourceAlert {
		t.Errorf("ResourceError.Type = %q, want %q", resErr.Type, ResourceAlert)
	}
	if resErr.Operation != "get" {
		t.Errorf("ResourceError.Operation = %q, want %q", resErr.Operation, "get")
	}
	if resErr.ID != "alert-123" {
		t.Errorf("ResourceError.ID = %q, want %q", resErr.ID, "alert-123")
	}
}

func TestList_Success(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 200,
		Body:       []byte(`{"alerts": []}`),
	}

	_, err := svc.List(context.Background(), ResourceAlert, &ListOptions{Limit: 50, Cursor: "abc"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/v1/alerts" {
		t.Errorf("Path = %q, want /v1/alerts", req.Path)
	}
	if req.Query["limit"] != "50" {
		t.Errorf("Query limit = %q, want %q", req.Query["limit"], "50")
	}
	if req.Query["cursor"] != "abc" {
		t.Errorf("Query cursor = %q, want %q", req.Query["cursor"], "abc")
	}
}

func TestList_NilOptions(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

	_, err := svc.List(context.Background(), ResourceDashboard, nil)
	if err != nil {
		t.Fatalf("List with nil options failed: %v", err)
	}

	req := mock.LastRequest()
	if len(req.Query) != 0 {
		t.Errorf("Expected no query params with nil options, got %v", req.Query)
	}
}

func TestCreate_Success(t *testing.T) {
	svc, mock := newTestService()

	mock.DefaultResponse = &client.Response{
		StatusCode: 201,
		Body:       []byte(`{}`),
	}

	data := map[string]interface{}{
		"name":      "new-alert",
		"is_active": true,
	}
	_, err := svc.Create(context.Background(), ResourceAlert, data)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/v1/alerts" {
		t.Errorf("Path = %q, want /v1/alerts", req.Path)
	}
}

func TestUpdate_UsesCorrectHTTPMethod(t *testing.T) {
	tests := []struct {
		resourceType   ResourceType
		expectedMethod string
	}{
		{ResourceAlert, "PATCH"},
		{ResourceDashboard, "PATCH"},
		{ResourcePolicy, "PATCH"},
		{ResourceE2M, "PUT"},
		{ResourceView, "PUT"},
		{ResourceViewFolder, "PUT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			svc, mock := newTestService()
			mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

			_, err := svc.Update(context.Background(), tt.resourceType, "id-123", map[string]interface{}{"name": "updated"})
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			req := mock.LastRequest()
			if req.Method != tt.expectedMethod {
				t.Errorf("Method = %q, want %q", req.Method, tt.expectedMethod)
			}
		})
	}
}

func TestDelete_Success(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 204, Body: []byte("")}

	err := svc.Delete(context.Background(), ResourceAlert, "alert-to-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", req.Method)
	}
	if req.Path != "/v1/alerts/alert-to-delete" {
		t.Errorf("Path = %q, want /v1/alerts/alert-to-delete", req.Path)
	}
}

func TestDelete_ClientError(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultError = errors.New("server error")
	mock.DefaultResponse = nil

	err := svc.Delete(context.Background(), ResourceAlert, "alert-123")
	if err == nil {
		t.Fatal("Expected error")
	}

	var resErr *ResourceError
	if !errors.As(err, &resErr) {
		t.Fatalf("Expected ResourceError, got %T", err)
	}
	if resErr.Operation != "delete" {
		t.Errorf("Operation = %q, want %q", resErr.Operation, "delete")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

	status, err := svc.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if !status.Healthy {
		t.Error("Expected healthy status")
	}
	if status.Status != "healthy" {
		t.Errorf("Status = %q, want %q", status.Status, "healthy")
	}
	if status.Checks["api"] != "ok" {
		t.Errorf("API check = %q, want %q", status.Checks["api"], "ok")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultError = errors.New("connection refused")
	mock.DefaultResponse = nil

	status, err := svc.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck should not return error, got: %v", err)
	}
	if status.Healthy {
		t.Error("Expected unhealthy status")
	}
	if status.Status != "unhealthy" {
		t.Errorf("Status = %q, want %q", status.Status, "unhealthy")
	}
	if !strings.Contains(status.Checks["api"], "failed") {
		t.Errorf("API check = %q, want to contain 'failed'", status.Checks["api"])
	}
}

func TestGetInstanceInfo(t *testing.T) {
	svc, mock := newTestService()
	mock.Instance = client.InstanceInfo{
		ServiceURL:   "https://test.api.eu-de.logs.cloud.ibm.com",
		Region:       "eu-de",
		InstanceName: "prod-logs",
	}

	info := svc.GetInstanceInfo()
	if info.ServiceURL != "https://test.api.eu-de.logs.cloud.ibm.com" {
		t.Errorf("ServiceURL = %q", info.ServiceURL)
	}
	if info.Region != "eu-de" {
		t.Errorf("Region = %q, want %q", info.Region, "eu-de")
	}
	if info.InstanceName != "prod-logs" {
		t.Errorf("InstanceName = %q, want %q", info.InstanceName, "prod-logs")
	}
}

func TestSubmitBackgroundQuery_Success(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

	_, err := svc.SubmitBackgroundQuery(context.Background(), &BackgroundQueryRequest{
		Query:     "source logs | limit 10",
		Syntax:    "dataprime",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("SubmitBackgroundQuery failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/v1/background_query" {
		t.Errorf("Path = %q, want /v1/background_query", req.Path)
	}
}

func TestGetBackgroundQueryStatus_Success(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

	status, err := svc.GetBackgroundQueryStatus(context.Background(), "query-abc")
	if err != nil {
		t.Fatalf("GetBackgroundQueryStatus failed: %v", err)
	}
	if status.QueryID != "query-abc" {
		t.Errorf("QueryID = %q, want %q", status.QueryID, "query-abc")
	}

	req := mock.LastRequest()
	if req.Path != "/v1/background_query/query-abc/status" {
		t.Errorf("Path = %q", req.Path)
	}
}

func TestCancelBackgroundQuery_Success(t *testing.T) {
	svc, mock := newTestService()
	mock.DefaultResponse = &client.Response{StatusCode: 204, Body: []byte("")}

	err := svc.CancelBackgroundQuery(context.Background(), "query-abc")
	if err != nil {
		t.Fatalf("CancelBackgroundQuery failed: %v", err)
	}

	req := mock.LastRequest()
	if req.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", req.Method)
	}
	if req.Path != "/v1/background_query/query-abc" {
		t.Errorf("Path = %q", req.Path)
	}
}

func TestAllResourceTypes_CorrectPaths(t *testing.T) {
	expectedPaths := map[ResourceType]string{
		ResourceAlert:           "/v1/alerts",
		ResourceAlertDefinition: "/v1/alert_definitions",
		ResourceDashboard:       "/v1/dashboards",
		ResourceDashboardFolder: "/v1/dashboard_folders",
		ResourcePolicy:          "/v1/policies",
		ResourceRuleGroup:       "/v1/rule_groups",
		ResourceOutgoingWebhook: "/v1/outgoing_webhooks",
		ResourceE2M:             "/v1/events2metrics",
		ResourceEnrichment:      "/v1/enrichments",
		ResourceView:            "/v1/views",
		ResourceViewFolder:      "/v1/view_folders",
		ResourceDataAccessRule:  "/v1/data_access_rules",
		ResourceStream:          "/v1/streams",
		ResourceEventStream:     "/v1/event_stream_targets",
	}

	for rt, expectedPath := range expectedPaths {
		t.Run(string(rt), func(t *testing.T) {
			svc, mock := newTestService()
			mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

			_, _ = svc.Get(context.Background(), rt, "test-id")

			req := mock.LastRequest()
			if req.Path != expectedPath+"/test-id" {
				t.Errorf("Path = %q, want %q", req.Path, expectedPath+"/test-id")
			}
		})
	}
}

// import errors for the test
var _ = errors.New
