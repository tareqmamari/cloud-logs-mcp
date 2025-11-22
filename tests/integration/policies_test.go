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

// TestPoliciesCRUD tests the complete lifecycle of data pipeline policies
func TestPoliciesCRUD(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var policyID string

	// Test: Create Policy
	t.Run("CreatePolicy", func(t *testing.T) {
		policyName := GenerateUniqueName("policy-crud")
		policyConfig := map[string]interface{}{
			"name":        policyName,
			"description": "Integration test policy",
			"priority":    "type_high",
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "test-app",
			},
			"subsystem_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "test-subsystem",
			},
			"log_rules": map[string]interface{}{
				"severities": []string{"info", "warning", "error"},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/policies",
			Body:   policyConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create policy")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain policy ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, policyName, result["name"], "Policy name should match")

		// Save policy ID for subsequent tests
		policyID = result["id"].(string)
		AssertValidUUID(t, policyID, "Policy ID should be a valid UUID")
	})

	// Test: Get Policy
	t.Run("GetPolicy", func(t *testing.T) {
		require.NotEmpty(t, policyID, "Policy ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/policies/" + policyID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get policy")
		require.NotNil(t, result, "Response should not be nil")

		// Verify policy details
		assert.Equal(t, policyID, result["id"], "Policy ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Contains(t, result, "priority", "Response should contain priority")
	})

	// Test: List Policies
	t.Run("ListPolicies", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/policies",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list policies")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "policies", "Response should contain policies array")
		policies, ok := result["policies"].([]interface{})
		require.True(t, ok, "Policies should be an array")

		// Verify our created policy is in the list
		found := false
		for _, policy := range policies {
			policyMap := policy.(map[string]interface{})
			if policyMap["id"] == policyID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created policy should be in the list")
	})

	// Test: Update Policy
	t.Run("UpdatePolicy", func(t *testing.T) {
		require.NotEmpty(t, policyID, "Policy ID should be set from create test")

		updatedDescription := "Updated integration test policy"
		updateConfig := map[string]interface{}{
			"name":        "updated-" + GenerateUniqueName("policy"),
			"description": updatedDescription,
			"priority":    "type_medium",
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "updated-app",
			},
			"subsystem_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "updated-subsystem",
			},
			"log_rules": map[string]interface{}{
				"severities": []string{"error", "critical"},
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/policies/" + policyID,
			Body:   updateConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update policy")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, policyID, result["id"], "Policy ID should remain the same")
		assert.Equal(t, updatedDescription, result["description"], "Description should be updated")
		assert.Equal(t, "type_medium", result["priority"], "Priority should be updated")
	})

	// Test: Delete Policy
	t.Run("DeletePolicy", func(t *testing.T) {
		require.NotEmpty(t, policyID, "Policy ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/policies/" + policyID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete policy")

		// Verify policy is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/policies/" + policyID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted policy should return 404")
	})
}

// TestPoliciesWithArchiveRetention tests policies with archive retention configuration
func TestPoliciesWithArchiveRetention(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("CreatePolicyWithArchiveRetention", func(t *testing.T) {
		t.Skip("Skipping test requiring valid archive retention ID")
		policyConfig := map[string]interface{}{
			"name":        GenerateUniqueName("policy-archive"),
			"description": "Policy with archive retention",
			"priority":    "type_high",
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "archive-app",
			},
			"archive_retention": map[string]interface{}{
				"id": "3d9a5b88-f344-47f2-893a-580e50d4f7b8",
			},
			"log_rules": map[string]interface{}{
				"severities": []string{"info", "warning", "error"},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/policies",
			Body:   policyConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create policy with archive retention")
		require.NotNil(t, result, "Response should not be nil")

		policyID := result["id"].(string)
		defer func() {
			// Cleanup
			deleteReq := &client.Request{
				Method: "DELETE",
				Path:   "/v1/policies/" + policyID,
			}
			tc.DoRequest(deleteReq)
		}()

		// Verify archive retention is set
		assert.Contains(t, result, "archive_retention", "Response should contain archive_retention")
	})
}

// TestPoliciesPriority tests policy priority ordering
func TestPoliciesPriority(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	createdPolicies := []string{}
	defer func() {
		// Cleanup created policies
		for _, id := range createdPolicies {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/policies/" + id,
			}
			tc.DoRequest(req) // Ignore errors during cleanup
		}
	}()

	priorities := []string{"type_low", "type_medium", "type_high"}

	// Create policies with different priorities
	for i, priority := range priorities {
		policyConfig := map[string]interface{}{
			"name":        GenerateUniqueName("policy-priority"),
			"description": "Priority test policy",
			"priority":    priority,
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "priority-app",
			},
			"subsystem_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "priority-subsystem",
			},
			"log_rules": map[string]interface{}{
				"severities": []string{"info", "warning", "error"},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/policies",
			Body:   policyConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create policy with priority %s", priority)
		createdPolicies = append(createdPolicies, result["id"].(string))

		// Small delay to avoid rate limiting
		if i < len(priorities)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// List policies and verify they exist
	t.Run("VerifyPoliciesCreated", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/policies",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list policies")

		policies, ok := result["policies"].([]interface{})
		require.True(t, ok, "Policies should be an array")

		foundCount := 0
		for _, policy := range policies {
			policyMap := policy.(map[string]interface{})
			for _, createdID := range createdPolicies {
				if policyMap["id"] == createdID {
					foundCount++
				}
			}
		}
		assert.Equal(t, len(createdPolicies), foundCount, "All created policies should be in the list")
	})
}

