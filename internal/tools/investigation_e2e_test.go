package tools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestE2E_InvestigationModeSelection tests that the correct mode is selected based on parameters
func TestE2E_InvestigationModeSelection(t *testing.T) {
	factory := NewQueryStrategyFactory()

	testCases := []struct {
		name         string
		params       map[string]interface{}
		expectedMode InvestigationMode
	}{
		{
			name:         "global_mode_no_params",
			params:       map[string]interface{}{},
			expectedMode: ModeGlobal,
		},
		{
			name: "component_mode_with_application",
			params: map[string]interface{}{
				"application": "api-gateway",
			},
			expectedMode: ModeComponent,
		},
		{
			name: "flow_mode_with_trace_id",
			params: map[string]interface{}{
				"trace_id": "abc123def456",
			},
			expectedMode: ModeFlow,
		},
		{
			name: "flow_mode_with_correlation_id",
			params: map[string]interface{}{
				"correlation_id": "req-12345",
			},
			expectedMode: ModeFlow,
		},
		{
			name: "flow_mode_takes_precedence_over_component",
			params: map[string]interface{}{
				"application": "api-gateway",
				"trace_id":    "abc123def456",
			},
			expectedMode: ModeFlow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode := factory.DetermineMode(tc.params)
			if mode != tc.expectedMode {
				t.Errorf("Expected mode %s, got %s", tc.expectedMode, mode)
			}
		})
	}
}

// TestE2E_GlobalModeStrategy tests the global investigation strategy
func TestE2E_GlobalModeStrategy(t *testing.T) {
	strategy := &GlobalModeStrategy{}

	t.Run("name", func(t *testing.T) {
		if strategy.Name() != "global" {
			t.Errorf("Expected name 'global', got '%s'", strategy.Name())
		}
	})

	t.Run("initial_queries", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode: ModeGlobal,
			TimeRange: InvestigationTimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
			},
		}

		queries := strategy.InitialQueries(ctx)
		if len(queries) < 3 {
			t.Errorf("Expected at least 3 queries, got %d", len(queries))
		}

		// Verify query IDs
		expectedIDs := []string{"global-error-rate", "global-error-timeline", "global-critical-errors"}
		for _, expectedID := range expectedIDs {
			found := false
			for _, q := range queries {
				if q.ID == expectedID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected query with ID '%s' not found", expectedID)
			}
		}
	})

	t.Run("analyze_error_rates", func(t *testing.T) {
		ctx := &SmartInvestigationContext{Mode: ModeGlobal}

		// Create mock query results with high error counts
		results := []ExecutedQuery{
			{
				QueryID: "global-error-rate",
				Events: []map[string]interface{}{
					{"applicationname": "api-gateway", "error_count": float64(150)},
					{"applicationname": "payment-service", "error_count": float64(50)},
					{"applicationname": "user-service", "error_count": float64(5)},
				},
			},
		}

		findings := strategy.AnalyzeResults(ctx, results)

		// Should find at least one high-severity finding
		var highSeverityCount int
		for _, f := range findings {
			if f.Severity == SeverityHigh || f.Severity == SeverityCritical {
				highSeverityCount++
			}
		}

		if highSeverityCount == 0 {
			t.Error("Expected at least one high/critical severity finding for high error counts")
		}

		// api-gateway should be identified with errors
		var foundAPIGateway bool
		for _, f := range findings {
			if f.Service == "api-gateway" {
				foundAPIGateway = true
				break
			}
		}
		if !foundAPIGateway {
			t.Error("Expected finding for api-gateway service")
		}
	})

	t.Run("analyze_error_timeline_spike", func(t *testing.T) {
		ctx := &SmartInvestigationContext{Mode: ModeGlobal}

		// Create mock results with a spike pattern
		results := []ExecutedQuery{
			{
				QueryID: "global-error-timeline",
				Events: []map[string]interface{}{
					{"time_bucket": "2024-01-15T10:00:00Z", "errors": float64(10)},
					{"time_bucket": "2024-01-15T10:05:00Z", "errors": float64(12)},
					{"time_bucket": "2024-01-15T10:10:00Z", "errors": float64(100)}, // Spike!
					{"time_bucket": "2024-01-15T10:15:00Z", "errors": float64(8)},
					{"time_bucket": "2024-01-15T10:20:00Z", "errors": float64(11)},
				},
			},
		}

		findings := strategy.AnalyzeResults(ctx, results)

		// Should detect the spike
		var foundSpike bool
		for _, f := range findings {
			if f.Type == FindingSpike {
				foundSpike = true
				if !strings.Contains(f.Summary, "spike") && !strings.Contains(f.Summary, "Spike") {
					t.Errorf("Spike finding should mention 'spike' in summary: %s", f.Summary)
				}
				break
			}
		}
		if !foundSpike {
			t.Error("Expected to detect error spike")
		}
	})
}

