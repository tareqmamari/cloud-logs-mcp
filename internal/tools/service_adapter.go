// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file provides an adapter to integrate the new service layer with
// existing tool implementations, enabling gradual migration.
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/service"
)

// ServiceProvider provides access to the LogsService interface.
// This enables dependency injection for better testability.
type ServiceProvider interface {
	GetLogsService() service.LogsService
}

// serviceProviderImpl holds the singleton service instance
type serviceProviderImpl struct {
	service service.LogsService
	mu      sync.RWMutex
}

var (
	globalServiceProvider *serviceProviderImpl
	serviceProviderOnce   sync.Once
)

// InitializeServiceProvider initializes the global service provider.
// This should be called once during server startup.
func InitializeServiceProvider(c client.Doer, logger *zap.Logger) {
	serviceProviderOnce.Do(func() {
		globalServiceProvider = &serviceProviderImpl{
			service: service.NewLogsService(c, logger, nil),
		}
	})
}

// GetServiceProvider returns the global service provider.
// Returns nil if not initialized.
func GetServiceProvider() ServiceProvider {
	return globalServiceProvider
}

// GetLogsService returns the LogsService instance
func (p *serviceProviderImpl) GetLogsService() service.LogsService {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.service
}

// SetLogsService allows replacing the service (useful for testing)
func (p *serviceProviderImpl) SetLogsService(s service.LogsService) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.service = s
}

// ========================================================================
// Service-Aware Base Tool
// ========================================================================

// ServiceAwareTool extends BaseTool with service layer integration
type ServiceAwareTool struct {
	*BaseTool
	serviceProvider ServiceProvider
}

// NewServiceAwareTool creates a new service-aware tool
func NewServiceAwareTool(c client.Doer, logger *zap.Logger) *ServiceAwareTool {
	return &ServiceAwareTool{
		BaseTool:        NewBaseTool(c, logger),
		serviceProvider: globalServiceProvider,
	}
}

// GetService returns the LogsService, falling back to direct client if unavailable
func (t *ServiceAwareTool) GetService() service.LogsService {
	if t.serviceProvider != nil {
		return t.serviceProvider.GetLogsService()
	}
	return nil
}

