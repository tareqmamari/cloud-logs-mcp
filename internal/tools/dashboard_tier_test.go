package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// tcoSessionRoutingAppTo builds a session whose TCO policy routes the given
// application name to the given tier, and everything else to the default.
func tcoSessionRoutingAppTo(app, tier, defaultTier string) *SessionContext {
	s := NewSessionContext("tier-test", "instance-1")
	s.SetTCOConfig(&TCOConfig{
		HasPolicies:       true,
		HasArchive:        true,
		HasFrequentSearch: tier == "frequent_search" || defaultTier == "frequent_search",
		DefaultTier:       defaultTier,
		PolicyCount:       1,
		LastUpdated:       time.Now(), // fresh, so the defensive refetch no-ops
		Policies: []TCOPolicyRule{
			{
				ApplicationRule: &TCOMatchRule{Name: app, RuleType: "is"},
				Tier:            tier,
				Priority:        "type_low",
			},
		},
	})
	return s
}

func luceneLogsQuery(value string) map[string]interface{} {
	return map[string]interface{}{
		"lucene_query": map[string]interface{}{"value": value},
	}
}

func TestWidgetTierResolver_ArchiveAppFromLucene(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "archive", "frequent_search")
	r := newWidgetTierResolver(session)

	dmt := r.dataModeTypeFor(luceneLogsQuery("applicationname:payment-service AND severity:error"))
	assert.Equal(t, "archive", dmt, "app routed to archive must select the archive data_mode_type")
}

func TestWidgetTierResolver_FrequentSearchApp(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "frequent_search", "archive")
	r := newWidgetTierResolver(session)

	dmt := r.dataModeTypeFor(luceneLogsQuery("applicationname:payment-service"))
	assert.Equal(t, "high_unspecified", dmt, "frequent_search maps to the high_unspecified data_mode_type")
}

func TestWidgetTierResolver_SubsystemFromStructuredFilter(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "archive", "frequent_search")
	// Route by subsystem too, using a structured observation_field filter.
	session.SetTCOConfig(&TCOConfig{
		HasPolicies: true, HasArchive: true, DefaultTier: "frequent_search", PolicyCount: 1,
		LastUpdated: time.Now(),
		Policies: []TCOPolicyRule{
			{SubsystemRule: &TCOMatchRule{Name: "billing", RuleType: "is"}, Tier: "archive"},
		},
	})
	r := newWidgetTierResolver(session)

	logs := map[string]interface{}{
		"filters": []interface{}{
			map[string]interface{}{
				"observation_field": map[string]interface{}{"keypath": []interface{}{"subsystemname"}, "scope": "label"},
				"operator": map[string]interface{}{
					"equals": map[string]interface{}{"selection": map[string]interface{}{"list": map[string]interface{}{"values": []interface{}{"billing"}}}},
				},
			},
		},
	}
	assert.Equal(t, "archive", r.dataModeTypeFor(logs))
}

func TestWidgetTierResolver_NoAppFallsBackToDefault(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "archive", "archive")
	r := newWidgetTierResolver(session)

	// A query with no application/subsystem should use the instance default,
	// which is archive here.
	assert.Equal(t, "archive", r.dataModeTypeFor(luceneLogsQuery("severity:error")))
}

func TestWidgetTierResolver_NilSessionKeepsFrequentSearch(t *testing.T) {
	r := newWidgetTierResolver(nil)
	assert.Equal(t, "high_unspecified", r.dataModeTypeFor(luceneLogsQuery("applicationname:x")))
}

// TestEnsureRequiredDashboardFields_TierFromTCO drives the full normalizer:
// a line-chart widget filtered to an archive-routed app must come out with
// data_mode_type "archive", while an explicit value is preserved.
func TestEnsureRequiredDashboardFields_TierFromTCO(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "archive", "frequent_search")
	resolver := newWidgetTierResolver(session)

	layout := lineChartLayout("applicationname:payment-service", nil)
	ensureRequiredDashboardFields(layout, resolver)
	assert.Equal(t, "archive", firstQueryDefDataMode(t, layout))

	// Explicit data_mode_type must win over TCO inference.
	explicit := lineChartLayout("applicationname:payment-service", map[string]interface{}{"data_mode_type": "high_unspecified"})
	ensureRequiredDashboardFields(explicit, newWidgetTierResolver(session))
	assert.Equal(t, "high_unspecified", firstQueryDefDataMode(t, explicit))
}

func TestCreateDashboard_SendsArchiveTierFromTCO(t *testing.T) {
	session := tcoSessionRoutingAppTo("payment-service", "archive", "frequent_search")

	mock := client.NewMockClient()
	// TCO fetch (defensive) + query validation + dashboard create all hit the
	// mock; a permissive default response keeps them happy.
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{"policies":[]}`)}
	mock.RespondWith(200, map[string]interface{}{"dashboard_id": "d1"})

	tool := NewCreateDashboardTool(mock, zap.NewNop())
	ctx := WithSession(testCtx(mock), session)

	layout := lineChartLayout("applicationname:payment-service", nil)
	_, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": layout})
	assert.NoError(t, err)

	sent, ok := mock.LastRequest().Body.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "archive", firstQueryDefDataMode(t, sent["layout"]))
}

// lineChartLayout builds a minimal one-widget line-chart dashboard layout
// whose logs query carries the given lucene filter. extraQueryDefFields, if
// set, are merged into the query definition (used to pin an explicit
// data_mode_type).
func lineChartLayout(lucene string, extraQueryDefFields map[string]interface{}) map[string]interface{} {
	queryDef := map[string]interface{}{
		"query": map[string]interface{}{
			"logs": map[string]interface{}{
				"lucene_query": map[string]interface{}{"value": lucene},
				"aggregations": []interface{}{map[string]interface{}{"count": map[string]interface{}{}}},
			},
		},
	}
	for k, v := range extraQueryDefFields {
		queryDef[k] = v
	}
	return map[string]interface{}{
		"sections": []interface{}{
			map[string]interface{}{
				"rows": []interface{}{
					map[string]interface{}{
						"widgets": []interface{}{
							map[string]interface{}{
								"definition": map[string]interface{}{
									"line_chart": map[string]interface{}{
										"query_definitions": []interface{}{queryDef},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func firstQueryDefDataMode(t *testing.T, layout interface{}) string {
	t.Helper()
	lm := layout.(map[string]interface{})
	section := lm["sections"].([]interface{})[0].(map[string]interface{})
	row := section["rows"].([]interface{})[0].(map[string]interface{})
	widget := row["widgets"].([]interface{})[0].(map[string]interface{})
	lc := widget["definition"].(map[string]interface{})["line_chart"].(map[string]interface{})
	qd := lc["query_definitions"].([]interface{})[0].(map[string]interface{})
	dmt, _ := qd["data_mode_type"].(string)
	return dmt
}
