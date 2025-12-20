//go:build integration

package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Comprehensive E2E Tests for All IBM Cloud Logs MCP Tools
// ============================================================================
//
// Run with:
//   LOGS_SERVICE_URL="..." LOGS_API_KEY="..." go test -tags=integration -v ./internal/tools/... -run "TestE2E_" -timeout 10m

// ============================================================================
// Query Tools
// ============================================================================

func TestE2E_QueryTool(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	queryTool := NewQueryTool(c, logger)
	ctx := context.Background()

	now := time.Now().UTC()
	startDate := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endDate := now.Format(time.RFC3339)

	t.Run("basic_query", func(t *testing.T) {
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      "source logs | limit 10",
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "frequent_search",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError, "Query should succeed")

		text := extractTextContent(result)
		t.Logf("Query result preview:\n%s", truncateForLog(text, 500))
	})

	t.Run("error_query", func(t *testing.T) {
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      "source logs | filter $m.severity >= 5 | limit 20",
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "frequent_search",
		})
		require.NoError(t, err)
		text := extractTextContent(result)
		t.Logf("Error query result:\n%s", truncateForLog(text, 500))
	})

	t.Run("aggregation_query", func(t *testing.T) {
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      "source logs | groupby $l.applicationname | aggregate count() as log_count | sortby -log_count | limit 10",
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "frequent_search",
		})
		require.NoError(t, err)
		text := extractTextContent(result)
		t.Logf("Aggregation result:\n%s", truncateForLog(text, 800))
	})
}

// ============================================================================
// Health Check Tool
// ============================================================================

func TestE2E_HealthCheckTool(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	healthTool := NewHealthCheckTool(c, logger)
	ctx := context.Background()

	t.Run("1h_health_check", func(t *testing.T) {
		result, err := healthTool.Execute(ctx, map[string]interface{}{
			"time_range": "1h",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError, "Health check should succeed")

		text := extractTextContent(result)
		t.Logf("Health check result:\n%s", truncateForLog(text, 1000))

		// Should contain health check sections
		assert.True(t, strings.Contains(text, "Health") || strings.Contains(text, "Status"),
			"Should contain health/status information")
	})
}

// ============================================================================
// Smart Investigation Tool
// ============================================================================

func TestE2E_SmartInvestigateTool(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	investigateTool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	t.Run("global_investigation", func(t *testing.T) {
		result, err := investigateTool.Execute(ctx, map[string]interface{}{
			"time_range": "1h",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError, "Investigation should succeed")

		text := extractTextContent(result)
		t.Logf("Investigation result:\n%s", truncateForLog(text, 1500))

		// Should have standard investigation sections
		assert.True(t, strings.Contains(text, "Investigation") || strings.Contains(text, "Mode"),
			"Should contain investigation report")
	})
}

// ============================================================================
// Alert Definition Tools
// ============================================================================

func TestE2E_AlertDefinitionTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	listAlertsTool := NewListAlertDefinitionsTool(c, logger)
	ctx := context.Background()

	t.Run("list_alert_definitions", func(t *testing.T) {
		result, err := listAlertsTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Alert definitions:\n%s", truncateForLog(text, 1000))

		// Result should not error
		assert.False(t, result.IsError, "List alert definitions should succeed")
	})
}

// ============================================================================
// Dashboard Tools
// ============================================================================

func TestE2E_DashboardTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_dashboards", func(t *testing.T) {
		listTool := NewListDashboardsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Dashboards list:\n%s", truncateForLog(text, 1000))
	})

	t.Run("list_dashboard_folders", func(t *testing.T) {
		listFoldersTool := NewListDashboardFoldersTool(c, logger)
		result, err := listFoldersTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Dashboard folders:\n%s", truncateForLog(text, 500))
	})
}

// ============================================================================
// Policy Tools
// ============================================================================

func TestE2E_PolicyTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_policies", func(t *testing.T) {
		listTool := NewListPoliciesTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Policies list:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Rule Group Tools
// ============================================================================

func TestE2E_RuleGroupTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_rule_groups", func(t *testing.T) {
		listTool := NewListRuleGroupsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Rule groups:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Webhook Tools
// ============================================================================

func TestE2E_WebhookTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_outgoing_webhooks", func(t *testing.T) {
		listTool := NewListOutgoingWebhooksTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Outgoing webhooks:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// E2M Tools
// ============================================================================

