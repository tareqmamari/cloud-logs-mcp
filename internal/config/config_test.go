package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"LOGS_SERVICE_URL": "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
				"LOGS_API_KEY":     "test-api-key", // pragma: allowlist secret
				"LOGS_REGION":      "us-south",
			},
			wantErr: false,
		},
		{
			name: "missing service URL",
			envVars: map[string]string{
				"LOGS_API_KEY": "test-api-key", // pragma: allowlist secret
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			envVars: map[string]string{
				"LOGS_SERVICE_URL": "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://[your-instance-id].api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", cfg.Timeout)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected default max_retries 3, got %d", cfg.MaxRetries)
	}

	if cfg.RateLimit != 100 {
		t.Errorf("Expected default rate_limit 100, got %d", cfg.RateLimit)
	}

	if !cfg.TLSVerify {
		t.Error("Expected TLSVerify to be true by default")
	}

	if !cfg.EnableRateLimit {
		t.Error("Expected EnableRateLimit to be true by default")
	}
}

func TestConfigRedact(t *testing.T) {
	cfg := &Config{
		ServiceURL: "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
		APIKey:     "secret-key-12345", // pragma: allowlist secret
	}

	redacted := cfg.Redact()

	if redacted.APIKey == cfg.APIKey { // pragma: allowlist secret
		t.Error("API key should be redacted")
	}

	// For keys longer than 8 chars, we show first 4 and last 4 characters
	expectedMasked := "secr...2345"        // pragma: allowlist secret
	if redacted.APIKey != expectedMasked { // pragma: allowlist secret
		t.Errorf("Expected %s, got %s", expectedMasked, redacted.APIKey)
	}

	if redacted.ServiceURL != cfg.ServiceURL {
		t.Error("ServiceURL should not be changed")
	}
}

func TestConfigRedactShortKey(t *testing.T) {
	cfg := &Config{
		ServiceURL: "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
		APIKey:     "short", // pragma: allowlist secret
	}

	redacted := cfg.Redact()

	// Short keys should be fully redacted
	if redacted.APIKey != "***REDACTED***" {
		t.Errorf("Expected ***REDACTED***, got %s", redacted.APIKey)
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "***"},
		{"exactly8", "***"},
		{"secret-key-12345", "secr...2345"}, // pragma: allowlist secret
		{"abcdefghijklmnopqrstuvwxyz", "abcd...wxyz"},
	}

	for _, tt := range tests {
		result := MaskAPIKey(tt.input)
		if result != tt.expected {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				ServiceURL:      "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				MaxRetries:      3,
				RateLimit:       100,
				EnableRateLimit: true,
				LogLevel:        "info",
			},
			wantErr: false,
		},
		{
			name: "invalid timeout",
			config: Config{
				ServiceURL: "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
				APIKey:     "test-key", // pragma: allowlist secret
				Timeout:    0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "invalid log level",
			config: Config{
				ServiceURL: "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
				APIKey:     "test-key", // pragma: allowlist secret
				Timeout:    30 * time.Second,
				LogLevel:   "invalid",
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
