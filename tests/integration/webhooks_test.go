//go:build integration

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// TestWebhooksCRUD tests the complete lifecycle of outgoing webhooks
func TestWebhooksCRUD(t *testing.T) {
	skipIfShort(t)
	t.Skip("Skipping TestWebhooksCRUD: generic webhooks not supported in current API")
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var webhookID string

	// Test: Create Webhook
	t.Run("CreateWebhook", func(t *testing.T) {
		webhookName := GenerateUniqueName("webhook-crud")
		webhookConfig := map[string]interface{}{
			"type": "generic",
			"name": webhookName,
			"url":  "https://example.com/webhook",
			"generic_webhook": map[string]interface{}{
				"method": "POST",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
				"payload": "{\"message\": \"{{message}}\"}",
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/outgoing_webhooks",
			Body:   webhookConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create webhook")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain webhook ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, webhookName, result["name"], "Webhook name should match")

		// Save webhook ID for subsequent tests
		webhookID = result["id"].(string)
		AssertValidUUID(t, webhookID, "Webhook ID should be a valid UUID")
	})

	// Test: Get Webhook
	t.Run("GetWebhook", func(t *testing.T) {
		require.NotEmpty(t, webhookID, "Webhook ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/outgoing_webhooks/" + webhookID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get webhook")
		require.NotNil(t, result, "Response should not be nil")

		// Verify webhook details
		assert.Equal(t, webhookID, result["id"], "Webhook ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Contains(t, result, "url", "Response should contain URL")
	})

	// Test: List Webhooks
	t.Run("ListWebhooks", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/outgoing_webhooks",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list webhooks")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "outgoing_webhooks", "Response should contain outgoing_webhooks array")
		webhooks, ok := result["outgoing_webhooks"].([]interface{})
		require.True(t, ok, "Webhooks should be an array")

		// Verify our created webhook is in the list
		found := false
		for _, webhook := range webhooks {
			webhookMap := webhook.(map[string]interface{})
			if webhookMap["id"] == webhookID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created webhook should be in the list")
	})

	// Test: Update Webhook
	t.Run("UpdateWebhook", func(t *testing.T) {
		require.NotEmpty(t, webhookID, "Webhook ID should be set from create test")

		updatedName := "updated-" + GenerateUniqueName("webhook")
		updateConfig := map[string]interface{}{
			"type": "generic",
			"name": updatedName,
			"url":  "https://example.com/webhook/updated",
			"generic_webhook": map[string]interface{}{
				"uuid":   "550e8400-e29b-41d4-a716-446655440000",
				"method": "put",
				"headers": map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token",
				},
				"payload": "{\"updated\": \"{{message}}\"}",
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/outgoing_webhooks/" + webhookID,
			Body:   updateConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update webhook")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, webhookID, result["id"], "Webhook ID should remain the same")
		assert.Equal(t, updatedName, result["name"], "Name should be updated")
	})

	// Test: Delete Webhook
	t.Run("DeleteWebhook", func(t *testing.T) {
		require.NotEmpty(t, webhookID, "Webhook ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/outgoing_webhooks/" + webhookID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete webhook")

		// Verify webhook is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/outgoing_webhooks/" + webhookID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted webhook should return 404")
	})
}

// TestWebhookTypes tests different webhook types
func TestWebhookTypes(t *testing.T) {
	skipIfShort(t)
	t.Skip("Skipping TestWebhookTypes: generic/slack/pagerduty webhooks not supported in current API")
	tc := NewTestContext(t)
	defer tc.Cleanup()

	testCases := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name: "GenericWebhook",
			config: map[string]interface{}{
				"type": "generic",
				"name": GenerateUniqueName("webhook-generic"),
				"url":  "https://example.com/generic",
				"generic_webhook": map[string]interface{}{
					"uuid":   "550e8400-e29b-41d4-a716-446655440000",
					"method": "post",
					"headers": map[string]string{
						"Content-Type": "application/json",
					},
					"payload": "{\"event\": \"{{event}}\"}",
				},
			},
		},
		{
			name: "SlackWebhook",
			config: map[string]interface{}{
				"type": "slack",
				"name": GenerateUniqueName("webhook-slack"),
				"url":  "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
				"slack": map[string]interface{}{
					"digests": []map[string]interface{}{
						{
							"type":      "error_and_critical_logs",
							"is_active": true,
						},
					},
					"attachments": []map[string]interface{}{
						{
							"type":      "metric_snapshot",
							"is_active": true,
						},
					},
				},
			},
		},
		{
			name: "PagerDutyWebhook",
			config: map[string]interface{}{
				"type": "pager_duty",
				"name": GenerateUniqueName("webhook-pagerduty"),
				"url":  "https://events.pagerduty.com/integration/key/enqueue",
				"pager_duty": map[string]interface{}{
					"service_key": "your-service-key-here", // pragma: allowlist secret
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := &client.Request{
				Method: "POST",
				Path:   "/v1/outgoing_webhooks",
				Body:   testCase.config,
			}

			result, err := tc.DoRequest(req)
			require.NoError(t, err, "Failed to create %s webhook", testCase.name)
			require.NotNil(t, result, "Response should not be nil")

			webhookID := result["id"].(string)
			defer func() {
				// Cleanup
				deleteReq := &client.Request{
					Method: "DELETE",
					Path:   "/v1/outgoing_webhooks/" + webhookID,
				}
				tc.DoRequest(deleteReq)
			}()

			// Verify webhook was created with correct type
			assert.Equal(t, testCase.config["type"], result["type"], "Webhook type should match")
			assert.Equal(t, testCase.config["name"], result["name"], "Webhook name should match")
		})
	}
}

// TestIBMEventNotificationsWebhook tests IBM Event Notifications integration
func TestIBMEventNotificationsWebhook(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	// Get Event Notifications instance ID from environment
	eventNotificationsInstanceID := os.Getenv("EVENT_NOTIFICATIONS_INSTANCE_ID")
	if eventNotificationsInstanceID == "" {
		t.Skip("Skipping TestIBMEventNotificationsWebhook: EVENT_NOTIFICATIONS_INSTANCE_ID environment variable not set")
	}

	t.Run("CreateIBMEventNotificationsWebhook", func(t *testing.T) {
		webhookConfig := map[string]interface{}{
			"type": "ibm_event_notifications",
			"name": GenerateUniqueName("webhook-ibm-en"),
			"ibm_event_notifications": map[string]interface{}{
				"event_notifications_instance_id": eventNotificationsInstanceID,
				"region_id":                       tc.Config.Region, // Use the same region as the Logs instance
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/outgoing_webhooks",
			Body:   webhookConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create IBM Event Notifications webhook")
		require.NotNil(t, result, "Response should not be nil")

		webhookID := result["id"].(string)
		defer func() {
			// Cleanup
			deleteReq := &client.Request{
				Method: "DELETE",
				Path:   "/v1/outgoing_webhooks/" + webhookID,
			}
			tc.DoRequest(deleteReq)
		}()

		// Verify IBM Event Notifications configuration
		assert.Equal(t, "ibm_event_notifications", result["type"], "Type should be ibm_event_notifications")
		assert.Contains(t, result, "ibm_event_notifications", "Response should contain ibm_event_notifications config")
	})
}

// TestWebhooksErrorHandling tests error scenarios for webhooks
func TestWebhooksErrorHandling(t *testing.T) {
	skipIfShort(t)
	t.Skip("Skipping TestWebhooksErrorHandling: generic webhooks not supported in current API")
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("GetNonExistentWebhook", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/outgoing_webhooks/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error")
	})

	t.Run("CreateWebhookWithInvalidURL", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"type": "generic",
			"name": GenerateUniqueName("invalid-webhook"),
			"url":  "not-a-valid-url",
			"generic_webhook": map[string]interface{}{
				"uuid":   "550e8400-e29b-41d4-a716-446655440000",
				"method": "post",
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/outgoing_webhooks",
			Body:   invalidConfig,
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error for invalid URL")
	})

	t.Run("UpdateNonExistentWebhook", func(t *testing.T) {
		updateConfig := map[string]interface{}{
			"type": "generic",
			"name": "test",
			"url":  "https://example.com/test",
			"generic_webhook": map[string]interface{}{
				"uuid":   "550e8400-e29b-41d4-a716-446655440000",
				"method": "post",
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/outgoing_webhooks/00000000-0000-0000-0000-000000000000",
			Body:   updateConfig,
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error for non-existent webhook")
	})

	t.Run("DeleteNonExistentWebhook", func(t *testing.T) {
		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/outgoing_webhooks/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error for non-existent webhook")
	})
}

// TestWebhooksPagination tests pagination for listing webhooks
func TestWebhooksPagination(t *testing.T) {
	skipIfShort(t)
	t.Skip("Skipping TestWebhooksPagination: generic webhooks not supported in current API")
	tc := NewTestContext(t)
	defer tc.Cleanup()

	// Create multiple webhooks for pagination testing
	createdWebhooks := []string{}
	defer func() {
		// Cleanup created webhooks
		for _, id := range createdWebhooks {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/outgoing_webhooks/" + id,
			}
			tc.DoRequest(req) // Ignore errors during cleanup
		}
	}()

	// Create 3 test webhooks
	for i := 0; i < 3; i++ {
		webhookConfig := map[string]interface{}{
			"type": "generic",
			"name": GenerateUniqueName("webhook-pagination"),
			"url":  "https://example.com/webhook",
			"generic_webhook": map[string]interface{}{
				"uuid":   "550e8400-e29b-41d4-a716-446655440000",
				"method": "post",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
				"payload": "{\"message\": \"test\"}",
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/outgoing_webhooks",
			Body:   webhookConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create webhook")
		createdWebhooks = append(createdWebhooks, result["id"].(string))

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("ListAllWebhooks", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/outgoing_webhooks",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list webhooks")

		webhooks, ok := result["outgoing_webhooks"].([]interface{})
		require.True(t, ok, "Webhooks should be an array")

		// Verify our created webhooks are in the list
		foundCount := 0
		for _, webhook := range webhooks {
			webhookMap := webhook.(map[string]interface{})
			for _, createdID := range createdWebhooks {
				if webhookMap["id"] == createdID {
					foundCount++
				}
			}
		}
		assert.GreaterOrEqual(t, foundCount, 1, "At least one created webhook should be in the list")
	})
}

// TestWebhookWithCustomHeaders tests webhooks with custom headers
func TestWebhookWithCustomHeaders(t *testing.T) {
	skipIfShort(t)
	t.Skip("Skipping TestWebhookWithCustomHeaders: generic webhooks not supported in current API")
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("CreateWebhookWithCustomHeaders", func(t *testing.T) {
		webhookConfig := map[string]interface{}{
			"type": "generic",
			"name": GenerateUniqueName("webhook-headers"),
			"url":  "https://example.com/webhook",
			"generic_webhook": map[string]interface{}{
				"uuid":   "550e8400-e29b-41d4-a716-446655440000",
				"method": "post",
				"headers": map[string]string{
					"Content-Type":    "application/json",
					"Authorization":   "Bearer secret-token",
					"X-Custom-Header": "custom-value",
					"X-Request-ID":    "{{request_id}}",
				},
				"payload": "{\"message\": \"{{message}}\", \"severity\": \"{{severity}}\"}",
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/outgoing_webhooks",
			Body:   webhookConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create webhook with custom headers")
		require.NotNil(t, result, "Response should not be nil")

		webhookID := result["id"].(string)
		defer func() {
			// Cleanup
			deleteReq := &client.Request{
				Method: "DELETE",
				Path:   "/v1/outgoing_webhooks/" + webhookID,
			}
			tc.DoRequest(deleteReq)
		}()

		// Verify webhook was created with headers
		assert.Contains(t, result, "generic_webhook", "Response should contain generic_webhook config")
	})
}
