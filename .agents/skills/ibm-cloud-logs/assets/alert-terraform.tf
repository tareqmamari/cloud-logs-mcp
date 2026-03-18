# Terraform configuration for IBM Cloud Logs alert
# Template extracted from generateTerraform() in investigation_remediation.go
#
# Usage:
#   1. Replace placeholder values (<alert_name>, <description>, <query_condition>)
#   2. Set the notification_integration_id variable
#   3. Run: terraform init && terraform apply

resource "ibm_logs_alert" "example_alert" {
  name        = "<alert_name>"
  description = "<description>"
  severity    = "ERROR"  # CRITICAL | ERROR | WARNING | INFO
  is_active   = true

  condition {
    logs_ratio_threshold {
      rules {
        condition {
          condition_type = "MORE_THAN"
          threshold      = 5
          time_window    = "FIVE_MINUTES"
        }
        override {
          priority = "P2"
        }
      }

      query_1 {
        search_query {
          query = "<query_condition>"
        }
      }

      query_2 {
        search_query {
          query = "$m.severity >= 1"
        }
      }
    }
  }

  notification_groups {
    notifications {
      notify_on      = ["Triggered"]
      integration_id = var.notification_integration_id
    }
  }
}

variable "notification_integration_id" {
  description = "Integration ID for alert notifications (Slack, PagerDuty, etc.)"
  type        = string
}

output "example_alert_alert_id" {
  description = "The ID of the created alert"
  value       = ibm_logs_alert.example_alert.id
}

# -----------------------------------------------------------------------
# Severity mapping reference:
#   critical --> CRITICAL  (P1 page)
#   high     --> ERROR     (P1/P2)
#   medium   --> WARNING   (P2 ticket)
#   low      --> INFO      (P3 informational)
#
# Condition types available:
#   - logs_ratio_threshold   (ratio of query_1 / query_2)
#   - logs_threshold         (absolute count)
#   - logs_new_value         (new value detection)
#   - logs_unique_count      (unique value count)
#
# Time window values:
#   FIVE_MINUTES, TEN_MINUTES, FIFTEEN_MINUTES,
#   TWENTY_MINUTES, THIRTY_MINUTES, ONE_HOUR,
#   TWO_HOURS, FOUR_HOURS, SIX_HOURS,
#   TWELVE_HOURS, TWENTY_FOUR_HOURS
# -----------------------------------------------------------------------
