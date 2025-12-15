// Package tools provides MCP tools for IBM Cloud Logs operations.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// AdvancedSuggestAlertTool generates SRE-grade alert recommendations
// based on Google SRE best practices, RED/USE methodologies, and
// multi-window burn rate alerting.
type AdvancedSuggestAlertTool struct {
	*BaseTool
}

// NewAdvancedSuggestAlertTool creates a new AdvancedSuggestAlertTool
func NewAdvancedSuggestAlertTool(c *client.Client, l *zap.Logger) *AdvancedSuggestAlertTool {
	return &AdvancedSuggestAlertTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *AdvancedSuggestAlertTool) Name() string { return "suggest_alert" }

// Description returns the tool description
func (t *AdvancedSuggestAlertTool) Description() string {
	return `Generate high-fidelity, low-noise alert recommendations based on SRE best practices.

**Key Features:**
- Symptom-based alerting (alerts on user-facing symptoms, not causes)
- Automatic methodology selection (RED for services, USE for resources)
- Multi-window burn rate alerting for SLO-based monitoring
- Dynamic baseline suggestions for seasonal metrics
- Severity classification based on user impact (P1/P2/P3)

**Methodologies:**
- **RED Method** (for services): Rate, Errors, Duration
- **USE Method** (for resources): Utilization, Saturation, Errors
- **Golden Signals**: Latency, Traffic, Errors, Saturation

**References:**
- Google SRE Handbook Chapter 5: Alerting
- "My Philosophy on Alerting" by Rob Ewaschuk (Google)
- SLO-based Multi-Window Multi-Burn-Rate Alerting

**Related tools:** create_alert_definition, create_alert, list_alerts, query_logs`
}

// InputSchema returns the enhanced input schema
func (t *AdvancedSuggestAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"service_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of service/component being monitored. Determines RED vs USE methodology.",
				"enum": []string{
					"web_service", "api_gateway", "database", "cache",
					"message_queue", "worker", "kubernetes", "serverless",
					"storage", "network", "load_balancer", "microservice",
					"monolith", "custom",
				},
			},
			"slo_target": map[string]interface{}{
				"type":        "number",
				"description": "Service Level Objective target (e.g., 0.999 for 99.9%). Enables burn rate alerting.",
				"minimum":     0.9,
				"maximum":     0.99999,
				"examples":    []float64{0.99, 0.999, 0.9999},
			},
			"slo_window_days": map[string]interface{}{
				"type":        "integer",
				"description": "SLO measurement window in days (default: 30)",
				"default":     30,
				"minimum":     1,
				"maximum":     90,
			},
			"criticality_tier": map[string]interface{}{
				"type":        "string",
				"description": "Service criticality tier. Affects alert severity and response expectations.",
				"enum":        []string{"tier1_critical", "tier2_important", "tier3_standard"},
				"default":     "tier2_important",
			},
			"is_user_facing": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the service directly affects end users. P1 alerts require user-facing + high burn rate.",
				"default":     false,
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "DataPrime query to base the alert on. Optional if use_case is provided.",
			},
			"use_case": map[string]interface{}{
				"type": "string",
				"description": `Description of what to alert on. Examples:
- "high error rate on API endpoints"
- "database connection pool exhaustion"
- "kafka consumer lag increasing"
- "slow response times in checkout service"`,
			},
			"team": map[string]interface{}{
				"type":        "string",
				"description": "Team responsible for this alert (for routing and labeling)",
			},
			"service_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the service being monitored",
			},
			"environment": map[string]interface{}{
				"type":        "string",
				"description": "Deployment environment",
				"enum":        []string{"production", "staging", "development"},
				"default":     "production",
			},
			"runbook_url": map[string]interface{}{
				"type":        "string",
				"description": "URL to the runbook for this alert. Highly recommended but not required.",
			},
			"enable_burn_rate": map[string]interface{}{
				"type":        "boolean",
				"description": "Enable multi-window burn rate alerting (requires slo_target)",
				"default":     true,
			},
			"enable_dynamic_baselines": map[string]interface{}{
				"type":        "boolean",
				"description": "Suggest dynamic baseline queries for metrics with seasonality",
				"default":     false,
			},
		},
		"required": []string{},
	}
}

