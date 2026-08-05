package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type Employee struct {
	ID                    uuid.UUID  `json:"id"`
	OrgID                 uuid.UUID  `json:"org_id"`
	UserID                *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode          string     `json:"employee_code"`
	Department            *string    `json:"department,omitempty"`
	JobTitle              *string    `json:"job_title,omitempty"`
	Salary                *float64   `json:"salary,omitempty"`
	HiredAt               time.Time  `json:"hired_at"`
	TaxCountry            *string    `json:"tax_country,omitempty"`
	TaxIDNumber           *string    `json:"tax_id_number,omitempty"`
	FilingStatus          *string    `json:"filing_status,omitempty"`
	WithholdingAllowances *int       `json:"withholding_allowances,omitempty"`
}

type AttendanceLog struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	EmployeeID   uuid.UUID  `json:"employee_id"`
	ClockIn      time.Time  `json:"clock_in"`
	ClockOut     *time.Time `json:"clock_out,omitempty"`
	LocationLat  *float64   `json:"location_lat,omitempty"`
	LocationLong *float64   `json:"location_long,omitempty"`
}

// AttendanceClockInResult is returned by the clock-in endpoint.
type AttendanceClockInResult struct {
	AttendanceLogID  uuid.UUID `json:"attendance_log_id"`
	ClockIn          time.Time `json:"clock_in"`
	IsWithinGeofence bool      `json:"is_within_geofence"`
}

type PayrollRun struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	PayPeriodStart time.Time `json:"pay_period_start"`
	PayPeriodEnd   time.Time `json:"pay_period_end"`
	TotalDisbursed float64   `json:"total_disbursed"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// ── Module 4.2: Shift Management & AI Shift Prediction ──

type ShiftTemplate struct {
	ID                uuid.UUID `json:"id"`
	OrgID             uuid.UUID `json:"org_id"`
	Name              string    `json:"name"`
	StartTime         string    `json:"start_time"`
	EndTime           string    `json:"end_time"`
	DayOfWeek         *int      `json:"day_of_week,omitempty"`
	Department        *string   `json:"department,omitempty"`
	RequiredHeadcount int       `json:"required_headcount"`
	CreatedAt         time.Time `json:"created_at"`
}

type ShiftAssignment struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	ShiftTemplateID uuid.UUID  `json:"shift_template_id"`
	EmployeeID      uuid.UUID  `json:"employee_id"`
	ShiftDate       string     `json:"shift_date"`
	ActualStart     *time.Time `json:"actual_start,omitempty"`
	ActualEnd       *time.Time `json:"actual_end,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ShiftPrediction struct {
	Date               string  `json:"date"`
	Department         *string `json:"department,omitempty"`
	PredictedHeadcount int     `json:"predicted_headcount"`
	Confidence         float64 `json:"confidence"`
	Reasoning          string  `json:"reasoning"`
}

type ClockOutResult struct {
	AttendanceLogID uuid.UUID `json:"attendance_log_id"`
	ClockIn         time.Time `json:"clock_in"`
	ClockOut        time.Time `json:"clock_out"`
	HoursWorked     float64   `json:"hours_worked"`
}

// ── Module 4.3: Payroll Tax Withholding Per Employee ──

type PayrollDisbursement struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	PayrollRunID    uuid.UUID `json:"payroll_run_id"`
	EmployeeID      uuid.UUID `json:"employee_id"`
	GrossPay        float64   `json:"gross_pay"`
	TaxWithheld     float64   `json:"tax_withheld"`
	SocialSecurity  float64   `json:"social_security"`
	OtherDeductions float64   `json:"other_deductions"`
	NetPay          float64   `json:"net_pay"`
	CreatedAt       time.Time `json:"created_at"`
}

type EmployeeTaxProfile struct {
	ID                      uuid.UUID `json:"id"`
	OrgID                   uuid.UUID `json:"org_id"`
	EmployeeID              uuid.UUID `json:"employee_id"`
	TaxCountry              string    `json:"tax_country"`
	TaxIdentificationNumber *string   `json:"tax_identification_number,omitempty"`
	FilingStatus            string    `json:"filing_status"`
	WithholdingAllowances   int       `json:"withholding_allowances"`
	AdditionalWithholding   float64   `json:"additional_withholding"`
	CreatedAt               time.Time `json:"created_at"`
}

