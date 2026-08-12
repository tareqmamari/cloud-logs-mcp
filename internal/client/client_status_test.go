package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcperrors "github.com/tareqmamari/cloud-logs-mcp/internal/errors"
)

// TestDo_ClassifiesHTTPErrorStatuses verifies that Client.Do surfaces non-2xx
// responses as classified, errors.As-matchable errors instead of silently
// returning (resp, nil). This is the fix for the top-priority bug: 4xx/401/403/404/409
// responses used to be indistinguishable from success at the client boundary.
func TestDo_ClassifiesHTTPErrorStatuses(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantErr      bool
		wantCode     mcperrors.ErrorCode
	}{
		{
			name:         "200 success - no error",
			statusCode:   http.StatusOK,
			responseBody: `{"ok":true}`,
			wantErr:      false,
		},
		{
			name:         "400 bad request",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error":"invalid query"}`,
			wantErr:      true,
			wantCode:     mcperrors.CodeInvalidInput,
		},
		{
			name:         "401 unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error":"invalid api key"}`,
			wantErr:      true,
			wantCode:     mcperrors.CodeUnauthorized,
		},
		{
			name:         "403 forbidden",
			statusCode:   http.StatusForbidden,
			responseBody: `{"error":"forbidden"}`,
			wantErr:      true,
			wantCode:     mcperrors.CodeForbidden,
		},
		{
			name:         "404 not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error":"not found"}`,
			wantErr:      true,
			wantCode:     mcperrors.CodeResourceNotFound,
		},
		{
			name:         "409 conflict",
			statusCode:   http.StatusConflict,
			responseBody: `{"error":"already exists"}`,
			wantErr:      true,
			wantCode:     mcperrors.CodeConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			c := newTestClient(server.URL, "test")
			req := &Request{Method: "GET", Path: "/v1/test"}

			resp, err := c.Do(context.Background(), req)

			if !tt.wantErr {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.statusCode, resp.StatusCode)
				return
			}

			require.Error(t, err)

			// The response must still be surfaced alongside the error so
			// callers that legitimately need the status code (e.g. health
			// checks) can inspect it.
			require.NotNil(t, resp)
			assert.Equal(t, tt.statusCode, resp.StatusCode)

			var structuredErr *mcperrors.StructuredError
			require.True(t, errors.As(err, &structuredErr), "error should be errors.As-matchable to *mcperrors.StructuredError, got %T: %v", err, err)
			assert.Equal(t, tt.wantCode, structuredErr.Code)
			assert.Equal(t, tt.statusCode, structuredErr.StatusCode)
		})
	}
}

// TestDo_ClassifiedErrorIncludesBoundedBodySnippet verifies the classified
// error carries a bounded (~2KB) snippet of the response body, not the
// entire (potentially huge) body.
func TestDo_ClassifiedErrorIncludesBoundedBodySnippet(t *testing.T) {
	hugeBody := strings.Repeat("x", 10_000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(hugeBody))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	req := &Request{Method: "GET", Path: "/v1/test"}

	_, err := c.Do(context.Background(), req)
	require.Error(t, err)

	var structuredErr *mcperrors.StructuredError
	require.True(t, errors.As(err, &structuredErr))

	// The full 10KB body must not appear verbatim in the error; it should be
	// capped to roughly 2KB.
	assert.Less(t, len(structuredErr.Message), 3000, "error message should be bounded, not include the full 10KB body")
}

// TestDo_MaxRetriesExceededIncludesClassifiedStatus verifies that when
// retries are exhausted against a persistently-retryable status (e.g. 503),
// the final error still carries the last response's status code and body
// snippet via the same classification path used for immediate 4xx errors,
// and the last response is returned alongside the error.
func TestDo_MaxRetriesExceededIncludesClassifiedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"still down"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.MaxRetries = 1
	c.config.RetryWaitMin = 1
	c.config.RetryWaitMax = 5

	req := &Request{Method: "GET", Path: "/v1/test"}

	resp, err := c.Do(context.Background(), req)
	require.Error(t, err)
	require.NotNil(t, resp, "final response should be returned alongside the max-retries-exceeded error")
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var structuredErr *mcperrors.StructuredError
	require.True(t, errors.As(err, &structuredErr), "max retries exceeded error should still classify via the HTTP status mapping path")
	assert.Equal(t, http.StatusServiceUnavailable, structuredErr.StatusCode)
	assert.Contains(t, err.Error(), "max retries exceeded")
}
