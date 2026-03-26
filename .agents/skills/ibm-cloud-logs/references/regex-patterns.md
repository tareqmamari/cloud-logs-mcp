# Regex Pattern Library for Parsing Rules

## Common Log Patterns

### Timestamps

**ISO 8601 (with milliseconds)**:
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)
```
Example: `2024-03-25T10:30:45.123Z`

**ISO 8601 (without milliseconds)**:
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?)
```
Example: `2024-03-25T10:30:45Z`

**Common Log Format**:
```regex
(?P<timestamp>\d{2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4})
```
Example: `25/Mar/2024:10:30:45 +0000`

**Syslog Format**:
```regex
(?P<timestamp>[A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})
```
Example: `Mar 25 10:30:45`

### Log Levels

**Standard Levels**:
```regex
(?P<level>DEBUG|INFO|WARN|WARNING|ERROR|CRITICAL|FATAL)
```

**Case Insensitive**:
```regex
(?P<level>(?i:debug|info|warn|warning|error|critical|fatal))
```

**Bracketed**:
```regex
\[(?P<level>DEBUG|INFO|WARN|WARNING|ERROR|CRITICAL)\]
```

### Network

**IPv4 Address**:
```regex
(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})
```

**IPv6 Address**:
```regex
(?P<ipv6>(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4})
```

**Port Number**:
```regex
(?P<port>\d{1,5})
```

**URL**:
```regex
(?P<url>https?://[^\s]+)
```

### HTTP

**HTTP Method**:
```regex
(?P<method>GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)
```

**HTTP Status Code**:
```regex
(?P<status>[1-5]\d{2})
```

**User Agent**:
```regex
"(?P<user_agent>[^"]*)"
```

### Identifiers

**UUID**:
```regex
(?P<uuid>[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})
```

**Email Address**:
```regex
(?P<email>[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})
```

**Username**:
```regex
(?P<username>[a-zA-Z0-9_-]+)
```

### Metrics

**Duration (milliseconds)**:
```regex
(?P<duration_ms>\d+)ms
```

**Duration (seconds)**:
```regex
(?P<duration_s>\d+(?:\.\d+)?)s
```

**Bytes**:
```regex
(?P<bytes>\d+)
```

**Percentage**:
```regex
(?P<percentage>\d+(?:\.\d+)?)%
```

## Complete Log Format Patterns

### Nginx Access Log
```regex
(?P<client_ip>\S+) \S+ \S+ \[(?P<timestamp>[^\]]+)\] "(?P<method>\w+) (?P<path>\S+) (?P<protocol>[^"]+)" (?P<status>\d+) (?P<bytes>\d+) "(?P<referrer>[^"]*)" "(?P<user_agent>[^"]*)"
```

Example log:
```
192.168.1.100 - - [25/Mar/2024:10:30:45 +0000] "GET /api/users HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"
```

### Apache Access Log
```regex
(?P<client_ip>\S+) \S+ \S+ \[(?P<timestamp>[^\]]+)\] "(?P<request>[^"]*)" (?P<status>\d+) (?P<bytes>\S+)
```

### Application Log (Structured)
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z) \[(?P<level>\w+)\] (?P<service>[\w-]+): (?P<message>.*)
```

Example log:
```
2024-03-25T10:30:45.123Z [ERROR] payment-service: Transaction failed for user 12345
```

### Syslog Format
```regex
(?P<timestamp>[A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}) (?P<hostname>\S+) (?P<process>\S+)\[(?P<pid>\d+)\]: (?P<message>.*)
```

### Java Stack Trace
```regex
(?P<exception>[\w\.]+Exception): (?P<message>.*)\n(?P<stack_trace>(?:\s+at .+\n)+)
```

## Sensitive Data Patterns (for Masking)

### Credit Card Numbers
```regex
\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b
```

### Social Security Numbers
```regex
\b\d{3}-\d{2}-\d{4}\b
```

### API Keys (generic)
```regex
(?i)api[_-]?key["\s:=]+([a-zA-Z0-9_-]{20,})
```

### Passwords
```regex
(?i)password["\s:=]+([^\s"]+)
```

### Email Addresses (for masking)
```regex
\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b
```

## Tips for Writing Regex

### Use Non-Greedy Matching
```regex
# Greedy (matches too much)
(?P<message>.*)

# Non-greedy (stops at first match)
(?P<message>.*?)

# Specific (best)
(?P<message>[^\n]+)
```

### Make Fields Optional
```regex
# Required field
(?P<user>\w+)

# Optional field
(?P<user>\w+)?

# Optional with non-capturing group
(?:user=(?P<user>\w+))?
```

### Escape Special Characters
```
Special characters that need escaping:
. * + ? [ ] ( ) { } ^ $ | \

Example:
\[(?P<level>\w+)\]  # Brackets are escaped
```

### Use Atomic Groups for Performance
```regex
# Can cause backtracking
(?P<message>.*)

# Atomic group (no backtracking)
(?P<message>(?>[^\n]+))
```

## Testing Your Regex

1. **Use regex101.com**:
   - Select "Python" flavor
   - Paste your regex pattern
   - Add sample log messages
   - Verify all named groups capture correctly

2. **Test with Multiple Samples**:
   - Normal cases
   - Edge cases (missing fields, special characters)
   - Error cases

3. **Check Performance**:
   - Avoid nested quantifiers
   - Use atomic groups
   - Be specific rather than greedy