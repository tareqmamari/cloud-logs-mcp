package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all configuration for the MCP server
type Config struct {
	// IBM Cloud Logs Service Configuration
	ServiceURL   string `json:"service_url"`
	APIKey       string `json:"api_key,omitempty"` // Not stored in files, from env only
	Region       string `json:"region"`
	InstanceName string `json:"instance_name,omitempty"` // Optional friendly name for this instance

	// HTTP Client Configuration
	Timeout         time.Duration `json:"timeout"`
	MaxRetries      int           `json:"max_retries"`
	RetryWaitMin    time.Duration `json:"retry_wait_min"`
	RetryWaitMax    time.Duration `json:"retry_wait_max"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout"`

	// Rate Limiting
	RateLimit         int `json:"rate_limit"`          // requests per second
	RateLimitBurst    int `json:"rate_limit_burst"`    // burst size
	EnableRateLimit   bool `json:"enable_rate_limit"`

	// Security
	TLSVerify       bool `json:"tls_verify"`
	AllowedIPRanges []string `json:"allowed_ip_ranges,omitempty"`

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
	}

	// Try to load from config file if specified
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		if err := loadFromFile(cfg, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables (these take precedence)
	loadFromEnv(cfg)

	return cfg, nil
}

func loadFromFile(cfg *Config, filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cfg)
}

func loadFromEnv(cfg *Config) {
	if v := os.Getenv("LOGS_SERVICE_URL"); v != "" {
		cfg.ServiceURL = v
	}
	if v := os.Getenv("LOGS_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("LOGS_REGION"); v != "" {
		cfg.Region = v
	}
	if v := os.Getenv("LOGS_INSTANCE_NAME"); v != "" {
		cfg.InstanceName = v
	}
	if v := os.Getenv("LOGS_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
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
	if v := os.Getenv("LOGS_ENABLE_RATE_LIMIT"); v != "" {
		cfg.EnableRateLimit = v == "true" || v == "1"
	}
	if v := os.Getenv("LOGS_TLS_VERIFY"); v != "" {
		cfg.TLSVerify = v == "true" || v == "1"
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("LOGS_ALLOWED_IP_RANGES"); v != "" {
		cfg.AllowedIPRanges = strings.Split(v, ",")
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
		redacted.APIKey = "***REDACTED***"
	}
	return &redacted
}
