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

---

## Manufacturing

### Create bill of materials

Creates a bill of materials (BOM) for a manufactured product, defining the raw materials, components, and quantities required.

```http
POST /api/v1/manufacturing/boms
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "product_name": "Widget Pro",
    "components": [
        {
            "material_name": "Aluminum Sheet",
            "quantity": 2.5,
            "unit": "kg"
        },
        {
            "material_name": "M4 Screws",
            "quantity": 12,
            "unit": "pcs"
        }
    ]
}
```

**Response** `201 Created`:
```json
{
    "bom_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_name": "Widget Pro",
    "component_count": 2
}
```

**Validation:**

| Field | Rule |
|---|---|
| `product_name` | Required, non-blank string |
| `components` | Required, array with at least one component |
| `components[].material_name` | Required, non-blank string |
| `components[].quantity` | Required, positive decimal |
| `components[].unit` | Required, non-blank string (e.g., `kg`, `pcs`, `m`) |

**Permissions:** Requires `manufacturing.boms.write` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, empty components array, invalid quantities |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Internal server error |

---

### Create work order with AI bottleneck prediction

Creates a manufacturing work order and runs AI bottleneck prediction to identify potential production delays before work begins.

```http
POST /api/v1/manufacturing/work-orders
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "bom_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity": 500,
    "scheduled_start": "2026-02-01",
    "priority": "high"
}
```

**Response** `201 Created`:
```json
{
    "work_order_id": "660e8400-e29b-41d4-a716-446655440000",
    "status": "scheduled",
    "ai_bottleneck_prediction": {
        "bottleneck_risk": "medium",
        "predicted_delay_days": 3,
        "affected_resource": "Aluminum Sheet"
    }
}
```

**Validation:**

| Field | Rule |
|---|---|
| `bom_id` | Required, valid UUID, must reference an existing BOM |
| `quantity` | Required, positive integer |
| `scheduled_start` | Required, valid date in `YYYY-MM-DD` format |
| `priority` | Required, one of `low`, `medium`, `high`, `critical` |

**AI bottleneck prediction:**

The AI engine simulates bottleneck analysis by evaluating component availability against the requested quantity. Predictions include a risk level, estimated delay in days, and the affected resource.

**Permissions:** Requires `manufacturing.workorders.write` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, invalid BOM ID, invalid priority |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `404` | BOM not found |
| `500` | Internal server error |

---

## Procurement

### Create purchase order with AI supplier risk rating

Creates a purchase order and runs AI supplier risk analysis, returning a risk rating for the supplier.

```http
POST /api/v1/procurement/purchase-orders
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "supplier_name": "Acme Supplies Inc.",
    "items": [
        {
            "description": "Aluminum Sheet",
            "quantity": 100,
            "unit_price": 12.50
        }
    ],
    "delivery_date": "2026-02-15"
}
```

**Response** `201 Created`:
```json
{
    "po_id": "770e8400-e29b-41d4-a716-446655440000",
    "total_amount": 1250.00,
    "ai_supplier_risk_rating": "low",
    "ai_risk_factors": [
        "Consistent on-time delivery history",
        "No recent quality incidents"
    ]
}
```

**Validation:**

| Field | Rule |
|---|---|
| `supplier_name` | Required, non-blank string |
| `items` | Required, array with at least one item |
| `items[].description` | Required, non-blank string |
| `items[].quantity` | Required, positive integer |
| `items[].unit_price` | Required, positive decimal |
| `delivery_date` | Required, valid date in `YYYY-MM-DD` format |

**AI supplier risk rating:**

The AI engine evaluates the supplier based on historical performance data and returns a risk rating (`low`, `medium`, `high`, or `critical`) along with contributing factors.

**Permissions:** Requires `procurement.po.write` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, empty items array, invalid quantities/prices |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Internal server error |

---

### Get supplier risk report

Retrieves a detailed AI-generated risk report for a supplier, including historical performance metrics and risk factors.

```http
GET /api/v1/procurement/supplier-risk?supplier_name=Acme
Content-Type: application/json
Authorization: Bearer <jwt>
```

**Query parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `supplier_name` | `string` | ✅ | Supplier name to look up (partial match supported) |

**Response** `200 OK`:
```json
{
    "supplier_name": "Acme Supplies Inc.",
    "overall_risk": "low",
    "metrics": {
        "on_time_delivery_pct": 94.5,
        "quality_incident_count": 2,
        "avg_response_time_hours": 4.2
    },
    "risk_factors": [
        "Consistent on-time delivery history",
        "No recent quality incidents"
    ],
    "recommendation": "Approved supplier — low risk"
}
```

| Field | Type | Description |
|---|---|---|
| `supplier_name` | `string` | Full supplier name |
| `overall_risk` | `string` | Overall risk rating: `low`, `medium`, `high`, or `critical` |
| `metrics` | `object` | Key performance metrics |
| `metrics.on_time_delivery_pct` | `float` | Percentage of on-time deliveries |
| `metrics.quality_incident_count` | `integer` | Number of quality incidents in the last 12 months |
| `metrics.avg_response_time_hours` | `float` | Average supplier response time in hours |
| `risk_factors` | `array` | List of identified risk factors |
| `recommendation` | `string` | AI-generated recommendation |

**Permissions:** Requires `procurement.supplier.read` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing `supplier_name` query parameter |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `404` | Supplier not found |
| `500` | Internal server error |
