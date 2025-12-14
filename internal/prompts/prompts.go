// Package prompts provides pre-built prompts for common IBM Cloud Logs operations.
//
// Terminology Note: "IBM Cloud Logs", "ICL", and "Cloud Logs" all refer to the same service.
// Users may use any of these terms interchangeably when referring to the logging service.
package prompts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// PromptDefinition represents a prompt with its metadata and handler
type PromptDefinition struct {
	// Prompt is the MCP prompt metadata
	Prompt *mcp.Prompt
	// Handler is the function that generates the prompt content
	Handler mcp.PromptHandler
}

// SessionContextProvider is an interface for accessing session context
// This allows the prompts package to access session data without circular imports
type SessionContextProvider interface {
	GetLastQuery() string
	GetAllFilters() map[string]string
	GetRecentTools(limit int) []RecentToolInfo
	GetInvestigation() *InvestigationInfo
	GetPreferences() *UserPreferencesInfo
	GetSuggestedNextTools() []string
}

// RecentToolInfo contains information about a recently used tool
type RecentToolInfo struct {
	Tool      string
	Timestamp time.Time
	Success   bool
}

// InvestigationInfo contains information about an active investigation
type InvestigationInfo struct {
	ID            string
	StartTime     time.Time
	Application   string
	TimeRange     string
	Hypothesis    string
	FindingsCount int
	ToolsUsed     []string
}

// UserPreferencesInfo contains learned user preferences
type UserPreferencesInfo struct {
	PreferredTimeRange   string
	PreferredSeverity    int
	FrequentApplications []string
	PreferredLimit       int
}

// Registry holds all registered prompts
type Registry struct {
	logger          *zap.Logger
	prompts         []*PromptDefinition
	contextProvider SessionContextProvider
}

// NewRegistry creates a new prompt registry with all available prompts
func NewRegistry(logger *zap.Logger) *Registry {
	r := &Registry{
		logger: logger,
	}
	r.registerPrompts()
	return r
}

// SetContextProvider sets the session context provider for context-aware prompts
func (r *Registry) SetContextProvider(provider SessionContextProvider) {
	r.contextProvider = provider
}

// GetPrompts returns all registered prompt definitions
func (r *Registry) GetPrompts() []*PromptDefinition {
	return r.prompts
}

// registerPrompts registers all available prompts
func (r *Registry) registerPrompts() {
	r.prompts = []*PromptDefinition{
		r.investigateErrorsPrompt(),
		r.setupMonitoringPrompt(),
		r.compareEnvironmentsPrompt(),
		r.debuggingWorkflowPrompt(),
		r.optimizeRetentionPrompt(),
		r.testLogIngestionPrompt(),
		r.createDashboardWorkflowPrompt(),
		r.continueInvestigationPrompt(),
		r.dataprimeTutorialPrompt(),
		r.quickStartPrompt(),
		r.securityAuditPrompt(),
		r.contextAwarePrompt(),
		r.smartSuggestPrompt(),
	}
}

// Helper to create a prompt result with user role
func createPromptResult(description, content string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: content,
				},
			},
		},
	}
}

// getStringArg safely extracts a string argument with a default value
func getStringArg(args map[string]string, key, defaultVal string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

// investigateErrorsPrompt creates the "investigate_errors" prompt definition
func (r *Registry) investigateErrorsPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "investigate_errors",
			Title:       "Investigate Error Spikes",
			Description: "Guide through investigating recent error spikes in IBM Cloud Logs",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "time_range",
					Description: "Time range to investigate (e.g., '1h', '24h', '7d')",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			timeRange := getStringArg(req.Params.Arguments, "time_range", "1h")

			content := fmt.Sprintf(`Let's investigate recent error spikes in your IBM Cloud Logs. I'll help you:

1. **Query recent errors** (last %s)
2. **List active alerts** that may have been triggered
3. **Check alert definitions** to understand thresholds
4. **Review policies** that may affect log routing

To get started, please use these tools in sequence:

1. First, run: query_logs with query "level:error" and time_range "%s"
2. Then, run: list_alerts to see if any alerts were triggered
3. For any alerts found, run: get_alert_definition with the alert_definition_id
4. Check: list_policies to understand log routing and retention

I'll help you correlate the errors with alerts and policies to identify the root cause.`, timeRange, timeRange)

			return createPromptResult("Investigate error spikes workflow", content), nil
		},
	}
}

