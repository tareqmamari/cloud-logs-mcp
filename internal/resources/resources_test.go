package resources

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/metrics"
)

var (
	testMetrics     *metrics.Metrics
	testMetricsOnce sync.Once
)

func getTestMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		reg := prometheus.NewRegistry()
		testMetrics = metrics.NewWithRegistry(zap.NewNop(), reg)
	})
	return testMetrics
}

func testConfig() *config.Config {
	return &config.Config{
		ServiceURL:            "https://api.logs.cloud.ibm.com",
		APIKey:                "test-secret-key-12345",
		Region:                "us-south",
		InstanceID:            "test-instance-123",
		InstanceName:          "test-instance",
		Timeout:               30 * time.Second,
		QueryTimeout:          60 * time.Second,
		BackgroundPollTimeout: 10 * time.Second,
		BulkOperationTimeout:  120 * time.Second,
		MaxRetries:            3,
		RateLimit:             10,
		RateLimitBurst:        20,
		EnableRateLimit:       true,
		EnableTracing:         true,
		EnableAuditLog:        true,
		LogLevel:              "info",
		LogFormat:             "json",
	}
}

func newTestRegistry() *Registry {
	return NewRegistry(testConfig(), getTestMetrics(), zap.NewNop(), "1.0.0-test")
}

func TestNewRegistry(t *testing.T) {
	r := newTestRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.config == nil {
		t.Error("expected config to be set")
	}
	if r.metrics == nil {
		t.Error("expected metrics to be set")
	}
	if r.logger == nil {
		t.Error("expected logger to be set")
	}
	if r.version != "1.0.0-test" {
		t.Errorf("expected version 1.0.0-test, got %s", r.version)
	}
}

func TestGetResources_Count(t *testing.T) {
	r := newTestRegistry()
	resources := r.GetResources()
	if len(resources) != 4 {
		t.Errorf("expected 4 resources, got %d", len(resources))
	}
}

func TestGetResources_URIs(t *testing.T) {
	r := newTestRegistry()
	resources := r.GetResources()

	expectedURIs := []string{
		"about://service",
		"config://current",
		"metrics://server",
		"health://status",
	}

	for i, expected := range expectedURIs {
		if i >= len(resources) {
			t.Fatalf("missing resource at index %d", i)
		}
		if resources[i].Resource.URI != expected {
			t.Errorf("resource[%d]: expected URI %q, got %q", i, expected, resources[i].Resource.URI)
		}
	}
}

func TestAboutResource_Handler(t *testing.T) {
	r := newTestRegistry()
	resources := r.GetResources()

	result, err := resources[0].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Contents))
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify expected top-level keys
	for _, key := range []string{"service", "query_language", "data_tiers", "mcp_server"} {
		if _, ok := data[key]; !ok {
			t.Errorf("expected key %q in about response", key)
		}
	}

	// Verify version is embedded
	mcpServer, ok := data["mcp_server"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcp_server to be a map")
	}
	if v, ok := mcpServer["version"].(string); !ok || v != "1.0.0-test" {
		t.Errorf("expected version 1.0.0-test, got %v", mcpServer["version"])
	}
}

