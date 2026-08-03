# Supply Chain & Inventory

## Overview

The Inventory module provides AI-driven reorder predictions and stockout forecasting. Products are managed per organization and predictions can be optionally filtered by warehouse.

---

## Endpoints

### Get AI reorder & stockout predictions

Returns AI-driven reorder predictions and stockout forecasts for all products in the organization. Optionally filtered by warehouse.

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
| `current_stock` | integer | Simulated current inventory level |
| `predicted_stockout_days` | integer | Estimated days until stock depletes (capped at 60) |
| `recommended_reorder_qty` | integer | AI-recommended quantity to reorder (includes safety buffer) |

**AI prediction logic:**

- Current stock is simulated between 50–100% of the product's `ai_reorder_point`
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
