// Package tools provides MCP tools for IBM Cloud Logs operations.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// GetAlertDefinitionTool retrieves a specific alert definition by ID
type GetAlertDefinitionTool struct {
	*BaseTool
}

// NewGetAlertDefinitionTool creates a new tool instance
func NewGetAlertDefinitionTool(client client.Doer, logger *zap.Logger) *GetAlertDefinitionTool {
	return &GetAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *GetAlertDefinitionTool) Name() string { return "get_alert_definition" }

// Description returns the tool description
func (t *GetAlertDefinitionTool) Description() string {
	return "Retrieve a specific alert definition by its ID"
}

// InputSchema returns the input schema
func (t *GetAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "string", "description": "Alert definition ID",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *GetAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: apiPath("/v1/alert_definitions", id)})
	if err != nil {
		return HandleGetError(err, "Alert definition", id, "list_alert_definitions"), nil
	}
	return t.FormatResponseWithSuggestions(result, "get_alert_definition")
}

// ListAlertDefinitionsTool lists all alert definitions
type ListAlertDefinitionsTool struct {
	*BaseTool
}

// NewListAlertDefinitionsTool creates a new tool instance
func NewListAlertDefinitionsTool(client client.Doer, logger *zap.Logger) *ListAlertDefinitionsTool {
	return &ListAlertDefinitionsTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *ListAlertDefinitionsTool) Name() string { return "list_alert_definitions" }

// Description returns the tool description
func (t *ListAlertDefinitionsTool) Description() string {
	return "List all alert definitions"
}

// InputSchema returns the input schema
func (t *ListAlertDefinitionsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *ListAlertDefinitionsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/alert_definitions"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "list_alert_definitions")
}

// CreateAlertDefinitionTool creates a new alert definition
type CreateAlertDefinitionTool struct {
	*BaseTool
}

// NewCreateAlertDefinitionTool creates a new tool instance
func NewCreateAlertDefinitionTool(client client.Doer, logger *zap.Logger) *CreateAlertDefinitionTool {
	return &CreateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *CreateAlertDefinitionTool) Name() string { return "create_alert_definition" }

// Description returns the tool description
func (t *CreateAlertDefinitionTool) Description() string {
	return `Create a new alert definition to monitor log patterns and trigger notifications.

**Related tools:** list_alert_definitions, get_alert_definition, create_alert, create_outgoing_webhook

**Alert Types:**
- logs_immediate: Triggered immediately when condition matches
- logs_threshold: Triggered when count exceeds threshold over time window
- logs_ratio: Triggered when ratio between two queries exceeds threshold
- logs_anomaly: Triggered on anomaly detection
- logs_new_value: Triggered when a new value appears in logs

**Label filters** (simple_filter.label_filters.application_name / subsystem_name)
use an "operation" enum of: is_or_unspecified, includes, ends_with, starts_with.
Common aliases (is, equals, contains) are auto-normalized to the canonical value;
run with dry_run: true to preview any normalization before creating.

**Tier note:** only high- and medium-priority logs are available to alerts. If
a TCO policy assigns the targeted application/subsystem a low (or unspecified)
priority, the alert will not fire — dry_run and the create response surface a
_tier_warnings note when that is the case.`
}

// InputSchema returns the input schema
func (t *CreateAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"definition": map[string]interface{}{
				"type":        "object",
				"description": "Alert definition configuration",
				"example": map[string]interface{}{
					"name":        "High Error Rate Alert",
					"description": "Fires when error logs exceed 10% of total logs over 5 minutes",
					"enabled":     true,
					"type":        "logs_ratio_threshold",
					"logs_ratio_threshold": map[string]interface{}{
						"condition_type": "more_than_or_unspecified",
						"group_by_for":   "both_or_unspecified",
						"numerator": map[string]interface{}{
							"simple_filter": map[string]interface{}{
								"lucene_query": "subsystemname:payment-service AND severity:error",
							},
						},
						"numerator_alias": "errors",
						"denominator": map[string]interface{}{
							"simple_filter": map[string]interface{}{
								"lucene_query": "subsystemname:payment-service",
							},
						},
						"denominator_alias": "total",
						"rules": []map[string]interface{}{
							{
								"condition": map[string]interface{}{
									"threshold": 0.1,
									"time_window": map[string]interface{}{
										"logs_ratio_time_window_specific_value": "minutes_5_or_unspecified",
									},
									"condition_type": "more_than_or_unspecified",
								},
								"override": map[string]interface{}{"priority": "p1"},
							},
						},
					},
				},
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the alert definition without creating it. Use this to preview and check for errors.",
				"default":     false,
			},
		},
		"required": []string{"definition"},
	}
}

