package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAlertTool_InputSchema(t *testing.T) {
	tool := &GetAlertTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestListAlertsTool_InputSchema(t *testing.T) {
	tool := &ListAlertsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])

	props := schema["properties"].(map[string]interface{})
	limitProp := props["limit"].(map[string]interface{})
	assert.Equal(t, "integer", limitProp["type"])

	cursorProp := props["cursor"].(map[string]interface{})
	assert.Equal(t, "string", cursorProp["type"])
}

func TestCreateAlertTool_InputSchema(t *testing.T) {
	tool := &CreateAlertTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"alert"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	alertProp := props["alert"].(map[string]interface{})
	assert.Equal(t, "object", alertProp["type"])
}

func TestDeleteAlertTool_InputSchema(t *testing.T) {
	tool := &DeleteAlertTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}
