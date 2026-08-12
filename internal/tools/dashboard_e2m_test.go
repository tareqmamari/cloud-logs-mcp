package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
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
	// a malformed aggregation object carrying two keys is ineligible (the
	// contract is exactly one aggregation; map iteration order must never
	// decide which one wins)
	multi := map[string]interface{}{"aggregations": []interface{}{
		map[string]interface{}{"count": map[string]interface{}{}, "sum": map[string]interface{}{}},
	}}
	assert.False(t, widgetAggregation(multi).eligible)
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

func TestMatchE2M_LabelCaseInsensitive(t *testing.T) {
	agg := widgetAgg{aggType: "count", sourceField: "", labels: []string{"applicationname"}, lucene: "", eligible: true}
	e2ms := []interface{}{e2mDef("", "message", "count", "count_by_app", []string{"applicationName"})}
	name, ok := matchE2M(agg, e2ms)
	assert.True(t, ok, "label match must be case-insensitive")
	assert.Equal(t, "count_by_app", name)
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

// barChartLayout wraps a logs query in a minimal one-widget dashboard layout.
func barChartLayout(logs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{"bar_chart": map[string]interface{}{"query": map[string]interface{}{"logs": logs}}},
				}},
			}},
		}},
	}
}

func TestFetchE2MList_UsesContextClientAndDeadline(t *testing.T) {
	// The advisory fetch must resolve the client the same way the main request
	// path does (context first) and must carry a short deadline so a slow E2M
	// endpoint cannot stall dashboard creation.
	ctxMock := client.NewMockClient()
	var e2mDeadline time.Time
	var hadDeadline bool
	start := time.Now()
	ctxMock.DoFunc = func(ctx context.Context, req *client.Request) (*client.Response, error) {
		if req.Method == "GET" && req.Path == "/v1/events2metrics" {
			e2mDeadline, hadDeadline = ctx.Deadline()
			return &client.Response{StatusCode: 200, Body: []byte(`{"events2metrics":[]}`)}, nil
		}
		return &client.Response{StatusCode: 200, Body: []byte(`{"dashboard_id":"d1"}`)}, nil
	}
	constructorMock := client.NewMockClient()

	tool := NewCreateDashboardTool(constructorMock, zap.NewNop())
	ctx := testCtx(ctxMock)

	logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
	result, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": barChartLayout(logs)})
	assert.NoError(t, err)
	assert.False(t, result.IsError)
	assert.True(t, hadDeadline, "advisory E2M fetch must carry a deadline")
	if hadDeadline {
		assert.LessOrEqual(t, e2mDeadline.Sub(start), 10*time.Second,
			"advisory E2M fetch deadline must be short")
	}
	assert.Zero(t, constructorMock.RequestCount(),
		"context-injected client must take precedence over the constructor client")
}

func TestFetchE2MList_CachedAcrossCalls(t *testing.T) {
	mock := client.NewMockClient()
	var e2mGETs int
	mock.DoFunc = func(_ context.Context, req *client.Request) (*client.Response, error) {
		if req.Method == "GET" && req.Path == "/v1/events2metrics" {
			e2mGETs++
			body, _ := json.Marshal(map[string]interface{}{
				"events2metrics": []interface{}{e2mDef("severity:error", "message", "count", "error_count_total", []string{"applicationname"})},
			})
			return &client.Response{StatusCode: 200, Body: body}, nil
		}
		return &client.Response{StatusCode: 200, Body: []byte(`{"dashboard_id":"d1"}`)}, nil
	}
	tool := NewCreateDashboardTool(mock, zap.NewNop())
	ctx := testCtx(mock) // one session, so both calls share one cache scope

	logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
	for i := 0; i < 2; i++ {
		result, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": barChartLayout(logs)})
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "error_count_total")
	}
	assert.Equal(t, 1, e2mGETs, "second create should reuse the cached list_e2m result")
}

