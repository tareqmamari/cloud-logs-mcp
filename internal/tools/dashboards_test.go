package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDashboardTool_InputSchema(t *testing.T) {
	tool := &GetDashboardTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"dashboard_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["dashboard_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestCreateDashboardTool_InputSchema(t *testing.T) {
	tool := &CreateDashboardTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"name", "layout"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	nameProp := props["name"].(map[string]interface{})
	assert.Equal(t, "string", nameProp["type"])

	layoutProp := props["layout"].(map[string]interface{})
	assert.Equal(t, "object", layoutProp["type"])
}

func TestUpdateDashboardTool_InputSchema(t *testing.T) {
	tool := &UpdateDashboardTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"dashboard_id", "name", "layout"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["dashboard_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestDeleteDashboardTool_InputSchema(t *testing.T) {
	tool := &DeleteDashboardTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"dashboard_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["dashboard_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}
