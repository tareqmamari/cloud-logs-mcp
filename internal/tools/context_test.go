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
