package mcperrors

import (
	"strings"
	"testing"
)

func TestSanitizeErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFree []string // strings that should NOT appear in output
		wantHas  []string // strings that SHOULD appear in output
	}{
		{
			name:     "empty string",
			input:    "",
			wantFree: []string{},
			wantHas:  []string{""},
		},
		{
			name:     "safe error message",
			input:    "Invalid query syntax at line 5",
			wantFree: []string{},
			wantHas:  []string{"Invalid query syntax at line 5"},
		},
		{
			name:     "API key in error",
			input:    `Authentication failed: invalid api_key "abcdef1234567890abcdef1234567890"`, // pragma: allowlist secret
			wantFree: []string{"abcdef1234567890abcdef1234567890"},                                // pragma: allowlist secret
			wantHas:  []string{"Authentication failed", "[REDACTED]"},
		},
		{
			name:     "bearer token",
			input:    `Unauthorized: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`, // pragma: allowlist secret
			wantFree: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},                                                                                    // pragma: allowlist secret
			wantHas:  []string{"Unauthorized", "[REDACTED]"},
		},
		{
			name:     "Unix file path",
			input:    `Error reading file: /usr/local/app/config/secrets.json not found`,
			wantFree: []string{"/usr/local/app/config/secrets.json"},
			wantHas:  []string{"Error reading file", "not found", "[REDACTED]"},
		},
		{
			name:     "Windows file path",
			input:    `Cannot access C:\Program Files\MyApp\config\database.ini`,
			wantFree: []string{`C:\Program Files\MyApp\config\database.ini`},
			wantHas:  []string{"Cannot access", "[REDACTED]"},
		},
		{
			name:     "internal URL",
			input:    `Failed to connect to http://localhost:8080/internal/admin/users`, // pragma: allowlist secret
			wantFree: []string{"http://localhost:8080/internal/admin/users"},
			wantHas:  []string{"Failed to connect", "[REDACTED]"},
		},
		{
			name:     "private IP address",
			input:    `Connection refused to 192.168.1.100:5432`,
			wantFree: []string{"192.168.1.100"},
			wantHas:  []string{"Connection refused", "[REDACTED]"},
		},
		{
			name:     "database connection string",
			input:    `DB error: postgresql://user:pass@db.internal:5432/mydb`, // pragma: allowlist secret
			wantFree: []string{"postgresql://user:pass@db.internal:5432/mydb"}, // pragma: allowlist secret
			wantHas:  []string{"DB error", "[REDACTED]"},
		},
		{
			name: "Java stack trace",
			input: `Exception in thread "main" java.lang.NullPointerException
	at com.example.MyClass.myMethod(MyClass.java:42)
	at com.example.Main.main(Main.java:10)`,
			wantFree: []string{"at com.example.MyClass.myMethod"},
			wantHas:  []string{"Exception in thread", "[REDACTED]"},
		},
		{
			name: "Python stack trace",
			input: `Traceback (most recent call last):
  File "/app/main.py", line 15, in <module>
    result = process_data()
  File "/app/processor.py", line 42, in process_data
    raise ValueError("Invalid input")`,
			wantFree: []string{"/app/main.py", "/app/processor.py"},
			wantHas:  []string{"Traceback", "[REDACTED]"},
		},
		{
			name: "Go stack trace",
			input: `panic: runtime error: invalid memory address
goroutine 1 [running]:
main.processRequest(0xc000010240)
	/home/user/app/handler.go:123 +0x85
main.main()
	/home/user/app/main.go:45 +0x2a`,
			wantFree: []string{"/home/user/app/handler.go", "/home/user/app/main.go"},
			wantHas:  []string{"panic", "[REDACTED]"},
		},
		{
			name:     "email address",
			input:    `User admin@internal.company.com not authorized`,
			wantFree: []string{"admin@internal.company.com"},
			wantHas:  []string{"User", "not authorized", "[REDACTED]"},
		},
		{
			name:     "UUID session ID",
			input:    `Session expired: 550e8400-e29b-41d4-a716-446655440000`,
			wantFree: []string{"550e8400-e29b-41d4-a716-446655440000"},
			wantHas:  []string{"Session expired", "[REDACTED]"},
		},
		{
			name:     "multiple sensitive items",
			input:    `Auth failed for user@example.com with token abc123def456ghi789 from IP 10.0.1.50`, // pragma: allowlist secret
			wantFree: []string{"user@example.com", "abc123def456ghi789", "10.0.1.50"},
			wantHas:  []string{"Auth failed", "[REDACTED]"},
		},
		{
			name:     "apikey format",
			input:    `Invalid apikey: test_key_51H7xxxxxxxxxxxxxxxxxxxxxxxxxxx`, // pragma: allowlist secret
			wantFree: []string{"test_key_51H7xxxxxxxxxxxxxxxxxxxxxxxxxxx"},
			wantHas:  []string{"Invalid", "[REDACTED]"},
		},
		{
			name:     "access token format",
			input:    `Expired access_token=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`, // pragma: allowlist secret
			wantFree: []string{"ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},            // pragma: allowlist secret
			wantHas:  []string{"Expired", "[REDACTED]"},
		},
		{
			name:     "internal service URL",
			input:    `Cannot reach https://api.internal.company.com/v1/users`, // pragma: allowlist secret
			wantFree: []string{"https://api.internal.company.com/v1/users"},    // pragma: allowlist secret
			wantHas:  []string{"Cannot reach", "[REDACTED]"},
		},
		{
			name:     "MongoDB connection string",
			input:    `Connection failed: mongodb://admin:password@cluster.internal:27017/mydb`, // pragma: allowlist secret
			wantFree: []string{"mongodb://admin:password@cluster.internal:27017/mydb"},          // pragma: allowlist secret
			wantHas:  []string{"Connection failed", "[REDACTED]"},
		},
		{
			name:     "all redacted becomes generic message",
			input:    `api_key: abcdefghijklmnopqrstuvwxyz123456`,  // pragma: allowlist secret
			wantFree: []string{"abcdefghijklmnopqrstuvwxyz123456"}, // pragma: allowlist secret
			wantHas:  []string{"Error details redacted for security"},
		},
		{
			name:     "mixed safe and sensitive content",
			input:    `Query failed: syntax error near 'SELECT' (token: abc123def456ghi789jkl012)`, // pragma: allowlist secret
			wantFree: []string{"abc123def456ghi789jkl012"},                                         // pragma: allowlist secret
			wantHas:  []string{"Query failed", "syntax error", "SELECT", "[REDACTED]"},
		},
		{
			name:     "preserve HTTP status codes",
			input:    `HTTP 401: Unauthorized - invalid credentials`,
			wantFree: []string{},
			wantHas:  []string{"HTTP 401", "Unauthorized", "invalid credentials"},
		},
		{
			name:     "preserve error codes",
			input:    `Error code E1234: Resource not found`,
			wantFree: []string{},
			wantHas:  []string{"Error code E1234", "Resource not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeErrorBody(tt.input)

			// Check that sensitive strings are NOT present
			for _, sensitive := range tt.wantFree {
				if strings.Contains(got, sensitive) {
					t.Errorf("sanitizeErrorBody() still contains sensitive data %q\nInput:  %s\nOutput: %s",
						sensitive, tt.input, got)
				}
			}

			// Check that expected strings ARE present
			for _, expected := range tt.wantHas {
				if !strings.Contains(got, expected) {
					t.Errorf("sanitizeErrorBody() missing expected content %q\nInput:  %s\nOutput: %s",
						expected, tt.input, got)
				}
			}
		})
	}
}

