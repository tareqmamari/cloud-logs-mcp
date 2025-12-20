// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements RCA document generation for structured incident reports.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ============================================================================
// RCA DOCUMENT GENERATION TOOL
// ============================================================================

// GenerateRCADocumentTool creates structured RCA documents from analysis findings
type GenerateRCADocumentTool struct {
	*BaseTool
}

// NewGenerateRCADocumentTool creates a new GenerateRCADocumentTool
func NewGenerateRCADocumentTool(c *client.Client, l *zap.Logger) *GenerateRCADocumentTool {
	return &GenerateRCADocumentTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GenerateRCADocumentTool) Name() string { return "generate_rca_document" }

// Annotations returns tool hints for LLMs
func (t *GenerateRCADocumentTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Generate RCA Document")
}

// DefaultTimeout returns the timeout
func (t *GenerateRCADocumentTool) DefaultTimeout() time.Duration {
	return 30 * time.Second
}

// Description returns the tool description
func (t *GenerateRCADocumentTool) Description() string {
	return `Generate a structured Root Cause Analysis (RCA) document template.

**Purpose:**
Creates an industry-standard RCA document pre-filled with findings from log analysis.
The document follows the 5 Whys methodology and includes sections for timeline,
impact assessment, root cause, and corrective actions.

**When to use:**
- After completing incident investigation with analyze_log_delta, analyze_causal_chain
- When documenting an incident for post-mortem review
- To create a shareable incident report for stakeholders

**Document sections:**
1. Executive Summary
2. Incident Timeline
3. Impact Assessment
4. Root Cause Analysis (5 Whys)
5. Log Evidence
6. Corrective Actions
7. Prevention Measures

**Output format:** Markdown document ready for editing and sharing.

**Related tools:** analyze_log_delta, analyze_causal_chain, investigate_incident`
}

// InputSchema returns the input schema
func (t *GenerateRCADocumentTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"incident_title": map[string]interface{}{
				"type":        "string",
				"description": "Short title describing the incident (e.g., 'API Gateway Timeout Spike')",
			},
			"incident_id": map[string]interface{}{
				"type":        "string",
				"description": "Incident tracking ID (e.g., INC-2024-0115, JIRA ticket)",
			},
			"incident_start": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the incident started (ISO 8601)",
			},
			"incident_end": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the incident was resolved (ISO 8601, optional if ongoing)",
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "Incident severity level",
				"enum":        []string{"SEV1", "SEV2", "SEV3", "SEV4"},
				"default":     "SEV2",
			},
			"affected_services": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of affected services/applications",
			},
			"root_cause_category": map[string]interface{}{
				"type":        "string",
				"description": "Primary root cause category from analysis",
				"enum": []string{
					"MEMORY_PRESSURE", "TIMEOUT", "NETWORK_FAILURE", "STORAGE_FAILURE",
					"AUTH_FAILURE", "CODE_BUG", "RATE_LIMITED", "DNS_FAILURE",
					"TLS_FAILURE", "DATABASE_FAILURE", "CPU_PRESSURE", "K8S_ORCHESTRATION",
					"CONFIGURATION_ERROR", "DEPENDENCY_FAILURE", "UNKNOWN",
				},
			},
			"root_cause_description": map[string]interface{}{
				"type":        "string",
				"description": "Detailed description of the root cause",
			},
			"error_patterns": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern":    map[string]interface{}{"type": "string"},
						"count":      map[string]interface{}{"type": "integer"},
						"severity":   map[string]interface{}{"type": "string"},
						"root_cause": map[string]interface{}{"type": "string"},
					},
				},
				"description": "Error patterns discovered during analysis",
			},
			"timeline_events": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"time":         map[string]interface{}{"type": "string"},
						"event":        map[string]interface{}{"type": "string"},
						"source":       map[string]interface{}{"type": "string"},
						"is_key_event": map[string]interface{}{"type": "boolean"},
					},
				},
				"description": "Timeline of events during the incident",
			},
			"include_template_sections": map[string]interface{}{
				"type":        "boolean",
				"description": "Include blank template sections for manual completion (default: true)",
				"default":     true,
			},
		},
		"required": []string{"incident_title", "incident_start"},
	}
}

