// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements standardized MCP response metadata.
package tools

import (
	"time"
)

// MCPMetadata provides standardized metadata for all MCP responses.
// This replaces ad-hoc fields like _query_metadata, _truncated_info with a consistent structure.
type MCPMetadata struct {
	// Execution metadata
	ExecutionTime time.Duration `json:"execution_time_ms,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
	ToolName      string        `json:"tool_name"`
	ToolNamespace ToolNamespace `json:"tool_namespace"`

	// Query-specific metadata (for query tools)
	Query *QueryMetadata `json:"query,omitempty"`

	// Pagination metadata
	Pagination *PaginationMetadata `json:"pagination,omitempty"`

	// Rate limiting info
	RateLimit *RateLimitMetadata `json:"rate_limit,omitempty"`

	// Caching info
	Cache *CacheMetadata `json:"cache,omitempty"`

	// Corrections/transformations applied
	Corrections []string `json:"corrections,omitempty"`

	// Warnings (non-fatal issues)
	Warnings []string `json:"warnings,omitempty"`

	// Hints for follow-up actions
	Hints []string `json:"hints,omitempty"`
}

// QueryMetadata contains query-specific information
type QueryMetadata struct {
	OriginalQuery  string `json:"original_query,omitempty"`
	CorrectedQuery string `json:"corrected_query,omitempty"`
	Syntax         string `json:"syntax"`
	Tier           string `json:"tier"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	Limit          int    `json:"limit"`
}

// PaginationMetadata contains pagination information
type PaginationMetadata struct {
	TotalCount     int    `json:"total_count"`
	ReturnedCount  int    `json:"returned_count"`
	IsTruncated    bool   `json:"is_truncated"`
	TruncatedField string `json:"truncated_field,omitempty"`
	NextCursor     string `json:"next_cursor,omitempty"`
	HasMore        bool   `json:"has_more"`

	// Pagination hints
	PaginationHint string `json:"pagination_hint,omitempty"`
}

// RateLimitMetadata contains rate limiting information
type RateLimitMetadata struct {
	Limit     int     `json:"limit"`
	Remaining int     `json:"remaining"`
	Reset     int64   `json:"reset_at,omitempty"`
	Available float64 `json:"available,omitempty"`
}

// CacheMetadata contains caching information
type CacheMetadata struct {
	Hit      bool          `json:"hit"`
	Key      string        `json:"key,omitempty"`
	TTL      time.Duration `json:"ttl_seconds,omitempty"`
	CachedAt time.Time     `json:"cached_at,omitempty"`
}

// NewMCPMetadata creates a new MCPMetadata with defaults
func NewMCPMetadata(toolName string) *MCPMetadata {
	return &MCPMetadata{
		Timestamp:     time.Now().UTC(),
		ToolName:      toolName,
		ToolNamespace: GetToolNamespace(toolName),
	}
}

// WithExecutionTime sets the execution time
func (m *MCPMetadata) WithExecutionTime(d time.Duration) *MCPMetadata {
	m.ExecutionTime = d
	return m
}

// WithQuery sets query metadata
func (m *MCPMetadata) WithQuery(q *QueryMetadata) *MCPMetadata {
	m.Query = q
	return m
}

// WithPagination sets pagination metadata
func (m *MCPMetadata) WithPagination(p *PaginationMetadata) *MCPMetadata {
	m.Pagination = p
	return m
}

// WithRateLimit sets rate limit metadata
func (m *MCPMetadata) WithRateLimit(r *RateLimitMetadata) *MCPMetadata {
	m.RateLimit = r
	return m
}

// WithCache sets cache metadata
func (m *MCPMetadata) WithCache(c *CacheMetadata) *MCPMetadata {
	m.Cache = c
	return m
}

// AddCorrection adds a correction message
func (m *MCPMetadata) AddCorrection(correction string) *MCPMetadata {
	m.Corrections = append(m.Corrections, correction)
	return m
}

// AddWarning adds a warning message
func (m *MCPMetadata) AddWarning(warning string) *MCPMetadata {
	m.Warnings = append(m.Warnings, warning)
	return m
}

// AddHint adds a hint for follow-up actions
func (m *MCPMetadata) AddHint(hint string) *MCPMetadata {
	m.Hints = append(m.Hints, hint)
	return m
}

