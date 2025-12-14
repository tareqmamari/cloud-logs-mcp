// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements TCO (Total Cost of Ownership) policy discovery and caching.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// FetchAndCacheTCOConfig fetches TCO policies from the API and caches the configuration
// in the session context. This should be called at session initialization.
func FetchAndCacheTCOConfig(ctx context.Context, c *client.Client, logger *zap.Logger) error {
	if c == nil {
		return nil // No client, skip TCO discovery
	}

	session := GetSessionFromContext(ctx)
	if session == nil {
		session = GetSession()
	}

	// Skip if config is fresh (less than 5 minutes old)
	if !session.IsTCOConfigStale() {
		return nil
	}

	config, err := fetchTCOConfig(ctx, c, logger)
	if err != nil {
		if logger != nil {
			logger.Warn("Failed to fetch TCO policies, using defaults",
				zap.Error(err))
		}
		// Set default config on error - use frequent_search for faster queries
		// When no policies exist, logs go to both tiers, so frequent_search is faster
		config = &TCOConfig{
			HasPolicies:       false,
			HasArchive:        true,
			HasFrequentSearch: true,
			DefaultTier:       "frequent_search",
			PolicyCount:       0,
			LastUpdated:       time.Now(),
		}
	}

	session.SetTCOConfig(config)
	return nil
}

// fetchTCOConfig fetches policies from the API and analyzes them
func fetchTCOConfig(ctx context.Context, c *client.Client, logger *zap.Logger) (*TCOConfig, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/policies",
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch policies: HTTP %d", resp.StatusCode)
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse policies response: %w", err)
	}

	return parseTCOPolicies(result, logger), nil
}

// parseTCOPolicies analyzes the policies response and extracts TCO configuration
func parseTCOPolicies(result map[string]interface{}, logger *zap.Logger) *TCOConfig {
	config := newDefaultTCOConfig()

	policies, ok := result["policies"].([]interface{})
	if !ok || len(policies) == 0 {
		if logger != nil {
			logger.Debug("No TCO policies configured, using frequent_search tier for faster queries")
		}
		return config
	}

	initConfigForPolicies(config, len(policies))
	processPolicies(config, policies)
	determineDefaultTier(config)
	logTCOConfig(config, logger)

	return config
}

// newDefaultTCOConfig creates a TCOConfig with default values for when no policies exist
func newDefaultTCOConfig() *TCOConfig {
	return &TCOConfig{
		HasPolicies:       false,
		HasArchive:        true,
		HasFrequentSearch: true,              // No policies = logs go to both tiers
		DefaultTier:       "frequent_search", // Use frequent_search for faster queries
		PolicyCount:       0,
		LastUpdated:       time.Now(),
		Policies:          []TCOPolicyRule{},
	}
}

// initConfigForPolicies initializes the config when policies exist
func initConfigForPolicies(config *TCOConfig, policyCount int) {
	config.HasPolicies = true
	config.PolicyCount = policyCount
	// When policies exist, reset defaults - policies determine tier availability
	config.HasFrequentSearch = false
	config.DefaultTier = "archive" // Will be updated based on policy analysis
}

// processPolicies processes each policy and updates the config
func processPolicies(config *TCOConfig, policies []interface{}) {
	// Analyze each policy to determine tier routing
	// Order matters - policies are processed in order, first match wins
	for _, p := range policies {
		policy, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if policy is enabled (default is true if not specified)
		if enabled, ok := policy["enabled"].(bool); ok && !enabled {
			continue // Skip disabled policies
		}

		processPolicy(config, policy)
	}
}

// processPolicy processes a single policy and adds it to the config
func processPolicy(config *TCOConfig, policy map[string]interface{}) {
	// Check priority field to determine tier
	// type_high and type_medium route to frequent_search (Priority Insights)
	// type_low routes to archive (COS)
	priority, _ := policy["priority"].(string)

	checkArchiveRetention(config, policy)
	tier := determineTier(config, priority)

	policyRule := TCOPolicyRule{
		Tier:     tier,
		Priority: priority,
	}

	policyRule.ApplicationRule = extractMatchRule(policy, "application_rule")
	policyRule.SubsystemRule = extractMatchRule(policy, "subsystem_rule")

	config.Policies = append(config.Policies, policyRule)
}

