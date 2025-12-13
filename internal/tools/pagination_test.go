package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	t.Run("time cursor roundtrip", func(t *testing.T) {
		ts := time.Now().UTC()
		original := CreateTimeCursor(ts, 100, "forward")

		encoded := EncodeCursor(original)
		assert.NotEmpty(t, encoded)

		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, CursorTypeTime, decoded.Type)
		assert.Equal(t, ts.Format(time.RFC3339Nano), decoded.Timestamp)
		assert.Equal(t, 100, decoded.Limit)
		assert.Equal(t, "forward", decoded.Direction)
	})

	t.Run("offset cursor roundtrip", func(t *testing.T) {
		original := CreateOffsetCursor(500, 100)

		encoded := EncodeCursor(original)
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, CursorTypeOffset, decoded.Type)
		assert.Equal(t, 500, decoded.Offset)
		assert.Equal(t, 100, decoded.Limit)
	})

	t.Run("ID cursor roundtrip", func(t *testing.T) {
		original := CreateIDCursor("event-123-456", 50, "backward")

		encoded := EncodeCursor(original)
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, CursorTypeID, decoded.Type)
		assert.Equal(t, "event-123-456", decoded.LastID)
		assert.Equal(t, 50, decoded.Limit)
		assert.Equal(t, "backward", decoded.Direction)
	})

	t.Run("invalid cursor returns error", func(t *testing.T) {
		_, err := DecodeCursor("not-valid-base64!")
		assert.Error(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		// Valid base64 but invalid JSON
		_, err := DecodeCursor("bm90LWpzb24=") // "not-json" in base64
		assert.Error(t, err)
	})
}

func TestExtractLastTimestamp(t *testing.T) {
	t.Run("extracts from timestamp field", func(t *testing.T) {
		events := []interface{}{
			map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z"},
			map[string]interface{}{"timestamp": "2024-01-01T12:00:00Z"},
		}

		ts, err := ExtractLastTimestamp(events)
		require.NoError(t, err)
		assert.Equal(t, 2024, ts.Year())
		assert.Equal(t, time.January, ts.Month())
		assert.Equal(t, 12, ts.Hour())
	})

	t.Run("extracts from @timestamp field", func(t *testing.T) {
		events := []interface{}{
			map[string]interface{}{"@timestamp": "2024-06-15T08:30:00.000Z"},
		}

		ts, err := ExtractLastTimestamp(events)
		require.NoError(t, err)
		assert.Equal(t, 2024, ts.Year())
		assert.Equal(t, time.June, ts.Month())
		assert.Equal(t, 8, ts.Hour())
	})

	t.Run("extracts from unix timestamp in milliseconds", func(t *testing.T) {
		// 1704067200000 = 2024-01-01T00:00:00Z in milliseconds
		events := []interface{}{
			map[string]interface{}{"timestamp": float64(1704067200000)},
		}

		ts, err := ExtractLastTimestamp(events)
		require.NoError(t, err)
		assert.Equal(t, 2024, ts.Year())
		assert.Equal(t, time.January, ts.Month())
		assert.Equal(t, 1, ts.Day())
	})

	t.Run("returns error for empty events", func(t *testing.T) {
		_, err := ExtractLastTimestamp([]interface{}{})
		assert.Error(t, err)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		events := []interface{}{
			map[string]interface{}{"data": "no timestamp here"},
		}

		_, err := ExtractLastTimestamp(events)
		assert.Error(t, err)
	})
}

func TestCreateCursorPaginationInfo(t *testing.T) {
	t.Run("creates pagination info with next cursor", func(t *testing.T) {
		events := []interface{}{
			map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z"},
			map[string]interface{}{"timestamp": "2024-01-01T11:00:00Z"},
			map[string]interface{}{"timestamp": "2024-01-01T12:00:00Z"},
		}

		info := CreateCursorPaginationInfo(events, 100, true)

		assert.Equal(t, 3, info.ReturnedCount)
		assert.True(t, info.HasNextPage)
		assert.NotEmpty(t, info.NextCursor)
		assert.Equal(t, 100, info.PageSize)
	})

	t.Run("no cursor when no more results", func(t *testing.T) {
		events := []interface{}{
			map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z"},
		}

		info := CreateCursorPaginationInfo(events, 100, false)

		assert.Equal(t, 1, info.ReturnedCount)
		assert.False(t, info.HasNextPage)
		assert.Empty(t, info.NextCursor)
	})

	t.Run("handles empty events", func(t *testing.T) {
		info := CreateCursorPaginationInfo([]interface{}{}, 100, false)

		assert.Equal(t, 0, info.ReturnedCount)
		assert.False(t, info.HasNextPage)
		assert.Empty(t, info.NextCursor)
	})
}

