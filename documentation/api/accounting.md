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

### 4. Register Fixed Asset

```
POST /api/v1/accounting/assets
```

Registers a new fixed asset for depreciation tracking. The asset is recorded with its purchase cost, salvage value, and useful life.

**Required permission:** `accounting.assets.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `asset_name` | string | **Yes** | Name/description of the asset |
| `purchase_date` | string | **Yes** | Purchase date in `YYYY-MM-DD` format |
| `purchase_cost` | decimal | **Yes** | Original purchase cost (must be > 0) |
| `salvage_value` | decimal | **Yes** | Estimated salvage value at end of life (must be ≥ 0) |
| `useful_life_years` | integer | **Yes** | Useful life in years (must be > 0) |

**Example request:**

```json
{
    "asset_name": "CNC Milling Machine",
    "purchase_date": "2025-06-01",
    "purchase_cost": 50000.00,
    "salvage_value": 5000.00,
    "useful_life_years": 10
}
```

**Response (201 Created):**

```json
{
    "asset_id": "880e8400-e29b-41d4-a716-446655440000",
    "asset_name": "CNC Milling Machine",
    "purchase_cost": 50000.00,
    "salvage_value": 5000.00,
    "useful_life_years": 10
}
```

| Field | Type | Description |
|---|---|---|
| `asset_id` | UUID | ID of the registered asset |
| `asset_name` | string | Name of the asset |
| `purchase_cost` | decimal | Original purchase cost |
| `salvage_value` | decimal | Estimated salvage value |
| `useful_life_years` | integer | Useful life in years |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, purchase_cost ≤ 0, salvage_value < 0, useful_life_years ≤ 0, invalid date |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.assets.write` permission |

---

### 5. Calculate Asset Depreciation

```
GET /api/v1/accounting/assets/{asset_id}/depreciation?from_date=YYYY-MM-DD&to_date=YYYY-MM-DD
```

Calculates straight-line depreciation for a fixed asset over a specified date range. Returns annual depreciation, accumulated depreciation, and current book value.

**Required permission:** `accounting.assets.read`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `asset_id` | UUID | **Yes** | ID of the asset to calculate depreciation for |

**Query parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `from_date` | string | **Yes** | Start date in `YYYY-MM-DD` format |
| `to_date` | string | **Yes** | End date in `YYYY-MM-DD` format |

**Response (200 OK):**

```json
{
    "asset_id": "880e8400-e29b-41d4-a716-446655440000",
    "annual_depreciation": 4500.00,
    "accumulated_depreciation": 9000.00,
    "current_book_value": 41000.00
}
```

| Field | Type | Description |
|---|---|---|
| `asset_id` | UUID | ID of the asset |
| `annual_depreciation` | decimal | Annual depreciation amount (straight-line: (cost - salvage) / useful life) |
| `accumulated_depreciation` | decimal | Total depreciation accumulated over the specified period |
| `current_book_value` | decimal | Current book value (purchase cost - accumulated depreciation) |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing query parameters, invalid date format, from_date > to_date |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.assets.read` permission |
| `404 Not Found` | Asset not found |

---

### 6. Create Tax Rate

```
POST /api/v1/accounting/tax-rates
```

Creates a new tax rate configuration for a specific country. Supports VAT, GST, sales tax, and other tax types.

**Required permission:** `accounting.tax.manage`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `country_code` | string | **Yes** | ISO 3166-1 alpha-2 country code (e.g., `US`, `GB`, `AU`) |
| `tax_name` | string | **Yes** | Tax name (e.g., `VAT`, `GST`, `Sales Tax`) |
| `tax_rate` | decimal | **Yes** | Tax rate as a decimal (e.g., `0.08` for 8%) |

**Example request:**

```json
{
    "country_code": "US",
    "tax_name": "VAT",
    "tax_rate": 0.08
}
```

**Response (201 Created):**

```json
{
    "tax_rate_id": "990e8400-e29b-41d4-a716-446655440000",
    "country_code": "US",
    "tax_name": "VAT",
    "tax_rate": 0.08
}
```

| Field | Type | Description |
|---|---|---|
| `tax_rate_id` | UUID | ID of the created tax rate |
| `country_code` | string | ISO 3166-1 alpha-2 country code |
| `tax_name` | string | Tax name |
| `tax_rate` | decimal | Tax rate as a decimal |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid country code, tax_rate < 0 or > 1 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.tax.manage` permission |
| `409 Conflict` | Tax rate already exists for this country and tax name |

