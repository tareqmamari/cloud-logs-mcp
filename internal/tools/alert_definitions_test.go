package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
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

func TestValidateAlertDefinition_RequiresConfigLevelConditionType(t *testing.T) {
	def := validDemoRatioDef()
	cfg := def["logs_ratio_threshold"].(map[string]interface{})
	delete(cfg, "condition_type") // API 400s: logs_ratio_threshold: missing field `condition_type`
	result := validateAlertDefinitionConfig(def)
	assert.False(t, result.Valid)
	assert.Contains(t, joinStrings(result.Errors, "\n"), "condition_type")
}

// thresholdDefWithLabelFilterOp builds a logs_threshold definition whose
// simple_filter carries an application_name label filter with the given
// operation, mirroring the nesting the live API reported in the 400:
// logs_threshold.logs_filter.simple_filter.label_filters.application_name[0].operation
func thresholdDefWithLabelFilterOp(op string) map[string]interface{} {
	return map[string]interface{}{
		"name": "App filtered alert",
		"type": "logs_threshold",
		"logs_threshold": map[string]interface{}{
			"condition_type": "more_than_or_unspecified",
			"logs_filter": map[string]interface{}{
				"simple_filter": map[string]interface{}{
					"label_filters": map[string]interface{}{
						"application_name": []interface{}{
							map[string]interface{}{"value": "payment-service", "operation": op},
						},
					},
				},
			},
		},
	}
}

func labelFilterOp(def map[string]interface{}) string {
	return def["logs_threshold"].(map[string]interface{})["logs_filter"].(map[string]interface{})["simple_filter"].(map[string]interface{})["label_filters"].(map[string]interface{})["application_name"].([]interface{})[0].(map[string]interface{})["operation"].(string)
}

func TestNormalizeAlertDefinition_RewritesLabelFilterIsAlias(t *testing.T) {
	def := thresholdDefWithLabelFilterOp("is")
	notes := normalizeAlertDefinition(def)
	assert.Equal(t, "is_or_unspecified", labelFilterOp(def), "operation \"is\" must be rewritten to the API's canonical variant")
	assert.NotEmpty(t, notes, "a normalization note should record the rewrite")
	assert.Contains(t, joinStrings(notes, "\n"), "is_or_unspecified")
}

func TestNormalizeAlertDefinition_RewritesContainsAlias(t *testing.T) {
	def := thresholdDefWithLabelFilterOp("contains")
	normalizeAlertDefinition(def)
	assert.Equal(t, "includes", labelFilterOp(def))
}

func TestNormalizeAlertDefinition_LeavesCanonicalAndUnknownUntouched(t *testing.T) {
	canonical := thresholdDefWithLabelFilterOp("starts_with")
	notes := normalizeAlertDefinition(canonical)
	assert.Equal(t, "starts_with", labelFilterOp(canonical))
	assert.Empty(t, notes, "canonical values need no rewrite")

	unknown := thresholdDefWithLabelFilterOp("no_such_op")
	normalizeAlertDefinition(unknown)
	assert.Equal(t, "no_such_op", labelFilterOp(unknown), "unknown values are left for the validator to reject, not silently changed")
}

func TestValidateAlertDefinition_WarnsOnLabelFilterAlias(t *testing.T) {
	result := validateAlertDefinitionConfig(thresholdDefWithLabelFilterOp("is"))
	assert.True(t, result.Valid, "a known alias is not an error; errors: %v", result.Errors)
	assert.Contains(t, joinStrings(result.Warnings, "\n"), "is_or_unspecified")
}

func TestValidateAlertDefinition_RejectsUnknownLabelFilterOperation(t *testing.T) {
	result := validateAlertDefinitionConfig(thresholdDefWithLabelFilterOp("no_such_op"))
	assert.False(t, result.Valid)
	errs := joinStrings(result.Errors, "\n")
	assert.Contains(t, errs, "no_such_op")
	assert.Contains(t, errs, "is_or_unspecified")
}

// TestCreateAlertDefinition_SendsNormalizedOperation drives the full Execute
// path with a mock client and asserts the request body that actually reaches
// the API carries the canonical operation, reproducing the live 400 fix
// end-to-end rather than only unit-testing the helper.
func TestCreateAlertDefinition_SendsNormalizedOperation(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{"id": "abc", "name": "App filtered alert"})

	tool := NewCreateAlertDefinitionTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	_, err := tool.Execute(ctx, map[string]interface{}{
		"definition": thresholdDefWithLabelFilterOp("is"),
	})
	assert.NoError(t, err)

	sent, ok := mock.LastRequest().Body.(map[string]interface{})
	assert.True(t, ok, "request body should be the definition map")
	assert.Equal(t, "is_or_unspecified", labelFilterOp(sent),
		"the body sent to the API must carry the canonical operation, not the alias")
}