// SuggestAlertInput represents the parsed input parameters
type SuggestAlertInput struct {
	ServiceType            ComponentType
	SLOTarget              float64
	SLOWindowDays          int
	CriticalityTier        string
	IsUserFacing           bool
	Query                  string
	UseCase                string
	Team                   string
	ServiceName            string
	Environment            string
	RunbookURL             string
	EnableBurnRate         bool
	EnableDynamicBaselines bool
}

// SuggestAlertOutput represents the complete response
type SuggestAlertOutput struct {
	Suggestions    []AdvancedAlertSuggestion `json:"suggestions"`
	Methodology    AlertingMethodology       `json:"methodology"`
	StrategyMatrix *AlertStrategyConfig      `json:"strategy_matrix,omitempty"`
	BurnRateConfig *BurnRateConfig           `json:"burn_rate_config,omitempty"`
	Warnings       []string                  `json:"warnings,omitempty"`
	NextSteps      []string                  `json:"next_steps"`
	References     []string                  `json:"references"`
}

// Execute executes the tool
func (t *AdvancedSuggestAlertTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	input, err := parseAdvancedAlertInput(args)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Validate at least one of query or use_case is provided
	if input.Query == "" && input.UseCase == "" {
		return NewToolResultError("Either 'query' or 'use_case' must be provided"), nil
	}

	output := &SuggestAlertOutput{
		References: []string{
			"Google SRE Handbook - Chapter 5: Alerting",
			"Google SRE Handbook - Chapter 6: Monitoring Distributed Systems",
			"'My Philosophy on Alerting' by Rob Ewaschuk",
			"SRE Workbook - Chapter 5: Alerting on SLOs",
		},
	}

	// Detect component type if not specified
	if input.ServiceType == "" || input.ServiceType == ComponentCustom {
		input.ServiceType = DetectComponentType(input.Query, input.UseCase)
	}

	// Get methodology based on component type
	output.Methodology = GetMethodologyForComponent(input.ServiceType)

	// Get strategy matrix for this component type
	output.StrategyMatrix = GetStrategyForComponent(input.ServiceType)

	// Calculate burn rate config if SLO is provided
	if input.SLOTarget > 0 && input.EnableBurnRate {
		output.BurnRateConfig = CalculateBurnRate(input.SLOTarget, input.SLOWindowDays)
	}

	// Generate suggestions based on methodology and inputs
	output.Suggestions = t.generateSuggestions(input, output)

	// Add warnings for missing recommended fields
	output.Warnings = t.generateWarnings(input)

	// Generate next steps
	output.NextSteps = t.generateNextSteps(input, output)

	// Format and return response
	result, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format response: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(result),
			},
		},
	}, nil
}

