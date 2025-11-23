package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngestLogsTool_InputSchema(t *testing.T) {
	tool := &IngestLogsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"logs"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	logsProp := props["logs"].(map[string]interface{})
	assert.Equal(t, "array", logsProp["type"])
}
