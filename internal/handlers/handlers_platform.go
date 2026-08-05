package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/ai/copilot/query ----

type copilotQueryRequest struct {
	Prompt string `json:"prompt" example:"Show top 5 sales reps by revenue in Q2"`
}

// DataTableRow is a single row in the AI copilot result table.
type DataTableRow map[string]any

// CopilotQueryResponse represents the text-to-SQL AI copilot result.
type CopilotQueryResponse struct {
	GeneratedSQL        string         `json:"generated_sql" example:"SELECT u.full_name AS sales_rep, SUM(d.amount) AS total_revenue FROM crm_deals d JOIN users u ON d.assigned_to = u.id WHERE d.stage = 'closed_won' AND d.created_at BETWEEN '2025-04-01' AND '2025-06-30' GROUP BY u.full_name ORDER BY total_revenue DESC LIMIT 5"`
	DataTable           []DataTableRow `json:"data_table"`
	ChartRecommendation string         `json:"chart_recommendation" example:"bar_chart"`
}

// copilotQueryHandler executes a text-to-SQL AI copilot query.
//
//	@Summary		Execute AI copilot text-to-SQL query
//	@Description	Converts a natural language prompt to SQL, executes the simulated query, and returns results with a chart recommendation. Requires `copilot.use` permission.
//	@Tags			AI & Platform
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	copilotQueryRequest	true	"Natural language prompt (e.g., 'Show top 5 sales reps by revenue in Q2')"
//	@Success		200	{object}	CopilotQueryResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/ai/copilot/query [post]
func (a *App) copilotQueryHandler(w http.ResponseWriter, r *http.Request) {
	var req copilotQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Prompt) {
		utils.WriteErr(w, http.StatusBadRequest, "prompt is required")
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

	userIDPtr := &userID

	result, err := a.Platform.ExecuteCopilotQuery(r.Context(), orgID, userIDPtr, req.Prompt)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not execute copilot query")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"generated_sql":        result.GeneratedSQL,
		"data_table":           result.DataTable,
		"chart_recommendation": result.ChartRecommendation,
	})
}

// ---- POST /api/v1/workflows/trigger ----

type triggerWorkflowRequest struct {
	EventType string                 `json:"event_type" example:"invoice.paid"`
	Payload   map[string]interface{} `json:"payload"`
}

// TriggerWorkflowResponse represents the workflow execution result.
type TriggerWorkflowResponse struct {
	WorkflowExecutionID string `json:"workflow_execution_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StepsExecuted       int    `json:"steps_executed" example:"4"`
	Status              string `json:"status" example:"success"`
}

// triggerWorkflowHandler triggers a low-code automated workflow by event type.
//
//	@Summary		Trigger automated workflow
//	@Description	Triggers a low-code automation workflow in response to an event (e.g., invoice.paid). Matches active workflows by event_type and executes their action steps. Accepts both Bearer token and webhook signature authentication. Requires `workflows.execute` permission.
//	@Tags			AI & Platform
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	triggerWorkflowRequest	true	"Event trigger payload"
//	@Success		200	{object}	TriggerWorkflowResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/workflows/trigger [post]
func (a *App) triggerWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var req triggerWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.EventType) {
		utils.WriteErr(w, http.StatusBadRequest, "event_type is required")
		return
	}

	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
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

	result, err := a.Platform.TriggerWorkflow(r.Context(), orgID, req.EventType, req.Payload)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not trigger workflow")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"workflow_execution_id": result.WorkflowExecutionID.String(),
		"steps_executed":        result.StepsExecuted,
		"status":                result.Status,
	})
}

// ---- GET /api/v1/security/audit-anomalies ----

// SecurityAnomalyItem represents a single anomaly in the response.
type SecurityAnomalyItem struct {
	LogID       int64   `json:"log_id" example:"10042"`
	Action      string  `json:"action" example:"user.login"`
	UserID      string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	IPAddress   string  `json:"ip_address" example:"192.168.1.100"`
	AnomalyType string  `json:"anomaly_type" example:"suspicious_login"`
	AIRiskScore float64 `json:"ai_risk_score" example:"0.87"`
}

// AuditAnomaliesResponse represents the security audit anomalies response.
type AuditAnomaliesResponse struct {
	Anomalies []SecurityAnomalyItem `json:"anomalies"`
}

// auditAnomaliesHandler fetches security threat and anomaly audit logs.
//
//	@Summary		Fetch security audit anomalies
//	@Description	Retrieves security audit log entries flagged as anomalies by the AI detection system. Filterable by severity level (low, medium, high, critical). Requires `security.audit.read` permission.
//	@Tags			AI & Platform
//	@Produce		json
//	@Security		BearerAuth
//	@Param			severity	query	string	false	"Filter by severity"	Enums(low, medium, high, critical)	default(high)
//	@Param			limit		query	int		false	"Maximum number of results (1-100)"	default(20)
//	@Success		200	{object}	AuditAnomaliesResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/security/audit-anomalies [get]
func (a *App) auditAnomaliesHandler(w http.ResponseWriter, r *http.Request) {
	severity := r.URL.Query().Get("severity")
	if !utils.NotBlank(severity) {
		severity = "high"
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			utils.WriteErr(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
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

	anomalies, err := a.Platform.FetchAuditAnomalies(r.Context(), orgID, severity, limit)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not fetch audit anomalies")
		return
	}

	items := make([]SecurityAnomalyItem, 0, len(anomalies))
	for _, a := range anomalies {
		items = append(items, SecurityAnomalyItem{
			LogID:       a.LogID,
			Action:      a.Action,
			UserID:      a.UserID,
			IPAddress:   a.IPAddress,
			AnomalyType: a.AnomalyType,
			AIRiskScore: a.AIRiskScore,
		})
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"anomalies": items,
	})
}

// ---- POST /api/v1/bi/dashboards ----

type createDashboardRequest struct {
	Name   string          `json:"name" example:"Executive Overview"`
	Config json.RawMessage `json:"config"`
}

// DashboardResponse represents a BI dashboard in API responses.
type DashboardResponse struct {
	ID        uuid.UUID       `json:"id"`
	OrgID     uuid.UUID       `json:"org_id"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// createDashboardHandler creates a new BI executive dashboard.
//
//	@Summary		Create BI dashboard
//	@Description	Creates a new BI executive dashboard with optional widget configuration. Requires `bi.dashboards.write` permission.
//	@Tags			BI & IoT
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createDashboardRequest	true	"Dashboard creation payload"
//	@Success		201	{object}	DashboardResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/bi/dashboards [post]
func (a *App) createDashboardHandler(w http.ResponseWriter, r *http.Request) {
	var req createDashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Name) {
		utils.WriteErr(w, http.StatusBadRequest, "name is required")
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

	config := req.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	dashboard, err := a.Platform.CreateDashboard(r.Context(), orgID, req.Name, config)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create dashboard")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":         dashboard.ID,
		"org_id":     dashboard.OrgID,
		"name":       dashboard.Name,
		"config":     dashboard.Config,
		"is_active":  dashboard.IsActive,
		"created_at": dashboard.CreatedAt,
		"updated_at": dashboard.UpdatedAt,
	})
}