// setupMonitoringPrompt creates the "setup_monitoring" prompt definition
func (r *Registry) setupMonitoringPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "setup_monitoring",
			Title:       "Setup Monitoring",
			Description: "Guide through setting up comprehensive monitoring for a service",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "service_name",
					Description: "Name of the service to monitor",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			serviceName := getStringArg(req.Params.Arguments, "service_name", "your-service")

			content := fmt.Sprintf(`I'll help you set up comprehensive monitoring for %s. Here's what we'll create:

**Step 1: Create Alert Definition**
First, we'll create an alert that triggers when error rate exceeds threshold:
- Use: create_alert_def
- Parameters:
  - name: "%s High Error Rate"
  - condition: error rate threshold
  - severity: "high"

**Step 2: Create Outgoing Webhook**
Set up a webhook to send notifications:
- Use: create_outgoing_webhook
- Parameters:
  - name: "%s Alerts"
  - url: your notification endpoint
  - type: "Slack" or "PagerDuty"

**Step 3: Create Alert**
Link the alert definition to the webhook:
- Use: create_alert
- Parameters:
  - alert_definition_id: from step 1
  - webhook_id: from step 2

**Step 4: Create Policy** (optional)
Set up log retention and routing:
- Use: create_policy
- Parameters:
  - name: "%s Logs"
  - priority: "high"
  - application_name: "%s"

Would you like to proceed with these steps? I'll guide you through each one.`, serviceName, serviceName, serviceName, serviceName, serviceName)

			return createPromptResult("Setup monitoring workflow", content), nil
		},
	}
}

// compareEnvironmentsPrompt creates the "compare_environments" prompt definition
func (r *Registry) compareEnvironmentsPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "compare_environments",
			Title:       "Compare Environments",
			Description: "Compare logs across production and staging environments",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "time_range",
					Description: "Time range to compare (e.g., '1h', '24h')",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			timeRange := getStringArg(req.Params.Arguments, "time_range", "1h")

			content := fmt.Sprintf(`I'll help you compare logs across production and staging environments. Here's the process:

**Step 1: Query Production Logs**
- Use: query_logs
- Parameters:
  - query: "application:prod AND level:error"
  - time_range: "%s"

**Step 2: Query Staging Logs**
- Use: query_logs
- Parameters:
  - query: "application:staging AND level:error"
  - time_range: "%s"

**Step 3: Compare Alert Configurations**
- Use: list_alerts for each environment
- Compare active alerts and their thresholds

**Step 4: Analyze Differences**
I'll help you:
- Identify error patterns unique to each environment
- Compare error rates and severity distribution
- Highlight configuration differences in alerts and policies

This comparison will help identify environment-specific issues and configuration drift.

Ready to start? Let's begin with querying production logs.`, timeRange, timeRange)

			return createPromptResult("Compare environments workflow", content), nil
		},
	}
}

// debuggingWorkflowPrompt creates the "debugging_workflow" prompt definition
func (r *Registry) debuggingWorkflowPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "debugging_workflow",
			Title:       "Debugging Workflow",
			Description: "Systematic debugging workflow for investigating issues",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "error_message",
					Description: "Error message or pattern to search for",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			errorMessage := getStringArg(req.Params.Arguments, "error_message", "your error message")

			content := fmt.Sprintf(`Let's debug this issue systematically. I'll guide you through a structured debugging workflow:

**Step 1: Search for the Error**
- Use: query_logs
- Query: "%s"
- Start with recent logs (last 1h), expand if needed

**Step 2: Analyze Context**
For each matching log entry, examine:
- Timestamp patterns (is it recurring?)
- Associated services/components
- Request traces (if available)

**Step 3: Check Related Resources**
- Use: list_enrichments to see if any data enrichment might be affecting logs
- Use: list_policies to verify log routing is correct
- Use: list_data_access_rules to ensure proper access controls

**Step 4: Correlation Analysis**
- Look for alerts triggered around the same time: list_alerts
- Check if Events-to-Metrics (E2M) captured this: list_e2m
- Review views that might filter this data: list_views

**Step 5: Root Cause Identification**
Based on the findings, I'll help you:
- Identify the root cause
- Suggest fixes or configuration changes
- Set up alerts to prevent recurrence

Let's start with searching for the error in recent logs.`, errorMessage)

			return createPromptResult("Debugging workflow", content), nil
		},
	}
}

