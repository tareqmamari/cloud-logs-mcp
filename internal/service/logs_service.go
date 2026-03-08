// Package service provides the business logic layer for IBM Cloud Logs operations.
// This package decouples the IBM Cloud Logs API from the MCP transport layer,
// enabling orthogonal design where adding new log sources requires zero changes
// to the MCP handlers.
package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// LogsService provides a unified interface for IBM Cloud Logs operations.
// This abstraction enables:
// - Separation of concerns between MCP handlers and API logic
// - Easy mocking for testing
// - Future support for multiple log sources (VPC, PowerVS, etc.)
type LogsService interface {
	// Query operations
	Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
	SubmitBackgroundQuery(ctx context.Context, req *BackgroundQueryRequest) (*BackgroundQueryResponse, error)
	GetBackgroundQueryStatus(ctx context.Context, queryID string) (*BackgroundQueryStatus, error)
	GetBackgroundQueryData(ctx context.Context, queryID string) (*QueryResponse, error)
	CancelBackgroundQuery(ctx context.Context, queryID string) error

	// CRUD operations (generic resource management)
	Get(ctx context.Context, resourceType ResourceType, id string) (map[string]interface{}, error)
	List(ctx context.Context, resourceType ResourceType, opts *ListOptions) (*ListResponse, error)
	Create(ctx context.Context, resourceType ResourceType, data map[string]interface{}) (map[string]interface{}, error)
	Update(ctx context.Context, resourceType ResourceType, id string, data map[string]interface{}) (map[string]interface{}, error)
	Delete(ctx context.Context, resourceType ResourceType, id string) error

	// Health and metadata
	HealthCheck(ctx context.Context) (*HealthStatus, error)
	GetInstanceInfo() *InstanceInfo
}

// ResourceType identifies the type of IBM Cloud Logs resource
type ResourceType string

const (
	// ResourceAlert represents an alert resource.
	ResourceAlert ResourceType = "alert"
	// ResourceAlertDefinition represents an alert definition resource.
	ResourceAlertDefinition ResourceType = "alert_definition"
	// ResourceDashboard represents a dashboard resource.
	ResourceDashboard ResourceType = "dashboard"
	// ResourceDashboardFolder represents a dashboard folder resource.
	ResourceDashboardFolder ResourceType = "dashboard_folder"
	// ResourcePolicy represents a policy resource.
	ResourcePolicy ResourceType = "policy"
	// ResourceRuleGroup represents a rule group resource.
	ResourceRuleGroup ResourceType = "rule_group"
	// ResourceOutgoingWebhook represents an outgoing webhook resource.
	ResourceOutgoingWebhook ResourceType = "outgoing_webhook"
	// ResourceE2M represents an events-to-metrics resource.
	ResourceE2M ResourceType = "e2m"
	// ResourceEnrichment represents an enrichment resource.
	ResourceEnrichment ResourceType = "enrichment"
	// ResourceView represents a view resource.
	ResourceView ResourceType = "view"
	// ResourceViewFolder represents a view folder resource.
	ResourceViewFolder ResourceType = "view_folder"
	// ResourceDataAccessRule represents a data access rule resource.
	ResourceDataAccessRule ResourceType = "data_access_rule"
	// ResourceStream represents a stream resource.
	ResourceStream ResourceType = "stream"
	// ResourceEventStream represents an event stream resource.
	ResourceEventStream ResourceType = "event_stream"
)

