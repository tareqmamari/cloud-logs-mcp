// Package mcperrors provides structured error handling for the IBM Cloud Logs MCP server.
package mcperrors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ErrorCategory classifies the type of error
type ErrorCategory string

const (
	// ClientError indicates the error was caused by the client (4xx)
	ClientError ErrorCategory = "CLIENT_ERROR"
	// ServerError indicates the error was caused by the server (5xx)
	ServerError ErrorCategory = "SERVER_ERROR"
	// ExternalError indicates the error was caused by an external dependency
	ExternalError ErrorCategory = "EXTERNAL_ERROR"
)

// ErrorCode represents a structured error code
type ErrorCode string

const (
	// CodeInvalidInput indicates invalid input from client
	CodeInvalidInput ErrorCode = "INVALID_INPUT"
	// CodeMissingParameter indicates a required parameter is missing
	CodeMissingParameter ErrorCode = "MISSING_PARAMETER"
	// CodeInvalidQuery indicates invalid query syntax
	CodeInvalidQuery ErrorCode = "INVALID_QUERY_SYNTAX"
	// CodeResourceNotFound indicates the requested resource was not found
	CodeResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	// CodeUnauthorized indicates the request is unauthorized
	CodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// CodeForbidden indicates the request is forbidden
	CodeForbidden ErrorCode = "FORBIDDEN"
	// CodeConflict indicates a conflict with existing resource
	CodeConflict ErrorCode = "CONFLICT"
	// CodeRateLimitExceeded indicates rate limit has been exceeded
	CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// CodeInternalError represents an internal server error
	CodeInternalError ErrorCode = "INTERNAL_ERROR"
	// CodeServiceUnavailable represents a service unavailable error
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	// CodeTimeout represents a timeout error
	CodeTimeout ErrorCode = "TIMEOUT"

	// CodeAPIError represents an API error
	CodeAPIError ErrorCode = "API_ERROR"
	// CodeAuthFailed represents an authentication failure
	CodeAuthFailed ErrorCode = "AUTH_FAILED"
	// CodeNetworkError represents a network error
	CodeNetworkError ErrorCode = "NETWORK_ERROR"
)

// StructuredError represents a detailed error with category, code, and recovery suggestion
type StructuredError struct {
	Code       ErrorCode     `json:"code"`
	Category   ErrorCategory `json:"category"`
	Message    string        `json:"message"`
	Details    interface{}   `json:"details,omitempty"`
	Suggestion string        `json:"suggestion,omitempty"`
	// StatusCode is the originating HTTP status code, when this error was
	// produced from an HTTP response (see FromHTTPStatus). Zero if not applicable.
	StatusCode int `json:"status_code,omitempty"`
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Category, e.Message)
}

