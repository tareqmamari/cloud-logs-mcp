package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPMetadata(t *testing.T) {
	metadata := NewMCPMetadata("query_logs")

	assert.Equal(t, "query_logs", metadata.ToolName)
	assert.Equal(t, NamespaceQuery, metadata.ToolNamespace)
	assert.False(t, metadata.Timestamp.IsZero())
}

func TestMCPMetadataChaining(t *testing.T) {
	metadata := NewMCPMetadata("create_alert").
		WithExecutionTime(100 * time.Millisecond).
		WithQuery(&QueryMetadata{
			Syntax: "dataprime",
			Tier:   "archive",
		}).
		AddCorrection("Fixed numeric severity").
		AddWarning("Large result set").
		AddHint("Consider using summary_only=true")

	assert.Equal(t, 100*time.Millisecond, metadata.ExecutionTime)
	assert.NotNil(t, metadata.Query)
	assert.Equal(t, "dataprime", metadata.Query.Syntax)
	assert.Len(t, metadata.Corrections, 1)
	assert.Len(t, metadata.Warnings, 1)
	assert.Len(t, metadata.Hints, 1)
}

func TestMCPMetadataToMap(t *testing.T) {
	metadata := NewMCPMetadata("query_logs").
		WithExecutionTime(500 * time.Millisecond).
		WithQuery(&QueryMetadata{
			Syntax:    "dataprime",
			Tier:      "archive",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-01-02T00:00:00Z",
			Limit:     100,
		}).
		WithPagination(&PaginationMetadata{
			TotalCount:    1000,
			ReturnedCount: 100,
			IsTruncated:   true,
			HasMore:       true,
		}).
		AddCorrection("Auto-corrected severity")

	m := metadata.ToMap()

	t.Run("has basic fields", func(t *testing.T) {
		assert.Equal(t, "query_logs", m["tool_name"])
		assert.Equal(t, "queries", m["tool_namespace"])
		assert.NotEmpty(t, m["timestamp"])
	})

	t.Run("has execution time", func(t *testing.T) {
		assert.Equal(t, int64(500), m["execution_time_ms"])
	})

	t.Run("has query metadata", func(t *testing.T) {
		query, ok := m["query"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "dataprime", query["syntax"])
		assert.Equal(t, "archive", query["tier"])
		assert.Equal(t, 100, query["limit"])
	})

	t.Run("has pagination metadata", func(t *testing.T) {
		pagination, ok := m["pagination"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 1000, pagination["total_count"])
		assert.Equal(t, 100, pagination["returned_count"])
		assert.Equal(t, true, pagination["is_truncated"])
		assert.Equal(t, true, pagination["has_more"])
	})

	t.Run("has corrections", func(t *testing.T) {
		corrections, ok := m["corrections"].([]string)
		require.True(t, ok)
		assert.Contains(t, corrections, "Auto-corrected severity")
	})
}

func TestAddMCPMetadataToResult(t *testing.T) {
	result := map[string]interface{}{
		"events": []interface{}{"event1", "event2"},
	}

	metadata := NewMCPMetadata("query_logs")
	AddMCPMetadataToResult(result, metadata)

	assert.Contains(t, result, "_mcp_metadata")
	mcpMeta, ok := result["_mcp_metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "query_logs", mcpMeta["tool_name"])
}

func TestCreatePaginationFromTruncation(t *testing.T) {
	pagination := CreatePaginationFromTruncation(1000, 100, "events")

	assert.Equal(t, 1000, pagination.TotalCount)
	assert.Equal(t, 100, pagination.ReturnedCount)
	assert.True(t, pagination.IsTruncated)
	assert.Equal(t, "events", pagination.TruncatedField)
	assert.True(t, pagination.HasMore)
	assert.NotEmpty(t, pagination.PaginationHint)
}

func TestMigrateLegacyMetadata(t *testing.T) {
	t.Run("migrates _query_metadata", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{},
			"_query_metadata": map[string]interface{}{
				"syntax":     "dataprime",
				"tier":       "archive",
				"start_date": "2024-01-01T00:00:00Z",
				"end_date":   "2024-01-02T00:00:00Z",
				"limit":      float64(100),
				"auto_corrections": []interface{}{
					"Fixed severity",
				},
			},
		}

		MigrateLegacyMetadata(result, "query_logs")

		// Old field should be removed
		assert.NotContains(t, result, "_query_metadata")

		// New field should exist
		assert.Contains(t, result, "_mcp_metadata")
		mcpMeta, ok := result["_mcp_metadata"].(map[string]interface{})
		require.True(t, ok)

		// Query info should be present
		query, ok := mcpMeta["query"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "dataprime", query["syntax"])
		assert.Equal(t, "archive", query["tier"])

		// Corrections should be migrated
		corrections, ok := mcpMeta["corrections"].([]string)
		require.True(t, ok)
		assert.Contains(t, corrections, "Fixed severity")
	})

	t.Run("migrates _truncated_info", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{},
			"_truncated_info": map[string]interface{}{
				"field":          "events",
				"original_count": float64(1000),
				"shown_count":    float64(100),
			},
		}

		MigrateLegacyMetadata(result, "list_alerts")

		// Old field should be removed
		assert.NotContains(t, result, "_truncated_info")

		// New field should have pagination
		mcpMeta, ok := result["_mcp_metadata"].(map[string]interface{})
		require.True(t, ok)

		pagination, ok := mcpMeta["pagination"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 1000, pagination["total_count"])
		assert.Equal(t, 100, pagination["returned_count"])
		assert.Equal(t, true, pagination["is_truncated"])
	})
}

func TestMCPMetadataWithRateLimit(t *testing.T) {
	metadata := NewMCPMetadata("query_logs").
		WithRateLimit(&RateLimitMetadata{
			Limit:     100,
			Remaining: 95,
			Available: 95.0,
		})

	m := metadata.ToMap()

	rateLimit, ok := m["rate_limit"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 100, rateLimit["limit"])
	assert.Equal(t, 95, rateLimit["remaining"])
}

func TestMCPMetadataWithCache(t *testing.T) {
	cachedAt := time.Now().Add(-5 * time.Minute)
	metadata := NewMCPMetadata("list_alerts").
		WithCache(&CacheMetadata{
			Hit:      true,
			Key:      "alerts:user123",
			TTL:      5 * time.Minute,
			CachedAt: cachedAt,
		})

	m := metadata.ToMap()

	cache, ok := m["cache"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, cache["hit"])
	assert.Equal(t, "alerts:user123", cache["key"])
	assert.Equal(t, 300, cache["ttl_seconds"])
}
