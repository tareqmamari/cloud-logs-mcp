// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements the Config Change Correlation Tool for causal pivoting,
// enabling AI agents to automatically correlate incidents with configuration,
// deployment, and IAM changes in the same timeframe.
package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ============================================================================
// CONFIG CHANGE CORRELATION TOOL (Causal Pivoting - SOTA 2025)
// ============================================================================

// ChangeType represents the type of configuration change.
// Custom change types can be registered using RegisterChangeType.
type ChangeType string

// Built-in change types
const (
	ChangeTypeDeployment    ChangeType = "DEPLOYMENT"
	ChangeTypeConfig        ChangeType = "CONFIG"
	ChangeTypeIAM           ChangeType = "IAM"
	ChangeTypeScaling       ChangeType = "SCALING"
	ChangeTypeNetworkPolicy ChangeType = "NETWORK_POLICY"
	ChangeTypeSecret        ChangeType = "SECRET"
	ChangeTypeFeatureFlag   ChangeType = "FEATURE_FLAG"
	ChangeTypeDatabase      ChangeType = "DATABASE"
	ChangeTypeInfra         ChangeType = "INFRASTRUCTURE"
	ChangeTypeUnknown       ChangeType = "UNKNOWN"
)

// ConfigChange represents a detected configuration or deployment change
type ConfigChange struct {
	Timestamp   time.Time  `json:"timestamp"`
	ChangeType  ChangeType `json:"change_type"`
	Description string     `json:"description"`
	Service     string     `json:"service,omitempty"`
	User        string     `json:"user,omitempty"`
	Resource    string     `json:"resource,omitempty"`
	BeforeValue string     `json:"before_value,omitempty"`
	AfterValue  string     `json:"after_value,omitempty"`
	RiskLevel   string     `json:"risk_level"`  // low, medium, high, critical
	Correlation float64    `json:"correlation"` // 0-1 correlation score with incident
}

// CorrelationAnalysis contains the results of correlating changes with an incident
type CorrelationAnalysis struct {
	IncidentTime     time.Time       `json:"incident_time"`
	WindowBefore     time.Duration   `json:"window_before"`
	WindowAfter      time.Duration   `json:"window_after"`
	TotalChanges     int             `json:"total_changes"`
	HighRiskChanges  int             `json:"high_risk_changes"`
	Changes          []*ConfigChange `json:"changes"`
	LikelyTrigger    *ConfigChange   `json:"likely_trigger,omitempty"`
	CorrelationScore float64         `json:"correlation_score"` // Overall correlation confidence
	Recommendation   string          `json:"recommendation"`
}

// ============================================================================
// DYNAMIC PATTERN REGISTRY (Extensible Change Detection)
// ============================================================================

// ChangePatternRegistry allows dynamic registration of change patterns.
// This enables users to extend change detection with custom patterns
// specific to their infrastructure and tooling.
type ChangePatternRegistry struct {
	patterns     map[ChangeType][]string
	riskPatterns map[string][]string // risk level -> patterns
}

// globalPatternRegistry is the singleton pattern registry
var globalPatternRegistry = NewChangePatternRegistry()

// NewChangePatternRegistry creates a new registry with default patterns
func NewChangePatternRegistry() *ChangePatternRegistry {
	r := &ChangePatternRegistry{
		patterns:     make(map[ChangeType][]string),
		riskPatterns: make(map[string][]string),
	}
	r.loadDefaults()
	return r
}