// ToJSON converts the error to JSON string
func (e *StructuredError) ToJSON() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"code":"%s","category":"%s","message":"%s"}`, e.Code, e.Category, e.Message)
	}
	return string(bytes)
}

// New creates a new structured error
func New(code ErrorCode, category ErrorCategory, message string) *StructuredError {
	return &StructuredError{
		Code:     code,
		Category: category,
		Message:  message,
	}
}

// WithDetails adds details to the error
func (e *StructuredError) WithDetails(details interface{}) *StructuredError {
	e.Details = details
	return e
}

// WithSuggestion adds a recovery suggestion to the error
func (e *StructuredError) WithSuggestion(suggestion string) *StructuredError {
	e.Suggestion = suggestion
	return e
}

// Common error constructors

// NewInvalidInput creates an invalid input error
func NewInvalidInput(message string) *StructuredError {
	return New(CodeInvalidInput, ClientError, message).
		WithSuggestion("Check the input parameters and try again")
}

// NewMissingParameter creates a missing parameter error
func NewMissingParameter(param string) *StructuredError {
	return New(CodeMissingParameter, ClientError, fmt.Sprintf("Required parameter '%s' is missing", param)).
		WithSuggestion(fmt.Sprintf("Provide the '%s' parameter", param))
}

// NewInvalidQuery creates an invalid query syntax error
func NewInvalidQuery(message string) *StructuredError {
	return New(CodeInvalidQuery, ClientError, message).
		WithSuggestion("Check query syntax or use simple text search")
}

// NewResourceNotFound creates a resource not found error
func NewResourceNotFound(resourceType, id string) *StructuredError {
	return New(CodeResourceNotFound, ClientError, fmt.Sprintf("%s with ID '%s' not found", resourceType, id)).
		WithSuggestion("Verify the ID and try again")
}

// NewUnauthorized creates an unauthorized error
func NewUnauthorized() *StructuredError {
	return New(CodeUnauthorized, ClientError, "Authentication required or credentials invalid").
		WithSuggestion("Check your API key and try again")
}

// NewRateLimitExceeded creates a rate limit exceeded error
func NewRateLimitExceeded() *StructuredError {
	return New(CodeRateLimitExceeded, ClientError, "Rate limit exceeded").
		WithSuggestion("Wait a moment and try again")
}

// NewInternalError creates an internal server error
func NewInternalError(message string) *StructuredError {
	return New(CodeInternalError, ServerError, message).
		WithSuggestion("Try again later or contact support if the issue persists")
}

// NewServiceUnavailable creates a service unavailable error
func NewServiceUnavailable() *StructuredError {
	return New(CodeServiceUnavailable, ServerError, "Service temporarily unavailable").
		WithSuggestion("Try again in a few moments")
}

// NewTimeout creates a timeout error
func NewTimeout(operation string) *StructuredError {
	return New(CodeTimeout, ServerError, fmt.Sprintf("Operation '%s' timed out", operation)).
		WithSuggestion("Try again or adjust timeout settings")
}

// NewAPIError creates an external API error
func NewAPIError(service string, statusCode int, message string) *StructuredError {
	return New(CodeAPIError, ExternalError, fmt.Sprintf("%s API error (HTTP %d): %s", service, statusCode, message)).
		WithDetails(map[string]interface{}{
			"service":     service,
			"status_code": statusCode,
		}).
		WithSuggestion("Check IBM Cloud Logs service status")
}

// NewAuthFailed creates an authentication failed error
func NewAuthFailed(message string) *StructuredError {
	return New(CodeAuthFailed, ExternalError, message).
		WithSuggestion("Check your IBM Cloud API key and permissions")
}

// NewNetworkError creates a network error
func NewNetworkError(message string) *StructuredError {
	return New(CodeNetworkError, ExternalError, message).
		WithSuggestion("Check your network connection and try again")
}

// sensitivePatterns defines regex patterns for detecting and redacting sensitive
// information in API error responses before they are propagated to AI agents.
var sensitivePatterns = []*regexp.Regexp{
	// Email addresses (might contain internal user info) - check first to avoid partial matches
	regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),

	// API keys, tokens, secrets in various contexts
	// Pattern 1: key/token followed by separator and value (api_key: xxx, token=xxx)
	regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|credential|auth)["\s:=]+([a-zA-Z0-9_\-\.]{10,})`),
	// Pattern 2: key/token within error messages ("Invalid API key: xxx")
	regexp.MustCompile(`(?i)(api[_-]?\s*key|token|secret|password|credential)\s*[:\s]+\s*([a-zA-Z0-9_\-\.]{10,})`),
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{15,}`),
	regexp.MustCompile(`(?i)(apikey|access[_-]?token)["\s:=]+[a-zA-Z0-9_\-\.]{10,}`),

	// Standalone tokens after "token" keyword (catches "token abc123..." pattern)
	regexp.MustCompile(`(?i)\btoken\s+[a-zA-Z0-9_\-\.]{10,}`),

	// File paths (Unix and Windows)
	regexp.MustCompile(`/(?:[a-zA-Z0-9_\-\.]+/){2,}[a-zA-Z0-9_\-\.]*`),              // Unix paths: /usr/local/bin/app
	regexp.MustCompile(`[a-zA-Z]:\\(?:[^\\/:*?"<>|\r\n]+\\){1,}[^\\/:*?"<>|\r\n]*`), // Windows paths: C:\Program Files\App

	// Internal URLs and endpoints
	regexp.MustCompile(`(?i)https?://(?:localhost|127\.0\.0\.1|10\.\d+\.\d+\.\d+|172\.(?:1[6-9]|2\d|3[01])\.\d+\.\d+|192\.168\.\d+\.\d+)[:\d]*/\S+`),
	// Match internal/private domains (.internal, .local, .corp, .private)
	regexp.MustCompile(`(?i)https?://[a-zA-Z0-9\-\.]+\.(?:internal|local|corp|private)[:\d/]\S*`),
	// Match URLs with "internal" or "api" in subdomain (internal.api.company.com, api.internal.company.com)
	regexp.MustCompile(`(?i)https?://[a-zA-Z0-9\-]*(?:internal|api)[a-zA-Z0-9\-]*\.[a-zA-Z0-9\-\.]+/\S*`),

	// Database connection strings
	regexp.MustCompile(`(?i)(mongodb|postgresql|mysql|redis)://[^\s]+`),

	// Stack traces (Java, Python, Go, Node.js patterns)
	regexp.MustCompile(`(?m)^\s*at\s+[\w\.$<>]+\([^)]+\)$`),    // Java/JavaScript
	regexp.MustCompile(`(?m)^\s*File\s+"[^"]+",\s+line\s+\d+`), // Python
	regexp.MustCompile(`(?m)^\s*[\w/\.]+:\d+\s+\+0x[0-9a-f]+`), // Go

	// UUIDs (might be session IDs or internal identifiers)
	regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),

	// IP addresses (internal network topology)
	regexp.MustCompile(`\b(?:10\.\d+\.\d+\.\d+|172\.(?:1[6-9]|2\d|3[01])\.\d+\.\d+|192\.168\.\d+\.\d+)\b`),
}

