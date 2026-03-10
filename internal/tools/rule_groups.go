package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"go.uber.org/zap"
)

// GetRuleGroupTool retrieves a specific rule group by ID.
type GetRuleGroupTool struct{ *BaseTool }

// NewGetRuleGroupTool creates a new tool instance
func NewGetRuleGroupTool(c client.Doer, l *zap.Logger) *GetRuleGroupTool {
	return &GetRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetRuleGroupTool) Name() string { return "get_rule_group" }

// Description returns the tool description
func (t *GetRuleGroupTool) Description() string {
	return "Get a rule group by ID"
}

// InputSchema returns the input schema
func (t *GetRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Rule group ID",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *GetRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(args, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return HandleGetError(err, "Rule group", id, "list_rule_groups"), nil
	}
	return t.FormatResponseWithSuggestions(res, "get_rule_group")
}

// ListRuleGroupsTool lists all rule groups.
type ListRuleGroupsTool struct{ *BaseTool }

// NewListRuleGroupsTool creates a new tool instance
func NewListRuleGroupsTool(c client.Doer, l *zap.Logger) *ListRuleGroupsTool {
	return &ListRuleGroupsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListRuleGroupsTool) Name() string { return "list_rule_groups" }

// Description returns the tool description
func (t *ListRuleGroupsTool) Description() string {
	return `List all rule groups for parsing and transforming log data.

**Related tools:** get_rule_group, create_rule_group, update_rule_group, delete_rule_group`
}

// InputSchema returns the input schema
func (t *ListRuleGroupsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *ListRuleGroupsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(res, "list_rule_groups")
}

// CreateRuleGroupTool creates a new rule group.
type CreateRuleGroupTool struct{ *BaseTool }

// NewCreateRuleGroupTool creates a new tool instance
func NewCreateRuleGroupTool(c client.Doer, l *zap.Logger) *CreateRuleGroupTool {
	return &CreateRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateRuleGroupTool) Name() string { return "create_rule_group" }

// Description returns the tool description
func (t *CreateRuleGroupTool) Description() string {
	return `Create a new rule group for parsing and transforming log data.

**Rule Types:**
- **extract_parameters**: Extract fields using regex while keeping original log
- **parse_parameters**: Parse log into JSON fields. Can transform in-place by setting destination_field = source_field
- **json_extract_parameters**: Extract JSON field to metadata
- **replace_parameters**: Replace text matching regex
- **allow_parameters**: Allow/block logs based on regex
- **block_parameters**: Block/allow logs based on regex
- **extract_timestamp_parameters**: Extract timestamp from logs
- **remove_fields_parameters**: Remove specific JSON fields
- **json_stringify_parameters**: Convert JSON to string
- **json_parse_parameters**: Parse string to JSON

**Valid Source Fields:**
- 'text' - Main log message field
- 'text.log' - Nested log field from JSON logs (common for Kubernetes logs)
- 'text.<fieldname>' - Any nested field under text
- 'json.<fieldname>' - Custom JSON fields

**Rule Matchers:**
- **application_name**: Match by application name
- **subsystem_name**: Match by subsystem name
- **severity**: Match by log severity

**Related tools:** list_rule_groups, get_rule_group, update_rule_group, delete_rule_group`
}

