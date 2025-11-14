package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/observability-c/logs-mcp-server/internal/auth"
	"github.com/observability-c/logs-mcp-server/internal/config"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Client is an HTTP client for the IBM Cloud Logs API
type Client struct {
	httpClient    *http.Client
	config        *config.Config
	logger        *zap.Logger
	rateLimiter   *rate.Limiter
	authenticator *auth.Authenticator
}

// New creates a new API client
func New(cfg *config.Config, logger *zap.Logger) (*Client, error) {
	// Create IBM Cloud authenticator
	authenticator, err := auth.New(cfg.APIKey, cfg.IAMURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !cfg.TLSVerify,
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	var rateLimiter *rate.Limiter
	if cfg.EnableRateLimit {
		rateLimiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimitBurst)
	}

	return &Client{
		httpClient:    httpClient,
		config:        cfg,
		logger:        logger,
		rateLimiter:   rateLimiter,
		authenticator: authenticator,
	}, nil
}

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Query   map[string]string
	Body    interface{}
	Headers map[string]string
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
			// Exponential backoff
			waitTime := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
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

	// Build URL
	url := fmt.Sprintf("%s%s", c.config.ServiceURL, req.Path)
	if len(req.Query) > 0 {
		url += "?"
		first := true
		for k, v := range req.Query {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
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
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "logs-mcp-server/0.1.0")

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
		zap.String("url", url),
	)

	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Error("HTTP request failed",
			zap.Error(err),
			zap.String("method", req.Method),
			zap.String("url", url),
			zap.Duration("duration", duration),
		)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("HTTP request completed",
		zap.String("method", req.Method),
		zap.String("url", url),
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

// isRetryable determines if an error is retryable
func isRetryable(err error) bool {
	// Network errors are retryable
	return true
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