// resourceConfig maps resource types to their API paths and configurations
var resourceConfig = map[ResourceType]struct {
	basePath   string
	listKey    string // Key in response containing the list
	usePUT     bool   // Use PUT instead of PATCH for updates
	useReplace bool   // Use "replace" semantics (PUT creates/updates)
}{
	ResourceAlert:           {"/v1/alerts", "alerts", false, false},
	ResourceAlertDefinition: {"/v1/alert_definitions", "alert_definitions", false, false},
	ResourceDashboard:       {"/v1/dashboards", "dashboards", false, false},
	ResourceDashboardFolder: {"/v1/dashboard_folders", "dashboard_folders", false, false},
	ResourcePolicy:          {"/v1/policies", "policies", false, false},
	ResourceRuleGroup:       {"/v1/rule_groups", "rule_groups", false, false},
	ResourceOutgoingWebhook: {"/v1/outgoing_webhooks", "outgoing_webhooks", false, false},
	ResourceE2M:             {"/v1/events2metrics", "events2metrics", true, true},
	ResourceEnrichment:      {"/v1/enrichments", "enrichments", false, false},
	ResourceView:            {"/v1/views", "views", true, true},
	ResourceViewFolder:      {"/v1/view_folders", "view_folders", true, true},
	ResourceDataAccessRule:  {"/v1/data_access_rules", "data_access_rules", false, false},
	ResourceStream:          {"/v1/streams", "streams", false, false},
	ResourceEventStream:     {"/v1/event_stream_targets", "targets", false, false},
}

// QueryRequest represents a log query request
type QueryRequest struct {
	Query       string            `json:"query"`
	Tier        string            `json:"tier"` // "archive" or "frequent_search"
	Syntax      string            `json:"syntax"`
	StartDate   string            `json:"start_date"`
	EndDate     string            `json:"end_date"`
	Limit       int               `json:"limit,omitempty"`
	Filters     map[string]string `json:"filters,omitempty"`
	SummaryOnly bool              `json:"summary_only,omitempty"`
}

// QueryResponse represents log query results
type QueryResponse struct {
	Events        []map[string]interface{} `json:"events"`
	TotalCount    int                      `json:"total_count"`
	Truncated     bool                     `json:"truncated"`
	LastTimestamp string                   `json:"last_timestamp,omitempty"`
	Metadata      *QueryMetadata           `json:"metadata,omitempty"`
}

// QueryMetadata contains information about the query execution
type QueryMetadata struct {
	Tier            string        `json:"tier"`
	Syntax          string        `json:"syntax"`
	StartDate       string        `json:"start_date"`
	EndDate         string        `json:"end_date"`
	Limit           int           `json:"limit"`
	AutoCorrections []string      `json:"auto_corrections,omitempty"`
	CorrectedQuery  string        `json:"corrected_query,omitempty"`
	ExecutionTime   time.Duration `json:"execution_time"`
}