func TestConfigResource_Handler(t *testing.T) {
	r := newTestRegistry()
	resources := r.GetResources()

	result, err := resources[1].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Contents[0].Text

	// Verify sensitive values are NOT exposed
	if strings.Contains(text, "test-secret-key-12345") {
		t.Error("config resource must not expose raw API key value")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify api_key_configured is true (key was set)
	apiKeyConfigured, ok := data["api_key_configured"].(bool)
	if !ok {
		t.Fatal("expected api_key_configured to be a bool")
	}
	if !apiKeyConfigured {
		t.Error("expected api_key_configured to be true")
	}

	// Verify other config fields are present
	for _, key := range []string{"service_url", "region", "instance_id", "max_retries", "log_level"} {
		if _, ok := data[key]; !ok {
			t.Errorf("expected key %q in config response", key)
		}
	}
}

func TestMetricsResource_Handler(t *testing.T) {
	r := newTestRegistry()
	resources := r.GetResources()

	result, err := resources[2].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify expected top-level keys
	for _, key := range []string{"requests", "latency", "tools"} {
		if _, ok := data[key]; !ok {
			t.Errorf("expected key %q in metrics response", key)
		}
	}
}

func TestHealthResource_Healthy(t *testing.T) {
	// Use a fresh metrics instance with no recorded failures
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(zap.NewNop(), reg)
	r := NewRegistry(testConfig(), m, zap.NewNop(), "1.0.0-test")

	resources := r.GetResources()
	result, err := resources[3].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	status, ok := data["status"].(string)
	if !ok || status != "healthy" {
		t.Errorf("expected status 'healthy', got %v", data["status"])
	}
}

func TestHealthResource_Degraded(t *testing.T) {
	// Error rate > 10% but <= 50%: 20 total, 5 failed = 25%
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(zap.NewNop(), reg)
	for i := 0; i < 15; i++ {
		m.RecordRequest(true, time.Millisecond, 200)
	}
	for i := 0; i < 5; i++ {
		m.RecordRequest(false, time.Millisecond, 500)
	}

	r := NewRegistry(testConfig(), m, zap.NewNop(), "1.0.0-test")
	resources := r.GetResources()
	result, err := resources[3].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	status, ok := data["status"].(string)
	if !ok || status != "degraded" {
		t.Errorf("expected status 'degraded', got %v", data["status"])
	}
}

func TestHealthResource_Unhealthy(t *testing.T) {
	// Error rate > 50%: 10 total, 8 failed = 80%
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegistry(zap.NewNop(), reg)
	for i := 0; i < 2; i++ {
		m.RecordRequest(true, time.Millisecond, 200)
	}
	for i := 0; i < 8; i++ {
		m.RecordRequest(false, time.Millisecond, 500)
	}

	r := NewRegistry(testConfig(), m, zap.NewNop(), "1.0.0-test")
	resources := r.GetResources()
	result, err := resources[3].Handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	status, ok := data["status"].(string)
	if !ok || status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %v", data["status"])
	}
}

func TestGetResourceTemplates_Count(t *testing.T) {
	r := newTestRegistry()
	templates := r.GetResourceTemplates()
	if len(templates) != 5 {
		t.Errorf("expected 5 templates, got %d", len(templates))
	}
}

func TestGetResourceTemplates_URIs(t *testing.T) {
	r := newTestRegistry()
	templates := r.GetResourceTemplates()

	expectedPrefixes := []string{"alert", "dashboard", "query", "webhook", "policy"}
	for _, prefix := range expectedPrefixes {
		found := false
		for _, tmpl := range templates {
			if strings.Contains(tmpl.URITemplate, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template URI containing %q", prefix)
		}
	}
}

func callTemplateHandler(t *testing.T, r *Registry, uri string) map[string]interface{} {
	t.Helper()
	handler := r.GetTemplateHandler()
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: uri},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error for URI %s: %v", uri, err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content for URI %s, got %d", uri, len(result.Contents))
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON for URI %s: %v", uri, err)
	}
	return data
}

func TestTemplateHandler_Alert(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://alert/test")
	if _, ok := data["alert"]; !ok {
		t.Error("expected 'alert' key in response")
	}
	if _, ok := data["_template_info"]; !ok {
		t.Error("expected '_template_info' key in response")
	}
}

func TestTemplateHandler_Dashboard(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://dashboard/test")
	if _, ok := data["layout"]; !ok {
		t.Error("expected 'layout' key in response")
	}
	if _, ok := data["_template_info"]; !ok {
		t.Error("expected '_template_info' key in response")
	}
}

func TestTemplateHandler_QueryDataPrime(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://query/dataprime")
	info, ok := data["_template_info"].(map[string]interface{})
	if !ok {
		t.Fatal("expected '_template_info' to be a map")
	}
	if syntax, ok := info["syntax"].(string); !ok || syntax != "dataprime" {
		t.Errorf("expected syntax 'dataprime', got %v", info["syntax"])
	}
	if _, ok := data["examples"]; !ok {
		t.Error("expected 'examples' key in response")
	}
}

func TestTemplateHandler_QueryLucene(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://query/lucene")
	info, ok := data["_template_info"].(map[string]interface{})
	if !ok {
		t.Fatal("expected '_template_info' to be a map")
	}
	if syntax, ok := info["syntax"].(string); !ok || syntax != "lucene" {
		t.Errorf("expected syntax 'lucene', got %v", info["syntax"])
	}
	if _, ok := data["operators"]; !ok {
		t.Error("expected 'operators' key in response")
	}
}

