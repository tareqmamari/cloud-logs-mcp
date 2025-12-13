// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements cursor-based pagination for large result sets.
package tools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// CursorType identifies the type of pagination cursor
type CursorType string

// Cursor types for pagination
const (
	CursorTypeTime   CursorType = "time"
	CursorTypeOffset CursorType = "offset"
	CursorTypeID     CursorType = "id"
)

// PaginationCursor represents an opaque cursor for pagination
type PaginationCursor struct {
	Type      CursorType `json:"t"`
	Timestamp string     `json:"ts,omitempty"` // ISO8601 timestamp for time-based
	Offset    int        `json:"o,omitempty"`  // Numeric offset for offset-based
	LastID    string     `json:"id,omitempty"` // Last seen ID for ID-based
	Direction string     `json:"d,omitempty"`  // "forward" or "backward"
	Limit     int        `json:"l,omitempty"`  // Page size
	Query     string     `json:"q,omitempty"`  // Original query hash (for validation)
}

// EncodeCursor encodes a cursor to a base64 string
func EncodeCursor(cursor *PaginationCursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 cursor string
func DecodeCursor(encoded string) (*PaginationCursor, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var cursor PaginationCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	return &cursor, nil
}

// CreateTimeCursor creates a time-based pagination cursor
func CreateTimeCursor(timestamp time.Time, limit int, direction string) *PaginationCursor {
	return &PaginationCursor{
		Type:      CursorTypeTime,
		Timestamp: timestamp.Format(time.RFC3339Nano),
		Limit:     limit,
		Direction: direction,
	}
}

// CreateOffsetCursor creates an offset-based pagination cursor
func CreateOffsetCursor(offset, limit int) *PaginationCursor {
	return &PaginationCursor{
		Type:   CursorTypeOffset,
		Offset: offset,
		Limit:  limit,
	}
}

// CreateIDCursor creates an ID-based pagination cursor
func CreateIDCursor(lastID string, limit int, direction string) *PaginationCursor {
	return &PaginationCursor{
		Type:      CursorTypeID,
		LastID:    lastID,
		Limit:     limit,
		Direction: direction,
	}
}

// PaginatedResponse wraps a response with pagination info
type PaginatedResponse struct {
	Data       interface{}           `json:"data"`
	Pagination *CursorPaginationInfo `json:"pagination"`
	Metadata   *MCPMetadata          `json:"_mcp_metadata,omitempty"`
}

// CursorPaginationInfo provides cursor-based pagination details in responses
// This extends the basic PaginationInfo with cursor support
type CursorPaginationInfo struct {
	TotalCount    int    `json:"total_count,omitempty"`
	ReturnedCount int    `json:"returned_count"`
	HasNextPage   bool   `json:"has_next_page"`
	HasPrevPage   bool   `json:"has_prev_page"`
	NextCursor    string `json:"next_cursor,omitempty"`
	PrevCursor    string `json:"prev_cursor,omitempty"`
	PageSize      int    `json:"page_size"`
}

// ExtractLastTimestamp extracts the last timestamp from query results for cursor creation
func ExtractLastTimestamp(events []interface{}) (time.Time, error) {
	if len(events) == 0 {
		return time.Time{}, fmt.Errorf("no events to extract timestamp from")
	}

	lastEvent, ok := events[len(events)-1].(map[string]interface{})
	if !ok {
		return time.Time{}, fmt.Errorf("invalid event format")
	}

	// Try common timestamp fields
	timestampFields := []string{"timestamp", "@timestamp", "time", "datetime", "_time"}
	for _, field := range timestampFields {
		if ts, ok := lastEvent[field]; ok {
			switch v := ts.(type) {
			case string:
				// Try parsing various time formats
				formats := []string{
					time.RFC3339Nano,
					time.RFC3339,
					"2006-01-02T15:04:05.000Z",
					"2006-01-02T15:04:05Z",
					"2006-01-02 15:04:05",
				}
				for _, format := range formats {
					if t, err := time.Parse(format, v); err == nil {
						return t, nil
					}
				}
			case float64:
				// Unix timestamp in seconds or milliseconds
				if v > 1e12 {
					return time.UnixMilli(int64(v)), nil
				}
				return time.Unix(int64(v), 0), nil
			case int64:
				if v > 1e12 {
					return time.UnixMilli(v), nil
				}
				return time.Unix(v, 0), nil
			}
		}
	}

	// Try nested in userData or labels
	if userData, ok := lastEvent["userData"].(map[string]interface{}); ok {
		for _, field := range timestampFields {
			if ts, ok := userData[field].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
					return t, nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("no timestamp found in event")
}

// ExtractFirstTimestamp extracts the first timestamp from query results
func ExtractFirstTimestamp(events []interface{}) (time.Time, error) {
	if len(events) == 0 {
		return time.Time{}, fmt.Errorf("no events to extract timestamp from")
	}
	// Temporarily swap to reuse extraction logic
	firstEvent := events[0]
	return ExtractLastTimestamp([]interface{}{firstEvent})
}

// CreateCursorPaginationInfo creates cursor-based pagination info from query results
func CreateCursorPaginationInfo(events []interface{}, limit int, hasMore bool) *CursorPaginationInfo {
	info := &CursorPaginationInfo{
		ReturnedCount: len(events),
		HasNextPage:   hasMore,
		HasPrevPage:   false, // Would need cursor context to know
		PageSize:      limit,
	}

	// Create next cursor if there are more results
	if hasMore && len(events) > 0 {
		if lastTs, err := ExtractLastTimestamp(events); err == nil {
			cursor := CreateTimeCursor(lastTs, limit, "forward")
			info.NextCursor = EncodeCursor(cursor)
		}
	}

	return info
}

// ApplyCursorToQueryParams applies cursor parameters to query arguments
func ApplyCursorToQueryParams(cursor *PaginationCursor, args map[string]interface{}) error {
	if cursor == nil {
		return nil
	}

	switch cursor.Type {
	case CursorTypeTime:
		if cursor.Timestamp != "" {
			ts, err := time.Parse(time.RFC3339Nano, cursor.Timestamp)
			if err != nil {
				return fmt.Errorf("invalid cursor timestamp: %w", err)
			}

			if cursor.Direction == "forward" {
				// Continue from after this timestamp
				args["start_date"] = ts.Format(time.RFC3339Nano)
			} else {
				// Go backward from this timestamp
				args["end_date"] = ts.Format(time.RFC3339Nano)
			}
		}
		if cursor.Limit > 0 {
			args["limit"] = cursor.Limit
		}

	case CursorTypeOffset:
		// Offset-based pagination would need API support
		// For now, we can simulate with time-based
		args["_offset"] = cursor.Offset
		if cursor.Limit > 0 {
			args["limit"] = cursor.Limit
		}

	case CursorTypeID:
		// ID-based would filter by ID > lastID
		if cursor.LastID != "" {
			args["_after_id"] = cursor.LastID
		}
		if cursor.Limit > 0 {
			args["limit"] = cursor.Limit
		}
	}

	return nil
}

// PaginationParams represents pagination parameters from user input
type PaginationParams struct {
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Direction string `json:"direction,omitempty"` // "forward" (default) or "backward"
}

// ParsePaginationParams extracts pagination parameters from tool arguments
func ParsePaginationParams(args map[string]interface{}) *PaginationParams {
	params := &PaginationParams{
		Direction: "forward",
	}

	if cursor, ok := args["cursor"].(string); ok {
		params.Cursor = cursor
	}

	if limit, ok := args["limit"].(float64); ok {
		params.Limit = int(limit)
	} else if limit, ok := args["limit"].(int); ok {
		params.Limit = limit
	}

	if direction, ok := args["direction"].(string); ok {
		if direction == "backward" {
			params.Direction = "backward"
		}
	}

	return params
}

// EnhanceResultWithPagination adds pagination info to a result map
func EnhanceResultWithPagination(result map[string]interface{}, events []interface{}, limit int, hasMore bool) {
	paginationInfo := CreateCursorPaginationInfo(events, limit, hasMore)

	// Add pagination to _mcp_metadata if it exists, otherwise create it
	if metadata, ok := result["_mcp_metadata"].(map[string]interface{}); ok {
		metadata["pagination"] = map[string]interface{}{
			"returned_count": paginationInfo.ReturnedCount,
			"has_next_page":  paginationInfo.HasNextPage,
			"has_prev_page":  paginationInfo.HasPrevPage,
			"page_size":      paginationInfo.PageSize,
			"next_cursor":    paginationInfo.NextCursor,
		}
	} else {
		result["_pagination"] = map[string]interface{}{
			"returned_count": paginationInfo.ReturnedCount,
			"has_next_page":  paginationInfo.HasNextPage,
			"next_cursor":    paginationInfo.NextCursor,
			"page_size":      paginationInfo.PageSize,
		}
	}
}
