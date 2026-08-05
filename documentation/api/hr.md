# HR, Workforce & Collaboration

The HR module provides endpoints for employee attendance (geofenced clock-in), AI-powered resume parsing with job matching, and RAG-based knowledge base semantic search.

## Base path

All HR endpoints are prefixed with `/api/v1/hr/`.

## Authentication

All HR endpoints require a **Bearer Token** in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

The token must contain the appropriate HR permission scopes.

## Endpoints

### 1. Geofenced Clock-In

```
POST /api/v1/hr/attendance/clock-in
```

Records an employee attendance clock-in event with GPS coordinates. Validates whether the location is within a configured geofence. The employee is identified from the JWT token's user identity.

**Required permission:** `hr.attendance.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `latitude` | decimal | **Yes** | GPS latitude (-90 to 90) |
| `longitude` | decimal | **Yes** | GPS longitude (-180 to 180) |

**Example request:**

```json
{
    "latitude": 37.7749,
    "longitude": -122.4194
}
```

**Response (200 OK):**

```json
{
    "attendance_log_id": "550e8400-e29b-41d4-a716-446655440000",
    "clock_in": "2025-01-15T09:00:00Z",
    "is_within_geofence": true
}
```

| Field | Type | Description |
|---|---|---|
| `attendance_log_id` | UUID | ID of the created attendance log entry |
| `clock_in` | ISO 8601 timestamp | Time the clock-in was recorded |
| `is_within_geofence` | boolean | Whether the GPS coordinates fall within the organization's configured geofence |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid latitude/longitude range |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.attendance.write` permission |
| `404 Not Found` | No employee record found for the authenticated user |

**Geofence behavior:**

The current implementation simulates a geofence centered on San Francisco (37.7749, -122.4194) with a ~1.1 km radius. In production, this would query the organization's configured office locations and use a point-in-polygon algorithm.

---

### 2. Parse Resume & Score Match

```
POST /api/v1/hr/ats/parse-resume
```

Accepts a resume file (PDF or Docx) via multipart form upload, extracts candidate details and skills using AI, scores the match against a specified job description, and stores the application.

**Required permission:** `hr.recruitment.write`

**Request (multipart/form-data):**

| Field | Type | Required | Description |
|---|---|---|---|
| `resume_file` | file | **Yes** | Resume file (PDF or Docx, max 10 MB) |
| `job_description_id` | UUID | **Yes** | UUID of the job description to match against |

**Example request (curl):**

```bash
curl -X POST http://localhost:8080/api/v1/hr/ats/parse-resume \
  -H "Authorization: Bearer <token>" \
  -F "resume_file=@jane_smith_resume.pdf" \
  -F "job_description_id=550e8400-e29b-41d4-a716-446655440000"
```

**Response (200 OK):**

```json
{
    "candidate_name": "Jane Smith",
    "email": "jane.smith@email.com",
    "extracted_skills": ["Python", "Go", "Docker", "Kubernetes"],
    "ai_match_score": 85
}
```

