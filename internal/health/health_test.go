package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// mockTokenValidator implements TokenValidator for testing.
type mockTokenValidator struct {
	err error
}

func (m *mockTokenValidator) ValidateToken() error {
	return m.err
}

// Tests for the HTTP health server handlers.
// The ready/live endpoints are fully testable without external dependencies.
// The health endpoint requires a valid Checker (with auth + client), so
// handler-level tests use a helper that directly constructs Server state.

func TestNewServer_DefaultBindAddr(t *testing.T) {
	logger := zap.NewNop()

	srv := NewServer(nil, logger, 8080, "", false)

	if srv.httpServer.Addr != "127.0.0.1:8080" {
		t.Errorf("Default bind address should be 127.0.0.1:8080, got %s", srv.httpServer.Addr)
	}
}

func TestNewServer_CustomBindAddr(t *testing.T) {
	logger := zap.NewNop()

	srv := NewServer(nil, logger, 9090, "0.0.0.0", false)

	if srv.httpServer.Addr != "0.0.0.0:9090" {
		t.Errorf("Custom bind address should be 0.0.0.0:9090, got %s", srv.httpServer.Addr)
	}
}

func TestNewServer_Timeouts(t *testing.T) {
	logger := zap.NewNop()

	srv := NewServer(nil, logger, 8080, "", false)

	if srv.httpServer.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", srv.httpServer.ReadTimeout)
	}
	if srv.httpServer.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", srv.httpServer.WriteTimeout)
	}
	if srv.httpServer.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", srv.httpServer.IdleTimeout)
	}
	if srv.httpServer.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 2s", srv.httpServer.ReadHeaderTimeout)
	}
}

// Ready endpoint tests - no external dependencies needed

func TestReadyHandler_NotReadyByDefault(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Not-ready should return 503, got %d", w.Code)
	}
	assertJSONContains(t, w.Body.Bytes(), "not_ready")
}

func TestReadyHandler_ReadyAfterSetReady(t *testing.T) {
	srv := newTestServer()
	srv.SetReady(true)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ready should return 200, got %d", w.Code)
	}
	assertJSONContains(t, w.Body.Bytes(), "ready")
}

func TestReadyHandler_NotReadyAfterSetReadyFalse(t *testing.T) {
	srv := newTestServer()
	srv.SetReady(true)
	srv.SetReady(false)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Not-ready should return 503, got %d", w.Code)
	}
}

func TestReadyHandler_RejectsNonGET(t *testing.T) {
	srv := newTestServer()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/ready", nil)
		w := httptest.NewRecorder()

		srv.readyHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /ready should return 405, got %d", method, w.Code)
		}
	}
}

// Live endpoint tests - no external dependencies needed

func TestLiveHandler_AlwaysAlive(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	srv.liveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Live endpoint should always return 200, got %d", w.Code)
	}
	assertJSONContains(t, w.Body.Bytes(), "alive")
}

func TestLiveHandler_AliveEvenWhenNotReady(t *testing.T) {
	srv := newTestServer()
	// Server is not ready, but liveness should still be OK

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	srv.liveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Live should return 200 even when not ready, got %d", w.Code)
	}
}

func TestLiveHandler_RejectsNonGET(t *testing.T) {
	srv := newTestServer()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/live", nil)
		w := httptest.NewRecorder()

		srv.liveHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /live should return 405, got %d", method, w.Code)
		}
	}
}

// Content-Type tests

func TestHandlers_ReturnJSON(t *testing.T) {
	srv := newTestServer()
	srv.SetReady(true)

	handlers := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"ready", srv.readyHandler},
		{"live", srv.liveHandler},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+h.name, nil)
			w := httptest.NewRecorder()

			h.handler(w, req)

			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("%s Content-Type = %q, want %q", h.name, ct, "application/json")
			}

			if !json.Valid(w.Body.Bytes()) {
				t.Errorf("%s returned invalid JSON: %s", h.name, w.Body.String())
			}
		})
	}
}

// Health check status types

func TestStatusConstants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("StatusHealthy = %q, want %q", StatusHealthy, "healthy")
	}
	if StatusDegraded != "degraded" {
		t.Errorf("StatusDegraded = %q, want %q", StatusDegraded, "degraded")
	}
	if StatusUnhealthy != "unhealthy" {
		t.Errorf("StatusUnhealthy = %q, want %q", StatusUnhealthy, "unhealthy")
	}
}

// SetReady atomicity test

func TestSetReady_ConcurrentAccess(t *testing.T) {
	srv := newTestServer()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			srv.SetReady(i%2 == 0)
		}
		close(done)
	}()

	// Concurrent reads while writing
	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()
		srv.readyHandler(w, req)
		// Just verify we don't panic - the status can be either
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Unexpected status %d during concurrent access", w.Code)
		}
	}
	<-done
}

// Health endpoint handler tests (tests HTTP behavior without real checker)

func TestHealthHandler_RejectsNonGET(t *testing.T) {
	srv := newTestServer()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/health", nil)
		w := httptest.NewRecorder()

		srv.healthHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /health should return 405, got %d", method, w.Code)
		}
	}
}

// Helpers

func newTestServer() *Server {
	logger := zap.NewNop()
	return NewServer(nil, logger, 0, "127.0.0.1", false)
}

func assertJSONContains(t *testing.T, body []byte, substr string) {
	t.Helper()
	if !strings.Contains(string(body), substr) {
		t.Errorf("Response body %q should contain %q", string(body), substr)
	}
}

// --- Checker tests using MockClient and mockTokenValidator ---

