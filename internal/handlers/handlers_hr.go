package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/hr/attendance/clock-in ----

type clockInRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ClockInResponse represents the response from a geofenced clock-in.
type ClockInResponse struct {
	AttendanceLogID  string `json:"attendance_log_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClockIn          string `json:"clock_in" example:"2025-01-15T09:00:00Z"`
	IsWithinGeofence bool   `json:"is_within_geofence" example:"true"`
}

// clockInHandler records an employee attendance clock-in with geofence validation.
//
//	@Summary		Geofenced clock-in
//	@Description	Records an employee attendance clock-in event with GPS coordinates. Validates whether the location is within a configured geofence. The employee is identified from the JWT token's user identity. Requires `hr.attendance.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	clockInRequest	true	"Clock-in payload with GPS coordinates"
//	@Success		200	{object}	ClockInResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/attendance/clock-in [post]
func (a *App) clockInHandler(w http.ResponseWriter, r *http.Request) {
	var req clockInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		utils.WriteErr(w, http.StatusBadRequest, "latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		utils.WriteErr(w, http.StatusBadRequest, "longitude must be between -180 and 180")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid user in token")
		return
	}

	result, err := a.HR.ClockIn(r.Context(), orgID, userID, req.Latitude, req.Longitude)
	if err != nil {
		if err == services.ErrEmployeeNotFound {
			utils.WriteErr(w, http.StatusNotFound, "no employee record found for this user. Contact your HR administrator.")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not record clock-in")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"attendance_log_id":  result.AttendanceLogID.String(),
		"clock_in":           result.ClockIn.Format(time.RFC3339),
		"is_within_geofence": result.IsWithinGeofence,
	})
}

// ---- POST /api/v1/hr/ats/parse-resume ----

const maxResumeSize = 10 << 20 // 10 MB

// ParseResumeResponse represents the response from AI resume parsing.
type ParseResumeResponse struct {
	CandidateName   string   `json:"candidate_name" example:"Jane Smith"`
	Email           string   `json:"email" example:"jane.smith@email.com"`
	ExtractedSkills []string `json:"extracted_skills" example:"Python,Go,Docker,Kubernetes"`
	AIMatchScore    int      `json:"ai_match_score" example:"85"`
}

// parseResumeHandler parses a resume file (PDF/Docx) and scores match against a job description.
//
//	@Summary		Parse resume & score match
//	@Description	Accepts a resume file (PDF or Docx) via multipart form upload, extracts candidate details and skills using AI, scores the match against a specified job description, and stores the application. Requires `hr.recruitment.write` permission.
//	@Tags			HR
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			resume_file			formData	file	true	"Resume file (PDF or Docx)"
//	@Param			job_description_id	formData	string	true	"UUID of the job description to match against"
//	@Success		200	{object}	ParseResumeResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		413	{object}	utils.Envelope
//	@Router			/api/v1/hr/ats/parse-resume [post]
func (a *App) parseResumeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form (limit to 10 MB)
	if err := r.ParseMultipartForm(maxResumeSize); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "could not parse multipart form")
		return
	}

	jobDescriptionIDStr := r.FormValue("job_description_id")
	if !utils.NotBlank(jobDescriptionIDStr) {
		utils.WriteErr(w, http.StatusBadRequest, "job_description_id is required")
		return
	}

	jobDescriptionID, err := uuid.Parse(jobDescriptionIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid job_description_id")
		return
	}

	// Retrieve the uploaded file
	file, header, err := r.FormFile("resume_file")
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "resume_file is required")
		return
	}
	defer file.Close()

	// Validate file type
	fileName := header.Filename
	if !isAllowedResumeType(fileName) {
		utils.WriteErr(w, http.StatusBadRequest, "resume_file must be a PDF or Docx file")
		return
	}

	// Read the file bytes
	resumeBytes, err := io.ReadAll(io.LimitReader(file, maxResumeSize))
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not read resume file")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	result, err := a.HR.ParseResume(r.Context(), orgID, jobDescriptionID, resumeBytes, fileName)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not parse resume")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"candidate_name":   result.CandidateName,
		"email":            result.Email,
		"extracted_skills": result.ExtractedSkills,
		"ai_match_score":   result.AIMatchScore,
	})
}

// isAllowedResumeType checks whether the file extension indicates a PDF or Docx file.
func isAllowedResumeType(fileName string) bool {
	lower := ""
	for _, r := range fileName {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}
	return len(lower) >= 5 && (lower[len(lower)-4:] == ".pdf" || lower[len(lower)-5:] == ".docx")
}

// ---- POST /api/v1/hr/knowledge/search ----

type knowledgeSearchRequest struct {
	Query string `json:"query"`
}

// KnowledgeSearchResponse represents the RAG-powered knowledge base search result.
type KnowledgeSearchResponse struct {
	AIAnswer        string                    `json:"ai_answer" example:"Based on our knowledge base, parental leave in Australia..."  `
	SourceDocuments []services.SourceDocument `json:"source_documents"`
}