// optimizeRetentionPrompt creates the "optimize_retention" prompt definition
func (r *Registry) optimizeRetentionPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "optimize_retention",
			Title:       "Optimize Log Retention",
			Description: "Analyze and optimize log retention settings for cost reduction",
			Arguments:   []*mcp.PromptArgument{},
		},
		Handler: func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			content := `I'll help you optimize log retention and reduce costs. Here's the analysis workflow:

**Step 1: Review Current Policies**
- Use: list_policies
- Identify retention settings for each policy
- Note priority levels and application filters

**Step 2: Analyze Events-to-Metrics (E2M)**
- Use: list_e2m
- Review which logs are converted to metrics
- Metrics have different retention/cost characteristics

**Step 3: Check Data Access Rules**
- Use: list_data_access_rules
- Ensure proper segmentation of logs by sensitivity
- High-value logs might need longer retention

**Step 4: Review Enrichments**
- Use: list_enrichments
- Data enrichments add value but also increase volume
- Identify which are essential vs. nice-to-have

**Step 5: Optimization Recommendations**
Based on the analysis, I'll suggest:
- Policies to archive or reduce retention for low-value logs
- E2M conversions for logs that only need aggregated metrics
- Data access rules to properly tier log storage
- Enrichments to disable if not actively used

**Cost Impact Analysis:**
- Calculate current vs. optimized retention costs
- Show savings potential by category
- Provide implementation timeline

Ready to analyze your current configuration?`

			return createPromptResult("Optimize retention workflow", content), nil
		},
	}
}

// testLogIngestionPrompt creates the "test_log_ingestion" prompt definition
func (r *Registry) testLogIngestionPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "test_log_ingestion",
			Title:       "Test Log Ingestion",
			Description: "Test and validate log ingestion into IBM Cloud Logs",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "application_name",
					Description: "Application name to use for test logs",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			applicationName := getStringArg(req.Params.Arguments, "application_name", "test-app")

			content := fmt.Sprintf(`I'll help you test log ingestion into IBM Cloud Logs. Here's the workflow:

**Step 1: Ingest Test Logs**
- Use: ingest_logs (or push/add logs)
- Parameters:
  - logs: array of log entries
    - applicationName: "%s"
    - subsystemName: "test"
    - severity: 1-6 (1=Debug, 2=Verbose, 3=Info, 4=Warning, 5=Error, 6=Critical)
    - text: "your log message"
    - timestamp: (optional, auto-generated if not provided)
    - json: (optional, structured metadata)

Example:
{
  "applicationName": "%s",
  "subsystemName": "api",
  "severity": 3,
  "text": "Test log message",
  "json": {
    "user_id": "12345",
    "endpoint": "/api/users"
  }
}

**Step 2: Verify Ingestion**
Wait a few seconds for indexing, then query:
- Use: query_logs
- Query: "application:%s"
- Time range: "5m"

**Step 3: Check Log Details**
Verify the ingested logs contain:
- Correct application and subsystem names
- Proper severity levels
- Structured JSON data (if provided)
- Accurate timestamps

**Ingestion Best Practices:**
- Batch multiple logs in a single request for efficiency
- Use structured JSON for searchable metadata
- Set appropriate severity levels for filtering
- Include timestamps for historical data import

**Note:** The ingestion endpoint uses a different subdomain (.ingress.)
than the management API (.api.), but this is handled automatically.

Ready to start ingesting logs?`, applicationName, applicationName, applicationName)

			return createPromptResult("Test log ingestion workflow", content), nil
		},
	}
}

// continueInvestigationPrompt creates a session-aware prompt for continuing investigations
func (r *Registry) continueInvestigationPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "continue_investigation",
			Title:       "Continue Investigation",
			Description: "Resume an ongoing investigation using session context and previous findings",
			Arguments:   []*mcp.PromptArgument{},
		},
		Handler: func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			content := `Continue the current investigation using session context.

**This prompt leverages your session state to:**
1. Resume from where you left off
2. Build on previous findings
3. Apply persistent filters automatically
4. Track progress toward resolution

**To use this effectively:**

1. **Check current session state:**
   - Use: session_context with action "show"
   - This reveals active filters, recent tools, and investigation status

2. **If an investigation is active:**
   - Review recorded findings with session_context
   - The hypothesis and evidence collected so far will be shown
   - Continue querying with filters already applied

3. **If no investigation is active:**
   - Start one with: session_context action "start_investigation"
   - Or use: investigate_incident to begin structured analysis

4. **As you discover issues:**
   - Record findings: session_context action "add_finding"
   - Update hypothesis: session_context action "set_hypothesis"
   - Set persistent filters: session_context action "set_filter"

5. **When complete:**
   - End investigation: session_context action "end_investigation"
   - This generates a summary of all findings and tools used

**Session-Aware Benefits:**
- Filters persist across tool calls
- Previous queries inform suggestions
- Investigation findings are tracked
- Tool chains are suggested based on context

Ready to continue? Start by checking your session state with session_context.`

			return createPromptResult("Continue investigation with session context", content), nil
		},
	}
}

