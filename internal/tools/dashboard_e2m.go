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

// matchE2M returns the target_metric_name of the first existing E2M whose
// definition computes the same metric as the widget aggregation, or ok=false.
func matchE2M(agg widgetAgg, e2ms []interface{}) (string, bool) {
	if !agg.eligible {
		return "", false
	}
	for _, e := range e2ms {
		e2m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if !e2mFilterCoversWidget(e2m, agg.lucene) {
			continue
		}
		if !stringSetEqual(e2mLabelSourceFields(e2m), agg.labels) {
			continue
		}
		if name, ok := e2mAggMetricName(e2m, agg); ok {
			return name, true
		}
	}
	return "", false
}

// e2mFilterCoversWidget reports whether the E2M's logs filter is equal-or-broader
// than the widget's: an empty E2M lucene matches all logs; otherwise require
// exact string equality.
func e2mFilterCoversWidget(e2m map[string]interface{}, widgetLucene string) bool {
	lq, _ := e2m["logs_query"].(map[string]interface{})
	e2mLucene, _ := lq["lucene"].(string)
	return e2mLucene == "" || e2mLucene == widgetLucene
}

// e2mLabelSourceFields returns the set of metric_labels[].source_field values.
func e2mLabelSourceFields(e2m map[string]interface{}) []string {
	arr, _ := e2m["metric_labels"].([]interface{})
	var out []string
	for _, l := range arr {
		if m, ok := l.(map[string]interface{}); ok {
			if s, ok := m["source_field"].(string); ok && s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// e2mAggMetricName finds an aggregation on the E2M matching the widget's
// agg_type (and measured field, unless count) and returns its target_metric_name.
func e2mAggMetricName(e2m map[string]interface{}, agg widgetAgg) (string, bool) {
	fields, _ := e2m["metric_fields"].([]interface{})
	for _, f := range fields {
		mf, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		if agg.aggType != "count" {
			src, _ := mf["source_field"].(string)
			if !strings.EqualFold(src, agg.sourceField) {
				continue
			}
		}
		aggs, _ := mf["aggregations"].([]interface{})
		for _, a := range aggs {
			am, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := am["agg_type"].(string); t == agg.aggType {
				if name, ok := am["target_metric_name"].(string); ok && name != "" {
					return name, true
				}
			}
		}
	}
	return "", false
}

// stringSetEqual reports set equality (order-independent) of two string slices.
func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}