// ---- GET /api/v1/bi/dashboards ----

// ListDashboardsResponse represents the list of BI dashboards.
type ListDashboardsResponse struct {
	Dashboards []DashboardResponse `json:"dashboards"`
}

// listDashboardsHandler returns all BI dashboards for the organization.
//
//	@Summary		List BI dashboards
//	@Description	Returns all BI executive dashboards for the authenticated organization. Requires `bi.dashboards.read` permission.
//	@Tags			BI & IoT
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	ListDashboardsResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/bi/dashboards [get]
func (a *App) listDashboardsHandler(w http.ResponseWriter, r *http.Request) {
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

	dashboards, err := a.Platform.ListDashboards(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not list dashboards")
		return
	}

	items := make([]DashboardResponse, 0, len(dashboards))
	for _, d := range dashboards {
		items = append(items, DashboardResponse{
			ID:        d.ID,
			OrgID:     d.OrgID,
			Name:      d.Name,
			Config:    d.Config,
			IsActive:  d.IsActive,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"dashboards": items,
	})
}

// ---- GET /api/v1/bi/dashboards/{dashboard_id}/data ----

// WidgetDataItem represents a single widget's data in the API response.
type WidgetDataItem struct {
	WidgetType         string  `json:"widget_type"`
	Title              string  `json:"title"`
	Data               [][]any `json:"data"`
	AnomalyDetected    bool    `json:"anomaly_detected"`
	AnomalyDescription string  `json:"anomaly_description,omitempty"`
}

// DashboardDataResponse represents the rendered dashboard data.
type DashboardDataResponse struct {
	DashboardID uuid.UUID        `json:"dashboard_id"`
	Widgets     []WidgetDataItem `json:"widgets"`
}

// getDashboardDataHandler returns simulated dashboard data with AI anomaly detection.
//
//	@Summary		Get dashboard data
//	@Description	Returns rendered widget data for a BI dashboard with AI-powered anomaly detection. Requires `bi.dashboards.read` permission.
//	@Tags			BI & IoT
//	@Produce		json
//	@Security		BearerAuth
//	@Param			dashboard_id	path	string	true	"Dashboard ID"
//	@Success		200	{object}	DashboardDataResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/bi/dashboards/{dashboard_id}/data [get]
func (a *App) getDashboardDataHandler(w http.ResponseWriter, r *http.Request) {
	dashboardIDStr := r.PathValue("dashboard_id")
	dashboardID, err := uuid.Parse(dashboardIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid dashboard_id")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	data, err := a.Platform.GetDashboardData(r.Context(), dashboardID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get dashboard data")
		return
	}

	widgets := make([]WidgetDataItem, 0, len(data.Widgets))
	for _, w := range data.Widgets {
		widgets = append(widgets, WidgetDataItem{
			WidgetType:         w.WidgetType,
			Title:              w.Title,
			Data:               w.Data,
			AnomalyDetected:    w.AnomalyDetected,
			AnomalyDescription: w.AnomalyDescription,
		})
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"dashboard_id": data.DashboardID,
		"widgets":      widgets,
	})
}

