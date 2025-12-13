// Package tools provides MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"errors"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// clientContextKey is the context key for the API client.
const clientContextKey contextKey = "api_client"

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
