package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// Query timeout constants
const (
	// DefaultQueryTimeout is the timeout for synchronous query requests (SSE streaming).
	// This is longer than the default HTTP timeout because SSE queries wait for
	// all results to stream before returning.
	DefaultQueryTimeout = 60 * time.Second
)

// Parameter aliases for IBM Cloud Logs terminology mapping
// Users may use familiar terms like "namespace", "application", "component", "resource", "subsystem"
// These map to IBM Cloud Logs concepts:
//   - namespace/application -> applicationName (the app generating logs)
//   - component/resource/subsystem -> subsystemName (the component within the app)
var (
	// applicationAliases maps user-friendly terms to "applicationName"
	applicationAliases = []string{"namespace", "app", "application", "service", "app_name", "application_name"}
	// subsystemAliases maps user-friendly terms to "subsystemName"
	subsystemAliases = []string{"component", "resource", "subsystem", "module", "component_name", "subsystem_name", "resource_name"}
)

// resolveAliasedParam checks for a parameter value using the canonical name first,
// then falls back to checking aliases. Returns the value and whether it was found.
func resolveAliasedParam(arguments map[string]interface{}, canonicalName string, aliases []string) (string, bool) {
	// First check canonical name
	if val, _ := GetStringParam(arguments, canonicalName, false); val != "" {
		return val, true
	}
	// Then check aliases
	for _, alias := range aliases {
		if val, _ := GetStringParam(arguments, alias, false); val != "" {
			return val, true
		}
	}
	return "", false
}

// normalizeTier maps user-friendly tier names to API values
func normalizeTier(tier string) string {
	// Convert to lowercase for case-insensitive matching
	tier = strings.ToLower(strings.TrimSpace(tier))

	// Map Priority Insights / frequent search aliases
	frequentSearchAliases := []string{
		"pi", "priority", "insights", "priority insights",
		"frequent", "quick", "fast", "hot", "realtime", "real-time",
	}
	for _, alias := range frequentSearchAliases {
		if strings.Contains(tier, alias) {
			return "frequent_search"
		}
	}

	// Map archive / cold storage aliases
	archiveAliases := []string{
		"archive", "storage", "cos", "cold", "s3", "object storage",
		"long term", "long-term", "historical",
	}
	for _, alias := range archiveAliases {
		if strings.Contains(tier, alias) {
			return "archive"
		}
	}

	// If it's already a valid value, return it
	validTiers := map[string]bool{
		"unspecified":     true,
		"archive":         true,
		"frequent_search": true,
	}
	if validTiers[tier] {
		return tier
	}

	// Default to archive if unrecognized
	return "archive"
}

// QueryTool executes a synchronous query
type QueryTool struct {
	*BaseTool
}

// NewQueryTool creates a new tool instance
func NewQueryTool(client *client.Client, logger *zap.Logger) *QueryTool {
	return &QueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *QueryTool) Name() string {
	return "query_logs"
}

// Annotations returns tool hints for LLMs
func (t *QueryTool) Annotations() *mcp.ToolAnnotations {
	return QueryAnnotations("Query Logs")
}

// Description returns the tool description
func (t *QueryTool) Description() string {
	return `Execute a synchronous query against IBM Cloud Logs. This is the DEFAULT method for querying logs.

Returns results via SSE streaming - wait for completion before processing.

**Quick syntax:** source logs | filter <condition> | limit N
- $l.applicationname, $l.subsystemname (labels)
- $m.severity: DEBUG/INFO/WARNING/ERROR/CRITICAL (metadata)
- $d.field (data fields from log payload)

**Related tools:**
- get_dataprime_reference: Full syntax documentation
- build_query: Construct queries without knowing syntax
- submit_background_query: For large/slow queries that may timeout

**Pagination:** Response includes 'last_timestamp' when more results exist. Use it as next 'start_date'.`
}

