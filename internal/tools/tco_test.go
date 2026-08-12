package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// failingDoer is a client.Doer whose Do always fails with an error carrying
// a classified snippet (as a real classifyResponseError from a non-2xx API
// response would), so tests can assert the failure never reaches a log sink
// unmasked.
type failingDoer struct {
	err error
}

func (f *failingDoer) Do(_ context.Context, _ *client.Request) (*client.Response, error) {
	return nil, f.err
}
func (f *failingDoer) GetInstanceInfo() client.InstanceInfo { return client.InstanceInfo{} }
func (f *failingDoer) Close() error                         { return nil }

// TestFetchAndCacheTCOConfig_SanitizesStartupLogError guards against startup
// logs bypassing masking: a TCO policy-fetch failure can carry a classified
// response-body snippet (e.g. an echoed api_key), and FetchAndCacheTCOConfig
// must run that error through security.SanitizeError before it reaches the
// zap sink, exactly like the tool-execution chokepoint does.
func TestFetchAndCacheTCOConfig_SanitizesStartupLogError(t *testing.T) {
	const secret = "api_key=abcdefghij1234567890secretvalue" // pragma: allowlist secret
	failingClient := &failingDoer{err: &testAPIError{msg: "policies fetch failed: " + secret}}

	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	session := NewSessionContext("test-user", "test-instance")
	ctx := WithSession(context.Background(), session)

	if err := FetchAndCacheTCOConfig(ctx, failingClient, logger); err != nil {
		t.Fatalf("FetchAndCacheTCOConfig returned unexpected error: %v", err)
	}

	entries := logs.All()
	found := false
	for _, entry := range entries {
		for _, field := range entry.Context {
			if field.Key != "error" {
				continue
			}
			found = true
			if strings.Contains(field.String, "abcdefghij1234567890secretvalue") { // pragma: allowlist secret
				t.Errorf("logged error field leaks the raw secret: %q", field.String)
			}
			if !strings.Contains(field.String, "REDACTED") {
				t.Errorf("logged error field does not look sanitized: %q", field.String)
			}
		}
	}
	if !found {
		t.Fatal("expected a log entry with an \"error\" string field, found none")
	}
}

// testAPIError is a minimal error type used to simulate a classified
// response error without depending on internal/errors' constructors.
type testAPIError struct{ msg string }

func (e *testAPIError) Error() string { return e.msg }

// TestDetermineTier pins the IBM Cloud Logs TCO priority-to-QUERY-tier
// mapping. type_high fans out to both frequent_search and archive; type_medium
// and type_low both land in archive for querying. This is only about where to
// read logs for queries — it is NOT alert availability: high and medium logs
// are alertable, low logs are not (that rule keys off priority, not this
// tier; see determineTier's doc and GetPriorityForAppAndSubsystem). type_low
// is IBM's low-priority store-and-search tier, not a "blocked/dropped"
// pipeline outcome.
func TestDetermineTier(t *testing.T) {
	tests := []struct {
		priority         string
		wantTier         string
		wantFrequentFlag bool
	}{
		{priority: "type_high", wantTier: "frequent_search", wantFrequentFlag: true},
		{priority: "type_medium", wantTier: "archive", wantFrequentFlag: false},
		{priority: "type_low", wantTier: "archive", wantFrequentFlag: false},
		{priority: "type_unspecified", wantTier: "archive", wantFrequentFlag: false},
		{priority: "", wantTier: "archive", wantFrequentFlag: false},
		{priority: "some_unknown_future_priority", wantTier: "archive", wantFrequentFlag: false},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			config := &TCOConfig{}
			got := determineTier(config, tt.priority)
			if got != tt.wantTier {
				t.Errorf("determineTier(%q) = %q, want %q", tt.priority, got, tt.wantTier)
			}
			if config.HasFrequentSearch != tt.wantFrequentFlag {
				t.Errorf("determineTier(%q): HasFrequentSearch = %v, want %v", tt.priority, config.HasFrequentSearch, tt.wantFrequentFlag)
			}
		})
	}
}