// TestE2E_ComponentModeStrategy tests the component investigation strategy
func TestE2E_ComponentModeStrategy(t *testing.T) {
	strategy := &ComponentModeStrategy{}

	t.Run("name", func(t *testing.T) {
		if strategy.Name() != "component" {
			t.Errorf("Expected name 'component', got '%s'", strategy.Name())
		}
	})

	t.Run("initial_queries_include_service", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:          ModeComponent,
			TargetService: "payment-api",
			TimeRange: InvestigationTimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
			},
		}

		queries := strategy.InitialQueries(ctx)

		// All queries should filter by the target service
		for _, q := range queries {
			if !strings.Contains(q.Query, "payment-api") {
				t.Errorf("Query %s should filter by target service 'payment-api'", q.ID)
			}
		}
	})

	t.Run("analyze_error_patterns", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:          ModeComponent,
			TargetService: "payment-api",
		}

		results := []ExecutedQuery{
			{
				QueryID: "component-error-patterns",
				Events: []map[string]interface{}{
					{"message": "Connection timeout to database", "occurrences": float64(45)},
					{"message": "Invalid payment method", "occurrences": float64(20)},
					{"message": "Rare error", "occurrences": float64(2)},
				},
			},
		}

		findings := strategy.AnalyzeResults(ctx, results)

		// Should find at least one recurring error pattern
		var foundRecurring bool
		for _, f := range findings {
			if strings.Contains(f.Summary, "Recurring") || strings.Contains(f.Summary, "pattern") {
				foundRecurring = true
				break
			}
		}
		if !foundRecurring {
			t.Error("Expected to find recurring error pattern")
		}
	})

	t.Run("analyze_dependency_issues", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:          ModeComponent,
			TargetService: "payment-api",
		}

		results := []ExecutedQuery{
			{
				QueryID: "component-dependencies",
				Events: []map[string]interface{}{
					{"message": "Connection refused to postgres:5432"},
					{"message": "Connection refused to postgres:5432"},
					{"message": "Connection refused to postgres:5432"},
					{"message": "timeout waiting for response from redis"},
					{"message": "timeout waiting for response from redis"},
					{"message": "timeout waiting for response from redis"},
				},
			},
		}

		findings := strategy.AnalyzeResults(ctx, results)

		// Should find dependency issues
		var foundDependency bool
		for _, f := range findings {
			if f.Type == FindingDependency {
				foundDependency = true
				break
			}
		}
		if !foundDependency {
			t.Error("Expected to find dependency issue")
		}
	})
}

