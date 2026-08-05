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

## BI Dashboards

### POST /api/v1/bi/dashboards

Creates a new BI dashboard with configurable widgets and data sources.

**Permission:** `bi.dashboards.write`

#### Request

```json
{
  "name": "Q2 Revenue Overview",
  "widgets": [
    {
      "type": "bar_chart",
      "data_source": "crm_deals",
      "title": "Revenue by Rep"
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | `string` | ✅ | Dashboard name |
| `widgets` | `array` | — | Array of widget configurations |
| `widgets[].type` | `string` | ✅ | Widget type (`bar_chart`, `line_chart`, `table`, `kpi`) |
| `widgets[].data_source` | `string` | ✅ | Data source table name |
| `widgets[].title` | `string` | ✅ | Widget display title |

#### Response `201 Created`

```json
{
  "dashboard_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Q2 Revenue Overview",
  "created_at": "2026-01-15T10:00:00Z"
}
```

| Field | Type | Description |
|---|---|---|
| `dashboard_id` | `string` (UUID) | ID of the created dashboard |
| `name` | `string` | Dashboard name |
| `created_at` | `string` (ISO 8601) | Creation timestamp |

#### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body or missing `name` |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `bi.dashboards.write` permission |
| `500` | Internal server error |

---

### GET /api/v1/bi/dashboards

Lists all BI dashboards for the organization.

**Permission:** `bi.dashboards.read`

#### Response `200 OK`

```json
{
  "dashboards": [
    {
      "dashboard_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Q2 Revenue Overview",
      "widget_count": 3,
      "created_at": "2026-01-15T10:00:00Z"
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `dashboards` | `array` | Array of dashboard summary objects |
| `dashboards[].dashboard_id` | `string` (UUID) | Dashboard ID |
| `dashboards[].name` | `string` | Dashboard name |
| `dashboards[].widget_count` | `integer` | Number of widgets on the dashboard |
| `dashboards[].created_at` | `string` (ISO 8601) | Creation timestamp |

#### Errors

| Status | Description |
|---|---|
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `bi.dashboards.read` permission |
| `500` | Internal server error |

---

### GET /api/v1/bi/dashboards/{dashboard_id}/data

Retrieves dashboard data with AI-powered anomaly detection. Each widget's data is analyzed for statistical anomalies and trends.

**Permission:** `bi.dashboards.read`

#### Path parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `dashboard_id` | `string` (UUID) | ✅ | ID of the dashboard to retrieve data for |

#### Response `200 OK`

```json
{
  "dashboard_id": "550e8400-e29b-41d4-a716-446655440000",
  "widgets": [
    {
      "widget_id": "660e8400-e29b-41d4-a716-446655440000",
      "title": "Revenue by Rep",
      "data": [
        {"sales_rep": "Alice Johnson", "total_revenue": 245000.50}
      ],
      "ai_anomalies": [
        {
          "description": "Revenue spike detected for Alice Johnson (+35% vs avg)",
          "severity": "medium"
        }
      ]
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `dashboard_id` | `string` (UUID) | Dashboard ID |
| `widgets` | `array` | Array of widget data objects |
| `widgets[].widget_id` | `string` (UUID) | Widget ID |
| `widgets[].title` | `string` | Widget title |
| `widgets[].data` | `array` | Widget data rows |
| `widgets[].ai_anomalies` | `array` | AI-detected anomalies in the widget data |
| `widgets[].ai_anomalies[].description` | `string` | Anomaly description |
| `widgets[].ai_anomalies[].severity` | `string` | Severity: `low`, `medium`, `high`, `critical` |

#### Errors

| Status | Description |
|---|---|
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `bi.dashboards.read` permission |
| `404` | Dashboard not found |
| `500` | Internal server error |

---

## IoT Gateway

### POST /api/v1/iot/devices

Registers a new IoT device in the organization's device fleet. Devices are identified by a unique device name and can optionally include a MAC address.

**Permission:** `iot.devices.write`

#### Request

```json
{
  "device_name": "Temperature Sensor A1",
  "device_type": "temperature_sensor",
  "mac_address": "AA:BB:CC:DD:EE:01"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `device_name` | `string` | ✅ | Unique device name |
| `device_type` | `string` | ✅ | Device type (e.g., `temperature_sensor`, `humidity_sensor`, `vibration_sensor`) |
| `mac_address` | `string` | — | Device MAC address (optional, unique) |

#### Response `201 Created`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "org_id": "660e8400-e29b-41d4-a716-446655440000",
  "device_name": "Temperature Sensor A1",
  "device_type": "temperature_sensor",
  "mac_address": "AA:BB:CC:DD:EE:01",
  "status": "online",
  "last_ping": "2026-01-15T14:30:00Z"
}
```

| Field | Type | Description |
|---|---|---|
| `id` | `string` (UUID) | ID of the registered device |
| `org_id` | `string` (UUID) | Organization ID |
| `device_name` | `string` | Device name |
| `device_type` | `string` | Device type |
| `mac_address` | `string` (or null) | Device MAC address |
| `status` | `string` | Device status — `"online"` on creation |
| `last_ping` | `string` (ISO 8601) | Last ping timestamp |

#### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body, missing `device_name` or `device_type` |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `iot.devices.write` permission |
| `409` | Device with this MAC address already exists |
| `500` | Internal server error |

---

### GET /api/v1/iot/devices

Lists all registered IoT devices for the organization.

**Permission:** `iot.devices.write`

#### Response `200 OK`

```json
{
  "devices": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "org_id": "660e8400-e29b-41d4-a716-446655440000",
      "device_name": "Temperature Sensor A1",
      "device_type": "temperature_sensor",
      "mac_address": "AA:BB:CC:DD:EE:01",
      "status": "online",
      "last_ping": "2026-01-15T14:30:00Z"
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `devices` | `array` | Array of IoT device objects |
| `devices[].id` | `string` (UUID) | Device ID |
| `devices[].org_id` | `string` (UUID) | Organization ID |
| `devices[].device_name` | `string` | Device name |
| `devices[].device_type` | `string` | Device type |
| `devices[].mac_address` | `string` (or null) | Device MAC address |
| `devices[].status` | `string` | Current device status |
| `devices[].last_ping` | `string` (ISO 8601 or null) | Timestamp of the last ping |

#### Errors

| Status | Description |
|---|---|
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `iot.devices.write` permission |
| `500` | Internal server error |

---

### POST /api/v1/iot/readings

Ingests a sensor reading from an IoT device. Readings are persisted to the `iot_device_readings` table. The AI engine analyzes the reading for anomalies and triggers alerts when values exceed expected thresholds.

**Permission:** `iot.readings.ingest`

#### Request

```json
{
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "metric_name": "temperature",
  "metric_value": 42.5,
  "unit": "celsius"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `device_id` | `string` (UUID) | ✅ | ID of the registered IoT device |
| `metric_name` | `string` | ✅ | Metric name (e.g., `temperature`, `humidity`, `vibration`) |
| `metric_value` | `float` | ✅ | Numeric reading value |
| `unit` | `string` | — | Unit of measurement (e.g., `celsius`, `percent`, `hz`) |

#### Response `200 OK`

```json
{
  "status": "processed",
  "anomaly_detected": false,
  "anomaly_description": null
}
```

| Field | Type | Description |
|---|---|---|
| `status` | `string` | Processing status — `"processed"` on success |
| `anomaly_detected` | `boolean` | Whether the AI engine flagged this reading as anomalous |
| `anomaly_description` | `string` (or null) | Description of the anomaly if detected, otherwise `null` |

#### AI anomaly detection

The AI engine checks readings against expected ranges:

| Metric | Expected range | Anomaly if |
|---|---|---|
| `temperature` | 15–35°C | Outside range |
| `humidity` | 30–70% | Outside range |
| `vibration` | 0–50 Hz | > 50 Hz |

#### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body, missing required fields |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `iot.readings.ingest` permission |
| `404` | Device not found |
| `500` | Internal server error |

---

### POST /api/v1/iot/readings/batch

High-frequency batch ingestion endpoint for edge devices. Accepts an array of readings and processes them in bulk, returning accepted and rejected counts.

**Permission:** `iot.readings.ingest`

#### Request

```json
{
  "readings": [
    {
      "device_id": "550e8400-e29b-41d4-a716-446655440000",
      "metric_name": "temperature",
      "metric_value": 22.5,
      "unit": "celsius"
    },
    {
      "device_id": "550e8400-e29b-41d4-a716-446655440001",
      "metric_name": "humidity",
      "metric_value": 55.0,
      "unit": "percent"
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `readings` | `array` | ✅ | Array of reading objects (must not be empty) |
| `readings[].device_id` | `string` (UUID) | ✅ | ID of the registered IoT device |
| `readings[].metric_name` | `string` | ✅ | Metric name |
| `readings[].metric_value` | `float` | ✅ | Numeric reading value |
| `readings[].unit` | `string` | — | Unit of measurement |

#### Response `200 OK`

```json
{
  "accepted": 2,
  "rejected": 0
}
```

| Field | Type | Description |
|---|---|---|
| `accepted` | `integer` | Number of readings successfully ingested |
| `rejected` | `integer` | Number of readings rejected (e.g., unknown device) |

#### Errors

| Status | Description |
|---|---|
| `400` | Invalid request body, empty `readings` array, or missing required fields in a reading |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks `iot.readings.ingest` permission |
| `500` | Internal server error |

---

## AI simulation notes

All AI features in Category 5 are simulated with heuristic logic:

- **Text-to-SQL:** Prompt classification uses keyword matching; generated SQL is pre-written for each category
- **Chart recommendation:** Based on keyword presence (`top`/`by` → bar, date columns → line, otherwise → table)
- **Workflow automation:** Steps are simulated with randomized counts (2–6 per workflow); no actual webhooks or actions are executed
- **Audit anomalies:** Anomaly types and risk scores are generated from action name heuristics; no real ML model evaluates the audit logs

In production, these would be replaced with an LLM for text-to-SQL, an automation engine (e.g., Temporal), and a real anomaly detection model.