type JobApplication struct {
	ID            uuid.UUID `json:"id"`
	OrgID         uuid.UUID `json:"org_id"`
	CandidateName string    `json:"candidate_name"`
	Email         string    `json:"email"`
	ResumeURL     string    `json:"resume_url"`
	AIMatchScore  *int      `json:"ai_match_score,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type KnowledgeBaseDocument struct {
	ID                uuid.UUID `json:"id"`
	OrgID             uuid.UUID `json:"org_id"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	VectorEmbeddingID *string   `json:"vector_embedding_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// ResumeParseResult is returned by the resume parsing endpoint.
type ResumeParseResult struct {
	CandidateName   string   `json:"candidate_name"`
	Email           string   `json:"email"`
	ExtractedSkills []string `json:"extracted_skills"`
	AIMatchScore    int      `json:"ai_match_score"`
}

// SourceDocument is a matching knowledge base document with a relevance score.
type SourceDocument struct {
	Title          string  `json:"title"`
	RelevanceScore float64 `json:"relevance_score"`
}

// KnowledgeSearchResult is returned by the knowledge search endpoint.
type KnowledgeSearchResult struct {
	AIAnswer        string           `json:"ai_answer"`
	SourceDocuments []SourceDocument `json:"source_documents"`
}

// ---- Repository ----

type hrRepository struct {
	pool *pgxpool.Pool
}

func newHRRepository(pool *pgxpool.Pool) *hrRepository {
	return &hrRepository{pool: pool}
}

func (r *hrRepository) GetEmployeeByUserAndOrg(ctx context.Context, orgID, userID uuid.UUID) (*Employee, error) {
	e := &Employee{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, employee_code, department, job_title, salary, hired_at
		FROM employees WHERE org_id = $1 AND user_id = $2
	`, orgID, userID).Scan(&e.ID, &e.OrgID, &e.UserID, &e.EmployeeCode, &e.Department, &e.JobTitle, &e.Salary, &e.HiredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (r *hrRepository) CreateEmployee(ctx context.Context, e *Employee) error {
	e.ID = uuid.New()
	if e.EmployeeCode == "" {
		e.EmployeeCode = fmt.Sprintf("EMP-%s", uuid.New().String()[:8])
	}
	if e.HiredAt.IsZero() {
		e.HiredAt = time.Now()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO employees (id, org_id, user_id, employee_code, department, job_title, salary, hired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, e.ID, e.OrgID, e.UserID, e.EmployeeCode, e.Department, e.JobTitle, e.Salary, e.HiredAt)
	return err
}

func (r *hrRepository) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*Employee, error) {
	e := &Employee{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, employee_code, department, job_title, salary, hired_at
		FROM employees WHERE id = $1
	`, id).Scan(&e.ID, &e.OrgID, &e.UserID, &e.EmployeeCode, &e.Department, &e.JobTitle, &e.Salary, &e.HiredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (r *hrRepository) ListEmployees(ctx context.Context, orgID uuid.UUID, department *string) ([]Employee, error) {
	var rows pgx.Rows
	var err error
	if department != nil && *department != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, org_id, user_id, employee_code, department, job_title, salary, hired_at
			FROM employees WHERE org_id = $1 AND department = $2
			ORDER BY hired_at DESC
		`, orgID, *department)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, org_id, user_id, employee_code, department, job_title, salary, hired_at
			FROM employees WHERE org_id = $1
			ORDER BY hired_at DESC
		`, orgID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.OrgID, &e.UserID, &e.EmployeeCode, &e.Department, &e.JobTitle, &e.Salary, &e.HiredAt); err != nil {
			return nil, err
		}
		employees = append(employees, e)
	}
	if employees == nil {
		employees = []Employee{}
	}
	return employees, rows.Err()
}

func (r *hrRepository) UpdateEmployee(ctx context.Context, id uuid.UUID, department, jobTitle *string, salary *float64) (*Employee, error) {
	e := &Employee{}
	err := r.pool.QueryRow(ctx, `
		UPDATE employees SET
			department = COALESCE($2, department),
			job_title = COALESCE($3, job_title),
			salary = COALESCE($4, salary)
		WHERE id = $1
		RETURNING id, org_id, user_id, employee_code, department, job_title, salary, hired_at
	`, id, department, jobTitle, salary).Scan(&e.ID, &e.OrgID, &e.UserID, &e.EmployeeCode, &e.Department, &e.JobTitle, &e.Salary, &e.HiredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (r *hrRepository) CreateAttendanceLog(ctx context.Context, al *AttendanceLog) error {
	al.ID = uuid.New()
	al.ClockIn = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO attendance_logs (id, org_id, employee_id, clock_in, clock_out, location_lat, location_long)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, al.ID, al.OrgID, al.EmployeeID, al.ClockIn, al.ClockOut, al.LocationLat, al.LocationLong)
	return err
}

func (r *hrRepository) CreateJobApplication(ctx context.Context, app *JobApplication) error {
	app.ID = uuid.New()
	app.CreatedAt = time.Now()
	if app.Status == "" {
		app.Status = "applied"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO job_applications (id, org_id, candidate_name, email, resume_url, ai_match_score, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, app.ID, app.OrgID, app.CandidateName, app.Email, app.ResumeURL, app.AIMatchScore, app.Status, app.CreatedAt)
	return err
}

func (r *hrRepository) SearchKnowledgeBase(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]KnowledgeBaseDocument, error) {
	if limit <= 0 {
		limit = 5
	}
	// Simple full-text search using ILIKE on title and content
	searchPattern := "%" + strings.ReplaceAll(query, " ", "%") + "%"
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, title, content, vector_embedding_id, created_at
		FROM knowledge_base_documents
		WHERE org_id = $1 AND (title ILIKE $2 OR content ILIKE $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, orgID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []KnowledgeBaseDocument
	for rows.Next() {
		var d KnowledgeBaseDocument
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Title, &d.Content, &d.VectorEmbeddingID, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if docs == nil {
		docs = []KnowledgeBaseDocument{}
	}
	return docs, rows.Err()
}

func (r *hrRepository) CreatePayrollRun(ctx context.Context, pr *PayrollRun) error {
	pr.ID = uuid.New()
	pr.CreatedAt = time.Now()
	if pr.Status == "" {
		pr.Status = "completed"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO payroll_runs (id, org_id, pay_period_start, pay_period_end, total_disbursed, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pr.ID, pr.OrgID, pr.PayPeriodStart, pr.PayPeriodEnd, pr.TotalDisbursed, pr.Status, pr.CreatedAt)
	return err
}

func (r *hrRepository) GetPayrollRunByID(ctx context.Context, id uuid.UUID) (*PayrollRun, error) {
	pr := &PayrollRun{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, pay_period_start, pay_period_end, total_disbursed, status, created_at
		FROM payroll_runs WHERE id = $1
	`, id).Scan(&pr.ID, &pr.OrgID, &pr.PayPeriodStart, &pr.PayPeriodEnd, &pr.TotalDisbursed, &pr.Status, &pr.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return pr, err
}

func (r *hrRepository) ListPayrollRuns(ctx context.Context, orgID uuid.UUID) ([]PayrollRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, pay_period_start, pay_period_end, total_disbursed, status, created_at
		FROM payroll_runs WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []PayrollRun
	for rows.Next() {
		var pr PayrollRun
		if err := rows.Scan(&pr.ID, &pr.OrgID, &pr.PayPeriodStart, &pr.PayPeriodEnd, &pr.TotalDisbursed, &pr.Status, &pr.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, pr)
	}
	if runs == nil {
		runs = []PayrollRun{}
	}
	return runs, rows.Err()
}

// ── Module 4.2: Shift repository methods ──

func (r *hrRepository) CreateShiftTemplate(ctx context.Context, t *ShiftTemplate) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO shift_templates (id, org_id, name, start_time, end_time, day_of_week, department, required_headcount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, t.ID, t.OrgID, t.Name, t.StartTime, t.EndTime, t.DayOfWeek, t.Department, t.RequiredHeadcount, t.CreatedAt)
	return err
}

