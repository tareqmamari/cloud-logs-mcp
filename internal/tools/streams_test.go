package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListStreamsTool_InputSchema(t *testing.T) {
	tool := &ListStreamsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	if req, ok := schema["required"]; ok {
		assert.Empty(t, req)
	}
}

func TestGetStreamTool_InputSchema(t *testing.T) {
	tool := &GetStreamTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"stream_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["stream_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestCreateStreamTool_InputSchema(t *testing.T) {
	tool := &CreateStreamTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"name", "dpxl_expression"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	nameProp := props["name"].(map[string]interface{})
	assert.Equal(t, "string", nameProp["type"])

	dpxlProp := props["dpxl_expression"].(map[string]interface{})
	assert.Equal(t, "string", dpxlProp["type"])
}

func TestUpdateStreamTool_InputSchema(t *testing.T) {
	tool := &UpdateStreamTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"stream_id", "name", "dpxl_expression", "compression_type", "ibm_event_streams"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["stream_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}

func TestDeleteStreamTool_InputSchema(t *testing.T) {
	tool := &DeleteStreamTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"stream_id"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	idProp := props["stream_id"].(map[string]interface{})
	assert.Equal(t, "string", idProp["type"])
}
