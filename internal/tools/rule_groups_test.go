package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRuleGroupTool_Name(t *testing.T) {
	tool := &GetRuleGroupTool{}
	assert.Equal(t, "get_rule_group", tool.Name())
}

func TestGetRuleGroupTool_Description(t *testing.T) {
	tool := &GetRuleGroupTool{}
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "rule group")
}

func TestGetRuleGroupTool_InputSchema(t *testing.T) {
	tool := &GetRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
	assert.Contains(t, idProp["description"], "Rule group ID")
}

func TestListRuleGroupsTool_Name(t *testing.T) {
	tool := &ListRuleGroupsTool{}
	assert.Equal(t, "list_rule_groups", tool.Name())
}

func TestListRuleGroupsTool_Description(t *testing.T) {
	tool := &ListRuleGroupsTool{}
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "rule group")
}

func TestListRuleGroupsTool_InputSchema(t *testing.T) {
	tool := &ListRuleGroupsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])

	// List tools typically don't have required parameters
	if required, ok := schema["required"]; ok {
		assert.Empty(t, required)
	}
}

func TestCreateRuleGroupTool_Name(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	assert.Equal(t, "create_rule_group", tool.Name())
}

func TestCreateRuleGroupTool_Description(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "rule group")
}

func TestCreateRuleGroupTool_InputSchema(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"rule_group"}, schema["required"])

	props := schema["properties"].(map[string]interface{})

	// Verify rule_group property exists
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	assert.Equal(t, "object", ruleGroupProp["type"])
	assert.NotEmpty(t, ruleGroupProp["description"])

	// Verify rule_group has properties
	ruleGroupProps := ruleGroupProp["properties"].(map[string]interface{})
	assert.NotNil(t, ruleGroupProps["name"])
	assert.NotNil(t, ruleGroupProps["description"])
	assert.NotNil(t, ruleGroupProps["enabled"])
	assert.NotNil(t, ruleGroupProps["rule_matchers"])
	assert.NotNil(t, ruleGroupProps["rule_subgroups"])
}

func TestCreateRuleGroupTool_InputSchema_RuleMatchers(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	ruleGroupProps := ruleGroupProp["properties"].(map[string]interface{})

	// Verify rule_matchers structure
	ruleMatchers := ruleGroupProps["rule_matchers"].(map[string]interface{})
	assert.Equal(t, "array", ruleMatchers["type"])

	items := ruleMatchers["items"].(map[string]interface{})
	assert.Equal(t, "object", items["type"])

	// Verify oneOf structure exists
	oneOf := items["oneOf"]
	assert.NotNil(t, oneOf)

	oneOfArray := oneOf.([]interface{})
	assert.GreaterOrEqual(t, len(oneOfArray), 3, "Should have at least 3 matcher types")
}

func TestCreateRuleGroupTool_InputSchema_RuleSubgroups(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	ruleGroupProps := ruleGroupProp["properties"].(map[string]interface{})

	// Verify rule_subgroups structure
	ruleSubgroups := ruleGroupProps["rule_subgroups"].(map[string]interface{})
	assert.Equal(t, "array", ruleSubgroups["type"])

	items := ruleSubgroups["items"].(map[string]interface{})
	assert.Equal(t, "object", items["type"])

	subgroupProps := items["properties"].(map[string]interface{})
	assert.NotNil(t, subgroupProps["rules"])
	assert.NotNil(t, subgroupProps["enabled"])
	assert.NotNil(t, subgroupProps["order"])
}

func TestCreateRuleGroupTool_InputSchema_Rules(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	ruleGroupProps := ruleGroupProp["properties"].(map[string]interface{})
	ruleSubgroups := ruleGroupProps["rule_subgroups"].(map[string]interface{})
	subgroupItems := ruleSubgroups["items"].(map[string]interface{})
	subgroupProps := subgroupItems["properties"].(map[string]interface{})

	// Verify rules structure
	rules := subgroupProps["rules"].(map[string]interface{})
	assert.Equal(t, "array", rules["type"])

	ruleItems := rules["items"].(map[string]interface{})
	assert.Equal(t, "object", ruleItems["type"])

	ruleProps := ruleItems["properties"].(map[string]interface{})
	assert.NotNil(t, ruleProps["name"])
	assert.NotNil(t, ruleProps["description"])
	assert.NotNil(t, ruleProps["enabled"])
	assert.NotNil(t, ruleProps["order"])
	assert.NotNil(t, ruleProps["source_field"])
	assert.NotNil(t, ruleProps["parameters"])
}

