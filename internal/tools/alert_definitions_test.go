package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAlertDefinitionTool_InputSchema(t *testing.T) {
	tool := &GetAlertDefinitionTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestListAlertDefinitionsTool_InputSchema(t *testing.T) {
	tool := &ListAlertDefinitionsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	// Required might be nil or empty slice, depending on implementation
	if req, ok := schema["required"]; ok {
		assert.Empty(t, req)
	}
}

func TestCreateAlertDefinitionTool_InputSchema(t *testing.T) {
	tool := &CreateAlertDefinitionTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"definition"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	defProp := props["definition"].(map[string]interface{})
	assert.Equal(t, "object", defProp["type"])
}

func TestUpdateAlertDefinitionTool_InputSchema(t *testing.T) {
	tool := &UpdateAlertDefinitionTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id", "definition"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])

	defProp := props["definition"].(map[string]interface{})
	assert.Equal(t, "object", defProp["type"])
}

func TestDeleteAlertDefinitionTool_InputSchema(t *testing.T) {
	tool := &DeleteAlertDefinitionTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

// --- dry-run validation against the live v3 API contract ---
// These cases mirror real HTTP 400s returned by the API (2026-07-17 demo
// session): the validator must reject what the API rejects and accept what
// the API accepts.

func validDemoRatioDef() map[string]interface{} {
	return map[string]interface{}{
		"name":    "[demo] payment-service error ratio",
		"type":    "logs_ratio_threshold",
		"enabled": true,
		"logs_ratio_threshold": map[string]interface{}{
			"condition_type": "more_than_or_unspecified",
			"group_by_for":   "both_or_unspecified",
			"numerator": map[string]interface{}{
				"simple_filter": map[string]interface{}{"lucene_query": "subsystemname:payment-service AND error"},
			},
			"numerator_alias": "errors",
			"denominator": map[string]interface{}{
				"simple_filter": map[string]interface{}{"lucene_query": "subsystemname:payment-service"},
			},
			"denominator_alias": "total",
			"rules": []interface{}{
				map[string]interface{}{
					"condition": map[string]interface{}{
						"threshold":      0.1,
						"time_window":    map[string]interface{}{"logs_ratio_time_window_specific_value": "minutes_5_or_unspecified"},
						"condition_type": "more_than_or_unspecified",
					},
					"override": map[string]interface{}{"priority": "p1"},
				},
			},
		},
	}
}

func TestValidateAlertDefinition_AcceptsV3Types(t *testing.T) {
	result := validateAlertDefinitionConfig(validDemoRatioDef())
	assert.True(t, result.Valid, "logs_ratio_threshold is a valid v3 type; errors: %v", result.Errors)
}

func TestValidateAlertDefinition_RejectsLegacyTypeNames(t *testing.T) {
	def := validDemoRatioDef()
	def["type"] = "logs_ratio" // v2 name — live API rejects it
	result := validateAlertDefinitionConfig(def)
	assert.False(t, result.Valid)
	assert.Contains(t, joinStrings(result.Errors, "\n"), "logs_ratio_threshold")
}

func TestValidateAlertDefinition_RejectsUppercasePriority(t *testing.T) {
	def := validDemoRatioDef()
	cfg := def["logs_ratio_threshold"].(map[string]interface{})
	rules := cfg["rules"].([]interface{})
	delete(rules[0].(map[string]interface{}), "override")
	def["priority"] = "P1" // live API wants lowercase p1
	result := validateAlertDefinitionConfig(def)
	assert.False(t, result.Valid)
	assert.Contains(t, joinStrings(result.Errors, "\n"), "p1")
}

func TestValidateAlertDefinition_RejectsPriorityWithOverrides(t *testing.T) {
	def := validDemoRatioDef()
	def["priority"] = "p2" // rules[0].override is set -> API 400s
	result := validateAlertDefinitionConfig(def)
	assert.False(t, result.Valid)
	assert.Contains(t, joinStrings(result.Errors, "\n"), "override")
}

func TestValidateAlertDefinition_RequiresMatchingTypeConfig(t *testing.T) {
	def := validDemoRatioDef()
	delete(def, "logs_ratio_threshold") // type says logs_ratio_threshold but no config
	result := validateAlertDefinitionConfig(def)
	assert.False(t, result.Valid)
	assert.Contains(t, joinStrings(result.Errors, "\n"), "logs_ratio_threshold")
}
