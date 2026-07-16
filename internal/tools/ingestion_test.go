package tools

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

func TestIngestLogsTool_InputSchema(t *testing.T) {
	tool := &IngestLogsTool{}
	schema := tool.InputSchema().(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, []string{"logs"}, schema["required"])

	props := schema["properties"].(map[string]interface{})
	logsProp := props["logs"].(map[string]interface{})
	assert.Equal(t, "array", logsProp["type"])
}

// TestIngestLogsTool_Execute_NonStringApplicationName is a regression test:
// JSON numbers decode as float64, so a caller sending applicationName: 123
// used to panic on an unchecked `.(string)` type assertion in the debug log
// statement instead of returning a clean validation error.
func TestIngestLogsTool_Execute_NonStringApplicationName(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewIngestLogsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	args := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"applicationName": float64(123), // JSON numbers arrive as float64
				"subsystemName":   "auth",
				"severity":        float64(3),
				"text":            "hello",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute returned unexpected error (want clean validation error result, not a Go error): %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected a validation error result for non-string applicationName, got %+v", result)
	}
}

// TestIngestLogsTool_Execute_NonStringSubsystemName mirrors the
// applicationName case for subsystemName.
func TestIngestLogsTool_Execute_NonStringSubsystemName(t *testing.T) {
	mock := client.NewMockClient()
	tool := NewIngestLogsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	args := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"applicationName": "api-gateway",
				"subsystemName":   float64(456),
				"severity":        float64(3),
				"text":            "hello",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected a validation error result for non-string subsystemName, got %+v", result)
	}
}

// TestIngestLogsTool_Execute_SetsRequestID verifies the ingestion POST
// carries a generated RequestID so the client's idempotency-gated retry
// logic (client.Do retries POSTs only when RequestID/Idempotency-Key is set)
// covers log ingestion too.
func TestIngestLogsTool_Execute_SetsRequestID(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{"status": "ok"})
	tool := NewIngestLogsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	args := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"applicationName": "api-gateway",
				"subsystemName":   "auth",
				"severity":        float64(3),
				"text":            "hello",
			},
		},
	}

	_, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	req := mock.LastRequest()
	if req.RequestID == "" {
		t.Error("expected non-empty RequestID on ingestion request for idempotent retries, got empty string")
	}
}

// TestIngestLogsTool_Execute_TimestampIsIntegerMillis verifies that an
// auto-generated timestamp is sent as an integer number of milliseconds
// since epoch (IBM Cloud Logs ingestion accepts epoch millis), not a
// fractional Unix-seconds value.
func TestIngestLogsTool_Execute_TimestampIsIntegerMillis(t *testing.T) {
	mock := client.NewMockClient()
	mock.RespondWith(200, map[string]interface{}{"status": "ok"})
	tool := NewIngestLogsTool(mock, zap.NewNop())
	ctx := testCtx(mock)

	args := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{
				"applicationName": "api-gateway",
				"subsystemName":   "auth",
				"severity":        float64(3),
				"text":            "hello",
			},
		},
	}

	_, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	req := mock.LastRequest()
	logs, ok := req.Body.([]map[string]interface{})
	if !ok || len(logs) != 1 {
		t.Fatalf("expected request body to be a single-element []map[string]interface{}, got %T: %v", req.Body, req.Body)
	}

	ts, ok := logs[0]["timestamp"].(int64)
	if !ok {
		t.Fatalf("expected timestamp to be an int64 (epoch millis), got %T: %v", logs[0]["timestamp"], logs[0]["timestamp"])
	}
	if ts <= 0 {
		t.Errorf("expected a positive epoch-millis timestamp, got %d", ts)
	}
}
