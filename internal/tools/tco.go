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
	config := &TCOConfig{
		HasPolicies:       false,
		HasArchive:        true,
		HasFrequentSearch: true,              // No policies = logs go to both tiers
		DefaultTier:       "frequent_search", // Use frequent_search for faster queries
		PolicyCount:       0,
		LastUpdated:       time.Now(),
		Policies:          []TCOPolicyRule{},
	}

	policies, ok := result["policies"].([]interface{})
	if !ok || len(policies) == 0 {
		if logger != nil {
			logger.Debug("No TCO policies configured, using frequent_search tier for faster queries")
		}
		return config
	}

	config.HasPolicies = true
	config.PolicyCount = len(policies)
	// When policies exist, reset defaults - policies determine tier availability
	config.HasFrequentSearch = false
	config.DefaultTier = "archive" // Will be updated based on policy analysis

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

		// Check priority field to determine tier
		// type_high and type_medium route to frequent_search (Priority Insights)
		// type_low routes to archive (COS)
		priority, _ := policy["priority"].(string)

		// Check archive_retention configuration
		if archiveRetention, ok := policy["archive_retention"].(map[string]interface{}); ok {
			if id, ok := archiveRetention["id"].(string); ok && id != "" {
				config.HasArchive = true
			}
		}

		// Determine tier based on priority
		tier := "archive" // Default
		switch priority {
		case "type_high", "type_medium":
			config.HasFrequentSearch = true
			tier = "frequent_search"
		case "type_low", "type_unspecified", "":
			tier = "archive"
		}

		// Build policy rule
		policyRule := TCOPolicyRule{
			Tier:     tier,
			Priority: priority,
		}

		// Extract application rule if present
		if appRule, ok := policy["application_rule"].(map[string]interface{}); ok {
			if appName, ok := appRule["name"].(string); ok && appName != "" {
				ruleType, _ := appRule["rule_type_id"].(string)
				policyRule.ApplicationRule = &TCOMatchRule{
					Name:     appName,
					RuleType: ruleType,
				}
			}
		}

		// Extract subsystem rule if present
		if subRule, ok := policy["subsystem_rule"].(map[string]interface{}); ok {
			if subName, ok := subRule["name"].(string); ok && subName != "" {
				ruleType, _ := subRule["rule_type_id"].(string)
				policyRule.SubsystemRule = &TCOMatchRule{
					Name:     subName,
					RuleType: ruleType,
				}
			}
		}

		config.Policies = append(config.Policies, policyRule)
	}

	// Determine default tier based on analysis
	// Prefer frequent_search (faster queries) unless logs are only in archive
	if config.HasFrequentSearch {
		config.DefaultTier = "frequent_search"
	} else {
		// Only archive tier has logs
		config.DefaultTier = "archive"
	}

	if logger != nil {
		logger.Debug("TCO configuration analyzed",
			zap.Int("policy_count", config.PolicyCount),
			zap.Bool("has_archive", config.HasArchive),
			zap.Bool("has_frequent_search", config.HasFrequentSearch),
			zap.String("default_tier", config.DefaultTier),
			zap.Int("policy_rules", len(config.Policies)))
	}

	return config
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
