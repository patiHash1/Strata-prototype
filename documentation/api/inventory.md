# Supply Chain & Inventory

## Overview

The Inventory module provides CRUD operations on stock levels (receive, issue, transfer, snapshot) and AI-driven reorder predictions with stockout forecasting. Products are managed per organization and predictions can be optionally filtered by warehouse.

---

## Endpoints

### 1. Receive Stock

```http
POST /api/v1/inventory/receive
Authorization: Bearer <jwt>
```

Records stock received into a warehouse, increasing the available quantity for a product.

**Required permission:** `inventory.receive`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | **Yes** | ID of the receiving warehouse |
| `product_id` | UUID | **Yes** | ID of the product being received |
| `quantity` | integer | **Yes** | Quantity received (must be > 0) |
| `reference` | string | No | Receipt reference (e.g., PO number, delivery note) |

**Example request:**

```json
{
    "warehouse_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_id": "550e8400-e29b-41d4-a716-446655440010",
    "quantity": 50,
    "reference": "PO-2026-0042"
}
```

**Response** `201 Created`:

```json
{
    "inventory_level_id": "dd0e8400-e29b-41d4-a716-446655440000",
    "quantity_available": 150,
    "quantity_reserved": 0
}
```

| Field | Type | Description |
|---|---|---|
| `inventory_level_id` | UUID | ID of the updated inventory level record |
| `quantity_available` | integer | Total available quantity after receiving |
| `quantity_reserved` | integer | Currently reserved quantity |

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, quantity ≤ 0, invalid UUIDs |
| `401` | Missing or invalid JWT |
| `403` | Token lacks `inventory.receive` permission |
| `404` | Warehouse or product not found |
| `500` | Internal server error |

---

### 2. Issue Stock

```http
POST /api/v1/inventory/issue
Authorization: Bearer <jwt>
```

Issues stock from a warehouse, decreasing the available quantity. Returns an error if insufficient stock is available.

**Required permission:** `inventory.issue`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | **Yes** | ID of the issuing warehouse |
| `product_id` | UUID | **Yes** | ID of the product being issued |
| `quantity` | integer | **Yes** | Quantity to issue (must be > 0) |
| `reference` | string | No | Issue reference (e.g., work order, shipment ID) |

**Example request:**

```json
{
    "warehouse_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_id": "550e8400-e29b-41d4-a716-446655440010",
    "quantity": 10,
    "reference": "WO-2026-0105"
}
```

**Response** `201 Created`:

```json
{
    "inventory_level_id": "dd0e8400-e29b-41d4-a716-446655440000",
    "quantity_available": 140,
    "quantity_reserved": 0
}
```

| Field | Type | Description |
|---|---|---|
| `inventory_level_id` | UUID | ID of the updated inventory level record |
| `quantity_available` | integer | Remaining available quantity after issuing |
| `quantity_reserved` | integer | Currently reserved quantity |

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, quantity ≤ 0, invalid UUIDs, insufficient stock |
| `401` | Missing or invalid JWT |
| `403` | Token lacks `inventory.issue` permission |
| `404` | Warehouse or product not found |
| `500` | Internal server error |

---

### 3. Transfer Stock

```http
POST /api/v1/inventory/transfer
Authorization: Bearer <jwt>
```

Transfers stock between two warehouses, decreasing available quantity at the source and increasing it at the destination.

**Required permission:** `inventory.transfer`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `from_warehouse_id` | UUID | **Yes** | Source warehouse ID |
| `to_warehouse_id` | UUID | **Yes** | Destination warehouse ID |
| `product_id` | UUID | **Yes** | ID of the product to transfer |
| `quantity` | integer | **Yes** | Quantity to transfer (must be > 0) |

**Example request:**

```json
{
    "from_warehouse_id": "550e8400-e29b-41d4-a716-446655440000",
    "to_warehouse_id": "550e8400-e29b-41d4-a716-446655440001",
    "product_id": "550e8400-e29b-41d4-a716-446655440010",
    "quantity": 25
}
```

**Response** `201 Created`:

```json
{
    "from_warehouse": {
        "warehouse_id": "550e8400-e29b-41d4-a716-446655440000",
        "quantity_available": 115,
        "quantity_reserved": 0
    },
    "to_warehouse": {
        "warehouse_id": "550e8400-e29b-41d4-a716-446655440001",
        "quantity_available": 75,
        "quantity_reserved": 0
    }
}
```