// loadDefaults loads the built-in patterns
func (r *ChangePatternRegistry) loadDefaults() {
	// Default change type patterns
	r.patterns = map[ChangeType][]string{
		ChangeTypeDeployment: {
			"deployment", "deploy", "release", "rollout", "rollback",
			"image updated", "container started", "pod created", "replica",
			"helm", "argocd", "flux", "spinnaker", "jenkins",
			"ci/cd", "pipeline", "build completed", "artifact",
		},
		ChangeTypeConfig: {
			"config", "configuration", "configmap", "setting",
			"property changed", "parameter", "environment variable",
			"feature toggle", "flag enabled", "flag disabled",
			"updated configuration", "reloaded config",
		},
		ChangeTypeIAM: {
			"iam", "rbac", "role", "permission", "access",
			"policy", "authorization", "authentication",
			"api key", "service account", "credential",
			"token", "oauth", "saml", "ldap",
		},
		ChangeTypeScaling: {
			"scale", "autoscale", "hpa", "vpa", "replica",
			"capacity", "instances", "nodes", "resize",
			"horizontal pod autoscaler", "vertical pod autoscaler",
			"cluster autoscaler", "spot instance",
		},
		ChangeTypeNetworkPolicy: {
			"network policy", "firewall", "security group",
			"ingress", "egress", "load balancer", "dns",
			"route", "endpoint", "service mesh", "istio",
			"envoy", "linkerd", "calico", "cilium",
		},
		ChangeTypeSecret: {
			"secret", "vault", "certificate", "cert",
			"tls", "ssl", "key rotation", "password",
			"encryption", "kms", "hsm",
		},
		ChangeTypeFeatureFlag: {
			"feature flag", "feature toggle", "launchdarkly",
			"split.io", "optimizely", "unleash", "flagsmith",
			"experiment", "a/b test", "canary",
		},
		ChangeTypeDatabase: {
			"migration", "schema", "database", "table",
			"index", "partition", "vacuum", "reindex",
			"failover", "replica promotion", "master switch",
		},
		ChangeTypeInfra: {
			"terraform", "cloudformation", "pulumi", "ansible",
			"infrastructure", "vpc", "subnet", "availability zone",
			"region", "datacenter", "hardware", "maintenance",
		},
	}

	// Default risk patterns
	r.riskPatterns = map[string][]string{
		"critical": {
			"production", "prod", "critical", "breaking",
			"rollback", "revert", "emergency", "hotfix",
			"security", "vulnerability", "cve", "patch",
		},
		"high": {
			"database", "schema", "migration", "iam",
			"permission", "access", "network", "firewall",
			"secret", "certificate", "key rotation",
		},
		"medium": {
			"deployment", "release", "config", "scale",
			"replica", "feature flag", "toggle",
		},
	}
}

// RegisterChangeType registers a new custom change type with patterns.
// If the change type already exists, patterns are appended.
func (r *ChangePatternRegistry) RegisterChangeType(changeType ChangeType, patterns []string) {
	existing := r.patterns[changeType]
	r.patterns[changeType] = append(existing, patterns...)
}

// RegisterRiskPatterns registers patterns for a risk level.
// Valid risk levels: "critical", "high", "medium", "low"
func (r *ChangePatternRegistry) RegisterRiskPatterns(riskLevel string, patterns []string) {
	existing := r.riskPatterns[riskLevel]
	r.riskPatterns[riskLevel] = append(existing, patterns...)
}

// GetPatterns returns patterns for a change type
func (r *ChangePatternRegistry) GetPatterns(changeType ChangeType) []string {
	return r.patterns[changeType]
}

// GetAllChangeTypes returns all registered change types
func (r *ChangePatternRegistry) GetAllChangeTypes() []ChangeType {
	types := make([]ChangeType, 0, len(r.patterns))
	for t := range r.patterns {
		types = append(types, t)
	}
	return types
}

// DetectChangeType identifies the type of change from message content
func (r *ChangePatternRegistry) DetectChangeType(msgLower string) ChangeType {
	for changeType, patterns := range r.patterns {
		for _, pattern := range patterns {
			if strings.Contains(msgLower, pattern) {
				return changeType
			}
		}
	}
	return ChangeTypeUnknown
}

// AssessRiskLevel determines the risk level of a change
func (r *ChangePatternRegistry) AssessRiskLevel(msgLower string) string {
	for level, patterns := range r.riskPatterns {
		for _, pattern := range patterns {
			if strings.Contains(msgLower, pattern) {
				return level
			}
		}
	}
	return "low"
}

// Reset resets the registry to default patterns
func (r *ChangePatternRegistry) Reset() {
	r.loadDefaults()
}

// ============================================================================
// GLOBAL REGISTRY FUNCTIONS
// ============================================================================

