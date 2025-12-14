// Package tools provides MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"errors"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// clientContextKey is the context key for the API client.
	clientContextKey contextKey = "api_client"
	// sessionContextKey is the context key for the user session.
	sessionContextKey contextKey = "session"
)

// ErrNoClientInContext is returned when no API client is found in the context.
var ErrNoClientInContext = errors.New("no API client in context")

// WithClient adds an API client to the context.
// This allows tools to retrieve the client during execution,
// enabling per-request client injection for future HTTP transport support.
func WithClient(ctx context.Context, c *client.Client) context.Context {
	return context.WithValue(ctx, clientContextKey, c)
}

// GetClientFromContext retrieves the API client from the context.
// Returns ErrNoClientInContext if no client is present.
func GetClientFromContext(ctx context.Context) (*client.Client, error) {
	c, ok := ctx.Value(clientContextKey).(*client.Client)
	if !ok || c == nil {
		return nil, ErrNoClientInContext
	}
	return c, nil
}

// WithSession adds a session context to the context.
// This enables per-request session injection for better testability
// and multi-tenant scenarios.
func WithSession(ctx context.Context, session *SessionContext) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSessionFromContext retrieves the session from the context.
// Falls back to the global session if not found in context.
// This provides backward compatibility while enabling context-based testing.
func GetSessionFromContext(ctx context.Context) *SessionContext {
	if session, ok := ctx.Value(sessionContextKey).(*SessionContext); ok && session != nil {
		return session
	}
	// Fall back to global session for backward compatibility
	return GetSession()
}
