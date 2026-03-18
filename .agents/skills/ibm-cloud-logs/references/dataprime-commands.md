# DataPrime Command Catalog

A comprehensive reference for all DataPrime query language commands, organized by category.

---

## Table of Contents

- [Source Commands](#source-commands)
- [Data Manipulation Commands](#data-manipulation-commands)
- [Ordering and Limiting](#ordering-and-limiting)
- [Field Selection](#field-selection)
- [Text Processing](#text-processing)
- [Search Commands](#search-commands)
- [Data Enrichment and Transformation](#data-enrichment-and-transformation)
- [Advanced Commands](#advanced-commands)

---

## Source Commands

These commands define the data source and time scope of a query.

### `source`

| Property | Value |
|----------|-------|
| Description | Primary data access mechanism |
| Syntax | `source <source_name>` |

**Examples:**
```dataprime
source logs
source spans
```

---

### `around`

| Property | Value |
|----------|-------|
| Description | Time-based query around a specific point |
| Syntax | `around <timestamp> <duration>` |

**Examples:**
```dataprime
source logs | around @'2024-01-01T12:00:00' 5m
```

---

### `between`

| Property | Value |
|----------|-------|
| Description | Query data within a defined time range |
| Syntax | `between <start_timestamp> and <end_timestamp>` |

**Examples:**
```dataprime
source logs | between @'2024-01-01' and @'2024-01-02'
```

---

### `last`

| Property | Value |
|----------|-------|
| Description | Query data from the most recent period |
| Syntax | `last <duration>` |

**Examples:**
```dataprime
source logs | last 1h
source logs | last 30m
```

---

### `timeshifted`

| Property | Value |
|----------|-------|
| Description | Query data with temporal offset for historical comparison |
| Syntax | `timeshifted <duration>` |

**Examples:**
```dataprime
source logs | timeshifted 1d
```

---

## Data Manipulation Commands

Commands that transform, filter, and aggregate data as it flows through the pipeline.

### `filter`

**Aliases:** `f`, `where`

| Property | Value |
|----------|-------|
| Description | Remove events that do not match a condition |
| Syntax | `filter <condition>` |

**Examples:**
```dataprime
filter $m.severity == ERROR
filter $l.applicationname == 'myapp'
filter status_code >= 400 && status_code < 500
```

---

### `aggregate`

**Aliases:** `agg`

| Property | Value |
|----------|-------|
| Description | Perform calculations across the entire dataset |
| Syntax | `aggregate <aggregation_expression> [as <alias>]` |

**Examples:**
```dataprime
aggregate count() as total
aggregate avg(duration) as avg_duration, max(duration) as max_duration
```

---

### `groupby`

| Property | Value |
|----------|-------|
| Description | Aggregate documents that share common values |
| Syntax | `groupby <expression> [as <alias>] [aggregate <agg_func>]` |

**Examples:**
```dataprime
groupby $l.applicationname aggregate count() as cnt
groupby status_code aggregate avg(duration) as avg_time
groupby true aggregate count()
```

**Notes:**
- Use `true` as the groupby expression to anchor a total aggregation across all records without a grouping key.

---

### `create`

**Aliases:** `add`, `c`, `a`

| Property | Value |
|----------|-------|
| Description | Define a new field based on an expression |
| Syntax | `create <keypath> from <expression>` |

**Examples:**
```dataprime
create full_name from concat(first_name, ' ', last_name)
create is_error from status_code >= 400
create duration_ms from duration * 1000
```

---

### `extract`

| Property | Value |
|----------|-------|
| Description | Parse unstructured data into structured fields |
| Syntax | `extract <field> into <target> using <extractor>` |

**Examples:**
```dataprime
extract message into fields using regexp(e=/(?<user>\w+) did (?<action>\w+)/)
extract json_string into parsed using jsonobject()
extract kvpairs into fields using kv()
```

---

### `join`

| Property | Value |
|----------|-------|
| Description | Merge results of current query with a second query |
| Syntax | `join [inner\|full] (<subquery>) on left=><field> == right=><field> into <keypath>` |

**Examples:**
```dataprime
source users | join (source logins | countby id) on left=>id == right=>id into logins
```

**Notes:**
- Join condition only supports keypath equality (`==`).
- One side of the join must be small (under 200MB).
- Use `filter` and `remove` to reduce query size before joining.

---

## Ordering and Limiting

Commands that control the number and order of records returned.

### `limit`

**Aliases:** `l`

| Property | Value |
|----------|-------|
| Description | Restrict the number of returned records |
| Syntax | `limit <number>` |

**Examples:**
```dataprime
limit 100
orderby timestamp desc | limit 10
```

**Notes:**
- Does not guarantee order unless used after `orderby`.

---

### `orderby`

**Aliases:** `sortby`

| Property | Value |
|----------|-------|
| Description | Sort results by specified expression |
| Syntax | `orderby <expression> [asc\|desc]` |

**Examples:**
```dataprime
orderby timestamp desc
orderby count asc
```

---

### `top`

| Property | Value |
|----------|-------|
| Description | Return highest-ranked records (combines orderby + limit) |
| Syntax | `top <limit> <expression> by <order_expression>` |

**Examples:**
```dataprime
top 5 time_taken_ms by action, user
```

---

### `bottom`

| Property | Value |
|----------|-------|
| Description | Return lowest-ranked records |
| Syntax | `bottom <limit> <expression> by <order_expression>` |

**Examples:**
```dataprime
bottom 10 response_time by endpoint
```

---

### `distinct`

| Property | Value |
|----------|-------|
| Description | Return unique values |
| Syntax | `distinct <expression>` |

**Examples:**
```dataprime
distinct user_id
distinct $l.applicationname
```

---

### `count`

| Property | Value |
|----------|-------|
| Description | Calculate record totals |
| Syntax | `count` |

**Examples:**
```dataprime
source logs | count
```

---

### `countby`

| Property | Value |
|----------|-------|
| Description | Grouped counting operations |
| Syntax | `countby <expression>` |

**Examples:**
```dataprime
countby status_code
countby $l.applicationname
```

---

## Field Selection

Commands that control which fields appear in the output.

### `choose`

| Property | Value |
|----------|-------|
| Description | Select specific fields to include in output |
| Syntax | `choose <field1>, <field2>, ...` |

**Examples:**
```dataprime
choose timestamp, message, severity
```

---

### `remove`

| Property | Value |
|----------|-------|
| Description | Remove fields from output |
| Syntax | `remove <field1>, <field2>, ...` |

**Examples:**
```dataprime
remove sensitive_data, internal_id
```

---

### `move`

| Property | Value |
|----------|-------|
| Description | Reposition or rename fields |
| Syntax | `move <source> to <destination>` |

**Examples:**
```dataprime
move old_name to new_name
```

---

## Text Processing

Commands for masking, redacting, and substituting text content in fields.

### `redact`

| Property | Value |
|----------|-------|
| Description | Mask sensitive information |
| Syntax | `redact <field> [using <pattern>]` |

**Examples:**
```dataprime
redact credit_card
redact email using /\w+@/
```

---

### `replace`

| Property | Value |
|----------|-------|
| Description | Substitution operations on strings |
| Syntax | `replace <field> <pattern> with <replacement>` |

**Examples:**
```dataprime
replace message /password=\w+/ with 'password=***'
```

---

## Search Commands

Commands for searching log content using text, patterns, or Lucene syntax.

### `lucene`

| Property | Value |
|----------|-------|
| Description | Execute Lucene query within DataPrime |
| Syntax | `lucene '<lucene_query>'` |

**Examples:**
```dataprime
lucene 'error AND timeout'
source logs | lucene 'status:500' | groupby path
```

**Notes:**
- Combines Lucene search with DataPrime pipeline transformations.
- Useful for leveraging existing Lucene query knowledge within a DataPrime pipeline.

---

### `find`

**Aliases:** `text`

| Property | Value |
|----------|-------|
| Description | Free-text search within a keypath |
| Syntax | `find '<text>' [in <field>]` |

**Examples:**
```dataprime
find 'error' in message
text 'timeout'
```

---

### `wildfind`

**Aliases:** `wildtext`

| Property | Value |
|----------|-------|
| Description | Pattern-based text searching with wildcards |
| Syntax | `wildfind '<pattern>'` |

**Examples:**
```dataprime
wildfind 'error*timeout'
```

---

## Data Enrichment and Transformation

Commands for enriching records with external data and transforming data structures.

### `enrich`

| Property | Value |
|----------|-------|
| Description | Augment data with additional context from lookup tables |
| Syntax | `enrich <field> using <lookup_table>` |

**Examples:**
```dataprime
enrich ip_address using geo_lookup
```

---

### `explode`

| Property | Value |
|----------|-------|
| Description | Expand nested arrays into separate rows |
| Syntax | `explode <array_field>` |

**Examples:**
```dataprime
explode tags
explode items into item
```

---

### `convert`

| Property | Value |
|----------|-------|
| Description | Transform data types |
| Syntax | `convert <field> to <type>` |

**Examples:**
```dataprime
convert duration to number
convert timestamp to string
```

---

## Advanced Commands

Commands for deduplication, session analysis, multi-level grouping, and dataset merging.

### `dedupeby`

| Property | Value |
|----------|-------|
| Description | Remove duplicate entries by criteria |
| Syntax | `dedupeby <expression>` |

**Examples:**
```dataprime
dedupeby request_id
dedupeby user_id, action
```

---

### `block`

| Property | Value |
|----------|-------|
| Description | Partition or segment data |
| Syntax | `block <expression>` |

**Examples:**
```dataprime
block session_id
```

---

### `stitch`

| Property | Value |
|----------|-------|
| Description | Combine related events |
| Syntax | `stitch <expression>` |

**Examples:**
```dataprime
stitch transaction_id
```

---

### `union`

| Property | Value |
|----------|-------|
| Description | Merge multiple datasets |
| Syntax | `union (<subquery>)` |

**Examples:**
```dataprime
source logs | union (source archive_logs)
```

---

### `multigroupby`

| Property | Value |
|----------|-------|
| Description | Multiple grouping levels |
| Syntax | `multigroupby <expr1>, <expr2> aggregate <func>` |

**Examples:**
```dataprime
multigroupby region, service aggregate count()
```

---

## Quick Reference Table

| Command | Aliases | Category | Purpose |
|---------|---------|----------|---------|
| `source` | — | Source | Primary data access |
| `around` | — | Source | Time-scoped query around a point |
| `between` | — | Source | Query within a time range |
| `last` | — | Source | Query the most recent period |
| `timeshifted` | — | Source | Query with historical offset |
| `filter` | `f`, `where` | Manipulation | Remove non-matching events |
| `aggregate` | `agg` | Manipulation | Calculate across entire dataset |
| `groupby` | — | Manipulation | Group and aggregate by value |
| `create` | `add`, `c`, `a` | Manipulation | Add a derived field |
| `extract` | — | Manipulation | Parse unstructured data |
| `join` | — | Manipulation | Merge with a second query |
| `limit` | `l` | Ordering/Limiting | Cap number of results |
| `orderby` | `sortby` | Ordering/Limiting | Sort results |
| `top` | — | Ordering/Limiting | Highest-ranked N records |
| `bottom` | — | Ordering/Limiting | Lowest-ranked N records |
| `distinct` | — | Ordering/Limiting | Return unique values |
| `count` | — | Ordering/Limiting | Total record count |
| `countby` | — | Ordering/Limiting | Grouped record count |
| `choose` | — | Field Selection | Include specific fields |
| `remove` | — | Field Selection | Exclude fields from output |
| `move` | — | Field Selection | Rename or reposition fields |
| `redact` | — | Text Processing | Mask sensitive data |
| `replace` | — | Text Processing | Substitute string content |
| `lucene` | — | Search | Lucene query within pipeline |
| `find` | `text` | Search | Free-text field search |
| `wildfind` | `wildtext` | Search | Wildcard text search |
| `enrich` | — | Enrichment | Add data from lookup tables |
| `explode` | — | Enrichment | Expand arrays into rows |
| `convert` | — | Enrichment | Transform data types |
| `dedupeby` | — | Advanced | Remove duplicates by criteria |
| `block` | — | Advanced | Partition data into segments |
| `stitch` | — | Advanced | Combine related events |
| `union` | — | Advanced | Merge datasets |
| `multigroupby` | — | Advanced | Multi-level grouping |