// RegisterChangeType registers a custom change type with the global registry.
// This allows users to extend change detection without modifying source code.
//
// Example:
//
//	RegisterChangeType("KUBERNETES_CRD", []string{"customresource", "crd", "operator"})
func RegisterChangeType(changeType ChangeType, patterns []string) {
	globalPatternRegistry.RegisterChangeType(changeType, patterns)
}

// RegisterRiskPatterns registers custom risk patterns with the global registry.
//
// Example:
//
//	RegisterRiskPatterns("critical", []string{"data-loss", "outage"})
func RegisterRiskPatterns(riskLevel string, patterns []string) {
	globalPatternRegistry.RegisterRiskPatterns(riskLevel, patterns)
}

// GetRegisteredChangeTypes returns all registered change types
func GetRegisteredChangeTypes() []ChangeType {
	return globalPatternRegistry.GetAllChangeTypes()
}

// ResetPatternRegistry resets the global registry to defaults
func ResetPatternRegistry() {
	globalPatternRegistry.Reset()
}

// GetConfigChangesInWindowTool queries for configuration changes near an incident.
// This enables causal pivoting by automatically correlating system changes
// with the onset of problems.
type GetConfigChangesInWindowTool struct {
	*BaseTool
}

// NewGetConfigChangesInWindowTool creates a new GetConfigChangesInWindowTool
func NewGetConfigChangesInWindowTool(c *client.Client, l *zap.Logger) *GetConfigChangesInWindowTool {
	return &GetConfigChangesInWindowTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetConfigChangesInWindowTool) Name() string { return "get_config_changes" }

// Annotations returns tool hints for LLMs
func (t *GetConfigChangesInWindowTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Get Config Changes")
}

// DefaultTimeout returns the timeout
func (t *GetConfigChangesInWindowTool) DefaultTimeout() time.Duration {
	return DefaultQueryTimeout
}

// Description returns the tool description
func (t *GetConfigChangesInWindowTool) Description() string {
	return `Find configuration, deployment, and IAM changes around an incident time.

**Purpose (Causal Pivoting - SOTA 2025):**
Automatically correlates incidents with system changes to identify potential triggers.
Uses semantic pattern matching to detect deployments, config changes, IAM updates,
scaling events, and infrastructure modifications.

**Best for:**
- Root cause analysis: "What changed before the incident?"
- Post-mortem investigation: "Was there a recent deployment?"
- Change correlation: "Did any config changes correlate with the error spike?"

**Detected change types:**
- DEPLOYMENT: Releases, rollouts, container updates
- CONFIG: ConfigMaps, environment variables, feature flags
- IAM: Permissions, roles, service accounts, credentials
- SCALING: Autoscaling events, replica changes
- NETWORK_POLICY: Firewall rules, ingress/egress, service mesh
- SECRET: Certificate rotations, key updates
- DATABASE: Migrations, schema changes, failovers
- INFRASTRUCTURE: Terraform, cloud resource changes

**Returns:**
- Chronologically sorted list of changes
- Risk assessment for each change
- Correlation score with incident timing
- Likely trigger identification

**Related tools:** analyze_log_delta, investigate_incident, get_trace_context`
}

// InputSchema returns the input schema
func (t *GetConfigChangesInWindowTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"incident_time": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the incident started (ISO 8601). Changes before this are most suspect.",
			},
			"window_before": map[string]interface{}{
				"type":        "string",
				"description": "How far before incident to search: 5m, 15m, 30m, 1h, 2h, 6h (default: 1h)",
				"enum":        []string{"5m", "15m", "30m", "1h", "2h", "6h"},
				"default":     "1h",
			},
			"window_after": map[string]interface{}{
				"type":        "string",
				"description": "How far after incident to search: 5m, 15m, 30m (default: 15m)",
				"enum":        []string{"5m", "15m", "30m"},
				"default":     "15m",
			},
			"service": map[string]interface{}{
				"type":        "string",
				"description": "Focus on changes for a specific service/application (optional)",
			},
			"change_types": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Filter to specific change types: DEPLOYMENT, CONFIG, IAM, SCALING, etc. (optional, all if not specified)",
			},
		},
		"required": []string{"incident_time"},
	}
}

