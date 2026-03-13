# Lucene Integration with DataPrime

IBM Cloud Logs supports Lucene queries both standalone and integrated within DataPrime pipelines.

## When to Use Lucene vs DataPrime

| Use Case | Recommended | Why |
|----------|-------------|-----|
| Free-text search across all fields | Lucene | Optimized for full-text search |
| Structured field filtering | DataPrime | Precise field access with `$l.`, `$m.`, `$d.` |
| Aggregations and analytics | DataPrime | `groupby`, `aggregate`, `percentile` |
| Complex regex patterns | DataPrime | `matches()` function |
| Quick keyword search | Lucene | Simple, fast syntax |
| Combined search + analytics | Both | `lucene` command in DataPrime pipeline |

## Lucene Syntax Reference

### Basic Queries

```
# Single term
error

# Phrase (exact sequence)
"connection timeout"

# Field-specific search
applicationname:payment-service
severity:5

# Wildcard
error*
status_code:50?
```

### Boolean Operators

Lucene uses `AND`, `OR`, `NOT` keywords (unlike DataPrime which uses `&&`, `||`, `!`):

```
# AND (both terms required)
error AND timeout

# OR (either term)
error OR warning

# NOT (exclude)
error NOT "health check"

# Grouping
(error OR warning) AND payment-service
```

### Range Queries

```
# Numeric range (inclusive)
severity:[4 TO 6]

# Greater than
severity:>=5

# Exclusive range
response_time_ms:{1000 TO *}
```

### Field Queries

```
# Exact match
applicationname:my-service

# Multiple values
applicationname:(service-a OR service-b)

# Exists (field has any value)
stack_trace:*

# Does not exist
NOT stack_trace:*
```

### Wildcard and Fuzzy

```
# Wildcard (* = any characters, ? = single character)
message:*timeout*
applicationname:prod-*

# Fuzzy search (Levenshtein distance)
message:timout~2
```

## Using Lucene in DataPrime

### The `lucene` Command

The `lucene` command lets you use Lucene syntax within a DataPrime pipeline:

```
# Basic lucene search in DataPrime
source logs | lucene 'error AND timeout'

# Lucene search followed by DataPrime aggregation
source logs | lucene 'status:500' | groupby $d.path aggregate count() as errors

# Combine lucene text search with DataPrime filters
source logs | lucene 'connection refused'
| filter $l.applicationname == 'api-gateway'
| orderby $m.timestamp desc | limit 50

# Lucene for text search + DataPrime for analytics
source logs | lucene '"out of memory"'
| groupby $l.applicationname, $l.subsystemname
| aggregate count() as occurrences
| orderby -occurrences | limit 20
```

### The `find` Command

DataPrime's `find` (alias: `text`) command provides simple text search:

```
# Search in a specific field
find 'error' in message

# Search across all text fields
find 'timeout'

# Combine with other DataPrime commands
source logs | find 'connection refused' in message
| filter $m.severity >= ERROR
| choose $m.timestamp, $l.applicationname, $d.message
| orderby $m.timestamp desc | limit 50
```

### The `wildfind` Command

Pattern-based text search with wildcards:

```
# Wildcard text search
wildfind 'error*timeout'

# In a pipeline
source logs | wildfind 'auth*fail*'
| groupby $l.applicationname aggregate count() as failures
```

## Lucene vs DataPrime Syntax Comparison

| Operation | Lucene | DataPrime |
|-----------|--------|-----------|
| Text search | `error timeout` | `find 'error' in message` |
| AND | `error AND timeout` | `&& ` |
| OR | `error OR warning` | `\|\|` |
| NOT | `NOT error` | `!` |
| Field match | `applicationname:myapp` | `$l.applicationname == 'myapp'` |
| Wildcard | `message:*timeout*` | `$d.message:string.contains('timeout')` |
| Regex | `message:/err[0-9]+/` | `$d.message:string.matches(/err[0-9]+/)` |
| Range | `severity:[4 TO 6]` | `$m.severity >= WARNING && $m.severity <= CRITICAL` |
| Exists | `field:*` | `$d.field != null` |

## Best Practices

1. **Use Lucene for initial discovery** — when you don't know exact field names, Lucene searches across all fields
2. **Switch to DataPrime for analysis** — once you know what you're looking for, DataPrime's aggregation is more powerful
3. **Combine both** — use `lucene` command for text matching, then pipe to DataPrime for grouping and counting
4. **Quote phrases** — always use double quotes for multi-word Lucene phrases: `"connection timeout"`
5. **Performance** — Lucene queries are optimized for the search index; adding DataPrime filters after Lucene is efficient

## Common Patterns

### Find errors mentioning a keyword, then analyze
```
source logs | lucene 'database AND (timeout OR "connection refused")'
| filter $m.severity >= ERROR
| groupby $l.applicationname aggregate count() as errors
| orderby -errors | limit 10
```

### Full-text search with time bucketing
```
source logs | lucene '"payment failed"'
| groupby roundTime($m.timestamp, 5m) as bucket
| aggregate count() as occurrences
| orderby bucket
```

### Search across services for a trace ID
```
source logs | lucene 'trace_id:abc123def456'
| choose $m.timestamp, $l.applicationname, $l.subsystemname, $d.message
| orderby $m.timestamp
```
