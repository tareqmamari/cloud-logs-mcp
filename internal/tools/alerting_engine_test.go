package tools

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestCalculateBurnRate(t *testing.T) {
	tests := []struct {
		name             string
		sloTarget        float64
		windowDays       int
		wantErrorBudget  float64
		wantFastBurnRate float64
		wantSlowBurnRate float64
	}{
		{
			name:             "99.9% SLO 30-day window",
			sloTarget:        0.999,
			windowDays:       30,
			wantErrorBudget:  0.001,
			wantFastBurnRate: 14.4,
			wantSlowBurnRate: 3.0,
		},
		{
			name:             "99% SLO 30-day window",
			sloTarget:        0.99,
			windowDays:       30,
			wantErrorBudget:  0.01,
			wantFastBurnRate: 14.4,
			wantSlowBurnRate: 3.0,
		},
		{
			name:             "99.99% SLO 30-day window",
			sloTarget:        0.9999,
			windowDays:       30,
			wantErrorBudget:  0.0001,
			wantFastBurnRate: 14.4,
			wantSlowBurnRate: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CalculateBurnRate(tt.sloTarget, tt.windowDays)

			// Verify error budget calculation
			if math.Abs(config.SLO.ErrorBudget-tt.wantErrorBudget) > 0.0001 {
				t.Errorf("ErrorBudget = %v, want %v", config.SLO.ErrorBudget, tt.wantErrorBudget)
			}

			// Verify fast burn windows exist
			if len(config.FastBurnWindows) == 0 {
				t.Error("Expected fast burn windows to be configured")
			} else if config.FastBurnWindows[0].BurnRate != tt.wantFastBurnRate {
				t.Errorf("FastBurnRate = %v, want %v", config.FastBurnWindows[0].BurnRate, tt.wantFastBurnRate)
			}

			// Verify slow burn windows exist
			if len(config.SlowBurnWindows) == 0 {
				t.Error("Expected slow burn windows to be configured")
			} else if config.SlowBurnWindows[0].BurnRate != tt.wantSlowBurnRate {
				t.Errorf("SlowBurnRate = %v, want %v", config.SlowBurnWindows[0].BurnRate, tt.wantSlowBurnRate)
			}

			// Verify severities
			if config.FastBurnWindows[0].Severity != SeverityP1Critical {
				t.Errorf("FastBurn Severity = %v, want P1", config.FastBurnWindows[0].Severity)
			}
			if config.SlowBurnWindows[0].Severity != SeverityP2Warning {
				t.Errorf("SlowBurn Severity = %v, want P2", config.SlowBurnWindows[0].Severity)
			}
		})
	}
}

func TestCalculateErrorThreshold(t *testing.T) {
	tests := []struct {
		name                 string
		sloTarget            float64
		budgetConsumptionPct float64
		windowDuration       time.Duration
		sloWindowDays        int
		wantThresholdApprox  float64
	}{
		{
			name:                 "2% budget in 1 hour for 99.9% SLO",
			sloTarget:            0.999,
			budgetConsumptionPct: 2.0,
			windowDuration:       1 * time.Hour,
			sloWindowDays:        30,
			// 0.02 * 0.001 * (30*24) / 1 = 0.0144
			wantThresholdApprox: 0.0144,
		},
		{
			name:                 "10% budget in 24 hours for 99.9% SLO",
			sloTarget:            0.999,
			budgetConsumptionPct: 10.0,
			windowDuration:       24 * time.Hour,
			sloWindowDays:        30,
			// 0.10 * 0.001 * (30*24) / 24 = 0.003
			wantThresholdApprox: 0.003,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := CalculateErrorThreshold(
				tt.sloTarget,
				tt.budgetConsumptionPct,
				tt.windowDuration,
				tt.sloWindowDays,
			)

			// Allow 1% tolerance for floating point comparisons
			tolerance := tt.wantThresholdApprox * 0.01
			if math.Abs(threshold-tt.wantThresholdApprox) > tolerance {
				t.Errorf("Threshold = %v, want approximately %v", threshold, tt.wantThresholdApprox)
			}
		})
	}
}