// Metadata returns semantic metadata for tool discovery
func (t *GetConfigChangesInWindowTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryObservability, CategoryAIHelper},
		Keywords:      []string{"config", "deployment", "change", "iam", "correlation", "trigger", "root cause", "what changed"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Find deployment that caused issue", "Correlate config changes with incidents", "IAM change audit"},
		RelatedTools:  []string{"analyze_log_delta", "investigate_incident", "get_trace_context"},
		ChainPosition: ChainMiddle,
	}
}

// Execute finds and analyzes configuration changes
func (t *GetConfigChangesInWindowTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse incident time
	incidentTimeStr, err := GetStringParam(args, "incident_time", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	incidentTime, err := time.Parse(time.RFC3339, incidentTimeStr)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Invalid incident_time format: %v", err)), nil
	}

	// Parse window parameters
	windowBeforeStr, _ := GetStringParam(args, "window_before", false)
	if windowBeforeStr == "" {
		windowBeforeStr = "1h"
	}
	windowBefore := parseDuration(windowBeforeStr, time.Hour)

	windowAfterStr, _ := GetStringParam(args, "window_after", false)
	if windowAfterStr == "" {
		windowAfterStr = "15m"
	}
	windowAfter := parseDuration(windowAfterStr, 15*time.Minute)

	// Optional filters
	service, _ := GetStringParam(args, "service", false)
	changeTypesRaw, _ := args["change_types"].([]interface{})
	var changeTypeFilters []ChangeType
	for _, ct := range changeTypesRaw {
		if ctStr, ok := ct.(string); ok {
			changeTypeFilters = append(changeTypeFilters, ChangeType(ctStr))
		}
	}

	// Calculate time range
	startTime := incidentTime.Add(-windowBefore)
	endTime := incidentTime.Add(windowAfter)

	// Build query for change-related logs
	query := buildChangeDetectionQuery(service)

	query, _, err = PrepareQuery(query, "archive", "dataprime")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Query preparation failed: %v", err)), nil
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body: map[string]interface{}{
			"query": query,
			"metadata": map[string]interface{}{
				"tier":       "archive",
				"syntax":     "dataprime",
				"start_date": startTime.Format(time.RFC3339),
				"end_date":   endTime.Format(time.RFC3339),
				"limit":      500,
			},
		},
		AcceptSSE: true,
		Timeout:   DefaultQueryTimeout,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}

	events, _ := result["events"].([]interface{})

	// Analyze events for changes
	analysis := analyzeChanges(events, incidentTime, windowBefore, windowAfter, changeTypeFilters)

	// Format response
	response := formatCorrelationAnalysis(analysis)

	// Record tool use
	session := GetSession()
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"incident_time": incidentTimeStr,
		"window_before": windowBeforeStr,
		"window_after":  windowAfterStr,
		"service":       service,
		"changes_found": len(analysis.Changes),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil
}

// buildChangeDetectionQuery creates a DataPrime query to find change-related logs
func buildChangeDetectionQuery(service string) string {
	// Build a comprehensive query that catches various change-related log patterns
	var query strings.Builder

	query.WriteString("source logs | filter ")

	// Message patterns for changes
	changeKeywords := []string{
		"deploy", "release", "rollout", "rollback",
		"config", "configuration", "setting",
		"iam", "rbac", "permission", "role",
		"scale", "autoscale", "replica",
		"secret", "certificate", "credential",
		"migration", "schema", "database",
		"terraform", "infrastructure", "policy",
		"feature flag", "toggle", "enabled", "disabled",
		"updated", "changed", "created", "deleted", "modified",
	}

	// Build OR conditions for message content
	conditions := make([]string, 0, len(changeKeywords))
	for _, kw := range changeKeywords {
		conditions = append(conditions, fmt.Sprintf("$d.message:string.contains('%s')", escapeDataPrimeString(kw)))
	}

	query.WriteString("(")
	query.WriteString(strings.Join(conditions, " || "))
	query.WriteString(")")

	// Add service filter if specified
	if service != "" {
		query.WriteString(fmt.Sprintf(" && $l.applicationname == '%s'", escapeDataPrimeString(service)))
	}

	query.WriteString(" | sortby $m.timestamp asc | limit 500")

	return query.String()
}

