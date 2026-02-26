// Package service provides the business logic layer for IBM Cloud Logs operations.
package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AgentAction indicates what action an AI agent should take in response to an error
type AgentAction string

const (
	// ActionRetry indicates the agent should retry the same operation
	ActionRetry AgentAction = "RETRY"
	// ActionRetryWithBackoff indicates the agent should wait then retry
	ActionRetryWithBackoff AgentAction = "RETRY_WITH_BACKOFF"
	// ActionElicit indicates the agent should gather more information from the user
	ActionElicit AgentAction = "ELICIT"
	// ActionChangeParams indicates the agent should modify parameters and retry
	ActionChangeParams AgentAction = "CHANGE_PARAMS"
	// ActionAbandon indicates the operation cannot succeed and should be abandoned
	ActionAbandon AgentAction = "ABANDON"
	// ActionEscalate indicates the issue needs human intervention
	ActionEscalate AgentAction = "ESCALATE"
)

// AgentActionableError provides structured error information for AI agents.
// This enables agents to automatically determine the correct response to errors.
type AgentActionableError struct {
	// Core error information
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Category ErrorType `json:"category"`

	// Agent guidance
	Action          AgentAction       `json:"action"`
	ActionReason    string            `json:"action_reason"`
	RetryAfterMs    int               `json:"retry_after_ms,omitempty"`
	SuggestedParams []ParamSuggestion `json:"suggested_params,omitempty"`
	ElicitQuestions []string          `json:"elicit_questions,omitempty"`

	// Context
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
	RequestID    string `json:"request_id,omitempty"`

	// Original error (not serialized)
	cause error
}

// ErrorCode provides machine-readable error classification
type ErrorCode string

const (
	// ErrInvalidQuery indicates an invalid query syntax or structure.
	ErrInvalidQuery ErrorCode = "INVALID_QUERY"
	// ErrInvalidDateRange indicates the date range is invalid (start after end).
	ErrInvalidDateRange ErrorCode = "INVALID_DATE_RANGE"
	// ErrMissingParameter indicates a required parameter was not provided.
	ErrMissingParameter ErrorCode = "MISSING_PARAMETER"
	// ErrInvalidParameter indicates a parameter value is invalid.
	ErrInvalidParameter ErrorCode = "INVALID_PARAMETER"
	// ErrQueryTooLong indicates the query exceeds the maximum length.
	ErrQueryTooLong ErrorCode = "QUERY_TOO_LONG"
	// ErrLimitExceeded indicates a limit was exceeded.
	ErrLimitExceeded ErrorCode = "LIMIT_EXCEEDED"

	// ErrResourceNotFound indicates the requested resource does not exist.
	ErrResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	// ErrResourceConflict indicates a conflicting resource state.
	ErrResourceConflict ErrorCode = "RESOURCE_CONFLICT"
	// ErrResourceLocked indicates the resource is locked.
	ErrResourceLocked ErrorCode = "RESOURCE_LOCKED"

	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrForbidden indicates insufficient permissions.
	ErrForbidden ErrorCode = "FORBIDDEN"
	// ErrInvalidAPIKey indicates the API key is invalid.
	ErrInvalidAPIKey ErrorCode = "INVALID_API_KEY" // #nosec G101 -- error code constant, not a credential
	// ErrTokenExpired indicates the authentication token has expired.
	ErrTokenExpired ErrorCode = "TOKEN_EXPIRED"

	// ErrRateLimited indicates the request was rate limited.
	ErrRateLimited ErrorCode = "RATE_LIMITED"
	// ErrQuotaExceeded indicates a usage quota was exceeded.
	ErrQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"

	// ErrServerError indicates an internal server error.
	ErrServerError ErrorCode = "SERVER_ERROR"
	// ErrServiceUnavailable indicates the service is temporarily unavailable.
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	// ErrTimeout indicates a general timeout.
	ErrTimeout ErrorCode = "TIMEOUT"
	// ErrNetworkError indicates a network connectivity issue.
	ErrNetworkError ErrorCode = "NETWORK_ERROR"

	// ErrQueryTimeout indicates a query exceeded the time limit.
	ErrQueryTimeout ErrorCode = "QUERY_TIMEOUT"
	// ErrQuerySyntax indicates a query syntax error.
	ErrQuerySyntax ErrorCode = "QUERY_SYNTAX"
	// ErrNoResults indicates the query returned no results.
	ErrNoResults ErrorCode = "NO_RESULTS"
	// ErrResultsTruncated indicates results were truncated due to size limits.
	ErrResultsTruncated ErrorCode = "RESULTS_TRUNCATED"
)