// TestE2E_FlowModeStrategy tests the flow investigation strategy
func TestE2E_FlowModeStrategy(t *testing.T) {
	strategy := &FlowModeStrategy{}

	t.Run("name", func(t *testing.T) {
		if strategy.Name() != "flow" {
			t.Errorf("Expected name 'flow', got '%s'", strategy.Name())
		}
	})

	t.Run("initial_queries_with_trace_id", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:    ModeFlow,
			TraceID: "abc123def456", // pragma: allowlist secret
			TimeRange: InvestigationTimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
			},
		}

		queries := strategy.InitialQueries(ctx)

		if len(queries) == 0 {
			t.Fatal("Expected at least one query")
		}

		// Query should include trace_id
		if !strings.Contains(queries[0].Query, "abc123def456") {
			t.Error("Query should filter by trace_id")
		}
	})

	t.Run("analyze_request_flow_with_error", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:    ModeFlow,
			TraceID: "abc123",
		}

		results := []ExecutedQuery{
			{
				QueryID: "flow-by-trace",
				Events: []map[string]interface{}{
					{"applicationname": "api-gateway", "severity": float64(3), "message": "Request received"},
					{"applicationname": "auth-service", "severity": float64(3), "message": "Token validated"},
					{"applicationname": "payment-api", "severity": float64(5), "message": "Database connection failed"},
					{"applicationname": "payment-api", "severity": float64(5), "message": "Request failed"},
				},
			},
		}

		findings := strategy.AnalyzeResults(ctx, results)

		if len(findings) == 0 {
			t.Fatal("Expected at least one finding")
		}

		// Should identify payment-api as the failure point
		var foundPaymentAPI bool
		for _, f := range findings {
			if f.Service == "payment-api" {
				foundPaymentAPI = true
				break
			}
		}
		if !foundPaymentAPI {
			t.Error("Expected to identify payment-api as failure point")
		}
	})
}

// TestE2E_HeuristicEngine tests the heuristic engine
func TestE2E_HeuristicEngine(t *testing.T) {
	engine := NewHeuristicEngine()

	testCases := []struct {
		name            string
		findingSummary  string
		expectMatch     bool
		expectMatchName string
	}{
		{
			name:            "timeout_detection",
			findingSummary:  "Connection timeout to downstream service",
			expectMatch:     true,
			expectMatchName: "timeout",
		},
		{
			name:            "memory_detection",
			findingSummary:  "java.lang.OutOfMemoryError: Java heap space",
			expectMatch:     true,
			expectMatchName: "memory",
		},
		{
			name:            "database_detection",
			findingSummary:  "Connection pool exhausted for postgres",
			expectMatch:     true,
			expectMatchName: "database",
		},
		{
			name:            "auth_detection",
			findingSummary:  "401 Unauthorized: Invalid token",
			expectMatch:     true,
			expectMatchName: "auth",
		},
		{
			name:            "rate_limit_detection",
			findingSummary:  "429 Too Many Requests",
			expectMatch:     true,
			expectMatchName: "rate",
		},
		{
			name:            "network_detection",
			findingSummary:  "Connection refused to service endpoint",
			expectMatch:     true,
			expectMatchName: "network",
		},
		{
			name:            "no_match",
			findingSummary:  "Successfully processed request",
			expectMatch:     false,
			expectMatchName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			finding := InvestigationFinding{
				Summary:  tc.findingSummary,
				Service:  "test-service",
				Severity: SeverityHigh,
			}

			actions := engine.AnalyzeAndSuggest([]InvestigationFinding{finding}, nil)

			if tc.expectMatch && len(actions) == 0 {
				t.Errorf("Expected heuristic to match for '%s'", tc.findingSummary)
			}
			if !tc.expectMatch && len(actions) > 0 {
				t.Errorf("Expected no heuristic match for '%s', but got %d actions", tc.findingSummary, len(actions))
			}
		})
	}
}

// TestE2E_HeuristicSOPs tests SOP generation from heuristics
func TestE2E_HeuristicSOPs(t *testing.T) {
	engine := NewHeuristicEngine()

	findings := []InvestigationFinding{
		{
			Summary:  "Connection timeout to downstream service",
			Service:  "api-gateway",
			Severity: SeverityHigh,
		},
	}

	sops := engine.GetMatchingSOPs(findings, nil)

	if len(sops) == 0 {
		t.Fatal("Expected at least one SOP")
	}

	// Verify SOP has required fields
	for _, sop := range sops {
		if sop.Trigger == "" {
			t.Error("SOP trigger should not be empty")
		}
		if sop.Procedure == "" {
			t.Error("SOP procedure should not be empty")
		}
		if sop.Escalation == "" {
			t.Error("SOP escalation should not be empty")
		}
	}
}

