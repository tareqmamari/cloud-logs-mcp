package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/config"
)

// mockAuthenticator implements the Authenticator interface for testing
type mockAuthenticator struct{}

func (m *mockAuthenticator) Authenticate(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer test-token")
	return nil
}

// newTestClient creates a client for testing with a mock authenticator
func newTestClient(serverURL string, version string) *Client {
	cfg := newTestConfig(serverURL)
	logger := newTestLogger()

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	return &Client{
		httpClient:    httpClient,
		config:        cfg,
		logger:        logger,
		authenticator: &mockAuthenticator{},
		version:       version,
	}
}

// newTestLogger creates a no-op logger for testing
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// newTestConfig creates a test configuration pointing to the given server URL
func newTestConfig(serverURL string) *config.Config {
	return &config.Config{
		ServiceURL:      serverURL,
		APIKey:          "test-api-key", // pragma: allowlist secret
		Region:          "us-south",
		Timeout:         5 * time.Second,
		MaxRetries:      2,
		RetryWaitMin:    100 * time.Millisecond,
		RetryWaitMax:    500 * time.Millisecond,
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSVerify:       false, // Disable for test server
		EnableRateLimit: false,
	}
}

func TestConvertToIngressURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard API URL",
			input:    "https://instance-id.api.us-south.logs.cloud.ibm.com",
			expected: "https://instance-id.ingress.us-south.logs.cloud.ibm.com",
		},
		{
			name:     "eu-de region",
			input:    "https://my-instance.api.eu-de.logs.cloud.ibm.com",
			expected: "https://my-instance.ingress.eu-de.logs.cloud.ibm.com",
		},
		{
			name:     "no .api. in URL",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "multiple .api. occurrences - only first replaced",
			input:    "https://api.api.region.api.com",
			expected: "https://api.ingress.region.api.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToIngressURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "connection reset error message",
			err:      &mockError{msg: "connection reset by peer"},
			expected: true,
		},
		{
			name:     "connection refused error message",
			err:      &mockError{msg: "connection refused"},
			expected: true,
		},
		{
			name:     "network unreachable error message",
			err:      &mockError{msg: "network is unreachable"},
			expected: true,
		},
		{
			name:     "i/o timeout error message",
			err:      &mockError{msg: "i/o timeout"},
			expected: true,
		},
		{
			name:     "TLS handshake timeout",
			err:      &mockError{msg: "TLS handshake timeout"},
			expected: true,
		},
		{
			name:     "EOF error",
			err:      &mockError{msg: "EOF"},
			expected: true,
		},
		{
			name:     "unknown error - not retryable",
			err:      &mockError{msg: "some random error"},
			expected: false,
		},
		{
			name:     "authentication error - not retryable",
			err:      &mockError{msg: "invalid credentials"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "429 Too Many Requests",
			statusCode: http.StatusTooManyRequests,
			expected:   true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: http.StatusBadGateway,
			expected:   true,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "504 Gateway Timeout",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
		{
			name:       "200 OK - no retry",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "201 Created - no retry",
			statusCode: http.StatusCreated,
			expected:   false,
		},
		{
			name:       "400 Bad Request - no retry",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "401 Unauthorized - no retry",
			statusCode: http.StatusUnauthorized,
			expected:   false,
		},
		{
			name:       "403 Forbidden - no retry",
			statusCode: http.StatusForbidden,
			expected:   false,
		},
		{
			name:       "404 Not Found - no retry",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLEncoding(t *testing.T) {
	tests := []struct {
		name           string
		query          map[string]string
		expectedParams []string // params that should be in the URL (order may vary)
	}{
		{
			name: "simple query params",
			query: map[string]string{
				"limit": "10",
				"page":  "1",
			},
			expectedParams: []string{"limit=10", "page=1"},
		},
		{
			name: "query params with special characters",
			query: map[string]string{
				"filter": "name=test&value=123",
				"query":  "level:error AND app:myapp",
			},
			expectedParams: []string{
				"filter=name%3Dtest%26value%3D123",
				"query=level%3Aerror+AND+app%3Amyapp",
			},
		},
		{
			name: "query params with spaces",
			query: map[string]string{
				"search": "hello world",
			},
			expectedParams: []string{"search=hello+world"},
		},
		{
			name: "query params with unicode",
			query: map[string]string{
				"name": "日本語",
			},
			expectedParams: []string{"name=%E6%97%A5%E6%9C%AC%E8%AA%9E"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			c := newTestClient(server.URL, "test")

			req := &Request{
				Method: "GET",
				Path:   "/v1/test",
				Query:  tt.query,
			}

			ctx := context.Background()
			_, _ = c.doRequest(ctx, req)

			for _, param := range tt.expectedParams {
				assert.Contains(t, capturedURL, param,
					"URL should contain properly encoded param: %s", param)
			}
		})
	}
}

func TestUserAgentHeader(t *testing.T) {
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "1.2.3")

	req := &Request{
		Method: "GET",
		Path:   "/v1/test",
	}

	ctx := context.Background()
	_, _ = c.doRequest(ctx, req)

	assert.Equal(t, "logs-mcp-server/1.2.3", capturedUserAgent)
}

func TestIdempotencyHeaders(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		requestID         string
		expectXRequestID  bool
		expectIdempotency bool
	}{
		{
			name:              "POST with request ID",
			method:            "POST",
			requestID:         "test-123",
			expectXRequestID:  true,
			expectIdempotency: true,
		},
		{
			name:              "PUT with request ID",
			method:            "PUT",
			requestID:         "test-456",
			expectXRequestID:  true,
			expectIdempotency: true,
		},
		{
			name:              "GET with request ID - no idempotency header",
			method:            "GET",
			requestID:         "test-789",
			expectXRequestID:  true,
			expectIdempotency: false,
		},
		{
			name:              "DELETE with request ID - no idempotency header",
			method:            "DELETE",
			requestID:         "test-abc",
			expectXRequestID:  true,
			expectIdempotency: false,
		},
		{
			name:              "POST without request ID",
			method:            "POST",
			requestID:         "",
			expectXRequestID:  false,
			expectIdempotency: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedHeaders http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeaders = r.Header
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			c := newTestClient(server.URL, "test")

			req := &Request{
				Method:    tt.method,
				Path:      "/v1/test",
				RequestID: tt.requestID,
			}

			ctx := context.Background()
			_, _ = c.doRequest(ctx, req)

			if tt.expectXRequestID {
				assert.Equal(t, tt.requestID, capturedHeaders.Get("X-Request-ID"))
			} else {
				assert.Empty(t, capturedHeaders.Get("X-Request-ID"))
			}

			if tt.expectIdempotency {
				assert.Equal(t, tt.requestID, capturedHeaders.Get("Idempotency-Key"))
			} else {
				assert.Empty(t, capturedHeaders.Get("Idempotency-Key"))
			}
		})
	}
}

