// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements advanced Root Cause Analysis (RCA) capabilities
// aligned with SOTA 2025 paradigms: Causal Discovery, System 2 Reasoning,
// and Semantic Log Clustering.
package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ============================================================================
// SYNC.POOL FOR HIGH-THROUGHPUT MEMORY OPTIMIZATION
// ============================================================================

// logEntryPool provides object reuse for log parsing to reduce GC pressure
// during high-throughput multi-agent swarm scenarios.
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{
			Labels:   make(map[string]string, 8),
			Metadata: make(map[string]interface{}, 8),
			Data:     make(map[string]interface{}, 16),
		}
	},
}

// LogEntry represents a parsed log entry with pooled allocation
type LogEntry struct {
	Timestamp   time.Time
	Severity    string
	SeverityNum int
	App         string
	Subsystem   string
	Message     string
	TraceID     string
	SpanID      string
	Labels      map[string]string
	Metadata    map[string]interface{}
	Data        map[string]interface{}
	Template    string // Extracted log template (invariant pattern)
	TemplateID  string // Hash of template for grouping
}

// Reset clears the LogEntry for reuse
func (e *LogEntry) Reset() {
	e.Timestamp = time.Time{}
	e.Severity = ""
	e.SeverityNum = 0
	e.App = ""
	e.Subsystem = ""
	e.Message = ""
	e.TraceID = ""
	e.SpanID = ""
	e.Template = ""
	e.TemplateID = ""
	for k := range e.Labels {
		delete(e.Labels, k)
	}
	for k := range e.Metadata {
		delete(e.Metadata, k)
	}
	for k := range e.Data {
		delete(e.Data, k)
	}
}

// AcquireLogEntry gets a LogEntry from the pool
func AcquireLogEntry() *LogEntry {
	return logEntryPool.Get().(*LogEntry)
}

// ReleaseLogEntry returns a LogEntry to the pool
func ReleaseLogEntry(e *LogEntry) {
	e.Reset()
	logEntryPool.Put(e)
}

// ============================================================================
// LOG TEMPLATE EXTRACTION (LogAssist/LogBatcher 2025 Pattern)
// ============================================================================

// templateVarPatterns are regex patterns for extracting variable parts of log messages
var templateVarPatterns = []*regexp.Regexp{
	// UUIDs
	regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),
	// Hex IDs (trace IDs, span IDs, etc.)
	regexp.MustCompile(`\b[0-9a-fA-F]{16,64}\b`),
	// IP addresses
	regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
	// Timestamps (various formats)
	regexp.MustCompile(`\d{4}[-/]\d{2}[-/]\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`),
	// Durations (e.g., 123ms, 1.5s)
	regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h)\b`),
	// Numbers (integers and floats)
	regexp.MustCompile(`\b\d+(?:\.\d+)?\b`),
	// Quoted strings
	regexp.MustCompile(`"[^"]*"`),
	regexp.MustCompile(`'[^']*'`),
	// File paths
	regexp.MustCompile(`/[^\s:]+`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
}

// ExtractLogTemplate extracts the invariant template from a log message
// by replacing variable parts with placeholders. This enables semantic
// clustering of logs with the same structure but different values.
func ExtractLogTemplate(message string) (template string, templateID string) {
	template = message

	// Replace variable patterns with placeholders
	placeholders := []string{"<UUID>", "<HEX>", "<IP>", "<TIME>", "<DUR>", "<NUM>", "<STR>", "<STR>", "<PATH>", "<EMAIL>"}
	for i, pattern := range templateVarPatterns {
		placeholder := "<VAR>"
		if i < len(placeholders) {
			placeholder = placeholders[i]
		}
		template = pattern.ReplaceAllString(template, placeholder)
	}

	// Normalize whitespace
	template = regexp.MustCompile(`\s+`).ReplaceAllString(template, " ")
	template = strings.TrimSpace(template)

	// Generate template ID (hash for efficient grouping)
	h := sha256.Sum256([]byte(template))
	templateID = hex.EncodeToString(h[:8])

	return template, templateID
}

