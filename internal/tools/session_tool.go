// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements the session context management tool.
package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// SessionContextTool manages the conversational session context
type SessionContextTool struct {
	*BaseTool
}

// NewSessionContextTool creates a new SessionContextTool
func NewSessionContextTool(c *client.Client, l *zap.Logger) *SessionContextTool {
	return &SessionContextTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *SessionContextTool) Name() string { return "session_context" }

// Description returns the tool description
func (t *SessionContextTool) Description() string {
	return `Manage conversational context that persists across tool calls.

**This tool enables:**
- Setting persistent filters that apply to subsequent queries
- Starting/ending structured investigations
- Recording findings during investigation
- Viewing session state and learned preferences
- Checking token budget and cost status

**Use Cases:**
- "Set filter for application=api-gateway" - all subsequent queries will include this filter
- "Start investigation for payment-service errors" - begins tracking findings
- "Add finding: database connection timeout" - records evidence
- "Show session" - displays current context, filters, and preferences
- "Show budget" - displays token usage, cost, and compression level

**Session State is Automatically Updated:**
- Last query is remembered for context
- Recent tools are tracked
- User preferences are learned from usage patterns
- Token budget is tracked across all tool calls`
}

// InputSchema returns the input schema
func (t *SessionContextTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform on session",
				"enum": []string{
					"show",          // Show current session state
					"show_budget",   // Show token budget status
					"set_filter",    // Set a persistent filter
					"clear_filters", // Clear all filters
					"start_investigation",
					"add_finding",
					"set_hypothesis",
					"end_investigation",
					"clear", // Clear entire session
				},
			},
			"filter_key": map[string]interface{}{
				"type":        "string",
				"description": "Filter key (for set_filter action)",
				"examples":    []string{"application", "subsystem", "severity", "time_range"},
			},
			"filter_value": map[string]interface{}{
				"type":        "string",
				"description": "Filter value (for set_filter action)",
			},
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Application name (for start_investigation)",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range (for start_investigation)",
			},
			"finding": map[string]interface{}{
				"type":        "string",
				"description": "Finding description (for add_finding)",
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "Finding severity (for add_finding)",
				"enum":        []string{"info", "warning", "critical"},
			},
			"hypothesis": map[string]interface{}{
				"type":        "string",
				"description": "Working hypothesis (for set_hypothesis)",
			},
		},
		"required": []string{"action"},
	}
}

// Annotations returns tool annotations
func (t *SessionContextTool) Annotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          "Session Context",
		ReadOnlyHint:   false, // Can modify session state
		IdempotentHint: false,
	}
}

