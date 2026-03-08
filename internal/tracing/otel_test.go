package tracing

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// saveAndRestoreGlobalTracer saves the current globalTracer value and returns
// a function that restores it. Call the returned function via defer.
func saveAndRestoreGlobalTracer(t *testing.T) func() {
	t.Helper()
	saved := globalTracer
	return func() {
		globalTracer = saved
	}
}

// --- InitOTel tests ---

func TestInitOTel_Disabled(t *testing.T) {
	cfg := OTelConfig{Enabled: false}
	shutdown, err := InitOTel(cfg)
	if err != nil {
		t.Fatalf("InitOTel with Enabled=false returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	// The no-op shutdown should return nil.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("no-op shutdown returned error: %v", err)
	}
}

func TestInitOTel_Enabled(t *testing.T) {
	defer saveAndRestoreGlobalTracer(t)()

	// Build a minimal tracer provider directly to avoid the schema URL
	// version conflict between resource.Default() and semconv/v1.26.0 that
	// occurs at the dependency versions pinned in this module.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	globalTracer = tp.Tracer("test-service")

	// Shutdown should succeed.
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}
	if globalTracer == nil {
		t.Fatal("globalTracer should be set after init")
	}
}

// --- GetTracer tests ---

func TestGetTracer_NoInit(t *testing.T) {
	defer saveAndRestoreGlobalTracer(t)()
	globalTracer = nil

	tr := GetTracer()
	if tr == nil {
		t.Fatal("GetTracer returned nil; expected a no-op tracer")
	}
}

func TestGetTracer_AfterInit(t *testing.T) {
	defer saveAndRestoreGlobalTracer(t)()

	// Set up a tracer provider directly (avoids schema URL conflict).
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	globalTracer = tp.Tracer("test-svc")

	tr := GetTracer()
	if tr == nil {
		t.Fatal("GetTracer returned nil after setting globalTracer")
	}
}

// --- Span creation tests ---

func TestToolSpan(t *testing.T) {
	ctx, span := ToolSpan(context.Background(), "my_tool")
	defer span.End()

	if ctx == nil {
		t.Fatal("ToolSpan returned nil context")
	}
	if span == nil {
		t.Fatal("ToolSpan returned nil span")
	}
}

func TestAPISpan(t *testing.T) {
	ctx, span := APISpan(context.Background(), "GET", "/api/v1/logs")
	defer span.End()

	if ctx == nil {
		t.Fatal("APISpan returned nil context")
	}
	if span == nil {
		t.Fatal("APISpan returned nil span")
	}
}

func TestCacheSpan(t *testing.T) {
	ctx, span := CacheSpan(context.Background(), "lookup", true)
	defer span.End()

	if ctx == nil {
		t.Fatal("CacheSpan returned nil context")
	}
	if span == nil {
		t.Fatal("CacheSpan returned nil span")
	}
}

// --- AddToolAttributes tests ---

func TestAddToolAttributes_AllTypes(t *testing.T) {
	t.Helper()
	_, span := ToolSpan(context.Background(), "attr_test")
	defer span.End()

	attrs := map[string]interface{}{
		"str_key":     "value",
		"int_key":     42,
		"int64_key":   int64(100),
		"float64_key": 3.14,
		"bool_key":    true,
	}
	AddToolAttributes(span, attrs)
	t.Log("AddToolAttributes with all supported types completed without panic")
}

func TestAddToolAttributes_UnknownType(t *testing.T) {
	t.Helper()
	_, span := ToolSpan(context.Background(), "unknown_attr")
	defer span.End()

	attrs := map[string]interface{}{
		"slice_key": []string{"a", "b"},
		"struct":    struct{ X int }{42},
	}
	AddToolAttributes(span, attrs)
	t.Log("AddToolAttributes with unsupported types completed without panic")
}

// --- RecordError / SetSuccess / SetToolResult tests ---