// ---- POST /api/v1/iot/devices ----

type registerDeviceRequest struct {
	DeviceName string `json:"device_name" example:"Temperature Sensor A1"`
	DeviceType string `json:"device_type" example:"temperature_sensor"`
	MACAddress string `json:"mac_address" example:"AA:BB:CC:DD:EE:01"`
}

// IoTDeviceResponse represents an IoT device in API responses.
type IoTDeviceResponse struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	DeviceName string     `json:"device_name"`
	DeviceType string     `json:"device_type"`
	MACAddress *string    `json:"mac_address,omitempty"`
	Status     string     `json:"status"`
	LastPing   *time.Time `json:"last_ping,omitempty"`
}

// registerDeviceHandler registers a new IoT device.
//
//	@Summary		Register IoT device
//	@Description	Registers a new IoT device for the organization. Requires `iot.devices.write` permission.
//	@Tags			BI & IoT
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	registerDeviceRequest	true	"IoT device registration payload"
//	@Success		201	{object}	IoTDeviceResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/iot/devices [post]
func (a *App) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.DeviceName) {
		utils.WriteErr(w, http.StatusBadRequest, "device_name is required")
		return
	}

	if !utils.NotBlank(req.DeviceType) {
		utils.WriteErr(w, http.StatusBadRequest, "device_type is required")
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

	device, err := a.Platform.RegisterDevice(r.Context(), orgID, req.DeviceName, req.DeviceType, req.MACAddress)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not register device")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":          device.ID,
		"org_id":      device.OrgID,
		"device_name": device.DeviceName,
		"device_type": device.DeviceType,
		"mac_address": device.MACAddress,
		"status":      device.Status,
		"last_ping":   device.LastPing,
	})
}

// ---- GET /api/v1/iot/devices ----

// ListIoTDevicesResponse represents the list of IoT devices.
type ListIoTDevicesResponse struct {
	Devices []IoTDeviceResponse `json:"devices"`
}

// listDevicesHandler returns all IoT devices for the organization.
//
//	@Summary		List IoT devices
//	@Description	Returns all registered IoT devices for the authenticated organization. Requires `iot.devices.write` permission.
//	@Tags			BI & IoT
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	ListIoTDevicesResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/iot/devices [get]
func (a *App) listDevicesHandler(w http.ResponseWriter, r *http.Request) {
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

	devices, err := a.Platform.ListDevices(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not list devices")
		return
	}

	items := make([]IoTDeviceResponse, 0, len(devices))
	for _, d := range devices {
		items = append(items, IoTDeviceResponse{
			ID:         d.ID,
			OrgID:      d.OrgID,
			DeviceName: d.DeviceName,
			DeviceType: d.DeviceType,
			MACAddress: d.MACAddress,
			Status:     d.Status,
			LastPing:   d.LastPing,
		})
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"devices": items,
	})
}

// ---- POST /api/v1/iot/readings ----

type ingestReadingRequest struct {
	DeviceID    uuid.UUID `json:"device_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MetricName  string    `json:"metric_name" example:"temperature"`
	MetricValue float64   `json:"metric_value" example:"72.5"`
	Unit        string    `json:"unit" example:"celsius"`
}

// IngestReadingResponse represents the result of ingesting a device reading.
type IngestReadingResponse struct {
	Status             string `json:"status" example:"processed"`
	AnomalyDetected    bool   `json:"anomaly_detected"`
	AnomalyDescription string `json:"anomaly_description,omitempty"`
}

// ingestReadingHandler processes an IoT device sensor reading.
//
//	@Summary		Ingest device reading
//	@Description	Processes a sensor reading from an IoT device and performs AI anomaly detection. Requires `iot.readings.ingest` permission.
//	@Tags			BI & IoT
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	ingestReadingRequest	true	"IoT device reading payload"
//	@Success		200	{object}	IngestReadingResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/iot/readings [post]
func (a *App) ingestReadingHandler(w http.ResponseWriter, r *http.Request) {
	var req ingestReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == uuid.Nil {
		utils.WriteErr(w, http.StatusBadRequest, "device_id is required")
		return
	}

	if !utils.NotBlank(req.MetricName) {
		utils.WriteErr(w, http.StatusBadRequest, "metric_name is required")
		return
	}

	reading := &services.IoTDeviceReading{
		DeviceID:    req.DeviceID,
		MetricName:  req.MetricName,
		MetricValue: req.MetricValue,
		Unit:        req.Unit,
		RecordedAt:  time.Now(),
	}

	if err := a.Platform.IngestDeviceReading(r.Context(), reading); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not ingest device reading")
		return
	}

	// Run anomaly detection for the response
	anomalyDetected, anomalyDesc := services.DetectReadingAnomaly(reading)

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"status":              "processed",
		"anomaly_detected":    anomalyDetected,
		"anomaly_description": anomalyDesc,
	})
}
