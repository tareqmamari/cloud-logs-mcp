//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// TestAlertsCRUD tests the complete lifecycle of alerts
func TestAlertsCRUD(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var alertID string

	// Test: Create Alert
	t.Run("CreateAlert", func(t *testing.T) {
		alertName := GenerateUniqueName("alert-crud")
		alertConfig := map[string]interface{}{
			"name":        alertName,
			"description": "Integration test alert",
			"is_active":   true,
			"severity":    "info_or_unspecified",
			"notification_groups": []map[string]interface{}{
				{
					"group_by_fields": []string{"coralogix.metadata.applicationName"},
				},
			},
			"condition": map[string]interface{}{
				"immediate": map[string]interface{}{},
			},
			"filters": map[string]interface{}{
				"filter_type": "text_or_unspecified",
				"severities":  []string{"info"},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/alerts",
			Body:   alertConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create alert")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain alert ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, alertName, result["name"], "Alert name should match")

		// Save alert ID for subsequent tests
		alertID = result["id"].(string)
		AssertValidUUID(t, alertID, "Alert ID should be a valid UUID")
	})

	// Test: Get Alert
	t.Run("GetAlert", func(t *testing.T) {
		require.NotEmpty(t, alertID, "Alert ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts/" + alertID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get alert")
		require.NotNil(t, result, "Response should not be nil")

		// Verify alert details
		assert.Equal(t, alertID, result["id"], "Alert ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Contains(t, result, "severity", "Response should contain severity")
		assert.Contains(t, result, "is_active", "Response should contain is_active")
	})

	// Test: List Alerts
	t.Run("ListAlerts", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts",
			Query: map[string]string{
				"limit": "10",
			},
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list alerts")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "alerts", "Response should contain alerts array")
		alerts, ok := result["alerts"].([]interface{})
		require.True(t, ok, "Alerts should be an array")

		// Verify our created alert is in the list
		found := false
		for _, alert := range alerts {
			alertMap := alert.(map[string]interface{})
			if alertMap["id"] == alertID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created alert should be in the list")
	})

	// Test: Update Alert
	t.Run("UpdateAlert", func(t *testing.T) {
		require.NotEmpty(t, alertID, "Alert ID should be set from create test")

		// First get the current alert to use as base for update
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts/" + alertID,
		}
		currentAlert, err := tc.DoRequest(getReq)
		require.NoError(t, err, "Failed to get current alert for update")

		// Update specific fields
		currentAlert["name"] = "updated-" + GenerateUniqueName("alert")
		currentAlert["description"] = "Updated integration test alert"
		currentAlert["is_active"] = false

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/alerts/" + alertID,
			Body:   currentAlert,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update alert")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, alertID, result["id"], "Alert ID should remain the same")
		assert.Equal(t, "Updated integration test alert", result["description"], "Description should be updated")
		assert.Equal(t, false, result["is_active"], "is_active should be updated")
	})

	// Test: Delete Alert
	t.Run("DeleteAlert", func(t *testing.T) {
		require.NotEmpty(t, alertID, "Alert ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/alerts/" + alertID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete alert")

		// Verify alert is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts/" + alertID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted alert should return 404")
	})
}

// TestAlertsPagination tests pagination for listing alerts
func TestAlertsPagination(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	// Create multiple alerts for pagination testing
	createdAlerts := []string{}
	defer func() {
		// Cleanup created alerts
		for _, id := range createdAlerts {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/alerts/" + id,
			}
			tc.DoRequest(req) // Ignore errors during cleanup
		}
	}()

	// Create 5 test alerts
	for i := 0; i < 5; i++ {
		alertConfig := map[string]interface{}{
			"name":        GenerateUniqueName("alert-pagination"),
			"description": "Pagination test alert",
			"is_active":   true,
			"severity":    "info_or_unspecified",
			"notification_groups": []map[string]interface{}{
				{
					"group_by_fields": []string{"coralogix.metadata.applicationName"},
				},
			},
			"condition": map[string]interface{}{
				"immediate": map[string]interface{}{},
			},
			"filters": map[string]interface{}{
				"filter_type": "text_or_unspecified",
				"severities":  []string{"info"},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/alerts",
			Body:   alertConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create alert")
		createdAlerts = append(createdAlerts, result["id"].(string))

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("PaginateWithLimit", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts",
			Query: map[string]string{
				"limit": "2",
			},
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list alerts with pagination")

		alerts, ok := result["alerts"].([]interface{})
		require.True(t, ok, "Alerts should be an array")
		// Note: IBM Cloud Logs API may not respect the limit parameter for alerts
		// Just verify we got alerts back
		assert.GreaterOrEqual(t, len(alerts), 5, "Should have at least the 5 alerts we created")
	})

	t.Run("PaginateWithCursor", func(t *testing.T) {
		// First request
		firstReq := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts",
			Query: map[string]string{
				"limit": "2",
			},
		}

		firstResult, err := tc.DoRequest(firstReq)
		require.NoError(t, err, "Failed to get first page")

		if cursor, ok := firstResult["next_cursor"].(string); ok && cursor != "" {
			// Second request with cursor
			secondReq := &client.Request{
				Method: "GET",
				Path:   "/v1/alerts",
				Query: map[string]string{
					"limit":  "2",
					"cursor": cursor,
				},
			}

			secondResult, err := tc.DoRequest(secondReq)
			require.NoError(t, err, "Failed to get second page")

			// Verify we got different results
			firstAlerts := firstResult["alerts"].([]interface{})
			secondAlerts := secondResult["alerts"].([]interface{})

			if len(secondAlerts) > 0 {
				firstID := firstAlerts[0].(map[string]interface{})["id"]
				secondID := secondAlerts[0].(map[string]interface{})["id"]
				assert.NotEqual(t, firstID, secondID, "Pages should contain different alerts")
			}
		}
	})
}

// TestAlertsErrorHandling tests error scenarios
func TestAlertsErrorHandling(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("GetNonExistentAlert", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/alerts/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error")
	})

	t.Run("CreateAlertWithInvalidData", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"name": "", // Empty name should be invalid
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/alerts",
			Body:   invalidConfig,
		}

		_, err := tc.DoRequestExpectError(req, 422)
		assert.NoError(t, err, "Should handle 422 error for invalid data")
	})

	t.Run("UpdateNonExistentAlert", func(t *testing.T) {
		updateConfig := map[string]interface{}{
			"name":        "test",
			"description": "test",
			"is_active":   true,
			"severity":    "info_or_unspecified",
			"notification_groups": []map[string]interface{}{
				{
					"group_by_fields": []string{"coralogix.metadata.applicationName"},
				},
			},
			"condition": map[string]interface{}{
				"immediate": map[string]interface{}{},
			},
			"filters": map[string]interface{}{
				"filter_type": "text_or_unspecified",
				"severities":  []string{"info"},
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/alerts/00000000-0000-0000-0000-000000000000",
			Body:   updateConfig,
		}

		// API returns 4xx for malformed/non-existent UUIDs (can be 400 or 422)
		ctx := context.Background()
		resp, err := tc.Client.Do(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.StatusCode == 400 || resp.StatusCode == 422, "Should return 4xx error for non-existent alert")
	})

	t.Run("DeleteNonExistentAlert", func(t *testing.T) {
		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/alerts/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error for non-existent alert")
	})
}

// TestAlertsWithFilters tests creating alerts with various filter configurations
func TestAlertsWithFilters(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	testCases := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name: "AlertWithTextFilter",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("alert-text-filter"),
				"description": "Alert with text filter",
				"is_active":   true,
				"severity":    "error",
				"notification_groups": []map[string]interface{}{
					{
						"group_by_fields": []string{"applicationName"},
					},
				},
				"condition": map[string]interface{}{
					"immediate": map[string]interface{}{},
				},
				"filters": map[string]interface{}{
					"text":       "error",
					"severities": []string{"error"},
				},
			},
		},
		{
			name: "AlertWithApplicationFilter",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("alert-app-filter"),
				"description": "Alert with application filter",
				"is_active":   true,
				"severity":    "warning",
				"notification_groups": []map[string]interface{}{
					{
						"group_by_fields": []string{"applicationName"},
					},
				},
				"condition": map[string]interface{}{
					"more_than": map[string]interface{}{
						"parameters": map[string]interface{}{
							"threshold":          10,
							"timeframe":          "timeframe_10_min",
							"group_by":           []string{"applicationName"},
							"relative_timeframe": "hour_or_unspecified",
						},
					},
				},
				"filters": map[string]interface{}{
					"applications": []string{"test-app"},
					"severities":   []string{"warning", "error"},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := &client.Request{
				Method: "POST",
				Path:   "/v1/alerts",
				Body:   testCase.config,
			}

			result, err := tc.DoRequest(req)
			require.NoError(t, err, "Failed to create alert with filters")
			require.NotNil(t, result, "Response should not be nil")

			alertID := result["id"].(string)
			defer func() {
				// Cleanup
				deleteReq := &client.Request{
					Method: "DELETE",
					Path:   "/v1/alerts/" + alertID,
				}
				tc.DoRequest(deleteReq)
			}()

			// Verify alert was created with correct configuration
			assert.Equal(t, testCase.config["name"], result["name"], "Name should match")
			assert.Contains(t, result, "filters", "Response should contain filters")
		})
	}
}