| Field | Type | Description |
|---|---|---|
| `from_warehouse.warehouse_id` | UUID | Source warehouse ID |
| `from_warehouse.quantity_available` | integer | Remaining available quantity at source |
| `from_warehouse.quantity_reserved` | integer | Reserved quantity at source |
| `to_warehouse.warehouse_id` | UUID | Destination warehouse ID |
| `to_warehouse.quantity_available` | integer | New available quantity at destination |
| `to_warehouse.quantity_reserved` | integer | Reserved quantity at destination |

**Errors:**

| Status | Condition |
|---|---|
| `400` | Missing required fields, quantity ≤ 0, invalid UUIDs, insufficient stock at source, same warehouse for source and destination |
| `401` | Missing or invalid JWT |
| `403` | Token lacks `inventory.transfer` permission |
| `404` | Warehouse or product not found |
| `500` | Internal server error |

---

### 4. Get Inventory Snapshot

```http
GET /api/v1/inventory/snapshot?warehouse_id=550e8400-e29b-41d4-a716-446655440000
Authorization: Bearer <jwt>
```

Returns a snapshot of current inventory levels, optionally filtered by warehouse.

**Required permission:** `inventory.snapshot`

**Query Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | No | Filter inventory levels by warehouse |

**Response** `200 OK`:

```json
{
    "warehouse_id": "550e8400-e29b-41d4-a716-446655440000",
    "levels": [
        {
            "product_id": "550e8400-e29b-41d4-a716-446655440010",
            "sku": "WGT-001",
            "quantity_available": 150,
            "quantity_reserved": 5
        },
        {
            "product_id": "550e8400-e29b-41d4-a716-446655440020",
            "sku": "WGT-002",
            "quantity_available": 200,
            "quantity_reserved": 30
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `warehouse_id` | UUID | Warehouse ID (null if not filtered) |
| `levels` | array | Array of inventory level records |
| `levels[].product_id` | UUID | Product identifier |
| `levels[].sku` | string | Stock keeping unit code |
| `levels[].quantity_available` | integer | Available (unreserved) quantity |
| `levels[].quantity_reserved` | integer | Reserved quantity |

**Errors:**

| Status | Condition |
|---|---|
| `400` | Invalid warehouse_id UUID |
| `401` | Missing or invalid JWT |
| `403` | Token lacks `inventory.snapshot` permission |
| `500` | Internal server error |

---

### 5. Get AI Reorder & Stockout Predictions

Returns AI-driven reorder predictions and stockout forecasts for all products in the organization. Uses actual inventory levels from the database (not simulated). Optionally filtered by warehouse, which now queries real `inventory_levels` data.

```http
GET /api/v1/inventory/reorder-predictions?warehouse_id=550e8400-e29b-41d4-a716-446655440000
Authorization: Bearer <jwt>
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | No | Filter predictions by warehouse |

**Response** `200 OK`:
```json
{
    "predictions": [
        {
            "product_id": "550e8400-e29b-41d4-a716-446655440010",
            "sku": "WGT-001",
            "current_stock": 5,
            "predicted_stockout_days": 3,
            "recommended_reorder_qty": 18
        },
        {
            "product_id": "550e8400-e29b-41d4-a716-446655440020",
            "sku": "WGT-002",
            "current_stock": 12,
            "predicted_stockout_days": 15,
            "recommended_reorder_qty": 20
        }
    ]
}
```

**Response fields:**

| Field | Type | Description |
|---|---|---|
| `product_id` | UUID | Product identifier |
| `sku` | string | Stock keeping unit code |
| `current_stock` | integer | Current inventory level from the database |
| `predicted_stockout_days` | integer | Estimated days until stock depletes (capped at 60) |
| `recommended_reorder_qty` | integer | AI-recommended quantity to reorder (includes safety buffer) |

**AI prediction logic:**

- Current stock is read from actual `inventory_levels` data (not simulated)
- Stockout days are calculated from a simulated daily consumption rate
- Reorder quantity includes a 20–50% safety buffer over the stockout shortfall

**Permissions:** Requires `inventory.read` permission.

**Errors:**

| Status | Condition |
|---|---|
| `400` | Invalid warehouse_id UUID |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Internal server error |