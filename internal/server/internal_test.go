// White-box tests for unexported server internals: the bounded-startup-I/O
// helpers and the health-server bind-then-serve wiring in Start(). These
// live in package server (rather than server_test) so they can exercise
// unexported functions/fields directly and, critically, so they can
// construct a *Server without going through NewWithDeps — which calls
// metrics.New() and registers Prometheus collectors on the global
// DefaultRegisterer. Only one Server may be constructed via NewWithDeps per
// test binary (see server_test.go), so these tests avoid that path
// entirely.
package server

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/health"
)

// slowAuthenticator implements Authenticator with a configurable delay
// before GetUserIdentity returns, used to simulate a hanging IAM call.
type slowAuthenticator struct {
	delay       time.Duration
	userID      string
	identityErr error
}

func (s *slowAuthenticator) ValidateToken() error { return nil }

func (s *slowAuthenticator) GetUserIdentity() (string, error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.identityErr != nil {
		return "", s.identityErr
	}
	return s.userID, nil
}

// --- boundedGetUserIdentity ---

func TestBoundedGetUserIdentity_ReturnsPromptlyAtContextDeadline(t *testing.T) {
	auth := &slowAuthenticator{delay: 2 * time.Second, userID: "user-1"}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := boundedGetUserIdentity(ctx, auth)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the authenticator hangs past the context deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
	// The underlying call sleeps for 2s; boundedGetUserIdentity must return
	// at the ~50ms deadline, not wait for the full 2s.
	if elapsed > 500*time.Millisecond {
		t.Errorf("boundedGetUserIdentity took %v, should have returned at the context deadline (~50ms)", elapsed)
	}
}

func TestBoundedGetUserIdentity_ReturnsResultWhenFast(t *testing.T) {
	auth := &slowAuthenticator{userID: "user-2"}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	userID, err := boundedGetUserIdentity(ctx, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-2" {
		t.Errorf("userID = %q, want %q", userID, "user-2")
	}
}

func TestBoundedGetUserIdentity_PropagatesAuthError(t *testing.T) {
	wantErr := errors.New("boom")
	auth := &slowAuthenticator{identityErr: wantErr}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := boundedGetUserIdentity(ctx, auth)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected the authenticator's own error to propagate, got %v", err)
	}
}

// --- startupTimeout / shutdownTimeout fallback helpers ---

func TestStartupTimeout_UsesConfiguredValue(t *testing.T) {
	cfg := &config.Config{Timeout: 7 * time.Second}
	if got := startupTimeout(cfg); got != 7*time.Second {
		t.Errorf("startupTimeout() = %v, want 7s", got)
	}
}

func TestStartupTimeout_FallsBackWhenUnset(t *testing.T) {
	cfg := &config.Config{}
	if got := startupTimeout(cfg); got != defaultStartupTimeout {
		t.Errorf("startupTimeout() = %v, want fallback %v", got, defaultStartupTimeout)
	}
}

func TestShutdownTimeout_UsesConfiguredValue(t *testing.T) {
	cfg := &config.Config{ShutdownTimeout: 45 * time.Second}
	if got := shutdownTimeout(cfg); got != 45*time.Second {
		t.Errorf("shutdownTimeout() = %v, want 45s", got)
	}
}

func TestShutdownTimeout_FallsBackWhenUnset(t *testing.T) {
	cfg := &config.Config{}
	if got := shutdownTimeout(cfg); got != defaultShutdownTimeout {
		t.Errorf("shutdownTimeout() = %v, want fallback %v", got, defaultShutdownTimeout)
	}
}

// --- Start(): health server bind-then-serve ordering ---

// stubValidator is a minimal health.TokenValidator for constructing a
// health.Checker without pulling in the auth package.
type stubValidator struct{}

func (stubValidator) ValidateToken() error { return nil }

// TestStart_HealthServerBindFailure verifies that Start() binds the health
// server synchronously and returns a clear error (instead of swallowing it
// in a background goroutine) when the configured port is unavailable — and
// that it does so before ever marking the server ready. Constructing the
// *Server directly (rather than via NewWithDeps) avoids the Prometheus
// global-registry duplicate-registration panic that limits this test binary
// to a single NewWithDeps call (see server_test.go).
func TestStart_HealthServerBindFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy a port: %v", err)
	}
	defer func() { _ = occupied.Close() }()
	port := occupied.Addr().(*net.TCPAddr).Port

	logger := zap.NewNop()
	checker := health.New(client.NewMockClient(), stubValidator{}, logger)
	hs := health.NewServer(checker, logger, port, "127.0.0.1", false)

	s := &Server{
		logger:       logger,
		config:       &config.Config{ShutdownTimeout: 5 * time.Second},
		healthServer: hs,
	}

	err = s.Start(context.Background())
	if err == nil {
		t.Fatal("Start() should fail when the health server port is already in use")
	}
	if hs.IsReady() {
		t.Error("health server should not be marked ready when the bind failed")
	}
}