// TestE2E_RemediationGenerator tests alert and dashboard generation
func TestE2E_RemediationGenerator(t *testing.T) {
	generator := NewRemediationGenerator()

	ctx := &IncidentContext{
		RootCause:        "Database connection pool exhaustion causing timeouts",
		AffectedServices: []string{"payment-api", "order-service"},
	}

	assets := generator.Generate(ctx, "high")

	t.Run("alert_generation", func(t *testing.T) {
		if assets.Alert == nil {
			t.Fatal("Expected alert to be generated")
		}

		if assets.Alert.Name == "" {
			t.Error("Alert name should not be empty")
		}
		if assets.Alert.Condition == "" {
			t.Error("Alert condition should not be empty")
		}
		if assets.Alert.Threshold <= 0 {
			t.Error("Alert threshold should be positive")
		}

		// Verify condition includes affected services
		if !strings.Contains(assets.Alert.Condition, "payment-api") {
			t.Error("Alert condition should include payment-api")
		}
		if !strings.Contains(assets.Alert.Condition, "order-service") {
			t.Error("Alert condition should include order-service")
		}

		// Verify Terraform is generated
		if assets.Alert.Terraform == "" {
			t.Error("Terraform should be generated")
		}
		if !strings.Contains(assets.Alert.Terraform, "resource") {
			t.Error("Terraform should contain resource definition")
		}

		// Verify IBM Cloud Logs JSON
		if assets.Alert.IBMCloudLogsJSON == nil {
			t.Error("IBM Cloud Logs JSON should be generated")
		}
	})

	t.Run("dashboard_generation", func(t *testing.T) {
		if assets.Dashboard == nil {
			t.Fatal("Expected dashboard to be generated")
		}

		if assets.Dashboard.Name == "" {
			t.Error("Dashboard name should not be empty")
		}
		if len(assets.Dashboard.Widgets) == 0 {
			t.Error("Dashboard should have widgets")
		}

		// Verify expected widget types
		expectedWidgets := []string{"Error Rate", "Errors by Service", "Latency"}
		for _, expected := range expectedWidgets {
			found := false
			for _, w := range assets.Dashboard.Widgets {
				if strings.Contains(w.Title, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected widget containing '%s'", expected)
			}
		}

		// Verify IBM Cloud Logs JSON
		if assets.Dashboard.IBMCloudLogsJSON == nil {
			t.Error("IBM Cloud Logs dashboard JSON should be generated")
		}
	})

	t.Run("sop_generation", func(t *testing.T) {
		// The root cause mentions "connection" which should trigger database SOP
		if len(assets.SOPRecommendations) == 0 {
			t.Error("Expected at least one SOP recommendation")
		}

		// Verify SOP content
		foundConnectionSOP := false
		for _, sop := range assets.SOPRecommendations {
			if strings.Contains(strings.ToLower(sop.Trigger), "connection") ||
				strings.Contains(strings.ToLower(sop.Procedure), "connection") {
				foundConnectionSOP = true
				break
			}
		}
		if !foundConnectionSOP {
			t.Error("Expected SOP related to connection issues")
		}
	})
}

// TestE2E_TerraformValidity tests that generated Terraform is syntactically valid
func TestE2E_TerraformValidity(t *testing.T) {
	generator := NewRemediationGenerator()

	ctx := &IncidentContext{
		RootCause:        "Test root cause",
		AffectedServices: []string{"test-service"},
	}

	assets := generator.Generate(ctx, "critical")

	tf := assets.Alert.Terraform

	// Basic syntax checks
	requiredPatterns := []string{
		"resource",
		"ibm_logs_alert",
		"name",
		"severity",
		"condition",
		"notification_groups",
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(tf, pattern) {
			t.Errorf("Terraform missing required pattern: %s", pattern)
		}
	}

	// Check for balanced braces
	openBraces := strings.Count(tf, "{")
	closeBraces := strings.Count(tf, "}")
	if openBraces != closeBraces {
		t.Errorf("Terraform has unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}
}

// TestE2E_IBMCloudLogsJSONValidity tests that generated IBM Cloud Logs JSON is valid
func TestE2E_IBMCloudLogsJSONValidity(t *testing.T) {
	generator := NewRemediationGenerator()

	ctx := &IncidentContext{
		RootCause:        "Test root cause",
		AffectedServices: []string{"test-service"},
	}

	assets := generator.Generate(ctx, "high")

	t.Run("alert_json", func(t *testing.T) {
		jsonBytes, err := json.Marshal(assets.Alert.IBMCloudLogsJSON)
		if err != nil {
			t.Fatalf("Alert JSON is not valid: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("Alert JSON cannot be parsed: %v", err)
		}

		// Check required fields
		requiredFields := []string{"name", "severity", "condition", "notifications"}
		for _, field := range requiredFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("Alert JSON missing field: %s", field)
			}
		}
	})

	t.Run("dashboard_json", func(t *testing.T) {
		jsonBytes, err := json.Marshal(assets.Dashboard.IBMCloudLogsJSON)
		if err != nil {
			t.Fatalf("Dashboard JSON is not valid: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("Dashboard JSON cannot be parsed: %v", err)
		}

		// Check required fields
		requiredFields := []string{"name", "widgets", "layout"}
		for _, field := range requiredFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("Dashboard JSON missing field: %s", field)
			}
		}
	})
}

// TestE2E_EvidenceSynthesis tests the evidence synthesis process
func TestE2E_EvidenceSynthesis(t *testing.T) {
	strategy := &GlobalModeStrategy{}

	t.Run("synthesize_with_critical_findings", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode: ModeGlobal,
			Findings: []InvestigationFinding{
				{
					Type:       FindingError,
					Service:    "api-gateway",
					Summary:    "High error rate in api-gateway",
					Severity:   SeverityCritical,
					Confidence: 0.9,
				},
				{
					Type:       FindingDependency,
					Service:    "database",
					Summary:    "Database connection failures",
					Severity:   SeverityHigh,
					Confidence: 0.85,
				},
			},
		}

		evidence := strategy.SynthesizeEvidence(ctx)

		if evidence.RootCause == "" {
			t.Error("Root cause should not be empty")
		}
		if evidence.Confidence <= 0 {
			t.Error("Confidence should be positive")
		}
		if len(evidence.AffectedServices) == 0 {
			t.Error("Affected services should not be empty")
		}
	})

	t.Run("synthesize_with_no_findings", func(t *testing.T) {
		ctx := &SmartInvestigationContext{
			Mode:     ModeGlobal,
			Findings: []InvestigationFinding{},
		}

		evidence := strategy.SynthesizeEvidence(ctx)

		if !strings.Contains(strings.ToLower(evidence.RootCause), "healthy") &&
			!strings.Contains(strings.ToLower(evidence.RootCause), "no") {
			t.Errorf("Expected 'healthy' or 'no issues' message, got: %s", evidence.RootCause)
		}
	})
}

