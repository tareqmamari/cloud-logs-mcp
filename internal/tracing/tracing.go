// Package tracing provides distributed tracing support for the MCP server.
// It generates and propagates trace IDs across requests for debugging and observability.
package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey contextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey contextKey = "span_id"
	// ParentSpanIDKey is the context key for parent span ID
	ParentSpanIDKey contextKey = "parent_span_id"
)

// HTTP headers for trace propagation
const (
	// TraceIDHeader is the HTTP header for trace ID propagation
	TraceIDHeader = "X-Trace-ID"
	// SpanIDHeader is the HTTP header for span ID
	SpanIDHeader = "X-Span-ID"
	// ParentSpanIDHeader is the HTTP header for parent span ID
	ParentSpanIDHeader = "X-Parent-Span-ID"
	// RequestIDHeader is the standard request ID header
	RequestIDHeader = "X-Request-ID"
)

// TraceInfo contains all trace-related identifiers
type TraceInfo struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
}

// idPool is a pool for reusing byte slices for ID generation
var idPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 16)
	},
}

// GenerateID generates a random 32-character hex ID (128 bits)
func GenerateID() string {
	b := idPool.Get().([]byte)
	defer idPool.Put(b)

	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a simpler ID if crypto/rand fails (should never happen)
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

// GenerateShortID generates a random 16-character hex ID (64 bits) for span IDs
func GenerateShortID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}

// NewTraceInfo creates a new trace with generated IDs
func NewTraceInfo() *TraceInfo {
	return &TraceInfo{
		TraceID: GenerateID(),
		SpanID:  GenerateShortID(),
	}
}

// NewSpan creates a new span under the given trace
func (t *TraceInfo) NewSpan() *TraceInfo {
	return &TraceInfo{
		TraceID:      t.TraceID,
		SpanID:       GenerateShortID(),
		ParentSpanID: t.SpanID,
	}
}

// WithTraceInfo adds trace information to a context
func WithTraceInfo(ctx context.Context, info *TraceInfo) context.Context {
	ctx = context.WithValue(ctx, TraceIDKey, info.TraceID)
	ctx = context.WithValue(ctx, SpanIDKey, info.SpanID)
	if info.ParentSpanID != "" {
		ctx = context.WithValue(ctx, ParentSpanIDKey, info.ParentSpanID)
	}
	return ctx
}

// FromContext extracts trace information from a context
func FromContext(ctx context.Context) *TraceInfo {
	info := &TraceInfo{}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		info.TraceID = traceID
	}
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		info.SpanID = spanID
	}
	if parentSpanID, ok := ctx.Value(ParentSpanIDKey).(string); ok {
		info.ParentSpanID = parentSpanID
	}

	return info
}

// GetTraceID extracts the trace ID from context, or generates a new one if not present
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}
	return GenerateID()
}

// GetSpanID extracts the span ID from context, or generates a new one if not present
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
		return spanID
	}
	return GenerateShortID()
}

// EnsureTraceContext ensures the context has trace information, adding it if missing
func EnsureTraceContext(ctx context.Context) context.Context {
	existing := FromContext(ctx)
	if existing.TraceID == "" {
		return WithTraceInfo(ctx, NewTraceInfo())
	}
	return ctx
}

// Headers returns the trace info as HTTP headers
func (t *TraceInfo) Headers() map[string]string {
	headers := map[string]string{
		TraceIDHeader:   t.TraceID,
		SpanIDHeader:    t.SpanID,
		RequestIDHeader: t.TraceID, // Also set as request ID for compatibility
	}
	if t.ParentSpanID != "" {
		headers[ParentSpanIDHeader] = t.ParentSpanID
	}
	return headers
}
