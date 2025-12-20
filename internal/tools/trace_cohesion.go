// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements Trace-Log Cohesion capabilities for contextual
// pivoting using TraceID to correlate logs across services.
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
// TRACE-LOG COHESION (Contextual Pivoting)
// ============================================================================

// GetTraceContextTool retrieves all logs associated with a trace ID
// across all services, enabling end-to-end transaction analysis.
type GetTraceContextTool struct {
	*BaseTool
}

// NewGetTraceContextTool creates a new GetTraceContextTool
func NewGetTraceContextTool(c *client.Client, l *zap.Logger) *GetTraceContextTool {
	return &GetTraceContextTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetTraceContextTool) Name() string { return "get_trace_context" }

// Annotations returns tool hints for LLMs
func (t *GetTraceContextTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Get Trace Context")
}

// DefaultTimeout returns the timeout
func (t *GetTraceContextTool) DefaultTimeout() time.Duration {
	return DefaultQueryTimeout
}

// Description returns the tool description
func (t *GetTraceContextTool) Description() string {
	return `Retrieve all logs for a trace ID to see the complete request flow.

**Purpose (Trace-Log Cohesion):**
Pivots from a single log entry to the entire distributed transaction,
showing all services involved and the order of operations.

**Best for:**
- Debugging distributed transactions
- Understanding request flow across microservices
- Finding where errors propagate from

**Input:**
- trace_id: The trace ID (from a log entry or APM tool)
- time_window: How far to search (default: 1h)

**Output:**
- Timeline of all logs with this trace ID
- Service call graph
- Error propagation analysis

**Related tools:** query_logs, analyze_log_delta, investigate_incident`
}

// InputSchema returns the input schema
func (t *GetTraceContextTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"trace_id": map[string]interface{}{
				"type":        "string",
				"description": "The trace ID to look up (e.g., from a log entry or APM)",
			},
			"time_window": map[string]interface{}{
				"type":        "string",
				"description": "Time window to search: 15m, 1h, 6h, 24h (default: 1h)",
				"enum":        []string{"15m", "1h", "6h", "24h"},
				"default":     "1h",
			},
		},
		"required": []string{"trace_id"},
	}
}

// Metadata returns semantic metadata for tool discovery
func (t *GetTraceContextTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryObservability},
		Keywords:      []string{"trace", "distributed", "transaction", "span", "correlation", "request flow", "microservices"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Debug distributed transactions", "Trace request flow", "Find error source"},
		RelatedTools:  []string{"query_logs", "analyze_log_delta", "investigate_incident"},
		ChainPosition: ChainMiddle,
	}
}

