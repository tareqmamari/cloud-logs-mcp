// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file defines types for the smart investigation system.
package tools

import (
	"sort"
	"strings"
	"time"
)

// InvestigationMode defines the scope of investigation
type InvestigationMode string

// Investigation mode constants
const (
	ModeGlobal    InvestigationMode = "global"    // System-wide health scan
	ModeComponent InvestigationMode = "component" // Single service focus
	ModeFlow      InvestigationMode = "flow"      // Request tracing
)

// SmartInvestigationContext holds the state of an ongoing investigation
type SmartInvestigationContext struct {
	Mode          InvestigationMode
	TimeRange     InvestigationTimeRange
	TargetService string // For component mode
	TraceID       string // For flow mode
	CorrelationID string // For flow mode
	Findings      []InvestigationFinding
	Hypotheses    []InvestigationHypothesis
	NextActions   []HeuristicAction
	QueryHistory  []ExecutedQuery
	EvidenceChain []Evidence
}

// InvestigationTimeRange defines the temporal scope
type InvestigationTimeRange struct {
	Start time.Time
	End   time.Time
}

// InvestigationFinding represents a discovered fact during investigation
type InvestigationFinding struct {
	Timestamp   time.Time
	Type        FindingType
	Service     string
	Summary     string
	Evidence    string
	Severity    InvestigationSeverity
	Confidence  float64 // 0.0 - 1.0
	QuerySource string  // Which query produced this finding
}

// FindingType categorizes findings
type FindingType string

// Finding type constants
const (
	FindingError      FindingType = "error"
	FindingLatency    FindingType = "latency"
	FindingResource   FindingType = "resource"
	FindingDependency FindingType = "dependency"
	FindingDeployment FindingType = "deployment"
	FindingSpike      FindingType = "spike"
)

// InvestigationSeverity levels
type InvestigationSeverity string

// Severity level constants
const (
	SeverityCritical InvestigationSeverity = "critical"
	SeverityHigh     InvestigationSeverity = "high"
	SeverityMedium   InvestigationSeverity = "medium"
	SeverityLow      InvestigationSeverity = "low"
)

// InvestigationHypothesis represents a potential root cause
type InvestigationHypothesis struct {
	ID          string
	Description string
	Confidence  float64
	Evidence    []string
	TestQuery   string // Query to validate/invalidate this hypothesis
}

// Evidence represents proof for conclusions
type Evidence struct {
	Timestamp   time.Time
	Type        string
	Description string
	DataPoints  []DataPoint
	Query       string // Query that produced this evidence
}

// DataPoint holds a metric value
type DataPoint struct {
	Metric string
	Value  interface{}
	Unit   string
}

// HeuristicAction represents an auto-suggested next step
type HeuristicAction struct {
	Priority    int
	Type        ActionType
	Description string
	Query       string
	Rationale   string
}

// ActionType categorizes actions
type ActionType string

// Action type constants
const (
	ActionQuery     ActionType = "query"
	ActionDrillDown ActionType = "drill_down"
	ActionCorrelate ActionType = "correlate"
	ActionTrace     ActionType = "trace"
	ActionAlert     ActionType = "create_alert"
)

// QueryPlan represents a planned query execution
type QueryPlan struct {
	ID        string
	Query     string
	Purpose   string
	Tier      string
	Priority  int
	DependsOn []string // IDs of queries that must complete first
}

// ExecutedQuery holds the result of an executed query
type ExecutedQuery struct {
	QueryID  string
	Query    string
	Events   []map[string]interface{}
	Metadata map[string]interface{}
	Duration time.Duration
	Error    error
}

// EvidenceSummary is the synthesized output of an investigation
type EvidenceSummary struct {
	RootCause        string
	Timeline         []TimelineEvent
	AffectedServices []string
	ImpactSummary    string
	Confidence       float64
	Recommendations  []InvestigationRecommendation
}

// TimelineEvent represents a point in the incident timeline
type TimelineEvent struct {
	Timestamp    time.Time
	Event        string
	Service      string
	Significance string
}