// Execute executes the tool
func (t *CreateAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	def, err := GetObjectParam(arguments, "definition", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Check for dry-run mode. Validation runs against the raw definition so
	// the preview reports which operations would be normalized.
	dryRun, _ := GetBoolParam(arguments, "dry_run", false)
	if dryRun {
		return t.validateAlertDefinition(ctx, def)
	}

	// Rewrite intuitive label-filter operation aliases (e.g. "is") to the
	// API's canonical enum ("is_or_unspecified") before sending, so the
	// request is not rejected for a predictable vocabulary mismatch.
	notes := normalizeAlertDefinition(def)

	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/alert_definitions", Body: def})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	attachNormalizationNotes(result, notes)
	attachTierWarnings(result, alertTierWarnings(def, sessionAsIs(ctx)))
	return t.FormatResponseWithSuggestions(result, "create_alert_definition")
}

// v3 alert definition enums, as enforced by the live API's deserializer.
var validAlertDefTypes = []string{
	"logs_immediate_or_unspecified",
	"logs_threshold",
	"logs_anomaly",
	"logs_ratio_threshold",
	"logs_new_value",
	"logs_unique_count",
	"logs_time_relative_threshold",
	"metric_threshold",
	"metric_anomaly",
	"flow",
}

var validAlertDefPriorities = []string{"p5_or_unspecified", "p4", "p3", "p2", "p1"}

// validLabelFilterOperations lists the canonical enum values the live API
// accepts for a label filter's "operation" field
// (ApisAlertDefinitionLogFilterOperationType). Note the API uses
// "is_or_unspecified" rather than a plain "is".
var validLabelFilterOperations = []string{"is_or_unspecified", "includes", "ends_with", "starts_with"}

// labelFilterOperationAliases maps intuitive-but-wrong operation values a
// caller is likely to send onto the API's canonical enum. Every canonical
// value also maps to itself so lookups are total. The alias set is kept
// deliberately small and unambiguous — anything not listed here is left
// untouched for the validator to reject rather than silently guessed at.
var labelFilterOperationAliases = map[string]string{
	// canonical -> itself
	"is_or_unspecified": "is_or_unspecified",
	"includes":          "includes",
	"ends_with":         "ends_with",
	"starts_with":       "starts_with",
	// friendly aliases -> canonical
	"is":         "is_or_unspecified",
	"equals":     "is_or_unspecified",
	"eq":         "is_or_unspecified",
	"contains":   "includes",
	"include":    "includes",
	"startswith": "starts_with",
	"endswith":   "ends_with",
}

// labelFilterKeys are the label_filters sub-arrays whose items carry an
// "operation" field. "severities" is intentionally excluded — it is a plain
// enum array, not a {value, operation} filter.
var labelFilterKeys = []string{"application_name", "subsystem_name"}

// normalizeAlertDefinition walks an alert definition in place and rewrites
// label-filter "operation" values that use a known alias (e.g. "is") to the
// API's canonical variant (e.g. "is_or_unspecified"). It returns a note for
// each rewrite it performed so callers can surface what changed. Unknown
// operation values are left untouched — normalization corrects predictable
// vocabulary mismatches, it does not invent values; the validator reports
// anything genuinely invalid.
func normalizeAlertDefinition(def map[string]interface{}) []string {
	var notes []string
	walkLabelFilters(def, func(filter map[string]interface{}) {
		op, ok := filter["operation"].(string)
		if !ok {
			return
		}
		canonical, known := labelFilterOperationAliases[op]
		if known && canonical != op {
			filter["operation"] = canonical
			notes = append(notes, fmt.Sprintf("label filter operation %q normalized to %q", op, canonical))
		}
	})
	return notes
}

// attachNormalizationNotes records any normalization notes on the API
// response so the caller can see which values were rewritten. It is a no-op
// when nothing was normalized or the response is not a JSON object.
func attachNormalizationNotes(result map[string]interface{}, notes []string) {
	if len(notes) == 0 || result == nil {
		return
	}
	result["_normalizations"] = notes
}

// walkLabelFilters recursively descends an alert definition and invokes fn
// for every label filter object (an item of a label_filters.application_name
// or .subsystem_name array), regardless of how deeply the label_filters map
// is nested (numerator/denominator, logs_filter, simple_filter, etc.).
func walkLabelFilters(node interface{}, fn func(filter map[string]interface{})) {
	switch n := node.(type) {
	case map[string]interface{}:
		if lf, ok := n["label_filters"].(map[string]interface{}); ok {
			for _, key := range labelFilterKeys {
				if arr, ok := lf[key].([]interface{}); ok {
					for _, item := range arr {
						if filter, ok := item.(map[string]interface{}); ok {
							fn(filter)
						}
					}
				}
			}
		}
		for _, v := range n {
			walkLabelFilters(v, fn)
		}
	case []interface{}:
		for _, v := range n {
			walkLabelFilters(v, fn)
		}
	}
}

// validateLabelFilterOperations inspects every label-filter operation in the
// definition. A canonical value passes silently; a known alias produces a
// warning noting the rewrite that create/update will apply; an unrecognized
// value is a hard error listing the valid variants.
func validateLabelFilterOperations(def map[string]interface{}, result *ValidationResult) {
	walkLabelFilters(def, func(filter map[string]interface{}) {
		op, ok := filter["operation"].(string)
		if !ok {
			return
		}
		canonical, known := labelFilterOperationAliases[op]
		switch {
		case !known:
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid label filter operation %q. Valid values: %s", op, joinStrings(validLabelFilterOperations, ", ")))
			result.Valid = false
		case canonical != op:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Label filter operation %q will be normalized to %q", op, canonical))
		}
	})
}

// alertTypeConfigKey returns the definition key that must hold the config
// object for a given alert type. For every type except the immediate default,
// the key matches the type name (e.g. type "logs_ratio_threshold" requires a
// "logs_ratio_threshold" object).
func alertTypeConfigKey(alertType string) string {
	if alertType == "logs_immediate_or_unspecified" {
		return ""
	}
	return alertType
}

// rulesHaveOverride reports whether any rule inside the type config object
// carries an override. The live API rejects definitions that set a top-level
// priority alongside rule overrides.
func rulesHaveOverride(typeConfig map[string]interface{}) bool {
	rules, ok := typeConfig["rules"].([]interface{})
	if !ok {
		return false
	}
	for _, r := range rules {
		if rule, ok := r.(map[string]interface{}); ok {
			if _, has := rule["override"]; has {
				return true
			}
		}
	}
	return false
}

// validateAlertDefinitionConfig checks an alert definition against the live
// v3 API contract. It intentionally covers the failure modes the API is known
// to reject (type/priority enums, missing type config, priority-vs-override
// conflict) rather than replicating the full server-side schema.
func validateAlertDefinitionConfig(def map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	for _, field := range []string{"name", "type"} {
		if _, ok := def[field]; !ok {
			result.Errors = append(result.Errors, "Missing required field: "+field)
			result.Valid = false
		}
	}

	if name, ok := def["name"].(string); ok {
		result.Summary["name"] = name
	}

	alertType, _ := def["type"].(string)
	if alertType != "" {
		result.Summary["type"] = alertType
		valid := false
		for _, t := range validAlertDefTypes {
			if alertType == t {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors,
				"Invalid alert type: "+alertType+". Valid types: "+joinStrings(validAlertDefTypes, ", "))
			result.Valid = false
		} else if key := alertTypeConfigKey(alertType); key != "" {
			typeConfig, ok := def[key].(map[string]interface{})
			if !ok {
				result.Errors = append(result.Errors,
					"Alert type "+alertType+" requires a matching config object under the key \""+key+"\"")
				result.Valid = false
			} else if alertType == "logs_threshold" || alertType == "logs_ratio_threshold" {
				// The API requires condition_type both inside each rule's
				// condition AND at the top of the type config object.
				if _, has := typeConfig["condition_type"]; !has {
					result.Errors = append(result.Errors,
						key+" requires a top-level \"condition_type\" field (in addition to the one inside each rule's condition)")
					result.Valid = false
				}
			}
		}
	}

	if priority, ok := def["priority"].(string); ok {
		valid := false
		for _, p := range validAlertDefPriorities {
			if priority == p {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors,
				"Invalid priority: "+priority+". Valid values (lowercase): "+joinStrings(validAlertDefPriorities, ", "))
			result.Valid = false
		}
		if key := alertTypeConfigKey(alertType); key != "" {
			if typeConfig, ok := def[key].(map[string]interface{}); ok && rulesHaveOverride(typeConfig) {
				result.Errors = append(result.Errors,
					"Cannot set a top-level priority when rules define an override — remove one of them")
				result.Valid = false
			}
		}
	}

	validateLabelFilterOperations(def, result)

	if result.Valid {
		result.Suggestions = append(result.Suggestions, "Alert definition configuration is valid")
		result.Suggestions = append(result.Suggestions, "Remove dry_run parameter to create the alert definition")
		result.Warnings = append(result.Warnings,
			"Dry-run checks known API constraints but is not a full schema validation — the API may still reject unknown fields or enum values")
	} else {
		result.Suggestions = append(result.Suggestions, "Fix the errors above before creating")
	}

	result.EstimatedImpact = &ImpactEstimate{RiskLevel: "low"}
	return result
}

// validateAlertDefinition performs dry-run validation. It also warns when the
// alert targets logs TCO routes to the archive tier only (such an alert never
// fires, since alerts evaluate on the frequent_search stream).
func (t *CreateAlertDefinitionTool) validateAlertDefinition(ctx context.Context, def map[string]interface{}) (*mcp.CallToolResult, error) {
	result := validateAlertDefinitionConfig(def)
	result.Warnings = append(result.Warnings, alertTierWarnings(def, sessionAsIs(ctx))...)
	return FormatDryRunResult(result, "Alert Definition", def), nil
}

// UpdateAlertDefinitionTool updates an existing alert definition
type UpdateAlertDefinitionTool struct {
	*BaseTool
}

// NewUpdateAlertDefinitionTool creates a new tool instance
func NewUpdateAlertDefinitionTool(client client.Doer, logger *zap.Logger) *UpdateAlertDefinitionTool {
	return &UpdateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *UpdateAlertDefinitionTool) Name() string { return "update_alert_definition" }

// Description returns the tool description
func (t *UpdateAlertDefinitionTool) Description() string {
	return "Update an existing alert definition"
}

// InputSchema returns the input schema
func (t *UpdateAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]interface{}{"type": "string", "description": "Alert definition ID"},
			"definition": map[string]interface{}{"type": "object", "description": "Updated definition"},
		},
		"required": []string{"id", "definition"},
	}
}