// createDashboardWorkflowPrompt creates the "create_dashboard_workflow" prompt definition
func (r *Registry) createDashboardWorkflowPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "create_dashboard_workflow",
			Title:       "Create Dashboard",
			Description: "Guide through creating a dashboard in IBM Cloud Logs",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "dashboard_name",
					Description: "Name for the new dashboard",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			dashboardName := getStringArg(req.Params.Arguments, "dashboard_name", "Custom Dashboard")

			content := fmt.Sprintf(`I'll help you create a dashboard in IBM Cloud Logs. Here's the complete workflow:

**Step 1: Design Dashboard Layout**
A dashboard consists of:
- **Sections**: Logical groupings of widgets
- **Rows**: Horizontal containers within sections (each has a height)
- **Widgets**: Visualizations like line charts, bar charts, data tables, etc.
- **Queries**: DataPrime or Lucene queries that power each widget

**Step 2: Choose Widget Types**
Available widget types:
- **line_chart**: Time-series line charts (great for trends)
- **bar_chart**: Bar charts for categorical data
- **pie_chart**: Pie charts for proportions
- **data_table**: Tabular data views
- **gauge**: Single metric gauges
- **horizontal_bar_chart**: Horizontal bar charts
- **markdown**: Text and documentation widgets

**Step 3: Create the Dashboard**
Use: create_dashboard
- name: "%s"
- description: "Description of dashboard purpose"
- layout: (see structure below)

**Dashboard Structure Example:**
{
  "sections": [{
    "id": {"value": "section-1"},
    "rows": [{
      "id": {"value": "row-1"},
      "appearance": {"height": 19},
      "widgets": [{
        "id": {"value": "widget-1"},
        "title": "Error Count",
        "definition": {
          "line_chart": {
            "query_definitions": [{
              "query": {
                "logs": {
                  "aggregations": [{"count": {}}],
                  "group_bys": [{"keypath": ["severity"], "scope": "metadata"}]
                }
              }
            }]
          }
        }
      }]
    }]
  }]
}

**Step 4: Organize Dashboard**
After creation, you can:
- Pin it: use pin_dashboard
- Move to folder: use move_dashboard_to_folder
- Set as default: use set_default_dashboard

**Step 5: Verify Dashboard**
- Use: list_dashboards to confirm creation
- Use: get_dashboard to view full details

**Best Practices:**
- Use meaningful widget titles and descriptions
- Group related widgets in the same section
- Set appropriate row heights (typical: 12-24)
- Use color schemes consistently (cold, warm, classic)

Ready to create your dashboard? Let me know what metrics you want to visualize!`, dashboardName)

			return createPromptResult("Create dashboard workflow", content), nil
		},
	}
}

// dataprimeTutorialPrompt creates an interactive DataPrime learning prompt
func (r *Registry) dataprimeTutorialPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "dataprime_tutorial",
			Title:       "Learn DataPrime",
			Description: "Interactive tutorial for learning DataPrime query syntax",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "skill_level",
					Description: "Your experience level: beginner, intermediate, advanced",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			skillLevel := getStringArg(req.Params.Arguments, "skill_level", "beginner")

			var content string
			switch skillLevel {
			case "intermediate":
				content = `# DataPrime Intermediate Tutorial

You're ready to learn more advanced DataPrime concepts!

**Aggregations and Grouping:**
` + "```" + `
source logs
| filter $d.severity >= 4
| groupby $l.applicationName
| count
| sort -_count
| limit 10
` + "```" + `

**Time-Based Analysis:**
` + "```" + `
source logs
| filter $d.status_code >= 500
| groupby $m.timestamp.bucket(15m)
| count
` + "```" + `

**String Operations:**
` + "```" + `
source logs
| filter $d.message.contains('timeout')
| extract $d.message into (duration using /took (\d+)ms/)
| filter duration > 1000
` + "```" + `

**Try these exercises:**
1. Count errors per application per hour
2. Find requests with response time > 5s
3. Extract and analyze error codes from messages

Use **build_query** to help construct queries!
Use **explain_query** to understand complex queries!
Use **validate_query** to check syntax before running!`

			case "advanced":
				content = `# DataPrime Advanced Tutorial

Master complex DataPrime patterns!

**Subqueries and Joins:**
` + "```" + `
source logs
| filter $d.trace_id in (
    source logs
    | filter $d.severity == 6
    | select $d.trace_id
  )
| sort $m.timestamp
` + "```" + `

**Window Functions:**
` + "```" + `
source logs
| groupby $l.applicationName, $m.timestamp.bucket(1h)
| count
| window rolling(3) as moving_avg
` + "```" + `

**Complex Extractions:**
` + "```" + `
source logs
| extract $d.message into (
    method using /\"(\w+)\s+\/api/,
    endpoint using /\"[A-Z]+\s+(\/[^\s]+)/,
    status using /HTTP\/\d\.\d\"\s+(\d+)/
  )
| groupby method, endpoint
| count
| sort -_count
` + "```" + `

**Performance Optimization:**
- Use specific time ranges
- Filter early in the pipeline
- Limit results for exploration
- Use aggregations instead of raw logs when possible

**Advanced exercises:**
1. Correlate errors with deployment events
2. Calculate p95 latency per endpoint
3. Build anomaly detection queries`

			default: // beginner
				content = `# DataPrime Beginner Tutorial

Welcome to DataPrime! Let's learn the basics.

**Basic Query Structure:**
` + "```" + `
source logs | filter <condition> | select <fields>
` + "```" + `

**Field References:**
- ` + "`$d.field`" + ` - Data fields (from log payload)
- ` + "`$l.field`" + ` - Labels (applicationName, subsystemName)
- ` + "`$m.field`" + ` - Metadata (timestamp, severity)

**Example 1: Filter by severity**
` + "```" + `
source logs | filter $d.severity == 'error'
` + "```" + `

**Example 2: Search in messages**
` + "```" + `
source logs | filter $d.message.contains('timeout')
` + "```" + `

**Example 3: Filter by application**
` + "```" + `
source logs | filter $l.applicationName == 'api-gateway'
` + "```" + `

**Example 4: Combine filters**
` + "```" + `
source logs
| filter $d.severity >= 4
| filter $l.applicationName == 'api-gateway'
| limit 100
` + "```" + `

**Try these tools to help you learn:**
- **build_query**: Describe what you want in plain English
- **explain_query**: Understand what a query does
- **validate_query**: Check if your query is correct
- **query_logs**: Run your query

Ready to try? Start with: query_logs with query "source logs | limit 10"`
			}

			return createPromptResult("DataPrime tutorial for "+skillLevel, content), nil
		},
	}
}

