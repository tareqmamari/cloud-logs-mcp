// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements the SmartInvestigateTool that provides autonomous incident investigation.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// SmartInvestigateTool provides autonomous incident investigation
type SmartInvestigateTool struct {
	*BaseTool
	strategyFactory *QueryStrategyFactory
	heuristicEngine *HeuristicEngine
	remediationGen  *RemediationGenerator
}

// NewSmartInvestigateTool creates a new smart investigation tool
func NewSmartInvestigateTool(c *client.Client, l *zap.Logger) *SmartInvestigateTool {
	return &SmartInvestigateTool{
		BaseTool:        NewBaseTool(c, l),
		strategyFactory: NewQueryStrategyFactory(),
		heuristicEngine: NewHeuristicEngine(),
		remediationGen:  NewRemediationGenerator(),
	}
}

// Name returns the tool name
func (t *SmartInvestigateTool) Name() string {
	return "smart_investigate"
}

// Description returns the tool description
func (t *SmartInvestigateTool) Description() string {
	return `Autonomous incident investigation that thinks like a senior SRE.

**Investigation Modes:**
- **global**: System-wide health scan - aggregates errors across ALL services
- **component**: Deep dive into a specific service and its dependencies
- **flow**: Trace a request across service boundaries using trace_id or correlation_id

**What it does:**
1. Determines investigation scope based on your inputs
2. Executes multi-phase query strategy
3. Applies heuristic rules to findings (timeout, memory, database, auth patterns)
4. Synthesizes evidence into a root cause statement
5. Suggests next investigation steps
6. Optionally generates alert and dashboard configurations

**When to use:**
- Incident response: "Why are users seeing 500 errors?"
- Proactive monitoring: "What's the system health?"
- Request tracing: "Why did this request fail?"

**Related tools:**
- investigate_incident: Simpler guided investigation
- query_logs: Direct log querying
- health_check: Quick system health overview`
}

// DefaultTimeout returns the timeout for investigation operations
func (t *SmartInvestigateTool) DefaultTimeout() time.Duration {
	return DefaultWorkflowTimeout
}

// Annotations returns tool hints for LLMs
func (t *SmartInvestigateTool) Annotations() *mcp.ToolAnnotations {
	return WorkflowAnnotations("Smart Investigate")
}

// InputSchema returns the input schema
func (t *SmartInvestigateTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Target application for component-mode investigation. If provided, investigation focuses on this specific service.",
			},
			"trace_id": map[string]interface{}{
				"type":        "string",
				"description": "Trace ID for flow-mode investigation. Traces a request across service boundaries.",
			},
			"correlation_id": map[string]interface{}{
				"type":        "string",
				"description": "Correlation ID for flow-mode investigation. Alternative to trace_id.",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Investigation window: 15m, 1h, 6h, 24h (default: 1h)",
				"enum":        []string{"15m", "1h", "6h", "24h"},
				"default":     "1h",
			},
			"generate_assets": map[string]interface{}{
				"type":        "boolean",
				"description": "Generate Terraform/JSON for alerts and dashboards based on findings",
				"default":     false,
			},
			"max_queries": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of queries to execute (default: 5, max: 10)",
				"minimum":     1,
				"maximum":     10,
				"default":     5,
			},
		},
	}
}