// Metadata returns semantic metadata for tool discovery
func (t *GenerateRCADocumentTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryAIHelper},
		Keywords:      []string{"rca", "root cause", "document", "report", "postmortem", "incident", "template"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Incident documentation", "Post-mortem creation", "RCA reporting"},
		RelatedTools:  []string{"analyze_log_delta", "analyze_causal_chain", "investigate_incident"},
		ChainPosition: ChainEnd,
	}
}

// Execute generates the RCA document
func (t *GenerateRCADocumentTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse required parameters
	incidentTitle, err := GetStringParam(args, "incident_title", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	incidentStartStr, err := GetStringParam(args, "incident_start", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	incidentStart, err := time.Parse(time.RFC3339, incidentStartStr)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Invalid incident_start format: %v", err)), nil
	}

	// Parse optional parameters
	incidentID, _ := GetStringParam(args, "incident_id", false)
	if incidentID == "" {
		incidentID = fmt.Sprintf("INC-%s", incidentStart.Format("20060102-1504"))
	}

	incidentEndStr, _ := GetStringParam(args, "incident_end", false)
	var incidentEnd *time.Time
	var duration string
	if incidentEndStr != "" {
		if end, err := time.Parse(time.RFC3339, incidentEndStr); err == nil {
			incidentEnd = &end
			duration = end.Sub(incidentStart).String()
		}
	}

	severity, _ := GetStringParam(args, "severity", false)
	if severity == "" {
		severity = "SEV2"
	}

	rootCauseCategory, _ := GetStringParam(args, "root_cause_category", false)
	rootCauseDescription, _ := GetStringParam(args, "root_cause_description", false)

	includeTemplateSections := true
	if val, ok := args["include_template_sections"].(bool); ok {
		includeTemplateSections = val
	}

	// Parse array parameters
	var affectedServices []string
	if services, ok := args["affected_services"].([]interface{}); ok {
		for _, s := range services {
			if str, ok := s.(string); ok {
				affectedServices = append(affectedServices, str)
			}
		}
	}

	var errorPatterns []map[string]interface{}
	if patterns, ok := args["error_patterns"].([]interface{}); ok {
		for _, p := range patterns {
			if pattern, ok := p.(map[string]interface{}); ok {
				errorPatterns = append(errorPatterns, pattern)
			}
		}
	}

	var timelineEvents []map[string]interface{}
	if events, ok := args["timeline_events"].([]interface{}); ok {
		for _, e := range events {
			if event, ok := e.(map[string]interface{}); ok {
				timelineEvents = append(timelineEvents, event)
			}
		}
	}

	// Build the RCA document
	doc := t.generateDocument(
		incidentTitle,
		incidentID,
		incidentStart,
		incidentEnd,
		duration,
		severity,
		affectedServices,
		rootCauseCategory,
		rootCauseDescription,
		errorPatterns,
		timelineEvents,
		includeTemplateSections,
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: doc},
		},
	}, nil
}

