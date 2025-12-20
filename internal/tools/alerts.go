package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// GetAlertTool retrieves a specific alert by ID
type GetAlertTool struct {
	*BaseTool
}

// NewGetAlertTool creates a new tool instance
func NewGetAlertTool(client *client.Client, logger *zap.Logger) *GetAlertTool {
	return &GetAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetAlertTool) Name() string {
	return "get_alert"
}

// Annotations returns tool hints for LLMs
func (t *GetAlertTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Get Alert")
}

// Description returns the tool description
func (t *GetAlertTool) Description() string {
	return "Retrieve a specific alert by its ID from IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *GetAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert",
			},
		},
		"required": []string{"id"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *GetAlertTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryAlerting, CategoryConfiguration},
		Keywords:     []string{"alert", "get", "retrieve", "fetch", "notification", "alarm"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"View alert configuration", "Check alert status", "Inspect alert details"},
		RelatedTools: []string{"list_alerts", "update_alert", "delete_alert", "list_alert_definitions"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":                    map[string]string{"type": "string"},
				"name":                  map[string]string{"type": "string"},
				"is_active":             map[string]string{"type": "boolean"},
				"alert_definition_id":   map[string]string{"type": "string"},
				"notification_group_id": map[string]string{"type": "string"},
			},
		},
		ChainPosition: ChainMiddle,
	}
}

// Execute executes the tool
func (t *GetAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()

	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/alerts/" + id,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return HandleGetError(err, "Alert", id, "list_alerts"), nil
	}

	// Record successful tool use and cache result
	session.RecordToolUse(t.Name(), true, map[string]interface{}{"id": id})
	session.CacheResult(t.Name(), result)

	return t.FormatResponseWithSuggestions(result, "get_alert")
}

// ListAlertsTool lists all alerts
type ListAlertsTool struct {
	*BaseTool
}

// NewListAlertsTool creates a new tool instance
func NewListAlertsTool(client *client.Client, logger *zap.Logger) *ListAlertsTool {
	return &ListAlertsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *ListAlertsTool) Name() string {
	return "list_alerts"
}

// Annotations returns tool hints for LLMs
func (t *ListAlertsTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("List Alerts")
}

// Description returns the tool description
func (t *ListAlertsTool) Description() string {
	return `List all alerts in IBM Cloud Logs.

**When to use:**
- Before creating a new alert (to check for duplicates)
- To audit current alerting configuration
- To find a specific alert's ID for updates or deletion
- After creating an alert (to verify it was created)

**Related tools:** get_alert, create_alert, update_alert, delete_alert, list_alert_definitions, list_outgoing_webhooks`
}

// InputSchema returns the input schema
func (t *ListAlertsTool) InputSchema() interface{} {
	// Use standardized pagination schema for consistency
	props := StandardPaginationSchema()
	return map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *ListAlertsTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryAlerting, CategoryDiscovery},
		Keywords:     []string{"alerts", "list", "all", "notifications", "alarms", "monitoring"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"View all configured alerts", "Audit alerting setup", "Find specific alert"},
		RelatedTools: []string{"get_alert", "create_alert", "list_alert_definitions", "list_outgoing_webhooks"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"alerts": map[string]interface{}{
					"type":  "array",
					"items": map[string]string{"type": "object"},
				},
				"total": map[string]string{"type": "integer"},
			},
		},
		ChainPosition: ChainStart,
	}
}

// Execute executes the tool
func (t *ListAlertsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()
	cacheHelper := GetCacheHelper()

	// Get pagination parameters
	pagination, err := GetPaginationParams(arguments)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	// Generate cache key based on pagination
	cacheKey := "all"
	if cursor, ok := pagination["cursor"].(string); ok && cursor != "" {
		cacheKey = "cursor:" + cursor
	}

	// Check cache first
	if cached, ok := cacheHelper.Get(t.Name(), cacheKey); ok {
		if cachedResult, ok := cached.(map[string]interface{}); ok {
			session.RecordToolUse(t.Name(), true, arguments)
			cachedResult["_cached"] = true
			return t.FormatResponseWithSuggestions(cachedResult, "list_alerts")
		}
	}

	// Build query with pagination
	query := make(map[string]string)
	AddPaginationToQuery(query, pagination)

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/alerts",
		Query:  query,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	// Cache the result
	cacheHelper.Set(t.Name(), cacheKey, result)

	// Record successful tool use and cache result
	session.RecordToolUse(t.Name(), true, arguments)
	session.CacheResult(t.Name(), result)

	return t.FormatResponseWithSuggestions(result, "list_alerts")
}