// Execute performs the smart investigation
func (t *SmartInvestigateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()

	// Determine investigation mode from parameters
	mode := t.strategyFactory.DetermineMode(args)
	strategy := t.strategyFactory.CreateStrategy(mode)

	// Build investigation context
	invCtx := &SmartInvestigationContext{
		Mode:      mode,
		TimeRange: t.parseTimeRange(args),
		Findings:  []InvestigationFinding{},
	}

	// Extract mode-specific parameters
	if app, _ := GetStringParam(args, "application", false); app != "" {
		invCtx.TargetService = app
	}
	if traceID, _ := GetStringParam(args, "trace_id", false); traceID != "" {
		invCtx.TraceID = traceID
	}
	if corrID, _ := GetStringParam(args, "correlation_id", false); corrID != "" {
		invCtx.CorrelationID = corrID
	}

	// Validate flow mode has required parameters
	if mode == ModeFlow && invCtx.TraceID == "" && invCtx.CorrelationID == "" {
		return NewToolResultError("Flow mode requires either trace_id or correlation_id"), nil
	}

	// Get max queries limit
	maxQueries, _ := GetIntParam(args, "max_queries", false)
	if maxQueries == 0 {
		maxQueries = 5
	}
	if maxQueries > 10 {
		maxQueries = 10
	}

	// Phase 1: Execute initial queries
	queryPlans := strategy.InitialQueries(invCtx)
	if len(queryPlans) > maxQueries {
		queryPlans = queryPlans[:maxQueries]
	}

	results := t.executeQueries(ctx, queryPlans, invCtx)

	// Phase 2: Analyze results
	invCtx.Findings = strategy.AnalyzeResults(invCtx, results)

	// Phase 3: Apply heuristics
	allEvents := t.collectEvents(results)
	heuristicActions := t.heuristicEngine.AnalyzeAndSuggest(invCtx.Findings, allEvents)
	invCtx.NextActions = append(invCtx.NextActions, heuristicActions...)

	// Also get strategy-specific actions
	strategyActions := strategy.SuggestNextActions(invCtx)
	invCtx.NextActions = append(invCtx.NextActions, strategyActions...)

	// Deduplicate and sort actions
	invCtx.NextActions = deduplicateActions(invCtx.NextActions)
	sortActionsByPriority(invCtx.NextActions)

	// Phase 4: Synthesize evidence
	evidence := strategy.SynthesizeEvidence(invCtx)

	// Phase 5: Generate assets if requested
	var assets *IncidentResponseAssets
	generateAssets, _ := GetBoolParam(args, "generate_assets", false)
	if generateAssets && len(invCtx.Findings) > 0 {
		remCtx := &IncidentContext{
			RootCause:        evidence.RootCause,
			AffectedServices: evidence.AffectedServices,
		}
		assets = t.remediationGen.Generate(remCtx, "high")
	}

	// Record in session
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"mode":           mode,
		"findings_count": len(invCtx.Findings),
		"root_cause":     evidence.RootCause,
	})

	// Format response
	return t.formatSmartResponse(invCtx, evidence, assets, results)
}

func (t *SmartInvestigateTool) parseTimeRange(args map[string]interface{}) InvestigationTimeRange {
	tr, _ := GetStringParam(args, "time_range", false)
	if tr == "" {
		tr = "1h"
	}

	end := time.Now().UTC()
	var start time.Time

	switch tr {
	case "15m":
		start = end.Add(-15 * time.Minute)
	case "6h":
		start = end.Add(-6 * time.Hour)
	case "24h":
		start = end.Add(-24 * time.Hour)
	default:
		start = end.Add(-1 * time.Hour)
	}

	return InvestigationTimeRange{Start: start, End: end}
}

func (t *SmartInvestigateTool) executeQueries(ctx context.Context, plans []QueryPlan, invCtx *SmartInvestigationContext) []ExecutedQuery {
	results := []ExecutedQuery{}

	for _, plan := range plans {
		// Clean up the query (remove extra whitespace/newlines)
		cleanQuery := cleanQueryString(plan.Query)

		// Auto-correct and validate DataPrime query
		cleanQuery, _ = AutoCorrectDataPrimeQuery(cleanQuery)

		// Determine tier based on TCO policies
		// For component mode with a specific app, use TCO-recommended tier
		// For global mode, use archive to ensure complete data coverage
		tier := plan.Tier
		if tier == "" {
			session := GetSession()
			if invCtx.TargetService != "" {
				// Component mode: use TCO-aware tier for the specific application
				tier = session.GetTierForApplication(invCtx.TargetService)
			} else {
				// Global mode: use archive for complete data (investigation needs all logs)
				tier = "archive"
			}
		}

		metadata := map[string]interface{}{
			"tier":       tier,
			"syntax":     "dataprime",
			"start_date": invCtx.TimeRange.Start.Format(time.RFC3339),
			"end_date":   invCtx.TimeRange.End.Format(time.RFC3339),
		}

		body := map[string]interface{}{
			"query":    cleanQuery,
			"metadata": metadata,
		}

		req := &client.Request{
			Method:    "POST",
			Path:      "/v1/query",
			Body:      body,
			AcceptSSE: true,
			Timeout:   60 * time.Second,
		}

		startTime := time.Now()
		result, err := t.ExecuteRequest(ctx, req)
		duration := time.Since(startTime)

		qr := ExecutedQuery{
			QueryID:  plan.ID,
			Query:    cleanQuery,
			Duration: duration,
			Error:    err,
			Metadata: metadata,
		}

		if err == nil && result != nil {
			qr.Events = extractSSEEvents(result)
		}

		results = append(results, qr)
		invCtx.QueryHistory = append(invCtx.QueryHistory, qr)
	}

	return results
}

