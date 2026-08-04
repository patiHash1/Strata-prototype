# AI & Platform API

The AI & Platform module provides text-to-SQL copilot, low-code workflow automation, and security audit anomaly detection. All endpoints require Bearer token authentication with the appropriate permissions.

## Endpoints

| Method | Endpoint | Permission | Description |
|---|---|---|---|
| `POST` | `/api/v1/ai/copilot/query` | `copilot.use` | Execute text-to-SQL AI copilot query |
| `POST` | `/api/v1/workflows/trigger` | `workflows.execute` | Trigger low-code automated workflow |
| `GET` | `/api/v1/security/audit-anomalies` | `security.audit.read` | Fetch security threat & anomaly audit logs |

---

## POST /api/v1/ai/copilot/query

Converts a natural language prompt to SQL, executes a simulated query, and returns results with a chart recommendation.

**Permission:** `copilot.use`

### Request

```json
{
  "prompt": "Show top 5 sales reps by revenue in Q2"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `prompt` | `string` | ✅ | Natural language prompt describing the desired query |

### Response `200 OK`

```json
{
  "generated_sql": "SELECT u.full_name AS sales_rep, SUM(d.amount) AS total_revenue\nFROM crm_deals d\nJOIN users u ON d.assigned_to = u.id\nWHERE d.stage = 'closed_won'\n  AND d.created_at BETWEEN '2025-04-01' AND '2025-06-30'\nGROUP BY u.full_name\nORDER BY total_revenue DESC\nLIMIT 5",
  "data_table": [
    {"sales_rep": "Alice Johnson", "total_revenue": 245000.50},
    {"sales_rep": "Bob Martinez", "total_revenue": 198750.00},
    {"sales_rep": "Carol Chen", "total_revenue": 176200.75},
    {"sales_rep": "David Kim", "total_revenue": 152300.25},
    {"sales_rep": "Eve Thompson", "total_revenue": 134500.00}
  ],
  "chart_recommendation": "bar_chart"
}
```

| Field | Type | Description |
|---|---|---|
| `generated_sql` | `string` | The SQL query generated from the natural language prompt |
| `data_table` | `array` | Array of result row objects matching the generated SQL |
| `chart_recommendation` | `string` | Recommended chart type (`bar_chart`, `line_chart`, `table`) |

### Prompt handling

The copilot recognizes several prompt categories and generates domain-appropriate SQL:

| Prompt category | Target table | Example prompt |
|---|---|---|
| Sales/Revenue | `crm_deals` | "Show top 5 sales reps by revenue in Q2" |
| Invoices/Overdue | `invoices` | "List all overdue invoices" |
| Attendance | `attendance_logs` | "Show attendance for the last 30 days" |
| Fleet/Vehicles | `fleet_telematics_logs` | "Show max speed by vehicle this week" |
| General | `organizations` | Falls back to a basic org query |

### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body or missing `prompt` |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `copilot.use` permission |
| `500` | Internal server error |

---

## POST /api/v1/workflows/trigger

Triggers a low-code automation workflow in response to an event type. Matches active workflows by `event_type` and simulates execution of their action steps.

**Permission:** `workflows.execute`

### Request

```json
{
  "event_type": "invoice.paid",
  "payload": {
    "invoice_id": "INV-2025-0042",
    "amount": 12500
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `event_type` | `string` | ✅ | Event type to match against workflow trigger definitions (e.g., `invoice.paid`, `deal.won`) |
| `payload` | `object` | — | JSON object containing event context data |

### Response `200 OK`

```json
{
  "workflow_execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "steps_executed": 4,
  "status": "success"
}
```

| Field | Type | Description |
|---|---|---|
| `workflow_execution_id` | `string` (UUID) | Unique identifier for this workflow execution |
| `steps_executed` | `integer` | Number of action steps executed |
| `status` | `string` | Execution status — always `"success"` for successful triggers |

### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body or missing `event_type` |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `workflows.execute` permission |
| `500` | Internal server error |

---

## GET /api/v1/security/audit-anomalies

Retrieves security audit log entries flagged as anomalies by the AI detection system. Results are filterable by severity level.

**Permission:** `security.audit.read`

### Query parameters

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `severity` | `string` | — | `high` | Filter by severity: `low`, `medium`, `high`, `critical` |
| `limit` | `integer` | — | `20` | Maximum number of results (1–100) |

### Response `200 OK`

```json
{
  "anomalies": [
    {
      "log_id": 10042,
      "action": "user.login",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "ip_address": "192.168.1.100",
      "anomaly_type": "suspicious_login",
      "ai_risk_score": 0.87
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `anomalies` | `array` | List of detected security anomalies |
| `anomalies[].log_id` | `integer` | Audit log entry ID |
| `anomalies[].action` | `string` | Action that triggered the anomaly |
| `anomalies[].user_id` | `string` (UUID or null) | User who performed the action |
| `anomalies[].ip_address` | `string` (or null) | IP address associated with the action |
| `anomalies[].anomaly_type` | `string` | Anomaly classification |
| `anomalies[].ai_risk_score` | `float` | AI risk score from 0.0 to 1.0 |

### Anomaly types

The AI system classifies anomalies into these categories based on the action type:

| Action contains | Anomaly type | Typical risk range |
|---|---|---|
| `login`, `signin` | `suspicious_login` | 0.75–0.95 |
| `delete`, `remove` | `unauthorized_delete_attempt` | 0.85–1.00 |
| `export`, `download` | `data_exfiltration` | 0.70–0.95 |
| `permission`, `role` | `privilege_escalation` | 0.80–1.00 |
| `apikey`, `api_key` | `api_key_abuse` | 0.65–0.95 |
| anything else | `anomalous_activity` | 0.60–0.90 |

### Severity thresholds

| Severity | Risk score threshold |
|---|---|
| `critical` | ≥ 0.9 |
| `high` | ≥ 0.7 |
| `medium` | ≥ 0.4 |
| `low` | All results |

### Errors

| Status | Description |
|---|---|
| `400` | Invalid `limit` parameter |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `security.audit.read` permission |
| `500` | Internal server error |

---

## AI simulation notes

All AI features in Category 5 are simulated with heuristic logic:

- **Text-to-SQL:** Prompt classification uses keyword matching; generated SQL is pre-written for each category
- **Chart recommendation:** Based on keyword presence (`top`/`by` → bar, date columns → line, otherwise → table)
- **Workflow automation:** Steps are simulated with randomized counts (2–6 per workflow); no actual webhooks or actions are executed
- **Audit anomalies:** Anomaly types and risk scores are generated from action name heuristics; no real ML model evaluates the audit logs

In production, these would be replaced with an LLM for text-to-SQL, an automation engine (e.g., Temporal), and a real anomaly detection model.
