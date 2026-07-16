// Package tools provides MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"errors"

	"github.com/tareqmamari/cloud-logs-mcp/internal/audit"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// clientContextKey is the context key for the API client.
	clientContextKey contextKey = "api_client"
	// sessionContextKey is the context key for the user session.
	sessionContextKey contextKey = "session"
	// sessionProviderContextKey is the context key for the session provider.
	sessionProviderContextKey contextKey = "session_provider"
	// auditLoggerContextKey is the context key for the audit logger.
	auditLoggerContextKey contextKey = "audit_logger"
)

// ErrNoClientInContext is returned when no API client is found in the context.
var ErrNoClientInContext = errors.New("no API client in context")

// WithClient adds an API client to the context.
// This allows tools to retrieve the client during execution,
// enabling per-request client injection for future HTTP transport support.
func WithClient(ctx context.Context, c client.Doer) context.Context {
	return context.WithValue(ctx, clientContextKey, c)
}

// GetClientFromContext retrieves the API client from the context.
// Returns ErrNoClientInContext if no client is present.
func GetClientFromContext(ctx context.Context) (client.Doer, error) {
	c, ok := ctx.Value(clientContextKey).(client.Doer)
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

// SessionProvider provides access to the current session.
// This interface allows different session resolution strategies
// (e.g., global, per-request, per-tenant).
type SessionProvider interface {
	GetSession() *SessionContext
}

// WithSessionProvider adds a SessionProvider to the context.
func WithSessionProvider(ctx context.Context, provider SessionProvider) context.Context {
	return context.WithValue(ctx, sessionProviderContextKey, provider)
}

// GetSessionProviderFromContext retrieves the SessionProvider from the context.
// Falls back to the global session manager if not found.
func GetSessionProviderFromContext(ctx context.Context) SessionProvider {
	if provider, ok := ctx.Value(sessionProviderContextKey).(SessionProvider); ok && provider != nil {
		return provider
	}
	return GetSessionManager()
}

// WithAuditLogger adds the server's audit logger to the context. This lets
// tools (e.g. GetAuditLogTool) read recent audit entries without the
// server->tools constructor wiring that every other tool uses, and without
// changing any tool's registered name or input schema.
func WithAuditLogger(ctx context.Context, logger *audit.Logger) context.Context {
	return context.WithValue(ctx, auditLoggerContextKey, logger)
}

// GetAuditLoggerFromContext retrieves the audit logger from the context.
// Returns nil, false if none is present (or audit logging is disabled).
func GetAuditLoggerFromContext(ctx context.Context) (*audit.Logger, bool) {
	logger, ok := ctx.Value(auditLoggerContextKey).(*audit.Logger)
	if !ok || logger == nil {
		return nil, false
	}
	return logger, true
}
