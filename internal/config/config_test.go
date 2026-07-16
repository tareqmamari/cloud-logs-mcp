package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
				"LOGS_SERVICE_URL": "https://abc123.api.us-south.logs.cloud.ibm.com",
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
				"LOGS_SERVICE_URL": "https://abc123.api.us-south.logs.cloud.ibm.com",
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
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
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

	if !cfg.EnableRateLimit {
		t.Error("Expected EnableRateLimit to be true by default")
	}
}

func TestConfigRedact(t *testing.T) {
	cfg := &Config{
		ServiceURL: "https://abc123.api.us-south.logs.cloud.ibm.com",
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
		ServiceURL: "https://abc123.api.us-south.logs.cloud.ibm.com",
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
		{"", "***"}, // stricter edge case: consolidated onto security.MaskAPIKey, which always masks
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

func TestExtractRegionFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		// Production endpoints
		{
			name:     "standard public URL",
			url:      "https://abc123.api.us-south.logs.cloud.ibm.com",
			expected: "us-south",
		},
		{
			name:     "private endpoint URL",
			url:      "https://abc123.api.private.us-south.logs.cloud.ibm.com",
			expected: "us-south",
		},
		{
			name:     "eu-de region",
			url:      "https://instance-id.api.eu-de.logs.cloud.ibm.com",
			expected: "eu-de",
		},
		{
			name:     "au-syd region",
			url:      "https://instance-id.api.au-syd.logs.cloud.ibm.com",
			expected: "au-syd",
		},
		{
			name:     "br-sao region private",
			url:      "https://instance-id.api.private.br-sao.logs.cloud.ibm.com",
			expected: "br-sao",
		},
		{
			name:     "jp-tok region",
			url:      "https://instance-id.api.jp-tok.logs.cloud.ibm.com",
			expected: "jp-tok",
		},
		// Dev endpoints (logs.dev.cloud.ibm.com)
		{
			name:     "dev endpoint with env name",
			url:      "https://instance-id.api.preprod.us-south.logs.dev.cloud.ibm.com",
			expected: "preprod.us-south",
		},
		{
			name:     "dev endpoint eu-de",
			url:      "https://abc123.api.dev1.eu-de.logs.dev.cloud.ibm.com",
			expected: "dev1.eu-de",
		},
		{
			name:     "dev ingest endpoint",
			url:      "https://instance-id.ingest.preprod.us-south.logs.dev.cloud.ibm.com",
			expected: "", // ingest endpoints not matched by api pattern
		},
		// Stage endpoints (logs.test.cloud.ibm.com)
		{
			name:     "stage endpoint us-south",
			url:      "https://instance-id.api.us-south.logs.test.cloud.ibm.com",
			expected: "us-south",
		},
		{
			name:     "stage endpoint eu-de",
			url:      "https://abc123.api.eu-de.logs.test.cloud.ibm.com",
			expected: "eu-de",
		},
		// Edge cases
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: "",
		},
		{
			name:     "non-IBM URL",
			url:      "https://api.example.com",
			expected: "",
		},
		{
			name:     "URL with placeholder brackets",
			url:      "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
			expected: "", // Brackets in URL make it unparseable - expected behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRegionFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractRegionFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestRegionAutoExtraction(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://instance-id.api.eu-de.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
	// Note: LOGS_REGION is NOT set

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Region != "eu-de" {
		t.Errorf("Expected region to be auto-extracted as 'eu-de', got %q", cfg.Region)
	}
}

func TestRegionExplicitOverride(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://instance-id.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key")     // pragma: allowlist secret
	_ = os.Setenv("LOGS_REGION", "custom-region") // Explicit override

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Explicit LOGS_REGION should take precedence
	if cfg.Region != "custom-region" {
		t.Errorf("Expected explicit region 'custom-region', got %q", cfg.Region)
	}
}

func TestBuildServiceURL(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		region     string
		expected   string
	}{
		{
			name:       "us-south region",
			instanceID: "abc123",
			region:     "us-south",
			expected:   "https://abc123.api.us-south.logs.cloud.ibm.com",
		},
		{
			name:       "eu-de region",
			instanceID: "my-instance",
			region:     "eu-de",
			expected:   "https://my-instance.api.eu-de.logs.cloud.ibm.com",
		},
		{
			name:       "empty instance ID",
			instanceID: "",
			region:     "us-south",
			expected:   "",
		},
		{
			name:       "empty region",
			instanceID: "abc123",
			region:     "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildServiceURL(tt.instanceID, tt.region)
			if result != tt.expected {
				t.Errorf("BuildServiceURL(%q, %q) = %q, want %q", tt.instanceID, tt.region, result, tt.expected)
			}
		})
	}
}