// quickStartPrompt provides a quick start guide for new users
func (r *Registry) quickStartPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "quick_start",
			Title:       "Quick Start Guide",
			Description: "Get started quickly with IBM Cloud Logs - essential commands and workflows",
			Arguments:   []*mcp.PromptArgument{},
		},
		Handler: func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			content := `# IBM Cloud Logs Quick Start Guide

**Not sure where to start? Here are the most common tasks:**

## 1. Check System Health
` + "```" + `
health_check
` + "```" + `
Quick overview of error rates and system status.

## 2. Search Logs
` + "```" + `
query_logs with query "severity:error" and time_range "1h"
` + "```" + `
Find specific log entries.

## 3. Investigate Issues
` + "```" + `
investigate_incident with application "your-app" and severity "error"
` + "```" + `
Automated analysis of error patterns.

## 4. View Alerts
` + "```" + `
list_alerts
` + "```" + `
See all configured alerting rules.

## 5. View Dashboards
` + "```" + `
list_dashboards
` + "```" + `
Browse available visualizations.

---

## Need Help Finding Tools?
` + "```" + `
discover_tools with intent "what you want to do"
` + "```" + `

**Example intents:**
- "investigate errors in production"
- "set up alerting for my service"
- "learn how to write queries"
- "reduce logging costs"

---

## Common Workflows

| Task | Start With |
|------|------------|
| Debug an issue | ` + "`investigate_incident`" + ` |
| Set up monitoring | ` + "`list_alerts`" + ` then ` + "`create_alert`" + ` |
| Create visualizations | ` + "`list_dashboards`" + ` then ` + "`create_dashboard`" + ` |
| Optimize costs | ` + "`list_policies`" + ` |
| Learn queries | ` + "`build_query`" + ` or ` + "`query_templates`" + ` |

---

**Pro Tips:**
- Use ` + "`session_context`" + ` to track your investigation progress
- Filters you set persist across tool calls
- Tool suggestions appear based on your recent activity

Ready to explore? Try ` + "`health_check`" + ` to see your current system status!`

			return createPromptResult("Quick start guide for IBM Cloud Logs", content), nil
		},
	}
}