// parseAdvancedAlertInput parses and validates input parameters
func parseAdvancedAlertInput(args map[string]interface{}) (*SuggestAlertInput, error) {
	input := &SuggestAlertInput{
		SLOWindowDays:   30,
		CriticalityTier: "tier2_important",
		Environment:     "production",
		EnableBurnRate:  true,
	}

	// Parse service_type
	if st, ok := args["service_type"].(string); ok && st != "" {
		input.ServiceType = ComponentType(st)
	}

	// Parse SLO target
	if slo, ok := args["slo_target"].(float64); ok {
		if slo < 0.9 || slo > 0.99999 {
			return nil, fmt.Errorf("slo_target must be between 0.9 and 0.99999 (90%%-99.999%%)")
		}
		input.SLOTarget = slo
	}

	// Parse SLO window
	if window, ok := args["slo_window_days"].(float64); ok {
		input.SLOWindowDays = int(window)
	}

	// Parse criticality tier
	if tier, ok := args["criticality_tier"].(string); ok && tier != "" {
		input.CriticalityTier = tier
	}

	// Parse is_user_facing
	if uf, ok := args["is_user_facing"].(bool); ok {
		input.IsUserFacing = uf
	}

	// Parse string fields
	if q, ok := args["query"].(string); ok {
		input.Query = q
	}
	if uc, ok := args["use_case"].(string); ok {
		input.UseCase = uc
	}
	if team, ok := args["team"].(string); ok {
		input.Team = team
	}
	if svc, ok := args["service_name"].(string); ok {
		input.ServiceName = svc
	}
	if env, ok := args["environment"].(string); ok {
		input.Environment = env
	}
	if rb, ok := args["runbook_url"].(string); ok {
		input.RunbookURL = rb
	}

	// Parse boolean flags
	if ebr, ok := args["enable_burn_rate"].(bool); ok {
		input.EnableBurnRate = ebr
	}
	if edb, ok := args["enable_dynamic_baselines"].(bool); ok {
		input.EnableDynamicBaselines = edb
	}

	return input, nil
}

// generateSuggestions creates alert suggestions based on input and methodology
func (t *AdvancedSuggestAlertTool) generateSuggestions(input *SuggestAlertInput, output *SuggestAlertOutput) []AdvancedAlertSuggestion {
	suggestions := []AdvancedAlertSuggestion{}

	// Get strategy config for this component type
	strategy := output.StrategyMatrix
	if strategy == nil {
		// Fallback to web_service strategy if none found
		strategy = GetStrategyForComponent(ComponentWebService)
	}

	// Generate use-case based suggestions
	if input.UseCase != "" {
		suggestions = append(suggestions, t.generateUseCaseSuggestions(input, strategy, output)...)
	}

	// Generate query-based suggestions
	if input.Query != "" {
		suggestions = append(suggestions, t.generateQuerySuggestion(input, strategy, output))
	}

	// If SLO-based burn rate alerting is enabled, enhance suggestions
	if output.BurnRateConfig != nil {
		suggestions = t.enhanceWithBurnRate(suggestions, output.BurnRateConfig, input)
	}

	// If no specific suggestions, generate from strategy matrix
	if len(suggestions) == 0 && strategy != nil {
		suggestions = t.generateFromStrategy(input, strategy, output)
	}

	return suggestions
}

