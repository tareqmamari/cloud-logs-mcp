// Package session provides session context management for the MCP server.
// It maintains state across tool calls within a conversation to enable
// intelligent tool chaining and contextual suggestions.
package session

import (
	"sync"
	"time"
)

// Context holds session state that persists across tool calls
type Context struct {
	mu sync.RWMutex

	// Query context
	LastQuery        *QueryInfo
	RecentQueries    []QueryInfo
	maxRecentQueries int

	// Resource context - last accessed resources by type
	LastResources map[string]*ResourceInfo

	// Error context
	RecentErrors    []ErrorInfo
	maxRecentErrors int

	// Session metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	ToolCalls int
}

// QueryInfo stores information about a query execution
type QueryInfo struct {
	Query       string                 `json:"query"`
	Syntax      string                 `json:"syntax"`
	Tier        string                 `json:"tier"`
	StartDate   string                 `json:"start_date,omitempty"`
	EndDate     string                 `json:"end_date,omitempty"`
	ResultCount int                    `json:"result_count"`
	HasErrors   bool                   `json:"has_errors"`
	TopApps     []string               `json:"top_apps,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceInfo stores information about an accessed resource
type ResourceInfo struct {
	Type      string    `json:"type"` // dashboard, alert, policy, etc.
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorInfo stores information about errors encountered
type ErrorInfo struct {
	Tool      string    `json:"tool"`
	Message   string    `json:"message"`
	Code      int       `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new session context
func New() *Context {
	return &Context{
		LastResources:    make(map[string]*ResourceInfo),
		RecentQueries:    make([]QueryInfo, 0, 10),
		RecentErrors:     make([]ErrorInfo, 0, 10),
		maxRecentQueries: 10,
		maxRecentErrors:  10,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// RecordQuery records a query execution
func (c *Context) RecordQuery(info QueryInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info.Timestamp = time.Now()
	c.LastQuery = &info
	c.UpdatedAt = time.Now()
	c.ToolCalls++

	// Add to recent queries, maintaining max size
	c.RecentQueries = append(c.RecentQueries, info)
	if len(c.RecentQueries) > c.maxRecentQueries {
		c.RecentQueries = c.RecentQueries[1:]
	}
}

// RecordResource records access to a resource
func (c *Context) RecordResource(resourceType, id, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastResources[resourceType] = &ResourceInfo{
		Type:      resourceType,
		ID:        id,
		Name:      name,
		Timestamp: time.Now(),
	}
	c.UpdatedAt = time.Now()
	c.ToolCalls++
}

// RecordError records an error encountered during tool execution
func (c *Context) RecordError(tool, message string, code int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.RecentErrors = append(c.RecentErrors, ErrorInfo{
		Tool:      tool,
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
	})
	if len(c.RecentErrors) > c.maxRecentErrors {
		c.RecentErrors = c.RecentErrors[1:]
	}
	c.UpdatedAt = time.Now()
}

// GetLastQuery returns the last query info (thread-safe copy)
func (c *Context) GetLastQuery() *QueryInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.LastQuery == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	copy := *c.LastQuery
	return &copy
}

// GetLastResource returns the last resource of a given type
func (c *Context) GetLastResource(resourceType string) *ResourceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if info, ok := c.LastResources[resourceType]; ok {
		// Return a copy
		copy := *info
		return &copy
	}
	return nil
}

// GetRecentQueries returns recent queries (thread-safe copy)
func (c *Context) GetRecentQueries() []QueryInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]QueryInfo, len(c.RecentQueries))
	copy(result, c.RecentQueries)
	return result
}

// GetRecentErrors returns recent errors (thread-safe copy)
func (c *Context) GetRecentErrors() []ErrorInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]ErrorInfo, len(c.RecentErrors))
	copy(result, c.RecentErrors)
	return result
}

// HasRecentErrors returns true if there were recent errors
func (c *Context) HasRecentErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.RecentErrors) > 0
}

// GetStats returns session statistics
func (c *Context) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
		"tool_calls":      c.ToolCalls,
		"queries_count":   len(c.RecentQueries),
		"resources_count": len(c.LastResources),
		"errors_count":    len(c.RecentErrors),
		"age_seconds":     time.Since(c.CreatedAt).Seconds(),
	}
}

// Clear resets the session context
func (c *Context) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastQuery = nil
	c.RecentQueries = make([]QueryInfo, 0, 10)
	c.LastResources = make(map[string]*ResourceInfo)
	c.RecentErrors = make([]ErrorInfo, 0, 10)
	c.UpdatedAt = time.Now()
	c.ToolCalls = 0
}

// SuggestNextTools suggests tools based on session context
func (c *Context) SuggestNextTools() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var suggestions []string

	// If last query had errors, suggest alert creation
	if c.LastQuery != nil && c.LastQuery.HasErrors {
		suggestions = append(suggestions, "create_alert")
		suggestions = append(suggestions, "create_dashboard")
	}

	// If we accessed a dashboard, suggest related tools
	if _, ok := c.LastResources["dashboard"]; ok {
		suggestions = append(suggestions, "pin_dashboard")
		suggestions = append(suggestions, "update_dashboard")
	}

	// If we accessed an alert, suggest related tools
	if _, ok := c.LastResources["alert"]; ok {
		suggestions = append(suggestions, "list_outgoing_webhooks")
	}

	// If there were recent errors, suggest debugging
	if len(c.RecentErrors) > 0 {
		suggestions = append(suggestions, "get_query_templates")
		suggestions = append(suggestions, "explain_query")
	}

	return suggestions
}
