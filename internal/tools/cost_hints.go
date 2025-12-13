// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements cost and impact hints for tools.
package tools

// CostLevel represents the resource cost of a tool execution
type CostLevel string

// Cost levels for API resource consumption
const (
	CostFree     CostLevel = "free"      // No API calls, local only
	CostLow      CostLevel = "low"       // Single simple API call
	CostMedium   CostLevel = "medium"    // Multiple API calls or moderate computation
	CostHigh     CostLevel = "high"      // Complex queries, large data processing
	CostVeryHigh CostLevel = "very_high" // Bulk operations, full scans
)

// ExecutionSpeed represents how fast a tool typically executes
type ExecutionSpeed string

// Execution speed levels
const (
	SpeedInstant ExecutionSpeed = "instant" // < 100ms, cached or local
	SpeedFast    ExecutionSpeed = "fast"    // < 1s, simple API call
	SpeedMedium  ExecutionSpeed = "medium"  // 1-10s, moderate processing
	SpeedSlow    ExecutionSpeed = "slow"    // 10-60s, complex operations
	SpeedAsync   ExecutionSpeed = "async"   // > 60s, background processing
)

// ImpactLevel represents the impact of a tool on the system
type ImpactLevel string

// Impact levels for system changes
const (
	ImpactNone     ImpactLevel = "none"     // Read-only, no system changes
	ImpactLow      ImpactLevel = "low"      // Minor changes, easily reversible
	ImpactMedium   ImpactLevel = "medium"   // Moderate changes, may affect workflows
	ImpactHigh     ImpactLevel = "high"     // Significant changes, affects alerting/monitoring
	ImpactCritical ImpactLevel = "critical" // Destructive or system-wide impact
)

// RateLimitImpact describes how a tool affects rate limits
type RateLimitImpact string

// Rate limit impact levels
const (
	RateLimitNone     RateLimitImpact = "none"     // Doesn't consume rate limit
	RateLimitMinimal  RateLimitImpact = "minimal"  // 1 API call
	RateLimitModerate RateLimitImpact = "moderate" // 2-5 API calls
	RateLimitHigh     RateLimitImpact = "high"     // 6-20 API calls
	RateLimitBurst    RateLimitImpact = "burst"    // 20+ API calls, may throttle
)

// CostHints provides comprehensive cost and impact information for a tool
type CostHints struct {
	// Cost estimates
	APICost         CostLevel       `json:"api_cost"`          // API resource consumption
	TokenCost       int             `json:"token_cost"`        // Estimated response token count
	RateLimitImpact RateLimitImpact `json:"rate_limit_impact"` // Impact on rate limits

	// Performance characteristics
	ExecutionSpeed  ExecutionSpeed `json:"execution_speed"`   // Typical execution time
	Cacheable       bool           `json:"cacheable"`         // Can results be cached
	CacheTTLSeconds int            `json:"cache_ttl_seconds"` // Suggested cache duration

	// System impact
	Impact          ImpactLevel `json:"impact"`           // System impact level
	Reversible      bool        `json:"reversible"`       // Can the action be undone
	RequiresConfirm bool        `json:"requires_confirm"` // Should prompt for confirmation

	// Dependencies
	RequiredTools     []string `json:"required_tools,omitempty"`     // Tools that should run first
	SuggestedFollowup []string `json:"suggested_followup,omitempty"` // Tools to run after

	// Usage guidance
	BestForBatch     bool   `json:"best_for_batch"`    // Good for batch operations
	BestForRealtime  bool   `json:"best_for_realtime"` // Good for real-time use
	RecommendedLimit int    `json:"recommended_limit"` // Suggested result limit
	Notes            string `json:"notes,omitempty"`   // Additional guidance
}

