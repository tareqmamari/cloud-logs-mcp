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
