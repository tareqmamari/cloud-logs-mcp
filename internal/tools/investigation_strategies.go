// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements investigation strategies for different modes.
package tools

import (
	"fmt"
	"strings"
	"time"
)

// QueryStrategyFactory creates the appropriate strategy based on mode
type QueryStrategyFactory struct{}

// NewQueryStrategyFactory creates a new factory
func NewQueryStrategyFactory() *QueryStrategyFactory {
	return &QueryStrategyFactory{}
}

// CreateStrategy returns the appropriate strategy for the given mode
func (f *QueryStrategyFactory) CreateStrategy(mode InvestigationMode) QueryStrategy {
	switch mode {
	case ModeGlobal:
		return &GlobalModeStrategy{}
	case ModeComponent:
		return &ComponentModeStrategy{}
	case ModeFlow:
		return &FlowModeStrategy{}
	default:
		return &GlobalModeStrategy{}
	}
}

// DetermineMode analyzes parameters to determine the best investigation mode
func (f *QueryStrategyFactory) DetermineMode(params map[string]interface{}) InvestigationMode {
	// If trace_id or correlation_id provided, use flow mode
	if traceID, _ := params["trace_id"].(string); traceID != "" {
		return ModeFlow
	}
	if corrID, _ := params["correlation_id"].(string); corrID != "" {
		return ModeFlow
	}

	// If specific application provided, use component mode
	if app, _ := params["application"].(string); app != "" {
		return ModeComponent
	}

	// Default to global mode
	return ModeGlobal
}

// ========================================================================
// GlobalModeStrategy - System-wide health scanning
// ========================================================================

// GlobalModeStrategy implements system-wide health scanning
type GlobalModeStrategy struct{}

// Name returns the strategy identifier
func (s *GlobalModeStrategy) Name() string {
	return string(ModeGlobal)
}

// InitialQueries returns the queries for global mode investigation
func (s *GlobalModeStrategy) InitialQueries(_ *SmartInvestigationContext) []QueryPlan {
	return []QueryPlan{
		{
			ID:       "global-error-rate",
			Priority: 1,
			Purpose:  "Calculate error rate per application",
			Query: `source logs
				| filter $m.severity >= ERROR
				| groupby $l.applicationname aggregate count() as error_count
				| sortby -error_count
				| limit 20`,
			Tier: "archive",
		},
		{
			ID:       "global-error-timeline",
			Priority: 1,
			Purpose:  "Error distribution over time",
			Query: `source logs
				| filter $m.severity >= WARNING
				| groupby formatTimestamp($m.timestamp, '%Y-%m-%d %H:%M') as time_bucket aggregate count() as errors
				| sortby time_bucket`,
			Tier: "archive",
		},
		{
			ID:       "global-critical-errors",
			Priority: 1,
			Purpose:  "Identify critical severity events",
			Query: `source logs
				| filter $m.severity == CRITICAL
				| limit 50`,
			Tier: "archive",
		},
	}
}

// AnalyzeResults processes query results for global mode
func (s *GlobalModeStrategy) AnalyzeResults(_ *SmartInvestigationContext, results []ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	for _, result := range results {
		switch result.QueryID {
		case "global-error-rate":
			findings = append(findings, s.analyzeErrorRates(result)...)
		case "global-error-timeline":
			findings = append(findings, s.analyzeErrorTimeline(result)...)
		case "global-critical-errors":
			findings = append(findings, s.analyzeCriticalErrors(result)...)
		}
	}

	return findings
}

