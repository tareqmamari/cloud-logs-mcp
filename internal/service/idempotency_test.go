package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// TestQuery_MarksRequestIdempotent verifies that Query — a read-only
// operation that happens to use HTTP POST — marks its client.Request as
// Idempotent so the client layer is permitted to retry it on transient
// failures (502/503/etc).
func TestQuery_MarksRequestIdempotent(t *testing.T) {
	mock := client.NewMockClient()
	logger := zap.NewNop()
	svc := NewLogsService(mock, logger, nil)

	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte("")}

	_, err := svc.Query(context.Background(), &QueryRequest{
		Query:     "source logs | limit 10",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-01-02T00:00:00Z",
	})
	require.NoError(t, err)

	req := mock.LastRequest()
	require.NotNil(t, req)
	assert.True(t, req.Idempotent, "Query is read-only and should be marked Idempotent so it can be safely retried")
}

// TestSubmitBackgroundQuery_NotMarkedIdempotent verifies that submitting a
// background query — which creates a new server-side resource — is left
// non-idempotent (no Idempotent flag, no RequestID), so the client layer
// will not retry it on a transient failure and risk duplicate submissions.
func TestSubmitBackgroundQuery_NotMarkedIdempotent(t *testing.T) {
	mock := client.NewMockClient()
	logger := zap.NewNop()
	svc := NewLogsService(mock, logger, nil)

	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}

	_, err := svc.SubmitBackgroundQuery(context.Background(), &BackgroundQueryRequest{
		Query:  "source logs | limit 10",
		Syntax: "dataprime",
	})
	require.NoError(t, err)

	req := mock.LastRequest()
	require.NotNil(t, req)
	assert.False(t, req.Idempotent, "background query submission is not idempotent - resubmitting creates a new query")
	assert.Empty(t, req.RequestID, "background query submission should not generate a retry-enabling RequestID")
}

// TestCreate_GeneratesRequestID verifies that Create generates a stable
// client-provided RequestID so the request stays retryable (via the
// Idempotency-Key header) without ever double-creating a resource.
func TestCreate_GeneratesRequestID(t *testing.T) {
	mock := client.NewMockClient()
	logger := zap.NewNop()
	svc := NewLogsService(mock, logger, nil)

	mock.DefaultResponse = &client.Response{StatusCode: 201, Body: []byte(`{}`)}

	_, err := svc.Create(context.Background(), ResourceAlert, map[string]interface{}{"name": "new-alert"})
	require.NoError(t, err)

	req := mock.LastRequest()
	require.NotNil(t, req)
	assert.NotEmpty(t, req.RequestID, "Create should generate a RequestID so it stays safely retryable")
}

// TestCreate_RequestIDDiffersAcrossCalls verifies each distinct Create call
// gets its own RequestID (uniqueness across separate logical operations,
// while a single Do() call reuses the same one across retry attempts —
// covered at the client layer).
func TestCreate_RequestIDDiffersAcrossCalls(t *testing.T) {
	mock := client.NewMockClient()
	logger := zap.NewNop()
	svc := NewLogsService(mock, logger, nil)

	mock.DefaultResponse = &client.Response{StatusCode: 201, Body: []byte(`{}`)}

	_, err := svc.Create(context.Background(), ResourceAlert, map[string]interface{}{"name": "alert-1"})
	require.NoError(t, err)
	firstID := mock.LastRequest().RequestID

	_, err = svc.Create(context.Background(), ResourceAlert, map[string]interface{}{"name": "alert-2"})
	require.NoError(t, err)
	secondID := mock.LastRequest().RequestID

	assert.NotEmpty(t, firstID)
	assert.NotEmpty(t, secondID)
	assert.NotEqual(t, firstID, secondID, "each Create call should get its own RequestID")
}