---

### 7. Calculate Tax

```
POST /api/v1/accounting/tax/calculate
```

Calculates tax for a given subtotal based on the configured tax rates for the specified country. Returns the tax amount, total, and a breakdown of applied rates.

**Required permission:** `accounting.tax.read`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `country_code` | string | **Yes** | ISO 3166-1 alpha-2 country code (e.g., `US`) |
| `subtotal` | decimal | **Yes** | Pre-tax subtotal amount (must be ≥ 0) |

**Example request:**

```json
{
    "country_code": "US",
    "subtotal": 1000.00
}
```

**Response (200 OK):**

```json
{
    "subtotal": 1000.00,
    "tax_amount": 80.00,
    "total_amount": 1080.00,
    "applied_rates": [
        {
            "tax_name": "VAT",
            "tax_rate": 0.08,
            "tax_amount": 80.00
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `subtotal` | decimal | Original pre-tax subtotal |
| `tax_amount` | decimal | Total tax amount across all applied rates |
| `total_amount` | decimal | Subtotal + tax amount |
| `applied_rates` | array | Breakdown of each applied tax rate |
| `applied_rates[].tax_name` | string | Name of the tax |
| `applied_rates[].tax_rate` | decimal | Applied tax rate as a decimal |
| `applied_rates[].tax_amount` | decimal | Tax amount for this rate |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid country code, subtotal < 0 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.tax.read` permission |
| `404 Not Found` | No tax rates configured for the specified country |

---

### 8. Import Bank Statement

```
POST /api/v1/accounting/bank-statements
```

Imports a bank statement with opening/closing balances and a list of transactions for reconciliation.

**Required permission:** `accounting.bankrec.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `bank_name` | string | **Yes** | Name of the bank |
| `account_number` | string | **Yes** | Bank account number |
| `statement_date` | string | **Yes** | Statement date in `YYYY-MM-DD` format |
| `opening_balance` | decimal | **Yes** | Opening balance on the statement |
| `closing_balance` | decimal | **Yes** | Closing balance on the statement |
| `transactions` | array | **Yes** | Array of bank transaction entries |
| `transactions[].transaction_date` | string | **Yes** | Transaction date in `YYYY-MM-DD` format |
| `transactions[].description` | string | **Yes** | Transaction description/payee |
| `transactions[].reference` | string | No | Bank reference number |
| `transactions[].debit` | decimal | No | Debit amount (default 0.00) |
| `transactions[].credit` | decimal | No | Credit amount (default 0.00) |
| `transactions[].amount` | decimal | **Yes** | Transaction amount |

**Example request:**

```json
{
    "bank_name": "Chase Business",
    "account_number": "1234567890",
    "statement_date": "2026-01-31",
    "opening_balance": 25000.00,
    "closing_balance": 28750.00,
    "transactions": [
        {
            "transaction_date": "2026-01-05",
            "description": "Customer Payment - Acme Corp",
            "reference": "CHK-1001",
            "debit": 0.00,
            "credit": 5000.00,
            "amount": 5000.00
        },
        {
            "transaction_date": "2026-01-10",
            "description": "Office Supplies - Staples",
            "reference": "DC-4521",
            "debit": 1250.00,
            "credit": 0.00,
            "amount": -1250.00
        }
    ]
}
```

**Response (201 Created):**

```json
{
    "statement_id": "aa0e8400-e29b-41d4-a716-446655440000",
    "status": "imported"
}
```

| Field | Type | Description |
|---|---|---|
| `statement_id` | UUID | ID of the imported bank statement |
| `status` | string | Import status — `"imported"` on success |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, empty transactions array, invalid date format |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.bankrec.write` permission |

---

### 9. Reconcile Bank Statement

```
POST /api/v1/accounting/bank-statements/{statement_id}/reconcile
```

Runs AI-powered reconciliation between the imported bank statement transactions and the general ledger entries. Matches transactions by amount proximity (±1%) and returns matched/unmatched counts.

**Required permission:** `accounting.bankrec.write`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `statement_id` | UUID | **Yes** | ID of the bank statement to reconcile |

**Response (200 OK):**