func (t *SmartInvestigateTool) collectEvents(results []ExecutedQuery) []map[string]interface{} {
	events := []map[string]interface{}{}
	for _, r := range results {
		events = append(events, r.Events...)
	}
	return events
}

func (t *SmartInvestigateTool) formatSmartResponse(
	invCtx *SmartInvestigationContext,
	evidence *EvidenceSummary,
	assets *IncidentResponseAssets,
	results []ExecutedQuery,
) (*mcp.CallToolResult, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("# Smart Investigation Report\n\n")
	sb.WriteString(fmt.Sprintf("**Mode:** %s\n", invCtx.Mode))
	sb.WriteString(fmt.Sprintf("**Time Range:** %s to %s\n",
		invCtx.TimeRange.Start.Format("15:04 MST"),
		invCtx.TimeRange.End.Format("15:04 MST")))

	if invCtx.TargetService != "" {
		sb.WriteString(fmt.Sprintf("**Target Service:** %s\n", invCtx.TargetService))
	}
	if invCtx.TraceID != "" {
		sb.WriteString(fmt.Sprintf("**Trace ID:** %s\n", invCtx.TraceID))
	}
	sb.WriteString("\n")

	// Query Execution Summary
	sb.WriteString("## Query Execution\n\n")
	successCount := 0
	for _, r := range results {
		status := "SUCCESS"
		if r.Error != nil {
			status = "ERROR"
		} else {
			successCount++
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s (%d events, %dms)\n",
			r.QueryID, status, len(r.Events), r.Duration.Milliseconds()))
	}
	sb.WriteString("\n")

	// Root Cause
	sb.WriteString("## Root Cause\n\n")
	if evidence.RootCause != "" {
		sb.WriteString(fmt.Sprintf("> **%s**\n\n", evidence.RootCause))
		sb.WriteString(fmt.Sprintf("_Confidence: %.0f%%_\n\n", evidence.Confidence*100))
	} else {
		sb.WriteString("> No critical issues identified.\n\n")
	}

	// Impact Summary
	if evidence.ImpactSummary != "" {
		sb.WriteString(fmt.Sprintf("**Impact:** %s\n\n", evidence.ImpactSummary))
	}

	// Affected Services
	if len(evidence.AffectedServices) > 0 {
		sb.WriteString("**Affected Services:** ")
		sb.WriteString(strings.Join(evidence.AffectedServices, ", "))
		sb.WriteString("\n\n")
	}

	// Findings
	if len(invCtx.Findings) > 0 {
		sb.WriteString("## Findings\n\n")

		// Sort by severity
		sortFindingsBySeverity(invCtx.Findings)

		for i, f := range invCtx.Findings {
			if i >= 10 { // Limit to top 10 findings
				sb.WriteString(fmt.Sprintf("\n_... and %d more findings_\n", len(invCtx.Findings)-10))
				break
			}

			icon := getSeverityIcon(f.Severity)
			sb.WriteString(fmt.Sprintf("%d. %s **[%s]** %s\n", i+1, icon, f.Severity, f.Summary))
			if f.Evidence != "" {
				sb.WriteString(fmt.Sprintf("   - Evidence: %s\n", f.Evidence))
			}
			if f.Service != "" {
				sb.WriteString(fmt.Sprintf("   - Service: %s\n", f.Service))
			}
		}
		sb.WriteString("\n")
	}

	// Suggested Next Actions
	if len(invCtx.NextActions) > 0 {
		sb.WriteString("## Suggested Next Actions\n\n")
		for i, action := range invCtx.NextActions {
			if i >= 5 { // Limit to top 5
				break
			}
			sb.WriteString(fmt.Sprintf("**%d. %s**\n", i+1, action.Description))
			sb.WriteString(fmt.Sprintf("   - Rationale: %s\n", action.Rationale))
			if action.Query != "" {
				sb.WriteString(fmt.Sprintf("   - Query: `%s`\n", truncateInvestigationQuery(action.Query)))
			}
		}
		sb.WriteString("\n")
	}

	// Generated Assets
	if assets != nil {
		sb.WriteString("## Generated Assets\n\n")
		sb.WriteString(FormatAssetsAsMarkdown(assets))
	}

	// SOPs from heuristics (if no assets generated)
	if assets == nil && len(invCtx.Findings) > 0 {
		allEvents := t.collectEvents(results)
		sops := t.heuristicEngine.GetMatchingSOPs(invCtx.Findings, allEvents)
		if len(sops) > 0 {
			sb.WriteString("## Recommended Procedures\n\n")
			for _, sop := range sops {
				sb.WriteString(fmt.Sprintf("**%s**\n\n", sop.Trigger))
				sb.WriteString(fmt.Sprintf("%s\n\n", sop.Procedure))
				sb.WriteString(fmt.Sprintf("_Escalation: %s_\n\n", sop.Escalation))
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
	}, nil
}

// Helper functions

func cleanQueryString(query string) string {
	// Remove leading/trailing whitespace from each line and collapse multiple spaces
	lines := strings.Split(query, "\n")
	cleanedLines := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}
	return strings.Join(cleanedLines, " ")
}