// ToMap converts MCPMetadata to a map for embedding in responses
func (m *MCPMetadata) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"timestamp":      m.Timestamp.Format(time.RFC3339),
		"tool_name":      m.ToolName,
		"tool_namespace": string(m.ToolNamespace),
	}

	if m.ExecutionTime > 0 {
		result["execution_time_ms"] = m.ExecutionTime.Milliseconds()
	}

	if m.Query != nil {
		query := map[string]interface{}{
			"syntax":     m.Query.Syntax,
			"tier":       m.Query.Tier,
			"start_date": m.Query.StartDate,
			"end_date":   m.Query.EndDate,
			"limit":      m.Query.Limit,
		}
		if m.Query.OriginalQuery != "" {
			query["original_query"] = m.Query.OriginalQuery
		}
		if m.Query.CorrectedQuery != "" {
			query["corrected_query"] = m.Query.CorrectedQuery
		}
		result["query"] = query
	}

	if m.Pagination != nil {
		pagination := map[string]interface{}{
			"total_count":    m.Pagination.TotalCount,
			"returned_count": m.Pagination.ReturnedCount,
			"is_truncated":   m.Pagination.IsTruncated,
			"has_more":       m.Pagination.HasMore,
		}
		if m.Pagination.TruncatedField != "" {
			pagination["truncated_field"] = m.Pagination.TruncatedField
		}
		if m.Pagination.NextCursor != "" {
			pagination["next_cursor"] = m.Pagination.NextCursor
		}
		if m.Pagination.PaginationHint != "" {
			pagination["pagination_hint"] = m.Pagination.PaginationHint
		}
		result["pagination"] = pagination
	}

	if m.RateLimit != nil {
		result["rate_limit"] = map[string]interface{}{
			"limit":     m.RateLimit.Limit,
			"remaining": m.RateLimit.Remaining,
			"available": m.RateLimit.Available,
		}
	}

	if m.Cache != nil {
		cache := map[string]interface{}{
			"hit": m.Cache.Hit,
		}
		if m.Cache.Key != "" {
			cache["key"] = m.Cache.Key
		}
		if m.Cache.TTL > 0 {
			cache["ttl_seconds"] = int(m.Cache.TTL.Seconds())
		}
		if !m.Cache.CachedAt.IsZero() {
			cache["cached_at"] = m.Cache.CachedAt.Format(time.RFC3339)
		}
		result["cache"] = cache
	}

	if len(m.Corrections) > 0 {
		result["corrections"] = m.Corrections
	}

	if len(m.Warnings) > 0 {
		result["warnings"] = m.Warnings
	}

	if len(m.Hints) > 0 {
		result["hints"] = m.Hints
	}

	return result
}

// AddMCPMetadataToResult adds standardized _mcp_metadata to a result map
func AddMCPMetadataToResult(result map[string]interface{}, metadata *MCPMetadata) {
	result["_mcp_metadata"] = metadata.ToMap()
}

// CreatePaginationFromTruncation creates pagination metadata from truncation info
func CreatePaginationFromTruncation(totalCount, shownCount int, field string) *PaginationMetadata {
	p := &PaginationMetadata{
		TotalCount:     totalCount,
		ReturnedCount:  shownCount,
		IsTruncated:    shownCount < totalCount,
		TruncatedField: field,
		HasMore:        shownCount < totalCount,
	}

	if p.IsTruncated {
		p.PaginationHint = "Use time-based pagination or add filters to retrieve all results"
	}

	return p
}

// MigrateLegacyMetadata converts legacy metadata fields to standardized format
// This helps during the transition from old format to new format
func MigrateLegacyMetadata(result map[string]interface{}, toolName string) {
	metadata := NewMCPMetadata(toolName)

	// Migrate _query_metadata
	if qm, ok := result["_query_metadata"].(map[string]interface{}); ok {
		query := &QueryMetadata{}
		if v, ok := qm["syntax"].(string); ok {
			query.Syntax = v
		}
		if v, ok := qm["tier"].(string); ok {
			query.Tier = v
		}
		if v, ok := qm["start_date"].(string); ok {
			query.StartDate = v
		}
		if v, ok := qm["end_date"].(string); ok {
			query.EndDate = v
		}
		if v, ok := qm["limit"].(float64); ok {
			query.Limit = int(v)
		}
		if v, ok := qm["limit"].(int); ok {
			query.Limit = v
		}
		if v, ok := qm["corrected_query"].(string); ok {
			query.CorrectedQuery = v
		}
		metadata.Query = query

		// Migrate auto_corrections
		if corrections, ok := qm["auto_corrections"].([]interface{}); ok {
			for _, c := range corrections {
				if s, ok := c.(string); ok {
					metadata.AddCorrection(s)
				}
			}
		}

		delete(result, "_query_metadata")
	}

	// Migrate _truncated_info
	if ti, ok := result["_truncated_info"].(map[string]interface{}); ok {
		var total, shown int
		var field string
		if v, ok := ti["original_count"].(float64); ok {
			total = int(v)
		}
		if v, ok := ti["original_count"].(int); ok {
			total = v
		}
		if v, ok := ti["shown_count"].(float64); ok {
			shown = int(v)
		}
		if v, ok := ti["shown_count"].(int); ok {
			shown = v
		}
		if v, ok := ti["field"].(string); ok {
			field = v
		}
		metadata.Pagination = CreatePaginationFromTruncation(total, shown, field)
		delete(result, "_truncated_info")
	}

	// Add standardized metadata
	AddMCPMetadataToResult(result, metadata)
}
