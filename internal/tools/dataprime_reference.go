package tools

// DataPrime Reference - Comprehensive language knowledge for IBM Cloud Logs (powered by Coralogix)
// This file contains complete DataPrime syntax, commands, functions, and best practices.

// DataPrime is a piped syntax query language for log and span analysis.
// Queries flow from left to right: source logs | filter ... | groupby ... | limit ...

// ==================== ACCESS MECHANISMS ====================
// DataPrime uses prefixes to access different data layers:
//
// $d - User Data (the log payload/content)
//      - Default prefix, can be omitted in most cases
//      - Contains all user log data: $d.user_id, $d.http.request.headers, etc.
//      - For JSON fields: $d.json_field or just json_field
//      - Examples: $d.message, $d.status_code, $d.kubernetes.pod_name
//
// $l - Labels (application-level metadata)
//      - applicationname - Application identifier (often maps to K8s namespace)
//      - subsystemname - Subsystem/component name
//      - computername - Host/computer name
//      - ipaddress - IP address
//      - threadid, processid - Thread/process identifiers
//      - classname, methodname - Code location
//      - category - Log category
//
// $m - Metadata (system-level metadata)
//      - severity - Log severity level (DEBUG, VERBOSE, INFO, WARNING, ERROR, CRITICAL)
//      - timestamp - Log timestamp
//      - priority - Log priority
//
// $p - Parameters (template variables in dashboards)
//      - Used to reference dashboard variables: $p.variableName

// SeverityLevels defines the valid severity levels and their numeric values
var SeverityLevels = map[string]int{
	"VERBOSE":  0,
	"DEBUG":    1,
	"INFO":     2,
	"WARNING":  3,
	"ERROR":    4,
	"CRITICAL": 5,
}

// ValidLabelFields contains all valid $l. (label) fields
var ValidLabelFields = map[string]bool{
	"applicationname": true,
	"subsystemname":   true,
	"computername":    true,
	"ipaddress":       true,
	"threadid":        true,
	"processid":       true,
	"classname":       true,
	"methodname":      true,
	"category":        true,
}

// ValidMetadataFields contains all valid $m. (metadata) fields
var ValidMetadataFields = map[string]bool{
	"severity":  true,
	"timestamp": true,
	"priority":  true,
}

// ==================== COMMANDS ====================
// DataPrime commands operate on rows and datasets.
// Commands are chained with the pipe (|) operator.