func TestLoadFromRegionAndInstanceID(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_REGION", "eu-de")
	_ = os.Setenv("LOGS_INSTANCE_ID", "my-instance-id")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expectedURL := "https://my-instance-id.api.eu-de.logs.cloud.ibm.com"
	if cfg.ServiceURL != expectedURL {
		t.Errorf("Expected ServiceURL %q, got %q", expectedURL, cfg.ServiceURL)
	}

	if cfg.Region != "eu-de" {
		t.Errorf("Expected Region 'eu-de', got %q", cfg.Region)
	}

	if cfg.InstanceID != "my-instance-id" {
		t.Errorf("Expected InstanceID 'my-instance-id', got %q", cfg.InstanceID)
	}
}

func TestInstanceIDAutoExtraction(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://extracted-instance.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.InstanceID != "extracted-instance" {
		t.Errorf("Expected InstanceID to be auto-extracted as 'extracted-instance', got %q", cfg.InstanceID)
	}

	if cfg.Region != "us-south" {
		t.Errorf("Expected Region to be auto-extracted as 'us-south', got %q", cfg.Region)
	}
}

func TestExtractInstanceIDFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		// Production endpoints
		{
			name:     "standard public URL",
			url:      "https://abc123-def456.api.us-south.logs.cloud.ibm.com",
			expected: "abc123-def456",
		},
		{
			name:     "private endpoint URL",
			url:      "https://my-instance-id.api.private.us-south.logs.cloud.ibm.com",
			expected: "my-instance-id",
		},
		{
			name:     "UUID-style instance ID",
			url:      "https://a1b2c3d4-e5f6-7890-abcd-ef1234567890.api.eu-de.logs.cloud.ibm.com",
			expected: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		// Dev endpoints
		{
			name:     "dev endpoint",
			url:      "https://instance-id.api.preprod.us-south.logs.dev.cloud.ibm.com",
			expected: "instance-id",
		},
		// Stage endpoints
		{
			name:     "stage endpoint",
			url:      "https://test-instance.api.us-south.logs.test.cloud.ibm.com",
			expected: "test-instance",
		},
		// Edge cases
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: "",
		},
		{
			name:     "URL without .api. segment",
			url:      "https://example.com",
			expected: "",
		},
		{
			name:     "URL with placeholder brackets",
			url:      "https://[your-instance-id].api.us-south.logs.cloud.ibm.com",
			expected: "", // Brackets make URL unparseable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractInstanceIDFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractInstanceIDFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
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
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				MaxRetries:      3,
				RateLimit:       100,
				EnableRateLimit: true,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid timeout",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         0,
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "invalid log level",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "invalid",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "http scheme rejected for non-local host",
			config: Config{
				ServiceURL:      "http://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "must use https",
		},
		{
			name: "http scheme allowed for localhost",
			config: Config{
				ServiceURL:      "http://localhost:9000",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "http scheme allowed for 127.0.0.1",
			config: Config{
				ServiceURL:      "http://127.0.0.1:9000",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "http scheme allowed for ::1",
			config: Config{
				ServiceURL:      "http://[::1]:9000",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "unparseable service URL rejected",
			config: Config{
				ServiceURL:      "://not-a-url",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "not a valid URL",
		},
		{
			name: "non-http(s) scheme rejected",
			config: Config{
				ServiceURL:      "ftp://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "must use https",
		},
		{
			name: "health port negative rejected",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
				HealthPort:      -1,
			},
			wantErr: true,
			errMsg:  "health_port",
		},
		{
			name: "health port too large rejected",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
				HealthPort:      70000,
			},
			wantErr: true,
			errMsg:  "health_port",
		},
		{
			name: "health port zero (disabled) accepted",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
				HealthPort:      0,
			},
			wantErr: false,
		},
		{
			name: "shutdown timeout zero rejected",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 0,
			},
			wantErr: true,
			errMsg:  "shutdown_timeout must be positive",
		},
		{
			name: "shutdown timeout negative rejected",
			config: Config{
				ServiceURL:      "https://abc123.api.us-south.logs.cloud.ibm.com",
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "shutdown_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// --- Strict env parsing ---

func TestLoadFromEnv_InvalidIntRejected(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"max retries", "LOGS_MAX_RETRIES"},
		{"rate limit", "LOGS_RATE_LIMIT"},
		{"rate limit burst", "LOGS_RATE_LIMIT_BURST"},
		{"health port", "LOGS_HEALTH_PORT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
			_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
			_ = os.Setenv(tt.env, "8080abc")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with %s=8080abc should fail", tt.env)
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Errorf("error %q should name the env var %q", err.Error(), tt.env)
			}
			if !strings.Contains(err.Error(), "8080abc") {
				t.Errorf("error %q should include the invalid value %q", err.Error(), "8080abc")
			}
		})
	}
}

