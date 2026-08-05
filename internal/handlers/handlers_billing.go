package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/billing/subscriptions ----

type createSubscriptionRequest struct {
	PlanCode        string `json:"plan_code"`
	PaymentMethodID string `json:"payment_method_id"`
}

// createSubscriptionHandler creates or upgrades a subscription.
//
//	@Summary		Create / upgrade subscription
//	@Description	Creates or upgrades a Stripe subscription for the organization. Requires `billing.manage` permission.
//	@Tags			Billing
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createSubscriptionRequest	true	"Subscription payload"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/billing/subscriptions [post]
func (a *App) createSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req createSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.PlanCode) {
		utils.WriteErr(w, http.StatusBadRequest, "plan_code is required")
		return
	}
	if !utils.NotBlank(req.PaymentMethodID) {
		utils.WriteErr(w, http.StatusBadRequest, "payment_method_id is required")
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

	sub, err := a.Billing.CreateOrUpgrade(r.Context(), orgID, req.PlanCode)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create/upgrade subscription")
		return
	}

	periodEnd := sub.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z07:00")

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"subscription_id":    sub.ID.String(),
		"status":             string(sub.Status),
		"current_period_end": periodEnd,
	})
}