func TestTCOConfig_Defaults(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")

	// Without TCO config, should return frequent_search as default (faster queries)
	if tier := session.GetDefaultTier(); tier != "frequent_search" {
		t.Errorf("Expected default tier 'frequent_search' without TCO config, got '%s'", tier)
	}

	// TCO config should be stale initially
	if !session.IsTCOConfigStale() {
		t.Error("Expected TCO config to be stale when not set")
	}
}

func TestTCOConfig_WithPolicies(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")

	// Set TCO config with policies using new structure
	config := &TCOConfig{
		HasPolicies:       true,
		HasArchive:        true,
		HasFrequentSearch: true,
		DefaultTier:       "frequent_search", // Prefer frequent_search when available (faster queries)
		PolicyCount:       3,
		LastUpdated:       time.Now(),
		Policies: []TCOPolicyRule{
			{
				ApplicationRule: &TCOMatchRule{Name: "api-gateway", RuleType: "is"},
				Tier:            "frequent_search",
			},
			{
				ApplicationRule: &TCOMatchRule{Name: "batch-worker", RuleType: "is"},
				Tier:            "archive",
			},
			{
				ApplicationRule: &TCOMatchRule{Name: "production", RuleType: "starts_with"},
				Tier:            "frequent_search",
			},
		},
	}
	session.SetTCOConfig(config)

	// Check default tier - frequent_search preferred when available
	if tier := session.GetDefaultTier(); tier != "frequent_search" {
		t.Errorf("Expected default tier 'frequent_search', got '%s'", tier)
	}

	// Check application-specific tier (exact match)
	if tier := session.GetTierForApplication("api-gateway"); tier != "frequent_search" {
		t.Errorf("Expected tier 'frequent_search' for api-gateway, got '%s'", tier)
	}

	if tier := session.GetTierForApplication("batch-worker"); tier != "archive" {
		t.Errorf("Expected tier 'archive' for batch-worker, got '%s'", tier)
	}

	// Check starts_with match
	if tier := session.GetTierForApplication("production-api"); tier != "frequent_search" {
		t.Errorf("Expected tier 'frequent_search' for production-api (starts_with match), got '%s'", tier)
	}

	// Unknown application should fall back to default (frequent_search)
	if tier := session.GetTierForApplication("unknown-app"); tier != "frequent_search" {
		t.Errorf("Expected tier 'frequent_search' for unknown app, got '%s'", tier)
	}

	// TCO config should not be stale
	if session.IsTCOConfigStale() {
		t.Error("Expected TCO config to not be stale after setting")
	}
}

func TestTCOConfig_Staleness(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")

	// Set TCO config with old timestamp
	config := &TCOConfig{
		HasPolicies: false,
		DefaultTier: "archive",
		LastUpdated: time.Now().Add(-2 * time.Hour), // 2 hours ago
	}
	session.SetTCOConfig(config)

	// Should be stale (older than 1 hour)
	if !session.IsTCOConfigStale() {
		t.Error("Expected TCO config to be stale after 2 hours")
	}
}

func TestParseTCOPolicies_NoPolicies(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{},
	}

	config := parseTCOPolicies(result, nil)

	if config.HasPolicies {
		t.Error("Expected HasPolicies to be false for empty policies")
	}
	// When no policies exist, logs go to both tiers - use frequent_search for faster queries
	if config.DefaultTier != "frequent_search" {
		t.Errorf("Expected default tier 'frequent_search', got '%s'", config.DefaultTier)
	}
	if !config.HasArchive {
		t.Error("Expected HasArchive to be true by default")
	}
	if !config.HasFrequentSearch {
		t.Error("Expected HasFrequentSearch to be true when no policies (logs go to both tiers)")
	}
}

