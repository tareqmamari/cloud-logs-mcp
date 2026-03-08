package security

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "standard API key",
			apiKey:   "abcdefghijklmnopqrstuvwxyz1234567890ABCD", //nolint:gosec // test value
			expected: "abcd...ABCD",
		},
		{
			name:     "exactly 9 characters",
			apiKey:   "123456789",
			expected: "1234...6789",
		},
		{
			name:     "exactly 8 characters shows stars",
			apiKey:   "12345678",
			expected: "***",
		},
		{
			name:     "short key",
			apiKey:   "abc",
			expected: "***",
		},
		{
			name:     "empty key",
			apiKey:   "",
			expected: "***",
		},
		{
			name:     "single character",
			apiKey:   "a",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestMaskAPIKey_NeverLeaksFullKey(t *testing.T) {
	keys := []string{
		"abcdefghijklmnopqrstuvwxyz1234567890ABCD",
		"short",
		"exactlyeight!",
		"a-very-long-api-key-that-should-definitely-be-masked-properly-in-all-cases",
	}

	for _, key := range keys {
		masked := MaskAPIKey(key)
		if len(key) > 8 && masked == key {
			t.Errorf("MaskAPIKey returned unmasked key for %q", key)
		}
		if masked != "***" && !strings.Contains(masked, "...") {
			t.Errorf("MaskAPIKey(%q) = %q, expected either '***' or to contain '...'", key, masked)
		}
	}
}

func TestMaskBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "standard bearer token",
			token:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature", //#nosec G101 -- test fixture
			expected: "eyJhbG...ture",
		},
		{
			name:     "short token",
			token:    "shorttkn",
			expected: "***REDACTED***",
		},
		{
			name:     "exactly 10 chars",
			token:    "1234567890",
			expected: "***REDACTED***",
		},
		{
			name:     "11 chars",
			token:    "12345678901",
			expected: "123456...8901",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskBearerToken(tt.token)
			if result != tt.expected {
				t.Errorf("MaskBearerToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveHeaders(t *testing.T) {
	t.Run("masks sensitive headers", func(t *testing.T) {
		headers := map[string][]string{
			"Authorization": {"Bearer eyJtoken..."}, //nolint:gosec // test value
			"X-Api-Key":     {"secret-key-value"},
			"Content-Type":  {"application/json"},
			"X-Request-Id":  {"req-12345"},
		}

		masked := MaskSensitiveHeaders(headers)

		if masked["Authorization"] != "***REDACTED***" {
			t.Errorf("Authorization not masked: %s", masked["Authorization"])
		}
		if masked["X-Api-Key"] != "***REDACTED***" {
			t.Errorf("X-Api-Key not masked: %s", masked["X-Api-Key"])
		}
		if masked["Content-Type"] != "application/json" {
			t.Errorf("Content-Type should not be masked: %s", masked["Content-Type"])
		}
		if masked["X-Request-Id"] != "req-12345" {
			t.Errorf("X-Request-Id should not be masked: %s", masked["X-Request-Id"])
		}
	})

	t.Run("case insensitive header matching", func(t *testing.T) {
		headers := map[string][]string{
			"AUTHORIZATION": {"Bearer token"}, //nolint:gosec // test value
			"Api-Key":       {"key-value"},
			"COOKIE":        {"session=abc"},
		}

		masked := MaskSensitiveHeaders(headers)

		if masked["AUTHORIZATION"] != "***REDACTED***" {
			t.Errorf("AUTHORIZATION not masked: %s", masked["AUTHORIZATION"])
		}
		if masked["Api-Key"] != "***REDACTED***" {
			t.Errorf("Api-Key not masked: %s", masked["Api-Key"])
		}
		if masked["COOKIE"] != "***REDACTED***" {
			t.Errorf("COOKIE not masked: %s", masked["COOKIE"])
		}
	})

	t.Run("all sensitive headers are masked", func(t *testing.T) {
		sensitiveNames := []string{
			"authorization", "x-api-key", "api-key", "apikey",
			"x-auth-token", "cookie", "set-cookie", "x-csrf-token",
		}

		for _, name := range sensitiveNames {
			headers := map[string][]string{
				name: {"sensitive-value"},
			}
			masked := MaskSensitiveHeaders(headers)
			if masked[name] != "***REDACTED***" {
				t.Errorf("Header %q not masked: %s", name, masked[name])
			}
		}
	})

	t.Run("non-sensitive headers preserved", func(t *testing.T) {
		nonSensitive := []string{
			"x-request-id", "x-trace-id", "x-correlation-id",
		}

		for _, name := range nonSensitive {
			headers := map[string][]string{
				name: {"some-value"},
			}
			masked := MaskSensitiveHeaders(headers)
			if masked[name] != "some-value" {
				t.Errorf("Header %q should not be masked: %s", name, masked[name])
			}
		}
	})

	t.Run("multi-value headers truncated", func(t *testing.T) {
		headers := map[string][]string{
			"Accept": {"text/html", "application/json", "text/plain"},
		}

		masked := MaskSensitiveHeaders(headers)
		if !strings.HasSuffix(masked["Accept"], "...") {
			t.Errorf("Multi-value header should end with '...': %s", masked["Accept"])
		}
	})

	t.Run("empty headers map", func(t *testing.T) {
		masked := MaskSensitiveHeaders(map[string][]string{})
		if len(masked) != 0 {
			t.Errorf("Expected empty map, got %v", masked)
		}
	})

	t.Run("empty values slice", func(t *testing.T) {
		headers := map[string][]string{
			"X-Custom": {},
		}
		masked := MaskSensitiveHeaders(headers)
		if val, ok := masked["X-Custom"]; ok && val != "" {
			t.Errorf("Empty values should produce empty result, got %q", val)
		}
	})
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldContain  string
		shouldNotMatch string // substring that should NOT appear in output
	}{
		{
			name:           "masks API key in config",
			input:          `api_key=abcdefghijklmnopqrstuvwxyz1234`,
			shouldContain:  "***REDACTED***",
			shouldNotMatch: "abcdefghijklmnopqrstuvwxyz1234",
		},
		{
			name:           "masks bearer token",
			input:          `Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9`,
			shouldContain:  "***REDACTED***",
			shouldNotMatch: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:           "masks password in URL",
			input:          `password=mysecretpassword123`,
			shouldContain:  "***REDACTED***",
			shouldNotMatch: "mysecretpassword123",
		},
		{
			name:           "masks secret values",
			input:          `secret=abcdefghijklmnopqrstuvwxyz`,
			shouldContain:  "***REDACTED***",
			shouldNotMatch: "abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:          "preserves non-sensitive data",
			input:         `hostname=myserver.example.com port=8080`,
			shouldContain: "hostname=myserver.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.input)
			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Expected result to contain %q, got %q", tt.shouldContain, result)
			}
			if tt.shouldNotMatch != "" && strings.Contains(result, tt.shouldNotMatch) {
				t.Errorf("Result should NOT contain %q, but it does: %q", tt.shouldNotMatch, result)
			}
		})
	}
}

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		shouldContain string
		shouldNotLeak []string // values that must NOT appear
	}{
		{
			name:          "masks api_key parameter",
			url:           "https://api.example.com/v1?api_key=secret123&format=json",
			shouldContain: "***REDACTED***",
			shouldNotLeak: []string{"secret123"},
		},
		{
			name:          "masks token parameter",
			url:           "https://api.example.com?token=mytoken123&limit=10",
			shouldContain: "***REDACTED***",
			shouldNotLeak: []string{"mytoken123"},
		},
		{
			name:          "masks password parameter",
			url:           "https://api.example.com?password=p@ssw0rd&user=admin",
			shouldContain: "***REDACTED***",
			shouldNotLeak: []string{"p@ssw0rd"},
		},
		{
			name:          "preserves non-sensitive parameters",
			url:           "https://api.example.com?limit=10&offset=20",
			shouldContain: "limit=10",
		},
		{
			name:          "masks multiple sensitive parameters",
			url:           "https://api.example.com?api_key=key1&secret=sec1&limit=10",
			shouldContain: "limit=10",
			shouldNotLeak: []string{"key1", "sec1"},
		},
		{
			name:          "handles URL without parameters",
			url:           "https://api.example.com/v1/alerts",
			shouldContain: "https://api.example.com/v1/alerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskURL(tt.url)
			if !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Expected result to contain %q, got %q", tt.shouldContain, result)
			}
			for _, leak := range tt.shouldNotLeak {
				if strings.Contains(result, leak) {
					t.Errorf("URL leaked sensitive value %q: %s", leak, result)
				}
			}
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	sensitive := []string{
		"password", "Password", "PASSWORD",
		"api_key", "API_KEY", "apikey", "ApiKey",
		"secret", "SECRET", "client_secret",
		"token", "Token", "access_token", "auth_token",
		"authorization", "Authorization",
		"credential", "Credentials",
		"private_key", "ssh_key",
		"certificate", "cert",
	}

	for _, field := range sensitive {
		if !IsSensitiveField(field) {
			t.Errorf("IsSensitiveField(%q) = false, want true", field)
		}
	}

	nonSensitive := []string{
		"name", "email", "hostname", "port",
		"limit", "offset", "query", "format",
		"region", "zone", "cluster",
	}

	for _, field := range nonSensitive {
		if IsSensitiveField(field) {
			t.Errorf("IsSensitiveField(%q) = true, want false", field)
		}
	}
}