```json
{
    "total_matched": 42,
    "total_unmatched": 3,
    "outstanding_debit": 1500.00,
    "outstanding_credit": 0.00,
    "reconciled_balance": 28750.00,
    "matches": [
        {
            "bank_transaction_id": "bb0e8400-e29b-41d4-a716-446655440000",
            "journal_entry_id": "660e8400-e29b-41d4-a716-446655440000",
            "amount_difference": 0.00,
            "confidence": 0.98
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `total_matched` | integer | Number of transactions matched to ledger entries |
| `total_unmatched` | integer | Number of transactions with no matching ledger entry |
| `outstanding_debit` | decimal | Total unmatched debit amount |
| `outstanding_credit` | decimal | Total unmatched credit amount |
| `reconciled_balance` | decimal | Reconciled balance after matching |
| `matches` | array | Array of matched transaction pairs |
| `matches[].bank_transaction_id` | UUID | ID of the matched bank transaction |
| `matches[].journal_entry_id` | UUID | ID of the matched journal entry |
| `matches[].amount_difference` | decimal | Absolute difference between matched amounts |
| `matches[].confidence` | decimal | AI confidence score (0.0–1.0) |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Invalid statement_id UUID |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.bankrec.write` permission |
| `404 Not Found` | Bank statement not found |

---

### 10. Create Exchange Rate

```
POST /api/v1/accounting/exchange-rates
```

Creates a new currency exchange rate for multi-currency accounting.

**Required permission:** `accounting.exchangerates.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `from_currency` | string | **Yes** | Source currency ISO 4217 code (e.g., `USD`, `EUR`) |
| `to_currency` | string | **Yes** | Target currency ISO 4217 code (e.g., `GBP`, `JPY`) |
| `rate` | decimal | **Yes** | Exchange rate (must be > 0) |
| `effective_date` | string | **Yes** | Date the rate becomes effective in `YYYY-MM-DD` format |

**Example request:**

```json
{
    "from_currency": "USD",
    "to_currency": "EUR",
    "rate": 0.92,
    "effective_date": "2026-01-15"
}
```

**Response (201 Created):**

```json
{
    "exchange_rate_id": "cc0e8400-e29b-41d4-a716-446655440000",
    "from_currency": "USD",
    "to_currency": "EUR",
    "rate": 0.92,
    "effective_date": "2026-01-15"
}
```

| Field | Type | Description |
|---|---|---|
| `exchange_rate_id` | UUID | ID of the created exchange rate |
| `from_currency` | string | Source currency code |
| `to_currency` | string | Target currency code |
| `rate` | decimal | Exchange rate |
| `effective_date` | string | Effective date |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid currency codes, rate ≤ 0, invalid date |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.exchangerates.write` permission |
| `409 Conflict` | Exchange rate already exists for this currency pair and effective date |

---

### 11. Convert Currency

```
POST /api/v1/accounting/convert
```

Converts an amount from one currency to another using the latest effective exchange rate.

**Required permission:** `accounting.currencyconvert.read`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `amount` | decimal | **Yes** | Amount to convert (must be ≥ 0) |
| `from_currency` | string | **Yes** | Source currency ISO 4217 code |
| `to_currency` | string | **Yes** | Target currency ISO 4217 code |

**Example request:**

```json
{
    "amount": 1000.00,
    "from_currency": "USD",
    "to_currency": "EUR"
}
```

**Response (200 OK):**

```json
{
    "original_amount": 1000.00,
    "from_currency": "USD",
    "to_currency": "EUR",
    "converted_amount": 920.00
}
```

| Field | Type | Description |
|---|---|---|
| `original_amount` | decimal | Original amount in source currency |
| `from_currency` | string | Source currency code |
| `to_currency` | string | Target currency code |
| `converted_amount` | decimal | Converted amount in target currency |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid currency codes, amount < 0 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `accounting.currencyconvert.read` permission |
| `404 Not Found` | No exchange rate found for the specified currency pair |

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

### Bank reconciliation auto-matching (Module 2.6)

Matches imported bank statement transactions against general ledger entries by:
- Amount proximity within ±1% tolerance
- Date proximity (within 5 business days)
- Reference/description keyword matching
- Confidence scoring from 0.0 to 1.0 based on match quality

Unmatched transactions are flagged as outstanding debit/credit for manual review.

### Multi-currency exchange rate management (Module 2.7)

Stores exchange rates per currency pair with effective dates. The currency conversion endpoint queries the latest effective rate for the given pair and performs the calculation. Rate conflict detection prevents duplicate entries for the same currency pair and effective date.