func TestGetBurnRateThreshold(t *testing.T) {
	tests := []struct {
		name          string
		sloTarget     float64
		burnRate      float64
		wantThreshold float64
	}{
		{
			name:          "14.4x burn rate for 99.9% SLO",
			sloTarget:     0.999,
			burnRate:      14.4,
			wantThreshold: 0.0144, // 0.001 * 14.4
		},
		{
			name:          "6x burn rate for 99.9% SLO",
			sloTarget:     0.999,
			burnRate:      6.0,
			wantThreshold: 0.006, // 0.001 * 6
		},
		{
			name:          "3x burn rate for 99% SLO",
			sloTarget:     0.99,
			burnRate:      3.0,
			wantThreshold: 0.03, // 0.01 * 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := GetBurnRateThreshold(tt.sloTarget, tt.burnRate)

			tolerance := tt.wantThreshold * 0.001
			if math.Abs(threshold-tt.wantThreshold) > tolerance {
				t.Errorf("Threshold = %v, want %v", threshold, tt.wantThreshold)
			}
		})
	}
}

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		name          string
		isUserFacing  bool
		burnRate      float64
		componentType ComponentType
		wantSeverity  SeverityLevel
	}{
		{
			name:          "User-facing with high burn rate -> P1",
			isUserFacing:  true,
			burnRate:      14.4,
			componentType: ComponentWebService,
			wantSeverity:  SeverityP1Critical,
		},
		{
			name:          "User-facing with medium burn rate -> P1",
			isUserFacing:  true,
			burnRate:      6.0,
			componentType: ComponentWebService,
			wantSeverity:  SeverityP1Critical,
		},
		{
			name:          "User-facing with low burn rate -> P2",
			isUserFacing:  true,
			burnRate:      3.0,
			componentType: ComponentWebService,
			wantSeverity:  SeverityP2Warning,
		},
		{
			name:          "Non-user-facing with high burn rate -> P2",
			isUserFacing:  false,
			burnRate:      14.4,
			componentType: ComponentWorker,
			wantSeverity:  SeverityP2Warning,
		},
		{
			name:          "Database with very high burn rate -> P1",
			isUserFacing:  false,
			burnRate:      15.0,
			componentType: ComponentDatabase,
			wantSeverity:  SeverityP1Critical,
		},
		{
			name:          "Low burn rate, not user-facing -> P3",
			isUserFacing:  false,
			burnRate:      0.5,
			componentType: ComponentWorker,
			wantSeverity:  SeverityP3Info,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := ClassifySeverity(tt.isUserFacing, tt.burnRate, tt.componentType)

			if severity != tt.wantSeverity {
				t.Errorf("Severity = %v, want %v", severity, tt.wantSeverity)
			}
		})
	}
}

func TestGetMethodologyForComponent(t *testing.T) {
	tests := []struct {
		componentType   ComponentType
		wantMethodology AlertingMethodology
	}{
		{ComponentWebService, MethodologyRED},
		{ComponentAPIGateway, MethodologyRED},
		{ComponentDatabase, MethodologyUSE},
		{ComponentCache, MethodologyUSE},
		{ComponentMessageQueue, MethodologyUSE},
		{ComponentWorker, MethodologyRED},
		{ComponentKubernetes, MethodologyUSE},
		{ComponentCustom, MethodologyRED}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.componentType), func(t *testing.T) {
			methodology := GetMethodologyForComponent(tt.componentType)

			if methodology != tt.wantMethodology {
				t.Errorf("Methodology = %v, want %v", methodology, tt.wantMethodology)
			}
		})
	}
}

