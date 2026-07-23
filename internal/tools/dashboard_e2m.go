// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file surfaces advisory E2M (events-to-metrics) recommendations for
// dashboard aggregation widgets. It never creates or switches anything.
package tools

import (
	"sort"
	"strings"
)

// widgetAgg is the normalized single aggregation a widget computes, expressed
// in E2M vocabulary so it can be matched against E2M definitions.
type widgetAgg struct {
	aggType     string   // count | sum | avg | min | max
	sourceField string   // measured field keypath joined by "."; "" for count
	labels      []string // group-by keypaths joined by ".", sorted
	lucene      string   // widget lucene filter; "" if none
	eligible    bool     // false unless exactly one count/sum/avg/min/max aggregation
}

// widgetAggTypes maps a dashboard logs-aggregation key to its E2M agg_type.
// count_distinct and percentile are intentionally absent — E2M has no equivalent.
var widgetAggTypes = map[string]string{
	"count": "count", "sum": "sum", "average": "avg", "min": "min", "max": "max",
}

// widgetAggregation normalizes a widget's logs query into a widgetAgg. It is a
// pure read-only function; a nil query or any shape without exactly one
// supported aggregation yields eligible=false.
func widgetAggregation(logs map[string]interface{}) widgetAgg {
	if logs == nil {
		return widgetAgg{}
	}
	aggObj, ok := singleAggregation(logs)
	if !ok {
		return widgetAgg{}
	}
	var key string
	var body map[string]interface{}
	for k, v := range aggObj {
		key = k
		body, _ = v.(map[string]interface{})
	}
	e2mType, supported := widgetAggTypes[key]
	if !supported {
		return widgetAgg{}
	}
	return widgetAgg{
		aggType:     e2mType,
		sourceField: observationKeypath(body),
		labels:      widgetGroupByLabels(logs),
		lucene:      luceneValue(logs),
		eligible:    true,
	}
}

// singleAggregation returns the sole aggregation object on a logs query,
// looking at the array form (aggregations[]) and the singular forms
// (aggregation / logs_aggregation). Returns ok=false unless exactly one exists.
func singleAggregation(logs map[string]interface{}) (map[string]interface{}, bool) {
	if arr, ok := logs["aggregations"].([]interface{}); ok {
		if len(arr) != 1 {
			return nil, false
		}
		m, ok := arr[0].(map[string]interface{})
		return m, ok
	}
	for _, key := range []string{"aggregation", "logs_aggregation"} {
		if m, ok := logs[key].(map[string]interface{}); ok {
			return m, true
		}
	}
	return nil, false
}

// observationKeypath joins an aggregation body's observation_field keypath by
// ".", or returns "" (e.g. for count, which has no field).
func observationKeypath(body map[string]interface{}) string {
	of, ok := body["observation_field"].(map[string]interface{})
	if !ok {
		return ""
	}
	return joinKeypath(of["keypath"])
}

// widgetGroupByLabels collects the group-by keypaths from either group_bys or
// group_names_fields, joined by "." and sorted for order-independent matching.
func widgetGroupByLabels(logs map[string]interface{}) []string {
	var out []string
	for _, key := range []string{"group_bys", "group_names_fields"} {
		arr, ok := logs[key].([]interface{})
		if !ok {
			continue
		}
		for _, g := range arr {
			if m, ok := g.(map[string]interface{}); ok {
				if kp := joinKeypath(m["keypath"]); kp != "" {
					out = append(out, kp)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// joinKeypath renders a []interface{} of strings as a "."-joined path.
func joinKeypath(v interface{}) string {
	arr, ok := v.([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(arr))
	for _, p := range arr {
		if s, ok := p.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ".")
}

// luceneValue reads logs.lucene_query.value or "".
func luceneValue(logs map[string]interface{}) string {
	lq, ok := logs["lucene_query"].(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := lq["value"].(string)
	return s
}