// knowledgeSearchHandler performs semantic search over the organization's knowledge base.
//
//	@Summary		RAG knowledge base semantic search
//	@Description	Performs a RAG-powered semantic search over the organization's knowledge base documents (e.g., HR policies, onboarding guides) and returns an AI-synthesized answer with source citations. Requires `knowledge.read` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	knowledgeSearchRequest	true	"Search query"
//	@Success		200	{object}	KnowledgeSearchResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/knowledge/search [post]
func (a *App) knowledgeSearchHandler(w http.ResponseWriter, r *http.Request) {
	var req knowledgeSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Query) {
		utils.WriteErr(w, http.StatusBadRequest, "query is required")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	result, err := a.HR.SearchKnowledge(r.Context(), orgID, req.Query)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not search knowledge base")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"ai_answer":        result.AIAnswer,
		"source_documents": result.SourceDocuments,
	})
}

// ---- POST /api/v1/hr/employees ----

type createEmployeeRequest struct {
	UserID       *string  `json:"user_id,omitempty"`
	EmployeeCode string   `json:"employee_code"`
	Department   *string  `json:"department,omitempty"`
	JobTitle     *string  `json:"job_title,omitempty"`
	Salary       *float64 `json:"salary,omitempty"`
	HiredAt      string   `json:"hired_at"`
}

// createEmployeeHandler creates a new employee record.
//
//	@Summary		Create employee
//	@Description	Creates a new employee record in the organization. Requires `hr.employees.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createEmployeeRequest	true	"Employee details"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/employees [post]
func (a *App) createEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.EmployeeCode) {
		utils.WriteErr(w, http.StatusBadRequest, "employee_code is required")
		return
	}

	var hiredAt time.Time
	if req.HiredAt != "" {
		var err error
		hiredAt, err = time.Parse(time.RFC3339, req.HiredAt)
		if err != nil {
			hiredAt, err = time.Parse("2006-01-02", req.HiredAt)
			if err != nil {
				utils.WriteErr(w, http.StatusBadRequest, "invalid hired_at format, use RFC3339 or YYYY-MM-DD")
				return
			}
		}
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	var userID *uuid.UUID
	if req.UserID != nil && *req.UserID != "" {
		parsed, err := uuid.Parse(*req.UserID)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		userID = &parsed
	}

	employee, err := a.HR.CreateEmployee(r.Context(), orgID, userID, req.EmployeeCode, req.Department, req.JobTitle, req.Salary, hiredAt)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create employee")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":            employee.ID.String(),
		"org_id":        employee.OrgID.String(),
		"user_id":       employee.UserID,
		"employee_code": employee.EmployeeCode,
		"department":    employee.Department,
		"job_title":     employee.JobTitle,
		"salary":        employee.Salary,
		"hired_at":      employee.HiredAt.Format(time.RFC3339),
	})
}

// ---- GET /api/v1/hr/employees ----

// listEmployeesHandler lists all employees in the org, with optional department filter.
//
//	@Summary		List employees
//	@Description	Lists all employees in the organization. Optionally filter by department via query parameter. Requires `hr.employees.read` permission.
//	@Tags			HR
//	@Produce		json
//	@Security		BearerAuth
//	@Param			department	query	string	false	"Filter by department name"
//	@Success		200	{array}		services.Employee
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/employees [get]
func (a *App) listEmployeesHandler(w http.ResponseWriter, r *http.Request) {
	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	var department *string
	if dep := r.URL.Query().Get("department"); dep != "" {
		department = &dep
	}

	employees, err := a.HR.ListEmployees(r.Context(), orgID, department)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not list employees")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"employees": employees,
	})
}

// ---- GET /api/v1/hr/employees/{employee_id} ----

// getEmployeeHandler retrieves a single employee by ID.
//
//	@Summary		Get employee
//	@Description	Retrieves an employee record by ID. Requires `hr.employees.read` permission.
//	@Tags			HR
//	@Produce		json
//	@Security		BearerAuth
//	@Param			employee_id	path	string	true	"Employee UUID"
//	@Success		200	{object}	services.Employee
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/employees/{employee_id} [get]
func (a *App) getEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	employeeIDStr := r.PathValue("employee_id")
	employeeID, err := uuid.Parse(employeeIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid employee_id")
		return
	}

	employee, err := a.HR.GetEmployeeByID(r.Context(), employeeID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get employee")
		return
	}
	if employee == nil {
		utils.WriteErr(w, http.StatusNotFound, "employee not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"id":            employee.ID.String(),
		"org_id":        employee.OrgID.String(),
		"user_id":       employee.UserID,
		"employee_code": employee.EmployeeCode,
		"department":    employee.Department,
		"job_title":     employee.JobTitle,
		"salary":        employee.Salary,
		"hired_at":      employee.HiredAt.Format(time.RFC3339),
	})
}

// ---- PATCH /api/v1/hr/employees/{employee_id} ----

type updateEmployeeRequest struct {
	Department *string  `json:"department,omitempty"`
	JobTitle   *string  `json:"job_title,omitempty"`
	Salary     *float64 `json:"salary,omitempty"`
}

