# Fleet & Telematics

## Overview

The Fleet module manages vehicle fleets, telemetry data ingestion, and AI-powered route optimization. Telemetry data is ingested via API key authentication, while route optimization uses standard Bearer token auth.

---

## Endpoints

### Ingest vehicle telemetry

Ingests real-time vehicle telemetry data (GPS, speed, engine temperature). Triggers AI predictive maintenance alerts when anomalies are detected. Authenticated via **API key** (`X-API-Key` header).

```http
POST /api/v1/fleet/telematics/ingest
Content-Type: application/json
X-API-Key: <api_key>

{
    "vehicle_vin": "1HGBH41JXMN109186",
    "latitude": 40.7128,
    "longitude": -74.0060,
    "speed_kmh": 85.5,
    "engine_temp_c": 95.2
}
```

**Response** `202 Accepted`:
```json
{
    "status": "queued",
    "processed_at": "2025-01-15T14:30:00Z",
    "ai_predictive_alert_triggered": false
}
```

**Validation:**

| Field | Rule |
|---|---|
| `vehicle_vin` | Required, non-blank, must match an existing fleet vehicle in the org |
| `latitude` | Required, decimal |
| `longitude` | Required, decimal |
| `speed_kmh` | Required, non-negative decimal |
| `engine_temp_c` | Optional, decimal — triggers alert if > 110°C |

**AI predictive alerting:**

- **High engine temperature** (> 110°C): triggers an alert
- **High speed** (> 130 km/h): triggers an alert
- Fuel level is estimated from engine temperature patterns

**Authentication:**

This endpoint uses **API key authentication** via the `X-API-Key` header. The key must have the `fleet.telematics.ingest` scope. API keys are created via `POST /api/v1/org/api-keys` with the appropriate scope.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields or invalid values |
| `401` | Missing, invalid, or expired API key |
| `403` | API key lacks required scopes |
| `404` | Vehicle VIN not found |
| `500` | Internal server error |

---

### Generate AI optimized routes

Generates AI-optimized delivery routes for a set of shipments using available vehicles. Returns a route plan with GeoJSON waypoints, predicted ETA, and carbon offset estimate.

```http
POST /api/v1/fleet/routes/optimize
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "shipment_ids": [
        "550e8400-e29b-41d4-a716-446655440001",
        "550e8400-e29b-41d4-a716-446655440002"
    ],
    "available_vehicle_ids": [
        "550e8400-e29b-41d4-a716-446655440010"
    ]
}
```

**Response** `200 OK`:
```json
{
    "route_plan_id": "550e8400-e29b-41d4-a716-446655440100",
    "optimized_waypoints": [
        {
            "type": "Point",
            "coordinates": [-74.0, 40.72]
        },
        {
            "type": "Point",
            "coordinates": [-73.98, 40.74]
        },
        {
            "type": "Point",
            "coordinates": [-73.9, 40.82]
        }
    ],
    "predicted_eta": "2025-01-15T16:45:00Z",
    "carbon_offset_kg": 12.45
}
```

**Validation:**

| Field | Rule |
|---|---|
| `shipment_ids` | Required, array of UUIDs, at least one ID |
| `available_vehicle_ids` | Required, array of UUIDs, at least one ID |

**Behavior:**

- All shipments are validated to exist and belong to the requesting organization
- All vehicles are validated similarly
- AI simulation generates a GeoJSON path with waypoints and a predicted ETA (30–150 minutes from now)
- Carbon offset is estimated based on shipment count

**Permissions:** Requires `fleet.routes.manage` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing shipment_ids or vehicle_ids, invalid UUIDs |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `404` | Shipments or vehicles not found |
| `500` | Internal server error |
