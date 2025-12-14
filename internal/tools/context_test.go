package tools

import (
	"context"
	"testing"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

func TestWithClientAndGetClientFromContext(t *testing.T) {
	ctx := context.Background()

	// Test: No client in context should return error
	_, err := GetClientFromContext(ctx)
	if err == nil {
		t.Error("Expected error when no client in context")
	}
	if err != ErrNoClientInContext {
		t.Errorf("Expected ErrNoClientInContext, got: %v", err)
	}

	// Test: With nil client should return error
	ctxWithNil := context.WithValue(ctx, clientContextKey, (*client.Client)(nil))
	_, err = GetClientFromContext(ctxWithNil)
	if err == nil {
		t.Error("Expected error when nil client in context")
	}

	// Test: With valid client should succeed
	// Note: We can't easily create a real client in tests without config,
	// but we can test the context mechanism with a nil-safe approach
	// For integration tests, use actual client instances
}

func TestContextKeyType(t *testing.T) {
	ctx := context.Background()

	// Test: Using wrong key type should not find client
	// We intentionally use a string here (not contextKey) to verify type safety
	wrongKey := "api_client"                                              //nolint:revive // intentionally using string to test type safety
	ctxWithWrongKey := context.WithValue(ctx, wrongKey, &client.Client{}) //nolint:staticcheck,revive // intentional for testing

	_, err := GetClientFromContext(ctxWithWrongKey)
	if err == nil {
		t.Error("Expected error when using wrong key type")
	}
}

func TestWithClientReturnsNewContext(t *testing.T) {
	ctx := context.Background()

	// WithClient should return a new context, not modify the original
	newCtx := WithClient(ctx, nil)

	if ctx == newCtx {
		t.Error("WithClient should return a new context")
	}
}

func TestGetClientFromContextWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Even with cancelled context, GetClientFromContext should work
	// (context values are preserved after cancellation)
	_, err := GetClientFromContext(ctx)
	if err != ErrNoClientInContext {
		t.Errorf("Expected ErrNoClientInContext for empty cancelled context, got: %v", err)
	}
}

func TestWithSessionAndGetSessionFromContext(t *testing.T) {
	ctx := context.Background()

	// Test: No session in context should fall back to global session
	session := GetSessionFromContext(ctx)
	if session == nil {
		t.Error("Expected fallback to global session, got nil")
	}

	// Test: With explicit session should return that session
	testSession := NewSessionContext("test-user", "test-instance")
	ctxWithSession := WithSession(ctx, testSession)

	retrievedSession := GetSessionFromContext(ctxWithSession)
	if retrievedSession != testSession {
		t.Error("Expected to retrieve the injected session")
	}
	if retrievedSession.UserID != "test-user" {
		t.Errorf("Expected UserID 'test-user', got %q", retrievedSession.UserID)
	}
	if retrievedSession.InstanceID != "test-instance" {
		t.Errorf("Expected InstanceID 'test-instance', got %q", retrievedSession.InstanceID)
	}
}

func TestWithSessionReturnsNewContext(t *testing.T) {
	ctx := context.Background()
	session := NewSessionContext("test-user", "test-instance")

	// WithSession should return a new context, not modify the original
	newCtx := WithSession(ctx, session)

	if ctx == newCtx {
		t.Error("WithSession should return a new context")
	}

	// Original context should still fall back to global
	originalSession := GetSessionFromContext(ctx)
	newSession := GetSessionFromContext(newCtx)

	if originalSession == newSession {
		t.Error("Original context should not have the injected session")
	}
}

func TestGetSessionFromContextWithNilSession(t *testing.T) {
	ctx := context.Background()

	// Explicitly set nil session in context
	ctxWithNil := context.WithValue(ctx, sessionContextKey, (*SessionContext)(nil))

	// Should fall back to global session
	session := GetSessionFromContext(ctxWithNil)
	if session == nil {
		t.Error("Expected fallback to global session when nil session in context")
	}
}

func TestSessionContextIsolation(t *testing.T) {
	// Create two independent sessions
	session1 := NewSessionContext("user-1", "instance-1")
	session2 := NewSessionContext("user-2", "instance-2")

	ctx1 := WithSession(context.Background(), session1)
	ctx2 := WithSession(context.Background(), session2)

	// Modify session1 through ctx1
	GetSessionFromContext(ctx1).SetFilter("key", "value1")

	// Session2 should not be affected
	if GetSessionFromContext(ctx2).GetFilter("key") != "" {
		t.Error("Session2 should not have the filter set on Session1")
	}

	// Verify session1 has the filter
	if GetSessionFromContext(ctx1).GetFilter("key") != "value1" {
		t.Error("Session1 should have the filter")
	}
}
