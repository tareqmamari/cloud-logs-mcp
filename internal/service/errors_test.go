package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewAgentError(t *testing.T) {
	err := NewAgentError(ErrInvalidQuery, "bad query", ActionChangeParams, "fix it")

	if err.Code != ErrInvalidQuery {
		t.Errorf("Code = %s, want %s", err.Code, ErrInvalidQuery)
	}
	if err.Message != "bad query" {
		t.Errorf("Message = %q, want %q", err.Message, "bad query")
	}
	if err.Action != ActionChangeParams {
		t.Errorf("Action = %s, want %s", err.Action, ActionChangeParams)
	}
	if err.Category != ErrorTypeClient {
		t.Errorf("Category = %s, want %s", err.Category, ErrorTypeClient)
	}
}

func TestAgentActionableError_Error(t *testing.T) {
	err := NewAgentError(ErrInvalidQuery, "bad query", ActionChangeParams, "fix it")
	msg := err.Error()

	if !strings.Contains(msg, "INVALID_QUERY") {
		t.Errorf("Error() should contain code: %s", msg)
	}
	if !strings.Contains(msg, "bad query") {
		t.Errorf("Error() should contain message: %s", msg)
	}
	if !strings.Contains(msg, "CHANGE_PARAMS") {
		t.Errorf("Error() should contain action: %s", msg)
	}
}

func TestAgentActionableError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := NewAgentError(ErrServerError, "wrapped", ActionRetry, "retry").WithCause(cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() should return original cause")
	}
}

func TestAgentActionableError_Chaining(t *testing.T) {
	err := NewAgentError(ErrResourceNotFound, "not found", ActionElicit, "ask user").
		WithResource("alert", "alert-123").
		WithHTTPStatus(404).
		WithRetryAfter(5000).
		WithSuggestedParams(ParamSuggestion{
			Param:        "id",
			CurrentValue: "alert-123",
			SuggestValue: "Use list_alerts to find valid IDs",
			Reason:       "resource not found",
		}).
		WithElicitQuestions("Would you like to list alerts?")

	if err.ResourceType != "alert" {
		t.Errorf("ResourceType = %q, want %q", err.ResourceType, "alert")
	}
	if err.ResourceID != "alert-123" {
		t.Errorf("ResourceID = %q, want %q", err.ResourceID, "alert-123")
	}
	if err.HTTPStatus != 404 {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, 404)
	}
	if err.RetryAfterMs != 5000 {
		t.Errorf("RetryAfterMs = %d, want %d", err.RetryAfterMs, 5000)
	}
	if len(err.SuggestedParams) != 1 {
		t.Fatalf("SuggestedParams len = %d, want 1", len(err.SuggestedParams))
	}
	if len(err.ElicitQuestions) != 1 {
		t.Fatalf("ElicitQuestions len = %d, want 1", len(err.ElicitQuestions))
	}
}

func TestAgentActionableError_ToJSON(t *testing.T) {
	err := NewAgentError(ErrRateLimited, "rate limited", ActionRetryWithBackoff, "wait").
		WithRetryAfter(3000)

	jsonStr := err.ToJSON()

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(jsonStr), &parsed); jsonErr != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", jsonErr)
	}

	if parsed["code"] != "RATE_LIMITED" {
		t.Errorf("JSON code = %v, want RATE_LIMITED", parsed["code"])
	}
	if parsed["action"] != "RETRY_WITH_BACKOFF" {
		t.Errorf("JSON action = %v, want RETRY_WITH_BACKOFF", parsed["action"])
	}
	if parsed["retry_after_ms"].(float64) != 3000 {
		t.Errorf("JSON retry_after_ms = %v, want 3000", parsed["retry_after_ms"])
	}
}

func TestAgentActionableError_FormatForAgent(t *testing.T) {
	err := NewAgentError(ErrQueryTimeout, "query timed out", ActionChangeParams, "reduce scope").
		WithRetryAfter(1000).
		WithSuggestedParams(ParamSuggestion{
			Param:        "limit",
			CurrentValue: "10000",
			SuggestValue: "100",
			Reason:       "reduce result set",
		}).
		WithElicitQuestions("Narrow the time range?")

	formatted := err.FormatForAgent()

	requiredSections := []string{
		"Error:", "query timed out",
		"Code:", "QUERY_TIMEOUT",
		"Recommended Action",
		"CHANGE_PARAMS",
		"Wait 1000ms",
		"Suggested Parameter Changes",
		"limit", "10000", "100",
		"Questions to Ask User",
		"Narrow the time range?",
	}

	for _, section := range requiredSections {
		if !strings.Contains(formatted, section) {
			t.Errorf("FormatForAgent() missing section %q in:\n%s", section, formatted)
		}
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected ErrorType
	}{
		{ErrInvalidQuery, ErrorTypeClient},
		{ErrInvalidDateRange, ErrorTypeClient},
		{ErrMissingParameter, ErrorTypeClient},
		{ErrInvalidParameter, ErrorTypeClient},
		{ErrQueryTooLong, ErrorTypeClient},
		{ErrLimitExceeded, ErrorTypeClient},
		{ErrResourceNotFound, ErrorTypeResource},
		{ErrResourceConflict, ErrorTypeResource},
		{ErrResourceLocked, ErrorTypeResource},
		{ErrUnauthorized, ErrorTypeAuth},
		{ErrForbidden, ErrorTypeAuth},
		{ErrInvalidAPIKey, ErrorTypeAuth},
		{ErrTokenExpired, ErrorTypeAuth},
		{ErrRateLimited, ErrorTypeClient},
		{ErrQuotaExceeded, ErrorTypeClient},
		{ErrServerError, ErrorTypeServer},
		{ErrServiceUnavailable, ErrorTypeServer},
		{ErrTimeout, ErrorTypeServer},
		{ErrNetworkError, ErrorTypeNetwork},
		{ErrQueryTimeout, ErrorTypeQuery},
		{ErrQuerySyntax, ErrorTypeQuery},
		{ErrNoResults, ErrorTypeQuery},
		{ErrResultsTruncated, ErrorTypeQuery},
		{ErrorCode("UNKNOWN_CODE"), ErrorTypeServer}, // default case
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			result := categorizeError(tt.code)
			if result != tt.expected {
				t.Errorf("categorizeError(%s) = %s, want %s", tt.code, result, tt.expected)
			}
		})
	}
}