func truncateInvestigationQuery(q string) string {
	q = cleanQueryString(q)
	if len(q) > 100 {
		return q[:97] + "..."
	}
	return q
}

func getSeverityIcon(severity InvestigationSeverity) string {
	switch severity {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityHigh:
		return "HIGH"
	case SeverityMedium:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func deduplicateActions(actions []HeuristicAction) []HeuristicAction {
	seen := make(map[string]bool)
	result := []HeuristicAction{}

	for _, action := range actions {
		key := action.Description
		if !seen[key] {
			seen[key] = true
			result = append(result, action)
		}
	}

	return result
}

// extractSSEEvents extracts events from IBM Cloud Logs SSE response format.
// The SSE response contains events like:
// - {"query_id": {...}}
// - {"result": {"results": [{"user_data": "{...json...}"}, ...]}}
// This function parses the nested structure and returns flattened event maps.
func extractSSEEvents(result map[string]interface{}) []map[string]interface{} {
	events := []map[string]interface{}{}

	rawEvents, ok := result["events"].([]interface{})
	if !ok {
		return events
	}

	for _, e := range rawEvents {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for result.results structure (aggregation queries)
		if resultObj, ok := em["result"].(map[string]interface{}); ok {
			if results, ok := resultObj["results"].([]interface{}); ok {
				for _, r := range results {
					rm, ok := r.(map[string]interface{})
					if !ok {
						continue
					}

					// Parse user_data JSON string into a map
					if userData, ok := rm["user_data"].(string); ok && userData != "" {
						var parsed map[string]interface{}
						if err := json.Unmarshal([]byte(userData), &parsed); err == nil {
							events = append(events, parsed)
						}
					}

					// Also check for direct fields (non-aggregation results)
					if len(rm) > 0 && rm["user_data"] == nil {
						events = append(events, rm)
					}
				}
			}
		}

		// Also handle direct event format (log queries without aggregation)
		// These come as: {"metadata": {...}, "labels": {...}, "user_data": "..."}
		if userData, ok := em["user_data"].(string); ok && userData != "" {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(userData), &parsed); err == nil {
				events = append(events, parsed)
			}
		}
	}

	return events
}