// TestPoliciesWithRuleMatchers tests various rule matcher configurations
func TestPoliciesWithRuleMatchers(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	testCases := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name: "PolicyWithStartsWith",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("policy-starts-with"),
				"description": "Policy with starts_with matcher",
				"priority":    "type_medium",
				"application_rule": map[string]interface{}{
					"rule_type_id": "start_with",
					"name":         "test-",
				},
				"log_rules": map[string]interface{}{
					"severities": []string{"info", "warning", "error"},
				},
			},
		},

		{
			name: "PolicyWithIncludes",
			config: map[string]interface{}{
				"name":        GenerateUniqueName("policy-includes"),
				"description": "Policy with includes matcher",
				"priority":    "type_medium",
				"application_rule": map[string]interface{}{
					"rule_type_id": "includes",
					"name":         "prod",
				},
				"log_rules": map[string]interface{}{
					"severities": []string{"info", "warning", "error"},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := &client.Request{
				Method: "POST",
				Path:   "/v1/policies",
				Body:   testCase.config,
			}

			result, err := tc.DoRequest(req)
			require.NoError(t, err, "Failed to create policy with rule matcher")
			require.NotNil(t, result, "Response should not be nil")

			policyID := result["id"].(string)
			defer func() {
				// Cleanup
				deleteReq := &client.Request{
					Method: "DELETE",
					Path:   "/v1/policies/" + policyID,
				}
				tc.DoRequest(deleteReq)
			}()

			// Verify policy was created with correct configuration
			assert.Equal(t, testCase.config["name"], result["name"], "Name should match")
			assert.Contains(t, result, "application_rule", "Response should contain application_rule")
		})
	}
}

// TestPoliciesErrorHandling tests error scenarios for policies
func TestPoliciesErrorHandling(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("GetNonExistentPolicy", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/policies/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 404)
		assert.NoError(t, err, "Should handle 404 error")
	})

	t.Run("CreatePolicyWithInvalidPriority", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"name":        GenerateUniqueName("invalid-policy"),
			"description": "Invalid policy",
			"priority":    "invalid_priority",
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "test",
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/policies",
			Body:   invalidConfig,
		}

		_, err := tc.DoRequestExpectError(req, 422)
		assert.NoError(t, err, "Should handle 422 error for invalid priority")
	})

	t.Run("UpdateNonExistentPolicy", func(t *testing.T) {
		updateConfig := map[string]interface{}{
			"name":        "test",
			"description": "test",
			"priority":    "type_medium",
			"application_rule": map[string]interface{}{
				"rule_type_id": "is",
				"name":         "test",
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/policies/00000000-0000-0000-0000-000000000000",
			Body:   updateConfig,
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error for non-existent policy")
	})
}