// TestE2E_InvestigationContextState tests investigation context state management
func TestE2E_InvestigationContextState(t *testing.T) {
	ctx := &SmartInvestigationContext{
		Mode:          ModeComponent,
		TargetService: "test-service",
		TimeRange: InvestigationTimeRange{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now(),
		},
	}

	t.Run("add_findings", func(t *testing.T) {
		ctx.Findings = append(ctx.Findings, InvestigationFinding{
			Type:       FindingError,
			Summary:    "Test finding",
			Severity:   SeverityHigh,
			Confidence: 0.8,
		})

		if len(ctx.Findings) != 1 {
			t.Errorf("Expected 1 finding, got %d", len(ctx.Findings))
		}
	})

	t.Run("add_query_history", func(t *testing.T) {
		ctx.QueryHistory = append(ctx.QueryHistory, ExecutedQuery{
			QueryID:  "test-query",
			Query:    "source logs | limit 10",
			Duration: 100 * time.Millisecond,
		})

		if len(ctx.QueryHistory) != 1 {
			t.Errorf("Expected 1 query in history, got %d", len(ctx.QueryHistory))
		}
	})

	t.Run("add_actions", func(t *testing.T) {
		ctx.NextActions = append(ctx.NextActions, HeuristicAction{
			Priority:    1,
			Description: "Test action",
			Rationale:   "Test rationale",
		})

		if len(ctx.NextActions) != 1 {
			t.Errorf("Expected 1 action, got %d", len(ctx.NextActions))
		}
	})
}

