package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/ai"
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

// BIDashboard represents a BI executive dashboard.
type BIDashboard struct {
	ID        uuid.UUID       `json:"id"`
	OrgID     uuid.UUID       `json:"org_id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// DashboardWidget defines a single widget configuration within a dashboard.
type DashboardWidget struct {
	WidgetType string `json:"widget_type"`
	Title      string `json:"title"`
	DataQuery  string `json:"data_query"`
	Position   int    `json:"position"`
}

// DashboardData contains the rendered data for a dashboard.
type DashboardData struct {
	DashboardID uuid.UUID    `json:"dashboard_id"`
	Widgets     []WidgetData `json:"widgets"`
}

// WidgetData contains the rendered data for a single widget.
type WidgetData struct {
	WidgetType         string  `json:"widget_type"`
	Title              string  `json:"title"`
	Data               [][]any `json:"data"`
	AnomalyDetected    bool    `json:"anomaly_detected"`
	AnomalyDescription string  `json:"anomaly_description,omitempty"`
}

// IoTDevice represents a registered IoT device.
type IoTDevice struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	DeviceName string     `json:"device_name"`
	DeviceType string     `json:"device_type"`
	MACAddress *string    `json:"mac_address,omitempty"`
	Status     string     `json:"status"`
	LastPing   *time.Time `json:"last_ping,omitempty"`
}

// IoTDeviceReading represents a single metric reading from an IoT device.
type IoTDeviceReading struct {
	DeviceID    uuid.UUID `json:"device_id"`
	MetricName  string    `json:"metric_name"`
	MetricValue float64   `json:"metric_value"`
	Unit        string    `json:"unit"`
	RecordedAt  time.Time `json:"recorded_at"`
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

// ---- BI Dashboard repository ----

func (r *platformRepository) CreateDashboard(ctx context.Context, d *BIDashboard) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bi_dashboards (id, org_id, name, config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, d.ID, d.OrgID, d.Name, d.Config, d.IsActive, d.CreatedAt, d.UpdatedAt)
	return err
}

func (r *platformRepository) GetDashboardByID(ctx context.Context, id uuid.UUID) (*BIDashboard, error) {
	d := &BIDashboard{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, config, is_active, created_at, updated_at
		FROM bi_dashboards WHERE id = $1
	`, id).Scan(&d.ID, &d.OrgID, &d.Name, &d.Config, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *platformRepository) ListDashboards(ctx context.Context, orgID uuid.UUID) ([]BIDashboard, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, config, is_active, created_at, updated_at
		FROM bi_dashboards WHERE org_id = $1 ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dashboards []BIDashboard
	for rows.Next() {
		var d BIDashboard
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Name, &d.Config, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		dashboards = append(dashboards, d)
	}
	if dashboards == nil {
		dashboards = []BIDashboard{}
	}
	return dashboards, rows.Err()
}

// ---- IoT Device repository ----

func (r *platformRepository) CreateDevice(ctx context.Context, d *IoTDevice) error {
	d.ID = uuid.New()
	now := time.Now()
	d.LastPing = &now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO iot_devices (id, org_id, device_name, device_type, mac_address, status, last_ping)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, d.ID, d.OrgID, d.DeviceName, d.DeviceType, d.MACAddress, d.Status, d.LastPing)
	return err
}

func (r *platformRepository) GetDeviceByMAC(ctx context.Context, macAddress string) (*IoTDevice, error) {
	d := &IoTDevice{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, device_name, device_type, mac_address, status, last_ping
		FROM iot_devices WHERE mac_address = $1
	`, macAddress).Scan(&d.ID, &d.OrgID, &d.DeviceName, &d.DeviceType, &d.MACAddress, &d.Status, &d.LastPing)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *platformRepository) GetDeviceByID(ctx context.Context, id uuid.UUID) (*IoTDevice, error) {
	d := &IoTDevice{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, device_name, device_type, mac_address, status, last_ping
		FROM iot_devices WHERE id = $1
	`, id).Scan(&d.ID, &d.OrgID, &d.DeviceName, &d.DeviceType, &d.MACAddress, &d.Status, &d.LastPing)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *platformRepository) InsertDeviceReading(ctx context.Context, orgID uuid.UUID, reading *IoTDeviceReading) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO iot_device_readings (org_id, device_id, metric_name, metric_value, unit, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, orgID, reading.DeviceID, reading.MetricName, reading.MetricValue, reading.Unit, reading.RecordedAt)
	return err
}