func TestApplyCursorToQueryParams(t *testing.T) {
	t.Run("applies time cursor for forward direction", func(t *testing.T) {
		cursor := CreateTimeCursor(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), 50, "forward")
		args := map[string]interface{}{}

		err := ApplyCursorToQueryParams(cursor, args)
		require.NoError(t, err)

		assert.Contains(t, args, "start_date")
		assert.Equal(t, 50, args["limit"])
	})

	t.Run("applies time cursor for backward direction", func(t *testing.T) {
		cursor := CreateTimeCursor(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), 50, "backward")
		args := map[string]interface{}{}

		err := ApplyCursorToQueryParams(cursor, args)
		require.NoError(t, err)

		assert.Contains(t, args, "end_date")
		assert.Equal(t, 50, args["limit"])
	})

	t.Run("applies offset cursor", func(t *testing.T) {
		cursor := CreateOffsetCursor(100, 25)
		args := map[string]interface{}{}

		err := ApplyCursorToQueryParams(cursor, args)
		require.NoError(t, err)

		assert.Equal(t, 100, args["_offset"])
		assert.Equal(t, 25, args["limit"])
	})

	t.Run("applies ID cursor", func(t *testing.T) {
		cursor := CreateIDCursor("last-event-id", 30, "forward")
		args := map[string]interface{}{}

		err := ApplyCursorToQueryParams(cursor, args)
		require.NoError(t, err)

		assert.Equal(t, "last-event-id", args["_after_id"])
		assert.Equal(t, 30, args["limit"])
	})

	t.Run("nil cursor does nothing", func(t *testing.T) {
		args := map[string]interface{}{"existing": "value"}

		err := ApplyCursorToQueryParams(nil, args)
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"existing": "value"}, args)
	})
}

func TestParsePaginationParams(t *testing.T) {
	t.Run("extracts cursor", func(t *testing.T) {
		args := map[string]interface{}{
			"cursor": "abc123",
		}

		params := ParsePaginationParams(args)
		assert.Equal(t, "abc123", params.Cursor)
	})

	t.Run("extracts limit as float64", func(t *testing.T) {
		args := map[string]interface{}{
			"limit": float64(100),
		}

		params := ParsePaginationParams(args)
		assert.Equal(t, 100, params.Limit)
	})

	t.Run("extracts limit as int", func(t *testing.T) {
		args := map[string]interface{}{
			"limit": 50,
		}

		params := ParsePaginationParams(args)
		assert.Equal(t, 50, params.Limit)
	})

	t.Run("extracts backward direction", func(t *testing.T) {
		args := map[string]interface{}{
			"direction": "backward",
		}

		params := ParsePaginationParams(args)
		assert.Equal(t, "backward", params.Direction)
	})

	t.Run("defaults to forward direction", func(t *testing.T) {
		args := map[string]interface{}{}

		params := ParsePaginationParams(args)
		assert.Equal(t, "forward", params.Direction)
	})
}

func TestEnhanceResultWithPagination(t *testing.T) {
	t.Run("adds pagination to result", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z"},
			},
		}

		events := result["events"].([]interface{})
		EnhanceResultWithPagination(result, events, 100, true)

		assert.Contains(t, result, "_pagination")
		pagination := result["_pagination"].(map[string]interface{})
		assert.Equal(t, 1, pagination["returned_count"])
		assert.Equal(t, true, pagination["has_next_page"])
		assert.NotEmpty(t, pagination["next_cursor"])
	})

	t.Run("integrates with existing _mcp_metadata", func(t *testing.T) {
		result := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z"},
			},
			"_mcp_metadata": map[string]interface{}{
				"tool_name": "query_logs",
			},
		}

		events := result["events"].([]interface{})
		EnhanceResultWithPagination(result, events, 50, false)

		mcpMeta := result["_mcp_metadata"].(map[string]interface{})
		assert.Contains(t, mcpMeta, "pagination")
	})
}