| Field | Type | Description |
|---|---|---|
| `candidate_name` | string | Extracted candidate name |
| `email` | string | Extracted candidate email |
| `extracted_skills` | array of strings | AI-extracted skills from the resume |
| `ai_match_score` | integer | AI match score against the job description (0–100) |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid file type (not PDF/Docx), invalid job_description_id |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.recruitment.write` permission |
| `413 Request Entity Too Large` | File exceeds 10 MB limit |

**Supported file types:**

| Extension | MIME Type |
|---|---|
| `.pdf` | application/pdf |
| `.docx` | application/vnd.openxmlformats-officedocument.wordprocessingml.document |

---

### 3. RAG Knowledge Base Semantic Search

```
POST /api/v1/hr/knowledge/search
```

Performs a RAG-powered semantic search over the organization's knowledge base documents (e.g., HR policies, onboarding guides) and returns an AI-synthesized answer with source citations.

**Required permission:** `knowledge.read`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `query` | string | **Yes** | Natural language search query |

**Example request:**

```json
{
    "query": "What is our parental leave policy in Australia?"
}
```

**Response (200 OK):**

```json
{
    "ai_answer": "Based on our knowledge base, here is the answer to \"What is our parental leave policy in Australia?\": The document **Parental Leave Policy - Australia** addresses this topic. Employees are entitled to 18 weeks of paid parental leave...",
    "source_documents": [
        {
            "title": "Parental Leave Policy - Australia",
            "relevance_score": 0.85
        },
        {
            "title": "Employee Benefits Handbook 2025",
            "relevance_score": 0.62
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `ai_answer` | string | AI-synthesized answer with inline citations to source documents |
| `source_documents` | array | Matching documents ranked by relevance |
| `source_documents[].title` | string | Document title |
| `source_documents[].relevance_score` | decimal | Relevance score (0.0–1.0), higher = more relevant |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing or empty query |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `knowledge.read` permission |

---

### 4. Clock Out

```
POST /api/v1/hr/attendance/clock-out
```

Clocks the authenticated user out, closing the most recent open attendance log. Uses the JWT to identify the user — no request body required.

**Required permission:** `hr.attendance.clockout`

**No request body required** (user identified from JWT).

**Response (200 OK):**

```json
{
    "attendance_log_id": "aa0e8400-e29b-41d4-a716-446655440000",
    "clock_in": "2026-01-15T09:00:00Z",
    "clock_out": "2026-01-15T17:30:00Z",
    "hours_worked": 8.5
}
```

| Field | Type | Description |
|---|---|---|
| `attendance_log_id` | UUID | ID of the attendance record |
| `clock_in` | string (ISO 8601) | Clock-in timestamp |
| `clock_out` | string (ISO 8601) | Clock-out timestamp |
| `hours_worked` | decimal | Total hours worked |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | No open attendance log found (already clocked out) |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.attendance.clockout` permission |

---

### 5. Create Shift Template

```
POST /api/v1/hr/shifts/templates
```

Creates a reusable shift template defining start/end times and required headcount, optionally scoped to a specific day of the week and department.

**Required permission:** `hr.shifts.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | **Yes** | Shift template name (e.g., `Morning Shift`) |
| `start_time` | string | **Yes** | Shift start time in `HH:MM` format (24-hour) |
| `end_time` | string | **Yes** | Shift end time in `HH:MM` format (24-hour) |
| `day_of_week` | integer | No | Day of week (0=Sunday, 6=Saturday). If omitted, applies to all days |
| `department` | string | No | Department the shift belongs to |
| `required_headcount` | integer | **Yes** | Minimum number of employees needed for this shift |

**Example request:**

```json
{
    "name": "Morning Shift",
    "start_time": "06:00",
    "end_time": "14:00",
    "day_of_week": 1,
    "department": "Warehouse",
    "required_headcount": 5
}
```

**Response (201 Created):**

```json
{
    "shift_template_id": "bb0e8400-e29b-41d4-a716-446655440000",
    "name": "Morning Shift",
    "start_time": "06:00",
    "end_time": "14:00",
    "required_headcount": 5
}
```

| Field | Type | Description |
|---|---|---|
| `shift_template_id` | UUID | ID of the created shift template |
| `name` | string | Shift template name |
| `start_time` | string | Shift start time |
| `end_time` | string | Shift end time |
| `required_headcount` | integer | Required headcount |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid time format, required_headcount ≤ 0 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.shifts.write` permission |

---

### 6. Assign Shift

```
POST /api/v1/hr/shifts/assignments
```

Assigns an employee to a shift template on a specific date.

**Required permission:** `hr.shifts.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `employee_id` | UUID | **Yes** | ID of the employee to assign |
| `shift_template_id` | UUID | **Yes** | ID of the shift template |
| `shift_date` | string | **Yes** | Date of the shift in `YYYY-MM-DD` format |

**Example request:**

```json
{
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "shift_template_id": "bb0e8400-e29b-41d4-a716-446655440000",
    "shift_date": "2026-01-20"
}
```

**Response (201 Created):**

```json
{
    "shift_assignment_id": "cc0e8400-e29b-41d4-a716-446655440000",
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "shift_date": "2026-01-20",
    "status": "scheduled"
}
```

| Field | Type | Description |
|---|---|---|
| `shift_assignment_id` | UUID | ID of the shift assignment |
| `employee_id` | UUID | ID of the assigned employee |
| `shift_date` | string | Date of the shift |
| `status` | string | Assignment status — `"scheduled"` on creation |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid UUIDs, invalid date format |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.shifts.write` permission |
| `404 Not Found` | Employee or shift template not found |

---

### 7. Predict Shift Needs

```
GET /api/v1/hr/shifts/predictions?from_date=YYYY-MM-DD&to_date=YYYY-MM-DD&department=...
```

Returns AI-predicted headcount needs per day based on historical patterns and seasonality. The AI simulation analyzes day-of-week patterns and monthly seasonal adjustments to produce confidence-scored predictions.

**Required permission:** `hr.shifts.write`

**Query parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `from_date` | string | **Yes** | Start date in `YYYY-MM-DD` format |
| `to_date` | string | **Yes** | End date in `YYYY-MM-DD` format |
| `department` | string | No | Filter predictions by department |

**Response (200 OK):**

```json
{
    "predictions": [
        {
            "date": "2026-01-20",
            "department": "Warehouse",
            "predicted_headcount_needed": 8,
            "confidence_score": 0.85,
            "reasoning": "Tuesday typically has 15% higher demand. Monthly seasonal factor: 1.05."
        },
        {
            "date": "2026-01-21",
            "department": "Warehouse",
            "predicted_headcount_needed": 6,
            "confidence_score": 0.78,
            "reasoning": "Wednesday baseline demand. Monthly seasonal factor: 1.05."
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `predictions` | array | Array of daily shift need predictions |
| `predictions[].date` | string | Prediction date |
| `predictions[].department` | string | Department name |
| `predictions[].predicted_headcount_needed` | integer | AI-predicted number of employees needed |
| `predictions[].confidence_score` | decimal | Confidence in the prediction (0.0–1.0) |
| `predictions[].reasoning` | string | Explanation of the AI's prediction logic |

**AI simulation logic:**

- **Day-of-week patterns:** Weekdays receive a demand multiplier based on historical data (e.g., Monday/Friday ±5%, Tuesday/Wednesday +10–15%, Thursday baseline)
- **Seasonal adjustments:** Monthly seasonal factors (e.g., December +30% for holiday rush, January ±0%, July −10% for summer slowdown)
- **Confidence scoring:** Based on data freshness and pattern consistency — higher confidence for well-established patterns, lower for weekends and holidays

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing query parameters, invalid date format, from_date > to_date |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.shifts.write` permission |

---

### 8. Get Employee Schedule

```
GET /api/v1/hr/shifts/schedule?employee_id=...&from_date=YYYY-MM-DD&to_date=YYYY-MM-DD
```

Retrieves an employee's shift schedule for a given date range.

**Required permission:** `hr.shifts.write`

**Query parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `employee_id` | UUID | **Yes** | ID of the employee |
| `from_date` | string | **Yes** | Start date in `YYYY-MM-DD` format |
| `to_date` | string | **Yes** | End date in `YYYY-MM-DD` format |

**Response (200 OK):**

```json
{
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "schedule": [
        {
            "shift_assignment_id": "cc0e8400-e29b-41d4-a716-446655440000",
            "shift_date": "2026-01-20",
            "start_time": "06:00",
            "end_time": "14:00",
            "shift_name": "Morning Shift",
            "status": "scheduled"
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `employee_id` | UUID | Employee ID |
| `schedule` | array | Array of scheduled shifts |
| `schedule[].shift_assignment_id` | UUID | Shift assignment ID |
| `schedule[].shift_date` | string | Shift date |
| `schedule[].start_time` | string | Shift start time |
| `schedule[].end_time` | string | Shift end time |
| `schedule[].shift_name` | string | Name of the shift template |
| `schedule[].status` | string | Assignment status |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing query parameters, invalid UUID, invalid date format, from_date > to_date |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.shifts.write` permission |
| `404 Not Found` | Employee not found |

---

### 9. Create Employee

```
POST /api/v1/hr/employees
```

Creates a new employee record linked to an existing user. Records department, job title, salary, and hire date.

**Required permission:** `hr.employees.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `user_id` | UUID | **Yes** | ID of the user to link as an employee |
| `employee_code` | string | **Yes** | Unique employee code (e.g., `EMP001`) |
| `department` | string | **Yes** | Department name (e.g., `Engineering`) |
| `job_title` | string | **Yes** | Job title (e.g., `Software Engineer`) |
| `salary` | decimal | **Yes** | Annual salary (must be > 0) |
| `hired_at` | string | **Yes** | Hire date in `YYYY-MM-DD` format |

**Example request:**

```json
{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "employee_code": "EMP001",
    "department": "Engineering",
    "job_title": "Software Engineer",
    "salary": 120000.00,
    "hired_at": "2025-03-15"
}
```

**Response (201 Created):**

```json
{
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "employee_code": "EMP001",
    "department": "Engineering",
    "job_title": "Software Engineer",
    "salary": 120000.00,
    "hired_at": "2025-03-15"
}
```

| Field | Type | Description |
|---|---|---|
| `employee_id` | UUID | ID of the created employee record |
| `employee_code` | string | Unique employee code |
| `department` | string | Department name |
| `job_title` | string | Job title |
| `salary` | decimal | Annual salary |
| `hired_at` | string | Hire date |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, salary ≤ 0, invalid date, duplicate employee_code |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.employees.write` permission |
| `404 Not Found` | User not found |

---

### 10. List Employees

```
GET /api/v1/hr/employees?department=Engineering
```

Lists all employees in the organization, optionally filtered by department.

**Required permission:** `hr.employees.read`

**Query parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `department` | string | No | Filter employees by department name |

**Response (200 OK):**

```json
{
    "employees": [
        {
            "employee_id": "880e8400-e29b-41d4-a716-446655440000",
            "employee_code": "EMP001",
            "department": "Engineering",
            "job_title": "Software Engineer",
            "salary": 120000.00,
            "hired_at": "2025-03-15"
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `employees` | array | Array of employee objects |

**Error responses:**

| Status | Condition |
|---|---|
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.employees.read` permission |

---

### 11. Get Employee

```
GET /api/v1/hr/employees/{employee_id}
```

Retrieves a single employee record by ID.

**Required permission:** `hr.employees.read`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `employee_id` | UUID | **Yes** | ID of the employee to retrieve |

**Response (200 OK):**

```json
{
    "employee": {
        "employee_id": "880e8400-e29b-41d4-a716-446655440000",
        "employee_code": "EMP001",
        "department": "Engineering",
        "job_title": "Software Engineer",
        "salary": 120000.00,
        "hired_at": "2025-03-15"
    }
}
```

| Field | Type | Description |
|---|---|---|
| `employee` | object | Full employee record |

**Error responses:**

| Status | Condition |
|---|---|
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.employees.read` permission |
| `404 Not Found` | Employee not found |

---

### 12. Update Employee

```
PATCH /api/v1/hr/employees/{employee_id}
```

Partially updates an employee record. Only the fields provided in the request body are updated.

**Required permission:** `hr.employees.write`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `employee_id` | UUID | **Yes** | ID of the employee to update |

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `department` | string | No | Updated department name |
| `job_title` | string | No | Updated job title |
| `salary` | decimal | No | Updated annual salary (must be > 0 if provided) |

**Example request:**

```json
{
    "department": "Engineering",
    "job_title": "Senior Engineer",
    "salary": 140000.00
}
```

**Response (200 OK):**

```json
{
    "message": "employee updated",
    "employee_id": "880e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Description |
|---|---|---|
| `message` | string | Confirmation message |
| `employee_id` | UUID | ID of the updated employee |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | No fields provided, salary ≤ 0 |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.employees.write` permission |
| `404 Not Found` | Employee not found |

---

### 13. Run Payroll

```
POST /api/v1/hr/payroll/runs
```

Executes a payroll run for all active employees within the specified pay period. Calculates disbursements and returns a summary.

**Required permission:** `hr.payroll.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `pay_period_start` | string | **Yes** | Pay period start date in `YYYY-MM-DD` format |
| `pay_period_end` | string | **Yes** | Pay period end date in `YYYY-MM-DD` format |

**Example request:**

```json
{
    "pay_period_start": "2026-01-01",
    "pay_period_end": "2026-01-15"
}
```

**Response (201 Created):**

```json
{
    "payroll_run_id": "990e8400-e29b-41d4-a716-446655440000",
    "total_disbursed": 250000.00,
    "status": "completed"
}
```

| Field | Type | Description |
|---|---|---|
| `payroll_run_id` | UUID | ID of the payroll run |
| `total_disbursed` | decimal | Total amount disbursed across all employees |
| `status` | string | Payroll run status — `"completed"` on success |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid date format, pay_period_start > pay_period_end |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.payroll.write` permission |

---

### 14. List Payroll Runs

```
GET /api/v1/hr/payroll/runs
```

Lists all payroll runs for the organization, ordered by most recent first.

**Required permission:** `hr.payroll.read`

**Response (200 OK):**

```json
{
    "payroll_runs": [
        {
            "payroll_run_id": "990e8400-e29b-41d4-a716-446655440000",
            "pay_period_start": "2026-01-01",
            "pay_period_end": "2026-01-15",
            "total_disbursed": 250000.00,
            "status": "completed"
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `payroll_runs` | array | Array of payroll run objects |

**Error responses:**

| Status | Condition |
|---|---|
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.payroll.read` permission |

---

### 15. Get Payroll Run

```
GET /api/v1/hr/payroll/runs/{run_id}
```

Retrieves a single payroll run by ID, including per-employee disbursement details.

**Required permission:** `hr.payroll.read`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `run_id` | UUID | **Yes** | ID of the payroll run to retrieve |

**Response (200 OK):**

```json
{
    "payroll_run": {
        "payroll_run_id": "990e8400-e29b-41d4-a716-446655440000",
        "pay_period_start": "2026-01-01",
        "pay_period_end": "2026-01-15",
        "total_disbursed": 250000.00,
        "status": "completed",
        "disbursements": [
            {
                "employee_id": "880e8400-e29b-41d4-a716-446655440000",
                "amount": 5000.00
            }
        ]
    }
}
```

| Field | Type | Description |
|---|---|---|
| `payroll_run` | object | Full payroll run record with disbursements |

**Error responses:**

| Status | Condition |
|---|---|
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.payroll.read` permission |
| `404 Not Found` | Payroll run not found |

---

### Payroll Run Detail

```
GET /api/v1/hr/payroll/runs/{run_id}/detail
```

Retrieves detailed payroll run information including per-employee tax breakdowns (gross pay, tax withheld, social security, net pay).

**Required permission:** `hr.payroll.read`

**Path parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `run_id` | UUID | **Yes** | ID of the payroll run to retrieve |

**Response (200 OK):**

```json
{
    "payroll_run": {
        "payroll_run_id": "990e8400-e29b-41d4-a716-446655440000",
        "pay_period_start": "2026-01-01",
        "pay_period_end": "2026-01-15",
        "total_disbursed": 250000.00,
        "status": "completed"
    },
    "disbursements": [
        {
            "employee_id": "880e8400-e29b-41d4-a716-446655440000",
            "employee_code": "EMP-001",
            "gross_pay": 6250.00,
            "tax_withheld": 1250.00,
            "social_security": 387.50,
            "net_pay": 4612.50
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `payroll_run` | object | Payroll run summary |
| `disbursements` | array | Per-employee disbursement details with tax breakdown |
| `disbursements[].employee_id` | UUID | Employee ID |
| `disbursements[].employee_code` | string | Employee code |
| `disbursements[].gross_pay` | decimal | Gross pay before deductions |
| `disbursements[].tax_withheld` | decimal | Income tax withheld |
| `disbursements[].social_security` | decimal | Social security contribution |
| `disbursements[].net_pay` | decimal | Net pay after all deductions |

**Error responses:**

| Status | Condition |
|---|---|
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.payroll.read` permission |
| `404 Not Found` | Payroll run not found |

---

### Set Employee Tax Profile

```
POST /api/v1/hr/payroll/tax-profiles
```

Creates a tax profile for an employee, configuring their tax country, filing status, allowances, and additional withholding amounts. Used by the payroll engine to calculate progressive US-style tax brackets.

**Required permission:** `hr.payroll.write`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `employee_id` | UUID | **Yes** | ID of the employee |
| `tax_country` | string | **Yes** | Tax jurisdiction country code (e.g., `US`) |
| `tax_id` | string | **Yes** | Employee tax identification number (e.g., SSN, EIN) |
| `filing_status` | string | **Yes** | Filing status: `single`, `married_joint`, `married_separate`, `head_of_household` |
| `allowances` | integer | No | Number of withholding allowances (default 0) |
| `additional_withholding` | decimal | No | Additional amount to withhold per pay period (default 0.00) |

**Example request:**

```json
{
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "tax_country": "US",
    "tax_id": "123-45-6789",
    "filing_status": "single",
    "allowances": 1,
    "additional_withholding": 50.00
}
```

**Response (201 Created):**

```json
{
    "tax_profile_id": "dd0e8400-e29b-41d4-a716-446655440000",
    "employee_id": "880e8400-e29b-41d4-a716-446655440000",
    "tax_country": "US",
    "filing_status": "single"
}
```

| Field | Type | Description |
|---|---|---|
| `tax_profile_id` | UUID | ID of the created tax profile |
| `employee_id` | UUID | Employee ID |
| `tax_country` | string | Tax jurisdiction country |
| `filing_status` | string | Employee filing status |

**Error responses:**

| Status | Condition |
|---|---|
| `400 Bad Request` | Missing required fields, invalid filing status, invalid employee_id |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Token lacks `hr.payroll.write` permission |
| `404 Not Found` | Employee not found |
| `409 Conflict` | Tax profile already exists for this employee |

---

## AI engine (simulated)

The current implementation uses simulated AI for all three endpoints. In production, these would be replaced with external AI/ML service calls.

### Geofence validation (Module 4.2)

Uses a simple distance-from-center calculation against a simulated office location. In production, this would query the organization's configured office locations and use a proper point-in-polygon algorithm (e.g., ray casting) against geofence boundaries.

### Resume parsing & match scoring (Module 4.4)

**Resume extraction:**
- Candidate name is derived from the uploaded file name
- Email is generated from the candidate name
- Skills are randomly selected from a pool of 30 common tech skills (3–7 skills per resume)

**Match scoring:**
- Base score: 40 + (number of extracted skills × 3)
- Jitter: ±10 points random variation
- Clamped to 0–100 range

In production, this would call an NLP service (e.g., AWS Textract, Google Document AI) for structured data extraction from PDF/Docx files, and an ML model for skill-to-job-description matching.

### Knowledge base search (Module 4.5)

**Search:** Uses PostgreSQL `ILIKE` for full-text search across document titles and content. Returns up to 5 matching documents ordered by recency.

**Answer synthesis:** Generates a simulated RAG answer by referencing the top-matching document's title and content excerpt. Relevance scores are computed based on term overlap between the query and document content.

In production, this would use a vector database (e.g., pgvector, Pinecone, Weaviate) with embedding-based semantic search and an LLM for answer synthesis with proper RAG pipeline (retrieval → augmentation → generation).

### Shift prediction (Module 4.6)

Uses day-of-week patterns and monthly seasonality to predict required headcount:
- **Day-of-week patterns:** Weekday demand multipliers (Monday +5%, Tuesday +15%, Wednesday +10%, Thursday baseline, Friday +5%, weekends −30%)
- **Monthly seasonal adjustments:** December +30% (holiday rush), June–August −10% (summer slowdown), other months near baseline
- **Confidence scoring:** Confidence is higher for weekdays with stable historical patterns (0.80–0.95), lower for weekends and holidays (0.60–0.75)

In production, this would be replaced with a time-series forecasting model (e.g., Prophet, ARIMA) trained on historical attendance and sales data.

### Tax calculation (Module 4.7)

Applies progressive US-style tax brackets with standard deduction and social security:
- **Standard deduction:** $13,850 (single), $27,700 (married joint) — annualized per pay period
- **Progressive brackets:** 10%, 12%, 22%, 24%, 32%, 35%, 37% applied to taxable income after deduction
- **Social security:** 6.2% on wages up to the annual wage base limit
- **Additional withholding:** Applied on top of calculated tax per the employee's tax profile

In production, this would integrate with a tax compliance service (e.g., Avalara, TaxJar) for multi-jurisdictional support.

---

## Database tables

The HR module introduces five new database tables:

| Table | Purpose |
|---|---|
| `employees` | Employee records linked to users and organizations |
| `attendance_logs` | Clock-in/out records with GPS coordinates |
| `payroll_runs` | Payroll processing batches |
| `payroll_disbursements` | Per-employee payroll disbursement records with tax breakdown |
| `employee_tax_profiles` | Employee tax profiles (country, filing status, allowances) |
| `shift_templates` | Reusable shift templates with start/end times and headcount |
| `shift_assignments` | Employee shift assignments per date |
| `job_applications` | Candidate applications with AI match scores |
| `knowledge_base_documents` | HR policy documents for RAG search |

See [Database Schema](../architecture/database.md) for full table definitions.

---

## Permissions

The HR module introduces seven new permissions:

| Permission Key | Module | Description |
|---|---|---|
| `hr.attendance.write` | hr | Clock in/out and manage attendance records |
| `hr.attendance.clockout` | hr | Clock out and close attendance logs |
| `hr.recruitment.write` | hr | Parse resumes and manage job applications |
| `hr.employees.write` | hr | Create and update employee records |
| `hr.employees.read` | hr | View employee records and details |
| `hr.payroll.write` | hr | Execute payroll runs and manage tax profiles |
| `hr.payroll.read` | hr | View payroll run history, details, and tax breakdowns |
| `hr.shifts.write` | hr | Create shift templates, assign shifts, predict needs, view schedules |
| `knowledge.read` | knowledge | Search and read knowledge base documents |