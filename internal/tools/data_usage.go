package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// ExportDataUsageTool exports data usage metrics
type ExportDataUsageTool struct {
	*BaseTool
}

func NewExportDataUsageTool(client *client.Client, logger *zap.Logger) *ExportDataUsageTool {
	return &ExportDataUsageTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *ExportDataUsageTool) Name() string {
	return "export_data_usage"
}

func (t *ExportDataUsageTool) Description() string {
	return "Export data usage metrics for the IBM Cloud Logs instance"
}

func (t *ExportDataUsageTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
		Required:   []string{},
	}
}

func (t *ExportDataUsageTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/data_usage",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateDataUsageMetricsExportStatusTool updates the data usage metrics export status
type UpdateDataUsageMetricsExportStatusTool struct {
	*BaseTool
}

func NewUpdateDataUsageMetricsExportStatusTool(client *client.Client, logger *zap.Logger) *UpdateDataUsageMetricsExportStatusTool {
	return &UpdateDataUsageMetricsExportStatusTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *UpdateDataUsageMetricsExportStatusTool) Name() string {
	return "update_data_usage_metrics_export_status"
}

func (t *UpdateDataUsageMetricsExportStatusTool) Description() string {
	return "Update the data usage metrics export status (enable/disable)"
}

func (t *UpdateDataUsageMetricsExportStatusTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"enabled": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to enable or disable data usage metrics export",
			},
		},
		Required: []string{"enabled"},
	}
}

func (t *UpdateDataUsageMetricsExportStatusTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	enabled, ok := arguments["enabled"].(bool)
	if !ok {
		return mcp.NewToolResultError("enabled parameter must be a boolean"), nil
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
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