// generateUseCaseSuggestions creates suggestions based on use case description
func (t *AdvancedSuggestAlertTool) generateUseCaseSuggestions(input *SuggestAlertInput, _ *AlertStrategyConfig, output *SuggestAlertOutput) []AdvancedAlertSuggestion {
	suggestions := []AdvancedAlertSuggestion{}
	useCaseLower := strings.ToLower(input.UseCase)

	// Error-related alerts (Symptom: Users seeing errors)
	if strings.Contains(useCaseLower, "error") || strings.Contains(useCaseLower, "failure") || strings.Contains(useCaseLower, "exception") {
		severity := ClassifySeverity(input.IsUserFacing, 6.0, input.ServiceType)

		suggestion := AdvancedAlertSuggestion{
			Name:        fmt.Sprintf("%s Error Rate Alert", formatServiceName(input.ServiceName)),
			Description: "Symptom-based alert: Monitors error rate as an indicator of user-facing availability issues",
			Severity:    severity,
			Methodology: output.Methodology,
			Signal:      "errors",
			Query:       `source logs | filter $m.severity >= 5 OR $d.status_code >= 500 | stats count() as error_count by bin(5m)`,
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  10,
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Labels: buildLabels(input, "errors"),
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			RunbookURL:       input.RunbookURL,
			SuggestedActions: GenerateDefaultActions(input.ServiceType, "errors"),
			Explanation: `This alert uses symptom-based alerting (SRE best practice).

Instead of alerting on infrastructure causes (CPU high, disk full), this alerts on
user-visible symptoms (error rate). This approach:
- Reduces alert noise (not all high CPU causes user impact)
- Ensures alerts are actionable (there's definitely a problem users are seeing)
- Follows the Google SRE principle: "Alert on symptoms, not causes"`,
			BestPractices: []string{
				"Alert on error RATE, not absolute count (to handle traffic variations)",
				"Use burn rate alerting with SLO for more nuanced thresholds",
				"Differentiate 4xx (client errors) from 5xx (server errors)",
				"Group alerts by service/endpoint to identify specific problem areas",
			},
			References: []string{
				"Google SRE Handbook - Chapter 5: Monitoring Distributed Systems",
				"'My Philosophy on Alerting' by Rob Ewaschuk",
			},
		}

		// Add default runbook URL if not provided
		if suggestion.RunbookURL == "" {
			suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, "error-rate")
		}

		suggestions = append(suggestions, suggestion)
	}

	// Latency-related alerts (Symptom: Users experiencing slow responses)
	if strings.Contains(useCaseLower, "latency") || strings.Contains(useCaseLower, "slow") ||
		strings.Contains(useCaseLower, "response time") || strings.Contains(useCaseLower, "duration") {
		severity := ClassifySeverity(input.IsUserFacing, 3.0, input.ServiceType)

		suggestion := AdvancedAlertSuggestion{
			Name:        fmt.Sprintf("%s High Latency Alert", formatServiceName(input.ServiceName)),
			Description: "Symptom-based alert: Monitors P99 latency to detect user-facing performance degradation",
			Severity:    severity,
			Methodology: output.Methodology,
			Signal:      "duration",
			Query:       `source logs | filter $d.response_time_ms exists | stats percentile($d.response_time_ms, 99) as p99_latency by bin(5m)`,
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  500, // 500ms default
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Labels: buildLabels(input, "latency"),
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			RunbookURL:       input.RunbookURL,
			SuggestedActions: GenerateDefaultActions(input.ServiceType, "duration"),
			Explanation: `This alert monitors P99 latency (99th percentile).

Why P99 instead of average?
- Average latency hides tail latency issues
- P99 shows what 1% of your users experience (often your most engaged users)
- A system with good average but bad P99 can still feel slow to many users

SLO-based approach: "99.9% of requests complete in under 200ms"`,
			BestPractices: []string{
				"Use percentile latency (P99, P95), not average",
				"Set thresholds based on SLO, not arbitrary values",
				"Different endpoints may need different latency budgets",
				"Consider user-perceived latency, not just server-side",
			},
			References: []string{
				"Google SRE Handbook - Chapter 4: Service Level Objectives",
				"'Latency SLOs Done Right' - Google Cloud Blog",
			},
		}

		if suggestion.RunbookURL == "" {
			suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, "high-latency")
		}

		suggestions = append(suggestions, suggestion)
	}

	// Saturation-related alerts (queue depth, connection pool, etc.)
	if strings.Contains(useCaseLower, "queue") || strings.Contains(useCaseLower, "saturation") ||
		strings.Contains(useCaseLower, "capacity") || strings.Contains(useCaseLower, "exhaustion") ||
		strings.Contains(useCaseLower, "lag") {
		severity := ClassifySeverity(input.IsUserFacing, 3.0, input.ServiceType)

		var query string
		var signal string
		switch input.ServiceType {
		case ComponentMessageQueue:
			query = `source logs | filter $d.component == 'queue' | stats max($d.queue_depth) as depth by $d.queue_name, bin(1m)`
			signal = "saturation"
		case ComponentDatabase:
			query = `source logs | filter $d.component == 'database' | stats avg($d.connections_active / $d.connections_max * 100) as utilization by bin(1m)`
			signal = "utilization"
		default:
			query = `source logs | filter $d.queue_depth exists OR $d.pending_count exists | stats max(coalesce($d.queue_depth, $d.pending_count)) as saturation by bin(1m)`
			signal = "saturation"
		}

		suggestion := AdvancedAlertSuggestion{
			Name:        fmt.Sprintf("%s Saturation Alert", formatServiceName(input.ServiceName)),
			Description: "USE Method alert: Monitors resource saturation as a leading indicator of failures",
			Severity:    severity,
			Methodology: MethodologyUSE,
			Signal:      signal,
			Query:       query,
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  80, // 80% default
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Labels: buildLabels(input, signal),
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			RunbookURL:       input.RunbookURL,
			SuggestedActions: GenerateDefaultActions(input.ServiceType, signal),
			Explanation: `This alert uses the USE Method (Brendan Gregg).

USE = Utilization, Saturation, Errors
- Utilization: How busy is the resource?
- Saturation: Is work queueing? (leading indicator!)
- Errors: Are operations failing?

Saturation alerts are valuable because they predict problems BEFORE they cause errors.`,
			BestPractices: []string{
				"Alert on saturation before utilization hits 100%",
				"Track rate of change, not just absolute value",
				"Different resources have different saturation thresholds",
				"Saturation is a leading indicator - act before errors occur",
			},
			References: []string{
				"'USE Method' by Brendan Gregg",
				"Google SRE Handbook - Four Golden Signals",
			},
		}

		if suggestion.RunbookURL == "" {
			suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, "saturation")
		}

		suggestions = append(suggestions, suggestion)
	}

	// Traffic-related alerts
	if strings.Contains(useCaseLower, "traffic") || strings.Contains(useCaseLower, "rate") ||
		strings.Contains(useCaseLower, "requests") || strings.Contains(useCaseLower, "throughput") {
		suggestion := AdvancedAlertSuggestion{
			Name:        fmt.Sprintf("%s Traffic Anomaly Alert", formatServiceName(input.ServiceName)),
			Description: "Golden Signals alert: Monitors request rate for traffic anomalies",
			Severity:    SeverityP2Warning,
			Methodology: MethodologyGoldenSignals,
			Signal:      "rate",
			Query:       `source logs | filter $d.type == 'request' OR $d.http_method exists | stats count() as request_rate by bin(5m)`,
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  0, // Should use dynamic baseline
				Operator:   "less_than",
				TimeWindow: "10m",
			},
			Labels: buildLabels(input, "traffic"),
			Schedule: AlertSchedule{
				Frequency:     "5m",
				ActiveWindows: "always",
			},
			RunbookURL:       input.RunbookURL,
			SuggestedActions: GenerateDefaultActions(input.ServiceType, "rate"),
			Explanation: `This alert monitors traffic rate as a Golden Signal.

Traffic anomalies can indicate:
- Traffic drop: Upstream service failure, DNS issues, load balancer problems
- Traffic spike: Attack, viral content, misconfigured retry logic

Important: Use DYNAMIC BASELINES for traffic alerts, not static thresholds.
Traffic naturally varies by time of day, day of week, etc.`,
			BestPractices: []string{
				"Use dynamic baselines accounting for seasonality (hour of day, day of week)",
				"Alert on BOTH high and low traffic anomalies",
				"Traffic drops are often more urgent than spikes",
				"Consider business events (sales, releases) in baseline",
			},
			References: []string{
				"Google SRE Handbook - Four Golden Signals",
				"'Alerting on Significant Change' - Google Cloud Blog",
			},
		}

		if suggestion.RunbookURL == "" {
			suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, "traffic-anomaly")
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// generateQuerySuggestion creates a suggestion from a provided query
func (t *AdvancedSuggestAlertTool) generateQuerySuggestion(input *SuggestAlertInput, _ *AlertStrategyConfig, output *SuggestAlertOutput) AdvancedAlertSuggestion {
	// Analyze query to determine appropriate thresholds and signal type
	queryLower := strings.ToLower(input.Query)

	threshold := 10
	timeWindow := "5m"
	signal := "errors"
	severity := SeverityP2Warning

	if strings.Contains(queryLower, "severity >= 6") || strings.Contains(queryLower, "critical") {
		threshold = 1
		timeWindow = "1m"
		severity = SeverityP1Critical
	} else if strings.Contains(queryLower, "severity >= 5") || strings.Contains(queryLower, "error") {
		threshold = 5
		timeWindow = "5m"
	} else if strings.Contains(queryLower, "response_time") || strings.Contains(queryLower, "latency") || strings.Contains(queryLower, "duration") {
		signal = "duration"
		threshold = 500
	} else if strings.Contains(queryLower, "queue") || strings.Contains(queryLower, "depth") {
		signal = "saturation"
		threshold = 100
	} else if strings.Contains(queryLower, "count()") || strings.Contains(queryLower, "rate") {
		signal = "rate"
	}

	if input.IsUserFacing {
		severity = ClassifySeverity(true, 6.0, input.ServiceType)
	}

	suggestion := AdvancedAlertSuggestion{
		Name:        fmt.Sprintf("%s Custom Query Alert", formatServiceName(input.ServiceName)),
		Description: "Alert based on custom query",
		Severity:    severity,
		Methodology: output.Methodology,
		Signal:      signal,
		Query:       input.Query,
		Condition: AlertCondition{
			Type:       "threshold",
			Threshold:  threshold,
			Operator:   "more_than",
			TimeWindow: timeWindow,
		},
		Labels: buildLabels(input, signal),
		Schedule: AlertSchedule{
			Frequency:     "1m",
			ActiveWindows: "always",
		},
		RunbookURL:       input.RunbookURL,
		SuggestedActions: GenerateDefaultActions(input.ServiceType, signal),
		Explanation:      fmt.Sprintf("Custom alert based on your query. Signal type detected: %s", signal),
		BestPractices: []string{
			"Test the query with query_logs before creating the alert",
			"Adjust threshold based on historical baseline",
			"Consider adding grouping (by application, endpoint) to reduce noise",
			"If using burn rate alerting, ensure query returns error counts",
		},
	}

	if suggestion.RunbookURL == "" {
		suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, "custom-query")
	}

	return suggestion
}