// updateEmployeeHandler updates an employee's department, job title, or salary.
//
//	@Summary		Update employee
//	@Description	Updates an employee's department, job title, or salary. Requires `hr.employees.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			employee_id	path	string					true	"Employee UUID"
//	@Param			body		body	updateEmployeeRequest	true	"Fields to update"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/employees/{employee_id} [patch]
func (a *App) updateEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	employeeIDStr := r.PathValue("employee_id")
	employeeID, err := uuid.Parse(employeeIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid employee_id")
		return
	}

	var req updateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	employee, err := a.HR.UpdateEmployee(r.Context(), employeeID, req.Department, req.JobTitle, req.Salary)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not update employee")
		return
	}
	if employee == nil {
		utils.WriteErr(w, http.StatusNotFound, "employee not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"id":            employee.ID.String(),
		"org_id":        employee.OrgID.String(),
		"user_id":       employee.UserID,
		"employee_code": employee.EmployeeCode,
		"department":    employee.Department,
		"job_title":     employee.JobTitle,
		"salary":        employee.Salary,
		"hired_at":      employee.HiredAt.Format(time.RFC3339),
	})
}

// ---- POST /api/v1/hr/payroll/runs ----

type runPayrollRequest struct {
	PayPeriodStart string `json:"pay_period_start"`
	PayPeriodEnd   string `json:"pay_period_end"`
}

// runPayrollHandler creates a payroll run for the organization.
//
//	@Summary		Run payroll
//	@Description	Creates a payroll run, simulating wage calculations, deductions, and tax withholdings for all active employees. Requires `hr.payroll.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	runPayrollRequest	true	"Payroll period"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/payroll/runs [post]
func (a *App) runPayrollHandler(w http.ResponseWriter, r *http.Request) {
	var req runPayrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.PayPeriodStart) || !utils.NotBlank(req.PayPeriodEnd) {
		utils.WriteErr(w, http.StatusBadRequest, "pay_period_start and pay_period_end are required")
		return
	}

	payPeriodStart, err := time.Parse(time.RFC3339, req.PayPeriodStart)
	if err != nil {
		payPeriodStart, err = time.Parse("2006-01-02", req.PayPeriodStart)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid pay_period_start format, use RFC3339 or YYYY-MM-DD")
			return
		}
	}

	payPeriodEnd, err := time.Parse(time.RFC3339, req.PayPeriodEnd)
	if err != nil {
		payPeriodEnd, err = time.Parse("2006-01-02", req.PayPeriodEnd)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid pay_period_end format, use RFC3339 or YYYY-MM-DD")
			return
		}
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	run, err := a.HR.RunPayroll(r.Context(), orgID, services.PayrollRunInput{
		PayPeriodStart: payPeriodStart,
		PayPeriodEnd:   payPeriodEnd,
	})
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not run payroll")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":               run.ID.String(),
		"org_id":           run.OrgID.String(),
		"pay_period_start": run.PayPeriodStart.Format(time.RFC3339),
		"pay_period_end":   run.PayPeriodEnd.Format(time.RFC3339),
		"total_disbursed":  run.TotalDisbursed,
		"status":           run.Status,
		"created_at":       run.CreatedAt.Format(time.RFC3339),
	})
}

// ---- GET /api/v1/hr/payroll/runs ----

// listPayrollRunsHandler lists all payroll runs for the organization.
//
//	@Summary		List payroll runs
//	@Description	Lists all payroll runs for the organization. Requires `hr.payroll.read` permission.
//	@Tags			HR
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		services.PayrollRun
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/payroll/runs [get]
func (a *App) listPayrollRunsHandler(w http.ResponseWriter, r *http.Request) {
	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	runs, err := a.HR.ListPayrollRuns(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not list payroll runs")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"payroll_runs": runs,
	})
}

// ---- GET /api/v1/hr/payroll/runs/{run_id} ----

// getPayrollRunHandler retrieves a single payroll run by ID.
//
//	@Summary		Get payroll run
//	@Description	Retrieves a payroll run by ID. Requires `hr.payroll.read` permission.
//	@Tags			HR
//	@Produce		json
//	@Security		BearerAuth
//	@Param			run_id	path	string	true	"Payroll Run UUID"
//	@Success		200	{object}	services.PayrollRun
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/payroll/runs/{run_id} [get]
func (a *App) getPayrollRunHandler(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid run_id")
		return
	}

	run, err := a.HR.GetPayrollRun(r.Context(), runID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get payroll run")
		return
	}
	if run == nil {
		utils.WriteErr(w, http.StatusNotFound, "payroll run not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"id":               run.ID.String(),
		"org_id":           run.OrgID.String(),
		"pay_period_start": run.PayPeriodStart.Format(time.RFC3339),
		"pay_period_end":   run.PayPeriodEnd.Format(time.RFC3339),
		"total_disbursed":  run.TotalDisbursed,
		"status":           run.Status,
		"created_at":       run.CreatedAt.Format(time.RFC3339),
	})
}
