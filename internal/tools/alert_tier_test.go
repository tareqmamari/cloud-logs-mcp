package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// sessionRoutingAppToPriority builds a session whose single TCO policy assigns
// the given priority to the given application. Tier is archive for medium/low
// (mirroring determineTier), but the alert warning keys off priority, not tier.
func sessionRoutingAppToPriority(app, priority string) *SessionContext {
	tier := "frequent_search"
	if priority != "type_high" {
		tier = "archive"
	}
	s := NewSessionContext("alert-tier-test", "instance-1")
	s.SetTCOConfig(&TCOConfig{
		HasPolicies:       true,
		HasArchive:        true,
		HasFrequentSearch: true,
		DefaultTier:       "frequent_search",
		PolicyCount:       1,
		LastUpdated:       time.Now(),
		Policies: []TCOPolicyRule{
			{ApplicationRule: &TCOMatchRule{Name: app, RuleType: "is"}, Tier: tier, Priority: priority},
		},
	})
	return s
}

func TestAlertTierWarnings_LowPriorityTargetWarns(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_low")
	// thresholdDefWithLabelFilterOp sets application_name value = payment-service
	warns := alertTierWarnings(thresholdDefWithLabelFilterOp("is"), session)
	assert.NotEmpty(t, warns, "an alert over low-priority logs should be warned about")
	assert.Contains(t, strings.Join(warns, "\n"), "will not fire")
}

func TestAlertTierWarnings_MediumPriorityNoWarn(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_medium")
	assert.Empty(t, alertTierWarnings(thresholdDefWithLabelFilterOp("is"), session),
		"medium-priority logs are alertable and must not warn")
}

func TestAlertTierWarnings_HighPriorityNoWarn(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_high")
	assert.Empty(t, alertTierWarnings(thresholdDefWithLabelFilterOp("is"), session))
}

func TestAlertTierWarnings_LuceneLowPriorityWarns(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_low")
	def := map[string]interface{}{
		"name": "x",
		"type": "logs_threshold",
		"logs_threshold": map[string]interface{}{
			"logs_filter": map[string]interface{}{
				"simple_filter": map[string]interface{}{
					"lucene_query": "applicationname:payment-service AND severity:error",
				},
			},
		},
	}
	assert.NotEmpty(t, alertTierWarnings(def, session))
}

func TestAlertTierWarnings_NilSession(t *testing.T) {
	assert.Empty(t, alertTierWarnings(thresholdDefWithLabelFilterOp("is"), nil))
}

func TestAlertTierWarnings_UnmatchedTargetNoWarn(t *testing.T) {
	// The alert targets an app no policy matches -> default (alertable) routing.
	session := sessionRoutingAppToPriority("other-app", "type_low")
	assert.Empty(t, alertTierWarnings(thresholdDefWithLabelFilterOp("is"), session))
}

func TestAlertTierWarnings_NoTargetNoWarn(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_low")
	def := map[string]interface{}{
		"name": "x",
		"type": "logs_threshold",
		"logs_threshold": map[string]interface{}{
			"logs_filter": map[string]interface{}{
				"simple_filter": map[string]interface{}{"lucene_query": "severity:error"},
			},
		},
	}
	assert.Empty(t, alertTierWarnings(def, session))
}

// TestCreateAlertDefinition_DryRunSurfacesLowPriorityWarning verifies the
// warning reaches the user through the dry-run Execute path, not just the helper.
func TestCreateAlertDefinition_DryRunSurfacesLowPriorityWarning(t *testing.T) {
	session := sessionRoutingAppToPriority("payment-service", "type_low")
	mock := client.NewMockClient()
	tool := NewCreateAlertDefinitionTool(mock, zap.NewNop())
	ctx := WithSession(testCtx(mock), session)

	result, err := tool.Execute(ctx, map[string]interface{}{
		"definition": thresholdDefWithLabelFilterOp("is"),
		"dry_run":    true,
	})
	assert.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(t, result), "will not fire")
}
