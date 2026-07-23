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
	"strconv"
	"strings"
	"time"

	"github.com/tareqmamari/cloud-logs-mcp/internal/security"
)

// Config holds all configuration for the MCP server
type Config struct {
	// IBM Cloud Logs Service Configuration
	ServiceURL   string `json:"service_url"`
	APIKey       string `json:"api_key,omitempty"` //nolint:gosec // Not stored in files, from env only
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

	// Observability
	EnableTracing   bool `json:"enable_tracing"`   // Enable distributed tracing (default: true)
	EnableAuditLog  bool `json:"enable_audit_log"` // Enable audit logging (default: true)
	MetricsEndpoint bool `json:"metrics_endpoint"` // Enable Prometheus metrics endpoint (default: true)

	// Health & Metrics HTTP Server
	HealthPort      int           `json:"health_port"`      // Port for health/metrics HTTP server (default: 0/disabled; set >0 to enable)
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
		// Health & shutdown defaults.
		// Opt-in: the health/metrics HTTP server is only started when
		// HealthPort > 0. Defaulting to 0 avoids binding a TCP port unless
		// explicitly requested, so multiple instances (e.g. one per
		// environment under Claude Desktop, which talks to each over stdio)
		// can run side by side without colliding on a shared health port.
		// Orchestrated deployments that want liveness/readiness/metrics set
		// LOGS_HEALTH_PORT (or health_port) explicitly.
		HealthPort:      0,
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
	if err := loadFromEnv(cfg); err != nil {
		return nil, err
	}

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
	// The path is operator-supplied (via the CONFIG_FILE env var), so an
	// absolute path or one containing ".." is legitimate; rejecting on ".."
	// after Clean is a no-op check that gives a false sense of safety.
	// Instead, verify the resolved path actually points at a regular file
	// before reading it, which is the check that matters for G304.
	cleanPath := filepath.Clean(path)

	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to access config file %q: %w", cleanPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("config file path %q is not a regular file", cleanPath)
	}

	// Read the file
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path is operator-supplied and verified to be a regular file above
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// api_key must never come from a config file (it's sourced from the
	// LOGS_API_KEY env var only, to avoid the key ending up in a file on
	// disk or in version control). Reject the file outright if it set one.
	if cfg.APIKey != "" {
		cfg.APIKey = ""
		return fmt.Errorf("config file %q must not set api_key; use the LOGS_API_KEY environment variable instead", cleanPath)
	}

	return nil
}

