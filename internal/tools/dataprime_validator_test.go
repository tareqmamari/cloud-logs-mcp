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
		// Note: Numeric severity values are now auto-corrected by AutoCorrectDataPrimeQuery(),
		// not rejected by ValidateDataPrimeQuery(). These tests verify validation passes
		// after auto-correction is applied.
		{
			name:        "numeric severity value 4 (auto-corrected to ERROR)",
			query:       "source logs | filter $m.severity >= ERROR", // After auto-correction
			shouldError: false,
		},
		{
			name:        "numeric severity value 0 (auto-corrected to VERBOSE)",
			query:       "source logs | filter $m.severity == VERBOSE", // After auto-correction
			shouldError: false,
		},
		{
			name:        "numeric severity value comparison (auto-corrected to INFO)",
			query:       "source logs | filter $m.severity != INFO", // After auto-correction
			shouldError: false,
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

func TestAutoCorrectDataPrimeQuery(t *testing.T) {
	tests := []struct {
		name                string
		query               string
		expectedQuery       string
		expectedCorrections int
	}{
		{
			name:                "no corrections needed",
			query:               "source logs | filter $m.severity >= ERROR",
			expectedQuery:       "source logs | filter $m.severity >= ERROR",
			expectedCorrections: 0,
		},
		// Mixed-type field string method corrections
		{
			name:                "$d.message.contains needs type cast",
			query:               "source logs | filter $d.message.contains('error')",
			expectedQuery:       "source logs | filter $d.message:string.contains('error')",
			expectedCorrections: 1,
		},
		{
			name:                "$d.msg.startsWith needs type cast",
			query:               "source logs | filter $d.msg.startsWith('ERROR')",
			expectedQuery:       "source logs | filter $d.msg:string.startsWith('ERROR')",
			expectedCorrections: 1,
		},
		{
			name:                "$d.log.matches needs type cast",
			query:               "source logs | filter $d.log.matches(/timeout/)",
			expectedQuery:       "source logs | filter $d.log:string.matches(/timeout/)",
			expectedCorrections: 1,
		},
		{
			name:                "$d.error.contains needs type cast",
			query:               "source logs | filter $d.error.contains('connection refused')",
			expectedQuery:       "source logs | filter $d.error:string.contains('connection refused')",
			expectedCorrections: 1,
		},
		{
			name:                "multiple mixed-type field corrections",
			query:               "source logs | filter $d.message.contains('error') && $d.details.contains('timeout')",
			expectedQuery:       "source logs | filter $d.message:string.contains('error') && $d.details:string.contains('timeout')",
			expectedCorrections: 2,
		},
		{
			name:                "already has type cast - no change",
			query:               "source logs | filter $d.message:string.contains('error')",
			expectedQuery:       "source logs | filter $d.message:string.contains('error')",
			expectedCorrections: 0,
		},
		{
			name:                "non-mixed-type field unchanged",
			query:               "source logs | filter $d.custom_field.contains('value')",
			expectedQuery:       "source logs | filter $d.custom_field.contains('value')",
			expectedCorrections: 0,
		},
		{
			name:                "numeric severity 5 to CRITICAL",
			query:               "source logs | filter $m.severity >= 5",
			expectedQuery:       "source logs | filter $m.severity >= CRITICAL",
			expectedCorrections: 1,
		},
		{
			name:                "numeric severity 4 to ERROR",
			query:               "source logs | filter $m.severity == 4",
			expectedQuery:       "source logs | filter $m.severity == ERROR",
			expectedCorrections: 1,
		},
		{
			name:                "numeric severity 0 to VERBOSE",
			query:               "source logs | filter $m.severity != 0",
			expectedQuery:       "source logs | filter $m.severity != VERBOSE",
			expectedCorrections: 1,
		},
		{
			name:                "multiple numeric severities",
			query:               "source logs | filter $m.severity >= 4 && $m.severity < 6",
			expectedQuery:       "source logs | filter $m.severity >= ERROR && $m.severity < CRITICAL",
			expectedCorrections: 2,
		},
		{
			name:                "lowercase severity typo",
			query:               "source logs | filter $m.severity == error",
			expectedQuery:       "source logs | filter $m.severity == ERROR",
			expectedCorrections: 1,
		},
		{
			name:                "abbreviated severity WARN",
			query:               "source logs | filter $m.severity >= WARN",
			expectedQuery:       "source logs | filter $m.severity >= WARNING",
			expectedCorrections: 1,
		},
		{
			name:                "abbreviated severity ERR",
			query:               "source logs | filter $m.severity == ERR",
			expectedQuery:       "source logs | filter $m.severity == ERROR",
			expectedCorrections: 1,
		},
		// sort -> orderby corrections
		{
			name:                "sort ascending to orderby",
			query:               "source logs | filter $l.applicationname == 'myapp' | sort $m.timestamp | limit 50",
			expectedQuery:       "source logs | filter $l.applicationname == 'myapp' | orderby $m.timestamp | limit 50",
			expectedCorrections: 1,
		},
		{
			name:                "sort descending to orderby desc",
			query:               "source logs | filter $l.subsystemname == 'api' | sort -$m.timestamp | limit 100",
			expectedQuery:       "source logs | filter $l.subsystemname == 'api' | orderby $m.timestamp desc | limit 100",
			expectedCorrections: 1,
		},
		{
			name:                "sort with $d field",
			query:               "source logs | sort -$d.response_time | limit 10",
			expectedQuery:       "source logs | orderby $d.response_time desc | limit 10",
			expectedCorrections: 1,
		},
		// sort with aggregated column names (without $ prefix)
		{
			name:                "sort with aggregated column name descending",
			query:               "source logs | groupby $l.applicationname aggregate count() as error_count | sort -error_count | limit 20",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate count() as error_count | orderby error_count desc | limit 20",
			expectedCorrections: 1,
		},
		{
			name:                "sort with aggregated column name ascending",
			query:               "source logs | groupby $l.applicationname aggregate count() as total | sort total | limit 10",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate count() as total | orderby total | limit 10",
			expectedCorrections: 1,
		},
		{
			name:                "sort with underscore in column name",
			query:               "source logs | groupby $l.applicationname aggregate avg($d.response_time) as avg_response_time | sort -avg_response_time",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate avg($d.response_time) as avg_response_time | orderby avg_response_time desc",
			expectedCorrections: 1,
		},
		// aggregate syntax corrections: "aggregate alias = func()" -> "aggregate func() as alias"
		{
			name:                "aggregate assignment syntax to as alias",
			query:               "source logs | groupby $l.applicationname aggregate total = count()",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate count() as total",
			expectedCorrections: 1,
		},
		{
			name:                "aggregate assignment with sum",
			query:               "source logs | groupby $l.applicationname aggregate total_bytes = sum($d.bytes)",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate sum($d.bytes) as total_bytes",
			expectedCorrections: 1,
		},
		{
			name:                "aggregate correct syntax unchanged",
			query:               "source logs | groupby $l.applicationname aggregate count() as total",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate count() as total",
			expectedCorrections: 0,
		},
		// count_distinct/distinct_count -> approx_count_distinct
		{
			name:                "count_distinct to approx_count_distinct",
			query:               "source logs | groupby $l.applicationname aggregate count_distinct($l.subsystemname) as unique_subsystems",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate approx_count_distinct($l.subsystemname) as unique_subsystems",
			expectedCorrections: 1,
		},
		{
			name:                "distinct_count to approx_count_distinct",
			query:               "source logs | groupby $l.applicationname aggregate distinct_count($d.user_id) as unique_users",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate approx_count_distinct($d.user_id) as unique_users",
			expectedCorrections: 1,
		},
		{
			name:                "approx_count_distinct unchanged",
			query:               "source logs | groupby $l.applicationname aggregate approx_count_distinct($l.subsystemname) as unique_subsystems",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate approx_count_distinct($l.subsystemname) as unique_subsystems",
			expectedCorrections: 0,
		},
		// time bucket corrections: $m.timestamp:1m -> roundTime($m.timestamp, 1m)
		{
			name:                "timestamp colon syntax to roundTime",
			query:               "source logs | groupby $m.timestamp:1m as time_bucket aggregate count() as cnt",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as cnt",
			expectedCorrections: 1,
		},
		{
			name:                "timestamp colon syntax with hours",
			query:               "source logs | groupby $m.timestamp:1h as bucket aggregate count() as cnt",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 1h) as bucket aggregate count() as cnt",
			expectedCorrections: 1,
		},
		// bucket() -> roundTime()
		{
			name:                "bucket function to roundTime",
			query:               "source logs | groupby bucket($m.timestamp, 5m) as time_bucket aggregate count() as errors",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 5m) as time_bucket aggregate count() as errors",
			expectedCorrections: 1,
		},
		{
			name:                "bucket function with hours",
			query:               "source logs | groupby bucket($m.timestamp, 1h) as bucket aggregate count() as cnt",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 1h) as bucket aggregate count() as cnt",
			expectedCorrections: 1,
		},
		{
			name:                "roundTime unchanged",
			query:               "source logs | groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as cnt",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 1m) as time_bucket aggregate count() as cnt",
			expectedCorrections: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrected, corrections := AutoCorrectDataPrimeQuery(tt.query)
			if corrected != tt.expectedQuery {
				t.Errorf("Expected query '%s', got '%s'", tt.expectedQuery, corrected)
			}
			if len(corrections) != tt.expectedCorrections {
				t.Errorf("Expected %d corrections, got %d: %v", tt.expectedCorrections, len(corrections), corrections)
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

func TestPrepareQuery(t *testing.T) {
	tests := []struct {
		name                string
		query               string
		tier                string
		syntax              string
		expectedQuery       string
		expectedCorrections int
		expectError         bool
	}{
		{
			name:                "valid query passes through",
			query:               "source logs | filter $m.severity >= ERROR | limit 10",
			tier:                "archive",
			syntax:              "dataprime",
			expectedQuery:       "source logs | filter $m.severity >= ERROR | limit 10",
			expectedCorrections: 0,
			expectError:         false,
		},
		{
			name:                "auto-corrects numeric severity",
			query:               "source logs | filter $m.severity >= 5 | limit 10",
			tier:                "archive",
			syntax:              "dataprime",
			expectedQuery:       "source logs | filter $m.severity >= CRITICAL | limit 10",
			expectedCorrections: 1,
			expectError:         false,
		},
		{
			name:                "auto-corrects count_distinct",
			query:               "source logs | groupby $l.applicationname aggregate count_distinct($l.subsystemname) as unique",
			tier:                "frequent_search",
			syntax:              "dataprime",
			expectedQuery:       "source logs | groupby $l.applicationname aggregate approx_count_distinct($l.subsystemname) as unique",
			expectedCorrections: 1,
			expectError:         false,
		},
		{
			name:                "auto-corrects bucket to roundTime",
			query:               "source logs | groupby bucket($m.timestamp, 1h) as bucket aggregate count() as cnt",
			tier:                "archive",
			syntax:              "dataprime",
			expectedQuery:       "source logs | groupby roundTime($m.timestamp, 1h) as bucket aggregate count() as cnt",
			expectedCorrections: 1,
			expectError:         false,
		},
		{
			name:                "skips non-dataprime queries",
			query:               "error AND timeout",
			tier:                "archive",
			syntax:              "lucene",
			expectedQuery:       "error AND timeout",
			expectedCorrections: 0,
			expectError:         false,
		},
		{
			name:                "rejects invalid query after corrections",
			query:               "source logs | filter $l.invalid_field ~~ 'test'",
			tier:                "archive",
			syntax:              "dataprime",
			expectedQuery:       "",
			expectedCorrections: 0,
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrected, corrections, err := PrepareQuery(tt.query, tt.tier, tt.syntax)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if corrected != tt.expectedQuery {
				t.Errorf("Expected query '%s', got '%s'", tt.expectedQuery, corrected)
			}

			if len(corrections) != tt.expectedCorrections {
				t.Errorf("Expected %d corrections, got %d: %v", tt.expectedCorrections, len(corrections), corrections)
			}
		})
	}
}

