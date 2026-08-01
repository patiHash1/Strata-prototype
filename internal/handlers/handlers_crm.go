package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/crm/leads ----

type createLeadRequest struct {
	FirstName         string   `json:"first_name"`
	LastName          *string  `json:"last_name,omitempty"`
	Email             string   `json:"email"`
	CompanyName       *string  `json:"company_name,omitempty"`
	EstimatedDealSize *float64 `json:"estimated_deal_size,omitempty"`
}

// CreateLeadResponse represents the payload returned when a lead is created.
type CreateLeadResponse struct {
	ContactID        string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AIWinProbability int    `json:"ai_win_probability" example:"65"`
	AssignedTo       string `json:"assigned_to" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// createLeadHandler creates a new lead, triggers AI scoring, and returns the result.
//
//	@Summary		Create lead & trigger AI scoring
//	@Description	Creates a new CRM contact as a lead, runs AI win probability scoring, and creates a linked deal. Requires `crm.leads.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createLeadRequest	true	"Lead payload"
//	@Success		201	{object}	CreateLeadResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/crm/leads [post]
func (a *App) createLeadHandler(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.FirstName) {
		utils.WriteErr(w, http.StatusBadRequest, "first_name is required")
		return
	}
	if !utils.IsEmail(req.Email) {
		utils.WriteErr(w, http.StatusBadRequest, "invalid email")
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

	contact, assignedTo, aiWinProb, err := a.CRM.CreateLead(
		r.Context(),
		orgID,
		req.FirstName,
		req.LastName,
		req.Email,
		req.CompanyName,
		req.EstimatedDealSize,
	)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create lead")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"contact_id":         contact.ID.String(),
		"ai_win_probability": aiWinProb,
		"assigned_to":        assignedTo.String(),
	})
}

// ---- POST /api/v1/crm/quotes/risk-analysis ----

type analyzeRiskRequest struct {
	QuoteID      string `json:"quote_id"`
	ContractText string `json:"contract_text"`
}

// AnalyzeRiskResponse represents the payload returned from contract risk analysis.
type AnalyzeRiskResponse struct {
	AIRiskScore    float64                  `json:"ai_risk_score" example:"35.5"`
	FlaggedClauses []services.FlaggedClause `json:"flagged_clauses"`
}

// analyzeRiskHandler performs AI risk analysis on a quote's contract text.
//
//	@Summary		Analyze contract risk
//	@Description	Runs AI risk analysis on a quote's contract text, identifying flagged clauses with risk levels and suggested fixes. Requires `crm.quotes.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	analyzeRiskRequest	true	"Risk analysis payload"
//	@Success		200	{object}	AnalyzeRiskResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/crm/quotes/risk-analysis [post]
func (a *App) analyzeRiskHandler(w http.ResponseWriter, r *http.Request) {
	var req analyzeRiskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	quoteID, err := uuid.Parse(req.QuoteID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid quote_id")
		return
	}
	if !utils.NotBlank(req.ContractText) {
		utils.WriteErr(w, http.StatusBadRequest, "contract_text is required")
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

	riskScore, clauses, err := a.CRM.AnalyzeContractRisk(r.Context(), orgID, quoteID, req.ContractText)
	if err != nil {
		if err == services.ErrQuoteNotFound {
			utils.WriteErr(w, http.StatusNotFound, "quote not found")
			return
		}
		if err == services.ErrQuoteNotInOrg {
			utils.WriteErr(w, http.StatusNotFound, "quote not found in this organization")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not analyze contract risk")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"ai_risk_score":   riskScore,
		"flagged_clauses": clauses,
	})
}
