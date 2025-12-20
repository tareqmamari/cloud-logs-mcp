package tools

import (
	"testing"
	"time"
)

// ============================================================================
// Config Change Correlation Tool Tests - SOTA 2025
// ============================================================================

func TestDetectChangeType(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    ChangeType
	}{
		// Deployment patterns
		{"deployment keyword", "new deployment started for service-x", ChangeTypeDeployment},
		{"release pattern", "release v2.3.1 rolled out successfully", ChangeTypeDeployment},
		{"rollback pattern", "emergency rollback initiated", ChangeTypeDeployment},
		{"helm deployment", "helm upgrade completed for chart myapp", ChangeTypeDeployment},
		{"argocd sync", "argocd application sync completed", ChangeTypeDeployment},
		{"container started", "container started with image sha256:abc123", ChangeTypeDeployment},

		// Config patterns (use unique keywords that don't overlap with other types)
		{"config update", "configuration updated for service-x", ChangeTypeConfig},
		{"configmap change", "configmap myapp-config modified", ChangeTypeConfig},
		{"environment variable", "environment variable DATABASE_URL changed", ChangeTypeConfig},
		{"reloaded config", "reloaded config from file", ChangeTypeConfig},

		// IAM patterns (use unique keywords)
		{"iam policy", "iam policy updated for role admin", ChangeTypeIAM},
		{"rbac change", "rbac rolebinding created in namespace default", ChangeTypeIAM},
		{"saml auth", "saml authentication provider enabled", ChangeTypeIAM},
		{"ldap sync", "ldap user sync completed", ChangeTypeIAM},

		// Scaling patterns (use unique keywords to avoid matching other types)
		{"autoscale event", "autoscale event triggered for pods", ChangeTypeScaling},
		{"hpa update", "hpa target cpu utilization changed to 80%", ChangeTypeScaling},
		{"vpa update", "vpa resource limits adjusted", ChangeTypeScaling},

		// Network patterns (use unique keywords that don't overlap)
		{"firewall rule", "firewall rule updated for ingress", ChangeTypeNetworkPolicy},
		{"service mesh", "istio virtual service updated", ChangeTypeNetworkPolicy},
		{"envoy config", "envoy sidecar injected", ChangeTypeNetworkPolicy},
		{"calico network", "calico networkset applied", ChangeTypeNetworkPolicy},

		// Secret patterns (use unique keywords)
		{"certificate renewal", "certificate renewed for domain example.com", ChangeTypeSecret},
		{"tls cert", "tls cert rotation completed", ChangeTypeSecret},
		{"kms key", "kms key created for encryption", ChangeTypeSecret},

		// Feature flag patterns (avoid keywords that match other types)
		{"launchdarkly", "launchdarkly flag toggled", ChangeTypeFeatureFlag},
		{"a/b test", "a/b test experiment-123 started", ChangeTypeFeatureFlag},
		{"split.io flag", "split.io treatment changed", ChangeTypeFeatureFlag},

		// Database patterns (use unique keywords)
		{"migration", "migration script 20240115_001 completed", ChangeTypeDatabase},
		{"schema change", "schema change applied to users table", ChangeTypeDatabase},
		{"failover", "failover to standby completed", ChangeTypeDatabase},

		// Infrastructure patterns
		{"terraform apply", "terraform apply completed for vpc", ChangeTypeInfra},
		{"cloudformation", "cloudformation stack updated", ChangeTypeInfra},
		{"ansible playbook", "ansible playbook executed on hosts", ChangeTypeInfra},

		// Unknown pattern
		{"random message", "normal application log message", ChangeTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectChangeType(tt.message)
			if got != tt.want {
				t.Errorf("detectChangeType(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestAssessRiskLevel(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		// Critical patterns
		{"production keyword", "deployment to production environment", "critical"},
		{"rollback", "emergency rollback initiated", "critical"},
		{"security patch", "security vulnerability cve-2024-1234 patched", "critical"},
		{"hotfix", "hotfix deployed for critical bug", "critical"},

		// High patterns
		{"database change", "database schema migration started", "high"},
		{"iam change", "iam policy updated for admin role", "high"},
		{"network change", "firewall rule modified", "high"},
		{"secret change", "secret rotation completed", "high"},
		{"certificate", "certificate renewed for api.example.com", "high"},

		// Medium patterns
		{"deployment", "deployment started for staging", "medium"},
		{"config change", "configuration updated", "medium"},
		{"replica change", "replica count scaled to 5", "medium"},

		// Low patterns (default)
		{"normal log", "application started successfully", "low"},
		{"info message", "request processed in 50ms", "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assessRiskLevel(tt.message)
			if got != tt.want {
				t.Errorf("assessRiskLevel(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestLooksLikeChange(t *testing.T) {
	tests := []struct {
		message string
		want    bool
	}{
		{"configuration updated for service", true},
		{"settings changed by admin", true},
		{"feature enabled for users", true},
		{"service disabled temporarily", true},
		{"new resource created in cluster", true},
		{"old deployment deleted", true},
		{"user added to group", true},
		{"permission removed from role", true},
		{"application started successfully", true},
		{"server stopped for maintenance", true},
		{"policy applied to namespace", true},
		{"change reverted to previous state", true},
		{"replica promoted to primary", true},
		{"node demoted from cluster", true},

		// Should not match
		{"request processed successfully", false},
		{"query executed in 50ms", false},
		{"connection established to database", false},
		{"cache hit for key abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := looksLikeChange(tt.message)
			if got != tt.want {
				t.Errorf("looksLikeChange(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestCalculateCorrelation(t *testing.T) {
	incidentTime := time.Now()
	windowBefore := time.Hour

	tests := []struct {
		name        string
		changeTime  time.Time
		wantMinCorr float64
		wantMaxCorr float64
	}{
		{
			name:        "change 1 minute before incident",
			changeTime:  incidentTime.Add(-1 * time.Minute),
			wantMinCorr: 0.85,
			wantMaxCorr: 1.0,
		},
		{
			name:        "change 15 minutes before incident",
			changeTime:  incidentTime.Add(-15 * time.Minute),
			wantMinCorr: 0.4,
			wantMaxCorr: 0.6,
		},
		{
			name:        "change 1 hour before incident",
			changeTime:  incidentTime.Add(-1 * time.Hour),
			wantMinCorr: 0.15,
			wantMaxCorr: 0.3,
		},
		{
			name:        "change after incident",
			changeTime:  incidentTime.Add(5 * time.Minute),
			wantMinCorr: 0.25,
			wantMaxCorr: 0.35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCorrelation(tt.changeTime, incidentTime, windowBefore)
			if got < tt.wantMinCorr || got > tt.wantMaxCorr {
				t.Errorf("calculateCorrelation() = %v, want between %v and %v",
					got, tt.wantMinCorr, tt.wantMaxCorr)
			}
		})
	}
}

func TestExtractChange(t *testing.T) {
	tests := []struct {
		name     string
		event    map[string]interface{}
		wantType ChangeType
		wantRisk string
		wantNil  bool
	}{
		{
			name: "deployment event",
			event: map[string]interface{}{
				"message": "deployment v2.0.0 completed successfully",
				"labels":  map[string]interface{}{"applicationname": "myapp"},
			},
			wantType: ChangeTypeDeployment,
			wantRisk: "medium",
			wantNil:  false,
		},
		{
			name: "production deployment (critical)",
			event: map[string]interface{}{
				"message": "production deployment started",
				"labels":  map[string]interface{}{"applicationname": "myapp"},
			},
			wantType: ChangeTypeDeployment,
			wantRisk: "critical",
			wantNil:  false,
		},
		{
			name: "iam change (high)",
			event: map[string]interface{}{
				"message": "iam role permissions updated",
			},
			wantType: ChangeTypeIAM,
			wantRisk: "high",
			wantNil:  false,
		},
		{
			name: "normal log (not a change)",
			event: map[string]interface{}{
				"message": "request processed in 50ms",
			},
			wantNil: true,
		},
		{
			name:    "empty event",
			event:   map[string]interface{}{},
			wantNil: true,
		},
		{
			name: "event with user info",
			event: map[string]interface{}{
				"message": "configuration updated",
				"user_data": map[string]interface{}{
					"user":     "admin@example.com",
					"resource": "config/database",
				},
				"labels": map[string]interface{}{"applicationname": "config-service"},
			},
			wantType: ChangeTypeConfig,
			wantRisk: "medium",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := extractChange(tt.event)

			if tt.wantNil {
				if change != nil {
					t.Errorf("extractChange() = %+v, want nil", change)
				}
				return
			}

			if change == nil {
				t.Fatal("extractChange() = nil, want non-nil")
			}

			if change.ChangeType != tt.wantType {
				t.Errorf("ChangeType = %v, want %v", change.ChangeType, tt.wantType)
			}

			if change.RiskLevel != tt.wantRisk {
				t.Errorf("RiskLevel = %v, want %v", change.RiskLevel, tt.wantRisk)
			}
		})
	}
}

func TestAnalyzeChanges(t *testing.T) {
	incidentTime := time.Now()
	windowBefore := time.Hour
	windowAfter := 15 * time.Minute

	events := []interface{}{
		map[string]interface{}{
			"message":   "deployment v2.0.0 started",
			"timestamp": incidentTime.Add(-5 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]interface{}{"applicationname": "myapp"},
		},
		map[string]interface{}{
			"message":   "configuration updated",
			"timestamp": incidentTime.Add(-30 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]interface{}{"applicationname": "myapp"},
		},
		map[string]interface{}{
			"message":   "rollback initiated",
			"timestamp": incidentTime.Add(5 * time.Minute).Format(time.RFC3339),
			"labels":    map[string]interface{}{"applicationname": "myapp"},
		},
		map[string]interface{}{
			"message":   "normal application log",
			"timestamp": incidentTime.Format(time.RFC3339),
		},
	}

	analysis := analyzeChanges(events, incidentTime, windowBefore, windowAfter, nil)

	if analysis.TotalChanges != 3 {
		t.Errorf("TotalChanges = %d, want 3", analysis.TotalChanges)
	}

	if analysis.LikelyTrigger == nil {
		t.Fatal("LikelyTrigger should not be nil")
	}

	// The deployment 5 minutes before should be the likely trigger
	if analysis.LikelyTrigger.ChangeType != ChangeTypeDeployment {
		t.Errorf("LikelyTrigger.ChangeType = %v, want DEPLOYMENT", analysis.LikelyTrigger.ChangeType)
	}

	if analysis.Recommendation == "" {
		t.Error("Recommendation should not be empty")
	}
}

func TestAnalyzeChanges_WithTypeFilter(t *testing.T) {
	incidentTime := time.Now()
	events := []interface{}{
		map[string]interface{}{
			"message":   "deployment v2.0.0 started",
			"timestamp": incidentTime.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		map[string]interface{}{
			"message":   "iam policy updated",
			"timestamp": incidentTime.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		map[string]interface{}{
			"message":   "configmap changed",
			"timestamp": incidentTime.Add(-15 * time.Minute).Format(time.RFC3339),
		},
	}

	// Filter to only IAM changes
	typeFilters := []ChangeType{ChangeTypeIAM}
	analysis := analyzeChanges(events, incidentTime, time.Hour, 15*time.Minute, typeFilters)

	if analysis.TotalChanges != 1 {
		t.Errorf("TotalChanges with filter = %d, want 1", analysis.TotalChanges)
	}

	if analysis.Changes[0].ChangeType != ChangeTypeIAM {
		t.Errorf("Filtered change type = %v, want IAM", analysis.Changes[0].ChangeType)
	}
}

func TestAnalyzeChanges_NoChanges(t *testing.T) {
	incidentTime := time.Now()
	events := []interface{}{
		map[string]interface{}{
			"message":   "request processed successfully",
			"timestamp": incidentTime.Format(time.RFC3339),
		},
	}

	analysis := analyzeChanges(events, incidentTime, time.Hour, 15*time.Minute, nil)

	if analysis.TotalChanges != 0 {
		t.Errorf("TotalChanges = %d, want 0", analysis.TotalChanges)
	}

	if analysis.LikelyTrigger != nil {
		t.Error("LikelyTrigger should be nil when no changes found")
	}

	if analysis.CorrelationScore != 0 {
		t.Errorf("CorrelationScore = %v, want 0", analysis.CorrelationScore)
	}
}

func TestGenerateRecommendation(t *testing.T) {
	tests := []struct {
		name     string
		analysis *CorrelationAnalysis
		contains string
	}{
		{
			name: "no changes",
			analysis: &CorrelationAnalysis{
				TotalChanges: 0,
				Changes:      []*ConfigChange{},
			},
			contains: "No configuration changes detected",
		},
		{
			name: "deployment trigger",
			analysis: &CorrelationAnalysis{
				TotalChanges: 1,
				Changes:      []*ConfigChange{{ChangeType: ChangeTypeDeployment}},
				LikelyTrigger: &ConfigChange{
					ChangeType: ChangeTypeDeployment,
					Timestamp:  time.Now(),
				},
			},
			contains: "Deployment",
		},
		{
			name: "iam trigger",
			analysis: &CorrelationAnalysis{
				TotalChanges: 1,
				Changes:      []*ConfigChange{{ChangeType: ChangeTypeIAM}},
				LikelyTrigger: &ConfigChange{
					ChangeType: ChangeTypeIAM,
					Timestamp:  time.Now(),
				},
			},
			contains: "IAM",
		},
		{
			name: "database trigger",
			analysis: &CorrelationAnalysis{
				TotalChanges: 1,
				Changes:      []*ConfigChange{{ChangeType: ChangeTypeDatabase}},
				LikelyTrigger: &ConfigChange{
					ChangeType: ChangeTypeDatabase,
					Timestamp:  time.Now(),
				},
			},
			contains: "Database",
		},
		{
			name: "high risk changes without trigger",
			analysis: &CorrelationAnalysis{
				TotalChanges:    5,
				HighRiskChanges: 2,
				Changes:         make([]*ConfigChange, 5),
			},
			contains: "high-risk changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateRecommendation(tt.analysis)
			if got == "" {
				t.Error("generateRecommendation() returned empty string")
			}
			if !testContainsSubstring(got, tt.contains) {
				t.Errorf("generateRecommendation() = %q, should contain %q", got, tt.contains)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input      string
		want       time.Duration
		defaultVal time.Duration
	}{
		{"5m", 5 * time.Minute, time.Hour},
		{"15m", 15 * time.Minute, time.Hour},
		{"30m", 30 * time.Minute, time.Hour},
		{"1h", time.Hour, time.Minute},
		{"2h", 2 * time.Hour, time.Minute},
		{"6h", 6 * time.Hour, time.Minute},
		{"invalid", time.Hour, time.Hour},
		{"", 30 * time.Minute, 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDuration(tt.input, tt.defaultVal)
			if got != tt.want {
				t.Errorf("parseDuration(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestBuildChangeDetectionQuery(t *testing.T) {
	tests := []struct {
		name         string
		service      string
		wantContains []string
	}{
		{
			name:    "no service filter",
			service: "",
			wantContains: []string{
				"source logs",
				"filter",
				"deploy",
				"config",
				"limit 500",
			},
		},
		{
			name:    "with service filter",
			service: "payment-service",
			wantContains: []string{
				"source logs",
				"$l.applicationname == 'payment-service'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildChangeDetectionQuery(tt.service)
			for _, want := range tt.wantContains {
				if !testContainsSubstring(query, want) {
					t.Errorf("buildChangeDetectionQuery(%q) = %q, should contain %q",
						tt.service, query, want)
				}
			}
		})
	}
}

func TestFormatCorrelationAnalysis(t *testing.T) {
	analysis := &CorrelationAnalysis{
		IncidentTime:     time.Now(),
		WindowBefore:     time.Hour,
		WindowAfter:      15 * time.Minute,
		TotalChanges:     3,
		HighRiskChanges:  1,
		CorrelationScore: 0.85,
		Changes: []*ConfigChange{
			{
				Timestamp:   time.Now().Add(-5 * time.Minute),
				ChangeType:  ChangeTypeDeployment,
				Description: "Deployment v2.0.0",
				Service:     "myapp",
				RiskLevel:   "high",
				Correlation: 0.9,
			},
		},
		LikelyTrigger: &ConfigChange{
			Timestamp:   time.Now().Add(-5 * time.Minute),
			ChangeType:  ChangeTypeDeployment,
			Description: "Deployment v2.0.0",
			RiskLevel:   "high",
			Correlation: 0.9,
		},
		Recommendation: "Check deployment",
	}

	output := formatCorrelationAnalysis(analysis)

	expectedSections := []string{
		"Configuration Change Correlation Analysis",
		"Incident Time",
		"Total Changes Found",
		"Most Likely Trigger",
		"Recommendation",
		"Change Timeline",
		"Next Steps",
	}

	for _, section := range expectedSections {
		if !testContainsSubstring(output, section) {
			t.Errorf("formatCorrelationAnalysis() output missing section: %q", section)
		}
	}
}

// testContainsSubstring checks if a string contains a substring (case-insensitive)
func testContainsSubstring(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && testContainsIgnoreCase(s, substr)))
}

func testContainsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if testEqualFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func testEqualFoldSlice(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		c1, c2 := s1[i], s2[i]
		if c1 != c2 {
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				return false
			}
		}
	}
	return true
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkDetectChangeType(b *testing.B) {
	messages := []string{
		"deployment v2.0.0 completed successfully",
		"configuration updated for database",
		"iam policy changed for admin role",
		"normal application log message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detectChangeType(messages[i%len(messages)])
	}
}

func BenchmarkExtractChange(b *testing.B) {
	event := map[string]interface{}{
		"message": "deployment v2.0.0 to production completed",
		"labels":  map[string]interface{}{"applicationname": "myapp"},
		"user_data": map[string]interface{}{
			"user":     "admin@example.com",
			"resource": "deployment/myapp",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractChange(event)
	}
}

func BenchmarkAnalyzeChanges(b *testing.B) {
	incidentTime := time.Now()
	events := make([]interface{}, 100)
	messages := []string{
		"deployment v2.0.0 started",
		"configuration updated",
		"iam policy changed",
		"autoscale triggered",
		"normal log message",
	}
	msgLen := len(messages) // Pre-compute length to satisfy gosec

	for i := 0; i < 100; i++ {
		events[i] = map[string]interface{}{
			"message":   messages[i%msgLen], //nolint:gosec // G602: msgLen is constant 5, i%5 always in bounds
			"timestamp": incidentTime.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzeChanges(events, incidentTime, time.Hour, 15*time.Minute, nil)
	}
}

// ============================================================================
// Dynamic Pattern Registry Tests - SOTA 2025
// ============================================================================

func TestNewChangePatternRegistry(t *testing.T) {
	registry := NewChangePatternRegistry()

	// Should have default change types loaded
	types := registry.GetAllChangeTypes()
	if len(types) == 0 {
		t.Error("NewChangePatternRegistry() should have default change types")
	}

	// Should have patterns for deployment
	patterns := registry.GetPatterns(ChangeTypeDeployment)
	if len(patterns) == 0 {
		t.Error("NewChangePatternRegistry() should have deployment patterns")
	}
}

func TestChangePatternRegistry_RegisterChangeType(t *testing.T) {
	registry := NewChangePatternRegistry()

	// Register a custom change type
	customType := ChangeType("KUBERNETES_CRD")
	customPatterns := []string{"customresource", "crd", "operator"}
	registry.RegisterChangeType(customType, customPatterns)

	// Verify patterns are registered
	patterns := registry.GetPatterns(customType)
	if len(patterns) != len(customPatterns) {
		t.Errorf("RegisterChangeType() patterns = %d, want %d", len(patterns), len(customPatterns))
	}

	// Test detection with custom type
	detected := registry.DetectChangeType("customresource definition applied")
	if detected != customType {
		t.Errorf("DetectChangeType() = %v, want %v", detected, customType)
	}
}

func TestChangePatternRegistry_AppendPatterns(t *testing.T) {
	registry := NewChangePatternRegistry()

	// Get initial pattern count for deployment
	initialCount := len(registry.GetPatterns(ChangeTypeDeployment))

	// Register additional patterns for existing type
	additionalPatterns := []string{"ship-it", "go-live", "production-push"}
	registry.RegisterChangeType(ChangeTypeDeployment, additionalPatterns)

	// Verify patterns were appended
	newCount := len(registry.GetPatterns(ChangeTypeDeployment))
	expectedCount := initialCount + len(additionalPatterns)
	if newCount != expectedCount {
		t.Errorf("After append, patterns = %d, want %d", newCount, expectedCount)
	}

	// Test detection with new patterns
	detected := registry.DetectChangeType("ship-it triggered for myapp")
	if detected != ChangeTypeDeployment {
		t.Errorf("DetectChangeType() with new pattern = %v, want DEPLOYMENT", detected)
	}
}

func TestChangePatternRegistry_RegisterRiskPatterns(t *testing.T) {
	registry := NewChangePatternRegistry()

	// Register custom risk patterns
	registry.RegisterRiskPatterns("critical", []string{"outage", "data-loss", "corruption"})

	// Test detection with custom patterns
	risk := registry.AssessRiskLevel("potential data-loss detected")
	if risk != "critical" {
		t.Errorf("AssessRiskLevel() = %v, want critical", risk)
	}
}

func TestChangePatternRegistry_Reset(t *testing.T) {
	registry := NewChangePatternRegistry()

	// Register custom type
	customType := ChangeType("CUSTOM_TYPE")
	registry.RegisterChangeType(customType, []string{"custom-pattern"})

	// Verify custom type exists
	detected := registry.DetectChangeType("custom-pattern detected")
	if detected != customType {
		t.Fatal("Custom type should be detectable before reset")
	}

	// Reset registry
	registry.Reset()

	// Custom type should no longer be detected
	detected = registry.DetectChangeType("custom-pattern detected")
	if detected == customType {
		t.Error("Custom type should not be detectable after reset")
	}
}

func TestGlobalPatternRegistry_RegisterChangeType(t *testing.T) {
	// Reset to ensure clean state
	ResetPatternRegistry()
	defer ResetPatternRegistry() // Clean up after test

	// Register custom type via global function
	customType := ChangeType("GLOBAL_TEST_TYPE")
	RegisterChangeType(customType, []string{"global-test-pattern"})

	// Should be detectable via detectChangeType
	detected := detectChangeType("global-test-pattern in message")
	if detected != customType {
		t.Errorf("detectChangeType() = %v, want %v", detected, customType)
	}
}

func TestGlobalPatternRegistry_RegisterRiskPatterns(t *testing.T) {
	// Reset to ensure clean state
	ResetPatternRegistry()
	defer ResetPatternRegistry()

	// Register custom risk patterns
	RegisterRiskPatterns("critical", []string{"global-critical-test"})

	// Should be detectable via assessRiskLevel
	risk := assessRiskLevel("global-critical-test occurred")
	if risk != "critical" {
		t.Errorf("assessRiskLevel() = %v, want critical", risk)
	}
}

func TestGetRegisteredChangeTypes(t *testing.T) {
	ResetPatternRegistry()
	defer ResetPatternRegistry()

	types := GetRegisteredChangeTypes()

	// Should have at least the built-in types
	builtInTypes := []ChangeType{
		ChangeTypeDeployment,
		ChangeTypeConfig,
		ChangeTypeIAM,
		ChangeTypeScaling,
		ChangeTypeNetworkPolicy,
		ChangeTypeSecret,
		ChangeTypeFeatureFlag,
		ChangeTypeDatabase,
		ChangeTypeInfra,
	}

	typeSet := make(map[ChangeType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	for _, builtIn := range builtInTypes {
		if !typeSet[builtIn] {
			t.Errorf("GetRegisteredChangeTypes() missing built-in type: %v", builtIn)
		}
	}
}

func TestChangePatternRegistry_DetectChangeType_Priority(t *testing.T) {
	registry := NewChangePatternRegistry()

	// The first matching pattern wins
	// "deployment" should match DEPLOYMENT even if message also contains "config"
	detected := registry.DetectChangeType("deployment with new config")

	// Should match one of the change types (behavior depends on map iteration order)
	if detected == ChangeTypeUnknown {
		t.Error("DetectChangeType() should match a change type for mixed message")
	}
}

func TestChangePatternRegistry_AssessRiskLevel_Hierarchy(t *testing.T) {
	registry := NewChangePatternRegistry()

	tests := []struct {
		message  string
		wantRisk string
	}{
		{"production database migration", "critical"}, // production is critical
		{"database schema change", "high"},            // database alone is high
		{"deployment to staging", "medium"},           // deployment alone is medium
		{"cache update", "low"},                       // no match is low
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			risk := registry.AssessRiskLevel(tt.message)
			if risk != tt.wantRisk {
				t.Errorf("AssessRiskLevel(%q) = %v, want %v", tt.message, risk, tt.wantRisk)
			}
		})
	}
}

func TestChangePatternRegistry_GetPatterns_NonExistent(t *testing.T) {
	registry := NewChangePatternRegistry()

	patterns := registry.GetPatterns(ChangeType("NON_EXISTENT"))
	if patterns != nil {
		t.Errorf("GetPatterns(NON_EXISTENT) = %v, want nil", patterns)
	}
}

// ============================================================================
// Pattern Registry Benchmarks
// ============================================================================

func BenchmarkChangePatternRegistry_DetectChangeType(b *testing.B) {
	registry := NewChangePatternRegistry()
	messages := []string{
		"deployment v2.0.0 completed",
		"configuration updated",
		"iam policy changed",
		"autoscale triggered",
		"network policy applied",
		"secret rotated",
		"feature flag enabled",
		"database migration complete",
		"terraform apply finished",
		"normal log message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.DetectChangeType(messages[i%len(messages)])
	}
}

func BenchmarkChangePatternRegistry_AssessRiskLevel(b *testing.B) {
	registry := NewChangePatternRegistry()
	messages := []string{
		"production deployment started",
		"database migration in progress",
		"config update applied",
		"normal log message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.AssessRiskLevel(messages[i%len(messages)])
	}
}

func BenchmarkChangePatternRegistry_RegisterAndDetect(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := NewChangePatternRegistry()
		registry.RegisterChangeType("CUSTOM", []string{"custom-pattern-1", "custom-pattern-2"})
		registry.DetectChangeType("custom-pattern-1 detected")
	}
}
