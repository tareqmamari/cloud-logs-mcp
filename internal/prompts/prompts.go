package prompts

import (
	"context"
	"fmt"

	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// PromptHandler handles a prompt execution
type PromptHandler struct {
	client *client.Client
	logger *zap.Logger
}

// NewPromptHandler creates a new prompt handler
func NewPromptHandler(client *client.Client, logger *zap.Logger) *PromptHandler {
	return &PromptHandler{
		client: client,
		logger: logger,
	}
}

// InvestigateErrorsPrompt handles the "investigate_errors" prompt
func (h *PromptHandler) InvestigateErrorsPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	timeRange, _ := arguments["time_range"].(string)
	if timeRange == "" {
		timeRange = "1h"
	}

	prompt := fmt.Sprintf(`Let's investigate recent error spikes in your IBM Cloud Logs. I'll help you:

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

	return prompt, nil
}

// SetupMonitoringPrompt handles the "setup_monitoring" prompt
func (h *PromptHandler) SetupMonitoringPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	serviceName, _ := arguments["service_name"].(string)
	if serviceName == "" {
		serviceName = "your-service"
	}

	prompt := fmt.Sprintf(`I'll help you set up comprehensive monitoring for %s. Here's what we'll create:

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

	return prompt, nil
}

// CompareEnvironmentsPrompt handles the "compare_environments" prompt
func (h *PromptHandler) CompareEnvironmentsPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	timeRange, _ := arguments["time_range"].(string)
	if timeRange == "" {
		timeRange = "1h"
	}

	prompt := fmt.Sprintf(`I'll help you compare logs across production and staging environments. Here's the process:

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

	return prompt, nil
}

// DebuggingWorkflowPrompt handles the "debugging_workflow" prompt
func (h *PromptHandler) DebuggingWorkflowPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	errorMessage, _ := arguments["error_message"].(string)
	if errorMessage == "" {
		errorMessage = "your error message"
	}

	prompt := fmt.Sprintf(`Let's debug this issue systematically. I'll guide you through a structured debugging workflow:

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

	return prompt, nil
}

// OptimizeRetentionPrompt handles the "optimize_retention" prompt
func (h *PromptHandler) OptimizeRetentionPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	prompt := `I'll help you optimize log retention and reduce costs. Here's the analysis workflow:

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

	return prompt, nil
}

// TestLogIngestionPrompt handles the "test_log_ingestion" prompt
func (h *PromptHandler) TestLogIngestionPrompt(ctx context.Context, arguments map[string]interface{}) (string, error) {
	applicationName, _ := arguments["application_name"].(string)
	if applicationName == "" {
		applicationName = "test-app"
	}

	prompt := fmt.Sprintf(`I'll help you test log ingestion into IBM Cloud Logs. Here's the workflow:

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

	return prompt, nil
}
