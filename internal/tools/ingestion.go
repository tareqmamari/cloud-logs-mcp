package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// IngestLogsTool implements log ingestion to IBM Cloud Logs.
// It sends log entries to the ingestion endpoint (uses .ingress. subdomain
// instead of the standard .api. subdomain).
//
// The tool supports:
//   - Automatic timestamp generation if not provided
//   - Severity levels from 1 (Debug) to 6 (Critical)
//   - Optional structured JSON data
//   - Batch ingestion of multiple log entries
type IngestLogsTool struct {
	*BaseTool
}

// NewIngestLogsTool creates a new IngestLogsTool instance.
func NewIngestLogsTool(client *client.Client, logger *zap.Logger) *IngestLogsTool {
	return &IngestLogsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *IngestLogsTool) Name() string {
	return "ingest_logs"
}

// Annotations returns tool hints for LLMs
func (t *IngestLogsTool) Annotations() *mcp.ToolAnnotations {
	return IngestionAnnotations("Ingest Logs")
}

// Description returns a human-readable description of the tool.
func (t *IngestLogsTool) Description() string {
	return `Ingest, push, or add log entries to IBM Cloud Logs for real-time log ingestion.

**Related tools:** query_logs (verify ingested logs), list_policies (check routing), list_enrichments (see applied enrichments)

**Severity Levels:**
- 1: Debug - Detailed debugging information
- 2: Verbose - Verbose output for troubleshooting
- 3: Info - Informational messages (default)
- 4: Warning - Warning conditions
- 5: Error - Error conditions
- 6: Critical - Critical/fatal conditions`
}

// InputSchema returns the JSON schema for the tool's input parameters.
// Aliases (namespace, app, component, etc.) are resolved at runtime.
func (t *IngestLogsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"logs": map[string]interface{}{
				"type":        "array",
				"description": "Array of log entries to ingest (max 1000 per batch)",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"applicationName": map[string]interface{}{
							"type":        "string",
							"description": "App name (also accepts: namespace, app, service)",
						},
						"subsystemName": map[string]interface{}{
							"type":        "string",
							"description": "Subsystem name (also accepts: component, resource, module)",
						},
						"severity": map[string]interface{}{
							"type":        "integer",
							"description": "1=Debug, 2=Verbose, 3=Info, 4=Warning, 5=Error, 6=Critical",
							"minimum":     1,
							"maximum":     6,
						},
						"text": map[string]interface{}{
							"type":        "string",
							"description": "Log message text",
						},
						"timestamp": map[string]interface{}{
							"type":        "number",
							"description": "Unix timestamp with nanoseconds (optional, defaults to now)",
						},
						"json": map[string]interface{}{
							"type":        "object",
							"description": "Structured data to attach to the log entry",
						},
					},
					"required":             []string{"applicationName", "subsystemName", "severity", "text"},
					"additionalProperties": true, // Allow aliases
				},
			},
		},
		"required": []string{"logs"},
	}
}

// MaxIngestionBatchSize is the maximum number of log entries allowed per ingestion request.
// This prevents DoS attacks and ensures reasonable request sizes.
const MaxIngestionBatchSize = 1000

// Execute ingests log entries to IBM Cloud Logs.
// It validates input, adds timestamps where missing, and sends logs to the
// ingestion endpoint (.ingress. subdomain).
//
// Returns an error if validation fails or the API request fails.
func (t *IngestLogsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get logs array
	logsRaw, ok := arguments["logs"].([]interface{})
	if !ok {
		return NewToolResultError("logs must be an array"), nil
	}

	if len(logsRaw) == 0 {
		return NewToolResultError("logs array cannot be empty"), nil
	}

	// Enforce batch size limit to prevent DoS and ensure reasonable request sizes
	if len(logsRaw) > MaxIngestionBatchSize {
		return NewToolResultError(fmt.Sprintf("batch size %d exceeds maximum allowed (%d). Please split into smaller batches", len(logsRaw), MaxIngestionBatchSize)), nil
	}

	// Process each log entry and add timestamp if missing
	logs := make([]map[string]interface{}, 0, len(logsRaw))
	for i, logRaw := range logsRaw {
		logEntry, ok := logRaw.(map[string]interface{})
		if !ok {
			return NewToolResultError("each log entry must be an object"), nil
		}

		// Resolve applicationName aliases (namespace, app, application, service)
		if _, exists := logEntry["applicationName"]; !exists {
			for _, alias := range []string{"namespace", "app", "application", "service", "app_name", "application_name"} {
				if val, exists := logEntry[alias]; exists {
					logEntry["applicationName"] = val
					delete(logEntry, alias) // Remove alias to avoid sending duplicate fields
					break
				}
			}
		}

		// Resolve subsystemName aliases (component, resource, module)
		if _, exists := logEntry["subsystemName"]; !exists {
			for _, alias := range []string{"component", "resource", "subsystem", "module", "component_name", "subsystem_name", "resource_name"} {
				if val, exists := logEntry[alias]; exists {
					logEntry["subsystemName"] = val
					delete(logEntry, alias) // Remove alias to avoid sending duplicate fields
					break
				}
			}
		}

		// Validate required fields after alias resolution
		if _, exists := logEntry["applicationName"]; !exists {
			return NewToolResultError("log entry missing required field: applicationName (or alias: namespace, app, application, service)"), nil
		}
		if _, exists := logEntry["subsystemName"]; !exists {
			return NewToolResultError("log entry missing required field: subsystemName (or alias: component, resource, module)"), nil
		}
		if _, exists := logEntry["severity"]; !exists {
			return NewToolResultError("log entry missing required field: severity"), nil
		}
		if _, exists := logEntry["text"]; !exists {
			return NewToolResultError("log entry missing required field: text"), nil
		}

		// Add current timestamp if not provided
		if _, exists := logEntry["timestamp"]; !exists {
			// Unix timestamp with nanoseconds
			now := time.Now()
			timestamp := float64(now.Unix()) + float64(now.Nanosecond())/1e9
			logEntry["timestamp"] = timestamp
		}

		logs = append(logs, logEntry)
		t.logger.Debug("Processing log entry",
			zap.Int("index", i),
			zap.String("application", logEntry["applicationName"].(string)),
		)
	}

	// Note: The ingestion endpoint is different from the management API
	// It uses: https://{instance-id}.ingress.{region}.logs.cloud.ibm.com/logs/v1/singles
	// We'll need to construct this from the service URL
	req := &client.Request{
		Method:         "POST",
		Path:           "/logs/v1/singles",
		Body:           logs,
		UseIngressHost: true, // Flag to use ingress endpoint instead of API endpoint
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "ingest_logs")
}
