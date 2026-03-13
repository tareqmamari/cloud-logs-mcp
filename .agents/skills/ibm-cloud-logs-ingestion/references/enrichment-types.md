# Enrichment Types Reference

Enrichments automatically add context to incoming logs during ingestion. They are managed through the `/v1/enrichments` API endpoint.

## Enrichment Object Schema

```json
{
  "name": "string (required)",
  "description": "string (optional)",
  "field_name": "string (required) -- source field for enrichment lookup",
  "enrichment_type": "string (required) -- geo_ip or custom_enrichment",
  "geo_ip_config": {},
  "custom_enrichment_config": {}
}
```

## geo_ip

Adds geographic information based on IP address values found in a specified log field.

### Fields Added

When a valid IP address is found in the source field, the following data is appended to the log entry:

- Country name and ISO code
- City name
- Latitude and longitude
- ASN (Autonomous System Number) information

### Configuration

```json
{
  "name": "Client IP Geolocation",
  "description": "Add geographic data from client IP addresses",
  "field_name": "json.client_ip",
  "enrichment_type": "geo_ip"
}
```

- `field_name` -- The log field containing the IP address to look up. Use dot notation for nested fields (e.g., `json.client_ip`, `json.source_ip`).
- No additional `geo_ip_config` object is required for basic geo-IP enrichment.

### Use Cases

- Map client IP addresses to geographic regions for traffic analysis
- Detect anomalous login locations for security monitoring
- Build geographic dashboards showing request distribution

## custom_enrichment

Adds fields from a lookup table, mapping a log field value to additional context stored in the table.

### Configuration

```json
{
  "name": "Customer Tier Lookup",
  "description": "Enrich logs with customer tier information",
  "field_name": "json.customer_id",
  "enrichment_type": "custom_enrichment",
  "custom_enrichment_config": {
    "lookup_table_id": "customer-tiers-table"
  }
}
```

- `field_name` -- The log field whose value is used as the lookup key
- `custom_enrichment_config.lookup_table_id` -- The ID of the lookup table to query

### Use Cases

- Map customer IDs to tier or plan information
- Map host names to environment labels (production, staging, development)
- Map error codes to human-readable descriptions
- Map service IDs to team ownership

## API Endpoints

| Operation | Method | Path |
|---|---|---|
| List all enrichments | GET | `/v1/enrichments` |
| Get all enrichments | GET | `/v1/enrichments` |
| Create enrichment | POST | `/v1/enrichments` |
| Update enrichment | PUT | `/v1/enrichments/{id}` |
| Delete enrichment | DELETE | `/v1/enrichments/{id}` |

## Related Tools

- `list_enrichments` / `get_enrichments` -- View configured enrichments
- `create_enrichment` -- Create a new enrichment
- `update_enrichment` -- Modify an existing enrichment
- `delete_enrichment` -- Remove an enrichment