func TestFromHTTPStatus_Sanitization(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantCode     ErrorCode
		wantCategory ErrorCategory
		wantFree     []string // sensitive data that should NOT appear
		wantHas      []string // safe content that SHOULD appear
	}{
		{
			name:         "401 with API key in response",
			statusCode:   401,
			responseBody: `{"error": "Invalid API key: sk_live_xxxxxxxxxxxxxxxx"}`,
			wantCode:     CodeUnauthorized,
			wantCategory: ClientError,
			wantFree:     []string{"sk_live_xxxxxxxxxxxxxxxx"},
			wantHas:      []string{"Authentication required", "[REDACTED]"},
		},
		{
			name:         "500 with stack trace",
			statusCode:   500,
			responseBody: "Internal error at /usr/local/app/handler.go:123",
			wantCode:     CodeAPIError,
			wantCategory: ExternalError,
			wantFree:     []string{"/usr/local/app/handler.go"},
			wantHas:      []string{"IBM Cloud Logs", "[REDACTED]"},
		},
		{
			name:         "403 with internal URL",
			statusCode:   403,
			responseBody: "Access denied to http://internal.api.company.com/admin",
			wantCode:     CodeForbidden,
			wantCategory: ClientError,
			wantFree:     []string{"http://internal.api.company.com/admin"},
			wantHas:      []string{"Access forbidden", "[REDACTED]"},
		},
		{
			name:         "400 with safe error message",
			statusCode:   400,
			responseBody: "Invalid query syntax: missing closing bracket",
			wantCode:     CodeInvalidInput,
			wantCategory: ClientError,
			wantFree:     []string{},
			wantHas:      []string{"Invalid request", "Invalid query syntax", "missing closing bracket"},
		},
		{
			name:         "429 with safe rate limit message",
			statusCode:   429,
			responseBody: "Rate limit exceeded: 100 requests per minute",
			wantCode:     CodeRateLimitExceeded,
			wantCategory: ClientError,
			wantFree:     []string{},
			wantHas:      []string{"Rate limit exceeded", "100 requests per minute"},
		},
		{
			name:         "404 with UUID",
			statusCode:   404,
			responseBody: "Resource 550e8400-e29b-41d4-a716-446655440000 not found",
			wantCode:     CodeResourceNotFound,
			wantCategory: ClientError,
			wantFree:     []string{"550e8400-e29b-41d4-a716-446655440000"},
			wantHas:      []string{"Resource not found", "[REDACTED]"},
		},
		{
			name:         "500 with database connection string",
			statusCode:   500,
			responseBody: "DB connection failed: postgresql://user:pass@db.internal:5432/logs", // pragma: allowlist secret
			wantCode:     CodeAPIError,
			wantCategory: ExternalError,
			wantFree:     []string{"postgresql://user:pass@db.internal:5432/logs"}, // pragma: allowlist secret
			wantHas:      []string{"IBM Cloud Logs", "[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromHTTPStatus(tt.statusCode, tt.responseBody)

			if err.Code != tt.wantCode {
				t.Errorf("FromHTTPStatus() code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.Category != tt.wantCategory {
				t.Errorf("FromHTTPStatus() category = %v, want %v", err.Category, tt.wantCategory)
			}

			if err.StatusCode != tt.statusCode {
				t.Errorf("FromHTTPStatus() statusCode = %v, want %v", err.StatusCode, tt.statusCode)
			}

			errMsg := err.Message

			// Check that sensitive strings are NOT present in the error message
			for _, sensitive := range tt.wantFree {
				if strings.Contains(errMsg, sensitive) {
					t.Errorf("FromHTTPStatus() message contains sensitive data %q\nMessage: %s",
						sensitive, errMsg)
				}
			}

			// Check that expected safe strings ARE present
			for _, expected := range tt.wantHas {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("FromHTTPStatus() message missing expected content %q\nMessage: %s",
						expected, errMsg)
				}
			}
		})
	}
}