// ToolCostRegistry maps tool names to their cost hints
var ToolCostRegistry = map[string]*CostHints{
	// Query tools
	"query_logs": {
		APICost:           CostMedium,
		TokenCost:         500,
		RateLimitImpact:   RateLimitModerate,
		ExecutionSpeed:    SpeedMedium,
		Cacheable:         true,
		CacheTTLSeconds:   60,
		Impact:            ImpactNone,
		Reversible:        true,
		RequiresConfirm:   false,
		SuggestedFollowup: []string{"investigate_incident", "create_dashboard", "suggest_alert"},
		BestForBatch:      false,
		BestForRealtime:   true,
		RecommendedLimit:  100,
		Notes:             "Use time filters to reduce response size and improve performance",
	},

	"submit_background_query": {
		APICost:           CostHigh,
		TokenCost:         100,
		RateLimitImpact:   RateLimitMinimal,
		ExecutionSpeed:    SpeedAsync,
		Cacheable:         false,
		Impact:            ImpactNone,
		Reversible:        true,
		RequiresConfirm:   false,
		RequiredTools:     []string{"build_query"},
		SuggestedFollowup: []string{"get_background_query_status", "get_background_query_data"},
		BestForBatch:      true,
		BestForRealtime:   false,
		Notes:             "Use for queries spanning more than 24 hours or expecting >10K results",
	},

	"build_query": {
		APICost:           CostFree,
		TokenCost:         200,
		RateLimitImpact:   RateLimitNone,
		ExecutionSpeed:    SpeedInstant,
		Cacheable:         true,
		CacheTTLSeconds:   3600,
		Impact:            ImpactNone,
		Reversible:        true,
		RequiresConfirm:   false,
		SuggestedFollowup: []string{"query_logs", "validate_query"},
		BestForRealtime:   true,
		Notes:             "Local query construction, no API calls",
	},

	// Alert tools
	"list_alerts": {
		APICost:           CostLow,
		TokenCost:         300,
		RateLimitImpact:   RateLimitMinimal,
		ExecutionSpeed:    SpeedFast,
		Cacheable:         true,
		CacheTTLSeconds:   300,
		Impact:            ImpactNone,
		Reversible:        true,
		SuggestedFollowup: []string{"get_alert", "create_alert"},
		BestForRealtime:   true,
		RecommendedLimit:  50,
	},

	"create_alert": {
		APICost:           CostLow,
		TokenCost:         150,
		RateLimitImpact:   RateLimitMinimal,
		ExecutionSpeed:    SpeedFast,
		Impact:            ImpactMedium,
		Reversible:        true,
		RequiresConfirm:   false,
		RequiredTools:     []string{"list_outgoing_webhooks"},
		SuggestedFollowup: []string{"list_alerts"},
		Notes:             "Creates real-time monitoring, consider using dry_run first",
	},

	"delete_alert": {
		APICost:         CostLow,
		TokenCost:       50,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactHigh,
		Reversible:      false,
		RequiresConfirm: true,
		Notes:           "Permanently deletes alert and all its history",
	},

	"suggest_alert": {
		APICost:           CostMedium,
		TokenCost:         400,
		RateLimitImpact:   RateLimitModerate,
		ExecutionSpeed:    SpeedMedium,
		Cacheable:         true,
		CacheTTLSeconds:   600,
		Impact:            ImpactNone,
		Reversible:        true,
		SuggestedFollowup: []string{"create_alert"},
		Notes:             "Analyzes log patterns to suggest alerts",
	},

	// Dashboard tools
	"list_dashboards": {
		APICost:           CostLow,
		TokenCost:         200,
		RateLimitImpact:   RateLimitMinimal,
		ExecutionSpeed:    SpeedFast,
		Cacheable:         true,
		CacheTTLSeconds:   300,
		Impact:            ImpactNone,
		SuggestedFollowup: []string{"get_dashboard"},
	},

	"create_dashboard": {
		APICost:         CostLow,
		TokenCost:       200,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactLow,
		Reversible:      true,
		Notes:           "Creates a new dashboard, can be complex with many widgets",
	},

	// Workflow tools
	"investigate_incident": {
		APICost:           CostHigh,
		TokenCost:         800,
		RateLimitImpact:   RateLimitHigh,
		ExecutionSpeed:    SpeedMedium,
		Cacheable:         false,
		Impact:            ImpactNone,
		SuggestedFollowup: []string{"suggest_alert", "create_dashboard"},
		BestForRealtime:   true,
		Notes:             "Runs multiple queries to analyze incident, may consume significant rate limit",
	},

	"health_check": {
		APICost:           CostMedium,
		TokenCost:         300,
		RateLimitImpact:   RateLimitModerate,
		ExecutionSpeed:    SpeedFast,
		Cacheable:         true,
		CacheTTLSeconds:   60,
		Impact:            ImpactNone,
		SuggestedFollowup: []string{"query_logs", "list_alerts"},
		BestForRealtime:   true,
		Notes:             "Quick system health overview",
	},

	// Ingestion tools
	"ingest_logs": {
		APICost:         CostMedium,
		TokenCost:       100,
		RateLimitImpact: RateLimitModerate,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactLow,
		Reversible:      false,
		BestForBatch:    true,
		Notes:           "Logs cannot be deleted after ingestion",
	},

	// Policy tools
	"create_policy": {
		APICost:         CostLow,
		TokenCost:       150,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactHigh,
		Reversible:      true,
		RequiresConfirm: false,
		Notes:           "Policies affect log routing and retention, consider using dry_run first",
	},

	"delete_policy": {
		APICost:         CostLow,
		TokenCost:       50,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactCritical,
		Reversible:      false,
		RequiresConfirm: true,
		Notes:           "May affect log routing and cause data loss",
	},

	// Webhook tools
	"create_outgoing_webhook": {
		APICost:           CostLow,
		TokenCost:         100,
		RateLimitImpact:   RateLimitMinimal,
		ExecutionSpeed:    SpeedFast,
		Impact:            ImpactMedium,
		Reversible:        true,
		SuggestedFollowup: []string{"create_alert"},
		Notes:             "Enables external notifications, consider testing with dry_run",
	},

	// E2M tools
	"create_e2m": {
		APICost:         CostLow,
		TokenCost:       150,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactMedium,
		Reversible:      true,
		Notes:           "Creates ongoing log-to-metric conversion, may increase metric volume",
	},

	// Discovery tools
	"discover_tools": {
		APICost:         CostFree,
		TokenCost:       300,
		RateLimitImpact: RateLimitNone,
		ExecutionSpeed:  SpeedInstant,
		Cacheable:       false,
		Impact:          ImpactNone,
		BestForRealtime: true,
		Notes:           "Local tool discovery, no API calls",
	},
}