func TestCheckAll_AllHealthy(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	status, checks := checker.CheckAll(context.Background())

	if status != StatusHealthy {
		t.Errorf("overall status = %q, want %q", status, StatusHealthy)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}

	// Authentication check
	authCheck := checks[0]
	if authCheck.Name != "authentication" {
		t.Errorf("first check name = %q, want %q", authCheck.Name, "authentication")
	}
	if authCheck.Status != StatusHealthy {
		t.Errorf("auth check status = %q, want %q", authCheck.Status, StatusHealthy)
	}
	if !strings.Contains(authCheck.Message, "successful") {
		t.Errorf("auth check message = %q, want to contain 'successful'", authCheck.Message)
	}

	// API connectivity check
	apiCheck := checks[1]
	if apiCheck.Name != "api_connectivity" {
		t.Errorf("second check name = %q, want %q", apiCheck.Name, "api_connectivity")
	}
	if apiCheck.Status != StatusHealthy {
		t.Errorf("api check status = %q, want %q", apiCheck.Status, StatusHealthy)
	}
}

func TestCheckAll_AuthFailed(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}
	validator := &mockTokenValidator{err: errors.New("token expired")}
	checker := New(mock, validator, zap.NewNop())

	status, checks := checker.CheckAll(context.Background())

	if status != StatusUnhealthy {
		t.Errorf("overall status = %q, want %q", status, StatusUnhealthy)
	}

	authCheck := checks[0]
	if authCheck.Status != StatusUnhealthy {
		t.Errorf("auth check status = %q, want %q", authCheck.Status, StatusUnhealthy)
	}
	if !strings.Contains(authCheck.Message, "Authentication failed") {
		t.Errorf("auth check message = %q, want to contain 'Authentication failed'", authCheck.Message)
	}
}

func TestCheckAll_APIUnreachable(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultError = errors.New("connection refused")
	mock.DefaultResponse = nil
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	status, checks := checker.CheckAll(context.Background())

	if status != StatusUnhealthy {
		t.Errorf("overall status = %q, want %q", status, StatusUnhealthy)
	}

	apiCheck := checks[1]
	if apiCheck.Status != StatusUnhealthy {
		t.Errorf("api check status = %q, want %q", apiCheck.Status, StatusUnhealthy)
	}
	if !strings.Contains(apiCheck.Message, "API unreachable") {
		t.Errorf("api check message = %q, want to contain 'API unreachable'", apiCheck.Message)
	}
}

func TestCheckAll_AuthOK_APIFailed_OverallUnhealthy(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultError = errors.New("timeout")
	mock.DefaultResponse = nil
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	status, checks := checker.CheckAll(context.Background())

	// Auth should be healthy
	if checks[0].Status != StatusHealthy {
		t.Errorf("auth check should be healthy, got %q", checks[0].Status)
	}
	// API should be unhealthy
	if checks[1].Status != StatusUnhealthy {
		t.Errorf("api check should be unhealthy, got %q", checks[1].Status)
	}
	// Overall should be unhealthy (worst wins)
	if status != StatusUnhealthy {
		t.Errorf("overall = %q, want %q", status, StatusUnhealthy)
	}
}

func TestCheckAll_BothFailed(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultError = errors.New("no route")
	mock.DefaultResponse = nil
	validator := &mockTokenValidator{err: errors.New("invalid key")}
	checker := New(mock, validator, zap.NewNop())

	status, _ := checker.CheckAll(context.Background())

	if status != StatusUnhealthy {
		t.Errorf("both failed: overall = %q, want %q", status, StatusUnhealthy)
	}
}

func TestCheckAll_VerifiesAPIRequest(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	_, _ = checker.CheckAll(context.Background())

	// Verify the API connectivity check sends the expected request
	req := mock.LastRequest()
	if req == nil {
		t.Fatal("expected a request to be made")
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/v1/alerts" {
		t.Errorf("Path = %q, want /v1/alerts", req.Path)
	}
	if req.Query["limit"] != "1" {
		t.Errorf("Query limit = %q, want %q", req.Query["limit"], "1")
	}
}

func TestCheckAll_CancelledContext(t *testing.T) {
	mock := client.NewMockClient()
	mock.DoFunc = func(ctx context.Context, _ *client.Request) (*client.Response, error) {
		return nil, ctx.Err()
	}
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status, checks := checker.CheckAll(ctx)

	// Auth should still work (doesn't use context)
	if checks[0].Status != StatusHealthy {
		t.Errorf("auth check should be healthy even with cancelled context")
	}
	// API check will fail due to cancelled context
	if checks[1].Status == StatusHealthy {
		t.Error("api check should not be healthy with cancelled context")
	}
	if status != StatusUnhealthy && status != StatusDegraded {
		t.Errorf("overall should be unhealthy or degraded, got %q", status)
	}
}

func TestHealthHandler_WithChecker_Healthy(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	srv := NewServer(checker, zap.NewNop(), 0, "127.0.0.1", false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthy server should return 200, got %d", w.Code)
	}
	if !json.Valid(w.Body.Bytes()) {
		t.Errorf("response is not valid JSON: %s", w.Body.String())
	}
	assertJSONContains(t, w.Body.Bytes(), "healthy")
}

func TestHealthHandler_WithChecker_Unhealthy(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultError = errors.New("fail")
	mock.DefaultResponse = nil
	validator := &mockTokenValidator{err: errors.New("bad token")}
	checker := New(mock, validator, zap.NewNop())

	srv := NewServer(checker, zap.NewNop(), 0, "127.0.0.1", false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("unhealthy server should return 503, got %d", w.Code)
	}
	assertJSONContains(t, w.Body.Bytes(), "unhealthy")
}