// generateFromStrategy creates suggestions from the strategy matrix
func (t *AdvancedSuggestAlertTool) generateFromStrategy(input *SuggestAlertInput, strategy *AlertStrategyConfig, _ *SuggestAlertOutput) []AdvancedAlertSuggestion {
	suggestions := []AdvancedAlertSuggestion{}

	for _, metric := range strategy.RecommendedMetrics {
		severity := ClassifySeverity(input.IsUserFacing, 3.0, input.ServiceType)

		suggestion := AdvancedAlertSuggestion{
			Name:        fmt.Sprintf("%s %s Alert", formatServiceName(input.ServiceName), cases.Title(language.English).String(metric.Name)),
			Description: metric.Description,
			Severity:    severity,
			Methodology: strategy.Methodology,
			Signal:      metric.Signal,
			Query:       metric.Query,
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  int(metric.DefaultThreshold),
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Labels: buildLabels(input, metric.Signal),
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			RunbookURL:       input.RunbookURL,
			SuggestedActions: GenerateDefaultActions(input.ServiceType, metric.Signal),
			Explanation:      fmt.Sprintf("%s Method metric: %s (%s)", strategy.Methodology, metric.Name, metric.Unit),
			BestPractices:    metric.BestPractices,
		}

		if suggestion.RunbookURL == "" {
			suggestion.RunbookURL = GenerateRunbookURL(input.ServiceType, metric.Name)
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// enhanceWithBurnRate adds burn rate alerting configuration to suggestions
func (t *AdvancedSuggestAlertTool) enhanceWithBurnRate(suggestions []AdvancedAlertSuggestion, burnRate *BurnRateConfig, input *SuggestAlertInput) []AdvancedAlertSuggestion {
	enhanced := []AdvancedAlertSuggestion{}

	for _, suggestion := range suggestions {
		// Only enhance error-rate type alerts with burn rate
		if suggestion.Signal != "errors" {
			enhanced = append(enhanced, suggestion)
			continue
		}

		// Add fast burn alert (P1 - page)
		if len(burnRate.FastBurnWindows) > 0 {
			fastBurn := burnRate.FastBurnWindows[0]
			threshold := GetBurnRateThreshold(burnRate.SLO.Target, fastBurn.BurnRate)

			fastBurnSuggestion := suggestion
			fastBurnSuggestion.Name = fmt.Sprintf("%s - Fast Burn (Page)", suggestion.Name)
			fastBurnSuggestion.Severity = SeverityP1Critical
			fastBurnSuggestion.BurnRateCondition = &BurnRateCondition{
				SLOTarget:          burnRate.SLO.Target,
				ErrorBudgetPercent: threshold * 100,
				BurnRate:           fastBurn.BurnRate,
				WindowDuration:     formatDuration(fastBurn.Duration),
				ConsumptionPercent: 2.0, // 2% budget in 1 hour
			}
			fastBurnSuggestion.Windows = []AlertWindow{
				{Duration: "1h", BurnRate: fastBurn.BurnRate, Type: "short"},
				{Duration: "5m", BurnRate: fastBurn.BurnRate, Type: "short"}, // Short window for confirmation
			}
			fastBurnSuggestion.Explanation = FormatBurnRateExplanation(
				burnRate.SLO.Target, fastBurn.BurnRate, fastBurn.Duration, input.SLOWindowDays,
			)
			fastBurnSuggestion.BestPractices = append(fastBurnSuggestion.BestPractices,
				"Fast burn alert: pages on-call immediately",
				"Requires both long and short window to fire (reduces flapping)",
				fmt.Sprintf("At %.1fx burn rate, you'd exhaust the error budget in %.1f days",
					fastBurn.BurnRate, float64(input.SLOWindowDays)/fastBurn.BurnRate),
			)

			enhanced = append(enhanced, fastBurnSuggestion)
		}

		// Add slow burn alert (P2 - ticket)
		if len(burnRate.SlowBurnWindows) > 0 {
			slowBurn := burnRate.SlowBurnWindows[0]
			threshold := GetBurnRateThreshold(burnRate.SLO.Target, slowBurn.BurnRate)

			slowBurnSuggestion := suggestion
			slowBurnSuggestion.Name = fmt.Sprintf("%s - Slow Burn (Ticket)", suggestion.Name)
			slowBurnSuggestion.Severity = SeverityP2Warning
			slowBurnSuggestion.BurnRateCondition = &BurnRateCondition{
				SLOTarget:          burnRate.SLO.Target,
				ErrorBudgetPercent: threshold * 100,
				BurnRate:           slowBurn.BurnRate,
				WindowDuration:     formatDuration(slowBurn.Duration),
				ConsumptionPercent: 10.0, // 10% budget in 24 hours
			}
			slowBurnSuggestion.Windows = []AlertWindow{
				{Duration: "24h", BurnRate: slowBurn.BurnRate, Type: "long"},
				{Duration: "6h", BurnRate: slowBurn.BurnRate, Type: "long"},
			}
			slowBurnSuggestion.Explanation = FormatBurnRateExplanation(
				burnRate.SLO.Target, slowBurn.BurnRate, slowBurn.Duration, input.SLOWindowDays,
			)
			slowBurnSuggestion.BestPractices = append(slowBurnSuggestion.BestPractices,
				"Slow burn alert: creates ticket for next business day",
				"Detects gradual degradation before it becomes critical",
				fmt.Sprintf("At %.1fx burn rate, error budget would be exhausted in %d days",
					slowBurn.BurnRate, input.SLOWindowDays),
			)

			enhanced = append(enhanced, slowBurnSuggestion)
		}
	}

	return enhanced
}

// generateWarnings creates warnings for missing recommended fields
func (t *AdvancedSuggestAlertTool) generateWarnings(input *SuggestAlertInput) []string {
	warnings := []string{}

	// Runbook URL - strongly recommended but not required
	if input.RunbookURL == "" {
		warnings = append(warnings,
			"⚠️  No runbook_url provided. A runbook is strongly recommended for actionable alerts. "+
				"Default runbook templates have been generated. Consider documenting:\n"+
				"   - Initial triage steps\n"+
				"   - Investigation procedures\n"+
				"   - Escalation paths\n"+
				"   - Known failure modes and mitigations")
	}

	// SLO not provided
	if input.SLOTarget == 0 {
		warnings = append(warnings,
			"⚠️  No slo_target provided. Static thresholds are used instead of burn rate alerting. "+
				"Consider defining an SLO (e.g., 99.9% availability) for more intelligent alerting that "+
				"accounts for error budget consumption rate.")
	}

	// Service name not provided
	if input.ServiceName == "" {
		warnings = append(warnings,
			"ℹ️  No service_name provided. Alert names will use generic labels. "+
				"Providing a service name improves alert routing and identification.")
	}

	// Team not provided
	if input.Team == "" {
		warnings = append(warnings,
			"ℹ️  No team provided. Alert routing may be ambiguous. "+
				"Consider specifying the responsible team for proper escalation.")
	}

	// User-facing not specified for web service
	if (input.ServiceType == ComponentWebService || input.ServiceType == ComponentAPIGateway) && !input.IsUserFacing {
		warnings = append(warnings,
			"ℹ️  is_user_facing not set for web service. If this service directly serves end users, "+
				"set is_user_facing=true for appropriate severity classification (P1 alerts).")
	}

	return warnings
}

// generateNextSteps creates recommended next steps
func (t *AdvancedSuggestAlertTool) generateNextSteps(input *SuggestAlertInput, _ *SuggestAlertOutput) []string {
	steps := []string{}

	// Always recommend testing query first
	steps = append(steps, "1. Test the suggested queries using query_logs to verify they return expected results")

	// Review and customize
	steps = append(steps, "2. Review and customize thresholds based on your service's historical baseline")

	// Create runbook if missing
	if input.RunbookURL == "" {
		steps = append(steps, "3. Create a runbook documenting investigation and remediation steps for each alert")
	}

	// Create alert definition
	steps = append(steps, fmt.Sprintf("%d. Use create_alert_definition to create the alert definition with the suggested configuration",
		len(steps)+1))

	// Set up notifications
	steps = append(steps, fmt.Sprintf("%d. Use list_outgoing_webhooks to find notification targets, then create_alert to link them",
		len(steps)+1))

	// SLO recommendation
	if input.SLOTarget == 0 {
		steps = append(steps, fmt.Sprintf("%d. Consider defining an SLO and re-running with slo_target for burn rate alerting",
			len(steps)+1))
	}

	return steps
}

// Helper functions

func formatServiceName(name string) string {
	if name == "" {
		return "Service"
	}
	// Capitalize first letter of each word
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func buildLabels(input *SuggestAlertInput, signal string) map[string]string {
	labels := map[string]string{
		"signal":      signal,
		"methodology": string(GetMethodologyForComponent(input.ServiceType)),
	}

	if input.Team != "" {
		labels["team"] = input.Team
	}
	if input.ServiceName != "" {
		labels["service"] = input.ServiceName
	}
	if input.Environment != "" {
		labels["environment"] = input.Environment
	}
	if input.CriticalityTier != "" {
		labels["criticality"] = input.CriticalityTier
	}

	return labels
}