func TestCreateRuleGroupTool_InputSchema_RuleParameters(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	ruleGroupProps := ruleGroupProp["properties"].(map[string]interface{})
	ruleSubgroups := ruleGroupProps["rule_subgroups"].(map[string]interface{})
	subgroupItems := ruleSubgroups["items"].(map[string]interface{})
	subgroupProps := subgroupItems["properties"].(map[string]interface{})
	rules := subgroupProps["rules"].(map[string]interface{})
	ruleItems := rules["items"].(map[string]interface{})
	ruleProps := ruleItems["properties"].(map[string]interface{})

	// Verify parameters field exists
	parameters := ruleProps["parameters"]
	assert.NotNil(t, parameters)

	parametersMap := parameters.(map[string]interface{})
	assert.Equal(t, "object", parametersMap["type"])
	assert.NotEmpty(t, parametersMap["description"])
}

func TestCreateRuleGroupTool_InputSchema_Examples(t *testing.T) {
	tool := &CreateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})

	// Verify examples exist
	examples := schema["examples"]
	assert.NotNil(t, examples)

	examplesArray := examples.([]interface{})
	assert.NotEmpty(t, examplesArray)

	// Verify first example structure
	firstExample := examplesArray[0].(map[string]interface{})
	ruleGroup := firstExample["rule_group"].(map[string]interface{})

	assert.NotEmpty(t, ruleGroup["name"])
	assert.NotEmpty(t, ruleGroup["description"])
	assert.NotNil(t, ruleGroup["enabled"])
	assert.NotNil(t, ruleGroup["rule_subgroups"])
}

func TestUpdateRuleGroupTool_Name(t *testing.T) {
	tool := &UpdateRuleGroupTool{}
	assert.Equal(t, "update_rule_group", tool.Name())
}

func TestUpdateRuleGroupTool_Description(t *testing.T) {
	tool := &UpdateRuleGroupTool{}
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "rule group")
}

func TestUpdateRuleGroupTool_InputSchema(t *testing.T) {
	tool := &UpdateRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])

	required := schema["required"].([]string)
	assert.Contains(t, required, "id")
	assert.Contains(t, required, "rule_group")

	props := schema["properties"].(map[string]interface{})

	// Verify id property
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])

	// Verify rule_group property
	ruleGroupProp := props["rule_group"].(map[string]interface{})
	assert.Equal(t, "object", ruleGroupProp["type"])
}

func TestDeleteRuleGroupTool_Name(t *testing.T) {
	tool := &DeleteRuleGroupTool{}
	assert.Equal(t, "delete_rule_group", tool.Name())
}

func TestDeleteRuleGroupTool_Description(t *testing.T) {
	tool := &DeleteRuleGroupTool{}
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "rule group")
}

func TestDeleteRuleGroupTool_InputSchema(t *testing.T) {
	tool := &DeleteRuleGroupTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestRuleGroupTools_AllHaveCorrectInterface(t *testing.T) {
	tools := []Tool{
		&GetRuleGroupTool{},
		&ListRuleGroupsTool{},
		&CreateRuleGroupTool{},
		&UpdateRuleGroupTool{},
		&DeleteRuleGroupTool{},
	}

	for _, tool := range tools {
		// Verify each tool implements the Tool interface
		assert.NotEmpty(t, tool.Name())
		assert.NotEmpty(t, tool.Description())
		assert.NotNil(t, tool.InputSchema())
	}
}

func TestRuleGroupTools_UniqueNames(t *testing.T) {
	tools := []Tool{
		&GetRuleGroupTool{},
		&ListRuleGroupsTool{},
		&CreateRuleGroupTool{},
		&UpdateRuleGroupTool{},
		&DeleteRuleGroupTool{},
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		name := tool.Name()
		assert.False(t, names[name], "Duplicate tool name: %s", name)
		names[name] = true
	}
}