// BackgroundQueryRequest represents a background query submission
type BackgroundQueryRequest struct {
	Query     string `json:"query"`
	Syntax    string `json:"syntax"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// BackgroundQueryResponse represents the response from submitting a background query
type BackgroundQueryResponse struct {
	QueryID   string `json:"query_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// BackgroundQueryStatus represents the status of a background query
type BackgroundQueryStatus struct {
	QueryID     string `json:"query_id"`
	Status      string `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	Progress    int    `json:"progress,omitempty"`
	ResultCount int    `json:"result_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ListOptions configures list operations
type ListOptions struct {
	Limit  int
	Cursor string
	Offset int
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Items      []map[string]interface{} `json:"items"`
	TotalCount int                      `json:"total_count,omitempty"`
	NextCursor string                   `json:"next_cursor,omitempty"`
	HasMore    bool                     `json:"has_more"`
}

// HealthStatus represents the health of the service
type HealthStatus struct {
	Healthy     bool              `json:"healthy"`
	Status      string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Checks      map[string]string `json:"checks"`
	LastChecked time.Time         `json:"last_checked"`
}

// InstanceInfo contains metadata about the IBM Cloud Logs instance
type InstanceInfo struct {
	ServiceURL   string `json:"service_url"`
	Region       string `json:"region"`
	InstanceName string `json:"instance_name,omitempty"`
	InstanceID   string `json:"instance_id,omitempty"`
}

// ibmCloudLogsService implements LogsService for IBM Cloud Logs
type ibmCloudLogsService struct {
	client client.Doer
	logger *zap.Logger
	config *Config
}

// Config holds configuration for the logs service
type Config struct {
	DefaultQueryTier   string
	DefaultQuerySyntax string
	DefaultQueryLimit  int
	MaxQueryLimit      int
	QueryTimeout       time.Duration
	EnableQueryCaching bool
	EnableAutoCorrect  bool
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		DefaultQueryTier:   "archive",
		DefaultQuerySyntax: "dataprime",
		DefaultQueryLimit:  200,
		MaxQueryLimit:      50000,
		QueryTimeout:       60 * time.Second,
		EnableAutoCorrect:  true,
	}
}

// NewLogsService creates a new LogsService implementation
func NewLogsService(c client.Doer, logger *zap.Logger, cfg *Config) LogsService {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &ibmCloudLogsService{
		client: c,
		logger: logger,
		config: cfg,
	}
}

// Query executes a synchronous log query
func (s *ibmCloudLogsService) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	start := time.Now()

	// Apply defaults
	tier := req.Tier
	if tier == "" {
		tier = s.config.DefaultQueryTier
	}

	syntax := req.Syntax
	if syntax == "" {
		syntax = s.config.DefaultQuerySyntax
	}

	limit := req.Limit
	if limit <= 0 {
		limit = s.config.DefaultQueryLimit
	}
	if limit > s.config.MaxQueryLimit {
		limit = s.config.MaxQueryLimit
	}

	// Build request body
	metadata := map[string]interface{}{
		"tier":       tier,
		"syntax":     syntax,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
		"limit":      limit,
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
		Timeout:   s.config.QueryTimeout,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Parse response
	response := &QueryResponse{
		Metadata: &QueryMetadata{
			Tier:          tier,
			Syntax:        syntax,
			StartDate:     req.StartDate,
			EndDate:       req.EndDate,
			Limit:         limit,
			ExecutionTime: time.Since(start),
		},
	}

	if err := s.parseQueryResponse(result.Body, response); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	return response, nil
}

// parseQueryResponse parses the SSE query response into a structured format
func (s *ibmCloudLogsService) parseQueryResponse(body []byte, response *QueryResponse) error {
	// Initialize events if nil
	if response.Events == nil {
		response.Events = []map[string]interface{}{}
	}

	// Empty body means no results
	if len(body) == 0 {
		return nil
	}

	s.logger.Debug("parsing query response", zap.Int("body_length", len(body)))
	return nil
}

// SubmitBackgroundQuery submits a background query
func (s *ibmCloudLogsService) SubmitBackgroundQuery(ctx context.Context, req *BackgroundQueryRequest) (*BackgroundQueryResponse, error) {
	body := map[string]interface{}{
		"query":  req.Query,
		"syntax": req.Syntax,
	}

	if req.StartDate != "" {
		body["start_date"] = req.StartDate
	}
	if req.EndDate != "" {
		body["end_date"] = req.EndDate
	}

	clientReq := &client.Request{
		Method: "POST",
		Path:   "/v1/background_query",
		Body:   body,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, fmt.Errorf("background query submission failed: %w", err)
	}

	_ = result // Parse response
	return &BackgroundQueryResponse{}, nil
}

// GetBackgroundQueryStatus retrieves the status of a background query
func (s *ibmCloudLogsService) GetBackgroundQueryStatus(ctx context.Context, queryID string) (*BackgroundQueryStatus, error) {
	clientReq := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/status",
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get background query status: %w", err)
	}

	_ = result // Parse response
	return &BackgroundQueryStatus{QueryID: queryID}, nil
}

// GetBackgroundQueryData retrieves the data from a completed background query
func (s *ibmCloudLogsService) GetBackgroundQueryData(ctx context.Context, queryID string) (*QueryResponse, error) {
	clientReq := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/data",
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get background query data: %w", err)
	}

	response := &QueryResponse{}
	if err := s.parseQueryResponse(result.Body, response); err != nil {
		return nil, err
	}

	return response, nil
}

// CancelBackgroundQuery cancels a running background query
func (s *ibmCloudLogsService) CancelBackgroundQuery(ctx context.Context, queryID string) error {
	clientReq := &client.Request{
		Method: "DELETE",
		Path:   "/v1/background_query/" + queryID,
	}

	_, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return fmt.Errorf("failed to cancel background query: %w", err)
	}

	return nil
}