// CreateAlertTool creates a new alert
type CreateAlertTool struct {
	*BaseTool
}

// NewCreateAlertTool creates a new tool instance
func NewCreateAlertTool(client *client.Client, logger *zap.Logger) *CreateAlertTool {
	return &CreateAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *CreateAlertTool) Name() string {
	return "create_alert"
}

// Annotations returns tool hints for LLMs
func (t *CreateAlertTool) Annotations() *mcp.ToolAnnotations {
	return CreateAnnotations("Create Alert")
}

// Description returns the tool description
func (t *CreateAlertTool) Description() string {
	return `Create a new alert in IBM Cloud Logs linking an alert definition to notification webhooks.

**Related tools:** list_alerts, get_alert, list_alert_definitions, create_alert_def, list_outgoing_webhooks, create_outgoing_webhook

**Prerequisites:**
1. Create an alert definition (create_alert_def) to define the trigger condition
2. Create an outgoing webhook (create_outgoing_webhook) for notifications
3. Use this tool to link them together`
}

// InputSchema returns the input schema
func (t *CreateAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"alert": map[string]interface{}{
				"type":        "object",
				"description": "The alert configuration object",
				"example": map[string]interface{}{
					"name":                  "Production Error Alert",
					"is_active":             true,
					"alert_definition_id":   "alert-def-uuid-here",
					"notification_group_id": "notification-group-uuid-here",
					"filters": map[string]interface{}{
						"severities": []string{"error", "critical"},
						"applications": []map[string]interface{}{
							{"id": "app-uuid", "name": "api-gateway"},
						},
					},
				},
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the alert configuration without creating it. Use this to preview what will be created and check for errors.",
				"default":     false,
			},
		},
		"required": []string{"alert"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *CreateAlertTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryAlerting, CategoryConfiguration},
		Keywords:     []string{"alert", "create", "new", "add", "notification", "alarm", "setup"},
		Complexity:   ComplexityModerate,
		UseCases:     []string{"Set up new alerting", "Configure notifications", "Create monitoring rules"},
		RelatedTools: []string{"list_alerts", "create_alert_def", "create_outgoing_webhook", "list_notification_groups"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":   map[string]string{"type": "string"},
				"name": map[string]string{"type": "string"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute executes the tool
func (t *CreateAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()
	cacheHelper := GetCacheHelper()

	alert, err := GetObjectParam(arguments, "alert", true)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	// Check for dry-run mode
	dryRun, _ := GetBoolParam(arguments, "dry_run", false)
	if dryRun {
		return t.validateAlert(alert)
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/alerts",
		Body:   alert,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	// Record successful tool use
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"alert_name": alert["name"],
	})

	return t.FormatResponseWithSuggestions(result, "create_alert")
}

// validateAlert performs dry-run validation for alert creation
func (t *CreateAlertTool) validateAlert(alert map[string]interface{}) (*mcp.CallToolResult, error) {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	// Validate required fields
	requiredFields := []string{"name"}
	for _, field := range requiredFields {
		if _, ok := alert[field]; !ok {
			result.Errors = append(result.Errors, "Missing required field: "+field)
			result.Valid = false
		}
	}

	// Validate name length
	if name, ok := alert["name"].(string); ok {
		if len(name) < 1 {
			result.Errors = append(result.Errors, "Alert name must not be empty")
			result.Valid = false
		}
		if len(name) > 4096 {
			result.Errors = append(result.Errors, "Alert name must be at most 4096 characters")
			result.Valid = false
		}
		result.Summary["name"] = name
	}

	// Validate is_active
	if isActive, ok := alert["is_active"].(bool); ok {
		result.Summary["is_active"] = isActive
	} else {
		result.Summary["is_active"] = true // default
	}

	// Validate alert_definition_id (recommended)
	if alertDefID, ok := alert["alert_definition_id"].(string); ok && alertDefID != "" {
		result.Summary["alert_definition_id"] = alertDefID
	} else {
		result.Warnings = append(result.Warnings, "No alert_definition_id provided - consider using list_alert_definitions to find an existing definition or create_alert_def to create one")
	}

	// Validate notification_group_id (recommended)
	if notifGroupID, ok := alert["notification_group_id"].(string); ok && notifGroupID != "" {
		result.Summary["notification_group_id"] = notifGroupID
	} else {
		result.Warnings = append(result.Warnings, "No notification_group_id provided - without this, alert triggers won't send notifications")
	}

	// Validate filters if provided
	if filters, ok := alert["filters"].(map[string]interface{}); ok {
		filterCount := 0
		if severities, ok := filters["severities"].([]interface{}); ok {
			filterCount += len(severities)
			result.Summary["severity_filters"] = len(severities)
		}
		if apps, ok := filters["applications"].([]interface{}); ok {
			filterCount += len(apps)
			result.Summary["application_filters"] = len(apps)
		}
		if subsystems, ok := filters["subsystems"].([]interface{}); ok {
			filterCount += len(subsystems)
			result.Summary["subsystem_filters"] = len(subsystems)
		}
		if filterCount == 0 {
			result.Warnings = append(result.Warnings, "No filters specified - alert will match all logs")
		}
	} else {
		result.Warnings = append(result.Warnings, "No filters specified - alert will match all logs")
	}

	// Add suggestions
	if result.Valid {
		result.Suggestions = append(result.Suggestions, "Alert configuration is valid")
		result.Suggestions = append(result.Suggestions, "Remove dry_run parameter to create the alert")
	} else {
		result.Suggestions = append(result.Suggestions, "Fix the errors above before creating the alert")
		result.Suggestions = append(result.Suggestions, "Use list_alert_definitions to find existing alert definitions")
		result.Suggestions = append(result.Suggestions, "Use list_outgoing_webhooks to find notification targets")
	}

	// Estimate impact
	result.EstimatedImpact = &ImpactEstimate{
		RiskLevel: "low",
	}

	return FormatDryRunResult(result, "Alert", alert), nil
}

// UpdateAlertTool updates an existing alert
type UpdateAlertTool struct {
	*BaseTool
}

// NewUpdateAlertTool creates a new tool instance
func NewUpdateAlertTool(client *client.Client, logger *zap.Logger) *UpdateAlertTool {
	return &UpdateAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *UpdateAlertTool) Name() string {
	return "update_alert"
}

// Annotations returns tool hints for LLMs
func (t *UpdateAlertTool) Annotations() *mcp.ToolAnnotations {
	return UpdateAnnotations("Update Alert")
}

// Description returns the tool description
func (t *UpdateAlertTool) Description() string {
	return "Update an existing alert in IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *UpdateAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert to update",
			},
			"alert": map[string]interface{}{
				"type":        "object",
				"description": "The updated alert configuration",
			},
		},
		"required": []string{"id", "alert"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *UpdateAlertTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryAlerting, CategoryConfiguration},
		Keywords:     []string{"alert", "update", "modify", "change", "edit", "notification"},
		Complexity:   ComplexityModerate,
		UseCases:     []string{"Modify alert configuration", "Update notification settings", "Change alert thresholds"},
		RelatedTools: []string{"get_alert", "list_alerts", "delete_alert"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":   map[string]string{"type": "string"},
				"name": map[string]string{"type": "string"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute executes the tool
func (t *UpdateAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	cacheHelper := GetCacheHelper()

	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	alert, err := GetObjectParam(arguments, "alert", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/alerts/" + id,
		Body:   alert,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	return t.FormatResponseWithSuggestions(result, "update_alert")
}

// DeleteAlertTool deletes an alert
type DeleteAlertTool struct {
	*BaseTool
}

// NewDeleteAlertTool creates a new tool instance
func NewDeleteAlertTool(client *client.Client, logger *zap.Logger) *DeleteAlertTool {
	return &DeleteAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *DeleteAlertTool) Name() string {
	return "delete_alert"
}

// Annotations returns tool hints for LLMs
func (t *DeleteAlertTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Alert")
}

// Description returns the tool description
func (t *DeleteAlertTool) Description() string {
	return "Delete an alert from IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *DeleteAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert to delete",
			},
			"confirm": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to confirm deletion. Required to prevent accidental deletions.",
				"default":     false,
			},
		},
		"required": []string{"id"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *DeleteAlertTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryAlerting, CategoryConfiguration},
		Keywords:     []string{"alert", "delete", "remove", "disable", "notification"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"Remove obsolete alerts", "Clean up alerting", "Disable notifications"},
		RelatedTools: []string{"get_alert", "list_alerts", "update_alert"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"success": map[string]string{"type": "boolean"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute executes the tool
func (t *DeleteAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	cacheHelper := GetCacheHelper()

	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Require explicit confirmation for destructive operations
	if shouldContinue, result := RequireConfirmation(arguments, "alert", id); !shouldContinue {
		return result, nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/alerts/" + id,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	return t.FormatResponseWithSuggestions(result, "delete_alert")
}
