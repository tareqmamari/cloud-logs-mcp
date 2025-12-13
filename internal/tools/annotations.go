// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Annotation helper functions to create common annotation patterns.
// These help ensure consistent annotation across all tools.

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// ReadOnlyAnnotations returns annotations for read-only tools (list, get operations).
// These tools don't modify any state and are safe to call repeatedly.
func ReadOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false), // IBM Cloud Logs is a bounded system
	}
}

// CreateAnnotations returns annotations for create operations.
// These tools create new resources but don't modify existing ones.
func CreateAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Create is additive, not destructive
		IdempotentHint:  false,          // Creating twice creates duplicates
		OpenWorldHint:   boolPtr(false),
	}
}

// UpdateAnnotations returns annotations for update operations.
// These tools modify existing resources but don't delete them.
func UpdateAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Updates preserve resources
		IdempotentHint:  true,           // Same update can be applied multiple times
		OpenWorldHint:   boolPtr(false),
	}
}

// DeleteAnnotations returns annotations for delete operations.
// These tools permanently remove resources and require caution.
func DeleteAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true), // Delete is destructive
		IdempotentHint:  true,          // Deleting twice is safe (already gone)
		OpenWorldHint:   boolPtr(false),
	}
}

// QueryAnnotations returns annotations for query tools.
// These tools read data but may interact with external data sources.
func QueryAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false), // Queries a bounded log system
	}
}

// WorkflowAnnotations returns annotations for workflow/composite tools.
// These tools orchestrate multiple operations.
func WorkflowAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true, // Workflows analyze but don't modify
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false),
	}
}

// IngestionAnnotations returns annotations for log ingestion tools.
// These tools add new data but don't modify existing data.
func IngestionAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Ingestion is additive
		IdempotentHint:  false,          // Same log can be ingested multiple times
		OpenWorldHint:   boolPtr(false),
	}
}

// DefaultAnnotations returns default annotations when no specific hints are needed.
// This provides a consistent baseline for tools.
func DefaultAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		OpenWorldHint: boolPtr(false),
	}
}
