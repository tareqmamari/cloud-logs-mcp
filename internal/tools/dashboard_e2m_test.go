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

func e2mDef(luceneFilter, srcField, aggType, metricName string, labelSrcFields []string) map[string]interface{} {
	labels := make([]interface{}, len(labelSrcFields))
	for i, f := range labelSrcFields {
		labels[i] = map[string]interface{}{"source_field": f, "target_label": f}
	}
	return map[string]interface{}{
		"logs_query":    map[string]interface{}{"lucene": luceneFilter},
		"metric_labels": labels,
		"metric_fields": []interface{}{
			map[string]interface{}{
				"source_field": srcField,
				"aggregations": []interface{}{
					map[string]interface{}{"agg_type": aggType, "target_metric_name": metricName},
				},
			},
		},
	}
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

func TestMatchE2M(t *testing.T) {
	agg := widgetAgg{aggType: "count", sourceField: "", labels: []string{"applicationname"}, lucene: "severity:error", eligible: true}

	// Exact match (count, same label, broader/equal lucene).
	e2ms := []interface{}{e2mDef("severity:error", "message", "count", "error_count_total", []string{"applicationname"})}
	name, ok := matchE2M(agg, e2ms)
	assert.True(t, ok)
	assert.Equal(t, "error_count_total", name)

	// Empty E2M lucene = matches all -> still a match.
	e2ms = []interface{}{e2mDef("", "message", "count", "all_count", []string{"applicationname"})}
	_, ok = matchE2M(agg, e2ms)
	assert.True(t, ok)

	// Label-set mismatch -> no match.
	e2ms = []interface{}{e2mDef("severity:error", "message", "count", "x", []string{"subsystemname"})}
	_, ok = matchE2M(agg, e2ms)
	assert.False(t, ok)

	// agg_type mismatch -> no match.
	e2ms = []interface{}{e2mDef("severity:error", "message", "sum", "x", []string{"applicationname"})}
	_, ok = matchE2M(agg, e2ms)
	assert.False(t, ok)

	// sum requires source_field to match.
	sumAgg := widgetAgg{aggType: "sum", sourceField: "duration", labels: nil, lucene: "", eligible: true}
	_, ok = matchE2M(sumAgg, []interface{}{e2mDef("", "latency", "sum", "x", nil)})
	assert.False(t, ok)
	name, ok = matchE2M(sumAgg, []interface{}{e2mDef("", "duration", "sum", "duration_sum", nil)})
	assert.True(t, ok)
	assert.Equal(t, "duration_sum", name)
}

func TestE2MRecommendations_MatchExisting(t *testing.T) {
	// Widget: count by applicationname, filter severity:error; an existing E2M matches.
	logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
	layout := map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{
						"bar_chart": map[string]interface{}{"query": map[string]interface{}{"logs": logs}},
					},
				}},
			}},
		}},
	}
	e2ms := []interface{}{e2mDef("severity:error", "message", "count", "error_count_total", []string{"applicationname"})}

	notes := e2mRecommendations(layout, e2ms, nil)
	assert.NotEmpty(t, notes)
	assert.Contains(t, notes[0], "error_count_total")
}

func TestE2MRecommendations_ArchiveSuggestsCreate(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_low") // archive tier (helper from alert_tier_test.go)
	logs := logsAgg("count", "", []string{"applicationname"}, "applicationname:payment-service")
	layout := map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{
						"bar_chart": map[string]interface{}{"query": map[string]interface{}{"logs": logs}},
					},
				}},
			}},
		}},
	}
	// No existing E2M -> archive-tier widget should suggest creating one.
	notes := e2mRecommendations(layout, nil, session)
	assert.NotEmpty(t, notes)
	assert.Contains(t, notes[0], "create_e2m")
}