// Execute retrieves trace context
func (t *GetTraceContextTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	traceID, err := GetStringParam(args, "trace_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	timeWindow, _ := GetStringParam(args, "time_window", false)
	if timeWindow == "" {
		timeWindow = "1h"
	}

	// Calculate time range
	endTime := time.Now().UTC()
	var startTime time.Time
	switch timeWindow {
	case "15m":
		startTime = endTime.Add(-15 * time.Minute)
	case "6h":
		startTime = endTime.Add(-6 * time.Hour)
	case "24h":
		startTime = endTime.Add(-24 * time.Hour)
	default:
		startTime = endTime.Add(-1 * time.Hour)
	}

	// Query logs with trace ID
	// DataPrime supports searching in user_data fields
	query := fmt.Sprintf(`source logs
		| filter $d.trace_id == '%s' || $d.traceId == '%s' || $d.traceID == '%s'
		| sortby $m.timestamp asc
		| limit 200`, escapeDataPrimeString(traceID), escapeDataPrimeString(traceID), escapeDataPrimeString(traceID))

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

	// Build response
	var response strings.Builder
	response.WriteString("# 🔗 Trace Context\n\n")
	response.WriteString(fmt.Sprintf("**Trace ID:** `%s`\n", traceID))
	response.WriteString(fmt.Sprintf("**Time Window:** %s\n", timeWindow))
	response.WriteString(fmt.Sprintf("**Logs Found:** %d\n\n", len(events)))

	if len(events) == 0 {
		response.WriteString("## No Logs Found\n\n")
		response.WriteString("No logs with this trace ID were found in the specified time window.\n\n")
		response.WriteString("**Suggestions:**\n")
		response.WriteString("- Expand the time window (try 24h)\n")
		response.WriteString("- Verify the trace ID format\n")
		response.WriteString("- Check if the application logs trace IDs\n")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: response.String()},
			},
		}, nil
	}

	// Analyze the trace
	traceAnalysis := t.analyzeTrace(events)

	// Service call graph
	response.WriteString("## 📊 Service Flow\n")
	response.WriteString("```\n")
	for i, svc := range traceAnalysis.Services {
		prefix := "├─"
		if i == len(traceAnalysis.Services)-1 {
			prefix = "└─"
		}
		status := "✓"
		if traceAnalysis.ServiceErrors[svc] > 0 {
			status = fmt.Sprintf("✗ (%d errors)", traceAnalysis.ServiceErrors[svc])
		}
		response.WriteString(fmt.Sprintf("%s %s %s\n", prefix, svc, status))
	}
	response.WriteString("```\n\n")

	// Timeline
	response.WriteString("## ⏱️ Timeline\n")
	response.WriteString(fmt.Sprintf("**Duration:** %s\n\n", traceAnalysis.Duration))

	response.WriteString("| Time | Service | Severity | Message |\n")
	response.WriteString("|------|---------|----------|----------|\n")

	for i, entry := range traceAnalysis.Timeline {
		if i >= 20 {
			response.WriteString(fmt.Sprintf("| ... | ... | ... | ... (%d more entries) |\n", len(traceAnalysis.Timeline)-20))
			break
		}
		var sev string
		switch entry.Severity {
		case "ERROR", "CRITICAL":
			sev = "🔴 " + entry.Severity
		case "WARNING":
			sev = "🟡 " + entry.Severity
		default:
			sev = entry.Severity
		}
		msg := truncateString(entry.Message, 60)
		response.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			entry.Timestamp.Format("15:04:05.000"),
			entry.Service,
			sev,
			msg))
	}
	response.WriteString("\n")

	// Error propagation
	if len(traceAnalysis.Errors) > 0 {
		response.WriteString("## 🚨 Errors in Trace\n")
		for i, err := range traceAnalysis.Errors {
			if i >= 5 {
				response.WriteString(fmt.Sprintf("... and %d more errors\n", len(traceAnalysis.Errors)-5))
				break
			}
			response.WriteString(fmt.Sprintf("- **%s** @ %s: %s\n",
				err.Service,
				err.Timestamp.Format("15:04:05.000"),
				truncateString(err.Message, 80)))
		}
		response.WriteString("\n")

		// Root cause suggestion
		if len(traceAnalysis.Errors) > 0 {
			firstError := traceAnalysis.Errors[0]
			response.WriteString("## 🎯 Likely Root Cause\n")
			response.WriteString(fmt.Sprintf("The first error occurred in **%s** at %s:\n",
				firstError.Service, firstError.Timestamp.Format("15:04:05.000")))
			response.WriteString(fmt.Sprintf("```\n%s\n```\n\n", truncateString(firstError.Message, 200)))
		}
	}

	// Record tool use
	session := GetSession()
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"trace_id":    traceID,
		"time_window": timeWindow,
		"events":      len(events),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// TraceAnalysis contains the analyzed trace information
type TraceAnalysis struct {
	TraceID       string
	Services      []string // Ordered by first appearance
	ServiceErrors map[string]int
	Timeline      []TraceEntry
	Errors        []TraceEntry
	StartTime     time.Time
	EndTime       time.Time
	Duration      string
}

// TraceEntry represents a single log entry in the trace
type TraceEntry struct {
	Timestamp time.Time
	Service   string
	Severity  string
	SpanID    string
	Message   string
}