// ErrorType classifies the general type of error
type ErrorType string

const (
	// ErrorTypeClient indicates a client-side error.
	ErrorTypeClient ErrorType = "CLIENT_ERROR"
	// ErrorTypeServer indicates a server-side error.
	ErrorTypeServer ErrorType = "SERVER_ERROR"
	// ErrorTypeNetwork indicates a network error.
	ErrorTypeNetwork ErrorType = "NETWORK_ERROR"
	// ErrorTypeAuth indicates an authentication error.
	ErrorTypeAuth ErrorType = "AUTH_ERROR"
	// ErrorTypeResource indicates a resource-related error.
	ErrorTypeResource ErrorType = "RESOURCE_ERROR"
	// ErrorTypeQuery indicates a query-related error.
	ErrorTypeQuery ErrorType = "QUERY_ERROR"
)

// ParamSuggestion provides a suggested parameter correction
type ParamSuggestion struct {
	Param        string `json:"param"`
	CurrentValue string `json:"current_value,omitempty"`
	SuggestValue string `json:"suggest_value"`
	Reason       string `json:"reason"`
}

// Error implements the error interface
func (e *AgentActionableError) Error() string {
	return fmt.Sprintf("[%s] %s (action: %s)", e.Code, e.Message, e.Action)
}

// Unwrap returns the underlying error
func (e *AgentActionableError) Unwrap() error {
	return e.cause
}