func TestTemplateHandler_WebhookSlack(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://webhook/slack")
	webhook, ok := data["webhook"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'webhook' to be a map")
	}
	if wt, ok := webhook["type"].(string); !ok || wt != "slack" {
		t.Errorf("expected webhook type 'slack', got %v", webhook["type"])
	}
}

func TestTemplateHandler_WebhookPagerDuty(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://webhook/pagerduty")
	webhook, ok := data["webhook"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'webhook' to be a map")
	}
	if wt, ok := webhook["type"].(string); !ok || wt != "pagerduty" {
		t.Errorf("expected webhook type 'pagerduty', got %v", webhook["type"])
	}
}

func TestTemplateHandler_WebhookGeneric(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://webhook/generic")
	webhook, ok := data["webhook"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'webhook' to be a map")
	}
	if wt, ok := webhook["type"].(string); !ok || wt != "generic" {
		t.Errorf("expected webhook type 'generic', got %v", webhook["type"])
	}
}

func TestTemplateHandler_PolicyLogs(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://policy/logs")
	info, ok := data["_template_info"].(map[string]interface{})
	if !ok {
		t.Fatal("expected '_template_info' to be a map")
	}
	if pt, ok := info["type"].(string); !ok || pt != "logs" {
		t.Errorf("expected policy type 'logs', got %v", info["type"])
	}
	if _, ok := data["_tco_tiers"]; !ok {
		t.Error("expected '_tco_tiers' key in logs policy response")
	}
}

func TestTemplateHandler_PolicySpans(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://policy/spans")
	info, ok := data["_template_info"].(map[string]interface{})
	if !ok {
		t.Fatal("expected '_template_info' to be a map")
	}
	if pt, ok := info["type"].(string); !ok || pt != "spans" {
		t.Errorf("expected policy type 'spans', got %v", info["type"])
	}
}

func TestTemplateHandler_Unknown(t *testing.T) {
	r := newTestRegistry()
	data := callTemplateHandler(t, r, "template://unknown/foo")
	if _, ok := data["error"]; !ok {
		t.Error("expected 'error' key for unknown template")
	}
	if _, ok := data["available_templates"]; !ok {
		t.Error("expected 'available_templates' key for unknown template")
	}
}

func TestMatchTemplate(t *testing.T) {
	tests := []struct {
		uri    string
		prefix string
		want   bool
	}{
		{"template://alert/test", "template://alert/", true},
		{"template://alert/", "template://alert/", false}, // no name after prefix
		{"template://dashboard/myboard", "template://dashboard/", true},
		{"other://something", "template://alert/", false},
		{"", "template://alert/", false},
	}

	for _, tt := range tests {
		got := matchTemplate(tt.uri, tt.prefix)
		if got != tt.want {
			t.Errorf("matchTemplate(%q, %q) = %v, want %v", tt.uri, tt.prefix, got, tt.want)
		}
	}
}

func TestExtractTemplateName(t *testing.T) {
	tests := []struct {
		uri    string
		prefix string
		want   string
	}{
		{"template://alert/my-alert", "template://alert/", "my-alert"},
		{"template://query/dataprime", "template://query/", "dataprime"},
		{"template://webhook/slack", "template://webhook/", "slack"},
	}

	for _, tt := range tests {
		got := extractTemplateName(tt.uri, tt.prefix)
		if got != tt.want {
			t.Errorf("extractTemplateName(%q, %q) = %q, want %q", tt.uri, tt.prefix, got, tt.want)
		}
	}
}

func TestFormatToolLatency(t *testing.T) {
	input := map[string]time.Duration{
		"query_logs":   150 * time.Millisecond,
		"create_alert": 2 * time.Second,
		"list_alerts":  500 * time.Microsecond,
	}

	result := formatToolLatency(input)

	expected := map[string]int64{
		"query_logs":   150,
		"create_alert": 2000,
		"list_alerts":  0, // 500 microseconds = 0ms
	}

	for key, want := range expected {
		got, ok := result[key]
		if !ok {
			t.Errorf("missing key %q in result", key)
			continue
		}
		if got != want {
			t.Errorf("formatToolLatency[%q] = %d, want %d", key, got, want)
		}
	}

	if len(result) != len(expected) {
		t.Errorf("expected %d entries, got %d", len(expected), len(result))
	}
}