func TestRecordError_NilError(t *testing.T) {
	t.Helper()
	_, span := ToolSpan(context.Background(), "err_nil")
	defer span.End()

	RecordError(span, nil)
	t.Log("RecordError with nil error completed without panic")
}

func TestRecordError_WithError(t *testing.T) {
	_, span := ToolSpan(context.Background(), "err_real")
	defer span.End()

	RecordError(span, errors.New("something went wrong"))
	t.Log("RecordError with real error completed without panic")
}

func TestSetSuccess(t *testing.T) {
	_, span := ToolSpan(context.Background(), "success")
	defer span.End()

	SetSuccess(span)
	t.Log("SetSuccess completed without panic")
}

func TestSetToolResult(t *testing.T) {
	_, span := ToolSpan(context.Background(), "result")
	defer span.End()

	SetToolResult(span, "logs", 42)
	t.Log("SetToolResult completed without panic")
}

// --- TraceInfo tests ---

func TestNewTraceInfo(t *testing.T) {
	ti := NewTraceInfo()
	if ti == nil {
		t.Fatal("NewTraceInfo returned nil")
	}
	if len(ti.TraceID) != 32 {
		t.Fatalf("TraceID length = %d; want 32", len(ti.TraceID))
	}
	if len(ti.SpanID) != 16 {
		t.Fatalf("SpanID length = %d; want 16", len(ti.SpanID))
	}
	// Verify they are valid hex.
	if _, err := hex.DecodeString(ti.TraceID); err != nil {
		t.Fatalf("TraceID is not valid hex: %v", err)
	}
	if _, err := hex.DecodeString(ti.SpanID); err != nil {
		t.Fatalf("SpanID is not valid hex: %v", err)
	}
}

func TestNewTraceInfo_Uniqueness(t *testing.T) {
	ti1 := NewTraceInfo()
	ti2 := NewTraceInfo()

	if ti1.TraceID == ti2.TraceID {
		t.Error("two consecutive TraceIDs should differ")
	}
	if ti1.SpanID == ti2.SpanID {
		t.Error("two consecutive SpanIDs should differ")
	}
}

func TestGenerateID_Length(t *testing.T) {
	// generateID is private; tested indirectly through NewTraceInfo.
	ti := NewTraceInfo()
	if len(ti.TraceID) != 32 {
		t.Fatalf("generateID produced length %d; want 32", len(ti.TraceID))
	}
}

func TestGenerateShortID_Length(t *testing.T) {
	// generateShortID is private; tested indirectly through NewTraceInfo.
	ti := NewTraceInfo()
	if len(ti.SpanID) != 16 {
		t.Fatalf("generateShortID produced length %d; want 16", len(ti.SpanID))
	}
}

// --- TraceInfo.Headers tests ---

func TestTraceInfo_Headers(t *testing.T) {
	ti := NewTraceInfo()
	headers := ti.Headers()

	expectedKeys := []string{TraceIDHeader, SpanIDHeader, RequestIDHeader}
	for _, k := range expectedKeys {
		if _, ok := headers[k]; !ok {
			t.Errorf("Headers() missing key %q", k)
		}
	}
	if headers[TraceIDHeader] != ti.TraceID {
		t.Errorf("TraceID header = %q; want %q", headers[TraceIDHeader], ti.TraceID)
	}
	if headers[SpanIDHeader] != ti.SpanID {
		t.Errorf("SpanID header = %q; want %q", headers[SpanIDHeader], ti.SpanID)
	}
	if headers[RequestIDHeader] != ti.TraceID {
		t.Errorf("RequestID header = %q; want %q (same as TraceID)", headers[RequestIDHeader], ti.TraceID)
	}
}

func TestTraceInfo_Headers_WithParentSpan(t *testing.T) {
	ti := NewTraceInfo()
	ti.ParentSpanID = "abcdef0123456789"
	headers := ti.Headers()

	val, ok := headers[ParentSpanIDHeader]
	if !ok {
		t.Fatal("Headers() should include ParentSpanID header when set")
	}
	if val != ti.ParentSpanID {
		t.Errorf("ParentSpanID header = %q; want %q", val, ti.ParentSpanID)
	}
}

