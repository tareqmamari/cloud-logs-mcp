// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements the heuristic engine for intelligent investigation.
package tools

import (
	"fmt"
	"sort"
	"strings"
)

// HeuristicMatcher defines the interface for pattern-based heuristics
type HeuristicMatcher interface {
	Name() string
	Matches(finding InvestigationFinding, events []map[string]interface{}) bool
	SuggestAction(finding InvestigationFinding) HeuristicAction
	GetSOP() *SOPRecommendation
}

// SOPRecommendation represents a Standard Operating Procedure
type SOPRecommendation struct {
	Trigger    string `json:"trigger"`
	Procedure  string `json:"procedure"`
	Escalation string `json:"escalation"`
}

// HeuristicEngine runs all matchers and collects suggestions
type HeuristicEngine struct {
	matchers []HeuristicMatcher
}

// NewHeuristicEngine creates a new heuristic engine with all matchers
func NewHeuristicEngine() *HeuristicEngine {
	return &HeuristicEngine{
		matchers: []HeuristicMatcher{
			&TimeoutHeuristic{},
			&MemoryHeuristic{},
			&DatabaseHeuristic{},
			&AuthHeuristic{},
			&RateLimitHeuristic{},
			&NetworkHeuristic{},
		},
	}
}

// AnalyzeAndSuggest processes findings and returns suggested actions
func (e *HeuristicEngine) AnalyzeAndSuggest(findings []InvestigationFinding, events []map[string]interface{}) []HeuristicAction {
	actions := []HeuristicAction{}
	seen := make(map[string]bool) // Deduplicate suggestions

	for _, finding := range findings {
		for _, matcher := range e.matchers {
			if matcher.Matches(finding, events) {
				action := matcher.SuggestAction(finding)
				key := action.Description
				if !seen[key] {
					actions = append(actions, action)
					seen[key] = true
				}
			}
		}
	}

	// Sort by priority
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Priority < actions[j].Priority
	})

	return actions
}

// GetMatchingSOPs returns SOPs that match the findings
func (e *HeuristicEngine) GetMatchingSOPs(findings []InvestigationFinding, events []map[string]interface{}) []SOPRecommendation {
	sops := []SOPRecommendation{}
	seen := make(map[string]bool)

	for _, finding := range findings {
		for _, matcher := range e.matchers {
			if matcher.Matches(finding, events) {
				sop := matcher.GetSOP()
				if sop != nil && !seen[sop.Trigger] {
					sops = append(sops, *sop)
					seen[sop.Trigger] = true
				}
			}
		}
	}

	return sops
}

// ========================================================================
// Heuristic Implementations
// ========================================================================

// TimeoutHeuristic detects and responds to timeout patterns
type TimeoutHeuristic struct{}

// Name implements Heuristic.
func (h *TimeoutHeuristic) Name() string {
	return "timeout_detector"
}

// Matches implements Heuristic.
func (h *TimeoutHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"timeout", "timed out", "deadline exceeded",
		"context deadline", "read timeout", "write timeout",
		"connection timeout", "request timeout", "504",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *TimeoutHeuristic) SuggestAction(finding InvestigationFinding) HeuristicAction {
	query := ""
	if finding.Service != "" {
		query = fmt.Sprintf(`source logs
			| filter $l.applicationname == '%s'
			| filter $d.duration_ms.exists()
			| calculate
				avg($d.duration_ms) as avg_latency,
				percentile($d.duration_ms, 95) as p95_latency,
				percentile($d.duration_ms, 99) as p99_latency
			| limit 1`, finding.Service)
	}

	return HeuristicAction{
		Priority:    1,
		Type:        ActionCorrelate,
		Description: "Check downstream service health and network latency",
		Query:       query,
		Rationale:   "Timeout errors indicate slow downstream services or network issues",
	}
}

// GetSOP implements Heuristic.
func (h *TimeoutHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Timeout errors detected",
		Procedure: `1. Check downstream service health status
2. Review network latency metrics
3. Verify connection pool settings
4. Check for resource contention (CPU/Memory)
5. Review recent deployments or configuration changes`,
		Escalation: "If unresolved in 15 minutes, escalate to Platform team",
	}
}

// MemoryHeuristic detects memory-related issues
type MemoryHeuristic struct{}

// Name implements Heuristic.
func (h *MemoryHeuristic) Name() string {
	return "memory_detector"
}

// Matches implements Heuristic.
func (h *MemoryHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"out of memory", "oom", "heap space", "memory limit",
		"gc overhead", "allocation failure", "java.lang.outofmemory",
		"fatal error: runtime: out of memory", "oomkilled",
		"memory pressure", "memory leak",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *MemoryHeuristic) SuggestAction(_ InvestigationFinding) HeuristicAction {
	return HeuristicAction{
		Priority:    1,
		Type:        ActionQuery,
		Description: "Check container resource limits and memory trends",
		Rationale:   "Memory errors indicate potential leaks or insufficient limits",
	}
}

// GetSOP implements Heuristic.
func (h *MemoryHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Memory pressure detected",
		Procedure: `1. Check container memory limits (kubectl top pods)
2. Review JVM heap settings (-Xmx, -Xms)
3. Analyze heap dumps if available
4. Check for memory leaks in recent deployments
5. Consider horizontal scaling
6. Review object caching configurations`,
		Escalation: "If OOMKilled, escalate to Development team immediately",
	}
}

// DatabaseHeuristic detects database-related issues
type DatabaseHeuristic struct{}

// Name implements Heuristic.
func (h *DatabaseHeuristic) Name() string {
	return "database_detector"
}

