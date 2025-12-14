// Package config provides configuration management for the IBM Cloud Logs MCP server.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config holds all configuration for the MCP server
type Config struct {
	// IBM Cloud Logs Service Configuration
	ServiceURL   string `json:"service_url"`
	APIKey       string `json:"api_key,omitempty"` // Not stored in files, from env only
	Region       string `json:"region"`
	InstanceID   string `json:"instance_id,omitempty"`   // Service instance ID (alternative to service_url)
	InstanceName string `json:"instance_name,omitempty"` // Optional friendly name for this instance
	IAMURL       string `json:"iam_url,omitempty"`       // Optional IAM endpoint (default: production, or iam.test.cloud.ibm.com for staging)

	// HTTP Client Configuration
	Timeout         time.Duration `json:"timeout"`
	MaxRetries      int           `json:"max_retries"`
	RetryWaitMin    time.Duration `json:"retry_wait_min"`
	RetryWaitMax    time.Duration `json:"retry_wait_max"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout"`

	// Operation-Specific Timeouts
	QueryTimeout          time.Duration `json:"query_timeout"`           // Timeout for synchronous queries (default: 60s)
	BackgroundPollTimeout time.Duration `json:"background_poll_timeout"` // Timeout for background query status checks (default: 10s)
	BulkOperationTimeout  time.Duration `json:"bulk_operation_timeout"`  // Timeout for bulk operations (default: 120s)

	// Rate Limiting
	RateLimit       int  `json:"rate_limit"`       // requests per second
	RateLimitBurst  int  `json:"rate_limit_burst"` // burst size
	EnableRateLimit bool `json:"enable_rate_limit"`

	// Security
	TLSVerify bool `json:"tls_verify"`

	// Observability
	EnableTracing   bool `json:"enable_tracing"`   // Enable distributed tracing (default: true)
	EnableAuditLog  bool `json:"enable_audit_log"` // Enable audit logging (default: true)
	MetricsEndpoint bool `json:"metrics_endpoint"` // Enable Prometheus metrics endpoint (default: true)

	// Health & Metrics HTTP Server
	HealthPort      int           `json:"health_port"`      // Port for health/metrics HTTP server (default: 8080, 0 to disable)
	HealthBindAddr  string        `json:"health_bind_addr"` // Bind address for health server (default: 127.0.0.1 for security)
	ShutdownTimeout time.Duration `json:"shutdown_timeout"` // Timeout for graceful shutdown (default: 30s)

	// Logging
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"` // json or console
}

// Load configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Second,
		RetryWaitMax:    30 * time.Second,
		MaxIdleConns:    10,
		IdleConnTimeout: 90 * time.Second,
		RateLimit:       100,
		RateLimitBurst:  20,
		EnableRateLimit: true,
		TLSVerify:       true,
		LogLevel:        "info",
		LogFormat:       "json",
		// Operation-specific timeouts
		QueryTimeout:          60 * time.Second,
		BackgroundPollTimeout: 10 * time.Second,
		BulkOperationTimeout:  120 * time.Second,
		// Observability defaults
		EnableTracing:   true,
		EnableAuditLog:  true,
		MetricsEndpoint: true, // Enabled by default for operational visibility
		// Health & shutdown defaults
		HealthPort:      8080,
		HealthBindAddr:  "127.0.0.1", // Bind to localhost by default for security
		ShutdownTimeout: 30 * time.Second,
	}

	// Try to load from config file if specified
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		if err := loadFromFile(cfg, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables (these take precedence)
	loadFromEnv(cfg)

	// If ServiceURL is provided, extract region and instance ID from it
	if cfg.ServiceURL != "" {
		if cfg.Region == "" {
			cfg.Region = ExtractRegionFromURL(cfg.ServiceURL)
		}
		if cfg.InstanceID == "" {
			cfg.InstanceID = ExtractInstanceIDFromURL(cfg.ServiceURL)
		}
	}

	// If ServiceURL is not provided but Region and InstanceID are, construct the URL
	if cfg.ServiceURL == "" && cfg.Region != "" && cfg.InstanceID != "" {
		cfg.ServiceURL = BuildServiceURL(cfg.InstanceID, cfg.Region)
	}

	return cfg, nil
}

func loadFromFile(cfg *Config, path string) error {
	// Validate and clean the file path to prevent path traversal attacks
	// This eliminates the G304 gosec finding by validating paths before access

	cleanPath := filepath.Clean(path)

	// Prevent path traversal by checking for ".." components
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: path traversal detected")
	}

	// Read the file
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path is validated above
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return json.Unmarshal(data, cfg)
}

func loadFromEnv(cfg *Config) {
	loadStringEnvs(cfg)
	loadDurationEnvs(cfg)
	loadIntEnvs(cfg)
	loadBoolEnvs(cfg)
}

func loadStringEnvs(cfg *Config) {
	if v := os.Getenv("LOGS_SERVICE_URL"); v != "" {
		cfg.ServiceURL = v
	}
	if v := os.Getenv("LOGS_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("LOGS_REGION"); v != "" {
		cfg.Region = v
	}
	if v := os.Getenv("LOGS_INSTANCE_ID"); v != "" {
		cfg.InstanceID = v
	}
	if v := os.Getenv("LOGS_INSTANCE_NAME"); v != "" {
		cfg.InstanceName = v
	}
	if v := os.Getenv("LOGS_IAM_URL"); v != "" {
		cfg.IAMURL = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("LOGS_HEALTH_BIND_ADDR"); v != "" {
		cfg.HealthBindAddr = v
	}
}

func loadDurationEnvs(cfg *Config) {
	if v := os.Getenv("LOGS_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv("LOGS_QUERY_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.QueryTimeout = d
		}
	}
	if v := os.Getenv("LOGS_BACKGROUND_POLL_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.BackgroundPollTimeout = d
		}
	}
	if v := os.Getenv("LOGS_BULK_OPERATION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.BulkOperationTimeout = d
		}
	}
	if v := os.Getenv("LOGS_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ShutdownTimeout = d
		}
	}
}