func (r *hrRepository) GetShiftTemplates(ctx context.Context, orgID uuid.UUID, department *string) ([]ShiftTemplate, error) {
	var rows pgx.Rows
	var err error
	if department != nil && *department != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, org_id, name, start_time, end_time, day_of_week, department, required_headcount, created_at
			FROM shift_templates WHERE org_id = $1 AND department = $2
			ORDER BY name
		`, orgID, *department)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, org_id, name, start_time, end_time, day_of_week, department, required_headcount, created_at
			FROM shift_templates WHERE org_id = $1
			ORDER BY name
		`, orgID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ShiftTemplate
	for rows.Next() {
		var t ShiftTemplate
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.StartTime, &t.EndTime, &t.DayOfWeek, &t.Department, &t.RequiredHeadcount, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []ShiftTemplate{}
	}
	return templates, rows.Err()
}

func (r *hrRepository) CreateShiftAssignment(ctx context.Context, a *ShiftAssignment) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	if a.Status == "" {
		a.Status = "scheduled"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO shift_assignments (id, org_id, shift_template_id, employee_id, shift_date, actual_start, actual_end, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, a.ID, a.OrgID, a.ShiftTemplateID, a.EmployeeID, a.ShiftDate, a.ActualStart, a.ActualEnd, a.Status, a.CreatedAt)
	return err
}

func (r *hrRepository) GetShiftAssignments(ctx context.Context, orgID, employeeID uuid.UUID, fromDate, toDate string) ([]ShiftAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, shift_template_id, employee_id, shift_date, actual_start, actual_end, status, created_at
		FROM shift_assignments
		WHERE org_id = $1 AND employee_id = $2 AND shift_date >= $3 AND shift_date <= $4
		ORDER BY shift_date, created_at
	`, orgID, employeeID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []ShiftAssignment
	for rows.Next() {
		var a ShiftAssignment
		if err := rows.Scan(&a.ID, &a.OrgID, &a.ShiftTemplateID, &a.EmployeeID, &a.ShiftDate, &a.ActualStart, &a.ActualEnd, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	if assignments == nil {
		assignments = []ShiftAssignment{}
	}
	return assignments, rows.Err()
}

func (r *hrRepository) UpdateAttendanceClockOut(ctx context.Context, logID uuid.UUID, clockOut time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE attendance_logs SET clock_out = $2 WHERE id = $1
	`, logID, clockOut)
	return err
}

