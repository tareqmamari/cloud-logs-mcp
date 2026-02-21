package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test helper functions for dashboard field validation

func TestEnsureQueryFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "adds filters to logs query",
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"lucene_query": map[string]interface{}{
						"value": "*",
					},
				},
			},
			expected: map[string]interface{}{
				"logs": map[string]interface{}{
					"lucene_query": map[string]interface{}{
						"value": "*",
					},
					"filters": []interface{}{},
				},
			},
		},
		{
			name: "preserves existing filters",
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"filters": []interface{}{
						map[string]interface{}{"field": "severity"},
					},
				},
			},
			expected: map[string]interface{}{
				"logs": map[string]interface{}{
					"filters": []interface{}{
						map[string]interface{}{"field": "severity"},
					},
				},
			},
		},
		{
			name: "handles dataprime query",
			input: map[string]interface{}{
				"dataprime": map[string]interface{}{
					"dataprime_query": map[string]interface{}{
						"text": "source logs",
					},
				},
			},
			expected: map[string]interface{}{
				"dataprime": map[string]interface{}{
					"dataprime_query": map[string]interface{}{
						"text": "source logs",
					},
					"filters": []interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureQueryFilters(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureQueryDefinitionFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "adds all required fields",
			input: map[string]interface{}{
				"query": map[string]interface{}{
					"logs": map[string]interface{}{},
				},
			},
			expected: map[string]interface{}{
				"query": map[string]interface{}{
					"logs": map[string]interface{}{
						"filters": []interface{}{},
					},
				},
				"is_visible":     true,
				"data_mode_type": "high_unspecified",
				"scale_type":     "linear",
				"unit":           "unspecified",
				"resolution": map[string]interface{}{
					"buckets_presented": 96,
				},
			},
		},
		{
			name: "preserves existing values",
			input: map[string]interface{}{
				"data_mode_type": "archive",
				"scale_type":     "logarithmic",
				"unit":           "bytes",
				"query": map[string]interface{}{
					"logs": map[string]interface{}{
						"filters": []interface{}{},
					},
				},
			},
			expected: map[string]interface{}{
				"is_visible":     true,
				"data_mode_type": "archive",
				"scale_type":     "logarithmic",
				"unit":           "bytes",
				"query": map[string]interface{}{
					"logs": map[string]interface{}{
						"filters": []interface{}{},
					},
				},
				"resolution": map[string]interface{}{
					"buckets_presented": 96,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureQueryDefinitionFields(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureLineChartFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "adds legend, tooltip, and stacked_line",
			input: map[string]interface{}{
				"query_definitions": []interface{}{
					map[string]interface{}{
						"query": map[string]interface{}{
							"logs": map[string]interface{}{},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"query_definitions": []interface{}{
					map[string]interface{}{
						"query": map[string]interface{}{
							"logs": map[string]interface{}{
								"filters": []interface{}{},
							},
						},
						"is_visible":     true,
						"data_mode_type": "high_unspecified",
						"scale_type":     "linear",
						"unit":           "unspecified",
						"resolution": map[string]interface{}{
							"buckets_presented": 96,
						},
					},
				},
				"legend": map[string]interface{}{
					"is_visible":     true,
					"group_by_query": true,
				},
				"tooltip": map[string]interface{}{
					"type":        "all",
					"show_labels": false,
				},
				"stacked_line": "unspecified",
			},
		},
		{
			name: "preserves existing legend and tooltip",
			input: map[string]interface{}{
				"legend": map[string]interface{}{
					"is_visible": false,
				},
				"tooltip": map[string]interface{}{
					"type": "single",
				},
				"query_definitions": []interface{}{},
			},
			expected: map[string]interface{}{
				"legend": map[string]interface{}{
					"is_visible": false,
				},
				"tooltip": map[string]interface{}{
					"type": "single",
				},
				"query_definitions": []interface{}{},
				"stacked_line":      "unspecified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureLineChartFields(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureChartFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:  "adds data_mode_type to chart",
			input: map[string]interface{}{},
			expected: map[string]interface{}{
				"data_mode_type": "high_unspecified",
			},
		},
		{
			name: "preserves existing data_mode_type",
			input: map[string]interface{}{
				"data_mode_type": "archive",
			},
			expected: map[string]interface{}{
				"data_mode_type": "archive",
			},
		},
		{
			name: "adds filters to query",
			input: map[string]interface{}{
				"query": map[string]interface{}{
					"logs": map[string]interface{}{},
				},
			},
			expected: map[string]interface{}{
				"data_mode_type": "high_unspecified",
				"query": map[string]interface{}{
					"logs": map[string]interface{}{
						"filters": []interface{}{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureChartFields(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureDataTableFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:  "adds data_mode_type to data table",
			input: map[string]interface{}{},
			expected: map[string]interface{}{
				"data_mode_type": "high_unspecified",
			},
		},
		{
			name: "preserves existing data_mode_type",
			input: map[string]interface{}{
				"data_mode_type": "archive",
			},
			expected: map[string]interface{}{
				"data_mode_type": "archive",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureDataTableFields(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureGaugeFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:  "adds data_mode_type to gauge",
			input: map[string]interface{}{},
			expected: map[string]interface{}{
				"data_mode_type": "high_unspecified",
			},
		},
		{
			name: "preserves existing data_mode_type",
			input: map[string]interface{}{
				"data_mode_type": "archive",
			},
			expected: map[string]interface{}{
				"data_mode_type": "archive",
			},
		},
		{
			name: "adds filters to query",
			input: map[string]interface{}{
				"query": map[string]interface{}{
					"logs": map[string]interface{}{},
				},
			},
			expected: map[string]interface{}{
				"data_mode_type": "high_unspecified",
				"query": map[string]interface{}{
					"logs": map[string]interface{}{
						"filters": []interface{}{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureGaugeFields(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestEnsureRequiredDashboardFields(t *testing.T) {
	t.Run("complete dashboard structure", func(t *testing.T) {
		layout := map[string]interface{}{
			"sections": []interface{}{
				map[string]interface{}{
					"rows": []interface{}{
						map[string]interface{}{
							"widgets": []interface{}{
								map[string]interface{}{
									"definition": map[string]interface{}{
										"line_chart": map[string]interface{}{
											"query_definitions": []interface{}{
												map[string]interface{}{
													"query": map[string]interface{}{
														"logs": map[string]interface{}{
															"lucene_query": map[string]interface{}{
																"value": "*",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ensureRequiredDashboardFields(layout)

		// Verify sections exist
		sections, ok := layout["sections"].([]interface{})
		require.True(t, ok)
		require.Len(t, sections, 1)

		// Verify rows exist
		section := sections[0].(map[string]interface{})
		rows, ok := section["rows"].([]interface{})
		require.True(t, ok)
		require.Len(t, rows, 1)

		// Verify widgets exist
		row := rows[0].(map[string]interface{})
		widgets, ok := row["widgets"].([]interface{})
		require.True(t, ok)
		require.Len(t, widgets, 1)

		// Verify widget has required fields
		widget := widgets[0].(map[string]interface{})
		appearance, ok := widget["appearance"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 0, appearance["width"])

		// Verify line_chart has required fields
		definition := widget["definition"].(map[string]interface{})
		lineChart := definition["line_chart"].(map[string]interface{})
		
		assert.NotNil(t, lineChart["legend"])
		assert.NotNil(t, lineChart["tooltip"])
		assert.Equal(t, "unspecified", lineChart["stacked_line"])

		// Verify query definition has required fields
		queryDefs := lineChart["query_definitions"].([]interface{})
		queryDef := queryDefs[0].(map[string]interface{})
		
		assert.Equal(t, "high_unspecified", queryDef["data_mode_type"])
		assert.Equal(t, "linear", queryDef["scale_type"])
		assert.Equal(t, "unspecified", queryDef["unit"])
		assert.NotNil(t, queryDef["resolution"])

		// Verify query has filters
		query := queryDef["query"].(map[string]interface{})
		logs := query["logs"].(map[string]interface{})
		filters, ok := logs["filters"].([]interface{})
		require.True(t, ok)
		assert.NotNil(t, filters)
	})

	t.Run("handles empty layout", func(t *testing.T) {
		layout := map[string]interface{}{}
		ensureRequiredDashboardFields(layout)
		// Should not panic
	})

	t.Run("handles nil sections", func(t *testing.T) {
		layout := map[string]interface{}{
			"sections": nil,
		}
		ensureRequiredDashboardFields(layout)
		// Should not panic
	})
}