// analyzeTrace processes trace events into a structured analysis
func (t *GetTraceContextTool) analyzeTrace(events []interface{}) *TraceAnalysis {
	analysis := &TraceAnalysis{
		Services:      make([]string, 0),
		ServiceErrors: make(map[string]int),
		Timeline:      make([]TraceEntry, 0),
		Errors:        make([]TraceEntry, 0),
	}

	seenServices := make(map[string]bool)

	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}

		entry := TraceEntry{
			Timestamp: extractTimestampFromEvent(eventMap),
			Service:   extractAppFromEvent(eventMap),
			Message:   extractMessageField(eventMap),
		}

		// Extract severity
		sev, _ := extractSeverityFromEvent(eventMap)
		entry.Severity = sev

		// Extract span ID if available
		if userData, ok := eventMap["user_data"].(map[string]interface{}); ok {
			if spanID, ok := userData["span_id"].(string); ok {
				entry.SpanID = spanID
			}
		}

		// Track services in order of first appearance
		if entry.Service != "" && !seenServices[entry.Service] {
			seenServices[entry.Service] = true
			analysis.Services = append(analysis.Services, entry.Service)
		}

		// Track errors by service
		if sev == "ERROR" || sev == "CRITICAL" {
			analysis.ServiceErrors[entry.Service]++
			analysis.Errors = append(analysis.Errors, entry)
		}

		analysis.Timeline = append(analysis.Timeline, entry)

		// Track time bounds
		if !entry.Timestamp.IsZero() {
			if analysis.StartTime.IsZero() || entry.Timestamp.Before(analysis.StartTime) {
				analysis.StartTime = entry.Timestamp
			}
			if entry.Timestamp.After(analysis.EndTime) {
				analysis.EndTime = entry.Timestamp
			}
		}
	}

	// Sort timeline by timestamp
	sort.Slice(analysis.Timeline, func(i, j int) bool {
		return analysis.Timeline[i].Timestamp.Before(analysis.Timeline[j].Timestamp)
	})

	// Sort errors by timestamp
	sort.Slice(analysis.Errors, func(i, j int) bool {
		return analysis.Errors[i].Timestamp.Before(analysis.Errors[j].Timestamp)
	})

	// Calculate duration
	if !analysis.StartTime.IsZero() && !analysis.EndTime.IsZero() {
		analysis.Duration = analysis.EndTime.Sub(analysis.StartTime).String()
	} else {
		analysis.Duration = "unknown"
	}

	return analysis
}

// ============================================================================
// HIERARCHICAL LOG GROUP TRAVERSAL (Causal Discovery)
// ============================================================================

// AnalyzeCausalChainTool traverses log groups hierarchically to
// differentiate symptoms from root causes.
type AnalyzeCausalChainTool struct {
	*BaseTool
}

// NewAnalyzeCausalChainTool creates a new AnalyzeCausalChainTool
func NewAnalyzeCausalChainTool(c *client.Client, l *zap.Logger) *AnalyzeCausalChainTool {
	return &AnalyzeCausalChainTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *AnalyzeCausalChainTool) Name() string { return "analyze_causal_chain" }

// Annotations returns tool hints for LLMs
func (t *AnalyzeCausalChainTool) Annotations() *mcp.ToolAnnotations {
	return WorkflowAnnotations("Analyze Causal Chain")
}

// DefaultTimeout returns the timeout
func (t *AnalyzeCausalChainTool) DefaultTimeout() time.Duration {
	return DefaultWorkflowTimeout
}

