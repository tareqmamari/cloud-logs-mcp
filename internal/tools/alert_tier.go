// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file warns when an alert targets logs that TCO routes to the archive
// tier only. Alerts evaluate on the frequent_search (Priority Insights)
// stream, so an alert over archive-only logs never fires. Unlike dashboards,
// alert definitions carry no tier field to correct — the safeguard is
// guidance, not a rewrite.
package tools

import (
	"fmt"
)

// alertablePriorities are the TCO policy priorities whose logs are available
// to alerts. Only high- and medium-priority logs reach the stream alerts
// evaluate; low-priority (and unspecified/blocked) logs do not, so an alert
// over them never fires.
var alertablePriorities = map[string]bool{
	"type_high":   true,
	"type_medium": true,
}

// alertTierWarnings inspects an alert definition's log filters and returns a
// warning for each targeted application/subsystem whose logs a TCO policy
// routes to a non-alertable priority (low/unspecified). Returns nil when
// there is no session, or every target is high/medium priority (or matches no
// policy, i.e. falls through to the default alertable routing).
func alertTierWarnings(def map[string]interface{}, session *SessionContext) []string {
	if session == nil {
		return nil
	}

	var warnings []string
	seen := map[string]bool{}
	walkSimpleFilters(def, func(sf map[string]interface{}) {
		app, subsystem := extractSimpleFilterTarget(sf)
		if app == "" && subsystem == "" {
			return
		}
		priority, matched := session.GetPriorityForAppAndSubsystem(app, subsystem)
		// No matching policy => default routing, which is alertable; and
		// high/medium are alertable. Only an explicit non-alertable priority
		// warrants a warning.
		if !matched || alertablePriorities[priority] {
			return
		}
		key := app + "\x00" + subsystem
		if seen[key] {
			return
		}
		seen[key] = true
		warnings = append(warnings, fmt.Sprintf(
			"Alert targets %s, whose logs a TCO policy assigns %s priority. "+
				"Only high- and medium-priority logs are available to alerts, so this alert will not fire. "+
				"Raise the TCO priority for these logs (to type_high or type_medium) or retarget the alert.",
			describeAlertTarget(app, subsystem), describePriority(priority)))
	})
	return warnings
}

// describePriority renders a TCO priority for a warning message.
func describePriority(priority string) string {
	if priority == "" {
		return "an unspecified"
	}
	return fmt.Sprintf("%q", priority)
}

// attachTierWarnings records archive-only tier warnings on an API response so
// the caller sees them even outside dry-run. No-op when there are no warnings
// or the response is not a JSON object.
func attachTierWarnings(result map[string]interface{}, warnings []string) {
	if len(warnings) == 0 || result == nil {
		return
	}
	result["_tier_warnings"] = warnings
}

// describeAlertTarget renders the app/subsystem pair for a warning message.
func describeAlertTarget(app, subsystem string) string {
	switch {
	case app != "" && subsystem != "":
		return fmt.Sprintf("application %q / subsystem %q", app, subsystem)
	case app != "":
		return fmt.Sprintf("application %q", app)
	default:
		return fmt.Sprintf("subsystem %q", subsystem)
	}
}

// walkSimpleFilters recursively descends an alert definition and invokes fn
// for every simple_filter object, regardless of nesting (logs_filter,
// ratio numerator/denominator, etc.).
func walkSimpleFilters(node interface{}, fn func(sf map[string]interface{})) {
	switch n := node.(type) {
	case map[string]interface{}:
		if sf, ok := n["simple_filter"].(map[string]interface{}); ok {
			fn(sf)
		}
		for _, v := range n {
			walkSimpleFilters(v, fn)
		}
	case []interface{}:
		for _, v := range n {
			walkSimpleFilters(v, fn)
		}
	}
}

// extractSimpleFilterTarget reads the application/subsystem a simple_filter
// targets, from either its lucene_query string or its structured
// label_filters arrays.
func extractSimpleFilterTarget(sf map[string]interface{}) (app, subsystem string) {
	if lucene, ok := sf["lucene_query"].(string); ok {
		app = firstLuceneMatch(appNameLucenePattern, lucene)
		subsystem = firstLuceneMatch(subNameLucenePattern, lucene)
	}
	if labelFilters, ok := sf["label_filters"].(map[string]interface{}); ok {
		if app == "" {
			app = firstLabelFilterValue(labelFilters, "application_name")
		}
		if subsystem == "" {
			subsystem = firstLabelFilterValue(labelFilters, "subsystem_name")
		}
	}
	return app, subsystem
}

// firstLabelFilterValue returns the first non-empty "value" from a
// label_filters sub-array (application_name or subsystem_name).
func firstLabelFilterValue(labelFilters map[string]interface{}, key string) string {
	arr, ok := labelFilters[key].([]interface{})
	if !ok {
		return ""
	}
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			if v, ok := m["value"].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}
