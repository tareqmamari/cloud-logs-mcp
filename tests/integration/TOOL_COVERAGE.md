# Integration Test Tool Coverage Analysis

## Summary

**Total MCP Tools Implemented:** 86 tools
**Tools with Integration Tests:** 30 tools (35%)
**Tools without Integration Tests:** 56 tools (65%)

## ✅ Covered by Integration Tests (30 tools)

### Alerts (5 tools) - ✅ TESTED & PASSING
- [x] `get_alert` - TestAlertsCRUD/GetAlert
- [x] `list_alerts` - TestAlertsCRUD/ListAlerts, TestAlertsPagination
- [x] `create_alert` - TestAlertsCRUD/CreateAlert, TestAlertsWithFilters
- [x] `update_alert` - TestAlertsCRUD/UpdateAlert
- [x] `delete_alert` - TestAlertsCRUD/DeleteAlert, TestAlertsErrorHandling

**Status:** 7 tests passing ✅

### Policies (5 tools) - 📝 TESTS WRITTEN
- [ ] `get_policy` - TestPoliciesCRUD/GetPolicy
- [ ] `list_policies` - TestPoliciesCRUD/ListPolicies
- [ ] `create_policy` - TestPoliciesCRUD/CreatePolicy
- [ ] `update_policy` - TestPoliciesCRUD/UpdatePolicy
- [ ] `delete_policy` - TestPoliciesCRUD/DeletePolicy

**Status:** Tests written, need API structure updates

### E2M (5 tools) - 📝 TESTS WRITTEN
- [ ] `get_e2m` - TestE2MCRUD/GetE2M
- [ ] `list_e2m` - TestE2MCRUD/ListE2M
- [ ] `create_e2m` - TestE2MCRUD/CreateE2M
- [ ] `replace_e2m` - TestE2MCRUD/UpdateE2M
- [ ] `delete_e2m` - TestE2MCRUD/DeleteE2M

**Status:** Tests written, need logs_query structure fix

### Views (5 tools) - 📝 TESTS WRITTEN
- [ ] `list_views` - TestViewsCRUD/ListViews
- [ ] `create_view` - TestViewsCRUD/CreateView
- [ ] `get_view` - TestViewsCRUD/GetView
- [ ] `replace_view` - TestViewsCRUD/UpdateView
- [ ] `delete_view` - TestViewsCRUD/DeleteView

**Status:** Tests written, need filters structure fix

### View Folders (5 tools) - 📝 TESTS WRITTEN
- [ ] `list_view_folders` - TestViewFoldersCRUD/ListViewFolders
- [ ] `create_view_folder` - TestViewFoldersCRUD/CreateViewFolder
- [ ] `get_view_folder` - TestViewFoldersCRUD/GetViewFolder
- [ ] `replace_view_folder` - TestViewFoldersCRUD/UpdateViewFolder
- [ ] `delete_view_folder` - TestViewFoldersCRUD/DeleteViewFolder

**Status:** Tests written, need validation

### Outgoing Webhooks (5 tools) - 📝 TESTS WRITTEN
- [ ] `get_outgoing_webhook` - TestWebhooksCRUD/GetWebhook
- [ ] `list_outgoing_webhooks` - TestWebhooksCRUD/ListWebhooks
- [ ] `create_outgoing_webhook` - TestWebhooksCRUD/CreateWebhook, TestWebhookTypes
- [ ] `update_outgoing_webhook` - TestWebhooksCRUD/UpdateWebhook
- [ ] `delete_outgoing_webhook` - TestWebhooksCRUD/DeleteWebhook

**Status:** Tests written, need IBM Event Notifications field name updates

## ⚠️ Not Covered by Integration Tests (56 tools)

### Alert Definitions (5 tools)
- [ ] `get_alert_definition`
- [ ] `list_alert_definitions`
- [ ] `create_alert_definition`
- [ ] `update_alert_definition`
- [ ] `delete_alert_definition`

**Priority:** Medium - Similar to alerts but separate API endpoint

### Rule Groups (5 tools)
- [ ] `get_rule_group`
- [ ] `list_rule_groups`
- [ ] `create_rule_group`
- [ ] `update_rule_group`
- [ ] `delete_rule_group`

**Priority:** High - Core functionality for log parsing rules

### Dashboards (9 tools)
- [ ] `get_dashboard`
- [ ] `list_dashboards`
- [ ] `create_dashboard`
- [ ] `update_dashboard`
- [ ] `delete_dashboard`
- [ ] `pin_dashboard`
- [ ] `unpin_dashboard`
- [ ] `set_default_dashboard`
- [ ] `move_dashboard_to_folder`

**Priority:** High - Important for visualization

### Dashboard Folders (5 tools)
- [ ] `get_dashboard_folder`
- [ ] `list_dashboard_folders`
- [ ] `create_dashboard_folder`
- [ ] `update_dashboard_folder`
- [ ] `delete_dashboard_folder`

**Priority:** Medium - Organizational feature

### Queries (5 tools)
- [ ] `query` - Run DataPrime/Lucene queries
- [ ] `submit_background_query`
- [ ] `get_background_query_status`
- [ ] `get_background_query_data`
- [ ] `cancel_background_query`

**Priority:** High - Core functionality for log searching