func (r *hrRepository) GetOpenAttendanceLog(ctx context.Context, orgID, employeeID uuid.UUID) (*AttendanceLog, error) {
	al := &AttendanceLog{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, employee_id, clock_in, clock_out, location_lat, location_long
		FROM attendance_logs
		WHERE org_id = $1 AND employee_id = $2 AND clock_out IS NULL
		ORDER BY clock_in DESC
		LIMIT 1
	`, orgID, employeeID).Scan(&al.ID, &al.OrgID, &al.EmployeeID, &al.ClockIn, &al.ClockOut, &al.LocationLat, &al.LocationLong)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return al, err
}

// ── Module 4.3: Payroll tax repository methods ──

func (r *hrRepository) CreatePayrollDisbursement(ctx context.Context, d *PayrollDisbursement) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO payroll_disbursements (id, org_id, payroll_run_id, employee_id, gross_pay, tax_withheld, social_security, other_deductions, net_pay, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, d.ID, d.OrgID, d.PayrollRunID, d.EmployeeID, d.GrossPay, d.TaxWithheld, d.SocialSecurity, d.OtherDeductions, d.NetPay, d.CreatedAt)
	return err
}

func (r *hrRepository) GetPayrollDisbursements(ctx context.Context, payrollRunID uuid.UUID) ([]PayrollDisbursement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, payroll_run_id, employee_id, gross_pay, tax_withheld, social_security, other_deductions, net_pay, created_at
		FROM payroll_disbursements WHERE payroll_run_id = $1
		ORDER BY created_at
	`, payrollRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disbursements []PayrollDisbursement
	for rows.Next() {
		var d PayrollDisbursement
		if err := rows.Scan(&d.ID, &d.OrgID, &d.PayrollRunID, &d.EmployeeID, &d.GrossPay, &d.TaxWithheld, &d.SocialSecurity, &d.OtherDeductions, &d.NetPay, &d.CreatedAt); err != nil {
			return nil, err
		}
		disbursements = append(disbursements, d)
	}
	if disbursements == nil {
		disbursements = []PayrollDisbursement{}
	}
	return disbursements, rows.Err()
}

func (r *hrRepository) CreateEmployeeTaxProfile(ctx context.Context, p *EmployeeTaxProfile) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO employee_tax_profiles (id, org_id, employee_id, tax_country, tax_identification_number, filing_status, withholding_allowances, additional_withholding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (employee_id) DO UPDATE SET
			tax_country = EXCLUDED.tax_country,
			tax_identification_number = EXCLUDED.tax_identification_number,
			filing_status = EXCLUDED.filing_status,
			withholding_allowances = EXCLUDED.withholding_allowances,
			additional_withholding = EXCLUDED.additional_withholding
	`, p.ID, p.OrgID, p.EmployeeID, p.TaxCountry, p.TaxIdentificationNumber, p.FilingStatus, p.WithholdingAllowances, p.AdditionalWithholding, p.CreatedAt)
	return err
}

func (r *hrRepository) GetEmployeeTaxProfile(ctx context.Context, orgID, employeeID uuid.UUID) (*EmployeeTaxProfile, error) {
	p := &EmployeeTaxProfile{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, employee_id, tax_country, tax_identification_number, filing_status, withholding_allowances, additional_withholding, created_at
		FROM employee_tax_profiles WHERE org_id = $1 AND employee_id = $2
	`, orgID, employeeID).Scan(&p.ID, &p.OrgID, &p.EmployeeID, &p.TaxCountry, &p.TaxIdentificationNumber, &p.FilingStatus, &p.WithholdingAllowances, &p.AdditionalWithholding, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ---- Error sentinel ----

// PayrollRunInput is the input for running payroll.
type PayrollRunInput struct {
	PayPeriodStart time.Time `json:"pay_period_start"`
	PayPeriodEnd   time.Time `json:"pay_period_end"`
}

var (
	ErrEmployeeNotFound      = errors.New("employee not found")
	ErrPayrollRunNotFound    = errors.New("payroll run not found")
	ErrNoOpenAttendanceLog   = errors.New("no open attendance log found")
	ErrShiftTemplateNotFound = errors.New("shift template not found")
)

// ---- Service ----

type HRService struct {
	repo *hrRepository
}

func NewHRService(pool *pgxpool.Pool) *HRService {
	return &HRService{repo: newHRRepository(pool)}
}

// CreateEmployee creates a new employee record in the organization.
func (s *HRService) CreateEmployee(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, employeeCode string, department, jobTitle *string, salary *float64, hiredAt time.Time) (*Employee, error) {
	e := &Employee{
		OrgID:        orgID,
		UserID:       userID,
		EmployeeCode: employeeCode,
		Department:   department,
		JobTitle:     jobTitle,
		Salary:       salary,
		HiredAt:      hiredAt,
	}
	if err := s.repo.CreateEmployee(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// GetEmployeeByID retrieves an employee by ID.
func (s *HRService) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*Employee, error) {
	return s.repo.GetEmployeeByID(ctx, id)
}

// ListEmployees lists all employees in an org, optionally filtered by department.
func (s *HRService) ListEmployees(ctx context.Context, orgID uuid.UUID, department *string) ([]Employee, error) {
	return s.repo.ListEmployees(ctx, orgID, department)
}

// UpdateEmployee updates an employee's department, job title, or salary.
func (s *HRService) UpdateEmployee(ctx context.Context, id uuid.UUID, department, jobTitle *string, salary *float64) (*Employee, error) {
	return s.repo.UpdateEmployee(ctx, id, department, jobTitle, salary)
}

// RunPayroll creates a payroll run with per-employee tax withholding calculations.
// For each employee, it looks up their tax profile (or uses defaults), calculates
// tax/social security using progressive tax brackets, stores individual
// disbursements, and computes the total.
func (s *HRService) RunPayroll(ctx context.Context, orgID uuid.UUID, input PayrollRunInput) (*PayrollRun, error) {
	// Fetch all employees in the org
	employees, err := s.repo.ListEmployees(ctx, orgID, nil)
	if err != nil {
		return nil, err
	}

	pr := &PayrollRun{
		OrgID:          orgID,
		PayPeriodStart: input.PayPeriodStart,
		PayPeriodEnd:   input.PayPeriodEnd,
	}
	if err := s.repo.CreatePayrollRun(ctx, pr); err != nil {
		return nil, err
	}

	var totalDisbursed float64
	for _, emp := range employees {
		if emp.Salary == nil {
			continue
		}
		grossPay := *emp.Salary

		// Look up tax profile, fall back to defaults if none exists
		profile, _ := s.repo.GetEmployeeTaxProfile(ctx, orgID, emp.ID)
		if profile == nil {
			profile = &EmployeeTaxProfile{
				TaxCountry:            "US",
				FilingStatus:          "single",
				WithholdingAllowances: 0,
				AdditionalWithholding: 0,
			}
		}

		taxWithheld, socialSecurity, netPay := s.CalculatePayrollTax(grossPay, profile)

		d := &PayrollDisbursement{
			OrgID:          orgID,
			PayrollRunID:   pr.ID,
			EmployeeID:     emp.ID,
			GrossPay:       grossPay,
			TaxWithheld:    taxWithheld,
			SocialSecurity: socialSecurity,
			NetPay:         netPay,
		}
		if err := s.repo.CreatePayrollDisbursement(ctx, d); err != nil {
			return nil, err
		}
		totalDisbursed += netPay
	}

	// Update the payroll run with the computed total
	pr.TotalDisbursed = totalDisbursed
	pr.Status = "completed"
	// Re-fetch to get the persisted record with total
	return s.repo.GetPayrollRunByID(ctx, pr.ID)
}

// GetPayrollRun retrieves a payroll run by ID.
func (s *HRService) GetPayrollRun(ctx context.Context, id uuid.UUID) (*PayrollRun, error) {
	return s.repo.GetPayrollRunByID(ctx, id)
}

// ListPayrollRuns lists all payroll runs for an org.
func (s *HRService) ListPayrollRuns(ctx context.Context, orgID uuid.UUID) ([]PayrollRun, error) {
	return s.repo.ListPayrollRuns(ctx, orgID)
}

// ClockIn records an attendance clock-in event for the employee associated with the
// given user, validates geofence, and returns the attendance log result.
func (s *HRService) ClockIn(ctx context.Context, orgID, userID uuid.UUID, latitude, longitude float64) (*AttendanceClockInResult, error) {
	// Look up employee by org + user
	employee, err := s.repo.GetEmployeeByUserAndOrg(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		return nil, ErrEmployeeNotFound
	}

	// Create the attendance log
	al := &AttendanceLog{
		OrgID:        orgID,
		EmployeeID:   employee.ID,
		LocationLat:  &latitude,
		LocationLong: &longitude,
	}
	if err := s.repo.CreateAttendanceLog(ctx, al); err != nil {
		return nil, err
	}

	// Simulate geofence check — in production this would compare against office
	// location(s) configured for the organization.
	isWithinGeofence := simulateGeofence(latitude, longitude)

	return &AttendanceClockInResult{
		AttendanceLogID:  al.ID,
		ClockIn:          al.ClockIn,
		IsWithinGeofence: isWithinGeofence,
	}, nil
}

// ── Module 4.2: Shift Management & AI Shift Prediction ──

// ClockOut finds the employee's open attendance log, records the clock-out time,
// and returns the hours worked.
func (s *HRService) ClockOut(ctx context.Context, orgID, userID uuid.UUID) (*ClockOutResult, error) {
	employee, err := s.repo.GetEmployeeByUserAndOrg(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		return nil, ErrEmployeeNotFound
	}

	log, err := s.repo.GetOpenAttendanceLog(ctx, orgID, employee.ID)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, ErrNoOpenAttendanceLog
	}

	now := time.Now()
	if err := s.repo.UpdateAttendanceClockOut(ctx, log.ID, now); err != nil {
		return nil, err
	}

	hoursWorked := now.Sub(log.ClockIn).Hours()

	return &ClockOutResult{
		AttendanceLogID: log.ID,
		ClockIn:         log.ClockIn,
		ClockOut:        now,
		HoursWorked:     mathRound(hoursWorked, 2),
	}, nil
}

// CreateShiftTemplate creates a new shift template for the organization.
func (s *HRService) CreateShiftTemplate(ctx context.Context, orgID uuid.UUID, name string, startTime, endTime string, dayOfWeek *int, department *string, requiredHeadcount int) (*ShiftTemplate, error) {
	if requiredHeadcount <= 0 {
		requiredHeadcount = 1
	}
	t := &ShiftTemplate{
		OrgID:             orgID,
		Name:              name,
		StartTime:         startTime,
		EndTime:           endTime,
		DayOfWeek:         dayOfWeek,
		Department:        department,
		RequiredHeadcount: requiredHeadcount,
	}
	if err := s.repo.CreateShiftTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// AssignShift assigns an employee to a shift template on a specific date.
func (s *HRService) AssignShift(ctx context.Context, orgID, employeeID, shiftTemplateID uuid.UUID, date string) (*ShiftAssignment, error) {
	a := &ShiftAssignment{
		OrgID:           orgID,
		ShiftTemplateID: shiftTemplateID,
		EmployeeID:      employeeID,
		ShiftDate:       date,
		Status:          "scheduled",
	}
	if err := s.repo.CreateShiftAssignment(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// PredictShiftNeeds returns AI-simulated staffing predictions for a date range.
// It looks at historical attendance patterns and seasonality to estimate
// required headcount per day.
func (s *HRService) PredictShiftNeeds(ctx context.Context, orgID uuid.UUID, department *string, fromDate, toDate string) ([]ShiftPrediction, error) {
	// Parse the date range
	from, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid from_date: %w", err)
	}
	to, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return nil, fmt.Errorf("invalid to_date: %w", err)
	}

	// Get shift templates for the department to understand baseline headcount
	templates, err := s.repo.GetShiftTemplates(ctx, orgID, department)
	if err != nil {
		return nil, err
	}

	// Calculate base headcount from templates
	baseHeadcount := 1
	if len(templates) > 0 {
		totalHC := 0
		for _, t := range templates {
			totalHC += t.RequiredHeadcount
		}
		baseHeadcount = totalHC / len(templates)
		if baseHeadcount < 1 {
			baseHeadcount = 1
		}
	}

	var predictions []ShiftPrediction
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		predicted, confidence, reasoning := simulateShiftPrediction(d, baseHeadcount, department)
		predictions = append(predictions, ShiftPrediction{
			Date:               dateStr,
			Department:         department,
			PredictedHeadcount: predicted,
			Confidence:         confidence,
			Reasoning:          reasoning,
		})
	}

	return predictions, nil
}

// GetEmployeeSchedule returns all shift assignments for an employee in a date range.
func (s *HRService) GetEmployeeSchedule(ctx context.Context, orgID, employeeID uuid.UUID, fromDate, toDate string) ([]ShiftAssignment, error) {
	return s.repo.GetShiftAssignments(ctx, orgID, employeeID, fromDate, toDate)
}

// ── Module 4.3: Payroll Tax Withholding Per Employee ──

// SetEmployeeTaxProfile creates or updates the tax profile for an employee.
func (s *HRService) SetEmployeeTaxProfile(ctx context.Context, orgID, employeeID uuid.UUID, taxCountry, taxID, filingStatus string, allowances int, additionalWithholding float64) (*EmployeeTaxProfile, error) {
	var taxIDPtr *string
	if taxID != "" {
		taxIDPtr = &taxID
	}
	p := &EmployeeTaxProfile{
		OrgID:                   orgID,
		EmployeeID:              employeeID,
		TaxCountry:              taxCountry,
		TaxIdentificationNumber: taxIDPtr,
		FilingStatus:            filingStatus,
		WithholdingAllowances:   allowances,
		AdditionalWithholding:   additionalWithholding,
	}
	if err := s.repo.CreateEmployeeTaxProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// CalculatePayrollTax computes progressive tax withholding, social security,
// and net pay for a given gross pay and employee tax profile.
// Simulates US-style progressive tax brackets.
func (s *HRService) CalculatePayrollTax(grossPay float64, profile *EmployeeTaxProfile) (taxWithheld, socialSecurity, netPay float64) {
	if profile == nil {
		profile = &EmployeeTaxProfile{
			TaxCountry:            "US",
			FilingStatus:          "single",
			WithholdingAllowances: 0,
			AdditionalWithholding: 0,
		}
	}

	// Social Security: 6.2% up to the wage base limit (simplified)
	const ssRate = 0.062
	const ssWageBase = 160200.0
	socialSecurity = mathRound(grossPay*ssRate, 2)
	if grossPay > ssWageBase {
		socialSecurity = mathRound(ssWageBase*ssRate, 2)
	}

	// Standard deduction based on filing status
	var standardDeduction float64
	switch profile.FilingStatus {
	case "married_joint":
		standardDeduction = 27700
	case "head_of_household":
		standardDeduction = 20800
	default: // single, married_separate
		standardDeduction = 13850
	}

	// Personal allowance reduction (each allowance reduces taxable income by ~$4,300)
	allowanceDeduction := float64(profile.WithholdingAllowances) * 4300

	taxableIncome := grossPay - standardDeduction - allowanceDeduction
	if taxableIncome < 0 {
		taxableIncome = 0
	}

	// Progressive tax brackets (simplified US federal, annualized from monthly)
	// We treat grossPay as monthly and annualize for bracket calculation
	annualIncome := taxableIncome * 12

	var annualTax float64
	switch {
	case annualIncome <= 11000:
		annualTax = annualIncome * 0.10
	case annualIncome <= 44725:
		annualTax = 1100 + (annualIncome-11000)*0.12
	case annualIncome <= 95375:
		annualTax = 5147 + (annualIncome-44725)*0.22
	case annualIncome <= 182100:
		annualTax = 16290 + (annualIncome-95375)*0.24
	case annualIncome <= 231250:
		annualTax = 37104 + (annualIncome-182100)*0.32
	case annualIncome <= 578125:
		annualTax = 52832 + (annualIncome-231250)*0.35
	default:
		annualTax = 174238.25 + (annualIncome-578125)*0.37
	}

	// Monthly tax withholding
	taxWithheld = mathRound(annualTax/12, 2)

	// Add additional withholding requested by employee
	taxWithheld += profile.AdditionalWithholding

	// Net pay
	netPay = mathRound(grossPay-taxWithheld-socialSecurity, 2)
	if netPay < 0 {
		netPay = 0
	}

	return taxWithheld, socialSecurity, netPay
}

// GetPayrollRunDetail retrieves a payroll run with its per-employee disbursements.
func (s *HRService) GetPayrollRunDetail(ctx context.Context, runID uuid.UUID) (*PayrollRun, []PayrollDisbursement, error) {
	pr, err := s.repo.GetPayrollRunByID(ctx, runID)
	if err != nil {
		return nil, nil, err
	}
	if pr == nil {
		return nil, nil, ErrPayrollRunNotFound
	}

	disbursements, err := s.repo.GetPayrollDisbursements(ctx, runID)
	if err != nil {
		return nil, nil, err
	}

	return pr, disbursements, nil
}

// ParseResume reads a resume file (PDF/Docx), simulates AI parsing to extract
// candidate details and skills, scores the match against a job description,
// and stores the application.
func (s *HRService) ParseResume(ctx context.Context, orgID uuid.UUID, jobDescriptionID uuid.UUID, resumeBytes []byte, fileName string) (*ResumeParseResult, error) {
	// Simulate AI extraction from the resume "file"
	name, email, skills := simulateResumeExtraction(resumeBytes, fileName)

	// Simulate AI match scoring against the job description
	matchScore := simulateAIMatchScore(skills)

	// Store the job application
	aiMatchScore := matchScore
	app := &JobApplication{
		OrgID:         orgID,
		CandidateName: name,
		Email:         email,
		ResumeURL:     fmt.Sprintf("resumes/%s/%s", orgID.String(), fileName),
		AIMatchScore:  &aiMatchScore,
	}
	if err := s.repo.CreateJobApplication(ctx, app); err != nil {
		return nil, err
	}

	return &ResumeParseResult{
		CandidateName:   name,
		Email:           email,
		ExtractedSkills: skills,
		AIMatchScore:    matchScore,
	}, nil
}

// SearchKnowledge performs a semantic search over the organization's knowledge
// base and returns a synthesized answer with source documents.
func (s *HRService) SearchKnowledge(ctx context.Context, orgID uuid.UUID, query string) (*KnowledgeSearchResult, error) {
	docs, err := s.repo.SearchKnowledgeBase(ctx, orgID, query, 5)
	if err != nil {
		return nil, err
	}

	// Simulate RAG: synthesize an AI answer and compute relevance scores
	sources := make([]SourceDocument, 0, len(docs))
	var answerBuilder strings.Builder

	answerBuilder.WriteString(synthesizeAIAnswer(query, docs))
	answerBuilder.WriteString("\n\nThe following documents provide more detail:")

	for i, d := range docs {
		relevance := simulateRelevanceScore(query, d.Content)
		sources = append(sources, SourceDocument{
			Title:          d.Title,
			RelevanceScore: relevance,
		})

		if i < 3 {
			answerBuilder.WriteString(fmt.Sprintf("\n\n**%s** (relevance: %.0f%%)", d.Title, relevance*100))
			// Include a short excerpt
			excerpt := d.Content
			if len(excerpt) > 200 {
				excerpt = excerpt[:200] + "..."
			}
			answerBuilder.WriteString("\n> ")
			answerBuilder.WriteString(excerpt)
		}
	}

	return &KnowledgeSearchResult{
		AIAnswer:        answerBuilder.String(),
		SourceDocuments: sources,
	}, nil
}

// ---- AI simulation helpers ----

// simulateGeofence returns whether the given coordinates are within a simulated
// geofenced office location. In production this would query the org's configured
// office locations and use a point-in-polygon algorithm.
func simulateGeofence(lat, lng float64) bool {
	// Simulated office center (San Francisco downtown area)
	const officeLat = 37.7749
	const officeLng = -122.4194
	const radiusDeg = 0.01 // ~1.1 km

	latDiff := lat - officeLat
	lngDiff := lng - officeLng

	return (latDiff*latDiff + lngDiff*lngDiff) <= (radiusDeg * radiusDeg)
}

// simulateResumeExtraction simulates AI-powered resume parsing. In production
// this would call an NLP service to extract structured data from PDF/Docx files.
func simulateResumeExtraction(data []byte, fileName string) (name, email string, skills []string) {
	// Use the file name to deterministically-ish generate a name
	base := strings.TrimSuffix(strings.TrimSuffix(fileName, ".pdf"), ".docx")
	parts := strings.Split(strings.ReplaceAll(base, "_", " "), " ")
	if len(parts) >= 2 {
		name = parts[0] + " " + parts[len(parts)-1]
	} else if len(parts) == 1 && parts[0] != "" {
		name = parts[0]
	} else {
		name = "John Doe"
	}
	name = titleCase(name)

	// Generate email from name
	emailName := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	email = emailName + "@email.com"

	// Simulate skill extraction based on file size heuristic
	allSkills := []string{
		"Python", "Java", "JavaScript", "TypeScript", "Go", "Rust",
		"React", "Angular", "Vue.js", "Node.js", "PostgreSQL", "MongoDB",
		"Docker", "Kubernetes", "AWS", "Azure", "GCP", "Terraform",
		"CI/CD", "Git", "Agile", "Scrum", "Machine Learning", "Data Analysis",
		"SQL", "REST APIs", "GraphQL", "Microservices", "Linux", "DevOps",
	}
	numSkills := 3 + rand.Intn(5) // 3–7 skills
	rand.Shuffle(len(allSkills), func(i, j int) {
		allSkills[i], allSkills[j] = allSkills[j], allSkills[i]
	})
	skills = allSkills[:numSkills]

	return name, email, skills
}

// simulateAIMatchScore generates a simulated AI match score (0–100) for a
// candidate against a job description based on extracted skills.
func simulateAIMatchScore(skills []string) int {
	baseScore := 40 + len(skills)*3
	jitter := rand.Intn(20) - 10 // -10 to +9
	score := baseScore + jitter
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// simulateRelevanceScore computes a simulated relevance score between a query
// and a document's content based on shared term overlap.
func simulateRelevanceScore(query, content string) float64 {
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	// Split query into terms and count matches
	terms := strings.Fields(queryLower)
	matchCount := 0
	for _, term := range terms {
		if len(term) > 2 && strings.Contains(contentLower, term) {
			matchCount++
		}
	}

	score := float64(matchCount) / float64(len(terms))
	// Add some randomness for realism
	score += (rand.Float64() - 0.5) * 0.2
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.05 {
		score = 0.05 + rand.Float64()*0.1
	}
	return mathRound(score, 4)
}

// synthesizeAIAnswer creates a simulated RAG-generated answer based on matched documents.
func synthesizeAIAnswer(query string, docs []KnowledgeBaseDocument) string {
	if len(docs) == 0 {
		return fmt.Sprintf("I couldn't find any specific documents in our knowledge base regarding \"%s\". Please try rephrasing your question or contact your HR representative for assistance.", query)
	}

	// Pick the best-matching document title to reference
	topDoc := docs[0]

	return fmt.Sprintf(
		"Based on our knowledge base, here is the answer to \"%s\": The document **%s** addresses this topic. %s",
		query,
		topDoc.Title,
		simulateAnswerFromContent(topDoc.Content),
	)
}

// simulateAnswerFromContent generates a short simulated answer snippet from document content.
func simulateAnswerFromContent(content string) string {
	if len(content) > 300 {
		content = content[:300]
	}
	// Strip any trailing incomplete sentence
	if idx := strings.LastIndex(content, "."); idx > len(content)/2 {
		content = content[:idx+1]
	}
	return content
}

// simulateShiftPrediction generates an AI-simulated staffing prediction for a
// given date. It considers day-of-week patterns, seasonality (month effects),
// and adds realistic jitter for confidence scoring.
func simulateShiftPrediction(date time.Time, baseHeadcount int, department *string) (predicted int, confidence float64, reasoning string) {
	weekday := date.Weekday()
	month := date.Month()

	// Base prediction starts from template headcount
	predicted = baseHeadcount

	// Day-of-week adjustment: weekdays need more staff, weekends less
	switch weekday {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
		predicted += 2
	case time.Friday:
		predicted += 1
	case time.Saturday:
		predicted -= 1
	case time.Sunday:
		predicted -= 2
	}

	// Seasonal/month adjustment: retail/hospitality peaks in Nov-Dec, summer dips
	switch month {
	case time.November, time.December:
		predicted += 3
	case time.June, time.July, time.August:
		predicted -= 1
	}

	// Ensure minimum of 1
	if predicted < 1 {
		predicted = 1
	}

	// Confidence: higher on weekdays with clear patterns, lower on weekends/holidays
	confidence = 0.85
	if weekday == time.Saturday || weekday == time.Sunday {
		confidence = 0.65
	}
	// Add small random jitter to confidence
	confidence += (rand.Float64() - 0.5) * 0.1
	if confidence > 0.95 {
		confidence = 0.95
	}
	if confidence < 0.5 {
		confidence = 0.5
	}
	confidence = mathRound(confidence, 2)

	// Build reasoning string
	var deptStr string
	if department != nil && *department != "" {
		deptStr = fmt.Sprintf(" for %s department", *department)
	}
	reasoning = fmt.Sprintf(
		"Based on historical attendance patterns%s: %s is a %s (baseline %d, adjusted for day-of-week and seasonal factors in %s).",
		deptStr, date.Format("Jan 2"), weekday.String(), baseHeadcount, month.String(),
	)

	return predicted, confidence, reasoning
}

// titleCase converts a lowercase string to title case (first letter of each word uppercase).
func titleCase(s string) string {
	result := make([]byte, 0, len(s))
	capitalize := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '-' || c == '_' {
			result = append(result, c)
			capitalize = true
		} else if capitalize && c >= 'a' && c <= 'z' {
			result = append(result, c-32) // uppercase
			capitalize = false
		} else {
			result = append(result, c)
			capitalize = false
		}
	}
	return string(result)
}
func mathRound(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}
