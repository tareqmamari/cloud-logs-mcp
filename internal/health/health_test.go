package health

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	mcperrors "github.com/tareqmamari/cloud-logs-mcp/internal/errors"
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

// --- Bind-then-serve lifecycle ---

// TestServer_ListenThenServe verifies the new bind-then-serve split: Listen
// binds synchronously and returns a usable listener, and Serve then accepts
// connections on it.
func TestServer_ListenThenServe(t *testing.T) {
	logger := zap.NewNop()
	srv := NewServer(nil, logger, 0, "127.0.0.1", false)

	ln, err := srv.Listen()
	if err != nil {
		t.Fatalf("Listen() failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	waitForServer(t, "http://"+addr+"/live", 2*time.Second)

	resp, err := http.Get("http://" + addr + "/live") //nolint:gosec
	if err != nil {
		t.Fatalf("GET /live failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /live status = %d, want 200", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Serve returned unexpected error after shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}

// TestServer_ListenFailsSynchronouslyOnBadAddress verifies that a bind
// failure (e.g. the port is already in use) is reported synchronously from
// Listen, before any goroutine is started and before the caller can mark the
// server ready.
func TestServer_ListenFailsSynchronouslyOnBadAddress(t *testing.T) {
	logger := zap.NewNop()

	// Occupy a port first.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy a port: %v", err)
	}
	defer func() { _ = occupied.Close() }()
	port := occupied.Addr().(*net.TCPAddr).Port

	srv := NewServer(nil, logger, port, "127.0.0.1", false)

	if _, err := srv.Listen(); err == nil {
		t.Fatal("Listen() should fail synchronously when the port is already in use")
	}
}

// Helpers

func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:gosec,bodyclose
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %v", url, timeout)
}

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

// TestCheckAll_APIReturnsClassifiedHTTPError verifies that when the client
// returns a non-nil response alongside a classified HTTP error (e.g. a 401
// from the real client's status-code classification), the health check uses
// the response to report the actual status code instead of a generic
// "API unreachable" message that would misleadingly suggest the API itself
// is down when in fact it responded (just with an error status).
func TestCheckAll_APIReturnsClassifiedHTTPError(t *testing.T) {
	mock := client.NewMockClient()
	mock.DoFunc = func(_ context.Context, _ *client.Request) (*client.Response, error) {
		resp := &client.Response{StatusCode: 401, Body: []byte(`{"error":"invalid api key"}`)}
		return resp, mcperrors.FromHTTPStatus(401, "invalid api key")
	}
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
	if !strings.Contains(apiCheck.Message, "401") {
		t.Errorf("api check message = %q, want to contain the HTTP status code 401", apiCheck.Message)
	}
	if strings.Contains(apiCheck.Message, "API unreachable") {
		t.Errorf("api check message = %q, should not say 'API unreachable' when the API actually responded with a status code", apiCheck.Message)
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

// TestCheckAll_ErrorMessagesDoNotLeakDetails verifies that health check
// Messages (which are served unauthenticated via /health) never embed raw
// upstream error text — which can include API response bodies or other
// sensitive detail. Detailed errors should only reach the server-side log.
func TestCheckAll_ErrorMessagesDoNotLeakDetails(t *testing.T) {
	const secretMarker = "sk-supersecret-should-not-appear-1234567890" // pragma: allowlist secret

	mock := client.NewMockClient()
	mock.DoFunc = func(_ context.Context, _ *client.Request) (*client.Response, error) {
		body := `{"error":"invalid key ` + secretMarker + `"}`
		return &client.Response{StatusCode: 401, Body: []byte(body)}, mcperrors.FromHTTPStatus(401, body)
	}
	validator := &mockTokenValidator{err: errors.New("token rejected, raw token: " + secretMarker)}
	checker := New(mock, validator, zap.NewNop())

	_, checks := checker.CheckAll(context.Background())

	for _, c := range checks {
		if strings.Contains(c.Message, secretMarker) {
			t.Errorf("check %q message leaked sensitive upstream detail: %q", c.Name, c.Message)
		}
	}

	authCheck := checks[0]
	if authCheck.Message != "Authentication failed" {
		t.Errorf("auth check message = %q, want generic %q", authCheck.Message, "Authentication failed")
	}
}

// TestCheckAll_APIUnreachable_GenericMessage verifies the default
// (non-HTTP-status) API-connectivity failure message is a fixed generic
// string, not formatted with the underlying error.
func TestCheckAll_APIUnreachable_GenericMessage(t *testing.T) {
	mock := client.NewMockClient()
	mock.DefaultError = errors.New("dial tcp 10.0.0.1:443: sensitive-network-detail")
	mock.DefaultResponse = nil
	validator := &mockTokenValidator{err: nil}
	checker := New(mock, validator, zap.NewNop())

	_, checks := checker.CheckAll(context.Background())

	apiCheck := checks[1]
	if apiCheck.Message != "API unreachable" {
		t.Errorf("api check message = %q, want exactly %q", apiCheck.Message, "API unreachable")
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