func TestValidateNoInjectionPatterns(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		shouldError bool
		description string
	}{
		// Valid queries that should pass
		{
			name:        "valid simple query",
			query:       "source logs | filter $m.severity >= ERROR | limit 100",
			shouldError: false,
			description: "Normal DataPrime query",
		},
		{
			name:        "valid query with contains",
			query:       "source logs | filter $d.message:string.contains('DROP TABLE')",
			shouldError: false,
			description: "Searching for SQL keywords in log content is legitimate",
		},
		{
			name:        "valid query with SELECT keyword in filter value",
			query:       "source logs | filter $d.query:string.contains('SELECT * FROM users')",
			shouldError: false,
			description: "SQL in log content is fine",
		},
		{
			name:        "valid query with double dash in string",
			query:       "source logs | filter $d.message:string.contains('value--other')",
			shouldError: false,
			description: "Double dash in middle of string is fine",
		},

		// SQL injection patterns that should be blocked
		{
			name:        "DROP TABLE injection",
			query:       "source logs; DROP TABLE users; --",
			shouldError: true,
			description: "Classic SQL injection with DROP",
		},
		{
			name:        "DELETE injection",
			query:       "source logs; DELETE FROM logs WHERE 1=1",
			shouldError: true,
			description: "DELETE statement injection",
		},
		{
			name:        "UNION SELECT injection",
			query:       "source logs UNION SELECT * FROM passwords",
			shouldError: true,
			description: "UNION-based SQL injection",
		},
		{
			name:        "UNION ALL SELECT injection",
			query:       "source logs UNION ALL SELECT username, password FROM users",
			shouldError: true,
			description: "UNION ALL injection variant",
		},
		{
			name:        "trailing comment injection",
			query:       "source logs | filter $l.applicationname == 'app' --",
			shouldError: true,
			description: "Trailing SQL comment used to bypass filters",
		},
		{
			name:        "block comment injection",
			query:       "source logs /* bypass */ | filter $m.severity >= ERROR",
			shouldError: true,
			description: "Block comment used for obfuscation",
		},
		{
			name:        "semicolon with comment",
			query:       "source logs; --comment",
			shouldError: true,
			description: "Statement termination with comment",
		},
		{
			name:        "OR 1=1 injection",
			query:       "source logs | filter $l.applicationname == 'x' OR 1=1",
			shouldError: true,
			description: "Classic OR injection to bypass filters",
		},
		{
			name:        "OR '1'='1' injection",
			query:       "source logs | filter $l.applicationname == 'x' OR '1'='1'",
			shouldError: true,
			description: "String-based OR injection",
		},
		{
			name:        "EXEC injection",
			query:       "source logs; EXEC('malicious code')",
			shouldError: true,
			description: "Command execution attempt",
		},
		{
			name:        "xp_cmdshell injection",
			query:       "source logs; xp_cmdshell 'whoami'",
			shouldError: true,
			description: "SQL Server command execution",
		},
		{
			name:        "system object access",
			query:       "source logs; SELECT * FROM sys.dm_exec_requests",
			shouldError: true,
			description: "System DMV access attempt",
		},
		{
			name:        "stacked SELECT",
			query:       "source logs; SELECT password FROM users",
			shouldError: true,
			description: "Stacked query with SELECT",
		},
		{
			name:        "stacked INSERT",
			query:       "source logs; INSERT INTO audit VALUES ('hacked')",
			shouldError: true,
			description: "Stacked query with INSERT",
		},
		{
			name:        "stacked UPDATE",
			query:       "source logs; UPDATE users SET admin=1",
			shouldError: true,
			description: "Stacked query with UPDATE",
		},
		{
			name:        "CREATE injection",
			query:       "source logs; CREATE TABLE backdoor (data text)",
			shouldError: true,
			description: "CREATE statement injection",
		},
		{
			name:        "ALTER injection",
			query:       "source logs; ALTER TABLE users ADD admin boolean",
			shouldError: true,
			description: "ALTER statement injection",
		},
		{
			name:        "TRUNCATE injection",
			query:       "source logs; TRUNCATE TABLE audit_log",
			shouldError: true,
			description: "TRUNCATE statement injection",
		},
		{
			name:        "case insensitive DROP",
			query:       "source logs; drop table users",
			shouldError: true,
			description: "Lowercase SQL keywords should still be caught",
		},
		{
			name:        "mixed case UNION",
			query:       "source logs UnIoN SeLeCt * FROM secrets",
			shouldError: true,
			description: "Mixed case evasion attempt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPrimeQuery(tt.query)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for injection pattern (%s) but got nil", tt.description)
				} else if !strings.Contains(err.Message, "unsafe") {
					t.Errorf("Expected 'unsafe' in error message, got: %s", err.Message)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid query (%s): %s", tt.description, err.Error())
				}
			}
		})
	}
}
