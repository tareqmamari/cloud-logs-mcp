// Package tools provides MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/tareqmamari/cloud-logs-mcp/internal/service"
)

// TestMockLogsService verifies the mock service implementation
func TestMockLogsService(t *testing.T) {
	mock := NewMockLogsService()
	ctx := context.Background()

	// Test Query
	t.Run("Query", func(t *testing.T) {
		mock.QueryResult = &service.QueryResponse{
			Events:     []map[string]interface{}{{"message": "test"}},
			TotalCount: 1,
		}

		resp, err := mock.Query(ctx, &service.QueryRequest{
			Query:     "source logs | limit 10",
			Tier:      "archive",
			Syntax:    "dataprime",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-01-02T00:00:00Z",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if resp.TotalCount != 1 {
			t.Errorf("Expected TotalCount=1, got %d", resp.TotalCount)
		}
		if len(mock.QueryCalls) != 1 {
			t.Errorf("Expected 1 query call, got %d", len(mock.QueryCalls))
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		mock.Reset()
		mock.GetResult = map[string]interface{}{"id": "test-id", "name": "test-alert"}

		result, err := mock.Get(ctx, service.ResourceAlert, "test-id")

		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result["id"] != "test-id" {
			t.Errorf("Expected id=test-id, got %v", result["id"])
		}
		if len(mock.GetCalls) != 1 {
			t.Errorf("Expected 1 get call, got %d", len(mock.GetCalls))
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		mock.Reset()
		mock.ListResult = &service.ListResponse{
			Items: []map[string]interface{}{
				{"id": "1", "name": "alert1"},
				{"id": "2", "name": "alert2"},
			},
			TotalCount: 2,
			HasMore:    false,
		}

		result, err := mock.List(ctx, service.ResourceAlert, &service.ListOptions{Limit: 50})

		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result.Items))
		}
	})

	// Test Create
	t.Run("Create", func(t *testing.T) {
		mock.Reset()

		result, err := mock.Create(ctx, service.ResourceAlert, map[string]interface{}{
			"name":      "new-alert",
			"is_active": true,
		})

		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if result["id"] == "" {
			t.Error("Expected id to be set")
		}
		if len(mock.CreateCalls) != 1 {
			t.Errorf("Expected 1 create call, got %d", len(mock.CreateCalls))
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		mock.Reset()

		result, err := mock.Update(ctx, service.ResourceAlert, "alert-id", map[string]interface{}{
			"name": "updated-alert",
		})

		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if result["name"] != "updated-alert" {
			t.Errorf("Expected name=updated-alert, got %v", result["name"])
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		mock.Reset()

		err := mock.Delete(ctx, service.ResourceAlert, "alert-id")

		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if len(mock.DeleteCalls) != 1 {
			t.Errorf("Expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
	})

	// Test HealthCheck
	t.Run("HealthCheck", func(t *testing.T) {
		mock.Reset()

		result, err := mock.HealthCheck(ctx)

		if err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
		if !result.Healthy {
			t.Error("Expected healthy=true")
		}
	})

	// Test GetInstanceInfo
	t.Run("GetInstanceInfo", func(t *testing.T) {
		info := mock.GetInstanceInfo()

		if info.Region != "us-south" {
			t.Errorf("Expected region=us-south, got %s", info.Region)
		}
	})
}

// TestMockLogsServiceErrors verifies error handling
func TestMockLogsServiceErrors(t *testing.T) {
	mock := NewMockLogsService()
	ctx := context.Background()

	t.Run("QueryError", func(t *testing.T) {
		mock.QueryError = service.NewQuerySyntaxError("invalid query", "syntax error at position 10")

		_, err := mock.Query(ctx, &service.QueryRequest{Query: "bad query"})

		if err == nil {
			t.Error("Expected error, got nil")
		}

		agentErr, ok := err.(*service.AgentActionableError)
		if !ok {
			t.Error("Expected AgentActionableError")
		} else if agentErr.Action != service.ActionChangeParams {
			t.Errorf("Expected action=CHANGE_PARAMS, got %s", agentErr.Action)
		}
	})

	t.Run("GetNotFoundError", func(t *testing.T) {
		mock.Reset()
		mock.GetError = service.NewResourceNotFoundError("Alert", "nonexistent-id", "list_alerts")

		_, err := mock.Get(ctx, service.ResourceAlert, "nonexistent-id")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		agentErr, ok := err.(*service.AgentActionableError)
		if !ok {
			t.Error("Expected AgentActionableError")
		} else if agentErr.Code != service.ErrResourceNotFound {
			t.Errorf("Expected code=RESOURCE_NOT_FOUND, got %s", agentErr.Code)
		}
	})
}

// TestServiceContextIntegration verifies context-based service injection
func TestServiceContextIntegration(t *testing.T) {
	mock := NewMockLogsService()
	ctx := context.Background()

	// Add service to context
	ctx = WithService(ctx, mock)

	// Retrieve from context
	svc := GetServiceFromContext(ctx)

	if svc == nil {
		t.Fatal("Expected service from context, got nil")
	}

	// Verify it's the same mock
	_, err := svc.Query(ctx, &service.QueryRequest{Query: "test"})
	if err != nil {
		t.Fatalf("Query through context failed: %v", err)
	}

	if len(mock.QueryCalls) != 1 {
		t.Errorf("Expected 1 query call, got %d", len(mock.QueryCalls))
	}
}

// TestSessionContextInjection verifies context-based session injection
// This demonstrates the new pattern where sessions are passed via context
// rather than using global state.
func TestSessionContextInjection(t *testing.T) {
	// Create an isolated session for testing
	testSession := NewSessionContext("test-user-injection", "test-instance")
	testSession.SetLastQuery("test query from injected session")

	ctx := context.Background()

	// Inject session into context
	ctx = WithSession(ctx, testSession)

	// Retrieve session from context
	retrieved := GetSessionFromContext(ctx)

	// Verify it's the same session
	if retrieved.GetLastQuery() != "test query from injected session" {
		t.Errorf("Expected injected session's last query, got %q", retrieved.GetLastQuery())
	}

	// Verify isolation - changing the retrieved session affects the original
	retrieved.SetLastQuery("modified query")
	if testSession.GetLastQuery() != "modified query" {
		t.Errorf("Session should be the same instance, not a copy")
	}
}

// TestSessionProviderContextInjection verifies SessionProvider injection
func TestSessionProviderContextInjection(t *testing.T) {
	// Create a mock session provider
	mockManager := NewSessionManager("")
	testSession := mockManager.GetOrCreateSession("test-api-key", "test-instance")
	testSession.SetLastQuery("provider test query")

	ctx := context.Background()

	// Inject session provider into context
	ctx = WithSessionProvider(ctx, mockManager)

	// Retrieve provider from context
	provider := GetSessionProviderFromContext(ctx)

	// Verify we can get sessions through the provider
	session := provider.GetSession()
	if session == nil {
		t.Fatal("Expected session from provider, got nil")
	}
}

// TestCacheHelperContextInjection verifies cache helper uses context session
func TestCacheHelperContextInjection(t *testing.T) {
	// Create an isolated session
	testSession := NewSessionContext("cache-test-user", "cache-test-instance")

	ctx := context.Background()
	ctx = WithSession(ctx, testSession)

	// Get cache helper from context
	cacheHelper := GetCacheHelperFromContext(ctx)

	// Verify it uses the session's user/instance IDs
	// (We can't access private fields, but we can verify it works)
	cacheHelper.Set("test_tool", "test_key", "test_value")
	cached, ok := cacheHelper.Get("test_tool", "test_key")

	if !ok {
		t.Error("Expected cached value to be found")
	}
	if cached != "test_value" {
		t.Errorf("Expected 'test_value', got %v", cached)
	}
}

// TestAgentActionableErrorFormatting verifies error formatting for agents
func TestAgentActionableErrorFormatting(t *testing.T) {
	testCases := []struct {
		name        string
		err         *service.AgentActionableError
		expectTexts []string
	}{
		{
			name: "query_syntax_error",
			err: service.NewQuerySyntaxError(
				"source logs | filter badfield",
				"unknown field 'badfield'",
			),
			expectTexts: []string{
				"CHANGE_PARAMS",
				"query",
				"syntax",
			},
		},
		{
			name: "rate_limit_error",
			err:  service.NewRateLimitError(5000),
			expectTexts: []string{
				"RETRY_WITH_BACKOFF",
				"5000",
				"Wait",
			},
		},
		{
			name: "auth_error",
			err:  service.NewAuthError("invalid token"),
			expectTexts: []string{
				"ESCALATE",
				"API key",
				"authentication",
			},
		},
		{
			name: "timeout_error",
			err:  service.NewTimeoutError("synchronous query"),
			expectTexts: []string{
				"CHANGE_PARAMS",
				"limit",
				"timed out",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := tc.err.FormatForAgent()

			for _, text := range tc.expectTexts {
				if !testContainsString(formatted, text) {
					t.Errorf("Expected formatted error to contain '%s', got:\n%s", text, formatted)
				}
			}
		})
	}
}

// TestSchemaDefinitions verifies schema definition helpers
func TestSchemaDefinitions(t *testing.T) {
	t.Run("RefSchema", func(t *testing.T) {
		ref := RefSchema("pagination")

		if ref["$ref"] != "#/definitions/pagination" {
			t.Errorf("Expected $ref=#/definitions/pagination, got %v", ref["$ref"])
		}
	})

	t.Run("QueryInputSchema", func(t *testing.T) {
		schema := QueryInputSchema()

		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected properties to be a map")
		}

		required, ok := schema["required"].([]string)
		if !ok {
			t.Fatal("Expected required to be a string slice")
		}

		// Check required fields
		requiredSet := make(map[string]bool)
		for _, r := range required {
			requiredSet[r] = true
		}

		expectedRequired := []string{"query", "start_date", "end_date"}
		for _, r := range expectedRequired {
			if !requiredSet[r] {
				t.Errorf("Expected '%s' to be required", r)
			}
		}

		// Check properties exist
		expectedProps := []string{"query", "tier", "syntax", "limit", "summary_only"}
		for _, p := range expectedProps {
			if _, ok := props[p]; !ok {
				t.Errorf("Expected property '%s' to exist", p)
			}
		}
	})

	t.Run("CRUDSchemas", func(t *testing.T) {
		getSchema := CRUDGetSchema("alert")
		if getSchema["required"].([]string)[0] != "id" {
			t.Error("Expected 'id' to be required in GET schema")
		}

		listSchema := CRUDListSchema()
		if _, ok := listSchema["properties"].(map[string]interface{})["limit"]; !ok {
			t.Error("Expected 'limit' property in LIST schema")
		}

		createSchema := CRUDCreateSchema("alert", map[string]interface{}{"name": "test"})
		if createSchema["required"].([]string)[0] != "alert" {
			t.Error("Expected 'alert' to be required in CREATE schema")
		}

		updateSchema := CRUDUpdateSchema("alert")
		required := updateSchema["required"].([]string)
		if len(required) != 2 {
			t.Errorf("Expected 2 required fields in UPDATE schema, got %d", len(required))
		}

		deleteSchema := CRUDDeleteSchema("alert")
		props := deleteSchema["properties"].(map[string]interface{})
		if _, ok := props["confirm"]; !ok {
			t.Error("Expected 'confirm' property in DELETE schema")
		}
	})
}

// helper function for tests - uses strings.Contains
func testContainsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
