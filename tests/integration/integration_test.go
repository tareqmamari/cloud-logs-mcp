// Package integration provides integration tests for the IBM Cloud Logs MCP server
// These tests require valid IBM Cloud credentials and will make real API calls
//
// To run integration tests:
//
//	export LOGS_API_KEY=your-api-key  // pragma: allowlist secret
//	export LOGS_INSTANCE_ID=your-instance-id
//	export LOGS_REGION=your-region
//	go test -v -tags=integration ./tests/integration/...
//
// Or use the Makefile:
//
//	make test-integration
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
	"github.com/tareqmamari/logs-mcp-server/internal/config"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	APIKey     string
	InstanceID string
	Region     string
	ServiceURL string
}

// TestContext holds shared test resources
type TestContext struct {
	Client *client.Client
	Config *TestConfig
	Logger *zap.Logger
	T      *testing.T
}

// NewTestContext creates a new test context with configured client
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	// Load test configuration from environment
	testConfig := &TestConfig{
		APIKey:     os.Getenv("LOGS_API_KEY"),
		InstanceID: os.Getenv("LOGS_INSTANCE_ID"),
		Region:     os.Getenv("LOGS_REGION"),
	}

	// Validate required environment variables
	require.NotEmpty(t, testConfig.APIKey, "LOGS_API_KEY environment variable must be set")
	require.NotEmpty(t, testConfig.InstanceID, "LOGS_INSTANCE_ID environment variable must be set")
	require.NotEmpty(t, testConfig.Region, "LOGS_REGION environment variable must be set")

	// Construct service URL
	testConfig.ServiceURL = fmt.Sprintf(
		"https://%s.api.%s.logs.cloud.ibm.com",
		testConfig.InstanceID,
		testConfig.Region,
	)

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "Failed to create logger")

	// Create client configuration
	cfg := &config.Config{
		APIKey:          testConfig.APIKey, // pragma: allowlist secret
		ServiceURL:      testConfig.ServiceURL,
		IAMURL:          "https://iam.cloud.ibm.com/identity/token",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Second,
		RetryWaitMax:    5 * time.Second,
		EnableRateLimit: true,
		RateLimit:       10,
		RateLimitBurst:  20,
		MaxIdleConns:    10,
		IdleConnTimeout: 90 * time.Second,
		TLSVerify:       true,
	}

	// Create client
	apiClient, err := client.New(cfg, logger)
	require.NoError(t, err, "Failed to create API client")

	return &TestContext{
		Client: apiClient,
		Config: testConfig,
		Logger: logger,
		T:      t,
	}
}

// Cleanup performs cleanup of test resources
func (tc *TestContext) Cleanup() {
	if tc.Client != nil {
		_ = tc.Client.Close() // Ignore error on cleanup
	}
}

// DoRequest executes an API request and returns the parsed response
func (tc *TestContext) DoRequest(req *client.Request) (map[string]interface{}, error) {
	tc.T.Helper()

	ctx := context.Background()
	resp, err := tc.Client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for successful status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Parse JSON response
	var result map[string]interface{}
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
}

// DoRequestExpectError executes an API request expecting an error response
func (tc *TestContext) DoRequestExpectError(req *client.Request, expectedStatus int) (map[string]interface{}, error) {
	tc.T.Helper()

	ctx := context.Background()
	resp, err := tc.Client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Verify expected status code
	assert.Equal(tc.T, expectedStatus, resp.StatusCode, "Expected status code mismatch")

	// Parse error response
	var result map[string]interface{}
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
	}

	return result, nil
}

// GenerateUniqueName creates a unique name for test resources
func GenerateUniqueName(prefix string) string {
	return fmt.Sprintf("%s-test-%d", prefix, time.Now().UnixNano())
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(ctx context.Context, interval, timeout time.Duration, condition func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition: %w", ctx.Err())
		case <-ticker.C:
			ok, err := condition()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
}

// AssertValidUUID checks if a string is a valid UUID
func AssertValidUUID(t *testing.T, id interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	idStr, ok := id.(string)
	require.True(t, ok, "ID must be a string")
	require.NotEmpty(t, idStr, msgAndArgs...)
	// Basic UUID format validation (relaxed for different UUID formats)
	require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, idStr, msgAndArgs...)
}

// AssertTimestamp checks if a field contains a valid timestamp
func AssertTimestamp(t *testing.T, timestamp interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	tsStr, ok := timestamp.(string)
	require.True(t, ok, "Timestamp must be a string")
	_, err := time.Parse(time.RFC3339, tsStr)
	assert.NoError(t, err, msgAndArgs...)
}

// skipIfShort skips the test if running in short mode
//
//nolint:unused // Used in integration tests which are tagged
func skipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

// TestMain validates the test environment
func TestMain(m *testing.M) {
	// Try to load .env file from project root
	// We ignore the error because the file might not exist (e.g. in CI)
	// and we want to support setting env vars directly
	_ = godotenv.Load("../../.env")

	// Check if running integration tests
	if os.Getenv("LOGS_API_KEY") == "" {
		fmt.Println("Skipping integration tests: LOGS_API_KEY not set")
		fmt.Println("To run integration tests, set the following environment variables:")
		fmt.Println("  export LOGS_API_KEY=your-api-key") // pragma: allowlist secret
		fmt.Println("  export LOGS_INSTANCE_ID=your-instance-id")
		fmt.Println("  export LOGS_REGION=your-region")
		fmt.Println("  Or create a .env file in the project root")
		os.Exit(0)
	}

	// Run tests
	exitCode := m.Run()
	os.Exit(exitCode)
}