// ExecuteResourceOperation executes a resource operation using the service layer
func (t *ServiceAwareTool) ExecuteResourceOperation(
	ctx context.Context,
	resourceType service.ResourceType,
	operation string,
	id string,
	data map[string]interface{},
) (map[string]interface{}, error) {
	svc := t.GetService()
	if svc == nil {
		// Fall back to direct client execution
		return t.executeViaClient(ctx, resourceType, operation, id, data)
	}

	switch operation {
	case "get":
		return svc.Get(ctx, resourceType, id)
	case "list":
		opts := &service.ListOptions{}
		if limit, ok := data["limit"].(int); ok {
			opts.Limit = limit
		}
		if cursor, ok := data["cursor"].(string); ok {
			opts.Cursor = cursor
		}
		resp, err := svc.List(ctx, resourceType, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"items":       resp.Items,
			"total_count": resp.TotalCount,
			"has_more":    resp.HasMore,
			"next_cursor": resp.NextCursor,
		}, nil
	case "create":
		return svc.Create(ctx, resourceType, data)
	case "update":
		return svc.Update(ctx, resourceType, id, data)
	case "delete":
		if err := svc.Delete(ctx, resourceType, id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"success": true}, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// executeViaClient falls back to direct client execution
func (t *ServiceAwareTool) executeViaClient(
	ctx context.Context,
	resourceType service.ResourceType,
	operation string,
	id string,
	data map[string]interface{},
) (map[string]interface{}, error) {
	// Map resource type to API path
	pathMap := map[service.ResourceType]string{
		service.ResourceAlert:           "/v1/alerts",
		service.ResourceAlertDefinition: "/v1/alert_definitions",
		service.ResourceDashboard:       "/v1/dashboards",
		service.ResourceDashboardFolder: "/v1/dashboard_folders",
		service.ResourcePolicy:          "/v1/policies",
		service.ResourceRuleGroup:       "/v1/rule_groups",
		service.ResourceOutgoingWebhook: "/v1/outgoing_webhooks",
		service.ResourceE2M:             "/v1/events2metrics",
		service.ResourceEnrichment:      "/v1/enrichments",
		service.ResourceView:            "/v1/views",
		service.ResourceViewFolder:      "/v1/view_folders",
		service.ResourceDataAccessRule:  "/v1/data_access_rules",
		service.ResourceStream:          "/v1/streams",
		service.ResourceEventStream:     "/v1/event_stream_targets",
	}

	basePath, ok := pathMap[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	var req *client.Request
	switch operation {
	case "get":
		req = &client.Request{Method: "GET", Path: basePath + "/" + id}
	case "list":
		req = &client.Request{Method: "GET", Path: basePath}
	case "create":
		req = &client.Request{Method: "POST", Path: basePath, Body: data}
	case "update":
		req = &client.Request{Method: "PUT", Path: basePath + "/" + id, Body: data}
	case "delete":
		req = &client.Request{Method: "DELETE", Path: basePath + "/" + id}
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return t.ExecuteRequest(ctx, req)
}

// ExecuteQuery executes a log query using the service layer
func (t *ServiceAwareTool) ExecuteQuery(ctx context.Context, req *service.QueryRequest) (*service.QueryResponse, error) {
	svc := t.GetService()
	if svc == nil {
		// Fall back to direct client execution
		return t.executeQueryViaClient(ctx, req)
	}
	return svc.Query(ctx, req)
}

// executeQueryViaClient falls back to direct client execution for queries
func (t *ServiceAwareTool) executeQueryViaClient(ctx context.Context, req *service.QueryRequest) (*service.QueryResponse, error) {
	start := time.Now()

	// Build request body
	metadata := map[string]interface{}{
		"tier":       req.Tier,
		"syntax":     req.Syntax,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
	}
	if req.Limit > 0 {
		metadata["limit"] = req.Limit
	}

	body := map[string]interface{}{
		"query":    req.Query,
		"metadata": metadata,
	}

	clientReq := &client.Request{
		Method:    "POST",
		Path:      "/v1/query",
		Body:      body,
		AcceptSSE: true,
		Timeout:   60 * time.Second,
	}

	result, err := t.ExecuteRequest(ctx, clientReq)
	if err != nil {
		return nil, err
	}

	// Convert to service response
	response := &service.QueryResponse{
		Metadata: &service.QueryMetadata{
			Tier:          req.Tier,
			Syntax:        req.Syntax,
			StartDate:     req.StartDate,
			EndDate:       req.EndDate,
			ExecutionTime: time.Since(start),
		},
	}

	// Extract events from result
	if events, ok := result["events"].([]interface{}); ok {
		response.Events = make([]map[string]interface{}, len(events))
		for i, e := range events {
			if event, ok := e.(map[string]interface{}); ok {
				response.Events[i] = event
			}
		}
		response.TotalCount = len(events)
	}

	return response, nil
}

// ========================================================================
// Error Handling Adapters
// ========================================================================

// HandleServiceError converts a service error to an agent-actionable MCP result
func HandleServiceError(err error, resourceType string, operation string) *service.AgentActionableError {
	// Check if it's already an agent-actionable error
	if agentErr, ok := err.(*service.AgentActionableError); ok {
		return agentErr
	}

	// Check if it's a resource error
	if resErr, ok := err.(*service.ResourceError); ok {
		return service.NewAgentError(
			service.ErrResourceNotFound,
			resErr.Error(),
			service.ActionElicit,
			fmt.Sprintf("%s operation failed - verify the %s exists", operation, resourceType),
		).WithResource(resourceType, resErr.ID)
	}

	// Generic error handling
	return service.NewAgentError(
		service.ErrServerError,
		fmt.Sprintf("%s %s failed: %s", operation, resourceType, err.Error()),
		service.ActionRetry,
		"Transient error - retry may succeed",
	)
}

// FormatAgentError formats an agent-actionable error for MCP response
func FormatAgentError(err *service.AgentActionableError) string {
	return err.FormatForAgent()
}

// ========================================================================
// Context Helpers for Service Layer
// ========================================================================

type serviceContextKey struct{}

// WithService adds a LogsService to the context
func WithService(ctx context.Context, svc service.LogsService) context.Context {
	return context.WithValue(ctx, serviceContextKey{}, svc)
}

// GetServiceFromContext retrieves the LogsService from context
func GetServiceFromContext(ctx context.Context) service.LogsService {
	if svc, ok := ctx.Value(serviceContextKey{}).(service.LogsService); ok {
		return svc
	}
	// Fall back to global provider
	if globalServiceProvider != nil {
		return globalServiceProvider.GetLogsService()
	}
	return nil
}

// ========================================================================
// Mock Service for Testing
// ========================================================================

// MockLogsService provides a test double for LogsService
type MockLogsService struct {
	// Configurable responses
	QueryResult           *service.QueryResponse
	QueryError            error
	GetResult             map[string]interface{}
	GetError              error
	ListResult            *service.ListResponse
	ListError             error
	CreateResult          map[string]interface{}
	CreateError           error
	UpdateResult          map[string]interface{}
	UpdateError           error
	DeleteError           error
	HealthResult          *service.HealthStatus
	HealthError           error
	BackgroundQueryResult *service.BackgroundQueryResponse
	BackgroundQueryError  error

	// Call tracking
	QueryCalls []service.QueryRequest
	GetCalls   []struct {
		Type service.ResourceType
		ID   string
	}
	ListCalls []struct {
		Type service.ResourceType
		Opts *service.ListOptions
	}
	CreateCalls []struct {
		Type service.ResourceType
		Data map[string]interface{}
	}
	UpdateCalls []struct {
		Type service.ResourceType
		ID   string
		Data map[string]interface{}
	}
	DeleteCalls []struct {
		Type service.ResourceType
		ID   string
	}

	mu sync.Mutex
}

// NewMockLogsService creates a new mock service
func NewMockLogsService() *MockLogsService {
	return &MockLogsService{}
}

// Query implements LogsService.Query
func (m *MockLogsService) Query(ctx context.Context, req *service.QueryRequest) (*service.QueryResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.QueryCalls = append(m.QueryCalls, *req)
	m.mu.Unlock()

	if m.QueryError != nil {
		return nil, m.QueryError
	}
	if m.QueryResult != nil {
		return m.QueryResult, nil
	}
	return &service.QueryResponse{Events: []map[string]interface{}{}}, nil
}

// SubmitBackgroundQuery implements LogsService.SubmitBackgroundQuery
func (m *MockLogsService) SubmitBackgroundQuery(ctx context.Context, req *service.BackgroundQueryRequest) (*service.BackgroundQueryResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if m.BackgroundQueryError != nil {
		return nil, m.BackgroundQueryError
	}
	if m.BackgroundQueryResult != nil {
		return m.BackgroundQueryResult, nil
	}
	return &service.BackgroundQueryResponse{QueryID: "mock-query-" + req.Query, Status: "pending"}, nil
}

// GetBackgroundQueryStatus implements LogsService.GetBackgroundQueryStatus
func (m *MockLogsService) GetBackgroundQueryStatus(ctx context.Context, queryID string) (*service.BackgroundQueryStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &service.BackgroundQueryStatus{QueryID: queryID, Status: "completed"}, nil
}

// GetBackgroundQueryData implements LogsService.GetBackgroundQueryData
func (m *MockLogsService) GetBackgroundQueryData(ctx context.Context, queryID string) (*service.QueryResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &service.QueryResponse{Events: []map[string]interface{}{{"query_id": queryID}}}, nil
}

// CancelBackgroundQuery implements LogsService.CancelBackgroundQuery
func (m *MockLogsService) CancelBackgroundQuery(ctx context.Context, queryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	m.DeleteCalls = append(m.DeleteCalls, struct {
		Type service.ResourceType
		ID   string
	}{"background_query", queryID})
	m.mu.Unlock()
	return nil
}

// Get implements LogsService.Get
func (m *MockLogsService) Get(ctx context.Context, resourceType service.ResourceType, id string) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.GetCalls = append(m.GetCalls, struct {
		Type service.ResourceType
		ID   string
	}{resourceType, id})
	m.mu.Unlock()

	if m.GetError != nil {
		return nil, m.GetError
	}
	if m.GetResult != nil {
		return m.GetResult, nil
	}
	return map[string]interface{}{"id": id, "name": "mock-resource"}, nil
}

// List implements LogsService.List
func (m *MockLogsService) List(ctx context.Context, resourceType service.ResourceType, opts *service.ListOptions) (*service.ListResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.ListCalls = append(m.ListCalls, struct {
		Type service.ResourceType
		Opts *service.ListOptions
	}{resourceType, opts})
	m.mu.Unlock()

	if m.ListError != nil {
		return nil, m.ListError
	}
	if m.ListResult != nil {
		return m.ListResult, nil
	}
	return &service.ListResponse{Items: []map[string]interface{}{}}, nil
}

// Create implements LogsService.Create
func (m *MockLogsService) Create(ctx context.Context, resourceType service.ResourceType, data map[string]interface{}) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct {
		Type service.ResourceType
		Data map[string]interface{}
	}{resourceType, data})
	m.mu.Unlock()

	if m.CreateError != nil {
		return nil, m.CreateError
	}
	if m.CreateResult != nil {
		return m.CreateResult, nil
	}
	result := map[string]interface{}{"id": "mock-created-id"}
	for k, v := range data {
		result[k] = v
	}
	return result, nil
}

// Update implements LogsService.Update
func (m *MockLogsService) Update(ctx context.Context, resourceType service.ResourceType, id string, data map[string]interface{}) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.UpdateCalls = append(m.UpdateCalls, struct {
		Type service.ResourceType
		ID   string
		Data map[string]interface{}
	}{resourceType, id, data})
	m.mu.Unlock()

	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	if m.UpdateResult != nil {
		return m.UpdateResult, nil
	}
	result := map[string]interface{}{"id": id}
	for k, v := range data {
		result[k] = v
	}
	return result, nil
}

