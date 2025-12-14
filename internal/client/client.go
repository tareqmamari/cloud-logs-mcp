// Package client provides HTTP client functionality for IBM Cloud Logs API.
package client

import (
	"bytes"
	"context"
	"crypto/rand"
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
	"github.com/tareqmamari/logs-mcp-server/internal/tracing"
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
	enableTracing bool
}

// RateLimitInfo contains information about the current rate limit state
type RateLimitInfo struct {
	Limit     int     `json:"limit"`     // Requests per second limit
	Burst     int     `json:"burst"`     // Burst size
	Available float64 `json:"available"` // Currently available tokens
	Enabled   bool    `json:"enabled"`   // Whether rate limiting is enabled
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
		enableTracing: cfg.EnableTracing,
	}, nil
}

// GetRateLimitInfo returns information about the current rate limit state
func (c *Client) GetRateLimitInfo() RateLimitInfo {
	info := RateLimitInfo{
		Limit:   c.config.RateLimit,
		Burst:   c.config.RateLimitBurst,
		Enabled: c.config.EnableRateLimit,
	}

	if c.rateLimiter != nil {
		info.Available = float64(c.rateLimiter.Tokens())
	}

	return info
}

// cryptoRandInt63 returns a non-negative random int64 using crypto/rand
func cryptoRandInt63() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	// Clear the sign bit to ensure non-negative result
	b[7] &= 0x7F
	var n int64
	for i := 0; i < 8; i++ {
		n |= int64(b[i]) << (8 * i)
	}
	return n
}

// cryptoRandDuration returns a random duration between 0 and maxVal using crypto/rand
func cryptoRandDuration(maxVal int64) time.Duration {
	if maxVal <= 0 {
		return 0
	}
	return time.Duration(cryptoRandInt63() % maxVal)
}

// Request represents an HTTP request
type Request struct {
	Method         string
	Path           string
	Query          map[string]string
	Body           interface{}
	Headers        map[string]string
	RequestID      string        // Optional client-provided request ID for idempotency
	UseIngressHost bool          // Use ingress endpoint instead of API endpoint for log ingestion
	AcceptSSE      bool          // Use text/event-stream Accept header for streaming responses (e.g., sync queries)
	Timeout        time.Duration // Optional per-request timeout (overrides client default)
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
			// Exponential backoff with jitter to prevent thundering herd
			// Cap shift value to prevent overflow (max 30 ensures we stay within reasonable time bounds)
			shift := min(attempt-1, 30)
			baseWait := c.config.RetryWaitMin * time.Duration(1<<shift)
			if baseWait > c.config.RetryWaitMax {
				baseWait = c.config.RetryWaitMax
			}

			// Add jitter: random value between 0 and 25% of base wait time
			// This spreads out retry attempts when multiple clients fail simultaneously
			jitter := cryptoRandDuration(int64(baseWait) / 4)
			waitTime := baseWait + jitter

			c.logger.Debug("Retrying request",
				zap.Int("attempt", attempt),
				zap.Duration("wait", waitTime),
				zap.Duration("jitter", jitter),
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
	if err := c.applyRateLimit(ctx); err != nil {
		return nil, err
	}

	ctx, cancel := c.applyTimeout(ctx, req)
	if cancel != nil {
		defer cancel()
	}

	requestURL := c.buildRequestURL(req)

	bodyReader, err := c.prepareBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(ctx, httpReq, req)

	if err := c.authenticator.Authenticate(httpReq); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	return c.executeRequest(httpReq, req, requestURL)
}

func (c *Client) applyRateLimit(ctx context.Context) error {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit wait failed: %w", err)
		}
	}
	return nil
}

func (c *Client) applyTimeout(ctx context.Context, req *Request) (context.Context, context.CancelFunc) {
	if req.Timeout > 0 {
		return context.WithTimeout(ctx, req.Timeout)
	}
	return ctx, nil
}

func (c *Client) buildRequestURL(req *Request) string {
	baseURL := c.config.ServiceURL
	if req.UseIngressHost {
		baseURL = convertToIngressURL(baseURL)
	}

	requestURL := fmt.Sprintf("%s%s", baseURL, req.Path)
	if len(req.Query) > 0 {
		params := url.Values{}
		for k, v := range req.Query {
			params.Add(k, v)
		}
		requestURL = fmt.Sprintf("%s?%s", requestURL, params.Encode())
	}
	return requestURL
}

func (c *Client) prepareBody(req *Request) (io.Reader, error) {
	if req.Body == nil {
		return nil, nil
	}
	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewReader(bodyBytes), nil
}

func (c *Client) setHeaders(ctx context.Context, httpReq *http.Request, req *Request) {
	httpReq.Header.Set("Content-Type", "application/json")
	if req.AcceptSSE {
		httpReq.Header.Set("Accept", "text/event-stream")
	} else {
		httpReq.Header.Set("Accept", "application/json")
	}
	httpReq.Header.Set("User-Agent", fmt.Sprintf("logs-mcp-server/%s", c.version))
	httpReq.Header.Set("MCP-Protocol-Version", "2025-06-18")

	c.setTracingHeaders(ctx, httpReq)
	c.setIdempotencyHeaders(httpReq, req)
}

func (c *Client) setTracingHeaders(ctx context.Context, httpReq *http.Request) {
	if !c.enableTracing {
		return
	}
	traceInfo := tracing.FromContext(ctx)
	if traceInfo.TraceID == "" {
		traceInfo = tracing.NewTraceInfo()
	}
	for k, v := range traceInfo.Headers() {
		httpReq.Header.Set(k, v)
	}
}

func (c *Client) setIdempotencyHeaders(httpReq *http.Request, req *Request) {
	if req.RequestID == "" {
		return
	}
	httpReq.Header.Set("X-Request-ID", req.RequestID)
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Idempotency-Key", req.RequestID)
	}
}

func (c *Client) executeRequest(httpReq *http.Request, req *Request, requestURL string) (*Response, error) {
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
