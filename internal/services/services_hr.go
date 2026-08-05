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
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode string     `json:"employee_code"`
	Department   *string    `json:"department,omitempty"`
	JobTitle     *string    `json:"job_title,omitempty"`
	Salary       *float64   `json:"salary,omitempty"`
	HiredAt      time.Time  `json:"hired_at"`
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

// ---- Error sentinel ----

// PayrollRunInput is the input for running payroll.
type PayrollRunInput struct {
	PayPeriodStart time.Time `json:"pay_period_start"`
	PayPeriodEnd   time.Time `json:"pay_period_end"`
}

var (
	ErrEmployeeNotFound   = errors.New("employee not found")
	ErrPayrollRunNotFound = errors.New("payroll run not found")
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

// RunPayroll creates a payroll run, simulating wage/deduction/tax calculations
// for all active employees, and returns the payroll run with total disbursed.
func (s *HRService) RunPayroll(ctx context.Context, orgID uuid.UUID, input PayrollRunInput) (*PayrollRun, error) {
	// Fetch all employees in the org
	employees, err := s.repo.ListEmployees(ctx, orgID, nil)
	if err != nil {
		return nil, err
	}

	// Simulate payroll calculation: sum salaries and apply deductions/taxes
	var totalDisbursed float64
	for _, emp := range employees {
		if emp.Salary != nil {
			// Simulate: 70% net pay after deductions and tax withholdings
			netPay := *emp.Salary * 0.70
			totalDisbursed += netPay
		}
	}

	pr := &PayrollRun{
		OrgID:          orgID,
		PayPeriodStart: input.PayPeriodStart,
		PayPeriodEnd:   input.PayPeriodEnd,
		TotalDisbursed: totalDisbursed,
	}
	if err := s.repo.CreatePayrollRun(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
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