func TestParseTCOPolicies_WithHighPriorityPolicy(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "Production Logs",
				"priority": "type_high",
				"application_rule": map[string]interface{}{
					"name":         "production-api",
					"rule_type_id": "is",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	if !config.HasPolicies {
		t.Error("Expected HasPolicies to be true")
	}
	if config.PolicyCount != 1 {
		t.Errorf("Expected PolicyCount to be 1, got %d", config.PolicyCount)
	}
	if !config.HasFrequentSearch {
		t.Error("Expected HasFrequentSearch to be true for high priority policy")
	}

	// Check policy rule
	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy rule, got %d", len(config.Policies))
		return
	}
	if config.Policies[0].Tier != "frequent_search" {
		t.Errorf("Expected policy tier 'frequent_search', got '%s'", config.Policies[0].Tier)
	}
	if config.Policies[0].ApplicationRule == nil || config.Policies[0].ApplicationRule.Name != "production-api" {
		t.Error("Expected application rule for production-api")
	}
}

func TestParseTCOPolicies_WithMediumPriorityPolicy(t *testing.T) {
	// type_medium means logs go to archive ONLY, not frequent_search
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "Standard Logs",
				"priority": "type_medium",
				"application_rule": map[string]interface{}{
					"name":         "standard-service",
					"rule_type_id": "is",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	if !config.HasPolicies {
		t.Error("Expected HasPolicies to be true")
	}
	// Medium priority does NOT go to frequent_search
	if config.HasFrequentSearch {
		t.Error("Expected HasFrequentSearch to be false for medium priority policy")
	}
	if config.DefaultTier != "archive" {
		t.Errorf("Expected default tier 'archive', got '%s'", config.DefaultTier)
	}

	// Check policy rule
	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy rule, got %d", len(config.Policies))
		return
	}
	if config.Policies[0].Tier != "archive" {
		t.Errorf("Expected policy tier 'archive' for medium priority, got '%s'", config.Policies[0].Tier)
	}
}

func TestParseTCOPolicies_WithLowPriorityPolicy(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "Debug Logs",
				"priority": "type_low",
				"application_rule": map[string]interface{}{
					"name":         "debug-service",
					"rule_type_id": "is",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	if !config.HasPolicies {
		t.Error("Expected HasPolicies to be true")
	}
	if config.HasFrequentSearch {
		t.Error("Expected HasFrequentSearch to be false for low priority policy")
	}
	if config.DefaultTier != "archive" {
		t.Errorf("Expected default tier 'archive', got '%s'", config.DefaultTier)
	}

	// Check policy rule
	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy rule, got %d", len(config.Policies))
		return
	}
	if config.Policies[0].Tier != "archive" {
		t.Errorf("Expected policy tier 'archive', got '%s'", config.Policies[0].Tier)
	}
}

func TestParseTCOPolicies_DisabledPolicy(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "Disabled Policy",
				"priority": "type_high",
				"enabled":  false, // Disabled!
				"application_rule": map[string]interface{}{
					"name":         "disabled-app",
					"rule_type_id": "is",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	// Policy count still includes disabled, but tier routing should not
	if config.HasFrequentSearch {
		t.Error("Expected HasFrequentSearch to be false for disabled policy")
	}

	// Disabled policy should not create policy rule
	if len(config.Policies) != 0 {
		t.Errorf("Expected 0 policy rules for disabled policy, got %d", len(config.Policies))
	}
}

func TestParseTCOPolicies_StartsWithRule(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "Production Prefix",
				"priority": "type_high",
				"application_rule": map[string]interface{}{
					"name":         "prod",
					"rule_type_id": "starts_with",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	// Check policy rule with starts_with
	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy rule, got %d", len(config.Policies))
		return
	}
	if config.Policies[0].ApplicationRule == nil {
		t.Error("Expected application rule")
		return
	}
	if config.Policies[0].ApplicationRule.RuleType != "starts_with" {
		t.Errorf("Expected rule type 'starts_with', got '%s'", config.Policies[0].ApplicationRule.RuleType)
	}
	if config.Policies[0].ApplicationRule.Name != "prod" {
		t.Errorf("Expected rule name 'prod', got '%s'", config.Policies[0].ApplicationRule.Name)
	}
	if config.Policies[0].Tier != "frequent_search" {
		t.Errorf("Expected tier 'frequent_search', got '%s'", config.Policies[0].Tier)
	}
}

func TestParseTCOPolicies_WithSubsystemRule(t *testing.T) {
	result := map[string]interface{}{
		"policies": []interface{}{
			map[string]interface{}{
				"name":     "API Gateway Logs",
				"priority": "type_high",
				"application_rule": map[string]interface{}{
					"name":         "api-gateway",
					"rule_type_id": "is",
				},
				"subsystem_rule": map[string]interface{}{
					"name":         "auth",
					"rule_type_id": "is",
				},
			},
		},
	}

	config := parseTCOPolicies(result, nil)

	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy rule, got %d", len(config.Policies))
		return
	}

	policy := config.Policies[0]
	if policy.ApplicationRule == nil || policy.ApplicationRule.Name != "api-gateway" {
		t.Error("Expected application rule for api-gateway")
	}
	if policy.SubsystemRule == nil || policy.SubsystemRule.Name != "auth" {
		t.Error("Expected subsystem rule for auth")
	}
	if policy.Tier != "frequent_search" {
		t.Errorf("Expected tier 'frequent_search', got '%s'", policy.Tier)
	}
}

