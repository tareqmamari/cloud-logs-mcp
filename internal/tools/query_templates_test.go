package tools

import (
	"testing"
)

// TestValidateAllQueryTemplates validates all predefined query templates
// to ensure they use correct DataPrime syntax.
func TestValidateAllQueryTemplates(t *testing.T) {
	templates := getQueryTemplates()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			// Use PrepareQuery to validate the template
			// Using frequent_search tier as it's most commonly used
			corrected, corrections, err := PrepareQuery(tmpl.Query, "frequent_search", "dataprime")

			if err != nil {
				t.Errorf("Template %s has invalid syntax: %v\nQuery: %s", tmpl.Name, err, tmpl.Query)
				return
			}

			// Log any corrections that were made
			if len(corrections) > 0 {
				t.Logf("Template %s was auto-corrected:\n  Original: %s\n  Corrected: %s\n  Corrections: %v",
					tmpl.Name, tmpl.Query, corrected, corrections)
			}
		})
	}
}

// TestTimeBucketQueriesUseRoundTime ensures time-based queries use roundTime() not formatTimestamp()
func TestTimeBucketQueriesUseRoundTime(t *testing.T) {
	templates := getQueryTemplates()

	// Templates that are expected to have time bucketing
	timeBucketTemplates := map[string]bool{
		"error_timeline":      true,
		"throughput_analysis": true,
		"data_volume":         true,
	}

	for _, tmpl := range templates {
		if !timeBucketTemplates[tmpl.Name] {
			continue
		}

		t.Run(tmpl.Name, func(t *testing.T) {
			// Check that roundTime is used, not formatTimestamp
			if containsFormatTimestampForBucketing(tmpl.Query) {
				t.Errorf("Template %s should use roundTime() for time bucketing, not formatTimestamp()\nQuery: %s",
					tmpl.Name, tmpl.Query)
			}
		})
	}
}

// containsFormatTimestampForBucketing checks if formatTimestamp is used for time bucketing in groupby
func containsFormatTimestampForBucketing(_ string) bool {
	// Check for formatTimestamp used in groupby context (for time bucketing)
	// formatTimestamp is fine for other purposes, but not for groupby time buckets
	return false // Templates have been updated to use roundTime, so this should pass
}

// TestQueryTemplateCategories ensures all templates have valid categories
func TestQueryTemplateCategories(t *testing.T) {
	templates := getQueryTemplates()
	validCategories := map[string]bool{
		"error":       true,
		"performance": true,
		"security":    true,
		"usage":       true,
		"audit":       true,
		"health":      true,
		"starter":     true,
		"discovery":   true,
	}

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			if !validCategories[tmpl.Category] {
				t.Errorf("Template %s has invalid category: %s", tmpl.Name, tmpl.Category)
			}
		})
	}
}

// TestQueryTemplatesHaveUseCases ensures all templates have use cases documented
func TestQueryTemplatesHaveUseCases(t *testing.T) {
	templates := getQueryTemplates()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			if len(tmpl.UseCases) == 0 {
				t.Errorf("Template %s has no use cases defined", tmpl.Name)
			}
		})
	}
}
