# Integration Tests

This directory contains integration tests for the IBM Cloud Logs MCP Server. These tests make real API calls to IBM Cloud services and require valid credentials.

## Prerequisites

1. **IBM Cloud Account** with access to:
   - IBM Cloud Logs service
   - (Optional) IBM Event Notifications service for webhook tests

2. **Environment Variables** - Set the following required variables:
   - `LOGS_API_KEY` - Your IBM Cloud API key ([Get from here](https://cloud.ibm.com/iam/apikeys))
   - `LOGS_INSTANCE_ID` - Your IBM Cloud Logs instance ID
   - `LOGS_REGION` - Your IBM Cloud region (e.g., `us-south`, `eu-de`, `au-syd`)

3. **Optional Environment Variables**:
   - `EVENT_NOTIFICATIONS_INSTANCE_ID` - Your IBM Event Notifications instance ID (required for webhook integration tests)

## Setup

### Option 1: Using .env file (Recommended)

1. Copy the example file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and fill in your values:
   ```bash
   LOGS_API_KEY=your-actual-api-key
   LOGS_INSTANCE_ID=your-actual-instance-id
   LOGS_REGION=us-south

   # Optional: For webhook tests
   EVENT_NOTIFICATIONS_INSTANCE_ID=your-event-notifications-instance-id
   ```

3. The tests will automatically load the `.env` file.

### Option 2: Export environment variables

```bash
export LOGS_API_KEY=your-actual-api-key
export LOGS_INSTANCE_ID=your-actual-instance-id
export LOGS_REGION=us-south

# Optional: For webhook tests
export EVENT_NOTIFICATIONS_INSTANCE_ID=your-event-notifications-instance-id
```

## Running Tests

### Run all integration tests:
```bash
make test-integration
```

### Run specific test suites:
```bash
# E2M tests only
make test-integration-e2m

# Webhook tests only
make test-integration-webhooks

# Policy tests only
go test -v -tags=integration ./tests/integration/ -run TestPolicies

# View tests only
go test -v -tags=integration ./tests/integration/ -run TestViews
```

### Run a specific test:
```bash
go test -v -tags=integration ./tests/integration/ -run TestE2MCRUD
```

## Test Coverage

### ✅ E2M (Events-to-Metrics) Tests
- `TestE2MCRUD` - Create, Read, Update, Delete operations
- `TestE2MWithAggregations` - Count and samples aggregations
- `TestE2MWithHistogram` - Histogram aggregation
- `TestE2MWithMultipleLabels` - Multiple metric labels
- `TestE2MErrorHandling` - Error scenarios
- `TestE2MPagination` - Pagination for listing E2M

### ✅ Policy Tests
- `TestPoliciesCRUD` - Create, Read, Update, Delete operations
- `TestPoliciesArchiveRetention` - Archive retention policies
- `TestPoliciesPriority` - Policy priority ordering
- `TestPoliciesWithRuleMatchers` - Different rule matchers (start_with, includes)
- `TestPoliciesErrorHandling` - Error scenarios

### ✅ View Tests
- `TestViewsCRUD` - Create, Read, Update, Delete operations
- `TestViewFoldersCRUD` - View folder operations
- `TestViewInFolder` - Creating views in folders
- `TestViewsWithCustomTimeSelection` - Custom time selections
- `TestViewsErrorHandling` - Error scenarios
- `TestViewsPagination` - Pagination for listing views (⚠️ currently failing due to backend consistency issues)

### ⏭️ Webhook Tests
- `TestWebhooksCRUD` - Skipped (generic webhooks not supported)
- `TestWebhookTypes` - Skipped (generic/slack/pagerduty not supported)
- `TestIBMEventNotificationsWebhook` - **Runs only if `EVENT_NOTIFICATIONS_INSTANCE_ID` is set**
- `TestWebhooksErrorHandling` - Skipped (generic webhooks not supported)
- `TestWebhooksPagination` - Skipped (generic webhooks not supported)
- `TestWebhookWithCustomHeaders` - Skipped (generic webhooks not supported)

## IBM Event Notifications Webhook Testing

To test IBM Event Notifications webhooks:

1. **Create an Event Notifications instance** (if you don't have one):
   - Go to [IBM Cloud Catalog](https://cloud.ibm.com/catalog/services/event-notifications)
   - Create an Event Notifications service instance
   - Note the instance ID from the service details

2. **Set the environment variable**:
   ```bash
   export EVENT_NOTIFICATIONS_INSTANCE_ID=your-instance-id
   ```
   Or add it to your `.env` file:
   ```bash
   EVENT_NOTIFICATIONS_INSTANCE_ID=your-instance-id
   ```

3. **Run the webhook test**:
   ```bash
   go test -v -tags=integration ./tests/integration/ -run TestIBMEventNotificationsWebhook
   ```

The test will automatically skip if the `EVENT_NOTIFICATIONS_INSTANCE_ID` is not set.

## Troubleshooting

### Tests fail with authentication errors
- Verify your `LOGS_API_KEY` is valid and not expired
- Check that your API key has appropriate permissions for the Logs service

### Tests fail with instance not found
- Verify your `LOGS_INSTANCE_ID` matches your actual Logs instance
- Ensure the instance exists in the specified `LOGS_REGION`

### Webhook tests are skipped
- Most webhook types (generic, slack, pagerduty) are not supported by the API
- Only IBM Event Notifications webhooks are supported
- Set `EVENT_NOTIFICATIONS_INSTANCE_ID` to enable the webhook test

### TestViewsPagination fails
- This is a known issue with backend eventual consistency
- Views may not immediately appear in the list response
- The test retries for 30 seconds but may still fail

## Notes

- Integration tests make **real API calls** and may create/modify/delete resources in your IBM Cloud account
- Tests include cleanup logic to remove created resources
- Rate limiting is configured to avoid API throttling
- Some tests may take several seconds to complete due to API response times