func TestFromHTTPError(t *testing.T) {
	tests := []struct {
		statusCode     int
		expectedCode   ErrorCode
		expectedAction AgentAction
	}{
		{http.StatusBadRequest, ErrInvalidParameter, ActionChangeParams},
		{http.StatusUnauthorized, ErrUnauthorized, ActionEscalate},
		{http.StatusForbidden, ErrForbidden, ActionEscalate},
		{http.StatusNotFound, ErrResourceNotFound, ActionElicit},
		{http.StatusConflict, ErrResourceConflict, ActionElicit},
		{http.StatusTooManyRequests, ErrRateLimited, ActionRetryWithBackoff},
		{http.StatusInternalServerError, ErrServerError, ActionRetry},
		{http.StatusBadGateway, ErrServerError, ActionRetry},
		{http.StatusServiceUnavailable, ErrServerError, ActionRetry},
		{http.StatusGatewayTimeout, ErrQueryTimeout, ActionChangeParams},
		{418, ErrServerError, ActionEscalate}, // I'm a teapot - unknown status
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			err := FromHTTPError(tt.statusCode, "test body", "alert")

			if err.Code != tt.expectedCode {
				t.Errorf("Code = %s, want %s", err.Code, tt.expectedCode)
			}
			if err.Action != tt.expectedAction {
				t.Errorf("Action = %s, want %s", err.Action, tt.expectedAction)
			}
		})
	}
}

func TestErrorFactoryFunctions(t *testing.T) {
	t.Run("NewQuerySyntaxError", func(t *testing.T) {
		err := NewQuerySyntaxError("bad query", "missing source")
		if err.Code != ErrQuerySyntax {
			t.Errorf("Code = %s, want %s", err.Code, ErrQuerySyntax)
		}
		if err.Action != ActionChangeParams {
			t.Errorf("Action = %s, want %s", err.Action, ActionChangeParams)
		}
		if len(err.SuggestedParams) == 0 {
			t.Error("Expected suggested params")
		}
	})

	t.Run("NewDateRangeError", func(t *testing.T) {
		err := NewDateRangeError("2024-01-02", "2024-01-01")
		if err.Code != ErrInvalidDateRange {
			t.Errorf("Code = %s, want %s", err.Code, ErrInvalidDateRange)
		}
		if len(err.SuggestedParams) != 2 {
			t.Errorf("Expected 2 suggested params, got %d", len(err.SuggestedParams))
		}
	})

	t.Run("NewResourceNotFoundError", func(t *testing.T) {
		err := NewResourceNotFoundError("alert", "a-123", "list_alerts")
		if err.Code != ErrResourceNotFound {
			t.Errorf("Code = %s, want %s", err.Code, ErrResourceNotFound)
		}
		if err.ResourceType != "alert" {
			t.Errorf("ResourceType = %q, want %q", err.ResourceType, "alert")
		}
		if len(err.ElicitQuestions) == 0 {
			t.Error("Expected elicit questions")
		}
	})

	t.Run("NewRateLimitError", func(t *testing.T) {
		err := NewRateLimitError(5000)
		if err.Code != ErrRateLimited {
			t.Errorf("Code = %s, want %s", err.Code, ErrRateLimited)
		}
		if err.RetryAfterMs != 5000 {
			t.Errorf("RetryAfterMs = %d, want 5000", err.RetryAfterMs)
		}
	})

	t.Run("NewAuthError", func(t *testing.T) {
		err := NewAuthError("invalid key")
		if err.Code != ErrUnauthorized {
			t.Errorf("Code = %s, want %s", err.Code, ErrUnauthorized)
		}
		if err.Action != ActionEscalate {
			t.Errorf("Action = %s, want %s", err.Action, ActionEscalate)
		}
	})

	t.Run("NewTimeoutError", func(t *testing.T) {
		err := NewTimeoutError("DataPrime")
		if err.Code != ErrQueryTimeout {
			t.Errorf("Code = %s, want %s", err.Code, ErrQueryTimeout)
		}
		if !strings.Contains(err.Message, "DataPrime") {
			t.Errorf("Message should mention query type: %s", err.Message)
		}
	})

	t.Run("NewServerError retryable", func(t *testing.T) {
		err := NewServerError("internal", true)
		if err.Action != ActionRetry {
			t.Errorf("Retryable server error should have RETRY action, got %s", err.Action)
		}
	})

	t.Run("NewServerError non-retryable", func(t *testing.T) {
		err := NewServerError("permanent", false)
		if err.Action != ActionEscalate {
			t.Errorf("Non-retryable server error should have ESCALATE action, got %s", err.Action)
		}
	})
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string", 10, "this is..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