// securityAuditPrompt guides through a security audit workflow
func (r *Registry) securityAuditPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "security_audit",
			Title:       "Security Audit",
			Description: "Guide through auditing security configurations and access patterns",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "focus_area",
					Description: "Area to focus on: access, authentication, data, all",
					Required:    false,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			focusArea := getStringArg(req.Params.Arguments, "focus_area", "all")

			content := fmt.Sprintf(`# Security Audit Workflow

**Focus Area: %s**

## Step 1: Review Data Access Rules
`+"```"+`
list_data_access_rules
`+"```"+`
Check who has access to what data:
- Review rule scopes and filters
- Identify overly permissive rules
- Ensure principle of least privilege

## Step 2: Audit Authentication Logs
`+"```"+`
query_logs with query "source logs | filter $d.event_type.contains('auth') | limit 100"
`+"```"+`
Look for:
- Failed authentication attempts
- Unusual login patterns
- Service account usage

## Step 3: Check Alert Configurations
`+"```"+`
list_alerts
`+"```"+`
Verify security alerts exist for:
- Authentication failures
- Privilege escalation
- Data access anomalies
- Suspicious patterns

## Step 4: Review Policies
`+"```"+`
list_policies
`+"```"+`
Ensure:
- Sensitive logs have appropriate retention
- Compliance requirements are met
- Logs aren't being dropped inappropriately

## Step 5: Examine Outgoing Webhooks
`+"```"+`
list_outgoing_webhooks
`+"```"+`
Verify:
- Webhook destinations are authorized
- No unexpected external endpoints
- HTTPS is used for all webhooks

## Step 6: Analyze Anomalies
`+"```"+`
query_logs with query "source logs | filter $d.severity >= 5 | filter $d.message.contains('unauthorized') OR $d.message.contains('forbidden') | limit 50"
`+"```"+`

## Security Checklist
- [ ] Data access rules follow least privilege
- [ ] Authentication failures are alerted
- [ ] Sensitive data has appropriate retention
- [ ] No unauthorized webhook destinations
- [ ] Security events are being logged
- [ ] Anomaly detection alerts are configured

## Recommended Actions
After the audit, consider:
1. **Create security alerts** using `+"`suggest_alert`"+` with security focus
2. **Document findings** in a security report
3. **Set up dashboards** for security monitoring
4. **Schedule regular audits** using this workflow

Ready to start? Begin with `+"`list_data_access_rules`"+` to review access controls.`, focusArea)

			return createPromptResult("Security audit workflow", content), nil
		},
	}
}

// contextAwarePrompt creates a dynamic prompt that adapts to session context
func (r *Registry) contextAwarePrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "context_aware_assist",
			Title:       "Context-Aware Assistant",
			Description: "Get personalized guidance based on your current session, active filters, recent activity, and learned preferences",
			Arguments:   []*mcp.PromptArgument{},
		},
		Handler: func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			var builder strings.Builder
			builder.WriteString("# Context-Aware Assistance\n\n")

			if r.contextProvider == nil {
				r.writeNoContextGuidance(&builder)
				return createPromptResult("Context-aware assistance", builder.String()), nil
			}

			builder.WriteString("Based on your current session, here's personalized guidance:\n\n")

			r.writeInvestigationSection(&builder)
			filters := r.writeFiltersSection(&builder)
			r.writeLastQuerySection(&builder)
			r.writePreferencesSection(&builder)
			recentTools := r.writeRecentActivitySection(&builder)

			if len(filters) == 0 && r.contextProvider.GetLastQuery() == "" && len(recentTools) == 0 {
				r.writeGettingStartedSection(&builder)
			}

			return createPromptResult("Context-aware assistance based on your session", builder.String()), nil
		},
	}
}

func (r *Registry) writeNoContextGuidance(builder *strings.Builder) {
	builder.WriteString("*Session context not available. Here's general guidance:*\n\n")
	builder.WriteString("## Getting Started\n")
	builder.WriteString("- Use `health_check` to see your system status\n")
	builder.WriteString("- Use `discover_tools` to find relevant tools for your task\n")
	builder.WriteString("- Use `query_logs` to search your logs\n")
}

func (r *Registry) writeInvestigationSection(builder *strings.Builder) {
	inv := r.contextProvider.GetInvestigation()
	if inv == nil {
		return
	}

	builder.WriteString("## 🔍 Active Investigation\n\n")
	fmt.Fprintf(builder, "**Investigation ID:** %s\n", inv.ID)
	fmt.Fprintf(builder, "**Started:** %s ago\n", time.Since(inv.StartTime).Round(time.Minute))
	if inv.Application != "" {
		fmt.Fprintf(builder, "**Application:** %s\n", inv.Application)
	}
	if inv.Hypothesis != "" {
		fmt.Fprintf(builder, "**Current Hypothesis:** %s\n", inv.Hypothesis)
	}
	fmt.Fprintf(builder, "**Findings:** %d recorded\n\n", inv.FindingsCount)

	usedTools := make(map[string]bool)
	for _, t := range inv.ToolsUsed {
		usedTools[t] = true
	}

	builder.WriteString("**Suggested Next Steps:**\n")
	investigationTools := []struct{ name, desc string }{
		{"query_logs", "Search for more evidence"},
		{"suggest_alert", "Create alerts based on findings"},
		{"create_dashboard", "Visualize patterns"},
		{"session_context", "Record a finding"},
	}
	for _, tool := range investigationTools {
		if !usedTools[tool.name] {
			fmt.Fprintf(builder, "- `%s` - %s\n", tool.name, tool.desc)
		}
	}
	builder.WriteString("\n")
}