// InvestigationRecommendation provides actionable guidance
type InvestigationRecommendation struct {
	Priority  int
	Category  string
	Action    string
	Rationale string
}

// QueryStrategy defines the interface for investigation strategies
type QueryStrategy interface {
	// Name returns the strategy identifier
	Name() string

	// InitialQueries returns the queries to execute first
	InitialQueries(ctx *SmartInvestigationContext) []QueryPlan

	// AnalyzeResults processes query results and returns findings
	AnalyzeResults(ctx *SmartInvestigationContext, results []ExecutedQuery) []InvestigationFinding

	// SuggestNextActions returns heuristic-driven next steps
	SuggestNextActions(ctx *SmartInvestigationContext) []HeuristicAction

	// SynthesizeEvidence creates the evidence summary
	SynthesizeEvidence(ctx *SmartInvestigationContext) *EvidenceSummary
}

// Helper functions for investigation types

// categorizeSeverityByCount returns severity based on error count
func categorizeSeverityByCount(count float64) InvestigationSeverity {
	switch {
	case count > 500:
		return SeverityCritical
	case count > 100:
		return SeverityHigh
	case count > 20:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// extractMessageFromEvent extracts the message from an event map
func extractMessageFromEvent(event map[string]interface{}) string {
	// Try direct message field
	if msg, ok := event["message"].(string); ok {
		return msg
	}

	// Try user_data
	if userData, ok := event["user_data"].(map[string]interface{}); ok {
		if msg, ok := userData["message"].(string); ok {
			return msg
		}
		if eventData, ok := userData["event"].(map[string]interface{}); ok {
			if msg, ok := eventData["_message"].(string); ok {
				return msg
			}
		}
	}

	// Try text field
	if text, ok := event["text"].(string); ok {
		return text
	}

	return ""
}

// normalizeMessageForPattern normalizes a message for pattern matching
func normalizeMessageForPattern(msg string) string {
	msg = strings.ToLower(msg)
	if len(msg) > 80 {
		msg = msg[:80]
	}
	return msg
}

// truncateText truncates a string to maxLen
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// synthesizeRootCauseFromFindings creates a root cause statement from findings
func synthesizeRootCauseFromFindings(findings []InvestigationFinding) string {
	if len(findings) == 0 {
		return "No root cause identified"
	}

	// Group findings by type
	var errorFindings, latencyFindings, dependencyFindings []InvestigationFinding

	for _, f := range findings {
		switch f.Type {
		case FindingError, FindingSpike:
			errorFindings = append(errorFindings, f)
		case FindingLatency:
			latencyFindings = append(latencyFindings, f)
		case FindingDependency:
			dependencyFindings = append(dependencyFindings, f)
		}
	}

	// Prioritize dependency issues as they often cause other problems
	if len(dependencyFindings) > 0 {
		return "Dependency failure: " + dependencyFindings[0].Summary
	}

	if len(errorFindings) > 0 {
		return "Error pattern: " + errorFindings[0].Summary
	}

	if len(latencyFindings) > 0 {
		return "Performance degradation: " + latencyFindings[0].Summary
	}

	return findings[0].Summary
}

// calculateConfidenceFromFindings calculates overall confidence from findings
func calculateConfidenceFromFindings(findings []InvestigationFinding) float64 {
	if len(findings) == 0 {
		return 0.0
	}

	total := 0.0
	for _, f := range findings {
		total += f.Confidence
	}
	return total / float64(len(findings))
}

// sortFindingsBySeverity sorts findings by severity (critical first)
func sortFindingsBySeverity(findings []InvestigationFinding) {
	severityOrder := map[InvestigationSeverity]int{
		SeverityCritical: 0,
		SeverityHigh:     1,
		SeverityMedium:   2,
		SeverityLow:      3,
	}

	sort.Slice(findings, func(i, j int) bool {
		return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
	})
}

// sortActionsByPriority sorts actions by priority (lower number = higher priority)
func sortActionsByPriority(actions []HeuristicAction) {
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Priority < actions[j].Priority
	})
}
