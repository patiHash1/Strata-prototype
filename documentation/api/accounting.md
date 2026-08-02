# Finance & Enterprise Accounting

The Accounting module provides endpoints for general ledger entries, invoice OCR processing, and AI-audited expense submissions.

## Base path

All accounting endpoints are prefixed with `/api/v1/accounting/`.

## Authentication

All accounting endpoints require a **Bearer Token** in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

The token must contain the appropriate accounting permission scopes.

## Endpoints

### 1. Post General Ledger Entry

```
POST /api/v1/accounting/journal-entries
```

Creates a journal entry with balanced debit/credit items. Total debits must equal total credits.

**Required permission:** `accounting.ledger.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `entry_date` | string | **Yes** | Date in `YYYY-MM-DD` format |
| `memo` | string | No | Optional memo/description |
| `items` | array | **Yes** | Array of journal line items |
| `items[].account_id` | UUID | **Yes** | Chart of accounts ID |
| `items[].debit` | decimal | No | Debit amount (default 0.00) |
| `items[].credit` | decimal | No | Credit amount (default 0.00) |

**Example request:**

```json
{
    "entry_date": "2026-01-15",
    "memo": "January office rent payment",
    "items": [
        {
            "account_id": "550e8400-e29b-41d4-a716-446655440000",
            "debit": 2500.00,
            "credit": 0.00
        },
        {
            "account_id": "550e8400-e29b-41d4-a716-446655440001",
            "debit": 0.00,
            "credit": 2500.00
        }
    ]
}
```

**Response (201 Created):**

```json
{
    "journal_entry_id": "660e8400-e29b-41d4-a716-446655440000",
    "entry_number": "JE-20260115-0042"
}
```

| Field | Type | Description |
|---|---|---|
| `journal_entry_id` | UUID | ID of the created journal entry |
| `entry_number` | string | Auto-generated entry number (format: `JE-YYYYMMDD-XXXX`) |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid date format, unbalanced debits/credits, no items |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.ledger.write` permission |
| `404 Not Found` | Account not found or not in the organization |

---

### 2. Upload Invoice PDF for Vision OCR

```
POST /api/v1/accounting/invoices/ocr
```

Uploads a PDF or image invoice file for AI-powered OCR extraction. Returns extracted vendor information, line items, tax, and total amounts. The invoice is also saved to the database.

**Required permission:** `accounting.invoices.write`

**Request:** Multipart form data

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | file | **Yes** | Invoice file (PDF or image, max 10 MB) |

**Example request (curl):**

```bash
curl -X POST http://localhost:8080/api/v1/accounting/invoices/ocr \
  -H "Authorization: Bearer <token>" \
  -F "file=@invoice.pdf"
```

**Response (200 OK):**

```json
{
    "vendor_name": "Acme Supplies Inc.",
    "invoice_number": "INV-4521",
    "line_items": [
        {
            "description": "Professional Services",
            "quantity": 3,
            "unit_price": 150.00,
            "total": 450.00
        },
        {
            "description": "Cloud Storage",
            "quantity": 1,
            "unit_price": 75.50,
            "total": 75.50
        }
    ],
    "tax_amount": 45.60,
    "total_amount": 571.10
}
```

| Field | Type | Description |
|---|---|---|
| `vendor_name` | string | Extracted vendor/supplier name |
| `invoice_number` | string | Extracted invoice number |
| `line_items` | array | Array of line items |
| `line_items[].description` | string | Item description |
| `line_items[].quantity` | integer | Quantity |
| `line_items[].unit_price` | decimal | Price per unit |
| `line_items[].total` | decimal | Line total (quantity × unit_price) |
| `tax_amount` | decimal | Extracted tax amount |
| `total_amount` | decimal | Extracted invoice total |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing file, invalid multipart form |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.invoices.write` permission |

---

### 3. Submit Expense with AI Fraud Audit

```
POST /api/v1/accounting/expenses
```

Creates an expense submission and runs AI fraud detection. Flags high-value expenses, policy violations, duplicate receipts, and suspicious submission patterns.

**Required permission:** `expenses.submit`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `amount` | decimal | **Yes** | Expense amount (must be > 0) |
| `category` | string | **Yes** | Expense category (e.g. `travel`, `entertainment`, `office`) |
| `receipt_file` | string | No | Receipt filename for audit |

**Example request:**

```json
{
    "amount": 1500.00,
    "category": "entertainment",
    "receipt_file": "dinner-receipt.pdf"
}
```

**Response (201 Created):**

```json
{
    "expense_id": "770e8400-e29b-41d4-a716-446655440000",
    "ai_fraud_flag": true,
    "ai_audit_notes": "Entertainment expense exceeds $1,000 threshold — policy limit is $1,000"
}
```

| Field | Type | Description |
|---|---|---|
| `expense_id` | UUID | ID of the created expense |
| `ai_fraud_flag` | boolean | Whether the expense was flagged by AI fraud detection |
| `ai_audit_notes` | string | Explanation of policy violations (if any) or compliance confirmation |

**Fraud detection rules:**

| Rule | Condition |
|---|---|
| High-value expense | Amount > $5,000 — requires manager approval |
| Entertainment limit | Category is `entertainment` and amount > $1,000 |
| Travel limit | Category is `travel` and amount > $3,000 — requires itinerary |
| Duplicate receipt | Receipt filename contains `dup` or `copy` |
| Weekend submission | Expense submitted on Saturday or Sunday |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, amount ≤ 0 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `expenses.submit` permission |

---

## AI engine (simulated)

The current implementation uses heuristic-based simulation. In production, these would be replaced with external AI/ML service calls.

### OCR processing (Module 2.2)

Generates realistic vendor names, invoice numbers, line items with descriptions, quantities, unit prices, and calculated tax/totals. Each call produces randomized but structurally valid output.

### Fraud audit (Module 2.3)

Applies five rule-based checks against the expense submission:
- Amount thresholds by category
- Duplicate receipt filename patterns
- Weekend submission timing
- High-value expense flagging

Flagged expenses return `ai_fraud_flag: true` with a semicolon-delimited list of violations in `ai_audit_notes`. Clean expenses return `ai_fraud_flag: false` with a compliance confirmation message.