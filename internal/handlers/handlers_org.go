package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/org/invitations ----

type inviteRequest struct {
	Email  string `json:"email"`
	RoleID string `json:"role_id"`
}

// inviteHandler sends an invitation to join the organization.
//
//	@Summary		Invite team member
//	@Description	Sends an invitation to join the organization. Requires `users.invite` permission.
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Param			body	body	inviteRequest	true	"Invitation payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/org/invitations [post]
func (a *App) inviteHandler(w http.ResponseWriter, r *http.Request) {
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.IsEmail(req.Email) {
		utils.WriteErr(w, http.StatusBadRequest, "invalid email")
		return
	}

	roleUUID, err := uuid.Parse(req.RoleID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid role_id")
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

	invToken := uuid.New().String() + uuid.New().String()

	inv := &services.OrganizationInvitation{
		OrgID:     orgID,
		Email:     req.Email,
		RoleID:    roleUUID,
		Token:     invToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := a.Orgs.CreateInvitation(r.Context(), inv); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create invitation")
		return
	}

	_ = a.Mailer.SendInvitation(req.Email, invToken)

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"invitation_id": inv.ID.String(),
		"token":         invToken,
		"expires_at":    inv.ExpiresAt.Format(time.RFC3339),
	})
}

// ---- POST /api/v1/org/roles ----

type createRoleRequest struct {
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids"`
}

// createRoleHandler creates a new dynamic role.
//
//	@Summary		Create dynamic role
//	@Description	Creates a new role with assigned permissions. Requires `rbac.manage` permission.
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Param			body	body	createRoleRequest	true	"Role payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/org/roles [post]
func (a *App) createRoleHandler(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Name) {
		utils.WriteErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.PermissionIDs) == 0 {
		utils.WriteErr(w, http.StatusBadRequest, "at least one permission_id is required")
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

	var permUUIDs []uuid.UUID
	for _, pid := range req.PermissionIDs {
		u, err := uuid.Parse(pid)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid permission_id: "+pid)
			return
		}
		permUUIDs = append(permUUIDs, u)
	}

	role, err := a.RBAC.CreateRole(r.Context(), orgID, req.Name, req.Description, permUUIDs)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create role")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"role_id":                    role.ID.String(),
		"name":                       role.Name,
		"assigned_permissions_count": len(permUUIDs),
	})
}

// ---- POST /api/v1/org/api-keys ----

type createAPIKeyRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

// createAPIKeyHandler generates a new API key.
//
//	@Summary		Generate API key
//	@Description	Creates a new API key for machine integrations. The plain-text secret is shown only once. Requires `apikeys.manage` permission.
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Param			body	body	createAPIKeyRequest	true	"API key payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/org/api-keys [post]
func (a *App) createAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
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

	rawSecret := uuid.New().String() + uuid.New().String()
	hash, err := a.Auth.HashPassword(rawSecret)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not generate key")
		return
	}

	if req.Scopes == nil {
		req.Scopes = []string{}
	}

	key := &services.APIKey{
		OrgID:   orgID,
		Name:    req.Name,
		KeyHash: hash,
		Scopes:  req.Scopes,
	}

	if req.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		key.ExpiresAt = &exp
	}

	if err := a.Orgs.CreateAPIKey(r.Context(), key); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create API key")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"api_key_id":        key.ID.String(),
		"plain_text_secret": rawSecret,
	})
}