func TestSanitizeErrorBody_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "very long string",
			input: strings.Repeat("a", 10000) + " api_key: " + strings.Repeat("x", 50),
			// Should still redact the API key even in a very long string
		},
		{
			name:  "unicode characters",
			input: "Error: 用户认证失败 token: abc123def456ghi789jkl012",
			// Should preserve unicode but redact token
		},
		{
			name:  "newlines and tabs",
			input: "Error:\n\tapi_key: abc123def456ghi789jkl012\n\tPlease check credentials",
			// Should handle whitespace correctly
		},
		{
			name:  "HTML entities",
			input: "Error: &lt;token&gt;abc123def456ghi789jkl012&lt;/token&gt;",
			// Should redact even with HTML encoding
		},
		{
			name:  "JSON escaped",
			input: `{"error": "Invalid credentials"}`,
			// JSON with safe error message - no credentials to leak
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeErrorBody(tt.input)

			// Basic sanity checks
			if got == "" && tt.input != "" {
				t.Error("sanitizeErrorBody() returned empty string for non-empty input")
			}

			// Should not contain obvious sensitive patterns
			sensitivePatterns := []string{
				"api_key:",
				"token:",
				"password:",
				"secret:",
			}
			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToLower(got), pattern) {
					// Check if there's a long alphanumeric string after the pattern
					idx := strings.Index(strings.ToLower(got), pattern)
					if idx >= 0 && idx+len(pattern)+20 < len(got) {
						substr := got[idx+len(pattern) : idx+len(pattern)+20]
						if isLikelyCredential(substr) {
							t.Errorf("sanitizeErrorBody() may have leaked credential after %q: %s", pattern, substr)
						}
					}
				}
			}
		})
	}
}

// isLikelyCredential checks if a string looks like a credential (long alphanumeric)
func isLikelyCredential(s string) bool {
	if len(s) < 15 {
		return false
	}
	alphanumCount := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			alphanumCount++
		}
	}
	return float64(alphanumCount)/float64(len(s)) > 0.8
}

func BenchmarkSanitizeErrorBody(b *testing.B) {
	testCases := []string{
		"Simple error message",
		"Error with api_key: abc123def456ghi789jkl012mno345pqr678",
		"Complex error with /usr/local/app/config/secrets.json and token: xyz789abc123def456",
		strings.Repeat("Normal text ", 100) + " api_key: " + strings.Repeat("x", 50),
	}

	for _, tc := range testCases {
		b.Run(tc[:min(len(tc), 30)], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = sanitizeErrorBody(tc)
			}
		})
	}
}