func TestLoadFromEnv_InvalidDurationRejected(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"timeout", "LOGS_TIMEOUT"},
		{"query timeout", "LOGS_QUERY_TIMEOUT"},
		{"background poll timeout", "LOGS_BACKGROUND_POLL_TIMEOUT"},
		{"bulk operation timeout", "LOGS_BULK_OPERATION_TIMEOUT"},
		{"shutdown timeout", "LOGS_SHUTDOWN_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
			_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
			_ = os.Setenv(tt.env, "not-a-duration")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with %s=not-a-duration should fail", tt.env)
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Errorf("error %q should name the env var %q", err.Error(), tt.env)
			}
			if !strings.Contains(err.Error(), "not-a-duration") {
				t.Errorf("error %q should include the invalid value %q", err.Error(), "not-a-duration")
			}
		})
	}
}

func TestLoadFromEnv_InvalidBoolRejected(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"enable rate limit", "LOGS_ENABLE_RATE_LIMIT"},
		{"enable tracing", "LOGS_ENABLE_TRACING"},
		{"enable audit log", "LOGS_ENABLE_AUDIT_LOG"},
		{"metrics endpoint", "LOGS_METRICS_ENDPOINT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
			_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
			_ = os.Setenv(tt.env, "yes")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with %s=yes should fail", tt.env)
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Errorf("error %q should name the env var %q", err.Error(), tt.env)
			}
			if !strings.Contains(err.Error(), "yes") {
				t.Errorf("error %q should include the invalid value %q", err.Error(), "yes")
			}
		})
	}
}

func TestLoadFromEnv_BoolAcceptsCanonicalForms(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
	_ = os.Setenv("LOGS_ENABLE_TRACING", "TRUE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if !cfg.EnableTracing {
		t.Error("LOGS_ENABLE_TRACING=TRUE should be parsed as true via strconv.ParseBool")
	}
}

func TestLoadFromEnv_ValidValuesStillWork(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret
	_ = os.Setenv("LOGS_MAX_RETRIES", "5")
	_ = os.Setenv("LOGS_TIMEOUT", "45s")
	_ = os.Setenv("LOGS_ENABLE_TRACING", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", cfg.Timeout)
	}
	if cfg.EnableTracing {
		t.Error("EnableTracing should be false")
	}
}

// --- api_key from file rejection ---

func TestLoadFromFile_RejectsAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"api_key":"leaked-key"}`), 0o600); err != nil { // pragma: allowlist secret
		t.Fatalf("failed to write test config file: %v", err)
	}

	os.Clearenv()
	_ = os.Setenv("CONFIG_FILE", path)
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when config file sets api_key")
	}
	if !strings.Contains(err.Error(), "LOGS_API_KEY") {
		t.Errorf("error %q should tell the user to use LOGS_API_KEY", err.Error())
	}
}

func TestLoadFromFile_AllowsOtherFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"region":"us-south","log_level":"debug"}`), 0o600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	os.Clearenv()
	_ = os.Setenv("CONFIG_FILE", path)
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should succeed for a config file without api_key: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

// --- Path guard ---

func TestLoadFromFile_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()

	os.Clearenv()
	_ = os.Setenv("CONFIG_FILE", dir)
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when CONFIG_FILE points at a directory")
	}
}

func TestLoadFromFile_RejectsMissingFile(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("CONFIG_FILE", "/nonexistent/path/config.json")
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when CONFIG_FILE does not exist")
	}
}