// DataPrimeCommands contains all available DataPrime commands
var DataPrimeCommands = map[string]DataPrimeCommandInfo{
	// Source Commands
	"source": {
		Name:        "source",
		Description: "Primary data access mechanism - specifies the data source",
		Syntax:      "source <source_name>",
		Examples:    []string{"source logs", "source spans"},
	},
	"around": {
		Name:        "around",
		Description: "Time-based query around a specific point",
		Syntax:      "around <timestamp> <duration>",
		Examples:    []string{"source logs | around @'2024-01-01T12:00:00' 5m"},
	},
	"between": {
		Name:        "between",
		Description: "Query data within a defined time range",
		Syntax:      "between <start_timestamp> and <end_timestamp>",
		Examples:    []string{"source logs | between @'2024-01-01' and @'2024-01-02'"},
	},
	"last": {
		Name:        "last",
		Description: "Query data from the most recent period",
		Syntax:      "last <duration>",
		Examples:    []string{"source logs | last 1h", "source logs | last 30m"},
	},
	"timeshifted": {
		Name:        "timeshifted",
		Description: "Query data with temporal offset for historical comparison",
		Syntax:      "timeshifted <duration>",
		Examples:    []string{"source logs | timeshifted 1d"},
	},

	// Data Manipulation Commands
	"filter": {
		Name:        "filter",
		Aliases:     []string{"f", "where"},
		Description: "Remove events that do not match a condition",
		Syntax:      "filter <condition>",
		Examples: []string{
			"filter $m.severity == ERROR",
			"filter $l.applicationname == 'myapp'",
			"filter status_code >= 400 && status_code < 500",
		},
	},
	"aggregate": {
		Name:        "aggregate",
		Aliases:     []string{"agg"},
		Description: "Perform calculations across the entire dataset",
		Syntax:      "aggregate <aggregation_expression> [as <alias>]",
		Examples: []string{
			"aggregate count() as total",
			"aggregate avg(duration) as avg_duration, max(duration) as max_duration",
		},
	},
	"groupby": {
		Name:        "groupby",
		Description: "Aggregate documents that share common values",
		Syntax:      "groupby <expression> [as <alias>] [aggregate <agg_func>]",
		Examples: []string{
			"groupby $l.applicationname aggregate count() as cnt",
			"groupby status_code aggregate avg(duration) as avg_time",
			"groupby true aggregate count()", // Use 'true' as anchor for total aggregation
		},
	},
	"create": {
		Name:        "create",
		Aliases:     []string{"add", "c", "a"},
		Description: "Define a new field based on an expression",
		Syntax:      "create <keypath> from <expression>",
		Examples: []string{
			"create full_name from concat(first_name, ' ', last_name)",
			"create is_error from status_code >= 400",
			"create duration_ms from duration * 1000",
		},
	},
	"extract": {
		Name:        "extract",
		Description: "Parse unstructured data into structured fields",
		Syntax:      "extract <field> into <target> using <extractor>",
		Examples: []string{
			"extract message into fields using regexp(e=/(?<user>\\w+) did (?<action>\\w+)/)",
			"extract json_string into parsed using jsonobject()",
			"extract kvpairs into fields using kv()",
		},
	},
	"join": {
		Name:        "join",
		Description: "Merge results of current query with a second query",
		Syntax:      "join [inner|full] (<subquery>) on left=><field> == right=><field> into <keypath>",
		Examples: []string{
			"source users | join (source logins | countby id) on left=>id == right=>id into logins",
		},
		Notes: []string{
			"Join condition only supports keypath equality (==)",
			"One side must be small (< 200MB)",
			"Use filter and remove to reduce query size",
		},
	},
	"limit": {
		Name:        "limit",
		Aliases:     []string{"l"},
		Description: "Restrict the number of returned records",
		Syntax:      "limit <number>",
		Examples:    []string{"limit 100", "orderby timestamp desc | limit 10"},
		Notes:       []string{"Does not guarantee order unless used after orderby"},
	},
	"orderby": {
		Name:        "orderby",
		Aliases:     []string{"sortby"},
		Description: "Sort results by specified expression",
		Syntax:      "orderby <expression> [asc|desc]",
		Examples:    []string{"orderby timestamp desc", "orderby count asc"},
	},
	"top": {
		Name:        "top",
		Description: "Return highest-ranked records (combines orderby + limit)",
		Syntax:      "top <limit> <expression> by <order_expression>",
		Examples:    []string{"top 5 time_taken_ms by action, user"},
	},
	"bottom": {
		Name:        "bottom",
		Description: "Return lowest-ranked records",
		Syntax:      "bottom <limit> <expression> by <order_expression>",
		Examples:    []string{"bottom 10 response_time by endpoint"},
	},
	"distinct": {
		Name:        "distinct",
		Description: "Return unique values",
		Syntax:      "distinct <expression>",
		Examples:    []string{"distinct user_id", "distinct $l.applicationname"},
	},
	"count": {
		Name:        "count",
		Description: "Calculate record totals",
		Syntax:      "count",
		Examples:    []string{"source logs | count"},
	},
	"countby": {
		Name:        "countby",
		Description: "Grouped counting operations",
		Syntax:      "countby <expression>",
		Examples:    []string{"countby status_code", "countby $l.applicationname"},
	},
	"choose": {
		Name:        "choose",
		Description: "Select specific fields to include in output",
		Syntax:      "choose <field1>, <field2>, ...",
		Examples:    []string{"choose timestamp, message, severity"},
	},
	"remove": {
		Name:        "remove",
		Description: "Remove fields from output",
		Syntax:      "remove <field1>, <field2>, ...",
		Examples:    []string{"remove sensitive_data, internal_id"},
	},
	"move": {
		Name:        "move",
		Description: "Reposition/rename fields",
		Syntax:      "move <source> to <destination>",
		Examples:    []string{"move old_name to new_name"},
	},
	"redact": {
		Name:        "redact",
		Description: "Mask sensitive information",
		Syntax:      "redact <field> [using <pattern>]",
		Examples:    []string{"redact credit_card", "redact email using /\\w+@/"},
	},
	"replace": {
		Name:        "replace",
		Description: "Substitution operations on strings",
		Syntax:      "replace <field> <pattern> with <replacement>",
		Examples:    []string{"replace message /password=\\w+/ with 'password=***'"},
	},
	"lucene": {
		Name:        "lucene",
		Description: "Execute Lucene query within DataPrime",
		Syntax:      "lucene '<lucene_query>'",
		Examples: []string{
			"lucene 'error AND timeout'",
			"source logs | lucene 'status:500' | groupby path",
		},
		Notes: []string{"Combines Lucene search with DataPrime transformations"},
	},
	"find": {
		Name:        "find",
		Aliases:     []string{"text"},
		Description: "Free-text search within a keypath",
		Syntax:      "find '<text>' [in <field>]",
		Examples:    []string{"find 'error' in message", "text 'timeout'"},
	},
	"wildfind": {
		Name:        "wildfind",
		Aliases:     []string{"wildtext"},
		Description: "Pattern-based text searching with wildcards",
		Syntax:      "wildfind '<pattern>'",
		Examples:    []string{"wildfind 'error*timeout'"},
	},
	"enrich": {
		Name:        "enrich",
		Description: "Augment data with additional context from lookup tables",
		Syntax:      "enrich <field> using <lookup_table>",
		Examples:    []string{"enrich ip_address using geo_lookup"},
	},
	"explode": {
		Name:        "explode",
		Description: "Expand nested arrays into separate rows",
		Syntax:      "explode <array_field>",
		Examples:    []string{"explode tags", "explode items into item"},
	},
	"dedupeby": {
		Name:        "dedupeby",
		Description: "Remove duplicate entries by criteria",
		Syntax:      "dedupeby <expression>",
		Examples:    []string{"dedupeby request_id", "dedupeby user_id, action"},
	},
	"block": {
		Name:        "block",
		Description: "Partition or segment data",
		Syntax:      "block <expression>",
		Examples:    []string{"block session_id"},
	},
	"stitch": {
		Name:        "stitch",
		Description: "Combine related events",
		Syntax:      "stitch <expression>",
		Examples:    []string{"stitch transaction_id"},
	},
	"union": {
		Name:        "union",
		Description: "Merge multiple datasets",
		Syntax:      "union (<subquery>)",
		Examples:    []string{"source logs | union (source archive_logs)"},
	},
	"convert": {
		Name:        "convert",
		Description: "Transform data types",
		Syntax:      "convert <field> to <type>",
		Examples:    []string{"convert duration to number", "convert timestamp to string"},
	},
	"multigroupby": {
		Name:        "multigroupby",
		Description: "Multiple grouping levels",
		Syntax:      "multigroupby <expr1>, <expr2> aggregate <func>",
		Examples:    []string{"multigroupby region, service aggregate count()"},
	},
}