// checkArchiveRetention checks if archive retention is configured
func checkArchiveRetention(config *TCOConfig, policy map[string]interface{}) {
	if archiveRetention, ok := policy["archive_retention"].(map[string]interface{}); ok {
		if id, ok := archiveRetention["id"].(string); ok && id != "" {
			config.HasArchive = true
		}
	}
}

// determineTier determines the tier based on priority
func determineTier(config *TCOConfig, priority string) string {
	switch priority {
	case "type_high", "type_medium":
		config.HasFrequentSearch = true
		return "frequent_search"
	case "type_low", "type_unspecified", "":
		return "archive"
	default:
		return "archive"
	}
}

// extractMatchRule extracts a match rule (application or subsystem) from a policy
func extractMatchRule(policy map[string]interface{}, ruleKey string) *TCOMatchRule {
	rule, ok := policy[ruleKey].(map[string]interface{})
	if !ok {
		return nil
	}

	name, ok := rule["name"].(string)
	if !ok || name == "" {
		return nil
	}

	ruleType, _ := rule["rule_type_id"].(string)
	return &TCOMatchRule{
		Name:     name,
		RuleType: ruleType,
	}
}

// determineDefaultTier sets the default tier based on analysis
func determineDefaultTier(config *TCOConfig) {
	// Prefer frequent_search (faster queries) unless logs are only in archive
	if config.HasFrequentSearch {
		config.DefaultTier = "frequent_search"
	} else {
		// Only archive tier has logs
		config.DefaultTier = "archive"
	}
}

// logTCOConfig logs the TCO configuration
func logTCOConfig(config *TCOConfig, logger *zap.Logger) {
	if logger != nil {
		logger.Debug("TCO configuration analyzed",
			zap.Int("policy_count", config.PolicyCount),
			zap.Bool("has_archive", config.HasArchive),
			zap.Bool("has_frequent_search", config.HasFrequentSearch),
			zap.String("default_tier", config.DefaultTier),
			zap.Int("policy_rules", len(config.Policies)))
	}
}

// GetTCOSummary returns a human-readable summary of TCO configuration
func GetTCOSummary(session *SessionContext) string {
	if session == nil {
		session = GetSession()
	}

	config := session.GetTCOConfig()
	if config == nil {
		return "TCO configuration not loaded. Logs default to frequent_search tier."
	}

	var sb strings.Builder
	sb.WriteString("TCO Configuration:\n")

	if !config.HasPolicies {
		sb.WriteString("- No TCO policies configured\n")
		sb.WriteString("- All logs go to both tiers, using frequent_search for faster queries\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("- %d policies configured\n", config.PolicyCount))

	if config.HasArchive {
		sb.WriteString("- Archive tier: enabled\n")
	}
	if config.HasFrequentSearch {
		sb.WriteString("- Frequent search tier: enabled\n")
	}

	sb.WriteString("- Default tier for queries: " + config.DefaultTier + "\n")

	if len(config.Policies) > 0 {
		sb.WriteString("- Policy rules (in order):\n")
		for i, policy := range config.Policies {
			sb.WriteString(fmt.Sprintf("  %d. ", i+1))
			if policy.ApplicationRule != nil {
				sb.WriteString(fmt.Sprintf("app %s '%s'", policy.ApplicationRule.RuleType, policy.ApplicationRule.Name))
			}
			if policy.SubsystemRule != nil {
				if policy.ApplicationRule != nil {
					sb.WriteString(" AND ")
				}
				sb.WriteString(fmt.Sprintf("subsystem %s '%s'", policy.SubsystemRule.RuleType, policy.SubsystemRule.Name))
			}
			if policy.ApplicationRule == nil && policy.SubsystemRule == nil {
				sb.WriteString("(all logs)")
			}
			sb.WriteString(fmt.Sprintf(" -> %s\n", policy.Tier))
		}
	}

	return sb.String()
}
