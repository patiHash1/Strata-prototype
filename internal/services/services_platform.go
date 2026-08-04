package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

// AICopilotConversation stores a user's text-to-SQL copilot interaction.
type AICopilotConversation struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	PromptText      string     `json:"prompt_text"`
	GeneratedSQL    *string    `json:"generated_sql,omitempty"`
	ResponsePayload []byte     `json:"response_payload,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AICopilotQueryResult is returned by the text-to-SQL copilot endpoint.
type AICopilotQueryResult struct {
	GeneratedSQL        string                   `json:"generated_sql"`
	DataTable           []map[string]interface{} `json:"data_table"`
	ChartRecommendation string                   `json:"chart_recommendation"`
}

// LowCodeWorkflow represents an automated workflow definition.
type LowCodeWorkflow struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	TriggerEvent string    `json:"trigger_event"`
	ActionSteps  []byte    `json:"action_steps"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// WorkflowExecutionResult is returned when a workflow is triggered.
type WorkflowExecutionResult struct {
	WorkflowExecutionID uuid.UUID `json:"workflow_execution_id"`
	StepsExecuted       int       `json:"steps_executed"`
	Status              string    `json:"status"`
}

// AuditLog represents an auditable action in the system.
type AuditLog struct {
	ID            int64      `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	Action        string     `json:"action"`
	IPAddress     *string    `json:"ip_address,omitempty"`
	AIAnomalyFlag bool       `json:"ai_anomaly_flag"`
	Metadata      []byte     `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// SecurityAnomaly represents an anomaly detected in the audit log.
type SecurityAnomaly struct {
	LogID       int64   `json:"log_id"`
	Action      string  `json:"action"`
	UserID      string  `json:"user_id"`
	IPAddress   string  `json:"ip_address"`
	AnomalyType string  `json:"anomaly_type"`
	AIRiskScore float64 `json:"ai_risk_score"`
}

// AIUsageLog tracks AI feature usage and credit consumption.
type AIUsageLog struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	FeatureUsed     string     `json:"feature_used"`
	CreditsConsumed int        `json:"credits_consumed"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ---- Repository ----

type platformRepository struct {
	pool *pgxpool.Pool
}

func newPlatformRepository(pool *pgxpool.Pool) *platformRepository {
	return &platformRepository{pool: pool}
}

func (r *platformRepository) CreateConversation(ctx context.Context, conv *AICopilotConversation) error {
	conv.ID = uuid.New()
	conv.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_copilot_conversations (id, org_id, user_id, prompt_text, generated_sql, response_payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, conv.ID, conv.OrgID, conv.UserID, conv.PromptText, conv.GeneratedSQL, conv.ResponsePayload, conv.CreatedAt)
	return err
}

func (r *platformRepository) FindWorkflowsByEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]LowCodeWorkflow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, trigger_event, action_steps, is_active, created_at
		FROM lowcode_workflows
		WHERE org_id = $1 AND trigger_event = $2 AND is_active = TRUE
	`, orgID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []LowCodeWorkflow
	for rows.Next() {
		var w LowCodeWorkflow
		if err := rows.Scan(&w.ID, &w.OrgID, &w.Name, &w.TriggerEvent, &w.ActionSteps, &w.IsActive, &w.CreatedAt); err != nil {
			return nil, err
		}
		workflows = append(workflows, w)
	}
	if workflows == nil {
		workflows = []LowCodeWorkflow{}
	}
	return workflows, rows.Err()
}

func (r *platformRepository) GetAnomaliesBySeverity(ctx context.Context, orgID uuid.UUID, severity string, limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 20
	}

	// Derive a minimum risk score based on severity string
	var minRiskScore float64
	switch severity {
	case "critical":
		minRiskScore = 0.9
	case "high":
		minRiskScore = 0.7
	case "medium":
		minRiskScore = 0.4
	default:
		minRiskScore = 0.0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, user_id, action, ip_address, ai_anomaly_flag, metadata, created_at
		FROM audit_logs
		WHERE org_id = $1 AND ai_anomaly_flag = TRUE
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.OrgID, &l.UserID, &l.Action, &l.IPAddress, &l.AIAnomalyFlag, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AuditLog{}
	}

	// Filter by severity post-query (simulated risk scores are generated below)
	// In production, the risk score would be stored in the metadata column
	_ = minRiskScore

	return logs, rows.Err()
}

