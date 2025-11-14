package errors

import (
	"encoding/json"
	"fmt"
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
	// Client errors
	CodeInvalidInput      ErrorCode = "INVALID_INPUT"
	CodeMissingParameter  ErrorCode = "MISSING_PARAMETER"
	CodeInvalidQuery      ErrorCode = "INVALID_QUERY_SYNTAX"
	CodeResourceNotFound  ErrorCode = "RESOURCE_NOT_FOUND"
	CodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	CodeForbidden         ErrorCode = "FORBIDDEN"
	CodeConflict          ErrorCode = "CONFLICT"
	CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Server errors
	CodeInternalError     ErrorCode = "INTERNAL_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeTimeout            ErrorCode = "TIMEOUT"

	// External errors
	CodeAPIError          ErrorCode = "API_ERROR"
	CodeAuthFailed        ErrorCode = "AUTH_FAILED"
	CodeNetworkError      ErrorCode = "NETWORK_ERROR"
)

// StructuredError represents a detailed error with category, code, and recovery suggestion
type StructuredError struct {
	Code       ErrorCode     `json:"code"`
	Category   ErrorCategory `json:"category"`
	Message    string        `json:"message"`
	Details    interface{}   `json:"details,omitempty"`
	Suggestion string        `json:"suggestion,omitempty"`
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

// FromHTTPStatus creates an appropriate error from HTTP status code
func FromHTTPStatus(statusCode int, responseBody string) *StructuredError {
	switch {
	case statusCode == 400:
		return NewInvalidInput(responseBody)
	case statusCode == 401:
		return NewUnauthorized()
	case statusCode == 403:
		return New(CodeForbidden, ClientError, "Access forbidden").
			WithSuggestion("Check your permissions for this resource")
	case statusCode == 404:
		return New(CodeResourceNotFound, ClientError, "Resource not found")
	case statusCode == 409:
		return New(CodeConflict, ClientError, "Resource conflict").
			WithSuggestion("Resource may already exist or be in use")
	case statusCode == 429:
		return NewRateLimitExceeded()
	case statusCode >= 500 && statusCode < 600:
		return NewAPIError("IBM Cloud Logs", statusCode, responseBody)
	default:
		return New(CodeInternalError, ServerError, fmt.Sprintf("Unexpected HTTP status %d: %s", statusCode, responseBody))
	}
}
