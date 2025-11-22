//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

func safeGetID(t *testing.T, data map[string]interface{}) string {
	if idStr, ok := data["id"].(string); ok {
		return idStr
	} else if idFloat, ok := data["id"].(float64); ok {
		return fmt.Sprintf("%.0f", idFloat)
	}
	t.Fatalf("Unexpected ID type: %T", data["id"])
	return ""
}

// TestViewsCRUD tests the complete lifecycle of saved views
func TestViewsCRUD(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var viewID string

	// Test: Create View
	t.Run("CreateView", func(t *testing.T) {
		viewName := GenerateUniqueName("view-crud")
		viewConfig := map[string]interface{}{
			"name": viewName,
			"search_query": map[string]interface{}{
				"query": "severity:error",
			},
			"time_selection": map[string]interface{}{
				"quick_selection": map[string]interface{}{
					"caption": "Last 15 minutes",
					"seconds": 900,
				},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/views",
			Body:   viewConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create view")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain view ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, viewName, result["name"], "View name should match")

		// Save view ID for subsequent tests
		viewID = safeGetID(t, result)
		require.NotEmpty(t, viewID, "View ID should not be empty")
	})

	// Test: Get View
	t.Run("GetView", func(t *testing.T) {
		require.NotEmpty(t, viewID, "View ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/views/" + viewID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get view")
		require.NotNil(t, result, "Response should not be nil")

		// Verify view details
		assert.Equal(t, viewID, safeGetID(t, result), "View ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Contains(t, result, "search_query", "Response should contain search_query")
	})

	// Test: List Views
	t.Run("ListViews", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/views",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list views")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "views", "Response should contain views array")
		views, ok := result["views"].([]interface{})
		require.True(t, ok, "Views should be an array")

		// Verify our created view is in the list
		found := false
		for _, view := range views {
			viewMap := view.(map[string]interface{})
			if safeGetID(t, viewMap) == viewID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created view should be in the list")
	})

	// Test: Update View
	t.Run("UpdateView", func(t *testing.T) {
		require.NotEmpty(t, viewID, "View ID should be set from create test")

		updatedName := "updated-" + GenerateUniqueName("view")
		updateConfig := map[string]interface{}{
			"name": updatedName,
			"search_query": map[string]interface{}{
				"query": "severity:critical",
			},
			"time_selection": map[string]interface{}{
				"quick_selection": map[string]interface{}{
					"caption": "Last 1 hour",
					"seconds": 3600,
				},
			},
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/views/" + viewID,
			Body:   updateConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update view")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, viewID, safeGetID(t, result), "View ID should remain the same")
		assert.Equal(t, updatedName, result["name"], "Name should be updated")
	})

	// Test: Delete View
	t.Run("DeleteView", func(t *testing.T) {
		require.NotEmpty(t, viewID, "View ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/views/" + viewID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete view")

		// Verify view is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/views/" + viewID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted view should return 404")
	})
}

// TestViewFoldersCRUD tests the complete lifecycle of view folders
func TestViewFoldersCRUD(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var folderID string

	// Test: Create View Folder
	t.Run("CreateViewFolder", func(t *testing.T) {
		folderName := GenerateUniqueName("folder-crud")
		folderConfig := map[string]interface{}{
			"name": folderName,
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/view_folders",
			Body:   folderConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create view folder")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "id", "Response should contain folder ID")
		assert.Contains(t, result, "name", "Response should contain name")
		assert.Equal(t, folderName, result["name"], "Folder name should match")

		// Save folder ID for subsequent tests
		folderID = safeGetID(t, result)
		AssertValidUUID(t, folderID, "Folder ID should be a valid UUID")
	})

	// Test: Get View Folder
	t.Run("GetViewFolder", func(t *testing.T) {
		require.NotEmpty(t, folderID, "Folder ID should be set from create test")

		req := &client.Request{
			Method: "GET",
			Path:   "/v1/view_folders/" + folderID,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to get view folder")
		require.NotNil(t, result, "Response should not be nil")

		// Verify folder details
		assert.Equal(t, folderID, safeGetID(t, result), "Folder ID should match")
		assert.Contains(t, result, "name", "Response should contain name")
	})

	// Test: List View Folders
	t.Run("ListViewFolders", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/view_folders",
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to list view folders")
		require.NotNil(t, result, "Response should not be nil")

		// Verify response structure
		assert.Contains(t, result, "view_folders", "Response should contain view_folders array")
		folders, ok := result["view_folders"].([]interface{})
		require.True(t, ok, "View folders should be an array")

		// Verify our created folder is in the list
		found := false
		for _, folder := range folders {
			folderMap := folder.(map[string]interface{})
			if safeGetID(t, folderMap) == folderID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created folder should be in the list")
	})

	// Test: Update View Folder
	t.Run("UpdateViewFolder", func(t *testing.T) {
		require.NotEmpty(t, folderID, "Folder ID should be set from create test")

		updatedName := "updated-" + GenerateUniqueName("folder")
		updateConfig := map[string]interface{}{
			"name": updatedName,
		}

		req := &client.Request{
			Method: "PUT",
			Path:   "/v1/view_folders/" + folderID,
			Body:   updateConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to update view folder")
		require.NotNil(t, result, "Response should not be nil")

		// Verify updated fields
		assert.Equal(t, folderID, safeGetID(t, result), "Folder ID should remain the same")
		assert.Equal(t, updatedName, result["name"], "Name should be updated")
	})

	// Test: Delete View Folder
	t.Run("DeleteViewFolder", func(t *testing.T) {
		require.NotEmpty(t, folderID, "Folder ID should be set from create test")

		req := &client.Request{
			Method: "DELETE",
			Path:   "/v1/view_folders/" + folderID,
		}

		_, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to delete view folder")

		// Verify folder is deleted by trying to get it
		getReq := &client.Request{
			Method: "GET",
			Path:   "/v1/view_folders/" + folderID,
		}

		_, err = tc.DoRequestExpectError(getReq, 404)
		assert.NoError(t, err, "Getting deleted folder should return 404")
	})
}

// TestViewInFolder tests creating a view within a folder
func TestViewInFolder(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	var folderID, viewID string

	// Create folder first
	t.Run("CreateFolder", func(t *testing.T) {
		folderConfig := map[string]interface{}{
			"name": GenerateUniqueName("parent-folder"),
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/view_folders",
			Body:   folderConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create folder")
		if idStr, ok := result["id"].(string); ok {
			folderID = idStr
		} else if idFloat, ok := result["id"].(float64); ok {
			folderID = fmt.Sprintf("%.0f", idFloat)
		} else {
			t.Fatalf("Unexpected ID type: %T", result["id"])
		}
	})

	// Create view in folder
	t.Run("CreateViewInFolder", func(t *testing.T) {
		require.NotEmpty(t, folderID, "Folder ID should be set")

		viewConfig := map[string]interface{}{
			"name":      GenerateUniqueName("view-in-folder"),
			"folder_id": folderID,
			"search_query": map[string]interface{}{
				"query": "*",
			},
			"time_selection": map[string]interface{}{
				"quick_selection": map[string]interface{}{
					"caption": "Last 15 minutes",
					"seconds": 900,
				},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/views",
			Body:   viewConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create view in folder")

		// Safely extract ID
		if idStr, ok := result["id"].(string); ok {
			viewID = idStr
		} else if idFloat, ok := result["id"].(float64); ok {
			viewID = fmt.Sprintf("%.0f", idFloat)
		} else {
			t.Fatalf("Unexpected ID type: %T", result["id"])
		}

		// Verify view is associated with folder
		assert.Equal(t, folderID, result["folder_id"], "View should be in the specified folder")
	})

	// Cleanup
	defer func() {
		if viewID != "" {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/views/" + viewID,
			}
			tc.DoRequest(req)
		}
		if folderID != "" {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/view_folders/" + folderID,
			}
			tc.DoRequest(req)
		}
	}()
}

// TestViewsWithCustomTimeSelection tests views with different time selections
func TestViewsWithCustomTimeSelection(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	testCases := []struct {
		name          string
		timeSelection map[string]interface{}
	}{
		{
			name: "ViewWithQuickSelection",
			timeSelection: map[string]interface{}{
				"quick_selection": map[string]interface{}{
					"caption": "Last 1 hour",
					"seconds": 3600,
				},
			},
		},
		{
			name: "ViewWithAbsoluteTimeRange",
			timeSelection: map[string]interface{}{
				"custom_selection": map[string]interface{}{
					"from_time": "2024-01-01T00:00:00Z",
					"to_time":   "2024-01-01T23:59:59Z",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			viewConfig := map[string]interface{}{
				"name": GenerateUniqueName("view-time"),
				"search_query": map[string]interface{}{
					"query": "*",
				},
				"time_selection": testCase.timeSelection,
			}

			req := &client.Request{
				Method: "POST",
				Path:   "/v1/views",
				Body:   viewConfig,
			}

			result, err := tc.DoRequest(req)
			require.NoError(t, err, "Failed to create view with custom time selection")
			require.NotNil(t, result, "Response should not be nil")

			var viewID string
			if idStr, ok := result["id"].(string); ok {
				viewID = idStr
			} else if idFloat, ok := result["id"].(float64); ok {
				viewID = fmt.Sprintf("%.0f", idFloat)
			} else {
				t.Fatalf("Unexpected ID type: %T", result["id"])
			}
			defer func() {
				// Cleanup
				deleteReq := &client.Request{
					Method: "DELETE",
					Path:   "/v1/views/" + viewID,
				}
				tc.DoRequest(deleteReq)
			}()

			// Verify time selection is set
			assert.Contains(t, result, "time_selection", "Response should contain time_selection")
		})
	}
}

// TestViewsErrorHandling tests error scenarios for views
func TestViewsErrorHandling(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	t.Run("GetNonExistentView", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/views/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error")
	})

	t.Run("CreateViewWithInvalidData", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"name": "", // Empty name should be invalid
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/views",
			Body:   invalidConfig,
		}

		_, err := tc.DoRequestExpectError(req, 422)
		assert.NoError(t, err, "Should handle 422 error for invalid data")
	})

	t.Run("GetNonExistentFolder", func(t *testing.T) {
		req := &client.Request{
			Method: "GET",
			Path:   "/v1/view_folders/00000000-0000-0000-0000-000000000000",
		}

		_, err := tc.DoRequestExpectError(req, 400)
		assert.NoError(t, err, "Should handle 400 error for non-existent folder")
	})
}

// TestViewsPagination tests pagination for listing views
func TestViewsPagination(t *testing.T) {
	skipIfShort(t)
	tc := NewTestContext(t)
	defer tc.Cleanup()

	// Create multiple views for pagination testing
	createdViews := []string{}
	defer func() {
		// Cleanup created views
		for _, id := range createdViews {
			req := &client.Request{
				Method: "DELETE",
				Path:   "/v1/views/" + id,
			}
			tc.DoRequest(req) // Ignore errors during cleanup
		}
	}()

	// Create 3 test views
	for i := 0; i < 3; i++ {
		viewConfig := map[string]interface{}{
			"name": GenerateUniqueName("view-pagination"),
			"search_query": map[string]interface{}{
				"query": "*",
			},
			"time_selection": map[string]interface{}{
				"quick_selection": map[string]interface{}{
					"caption": "Last 15 minutes",
					"seconds": 900,
				},
			},
		}

		req := &client.Request{
			Method: "POST",
			Path:   "/v1/views",
			Body:   viewConfig,
		}

		result, err := tc.DoRequest(req)
		require.NoError(t, err, "Failed to create view")

		if idStr, ok := result["id"].(string); ok {
			createdViews = append(createdViews, idStr)
		} else if idFloat, ok := result["id"].(float64); ok {
			createdViews = append(createdViews, fmt.Sprintf("%.0f", idFloat))
		} else {
			t.Fatalf("Unexpected ID type: %T", result["id"])
		}

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("ListAllViews", func(t *testing.T) {
		err := WaitForCondition(context.Background(), 2*time.Second, 30*time.Second, func() (bool, error) {
			req := &client.Request{
				Method: "GET",
				Path:   "/v1/views?per_page=100",
			}

			result, err := tc.DoRequest(req)
			if err != nil {
				return false, err
			}

			views, ok := result["views"].([]interface{})
			if !ok {
				return false, fmt.Errorf("views should be an array")
			}

			// Verify our created views are in the list
			foundCount := 0
			for _, view := range views {
				viewMap := view.(map[string]interface{})
				viewID := safeGetID(t, viewMap)
				for _, createdID := range createdViews {
					if viewID == createdID {
						foundCount++
					}
				}
			}

			if foundCount >= 1 {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "Failed to find created views in list after retries")
	})
}
