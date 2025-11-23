# Integration Tests - Quick Start Guide

Get started with IBM Cloud Logs integration tests in 5 minutes!

## 1. Prerequisites

- IBM Cloud account
- IBM Cloud Logs instance
- Go 1.21 or later

## 2. Get Your Credentials

### IBM Cloud API Key
```bash
# Login to IBM Cloud CLI
ibmcloud login

# Create an API key
ibmcloud iam api-key-create logs-mcp-tests -d "API key for logs MCP integration tests"

# Copy the API Key value
```

Or create via web UI: https://cloud.ibm.com/iam/apikeys

### Instance ID and Region
```bash
# List your IBM Cloud Logs instances
ibmcloud resource service-instances --service-name logs

# Get instance details
ibmcloud resource service-instance <instance-name>

# Note the Instance ID (GUID) and Region
```

## 3. Set Environment Variables

```bash
export LOGS_API_KEY="your-api-key-here"
export LOGS_INSTANCE_ID="your-instance-id-here"
export LOGS_REGION="us-south"  # or your region
```

**Tip:** Create a `.env` file and source it:
```bash
cp tests/integration/.env.example tests/integration/.env
# Edit tests/integration/.env with your credentials
source tests/integration/.env
```

## 4. Run Tests

### Run All Integration Tests
```bash
make test-integration
```

### Run Specific Test Suite
```bash
# Alert tests
make test-integration-alerts

# Policy tests
make test-integration-policies

# E2M tests
make test-integration-e2m

# View tests
make test-integration-views

# Webhook tests
make test-integration-webhooks
```

### Run Individual Test
```bash
go test -v -tags=integration ./tests/integration/ -run TestAlertsCRUD
```

## 5. Understanding Test Output

### Successful Test
```
=== RUN   TestAlertsCRUD
=== RUN   TestAlertsCRUD/CreateAlert
=== RUN   TestAlertsCRUD/GetAlert
=== RUN   TestAlertsCRUD/ListAlerts
=== RUN   TestAlertsCRUD/UpdateAlert
=== RUN   TestAlertsCRUD/DeleteAlert
--- PASS: TestAlertsCRUD (5.23s)
    --- PASS: TestAlertsCRUD/CreateAlert (1.45s)
    --- PASS: TestAlertsCRUD/GetAlert (0.82s)
    --- PASS: TestAlertsCRUD/ListAlerts (1.23s)
    --- PASS: TestAlertsCRUD/UpdateAlert (1.15s)
    --- PASS: TestAlertsCRUD/DeleteAlert (0.58s)
PASS
ok      github.com/tareqmamari/logs-mcp-server/tests/integration    5.234s
```

### Failed Test
```
--- FAIL: TestAlertsCRUD (2.15s)
    --- FAIL: TestAlertsCRUD/CreateAlert (2.15s)
        alerts_test.go:45: Failed to create alert: HTTP 401: Unauthorized
```

## 6. Troubleshooting

### "LOGS_API_KEY not set"

If you see this error, it means the environment variables are not exported in your shell.

Check if they are set:
```bash
echo $LOGS_API_KEY
```

If empty, export them again:
```bash
export LOGS_API_KEY="your-key-here"
```

### "401 Unauthorized"
- Verify API key is correct and not expired
- Check IAM permissions for IBM Cloud Logs service
- Ensure you're using the correct region and instance ID

### "404 Not Found"
- Verify instance ID is correct
- Check region matches your instance location
- Ensure IBM Cloud Logs instance is active

### "429 Too Many Requests"
- Tests include delays to avoid rate limiting
- Wait a few minutes and retry
- Run individual test suites instead of all tests

### "timeout waiting"
- Increase timeout in test configuration
- Check network connectivity to IBM Cloud
- Verify IBM Cloud Logs service is operational

## 7. Test Examples

### Alert with Text Filter
```go
alertConfig := map[string]interface{}{
    "name":        "Error Alert",
    "description": "Alert on error logs",
    "is_active":   true,
    "severity":    "error",
    "notification_groups": []map[string]interface{}{
        {
            "group_by_fields": []string{"applicationName"},
        },
    },
    "condition": map[string]interface{}{
        "immediate": map[string]interface{}{},
    },
    "filters": map[string]interface{}{
        "text": "error",
        "severities": []string{"error"},
    },
}
```

### E2M with Count Aggregation
```go
e2mConfig := map[string]interface{}{
    "name":        "Error Count",
    "description": "Count error logs",
    "logs_query": map[string]interface{}{
        "lucene": "severity:error",
    },
    "metric_labels": []map[string]interface{}{
        {
            "target_label": "service",
            "source_field": "applicationName",
        },
    },
    "metric_fields": []map[string]interface{}{
        {
            "target_base_metric_name": "error_count",
            "aggregations": []map[string]interface{}{
                {
                    "enabled":            true,
                    "agg_type":           "count",
                    "target_metric_name": "total_errors",
                },
            },
        },
    },
    "type": "logs2metrics",
}
```

## 8. Best Practices

1. **Use a Test Instance**: Don't run tests against production
2. **Clean Up Resources**: Tests auto-cleanup, but verify
3. **Monitor Costs**: Integration tests create real resources
4. **Rotate API Keys**: Change keys regularly
5. **Run Selectively**: Run individual suites during development
6. **Check Rate Limits**: Space out test runs if hitting limits

## 9. CI/CD Integration

### GitHub Actions
```yaml
- name: Run Integration Tests
  env:
    LOGS_API_KEY: ${{ secrets.LOGS_API_KEY }}
    LOGS_INSTANCE_ID: ${{ secrets.LOGS_INSTANCE_ID }}
    LOGS_REGION: ${{ secrets.LOGS_REGION }}
  run: make test-integration
```

See [.github/workflows/integration-tests.yml.example](../../.github/workflows/integration-tests.yml.example) for complete example.

## 10. Next Steps

- Read the full [Integration Tests README](README.md)
- Check [Terraform examples](https://github.com/IBM-Cloud/terraform-provider-ibm/tree/master/examples/ibm-logs)
- Review [IBM Cloud Logs API docs](https://cloud.ibm.com/apidocs/logs-service-api)
- Explore the [API specification](../../logs-service-api.json)

## Need Help?

- Check the [troubleshooting section](#6-troubleshooting)
- Review test logs for detailed error messages
- Consult [IBM Cloud Logs documentation](https://cloud.ibm.com/docs/cloud-logs)
- Open an issue with test output and error messages