// DataPrimeCommandInfo contains metadata about a DataPrime command
type DataPrimeCommandInfo struct {
	Name        string
	Aliases     []string
	Description string
	Syntax      string
	Examples    []string
	Notes       []string
}

// ==================== OPERATORS ====================
// DataPrime supports various operators for comparisons and logic.

// ComparisonOperators lists valid comparison operators
var ComparisonOperators = map[string]string{
	"==": "Equality comparison",
	"!=": "Inequality comparison",
	">":  "Greater than",
	"<":  "Less than",
	">=": "Greater than or equal",
	"<=": "Less than or equal",
	"~":  "Free text search / contains (for string fields)",
	"!~": "Does not contain (for string fields)",
}

// LogicalOperators lists valid logical operators
var LogicalOperators = map[string]string{
	"&&": "Logical AND (NOT 'AND')",
	"||": "Logical OR (NOT 'OR')",
	"!":  "Logical NOT",
}

// NOTE: The ~~ operator is NOT supported in DataPrime!
// Use the matches() function with regex instead: field.matches(/pattern/)
// Or use contains() for simple substring matching: field.contains('text')

// ==================== FUNCTIONS ====================
// DataPrime functions transform individual values.
// Most functions support both function and method notation:
// - Function: toLowerCase(field)
// - Method: field.toLowerCase()

