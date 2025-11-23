package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ExportDataUsageTool exports data usage metrics
type ExportDataUsageTool struct {
	*BaseTool
}

// NewExportDataUsageTool creates a new tool instance
func NewExportDataUsageTool(client *client.Client, logger *zap.Logger) *ExportDataUsageTool {
	return &ExportDataUsageTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *ExportDataUsageTool) Name() string {
	return "export_data_usage"
}

// Description returns the tool description
func (t *ExportDataUsageTool) Description() string {
	return "Export data usage metrics for the IBM Cloud Logs instance"
}

// InputSchema returns the input schema
func (t *ExportDataUsageTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *ExportDataUsageTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/data_usage",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateDataUsageMetricsExportStatusTool updates the data usage metrics export status
type UpdateDataUsageMetricsExportStatusTool struct {
	*BaseTool
}

// NewUpdateDataUsageMetricsExportStatusTool creates a new tool instance
func NewUpdateDataUsageMetricsExportStatusTool(client *client.Client, logger *zap.Logger) *UpdateDataUsageMetricsExportStatusTool {
	return &UpdateDataUsageMetricsExportStatusTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *UpdateDataUsageMetricsExportStatusTool) Name() string {
	return "update_data_usage_metrics_export_status"
}

// Description returns the tool description
func (t *UpdateDataUsageMetricsExportStatusTool) Description() string {
	return "Update the data usage metrics export status (enable/disable)"
}

// InputSchema returns the input schema
func (t *UpdateDataUsageMetricsExportStatusTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enabled": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to enable or disable data usage metrics export",
			},
		},
		"required": []string{"enabled"},
	}
}

// Execute executes the tool
func (t *UpdateDataUsageMetricsExportStatusTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	enabled, ok := arguments["enabled"].(bool)
	if !ok {
		return NewToolResultError("enabled parameter must be a boolean"), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/data_usage/metrics_export",
		Body: map[string]interface{}{
			"enabled": enabled,
		},
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
