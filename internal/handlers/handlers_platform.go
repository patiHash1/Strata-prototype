package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/ai/copilot/query ----

type copilotQueryRequest struct {
	Prompt string `json:"prompt"`
}

// CopilotQueryResponse represents the text-to-SQL AI copilot result.
type CopilotQueryResponse struct {
	GeneratedSQL        string                   `json:"generated_sql" example:"SELECT u.full_name AS sales_rep, SUM(d.amount) AS total_revenue FROM crm_deals d JOIN users u ON d.assigned_to = u.id WHERE d.stage = 'closed_won' AND d.created_at BETWEEN '2025-04-01' AND '2025-06-30' GROUP BY u.full_name ORDER BY total_revenue DESC LIMIT 5"`
	DataTable           []map[string]interface{} `json:"data_table"`
	ChartRecommendation string                   `json:"chart_recommendation" example:"bar_chart"`
}

// copilotQueryHandler executes a text-to-SQL AI copilot query.
//
//	@Summary		Execute AI copilot text-to-SQL query
//	@Description	Converts a natural language prompt to SQL, executes the simulated query, and returns results with a chart recommendation. Requires `copilot.use` permission.
//	@Tags			AI & Platform
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	copilotQueryRequest	true	"Natural language prompt"
//	@Success		200	{object}	CopilotQueryResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
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
	EventType string                 `json:"event_type"`
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
//	@Description	Triggers a low-code automation workflow in response to an event (e.g., invoice.paid). Matches active workflows by event_type and executes their action steps. Supports both Bearer token and webhook signature auth. Requires `workflows.execute` permission.
//	@Tags			AI & Platform
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	triggerWorkflowRequest	true	"Event trigger payload"
//	@Success		200	{object}	TriggerWorkflowResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
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
//	@Description	Retrieves security audit log entries flagged as anomalies by the AI detection system. Filterable by severity level. Requires `security.audit.read` permission.
//	@Tags			AI & Platform
//	@Produce		json
//	@Security		BearerAuth
//	@Param			severity	query	string	false	"Filter by severity: low, medium, high, critical"	default(high)
//	@Param			limit		query	int		false	"Maximum number of results"	default(20)
//	@Success		200	{object}	AuditAnomaliesResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
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
