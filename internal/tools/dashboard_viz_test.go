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
	// Unspecified type -> adopt recommendation (auto-build).
	a := newVizAdvisor()
	newType, changed := a.advise("", aggLogs(1, 0))
	assert.Equal(t, "gauge", newType)
	assert.True(t, changed)

	// line_chart for a single breakdown is acceptable -> no change, no note.
	a = newVizAdvisor()
	newType, changed = a.advise("line_chart", aggLogs(1, 1))
	assert.Equal(t, "line_chart", newType)
	assert.False(t, changed)
	assert.Empty(t, a.notes)

	// Clear mismatch: raw logs in a gauge -> recommend table, do NOT rewrite.
	a = newVizAdvisor()
	newType, changed = a.advise("gauge", map[string]interface{}{})
	assert.Equal(t, "gauge", newType)
	assert.False(t, changed)
	assert.NotEmpty(t, a.notes)
	assert.Contains(t, a.notes[0], "data_table")

	// Clear mismatch: scalar in a pie -> recommend gauge, do NOT rewrite.
	a = newVizAdvisor()
	_, changed = a.advise("pie_chart", aggLogs(1, 0))
	assert.False(t, changed)
	assert.NotEmpty(t, a.notes)
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