func (r *platformRepository) LogAIUsage(ctx context.Context, log *AIUsageLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_usage_logs (id, org_id, user_id, feature_used, credits_consumed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.ID, log.OrgID, log.UserID, log.FeatureUsed, log.CreditsConsumed, log.CreatedAt)
	return err
}

// ---- Service ----

// PlatformService provides AI Copilot, Workflow Automation, and Security intelligence.
type PlatformService struct {
	repo *platformRepository
}

// NewPlatformService creates a new PlatformService backed by the given pool.
func NewPlatformService(pool *pgxpool.Pool) *PlatformService {
	return &PlatformService{repo: newPlatformRepository(pool)}
}

// ExecuteCopilotQuery simulates a text-to-SQL AI copilot. It takes a natural language
// prompt, generates SQL, simulates query results, and recommends a chart type.
func (s *PlatformService) ExecuteCopilotQuery(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, prompt string) (*AICopilotQueryResult, error) {
	// Simulate AI text-to-SQL generation
	generatedSQL, tableName := simulateTextToSQL(prompt)
	dataTable := simulateQueryResults(prompt, tableName)
	chartRec := recommendChart(prompt, dataTable)

	// Persist the conversation
	generatedSQLPtr := &generatedSQL
	conv := &AICopilotConversation{
		OrgID:        orgID,
		UserID:       userID,
		PromptText:   prompt,
		GeneratedSQL: generatedSQLPtr,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("store copilot conversation: %w", err)
	}

	// Log AI usage
	usage := &AIUsageLog{
		OrgID:           orgID,
		UserID:          userID,
		FeatureUsed:     "copilot.query",
		CreditsConsumed: 1,
	}
	if err := s.repo.LogAIUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("log ai usage: %w", err)
	}

	return &AICopilotQueryResult{
		GeneratedSQL:        generatedSQL,
		DataTable:           dataTable,
		ChartRecommendation: chartRec,
	}, nil
}

// TriggerWorkflow executes a low-code automation workflow in response to an event.
// Returns the execution ID, number of steps run, and overall status.
func (s *PlatformService) TriggerWorkflow(ctx context.Context, orgID uuid.UUID, eventType string, payload map[string]interface{}) (*WorkflowExecutionResult, error) {
	// Find matching active workflows for this event type
	workflows, err := s.repo.FindWorkflowsByEvent(ctx, orgID, eventType)
	if err != nil {
		return nil, fmt.Errorf("find workflows: %w", err)
	}

	executionID := uuid.New()

	// Simulate workflow execution
	stepsExecuted := 0
	if len(workflows) > 0 {
		// Count steps from the first matching workflow's action_steps (JSONB)
		for _, w := range workflows {
			// Simulate steps from the action_steps count — in production this would
			// parse the JSONB array and execute each step (send email, call webhook, etc.)
			stepsExecuted += simulateWorkflowSteps(w.Name)
		}
	}

	// If no workflows matched, execute a default "event processed" step count
	if stepsExecuted == 0 {
		stepsExecuted = 1 + rand.Intn(3)
	}

	// Log AI usage for the workflow execution
	usage := &AIUsageLog{
		OrgID:           orgID,
		FeatureUsed:     "workflows.execute",
		CreditsConsumed: 1,
	}
	if err := s.repo.LogAIUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("log ai usage: %w", err)
	}

	return &WorkflowExecutionResult{
		WorkflowExecutionID: executionID,
		StepsExecuted:       stepsExecuted,
		Status:              "success",
	}, nil
}

// FetchAuditAnomalies retrieves security audit log entries flagged by the AI anomaly
// detection system, filtered by severity level.
func (s *PlatformService) FetchAuditAnomalies(ctx context.Context, orgID uuid.UUID, severity string, limit int) ([]SecurityAnomaly, error) {
	logs, err := s.repo.GetAnomaliesBySeverity(ctx, orgID, severity, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch anomalies: %w", err)
	}

	anomalies := make([]SecurityAnomaly, 0, len(logs))
	for _, l := range logs {
		anomalyType, riskScore := simulateAnomalyClassification(l.Action, l.IPAddress)

		// Apply severity filter
		if !matchesSeverity(riskScore, severity) {
			continue
		}

		userIDStr := ""
		if l.UserID != nil {
			userIDStr = l.UserID.String()
		}
		ipStr := ""
		if l.IPAddress != nil {
			ipStr = *l.IPAddress
		}

		anomalies = append(anomalies, SecurityAnomaly{
			LogID:       l.ID,
			Action:      l.Action,
			UserID:      userIDStr,
			IPAddress:   ipStr,
			AnomalyType: anomalyType,
			AIRiskScore: riskScore,
		})
	}

	return anomalies, nil
}