func TestAcceptHeader(t *testing.T) {
	tests := []struct {
		name           string
		acceptSSE      bool
		expectedAccept string
	}{
		{
			name:           "standard JSON accept",
			acceptSSE:      false,
			expectedAccept: "application/json",
		},
		{
			name:           "SSE accept for streaming",
			acceptSSE:      true,
			expectedAccept: "text/event-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAccept string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedAccept = r.Header.Get("Accept")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			c := newTestClient(server.URL, "test")

			req := &Request{
				Method:    "GET",
				Path:      "/v1/test",
				AcceptSSE: tt.acceptSSE,
			}

			ctx := context.Background()
			_, _ = c.doRequest(ctx, req)

			assert.Equal(t, tt.expectedAccept, capturedAccept)
		})
	}
}

func TestResponseParsing(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful response",
			statusCode:   http.StatusOK,
			responseBody: `{"id": "123", "name": "test"}`,
			expectError:  false,
		},
		{
			name:         "empty response body",
			statusCode:   http.StatusNoContent,
			responseBody: "",
			expectError:  false,
		},
		{
			name:         "large response body",
			statusCode:   http.StatusOK,
			responseBody: `{"data": "` + strings.Repeat("x", 10000) + `"}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			c := newTestClient(server.URL, "test")

			req := &Request{
				Method: "GET",
				Path:   "/v1/test",
			}

			ctx := context.Background()
			resp, err := c.doRequest(ctx, req)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.statusCode, resp.StatusCode)
				assert.Equal(t, tt.responseBody, string(resp.Body))
			}
		})
	}
}

func TestRequestBody(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = readAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")

	testBody := map[string]interface{}{
		"name":        "test-alert",
		"description": "Test description",
		"enabled":     true,
		"threshold":   42.5,
		"tags":        []string{"tag1", "tag2"},
	}

	req := &Request{
		Method: "POST",
		Path:   "/v1/alerts",
		Body:   testBody,
	}

	ctx := context.Background()
	_, err := c.doRequest(ctx, req)
	require.NoError(t, err)

	// Verify body was sent correctly
	var receivedBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &receivedBody)
	require.NoError(t, err)

	assert.Equal(t, "test-alert", receivedBody["name"])
	assert.Equal(t, "Test description", receivedBody["description"])
	assert.Equal(t, true, receivedBody["enabled"])
	assert.Equal(t, 42.5, receivedBody["threshold"])
}

func TestCustomHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")

	req := &Request{
		Method: "GET",
		Path:   "/v1/test",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Another":       "another-value",
		},
	}

	ctx := context.Background()
	_, _ = c.doRequest(ctx, req)

	assert.Equal(t, "custom-value", capturedHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "another-value", capturedHeaders.Get("X-Another"))
}

func TestContextCancellation(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")

	req := &Request{
		Method: "GET",
		Path:   "/v1/test",
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doRequest(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// Helper types and functions

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return result, nil
}

func TestParseRetryAfter(t *testing.T) {
	c := newTestClient("http://localhost", "test")

	tests := []struct {
		name          string
		retryAfter    string
		expectedMin   time.Duration
		expectedMax   time.Duration
		expectNonZero bool
	}{
		{
			name:          "empty header",
			retryAfter:    "",
			expectNonZero: false,
		},
		{
			name:          "delta-seconds - 30 seconds",
			retryAfter:    "30",
			expectedMin:   30 * time.Second,
			expectedMax:   30 * time.Second,
			expectNonZero: true,
		},
		{
			name:          "delta-seconds - 120 seconds",
			retryAfter:    "120",
			expectedMin:   120 * time.Second,
			expectedMax:   120 * time.Second,
			expectNonZero: true,
		},
		{
			name:          "delta-seconds - very large (capped at 1 hour)",
			retryAfter:    "7200",
			expectedMin:   time.Hour,
			expectedMax:   time.Hour,
			expectNonZero: true,
		},
		{
			name:          "delta-seconds - zero",
			retryAfter:    "0",
			expectNonZero: false,
		},
		{
			name:          "delta-seconds - negative",
			retryAfter:    "-10",
			expectNonZero: false,
		},
		{
			name:          "invalid format",
			retryAfter:    "not-a-number",
			expectNonZero: false,
		},
		{
			name:          "floating point (parsed as valid by Go)",
			retryAfter:    "30.5",
			expectedMin:   30500 * time.Millisecond,
			expectedMax:   30500 * time.Millisecond,
			expectNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := make(http.Header)
			if tt.retryAfter != "" {
				headers.Set("Retry-After", tt.retryAfter)
			}

			result := c.parseRetryAfter(headers)

			if tt.expectNonZero {
				assert.GreaterOrEqual(t, result, tt.expectedMin, "Duration should be >= expected min")
				assert.LessOrEqual(t, result, tt.expectedMax, "Duration should be <= expected max")
			} else {
				assert.Equal(t, time.Duration(0), result, "Expected zero duration")
			}
		})
	}
}

func TestCalculateRetryWait_WithRetryAfterHeader(t *testing.T) {
	c := newTestClient("http://localhost", "test")
	// Override RetryWaitMax to allow testing longer Retry-After values
	c.config.RetryWaitMax = 2 * time.Minute

	// Test with 429 response and Retry-After header
	headers := make(http.Header)
	headers.Set("Retry-After", "60")

	lastResp := &Response{
		StatusCode: http.StatusTooManyRequests,
		Headers:    headers,
	}

	waitTime := c.calculateRetryWait(1, lastResp)

	// Should be around 60s + jitter (up to 25% = 15s)
	assert.GreaterOrEqual(t, waitTime, 60*time.Second, "Wait time should be at least 60s")
	assert.LessOrEqual(t, waitTime, 75*time.Second, "Wait time should be at most 75s (60s + 25% jitter)")
}

func TestCalculateRetryWait_Without429(t *testing.T) {
	c := newTestClient("http://localhost", "test")

	// Test with non-429 response (should use exponential backoff)
	headers := make(http.Header)
	headers.Set("Retry-After", "60") // This should be ignored

	lastResp := &Response{
		StatusCode: http.StatusInternalServerError, // 500, not 429
		Headers:    headers,
	}

	waitTime := c.calculateRetryWait(1, lastResp)

	// Should use exponential backoff, not Retry-After
	// First retry: base = RetryWaitMin (100ms) * 2^0 = 100ms, plus up to 25% jitter
	assert.Less(t, waitTime, 60*time.Second, "Wait time should use backoff, not Retry-After")
}

func TestCalculateRetryWait_NilResponse(t *testing.T) {
	c := newTestClient("http://localhost", "test")

	// Test with nil response (network error case)
	waitTime := c.calculateRetryWait(1, nil)

	// Should use exponential backoff
	// First retry: base = RetryWaitMin (100ms) * 2^0 = 100ms, plus up to 25% jitter
	assert.GreaterOrEqual(t, waitTime, 100*time.Millisecond)
	assert.LessOrEqual(t, waitTime, 125*time.Millisecond) // 100ms + 25% jitter
}

func TestCalculateRetryWait_ExponentialBackoff(t *testing.T) {
	c := newTestClient("http://localhost", "test")

	// Test exponential backoff across multiple attempts
	var prevWait time.Duration
	for attempt := 1; attempt <= 3; attempt++ {
		waitTime := c.calculateRetryWait(attempt, nil)

		if attempt > 1 {
			// Each attempt should generally have longer base wait (though jitter adds variance)
			// Just verify it's within reasonable bounds
			assert.GreaterOrEqual(t, waitTime, c.config.RetryWaitMin)
			assert.LessOrEqual(t, waitTime, c.config.RetryWaitMax+c.config.RetryWaitMax/4)
		}

		prevWait = waitTime
		_ = prevWait // silence unused warning
	}
}

func TestRetryWith429AndRetryAfter(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request: return 429 with Retry-After
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		// Subsequent requests: success
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	// Override RetryWaitMax to allow testing the Retry-After value
	c.config.RetryWaitMax = 5 * time.Second

	req := &Request{
		Method: "GET",
		Path:   "/v1/test",
	}

	ctx := context.Background()
	start := time.Now()
	resp, err := c.Do(ctx, req)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, requestCount, "Should have made 2 requests (1 failed + 1 success)")

	// Should have waited at least 1 second (the Retry-After value)
	assert.GreaterOrEqual(t, elapsed, 1*time.Second, "Should respect Retry-After header")
}
