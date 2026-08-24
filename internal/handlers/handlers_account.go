package handlers

import (
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- GET /api/v1/account ----

// AccountProfileResponse represents the authenticated user's own profile.
type AccountProfileResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Email       string  `json:"email" example:"alice@example.com"`
	FullName    string  `json:"full_name" example:"Alice Johnson"`
	PhoneNumber *string `json:"phone_number,omitempty" example:"+1-555-0100"`
	MFAEnabled  bool    `json:"mfa_enabled" example:"false"`
	CreatedAt   string  `json:"created_at" example:"2025-01-15T10:30:00Z"`
}

// getAccountHandler returns the authenticated user's profile.
//
//	@ID			getAccount
//	@Summary		Get own profile
//	@Description	Returns the profile of the currently authenticated user.
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	AccountProfileResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/account [get]
func (a *App) getAccountHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	user, err := a.Users.GetByID(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not retrieve profile")
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusNotFound, "user not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"id":           user.ID.String(),
		"email":        user.Email,
		"full_name":    user.FullName,
		"phone_number": user.PhoneNumber,
		"mfa_enabled":  user.MFAEnabled,
		"created_at":   user.CreatedAt,
	})
}

// ---- PATCH /api/v1/account ----

type updateAccountRequest struct {
	FullName    *string `json:"full_name,omitempty"`
	Email       *string `json:"email,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
}

// updateAccountHandler updates the authenticated user's profile fields.
//
//	@ID			updateAccount
//	@Summary		Update own profile
//	@Description	Partially updates profile fields (full_name, email, phone_number) for the authenticated user.
//	@Tags			Account
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	updateAccountRequest	true	"Fields to update"
//	@Success		200	{object}	AccountProfileResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		409	{object}	utils.Envelope
//	@Router			/api/v1/account [patch]
func (a *App) updateAccountHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req updateAccountRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	// Require at least one field.
	if req.FullName == nil && req.Email == nil && req.PhoneNumber == nil {
		utils.WriteErr(w, http.StatusBadRequest, "at least one field (full_name, email, phone_number) is required")
		return
	}

	// Validate fields when provided.
	if req.FullName != nil && !utils.NotBlank(*req.FullName) {
		utils.WriteErr(w, http.StatusBadRequest, "full_name must not be blank")
		return
	}
	if req.Email != nil && !utils.IsEmail(*req.Email) {
		utils.WriteErr(w, http.StatusBadRequest, "invalid email format")
		return
	}

	if err := a.Users.UpdateProfile(r.Context(), userID, req.FullName, req.Email, req.PhoneNumber); err != nil {
		writeServiceErr(w, err, "could not update profile")
		return
	}

	// Re-fetch the updated user.
	user, err := a.Users.GetByID(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not retrieve updated profile")
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusNotFound, "user not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"id":           user.ID.String(),
		"email":        user.Email,
		"full_name":    user.FullName,
		"phone_number": user.PhoneNumber,
		"mfa_enabled":  user.MFAEnabled,
		"created_at":   user.CreatedAt,
	})
}

// ---- DELETE /api/v1/account ----

// deleteAccountHandler removes the authenticated user's account.
//
//	@ID			deleteAccount
//	@Summary		Delete own account
//	@Description	Permanently deletes the authenticated user's account and all associated data.
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Success		204	"No Content"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/account [delete]
func (a *App) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if err := a.Users.DeleteAccount(r.Context(), userID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---- GET /api/v1/account/organizations ----

// AccountOrgResponse is a single organization membership for the authenticated user.
type AccountOrgResponse struct {
	OrgID    string `json:"org_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RoleID   string `json:"role_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	IsActive bool   `json:"is_active" example:"true"`
	JoinedAt string `json:"joined_at" example:"2025-01-15T10:30:00Z"`
}

// listMyOrganizationsHandler returns all org memberships for the authenticated user.
//
//	@ID			listMyOrganizations
//	@Summary		List my organizations
//	@Description	Returns all organization memberships for the currently authenticated user.
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}	AccountOrgResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/account/organizations [get]
func (a *App) listMyOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	members, err := a.Users.ListMembersByUser(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not retrieve organizations")
		return
	}

	orgs := make([]AccountOrgResponse, 0, len(members))
	for _, m := range members {
		orgs = append(orgs, AccountOrgResponse{
			OrgID:    m.OrgID.String(),
			RoleID:   m.RoleID.String(),
			IsActive: m.IsActive,
			JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"organizations": orgs,
	})
}
