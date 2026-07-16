package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDo_POSTWithoutIdempotencyNotRetriedOn502 verifies that a POST request
// with neither Idempotent=true nor a RequestID is NOT retried on a
// retryable status (502) — retrying a non-idempotent write could duplicate
// side effects on the server.
func TestDo_POSTWithoutIdempotencyNotRetriedOn502(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"bad gateway"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.MaxRetries = 3
	c.config.RetryWaitMin = 1
	c.config.RetryWaitMax = 5

	req := &Request{
		Method: "POST",
		Path:   "/v1/query",
		Body:   map[string]string{"query": "source logs"},
		// No Idempotent flag, no RequestID.
	}

	_, err := c.Do(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "non-idempotent POST should not be retried")
}

// TestDo_POSTWithRequestIDIsRetriedAndKeyStable verifies that a POST request
// with a RequestID set IS retried on a retryable status, and that the
// Idempotency-Key header is constant across every attempt.
func TestDo_POSTWithRequestIDIsRetriedAndKeyStable(t *testing.T) {
	var requestCount int32
	var seenKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("Idempotency-Key"))
		n := atomic.AddInt32(&requestCount, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"bad gateway"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.MaxRetries = 3
	c.config.RetryWaitMin = 1
	c.config.RetryWaitMax = 5

	req := &Request{
		Method:    "POST",
		Path:      "/v1/alerts",
		Body:      map[string]string{"name": "new-alert"},
		RequestID: "fixed-request-id-123",
	}

	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount), "idempotent POST should be retried until success")

	require.Len(t, seenKeys, 3)
	for _, k := range seenKeys {
		assert.Equal(t, "fixed-request-id-123", k, "Idempotency-Key must stay constant across retry attempts")
	}
}

// TestDo_POSTWithIdempotentFlagIsRetried verifies that Idempotent=true alone
// (without a RequestID) is sufficient to permit retries — this is the path
// used for read-only POSTs such as Query.
func TestDo_POSTWithIdempotentFlagIsRetried(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.MaxRetries = 3
	c.config.RetryWaitMin = 1
	c.config.RetryWaitMax = 5

	req := &Request{
		Method:     "POST",
		Path:       "/v1/query",
		Body:       map[string]string{"query": "source logs"},
		Idempotent: true,
	}

	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount), "Idempotent=true POST should be retried")
}

// TestDo_GETIsRetried verifies GET requests keep their existing
// retry-on-transient-status behavior with no idempotency flag required.
func TestDo_GETIsRetried(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, "test")
	c.config.MaxRetries = 3
	c.config.RetryWaitMin = 1
	c.config.RetryWaitMax = 5

	req := &Request{Method: "GET", Path: "/v1/alerts"}

	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount), "GET should be retried without any idempotency flag")
}
