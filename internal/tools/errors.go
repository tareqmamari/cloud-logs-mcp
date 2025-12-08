package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// APIError represents a structured API error with status code
type APIError struct {
	StatusCode int
	Message    string
	RequestID  string // Request ID for support/debugging
	Details    map[string]interface{}
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s (Request-ID: %s)", e.Message, e.RequestID)
	}
	return e.Message
}

// IsNotFound returns true if this is a 404 error
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsTimeout returns true if this is a timeout error
func (e *APIError) IsTimeout() bool {
	return e.StatusCode == 408 || e.StatusCode == 504
}

// NewToolResultError creates a new tool result with an error message
func NewToolResultError(message string) *mcp.CallToolResult {
	// Ensure message is never empty
	if message == "" {
		message = "An unknown error occurred"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: message,
			},
		},
		IsError: true,
	}
}

// NewToolResultErrorWithSuggestion creates a tool result with an error and recovery guidance
func NewToolResultErrorWithSuggestion(message, suggestion string) *mcp.CallToolResult {
	fullMessage := fmt.Sprintf("%s\n\n💡 **Suggestion:** %s", message, suggestion)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fullMessage,
			},
		},
		IsError: true,
	}
}

// NewResourceNotFoundError creates an error for missing resources with list suggestion
func NewResourceNotFoundError(resourceType, id, listToolName string) *mcp.CallToolResult {
	message := fmt.Sprintf("%s not found with ID: %s", resourceType, id)
	suggestion := fmt.Sprintf("Use '%s' to see available %ss and their IDs.", listToolName, strings.ToLower(resourceType))
	return NewToolResultErrorWithSuggestion(message, suggestion)
}

// NewTimeoutErrorWithFallback creates a timeout error with alternative tool suggestion
func NewTimeoutErrorWithFallback(operation, fallbackTool, fallbackReason string) *mcp.CallToolResult {
	message := fmt.Sprintf("Operation '%s' timed out", operation)
	suggestion := fmt.Sprintf("For large operations, use '%s' which %s.", fallbackTool, fallbackReason)
	return NewToolResultErrorWithSuggestion(message, suggestion)
}

// HandleGetError handles errors from get_* tools with appropriate suggestions
func HandleGetError(err error, resourceType, resourceID, listToolName string) *mcp.CallToolResult {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsNotFound() {
			return NewResourceNotFoundError(resourceType, resourceID, listToolName)
		}
		if apiErr.IsTimeout() {
			return NewToolResultErrorWithSuggestion(
				apiErr.Message,
				"Try again in a few moments, or check your network connection.",
			)
		}
	}
	return NewToolResultError(err.Error())
}

// HandleQueryError handles errors from query tools with appropriate suggestions
func HandleQueryError(err error, queryType string) *mcp.CallToolResult {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsTimeout() {
			return NewTimeoutErrorWithFallback(
				queryType,
				"submit_background_query",
				"processes queries asynchronously and can handle larger time ranges",
			)
		}
	}
	// Check for context timeout
	if strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "context canceled") {
		return NewTimeoutErrorWithFallback(
			queryType,
			"submit_background_query",
			"processes queries asynchronously and can handle larger time ranges",
		)
	}
	return NewToolResultError(err.Error())
}