// GetCostHints returns cost hints for a tool, or default hints if not found
func GetCostHints(toolName string) *CostHints {
	if hints, ok := ToolCostRegistry[toolName]; ok {
		return hints
	}

	// Return default hints for unknown tools
	return &CostHints{
		APICost:         CostLow,
		TokenCost:       100,
		RateLimitImpact: RateLimitMinimal,
		ExecutionSpeed:  SpeedFast,
		Impact:          ImpactNone,
		Reversible:      true,
	}
}

// ShouldConfirm returns true if the tool should prompt for confirmation
func ShouldConfirm(toolName string) bool {
	hints := GetCostHints(toolName)
	return hints.RequiresConfirm
}

// GetSuggestedFollowup returns tools to suggest after executing this tool
func GetSuggestedFollowup(toolName string) []string {
	hints := GetCostHints(toolName)
	return hints.SuggestedFollowup
}

// EstimateTokenCost estimates the total token cost for a sequence of tools
func EstimateTokenCost(toolNames []string) int {
	total := 0
	for _, name := range toolNames {
		hints := GetCostHints(name)
		total += hints.TokenCost
	}
	return total
}

// IsCacheable returns true if the tool's results can be cached
func IsCacheable(toolName string) bool {
	hints := GetCostHints(toolName)
	return hints.Cacheable
}

// GetCacheTTL returns the suggested cache TTL in seconds
func GetCacheTTL(toolName string) int {
	hints := GetCostHints(toolName)
	return hints.CacheTTLSeconds
}