// LogCluster represents a group of logs with the same template
type LogCluster struct {
	TemplateID  string    `json:"template_id"`
	Template    string    `json:"template"`
	Count       int       `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Severity    string    `json:"severity"`     // Highest severity in cluster
	SeverityNum int       `json:"severity_num"` // Numeric for sorting
	Apps        []string  `json:"apps"`         // Applications contributing to this cluster
	Samples     []string  `json:"samples"`      // Sample raw messages (max 3)
	TrendRatio  float64   `json:"trend_ratio"`  // Recent vs historical ratio (spike detection)
	IsAnomaly   bool      `json:"is_anomaly"`   // True if spike detected
	RootCause   string    `json:"root_cause"`   // Inferred root cause category
}

// ClusterLogs groups logs by their template for pattern analysis.
// Uses sync.Pool for memory efficiency in high-throughput multi-agent swarm scenarios.
func ClusterLogs(events []interface{}) []*LogCluster {
	clusters := make(map[string]*LogCluster)
	appSets := make(map[string]map[string]bool)

	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}

		// Use pooled LogEntry for efficient parsing
		entry := AcquireLogEntry()
		parseEventIntoEntry(eventMap, entry)

		if entry.Message == "" {
			ReleaseLogEntry(entry)
			continue
		}

		template, templateID := ExtractLogTemplate(entry.Message)

		cluster, exists := clusters[templateID]
		if !exists {
			cluster = &LogCluster{
				TemplateID: templateID,
				Template:   template,
				Samples:    make([]string, 0, 3),
			}
			clusters[templateID] = cluster
			appSets[templateID] = make(map[string]bool)
		}

		cluster.Count++

		// Track severity (keep highest)
		if entry.SeverityNum > cluster.SeverityNum {
			cluster.Severity = entry.Severity
			cluster.SeverityNum = entry.SeverityNum
		}

		// Track applications
		if entry.App != "" {
			appSets[templateID][entry.App] = true
		}

		// Collect samples (max 3)
		if len(cluster.Samples) < 3 {
			msg := entry.Message
			if len(msg) > 200 {
				msg = msg[:197] + "..."
			}
			cluster.Samples = append(cluster.Samples, msg)
		}

		// Track time range
		if !entry.Timestamp.IsZero() {
			if cluster.FirstSeen.IsZero() || entry.Timestamp.Before(cluster.FirstSeen) {
				cluster.FirstSeen = entry.Timestamp
			}
			if entry.Timestamp.After(cluster.LastSeen) {
				cluster.LastSeen = entry.Timestamp
			}
		}

		// Return entry to pool
		ReleaseLogEntry(entry)
	}

	// Convert app sets to slices and infer root causes
	result := make([]*LogCluster, 0, len(clusters))
	for templateID, cluster := range clusters {
		for app := range appSets[templateID] {
			cluster.Apps = append(cluster.Apps, app)
		}
		cluster.RootCause = inferRootCause(cluster.Template)
		result = append(result, cluster)
	}

	// Sort by count (descending) then severity
	sort.Slice(result, func(i, j int) bool {
		if result[i].SeverityNum != result[j].SeverityNum {
			return result[i].SeverityNum > result[j].SeverityNum
		}
		return result[i].Count > result[j].Count
	})

	return result
}

// parseEventIntoEntry populates a LogEntry from an event map.
// This enables efficient reuse of LogEntry via sync.Pool.
func parseEventIntoEntry(eventMap map[string]interface{}, entry *LogEntry) {
	// Extract message
	entry.Message = extractMessageField(eventMap)

	// Extract severity
	entry.Severity, entry.SeverityNum = extractSeverityFromEvent(eventMap)

	// Extract application
	entry.App = extractAppFromEvent(eventMap)

	// Extract subsystem
	entry.Subsystem = extractSubsystemFromEvent(eventMap)

	// Extract timestamp
	entry.Timestamp = extractTimestampFromEvent(eventMap)

	// Extract trace context
	entry.TraceID, entry.SpanID = extractTraceContext(eventMap)

	// Extract template (will be set by caller if needed)
	entry.Template = ""
	entry.TemplateID = ""
}

// extractSubsystemFromEvent extracts subsystem name from event
func extractSubsystemFromEvent(event map[string]interface{}) string {
	// Try labels
	if labels, ok := event["labels"].(map[string]interface{}); ok {
		if sub, ok := labels["subsystemname"].(string); ok {
			return sub
		}
	}

	// Try direct field
	if sub, ok := event["subsystemname"].(string); ok {
		return sub
	}
	if sub, ok := event["subsystem"].(string); ok {
		return sub
	}

	return ""
}

// extractTraceContext extracts trace_id and span_id from event
func extractTraceContext(event map[string]interface{}) (traceID, spanID string) {
	// Try user_data first (common for structured logs)
	if userData, ok := event["user_data"].(map[string]interface{}); ok {
		if tid, ok := userData["trace_id"].(string); ok {
			traceID = tid
		} else if tid, ok := userData["traceId"].(string); ok {
			traceID = tid
		} else if tid, ok := userData["traceID"].(string); ok {
			traceID = tid
		}

		if sid, ok := userData["span_id"].(string); ok {
			spanID = sid
		} else if sid, ok := userData["spanId"].(string); ok {
			spanID = sid
		} else if sid, ok := userData["spanID"].(string); ok {
			spanID = sid
		}
	}

	// Try direct fields if not found in user_data
	if traceID == "" {
		if tid, ok := event["trace_id"].(string); ok {
			traceID = tid
		} else if tid, ok := event["traceId"].(string); ok {
			traceID = tid
		}
	}

	if spanID == "" {
		if sid, ok := event["span_id"].(string); ok {
			spanID = sid
		} else if sid, ok := event["spanId"].(string); ok {
			spanID = sid
		}
	}

	return traceID, spanID
}

// inferRootCause categorizes the likely root cause based on template patterns
func inferRootCause(template string) string {
	lower := strings.ToLower(template)

	// Root cause categories aligned with AURORA/RCD 2025 causal discovery
	rootCauses := []struct {
		patterns []string
		cause    string
	}{
		{[]string{"oom", "out of memory", "heap", "memory limit", "cannot allocate"}, "MEMORY_PRESSURE"},
		{[]string{"timeout", "timed out", "deadline exceeded", "context deadline"}, "TIMEOUT"},
		{[]string{"connection refused", "connection reset", "no route to host", "network unreachable"}, "NETWORK_FAILURE"},
		{[]string{"disk full", "no space left", "i/o error", "read-only file system"}, "STORAGE_FAILURE"},
		{[]string{"permission denied", "unauthorized", "forbidden", "401", "403"}, "AUTH_FAILURE"},
		{[]string{"null pointer", "nil pointer", "segmentation fault", "panic"}, "CODE_BUG"},
		{[]string{"rate limit", "throttl", "too many requests", "429"}, "RATE_LIMITED"},
		{[]string{"dns", "name resolution", "could not resolve"}, "DNS_FAILURE"},
		{[]string{"certificate", "ssl", "tls", "x509"}, "TLS_FAILURE"},
		{[]string{"database", "sql", "query failed", "deadlock"}, "DATABASE_FAILURE"},
		{[]string{"cpu", "load average", "high load"}, "CPU_PRESSURE"},
		{[]string{"kubernetes", "pod", "container", "evict"}, "K8S_ORCHESTRATION"},
	}

	for _, rc := range rootCauses {
		for _, pattern := range rc.patterns {
			if strings.Contains(lower, pattern) {
				return rc.cause
			}
		}
	}

	return "UNKNOWN"
}

// extractMessageField extracts the message from various event structures
func extractMessageField(event map[string]interface{}) string {
	// Try direct message fields
	for _, field := range []string{"message", "msg", "text", "log", "_message"} {
		if msg, ok := event[field].(string); ok && msg != "" {
			return msg
		}
	}

	// Try nested user_data
	if userData, ok := event["user_data"].(map[string]interface{}); ok {
		for _, field := range []string{"message", "msg", "text"} {
			if msg, ok := userData[field].(string); ok && msg != "" {
				return msg
			}
		}
	}

	return ""
}

// extractSeverityFromEvent extracts severity string and numeric value
func extractSeverityFromEvent(event map[string]interface{}) (string, int) {
	severityMap := map[string]int{
		"VERBOSE": 1, "DEBUG": 2, "INFO": 3, "WARNING": 4, "ERROR": 5, "CRITICAL": 6,
	}

	// Try direct severity
	if sev, ok := event["severity"].(string); ok {
		return sev, severityMap[strings.ToUpper(sev)]
	}
	if sevNum, ok := event["severity"].(float64); ok {
		for name, num := range severityMap {
			if num == int(sevNum) {
				return name, int(sevNum)
			}
		}
	}

	// Try metadata
	if meta, ok := event["metadata"].(map[string]interface{}); ok {
		if sev, ok := meta["severity"].(string); ok {
			return sev, severityMap[strings.ToUpper(sev)]
		}
	}

	return "INFO", 3
}

// extractAppFromEvent extracts application name from event
func extractAppFromEvent(event map[string]interface{}) string {
	// Try labels
	if labels, ok := event["labels"].(map[string]interface{}); ok {
		if app, ok := labels["applicationname"].(string); ok {
			return app
		}
	}

	// Try direct field
	if app, ok := event["applicationname"].(string); ok {
		return app
	}
	if app, ok := event["app"].(string); ok {
		return app
	}

	return ""
}

// extractTimestampFromEvent parses timestamp from event
func extractTimestampFromEvent(event map[string]interface{}) time.Time {
	for _, field := range []string{"timestamp", "@timestamp", "time", "_time"} {
		if ts, ok := event[field].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				return t
			}
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// ============================================================================
// VERIFICATION TRACES (System 2 Reasoning / GRPO 2025)
// ============================================================================

// VerificationTrace provides structured signals for reasoning models
// to verify execution, detect inconsistencies, and self-correct.
type VerificationTrace struct {
	// Execution metadata
	ToolName     string `json:"tool_name"`
	ExecutionMs  int64  `json:"execution_ms"`
	Timestamp    string `json:"timestamp"`
	RequestHash  string `json:"request_hash"`  // For deduplication
	ResponseHash string `json:"response_hash"` // For idempotency

	// Result verification
	ResultCount    int    `json:"result_count"`
	Truncated      bool   `json:"truncated"`
	TruncatedFrom  int    `json:"truncated_from,omitempty"`
	CompressionLvl string `json:"compression_level,omitempty"`

	// Causal chain metadata (for hierarchical traversal)
	CausalDepth      int    `json:"causal_depth"`          // How many layers deep in investigation
	ParentToolCall   string `json:"parent_tool,omitempty"` // Previous tool in chain
	CausalHypothesis string `json:"causal_hypothesis,omitempty"`
	EvidenceStrength string `json:"evidence_strength"` // strong, moderate, weak

	// Delta analysis (for AnalyzeLogDelta pattern)
	FrequencyHash  string         `json:"frequency_hash,omitempty"`  // Hash of pattern distribution
	DeltaFromPrev  map[string]int `json:"delta_from_prev,omitempty"` // Changes from previous query
	AnomaliesFound []string       `json:"anomalies_found,omitempty"`

	// Retry guidance
	RetryAllowed    bool   `json:"retry_allowed"`
	SuggestedAction string `json:"suggested_action,omitempty"`
}

// NewVerificationTrace creates a verification trace for a tool execution
func NewVerificationTrace(toolName string, startTime time.Time) *VerificationTrace {
	return &VerificationTrace{
		ToolName:         toolName,
		ExecutionMs:      time.Since(startTime).Milliseconds(),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		RetryAllowed:     !isDestructiveTool(toolName),
		EvidenceStrength: "moderate",
	}
}

// ============================================================================
// ANALYZE LOG DELTA TOOL (Outcome-Oriented System 2 Reasoning)
// ============================================================================

// AnalyzeLogDeltaTool compares log patterns between time windows
// to identify what changed, enabling root cause analysis.
type AnalyzeLogDeltaTool struct {
	*BaseTool
}

// NewAnalyzeLogDeltaTool creates a new AnalyzeLogDeltaTool
func NewAnalyzeLogDeltaTool(c *client.Client, l *zap.Logger) *AnalyzeLogDeltaTool {
	return &AnalyzeLogDeltaTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *AnalyzeLogDeltaTool) Name() string { return "analyze_log_delta" }

// Annotations returns tool hints for LLMs
func (t *AnalyzeLogDeltaTool) Annotations() *mcp.ToolAnnotations {
	return WorkflowAnnotations("Analyze Log Delta")
}

// DefaultTimeout returns the timeout for delta analysis
func (t *AnalyzeLogDeltaTool) DefaultTimeout() time.Duration {
	return DefaultWorkflowTimeout
}

// Description returns the tool description
func (t *AnalyzeLogDeltaTool) Description() string {
	return `Analyze changes in log patterns between two time windows (delta analysis).

**Purpose (System 2 Reasoning):**
Returns structured "Verification Traces" with frequency hashes and pattern diffs,
enabling reasoning models to identify what changed and infer causality.

**Best for:**
- Root cause analysis: "What changed when the incident started?"
- Deployment verification: "Did the new release introduce new error patterns?"
- Anomaly detection: "Are there unusual patterns compared to baseline?"

**Returns:**
- Clustered log templates (semantic grouping)
- Pattern frequency diffs (appeared/disappeared/changed)
- Causal hypotheses ranked by evidence strength
- Verification traces for model self-correction

**Related tools:** investigate_incident, query_logs, health_check`
}

// InputSchema returns the input schema
func (t *AnalyzeLogDeltaTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Application name to analyze (optional, analyzes all if not specified)",
			},
			"incident_time": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the incident started (ISO 8601). Compares before vs after this time.",
			},
			"window_size": map[string]interface{}{
				"type":        "string",
				"description": "Size of comparison windows: 5m, 15m, 30m, 1h (default: 15m)",
				"enum":        []string{"5m", "15m", "30m", "1h"},
				"default":     "15m",
			},
			"min_severity": map[string]interface{}{
				"type":        "string",
				"description": "Minimum severity to analyze: warning, error, critical (default: warning)",
				"enum":        []string{"warning", "error", "critical"},
				"default":     "warning",
			},
		},
		"required": []string{"incident_time"},
	}
}

// Metadata returns semantic metadata for tool discovery
func (t *AnalyzeLogDeltaTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryObservability, CategoryAIHelper},
		Keywords:      []string{"delta", "diff", "change", "compare", "root cause", "before after", "anomaly", "regression"},
		Complexity:    ComplexityAdvanced,
		UseCases:      []string{"Root cause analysis", "Deployment verification", "Anomaly detection", "Incident investigation"},
		RelatedTools:  []string{"investigate_incident", "query_logs", "health_check"},
		ChainPosition: ChainMiddle,
	}
}

// Execute runs the delta analysis
func (t *AnalyzeLogDeltaTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	startTime := time.Now()
	session := GetSession()

	// Parse parameters
	application, _ := GetStringParam(args, "application", false)
	incidentTimeStr, err := GetStringParam(args, "incident_time", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	windowSize, _ := GetStringParam(args, "window_size", false)
	if windowSize == "" {
		windowSize = "15m"
	}
	minSeverity, _ := GetStringParam(args, "min_severity", false)
	if minSeverity == "" {
		minSeverity = "warning"
	}

	// Parse incident time
	incidentTime, err := time.Parse(time.RFC3339, incidentTimeStr)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Invalid incident_time format: %v", err)), nil
	}

	// Calculate window duration
	var windowDuration time.Duration
	switch windowSize {
	case "5m":
		windowDuration = 5 * time.Minute
	case "30m":
		windowDuration = 30 * time.Minute
	case "1h":
		windowDuration = 1 * time.Hour
	default:
		windowDuration = 15 * time.Minute
	}

	// Define time windows
	// "Before" window: [incident - 2*window, incident - window]
	// "After" window: [incident, incident + window]
	beforeStart := incidentTime.Add(-2 * windowDuration)
	beforeEnd := incidentTime.Add(-windowDuration)
	afterStart := incidentTime
	afterEnd := incidentTime.Add(windowDuration)

	// Build query
	severityValue := "WARNING"
	switch minSeverity {
	case "error":
		severityValue = "ERROR"
	case "critical":
		severityValue = "CRITICAL"
	}

	queryBase := fmt.Sprintf("source logs | filter $m.severity >= %s", severityValue)
	if application != "" {
		queryBase += fmt.Sprintf(" | filter $l.applicationname == '%s'", escapeDataPrimeString(application))
	}
	queryBase += " | limit 500"

	// Execute queries for both windows
	beforeEvents, err := t.queryWindow(ctx, queryBase, beforeStart, beforeEnd)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to query 'before' window: %v", err)), nil
	}

	afterEvents, err := t.queryWindow(ctx, queryBase, afterStart, afterEnd)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to query 'after' window: %v", err)), nil
	}

	// Cluster logs by template
	beforeClusters := ClusterLogs(beforeEvents)
	afterClusters := ClusterLogs(afterEvents)

	// Analyze delta
	delta := t.analyzeDelta(beforeClusters, afterClusters)

	// Build verification trace
	trace := NewVerificationTrace(t.Name(), startTime)
	trace.CausalDepth = 1
	trace.ResultCount = len(afterEvents)
	if session.GetInvestigation() != nil {
		trace.CausalDepth = len(session.GetInvestigation().ToolsUsed) + 1
		trace.ParentToolCall = session.GetInvestigation().ToolsUsed[len(session.GetInvestigation().ToolsUsed)-1]
	}

	// Format response
	var response strings.Builder
	response.WriteString("# 🔬 Log Delta Analysis\n\n")
	response.WriteString(fmt.Sprintf("**Incident Time:** %s\n", incidentTime.Format(time.RFC3339)))
	response.WriteString(fmt.Sprintf("**Window Size:** %s\n", windowSize))
	response.WriteString(fmt.Sprintf("**Application:** %s\n\n", orDefault(application, "All")))

	response.WriteString("## Time Windows\n")
	response.WriteString(fmt.Sprintf("- **Before:** %s → %s (%d events)\n",
		beforeStart.Format("15:04:05"), beforeEnd.Format("15:04:05"), len(beforeEvents)))
	response.WriteString(fmt.Sprintf("- **After:** %s → %s (%d events)\n\n",
		afterStart.Format("15:04:05"), afterEnd.Format("15:04:05"), len(afterEvents)))

	// New patterns (appeared after incident)
	if len(delta.NewPatterns) > 0 {
		response.WriteString("## 🆕 New Error Patterns (appeared after incident)\n")
		for i, p := range delta.NewPatterns {
			if i >= 5 {
				response.WriteString(fmt.Sprintf("... and %d more\n", len(delta.NewPatterns)-5))
				break
			}
			response.WriteString(fmt.Sprintf("\n### Pattern: `%s`\n", p.TemplateID))
			response.WriteString(fmt.Sprintf("- **Count:** %d occurrences\n", p.Count))
			response.WriteString(fmt.Sprintf("- **Severity:** %s\n", p.Severity))
			response.WriteString(fmt.Sprintf("- **Root Cause Category:** %s\n", p.RootCause))
			response.WriteString(fmt.Sprintf("- **Template:** `%s`\n", truncateString(p.Template, 100)))
			if len(p.Samples) > 0 {
				response.WriteString(fmt.Sprintf("- **Sample:** `%s`\n", p.Samples[0]))
			}
		}
		response.WriteString("\n")
	}

	// Spiking patterns
	if len(delta.SpikingPatterns) > 0 {
		response.WriteString("## 📈 Spiking Patterns (increased after incident)\n")
		for i, sp := range delta.SpikingPatterns {
			if i >= 5 {
				response.WriteString(fmt.Sprintf("... and %d more\n", len(delta.SpikingPatterns)-5))
				break
			}
			response.WriteString(fmt.Sprintf("- **%s** (%s): %d → %d (%.1fx increase)\n",
				sp.Pattern.TemplateID, sp.Pattern.RootCause,
				sp.BeforeCount, sp.AfterCount, sp.Ratio))
		}
		response.WriteString("\n")
	}

	// Disappeared patterns
	if len(delta.DisappearedPatterns) > 0 {
		response.WriteString("## ❌ Disappeared Patterns (stopped after incident)\n")
		for i, p := range delta.DisappearedPatterns {
			if i >= 3 {
				break
			}
			response.WriteString(fmt.Sprintf("- `%s` (%s): was %d occurrences\n",
				p.TemplateID, p.RootCause, p.Count))
		}
		response.WriteString("\n")
	}

	// Causal hypotheses
	response.WriteString("## 🎯 Causal Hypotheses\n")
	hypotheses := t.generateCausalHypotheses(delta)
	for i, h := range hypotheses {
		response.WriteString(fmt.Sprintf("%d. **[%s]** %s\n", i+1, h.Strength, h.Description))
	}

	// Verification trace (for System 2 reasoning models)
	response.WriteString("\n---\n### Verification Trace\n")
	response.WriteString("```json\n")
	response.WriteString(fmt.Sprintf(`{
  "tool": "%s",
  "execution_ms": %d,
  "before_events": %d,
  "after_events": %d,
  "new_patterns": %d,
  "spiking_patterns": %d,
  "evidence_strength": "%s",
  "retry_allowed": %v
}`, t.Name(), trace.ExecutionMs, len(beforeEvents), len(afterEvents),
		len(delta.NewPatterns), len(delta.SpikingPatterns),
		trace.EvidenceStrength, trace.RetryAllowed))
	response.WriteString("\n```\n")

	// Record tool use
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"application":   application,
		"incident_time": incidentTimeStr,
		"window_size":   windowSize,
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// queryWindow executes a query for a specific time window
func (t *AnalyzeLogDeltaTool) queryWindow(ctx context.Context, query string, start, end time.Time) ([]interface{}, error) {
	// Prepare query
	query, _, err := PrepareQuery(query, "archive", "dataprime")
	if err != nil {
		return nil, err
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body: map[string]interface{}{
			"query": query,
			"metadata": map[string]interface{}{
				"tier":       "archive",
				"syntax":     "dataprime",
				"start_date": start.Format(time.RFC3339),
				"end_date":   end.Format(time.RFC3339),
				"limit":      500,
			},
		},
		AcceptSSE: true,
		Timeout:   DefaultQueryTimeout,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	events, _ := result["events"].([]interface{})
	return events, nil
}

// DeltaAnalysis contains the comparison between two time windows
type DeltaAnalysis struct {
	NewPatterns         []*LogCluster    // Patterns that appeared after incident
	DisappearedPatterns []*LogCluster    // Patterns that stopped after incident
	SpikingPatterns     []SpikingPattern // Patterns that increased significantly
	StablePatterns      []*LogCluster    // Patterns that remained similar
}

// SpikingPattern represents a pattern that increased in frequency
type SpikingPattern struct {
	Pattern     *LogCluster
	BeforeCount int
	AfterCount  int
	Ratio       float64
}

// analyzeDelta compares before and after clusters
func (t *AnalyzeLogDeltaTool) analyzeDelta(before, after []*LogCluster) *DeltaAnalysis {
	delta := &DeltaAnalysis{}

	beforeMap := make(map[string]*LogCluster)
	for _, c := range before {
		beforeMap[c.TemplateID] = c
	}

	afterMap := make(map[string]*LogCluster)
	for _, c := range after {
		afterMap[c.TemplateID] = c
	}

	// Find new patterns (in after but not in before)
	for id, afterCluster := range afterMap {
		if _, exists := beforeMap[id]; !exists {
			delta.NewPatterns = append(delta.NewPatterns, afterCluster)
		}
	}

	// Find disappeared patterns (in before but not in after)
	for id, beforeCluster := range beforeMap {
		if _, exists := afterMap[id]; !exists {
			delta.DisappearedPatterns = append(delta.DisappearedPatterns, beforeCluster)
		}
	}

	// Find spiking patterns (significant increase)
	for id, afterCluster := range afterMap {
		if beforeCluster, exists := beforeMap[id]; exists {
			if beforeCluster.Count > 0 {
				ratio := float64(afterCluster.Count) / float64(beforeCluster.Count)
				if ratio >= 2.0 { // 2x or more increase
					delta.SpikingPatterns = append(delta.SpikingPatterns, SpikingPattern{
						Pattern:     afterCluster,
						BeforeCount: beforeCluster.Count,
						AfterCount:  afterCluster.Count,
						Ratio:       ratio,
					})
				} else if ratio > 0.5 && ratio < 2.0 {
					delta.StablePatterns = append(delta.StablePatterns, afterCluster)
				}
			}
		}
	}

	// Sort spiking patterns by ratio (descending)
	sort.Slice(delta.SpikingPatterns, func(i, j int) bool {
		return delta.SpikingPatterns[i].Ratio > delta.SpikingPatterns[j].Ratio
	})

	// Sort new patterns by severity then count
	sort.Slice(delta.NewPatterns, func(i, j int) bool {
		if delta.NewPatterns[i].SeverityNum != delta.NewPatterns[j].SeverityNum {
			return delta.NewPatterns[i].SeverityNum > delta.NewPatterns[j].SeverityNum
		}
		return delta.NewPatterns[i].Count > delta.NewPatterns[j].Count
	})

	return delta
}

// CausalHypothesis represents a ranked hypothesis about root cause
type CausalHypothesis struct {
	Description string
	Strength    string // strong, moderate, weak
	Category    string
}

// generateCausalHypotheses creates ranked hypotheses based on delta analysis
func (t *AnalyzeLogDeltaTool) generateCausalHypotheses(delta *DeltaAnalysis) []CausalHypothesis {
	var hypotheses []CausalHypothesis

	// Analyze new patterns for strong signals
	rootCauseCounts := make(map[string]int)
	for _, p := range delta.NewPatterns {
		rootCauseCounts[p.RootCause] += p.Count
	}
	for _, sp := range delta.SpikingPatterns {
		rootCauseCounts[sp.Pattern.RootCause] += sp.AfterCount
	}

	// Generate hypotheses based on dominant root cause categories
	type rcCount struct {
		cause string
		count int
	}
	var sortedCauses []rcCount
	for cause, count := range rootCauseCounts {
		if cause != "UNKNOWN" {
			sortedCauses = append(sortedCauses, rcCount{cause, count})
		}
	}
	sort.Slice(sortedCauses, func(i, j int) bool {
		return sortedCauses[i].count > sortedCauses[j].count
	})

	// Map root causes to hypotheses
	causeDescriptions := map[string]string{
		"MEMORY_PRESSURE":   "Memory exhaustion - check container/pod memory limits and for memory leaks",
		"TIMEOUT":           "Service timeouts - investigate downstream service latency or network issues",
		"NETWORK_FAILURE":   "Network connectivity issues - check network policies, DNS, and service mesh",
		"STORAGE_FAILURE":   "Storage issues - verify disk space, IOPS limits, and mount points",
		"AUTH_FAILURE":      "Authentication/authorization failures - check credentials, tokens, and RBAC",
		"CODE_BUG":          "Application code error - review recent deployments and code changes",
		"RATE_LIMITED":      "Rate limiting triggered - check API quotas and client request patterns",
		"DNS_FAILURE":       "DNS resolution failures - verify DNS configuration and CoreDNS health",
		"TLS_FAILURE":       "TLS/certificate issues - check certificate expiry and trust chains",
		"DATABASE_FAILURE":  "Database errors - check connection pools, query performance, and deadlocks",
		"CPU_PRESSURE":      "CPU contention - review resource limits and horizontal scaling needs",
		"K8S_ORCHESTRATION": "Kubernetes orchestration issues - check pod scheduling, evictions, and node health",
	}

	for i, rc := range sortedCauses {
		if i >= 3 {
			break
		}
		strength := "weak"
		if i == 0 && rc.count > 10 {
			strength = "strong"
		} else if i == 0 || rc.count > 5 {
			strength = "moderate"
		}

		desc := causeDescriptions[rc.cause]
		if desc == "" {
			desc = fmt.Sprintf("Unknown issue category: %s", rc.cause)
		}

		hypotheses = append(hypotheses, CausalHypothesis{
			Description: desc,
			Strength:    strength,
			Category:    rc.cause,
		})
	}

	// If no strong signals, add generic hypotheses
	if len(hypotheses) == 0 {
		hypotheses = append(hypotheses,
			CausalHypothesis{
				Description: "Insufficient error patterns to determine root cause - consider expanding time window",
				Strength:    "weak",
				Category:    "UNKNOWN",
			},
		)
	}

	return hypotheses
}

// orDefault returns s if non-empty, otherwise def
func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
