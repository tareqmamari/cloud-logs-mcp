// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// GetAllTools returns all available MCP tools, constructed from the single
// descriptor table (see descriptors.go). This is the same list the server
// registers, so tests exercising GetAllTools exercise exactly what is served.
func GetAllTools(c client.Doer, logger *zap.Logger) []Tool {
	descriptors := Descriptors()
	all := make([]Tool, 0, len(descriptors))
	for _, d := range descriptors {
		all = append(all, d.New(c, logger))
	}
	return all
}

// GetToolCount returns the total number of registered tools. It is derived from
// the descriptor table so it can never fall out of sync with the served set.
func GetToolCount() int {
	return len(Descriptors())
}