func (s *GlobalModeStrategy) analyzeErrorRates(result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	for _, event := range result.Events {
		appName := getStringFromEvent(event, "applicationname", "$l.applicationname")
		errorCount := getFloatFromEvent(event, "error_count")

		if errorCount > 10 {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingError,
				Service:     appName,
				Summary:     fmt.Sprintf("High error volume: %d errors in time window", int(errorCount)),
				Severity:    categorizeSeverityByCount(errorCount),
				Confidence:  0.9,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

func (s *GlobalModeStrategy) analyzeErrorTimeline(result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	if len(result.Events) < 3 {
		return findings
	}

	// Calculate average and detect spikes
	var total float64
	for _, event := range result.Events {
		errors := getFloatFromEvent(event, "errors")
		total += errors
	}
	avg := total / float64(len(result.Events))

	for _, event := range result.Events {
		errors := getFloatFromEvent(event, "errors")
		timeBucket := getStringFromEvent(event, "time_bucket", "")

		// Spike detection: 3x average
		if errors > avg*3 && errors > 10 {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingSpike,
				Summary:     fmt.Sprintf("Error spike at %s: %.0f errors (%.0fx average)", timeBucket, errors, errors/avg),
				Evidence:    fmt.Sprintf("Average: %.1f errors/5min, Spike: %.0f errors", avg, errors),
				Severity:    SeverityHigh,
				Confidence:  0.85,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

func (s *GlobalModeStrategy) analyzeCriticalErrors(result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	// Group critical errors by message pattern
	messagePatterns := make(map[string]int)
	for _, event := range result.Events {
		msg := extractMessageFromEvent(event)
		pattern := normalizeMessageForPattern(msg)
		if pattern != "" {
			messagePatterns[pattern]++
		}
	}

	for pattern, count := range messagePatterns {
		if count >= 3 {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingError,
				Summary:     fmt.Sprintf("Recurring critical error: %s (%d occurrences)", truncateText(pattern, 60), count),
				Severity:    SeverityCritical,
				Confidence:  0.95,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

// SuggestNextActions returns heuristic-driven next steps for global mode
func (s *GlobalModeStrategy) SuggestNextActions(ctx *SmartInvestigationContext) []HeuristicAction {
	actions := []HeuristicAction{}
	seen := make(map[string]bool)

	for _, finding := range ctx.Findings {
		if finding.Service != "" && !seen[finding.Service] {
			seen[finding.Service] = true
			actions = append(actions, HeuristicAction{
				Priority:    1,
				Type:        ActionDrillDown,
				Description: fmt.Sprintf("Drill down into %s errors", finding.Service),
				Query: fmt.Sprintf(`source logs
					| filter $l.applicationname == '%s' && $m.severity >= ERROR
					| limit 100`, finding.Service),
				Rationale: "High error volume warrants detailed investigation",
			})
		}
	}

	return actions
}

// SynthesizeEvidence creates the evidence summary for global mode
func (s *GlobalModeStrategy) SynthesizeEvidence(ctx *SmartInvestigationContext) *EvidenceSummary {
	summary := &EvidenceSummary{
		Timeline:         []TimelineEvent{},
		AffectedServices: []string{},
		Recommendations:  []InvestigationRecommendation{},
	}

	// Collect affected services
	serviceSet := make(map[string]bool)
	var criticalFindings []InvestigationFinding

	for _, f := range ctx.Findings {
		if f.Service != "" {
			serviceSet[f.Service] = true
		}
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			criticalFindings = append(criticalFindings, f)
		}
	}

	for svc := range serviceSet {
		summary.AffectedServices = append(summary.AffectedServices, svc)
	}

	// Synthesize root cause
	if len(criticalFindings) > 0 {
		summary.RootCause = synthesizeRootCauseFromFindings(criticalFindings)
		summary.Confidence = calculateConfidenceFromFindings(criticalFindings)
	} else if len(ctx.Findings) > 0 {
		summary.RootCause = synthesizeRootCauseFromFindings(ctx.Findings)
		summary.Confidence = calculateConfidenceFromFindings(ctx.Findings)
	} else {
		summary.RootCause = "No critical issues detected. System appears healthy."
		summary.Confidence = 0.7
	}

	// Impact summary
	summary.ImpactSummary = fmt.Sprintf("%d services affected, %d findings identified",
		len(summary.AffectedServices), len(ctx.Findings))

	return summary
}

// ========================================================================
// ComponentModeStrategy - Single service deep dive
// ========================================================================

// ComponentModeStrategy implements single service deep dive
type ComponentModeStrategy struct{}

// Name returns the strategy identifier
func (s *ComponentModeStrategy) Name() string {
	return string(ModeComponent)
}

// InitialQueries returns the queries for component mode investigation
func (s *ComponentModeStrategy) InitialQueries(ctx *SmartInvestigationContext) []QueryPlan {
	svc := ctx.TargetService

	return []QueryPlan{
		{
			ID:       "component-errors",
			Priority: 1,
			Purpose:  fmt.Sprintf("All errors from %s", svc),
			Query: fmt.Sprintf(`source logs
				| filter $l.applicationname == '%s' && $m.severity >= ERROR
				| limit 200`, svc),
			Tier: "archive",
		},
		{
			ID:       "component-error-patterns",
			Priority: 1,
			Purpose:  "Group errors by message pattern",
			Query: fmt.Sprintf(`source logs
				| filter $l.applicationname == '%s' && $m.severity >= ERROR
				| groupby $d.message aggregate count() as occurrences
				| sortby -occurrences
				| limit 20`, svc),
			Tier: "archive",
		},
		{
			ID:       "component-subsystems",
			Priority: 2,
			Purpose:  "Error distribution by subsystem",
			Query: fmt.Sprintf(`source logs
				| filter $l.applicationname == '%s' && $m.severity >= WARNING
				| groupby $l.subsystemname aggregate count() as errors
				| sortby -errors`, svc),
			Tier: "archive",
		},
		{
			ID:        "component-dependencies",
			Priority:  3,
			Purpose:   "Identify downstream calls and failures",
			DependsOn: []string{"component-errors"},
			Query: fmt.Sprintf(`source logs
				| filter $l.applicationname == '%s'
				  && ($d.message.contains('connection')
					  || $d.message.contains('timeout')
					  || $d.message.contains('refused'))
				| limit 100`, svc),
			Tier: "archive",
		},
	}
}

// AnalyzeResults processes query results for component mode
func (s *ComponentModeStrategy) AnalyzeResults(ctx *SmartInvestigationContext, results []ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	for _, result := range results {
		switch result.QueryID {
		case "component-error-patterns":
			findings = append(findings, s.analyzeErrorPatterns(ctx, result)...)
		case "component-dependencies":
			findings = append(findings, s.analyzeDependencyIssues(ctx, result)...)
		case "component-subsystems":
			findings = append(findings, s.analyzeSubsystems(ctx, result)...)
		}
	}

	return findings
}

func (s *ComponentModeStrategy) analyzeErrorPatterns(ctx *SmartInvestigationContext, result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	for i, event := range result.Events {
		if i >= 5 { // Top 5 patterns
			break
		}

		msg := getStringFromEvent(event, "message", "$d.message")
		count := getFloatFromEvent(event, "occurrences")

		if count > 5 {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingError,
				Service:     ctx.TargetService,
				Summary:     fmt.Sprintf("Recurring error pattern: %s", truncateText(msg, 80)),
				Evidence:    fmt.Sprintf("%d occurrences", int(count)),
				Severity:    categorizeSeverityByCount(count),
				Confidence:  0.9,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

func (s *ComponentModeStrategy) analyzeDependencyIssues(ctx *SmartInvestigationContext, result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	// Pattern to description mapping
	patterns := map[string]string{
		"connection refused":   "Network/service connectivity failure",
		"timeout":              "Downstream service not responding",
		"econnreset":           "Connection reset by peer",
		"etimedout":            "Connection timed out",
		"pool exhausted":       "Connection pool exhaustion",
		"deadlock":             "Database deadlock detected",
		"too many connections": "Connection limit exceeded",
	}

	patternCounts := make(map[string]int)
	for _, event := range result.Events {
		msg := strings.ToLower(extractMessageFromEvent(event))
		for pattern := range patterns {
			if strings.Contains(msg, pattern) {
				patternCounts[pattern]++
			}
		}
	}

	for pattern, count := range patternCounts {
		if count >= 3 {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingDependency,
				Service:     ctx.TargetService,
				Summary:     fmt.Sprintf("%s - %s", patterns[pattern], pattern),
				Evidence:    fmt.Sprintf("%d occurrences detected", count),
				Severity:    SeverityHigh,
				Confidence:  0.85,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

func (s *ComponentModeStrategy) analyzeSubsystems(ctx *SmartInvestigationContext, result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	// Find subsystems with high error counts
	for _, event := range result.Events {
		subsystem := getStringFromEvent(event, "subsystemname", "$l.subsystemname")
		errors := getFloatFromEvent(event, "errors")

		if errors > 20 && subsystem != "" {
			findings = append(findings, InvestigationFinding{
				Timestamp:   time.Now(),
				Type:        FindingError,
				Service:     fmt.Sprintf("%s/%s", ctx.TargetService, subsystem),
				Summary:     fmt.Sprintf("High error count in subsystem %s: %d errors", subsystem, int(errors)),
				Severity:    categorizeSeverityByCount(errors),
				Confidence:  0.8,
				QuerySource: result.QueryID,
			})
		}
	}

	return findings
}

// SuggestNextActions returns heuristic-driven next steps for component mode
func (s *ComponentModeStrategy) SuggestNextActions(ctx *SmartInvestigationContext) []HeuristicAction {
	actions := []HeuristicAction{}

	for _, finding := range ctx.Findings {
		if finding.Type == FindingDependency {
			actions = append(actions, HeuristicAction{
				Priority:    1,
				Type:        ActionCorrelate,
				Description: "Check downstream service health",
				Rationale:   fmt.Sprintf("Dependency issue detected: %s", finding.Summary),
			})
		}
	}

	// Always suggest checking recent deployments
	actions = append(actions, HeuristicAction{
		Priority:    3,
		Type:        ActionQuery,
		Description: "Check for recent deployment correlations",
		Rationale:   "Errors may correlate with recent code changes",
	})

	return actions
}

// SynthesizeEvidence creates the evidence summary for component mode
func (s *ComponentModeStrategy) SynthesizeEvidence(ctx *SmartInvestigationContext) *EvidenceSummary {
	summary := &EvidenceSummary{
		AffectedServices: []string{ctx.TargetService},
	}

	// Find the dominant finding
	var dominantFinding *InvestigationFinding
	maxConfidence := 0.0

	for i := range ctx.Findings {
		f := &ctx.Findings[i]
		if f.Confidence > maxConfidence {
			dominantFinding = f
			maxConfidence = f.Confidence
		}
	}

	if dominantFinding != nil {
		summary.RootCause = dominantFinding.Summary
		summary.Confidence = dominantFinding.Confidence
	} else {
		summary.RootCause = fmt.Sprintf("No significant issues found in %s", ctx.TargetService)
		summary.Confidence = 0.6
	}

	summary.ImpactSummary = fmt.Sprintf("%d findings for service %s", len(ctx.Findings), ctx.TargetService)

	return summary
}

// ========================================================================
// FlowModeStrategy - Request tracing across services
// ========================================================================

// FlowModeStrategy implements request tracing across services
type FlowModeStrategy struct{}

// Name returns the strategy identifier
func (s *FlowModeStrategy) Name() string {
	return string(ModeFlow)
}

// InitialQueries returns the queries for flow mode investigation
func (s *FlowModeStrategy) InitialQueries(ctx *SmartInvestigationContext) []QueryPlan {
	queries := []QueryPlan{}

	// Query by trace_id if available
	if ctx.TraceID != "" {
		queries = append(queries, QueryPlan{
			ID:       "flow-by-trace",
			Priority: 1,
			Purpose:  "Trace request flow by trace_id",
			Query: fmt.Sprintf(`source logs
				| filter $d.trace_id == '%s'
				| sortby $m.timestamp asc
				| limit 500`, ctx.TraceID),
			Tier: "archive",
		})
	}

	// Query by correlation_id if available
	if ctx.CorrelationID != "" {
		queries = append(queries, QueryPlan{
			ID:       "flow-by-correlation",
			Priority: 1,
			Purpose:  "Trace request flow by correlation_id",
			Query: fmt.Sprintf(`source logs
				| filter $d.correlation_id == '%s'
				| sortby $m.timestamp asc
				| limit 500`, ctx.CorrelationID),
			Tier: "archive",
		})
	}

	// If neither provided, return empty (caller should validate)
	return queries
}

// AnalyzeResults processes query results for flow mode
func (s *FlowModeStrategy) AnalyzeResults(_ *SmartInvestigationContext, results []ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	for _, result := range results {
		if result.QueryID == "flow-by-trace" || result.QueryID == "flow-by-correlation" {
			findings = append(findings, s.analyzeRequestFlow(result)...)
		}
	}

	return findings
}

func (s *FlowModeStrategy) analyzeRequestFlow(result ExecutedQuery) []InvestigationFinding {
	findings := []InvestigationFinding{}

	// Build timeline and identify where errors occur
	var services []string
	var firstError *map[string]interface{}
	serviceSet := make(map[string]bool)

	for i := range result.Events {
		event := result.Events[i]
		svc := getStringFromEvent(event, "applicationname", "$l.applicationname")
		if svc == "" {
			svc = getStringFromEvent(event, "app", "")
		}

		if svc != "" && !serviceSet[svc] {
			serviceSet[svc] = true
			services = append(services, svc)
		}

		severity := getFloatFromEvent(event, "severity")
		if severity == 0 {
			// Try metadata
			if meta, ok := event["metadata"].(map[string]interface{}); ok {
				severity = getFloatFromMap(meta, "severity")
			}
		}

		if severity >= 5 && firstError == nil {
			firstError = &event
		}
	}

	if firstError != nil {
		svc := getStringFromEvent(*firstError, "applicationname", "$l.applicationname")
		msg := extractMessageFromEvent(*firstError)

		findings = append(findings, InvestigationFinding{
			Timestamp:   time.Now(),
			Type:        FindingError,
			Service:     svc,
			Summary:     fmt.Sprintf("Request failed at %s: %s", svc, truncateText(msg, 80)),
			Evidence:    fmt.Sprintf("Request traversed: %s", strings.Join(services, " → ")),
			Severity:    SeverityHigh,
			Confidence:  0.9,
			QuerySource: result.QueryID,
		})
	} else if len(services) > 0 {
		// No error found, but we have the flow
		findings = append(findings, InvestigationFinding{
			Timestamp:   time.Now(),
			Type:        FindingError,
			Summary:     "Request flow traced successfully, no errors detected",
			Evidence:    fmt.Sprintf("Services: %s", strings.Join(services, " → ")),
			Severity:    SeverityLow,
			Confidence:  0.7,
			QuerySource: result.QueryID,
		})
	}

	return findings
}

// SuggestNextActions returns heuristic-driven next steps for flow mode
func (s *FlowModeStrategy) SuggestNextActions(ctx *SmartInvestigationContext) []HeuristicAction {
	actions := []HeuristicAction{}

	for _, finding := range ctx.Findings {
		if finding.Service != "" && finding.Severity != SeverityLow {
			actions = append(actions, HeuristicAction{
				Priority:    1,
				Type:        ActionDrillDown,
				Description: fmt.Sprintf("Investigate %s service in detail", finding.Service),
				Query: fmt.Sprintf(`source logs
					| filter $l.applicationname == '%s' && $m.severity >= ERROR
					| limit 100`, finding.Service),
				Rationale: "Request failed at this service",
			})
		}
	}

	return actions
}

// SynthesizeEvidence creates the evidence summary for flow mode
func (s *FlowModeStrategy) SynthesizeEvidence(ctx *SmartInvestigationContext) *EvidenceSummary {
	summary := &EvidenceSummary{
		AffectedServices: []string{},
	}

	if len(ctx.Findings) > 0 {
		summary.RootCause = ctx.Findings[0].Summary
		if ctx.Findings[0].Service != "" {
			summary.AffectedServices = []string{ctx.Findings[0].Service}
		}
		summary.Confidence = ctx.Findings[0].Confidence
	} else {
		summary.RootCause = "Unable to trace request flow"
		summary.Confidence = 0.3
	}

	summary.ImpactSummary = fmt.Sprintf("%d findings from flow analysis", len(ctx.Findings))

	return summary
}

// ========================================================================
// Helper functions
// ========================================================================

// getStringFromEvent extracts a string value from an event, trying multiple keys
func getStringFromEvent(event map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := event[key].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// getFloatFromEvent extracts a float value from an event
func getFloatFromEvent(event map[string]interface{}, key string) float64 {
	if val, ok := event[key].(float64); ok {
		return val
	}
	return 0
}

// getFloatFromMap extracts a float value from a map
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}
