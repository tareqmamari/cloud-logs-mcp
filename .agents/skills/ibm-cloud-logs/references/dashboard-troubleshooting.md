# Dashboard Troubleshooting Reference

## Quick Diagnosis Flowchart

```
Dashboard Issue
    ↓
Is data showing at all?
    ↓ NO
    ├─→ Check TCO Policy (logs in High/Medium?) → Fix policy routing
    ├─→ Test query in Explore Logs → Fix query syntax
    ├─→ Check time range → Expand time range
    └─→ Verify data source → Configure data source
    ↓ YES (Partial data)
    ├─→ Check which data is missing → Identify pattern
    ├─→ Review filters → Adjust filters
    └─→ Check TCO for missing sources → Fix policy
```

## Common Issues and Solutions

### Issue 1: Dashboard Shows No Data (80% of cases)

**Root Cause**: TCO policy routes logs to Low priority (Store & search)

**Quick Fix**:
1. Go to Explore Logs
2. Search for logs: `source logs | filter $l.applicationname == 'your-app'`
3. Check pipeline indicator
4. If "Store & search" → Modify TCO policy to route to High or Medium

**Example Policy Fix**:
```
Before:
  Application: web-app
  Pipeline: Store & search (Low)

After:
  Application: web-app
  Pipeline: Analyze & alert (Medium)
```

### Issue 2: Incorrect Query Syntax (15% of cases)

**Common Mistakes**:
```
Wrong: filter severity == "ERROR"
Right: filter $m.severity == ERROR

Wrong: filter applicationname == web-app
Right: filter $l.applicationname == 'web-app'

Wrong: groupby timestamp
Right: groupby $m.timestamp
```

### Issue 3: Time Range Too Narrow (3% of cases)

**Solution**: Expand time range
- From "Last 5 minutes" → "Last 1 hour"
- From "Last 1 hour" → "Last 24 hours"

### Issue 4: Partial Data Showing

**Root Cause**: Mixed TCO policies (some logs in High/Medium, others in Low)

**Solution**: Create policies to route all dashboard-critical logs to High/Medium

## Diagnostic Commands

### Check Log Pipeline
```
source logs
| filter $l.applicationname == 'your-app'
| limit 1
```
Look at the pipeline indicator in results.

### Verify Field Existence
```
source logs
| filter $l.applicationname == 'your-app'
| limit 100
| create has_field = $d.your_field != null
| groupby has_field aggregate count()
```

### Check Data Volume
```
source logs
| filter $l.applicationname == 'your-app'
| groupby $m.timestamp aggregate count() as log_count
```

## Prevention Best Practices

1. **Always test queries in Explore Logs first**
2. **Verify logs are in High/Medium priority before creating dashboards**
3. **Use specific filters to reduce query complexity**
4. **Document dashboard purpose and data sources**
5. **Monitor dashboard performance regularly**