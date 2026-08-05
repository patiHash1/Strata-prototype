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

### 4. Create Employee

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

### 5. List Employees

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

### 6. Get Employee

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

### 7. Update Employee

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

### 8. Run Payroll

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

### 9. List Payroll Runs

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

### 10. Get Payroll Run

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

---

## Database tables

The HR module introduces five new database tables:

| Table | Purpose |
|---|---|
| `employees` | Employee records linked to users and organizations |
| `attendance_logs` | Clock-in/out records with GPS coordinates |
| `payroll_runs` | Payroll processing batches |
| `job_applications` | Candidate applications with AI match scores |
| `knowledge_base_documents` | HR policy documents for RAG search |

See [Database Schema](../architecture/database.md) for full table definitions.

---

## Permissions

The HR module introduces seven new permissions:

| Permission Key | Module | Description |
|---|---|---|
| `hr.attendance.write` | hr | Clock in/out and manage attendance records |
| `hr.recruitment.write` | hr | Parse resumes and manage job applications |
| `hr.employees.write` | hr | Create and update employee records |
| `hr.employees.read` | hr | View employee records and details |
| `hr.payroll.write` | hr | Execute payroll runs |
| `hr.payroll.read` | hr | View payroll run history and details |
| `knowledge.read` | knowledge | Search and read knowledge base documents |