// Matches implements Heuristic.
func (h *DatabaseHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"connection pool", "too many connections", "deadlock",
		"lock wait timeout", "cannot acquire", "database",
		"sql", "query failed", "transaction", "postgres", "mysql",
		"mongodb", "redis", "connection refused", "max_connections",
		"slow query", "query timeout",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *DatabaseHeuristic) SuggestAction(finding InvestigationFinding) HeuristicAction {
	query := ""
	if finding.Service != "" {
		query = fmt.Sprintf(`source logs
			| filter $l.applicationname == '%s' && $d.sql.exists()
			| groupby $d.sql
			| calculate
				avg($d.exec_ms) as avg_time,
				max($d.exec_ms) as max_time,
				count() as query_count
			| sortby -avg_time
			| limit 10`, finding.Service)
	}

	return HeuristicAction{
		Priority:    1,
		Type:        ActionQuery,
		Description: "Analyze slow database queries",
		Query:       query,
		Rationale:   "Database issues often cause cascading failures",
	}
}

// GetSOP implements Heuristic.
func (h *DatabaseHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Database connection/query issues detected",
		Procedure: `1. Check database connection pool settings
2. Review slow query logs
3. Check database CPU and memory utilization
4. Verify max_connections settings
5. Look for long-running transactions
6. Check for table locks or deadlocks`,
		Escalation: "If database-related, escalate to DBA team",
	}
}

// AuthHeuristic detects authentication/authorization issues
type AuthHeuristic struct{}

// Name implements Heuristic.
func (h *AuthHeuristic) Name() string {
	return "auth_detector"
}

// Matches implements Heuristic.
func (h *AuthHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"unauthorized", "forbidden", "401", "403",
		"authentication failed", "invalid token", "expired token",
		"access denied", "permission denied", "invalid credentials",
		"jwt", "oauth", "saml",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *AuthHeuristic) SuggestAction(finding InvestigationFinding) HeuristicAction {
	query := ""
	if finding.Service != "" {
		query = fmt.Sprintf(`source logs
			| filter $l.applicationname == '%s'
			| filter $d.message.contains('401') || $d.message.contains('403') || $d.message.contains('auth')
			| groupby $d.user_id, $d.endpoint
			| calculate count() as failures
			| sortby -failures
			| limit 20`, finding.Service)
	}

	return HeuristicAction{
		Priority:    2,
		Type:        ActionQuery,
		Description: "Investigate authentication failures",
		Query:       query,
		Rationale:   "Auth failures may indicate credential issues or security incidents",
	}
}

// GetSOP implements Heuristic.
func (h *AuthHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Authentication/Authorization failures detected",
		Procedure: `1. Verify service credentials and API keys
2. Check IAM policy changes
3. Review token expiration settings
4. Check for certificate issues
5. Verify OAuth/OIDC provider status
6. Review recent permission changes`,
		Escalation: "If security incident suspected, escalate to Security team immediately",
	}
}

// RateLimitHeuristic detects rate limiting issues
type RateLimitHeuristic struct{}

// Name implements Heuristic.
func (h *RateLimitHeuristic) Name() string {
	return "rate_limit_detector"
}

// Matches implements Heuristic.
func (h *RateLimitHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"rate limit", "429", "too many requests", "throttled",
		"quota exceeded", "limit exceeded", "backoff",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *RateLimitHeuristic) SuggestAction(_ InvestigationFinding) HeuristicAction {
	return HeuristicAction{
		Priority:    2,
		Type:        ActionCorrelate,
		Description: "Analyze request patterns and rate limits",
		Rationale:   "Rate limiting indicates traffic spikes or misconfigured limits",
	}
}

// GetSOP implements Heuristic.
func (h *RateLimitHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Rate limiting detected",
		Procedure: `1. Identify the source of excessive requests
2. Review rate limit configurations
3. Check for retry storms
4. Implement exponential backoff if not present
5. Consider request caching or batching
6. Contact API provider if external limit`,
		Escalation: "If business-critical, escalate to Engineering lead",
	}
}

// NetworkHeuristic detects network-related issues
type NetworkHeuristic struct{}

// Name implements Heuristic.
func (h *NetworkHeuristic) Name() string {
	return "network_detector"
}

// Matches implements Heuristic.
func (h *NetworkHeuristic) Matches(finding InvestigationFinding, _ []map[string]interface{}) bool {
	patterns := []string{
		"connection refused", "connection reset", "no route to host",
		"network unreachable", "dns", "econnrefused", "econnreset",
		"socket", "tcp", "ssl", "tls", "certificate",
		"502", "503", "bad gateway", "service unavailable",
	}

	summary := strings.ToLower(finding.Summary)
	for _, p := range patterns {
		if strings.Contains(summary, p) {
			return true
		}
	}
	return false
}

// SuggestAction implements Heuristic.
func (h *NetworkHeuristic) SuggestAction(_ InvestigationFinding) HeuristicAction {
	return HeuristicAction{
		Priority:    1,
		Type:        ActionCorrelate,
		Description: "Check network connectivity and DNS resolution",
		Rationale:   "Network errors indicate infrastructure or connectivity issues",
	}
}

// GetSOP implements Heuristic.
func (h *NetworkHeuristic) GetSOP() *SOPRecommendation {
	return &SOPRecommendation{
		Trigger: "Network connectivity issues detected",
		Procedure: `1. Verify DNS resolution
2. Check network policies and security groups
3. Verify service endpoints are accessible
4. Check load balancer health
5. Review SSL/TLS certificate validity
6. Check for network partitions`,
		Escalation: "If infrastructure-wide, escalate to Platform/Network team",
	}
}