func loadFromEnv(cfg *Config) error {
	loadStringEnvs(cfg)
	if err := loadDurationEnvs(cfg); err != nil {
		return err
	}
	if err := loadIntEnvs(cfg); err != nil {
		return err
	}
	if err := loadBoolEnvs(cfg); err != nil {
		return err
	}
	return nil
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

// durationEnvVars lists the env vars parsed as time.Duration, in load order.
func durationEnvVars(cfg *Config) []struct {
	name string
	dst  *time.Duration
} {
	return []struct {
		name string
		dst  *time.Duration
	}{
		{"LOGS_TIMEOUT", &cfg.Timeout},
		{"LOGS_QUERY_TIMEOUT", &cfg.QueryTimeout},
		{"LOGS_BACKGROUND_POLL_TIMEOUT", &cfg.BackgroundPollTimeout},
		{"LOGS_BULK_OPERATION_TIMEOUT", &cfg.BulkOperationTimeout},
		{"LOGS_SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	}
}

// loadDurationEnvs parses duration env vars strictly: an invalid value fails
// the load rather than silently keeping the default, naming the offending
// variable and value in the returned error.
func loadDurationEnvs(cfg *Config) error {
	for _, f := range durationEnvVars(cfg) {
		v := os.Getenv(f.name)
		if v == "" {
			continue
		}
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration for %s=%q: %w", f.name, v, err)
		}
		*f.dst = d
	}
	return nil
}

// intEnvVars lists the env vars parsed as int, in load order.
func intEnvVars(cfg *Config) []struct {
	name string
	dst  *int
} {
	return []struct {
		name string
		dst  *int
	}{
		{"LOGS_MAX_RETRIES", &cfg.MaxRetries},
		{"LOGS_RATE_LIMIT", &cfg.RateLimit},
		{"LOGS_RATE_LIMIT_BURST", &cfg.RateLimitBurst},
		{"LOGS_HEALTH_PORT", &cfg.HealthPort},
	}
}

// loadIntEnvs parses integer env vars strictly: an invalid value fails the
// load rather than silently keeping the default, naming the offending
// variable and value in the returned error.
func loadIntEnvs(cfg *Config) error {
	for _, f := range intEnvVars(cfg) {
		v := os.Getenv(f.name)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid integer for %s=%q: %w", f.name, v, err)
		}
		*f.dst = n
	}
	return nil
}

// boolEnvVars lists the env vars parsed as bool, in load order.
func boolEnvVars(cfg *Config) []struct {
	name string
	dst  *bool
} {
	return []struct {
		name string
		dst  *bool
	}{
		{"LOGS_ENABLE_RATE_LIMIT", &cfg.EnableRateLimit},
		{"LOGS_ENABLE_TRACING", &cfg.EnableTracing},
		{"LOGS_ENABLE_AUDIT_LOG", &cfg.EnableAuditLog},
		{"LOGS_METRICS_ENDPOINT", &cfg.MetricsEndpoint},
	}
}

// loadBoolEnvs parses boolean env vars strictly via strconv.ParseBool
// (accepts 1/t/T/TRUE/true/True/0/f/F/FALSE/false/False). An invalid value
// (e.g. "yes", which was previously silently treated as false) fails the
// load, naming the offending variable and value in the returned error.
func loadBoolEnvs(cfg *Config) error {
	for _, f := range boolEnvVars(cfg) {
		v := os.Getenv(f.name)
		if v == "" {
			continue
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s=%q: %w", f.name, v, err)
		}
		*f.dst = b
	}
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServiceURL == "" {
		return errors.New("LOGS_SERVICE_URL is required")
	}
	if err := validateServiceURL(c.ServiceURL); err != nil {
		return err
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
	if err := validateHealthPort(c.HealthPort); err != nil {
		return err
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown_timeout must be positive")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// validateServiceURL requires ServiceURL to be a parseable URL with scheme
// https. http is allowed only against localhost/127.0.0.1/::1 so local
// development (e.g. against a local mock server) keeps working.
func validateServiceURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("LOGS_SERVICE_URL is not a valid URL: %q: %w", raw, err)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("LOGS_SERVICE_URL must use https (http is only allowed for localhost/127.0.0.1/::1): %q", raw)
	default:
		return fmt.Errorf("LOGS_SERVICE_URL must use https scheme: %q", raw)
	}
}

// validateHealthPort requires the health port to be 0 (disabled) or a valid
// TCP port number (1-65535).
func validateHealthPort(port int) error {
	if port == 0 {
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("health_port must be 0 (disabled) or between 1 and 65535, got %d", port)
	}
	return nil
}

// MarshalJSON implements json.Marshaler, redacting APIKey so that any future
// (or accidental) serialization of Config — e.g. logging, debug endpoints —
// can't leak the raw API key. Use Redact() for a partially-masked copy
// intended for human-readable debug output; MarshalJSON fully redacts.
func (c *Config) MarshalJSON() ([]byte, error) {
	type alias Config

	redactedKey := ""
	if c.APIKey != "" {
		redactedKey = "***REDACTED***"
	}

	return json.Marshal(&struct { // #nosec G117 -- this is the redaction: APIKey is set to redactedKey (never the raw key) above
		APIKey string `json:"api_key,omitempty"`
		*alias
	}{
		APIKey: redactedKey,
		alias:  (*alias)(c),
	})
}

// durationJSONFields lists the duration fields' JSON tag name alongside a
// pointer to the field itself, in struct order. Shared by UnmarshalJSON so
// each field is named consistently in parse errors.
func durationJSONFields(c *Config) []struct {
	name string
	dst  *time.Duration
} {
	return []struct {
		name string
		dst  *time.Duration
	}{
		{"timeout", &c.Timeout},
		{"retry_wait_min", &c.RetryWaitMin},
		{"retry_wait_max", &c.RetryWaitMax},
		{"idle_conn_timeout", &c.IdleConnTimeout},
		{"query_timeout", &c.QueryTimeout},
		{"background_poll_timeout", &c.BackgroundPollTimeout},
		{"bulk_operation_timeout", &c.BulkOperationTimeout},
		{"shutdown_timeout", &c.ShutdownTimeout},
	}
}

// UnmarshalJSON implements json.Unmarshaler. encoding/json's default
// time.Duration handling only accepts an integer number of nanoseconds, but
// config.example.json (and any hand-written config file) documents durations
// as human-readable strings like "30s" - the documented, copy-pasteable
// format everywhere else in this codebase (env vars via time.ParseDuration,
// CLI flags, etc). Without this, loading the documented example file either
// fails outright or - depending on the JSON shape - silently leaves duration
// fields at their defaults.
//
// Uses the shadow-struct pattern (mirroring MarshalJSON's redaction above):
// an anonymous struct embeds *alias (all of Config's fields via promotion)
// and additionally declares each duration field as json.RawMessage with the
// same json tag. json.Unmarshal resolves the tag conflict in favor of the
// shallower, explicitly-declared field, so the raw value is captured there
// and the promoted time.Duration field from *alias is left untouched. Each
// captured value is then parsed by parseDurationJSON, which accepts either a
// duration string ("30s", the documented config-file format) or a JSON
// number of nanoseconds (what MarshalJSON emits via time.Duration's default
// encoding, needed so Marshal->Unmarshal round-trips); an unparseable value
// returns an error naming both the field and the offending value rather than
// a generic encoding/json error.
//
// This runs before loadFromFile's api_key-from-file rejection (which
// inspects c.APIKey after Unmarshal completes), and composes with it
// unchanged: api_key is an ordinary string field with no shadow entry, so it
// unmarshals normally.
func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config

	shadow := &struct {
		Timeout               json.RawMessage `json:"timeout"`
		RetryWaitMin          json.RawMessage `json:"retry_wait_min"`
		RetryWaitMax          json.RawMessage `json:"retry_wait_max"`
		IdleConnTimeout       json.RawMessage `json:"idle_conn_timeout"`
		QueryTimeout          json.RawMessage `json:"query_timeout"`
		BackgroundPollTimeout json.RawMessage `json:"background_poll_timeout"`
		BulkOperationTimeout  json.RawMessage `json:"bulk_operation_timeout"`
		ShutdownTimeout       json.RawMessage `json:"shutdown_timeout"`
		*alias
	}{
		alias: (*alias)(c),
	}

	if err := json.Unmarshal(data, shadow); err != nil {
		return err
	}

	rawByField := map[string]json.RawMessage{
		"timeout":                 shadow.Timeout,
		"retry_wait_min":          shadow.RetryWaitMin,
		"retry_wait_max":          shadow.RetryWaitMax,
		"idle_conn_timeout":       shadow.IdleConnTimeout,
		"query_timeout":           shadow.QueryTimeout,
		"background_poll_timeout": shadow.BackgroundPollTimeout,
		"bulk_operation_timeout":  shadow.BulkOperationTimeout,
		"shutdown_timeout":        shadow.ShutdownTimeout,
	}

	for _, f := range durationJSONFields(c) {
		raw := rawByField[f.name]
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		parsed, err := parseDurationJSON(f.name, raw)
		if err != nil {
			return err
		}
		*f.dst = parsed
	}

	return nil
}

// parseDurationJSON parses a single duration field's raw JSON value,
// accepting either a human-readable duration string ("30s") or a JSON number
// of nanoseconds (encoding/json's default time.Duration representation, and
// what Config's own MarshalJSON emits). name is the JSON field name, used to
// produce an error that names the offending field and value.
func parseDurationJSON(name string, raw json.RawMessage) (time.Duration, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("config field %q: invalid duration %q: %w", name, s, err)
		}
		return d, nil
	}

	var nanos int64
	if err := json.Unmarshal(raw, &nanos); err == nil {
		return time.Duration(nanos), nil
	}

	return 0, fmt.Errorf("config field %q: invalid duration %s: must be a duration string (e.g. %q) or integer nanoseconds", name, string(raw), "30s")
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

// MaskAPIKey returns a masked version of an API key for safe logging.
// Delegates to security.MaskAPIKey, the single source of truth for this
// logic; that implementation's edge-case behavior is stricter (an empty key
// still yields a masked placeholder instead of an empty string).
func MaskAPIKey(apiKey string) string {
	return security.MaskAPIKey(apiKey)
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