// validQueryFields defines all valid fields for the query_logs tool
// This includes API fields and convenience aliases for filtering
var validQueryFields = map[string]bool{
	// API fields (from OpenAPI spec)
	"query":                    true,
	"tier":                     true,
	"syntax":                   true,
	"start_date":               true,
	"end_date":                 true,
	"limit":                    true,
	"default_source":           true,
	"strict_fields_validation": true,
	"now_date":                 true,
	// Token optimization field
	"summary_only": true,
	// Convenience filter aliases (resolved to query filters)
	"applicationName":  true,
	"namespace":        true,
	"app":              true,
	"application":      true,
	"service":          true,
	"app_name":         true,
	"application_name": true,
	"subsystemName":    true,
	"component":        true,
	"resource":         true,
	"subsystem":        true,
	"module":           true,
	"component_name":   true,
	"subsystem_name":   true,
	"resource_name":    true,
}

// InputSchema returns the input schema
func (t *QueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The query string to execute (DataPrime or Lucene syntax). Max 4096 characters.",
				"minLength":   1,
				"maxLength":   4096,
				"examples": []string{
					"source logs | filter $m.severity >= 5",
					"source logs | filter $l.applicationname == 'api-gateway' && $m.severity >= 4",
					"source logs | filter $d.message.contains('timeout') | limit 100",
				},
			},
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "Log tier to query. archive (default, aliases: COS, storage, cold), frequent_search (aliases: PI, priority, insights, quick), or unspecified",
				"enum":        []string{"unspecified", "archive", "frequent_search"},
				"default":     "archive",
			},
			"syntax": map[string]interface{}{
				"type":        "string",
				"description": "Query syntax: dataprime (default), lucene, dataprime_utf8_base64, lucene_utf8_base64, or unspecified",
				"enum":        []string{"unspecified", "lucene", "dataprime", "lucene_utf8_base64", "dataprime_utf8_base64"},
				"default":     "dataprime",
			},
			"start_date": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Start date for the query (required, ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Always specify a time range for efficient queries.",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "End date for the query (required, ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Always specify a time range for efficient queries.",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 200 to prevent hitting response size limits). Increase if needed, max for frequent_search: 12000, max for archive: 50000.",
				"minimum":     0,
				"maximum":     50000,
			},
			"default_source": map[string]interface{}{
				"type":        "string",
				"description": "Default source when omitted in query (e.g., 'logs'). If not specified, 'source logs' must be in the query.",
			},
			"strict_fields_validation": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, reject queries using unknown fields not detected in ingested data. Default: false.",
				"default":     false,
			},
			"now_date": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Override current time for time-based functions like now() in DataPrime. Defaults to system time.",
			},
			"summary_only": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, return only statistical summary (severity distribution, top apps, counts) without raw events. Reduces response tokens by ~90%. Default: false.",
				"default":     false,
			},
			// Application filter with aliases
			"applicationName": map[string]interface{}{
				"type":        "string",
				"description": "Filter by application name (aliases: namespace, app, application, service). Maps to the applicationName label in IBM Cloud Logs.",
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Alias for applicationName - filter by namespace/application name",
			},
			// Subsystem filter with aliases
			"subsystemName": map[string]interface{}{
				"type":        "string",
				"description": "Filter by subsystem name (aliases: component, resource, module). Maps to the subsystemName label in IBM Cloud Logs.",
			},
			"component": map[string]interface{}{
				"type":        "string",
				"description": "Alias for subsystemName - filter by component/resource name",
			},
		},
		"required":             []string{"query", "start_date", "end_date"},
		"additionalProperties": false,
	}
}

// validateQueryFields checks for unknown fields in the arguments
func validateQueryFields(arguments map[string]interface{}) error {
	var unknownFields []string
	for field := range arguments {
		if !validQueryFields[field] {
			unknownFields = append(unknownFields, field)
		}
	}
	if len(unknownFields) > 0 {
		return fmt.Errorf("unknown field(s): %s (valid fields: query, tier, syntax, start_date, end_date, limit, default_source, strict_fields_validation, now_date, applicationName, subsystemName)",
			strings.Join(unknownFields, ", "))
	}
	return nil
}

