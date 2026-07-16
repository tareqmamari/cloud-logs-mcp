// Package health provides health checking functionality for the MCP server.
package health

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// TokenValidator validates authentication tokens.
// *auth.Authenticator satisfies this interface.
type TokenValidator interface {
	ValidateToken() error
}

// Status represents the health status
type Status string

const (
	// StatusHealthy indicates the service is healthy
	StatusHealthy Status = "healthy"
	// StatusDegraded indicates the service is degraded
	StatusDegraded Status = "degraded"
	// StatusUnhealthy indicates the service is unhealthy
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check result
type Check struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// Checker performs health checks
type Checker struct {
	client        client.Doer
	authenticator TokenValidator
	logger        *zap.Logger
}

// New creates a new health checker
func New(c client.Doer, authenticator TokenValidator, logger *zap.Logger) *Checker {
	return &Checker{
		client:        c,
		authenticator: authenticator,
		logger:        logger,
	}
}

// CheckAll performs all health checks
func (c *Checker) CheckAll(ctx context.Context) (Status, []Check) {
	checks := []Check{
		c.checkAuthentication(),
		c.checkAPIConnectivity(ctx),
	}

	// Determine overall status
	overallStatus := StatusHealthy
	for _, check := range checks {
		if check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		} else if check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return overallStatus, checks
}

// checkAuthentication verifies authentication is working
func (c *Checker) checkAuthentication() Check {
	start := time.Now()
	check := Check{
		Name:      "authentication",
		Timestamp: start,
	}

	err := c.authenticator.ValidateToken()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusUnhealthy
		// Keep the public message generic: the underlying error can embed
		// upstream response bodies (e.g. IAM error detail) that must not be
		// exposed via the unauthenticated /health endpoint. The detailed
		// error is logged server-side below instead.
		check.Message = "Authentication failed"
		c.logger.Error("Health check failed: authentication",
			zap.Error(err),
			zap.Duration("duration", check.Duration),
		)
	} else {
		check.Status = StatusHealthy
		check.Message = "Authentication successful"
		c.logger.Debug("Health check passed: authentication",
			zap.Duration("duration", check.Duration),
		)
	}

	return check
}

// checkAPIConnectivity verifies API connectivity
func (c *Checker) checkAPIConnectivity(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:      "api_connectivity",
		Timestamp: start,
	}

	// Try a simple API call (list alerts with limit 1)
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/alerts",
		Query:  map[string]string{"limit": "1"},
	}

	// Use a short timeout for health checks
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.Do(checkCtx, req)
	check.Duration = time.Since(start)

	switch {
	case err == nil:
		check.Status = StatusHealthy
		check.Message = "API reachable"
		c.logger.Debug("Health check passed: API connectivity",
			zap.Duration("duration", check.Duration),
		)
	case resp != nil:
		// The client returned a response alongside the error: the API is
		// reachable, it just responded with an error status (e.g. 401/404).
		// Report the actual status instead of a misleading "unreachable".
		// The status code alone is safe to expose; the error's formatted
		// text can embed the upstream response body, so it is logged
		// server-side only, not included in the public message.
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("API returned HTTP %d", resp.StatusCode)
		c.logger.Warn("Health check failed: API returned error status",
			zap.Int("status", resp.StatusCode),
			zap.Error(err),
			zap.Duration("duration", check.Duration),
		)
	case check.Duration > 3*time.Second:
		// Degraded if we can't reach the API in time, but auth works
		check.Status = StatusDegraded
		check.Message = "API responding slowly"
		c.logger.Warn("Health check failed: API connectivity",
			zap.Error(err),
			zap.Duration("duration", check.Duration),
		)
	default:
		// Keep the public message generic; log the detailed error
		// server-side instead of embedding it in the unauthenticated
		// /health response.
		check.Status = StatusUnhealthy
		check.Message = "API unreachable"
		c.logger.Warn("Health check failed: API connectivity",
			zap.Error(err),
			zap.Duration("duration", check.Duration),
		)
	}

	return check
}
