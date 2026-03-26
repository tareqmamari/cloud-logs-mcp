# Parsing Rules Troubleshooting Guide

## Quick Diagnosis Flowchart

```
Parsing Issue
    ↓
Are fields being extracted?
    ↓ NO
    ├─→ Check rule order (specific before general) → Reorder rules
    ├─→ Test regex pattern → Fix pattern
    ├─→ Verify source field → Correct field name
    └─→ Check rule is enabled → Enable rule
    ↓ YES (Partial extraction)
    ├─→ Check regex for all log formats → Make pattern flexible
    ├─→ Review optional groups → Add (?: ...)? for optional fields
    └─→ Test with edge cases → Handle missing fields
```

## Issue 1: Fields Not Being Extracted (70% of cases)

### Root Cause: Wrong Rule Order

**Problem**: General rule above specific rule catches all logs first

**Diagnosis**:
```
1. List all parsing rules in order
2. Identify which rule matches first
3. Check if a general pattern (.*) is above specific patterns
```

**Solution**:
```
Reorder rules - specific patterns BEFORE general patterns

Wrong Order:
  Rule 1 (Order: 1): Pattern: .*
  Rule 2 (Order: 2): Pattern: ERROR.*

Right Order:
  Rule 1 (Order: 1): Pattern: ERROR.*
  Rule 2 (Order: 2): Pattern: .*
```

**How to Fix**:
1. Go to Parsing Rules configuration
2. Edit rule order/priority
3. Move specific rules to lower order numbers (higher priority)
4. Save and test with sample logs

## Issue 2: Regex Pattern Not Matching

### Root Cause: Incorrect regex syntax or pattern

**Diagnosis**:
```
1. Copy sample log message
2. Test regex on regex101.com
3. Check if pattern matches
4. Verify named capture groups
```

**Common Mistakes**:

**Mistake 1: Not Escaping Special Characters**
```
Wrong: [ERROR] (?P<message>.*)
Right: \[ERROR\] (?P<message>.*)

Characters to escape: . * + ? [ ] ( ) { } ^ $ | \
```

**Mistake 2: Greedy Matching**
```
Wrong: (?P<message>.*) ERROR
Right: (?P<message>.*?) ERROR

Use .*? for non-greedy matching
```

**Mistake 3: Missing Named Groups**
```
Wrong: (\d{4}-\d{2}-\d{2})
Right: (?P<date>\d{4}-\d{2}-\d{2})

Always use (?P<name>...) for field extraction
```

**Mistake 4: Case Sensitivity**
```
Wrong: (?P<level>ERROR)
Right: (?P<level>(?i:error))

Use (?i:...) for case-insensitive matching
```

## Issue 3: Some Logs Parsed, Others Not

### Root Cause: Log format variations

**Diagnosis**:
```
1. Collect sample logs that work
2. Collect sample logs that don't work
3. Compare formats
4. Identify differences
```

**Solution**:
```
Make regex pattern flexible with optional groups

Example - Handle optional fields:
(?P<timestamp>\S+) (?P<level>\w+)? (?P<message>.*)

The ? makes level optional
```

**Multiple Format Handling**:
```
Option 1: Create separate rules for each format
  Rule 1: Format A pattern
  Rule 2: Format B pattern

Option 2: Use alternation in regex
  (?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}|\d{2}/[A-Za-z]{3}/\d{4})
```

## Issue 4: Wrong Source Field

### Root Cause: Parsing incorrect field

**Diagnosis**:
```
1. Check log structure in Explore Logs
2. Identify which field contains the log message
3. Common fields:
   - text: Main log message
   - text.log: Nested log (Kubernetes)
   - json.message: JSON field
```

**Solution**:
```
Update source field in parsing rule:

For standard logs:
  Source Field: text

For Kubernetes logs:
  Source Field: text.log

For JSON logs:
  Source Field: json.message
```

## Issue 5: Fields Overwriting Each Other

### Root Cause: Multiple rules extracting to same field names

**Diagnosis**:
```
1. Check all parsing rules
2. Identify duplicate field names
3. Verify rule order
```

**Solution**:
```
Option 1: Use unique field names
  Rule 1: Extract to error_timestamp
  Rule 2: Extract to info_timestamp

Option 2: Ensure only one rule matches per log
  Use specific patterns so rules don't overlap

Option 3: Adjust rule order
  Put most specific rule first
```

## Issue 6: Parsing Performance is Slow

### Root Cause: Complex regex with backtracking

**Diagnosis**:
```
1. Review regex complexity
2. Check for nested quantifiers
3. Look for catastrophic backtracking patterns
```

**Solution**:
```
1. Simplify regex patterns
   Wrong: (.*)*
   Right: [^\n]+

2. Use atomic groups
   Wrong: (?P<message>.*)
   Right: (?P<message>(?>[^\n]+))

3. Be specific instead of greedy
   Wrong: .*
   Right: [a-zA-Z0-9_-]+

4. Reduce number of rules
   Combine similar patterns where possible
```

## Issue 7: Rule Not Enabled

### Root Cause: Rule or rule group is disabled

**Diagnosis**:
```
1. Check rule status
2. Check rule group status
3. Verify no maintenance windows
```

**Solution**:
```
1. Go to Parsing Rules
2. Find the rule
3. Enable the rule
4. Enable the rule group
5. Save changes
```

## Diagnostic Commands

### Test Parsing in Explore Logs
```
1. Go to Explore Logs
2. Search for logs that should be parsed
3. Check if fields are extracted
4. Look for parsed fields in log details
```

### Verify Field Extraction
```
source logs
| filter $l.applicationname == 'your-app'
| limit 10
| create has_parsed_field = $d.your_field != null
```

### Check Parsing Success Rate
```
source logs
| filter $l.applicationname == 'your-app'
| create is_parsed = $d.parsed_field != null
| groupby is_parsed aggregate count()
```

## Prevention Best Practices

### 1. Test Before Deploy
```
✅ Test regex on regex101.com
✅ Test with multiple sample logs
✅ Test with edge cases
✅ Verify all named groups work
✅ Check performance with complex patterns
```

### 2. Document Rules
```
✅ Add clear rule names
✅ Document what each rule does
✅ Include sample logs in description
✅ Note any special handling
```

### 3. Monitor Parsing
```
✅ Check parsing success rate
✅ Monitor for unparsed logs
✅ Review parsing errors
✅ Update rules as log formats change
```

### 4. Rule Ordering Strategy
```
Order 1-10: Blocking rules
Order 11-20: Data masking
Order 21-30: Specific parsing
Order 31-40: General parsing
Order 41+: Fallback rules
```

### 5. Use Version Control
```
✅ Document rule changes
✅ Test in non-production first
✅ Keep backup of working rules
✅ Roll back if issues occur
```

## Common Error Messages

### "No fields extracted"
- Check rule order
- Verify regex pattern
- Test with sample logs
- Check source field

### "Invalid regex pattern"
- Check for syntax errors
- Verify named groups
- Test on regex101.com
- Check for unescaped special chars

### "Rule not matching"
- Verify log format
- Check for format variations
- Make pattern more flexible
- Add optional groups

### "Performance degraded"
- Simplify regex patterns
- Reduce number of rules
- Use atomic groups
- Avoid nested quantifiers