// applyQueryFilters adds application/subsystem filters to the query
func applyQueryFilters(query string, arguments map[string]interface{}) string {
	var filters []string

	if appName, found := resolveAliasedParam(arguments, "applicationName", applicationAliases); found {
		filters = append(filters, `$l.applicationname == '`+escapeDataPrimeString(appName)+`'`)
	}

	if subsysName, found := resolveAliasedParam(arguments, "subsystemName", subsystemAliases); found {
		filters = append(filters, `$l.subsystemname == '`+escapeDataPrimeString(subsysName)+`'`)
	}

	if len(filters) == 0 {
		return query
	}

	filterExpr := strings.Join(filters, " && ")
	if strings.Contains(strings.ToLower(query), "| filter") {
		return query + " && " + filterExpr
	}
	return query + " | filter " + filterExpr
}

// buildQueryMetadata builds the metadata object for the query API
func buildQueryMetadata(arguments map[string]interface{}) (map[string]interface{}, string, string, error) {
	metadata := make(map[string]interface{})

	// Tier with default and normalization
	tier, _ := GetStringParam(arguments, "tier", false)
	if tier == "" {
		tier = "archive"
	} else {
		tier = normalizeTier(tier)
	}
	metadata["tier"] = tier

	// Syntax with default and validation
	syntax, _ := GetStringParam(arguments, "syntax", false)
	if syntax == "" {
		syntax = "dataprime"
	}
	validSyntax := map[string]bool{
		"unspecified": true, "lucene": true, "dataprime": true,
		"lucene_utf8_base64": true, "dataprime_utf8_base64": true,
	}
	if !validSyntax[syntax] {
		return nil, "", "", fmt.Errorf("invalid syntax '%s' (valid: unspecified, lucene, dataprime, lucene_utf8_base64, dataprime_utf8_base64)", syntax)
	}
	metadata["syntax"] = syntax

	// Required date range
	startDate, err := GetStringParam(arguments, "start_date", true)
	if err != nil {
		return nil, "", "", err
	}
	metadata["start_date"] = startDate

	endDate, err := GetStringParam(arguments, "end_date", true)
	if err != nil {
		return nil, "", "", err
	}
	metadata["end_date"] = endDate

	// Limit with validation
	limit, _ := GetIntParam(arguments, "limit", false)
	if limit > 0 {
		maxLimit := 50000
		if tier == "frequent_search" {
			maxLimit = 12000
		}
		if limit > maxLimit {
			return nil, "", "", fmt.Errorf("limit %d exceeds maximum for tier '%s' (max: %d)", limit, tier, maxLimit)
		}
		metadata["limit"] = limit
	} else {
		metadata["limit"] = 200
	}

	// Optional fields
	if defaultSource, _ := GetStringParam(arguments, "default_source", false); defaultSource != "" {
		metadata["default_source"] = defaultSource
	}
	if strictValidation, err := GetBoolParam(arguments, "strict_fields_validation", false); err == nil && strictValidation {
		metadata["strict_fields_validation"] = strictValidation
	}
	if nowDate, _ := GetStringParam(arguments, "now_date", false); nowDate != "" {
		metadata["now_date"] = nowDate
	}

	return metadata, tier, syntax, nil
}

// addQueryMetadataToResult adds query execution metadata to the result
func addQueryMetadataToResult(result map[string]interface{}, metadata map[string]interface{}, tier, syntax, query string, corrections []string) {
	queryMeta := map[string]interface{}{
		"tier":       tier,
		"syntax":     syntax,
		"start_date": metadata["start_date"],
		"end_date":   metadata["end_date"],
		"limit":      metadata["limit"],
	}
	if len(corrections) > 0 {
		queryMeta["auto_corrections"] = corrections
		queryMeta["corrected_query"] = query
	}
	result["_query_metadata"] = queryMeta
}