func TestDetectComponentType(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		useCase       string
		wantComponent ComponentType
	}{
		{
			name:          "Detect web service from HTTP keywords",
			query:         "filter $d.http_status >= 500",
			useCase:       "high error rate on API",
			wantComponent: ComponentWebService,
		},
		{
			name:          "Detect database from SQL keywords",
			query:         "filter $d.query_duration_ms > 1000",
			useCase:       "slow database queries",
			wantComponent: ComponentDatabase,
		},
		{
			name:          "Detect cache from cache keywords",
			query:         "",
			useCase:       "cache hit rate dropping",
			wantComponent: ComponentCache,
		},
		{
			name:          "Detect message queue from kafka keywords",
			query:         "filter $d.consumer_lag > 10000",
			useCase:       "kafka consumer lag",
			wantComponent: ComponentMessageQueue,
		},
		{
			name:          "Detect kubernetes from pod keywords",
			query:         "filter $d.kubernetes.pod_name exists",
			useCase:       "pod restarts",
			wantComponent: ComponentKubernetes,
		},
		{
			name:          "Detect worker from job keywords",
			query:         "",
			useCase:       "background job failures",
			wantComponent: ComponentWorker,
		},
		{
			name:          "Default to custom for unknown patterns",
			query:         "filter $d.something == true",
			useCase:       "some random metric",
			wantComponent: ComponentCustom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			componentType := DetectComponentType(tt.query, tt.useCase)

			if componentType != tt.wantComponent {
				t.Errorf("ComponentType = %v, want %v", componentType, tt.wantComponent)
			}
		})
	}
}

func TestValidateActionability(t *testing.T) {
	tests := []struct {
		name       string
		suggestion AdvancedAlertSuggestion
		wantErrors int
	}{
		{
			name: "Valid alert with runbook and actions",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "https://runbooks.example.com/high-error-rate",
				SuggestedActions: []string{"Check error logs", "Verify deployment"},
			},
			wantErrors: 0,
		},
		{
			name: "Missing runbook URL",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "",
				SuggestedActions: []string{"Check error logs"},
			},
			wantErrors: 1,
		},
		{
			name: "Missing suggested actions",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "https://runbooks.example.com/alert",
				SuggestedActions: []string{},
			},
			wantErrors: 1,
		},
		{
			name: "Missing both",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "",
				SuggestedActions: nil,
			},
			wantErrors: 2,
		},
		{
			name: "Invalid runbook URL format",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "not-a-url",
				SuggestedActions: []string{"Do something"},
			},
			wantErrors: 1,
		},
		{
			name: "Valid with path-based runbook",
			suggestion: AdvancedAlertSuggestion{
				RunbookURL:       "/runbooks/web_service/error-rate",
				SuggestedActions: []string{"Check logs"},
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateActionability(&tt.suggestion)

			if len(errors) != tt.wantErrors {
				t.Errorf("Got %d errors, want %d: %v", len(errors), tt.wantErrors, errors)
			}
		})
	}
}

func TestGenerateRunbookURL(t *testing.T) {
	tests := []struct {
		componentType ComponentType
		alertName     string
		wantURL       string
	}{
		{
			componentType: ComponentWebService,
			alertName:     "High Error Rate",
			wantURL:       "/runbooks/web_service/high-error-rate",
		},
		{
			componentType: ComponentDatabase,
			alertName:     "Connection Pool Exhaustion",
			wantURL:       "/runbooks/database/connection-pool-exhaustion",
		},
		{
			componentType: ComponentKubernetes,
			alertName:     "Pod_Restart_Alert",
			wantURL:       "/runbooks/kubernetes/pod-restart-alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.alertName, func(t *testing.T) {
			url := GenerateRunbookURL(tt.componentType, tt.alertName)

			if url != tt.wantURL {
				t.Errorf("URL = %v, want %v", url, tt.wantURL)
			}
		})
	}
}

func TestGenerateDefaultActions(t *testing.T) {
	tests := []struct {
		componentType  ComponentType
		signal         string
		wantMinActions int
	}{
		{ComponentWebService, "errors", 6},
		{ComponentDatabase, "saturation", 9},
		{ComponentKubernetes, "utilization", 9},
		{ComponentMessageQueue, "saturation", 9},
		{ComponentCache, "errors", 6},
	}

	for _, tt := range tests {
		t.Run(string(tt.componentType)+"_"+tt.signal, func(t *testing.T) {
			actions := GenerateDefaultActions(tt.componentType, tt.signal)

			if len(actions) < tt.wantMinActions {
				t.Errorf("Got %d actions, want at least %d", len(actions), tt.wantMinActions)
			}

			// Verify actions are numbered
			if len(actions) > 0 && actions[0][0] != '1' {
				t.Error("Actions should be numbered starting with 1")
			}
		})
	}
}

