# Alert Troubleshooting Guide

## Quick Diagnosis Table

| Symptom | Most Likely Cause | Quick Fix | Detailed Section |
|---------|-------------------|-----------|------------------|
| Alert never triggers | Logs in Low priority (TCO) | Route to High/Medium | [TCO Issues](#tco-policy-issues) |
| Alert triggers too often | Threshold too low | Increase threshold | [False Positives](#false-positives) |
| No notifications received | Channel misconfigured | Verify channel setup | [Notification Issues](#notification-issues) |
| Query returns no results | Incorrect field names | Validate query | [Query Issues](#query-issues) |
| Alert triggers incorrectly | Wrong condition | Review alert logic | [Configuration Issues](#configuration-issues) |
| Alert disabled unexpectedly | Maintenance window | Check schedule | [Alert Status Issues](#alert-status-issues) |

---

## TCO Policy Issues

### Problem: Alert Never Triggers (Most Common Issue)

#### Symptoms
- Alert configured correctly
- Logs exist that match conditions
- Alert never fires
- No errors in alert configuration

#### Root Cause
**Logs are routed to Low priority (Store & search) by TCO policy**

⚠️ **CRITICAL**: Alerts ONLY work with High and Medium priority logs. This is the #1 cause of alert failures.

#### Diagnosis Steps

**Step 1: Check Log Priority**
```
1. Go to IBM Cloud Logs UI
2. Navigate to Explore Logs
3. Run your alert query
4. Check which pipeline the results are in:
   
   Look for pipeline indicator:
   - "Priority insights" (High) → ✅ Alerts work
   - "Analyze & alert" (Medium) → ✅ Alerts work
   - "Store & search" (Low) → ❌ Alerts DON'T work
```

**Step 2: Review TCO Policies**
```
1. Go to Data Flow → TCO Policies
2. Find policies matching your logs
3. Check pipeline assignment
4. Identify which policy is routing your logs
```

**Step 3: Verify Log Routing**
```
Test with sample log:
- Application: payment-service
- Subsystem: transaction
- Severity: ERROR

Match against policies in priority order:
Policy 1: App="payment-service", Sev="CRITICAL" → No match
Policy 2: App="*", Sev="*", Pipeline="Low" → MATCH! (Problem found)
```

#### Solutions

**Solution 1: Modify Existing TCO Policy**
```yaml
Before:
  Application: payment-service
  Subsystem: *
  Severity: *
  Pipeline: Store & search (Low)

After:
  Application: payment-service
  Subsystem: *
  Severity: ERROR, CRITICAL
  Pipeline: Priority insights (High)
```

**Solution 2: Create Higher Priority Policy**
```yaml
New Policy (Priority 1):
  Application: payment-service
  Subsystem: transaction
  Severity: ERROR, CRITICAL
  Pipeline: Priority insights (High)
  Reason: Critical alerts needed

Existing Policy (Priority 2):
  Application: payment-service
  Subsystem: *
  Severity: DEBUG, VERBOSE
  Pipeline: Store & search (Low)
  Reason: Archive only
```

**Solution 3: Adjust Policy Order**
```yaml
Wrong Order:
  Policy 1: App="*", Sev="*" → Low (catches everything first)
  Policy 2: App="payment", Sev="ERROR" → High (never reached)

Correct Order:
  Policy 1: App="payment", Sev="ERROR" → High (specific first)
  Policy 2: App="*", Sev="*" → Low (catch-all last)
```

#### Validation
```
After fixing TCO policy:
1. Wait 5-10 minutes for changes to propagate
2. Generate test logs matching alert conditions
3. Verify logs appear in High or Medium priority
4. Confirm alert triggers
5. Check notification received
```

---

## Query Issues

### Problem: Query Returns No Results

#### Symptoms
- Alert query doesn't match any logs
- Query works in Explore Logs but not in alert
- Field names seem correct

#### Important: Alert Severity vs Log Severity Filter

⚠️ **CRITICAL DISTINCTION**:
- **Alert Severity** (`"severity": "error"` in alert config): Classification of the alert itself (info/warning/error/critical) - NOT a filter on logs
- **Log Severity Filter** (`filters.severities: ["error"]`): Actual filter that restricts which log severities trigger the alert

**Example Alert Configuration**:
```json
{
  "name": "my-alert",
  "severity": "error",           ← Alert's own severity (classification)
  "filters": {
    "severities": ["error", "critical"]  ← Filter on log severities (optional)
  }
}
```

If `filters.severities` is empty `[]`, the alert matches **ALL log severities** (debug, verbose, info, warning, error, critical).

#### Common Causes

**Cause 1: Incorrect Field Names**
```
Wrong: filter severity == "ERROR"
Right: filter $m.severity == ERROR

Wrong: filter app == "payment"
Right: filter $l.applicationname == "payment"

Wrong: filter error_msg != null
Right: filter error_message != null
```

**Cause 2: Case Sensitivity**
```
Wrong: filter $m.severity == "error"
Right: filter $m.severity == ERROR

Wrong: filter $l.applicationname == "Payment-Service"
Right: filter $l.applicationname == "payment-service"
```

**Cause 3: Missing Field Prefix**
```
Wrong: filter severity == ERROR
Right: filter $m.severity == ERROR

Wrong: filter applicationname == "web-app"
Right: filter $l.applicationname == "web-app"
```

#### Diagnosis Steps

**Step 1: Test Query in Explore Logs**
```
1. Copy alert query
2. Go to Explore Logs
3. Paste and run query
4. Check if results appear
5. If no results → Query issue
6. If results appear → Check TCO policy
```

**Step 2: Inspect Sample Log**
```
1. Find a log that should match
2. View log details
3. Note exact field names
4. Check field values
5. Verify data types
```

**Step 3: Simplify Query**
```
Start simple and add filters:

Step 1: source logs
Step 2: source logs | filter $l.applicationname == "payment"
Step 3: source logs | filter $l.applicationname == "payment" && $m.severity >= ERROR
```

#### Solutions

**Solution 1: Fix Field Names**
```yaml
Before:
  source logs
  | filter severity == ERROR
  | filter app == "payment"

After:
  source logs
  | filter $m.severity >= ERROR
  | filter $l.applicationname == "payment-service"
```

**Solution 2: Handle Missing Fields**
```yaml
Add null checks:
  source logs
  | filter error_code != null
  | filter error_code >= 500
```

**Solution 3: Use String Functions**
```yaml
If exact match fails:
  source logs
  | filter $l.applicationname:string.startsWith('payment')
  | filter $d.message:string.contains('error')
```

---

## False Positives

### Problem: Alert Triggers Too Frequently

#### Symptoms
- Alert fires constantly
- Too many notifications
- Alert fatigue
- Most alerts are not actionable

#### Common Causes

**Cause 1: Threshold Too Low**
```
Problem: Alert on > 1 error
Reality: Normal to have 5-10 errors per hour
Solution: Increase threshold to > 20
```

**Cause 2: Time Window Too Short**
```
Problem: 1-minute window catches transient spikes
Reality: Brief spikes are normal
Solution: Extend to 5 or 10 minutes
```

**Cause 3: Filters Too Broad**
```
Problem: Alerting on all errors
Reality: Some errors are expected
Solution: Add more specific filters
```

#### Solutions

**Solution 1: Adjust Threshold**
```yaml
Before:
  Condition: More than 1
  Time Window: 5 minutes

After:
  Condition: More than 20
  Time Window: 5 minutes
```

**Solution 2: Extend Time Window**
```yaml
Before:
  Condition: More than 10
  Time Window: 1 minute

After:
  Condition: More than 10
  Time Window: 10 minutes
```

**Solution 3: Add Specific Filters**
```yaml
Before:
  source logs
  | filter $m.severity >= ERROR

After:
  source logs
  | filter $m.severity >= ERROR
  | filter $l.applicationname == "payment-service"
  | filter $d.error_code >= 500
  | filter $d.error_code < 600
```

**Solution 4: Use Ratio Alert Instead**
```yaml
Instead of counting errors, monitor error rate:

Type: Ratio Alert
Query 1 (Errors):
  source logs | filter $m.severity >= ERROR | count
Query 2 (Total):
  source logs | count
Condition: Ratio > 0.05 (5%)
```

**Solution 5: Implement Alert Suppression**
```yaml
Add minimum time between alerts:
  Suppress for: 15 minutes
  
Or group similar alerts:
  Group by: $l.applicationname
  Group window: 10 minutes
```

---

## Notification Issues

### Problem: Notifications Not Received

#### Symptoms
- Alert triggers (visible in UI)
- No email/Slack/PagerDuty notification
- Notification channel configured

#### Diagnosis by Channel

### Email Notifications

**Check 1: Verify Email Address**
```
1. Go to Alert configuration
2. Check notification settings
3. Verify email addresses are correct
4. Check for typos
```

**Check 2: Check Spam/Junk Folder**
```
1. Search email for "IBM Cloud Logs"
2. Check spam/junk folder
3. Add sender to safe list
4. Check email filters
```

**Check 3: Verify Email Service**
```
1. Test with different email address
2. Check email service status
3. Verify no email blocks
```

### Slack Notifications

**Check 1: Verify Webhook URL**
```
1. Go to Slack workspace settings
2. Check incoming webhooks
3. Verify webhook is active
4. Test webhook with curl:
   curl -X POST -H 'Content-type: application/json' \
   --data '{"text":"Test"}' \
   YOUR_WEBHOOK_URL
```

**Check 2: Check Channel Permissions**
```
1. Verify bot has access to channel
2. Check channel is not archived
3. Verify channel name is correct
4. Check workspace permissions
```

**Check 3: Verify Integration**
```
1. Re-create webhook if needed
2. Update webhook URL in alert
3. Test with simple message
```

### PagerDuty Notifications

**Check 1: Verify Integration Key**
```
1. Go to PagerDuty service
2. Check integration key
3. Verify key is active
4. Update key in alert if changed
```

**Check 2: Check Service Status**
```
1. Verify PagerDuty service is active
2. Check escalation policy
3. Verify on-call schedule
4. Check notification rules
```

**Check 3: Test Integration**
```
1. Send test event to PagerDuty
2. Verify incident created
3. Check notification delivery
```

### Webhook Notifications

**Check 1: Verify Endpoint**
```
1. Check webhook URL is accessible
2. Verify endpoint accepts POST requests
3. Check for authentication requirements
4. Test with curl:
   curl -X POST YOUR_ENDPOINT \
   -H "Content-Type: application/json" \
   -d '{"test": "data"}'
```

**Check 2: Check Response**
```
1. Verify endpoint returns 200 OK
2. Check for error responses
3. Review endpoint logs
4. Verify payload format
```

---

## Configuration Issues

### Problem: Alert Triggers Incorrectly

#### Symptoms
- Alert fires when it shouldn't
- Alert doesn't fire when it should
- Unexpected alert behavior

#### Common Causes

**Cause 1: Wrong Condition Operator**
```
Problem: Using "More than" when should use "Less than"
Example: Alert on low log volume
Wrong: More than 10
Right: Less than 10
```

**Cause 2: Incorrect Time Window**
```
Problem: Time window doesn't match intent
Example: Want to catch sustained issues
Wrong: 1 minute window (catches transients)
Right: 15 minute window (catches sustained)
```

**Cause 3: Group By Issues**
```
Problem: Grouping creates too many alerts
Example: Group by user_id creates thousands of alerts
Solution: Group by service or region instead
```

#### Solutions

**Solution 1: Review Alert Logic**
```yaml
Verify:
- Condition operator is correct
- Threshold makes sense
- Time window is appropriate
- Group by is necessary
```

**Solution 2: Test with Historical Data**
```
1. Run query over past 24 hours
2. Check when it would have triggered
3. Verify triggers are appropriate
4. Adjust configuration
```

**Solution 3: Use Alert Preview**
```
1. Configure alert
2. Use preview feature
3. Check historical triggers
4. Adjust before saving
```

---

## Alert Status Issues

### Problem: Alert Disabled Unexpectedly

#### Symptoms
- Alert was working
- Alert stopped triggering
- Alert shows as disabled

#### Common Causes

**Cause 1: Maintenance Window**
```
Check:
1. Go to Alert configuration
2. Check maintenance windows
3. Verify current time not in window
4. Adjust or remove window
```

**Cause 2: Manual Disable**
```
Check:
1. Review alert history
2. Check who disabled alert
3. Verify reason for disable
4. Re-enable if appropriate
```

**Cause 3: System Disable**
```
Reasons:
- Too many failures
- Invalid configuration
- Quota exceeded

Solution:
1. Check alert logs
2. Fix underlying issue
3. Re-enable alert
```

---

## Performance Issues

### Problem: Alert Query Slow

#### Symptoms
- Alert takes long to evaluate
- Timeouts
- Delayed notifications

#### Solutions

**Solution 1: Optimize Query**
```yaml
Before:
  source logs
  | groupby $l.applicationname aggregate count()
  | filter count > 10

After:
  source logs
  | filter $m.severity >= ERROR
  | groupby $l.applicationname aggregate count()
  | filter count > 10
```

**Solution 2: Reduce Time Window**
```yaml
Before: 1 hour window
After: 15 minute window
```

**Solution 3: Add Specific Filters Early**
```yaml
Add filters before aggregations:
  source logs
  | filter $l.applicationname == "payment"
  | filter $m.severity >= ERROR
  | count
```

---

## Common Error Messages

### "Query syntax error"
**Cause**: Invalid DataPrime syntax  
**Fix**: Validate query in Explore Logs first

### "Field not found"
**Cause**: Field name doesn't exist in logs  
**Fix**: Check sample logs for correct field names

### "Threshold not met"
**Cause**: Normal - condition not satisfied  
**Fix**: Verify threshold is appropriate

### "Notification channel not configured"
**Cause**: Channel setup incomplete  
**Fix**: Complete channel configuration

### "Alert quota exceeded"
**Cause**: Too many alerts created  
**Fix**: Remove unused alerts or contact support

### "Invalid time window"
**Cause**: Time window out of allowed range  
**Fix**: Use 1 minute to 24 hours

---

## Debugging Workflow

### Step-by-Step Debugging Process

```
1. ✅ Check Alert Status
   - Is alert enabled?
   - Any maintenance windows?
   - Check alert history

2. ✅ Verify TCO Policy (MOST IMPORTANT)
   - Are logs in High/Medium priority?
   - Check TCO policy routing
   - Test with sample logs

3. ✅ Validate Query
   - Test in Explore Logs
   - Check field names
   - Verify filters

4. ✅ Review Conditions
   - Is threshold appropriate?
   - Is time window correct?
   - Check operators

5. ✅ Test Notifications
   - Verify channel configuration
   - Test with manual trigger
   - Check recipient settings

6. ✅ Check Logs
   - Review alert evaluation logs
   - Check for errors
   - Verify trigger history

7. ✅ Generate Test Data
   - Create logs matching conditions
   - Wait for evaluation period
   - Verify alert triggers
```

---

## Prevention Best Practices

### 1. Test Before Deploying
```
- Test query in Explore Logs
- Verify with historical data
- Use alert preview
- Start with low severity
```

### 2. Document Alerts
```
Include in alert description:
- What it monitors
- Why it's important
- How to investigate
- Who to contact
```

### 3. Regular Reviews
```
Monthly:
- Review alert effectiveness
- Check false positive rate
- Adjust thresholds
- Remove obsolete alerts
```

### 4. Monitor Alert Health
```
Track:
- Alert trigger frequency
- False positive rate
- Response time
- Resolution time
```

### 5. Use Alert Hierarchy
```
Critical → Immediate action
High → Urgent attention
Medium → Should address
Low → Informational
```

---

## Escalation Checklist

Before escalating to support, gather:

- [ ] Alert configuration (name, type, query)
- [ ] Sample logs that should trigger alert
- [ ] TCO policy configuration
- [ ] Alert evaluation history
- [ ] Notification channel setup
- [ ] Screenshots of issue
- [ ] Steps already attempted
- [ ] Expected vs actual behavior
- [ ] Timeline of when issue started
- [ ] Recent configuration changes

---

## Quick Reference Commands

### Test Alert Query
```
1. Go to Explore Logs
2. Paste alert query
3. Run query
4. Verify results
```

### Check Log Priority
```
1. Run query in Explore Logs
2. Look for pipeline indicator
3. Verify High or Medium priority
```

### Verify TCO Policy
```
1. Go to Data Flow → TCO Policies
2. Find matching policy
3. Check pipeline assignment
4. Verify policy order
```

### Test Notification Channel
```
Email: Send test email
Slack: Test webhook with curl
PagerDuty: Create test incident
Webhook: Test endpoint with curl