func (r *Registry) writeFiltersSection(builder *strings.Builder) map[string]string {
	filters := r.contextProvider.GetAllFilters()
	if len(filters) == 0 {
		return filters
	}

	builder.WriteString("## 🎯 Active Filters\n\n")
	builder.WriteString("These filters are automatically applied to your queries:\n")
	for key, value := range filters {
		fmt.Fprintf(builder, "- **%s:** `%s`\n", key, value)
	}
	builder.WriteString("\n*Use `session_context` with action `clear_filter` to remove filters.*\n\n")
	return filters
}

func (r *Registry) writeLastQuerySection(builder *strings.Builder) {
	lastQuery := r.contextProvider.GetLastQuery()
	if lastQuery == "" {
		return
	}

	builder.WriteString("## 📝 Last Query\n\n")
	builder.WriteString("```\n")
	builder.WriteString(lastQuery)
	builder.WriteString("\n```\n")
	builder.WriteString("*Use `explain_query` to understand this query or `build_query` to modify it.*\n\n")
}

func (r *Registry) writePreferencesSection(builder *strings.Builder) {
	prefs := r.contextProvider.GetPreferences()
	if prefs == nil {
		return
	}

	hasPrefs := prefs.PreferredTimeRange != "" || len(prefs.FrequentApplications) > 0 || prefs.PreferredLimit > 0
	if !hasPrefs {
		return
	}

	builder.WriteString("## ⚙️ Learned Preferences\n\n")
	builder.WriteString("Based on your usage patterns:\n")
	if prefs.PreferredTimeRange != "" {
		fmt.Fprintf(builder, "- **Default time range:** %s\n", prefs.PreferredTimeRange)
	}
	if len(prefs.FrequentApplications) > 0 {
		fmt.Fprintf(builder, "- **Frequent applications:** %s\n", strings.Join(prefs.FrequentApplications, ", "))
	}
	if prefs.PreferredLimit > 0 {
		fmt.Fprintf(builder, "- **Default result limit:** %d\n", prefs.PreferredLimit)
	}
	builder.WriteString("\n")
}

func (r *Registry) writeRecentActivitySection(builder *strings.Builder) []RecentToolInfo {
	recentTools := r.contextProvider.GetRecentTools(5)
	if len(recentTools) == 0 {
		return recentTools
	}

	builder.WriteString("## 🕐 Recent Activity\n\n")
	builder.WriteString("Your recent tool usage:\n")
	for _, t := range recentTools {
		status := "✅"
		if !t.Success {
			status = "❌"
		}
		fmt.Fprintf(builder, "- %s `%s` (%s ago)\n", status, t.Tool, time.Since(t.Timestamp).Round(time.Second))
	}
	builder.WriteString("\n")

	if suggested := r.contextProvider.GetSuggestedNextTools(); len(suggested) > 0 {
		builder.WriteString("**Based on your patterns, you might want to use:**\n")
		for _, tool := range suggested {
			fmt.Fprintf(builder, "- `%s`\n", tool)
		}
		builder.WriteString("\n")
	}
	return recentTools
}

func (r *Registry) writeGettingStartedSection(builder *strings.Builder) {
	builder.WriteString("## 🚀 Getting Started\n\n")
	builder.WriteString("No session context yet. Here are some ways to begin:\n\n")
	builder.WriteString("1. **Check system health:** `health_check`\n")
	builder.WriteString("2. **Discover tools:** `discover_tools` with your intent\n")
	builder.WriteString("3. **Search logs:** `query_logs` with your search criteria\n")
	builder.WriteString("4. **Start an investigation:** `investigate_incident`\n")
}

