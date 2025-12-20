// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool defines the interface that all MCP tools must implement.
// This provides a standard contract for tool registration and execution.
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human-readable description of what this tool does
	Description() string

	// InputSchema returns the JSON Schema for the tool's input parameters
	InputSchema() interface{}

	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error)

	// Annotations returns optional hints about tool behavior for LLMs.
	// These hints help LLMs make better decisions about tool usage.
	// Returns nil if no annotations are needed (defaults will be used).
	Annotations() *mcp.ToolAnnotations

	// DefaultTimeout returns the recommended timeout for this tool type.
	// Returns 0 to use the client/server default timeout.
	// This allows different tool categories (queries, workflows, etc.)
	// to have appropriate timeout values based on their expected execution time.
	DefaultTimeout() time.Duration
}

// EnhancedTool extends Tool with semantic discovery capabilities.
// Tools implementing this interface can be discovered by intent, category, and keywords.
type EnhancedTool interface {
	Tool

	// Metadata returns semantic metadata for tool discovery
	Metadata() *ToolMetadata
}

// ToolMetadata provides semantic information for intelligent tool discovery
type ToolMetadata struct {
	// Categories this tool belongs to (e.g., ["query", "observability"])
	Categories []ToolCategory `json:"categories"`

	// Keywords for semantic matching (e.g., ["error", "debug", "investigate"])
	Keywords []string `json:"keywords"`

	// Complexity level: "simple", "intermediate", "advanced"
	Complexity string `json:"complexity"`

	// UseCases describes when to use this tool
	UseCases []string `json:"use_cases"`

	// RelatedTools lists tools commonly used together
	RelatedTools []string `json:"related_tools"`

	// OutputSchema describes the structure of the tool's response
	OutputSchema interface{} `json:"output_schema,omitempty"`

	// ChainPosition indicates where this tool fits in workflows
	// "starter" - good for beginning investigations
	// "middle" - used after initial discovery
	// "finisher" - concluding actions like create/delete
	ChainPosition string `json:"chain_position"`

	// Icon is the URI for the tool's icon (MCP 2025-11-25)
	// Can be a data URI (data:image/svg+xml;base64,...) or HTTPS URL
	Icon string `json:"icon,omitempty"`
}

// ToolCategory represents the functional category of a tool
type ToolCategory string

// Tool categories for functional grouping
const (
	CategoryQuery         ToolCategory = "query"
	CategoryAlert         ToolCategory = "alert"
	CategoryAlerting      ToolCategory = "alerting"
	CategoryDashboard     ToolCategory = "dashboard"
	CategoryPolicy        ToolCategory = "policy"
	CategoryWebhook       ToolCategory = "webhook"
	CategoryE2M           ToolCategory = "e2m"
	CategoryRuleGroup     ToolCategory = "rule_group"
	CategoryDataAccess    ToolCategory = "data_access"
	CategoryEnrichment    ToolCategory = "enrichment"
	CategoryView          ToolCategory = "view"
	CategoryStream        ToolCategory = "stream"
	CategoryIngestion     ToolCategory = "ingestion"
	CategoryWorkflow      ToolCategory = "workflow"
	CategoryDataUsage     ToolCategory = "data_usage"
	CategoryAIHelper      ToolCategory = "ai_helper"
	CategoryMeta          ToolCategory = "meta"
	CategoryObservability ToolCategory = "observability"
	CategorySecurity      ToolCategory = "security"
	CategoryConfiguration ToolCategory = "configuration"
	CategoryDiscovery     ToolCategory = "discovery"
	CategoryVisualization ToolCategory = "visualization"
)

// ToolComplexity levels
const (
	ComplexitySimple       = "simple"
	ComplexityModerate     = "moderate"
	ComplexityIntermediate = "intermediate"
	ComplexityAdvanced     = "advanced"
)

// ChainPosition values
const (
	ChainStart    = "start"
	ChainStarter  = "starter"
	ChainMiddle   = "middle"
	ChainEnd      = "end"
	ChainFinisher = "finisher"
)
