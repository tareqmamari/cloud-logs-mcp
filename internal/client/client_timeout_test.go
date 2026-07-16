package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestNew_DoesNotSetBlanketClientTimeout verifies that the real constructor
// (New) does not impose a client-wide http.Client.Timeout, since that would
// silently cap every per-request context.WithTimeout(ctx, req.Timeout)
// regardless of what the caller asked for.
func TestNew_DoesNotSetBlanketClientTimeout(t *testing.T) {
	cfg := newTestConfig("http://example.invalid")
	logger := zap.NewNop()

	c, err := New(cfg, logger, "test")
	require.NoError(t, err)
	defer func() { _ = c.Close() }()

	assert.Equal(t, time.Duration(0), c.httpClient.Timeout, "http.Client.Timeout should be 0; deadlines must be enforced exclusively via per-request contexts")
}

// TestPerRequestTimeoutNotTruncatedByClientTimeout proves that a per-request
// timeout longer than the client-wide default (cfg.Timeout) is honored, not
// silently capped by http.Client.Timeout. Before the fix, New() set
// http.Client.Timeout = cfg.Timeout, which truncated every request to the
// client-wide default regardless of what the caller set on Request.Timeout
// (e.g. QueryTimeout=60s, BulkOperationTimeout=120s never actually applied).
func TestPerRequestTimeoutNotTruncatedByClientTimeout(t *testing.T) {
	// Handler responds after a delay that is longer than cfg.Timeout but
	// shorter than the per-request timeout.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.Timeout = 50 * time.Millisecond // client-wide default: too short

	req := &Request{
		Method:  "GET",
		Path:    "/v1/test",
		Timeout: 300 * time.Millisecond, // per-request timeout: long enough
	}

	resp, err := c.doRequest(context.Background(), req)
	require.NoError(t, err, "request should succeed: per-request timeout (300ms) should apply, not the shorter client default (50ms)")
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestZeroRequestTimeoutFallsBackToClientDefault verifies that when
// Request.Timeout is unset (0), the client-wide default (cfg.Timeout) is
// still enforced so no request is ever unbounded.
func TestZeroRequestTimeoutFallsBackToClientDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.Timeout = 50 * time.Millisecond

	req := &Request{
		Method: "GET",
		Path:   "/v1/test",
		// Timeout left unset (0): should fall back to cfg.Timeout (50ms).
	}

	_, err := c.doRequest(context.Background(), req)
	require.Error(t, err, "request should time out via the client-wide default when Request.Timeout is unset")
}