// generateDocument creates the RCA document content
func (t *GenerateRCADocumentTool) generateDocument(
	title string,
	incidentID string,
	start time.Time,
	end *time.Time,
	duration string,
	severity string,
	services []string,
	rootCauseCategory string,
	rootCauseDescription string,
	errorPatterns []map[string]interface{},
	timelineEvents []map[string]interface{},
	includeTemplate bool,
) string {
	var doc strings.Builder

	// Header
	doc.WriteString(fmt.Sprintf("# Root Cause Analysis: %s\n\n", title))
	doc.WriteString("---\n\n")

	// Metadata table
	doc.WriteString("## Document Information\n\n")
	doc.WriteString("| Field | Value |\n")
	doc.WriteString("|-------|-------|\n")
	doc.WriteString(fmt.Sprintf("| **Incident ID** | %s |\n", incidentID))
	doc.WriteString(fmt.Sprintf("| **Severity** | %s |\n", severity))
	doc.WriteString(fmt.Sprintf("| **Status** | %s |\n", getIncidentStatus(end)))
	doc.WriteString(fmt.Sprintf("| **Start Time** | %s |\n", start.Format("2006-01-02 15:04:05 MST")))
	if end != nil {
		doc.WriteString(fmt.Sprintf("| **End Time** | %s |\n", end.Format("2006-01-02 15:04:05 MST")))
		doc.WriteString(fmt.Sprintf("| **Duration** | %s |\n", duration))
	} else {
		doc.WriteString("| **End Time** | _Ongoing_ |\n")
		doc.WriteString("| **Duration** | _TBD_ |\n")
	}
	doc.WriteString(fmt.Sprintf("| **Document Created** | %s |\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	doc.WriteString("| **Author** | _[Fill in]_ |\n")
	doc.WriteString("\n")

	// Executive Summary
	doc.WriteString("## 1. Executive Summary\n\n")
	if rootCauseCategory != "" && rootCauseDescription != "" {
		doc.WriteString(fmt.Sprintf("**Root Cause Category:** %s\n\n", rootCauseCategory))
		doc.WriteString(fmt.Sprintf("%s\n\n", rootCauseDescription))
	} else if includeTemplate {
		doc.WriteString("_[Provide a 2-3 sentence summary of what happened, the impact, and how it was resolved]_\n\n")
		doc.WriteString("**Example:**\n")
		doc.WriteString("> On [date], [service] experienced [issue] affecting [X users/requests]. ")
		doc.WriteString("The root cause was identified as [cause]. The incident was resolved by [action] ")
		doc.WriteString("and service was fully restored at [time].\n\n")
	}

	// Affected Services
	doc.WriteString("## 2. Affected Services\n\n")
	if len(services) > 0 {
		for _, svc := range services {
			doc.WriteString(fmt.Sprintf("- %s\n", svc))
		}
	} else if includeTemplate {
		doc.WriteString("- _[Service 1]_\n")
		doc.WriteString("- _[Service 2]_\n")
	}
	doc.WriteString("\n")

	// Impact Assessment
	doc.WriteString("## 3. Impact Assessment\n\n")
	if includeTemplate {
		doc.WriteString("| Metric | Value |\n")
		doc.WriteString("|--------|-------|\n")
		doc.WriteString("| **Users Affected** | _[Number or percentage]_ |\n")
		doc.WriteString("| **Requests Failed** | _[Number or percentage]_ |\n")
		doc.WriteString("| **Revenue Impact** | _[If applicable]_ |\n")
		doc.WriteString("| **SLA Breach** | _[Yes/No, which SLA]_ |\n")
		doc.WriteString("| **Data Loss** | _[Yes/No, describe if yes]_ |\n")
		doc.WriteString("\n")
	}

	// Timeline
	doc.WriteString("## 4. Incident Timeline\n\n")
	if len(timelineEvents) > 0 {
		doc.WriteString("| Time | Event | Source |\n")
		doc.WriteString("|------|-------|--------|\n")
		for _, event := range timelineEvents {
			timeStr, _ := event["time"].(string)
			eventStr, _ := event["event"].(string)
			source, _ := event["source"].(string)
			isKey, _ := event["is_key_event"].(bool)

			if isKey {
				doc.WriteString(fmt.Sprintf("| **%s** | **%s** | %s |\n", timeStr, eventStr, source))
			} else {
				doc.WriteString(fmt.Sprintf("| %s | %s | %s |\n", timeStr, eventStr, source))
			}
		}
	} else if includeTemplate {
		doc.WriteString("| Time | Event | Source |\n")
		doc.WriteString("|------|-------|--------|\n")
		doc.WriteString(fmt.Sprintf("| %s | **Incident began** - first errors observed | Logs |\n", start.Format("15:04:05")))
		doc.WriteString("| _[HH:MM:SS]_ | Alert triggered | Monitoring |\n")
		doc.WriteString("| _[HH:MM:SS]_ | On-call engineer paged | PagerDuty |\n")
		doc.WriteString("| _[HH:MM:SS]_ | Root cause identified | Investigation |\n")
		doc.WriteString("| _[HH:MM:SS]_ | Mitigation applied | Manual |\n")
		doc.WriteString("| _[HH:MM:SS]_ | Service fully restored | Verification |\n")
	}
	doc.WriteString("\n")

	// Root Cause Analysis - 5 Whys
	doc.WriteString("## 5. Root Cause Analysis (5 Whys)\n\n")
	if rootCauseCategory != "" {
		doc.WriteString(fmt.Sprintf("**Primary Root Cause Category:** `%s`\n\n", rootCauseCategory))
	}

	if includeTemplate {
		doc.WriteString("### 5 Whys Analysis\n\n")
		doc.WriteString("1. **Why did the incident occur?**\n")
		doc.WriteString("   - _[Direct cause - e.g., \"Service returned 500 errors\"]_\n\n")
		doc.WriteString("2. **Why did that happen?**\n")
		doc.WriteString("   - _[e.g., \"Database connection pool exhausted\"]_\n\n")
		doc.WriteString("3. **Why did that happen?**\n")
		doc.WriteString("   - _[e.g., \"Slow query holding connections\"]_\n\n")
		doc.WriteString("4. **Why did that happen?**\n")
		doc.WriteString("   - _[e.g., \"Missing index on frequently queried column\"]_\n\n")
		doc.WriteString("5. **Why did that happen?**\n")
		doc.WriteString("   - _[Root cause - e.g., \"Schema migration didn't include index creation\"]_\n\n")
	}

	// Log Evidence
	doc.WriteString("## 6. Log Evidence\n\n")
	if len(errorPatterns) > 0 {
		doc.WriteString("### Error Patterns Identified\n\n")
		doc.WriteString("| Pattern | Count | Severity | Root Cause |\n")
		doc.WriteString("|---------|-------|----------|------------|\n")
		for _, p := range errorPatterns {
			pattern, _ := p["pattern"].(string)
			count, _ := p["count"].(float64)
			sev, _ := p["severity"].(string)
			rc, _ := p["root_cause"].(string)

			// Truncate pattern for table
			if len(pattern) > 60 {
				pattern = pattern[:57] + "..."
			}
			doc.WriteString(fmt.Sprintf("| `%s` | %.0f | %s | %s |\n", pattern, count, sev, rc))
		}
		doc.WriteString("\n")
	}

	doc.WriteString("### Key Log Entries\n\n")
	if includeTemplate {
		doc.WriteString("```\n")
		doc.WriteString("[Paste relevant log entries here]\n")
		doc.WriteString("```\n\n")
		doc.WriteString("**Query used:**\n")
		doc.WriteString("```dataprime\n")
		doc.WriteString("source logs\n")
		doc.WriteString("| filter $m.severity >= ERROR\n")
		doc.WriteString("| filter $m.timestamp >= now() - 1h\n")
		doc.WriteString("| limit 100\n")
		doc.WriteString("```\n\n")
	}

	// Contributing Factors
	doc.WriteString("## 7. Contributing Factors\n\n")
	if includeTemplate {
		doc.WriteString("_[List factors that contributed to the incident or its duration]_\n\n")
		doc.WriteString("- [ ] Recent deployment/change\n")
		doc.WriteString("- [ ] Missing or inadequate monitoring\n")
		doc.WriteString("- [ ] Missing or inadequate alerting\n")
		doc.WriteString("- [ ] Insufficient capacity/scaling\n")
		doc.WriteString("- [ ] Configuration drift\n")
		doc.WriteString("- [ ] Documentation gaps\n")
		doc.WriteString("- [ ] Process/runbook gaps\n")
		doc.WriteString("- [ ] Third-party/dependency issue\n")
		doc.WriteString("\n")
	}

	// Corrective Actions
	doc.WriteString("## 8. Corrective Actions\n\n")
	if includeTemplate {
		doc.WriteString("### Immediate Actions (Completed)\n\n")
		doc.WriteString("| Action | Owner | Status | Date |\n")
		doc.WriteString("|--------|-------|--------|------|\n")
		doc.WriteString("| _[Action taken to resolve incident]_ | _[Name]_ | Done | _[Date]_ |\n")
		doc.WriteString("\n")

		doc.WriteString("### Short-term Actions (1-2 weeks)\n\n")
		doc.WriteString("| Action | Owner | Priority | Due Date |\n")
		doc.WriteString("|--------|-------|----------|----------|\n")
		doc.WriteString("| _[Preventive action 1]_ | _[Name]_ | High | _[Date]_ |\n")
		doc.WriteString("| _[Preventive action 2]_ | _[Name]_ | Medium | _[Date]_ |\n")
		doc.WriteString("\n")

		doc.WriteString("### Long-term Actions (1-3 months)\n\n")
		doc.WriteString("| Action | Owner | Priority | Due Date |\n")
		doc.WriteString("|--------|-------|----------|----------|\n")
		doc.WriteString("| _[Systemic improvement 1]_ | _[Name]_ | Medium | _[Date]_ |\n")
		doc.WriteString("\n")
	}

	// Prevention Measures
	doc.WriteString("## 9. Prevention Measures\n\n")
	if includeTemplate {
		doc.WriteString("### Monitoring Improvements\n\n")
		doc.WriteString("- [ ] Add alert for _[specific condition]_\n")
		doc.WriteString("- [ ] Create dashboard for _[metric/service]_\n")
		doc.WriteString("- [ ] Implement SLO tracking for _[service]_\n")
		doc.WriteString("\n")

		doc.WriteString("### Process Improvements\n\n")
		doc.WriteString("- [ ] Update runbook for _[scenario]_\n")
		doc.WriteString("- [ ] Add pre-deployment check for _[condition]_\n")
		doc.WriteString("- [ ] Improve on-call escalation for _[situation]_\n")
		doc.WriteString("\n")

		doc.WriteString("### Technical Improvements\n\n")
		doc.WriteString("- [ ] Implement _[safeguard/circuit breaker/etc.]_\n")
		doc.WriteString("- [ ] Add automated testing for _[scenario]_\n")
		doc.WriteString("- [ ] Improve error handling in _[component]_\n")
		doc.WriteString("\n")
	}

	// Lessons Learned
	doc.WriteString("## 10. Lessons Learned\n\n")
	if includeTemplate {
		doc.WriteString("### What went well?\n\n")
		doc.WriteString("- _[e.g., \"Quick detection - alert fired within 2 minutes\"]_\n")
		doc.WriteString("- _[e.g., \"Effective communication during incident\"]_\n")
		doc.WriteString("\n")

		doc.WriteString("### What could be improved?\n\n")
		doc.WriteString("- _[e.g., \"Runbook was outdated\"]_\n")
		doc.WriteString("- _[e.g., \"Took too long to identify root cause\"]_\n")
		doc.WriteString("\n")

		doc.WriteString("### Action items from lessons learned\n\n")
		doc.WriteString("- _[Specific action to address improvement area]_\n")
		doc.WriteString("\n")
	}

	// Appendix
	doc.WriteString("## Appendix\n\n")
	doc.WriteString("### A. Related Links\n\n")
	if includeTemplate {
		doc.WriteString("- Alert link: _[URL]_\n")
		doc.WriteString("- Dashboard link: _[URL]_\n")
		doc.WriteString("- Related incidents: _[INC-xxx]_\n")
		doc.WriteString("- Slack thread: _[URL]_\n")
		doc.WriteString("\n")
	}

	doc.WriteString("### B. Attendees (Post-mortem Review)\n\n")
	if includeTemplate {
		doc.WriteString("- _[Name, Role]_\n")
		doc.WriteString("- _[Name, Role]_\n")
		doc.WriteString("\n")
	}

	// Footer
	doc.WriteString("---\n\n")
	doc.WriteString("_This RCA document was generated using IBM Cloud Logs MCP Server._\n")
	doc.WriteString("_Review and complete all sections marked with [Fill in] before sharing._\n")

	return doc.String()
}

// getIncidentStatus returns the incident status based on end time
func getIncidentStatus(end *time.Time) string {
	if end == nil {
		return "Ongoing"
	}
	return "Resolved"
}