// Execute executes the tool
func (t *QueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()

	// Validate fields
	if err := validateQueryFields(arguments); err != nil {
		return NewToolResultError(err.Error()), nil
	}

	query, err := GetStringParam(arguments, "query", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Apply session filters if not explicitly specified
	if appName, found := resolveAliasedParam(arguments, "applicationName", applicationAliases); !found {
		if sessionApp := session.GetFilter("application"); sessionApp != "" {
			arguments["applicationName"] = sessionApp
		}
	} else {
		session.SetFilter("last_queried_app", appName)
	}

	if len(query) > 4096 {
		return NewToolResultError(fmt.Sprintf("Query too long: %d characters (max 4096)", len(query))), nil
	}

	// Apply filters to query
	query = applyQueryFilters(query, arguments)

	// Build metadata
	metadata, tier, syntax, err := buildQueryMetadata(arguments)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Auto-correct and validate DataPrime query
	var queryCorrections []string
	if syntax == "dataprime" || syntax == "dataprime_utf8_base64" {
		query, queryCorrections = AutoCorrectDataPrimeQuery(query)
		if validationErr := ValidateDataPrimeQuery(query); validationErr != nil {
			return NewToolResultError(validationErr.Error()), nil
		}
	}

	// Execute request
	body := map[string]interface{}{
		"query":    query,
		"metadata": metadata,
	}

	req := &client.Request{
		Method:    "POST",
		Path:      "/v1/query",
		Body:      body,
		AcceptSSE: true,
		Timeout:   DefaultQueryTimeout,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, map[string]interface{}{
			"query": query,
			"error": err.Error(),
		})
		return NewToolResultError(FormatQueryError(query, err.Error())), nil
	}

	// Record success and update session
	session.RecordToolUse(t.Name(), true, map[string]interface{}{
		"query":      query,
		"start_date": metadata["start_date"],
		"end_date":   metadata["end_date"],
		"tier":       tier,
	})
	session.SetLastQuery(query)

	if limit, _ := GetIntParam(arguments, "limit", false); limit > 0 {
		session.GetPreferences().PreferredLimit = limit
	}

	// Add metadata to result
	if result == nil {
		result = make(map[string]interface{})
	}
	addQueryMetadataToResult(result, metadata, tier, syntax, query, queryCorrections)

	// Return response
	summaryOnly, _ := GetBoolParam(arguments, "summary_only", false)
	if summaryOnly {
		return t.FormatCompactSummary(result, "query_logs")
	}

	return t.FormatResponseWithSummaryAndSuggestions(result, "query results", "query_logs")
}

// SubmitBackgroundQueryTool submits an asynchronous background query
type SubmitBackgroundQueryTool struct {
	*BaseTool
}

// NewSubmitBackgroundQueryTool creates a new tool instance
func NewSubmitBackgroundQueryTool(client *client.Client, logger *zap.Logger) *SubmitBackgroundQueryTool {
	return &SubmitBackgroundQueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *SubmitBackgroundQueryTool) Name() string {
	return "submit_background_query"
}

// Annotations returns tool hints for LLMs
func (t *SubmitBackgroundQueryTool) Annotations() *mcp.ToolAnnotations {
	return CreateAnnotations("Submit Background Query")
}

// Description returns the tool description
func (t *SubmitBackgroundQueryTool) Description() string {
	return `Submit an asynchronous background query for large-scale log analysis. This is a fire-and-forget operation.

WARNING: Do NOT use this for normal queries. Use query_logs instead, which is the default and preferred method.

Only use background queries when:
- The user explicitly requests a background/async query
- Querying very large time ranges that may timeout with sync queries
- Running queries that don't need immediate results

After submitting, use get_background_query_status to check progress and get_background_query_data to retrieve results when complete.`
}

// InputSchema returns the input schema
func (t *SubmitBackgroundQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The query string to execute (1-4096 characters)",
			},
			"syntax": map[string]interface{}{
				"type":        "string",
				"description": "Query syntax: dataprime (default), lucene, or unspecified",
				"enum":        []string{"unspecified", "lucene", "dataprime"},
				"default":     "dataprime",
			},
			"start_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Optional, defaults to end - 15 minutes",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"description": "End date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Optional, defaults to now",
			},
		},
		"required": []string{"query", "syntax"},
	}
}

