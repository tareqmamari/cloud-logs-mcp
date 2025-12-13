// Package tracing provides distributed tracing support using OpenTelemetry.
package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig holds OpenTelemetry configuration
type OTelConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Enabled        bool
}

// Global tracer
var globalTracer trace.Tracer

// InitOTel initializes OpenTelemetry with the given configuration.
// Returns a shutdown function that should be called on application exit.
func InitOTel(cfg OTelConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// Return no-op shutdown
		return func(context.Context) error { return nil }, nil
	}

	// Create stdout exporter for now (can be replaced with OTLP exporter)
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stderr),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Create global tracer
	globalTracer = tp.Tracer(cfg.ServiceName)

	// Return shutdown function
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	if globalTracer == nil {
		// Return no-op tracer if not initialized
		return otel.Tracer("noop")
	}
	return globalTracer
}

// SpanKind represents the role of a span
type SpanKind string

// Span kinds for categorizing trace spans
const (
	SpanKindTool     SpanKind = "tool"
	SpanKindAPI      SpanKind = "api"
	SpanKindCache    SpanKind = "cache"
	SpanKindInternal SpanKind = "internal"
)

// ToolSpan starts a new span for a tool execution
func ToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, "mcp.tool."+toolName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("mcp.tool.name", toolName),
			attribute.String("mcp.span.kind", string(SpanKindTool)),
		),
	)
}

// APISpan starts a new span for an API call
func APISpan(ctx context.Context, method, path string) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, "mcp.api."+method,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", path),
			attribute.String("mcp.span.kind", string(SpanKindAPI)),
		),
	)
}

// CacheSpan starts a new span for a cache operation
func CacheSpan(ctx context.Context, operation string, hit bool) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, "mcp.cache."+operation,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.Bool("cache.hit", hit),
			attribute.String("mcp.span.kind", string(SpanKindCache)),
		),
	)
}

// AddToolAttributes adds common tool attributes to a span
func AddToolAttributes(span trace.Span, attrs map[string]interface{}) {
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			span.SetAttributes(attribute.String("mcp.tool.arg."+k, val))
		case int:
			span.SetAttributes(attribute.Int("mcp.tool.arg."+k, val))
		case int64:
			span.SetAttributes(attribute.Int64("mcp.tool.arg."+k, val))
		case float64:
			span.SetAttributes(attribute.Float64("mcp.tool.arg."+k, val))
		case bool:
			span.SetAttributes(attribute.Bool("mcp.tool.arg."+k, val))
		}
	}
}

// RecordError records an error on the span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
	}
}

// SetSuccess marks the span as successful
func SetSuccess(span trace.Span) {
	span.SetAttributes(attribute.Bool("mcp.success", true))
}

// SetToolResult records the result type of a tool execution
func SetToolResult(span trace.Span, resultType string, itemCount int) {
	span.SetAttributes(
		attribute.String("mcp.result.type", resultType),
		attribute.Int("mcp.result.count", itemCount),
	)
}

// TraceInfo provides trace and span IDs for audit logging and HTTP header propagation
type TraceInfo struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// HTTP headers for trace propagation
const (
	TraceIDHeader      = "X-Trace-ID"
	SpanIDHeader       = "X-Span-ID"
	ParentSpanIDHeader = "X-Parent-Span-ID"
	RequestIDHeader    = "X-Request-ID"
)

// NewTraceInfo creates a new TraceInfo with generated IDs
func NewTraceInfo() *TraceInfo {
	return &TraceInfo{
		TraceID: generateID(),
		SpanID:  generateShortID(),
	}
}

// generateID creates a 32-char hex string for trace IDs
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

// generateShortID creates a 16-char hex string for span IDs
func generateShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}

// Headers returns trace info as HTTP headers for propagation
func (t *TraceInfo) Headers() map[string]string {
	headers := map[string]string{
		TraceIDHeader:   t.TraceID,
		SpanIDHeader:    t.SpanID,
		RequestIDHeader: t.TraceID,
	}
	if t.ParentSpanID != "" {
		headers[ParentSpanIDHeader] = t.ParentSpanID
	}
	return headers
}

// FromContext extracts trace information from context for audit logging
func FromContext(ctx context.Context) *TraceInfo {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		return &TraceInfo{}
	}

	sc := span.SpanContext()
	return &TraceInfo{
		TraceID: sc.TraceID().String(),
		SpanID:  sc.SpanID().String(),
	}
}
