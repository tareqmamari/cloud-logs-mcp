// Package prompts provides pre-built prompts for common IBM Cloud Logs operations.
//
// Terminology Note: "IBM Cloud Logs", "ICL", and "Cloud Logs" all refer to the same service.
// Users may use any of these terms interchangeably when referring to the logging service.
package prompts

import (
	"context"
	"fmt"

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

// Registry holds all registered prompts
type Registry struct {
	logger  *zap.Logger
	prompts []*PromptDefinition
}

// NewRegistry creates a new prompt registry with all available prompts
func NewRegistry(logger *zap.Logger) *Registry {
	r := &Registry{
		logger: logger,
	}
	r.registerPrompts()
	return r
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
