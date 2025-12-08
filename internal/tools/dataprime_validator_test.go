package tools

import (
	"strings"
	"testing"
)

func TestValidateDataPrimeQuery_TildeOperator(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		shouldError bool
	}{
		{
			name:        "valid contains function",
			query:       "source logs | filter $d.message.contains('error')",
			shouldError: false,
		},
		{
			name:        "valid matches function",
			query:       "source logs | filter $d.message.matches(/error.*timeout/)",
			shouldError: false,
		},
		{
			name:        "invalid ~~ operator",
			query:       "source logs | filter $d.message ~~ 'error'",
			shouldError: true,
		},
		{
			name:        "invalid !~~ operator",
			query:       "source logs | filter $d.message !~~ 'debug'",
			shouldError: true,
		},
		{
			name:        "valid == on $l field",
			query:       "source logs | filter $l.applicationname == 'myapp'",
			shouldError: false,
		},
		{
			name:        "valid startsWith function",
			query:       "source logs | filter $l.applicationname.startsWith('prod-')",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPrimeQuery(tt.query)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateDataPrimeQuery_FieldReferences(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		shouldError bool
		errorField  string
	}{
		{
			name:        "valid applicationname",
			query:       "source logs | filter $l.applicationname == 'myapp'",
			shouldError: false,
		},
		{
			name:        "valid subsystemname",
			query:       "source logs | filter $l.subsystemname == 'api'",
			shouldError: false,
		},
		{
			name:        "valid severity",
			query:       "source logs | filter $m.severity >= WARNING",
			shouldError: false,
		},
		{
			name:        "invalid $l.namespace",
			query:       "source logs | filter $l.namespace == 'default'",
			shouldError: true,
			errorField:  "$l.namespace",
		},
		{
			name:        "invalid $l.pod",
			query:       "source logs | filter $l.pod == 'my-pod'",
			shouldError: true,
			errorField:  "$l.pod",
		},
		{
			name:        "invalid $m.level",
			query:       "source logs | filter $m.level == 'error'",
			shouldError: true,
			errorField:  "$m.level",
		},
		{
			name:        "$d fields are not validated (dynamic)",
			query:       "source logs | filter $d.anything == 'value'",
			shouldError: false,
		},
		{
			name:        "valid computername label",
			query:       "source logs | filter $l.computername == 'server1'",
			shouldError: false,
		},
		{
			name:        "valid timestamp metadata",
			query:       "source logs | filter $m.timestamp > @'2024-01-01'",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPrimeQuery(tt.query)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorField != "" && err.Field != tt.errorField {
					t.Errorf("Expected error field %s, got %s", tt.errorField, err.Field)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateDataPrimeQuery_CommonMistakes(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "valid query",
			query:       "source logs | filter $l.applicationname == 'myapp' && $m.severity >= WARNING",
			shouldError: false,
		},
		{
			name:          "AND instead of &&",
			query:         "source logs | filter $l.applicationname == 'myapp' AND $m.severity >= WARNING",
			shouldError:   true,
			errorContains: "&&",
		},
		{
			name:          "OR instead of ||",
			query:         "source logs | filter $l.applicationname == 'app1' OR $l.applicationname == 'app2'",
			shouldError:   true,
			errorContains: "||",
		},
		{
			name:          "double quotes instead of single",
			query:         `source logs | filter $l.applicationname == "myapp"`,
			shouldError:   true,
			errorContains: "single quotes",
		},
		{
			name:          "typo $l.application instead of applicationname",
			query:         "source logs | filter $l.application == 'myapp'",
			shouldError:   true,
			errorContains: "applicationname",
		},
		{
			name:          "typo $m.level instead of severity",
			query:         "source logs | filter $m.level == ERROR",
			shouldError:   true,
			errorContains: "severity",
		},
		{
			name:          "LIKE keyword (SQL syntax)",
			query:         "source logs | filter $d.message LIKE '%error%'",
			shouldError:   true,
			errorContains: "LIKE",
		},
		{
			name:          "IN keyword (SQL syntax)",
			query:         "source logs | filter $l.applicationname IN ('app1', 'app2')",
			shouldError:   true,
			errorContains: "IN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPrimeQuery(tt.query)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateDataPrimeQuery_SeverityValues(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "valid severity ERROR",
			query:       "source logs | filter $m.severity == ERROR",
			shouldError: false,
		},
		{
			name:        "valid severity WARNING",
			query:       "source logs | filter $m.severity >= WARNING",
			shouldError: false,
		},
		{
			name:        "valid severity comparison",
			query:       "source logs | filter $m.severity != DEBUG",
			shouldError: false,
		},
		{
			name:        "invalid severity value",
			query:       "source logs | filter $m.severity == FATAL",
			shouldError: true,
		},
		{
			name:        "invalid lowercase severity",
			query:       "source logs | filter $m.severity == error",
			shouldError: false, // We uppercase and check, so 'error' becomes ERROR which is valid
		},
		{
			name:          "numeric severity value 4",
			query:         "source logs | filter $m.severity >= 4",
			shouldError:   true,
			errorContains: "Numeric severity values are not allowed",
		},
		{
			name:          "numeric severity value 0",
			query:         "source logs | filter $m.severity == 0",
			shouldError:   true,
			errorContains: "VERBOSE",
		},
		{
			name:          "numeric severity value comparison",
			query:         "source logs | filter $m.severity != 2",
			shouldError:   true,
			errorContains: "INFO",
		},
		{
			name:        "valid VERBOSE severity",
			query:       "source logs | filter $m.severity == VERBOSE",
			shouldError: false,
		},
		{
			name:        "valid CRITICAL severity",
			query:       "source logs | filter $m.severity == CRITICAL",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPrimeQuery(tt.query)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestSuggestQueryFix(t *testing.T) {
	tests := []struct {
		name          string
		errorMessage  string
		shouldContain []string
	}{
		{
			name:          "tilde operator error",
			errorMessage:  "~~ is not supported",
			shouldContain: []string{"contains", "matches", "Lucene"},
		},
		{
			name:          "keypath error",
			errorMessage:  "keypath does not exist",
			shouldContain: []string{"prefix", "$l.", "$m.", "$d."},
		},
		{
			name:          "compilation error",
			errorMessage:  "Compilation error: something went wrong",
			shouldContain: []string{"==", "&&", "single quotes"},
		},
		{
			name:          "unknown function error",
			errorMessage:  "unknown function: foo",
			shouldContain: []string{"count()", "contains()", "avg()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := SuggestQueryFix("test query", tt.errorMessage)
			for _, contain := range tt.shouldContain {
				if !strings.Contains(suggestion, contain) {
					t.Errorf("Expected suggestion to contain '%s', got: %s", contain, suggestion)
				}
			}
		})
	}
}

func TestFormatQueryError(t *testing.T) {
	query := "source logs | filter $l.namespace == 'default'"
	apiError := "keypath does not exist: '$l.namespace'"

	result := FormatQueryError(query, apiError)

	// Should contain the query
	if !strings.Contains(result, "namespace") {
		t.Error("Expected result to contain the query")
	}

	// Should contain the error
	if !strings.Contains(result, "keypath") {
		t.Error("Expected result to contain the error")
	}

	// Should contain suggestion
	if !strings.Contains(result, "prefix") {
		t.Error("Expected result to contain suggestions")
	}
}

func TestGetDataPrimeQuickReference(t *testing.T) {
	ref := GetDataPrimeQuickReference()

	// Should contain key sections
	sections := []string{
		"Field Access",
		"$d.",
		"$l.",
		"$m.",
		"Operators",
		"&&",
		"||",
		"String Functions",
		"contains",
		"matches",
		"Severity Levels",
		"ERROR",
		"WARNING",
	}

	for _, section := range sections {
		if !strings.Contains(ref, section) {
			t.Errorf("Quick reference should contain '%s'", section)
		}
	}
}