func TestTCOPolicyMatching(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")
	config := &TCOConfig{
		HasPolicies:       true,
		HasArchive:        true,
		HasFrequentSearch: true,
		DefaultTier:       "frequent_search", // Prefer frequent_search when available
		PolicyCount:       3,
		LastUpdated:       time.Now(),
		Policies: []TCOPolicyRule{
			{
				// Match api-gateway + auth subsystem -> frequent_search
				ApplicationRule: &TCOMatchRule{Name: "api-gateway", RuleType: "is"},
				SubsystemRule:   &TCOMatchRule{Name: "auth", RuleType: "is"},
				Tier:            "frequent_search",
			},
			{
				// Match api-gateway (any subsystem) -> archive
				ApplicationRule: &TCOMatchRule{Name: "api-gateway", RuleType: "is"},
				Tier:            "archive",
			},
			{
				// Match production-* -> frequent_search
				ApplicationRule: &TCOMatchRule{Name: "production", RuleType: "starts_with"},
				Tier:            "frequent_search",
			},
		},
	}
	session.SetTCOConfig(config)

	// Test app + subsystem match (first rule)
	if tier := session.GetTierForAppAndSubsystem("api-gateway", "auth"); tier != "frequent_search" {
		t.Errorf("Expected 'frequent_search' for api-gateway+auth, got '%s'", tier)
	}

	// Test app match only (second rule - first rule doesn't match due to subsystem)
	if tier := session.GetTierForAppAndSubsystem("api-gateway", "logging"); tier != "archive" {
		t.Errorf("Expected 'archive' for api-gateway+logging, got '%s'", tier)
	}

	// Test starts_with match
	if tier := session.GetTierForApplication("production-api"); tier != "frequent_search" {
		t.Errorf("Expected 'frequent_search' for production-api, got '%s'", tier)
	}

	// Test no match - fallback to default (frequent_search)
	if tier := session.GetTierForApplication("unknown-service"); tier != "frequent_search" {
		t.Errorf("Expected 'frequent_search' for unknown-service (default), got '%s'", tier)
	}
}

func TestGetTCOSummary_NoConfig(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")
	summary := GetTCOSummary(session)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !tcoContains(summary, "not loaded") {
		t.Error("Expected summary to mention TCO not loaded")
	}
}

func TestGetTCOSummary_WithConfig(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")
	config := &TCOConfig{
		HasPolicies:       true,
		HasArchive:        true,
		HasFrequentSearch: true,
		DefaultTier:       "archive",
		PolicyCount:       2,
		LastUpdated:       time.Now(),
	}
	session.SetTCOConfig(config)

	summary := GetTCOSummary(session)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !tcoContains(summary, "Archive tier: enabled") {
		t.Error("Expected summary to mention archive tier")
	}
	if !tcoContains(summary, "Frequent search tier: enabled") {
		t.Error("Expected summary to mention frequent search tier")
	}
}

// Helper function for string contains check
func tcoContains(s, substr string) bool {
	return len(s) >= len(substr) && tcoContainsHelper(s, substr)
}

func tcoContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
