package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

func aggLogs(aggs, groupBys int) map[string]interface{} {
	a := make([]interface{}, aggs)
	for i := range a {
		a[i] = map[string]interface{}{"count": map[string]interface{}{}}
	}
	g := make([]interface{}, groupBys)
	for i := range g {
		g[i] = map[string]interface{}{"keypath": []interface{}{"applicationname"}, "scope": "label"}
	}
	return map[string]interface{}{"aggregations": a, "group_bys": g}
}

func TestRecommendWidgetType(t *testing.T) {
	cases := []struct {
		name string
		logs map[string]interface{}
		want string
	}{
		{"raw logs -> table", map[string]interface{}{}, "data_table"},
		{"scalar agg no groupby -> gauge", aggLogs(1, 0), "gauge"},
		{"single breakdown -> bar", aggLogs(1, 1), "bar_chart"},
		{"multi groupby -> table", aggLogs(1, 2), "data_table"},
		{"multi agg -> table", aggLogs(2, 1), "data_table"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := recommendWidgetType(tc.logs)
			assert.Equal(t, tc.want, got)
			assert.NotEmpty(t, reason)
		})
	}
	got, _ := recommendWidgetType(nil)
	assert.Equal(t, "", got)
}

func TestVizAdvisor_Advise(t *testing.T) {
	// line_chart for a single breakdown is acceptable -> no note.
	a := newVizAdvisor()
	a.advise("line_chart", aggLogs(1, 1))
	assert.Empty(t, a.notes)

	// Clear mismatch: raw logs in a gauge -> recommend data_table, non-destructive.
	a = newVizAdvisor()
	a.advise("gauge", map[string]interface{}{})
	assert.NotEmpty(t, a.notes)
	assert.Contains(t, a.notes[0], "data_table")

	// Clear mismatch: scalar in a pie -> recommend gauge, non-destructive.
	a = newVizAdvisor()
	a.advise("pie_chart", aggLogs(1, 0))
	assert.NotEmpty(t, a.notes)
	assert.Contains(t, a.notes[0], "gauge")

	// Empty recommendation (nil logs) -> no note.
	a = newVizAdvisor()
	a.advise("gauge", nil)
	assert.Empty(t, a.notes)

	// Acceptable same-type pairs -> no note.
	a = newVizAdvisor()
	a.advise("gauge", aggLogs(1, 0))
	a.advise("data_table", aggLogs(2, 1))
	assert.Empty(t, a.notes)
}

func TestAdviseWidgetType_LineChartExempt(t *testing.T) {
	advisor := newVizAdvisor()
	// A line_chart whose query would otherwise recommend data_table (two
	// aggregations) must still get no recommendation — line charts are
	// intentionally exempt because their time axis validates any shape.
	definition := map[string]interface{}{
		"line_chart": map[string]interface{}{
			"query_definitions": []interface{}{
				map[string]interface{}{
					"query": map[string]interface{}{
						"logs": map[string]interface{}{
							"aggregations": []interface{}{
								map[string]interface{}{"count": map[string]interface{}{}},
								map[string]interface{}{"count": map[string]interface{}{}},
							},
						},
					},
				},
			},
		},
	}
	adviseWidgetType(definition, advisor)
	assert.Empty(t, advisor.notes)
}

func TestEnsureRequiredDashboardFields_VizRecommendation(t *testing.T) {
	advisor := newVizAdvisor()
	// A gauge widget whose query is raw logs (no aggregation) -> mismatch note.
	layout := map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{
						"gauge": map[string]interface{}{
							"query": map[string]interface{}{"logs": map[string]interface{}{}},
						},
					},
				}},
			}},
		}},
	}
	ensureRequiredDashboardFields(layout, nil, advisor)
	assert.NotEmpty(t, advisor.notes)
	assert.Contains(t, advisor.notes[0], "data_table")
}

func TestCreateDashboard_SurfacesVizRecommendation(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{"dashboard_id": "d1"})
	tool := NewCreateDashboardTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	// A gauge widget over raw logs -> a recommendation to use a data_table.
	layout := map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{
						"gauge": map[string]interface{}{
							"query": map[string]interface{}{"logs": map[string]interface{}{}},
						},
					},
				}},
			}},
		}},
	}
	result, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": layout})
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, result), "data_table")
}
