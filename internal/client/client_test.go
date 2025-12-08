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
