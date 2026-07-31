# Billing

## Overview

The Billing module manages subscriptions for organizations. It integrates with Stripe for payment processing. Currently, it supports creating or upgrading subscriptions.

### Subscription model

```go
type Subscription struct {
    ID               uuid.UUID
    OrgID            uuid.UUID
    PlanCode         string             // e.g., "starter", "professional", "enterprise"
    Status           SubscriptionStatus
    StripeCustomerID *string            // Stripe customer ID (sensitive)
    StripeSubID      *string            // Stripe subscription ID (sensitive)
    CurrentPeriodEnd *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### Subscription statuses

| Status | Description |
|---|---|
| `active` | Subscription is active and in good standing |
| `past_due` | Payment failed, subscription is past due |
| `canceled` | Subscription has been canceled |
| `trialing` | Subscription is in trial period |

---

## Endpoints

### Create or upgrade a subscription

Creates a new subscription or upgrades an existing one for the organization. Requires `billing.manage` permission.

```http
POST /api/v1/billing/subscriptions
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "plan_code": "professional",
    "payment_method_id": "pm_1234567890"
}
```

**Response** `200 OK`:
```json
{
    "subscription_id": "550e8400-e29b-41d4-a716-446655440060",
    "status": "active",
    "current_period_end": "2026-09-01T12:00:00Z"
}
```

**Behavior:**
1. If the organization already has a subscription, it is upgraded to the new plan
2. If no subscription exists, a new one is created with status `active`
3. The `payment_method_id` is accepted but not yet processed — Stripe integration is pending

**Validation:**
| Field | Rule |
|---|---|
| `plan_code` | Required, non-blank |
| `payment_method_id` | Required, non-blank |

**Errors:**
| Status | Condition |
|---|---|
| `400` | Missing plan_code or payment_method_id |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Internal server error |

## Future enhancements

- **Stripe webhook handling** — sync subscription status changes from Stripe
- **Plan listing** — GET endpoint to list available plans and pricing
- **Invoice history** — GET endpoint to view past invoices
- **Payment method management** — add/update/remove payment methods
- **Cancel subscription** — endpoint to cancel at period end
- **Trial management** — automatic trial period handling