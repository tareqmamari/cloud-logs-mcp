// Package client provides HTTP client functionality for IBM Cloud Logs API.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/tareqmamari/logs-mcp-server/internal/auth"
	"github.com/tareqmamari/logs-mcp-server/internal/config"
)

// Authenticator is the interface for adding authentication to requests
type Authenticator interface {
	Authenticate(req *http.Request) error
}

// Client is an HTTP client for the IBM Cloud Logs API
type Client struct {
	httpClient    *http.Client
	config        *config.Config
	logger        *zap.Logger
	rateLimiter   *rate.Limiter
	authenticator Authenticator
	version       string
}

// New creates a new API client
func New(cfg *config.Config, logger *zap.Logger, version string) (*Client, error) {
	// Create IBM Cloud authenticator
	authenticator, err := auth.New(cfg.APIKey, cfg.IAMURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	// Configure TLS with secure defaults
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	// Only disable TLS verification if explicitly configured (for testing environments)
	// By default, cfg.TLSVerify is true, so verification is enabled
	if !cfg.TLSVerify {
		tlsConfig.InsecureSkipVerify = true
		logger.Warn("TLS certificate verification is DISABLED - this is insecure and should only be used for testing",
			zap.String("service_url", cfg.ServiceURL),
		)
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	var rateLimiter *rate.Limiter
	if cfg.EnableRateLimit {
		rateLimiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimitBurst)
	}

	// Use provided version or default to "dev"
	if version == "" {
		version = "dev"
	}

	return &Client{
		httpClient:    httpClient,
		config:        cfg,
		logger:        logger,
		rateLimiter:   rateLimiter,
		authenticator: authenticator,
		version:       version,
	}, nil
}

// Request represents an HTTP request
type Request struct {
	Method         string
	Path           string
	Query          map[string]string
	Body           interface{}
	Headers        map[string]string
	RequestID      string // Optional client-provided request ID for idempotency
	UseIngressHost bool   // Use ingress endpoint instead of API endpoint for log ingestion
	AcceptSSE      bool   // Use text/event-stream Accept header for streaming responses (e.g., sync queries)
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request with retry logic
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with overflow protection
			// Cap shift value to prevent overflow (max 30 ensures we stay within reasonable time bounds)
			shift := min(attempt-1, 30)
			waitTime := c.config.RetryWaitMin * time.Duration(1<<shift)
			if waitTime > c.config.RetryWaitMax {
				waitTime = c.config.RetryWaitMax
			}

			c.logger.Debug("Retrying request",
				zap.Int("attempt", attempt),
				zap.Duration("wait", waitTime),
			)

			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			lastErr = err
			// Retry on network errors
			if isRetryable(err) {
				continue
			}
			return nil, err
		}

		// Retry on specific HTTP status codes
		if shouldRetry(resp.StatusCode) {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(resp.Body))
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, req *Request) (*Response, error) {
	// Apply rate limiting
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	// Build URL - use ingress endpoint for log ingestion, API endpoint for everything else
	baseURL := c.config.ServiceURL
	if req.UseIngressHost {
		// Convert API URL to ingress URL
		// From: https://{instance-id}.api.{region}.logs.cloud.ibm.com
		// To:   https://{instance-id}.ingress.{region}.logs.cloud.ibm.com
		baseURL = convertToIngressURL(baseURL)
	}

	// Build URL with proper encoding
	requestURL := fmt.Sprintf("%s%s", baseURL, req.Path)
	if len(req.Query) > 0 {
		params := url.Values{}
		for k, v := range req.Query {
			params.Add(k, v)
		}
		requestURL = fmt.Sprintf("%s?%s", requestURL, params.Encode())
	}

	// Prepare body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if req.AcceptSSE {
		httpReq.Header.Set("Accept", "text/event-stream")
	} else {
		httpReq.Header.Set("Accept", "application/json")
	}
	httpReq.Header.Set("User-Agent", fmt.Sprintf("logs-mcp-server/%s", c.version))

	// Add idempotency key if provided
	if req.RequestID != "" {
		httpReq.Header.Set("X-Request-ID", req.RequestID)
		// Some APIs use Idempotency-Key header for POST/PUT operations
		if req.Method == "POST" || req.Method == "PUT" {
			httpReq.Header.Set("Idempotency-Key", req.RequestID)
		}
	}

	// Add IBM Cloud authentication (bearer token)
	if err := c.authenticator.Authenticate(httpReq); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	c.logger.Debug("Executing HTTP request",
		zap.String("method", req.Method),
		zap.String("url", requestURL),
	)

	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Error("HTTP request failed",
			zap.Error(err),
			zap.String("method", req.Method),
			zap.String("url", requestURL),
			zap.Duration("duration", duration),
		)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			c.logger.Warn("Failed to close response body", zap.Error(closeErr))
		}
	}()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("HTTP request completed",
		zap.String("method", req.Method),
		zap.String("url", requestURL),
		zap.Int("status", httpResp.StatusCode),
		zap.Duration("duration", duration),
		zap.Int("response_size", len(body)),
	)

	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       body,
		Headers:    httpResp.Header,
	}, nil
}

// convertToIngressURL converts an API URL to an ingress URL for log ingestion
// From: https://{instance-id}.api.{region}.logs.cloud.ibm.com
// To:   https://{instance-id}.ingress.{region}.logs.cloud.ibm.com
func convertToIngressURL(apiURL string) string {
	return strings.Replace(apiURL, ".api.", ".ingress.", 1)
}

// isRetryable determines if an error is retryable (transient network errors)
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for network-related errors that are typically transient
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are retryable
		if netErr.Timeout() {
			return true
		}
	}

	// Check for specific syscall errors that indicate transient network issues
	var syscallErr *net.OpError
	if errors.As(err, &syscallErr) {
		// Connection refused, reset, or network unreachable are retryable
		if errors.Is(syscallErr.Err, syscall.ECONNREFUSED) ||
			errors.Is(syscallErr.Err, syscall.ECONNRESET) ||
			errors.Is(syscallErr.Err, syscall.ENETUNREACH) ||
			errors.Is(syscallErr.Err, syscall.EHOSTUNREACH) ||
			errors.Is(syscallErr.Err, syscall.ETIMEDOUT) {
			return true
		}
	}

	// Check for DNS errors - temporary DNS failures are retryable
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Temporary()
	}

	// Check error message for common transient patterns
	errStr := err.Error()
	transientPatterns := []string{
		"connection reset",
		"connection refused",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"TLS handshake timeout",
		"EOF",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	// Default: don't retry unknown errors to avoid retrying permanent failures
	return false
}

// shouldRetry determines if an HTTP status code should trigger a retry
func shouldRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
