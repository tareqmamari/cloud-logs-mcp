package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func logsAgg(aggType, field string, groupBys []string, lucene string) map[string]interface{} {
	agg := map[string]interface{}{aggType: map[string]interface{}{}}
	if field != "" {
		agg[aggType] = map[string]interface{}{"observation_field": map[string]interface{}{"keypath": []interface{}{field}}}
	}
	gb := make([]interface{}, len(groupBys))
	for i, g := range groupBys {
		gb[i] = map[string]interface{}{"keypath": []interface{}{g}}
	}
	logs := map[string]interface{}{"aggregations": []interface{}{agg}, "group_bys": gb}
	if lucene != "" {
		logs["lucene_query"] = map[string]interface{}{"value": lucene}
	}
	return logs
}

func TestWidgetAggregation(t *testing.T) {
	// count by application
	a := widgetAggregation(logsAgg("count", "", []string{"applicationname"}, "severity:error"))
	assert.True(t, a.eligible)
	assert.Equal(t, "count", a.aggType)
	assert.Equal(t, "", a.sourceField)
	assert.Equal(t, []string{"applicationname"}, a.labels)
	assert.Equal(t, "severity:error", a.lucene)

	// average maps to avg, carries source field
	a = widgetAggregation(logsAgg("average", "duration", nil, ""))
	assert.True(t, a.eligible)
	assert.Equal(t, "avg", a.aggType)
	assert.Equal(t, "duration", a.sourceField)

	// count_distinct is ineligible
	assert.False(t, widgetAggregation(logsAgg("count_distinct", "user", nil, "")).eligible)
	// no aggregation is ineligible
	assert.False(t, widgetAggregation(map[string]interface{}{}).eligible)
	// nil is ineligible
	assert.False(t, widgetAggregation(nil).eligible)
}