func loadIntEnvs(cfg *Config) {
	if v := os.Getenv("LOGS_MAX_RETRIES"); v != "" {
		var retries int
		if _, err := fmt.Sscanf(v, "%d", &retries); err == nil {
			cfg.MaxRetries = retries
		}
	}
	if v := os.Getenv("LOGS_RATE_LIMIT"); v != "" {
		var limit int
		if _, err := fmt.Sscanf(v, "%d", &limit); err == nil {
			cfg.RateLimit = limit
		}
	}
	if v := os.Getenv("LOGS_RATE_LIMIT_BURST"); v != "" {
		var burst int
		if _, err := fmt.Sscanf(v, "%d", &burst); err == nil {
			cfg.RateLimitBurst = burst
		}
	}
	if v := os.Getenv("LOGS_HEALTH_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			cfg.HealthPort = port
		}
	}
}

func loadBoolEnvs(cfg *Config) {
	if v := os.Getenv("LOGS_ENABLE_RATE_LIMIT"); v != "" {
		cfg.EnableRateLimit = v == "true" || v == "1"
	}
	if v := os.Getenv("LOGS_TLS_VERIFY"); v != "" {
		cfg.TLSVerify = v == "true" || v == "1"
	}
	if v := os.Getenv("LOGS_ENABLE_TRACING"); v != "" {
		cfg.EnableTracing = v == "true" || v == "1"
	}
	if v := os.Getenv("LOGS_ENABLE_AUDIT_LOG"); v != "" {
		cfg.EnableAuditLog = v == "true" || v == "1"
	}
	if v := os.Getenv("LOGS_METRICS_ENDPOINT"); v != "" {
		cfg.MetricsEndpoint = v == "true" || v == "1"
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServiceURL == "" {
		return errors.New("LOGS_SERVICE_URL is required")
	}
	if c.APIKey == "" {
		return errors.New("LOGS_API_KEY is required")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return errors.New("max_retries must be non-negative")
	}
	if c.RateLimit <= 0 && c.EnableRateLimit {
		return errors.New("rate_limit must be positive when rate limiting is enabled")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// Redact returns a copy of the config with sensitive data removed
func (c *Config) Redact() *Config {
	redacted := *c
	if redacted.APIKey != "" {
		// Show first 4 and last 4 characters for debugging, fully mask short keys
		if len(redacted.APIKey) > 8 {
			redacted.APIKey = redacted.APIKey[:4] + "..." + redacted.APIKey[len(redacted.APIKey)-4:]
		} else {
			redacted.APIKey = "***REDACTED***"
		}
	}
	return &redacted
}

// MaskAPIKey returns a masked version of an API key for safe logging
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// ExtractRegionFromURL extracts the IBM Cloud region from a service URL.
// Supports formats:
//   - [instance-id].api.[region].logs.cloud.ibm.com
//   - [instance-id].api.private.[region].logs.cloud.ibm.com
//
// Returns empty string if the region cannot be extracted.
func ExtractRegionFromURL(serviceURL string) string {
	if serviceURL == "" {
		return ""
	}

	parsed, err := url.Parse(serviceURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	// Production: [instance-id].api.[private.]<region>.logs.cloud.ibm.com
	prodPattern := regexp.MustCompile(`\.api\.(?:private\.)?([a-z]{2}-[a-z]+)\.logs\.cloud\.ibm\.com$`)
	if matches := prodPattern.FindStringSubmatch(host); len(matches) >= 2 {
		return matches[1]
	}

	// Dev: [instance-id].api.<env-name>.<region>.logs.dev.cloud.ibm.com
	// Region is "env-name.region" (e.g., "preprod.us-south")
	devPattern := regexp.MustCompile(`\.api\.([a-z0-9-]+)\.([a-z]{2}-[a-z]+)\.logs\.dev\.cloud\.ibm\.com$`)
	if matches := devPattern.FindStringSubmatch(host); len(matches) >= 3 {
		return matches[1] + "." + matches[2]
	}

	// Stage: [instance-id].api.<region>.logs.test.cloud.ibm.com
	stagePattern := regexp.MustCompile(`\.api\.([a-z]{2}-[a-z]+)\.logs\.test\.cloud\.ibm\.com$`)
	if matches := stagePattern.FindStringSubmatch(host); len(matches) >= 2 {
		return matches[1]
	}

	return ""
}

// BuildServiceURL constructs an IBM Cloud Logs service URL from instance ID and region.
// Returns the production API endpoint URL.
func BuildServiceURL(instanceID, region string) string {
	if instanceID == "" || region == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.api.%s.logs.cloud.ibm.com", instanceID, region)
}

// ExtractInstanceIDFromURL extracts the service instance ID from a service URL.
// The instance ID is the first component of the hostname.
// Returns empty string if the instance ID cannot be extracted.
func ExtractInstanceIDFromURL(serviceURL string) string {
	if serviceURL == "" {
		return ""
	}

	parsed, err := url.Parse(serviceURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	// Instance ID is the first part before ".api."
	idx := strings.Index(host, ".api.")
	if idx > 0 {
		return host[:idx]
	}

	return ""
}
