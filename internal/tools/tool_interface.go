// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool defines the interface that all MCP tools must implement.
// This provides a standard contract for tool registration and execution.
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human-readable description of what this tool does
	Description() string

	// InputSchema returns the JSON Schema for the tool's input parameters
	InputSchema() interface{}

	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}