// Execute executes the tool
func (t *SubmitBackgroundQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(arguments, "query", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	syntax, err := GetStringParam(arguments, "syntax", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Build request body with required fields
	body := map[string]interface{}{
		"query":  query,
		"syntax": syntax,
	}

	// Add optional date fields if provided
	if startDate, _ := GetStringParam(arguments, "start_date", false); startDate != "" {
		body["start_date"] = startDate
	}
	if endDate, _ := GetStringParam(arguments, "end_date", false); endDate != "" {
		body["end_date"] = endDate
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/background_query",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "submit_background_query")
}

// GetBackgroundQueryStatusTool checks the status of a background query
type GetBackgroundQueryStatusTool struct {
	*BaseTool
}

// NewGetBackgroundQueryStatusTool creates a new tool instance
func NewGetBackgroundQueryStatusTool(client *client.Client, logger *zap.Logger) *GetBackgroundQueryStatusTool {
	return &GetBackgroundQueryStatusTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetBackgroundQueryStatusTool) Name() string {
	return "get_background_query_status"
}

// Annotations returns tool hints for LLMs
func (t *GetBackgroundQueryStatusTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Get Background Query Status")
}

// Description returns the tool description
func (t *GetBackgroundQueryStatusTool) Description() string {
	return "Check the status of a background query"
}

// InputSchema returns the input schema
func (t *GetBackgroundQueryStatusTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query",
			},
		},
		"required": []string{"query_id"},
	}
}

// Execute executes the tool
func (t *GetBackgroundQueryStatusTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/status",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return HandleGetError(err, "Background query", queryID, "submit_background_query"), nil
	}

	return t.FormatResponseWithSuggestions(result, "get_background_query_status")
}

// GetBackgroundQueryDataTool retrieves the results of a background query
type GetBackgroundQueryDataTool struct {
	*BaseTool
}

// NewGetBackgroundQueryDataTool creates a new tool instance
func NewGetBackgroundQueryDataTool(client *client.Client, logger *zap.Logger) *GetBackgroundQueryDataTool {
	return &GetBackgroundQueryDataTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetBackgroundQueryDataTool) Name() string {
	return "get_background_query_data"
}

// Annotations returns tool hints for LLMs
func (t *GetBackgroundQueryDataTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Get Background Query Data")
}

// Description returns the tool description
func (t *GetBackgroundQueryDataTool) Description() string {
	return "Retrieve the results of a completed background query"
}

// InputSchema returns the input schema
func (t *GetBackgroundQueryDataTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query",
			},
		},
		"required": []string{"query_id"},
	}
}

// Execute executes the tool
func (t *GetBackgroundQueryDataTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/data",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return HandleGetError(err, "Background query data", queryID, "get_background_query_status"), nil
	}

	return t.FormatResponseWithSummary(result, "query results")
}

// CancelBackgroundQueryTool cancels a running background query
type CancelBackgroundQueryTool struct {
	*BaseTool
}

// NewCancelBackgroundQueryTool creates a new CancelBackgroundQueryTool
func NewCancelBackgroundQueryTool(client *client.Client, logger *zap.Logger) *CancelBackgroundQueryTool {
	return &CancelBackgroundQueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *CancelBackgroundQueryTool) Name() string {
	return "cancel_background_query"
}

// Annotations returns tool hints for LLMs
func (t *CancelBackgroundQueryTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Cancel Background Query")
}

// Description returns the tool description
func (t *CancelBackgroundQueryTool) Description() string {
	return "Cancel a running background query"
}

// InputSchema returns the input schema
func (t *CancelBackgroundQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query to cancel",
			},
		},
		"required": []string{"query_id"},
	}
}

// Execute executes the tool
func (t *CancelBackgroundQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/background_query/" + queryID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "cancel_background_query")
}