func (r *platformRepository) UpdateDeviceLastPing(ctx context.Context, deviceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE iot_devices SET last_ping = NOW() WHERE id = $1`, deviceID)
	return err
}

func (r *platformRepository) ListDevices(ctx context.Context, orgID uuid.UUID) ([]IoTDevice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, device_name, device_type, mac_address, status, last_ping
		FROM iot_devices WHERE org_id = $1 ORDER BY device_name
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []IoTDevice
	for rows.Next() {
		var d IoTDevice
		if err := rows.Scan(&d.ID, &d.OrgID, &d.DeviceName, &d.DeviceType, &d.MACAddress, &d.Status, &d.LastPing); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	if devices == nil {
		devices = []IoTDevice{}
	}
	return devices, rows.Err()
}

// ---- Service ----

// PlatformService provides AI Copilot, Workflow Automation, and Security intelligence.
type PlatformService struct {
	repo *platformRepository
	ai   ai.Inferrer
}

func NewPlatformService(pool *pgxpool.Pool, aiSvc ai.Inferrer) *PlatformService {
	return &PlatformService{repo: newPlatformRepository(pool), ai: aiSvc}
}

// ExecuteCopilotQuery simulates a text-to-SQL AI copilot. It takes a natural language
// prompt, generates SQL, simulates query results, and recommends a chart type.
func (s *PlatformService) ExecuteCopilotQuery(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, prompt string) (*AICopilotQueryResult, error) {
	// Simulate AI text-to-SQL generation
	sqlResp, err := s.ai.Infer(ctx, &ai.TextToSQLRequest{Prompt: prompt})
	if err != nil {
		return nil, err
	}
	ts := sqlResp.(*ai.TextToSQLResponse)
	generatedSQL := ts.SQL
	tableName := ts.TableName

	dataResp, err := s.ai.Infer(ctx, &ai.QueryResultsRequest{Prompt: prompt, TableName: tableName})
	if err != nil {
		return nil, err
	}
	dataTable := dataResp.(*ai.QueryResultsResponse).Data
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
		resp, err := s.ai.Infer(ctx, &ai.AnomalyRequest{Action: l.Action, IPAddress: l.IPAddress})
		if err != nil {
			return nil, err
		}
		anom := resp.(*ai.AnomalyResponse)
		anomalyType := anom.Type
		riskScore := anom.RiskScore

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

// ---- BI Dashboard service methods ----

// CreateDashboard creates a new BI executive dashboard.
func (s *PlatformService) CreateDashboard(ctx context.Context, orgID uuid.UUID, name string, config json.RawMessage) (*BIDashboard, error) {
	d := &BIDashboard{
		OrgID:    orgID,
		Name:     name,
		Config:   config,
		IsActive: true,
	}
	if err := s.repo.CreateDashboard(ctx, d); err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}
	return d, nil
}

// GetDashboardData returns simulated dashboard data with AI anomaly detection.
func (s *PlatformService) GetDashboardData(ctx context.Context, dashboardID uuid.UUID) (*DashboardData, error) {
	d, err := s.repo.GetDashboardByID(ctx, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}

	// Parse widgets from config
	var widgets []DashboardWidget
	if len(d.Config) > 0 {
		if err := json.Unmarshal(d.Config, &widgets); err != nil {
			// If config is not a widget array, treat it as empty
			widgets = nil
		}
	}

	// Generate simulated widget data with AI anomaly detection
	widgetData := make([]WidgetData, 0, len(widgets))
	for _, w := range widgets {
		data, anomalyDetected, anomalyDesc := simulateWidgetData(w)
		widgetData = append(widgetData, WidgetData{
			WidgetType:         w.WidgetType,
			Title:              w.Title,
			Data:               data,
			AnomalyDetected:    anomalyDetected,
			AnomalyDescription: anomalyDesc,
		})
	}

	// If no widgets configured, provide default widgets
	if len(widgetData) == 0 {
		widgetData = getDefaultWidgets()
	}

	return &DashboardData{
		DashboardID: dashboardID,
		Widgets:     widgetData,
	}, nil
}

// ListDashboards returns all dashboards for an organization.
func (s *PlatformService) ListDashboards(ctx context.Context, orgID uuid.UUID) ([]BIDashboard, error) {
	dashboards, err := s.repo.ListDashboards(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list dashboards: %w", err)
	}
	return dashboards, nil
}

// ---- IoT Device service methods ----

// RegisterDevice registers a new IoT device.
func (s *PlatformService) RegisterDevice(ctx context.Context, orgID uuid.UUID, deviceName, deviceType, macAddress string) (*IoTDevice, error) {
	var macPtr *string
	if macAddress != "" {
		macPtr = &macAddress
	}

	d := &IoTDevice{
		OrgID:      orgID,
		DeviceName: deviceName,
		DeviceType: deviceType,
		MACAddress: macPtr,
		Status:     "online",
	}
	if err := s.repo.CreateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}
	return d, nil
}

// IngestDeviceReading processes a device reading, persists it, and checks for anomalies.
func (s *PlatformService) IngestDeviceReading(ctx context.Context, orgID uuid.UUID, reading *IoTDeviceReading) (*IngestReadingResult, error) {
	// Validate the device exists
	device, err := s.repo.GetDeviceByID(ctx, reading.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("device lookup: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	// Persist the reading
	reading.RecordedAt = time.Now()
	if err := s.repo.InsertDeviceReading(ctx, orgID, reading); err != nil {
		return nil, fmt.Errorf("insert reading: %w", err)
	}

	// Update device last_ping
	if err := s.repo.UpdateDeviceLastPing(ctx, reading.DeviceID); err != nil {
		return nil, fmt.Errorf("update last ping: %w", err)
	}

	// AI anomaly detection behind the seam
	readResp, err := s.ai.Infer(ctx, &ai.ReadingAnomalyRequest{
		MetricName:  reading.MetricName,
		MetricValue: reading.MetricValue,
		Unit:        reading.Unit,
	})
	if err != nil {
		return nil, err
	}
	ra := readResp.(*ai.ReadingAnomalyResponse)
	anomalyDetected := ra.AnomalyDetected
	anomalyDesc := ra.AnomalyDescription

	// Log AI usage for anomaly detection
	usage := &AIUsageLog{
		OrgID:           orgID,
		FeatureUsed:     "iot.anomaly.detection",
		CreditsConsumed: 1,
	}
	if anomalyDetected {
		usage.CreditsConsumed = 2
	}
	_ = s.repo.LogAIUsage(ctx, usage)

	return &IngestReadingResult{
		Status:             "processed",
		AnomalyDetected:    anomalyDetected,
		AnomalyDescription: anomalyDesc,
	}, nil
}

// IngestDeviceReadingBatch processes multiple device readings in batch for high-frequency ingestion.
func (s *PlatformService) IngestDeviceReadingBatch(ctx context.Context, orgID uuid.UUID, readings []IoTDeviceReading) (int, int, error) {
	accepted := 0
	rejected := 0

	for i := range readings {
		readings[i].RecordedAt = time.Now()
		if err := s.repo.InsertDeviceReading(ctx, orgID, &readings[i]); err != nil {
			rejected++
			continue
		}
		accepted++
	}

	return accepted, rejected, nil
}

// IngestReadingResult is the response from a device reading ingestion.
type IngestReadingResult struct {
	Status             string `json:"status"`
	AnomalyDetected    bool   `json:"anomaly_detected"`
	AnomalyDescription string `json:"anomaly_description,omitempty"`
}

// ListDevices returns all IoT devices for an organization.
func (s *PlatformService) ListDevices(ctx context.Context, orgID uuid.UUID) ([]IoTDevice, error) {
	devices, err := s.repo.ListDevices(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

// ---- BI & IoT simulation helpers ----

// simulateWidgetData generates mock data for a dashboard widget with anomaly detection.
func simulateWidgetData(w DashboardWidget) (data [][]any, anomalyDetected bool, anomalyDesc string) {
	switch w.WidgetType {
	case "bar_chart", "column_chart":
		data = [][]any{
			{"Q1", 245000.50},
			{"Q2", 198750.00},
			{"Q3", 312400.75},
			{"Q4", 278900.25},
		}
		// Simulate anomaly: Q3 spike
		if rand.Float64() < 0.3 {
			anomalyDetected = true
			anomalyDesc = "Unusual revenue spike detected in Q3 (+57% vs Q2 average)"
		}
	case "line_chart":
		data = [][]any{
			{"Jan", 1200.0}, {"Feb", 1350.0}, {"Mar", 1280.0},
			{"Apr", 1420.0}, {"May", 1380.0}, {"Jun", 1550.0},
		}
	case "pie_chart":
		data = [][]any{
			{"Product A", 35.0}, {"Product B", 28.0},
			{"Product C", 22.0}, {"Product D", 15.0},
		}
	case "kpi":
		data = [][]any{
			{"Revenue", "$1,035,051.50"},
			{"Growth", "+12.4%"},
			{"Customers", "1,247"},
		}
	default:
		data = [][]any{
			{"Metric", "Value"},
			{"Sample", 100.0},
		}
	}
	return
}

// getDefaultWidgets returns default dashboard widgets when none are configured.
func getDefaultWidgets() []WidgetData {
	return []WidgetData{
		{
			WidgetType:      "kpi",
			Title:           "Key Metrics",
			Data:            [][]any{{"Revenue", "$1,035,051.50"}, {"Growth", "+12.4%"}, {"Customers", "1,247"}},
			AnomalyDetected: false,
		},
		{
			WidgetType:         "bar_chart",
			Title:              "Quarterly Revenue",
			Data:               [][]any{{"Q1", 245000.50}, {"Q2", 198750.00}, {"Q3", 312400.75}, {"Q4", 278900.25}},
			AnomalyDetected:    true,
			AnomalyDescription: "Unusual revenue spike detected in Q3 (+57% vs Q2 average)",
		},
		{
			WidgetType:      "line_chart",
			Title:           "Monthly Trends",
			Data:            [][]any{{"Jan", 1200.0}, {"Feb", 1350.0}, {"Mar", 1280.0}, {"Apr", 1420.0}, {"May", 1380.0}, {"Jun", 1550.0}},
			AnomalyDetected: false,
		},
	}
}