func TestAlertingStrategyMatrix(t *testing.T) {
	// Verify all expected component types have strategies
	expectedComponents := []ComponentType{
		ComponentWebService,
		ComponentAPIGateway,
		ComponentDatabase,
		ComponentCache,
		ComponentMessageQueue,
		ComponentWorker,
		ComponentKubernetes,
	}

	for _, compType := range expectedComponents {
		t.Run(string(compType), func(t *testing.T) {
			strategy, ok := AlertingStrategyMatrix[compType]
			if !ok {
				t.Errorf("Missing strategy for component type: %v", compType)
				return
			}

			// Verify strategy has recommended metrics
			if len(strategy.RecommendedMetrics) == 0 {
				t.Error("Strategy should have recommended metrics")
			}

			// Verify each metric has required fields
			for _, metric := range strategy.RecommendedMetrics {
				if metric.Name == "" {
					t.Error("Metric missing name")
				}
				if metric.Query == "" {
					t.Error("Metric missing query")
				}
				if metric.Signal == "" {
					t.Error("Metric missing signal type")
				}
			}

			// Verify runbook template exists
			if strategy.RunbookTemplate == "" {
				t.Error("Strategy should have a runbook template")
			}
		})
	}
}

func TestDynamicBaselineCalculator(t *testing.T) {
	calc := NewDynamicBaselineCalculator("hourly", 3.0)

	t.Run("Insufficient data points", func(t *testing.T) {
		_, _, err := calc.CalculateThreshold([]float64{1.0, 2.0, 3.0})
		if err == nil {
			t.Error("Expected error for insufficient data points")
		}
	})

	t.Run("Valid data points", func(t *testing.T) {
		// Generate 50 data points with mean=100, stddev~10
		data := make([]float64, 50)
		for i := range data {
			data[i] = 100.0 + float64(i%10) - 5.0
		}

		lower, upper, err := calc.CalculateThreshold(data)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify bounds are reasonable
		if lower >= upper {
			t.Errorf("Lower bound (%v) should be less than upper bound (%v)", lower, upper)
		}

		// With 3 std dev multiplier, bounds should be reasonable
		if lower < 0 {
			t.Error("Lower bound should not be negative for non-negative metrics")
		}
	})
}

func TestFormatBurnRateExplanation(t *testing.T) {
	explanation := FormatBurnRateExplanation(0.999, 14.4, 1*time.Hour, 30)

	// Verify key information is present
	if !alertingTestContains(explanation, "99.900%") {
		t.Errorf("Explanation should include SLO percentage, got: %s", explanation)
	}
	if !alertingTestContains(explanation, "14.4x") {
		t.Errorf("Explanation should include burn rate, got: %s", explanation)
	}
	if !alertingTestContains(explanation, "1h") {
		t.Errorf("Explanation should include window duration, got: %s", explanation)
	}
	if !alertingTestContains(explanation, "30-day") {
		t.Errorf("Explanation should include SLO window, got: %s", explanation)
	}
}

func TestGenerateDynamicBaselineQuery(t *testing.T) {
	tests := []struct {
		metricField     string
		seasonalityType string
		lookbackDays    int
		wantContains    []string
	}{
		{
			metricField:     "response_time_ms",
			seasonalityType: "hourly",
			lookbackDays:    7,
			wantContains:    []string{"response_time_ms", "avg", "stddev", "hour_of_day"},
		},
		{
			metricField:     "request_count",
			seasonalityType: "daily",
			lookbackDays:    30,
			wantContains:    []string{"request_count", "avg", "stddev", "day_of_week"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.seasonalityType, func(t *testing.T) {
			query := GenerateDynamicBaselineQuery(tt.metricField, tt.seasonalityType, tt.lookbackDays)

			for _, want := range tt.wantContains {
				if !alertingTestContains(query, want) {
					t.Errorf("Query should contain %q, got: %s", want, query)
				}
			}
		})
	}
}

// Helper function for alerting engine tests
func alertingTestContains(s, substr string) bool {
	return strings.Contains(s, substr)
}