// analyzeChanges processes log events to detect and categorize changes
func analyzeChanges(events []interface{}, incidentTime time.Time, windowBefore, windowAfter time.Duration, typeFilters []ChangeType) *CorrelationAnalysis {
	analysis := &CorrelationAnalysis{
		IncidentTime: incidentTime,
		WindowBefore: windowBefore,
		WindowAfter:  windowAfter,
		Changes:      make([]*ConfigChange, 0),
	}

	typeFilterSet := make(map[ChangeType]bool)
	for _, ct := range typeFilters {
		typeFilterSet[ct] = true
	}

	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}

		change := extractChange(eventMap)
		if change == nil {
			continue
		}

		// Apply type filter if specified
		if len(typeFilterSet) > 0 && !typeFilterSet[change.ChangeType] {
			continue
		}

		// Calculate correlation score based on timing
		change.Correlation = calculateCorrelation(change.Timestamp, incidentTime, windowBefore)

		analysis.Changes = append(analysis.Changes, change)
		analysis.TotalChanges++

		if change.RiskLevel == "high" || change.RiskLevel == "critical" {
			analysis.HighRiskChanges++
		}
	}

	// Sort by timestamp
	sort.Slice(analysis.Changes, func(i, j int) bool {
		return analysis.Changes[i].Timestamp.Before(analysis.Changes[j].Timestamp)
	})

	// Find likely trigger (highest correlation score before incident)
	var maxCorrelation float64
	for _, change := range analysis.Changes {
		if change.Timestamp.Before(incidentTime) && change.Correlation > maxCorrelation {
			maxCorrelation = change.Correlation
			analysis.LikelyTrigger = change
		}
	}

	// Calculate overall correlation score
	analysis.CorrelationScore = calculateOverallCorrelation(analysis)

	// Generate recommendation
	analysis.Recommendation = generateRecommendation(analysis)

	return analysis
}

// extractChange detects a change from a log event
func extractChange(event map[string]interface{}) *ConfigChange {
	msg := extractMessageField(event)
	if msg == "" {
		return nil
	}

	msgLower := strings.ToLower(msg)

	// Detect change type
	changeType := detectChangeType(msgLower)
	if changeType == ChangeTypeUnknown {
		// Check if it still looks like a change event
		if !looksLikeChange(msgLower) {
			return nil
		}
	}

	change := &ConfigChange{
		Timestamp:   extractTimestampFromEvent(event),
		ChangeType:  changeType,
		Description: truncateString(msg, 200),
		RiskLevel:   assessRiskLevel(msgLower),
	}

	// Extract service/app
	change.Service = extractAppFromEvent(event)

	// Try to extract user from common fields
	if userData, ok := event["user_data"].(map[string]interface{}); ok {
		if user, ok := userData["user"].(string); ok {
			change.User = user
		} else if user, ok := userData["actor"].(string); ok {
			change.User = user
		} else if user, ok := userData["principal"].(string); ok {
			change.User = user
		}

		if resource, ok := userData["resource"].(string); ok {
			change.Resource = resource
		}
	}

	return change
}

// detectChangeType identifies the type of change from message content.
// Uses the global pattern registry for extensibility.
func detectChangeType(msgLower string) ChangeType {
	return globalPatternRegistry.DetectChangeType(msgLower)
}

// looksLikeChange checks if message appears to be a change event
func looksLikeChange(msgLower string) bool {
	changeIndicators := []string{
		"updated", "changed", "modified", "created", "deleted",
		"added", "removed", "enabled", "disabled", "started", "stopped",
		"applied", "reverted", "rolled", "promoted", "demoted",
	}
	for _, indicator := range changeIndicators {
		if strings.Contains(msgLower, indicator) {
			return true
		}
	}
	return false
}

