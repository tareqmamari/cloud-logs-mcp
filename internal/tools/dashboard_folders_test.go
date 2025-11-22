package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListDashboardFoldersTool_InputSchema(t *testing.T) {
	tool := &ListDashboardFoldersTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	if req, ok := schema["required"]; ok {
		assert.Empty(t, req)
	}
}

func TestGetDashboardFolderTool_InputSchema(t *testing.T) {
	tool := &GetDashboardFolderTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"folder_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["folder_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestCreateDashboardFolderTool_InputSchema(t *testing.T) {
	tool := &CreateDashboardFolderTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"name"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	nameProp := props["name"].(map[string]interface{})
	assert.Equal(t, "string", nameProp["type"])

	parentProp := props["parent_id"].(map[string]interface{})
	assert.Equal(t, "string", parentProp["type"])
}

func TestUpdateDashboardFolderTool_InputSchema(t *testing.T) {
	tool := &UpdateDashboardFolderTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"folder_id", "name"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["folder_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])

	nameProp := props["name"].(map[string]interface{})
	assert.Equal(t, "string", nameProp["type"])
}

func TestDeleteDashboardFolderTool_InputSchema(t *testing.T) {
	tool := &DeleteDashboardFolderTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"folder_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["folder_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestMoveDashboardToFolderTool_InputSchema(t *testing.T) {
	tool := &MoveDashboardToFolderTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"dashboard_id", "folder_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	dashIDProp := props["dashboard_id"].(map[string]interface{})
	assert.Equal(t, "string", dashIDProp["type"])

	folderIDProp := props["folder_id"].(map[string]interface{})
	assert.Equal(t, "string", folderIDProp["type"])
}
