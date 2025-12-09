// Package audit provides audit logging for tracking tool executions and operations.
// This helps with debugging, compliance, and understanding usage patterns.
package audit

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/tracing"
)

// Entry represents a single audit log entry
type Entry struct {
	Timestamp   time.Time              `json:"timestamp"`
	TraceID     string                 `json:"trace_id"`
	SpanID      string                 `json:"span_id,omitempty"`
	Tool        string                 `json:"tool"`
	Operation   string                 `json:"operation"` // create, read, update, delete, query, etc.
	Resource    string                 `json:"resource,omitempty"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Success     bool                   `json:"success"`
	Duration    time.Duration          `json:"duration_ms"`
	ErrorCode   string                 `json:"error_code,omitempty"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	InputHash   string                 `json:"input_hash,omitempty"` // Hash of input for privacy
	ResultCount int                    `json:"result_count,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	enabled bool
	logger  *zap.Logger

	// In-memory buffer for recent entries (for the audit tool)
	mu         sync.RWMutex
	entries    []Entry
	maxEntries int
}

// NewLogger creates a new audit logger
func NewLogger(logger *zap.Logger, enabled bool) *Logger {
	return &Logger{
		enabled:    enabled,
		logger:     logger.Named("audit"),
		entries:    make([]Entry, 0, 1000),
		maxEntries: 1000, // Keep last 1000 entries in memory
	}
}

// Log records an audit entry
func (l *Logger) Log(ctx context.Context, entry Entry) {
	if !l.enabled {
		return
	}

	// Enrich with trace information
	traceInfo := tracing.FromContext(ctx)
	if traceInfo.TraceID != "" {
		entry.TraceID = traceInfo.TraceID
	}
	if traceInfo.SpanID != "" {
		entry.SpanID = traceInfo.SpanID
	}

	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Log to structured logger
	fields := []zap.Field{
		zap.Time("timestamp", entry.Timestamp),
		zap.String("trace_id", entry.TraceID),
		zap.String("tool", entry.Tool),
		zap.String("operation", entry.Operation),
		zap.Bool("success", entry.Success),
		zap.Duration("duration", entry.Duration),
	}

	if entry.SpanID != "" {
		fields = append(fields, zap.String("span_id", entry.SpanID))
	}
	if entry.Resource != "" {
		fields = append(fields, zap.String("resource", entry.Resource))
	}
	if entry.ResourceID != "" {
		fields = append(fields, zap.String("resource_id", entry.ResourceID))
	}
	if entry.ErrorCode != "" {
		fields = append(fields, zap.String("error_code", entry.ErrorCode))
	}
	if entry.ErrorMsg != "" {
		fields = append(fields, zap.String("error_message", entry.ErrorMsg))
	}
	if entry.ResultCount > 0 {
		fields = append(fields, zap.Int("result_count", entry.ResultCount))
	}

	l.logger.Info("audit", fields...)

	// Store in memory buffer
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) >= l.maxEntries {
		// Remove oldest entry
		l.entries = l.entries[1:]
	}
	l.entries = append(l.entries, entry)
}

// LogToolExecution is a convenience method for logging tool executions
func (l *Logger) LogToolExecution(ctx context.Context, toolName string, operation string, resource string, resourceID string, success bool, duration time.Duration, err error) {
	entry := Entry{
		Tool:       toolName,
		Operation:  operation,
		Resource:   resource,
		ResourceID: resourceID,
		Success:    success,
		Duration:   duration,
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	l.Log(ctx, entry)
}

// GetRecentEntries returns the most recent audit entries
func (l *Logger) GetRecentEntries(limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}

	// Return most recent entries (from the end)
	start := len(l.entries) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Entry, limit)
	copy(result, l.entries[start:])

	// Reverse to get newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetEntriesByTool returns audit entries for a specific tool
func (l *Logger) GetEntriesByTool(toolName string, limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []Entry
	// Iterate from newest to oldest
	for i := len(l.entries) - 1; i >= 0 && len(result) < limit; i-- {
		if l.entries[i].Tool == toolName {
			result = append(result, l.entries[i])
		}
	}

	return result
}

// GetEntriesByTraceID returns all entries for a specific trace
func (l *Logger) GetEntriesByTraceID(traceID string) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []Entry
	for _, entry := range l.entries {
		if entry.TraceID == traceID {
			result = append(result, entry)
		}
	}

	return result
}

// GetStats returns statistics about audit entries
func (l *Logger) GetStats() Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := Stats{
		TotalEntries:    len(l.entries),
		ToolUsage:       make(map[string]int),
		OperationCounts: make(map[string]int),
		ErrorCounts:     make(map[string]int),
	}

	var successCount int
	var totalDuration time.Duration

	for _, entry := range l.entries {
		stats.ToolUsage[entry.Tool]++
		stats.OperationCounts[entry.Operation]++

		if entry.Success {
			successCount++
		} else if entry.ErrorCode != "" {
			stats.ErrorCounts[entry.ErrorCode]++
		}

		totalDuration += entry.Duration
	}

	if len(l.entries) > 0 {
		stats.SuccessRate = float64(successCount) / float64(len(l.entries)) * 100
		stats.AverageDuration = totalDuration / time.Duration(len(l.entries))
	}

	return stats
}

// Stats contains aggregated audit statistics
type Stats struct {
	TotalEntries    int            `json:"total_entries"`
	SuccessRate     float64        `json:"success_rate_pct"`
	AverageDuration time.Duration  `json:"average_duration"`
	ToolUsage       map[string]int `json:"tool_usage"`
	OperationCounts map[string]int `json:"operation_counts"`
	ErrorCounts     map[string]int `json:"error_counts"`
}

// ToJSON returns the stats as JSON
func (s Stats) ToJSON() string {
	data, _ := json.MarshalIndent(s, "", "  ")
	return string(data)
}

// Clear clears all audit entries (useful for testing)
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

// IsEnabled returns whether audit logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}
