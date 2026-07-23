package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