// ToJSON returns the error as a JSON string for agent consumption
func (e *AgentActionableError) ToJSON() string {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"code":"%s","message":"%s","action":"%s"}`, e.Code, e.Message, e.Action)
	}
	return string(data)
}

// FormatForAgent returns a markdown-formatted error message optimized for LLM consumption
func (e *AgentActionableError) FormatForAgent() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## ⚠️ Error: %s\n\n", e.Message)
	fmt.Fprintf(&sb, "**Code:** `%s`\n", e.Code)
	fmt.Fprintf(&sb, "**Category:** %s\n", e.Category)

	sb.WriteString("\n### Recommended Action\n\n")
	fmt.Fprintf(&sb, "**Action:** `%s`\n", e.Action)
	fmt.Fprintf(&sb, "**Reason:** %s\n", e.ActionReason)

	if e.RetryAfterMs > 0 {
		fmt.Fprintf(&sb, "\n⏱️ Wait %dms before retrying.\n", e.RetryAfterMs)
	}

	if len(e.SuggestedParams) > 0 {
		sb.WriteString("\n### Suggested Parameter Changes\n\n")
		for _, p := range e.SuggestedParams {
			if p.CurrentValue != "" {
				fmt.Fprintf(&sb, "- **%s:** Change from `%s` to `%s` (%s)\n", p.Param, p.CurrentValue, p.SuggestValue, p.Reason)
			} else {
				fmt.Fprintf(&sb, "- **%s:** Set to `%s` (%s)\n", p.Param, p.SuggestValue, p.Reason)
			}
		}
	}

	if len(e.ElicitQuestions) > 0 {
		sb.WriteString("\n### Questions to Ask User\n\n")
		for i, q := range e.ElicitQuestions {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, q)
		}
	}

	return sb.String()
}

// NewAgentError creates a new agent-actionable error
func NewAgentError(code ErrorCode, message string, action AgentAction, reason string) *AgentActionableError {
	return &AgentActionableError{
		Code:         code,
		Message:      message,
		Category:     categorizeError(code),
		Action:       action,
		ActionReason: reason,
	}
}

// categorizeError determines the error category from the error code
func categorizeError(code ErrorCode) ErrorType {
	switch code {
	case ErrInvalidQuery, ErrInvalidDateRange, ErrMissingParameter, ErrInvalidParameter, ErrQueryTooLong, ErrLimitExceeded:
		return ErrorTypeClient
	case ErrResourceNotFound, ErrResourceConflict, ErrResourceLocked:
		return ErrorTypeResource
	case ErrUnauthorized, ErrForbidden, ErrInvalidAPIKey, ErrTokenExpired:
		return ErrorTypeAuth
	case ErrRateLimited, ErrQuotaExceeded:
		return ErrorTypeClient
	case ErrServerError, ErrServiceUnavailable, ErrTimeout:
		return ErrorTypeServer
	case ErrNetworkError:
		return ErrorTypeNetwork
	case ErrQueryTimeout, ErrQuerySyntax, ErrNoResults, ErrResultsTruncated:
		return ErrorTypeQuery
	default:
		return ErrorTypeServer
	}
}

// WithCause adds the underlying error
func (e *AgentActionableError) WithCause(err error) *AgentActionableError {
	e.cause = err
	return e
}

// WithResource adds resource context
func (e *AgentActionableError) WithResource(resourceType, resourceID string) *AgentActionableError {
	e.ResourceType = resourceType
	e.ResourceID = resourceID
	return e
}

// WithHTTPStatus adds HTTP status code context
func (e *AgentActionableError) WithHTTPStatus(status int) *AgentActionableError {
	e.HTTPStatus = status
	return e
}

// WithRetryAfter specifies when the agent should retry
func (e *AgentActionableError) WithRetryAfter(ms int) *AgentActionableError {
	e.RetryAfterMs = ms
	return e
}

// WithSuggestedParams adds parameter suggestions
func (e *AgentActionableError) WithSuggestedParams(suggestions ...ParamSuggestion) *AgentActionableError {
	e.SuggestedParams = append(e.SuggestedParams, suggestions...)
	return e
}

// WithElicitQuestions adds questions for the agent to ask the user
func (e *AgentActionableError) WithElicitQuestions(questions ...string) *AgentActionableError {
	e.ElicitQuestions = append(e.ElicitQuestions, questions...)
	return e
}

// ========================================================================
// Error factory functions for common error scenarios
// ========================================================================

// NewQuerySyntaxError creates an error for invalid query syntax
func NewQuerySyntaxError(query string, details string) *AgentActionableError {
	return NewAgentError(
		ErrQuerySyntax,
		fmt.Sprintf("Invalid query syntax: %s", details),
		ActionChangeParams,
		"Fix the query syntax and retry",
	).WithSuggestedParams(ParamSuggestion{
		Param:        "query",
		CurrentValue: truncateString(query, 50),
		SuggestValue: "Use 'source logs | filter <condition> | limit N' format",
		Reason:       details,
	})
}

// NewDateRangeError creates an error for invalid date ranges
func NewDateRangeError(startDate, endDate string) *AgentActionableError {
	return NewAgentError(
		ErrInvalidDateRange,
		fmt.Sprintf("Invalid date range: start_date (%s) must be before end_date (%s)", startDate, endDate),
		ActionChangeParams,
		"Correct the date range and retry",
	).WithSuggestedParams(
		ParamSuggestion{
			Param:        "start_date",
			CurrentValue: startDate,
			SuggestValue: "Use ISO 8601 format: 2024-01-01T00:00:00Z",
			Reason:       "Ensure start_date is before end_date",
		},
		ParamSuggestion{
			Param:        "end_date",
			CurrentValue: endDate,
			SuggestValue: "Use ISO 8601 format: 2024-01-02T00:00:00Z",
			Reason:       "Ensure end_date is after start_date",
		},
	)
}

// NewResourceNotFoundError creates an error for missing resources
func NewResourceNotFoundError(resourceType, resourceID, listTool string) *AgentActionableError {
	return NewAgentError(
		ErrResourceNotFound,
		fmt.Sprintf("%s with ID '%s' not found", resourceType, resourceID),
		ActionElicit,
		fmt.Sprintf("Use '%s' to find valid IDs or ask user for correct ID", listTool),
	).WithResource(resourceType, resourceID).WithElicitQuestions(
		fmt.Sprintf("The %s with ID '%s' was not found. Would you like me to list available %ss?", resourceType, resourceID, resourceType),
	)
}

// NewRateLimitError creates an error for rate limiting
func NewRateLimitError(retryAfterMs int) *AgentActionableError {
	return NewAgentError(
		ErrRateLimited,
		"Rate limit exceeded",
		ActionRetryWithBackoff,
		fmt.Sprintf("Wait %dms before retrying", retryAfterMs),
	).WithRetryAfter(retryAfterMs)
}

// NewAuthError creates an error for authentication failures
func NewAuthError(details string) *AgentActionableError {
	return NewAgentError(
		ErrUnauthorized,
		"Authentication failed: "+details,
		ActionEscalate,
		"User needs to check API key configuration",
	).WithElicitQuestions(
		"There's an authentication issue. Please verify your IBM Cloud API key is correctly configured.",
	)
}

// NewTimeoutError creates an error for query timeouts
func NewTimeoutError(queryType string) *AgentActionableError {
	return NewAgentError(
		ErrQueryTimeout,
		fmt.Sprintf("%s query timed out", queryType),
		ActionChangeParams,
		"Reduce query scope or use background query for large result sets",
	).WithSuggestedParams(
		ParamSuggestion{
			Param:        "limit",
			SuggestValue: "100",
			Reason:       "Reduce result set size",
		},
		ParamSuggestion{
			Param:        "start_date/end_date",
			SuggestValue: "Narrow time range",
			Reason:       "Reduce data scanned",
		},
	).WithElicitQuestions(
		"The query timed out. Would you like me to: 1) Reduce the time range, 2) Add more filters, or 3) Use a background query for this large dataset?",
	)
}

// NewServerError creates an error for server-side issues
func NewServerError(details string, retryable bool) *AgentActionableError {
	action := ActionEscalate
	reason := "Server error requires human investigation"
	if retryable {
		action = ActionRetry
		reason = "Transient error, retry may succeed"
	}

	return NewAgentError(
		ErrServerError,
		"Server error: "+details,
		action,
		reason,
	)
}

// FromHTTPError converts an HTTP status code to an agent-actionable error
func FromHTTPError(statusCode int, body string, resourceType string) *AgentActionableError {
	switch statusCode {
	case http.StatusBadRequest:
		return NewAgentError(
			ErrInvalidParameter,
			fmt.Sprintf("Invalid request: %s", body),
			ActionChangeParams,
			"Review and fix the request parameters",
		)

	case http.StatusUnauthorized:
		return NewAuthError("API key invalid or expired")

	case http.StatusForbidden:
		return NewAgentError(
			ErrForbidden,
			"Access forbidden",
			ActionEscalate,
			"User lacks permission for this operation",
		).WithElicitQuestions(
			"You don't have permission for this operation. Please verify your access level with your administrator.",
		)

	case http.StatusNotFound:
		return NewAgentError(
			ErrResourceNotFound,
			fmt.Sprintf("%s not found", resourceType),
			ActionElicit,
			"Resource doesn't exist - verify ID or list available resources",
		)

	case http.StatusConflict:
		return NewAgentError(
			ErrResourceConflict,
			"Resource conflict",
			ActionElicit,
			"Resource already exists or is in a conflicting state",
		).WithElicitQuestions(
			"A resource conflict occurred. Would you like me to update the existing resource instead?",
		)

	case http.StatusTooManyRequests:
		return NewRateLimitError(5000) // Default 5 second wait

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewServerError(body, true)

	case http.StatusGatewayTimeout:
		return NewTimeoutError("API")

	default:
		return NewAgentError(
			ErrServerError,
			fmt.Sprintf("Unexpected error (HTTP %d): %s", statusCode, body),
			ActionEscalate,
			"Unexpected error requires investigation",
		).WithHTTPStatus(statusCode)
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