// TestE2E_FindingSortingAndPrioritization tests finding sorting
func TestE2E_FindingSortingAndPrioritization(t *testing.T) {
	findings := []InvestigationFinding{
		{Summary: "Low severity", Severity: SeverityLow},
		{Summary: "Critical severity", Severity: SeverityCritical},
		{Summary: "Medium severity", Severity: SeverityMedium},
		{Summary: "High severity", Severity: SeverityHigh},
	}

	sortFindingsBySeverity(findings)

	// Critical should be first
	if findings[0].Severity != SeverityCritical {
		t.Errorf("Expected critical first, got %s", findings[0].Severity)
	}

	// Low should be last
	if findings[len(findings)-1].Severity != SeverityLow {
		t.Errorf("Expected low last, got %s", findings[len(findings)-1].Severity)
	}
}

// TestE2E_ActionSortingAndDeduplication tests action deduplication
func TestE2E_ActionSortingAndDeduplication(t *testing.T) {
	actions := []HeuristicAction{
		{Priority: 3, Description: "Low priority"},
		{Priority: 1, Description: "High priority"},
		{Priority: 1, Description: "High priority"}, // Duplicate
		{Priority: 2, Description: "Medium priority"},
	}

	deduplicated := deduplicateActions(actions)
	sortActionsByPriority(deduplicated)

	if len(deduplicated) != 3 {
		t.Errorf("Expected 3 actions after deduplication, got %d", len(deduplicated))
	}

	// Highest priority (1) should be first
	if deduplicated[0].Priority != 1 {
		t.Errorf("Expected priority 1 first, got %d", deduplicated[0].Priority)
	}
}

// TestE2E_QueryCleaning tests query string cleaning
func TestE2E_QueryCleaning(t *testing.T) {
	query := `source logs
		| filter $m.severity >= 5
		| groupby $l.applicationname
		| limit 10`

	cleaned := cleanQueryString(query)

	// Should be single line
	if strings.Contains(cleaned, "\n") {
		t.Error("Cleaned query should not contain newlines")
	}

	// Should preserve essential parts
	requiredParts := []string{"source logs", "filter", "groupby", "limit"}
	for _, part := range requiredParts {
		if !strings.Contains(cleaned, part) {
			t.Errorf("Cleaned query missing: %s", part)
		}
	}
}

// TestE2E_HelperFunctions tests helper functions
func TestE2E_HelperFunctions(t *testing.T) {
	t.Run("categorize_severity_by_count", func(t *testing.T) {
		testCases := []struct {
			count    float64
			expected InvestigationSeverity
		}{
			{600, SeverityCritical},
			{200, SeverityHigh},
			{50, SeverityMedium},
			{5, SeverityLow},
		}

		for _, tc := range testCases {
			result := categorizeSeverityByCount(tc.count)
			if result != tc.expected {
				t.Errorf("Count %.0f: expected %s, got %s", tc.count, tc.expected, result)
			}
		}
	})

	t.Run("truncate_text", func(t *testing.T) {
		longText := "This is a very long text that should be truncated"
		truncated := truncateText(longText, 20)

		if len(truncated) > 20 {
			t.Errorf("Truncated text too long: %d chars", len(truncated))
		}
		if !strings.HasSuffix(truncated, "...") {
			t.Error("Truncated text should end with ...")
		}
	})

	t.Run("extract_message_from_event", func(t *testing.T) {
		testCases := []struct {
			name     string
			event    map[string]interface{}
			expected string
		}{
			{
				name:     "direct_message",
				event:    map[string]interface{}{"message": "Direct message"},
				expected: "Direct message",
			},
			{
				name: "user_data_message",
				event: map[string]interface{}{
					"user_data": map[string]interface{}{
						"message": "User data message",
					},
				},
				expected: "User data message",
			},
			{
				name:     "text_field",
				event:    map[string]interface{}{"text": "Text field"},
				expected: "Text field",
			},
			{
				name:     "empty",
				event:    map[string]interface{}{},
				expected: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := extractMessageFromEvent(tc.event)
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			})
		}
	})
}