// Execute executes the tool
func (t *UpdateAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	def, err := GetObjectParam(arguments, "definition", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	notes := normalizeAlertDefinition(def)
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: apiPath("/v1/alert_definitions", id), Body: def})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	attachNormalizationNotes(result, notes)
	attachTierWarnings(result, alertTierWarnings(def, sessionAsIs(ctx)))
	return t.FormatResponseWithSuggestions(result, "update_alert_definition")
}

// DeleteAlertDefinitionTool deletes an alert definition
type DeleteAlertDefinitionTool struct {
	*BaseTool
}

// NewDeleteAlertDefinitionTool creates a new tool instance
func NewDeleteAlertDefinitionTool(client client.Doer, logger *zap.Logger) *DeleteAlertDefinitionTool {
	return &DeleteAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *DeleteAlertDefinitionTool) Name() string { return "delete_alert_definition" }

// Annotations returns tool hints for LLMs
func (t *DeleteAlertDefinitionTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Alert Definition")
}

// Description returns the tool description
func (t *DeleteAlertDefinitionTool) Description() string {
	return "Delete an alert definition"
}

// InputSchema returns the input schema
func (t *DeleteAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string", "description": "Alert definition ID"},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *DeleteAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: apiPath("/v1/alert_definitions", id)})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "delete_alert_definition")
}