func TestSanitizeError(t *testing.T) {
	t.Run("nil error returns empty", func(t *testing.T) {
		result := SanitizeError(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil error, got %q", result)
		}
	})

	t.Run("masks sensitive data in error", func(t *testing.T) {
		err := errors.New("authentication failed with api_key=abcdefghijklmnopqrstuvwxyz1234")
		result := SanitizeError(err)
		if strings.Contains(result, "abcdefghijklmnopqrstuvwxyz1234") {
			t.Errorf("Sanitized error leaked API key: %s", result)
		}
		if !strings.Contains(result, "***REDACTED***") {
			t.Errorf("Expected REDACTED marker in sanitized error: %s", result)
		}
	})

	t.Run("preserves non-sensitive error", func(t *testing.T) {
		err := errors.New("connection refused to host:8080")
		result := SanitizeError(err)
		if !strings.Contains(result, "connection refused") {
			t.Errorf("Expected error message preserved: %s", result)
		}
	})
}

func TestMaskSensitiveData_ConcurrentAccess(_ *testing.T) {
	// Verify regex patterns are safe for concurrent use
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = MaskSensitiveData("api_key=secret12345678901234567890")
			_ = MaskSensitiveData("Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
			_ = MaskSensitiveData("password=mysecret123")
		}()
	}

	wg.Wait()
	// Test passes if no panic/race detected (run with -race)
}