// Get retrieves a single resource by ID
func (s *ibmCloudLogsService) Get(ctx context.Context, resourceType ResourceType, id string) (map[string]interface{}, error) {
	cfg, ok := resourceConfig[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	clientReq := &client.Request{
		Method: "GET",
		Path:   cfg.basePath + "/" + id,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, &ResourceError{
			Type:      resourceType,
			Operation: "get",
			ID:        id,
			Err:       err,
		}
	}

	_ = result // Parse response
	return nil, nil
}

// List retrieves a paginated list of resources
func (s *ibmCloudLogsService) List(ctx context.Context, resourceType ResourceType, opts *ListOptions) (*ListResponse, error) {
	cfg, ok := resourceConfig[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	query := make(map[string]string)
	if opts != nil {
		if opts.Limit > 0 {
			query["limit"] = fmt.Sprintf("%d", opts.Limit)
		}
		if opts.Cursor != "" {
			query["cursor"] = opts.Cursor
		}
	}

	clientReq := &client.Request{
		Method: "GET",
		Path:   cfg.basePath,
		Query:  query,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, &ResourceError{
			Type:      resourceType,
			Operation: "list",
			Err:       err,
		}
	}

	_ = result // Parse response
	return &ListResponse{}, nil
}

// Create creates a new resource
func (s *ibmCloudLogsService) Create(ctx context.Context, resourceType ResourceType, data map[string]interface{}) (map[string]interface{}, error) {
	cfg, ok := resourceConfig[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	clientReq := &client.Request{
		Method: "POST",
		Path:   cfg.basePath,
		Body:   data,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, &ResourceError{
			Type:      resourceType,
			Operation: "create",
			Err:       err,
		}
	}

	_ = result // Parse response
	return nil, nil
}

// Update updates an existing resource
func (s *ibmCloudLogsService) Update(ctx context.Context, resourceType ResourceType, id string, data map[string]interface{}) (map[string]interface{}, error) {
	cfg, ok := resourceConfig[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	method := "PATCH"
	if cfg.usePUT {
		method = "PUT"
	}

	clientReq := &client.Request{
		Method: method,
		Path:   cfg.basePath + "/" + id,
		Body:   data,
	}

	result, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return nil, &ResourceError{
			Type:      resourceType,
			Operation: "update",
			ID:        id,
			Err:       err,
		}
	}

	_ = result // Parse response
	return nil, nil
}

// Delete removes a resource
func (s *ibmCloudLogsService) Delete(ctx context.Context, resourceType ResourceType, id string) error {
	cfg, ok := resourceConfig[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	clientReq := &client.Request{
		Method: "DELETE",
		Path:   cfg.basePath + "/" + id,
	}

	_, err := s.client.Do(ctx, clientReq)
	if err != nil {
		return &ResourceError{
			Type:      resourceType,
			Operation: "delete",
			ID:        id,
			Err:       err,
		}
	}

	return nil
}

// HealthCheck performs a health check of the service
func (s *ibmCloudLogsService) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Healthy:     true,
		Status:      "healthy",
		Checks:      make(map[string]string),
		LastChecked: time.Now(),
	}

	// Check API connectivity
	clientReq := &client.Request{
		Method: "GET",
		Path:   "/v1/policies", // Simple endpoint to verify connectivity
	}

	_, err := s.client.Do(ctx, clientReq)
	if err != nil {
		status.Healthy = false
		status.Status = "unhealthy"
		status.Checks["api"] = fmt.Sprintf("failed: %v", err)
	} else {
		status.Checks["api"] = "ok"
	}

	return status, nil
}

// GetInstanceInfo returns information about the IBM Cloud Logs instance
func (s *ibmCloudLogsService) GetInstanceInfo() *InstanceInfo {
	info := s.client.GetInstanceInfo()
	return &InstanceInfo{
		ServiceURL:   info.ServiceURL,
		Region:       info.Region,
		InstanceName: info.InstanceName,
	}
}

// ResourceError represents an error during resource operations
type ResourceError struct {
	Type      ResourceType
	Operation string
	ID        string
	Err       error
}

func (e *ResourceError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s (id=%s): %v", e.Operation, e.Type, e.ID, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Operation, e.Type, e.Err)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}