// InputSchema returns the input schema
func (t *CreateRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rule_group": map[string]interface{}{
				"type":        "object",
				"description": "Rule group configuration",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the rule group",
						"minLength":   1,
						"maxLength":   255,
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the rule group purpose",
						"minLength":   1,
						"maxLength":   4096,
					},
					"enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the rule group is enabled",
						"default":     true,
					},
					"order": map[string]interface{}{
						"type":        "integer",
						"description": "Execution order (lower runs first)",
						"minimum":     0,
						"maximum":     4294967295,
					},
					"rule_matchers": map[string]interface{}{
						"type":        "array",
						"description": "Optional matchers to filter which logs this rule group processes",
						"items": map[string]interface{}{
							"type": "object",
							"oneOf": []interface{}{
								map[string]interface{}{
									"properties": map[string]interface{}{
										"application_name": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"value": map[string]interface{}{
													"type":        "string",
													"description": "Application name to match",
												},
											},
											"required": []string{"value"},
										},
									},
								},
								map[string]interface{}{
									"properties": map[string]interface{}{
										"subsystem_name": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"value": map[string]interface{}{
													"type":        "string",
													"description": "Subsystem name to match",
												},
											},
											"required": []string{"value"},
										},
									},
								},
								map[string]interface{}{
									"properties": map[string]interface{}{
										"severity": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"value": map[string]interface{}{
													"type": "string",
													"enum": []string{
														"debug_or_unspecified",
														"verbose",
														"info",
														"warning",
														"error",
														"critical",
													},
													"description": "Severity level to match",
												},
											},
											"required": []string{"value"},
										},
									},
								},
							},
						},
					},
					"rule_subgroups": map[string]interface{}{
						"type":        "array",
						"description": "Rule subgroups executed in order",
						"minItems":    1,
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"enabled": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether this subgroup is enabled",
								},
								"order": map[string]interface{}{
									"type":        "integer",
									"description": "Execution order within the rule group",
									"minimum":     0,
									"maximum":     4294967295,
								},
								"rules": map[string]interface{}{
									"type":        "array",
									"description": "Rules to execute",
									"minItems":    1,
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"name": map[string]interface{}{
												"type":        "string",
												"description": "Rule name",
												"minLength":   1,
												"maxLength":   4096,
											},
											"description": map[string]interface{}{
												"type":        "string",
												"description": "Rule description",
												"minLength":   1,
												"maxLength":   4096,
											},
											"source_field": map[string]interface{}{
												"type":        "string",
												"description": "Field to apply the rule to. Valid fields:\n- 'text' - Main log message\n- 'text.log' - Nested log field from JSON logs (common for Kubernetes)\n- 'text.<field>' - Any nested field under text\n- 'json.<field>' - Custom JSON fields\nNote: Use lowercase for field names.",
												"minLength":   1,
												"maxLength":   4096,
												"examples":    []string{"text", "text.log", "json.message"},
											},
											"enabled": map[string]interface{}{
												"type":        "boolean",
												"description": "Whether this rule is enabled",
											},
											"order": map[string]interface{}{
												"type":        "integer",
												"description": "Execution order within the subgroup",
												"minimum":     0,
												"maximum":     4294967295,
											},
											"parameters": map[string]interface{}{
												"type":        "object",
												"description": "Rule parameters (one of: extract_parameters, parse_parameters, json_extract_parameters, replace_parameters, allow_parameters, block_parameters, extract_timestamp_parameters, remove_fields_parameters, json_stringify_parameters, json_parse_parameters)",
											},
										},
										"required": []string{"source_field", "parameters", "enabled", "order"},
									},
								},
							},
							"required": []string{"rules", "order"},
						},
					},
				},
				"required": []string{"name", "rule_subgroups"},
			},
		},
		"required": []string{"rule_group"},
		"examples": []interface{}{
			map[string]interface{}{
				"rule_group": map[string]interface{}{
					"name":        "Nginx Error Log Parser",
					"description": "Parse nginx error logs into structured JSON",
					"enabled":     true,
					"order":       1,
					"rule_matchers": []interface{}{
						map[string]interface{}{
							"subsystem_name": map[string]interface{}{
								"value": "nginx",
							},
						},
					},
					"rule_subgroups": []interface{}{
						map[string]interface{}{
							"enabled": true,
							"order":   1,
							"rules": []interface{}{
								map[string]interface{}{
									"name":         "Parse Nginx Error from text.log",
									"description":  "Extract fields from nginx error log in text.log field (common for Kubernetes logs)",
									"source_field": "text.log",
									"enabled":      true,
									"order":        1,
									"parameters": map[string]interface{}{
										"parse_parameters": map[string]interface{}{
											"destination_field": "text.log",
											"rule":              `(?P<timestamp>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<level>\w+)\] (?P<pid>\d+)#(?P<tid>\d+): \*(?P<cid>\d+) (?P<message>.*), client: (?P<client_ip>[\d\.]+), server: (?P<server>[^,]+)`,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// validateSourceFields validates source_field values in a rule group
func validateSourceFields(rg map[string]interface{}) error {
	validPrefixes := []string{"text", "json", "kubernetes", "log"}

	subgroups, ok := rg["rule_subgroups"].([]interface{})
	if !ok {
		return nil // Will be caught by API validation
	}

	for _, sg := range subgroups {
		subgroup, ok := sg.(map[string]interface{})
		if !ok {
			continue
		}

		rules, ok := subgroup["rules"].([]interface{})
		if !ok {
			continue
		}

		for _, r := range rules {
			rule, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			sourceField, ok := rule["source_field"].(string)
			if !ok || sourceField == "" {
				continue
			}

			// Check if source_field starts with a valid prefix
			valid := false
			for _, prefix := range validPrefixes {
				if sourceField == prefix || len(sourceField) > len(prefix) && sourceField[:len(prefix)+1] == prefix+"." {
					valid = true
					break
				}
			}

			if !valid {
				return fmt.Errorf("invalid source_field '%s'. Valid fields:\n- text (main log message)\n- text.log (nested log field, common for Kubernetes)\n- text.<fieldname> (any nested field)\n- json.<fieldname> (custom JSON fields)\n- kubernetes.<fieldname> (Kubernetes metadata)\n- log.<fieldname> (log metadata)", sourceField)
			}
		}
	}

	return nil
}

// Execute executes the tool
func (t *CreateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	rg, err := GetObjectParam(args, "rule_group", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Validate source fields before sending to API
	if err := validateSourceFields(rg); err != nil {
		return NewToolResultError(fmt.Sprintf("Validation error: %v", err)), nil
	}

	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/rule_groups", Body: rg})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(res, "create_rule_group")
}

// UpdateRuleGroupTool updates an existing rule group.
type UpdateRuleGroupTool struct{ *BaseTool }

// NewUpdateRuleGroupTool creates a new tool instance
func NewUpdateRuleGroupTool(c client.Doer, l *zap.Logger) *UpdateRuleGroupTool {
	return &UpdateRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdateRuleGroupTool) Name() string { return "update_rule_group" }

// Description returns the tool description
func (t *UpdateRuleGroupTool) Description() string {
	return "Update a rule group"
}

// InputSchema returns the input schema
func (t *UpdateRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Rule group ID to update",
			},
			"rule_group": map[string]interface{}{
				"type":        "object",
				"description": "Updated rule group configuration (same structure as create_rule_group)",
			},
		},
		"required": []string{"id", "rule_group"},
	}
}

// Execute executes the tool
func (t *UpdateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(args, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	rg, err := GetObjectParam(args, "rule_group", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Validate source fields before sending to API
	if err := validateSourceFields(rg); err != nil {
		return NewToolResultError(fmt.Sprintf("Validation error: %v", err)), nil
	}

	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/rule_groups/" + id, Body: rg})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(res, "update_rule_group")
}

// DeleteRuleGroupTool deletes a rule group.
type DeleteRuleGroupTool struct{ *BaseTool }

// NewDeleteRuleGroupTool creates a new tool instance
func NewDeleteRuleGroupTool(c client.Doer, l *zap.Logger) *DeleteRuleGroupTool {
	return &DeleteRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteRuleGroupTool) Name() string { return "delete_rule_group" }

// Annotations returns tool hints for LLMs
func (t *DeleteRuleGroupTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Rule Group")
}

// Description returns the tool description
func (t *DeleteRuleGroupTool) Description() string {
	return "Delete a rule group"
}

// InputSchema returns the input schema
func (t *DeleteRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Rule group ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *DeleteRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(args, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(res, "delete_rule_group")
}

// Made with Bob
