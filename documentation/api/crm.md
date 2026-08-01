# CRM & Revenue Operations

The CRM module provides endpoints for managing leads, deals, quotes, and AI-powered risk analysis and scoring.

## Base path

All CRM endpoints are prefixed with `/api/v1/crm/`.

## Authentication

All CRM endpoints require a **Bearer Token** in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

The token must contain the appropriate CRM permission scopes.

## Endpoints

### 1. Create Lead & Trigger AI Scoring

```
POST /api/v1/crm/leads
```

Creates a new CRM contact as a lead, runs AI win probability scoring, and creates a linked deal.

**Required permission:** `crm.leads.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `first_name` | string | **Yes** | Lead's first name |
| `last_name` | string | No | Lead's last name |
| `email` | string | **Yes** | Lead's email address |
| `company_name` | string | No | Company the lead represents |
| `estimated_deal_size` | decimal | No | Estimated deal value |

**Example request:**

```json
{
    "first_name": "Jane",
    "last_name": "Smith",
    "email": "jane.smith@acmecorp.com",
    "company_name": "Acme Corp",
    "estimated_deal_size": 50000.00
}
```

**Response (201 Created):**

```json
{
    "contact_id": "550e8400-e29b-41d4-a716-446655440000",
    "ai_win_probability": 65,
    "assigned_to": "550e8400-e29b-41d4-a716-446655440001"
}
```

| Field | Type | Description |
|---|---|---|
| `contact_id` | UUID | ID of the created CRM contact |
| `ai_win_probability` | integer | AI-predicted win likelihood (0–100) |
| `assigned_to` | UUID | User ID the lead is assigned to |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid email |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `crm.leads.write` permission |

---

### 2. Analyze Contract Risk

```
POST /api/v1/crm/quotes/risk-analysis
```

Runs AI risk analysis on a quote's contract text, identifying flagged clauses with risk levels and suggested fixes. The risk score is persisted to the quote record.

**Required permission:** `crm.quotes.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `quote_id` | UUID | **Yes** | ID of the quote to analyze |
| `contract_text` | string | **Yes** | Full contract text to analyze |

**Example request:**

```json
{
    "quote_id": "660e8400-e29b-41d4-a716-446655440000",
    "contract_text": "This agreement includes an unlimited indemnification clause..."
}
```

**Response (200 OK):**

```json
{
    "ai_risk_score": 35.5,
    "flagged_clauses": [
        {
            "clause": "Unlimited indemnification clause detected",
            "risk_level": "high",
            "suggested_fix": "Cap indemnification liability to the total contract value"
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `ai_risk_score` | decimal | AI risk score (0–100), higher = riskier |
| `flagged_clauses` | array | Array of flagged clause objects |
| `flagged_clauses[].clause` | string | Description of the risky clause |
| `flagged_clauses[].risk_level` | string | Risk level: `low`, `medium`, `high`, or `critical` |
| `flagged_clauses[].suggested_fix` | string | Recommended action to mitigate the risk |

**Risk levels:**

| Level | Weight | Description |
|---|---|---|
| `critical` | +30 pts | Terms that pose existential risk to the deal or business |
| `high` | +20 pts | Significant risk requiring immediate attention |
| `medium` | +10 pts | Moderate risk, should be addressed before signing |
| `low` | +5 pts | Minor concerns, acceptable with minor revisions |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid quote_id |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `crm.quotes.write` permission |
| `404 Not Found` | Quote not found or not in the user's organization |

---

## AI engine (simulated)

The current implementation uses a keyword-based heuristic analysis engine. In production, this would be replaced with an external AI/ML service call. The simulation provides realistic output to verify request/response contracts during development.

### Win probability scoring (Module 1.1)

Returns a random value in the 40–80 range, simulating a reasonable lead quality score.

### Contract risk analysis (Module 1.2)

Scans contract text for high-risk keywords and phrases:

| Keyword | Risk Level | Mitigation |
|---|---|---|
| `indemnification` | high | Cap liability to contract value |
| `penalty` | critical | Negotiate mutual or capped terms |
| `termination` | medium | Add mutual clause with notice period |
| `confidential` | low | Limit duration to 3 years post-termination |

Clean contracts (no flagged keywords) receive a baseline risk score of 5–15.