// ---- AI simulation helpers ----

// simulateTextToSQL generates a plausible SQL query from a natural language prompt.
// In production this would call an LLM or a fine-tuned model.
func simulateTextToSQL(prompt string) (sql, tableName string) {
	lower := strings.ToLower(prompt)

	switch {
	case strings.Contains(lower, "sales rep") || strings.Contains(lower, "revenue"):
		return `SELECT u.full_name AS sales_rep, SUM(d.amount) AS total_revenue
FROM crm_deals d
JOIN users u ON d.assigned_to = u.id
WHERE d.stage = 'closed_won'
  AND d.created_at BETWEEN '2025-04-01' AND '2025-06-30'
GROUP BY u.full_name
ORDER BY total_revenue DESC
LIMIT 5`, "crm_deals"
	case strings.Contains(lower, "invoice") || strings.Contains(lower, "overdue"):
		return `SELECT i.invoice_number, c.first_name || ' ' || COALESCE(c.last_name, '') AS customer,
       i.total_amount, i.due_date,
       CURRENT_DATE - i.due_date AS days_overdue
FROM invoices i
JOIN crm_contacts c ON i.contact_id = c.id
WHERE i.status = 'sent' AND i.due_date < CURRENT_DATE
ORDER BY days_overdue DESC
LIMIT 10`, "invoices"
	case strings.Contains(lower, "attendance") || strings.Contains(lower, "clock"):
		return `SELECT e.employee_code, e.department,
       COUNT(a.id) AS clock_ins,
       MIN(a.clock_in) AS first_clock_in,
       MAX(a.clock_in) AS last_clock_in
FROM employees e
JOIN attendance_logs a ON a.employee_id = e.id
WHERE a.clock_in >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY e.employee_code, e.department
ORDER BY clock_ins DESC`, "attendance_logs"
	case strings.Contains(lower, "fleet") || strings.Contains(lower, "vehicle"):
		return `SELECT v.license_plate, v.make, v.model,
       MAX(t.speed_kmh) AS max_speed,
       AVG(t.speed_kmh) AS avg_speed,
       AVG(t.fuel_level_pct) AS avg_fuel_pct
FROM fleet_vehicles v
JOIN fleet_telematics_logs t ON t.vehicle_id = v.id
WHERE t.recorded_at >= NOW() - INTERVAL '7 days'
GROUP BY v.license_plate, v.make, v.model
ORDER BY max_speed DESC`, "fleet_telematics_logs"
	default:
		return `SELECT id, org_id, created_at
FROM organizations
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 5`, "organizations"
	}
}