// Execute executes the tool
func (t *SessionContextTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	action, err := GetStringParam(args, "action", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	session := GetSession()

	switch action {
	case "show":
		return t.showSession(session)

	case "show_budget":
		return t.showBudget()

	case "set_filter":
		key, _ := GetStringParam(args, "filter_key", false)
		value, _ := GetStringParam(args, "filter_value", false)
		if key == "" || value == "" {
			return NewToolResultError("Both filter_key and filter_value are required"), nil
		}
		session.SetFilter(key, value)
		return t.formatResult(map[string]interface{}{
			"status":         "filter_set",
			"key":            key,
			"value":          value,
			"message":        "Filter will be applied to subsequent queries automatically",
			"active_filters": session.GetAllFilters(),
		})

	case "clear_filters":
		session.ClearFilters()
		return t.formatResult(map[string]interface{}{
			"status":  "filters_cleared",
			"message": "All filters have been cleared",
		})

	case "start_investigation":
		app, _ := GetStringParam(args, "application", false)
		timeRange, _ := GetStringParam(args, "time_range", false)
		if timeRange == "" {
			timeRange = "1h"
		}
		id := session.StartInvestigation(app, timeRange)
		return t.formatResult(map[string]interface{}{
			"status":           "investigation_started",
			"investigation_id": id,
			"application":      app,
			"time_range":       timeRange,
			"message":          "Investigation started. Use add_finding to record discoveries, set_hypothesis for working theory.",
			"next_steps": []string{
				"Use query_logs to search for relevant logs",
				"Use add_finding to record discoveries",
				"Use set_hypothesis to document your theory",
				"Use end_investigation when complete",
			},
		})

	case "add_finding":
		finding, _ := GetStringParam(args, "finding", false)
		severity, _ := GetStringParam(args, "severity", false)
		if finding == "" {
			return NewToolResultError("finding is required"), nil
		}
		if severity == "" {
			severity = "info"
		}
		session.AddFinding(t.Name(), finding, severity, "")
		inv := session.GetInvestigation()
		return t.formatResult(map[string]interface{}{
			"status":         "finding_added",
			"finding":        finding,
			"severity":       severity,
			"total_findings": len(inv.Findings),
			"investigation":  inv,
		})

	case "set_hypothesis":
		hypothesis, _ := GetStringParam(args, "hypothesis", false)
		if hypothesis == "" {
			return NewToolResultError("hypothesis is required"), nil
		}
		session.SetHypothesis(hypothesis)
		return t.formatResult(map[string]interface{}{
			"status":     "hypothesis_set",
			"hypothesis": hypothesis,
			"message":    "Working hypothesis updated. Continue investigation to gather evidence.",
		})

	case "end_investigation":
		inv := session.EndInvestigation()
		if inv == nil {
			return NewToolResultError("No active investigation to end"), nil
		}
		return t.formatResult(map[string]interface{}{
			"status":           "investigation_ended",
			"investigation_id": inv.ID,
			"duration":         inv.StartTime.String(),
			"total_findings":   len(inv.Findings),
			"findings":         inv.Findings,
			"hypothesis":       inv.Hypothesis,
			"tools_used":       inv.ToolsUsed,
			"recommendations": []string{
				"Consider creating an alert for the identified pattern",
				"Document root cause for future reference",
				"Set up dashboard to monitor this condition",
			},
		})

	case "clear":
		// Reset the current user's session
		session.ClearSession()
		return t.formatResult(map[string]interface{}{
			"status":  "session_cleared",
			"message": "Session has been cleared. All filters, context, and learned preferences reset.",
		})

	default:
		return NewToolResultError("Unknown action: " + action), nil
	}
}

// showBudget returns the current token budget status
func (t *SessionContextTool) showBudget() (*mcp.CallToolResult, error) {
	budget := GetBudgetContext()
	summary := budget.GetSummary()

	// Extract nested maps
	tokens := summary["tokens"].(map[string]interface{})
	cost := summary["cost"].(map[string]interface{})
	execution := summary["execution"].(map[string]interface{})

	// Convert millicents to USD for display
	usedMillicents := cost["used_millicents"].(int)
	maxMillicents := cost["max_millicents"].(int)

	result := map[string]interface{}{
		"budget_status": map[string]interface{}{
			"tokens": map[string]interface{}{
				"used":            tokens["used"],
				"remaining":       tokens["remaining"],
				"max":             tokens["max"],
				"usage_percent":   tokens["usage_pct"],
				"counting_method": tokens["counting_method"],
				"accuracy":        tokens["accuracy"],
			},
			"cost": map[string]interface{}{
				"used_usd":      float64(usedMillicents) / 100000,
				"max_usd":       float64(maxMillicents) / 100000,
				"remaining_pct": cost["remaining_pct"],
			},
			"execution":         execution,
			"compression_level": budget.GetCompressionLevel(),
		},
		"recommendations": t.getBudgetRecommendations(budget),
	}

	return t.formatResult(result)
}

// getBudgetRecommendations provides actionable recommendations based on budget state
func (t *SessionContextTool) getBudgetRecommendations(budget *BudgetContext) []string {
	recommendations := []string{}
	compression := budget.GetCompressionLevel()

	switch compression {
	case BudgetCompressionNone:
		recommendations = append(recommendations, "Budget healthy - full results available")
	case BudgetCompressionLight:
		recommendations = append(recommendations, "Budget at 25-50% - consider using summary_only=true for large queries")
	case BudgetCompressionMedium:
		recommendations = append(recommendations, "Budget at 50-75% - results will be summarized automatically")
		recommendations = append(recommendations, "Use specific filters to reduce result size")
	case BudgetCompressionHeavy:
		recommendations = append(recommendations, "Budget at 75-90% - only essential data returned")
		recommendations = append(recommendations, "Consider ending session or focusing on specific issues")
	case BudgetCompressionMinimal:
		recommendations = append(recommendations, "Budget nearly exhausted (>90%) - minimal responses only")
		recommendations = append(recommendations, "Complete current task or start new session")
	}

	return recommendations
}

// showSession returns the current session state
func (t *SessionContextTool) showSession(session *SessionContext) (*mcp.CallToolResult, error) {
	summary := session.GetSessionSummary()

	// Add recent tools
	recentTools := session.GetRecentTools(5)
	toolNames := make([]string, len(recentTools))
	for i, rt := range recentTools {
		toolNames[i] = rt.Tool
	}
	summary["recent_tools"] = toolNames

	return t.formatResult(summary)
}

// formatResult formats the result as JSON
func (t *SessionContextTool) formatResult(data map[string]interface{}) (*mcp.CallToolResult, error) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return NewToolResultError("Failed to format result: " + err.Error()), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(output),
			},
		},
	}, nil
}

// Metadata returns tool metadata for discovery
func (t *SessionContextTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryMeta, CategoryWorkflow},
		Keywords:      []string{"session", "context", "filter", "investigation", "memory", "state"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Manage session state", "Track investigations", "Set persistent filters"},
		RelatedTools:  []string{"discover_tools", "investigate_incident"},
		ChainPosition: ChainStarter,
	}
}