func TestE2E_E2MTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_e2m", func(t *testing.T) {
		listTool := NewListE2MTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("E2M configurations:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Data Access Rules Tools
// ============================================================================

func TestE2E_DataAccessRulesTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_data_access_rules", func(t *testing.T) {
		listTool := NewListDataAccessRulesTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Data access rules:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Enrichment Tools
// ============================================================================

func TestE2E_EnrichmentTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_enrichments", func(t *testing.T) {
		listTool := NewListEnrichmentsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Enrichments:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// View Tools
// ============================================================================

func TestE2E_ViewTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_views", func(t *testing.T) {
		listTool := NewListViewsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Views:\n%s", truncateForLog(text, 1000))
	})

	t.Run("list_view_folders", func(t *testing.T) {
		listTool := NewListViewFoldersTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("View folders:\n%s", truncateForLog(text, 500))
	})
}

// ============================================================================
// Streams Tools
// ============================================================================

func TestE2E_StreamsTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_streams", func(t *testing.T) {
		listTool := NewListStreamsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Streams:\n%s", truncateForLog(text, 1000))
	})

	t.Run("get_event_stream_targets", func(t *testing.T) {
		listTool := NewGetEventStreamTargetsTool(c, logger)
		result, err := listTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Event stream targets:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Alerts (Active) Tools
// ============================================================================

func TestE2E_AlertsTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("list_alerts", func(t *testing.T) {
		listTool := NewListAlertsTool(c, logger)

		now := time.Now().UTC()
		startDate := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endDate := now.Format(time.RFC3339)

		result, err := listTool.Execute(ctx, map[string]interface{}{
			"start_time": startDate,
			"end_time":   endDate,
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Active alerts:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Dry Run Validation Tests
// ============================================================================

func TestE2E_DryRunValidation(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("create_alert_dry_run", func(t *testing.T) {
		createTool := NewCreateAlertDefinitionTool(c, logger)

		result, err := createTool.Execute(ctx, map[string]interface{}{
			"definition": map[string]interface{}{
				"name":        "Test Alert E2E",
				"description": "E2E test alert - dry run",
				"priority":    "P3",
				"type":        "standard",
				"condition": map[string]interface{}{
					"metric_alert_condition": map[string]interface{}{
						"query": "source logs | filter $m.severity >= 5",
					},
				},
			},
			"dry_run": true,
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Alert dry run result:\n%s", truncateForLog(text, 1000))

		// Should contain dry run validation output
		assert.True(t, strings.Contains(text, "Dry Run") || strings.Contains(text, "Validation"),
			"Should contain dry run output")
	})

	t.Run("create_policy_dry_run", func(t *testing.T) {
		createTool := NewCreatePolicyTool(c, logger)

		result, err := createTool.Execute(ctx, map[string]interface{}{
			"policy": map[string]interface{}{
				"name":        "Test Policy E2E",
				"description": "E2E test policy - dry run",
				"priority":    "type_medium",
			},
			"dry_run": true,
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Policy dry run result:\n%s", truncateForLog(text, 1000))
	})

	t.Run("create_webhook_dry_run", func(t *testing.T) {
		createTool := NewCreateOutgoingWebhookTool(c, logger)

		result, err := createTool.Execute(ctx, map[string]interface{}{
			"webhook": map[string]interface{}{
				"name": "Test Webhook E2E",
				"type": "generic",
				"url":  "https://example.com/webhook",
			},
			"dry_run": true,
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Webhook dry run result:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Config Correlation Tool (RCA)
// ============================================================================

func TestE2E_ConfigCorrelationTool(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("config_changes_correlation", func(t *testing.T) {
		correlationTool := NewGetConfigChangesInWindowTool(c, logger)

		now := time.Now().UTC()
		incidentTime := now.Add(-30 * time.Minute).Format(time.RFC3339)

		result, err := correlationTool.Execute(ctx, map[string]interface{}{
			"incident_time": incidentTime,
			"window_before": "1h",
			"window_after":  "15m",
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Config correlation result:\n%s", truncateForLog(text, 1500))
	})
}

// ============================================================================
// Discovery Tools
// ============================================================================

func TestE2E_DiscoveryTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("discover_tools", func(t *testing.T) {
		discoverTool := NewDiscoverToolsTool(c, logger)
		result, err := discoverTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Discovered tools:\n%s", truncateForLog(text, 1000))
	})

	t.Run("search_tools", func(t *testing.T) {
		searchTool := NewSearchToolsTool(c, logger)
		result, err := searchTool.Execute(ctx, map[string]interface{}{
			"query": "alert",
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Search tools result:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Query Templates Tool
// ============================================================================

func TestE2E_QueryTemplatesTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("query_templates", func(t *testing.T) {
		templatesTool := NewQueryTemplatesTool(c, logger)
		result, err := templatesTool.Execute(ctx, map[string]interface{}{})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("Query templates:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// DataPrime Reference Tool
// ============================================================================

func TestE2E_DataPrimeReferenceTools(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	t.Run("dataprime_reference", func(t *testing.T) {
		referenceTool := NewDataPrimeReferenceTool(c, logger)
		result, err := referenceTool.Execute(ctx, map[string]interface{}{
			"topic": "filter",
		})
		require.NoError(t, err)

		text := extractTextContent(result)
		t.Logf("DataPrime reference:\n%s", truncateForLog(text, 1000))
	})
}

// ============================================================================
// Complete Tool Coverage Summary
// ============================================================================

func TestE2E_ToolCoverageSummary(t *testing.T) {
	c := getTestClient(t)
	logger, _ := getTestLogger()

	ctx := context.Background()

	// This test documents all tools tested and their status
	tools := []struct {
		name     string
		category string
		test     func() error
	}{
		{"query", "Query", func() error {
			tool := NewQueryTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{
				"query":      "source logs | limit 1",
				"start_date": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				"end_date":   time.Now().Format(time.RFC3339),
				"tier":       "frequent_search",
			})
			return err
		}},
		{"health_check", "Monitoring", func() error {
			tool := NewHealthCheckTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{"time_range": "15m"})
			return err
		}},
		{"smart_investigate", "Investigation", func() error {
			tool := NewSmartInvestigateTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{"time_range": "15m"})
			return err
		}},
		{"list_alert_definitions", "Alerts", func() error {
			tool := NewListAlertDefinitionsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_dashboards", "Dashboards", func() error {
			tool := NewListDashboardsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_policies", "Policies", func() error {
			tool := NewListPoliciesTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_rule_groups", "Rules", func() error {
			tool := NewListRuleGroupsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_outgoing_webhooks", "Webhooks", func() error {
			tool := NewListOutgoingWebhooksTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_e2m", "E2M", func() error {
			tool := NewListE2MTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_data_access_rules", "Access Control", func() error {
			tool := NewListDataAccessRulesTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_enrichments", "Enrichments", func() error {
			tool := NewListEnrichmentsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_views", "Views", func() error {
			tool := NewListViewsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"list_streams", "Streams", func() error {
			tool := NewListStreamsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"get_event_stream_targets", "Event Streams", func() error {
			tool := NewGetEventStreamTargetsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"discover_tools", "Discovery", func() error {
			tool := NewDiscoverToolsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"search_tools", "Discovery", func() error {
			tool := NewSearchToolsTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{"query": "alert"})
			return err
		}},
		{"query_templates", "Templates", func() error {
			tool := NewQueryTemplatesTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{})
			return err
		}},
		{"dataprime_reference", "Reference", func() error {
			tool := NewDataPrimeReferenceTool(c, logger)
			_, err := tool.Execute(ctx, map[string]interface{}{"topic": "filter"})
			return err
		}},
		{"config_correlation", "RCA", func() error {
			tool := NewGetConfigChangesInWindowTool(c, logger)
			now := time.Now().UTC()
			_, err := tool.Execute(ctx, map[string]interface{}{
				"incident_time": now.Add(-30 * time.Minute).Format(time.RFC3339),
				"window_before": "1h",
				"window_after":  "15m",
			})
			return err
		}},
	}

	// Run all tools and report status
	t.Log("\n============================================")
	t.Log("Tool Coverage Summary")
	t.Log("============================================")

	passed := 0
	failed := 0

	for _, tool := range tools {
		err := tool.test()
		status := "✓ PASS"
		if err != nil {
			status = "✗ FAIL: " + err.Error()
			failed++
		} else {
			passed++
		}
		t.Logf("[%s] %s: %s", tool.category, tool.name, status)
	}

	t.Log("============================================")
	t.Logf("Total: %d passed, %d failed", passed, failed)
	t.Log("============================================")

	assert.Equal(t, 0, failed, "All tools should pass E2E tests")
}
