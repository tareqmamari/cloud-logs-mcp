package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewLogsService(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		svc := NewLogsService(nil, logger, nil)
		if svc == nil {
			t.Fatal("NewLogsService returned nil")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		cfg := &Config{
			DefaultQueryTier:   "frequent_search",
			DefaultQuerySyntax: "lucene",
			DefaultQueryLimit:  50,
			MaxQueryLimit:      1000,
		}
		svc := NewLogsService(nil, logger, cfg)
		if svc == nil {
			t.Fatal("NewLogsService returned nil")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultQueryTier != "archive" {
		t.Errorf("DefaultQueryTier = %q, want %q", cfg.DefaultQueryTier, "archive")
	}
	if cfg.DefaultQuerySyntax != "dataprime" {
		t.Errorf("DefaultQuerySyntax = %q, want %q", cfg.DefaultQuerySyntax, "dataprime")
	}
	if cfg.DefaultQueryLimit != 200 {
		t.Errorf("DefaultQueryLimit = %d, want %d", cfg.DefaultQueryLimit, 200)
	}
	if cfg.MaxQueryLimit != 50000 {
		t.Errorf("MaxQueryLimit = %d, want %d", cfg.MaxQueryLimit, 50000)
	}
	if cfg.QueryTimeout != 60*time.Second {
		t.Errorf("QueryTimeout = %v, want 60s", cfg.QueryTimeout)
	}
	if !cfg.EnableAutoCorrect {
		t.Error("EnableAutoCorrect should default to true")
	}
}

func TestResourceConfig_AllResourceTypesConfigured(t *testing.T) {
	expectedTypes := []ResourceType{
		ResourceAlert, ResourceAlertDefinition, ResourceDashboard,
		ResourceDashboardFolder, ResourcePolicy, ResourceRuleGroup,
		ResourceOutgoingWebhook, ResourceE2M, ResourceEnrichment,
		ResourceView, ResourceViewFolder, ResourceDataAccessRule,
		ResourceStream, ResourceEventStream,
	}

	for _, rt := range expectedTypes {
		cfg, ok := resourceConfig[rt]
		if !ok {
			t.Errorf("Resource type %q not in resourceConfig", rt)
			continue
		}
		if cfg.basePath == "" {
			t.Errorf("Resource type %q has empty basePath", rt)
		}
		if cfg.listKey == "" {
			t.Errorf("Resource type %q has empty listKey", rt)
		}
	}
}

func TestGet_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Get(context.Background(), ResourceType("nonexistent"), "id-123")
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
	if !strings.Contains(err.Error(), "unknown resource type") {
		t.Errorf("Expected 'unknown resource type' error, got: %v", err)
	}
}

func TestList_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.List(context.Background(), ResourceType("nonexistent"), nil)
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestCreate_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Create(context.Background(), ResourceType("nonexistent"), map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestUpdate_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	_, err := svc.Update(context.Background(), ResourceType("nonexistent"), "id", map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestDelete_UnknownResourceType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewLogsService(nil, logger, nil)

	err := svc.Delete(context.Background(), ResourceType("nonexistent"), "id")
	if err == nil {
		t.Fatal("Expected error for unknown resource type")
	}
}

func TestResourceConfig_HTTPMethodSelection(t *testing.T) {
	// Verify the resourceConfig map correctly flags which resources use PUT vs PATCH
	tests := []struct {
		resourceType ResourceType
		expectUsePUT bool
	}{
		{ResourceAlert, false},
		{ResourceAlertDefinition, false},
		{ResourceDashboard, false},
		{ResourceDashboardFolder, false},
		{ResourcePolicy, false},
		{ResourceRuleGroup, false},
		{ResourceOutgoingWebhook, false},
		{ResourceE2M, true}, // E2M uses PUT (replace semantics)
		{ResourceEnrichment, false},
		{ResourceView, true},       // Views use PUT (replace semantics)
		{ResourceViewFolder, true}, // View folders use PUT (replace semantics)
		{ResourceDataAccessRule, false},
		{ResourceStream, false},
		{ResourceEventStream, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			cfg, ok := resourceConfig[tt.resourceType]
			if !ok {
				t.Fatalf("Resource type %q not found in resourceConfig", tt.resourceType)
			}
			if cfg.usePUT != tt.expectUsePUT {
				t.Errorf("resourceConfig[%s].usePUT = %v, want %v", tt.resourceType, cfg.usePUT, tt.expectUsePUT)
			}
		})
	}
}

func TestResourceError(t *testing.T) {
	t.Run("with ID", func(t *testing.T) {
		err := &ResourceError{
			Type:      ResourceAlert,
			Operation: "get",
			ID:        "alert-123",
			Err:       errors.New("not found"),
		}

		msg := err.Error()
		if !strings.Contains(msg, "get") {
			t.Errorf("Error should contain operation: %s", msg)
		}
		if !strings.Contains(msg, "alert") {
			t.Errorf("Error should contain type: %s", msg)
		}
		if !strings.Contains(msg, "alert-123") {
			t.Errorf("Error should contain ID: %s", msg)
		}
		if !strings.Contains(msg, "not found") {
			t.Errorf("Error should contain cause: %s", msg)
		}
	})

	t.Run("without ID", func(t *testing.T) {
		err := &ResourceError{
			Type:      ResourceDashboard,
			Operation: "list",
			Err:       errors.New("timeout"),
		}

		msg := err.Error()
		if strings.Contains(msg, "id=") {
			t.Errorf("Error without ID should not contain 'id=': %s", msg)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &ResourceError{
			Type:      ResourceAlert,
			Operation: "delete",
			Err:       cause,
		}

		if err.Unwrap() != cause {
			t.Error("Unwrap should return original cause")
		}
	})
}

// import errors for the test
var _ = errors.New