// smartSuggestPrompt creates a prompt that provides intelligent suggestions
func (r *Registry) smartSuggestPrompt() *PromptDefinition {
	return &PromptDefinition{
		Prompt: &mcp.Prompt{
			Name:        "smart_suggest",
			Title:       "Smart Suggestions",
			Description: "Get intelligent tool and workflow suggestions based on your goal and current context",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "goal",
					Description: "What you want to accomplish (e.g., 'debug production errors', 'set up monitoring')",
					Required:    true,
				},
			},
		},
		Handler: func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			goal := getStringArg(req.Params.Arguments, "goal", "")
			if goal == "" {
				return createPromptResult("Smart suggestions", "Please provide a goal to get personalized suggestions."), nil
			}

			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("# Smart Suggestions for: %s\n\n", goal))

			goalLower := strings.ToLower(goal)

			// Analyze the goal and provide tailored suggestions
			var primaryTools []string
			var workflow string
			var tips []string

			switch {
			case containsAny(goalLower, "error", "debug", "issue", "problem", "crash", "fail"):
				primaryTools = []string{"investigate_incident", "query_logs", "health_check"}
				workflow = "error_investigation"
				tips = []string{
					"Start with `health_check` for a quick overview",
					"Use `investigate_incident` for guided analysis",
					"Set filters with `session_context` to focus on specific services",
					"Record findings as you discover them for the postmortem",
				}

			case containsAny(goalLower, "alert", "monitor", "notify", "threshold"):
				primaryTools = []string{"suggest_alert", "create_alert", "create_outgoing_webhook"}
				workflow = "monitoring_setup"
				tips = []string{
					"Use `suggest_alert` to get AI-recommended alert configurations",
					"Create webhooks first if you need Slack/PagerDuty notifications",
					"Test with dry_run: true before creating alerts",
					"Consider different severity levels for different conditions",
				}

			case containsAny(goalLower, "dashboard", "visualiz", "chart", "graph"):
				primaryTools = []string{"create_dashboard", "list_dashboards", "query_logs"}
				workflow = "dashboard_creation"
				tips = []string{
					"Query logs first to understand what data is available",
					"Use line_chart for time-series data, bar_chart for categories",
					"Group related widgets in sections for better organization",
					"Start with a simple dashboard and iterate",
				}

			case containsAny(goalLower, "cost", "retention", "optimi", "save", "reduce"):
				primaryTools = []string{"list_policies", "list_e2m", "export_data_usage"}
				workflow = "cost_optimization"
				tips = []string{
					"Review current policies to understand retention settings",
					"Consider E2M to convert high-volume logs to metrics",
					"Archive logs to lower-cost tiers where appropriate",
					"Use priority levels to tier your data",
				}

			case containsAny(goalLower, "learn", "dataprime", "query", "syntax", "how to"):
				primaryTools = []string{"query_templates", "build_query", "explain_query"}
				workflow = "query_learning"
				tips = []string{
					"Use `build_query` to convert natural language to DataPrime",
					"Use `explain_query` to understand existing queries",
					"Start with simple filters and add complexity gradually",
					"Use `validate_query` to check syntax before running",
				}

			case containsAny(goalLower, "security", "audit", "access", "permission"):
				primaryTools = []string{"list_data_access_rules", "query_logs", "list_alerts"}
				workflow = "security_investigation"
				tips = []string{
					"Review data access rules for least privilege",
					"Search for authentication failures and unauthorized access",
					"Ensure security-related alerts are configured",
					"Check webhook destinations for unauthorized endpoints",
				}

			case containsAny(goalLower, "export", "stream", "kafka", "siem"):
				primaryTools = []string{"list_streams", "create_stream", "query_logs"}
				workflow = "log_export_setup"
				tips = []string{
					"Query logs first to verify the data you want to export",
					"Use DPXL expressions to filter what gets streamed",
					"Consider compression settings for large volumes",
					"Test with a small time window first",
				}

			default:
				primaryTools = []string{"discover_tools", "health_check", "query_logs"}
				workflow = ""
				tips = []string{
					"Use `discover_tools` with your intent for best matches",
					"Start with `health_check` to understand system state",
					"Use `session_context` to set up filters for your work",
				}
			}

			// Write recommendations
			builder.WriteString("## Recommended Tools\n\n")
			for i, tool := range primaryTools {
				builder.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, tool))
			}
			builder.WriteString("\n")

			if workflow != "" {
				builder.WriteString(fmt.Sprintf("## Suggested Workflow: %s\n\n", workflow))
				builder.WriteString("Use `discover_tools` with this intent to see the full workflow chain.\n\n")
			}

			builder.WriteString("## Tips\n\n")
			for _, tip := range tips {
				builder.WriteString(fmt.Sprintf("- %s\n", tip))
			}
			builder.WriteString("\n")

			// Add context-aware suggestions if available
			if r.contextProvider != nil {
				if inv := r.contextProvider.GetInvestigation(); inv != nil {
					builder.WriteString("## 💡 Context Note\n\n")
					builder.WriteString(fmt.Sprintf("You have an active investigation (%s). ", inv.ID))
					builder.WriteString("Your new activity will be tracked as part of this investigation.\n\n")
				}

				if filters := r.contextProvider.GetAllFilters(); len(filters) > 0 {
					builder.WriteString("## 🎯 Active Filters Applied\n\n")
					for key, value := range filters {
						builder.WriteString(fmt.Sprintf("- %s: `%s`\n", key, value))
					}
					builder.WriteString("\n")
				}
			}

			builder.WriteString("---\n")
			builder.WriteString(fmt.Sprintf("*Ready to start? Try: `%s`*\n", primaryTools[0]))

			return createPromptResult(fmt.Sprintf("Smart suggestions for: %s", goal), builder.String()), nil
		},
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