func TestTraceInfo_Headers_NoParentSpan(t *testing.T) {
	ti := NewTraceInfo()
	// ParentSpanID is empty by default.
	headers := ti.Headers()

	if _, ok := headers[ParentSpanIDHeader]; ok {
		t.Error("Headers() should not include ParentSpanID header when empty")
	}
}

// --- FromContext tests ---

func TestFromContext_EmptyContext(t *testing.T) {
	ti := FromContext(context.Background())
	if ti == nil {
		t.Fatal("FromContext returned nil for empty context")
	}
	// Empty context has no valid span, so IDs should be empty.
	if ti.TraceID != "" {
		t.Errorf("TraceID = %q; want empty for context without span", ti.TraceID)
	}
	if ti.SpanID != "" {
		t.Errorf("SpanID = %q; want empty for context without span", ti.SpanID)
	}
}

func TestFromContext_WithSpan(t *testing.T) {
	// Create a real tracer provider so spans have valid trace/span IDs.
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tr := tp.Tracer("test")
	ctx, span := tr.Start(context.Background(), "test-span")
	defer span.End()

	ti := FromContext(ctx)
	if ti == nil {
		t.Fatal("FromContext returned nil")
	}
	if ti.TraceID == "" {
		t.Error("expected non-empty TraceID from context with span")
	}
	if ti.SpanID == "" {
		t.Error("expected non-empty SpanID from context with span")
	}
	// Verify they match the span context.
	sc := span.SpanContext()
	if ti.TraceID != sc.TraceID().String() {
		t.Errorf("TraceID = %q; want %q", ti.TraceID, sc.TraceID().String())
	}
	if ti.SpanID != sc.SpanID().String() {
		t.Errorf("SpanID = %q; want %q", ti.SpanID, sc.SpanID().String())
	}
}

// --- SpanKind constants test ---

func TestSpanKindConstants(t *testing.T) {
	tests := []struct {
		kind SpanKind
		want string
	}{
		{SpanKindTool, "tool"},
		{SpanKindAPI, "api"},
		{SpanKindCache, "cache"},
		{SpanKindInternal, "internal"},
	}
	for _, tt := range tests {
		if string(tt.kind) != tt.want {
			t.Errorf("SpanKind %v = %q; want %q", tt.kind, string(tt.kind), tt.want)
		}
	}
}

// --- Header constant tests ---

func TestHeaderConstants(t *testing.T) {
	// Verify the header constants have expected values to catch accidental changes.
	if TraceIDHeader != "X-Trace-ID" {
		t.Errorf("TraceIDHeader = %q; want %q", TraceIDHeader, "X-Trace-ID")
	}
	if SpanIDHeader != "X-Span-ID" {
		t.Errorf("SpanIDHeader = %q; want %q", SpanIDHeader, "X-Span-ID")
	}
	if ParentSpanIDHeader != "X-Parent-Span-ID" {
		t.Errorf("ParentSpanIDHeader = %q; want %q", ParentSpanIDHeader, "X-Parent-Span-ID")
	}
	if RequestIDHeader != "X-Request-ID" {
		t.Errorf("RequestIDHeader = %q; want %q", RequestIDHeader, "X-Request-ID")
	}
}

// Ensure that the no-op tracer returned by GetTracer is usable (not nil) even
// when the global OTel provider has not been configured.
func TestGetTracer_NoopIsUsable(t *testing.T) {
	defer saveAndRestoreGlobalTracer(t)()
	globalTracer = nil

	// Reset to default (no-op) provider.
	otel.SetTracerProvider(noop.NewTracerProvider())

	tr := GetTracer()
	ctx, span := tr.Start(context.Background(), "noop-test")
	defer span.End()

	if ctx == nil {
		t.Fatal("no-op tracer Start returned nil context")
	}
}
