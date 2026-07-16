package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	mcperrors "github.com/tareqmamari/cloud-logs-mcp/internal/errors"
)

// TestServiceMethods_PropagateClassifiedHTTPErrors verifies that when the
// underlying Doer returns a classified HTTP error (as the real client now
// does per FromHTTPStatus), every LogsService method propagates it in a way
// that remains errors.As-matchable to *mcperrors.StructuredError — even
// though several methods wrap the error (fmt.Errorf %w, or *ResourceError).
func TestServiceMethods_PropagateClassifiedHTTPErrors(t *testing.T) {
	notFound := mcperrors.FromHTTPStatus(404, "alert not found")
	unauthorized := mcperrors.FromHTTPStatus(401, "invalid api key")

	tests := []struct {
		name string
		call func(svc LogsService) error
	}{
		{
			name: "Query",
			call: func(svc LogsService) error {
				_, err := svc.Query(context.Background(), &QueryRequest{Query: "source logs", StartDate: "a", EndDate: "b"})
				return err
			},
		},
		{
			name: "SubmitBackgroundQuery",
			call: func(svc LogsService) error {
				_, err := svc.SubmitBackgroundQuery(context.Background(), &BackgroundQueryRequest{Query: "source logs"})
				return err
			},
		},
		{
			name: "GetBackgroundQueryStatus",
			call: func(svc LogsService) error {
				_, err := svc.GetBackgroundQueryStatus(context.Background(), "q-1")
				return err
			},
		},
		{
			name: "GetBackgroundQueryData",
			call: func(svc LogsService) error {
				_, err := svc.GetBackgroundQueryData(context.Background(), "q-1")
				return err
			},
		},
		{
			name: "CancelBackgroundQuery",
			call: func(svc LogsService) error {
				return svc.CancelBackgroundQuery(context.Background(), "q-1")
			},
		},
		{
			name: "Get",
			call: func(svc LogsService) error {
				_, err := svc.Get(context.Background(), ResourceAlert, "alert-123")
				return err
			},
		},
		{
			name: "List",
			call: func(svc LogsService) error {
				_, err := svc.List(context.Background(), ResourceAlert, nil)
				return err
			},
		},
		{
			name: "Create",
			call: func(svc LogsService) error {
				_, err := svc.Create(context.Background(), ResourceAlert, map[string]interface{}{})
				return err
			},
		},
		{
			name: "Update",
			call: func(svc LogsService) error {
				_, err := svc.Update(context.Background(), ResourceAlert, "alert-123", map[string]interface{}{})
				return err
			},
		},
		{
			name: "Delete",
			call: func(svc LogsService) error {
				return svc.Delete(context.Background(), ResourceAlert, "alert-123")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := client.NewMockClient()
			mock.RespondWithError(notFound)
			logger := zap.NewNop()
			svc := NewLogsService(mock, logger, nil)

			err := tt.call(svc)
			require.Error(t, err)

			var structuredErr *mcperrors.StructuredError
			require.True(t, errors.As(err, &structuredErr),
				"%s should propagate a *mcperrors.StructuredError reachable via errors.As, got %T: %v", tt.name, err, err)
			assert.Equal(t, mcperrors.CodeResourceNotFound, structuredErr.Code)
			assert.Equal(t, 404, structuredErr.StatusCode)
		})
	}

	// HealthCheck deliberately does not return a Go error on API failure -
	// it reports unhealthy status instead - so verify that shape directly.
	t.Run("HealthCheck", func(t *testing.T) {
		mock := client.NewMockClient()
		mock.RespondWithError(unauthorized)
		logger := zap.NewNop()
		svc := NewLogsService(mock, logger, nil)

		status, err := svc.HealthCheck(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Healthy)
		assert.Contains(t, status.Checks["api"], "UNAUTHORIZED")
	})
}
