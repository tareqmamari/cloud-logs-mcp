// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file classifies the appropriate visualization for a dashboard widget
// from the shape of its logs query (aggregations and group-bys).
package tools

import "fmt"

// logsAggregationCount returns the number of aggregations on a widget logs query.
func logsAggregationCount(logs map[string]interface{}) int {
	aggs, _ := logs["aggregations"].([]interface{})
	return len(aggs)
}

// logsGroupByCount returns the number of group-by dimensions on a logs query.
func logsGroupByCount(logs map[string]interface{}) int {
	groupBys, _ := logs["group_bys"].([]interface{})
	return len(groupBys)
}

// recommendWidgetType recommends a visualization type from the query shape.
// It is deliberately conservative and returns only the types it can infer
// unambiguously: data_table (raw or multi-dimensional), gauge (single scalar),
// or bar_chart (single categorical breakdown). It never returns line_chart or
// pie_chart, which are legitimate caller-chosen variants of a bar/breakdown
// that the query shape alone cannot distinguish. Returns ("", "") for a nil
// query.
func recommendWidgetType(logs map[string]interface{}) (widgetType string, reason string) {
	if logs == nil {
		return "", ""
	}
	aggs := logsAggregationCount(logs)
	groupBys := logsGroupByCount(logs)

	switch {
	case aggs == 0:
		return "data_table", "query has no aggregation; a table shows raw log rows"
	case aggs > 1 || groupBys > 1:
		return "data_table", "query has multiple aggregations/dimensions; a table shows them all"
	case groupBys == 0:
		return "gauge", "single aggregation with no breakdown; a gauge shows the scalar value"
	default:
		return "bar_chart", "single aggregation broken down by one dimension; a bar chart compares categories"
	}
}

// vizAdvisor decides each widget's visualization type from its query shape and
// accumulates human-readable notes describing auto-adopted types and
// non-destructive recommendations, surfaced to the caller as
// _viz_recommendations.
type vizAdvisor struct {
	notes []string
}

func newVizAdvisor() *vizAdvisor { return &vizAdvisor{} }

// vizAcceptableFor lists, for each recommended type, the caller-chosen types
// that are considered a fine fit (so no recommendation is emitted).
var vizAcceptableFor = map[string]map[string]bool{
	// A single breakdown is validly shown as a bar, line, or pie.
	"bar_chart": {"bar_chart": true, "line_chart": true, "pie_chart": true},
	// A scalar is validly a gauge (single stat).
	"gauge": {"gauge": true},
	// Raw / multi-dimensional data belongs in a table.
	"data_table": {"data_table": true},
}

// vizAcceptable reports whether the caller's type is an acceptable fit for the
// recommended type.
func vizAcceptable(currentType, recommended string) bool {
	return vizAcceptableFor[recommended][currentType]
}

// advise records a recommendation note when a widget's chosen type clearly
// mismatches its query shape. It is strictly non-destructive: it only appends a
// note and never changes the caller's type. Auto-building an unspecified type
// was deliberately dropped, so there is no adoption path.
func (a *vizAdvisor) advise(currentType string, logs map[string]interface{}) {
	recommended, reason := recommendWidgetType(logs)
	if recommended == "" || currentType == "" || vizAcceptable(currentType, recommended) {
		return
	}
	a.notes = append(a.notes, fmt.Sprintf("widget uses %s but %s is recommended: %s", currentType, recommended, reason))
}

// attachVizRecommendations records visualization notes on the API response.
func attachVizRecommendations(result map[string]interface{}, advisor *vizAdvisor) {
	if advisor == nil || len(advisor.notes) == 0 || result == nil {
		return
	}
	result["_viz_recommendations"] = advisor.notes
}