// AggregationFunctions lists all aggregation functions
var AggregationFunctions = []string{
	"count()",                        // Count records
	"count_if(<condition>)",          // Count records matching condition
	"sum(<field>)",                   // Sum of values
	"avg(<field>)",                   // Average of values
	"min(<field>)",                   // Minimum value
	"max(<field>)",                   // Maximum value
	"min_by(<field>, <expr>)",        // Record with minimum value
	"max_by(<field>, <expr>)",        // Record with maximum value
	"distinct_count(<field>)",        // Count unique values
	"distinct_count_if(<f>,<c>)",     // Count unique values matching condition
	"percentile(<field>, <p>)",       // Percentile value
	"stddev(<field>)",                // Standard deviation
	"variance(<field>)",              // Variance
	"sample_stddev(<field>)",         // Sample standard deviation
	"sample_variance(<field>)",       // Sample variance
	"any_value(<field>)",             // Any value from group
	"collect(<field>)",               // Collect values into array
	"approx_count_distinct(<field>)", // Approximate unique count
}

// StringFunctions lists all string manipulation functions
var StringFunctions = []string{
	"toLowerCase(<string>)",               // Convert to lowercase
	"toUpperCase(<string>)",               // Convert to uppercase
	"trim(<string>)",                      // Remove whitespace
	"concat(<s1>, <s2>, ...)",             // Concatenate strings
	"substr(<string>, <start>, <len>)",    // Substring
	"length(<string>)",                    // String length
	"contains(<string>, <substr>)",        // Check if contains substring
	"startsWith(<string>, <prefix>)",      // Check if starts with
	"endsWith(<string>, <suffix>)",        // Check if ends with
	"matches(<string>, /<regex>/)",        // Regex matching (USE THIS instead of ~~)
	"replace(<string>, /<regex>/, <rep>)", // Replace with regex
	"split(<string>, <delimiter>)",        // Split into array
	"indexOf(<string>, <substr>)",         // Find index of substring
}

// TimeFunctions lists all time-related functions
var TimeFunctions = []string{
	"now()",                                  // Current timestamp
	"parseTimestamp(<string>, <format>)",     // Parse string to timestamp
	"formatTimestamp(<timestamp>, <format>)", // Format timestamp to string
	"diffTime(<ts1>, <ts2>)",                 // Difference between timestamps
	"addTime(<timestamp>, <interval>)",       // Add interval to timestamp
	"formatInterval(<interval>, <unit>)",     // Format interval (ms, s, h)
	"parseInterval(<string>)",                // Parse interval string
}

// ConditionalFunctions lists conditional/logic functions
var ConditionalFunctions = []string{
	"if(<condition>, <then>, <else>)",                 // Conditional expression
	"case { <cond1> -> <val1>, ... }",                 // Case expression
	"case_equals { <field>, <val1> -> <res1>, ...}",   // Case by equality
	"case_contains { <field>, <sub1> -> <res1>, ...}", // Case by contains
	"coalesce(<val1>, <val2>, ...)",                   // First non-null value
}

// ArrayFunctions lists array manipulation functions
var ArrayFunctions = []string{
	"arrayAppend(<array>, <elem>)",    // Append element
	"arraySort(<array>)",              // Sort array
	"arrayLength(<array>)",            // Array length
	"arrayContains(<array>, <elem>)",  // Check if contains
	"setUnion(<arr1>, <arr2>)",        // Union of arrays
	"setIntersection(<arr1>, <arr2>)", // Intersection of arrays
}

