package tools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
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

// Description returns a human-readable description of the tool.
func (t *IngestLogsTool) Description() string {
	return "Ingest, push, or add log entries to IBM Cloud Logs for real-time log ingestion"
}

// InputSchema returns the JSON schema for the tool's input parameters.
// Required fields: logs (array of log entries)
// Each log entry must have: applicationName, subsystemName, severity, text
// Optional fields: timestamp, json
func (t *IngestLogsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"logs": map[string]interface{}{
				"type":        "array",
				"description": "Array of log entries to ingest",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"applicationName": map[string]interface{}{
							"type":        "string",
							"description": "Name of the application generating the log",
						},
						"subsystemName": map[string]interface{}{
							"type":        "string",
							"description": "Name of the subsystem within the application",
						},
						"severity": map[string]interface{}{
							"type":        "integer",
							"description": "Log severity level (1=Debug, 2=Verbose, 3=Info, 4=Warning, 5=Error, 6=Critical)",
							"minimum":     1,
							"maximum":     6,
						},
						"text": map[string]interface{}{
							"type":        "string",
							"description": "The log message text",
						},
						"timestamp": map[string]interface{}{
							"type":        "number",
							"description": "Unix timestamp with nanoseconds (e.g., 1699564800.123456789). If not provided, current time will be used.",
						},
						"json": map[string]interface{}{
							"type":        "object",
							"description": "Optional JSON object containing structured log data",
						},
					},
					"required": []string{"applicationName", "subsystemName", "severity", "text"},
				},
			},
		},
		Required: []string{"logs"},
	}
}

// Execute ingests log entries to IBM Cloud Logs.
// It validates input, adds timestamps where missing, and sends logs to the
// ingestion endpoint (.ingress. subdomain).
//
// Returns an error if validation fails or the API request fails.
func (t *IngestLogsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get logs array
	logsRaw, ok := arguments["logs"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("logs must be an array"), nil
	}

	if len(logsRaw) == 0 {
		return mcp.NewToolResultError("logs array cannot be empty"), nil
	}

	// Process each log entry and add timestamp if missing
	logs := make([]map[string]interface{}, 0, len(logsRaw))
	for i, logRaw := range logsRaw {
		logEntry, ok := logRaw.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("each log entry must be an object"), nil
		}

		// Validate required fields
		if _, exists := logEntry["applicationName"]; !exists {
			return mcp.NewToolResultError("log entry missing required field: applicationName"), nil
		}
		if _, exists := logEntry["subsystemName"]; !exists {
			return mcp.NewToolResultError("log entry missing required field: subsystemName"), nil
		}
		if _, exists := logEntry["severity"]; !exists {
			return mcp.NewToolResultError("log entry missing required field: severity"), nil
		}
		if _, exists := logEntry["text"]; !exists {
			return mcp.NewToolResultError("log entry missing required field: text"), nil
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
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
