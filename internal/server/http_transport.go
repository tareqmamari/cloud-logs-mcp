// Package server provides HTTP transport for multi-tenant MCP server deployment.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/auth"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	itools "github.com/tareqmamari/cloud-logs-mcp/internal/tools"
)

// HTTPTransportServer wraps the MCP server with HTTP transport for multi-tenant deployment.
type HTTPTransportServer struct {
	logger      *zap.Logger
	version     string
	port        int
	sseHandler  *mcp.SSEHandler
	sessionMu   sync.Mutex
	sseSessions map[string]*mcp.SSEServerTransport
}

// NewHTTPTransportServer creates a new HTTP transport server.
func NewHTTPTransportServer(logger *zap.Logger, version string, port int) *HTTPTransportServer {
	s := &HTTPTransportServer{
		logger:      logger,
		version:     version,
		port:        port,
		sseSessions: make(map[string]*mcp.SSEServerTransport),
	}

	s.sseHandler = mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s.buildServerForRequest(r)
	}, nil)

	return s
}

// Start starts the HTTP server for multi-tenant MCP requests.
func (s *HTTPTransportServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// MCP endpoints.
	mux.Handle("/mcp", s.sseHandler)
	mux.HandleFunc("/message", s.handleMessage)

	// Health check endpoint for Code Engine readiness/liveness.
	mux.HandleFunc("/health", s.handleHealth)

	// Info endpoint.
	mux.HandleFunc("/", s.handleInfo)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.corsMiddleware(s.loggingMiddleware(mux)),
	}

	s.logger.Info("Starting HTTP transport server",
		zap.Int("port", s.port),
		zap.String("version", s.version),
		zap.String("mcp_endpoint", "/mcp"),
	)

	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

func (s *HTTPTransportServer) buildServerForRequest(r *http.Request) *mcp.Server {
	cfg, err := s.extractConfigFromRequest(r)
	if err != nil {
		s.logger.Warn("Invalid multi-tenant request configuration", zap.Error(err))
		return mcp.NewServer(&mcp.Implementation{
			Name:    "IBM Cloud Logs MCP Server",
			Version: s.version,
		}, &mcp.ServerOptions{
			Instructions: "Missing or invalid X-Logs-Service-URL / X-Logs-API-Key headers",
			HasTools:     false,
			HasPrompts:   false,
			HasResources: false,
		})
	}

	apiClient, err := client.New(cfg, s.logger, s.version)
	if err != nil {
		s.logger.Error("Failed creating API client for HTTP request", zap.Error(err))
		return mcp.NewServer(&mcp.Implementation{
			Name:    "IBM Cloud Logs MCP Server",
			Version: s.version,
		}, &mcp.ServerOptions{
			Instructions: "Failed to initialize IBM Cloud Logs client for this request",
			HasTools:     false,
			HasPrompts:   false,
			HasResources: false,
		})
	}

	_, err = auth.New(cfg.APIKey, cfg.IAMURL, s.logger)
	if err != nil {
		s.logger.Error("Failed creating authenticator for HTTP request", zap.Error(err))
		return mcp.NewServer(&mcp.Implementation{
			Name:    "IBM Cloud Logs MCP Server",
			Version: s.version,
		}, &mcp.ServerOptions{
			Instructions: "Failed to initialize IBM Cloud IAM authenticator for this request",
			HasTools:     false,
			HasPrompts:   false,
			HasResources: false,
		})
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "IBM Cloud Logs MCP Server",
		Version: s.version,
	}, &mcp.ServerOptions{
		HasTools:     true,
		HasPrompts:   false,
		HasResources: false,
	})

	for _, t := range itools.GetAllTools(apiClient, s.logger) {
		tool := t
		mcpServer.AddTool(&mcp.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: tool.Annotations(),
		}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]interface{}
			if len(request.Params.Arguments) > 0 {
				if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{
							&mcp.TextContent{Text: fmt.Sprintf("failed to parse tool arguments: %v", err)},
						},
					}, nil
				}
			}
			return tool.Execute(ctx, args)
		})
	}

	return mcpServer
}

func (s *HTTPTransportServer) extractConfigFromRequest(r *http.Request) (*config.Config, error) {
	serviceURL := strings.TrimSpace(r.Header.Get("X-Logs-Service-URL"))
	apiKey := strings.TrimSpace(r.Header.Get("X-Logs-API-Key"))

	if serviceURL == "" {
		return nil, fmt.Errorf("missing X-Logs-Service-URL header")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("missing X-Logs-API-Key header")
	}

	cfg := &config.Config{
		ServiceURL:        serviceURL,
		APIKey:            apiKey,
		Region:            config.ExtractRegionFromURL(serviceURL),
		InstanceID:        config.ExtractInstanceIDFromURL(serviceURL),
		Timeout:           30 * time.Second,
		MaxRetries:        3,
		RetryWaitMin:      1 * time.Second,
		RetryWaitMax:      30 * time.Second,
		MaxIdleConns:      10,
		IdleConnTimeout:   90 * time.Second,
		QueryTimeout:      60 * time.Second,
		BulkOperationTimeout: 120 * time.Second,
		BackgroundPollTimeout: 10 * time.Second,
		RateLimit:         100,
		RateLimitBurst:    20,
		EnableRateLimit:   true,
		EnableTracing:     true,
		EnableAuditLog:    true,
		LogLevel:          "info",
		LogFormat:         "json",
	}

	if cfg.Region == "" || cfg.InstanceID == "" {
		return nil, fmt.Errorf("could not extract region/instance ID from X-Logs-Service-URL")
	}

	return cfg, nil
}

func (s *HTTPTransportServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "missing sessionId query parameter", http.StatusBadRequest)
		return
	}

	s.sessionMu.Lock()
	transport, ok := s.sseSessions[sessionID]
	s.sessionMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	transport.ServeHTTP(w, r)
}

// handleHealth handles health check requests.
func (s *HTTPTransportServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "healthy",
		"version":  s.version,
		"mode":     "multi-tenant-http",
		"mcp_path": "/mcp",
	})
}

// handleInfo provides server information.
func (s *HTTPTransportServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"name":      "IBM Cloud Logs MCP Server",
		"version":   s.version,
		"transport": "http",
		"mode":      "multi-tenant",
		"endpoints": map[string]string{
			"health":  "/health",
			"mcp":     "/mcp",
			"message": "/message?sessionId=<id>",
		},
		"required_headers": []string{
			"X-Logs-Service-URL",
			"X-Logs-API-Key",
		},
	})
}

// loggingMiddleware logs HTTP requests.
func (s *HTTPTransportServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers.
func (s *HTTPTransportServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Last-Event-ID, X-Logs-Service-URL, X-Logs-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