// ==================== BEST PRACTICES ====================

// DataPrimeBestPractices contains tips for writing efficient queries
var DataPrimeBestPractices = []string{
	"Use exact matches (==) instead of contains() or matches() when possible - they're faster due to indexing",
	"Always specify a time range to limit the data scanned",
	"Use filter early in the query to reduce data before expensive operations",
	"Avoid groupby on high-cardinality fields without limits",
	"Use 'groupby true aggregate' when you need total aggregation without grouping",
	"The $d prefix is optional for data fields - 'status_code' equals '$d.status_code'",
	"Use Lucene syntax for free-text search: lucene 'error AND timeout'",
	"For regex matching, use matches() function: field.matches(/pattern/)",
	"String comparisons are case-sensitive - use toLowerCase() for case-insensitive matching",
	"One side of a join must be < 200MB - use filter and remove to reduce size",
	"groupby removes all fields not explicitly included - add fields as keys or in aggregations",
	"Aggregations ignore null values - ensure fields are populated",
	"Can only groupby scalar values - flatten arrays/objects first",
}

// ==================== COMMON MISTAKES ====================

// DataPrimeCommonMistakes maps common errors to their corrections
var DataPrimeCommonMistakes = map[string]string{
	"AND":           "Use && instead of AND for logical AND",
	"OR":            "Use || instead of OR for logical OR",
	"=":             "Use == for equality comparison (not =)",
	"~~":            "The ~~ operator is NOT supported. Use matches() for regex: field.matches(/pattern/), or contains() for substring: field.contains('text')",
	"$d.text":       "$d.text is not a standard field. Use Lucene syntax for free-text search, or check your actual log field names",
	"$l.namespace":  "Use $l.applicationname instead - K8s namespace typically maps to applicationname",
	"$l.pod":        "Pod name is usually in $d. (data) or part of applicationname/subsystemname",
	"$m.level":      "Use $m.severity for log level (VERBOSE, DEBUG, INFO, WARNING, ERROR, CRITICAL)",
	"double_quotes": "Use single quotes for string values: 'value' (not \"value\")",
}

// GetDataPrimeHelp returns comprehensive help for DataPrime query writing
func GetDataPrimeHelp() string {
	return `# DataPrime Query Language Reference

## Access Mechanisms
- $d. (or no prefix): User data/payload - e.g., $d.status_code, message
- $l.: Labels - applicationname, subsystemname, computername, etc.
- $m.: Metadata - severity, timestamp, priority

## Severity Levels
VERBOSE (0) < DEBUG (1) < INFO (2) < WARNING (3) < ERROR (4) < CRITICAL (5)

## Basic Query Structure
source logs | filter <condition> | groupby <field> aggregate <func> | limit <n>

## Common Commands
- filter: filter $m.severity >= WARNING
- groupby: groupby $l.applicationname aggregate count() as cnt
- create: create is_error from status_code >= 400
- extract: extract message into fields using regexp(e=/pattern/)
- limit: limit 100
- orderby: orderby timestamp desc

## String Operations (NO ~~ operator!)
- Exact match: field == 'value'
- Contains: field.contains('substring')
- Starts with: field.startsWith('prefix')
- Ends with: field.endsWith('suffix')
- Regex: field.matches(/pattern/)
- Case-insensitive: field.toLowerCase().contains('text')

## Free Text Search
Use Lucene syntax: lucene 'error AND timeout'
Or find command: find 'error' in message

## Aggregations
count(), sum(field), avg(field), min(field), max(field), distinct_count(field)
For total aggregation: groupby true aggregate count()

## Time Functions
now(), parseTimestamp(str, format), formatTimestamp(ts, format), diffTime(ts1, ts2)

## Key Rules
1. Use && and || for AND/OR (not AND/OR keywords)
2. Use == for equality (not =)
3. Use single quotes for strings: 'value'
4. Use matches() for regex, contains() for substring (NO ~~ operator)
5. $d prefix is optional for data fields
`
}
