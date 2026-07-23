// Package tools provides MCP tools for IBM Cloud Logs operations.
// This file resolves the query tier (data_mode_type) for dashboard widgets
// from the session's cached TCO policies, so panels query the tier where the
// targeted application/subsystem's logs actually live.
package tools

import (
	"context"
	"fmt"
	"regexp"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// resolverFromContext builds a widgetTierResolver backed by the session's
// cached TCO policies. It defensively refreshes the TCO cache (a no-op when
// already fresh) so the resolver works even if the server's startup fetch was
// skipped; any fetch error is non-fatal and simply leaves the resolver to
// fall back to default-tier behavior.
func resolverFromContext(ctx context.Context, c client.Doer, logger *zap.Logger) *widgetTierResolver {
	_ = FetchAndCacheTCOConfig(ctx, c, logger)
	session := GetSessionFromContext(ctx)
	if session == nil {
		session = GetSession()
	}
	return newWidgetTierResolver(session)
}

// attachTierSelection records the per-widget tier decisions on the API
// response so the caller can see which tier each panel queries and why. It is
// a no-op when nothing was resolved or the response is not a JSON object.
func attachTierSelection(result map[string]interface{}, resolver *widgetTierResolver) {
	if resolver == nil || len(resolver.notes) == 0 || result == nil {
		return
	}
	result["_tier_selection"] = resolver.notes
}

// dataModeTypeForTier maps a TCO tier name onto the dashboard widget
// data_mode_type enum. The dashboard API expresses a widget's query tier via
// data_mode_type: "high_unspecified" is the Priority Insights /
// frequent_search tier, "archive" is the archive (TCO/COS) tier.
func dataModeTypeForTier(tier string) string {
	if tier == "archive" {
		return "archive"
	}
	return "high_unspecified"
}

// widgetTierResolver picks a widget's data_mode_type from the session's
// cached TCO policies, keyed on the application/subsystem the widget's logs
// query targets. It accumulates one human-readable note per resolved widget
// so the choice can be surfaced back to the caller.
type widgetTierResolver struct {
	session *SessionContext
	notes   []string
}

func newWidgetTierResolver(session *SessionContext) *widgetTierResolver {
	return &widgetTierResolver{session: session}
}

// dataModeTypeFor returns the data_mode_type for a widget's logs query (the
// value found under a widget query's "logs" key). It extracts the
// application/subsystem the query targets and resolves the tier via TCO
// policy; when it cannot determine an app/subsystem it falls back to the
// instance default tier. With no session it defaults to the frequent_search
// tier (high_unspecified), matching the historical hard-coded behavior.
func (r *widgetTierResolver) dataModeTypeFor(logsQuery map[string]interface{}) string {
	if r == nil || r.session == nil {
		return "high_unspecified"
	}
	app, subsystem := extractAppSubsystem(logsQuery)
	tier := r.session.GetTierForAppAndSubsystem(app, subsystem)
	r.notes = append(r.notes, tierSelectionNote(app, subsystem, tier))
	return dataModeTypeForTier(tier)
}

// tierSelectionNote renders a short explanation of why a widget got its tier.
func tierSelectionNote(app, subsystem, tier string) string {
	switch {
	case app != "" && subsystem != "":
		return fmt.Sprintf("app %q / subsystem %q -> %s tier", app, subsystem, tier)
	case app != "":
		return fmt.Sprintf("app %q -> %s tier", app, tier)
	case subsystem != "":
		return fmt.Sprintf("subsystem %q -> %s tier", subsystem, tier)
	default:
		return fmt.Sprintf("no app/subsystem filter -> instance default %s tier", tier)
	}
}

var (
	appNameLucenePattern = regexp.MustCompile(`(?i)\bapplicationname\s*:\s*(?:"([^"]+)"|'([^']+)'|([^\s()"']+))`)
	subNameLucenePattern = regexp.MustCompile(`(?i)\bsubsystemname\s*:\s*(?:"([^"]+)"|'([^']+)'|([^\s()"']+))`)
)

// extractAppSubsystem finds the application and/or subsystem a widget's logs
// query targets. It understands the two representations the dashboard tool
// and callers commonly produce: a lucene_query string (applicationname:foo)
// and structured filters keyed by an observation_field keypath. Values it
// cannot confidently read are returned empty, so the resolver falls back to
// the instance default tier rather than guessing.
func extractAppSubsystem(logsQuery map[string]interface{}) (app, subsystem string) {
	if logsQuery == nil {
		return "", ""
	}

	// 1. Lucene query string (e.g. "applicationname:payment AND severity:error").
	if lq, ok := logsQuery["lucene_query"].(map[string]interface{}); ok {
		if value, ok := lq["value"].(string); ok {
			app = firstLuceneMatch(appNameLucenePattern, value)
			subsystem = firstLuceneMatch(subNameLucenePattern, value)
		}
	}

	// 2. Structured filters carrying an observation_field keypath + value.
	if app == "" || subsystem == "" {
		if filters, ok := logsQuery["filters"].([]interface{}); ok {
			for _, f := range filters {
				filter, ok := f.(map[string]interface{})
				if !ok {
					continue
				}
				key := observationFieldKey(filter)
				if key == "" {
					continue
				}
				val := firstStringValue(filter)
				if val == "" {
					continue
				}
				switch key {
				case "applicationname":
					if app == "" {
						app = val
					}
				case "subsystemname":
					if subsystem == "" {
						subsystem = val
					}
				}
			}
		}
	}

	return app, subsystem
}

// firstLuceneMatch returns the first captured group (the value, from whichever
// quoting alternative matched) of the pattern against s, or "".
func firstLuceneMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// observationFieldKey returns the lowercased last element of a filter's
// observation_field keypath (e.g. "applicationname"), or "".
func observationFieldKey(filter map[string]interface{}) string {
	of, ok := filter["observation_field"].(map[string]interface{})
	if !ok {
		return ""
	}
	keypath, ok := of["keypath"].([]interface{})
	if !ok || len(keypath) == 0 {
		return ""
	}
	last, _ := keypath[len(keypath)-1].(string)
	return last
}

// firstStringValue recursively finds the first non-empty string held under a
// "values"/"value" key within a filter's operator selection. This tolerates
// the several nesting shapes the dashboard filter operators use without
// hard-coding one exact path.
func firstStringValue(node interface{}) string {
	switch n := node.(type) {
	case map[string]interface{}:
		for _, key := range []string{"value", "values"} {
			if v, ok := n[key]; ok {
				if s := firstStringValue(v); s != "" {
					return s
				}
			}
		}
		// Descend into everything except the observation_field (which holds
		// the keypath, not a filter value).
		for k, v := range n {
			if k == "observation_field" {
				continue
			}
			if s := firstStringValue(v); s != "" {
				return s
			}
		}
	case []interface{}:
		for _, v := range n {
			if s := firstStringValue(v); s != "" {
				return s
			}
		}
	case string:
		return n
	}
	return ""
}
