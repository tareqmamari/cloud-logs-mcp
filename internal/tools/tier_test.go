package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeTier verifies tier alias mapping
func TestNormalizeTier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Frequent search aliases
		{"PI alias", "PI", "frequent_search"},
		{"priority alias", "priority", "frequent_search"},
		{"insights alias", "insights", "frequent_search"},
		{"priority insights", "priority insights", "frequent_search"},
		{"quick alias", "quick", "frequent_search"},
		{"fast alias", "fast", "frequent_search"},
		{"hot alias", "hot", "frequent_search"},
		{"realtime alias", "realtime", "frequent_search"},
		{"real-time alias", "real-time", "frequent_search"},
		{"frequent alias", "frequent", "frequent_search"},

		// Archive aliases
		{"archive alias", "archive", "archive"},
		{"storage alias", "storage", "archive"},
		{"COS alias", "COS", "archive"},
		{"cold alias", "cold", "archive"},
		{"s3 alias", "s3", "archive"},
		{"object storage", "object storage", "archive"},
		{"long term", "long term", "archive"},
		{"long-term", "long-term", "archive"},
		{"historical", "historical", "archive"},

		// Valid API values (pass through)
		{"unspecified value", "unspecified", "unspecified"},
		{"archive value", "archive", "archive"},
		{"frequent_search value", "frequent_search", "frequent_search"},

		// Case insensitive
		{"uppercase PI", "PI", "frequent_search"},
		{"lowercase pi", "pi", "frequent_search"},
		{"mixed case Archive", "Archive", "archive"},

		// With whitespace
		{"whitespace pi", "  pi  ", "frequent_search"},
		{"whitespace archive", "  archive  ", "archive"},

		// Unknown value (defaults to archive)
		{"unknown value", "unknown", "archive"},
		{"empty string", "", "archive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTier(tt.input)
			assert.Equal(t, tt.expected, result, "normalizeTier(%q) should return %q", tt.input, tt.expected)
		})
	}
}