// Description returns the tool description
func (t *AnalyzeCausalChainTool) Description() string {
	return `Analyze log groups hierarchically to differentiate symptoms from root causes.

**Purpose (AURORA/RCD 2025 Causal Discovery):**
Traverses the dependency graph of errors to find the originating failure,
distinguishing cascading effects (symptoms) from the root cause.

**Best for:**
- Complex incidents with cascading failures
- Understanding error propagation paths
- Finding the true root cause among many errors

**Algorithm:**
1. Identify all error patterns in the time window
2. Build temporal dependency graph (which errors preceded others)
3. Analyze service dependencies to find upstream sources
4. Rank root cause candidates by causal evidence

**Related tools:** analyze_log_delta, get_trace_context, investigate_incident`
}

// InputSchema returns the input schema
func (t *AnalyzeCausalChainTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"symptom_query": map[string]interface{}{
				"type":        "string",
				"description": "Description of the symptom you're investigating (e.g., 'high latency on checkout')",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to analyze: 15m, 1h, 6h (default: 1h)",
				"enum":        []string{"15m", "1h", "6h"},
				"default":     "1h",
			},
			"affected_service": map[string]interface{}{
				"type":        "string",
				"description": "Service showing the symptom (optional, helps focus the analysis)",
			},
		},
		"required": []string{"symptom_query"},
	}
}

// Metadata returns semantic metadata for tool discovery
func (t *AnalyzeCausalChainTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryObservability, CategoryAIHelper},
		Keywords:      []string{"causal", "root cause", "dependency", "cascade", "propagation", "upstream", "downstream"},
		Complexity:    ComplexityAdvanced,
		UseCases:      []string{"Find root cause", "Analyze cascading failures", "Understand error propagation"},
		RelatedTools:  []string{"analyze_log_delta", "get_trace_context", "investigate_incident"},
		ChainPosition: ChainMiddle,
	}
}