// assessRiskLevel determines the risk level of a change.
// Uses the global pattern registry for extensibility.
func assessRiskLevel(msgLower string) string {
	return globalPatternRegistry.AssessRiskLevel(msgLower)
}

// calculateCorrelation calculates correlation score based on timing
func calculateCorrelation(changeTime, incidentTime time.Time, windowBefore time.Duration) float64 {
	if changeTime.After(incidentTime) {
		// Changes after incident have lower correlation (could be response actions)
		return 0.3
	}

	// Calculate how close to incident time (0-1 scale)
	timeDiff := incidentTime.Sub(changeTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// Changes closer to incident time have higher correlation
	// Using exponential decay
	halfLife := windowBefore / 4
	correlation := 1.0
	if halfLife > 0 {
		correlation = 1.0 / (1.0 + float64(timeDiff)/float64(halfLife))
	}

	return correlation
}

// calculateOverallCorrelation calculates the overall correlation confidence
func calculateOverallCorrelation(analysis *CorrelationAnalysis) float64 {
	if analysis.TotalChanges == 0 {
		return 0.0
	}

	// Factors that increase confidence:
	// 1. High-risk changes before incident
	// 2. Changes close to incident time
	// 3. Fewer total changes (more focused)

	var score float64

	// Weight by high-risk changes
	if analysis.HighRiskChanges > 0 {
		score += 0.3
	}

	// Weight by likely trigger confidence
	if analysis.LikelyTrigger != nil {
		score += analysis.LikelyTrigger.Correlation * 0.5
	}

	// Bonus for focused changes (fewer is often clearer)
	if analysis.TotalChanges <= 5 {
		score += 0.2
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateRecommendation creates actionable recommendation based on analysis
func generateRecommendation(analysis *CorrelationAnalysis) string {
	if analysis.TotalChanges == 0 {
		return "No configuration changes detected in the time window. Consider expanding the search window or investigating other potential causes (resource exhaustion, external dependencies, traffic spikes)."
	}

	if analysis.LikelyTrigger != nil {
		trigger := analysis.LikelyTrigger
		switch trigger.ChangeType {
		case ChangeTypeDeployment:
			return fmt.Sprintf("**Likely Trigger: Deployment** at %s. Recommend: Check deployment diff, verify rollout status, consider rollback if issue persists.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeConfig:
			return fmt.Sprintf("**Likely Trigger: Configuration Change** at %s. Recommend: Review config diff, verify values are correct, check for typos or invalid values.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeIAM:
			return fmt.Sprintf("**Likely Trigger: IAM/Permission Change** at %s. Recommend: Verify service accounts have required permissions, check role bindings, review access policies.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeScaling:
			return fmt.Sprintf("**Likely Trigger: Scaling Event** at %s. Recommend: Check if scaling was intentional, verify resource limits, monitor for resource contention.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeNetworkPolicy:
			return fmt.Sprintf("**Likely Trigger: Network Policy Change** at %s. Recommend: Verify network connectivity, check firewall rules, test service-to-service communication.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeSecret:
			return fmt.Sprintf("**Likely Trigger: Secret/Certificate Change** at %s. Recommend: Verify secret values are correct, check certificate validity, ensure proper propagation.",
				trigger.Timestamp.Format("15:04:05"))
		case ChangeTypeDatabase:
			return fmt.Sprintf("**Likely Trigger: Database Change** at %s. Recommend: Verify migration completed successfully, check query performance, review schema changes.",
				trigger.Timestamp.Format("15:04:05"))
		default:
			return fmt.Sprintf("**Likely Trigger: %s** at %s. Recommend: Review the change details and verify it was applied correctly.",
				trigger.ChangeType, trigger.Timestamp.Format("15:04:05"))
		}
	}

	if analysis.HighRiskChanges > 0 {
		return fmt.Sprintf("Found %d high-risk changes in the time window. Recommend: Review each high-risk change chronologically and correlate with symptom onset.",
			analysis.HighRiskChanges)
	}

	return fmt.Sprintf("Found %d changes in the time window. No single likely trigger identified. Recommend: Review changes chronologically and use analyze_log_delta to identify which change correlates with symptom onset.",
		analysis.TotalChanges)
}

// formatCorrelationAnalysis formats the analysis as markdown
func formatCorrelationAnalysis(analysis *CorrelationAnalysis) string {
	var sb strings.Builder

	sb.WriteString("# 🔄 Configuration Change Correlation Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Incident Time:** %s\n", analysis.IncidentTime.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Search Window:** -%s to +%s\n", analysis.WindowBefore, analysis.WindowAfter))
	sb.WriteString(fmt.Sprintf("**Total Changes Found:** %d\n", analysis.TotalChanges))
	sb.WriteString(fmt.Sprintf("**High-Risk Changes:** %d\n", analysis.HighRiskChanges))
	sb.WriteString(fmt.Sprintf("**Correlation Confidence:** %.0f%%\n\n", analysis.CorrelationScore*100))

	// Likely trigger
	if analysis.LikelyTrigger != nil {
		sb.WriteString("## 🎯 Most Likely Trigger\n\n")
		trigger := analysis.LikelyTrigger
		sb.WriteString("| Property | Value |\n")
		sb.WriteString("|----------|-------|\n")
		sb.WriteString(fmt.Sprintf("| **Type** | %s |\n", trigger.ChangeType))
		sb.WriteString(fmt.Sprintf("| **Time** | %s |\n", trigger.Timestamp.Format("15:04:05")))
		sb.WriteString(fmt.Sprintf("| **Risk Level** | %s |\n", trigger.RiskLevel))
		sb.WriteString(fmt.Sprintf("| **Correlation** | %.0f%% |\n", trigger.Correlation*100))
		if trigger.Service != "" {
			sb.WriteString(fmt.Sprintf("| **Service** | %s |\n", trigger.Service))
		}
		if trigger.User != "" {
			sb.WriteString(fmt.Sprintf("| **User** | %s |\n", trigger.User))
		}
		sb.WriteString(fmt.Sprintf("\n**Description:** %s\n\n", trigger.Description))
	}

	// Recommendation
	sb.WriteString("## 📋 Recommendation\n\n")
	sb.WriteString(analysis.Recommendation)
	sb.WriteString("\n\n")

	// Timeline of changes
	if len(analysis.Changes) > 0 {
		sb.WriteString("## ⏱️ Change Timeline\n\n")
		sb.WriteString("| Time | Type | Risk | Service | Description |\n")
		sb.WriteString("|------|------|------|---------|-------------|\n")

		for i, change := range analysis.Changes {
			if i >= 15 {
				sb.WriteString(fmt.Sprintf("| ... | ... | ... | ... | ... (%d more changes) |\n", len(analysis.Changes)-15))
				break
			}

			// Mark incident time in timeline
			timeStr := change.Timestamp.Format("15:04:05")
			if change.Timestamp.After(analysis.IncidentTime) {
				timeStr = timeStr + " (after)"
			}

			var riskEmoji string
			switch change.RiskLevel {
			case "critical":
				riskEmoji = "🔴"
			case "high":
				riskEmoji = "🟠"
			case "medium":
				riskEmoji = "🟡"
			default:
				riskEmoji = "🟢"
			}

			desc := truncateString(change.Description, 50)
			sb.WriteString(fmt.Sprintf("| %s | %s | %s %s | %s | %s |\n",
				timeStr, change.ChangeType, riskEmoji, change.RiskLevel, change.Service, desc))
		}
		sb.WriteString("\n")
	}

	// Related tools
	sb.WriteString("## 🔗 Next Steps\n\n")
	sb.WriteString("- Use `analyze_log_delta` to compare log patterns before/after the likely trigger\n")
	sb.WriteString("- Use `get_trace_context` with a trace ID from an affected request\n")
	sb.WriteString("- Use `investigate_incident` for comprehensive error analysis\n")

	return sb.String()
}

// parseDuration parses a duration string with a default fallback
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	switch s {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "2h":
		return 2 * time.Hour
	case "6h":
		return 6 * time.Hour
	default:
		return defaultVal
	}
}
