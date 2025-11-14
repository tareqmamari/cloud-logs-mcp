package health

import (
	"context"
	"fmt"
	"time"

	"github.com/observability-c/logs-mcp-server/internal/auth"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
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
	client        *client.Client
	authenticator *auth.Authenticator
	logger        *zap.Logger
}

// New creates a new health checker
func New(client *client.Client, authenticator *auth.Authenticator, logger *zap.Logger) *Checker {
	return &Checker{
		client:        client,
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
		check.Message = fmt.Sprintf("Authentication failed: %v", err)
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

	_, err := c.client.Do(checkCtx, req)
	check.Duration = time.Since(start)

	if err != nil {
		// Degraded if we can't reach the API, but auth works
		if check.Duration > 3*time.Second {
			check.Status = StatusDegraded
			check.Message = "API responding slowly"
		} else {
			check.Status = StatusUnhealthy
			check.Message = fmt.Sprintf("API unreachable: %v", err)
		}
		c.logger.Warn("Health check failed: API connectivity",
			zap.Error(err),
			zap.Duration("duration", check.Duration),
		)
	} else {
		check.Status = StatusHealthy
		check.Message = "API reachable"
		c.logger.Debug("Health check passed: API connectivity",
			zap.Duration("duration", check.Duration),
		)
	}

	return check
}
