package mcperrors

import (
	"testing"
)

func TestStructuredError(t *testing.T) {
	tests := []struct {
		name     string
		error    *StructuredError
		wantCode ErrorCode
		wantCat  ErrorCategory
	}{
		{
			name:     "invalid input error",
			error:    NewInvalidInput("test message"),
			wantCode: CodeInvalidInput,
			wantCat:  ClientError,
		},
		{
			name:     "missing parameter error",
			error:    NewMissingParameter("param1"),
			wantCode: CodeMissingParameter,
			wantCat:  ClientError,
		},
		{
			name:     "invalid query error",
			error:    NewInvalidQuery("syntax error"),
			wantCode: CodeInvalidQuery,
			wantCat:  ClientError,
		},
		{
			name:     "resource not found error",
			error:    NewResourceNotFound("alert", "123"),
			wantCode: CodeResourceNotFound,
			wantCat:  ClientError,
		},
		{
			name:     "unauthorized error",
			error:    NewUnauthorized(),
			wantCode: CodeUnauthorized,
			wantCat:  ClientError,
		},
		{
			name:     "rate limit exceeded error",
			error:    NewRateLimitExceeded(),
			wantCode: CodeRateLimitExceeded,
			wantCat:  ClientError,
		},
		{
			name:     "internal error",
			error:    NewInternalError("something went wrong"),
			wantCode: CodeInternalError,
			wantCat:  ServerError,
		},
		{
			name:     "service unavailable error",
			error:    NewServiceUnavailable(),
			wantCode: CodeServiceUnavailable,
			wantCat:  ServerError,
		},
		{
			name:     "timeout error",
			error:    NewTimeout("query"),
			wantCode: CodeTimeout,
			wantCat:  ServerError,
		},
		{
			name:     "API error",
			error:    NewAPIError("IBM Cloud Logs", 500, "internal error"),
			wantCode: CodeAPIError,
			wantCat:  ExternalError,
		},
		{
			name:     "auth failed error",
			error:    NewAuthFailed("invalid credentials"),
			wantCode: CodeAuthFailed,
			wantCat:  ExternalError,
		},
		{
			name:     "network error",
			error:    NewNetworkError("connection refused"),
			wantCode: CodeNetworkError,
			wantCat:  ExternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", tt.error.Code, tt.wantCode)
			}
			if tt.error.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", tt.error.Category, tt.wantCat)
			}
			if tt.error.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestStructuredErrorWithDetails(t *testing.T) {
	err := NewInvalidInput("test").WithDetails(map[string]interface{}{
		"field": "name",
		"value": "invalid",
	})

	if err.Details == nil {
		t.Error("Details should not be nil")
	}

	details, ok := err.Details.(map[string]interface{})
	if !ok {
		t.Error("Details should be a map")
	}

	if details["field"] != "name" {
		t.Errorf("Details[field] = %v, want 'name'", details["field"])
	}
}

func TestStructuredErrorWithSuggestion(t *testing.T) {
	err := NewInvalidInput("test").WithSuggestion("try again")

	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}

	if err.Suggestion != "try again" {
		t.Errorf("Suggestion = %v, want 'try again'", err.Suggestion)
	}
}

func TestStructuredErrorToJSON(t *testing.T) {
	err := NewInvalidInput("test message")
	json := err.ToJSON()

	if json == "" {
		t.Error("JSON should not be empty")
	}

	// Should contain the code
	if !contains(json, string(CodeInvalidInput)) {
		t.Errorf("JSON should contain code: %s", json)
	}

	// Should contain the category
	if !contains(json, string(ClientError)) {
		t.Errorf("JSON should contain category: %s", json)
	}

	// Should contain the message
	if !contains(json, "test message") {
		t.Errorf("JSON should contain message: %s", json)
	}
}

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   ErrorCode
		wantCat    ErrorCategory
	}{
		{
			name:       "400 bad request",
			statusCode: 400,
			body:       "invalid input",
			wantCode:   CodeInvalidInput,
			wantCat:    ClientError,
		},
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       "unauthorized",
			wantCode:   CodeUnauthorized,
			wantCat:    ClientError,
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			body:       "forbidden",
			wantCode:   CodeForbidden,
			wantCat:    ClientError,
		},
		{
			name:       "404 not found",
			statusCode: 404,
			body:       "not found",
			wantCode:   CodeResourceNotFound,
			wantCat:    ClientError,
		},
		{
			name:       "409 conflict",
			statusCode: 409,
			body:       "conflict",
			wantCode:   CodeConflict,
			wantCat:    ClientError,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			body:       "too many requests",
			wantCode:   CodeRateLimitExceeded,
			wantCat:    ClientError,
		},
		{
			name:       "500 internal error",
			statusCode: 500,
			body:       "internal error",
			wantCode:   CodeAPIError,
			wantCat:    ExternalError,
		},
		{
			name:       "503 service unavailable",
			statusCode: 503,
			body:       "service unavailable",
			wantCode:   CodeAPIError,
			wantCat:    ExternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromHTTPStatus(tt.statusCode, tt.body)

			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", err.Category, tt.wantCat)
			}

			if err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestErrorInterface(t *testing.T) {
	err := NewInvalidInput("test")

	// Should implement error interface
	var _ error = err

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}

	// Should contain code
	if !contains(errStr, string(CodeInvalidInput)) {
		t.Errorf("Error() should contain code: %s", errStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