### Data Access Rules (5 tools)
- [ ] `list_data_access_rules`
- [ ] `get_data_access_rule`
- [ ] `create_data_access_rule`
- [ ] `update_data_access_rule`
- [ ] `delete_data_access_rule`

**Priority:** Low - Security/access control feature

### Enrichments (5 tools)
- [ ] `list_enrichments`
- [ ] `get_enrichments`
- [ ] `create_enrichment`
- [ ] `update_enrichment`
- [ ] `delete_enrichment`

**Priority:** Medium - Data enhancement feature

### Streams (5 tools)
- [ ] `get_stream`
- [ ] `list_streams`
- [ ] `create_stream`
- [ ] `update_stream`
- [ ] `delete_stream`

**Priority:** High - Data streaming to external systems

### Event Stream Targets (4 tools)
- [ ] `get_event_stream_targets`
- [ ] `create_event_stream_target`
- [ ] `update_event_stream_target`
- [ ] `delete_event_stream_target`

**Priority:** Medium - Event streaming configuration

### Data Usage (2 tools)
- [ ] `export_data_usage`
- [ ] `update_data_usage_metrics_export_status`

**Priority:** Low - Metrics and monitoring

### Ingestion (1 tool)
- [ ] `ingest_logs`

**Priority:** Medium - Log ingestion testing

## 📊 Coverage by Category

| Category | Total Tools | Tested | Coverage |
|----------|-------------|--------|----------|
| **Alerts** | 5 | 5 ✅ | 100% |
| **Policies** | 5 | 5 📝 | Written |
| **E2M** | 5 | 5 📝 | Written |
| **Views** | 5 | 5 📝 | Written |
| **View Folders** | 5 | 5 📝 | Written |
| **Webhooks** | 5 | 5 📝 | Written |
| **Alert Definitions** | 5 | 0 | 0% |
| **Rule Groups** | 5 | 0 | 0% |
| **Dashboards** | 9 | 0 | 0% |
| **Dashboard Folders** | 5 | 0 | 0% |
| **Queries** | 5 | 0 | 0% |
| **Data Access Rules** | 5 | 0 | 0% |
| **Enrichments** | 5 | 0 | 0% |
| **Streams** | 5 | 0 | 0% |
| **Event Stream Targets** | 4 | 0 | 0% |
| **Data Usage** | 2 | 0 | 0% |
| **Ingestion** | 1 | 0 | 0% |
| **TOTAL** | **86** | **30** | **35%** |

## 🎯 Recommended Next Steps

### Phase 1: Fix Existing Tests (High Priority)
1. **Policies** - Add `enabled: true` field
2. **E2M** - Fix `logs_query` structure with filters
3. **Views** - Fix `filters.filters` nesting
4. **Webhooks** - Update IBM Event Notifications field names

### Phase 2: Add High-Priority Missing Tests
1. **Queries** (5 tools) - Core log searching functionality
   - Especially `query` tool for DataPrime/Lucene
2. **Rule Groups** (5 tools) - Log parsing and processing
3. **Dashboards** (9 tools) - Visualization layer
4. **Streams** (5 tools) - Data export functionality

### Phase 3: Complete Coverage (Medium Priority)
1. **Alert Definitions** (5 tools)
2. **Dashboard Folders** (5 tools)
3. **Enrichments** (5 tools)
4. **Event Stream Targets** (4 tools)

### Phase 4: Low Priority
1. **Data Access Rules** (5 tools)
2. **Data Usage** (2 tools)
3. **Ingestion** (1 tool)

## 📝 Test Template for Missing Tools

For each missing tool category, follow this pattern:

```go
func TestResourceCRUD(t *testing.T) {
    skipIfShort(t)
    tc := NewTestContext(t)
    defer tc.Cleanup()

    var resourceID string

    t.Run("CreateResource", func(t *testing.T) {
        // Create with proper structure from Terraform examples
    })

    t.Run("GetResource", func(t *testing.T) {
        // Verify resource exists and has correct fields
    })

    t.Run("ListResources", func(t *testing.T) {
        // Verify resource in list
    })

    t.Run("UpdateResource", func(t *testing.T) {
        // GET before PUT pattern
    })

    t.Run("DeleteResource", func(t *testing.T) {
        // Delete and verify 404
    })
}
```

## 🔍 Current Test Statistics

- **Test Files:** 6 files
- **Test Lines:** ~2,200 lines
- **Passing Tests:** 7 tests (alerts only)
- **Written but Untested:** 23 tools (need minor fixes)
- **Not Written:** 56 tools

## 💡 Testing Gaps by Importance

**Critical (Should Test):**
- Queries (log searching - core feature)
- Rule Groups (log processing)
- Dashboards (visualization)
- Streams (data export)

**Important (Nice to Have):**
- Alert Definitions
- Dashboard Folders
- Enrichments
- Event Stream Targets

**Optional (Can Skip):**
- Data Access Rules
- Data Usage
- Ingestion (tested indirectly)

## 📚 Reference Materials

- **API Structures:** [API_STRUCTURES.md](API_STRUCTURES.md)
- **Terraform Examples:** https://github.com/IBM-Cloud/terraform-provider-ibm/tree/master/examples/ibm-logs
- **API Spec:** [logs-service-api.json](../../logs-service-api.json)
- **Passing Tests:** [alerts_test.go](alerts_test.go)
