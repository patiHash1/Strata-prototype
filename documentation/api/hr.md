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

The HR module introduces three new permissions:

| Permission Key | Module | Description |
|---|---|---|
| `hr.attendance.write` | hr | Clock in/out and manage attendance records |
| `hr.recruitment.write` | hr | Parse resumes and manage job applications |
| `knowledge.read` | knowledge | Search and read knowledge base documents |