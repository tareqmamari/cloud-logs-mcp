// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file surfaces advisory E2M (events-to-metrics) recommendations for
// dashboard aggregation widgets. It never creates or switches anything.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/security"
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
	if !ok || len(aggObj) != 1 {
		// A well-formed aggregation object has exactly one key (its type);
		// anything else is malformed and map iteration order must not pick.
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
		if !stringSetEqual(lowerAll(e2mLabelSourceFields(e2m)), lowerAll(agg.labels)) {
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

// lowerAll returns a copy of ss with every element lowercased, so label-set
// matching is case-insensitive — E2M metric_labels source_field values are
// user-authored and often camelCase (e.g. applicationName) while dashboard
// group-by keypaths are lowercase (e.g. applicationname). This aligns label
// matching with the case-insensitive (strings.EqualFold) measured-field match.
func lowerAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	return out
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

// e2mRecommendations walks a dashboard layout and returns one advisory note per
// eligible aggregation widget: a match against an existing metric, or (for
// archive-tier logs) a suggestion to create an E2M. Purely advisory.
func e2mRecommendations(layout interface{}, e2ms []interface{}, session *SessionContext) []string {
	var notes []string
	forEachWidgetLogsQuery(layout, func(logs map[string]interface{}) {
		agg := widgetAggregation(logs)
		if !agg.eligible {
			return
		}
		if name, ok := matchE2M(agg, e2ms); ok {
			notes = append(notes, fmt.Sprintf(
				"widget aggregation matches existing metric %q — a metrics-backed widget (query.metrics.promql_query) would query faster than logs.", name))
			return
		}
		if session == nil {
			return
		}
		app, subsystem := extractAppSubsystem(logs)
		if app == "" && subsystem == "" {
			return
		}
		if session.GetTierForAppAndSubsystem(app, subsystem) != "archive" {
			return
		}
		body, _ := json.Marshal(map[string]interface{}{"e2m": buildCreateE2MBody(agg)})
		notes = append(notes, fmt.Sprintf(
			"widget aggregates archive-tier logs; a metrics-backed widget would be faster. Suggested create_e2m body: %s", string(body)))
	})
	return notes
}

// forEachWidgetLogsQuery invokes fn with each widget's logs query (definition
// key's node["query"]["logs"], and every line_chart query_definitions[].query.logs).
func forEachWidgetLogsQuery(layout interface{}, fn func(logs map[string]interface{})) {
	lm, ok := layout.(map[string]interface{})
	if !ok {
		return
	}
	sections, _ := lm["sections"].([]interface{})
	for _, s := range sections {
		sm, _ := s.(map[string]interface{})
		rows, _ := sm["rows"].([]interface{})
		for _, r := range rows {
			rm, _ := r.(map[string]interface{})
			widgets, _ := rm["widgets"].([]interface{})
			for _, w := range widgets {
				wm, _ := w.(map[string]interface{})
				def, _ := wm["definition"].(map[string]interface{})
				for _, node := range def {
					nm, ok := node.(map[string]interface{})
					if !ok {
						continue
					}
					if logs := logsQueryOf(nm); logs != nil {
						fn(logs)
					}
					if qds, ok := nm["query_definitions"].([]interface{}); ok {
						for _, qd := range qds {
							if qdm, ok := qd.(map[string]interface{}); ok {
								if logs := logsQueryOf(qdm); logs != nil {
									fn(logs)
								}
							}
						}
					}
				}
			}
		}
	}
}

// buildCreateE2MBody derives a suggested create_e2m body from a widget aggregation.
func buildCreateE2MBody(agg widgetAgg) map[string]interface{} {
	labels := make([]interface{}, 0, len(agg.labels))
	for _, l := range agg.labels {
		labels = append(labels, map[string]interface{}{"target_label": l, "source_field": l})
	}
	base := "widget_metric"
	if agg.sourceField != "" {
		base = agg.sourceField
	}
	return map[string]interface{}{
		"name":          "dashboard_" + agg.aggType + "_" + base,
		"type":          "logs2metrics",
		"logs_query":    map[string]interface{}{"lucene": agg.lucene},
		"metric_labels": labels,
		"metric_fields": []interface{}{map[string]interface{}{
			"source_field":            firstNonEmpty(agg.sourceField, "message"),
			"target_base_metric_name": base,
			"aggregations": []interface{}{map[string]interface{}{
				"enabled": true, "agg_type": agg.aggType,
				"target_metric_name": base + "_" + agg.aggType,
			}},
		}},
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// attachE2MRecommendations records advisory E2M notes on the API response.
func attachE2MRecommendations(result map[string]interface{}, notes []string) {
	if len(notes) == 0 || result == nil {
		return
	}
	result["_e2m_recommendations"] = notes
}

// e2mFetchTimeout bounds the advisory list_e2m lookup. The recommendations
// are optional, so a slow or retrying E2M endpoint must never stall a
// dashboard create/update for more than this.
const e2mFetchTimeout = 5 * time.Second

// e2mFetchCacheKey scopes the advisory fetch's cache entry under the
// "list_e2m" tool namespace, so the existing create/update/delete_e2m
// invalidation hooks clear it too.
const e2mFetchCacheKey = "dashboard_advisory"

// fetchE2MList best-effort GETs the E2M list; returns the events2metrics array
// or nil on any error (E2M recommendations are optional, never fatal).
// Successful results are cached per session under the list_e2m TTL so repeated
// dashboard edits don't refetch, and the request carries a short deadline.
func fetchE2MList(ctx context.Context, c client.Doer, logger *zap.Logger) []interface{} {
	if c == nil {
		return nil
	}
	cacheHelper := GetCacheHelperFromContext(ctx)
	if cached, ok := cacheHelper.Get("list_e2m", e2mFetchCacheKey); ok {
		if arr, ok := cached.([]interface{}); ok {
			return arr
		}
	}
	ctx, cancel := context.WithTimeout(ctx, e2mFetchTimeout)
	defer cancel()
	resp, err := c.Do(ctx, &client.Request{Method: "GET", Path: "/v1/events2metrics", Timeout: e2mFetchTimeout})
	if err != nil {
		if logger != nil {
			logger.Debug("Failed to fetch E2M list for dashboard recommendations",
				zap.String("error", security.SanitizeError(err)))
		}
		return nil
	}
	if resp == nil || resp.StatusCode >= 400 {
		return nil
	}
	var parsed map[string]interface{}
	if json.Unmarshal(resp.Body, &parsed) != nil {
		return nil
	}
	arr, _ := parsed["events2metrics"].([]interface{})
	if arr != nil {
		cacheHelper.Set("list_e2m", e2mFetchCacheKey, arr)
	}
	return arr
}