func TestCreateDashboard_E2MFetchIsBestEffort(t *testing.T) {
	// Every failure mode of the advisory fetch must leave the dashboard create
	// untouched: success response, no _e2m_recommendations, no error.
	cases := []struct {
		name    string
		respond func() (*client.Response, error)
	}{
		{"transport error", func() (*client.Response, error) {
			return nil, context.DeadlineExceeded
		}},
		{"server error", func() (*client.Response, error) {
			return &client.Response{StatusCode: 500, Body: []byte(`boom`)}, nil
		}},
		{"malformed JSON", func() (*client.Response, error) {
			return &client.Response{StatusCode: 200, Body: []byte(`{not json`)}, nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := client.NewMockClient()
			mock.DoFunc = func(_ context.Context, req *client.Request) (*client.Response, error) {
				if req.Method == "GET" && req.Path == "/v1/events2metrics" {
					return tc.respond()
				}
				return &client.Response{StatusCode: 200, Body: []byte(`{"dashboard_id":"d1"}`)}, nil
			}
			tool := NewCreateDashboardTool(mock, zap.NewNop())
			ctx := testCtx(mock)

			logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
			result, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": barChartLayout(logs)})
			assert.NoError(t, err)
			assert.False(t, result.IsError, "E2M fetch failure must never fail the create")
			assert.NotContains(t, resultText(t, result), "_e2m_recommendations")
		})
	}
}

func TestUpdateDashboard_SurfacesE2MRecommendation(t *testing.T) {
	// Pins the update-path wiring: create and update attach the same advisory
	// notes, so a regression in UpdateDashboardTool.Execute must fail here.
	mock := client.NewMockClient()
	mock.DoFunc = func(_ context.Context, req *client.Request) (*client.Response, error) {
		if req.Method == "GET" && req.Path == "/v1/events2metrics" {
			body, _ := json.Marshal(map[string]interface{}{
				"events2metrics": []interface{}{e2mDef("severity:error", "message", "count", "error_count_total", []string{"applicationname"})},
			})
			return &client.Response{StatusCode: 200, Body: body}, nil
		}
		return &client.Response{StatusCode: 200, Body: []byte(`{"dashboard_id":"d1"}`)}, nil
	}
	tool := NewUpdateDashboardTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
	result, err := tool.Execute(ctx, map[string]interface{}{
		"dashboard_id": "d1",
		"name":         "d",
		"layout":       barChartLayout(logs),
	})
	assert.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(t, result), "error_count_total")
}

func TestCreateDashboard_SurfacesE2MRecommendation(t *testing.T) {
	mock := client.NewMockClient()
	// list_e2m (GET /v1/events2metrics) returns a matching metric; the create returns an id.
	mock.DoFunc = func(_ context.Context, req *client.Request) (*client.Response, error) {
		if req.Method == "GET" && req.Path == "/v1/events2metrics" {
			body, _ := json.Marshal(map[string]interface{}{
				"events2metrics": []interface{}{e2mDef("severity:error", "message", "count", "error_count_total", []string{"applicationname"})},
			})
			return &client.Response{StatusCode: 200, Body: body}, nil
		}
		body, _ := json.Marshal(map[string]interface{}{"dashboard_id": "d1"})
		return &client.Response{StatusCode: 200, Body: body}, nil
	}
	tool := NewCreateDashboardTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	logs := logsAgg("count", "", []string{"applicationname"}, "severity:error")
	layout := map[string]interface{}{
		"sections": []interface{}{map[string]interface{}{
			"rows": []interface{}{map[string]interface{}{
				"widgets": []interface{}{map[string]interface{}{
					"definition": map[string]interface{}{"bar_chart": map[string]interface{}{"query": map[string]interface{}{"logs": logs}}},
				}},
			}},
		}},
	}
	result, err := tool.Execute(ctx, map[string]interface{}{"name": "d", "layout": layout})
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, result), "error_count_total")
}