// simulateQueryResults generates mock data rows based on the prompt context.
func simulateQueryResults(prompt, tableName string) []map[string]interface{} {
	lower := strings.ToLower(prompt)

	switch {
	case strings.Contains(lower, "sales rep") || strings.Contains(lower, "revenue"):
		return []map[string]interface{}{
			{"sales_rep": "Alice Johnson", "total_revenue": 245000.50},
			{"sales_rep": "Bob Martinez", "total_revenue": 198750.00},
			{"sales_rep": "Carol Chen", "total_revenue": 176200.75},
			{"sales_rep": "David Kim", "total_revenue": 152300.25},
			{"sales_rep": "Eve Thompson", "total_revenue": 134500.00},
		}
	case strings.Contains(lower, "invoice") || strings.Contains(lower, "overdue"):
		return []map[string]interface{}{
			{"invoice_number": "INV-2025-0042", "customer": "Acme Corp", "total_amount": 12500.00, "due_date": "2025-06-15", "days_overdue": 51},
			{"invoice_number": "INV-2025-0051", "customer": "Globex Inc", "total_amount": 8750.00, "due_date": "2025-07-01", "days_overdue": 35},
			{"invoice_number": "INV-2025-0058", "customer": "Initech", "total_amount": 3200.00, "due_date": "2025-07-15", "days_overdue": 21},
		}
	case strings.Contains(lower, "attendance") || strings.Contains(lower, "clock"):
		return []map[string]interface{}{
			{"employee_code": "EMP-001", "department": "Engineering", "clock_ins": 22, "first_clock_in": "2025-07-06T08:55:00Z", "last_clock_in": "2025-08-04T09:02:00Z"},
			{"employee_code": "EMP-002", "department": "Sales", "clock_ins": 21, "first_clock_in": "2025-07-06T08:30:00Z", "last_clock_in": "2025-08-04T08:45:00Z"},
			{"employee_code": "EMP-003", "department": "Engineering", "clock_ins": 20, "first_clock_in": "2025-07-07T09:10:00Z", "last_clock_in": "2025-08-04T09:15:00Z"},
		}
	case strings.Contains(lower, "fleet") || strings.Contains(lower, "vehicle"):
		return []map[string]interface{}{
			{"license_plate": "ABC-1234", "make": "Ford", "model": "Transit", "max_speed": 112.5, "avg_speed": 68.3, "avg_fuel_pct": 72.1},
			{"license_plate": "XYZ-5678", "make": "Mercedes", "model": "Sprinter", "max_speed": 105.0, "avg_speed": 62.7, "avg_fuel_pct": 65.4},
			{"license_plate": "DEF-9012", "make": "Ram", "model": "ProMaster", "max_speed": 98.2, "avg_speed": 55.9, "avg_fuel_pct": 58.3},
		}
	default:
		return []map[string]interface{}{
			{"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "org_id": "f1e2d3c4-b5a6-9870-fedc-ba0987654321", "created_at": "2025-01-15T10:00:00Z"},
		}
	}
}

// recommendChart suggests a chart type based on the prompt and data shape.
func recommendChart(prompt string, data []map[string]interface{}) string {
	if len(data) == 0 {
		return "none"
	}

	lower := strings.ToLower(prompt)

	// Check column types to recommend appropriate chart
	firstRow := data[0]
	hasNumeric := false
	hasTimeField := false
	for k, v := range firstRow {
		if _, ok := v.(float64); ok {
			hasNumeric = true
		}
		if strings.Contains(strings.ToLower(k), "date") || strings.Contains(strings.ToLower(k), "time") {
			hasTimeField = true
		}
	}

	if strings.Contains(lower, "top") || strings.Contains(lower, "by") || len(data) <= 5 && hasNumeric {
		return "bar_chart"
	}
	if hasTimeField && hasNumeric {
		return "line_chart"
	}
	return "table"
}

// simulateWorkflowSteps generates a realistic step count for a workflow execution.
func simulateWorkflowSteps(workflowName string) int {
	baseSteps := 2 + rand.Intn(5)
	return baseSteps
}

// simulateAnomalyClassification generates a plausible anomaly type and risk score
// based on the audited action and IP address.
func simulateAnomalyClassification(action string, ipAddress *string) (string, float64) {
	lower := strings.ToLower(action)

	switch {
	case strings.Contains(lower, "login") || strings.Contains(lower, "signin"):
		return "suspicious_login", 0.75 + rand.Float64()*0.2
	case strings.Contains(lower, "delete") || strings.Contains(lower, "remove"):
		return "unauthorized_delete_attempt", 0.85 + rand.Float64()*0.15
	case strings.Contains(lower, "export") || strings.Contains(lower, "download"):
		return "data_exfiltration", 0.7 + rand.Float64()*0.25
	case strings.Contains(lower, "permission") || strings.Contains(lower, "role"):
		return "privilege_escalation", 0.8 + rand.Float64()*0.2
	case strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key"):
		return "api_key_abuse", 0.65 + rand.Float64()*0.3
	default:
		return "anomalous_activity", 0.6 + rand.Float64()*0.3
	}
}

// matchesSeverity checks if a risk score meets the severity threshold.
func matchesSeverity(riskScore float64, severity string) bool {
	switch severity {
	case "critical":
		return riskScore >= 0.9
	case "high":
		return riskScore >= 0.7
	case "medium":
		return riskScore >= 0.4
	case "low":
		return true
	default:
		return true
	}
}