func TestLoadFromFile_DotDotInPathIsNotSpecialCased(t *testing.T) {
	// Operator-set paths containing ".." (e.g. relative to a working directory)
	// are legitimate and must not be rejected purely for containing "..".
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o750); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"region":"us-south"}`), 0o600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Reference the file via a path that traverses through ".." after cleaning
	// still resolves to the same real file.
	dotdotPath := filepath.Join(nested, "..", "config.json")

	os.Clearenv()
	_ = os.Setenv("CONFIG_FILE", dotdotPath)
	_ = os.Setenv("LOGS_SERVICE_URL", "https://abc123.api.us-south.logs.cloud.ibm.com")
	_ = os.Setenv("LOGS_API_KEY", "test-key") // pragma: allowlist secret

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should succeed for a legitimate path containing '..', got: %v", err)
	}
	if cfg.Region != "us-south" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-south")
	}
}

// --- MarshalJSON redaction ---

func TestConfigMarshalJSON_RedactsAPIKey(t *testing.T) {
	cfg := &Config{
		ServiceURL: "https://abc123.api.us-south.logs.cloud.ibm.com",
		APIKey:     "super-secret-api-key", // pragma: allowlist secret
	}

	data, err := json.Marshal(cfg) // #nosec G117 -- exercising Config.MarshalJSON's redaction; asserted below
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if strings.Contains(string(data), "super-secret-api-key") {
		t.Errorf("marshaled config leaked the API key: %s", data)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal marshaled config: %v", err)
	}
	if decoded["api_key"] == "" || decoded["api_key"] == nil { // pragma: allowlist secret
		t.Errorf("expected api_key field to be present but redacted, got %v", decoded["api_key"])
	}
	if decoded["service_url"] != cfg.ServiceURL {
		t.Errorf("service_url = %v, want %v", decoded["service_url"], cfg.ServiceURL)
	}
}

func TestConfigMarshalJSON_EmptyAPIKey(t *testing.T) {
	cfg := &Config{
		ServiceURL: "https://abc123.api.us-south.logs.cloud.ibm.com",
	}

	data, err := json.Marshal(cfg) // #nosec G117 -- exercising Config.MarshalJSON's redaction; asserted below
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal marshaled config: %v", err)
	}
	if v, ok := decoded["api_key"]; ok && v != "" {
		t.Errorf("expected empty api_key to stay empty, got %v", v)
	}
}

// --- ServiceURL scheme validation (standalone Validate() cases) ---

func TestValidateServiceURLScheme(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https any host", "https://abc123.api.us-south.logs.cloud.ibm.com", false},
		{"http localhost", "http://localhost:8080", false},
		{"http 127.0.0.1", "http://127.0.0.1:8080", false},
		{"http ::1", "http://[::1]:8080", false},
		{"http remote host rejected", "http://abc123.api.us-south.logs.cloud.ibm.com", true},
		{"unparseable URL rejected", "://bad", true},
		{"empty scheme rejected", "abc123.api.us-south.logs.cloud.ibm.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ServiceURL:      tt.url,
				APIKey:          "test-key", // pragma: allowlist secret
				Timeout:         30 * time.Second,
				LogLevel:        "info",
				ShutdownTimeout: 30 * time.Second,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() for url %q error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestExampleConfigServiceURLIsCopyPasteable guards against config.example.json's
// service_url placeholder using a bracketed form like "[your-instance-id]",
// which url.Parse interprets as an IPv6 host literal and rejects outright.
// A user who copies the example verbatim (only swapping in their own values)
// would otherwise hit a confusing "not a valid URL" error before ever
// reaching the strict-https validation this test also exercises.
//
// This reads the raw service_url field directly (rather than going through
// Load()) because config.example.json's duration fields (e.g. "30s") are
// documentation-oriented and not accepted by encoding/json's default
// time.Duration unmarshaling - a separate, pre-existing concern unrelated to
// the URL placeholder this test guards against.
func TestExampleConfigServiceURLIsCopyPasteable(t *testing.T) {
	data, err := os.ReadFile("../../config.example.json")
	if err != nil {
		t.Fatalf("failed to read config.example.json: %v", err)
	}

	var raw struct {
		ServiceURL string `json:"service_url"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to parse config.example.json: %v", err)
	}
	if raw.ServiceURL == "" {
		t.Fatal("config.example.json has no service_url set")
	}

	if err := validateServiceURL(raw.ServiceURL); err != nil {
		t.Errorf("config.example.json's service_url %q fails strict https validation: %v", raw.ServiceURL, err)
	}

	if strings.ContainsAny(raw.ServiceURL, "[]") {
		t.Errorf("config.example.json's service_url %q still contains bracketed placeholder characters", raw.ServiceURL)
	}
}
