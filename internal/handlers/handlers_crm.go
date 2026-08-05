package handlers

import (
	"encoding/json"
	"net/http"
	"time"

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

// ---- POST /api/v1/crm/tickets ----

type createTicketRequest struct {
	ContactID   string `json:"contact_id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

// CreateTicketResponse represents the payload returned when a support ticket is created.
type CreateTicketResponse struct {
	TicketID            string  `json:"ticket_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AISentimentScore    float64 `json:"ai_sentiment_score" example:"-0.45"`
	Priority            string  `json:"priority" example:"high"`
	AISuggestedResponse string  `json:"ai_suggested_response" example:"Thank you for reaching out..."`
}

// createTicketHandler creates a support ticket with AI sentiment analysis and auto-routing.
//
//	@Summary		Auto-route support ticket & analyze sentiment
//	@Description	Creates a helpdesk ticket, runs AI sentiment analysis on the description, auto-assigns priority and routing. Requires `crm.tickets.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createTicketRequest	true	"Ticket payload"
//	@Success		201	{object}	CreateTicketResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/crm/tickets [post]
func (a *App) createTicketHandler(w http.ResponseWriter, r *http.Request) {
	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contactID, err := uuid.Parse(req.ContactID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid contact_id")
		return
	}
	if !utils.NotBlank(req.Subject) {
		utils.WriteErr(w, http.StatusBadRequest, "subject is required")
		return
	}
	if !utils.NotBlank(req.Description) {
		utils.WriteErr(w, http.StatusBadRequest, "description is required")
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

	ticket, err := a.CRM.CreateTicket(r.Context(), orgID, contactID, req.Subject, req.Description)
	if err != nil {
		if err == services.ErrContactNotFound {
			utils.WriteErr(w, http.StatusNotFound, "contact not found")
			return
		}
		if err == services.ErrContactNotInOrg {
			utils.WriteErr(w, http.StatusNotFound, "contact not found in this organization")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not create ticket")
		return
	}

	sentimentScore := 0.0
	if ticket.AISentimentScore != nil {
		sentimentScore = *ticket.AISentimentScore
	}
	suggestedResponse := ""
	if ticket.AISuggestedResponse != nil {
		suggestedResponse = *ticket.AISuggestedResponse
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"ticket_id":             ticket.ID.String(),
		"ai_sentiment_score":    sentimentScore,
		"priority":              ticket.Priority,
		"ai_suggested_response": suggestedResponse,
	})
}

// ---- POST /api/v1/crm/field-visits ----

type scheduleFieldVisitRequest struct {
	ContactID    *string  `json:"contact_id,omitempty"`
	SalesRepID   *string  `json:"sales_rep_id,omitempty"`
	ScheduledAt  string   `json:"scheduled_at"`
	LocationLat  *float64 `json:"location_lat,omitempty"`
	LocationLong *float64 `json:"location_long,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

// ScheduleFieldVisitResponse represents the payload returned when a field visit is scheduled.
type ScheduleFieldVisitResponse struct {
	VisitID             string `json:"visit_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EstimatedTravelTime int    `json:"estimated_travel_time_minutes" example:"25"`
}

// scheduleFieldVisitHandler schedules a field sales visit with AI route optimization.
//
//	@Summary		Schedule field sales visit
//	@Description	Creates a field sales visit and simulates AI route optimization, returning estimated travel time. Requires `crm.fieldvisits.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	scheduleFieldVisitRequest	true	"Field visit payload"
//	@Success		201	{object}	ScheduleFieldVisitResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/crm/field-visits [post]
func (a *App) scheduleFieldVisitHandler(w http.ResponseWriter, r *http.Request) {
	var req scheduleFieldVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.ScheduledAt) {
		utils.WriteErr(w, http.StatusBadRequest, "scheduled_at is required")
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid scheduled_at format, use RFC3339")
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

	var contactID *uuid.UUID
	if req.ContactID != nil && *req.ContactID != "" {
		id, err := uuid.Parse(*req.ContactID)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid contact_id")
			return
		}
		contactID = &id
	}

	var salesRepID *uuid.UUID
	if req.SalesRepID != nil && *req.SalesRepID != "" {
		id, err := uuid.Parse(*req.SalesRepID)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid sales_rep_id")
			return
		}
		salesRepID = &id
	}

	visit, estimatedTravelTime, err := a.CRM.ScheduleFieldVisit(
		r.Context(),
		orgID,
		contactID,
		salesRepID,
		scheduledAt,
		req.LocationLat,
		req.LocationLong,
		req.Notes,
	)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not schedule field visit")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"visit_id":                      visit.ID.String(),
		"estimated_travel_time_minutes": estimatedTravelTime,
	})
}

// ---- POST /api/v1/crm/campaigns ----

type createCampaignRequest struct {
	Name    string   `json:"name"`
	Channel string   `json:"channel"`
	Budget  *float64 `json:"budget,omitempty"`
}

// CreateCampaignResponse represents the payload returned when a campaign is created.
type CreateCampaignResponse struct {
	CampaignID              string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AITargetSegmentCriteria string `json:"ai_target_segment_criteria" example:"{\"criteria\": \"contacts with open rate > 30%\"}"`
}

// createCampaignHandler creates a marketing campaign with AI audience segmentation.
//
//	@Summary		Create marketing campaign
//	@Description	Creates a new marketing campaign with AI-powered audience segmentation. Requires `crm.campaigns.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createCampaignRequest	true	"Campaign payload"
//	@Success		201	{object}	CreateCampaignResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/crm/campaigns [post]
func (a *App) createCampaignHandler(w http.ResponseWriter, r *http.Request) {
	var req createCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Name) {
		utils.WriteErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if !utils.NotBlank(req.Channel) {
		utils.WriteErr(w, http.StatusBadRequest, "channel is required")
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

	campaign, segmentCriteria, err := a.CRM.CreateCampaign(
		r.Context(),
		orgID,
		req.Name,
		req.Channel,
		req.Budget,
	)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create campaign")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"campaign_id":                campaign.ID.String(),
		"ai_target_segment_criteria": segmentCriteria,
	})
}

// ---- POST /api/v1/crm/campaigns/{campaign_id}/launch ----

type launchCampaignResponse struct {
	CampaignID     string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status         string `json:"status" example:"active"`
	EstimatedReach int    `json:"estimated_reach" example:"12500"`
}

// launchCampaignHandler launches a marketing campaign and returns estimated reach.
//
//	@Summary		Launch marketing campaign
//	@Description	Launches a draft campaign, activating it and returning AI-estimated reach. Requires `crm.campaigns.write` permission.
//	@Tags			CRM
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			campaign_id	path	string	true	"Campaign ID"
//	@Success		200	{object}	launchCampaignResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/crm/campaigns/{campaign_id}/launch [post]
func (a *App) launchCampaignHandler(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid campaign_id")
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

	campaign, estimatedReach, err := a.CRM.LaunchCampaign(r.Context(), orgID, campaignID)
	if err != nil {
		if err == services.ErrCampaignNotFound {
			utils.WriteErr(w, http.StatusNotFound, "campaign not found")
			return
		}
		if err == services.ErrCampaignNotInOrg {
			utils.WriteErr(w, http.StatusNotFound, "campaign not found in this organization")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not launch campaign")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"campaign_id":     campaign.ID.String(),
		"status":          campaign.Status,
		"estimated_reach": estimatedReach,
	})
}
