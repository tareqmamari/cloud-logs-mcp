package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateE2MTool_InputSchema_DocumentsAggregations(t *testing.T) {
	tool := &CreateE2MTool{}
	schema := tool.InputSchema().(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	e2m := props["e2m"].(map[string]interface{})["properties"].(map[string]interface{})

	mf := e2m["metric_fields"].(map[string]interface{})
	assert.Equal(t, "array", mf["type"])
	items := mf["items"].(map[string]interface{})["properties"].(map[string]interface{})
	// metric_fields items must document source_field, target_base_metric_name, aggregations
	assert.Contains(t, items, "source_field")
	assert.Contains(t, items, "target_base_metric_name")
	aggs := items["aggregations"].(map[string]interface{})
	aggItems := aggs["items"].(map[string]interface{})["properties"].(map[string]interface{})
	assert.Contains(t, aggItems, "agg_type")
	assert.Contains(t, aggItems, "target_metric_name")

	ml := e2m["metric_labels"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
	assert.Contains(t, ml, "source_field")
	assert.Contains(t, ml, "target_label")
}