// sanitizeErrorBody removes potentially sensitive information from API error
// response bodies before they are included in error messages shown to AI agents.
// This prevents accidental disclosure of credentials, internal paths, stack traces,
// or other sensitive details that might be present in raw API error responses.
func sanitizeErrorBody(body string) string {
	if body == "" {
		return body
	}

	sanitized := body

	// Apply all sensitive pattern redactions
	for _, pattern := range sensitivePatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "[REDACTED]")
	}

	// If the entire body was redacted (common for auth errors with tokens),
	// return a generic message instead of "[REDACTED]"
	if strings.TrimSpace(sanitized) == "[REDACTED]" {
		return "Error details redacted for security"
	}

	// Remove excessive whitespace that might remain after redaction
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}

// FromHTTPStatus creates an appropriate error from HTTP status code and response body.
// The returned error always has StatusCode set to statusCode, and the response body
// (or a bounded snippet of it, if the caller truncated it) is sanitized to remove
// sensitive information before being folded into Message.
func FromHTTPStatus(statusCode int, responseBody string) *StructuredError {
	// Sanitize the response body before including it in any error message
	sanitizedBody := sanitizeErrorBody(responseBody)

	var se *StructuredError
	switch {
	case statusCode == 400:
		se = NewInvalidInput(fmt.Sprintf("Invalid request: %s", sanitizedBody))
	case statusCode == 401:
		se = New(CodeUnauthorized, ClientError, fmt.Sprintf("Authentication required or credentials invalid: %s", sanitizedBody)).
			WithSuggestion("Check your API key and try again")
	case statusCode == 403:
		se = New(CodeForbidden, ClientError, fmt.Sprintf("Access forbidden: %s", sanitizedBody)).
			WithSuggestion("Check your permissions for this resource")
	case statusCode == 404:
		se = New(CodeResourceNotFound, ClientError, fmt.Sprintf("Resource not found: %s", sanitizedBody))
	case statusCode == 409:
		se = New(CodeConflict, ClientError, fmt.Sprintf("Resource conflict: %s", sanitizedBody)).
			WithSuggestion("Resource may already exist or be in use")
	case statusCode == 429:
		se = New(CodeRateLimitExceeded, ClientError, fmt.Sprintf("Rate limit exceeded: %s", sanitizedBody)).
			WithSuggestion("Wait a moment and try again")
	case statusCode >= 500 && statusCode < 600:
		se = NewAPIError("IBM Cloud Logs", statusCode, sanitizedBody)
	default:
		se = New(CodeInternalError, ServerError, fmt.Sprintf("Unexpected HTTP status %d: %s", statusCode, sanitizedBody))
	}
	se.StatusCode = statusCode
	return se
}