// Execute analyzes the causal chain
func (t *AnalyzeCausalChainTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	symptom, err := GetStringParam(args, "symptom_query", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	timeRange, _ := GetStringParam(args, "time_range", false)
	if timeRange == "" {
		timeRange = "1h"
	}

	affectedService, _ := GetStringParam(args, "affected_service", false)

	// Calculate time range
	endTime := time.Now().UTC()
	var startTime time.Time
	switch timeRange {
	case "15m":
		startTime = endTime.Add(-15 * time.Minute)
	case "6h":
		startTime = endTime.Add(-6 * time.Hour)
	default:
		startTime = endTime.Add(-1 * time.Hour)
	}

	// Query all error logs in the time range
	query := "source logs | filter $m.severity >= ERROR"
	if affectedService != "" {
		query += fmt.Sprintf(" | filter $l.applicationname == '%s'", escapeDataPrimeString(affectedService))
	}
	query += " | limit 1000"

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
			},
		},
		AcceptSSE: true,
		Timeout:   DefaultWorkflowTimeout,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}

	events, _ := result["events"].([]interface{})

	// Cluster logs and build causal graph
	clusters := ClusterLogs(events)
	causalGraph := t.buildCausalGraph(clusters, events)

	// Build response
	var response strings.Builder
	response.WriteString("# 🔍 Causal Chain Analysis\n\n")
	response.WriteString(fmt.Sprintf("**Symptom:** %s\n", symptom))
	response.WriteString(fmt.Sprintf("**Time Range:** %s\n", timeRange))
	if affectedService != "" {
		response.WriteString(fmt.Sprintf("**Affected Service:** %s\n", affectedService))
	}
	response.WriteString(fmt.Sprintf("**Error Clusters Analyzed:** %d\n\n", len(clusters)))

	if len(clusters) == 0 {
		response.WriteString("## No Errors Found\n\n")
		response.WriteString("No error logs were found in the specified time range.\n")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: response.String()},
			},
		}, nil
	}

	// Causal hierarchy
	response.WriteString("## 📊 Causal Hierarchy\n\n")
	response.WriteString("Errors ordered from most likely root cause to downstream symptoms:\n\n")

	for i, node := range causalGraph.Nodes {
		if i >= 10 {
			response.WriteString(fmt.Sprintf("... and %d more error patterns\n", len(causalGraph.Nodes)-10))
			break
		}

		indent := strings.Repeat("  ", node.Depth)
		icon := "├─"
		if i == len(causalGraph.Nodes)-1 || (i < len(causalGraph.Nodes)-1 && causalGraph.Nodes[i+1].Depth <= node.Depth) {
			icon = "└─"
		}

		label := "SYMPTOM"
		if node.IsRootCause {
			label = "🎯 ROOT CAUSE"
		} else if node.Depth == 0 {
			label = "UPSTREAM"
		}

		response.WriteString(fmt.Sprintf("%s%s **[%s]** %s\n", indent, icon, label, node.Cluster.RootCause))
		response.WriteString(fmt.Sprintf("%s   - Apps: %s\n", indent, strings.Join(node.Cluster.Apps, ", ")))
		response.WriteString(fmt.Sprintf("%s   - Count: %d | First: %s\n", indent, node.Cluster.Count, node.Cluster.FirstSeen.Format("15:04:05")))
		response.WriteString(fmt.Sprintf("%s   - Template: `%s`\n\n", indent, truncateString(node.Cluster.Template, 80)))
	}

	// Root cause summary
	response.WriteString("## 🎯 Root Cause Assessment\n\n")
	if len(causalGraph.RootCauses) > 0 {
		for i, rc := range causalGraph.RootCauses {
			strength := "moderate"
			if rc.Confidence > 0.8 {
				strength = "strong"
			} else if rc.Confidence < 0.5 {
				strength = "weak"
			}
			response.WriteString(fmt.Sprintf("%d. **[%s confidence]** %s in **%s**\n",
				i+1, strength, rc.Cluster.RootCause, strings.Join(rc.Cluster.Apps, ", ")))
			response.WriteString(fmt.Sprintf("   - First occurrence: %s\n", rc.Cluster.FirstSeen.Format("15:04:05")))
			response.WriteString(fmt.Sprintf("   - Pattern: `%s`\n\n", truncateString(rc.Cluster.Template, 60)))
		}
	} else {
		response.WriteString("Unable to determine a clear root cause from the available data.\n")
		response.WriteString("**Suggestions:**\n")
		response.WriteString("- Expand the time range to capture the initial failure\n")
		response.WriteString("- Use `analyze_log_delta` to compare before/after incident time\n")
	}

	// Propagation path
	if len(causalGraph.PropagationPath) > 0 {
		response.WriteString("## 🔄 Error Propagation Path\n")
		response.WriteString("```\n")
		for i, step := range causalGraph.PropagationPath {
			arrow := "──▶"
			if i == len(causalGraph.PropagationPath)-1 {
				arrow = ""
			}
			response.WriteString(fmt.Sprintf("%s %s ", step, arrow))
		}
		response.WriteString("\n```\n\n")
	}

	// Recommended actions
	response.WriteString("## 📋 Recommended Actions\n")
	if len(causalGraph.RootCauses) > 0 {
		rc := causalGraph.RootCauses[0]
		response.WriteString(fmt.Sprintf("1. Investigate **%s** in %s\n", rc.Cluster.RootCause, strings.Join(rc.Cluster.Apps, ", ")))
		response.WriteString(fmt.Sprintf("2. Check logs around %s for the initial trigger\n", rc.Cluster.FirstSeen.Format("15:04:05")))
	}
	response.WriteString("3. Use `get_trace_context` with a trace ID from the affected request\n")
	response.WriteString("4. Use `analyze_log_delta` to compare with a healthy baseline\n")

	// Record tool use
	session := GetSession()
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"symptom":          symptom,
		"time_range":       timeRange,
		"affected_service": affectedService,
		"clusters":         len(clusters),
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// CausalGraph represents the dependency graph of error patterns
type CausalGraph struct {
	Nodes           []CausalNode
	RootCauses      []RootCauseCandidate
	PropagationPath []string
}

// CausalNode represents a node in the causal graph
type CausalNode struct {
	Cluster     *LogCluster
	Depth       int // 0 = potential root cause, higher = more downstream
	IsRootCause bool
	Children    []*CausalNode
}

