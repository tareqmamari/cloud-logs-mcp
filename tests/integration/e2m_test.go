//go:build integration
// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// TestE2MCRUD tests the complete lifecycle of Events-to-Metrics (E2M)
func TestE2MCRUD(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var e2mID string

	// Test: Create E2M
	t.Run("CreateE2M", func(t *testing.T) {
		e2mName := GenerateUniqueName("e2m-crud")
		e2mConfig := map[string]interface{}{
			"name":        e2mName,
			"description": "Integration test E2M",
			"logs_query": map[string]interface{}{
				"lucene": "severity:error",
			},
			"metric_labels": []map[string]interface{}{
				{
					"target_label": "service",
					"source_field": "applicationName",
				},
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "error_count",
					"source_field":            "message",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "count",
							"target_metric_name": "total_errors",
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/events2metrics",
			Body:   e2mConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create E2M")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain E2M ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, e2mName, result["name"], "E2M name should match")

		// Save E2M ID for subsequent tests
		e2mID = result["id"].(string)
		AssertValidUUID(t, e2mID, "E2M ID should be a valid UUID")
	})

	// Test: Get E2M
	t.Run("GetE2M", func(t *testing.T) {
		require.NotEmpty(t, e2mID, "E2M ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/events2metrics/" + e2mID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get E2M")
		require.NotNil(t, result, "Response should not be nil")

		// Verify E2M details
		assert.Equal(t, e2mID, result["id"], "E2M ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Contains(t, result, "logs_query", "Response should contain logs_query")
		assert.Contains(t, result, "metric_fields", "Response should contain metric_fields")
	})

	// Test: List E2M
	t.Run("ListE2M", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/events2metrics",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list E2M")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "events2metrics", "Response should contain events2metrics array")
		e2ms, ok := result["events2metrics"].([]interface{})
		require.True(t, ok, "E2M should be an array")

		// Verify our created E2M is in the list
		found := false
		for _, e2m := range e2ms {
			e2mMap := e2m.(map[string]interface{})
			if e2mMap["id"] == e2mID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created E2M should be in the list")
	})

	// Test: Update E2M
	t.Run("UpdateE2M", func(t *testing.T) {
		require.NotEmpty(t, e2mID, "E2M ID should be set from create test")

		updatedDescription := "Updated integration test E2M"
		updateConfig := map[string]interface{}{
			"name":        "updated-" + GenerateUniqueName("e2m"),
			"description": updatedDescription,
			"logs_query": map[string]interface{}{
				"lucene": "severity:critical",
			},
			"metric_labels": []map[string]interface{}{
				{
					"target_label": "service",
					"source_field": "applicationName",
				},
				{
					"target_label": "environment",
					"source_field": "subsystemName",
				},
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "critical_count",
					"source_field":            "message",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "count",
							"target_metric_name": "total_critical",
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/events2metrics/" + e2mID,
			Body:   updateConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update E2M")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, e2mID, result["id"], "E2M ID should remain the same")
		assert.Equal(t, updatedDescription, result["description"], "Description should be updated")
	})

	// Test: Delete E2M
	t.Run("DeleteE2M", func(t *testing.T) {
		require.NotEmpty(t, e2mID, "E2M ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/events2metrics/" + e2mID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete E2M")

		// Verify E2M is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/events2metrics/" + e2mID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted E2M should return 404")
	})
}

// TestE2MWithAggregations tests E2M with different aggregation types
func TestE2MWithAggregations(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	testCases := []struct {
		name    string
		aggType string
		config  map[string]interface{}
	}{
		{
			name:    "E2MWithCountAggregation",
			aggType: "count",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("e2m-count"),
				"description": "E2M with count aggregation",
				"logs_query": map[string]interface{}{
					"lucene": "level:INFO",
				},
				"metric_labels": []map[string]interface{}{
					{
						"target_label": "app",
						"source_field": "applicationName",
					},
				},
				"metric_fields": []map[string]interface{}{
					{
						"target_base_metric_name": "info_logs",
						"source_field":            "message",
						"aggregations": []map[string]interface{}{
							{
								"enabled":            true,
								"agg_type":           "count",
								"target_metric_name": "total_info_logs",
							},
						},
					},
				},
				"type": "logs2metrics",
			},
		},
		{
			name:    "E2MWithSamplesAggregation",
			aggType: "samples",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("e2m-samples"),
				"description": "E2M with samples aggregation",
				"logs_query": map[string]interface{}{
					"lucene": "*",
				},
				"metric_labels": []map[string]interface{}{
					{
						"target_label": "service",
						"source_field": "applicationName",
					},
				},
				"metric_fields": []map[string]interface{}{
					{
						"target_base_metric_name": "log_samples",
						"source_field":            "timestamp",
						"aggregations": []map[string]interface{}{
							{
								"enabled":            true,
								"agg_type":           "samples",
								"target_metric_name": "log_sample_count",
								"samples": map[string]interface{}{
									"sample_type": "max",
								},
							},
						},
					},
				},
				"type": "logs2metrics",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := &client.Request{
				Method: "POST",
				Path:   "/v1/events2metrics",
				Body:   testCase.config,
			}

			result, err := tc.DoRequest(req)
			require.NoError(t, err, "Failed to create E2M with %s aggregation", testCase.aggType)
			require.NotNil(t, result, "Response should not be nil")

			e2mID := result["id"].(string)
			defer func() {
				// Cleanup
				deleteReq := &client.Request{
					Method: "DELETE",
					Path:   "/v1/events2metrics/" + e2mID,
				}
				tc.DoRequest(deleteReq)
			}()

			// Verify E2M was created with correct configuration
			assert.Equal(t, testCase.config["name"], result["name"], "Name should match")
			assert.Contains(t, result, "metric_fields", "Response should contain metric_fields")
		})
	}
}

// TestE2MWithHistogram tests E2M histogram aggregation
func TestE2MWithHistogram(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("CreateE2MWithHistogram", func(t *testing.T) {
		e2mConfig := map[string]interface{}{
			"name":        GenerateUniqueName("e2m-histogram"),
			"description": "E2M with histogram aggregation",
			"logs_query": map[string]interface{}{
				"lucene": "response_time:*",
			},
			"metric_labels": []map[string]interface{}{
				{
					"target_label": "endpoint",
					"source_field": "path",
				},
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "response_time",
					"source_field":            "response_time",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "histogram",
							"target_metric_name": "response_time_histogram",
							"histogram": map[string]interface{}{
								"buckets": []float64{0.1, 0.5, 1.0, 2.0, 5.0},
							},
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/events2metrics",
			Body:   e2mConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create E2M with histogram")
		require.NotNil(t, result, "Response should not be nil")

		e2mID := result["id"].(string)
		defer func() {
			// Cleanup
			deleteReq := &client.Request{
				Method: "DELETE",
				Path:   "/v1/events2metrics/" + e2mID,
			}
			tc.DoRequest(deleteReq)
		}()

		// Verify histogram configuration
		assert.Contains(t, result, "metric_fields", "Response should contain metric_fields")
	})
}

// TestE2MWithMultipleLabels tests E2M with multiple metric labels
func TestE2MWithMultipleLabels(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("CreateE2MWithMultipleLabels", func(t *testing.T) {
		e2mConfig := map[string]interface{}{
			"name":        GenerateUniqueName("e2m-multi-labels"),
			"description": "E2M with multiple metric labels",
			"logs_query": map[string]interface{}{
				"lucene": "*",
			},
			"metric_labels": []map[string]interface{}{
				{
					"target_label": "application",
					"source_field": "applicationName",
				},
				{
					"target_label": "subsystem",
					"source_field": "subsystemName",
				},
				{
					"target_label": "severity",
					"source_field": "severity",
				},
				{
					"target_label": "environment",
					"source_field": "env",
				},
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "log_count_by_labels",
					"source_field":            "message",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "count",
							"target_metric_name": "total_logs_labeled",
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/events2metrics",
			Body:   e2mConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create E2M with multiple labels")
		require.NotNil(t, result, "Response should not be nil")

		e2mID := result["id"].(string)
		defer func() {
			// Cleanup
			deleteReq := &client.Request{
				Method: "DELETE",
				Path:   "/v1/events2metrics/" + e2mID,
			}
			tc.DoRequest(deleteReq)
		}()

		// Verify all labels are present
		assert.Contains(t, result, "metric_labels", "Response should contain metric_labels")
		labels := result["metric_labels"].([]interface{})
		assert.Len(t, labels, 4, "Should have 4 metric labels")
	})
}

// TestE2MErrorHandling tests error scenarios for E2M
func TestE2MErrorHandling(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("GetNonExistentE2M", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/events2metrics/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error")
	})

	t.Run("CreateE2MWithInvalidQuery", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"name":        GenerateUniqueName("invalid-e2m"),
			"description": "Invalid E2M",
			"logs_query": map[string]interface{}{
				"lucene": "", // Empty query
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "test",
				},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/events2metrics",
			Body:   invalidConfig,
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error for invalid query")
	})

	t.Run("UpdateNonExistentE2M", func(t *testing.T) {
		updateConfig := map[string]interface{}{
			"name":        "test",
			"description": "test",
			"logs_query": map[string]interface{}{
				"lucene": "*",
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": "test",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "count",
							"target_metric_name": "test_count",
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/events2metrics/00000000-0000-0000-0000-000000000000",
			Body:   updateConfig,
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error for non-existent E2M")
	})
}

// TestE2MPagination tests pagination for listing E2M
func TestE2MPagination(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	// Create multiple E2Ms for pagination testing
	createdE2Ms := []string{}
	defer func() {
		// Cleanup created E2Ms
		for _, id := range createdE2Ms {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/events2metrics/" + id,
			}
			tc.DoRequest(req) // Ignore errors during cleanup
		}
	}()

	// Create 3 test E2Ms
	for i := 0; i < 3; i++ {
		e2mConfig := map[string]interface{}{
			"name":        GenerateUniqueName("e2m-pagination"),
			"description": "Pagination test E2M",
			"logs_query": map[string]interface{}{
				"lucene": "*",
			},
			"metric_labels": []map[string]interface{}{
				{
					"target_label": "app",
					"source_field": "applicationName",
				},
			},
			"metric_fields": []map[string]interface{}{
				{
					"target_base_metric_name": GenerateUniqueName("count"),
					"source_field":            "message",
					"aggregations": []map[string]interface{}{
						{
							"enabled":            true,
							"agg_type":           "count",
							"target_metric_name": GenerateUniqueName("total"),
						},
					},
				},
			},
			"type": "logs2metrics",
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/events2metrics",
			Body:   e2mConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create E2M")
		createdE2Ms = append(createdE2Ms, result["id"].(string))

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("ListAllE2M", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/events2metrics",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list E2M")

		e2ms, ok := result["events2metrics"].([]interface{})
		require.True(t, ok, "E2M should be an array")

		// Verify our created E2Ms are in the list
		foundCount := 0
		for _, e2m := range e2ms {
			e2mMap := e2m.(map[string]interface{})
			for _, createdID := range createdE2Ms {
				if e2mMap["id"] == createdID {
					foundCount++
				}
			}
		}
		assert.GreaterOrEqual(t, foundCount, 1, "At least one created E2M should be in the list")
	})
}
