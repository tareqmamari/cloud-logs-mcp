// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

// OutputSchemas provides standard JSON Schema definitions for tool outputs.
// These help LLMs understand the structure of responses for better planning.

// CommonOutputSchemas contains reusable output schema definitions
var CommonOutputSchemas = struct {
	// SingleItem is the schema for single resource responses (get operations)
	SingleItem map[string]interface{}
	// ItemList is the schema for list responses
	ItemList map[string]interface{}
	// QueryResult is the schema for query responses
	QueryResult map[string]interface{}
	// Success is the schema for simple success/failure responses
	Success map[string]interface{}
	// ValidationResult is the schema for dry-run validation responses
	ValidationResult map[string]interface{}
}{
	SingleItem: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Unique identifier of the resource",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable name of the resource",
			},
			"created_at": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Creation timestamp",
			},
			"updated_at": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Last update timestamp",
			},
		},
	},
	ItemList: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"items": map[string]interface{}{
				"type":        "array",
				"description": "Array of resource objects",
				"items":       map[string]interface{}{"type": "object"},
			},
			"total": map[string]interface{}{
				"type":        "integer",
				"description": "Total number of items available",
			},
			"_pagination": map[string]interface{}{
				"type":        "object",
				"description": "Pagination metadata (present when more results available)",
				"properties": map[string]interface{}{
					"has_more":    map[string]interface{}{"type": "boolean"},
					"next_cursor": map[string]interface{}{"type": "string"},
				},
			},
		},
	},
	QueryResult: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"events": map[string]interface{}{
				"type":        "array",
				"description": "Array of log events matching the query",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"timestamp":       map[string]interface{}{"type": "string"},
						"severity":        map[string]interface{}{"type": "integer"},
						"applicationname": map[string]interface{}{"type": "string"},
						"subsystemname":   map[string]interface{}{"type": "string"},
						"text":            map[string]interface{}{"type": "string"},
					},
				},
			},
			"_truncated": map[string]interface{}{
				"type":        "boolean",
				"description": "True if results were truncated due to size limits",
			},
			"_total_events": map[string]interface{}{
				"type":        "integer",
				"description": "Total events before truncation (when truncated)",
			},
		},
	},
	Success: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"success": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the operation succeeded",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable result message",
			},
		},
	},
	ValidationResult: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"valid": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the configuration is valid",
			},
			"errors": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of validation errors",
			},
			"warnings": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of warnings (non-blocking issues)",
			},
			"summary": map[string]interface{}{
				"type":        "object",
				"description": "Summary of what would be created/modified",
			},
		},
	},
}

// AlertOutputSchema is the output schema for alert resources
var AlertOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"id": map[string]interface{}{
			"type":        "string",
			"description": "Unique alert identifier (UUID)",
		},
		"name": map[string]interface{}{
			"type":        "string",
			"description": "Alert name",
		},
		"is_active": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether the alert is currently active",
		},
		"alert_definition_id": map[string]interface{}{
			"type":        "string",
			"description": "ID of the associated alert definition",
		},
		"notification_group_id": map[string]interface{}{
			"type":        "string",
			"description": "ID of the notification group for this alert",
		},
		"filters": map[string]interface{}{
			"type":        "object",
			"description": "Log filters applied to this alert",
		},
	},
}

// DashboardOutputSchema is the output schema for dashboard resources
var DashboardOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"id": map[string]interface{}{
			"type":        "string",
			"description": "Unique dashboard identifier (UUID)",
		},
		"name": map[string]interface{}{
			"type":        "string",
			"description": "Dashboard name",
		},
		"description": map[string]interface{}{
			"type":        "string",
			"description": "Dashboard description",
		},
		"folder_id": map[string]interface{}{
			"type":        "string",
			"description": "ID of the folder containing this dashboard",
		},
		"is_pinned": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether the dashboard is pinned for quick access",
		},
		"widgets": map[string]interface{}{
			"type":        "array",
			"description": "Array of dashboard widgets/panels",
		},
	},
}

// PolicyOutputSchema is the output schema for policy resources
var PolicyOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"id": map[string]interface{}{
			"type":        "string",
			"description": "Unique policy identifier (UUID)",
		},
		"name": map[string]interface{}{
			"type":        "string",
			"description": "Policy name",
		},
		"priority": map[string]interface{}{
			"type":        "string",
			"description": "Policy priority tier (frequent_search, high, medium, low, archive)",
		},
		"application_rule": map[string]interface{}{
			"type":        "object",
			"description": "Application matching rules",
		},
		"subsystem_rule": map[string]interface{}{
			"type":        "object",
			"description": "Subsystem matching rules",
		},
	},
}

// WebhookOutputSchema is the output schema for outgoing webhook resources
var WebhookOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"id": map[string]interface{}{
			"type":        "string",
			"description": "Unique webhook identifier (UUID)",
		},
		"name": map[string]interface{}{
			"type":        "string",
			"description": "Webhook name",
		},
		"type": map[string]interface{}{
			"type":        "string",
			"description": "Webhook type (slack, pagerduty, email, generic)",
		},
		"url": map[string]interface{}{
			"type":        "string",
			"description": "Webhook destination URL",
		},
		"is_active": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether the webhook is currently active",
		},
	},
}

// QueryBackgroundOutputSchema is the output schema for background query operations
var QueryBackgroundOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"query_id": map[string]interface{}{
			"type":        "string",
			"description": "Unique identifier for the background query",
		},
		"status": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"pending", "running", "completed", "failed", "cancelled"},
			"description": "Current status of the query",
		},
		"progress": map[string]interface{}{
			"type":        "integer",
			"description": "Query progress percentage (0-100)",
		},
		"result_count": map[string]interface{}{
			"type":        "integer",
			"description": "Number of results (when completed)",
		},
	},
}

// IngestionOutputSchema is the output schema for log ingestion
var IngestionOutputSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"success": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether ingestion succeeded",
		},
		"ingested_count": map[string]interface{}{
			"type":        "integer",
			"description": "Number of log entries successfully ingested",
		},
		"failed_count": map[string]interface{}{
			"type":        "integer",
			"description": "Number of log entries that failed to ingest",
		},
	},
}