// Delete implements LogsService.Delete
func (m *MockLogsService) Delete(ctx context.Context, resourceType service.ResourceType, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	m.DeleteCalls = append(m.DeleteCalls, struct {
		Type service.ResourceType
		ID   string
	}{resourceType, id})
	m.mu.Unlock()

	return m.DeleteError
}

// HealthCheck implements LogsService.HealthCheck
func (m *MockLogsService) HealthCheck(ctx context.Context) (*service.HealthStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if m.HealthError != nil {
		return nil, m.HealthError
	}
	if m.HealthResult != nil {
		return m.HealthResult, nil
	}
	return &service.HealthStatus{Healthy: true, Status: "healthy"}, nil
}

// GetInstanceInfo implements LogsService.GetInstanceInfo
func (m *MockLogsService) GetInstanceInfo() *service.InstanceInfo {
	return &service.InstanceInfo{
		ServiceURL:   "https://mock.logs.cloud.ibm.com",
		Region:       "us-south",
		InstanceName: "mock-instance",
	}
}

// Reset clears all recorded calls (useful between tests)
func (m *MockLogsService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryCalls = nil
	m.GetCalls = nil
	m.ListCalls = nil
	m.CreateCalls = nil
	m.UpdateCalls = nil
	m.DeleteCalls = nil
}
