// Package health provides health checking and HTTP endpoints for the MCP server.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server provides HTTP endpoints for health checks and metrics.
// It exposes:
//   - /health - Full health check with component status
//   - /ready  - Readiness probe for K8s (checks if server can handle requests)
//   - /live   - Liveness probe for K8s (checks if server is running)
//   - /metrics - Prometheus metrics (if enabled)
type Server struct {
	checker        *Checker
	logger         *zap.Logger
	httpServer     *http.Server
	port           int
	metricsEnabled bool

	// ready indicates if the server is ready to handle requests
	ready atomic.Bool
}

// NewServer creates a new health HTTP server.
// bindAddr specifies the interface to bind to (default: 127.0.0.1 for security).
// Use "0.0.0.0" only when the health endpoint needs to be accessible externally
// (e.g., in Kubernetes environments).
func NewServer(checker *Checker, logger *zap.Logger, port int, bindAddr string, metricsEnabled bool) *Server {
	s := &Server{
		checker:        checker,
		logger:         logger,
		port:           port,
		metricsEnabled: metricsEnabled,
	}

	// Default to localhost for security if not specified
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/live", s.liveHandler)

	if metricsEnabled {
		mux.Handle("/metrics", promhttp.Handler())
	}

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", bindAddr, port),
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	return s
}

// SetReady marks the server as ready to handle requests.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Start starts the HTTP health server.
func (s *Server) Start() error {
	s.logger.Info("Starting health HTTP server",
		zap.Int("port", s.port),
		zap.Bool("metrics_enabled", s.metricsEnabled),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("health server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down health HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Response represents the response from /health endpoint.
type Response struct {
	Status    Status    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []Check   `json:"checks"`
}

// healthHandler handles the /health endpoint.
// Returns full health status with all component checks.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status, checks := s.checker.CheckAll(ctx)

	response := Response{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")

	switch status {
	case StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK) // Degraded is still OK for K8s
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// readyHandler handles the /ready endpoint (K8s readiness probe).
// Returns 200 if the server is ready to handle requests, 503 otherwise.
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"not_ready"}`))
		return
	}

	// Quick check - just verify we're ready, don't do full health check
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

// liveHandler handles the /live endpoint (K8s liveness probe).
// Returns 200 if the server process is running.
// This is a simple check - if we can respond, we're alive.
func (s *Server) liveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}`))
}
