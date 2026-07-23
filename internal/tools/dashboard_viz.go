// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file classifies the appropriate visualization for a dashboard widget
// from the shape of its logs query (aggregations and group-bys).
package tools

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