// RootCauseCandidate represents a potential root cause
type RootCauseCandidate struct {
	Cluster    *LogCluster
	Confidence float64 // 0-1 confidence score
}

// buildCausalGraph builds a causal dependency graph from error clusters
func (t *AnalyzeCausalChainTool) buildCausalGraph(clusters []*LogCluster, _ []interface{}) *CausalGraph {
	graph := &CausalGraph{
		Nodes:      make([]CausalNode, 0, len(clusters)),
		RootCauses: make([]RootCauseCandidate, 0),
	}

	if len(clusters) == 0 {
		return graph
	}

	// Sort clusters by first seen time (earliest first)
	sorted := make([]*LogCluster, len(clusters))
	copy(sorted, clusters)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FirstSeen.Before(sorted[j].FirstSeen)
	})

	// Assign depths based on temporal ordering and root cause category
	// Errors that appear first and are fundamental (network, memory, etc.) are more likely root causes
	rootCauseWeight := map[string]int{
		"MEMORY_PRESSURE":   1,
		"NETWORK_FAILURE":   1,
		"STORAGE_FAILURE":   1,
		"DNS_FAILURE":       1,
		"DATABASE_FAILURE":  2,
		"CPU_PRESSURE":      2,
		"TLS_FAILURE":       2,
		"AUTH_FAILURE":      3,
		"RATE_LIMITED":      3,
		"TIMEOUT":           4, // Often a symptom
		"CODE_BUG":          3,
		"K8S_ORCHESTRATION": 2,
		"UNKNOWN":           5,
	}

	// Calculate scores for each cluster
	type scoredCluster struct {
		cluster   *LogCluster
		score     float64
		timeScore float64
		typeScore float64
	}
	scored := make([]scoredCluster, len(sorted))

	firstTime := sorted[0].FirstSeen
	lastTime := sorted[len(sorted)-1].FirstSeen
	timeSpan := lastTime.Sub(firstTime).Seconds()
	if timeSpan < 1 {
		timeSpan = 1
	}

	for i, c := range sorted {
		// Time score: earlier = higher score (1.0 for first, 0.0 for last)
		timeScore := 1.0
		if len(sorted) > 1 {
			timeScore = 1.0 - (c.FirstSeen.Sub(firstTime).Seconds() / timeSpan)
		}

		// Type score: lower weight = more likely root cause
		weight := rootCauseWeight[c.RootCause]
		if weight == 0 {
			weight = 5
		}
		typeScore := 1.0 - (float64(weight-1) / 5.0)

		// Combined score (time weighted more heavily)
		score := (timeScore * 0.6) + (typeScore * 0.4)

		scored[i] = scoredCluster{
			cluster:   c,
			score:     score,
			timeScore: timeScore,
			typeScore: typeScore,
		}
	}

	// Sort by score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Build nodes with depth based on score ranking
	for i, sc := range scored {
		depth := 0
		if i > 0 {
			depth = i // Depth increases with lower scores
		}
		if depth > 3 {
			depth = 3 // Cap depth at 3
		}

		isRootCause := i == 0 && sc.score > 0.6

		node := CausalNode{
			Cluster:     sc.cluster,
			Depth:       depth,
			IsRootCause: isRootCause,
		}
		graph.Nodes = append(graph.Nodes, node)

		if isRootCause || (i < 3 && sc.score > 0.5) {
			graph.RootCauses = append(graph.RootCauses, RootCauseCandidate{
				Cluster:    sc.cluster,
				Confidence: sc.score,
			})
		}
	}

	// Build propagation path from apps
	seenApps := make(map[string]bool)
	for _, node := range graph.Nodes {
		for _, app := range node.Cluster.Apps {
			if !seenApps[app] {
				seenApps[app] = true
				graph.PropagationPath = append(graph.PropagationPath, app)
			}
		}
	}

	return graph
}
