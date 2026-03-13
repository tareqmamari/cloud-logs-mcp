# DataPrime Functions Reference

Complete reference for all DataPrime functions available in IBM Cloud Logs.
Most functions support both function notation `func(field)` and method notation `field.func()`.

## Aggregation Functions

Used with `aggregate` and `groupby ... aggregate` commands.

| Function | Syntax | Description |
|----------|--------|-------------|
| `count()` | `count()` | Count all records |
| `count_if()` | `count_if(<condition>)` | Count records matching condition |
| `sum()` | `sum(<field>)` | Sum of numeric values |
| `avg()` | `avg(<field>)` | Average of numeric values |
| `min()` | `min(<field>)` | Minimum value |
| `max()` | `max(<field>)` | Maximum value |
| `min_by()` | `min_by(<field>, <expr>)` | Record with minimum value of expr |
| `max_by()` | `max_by(<field>, <expr>)` | Record with maximum value of expr |
| `distinct_count()` | `distinct_count(<field>)` | Count unique values |
| `distinct_count_if()` | `distinct_count_if(<field>, <condition>)` | Count unique values matching condition |
| `percentile()` | `percentile(<field>, <p>)` | Percentile value (0-100) |
| `stddev()` | `stddev(<field>)` | Standard deviation |
| `variance()` | `variance(<field>)` | Variance |
| `sample_stddev()` | `sample_stddev(<field>)` | Sample standard deviation |
| `sample_variance()` | `sample_variance(<field>)` | Sample variance |
| `any_value()` | `any_value(<field>)` | Any value from the group |
| `collect()` | `collect(<field>)` | Collect values into array |
| `approx_count_distinct()` | `approx_count_distinct(<field>)` | Approximate unique count (preferred over distinct_count for large datasets) |

### Aggregation Examples

```
# Count errors per application
source logs | filter $m.severity >= ERROR
| groupby $l.applicationname aggregate count() as errors

# Latency percentiles
source logs | groupby $d.endpoint
| aggregate percentile($d.response_time_ms, 50) as p50,
          percentile($d.response_time_ms, 95) as p95,
          percentile($d.response_time_ms, 99) as p99

# Count distinct users per service
source logs | groupby $l.applicationname
| aggregate approx_count_distinct($d.user_id) as unique_users

# Conditional counting
source logs | groupby $l.applicationname
| aggregate count() as total,
          count_if($m.severity >= ERROR) as errors

# Total aggregation (no grouping)
source logs | groupby true aggregate count() as total
```

### Important Notes
- Use `approx_count_distinct()` instead of `count_distinct()` or `distinct_count()` — only the approximate variant is supported in the frequent_search tier
- Aggregations ignore null values
- `groupby true aggregate ...` computes totals without grouping

---

## String Functions

| Function | Syntax | Description |
|----------|--------|-------------|
| `toLowerCase()` | `toLowerCase(<string>)` | Convert to lowercase |
| `toUpperCase()` | `toUpperCase(<string>)` | Convert to uppercase |
| `trim()` | `trim(<string>)` | Remove leading/trailing whitespace |
| `concat()` | `concat(<s1>, <s2>, ...)` | Concatenate strings |
| `substr()` | `substr(<string>, <start>, <len>)` | Extract substring |
| `length()` | `length(<string>)` | String length |
| `contains()` | `contains(<string>, <substr>)` | Check if contains substring |
| `startsWith()` | `startsWith(<string>, <prefix>)` | Check if starts with prefix |
| `endsWith()` | `endsWith(<string>, <suffix>)` | Check if ends with suffix |
| `matches()` | `matches(<string>, /<regex>/)` | Regex pattern matching |
| `replace()` | `replace(<string>, /<regex>/, <rep>)` | Replace with regex |
| `split()` | `split(<string>, <delimiter>)` | Split into array |
| `indexOf()` | `indexOf(<string>, <substr>)` | Find index of substring |

### String Examples

```
# Substring search (method notation)
source logs | filter $d.message:string.contains('timeout')

# Regex matching
source logs | filter $d.message:string.matches(/error.*connection/)

# Case-insensitive search
source logs | filter $d.message:string.toLowerCase().contains('error')

# Prefix matching
source logs | filter $l.applicationname.startsWith('prod-')

# Create computed field
source logs | create short_msg from substr($d.message:string, 0, 100)
```

### CRITICAL: Mixed-Type Fields Need `:string` Cast

Fields like `message`, `msg`, `log`, `error`, `text`, `body`, `content`, `payload`, `data`, `err`, `reason`, `details` often have mixed `object|string` type. You MUST add `:string` before calling string methods:

```
WRONG: $d.message.contains('error')
RIGHT: $d.message:string.contains('error')

WRONG: $d.error.matches(/timeout/)
RIGHT: $d.error:string.matches(/timeout/)
```

### CRITICAL: Use `matches()` Not `~~`

The `~~` operator is NOT supported in DataPrime. Use `matches()` for regex and `contains()` for substring:

```
WRONG: filter $d.message ~~ 'error.*timeout'
RIGHT: filter $d.message:string.matches(/error.*timeout/)
RIGHT: filter $d.message:string.contains('error')
```

---

## Time Functions

| Function | Syntax | Description |
|----------|--------|-------------|
| `now()` | `now()` | Current timestamp |
| `parseTimestamp()` | `parseTimestamp(<string>, <format>)` | Parse string to timestamp |
| `formatTimestamp()` | `formatTimestamp(<timestamp>, <format>)` | Format timestamp to string |
| `roundTime()` | `roundTime(<timestamp>, <interval>)` | Round to interval (for time bucketing) |
| `diffTime()` | `diffTime(<ts1>, <ts2>)` | Difference between timestamps |
| `addTime()` | `addTime(<timestamp>, <interval>)` | Add interval to timestamp |
| `formatInterval()` | `formatInterval(<interval>, <unit>)` | Format interval (ms, s, h) |
| `parseInterval()` | `parseInterval(<string>)` | Parse interval string |

### Time Intervals
- Seconds: `1s`, `30s`
- Minutes: `1m`, `5m`, `15m`
- Hours: `1h`, `6h`, `12h`
- Days: `1d`, `7d`

### Time Examples

```
# Time bucketing for trends (use roundTime, NOT bucket())
source logs | groupby roundTime($m.timestamp, 5m) as time_bucket
| aggregate count() as events
| orderby time_bucket

# Error timeline with 1-hour buckets
source logs | filter $m.severity >= ERROR
| groupby roundTime($m.timestamp, 1h) as hour
| aggregate count() as errors
| orderby hour

# Timestamp literals use @'' syntax
source logs | between @'2024-01-01' and @'2024-01-02'

# Calculate request duration
source logs | create duration from diffTime($d.end_time, $d.start_time)
```

### CRITICAL: Use `roundTime()` Not `bucket()`

`bucket()` is not a valid DataPrime function. Use `roundTime()` for time bucketing:

```
WRONG: groupby bucket($m.timestamp, 5m) as time_bucket
RIGHT: groupby roundTime($m.timestamp, 5m) as time_bucket

WRONG: $m.timestamp:5m
RIGHT: roundTime($m.timestamp, 5m)
```

---

## Conditional Functions

| Function | Syntax | Description |
|----------|--------|-------------|
| `if()` | `if(<condition>, <then>, <else>)` | Conditional expression |
| `case` | `case { <cond1> -> <val1>, ... }` | Multi-branch conditional |
| `case_equals` | `case_equals { <field>, <val1> -> <res1>, ... }` | Case by equality |
| `case_contains` | `case_contains { <field>, <sub1> -> <res1>, ... }` | Case by contains |
| `coalesce()` | `coalesce(<val1>, <val2>, ...)` | First non-null value |

### Conditional Examples

```
# Simple if/else
source logs | create status_group from if(status_code >= 400, 'error', 'success')

# Multi-branch case
source logs | create severity_label from case {
  $m.severity == CRITICAL -> 'CRITICAL',
  $m.severity == ERROR -> 'ERROR',
  $m.severity == WARNING -> 'WARNING',
  true -> 'OK'
}

# Coalesce for fallback values
source logs | create display_name from coalesce($d.username, $d.user_id, 'anonymous')
```

---

## Array Functions

| Function | Syntax | Description |
|----------|--------|-------------|
| `arrayAppend()` | `arrayAppend(<array>, <elem>)` | Append element to array |
| `arraySort()` | `arraySort(<array>)` | Sort array elements |
| `arrayLength()` | `arrayLength(<array>)` | Get array length |
| `arrayContains()` | `arrayContains(<array>, <elem>)` | Check if array contains element |
| `setUnion()` | `setUnion(<arr1>, <arr2>)` | Union of two arrays |
| `setIntersection()` | `setIntersection(<arr1>, <arr2>)` | Intersection of two arrays |

### Array Examples

```
# Filter by array contents
source logs | filter arrayContains($d.tags, 'production')

# Expand arrays into rows
source logs | explode $d.items into item

# Check array size
source logs | filter arrayLength($d.errors) > 0
```
