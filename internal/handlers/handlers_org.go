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
//	@Security		BearerAuth
//	@Param			body	body	inviteRequest	true	"Invitation payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
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
//	@Security		BearerAuth
//	@Param			body	body	createRoleRequest	true	"Role payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
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

// ---- PATCH /api/v1/org/members/{member_id} ----

type updateMemberRequest struct {
	RoleID *string `json:"role_id,omitempty"`
}

// updateMemberHandler updates admin-managed information for an organization member.
//
//	@Summary		Update organization member
//	@Description	Updates a member's role or other admin-managed fields. Requires `users.manage` permission.
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			member_id	path	string	true	"Member ID"
//	@Param			body		body	updateMemberRequest	true	"Update payload"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/org/members/{member_id} [patch]
func (a *App) updateMemberHandler(w http.ResponseWriter, r *http.Request) {
	memberIDStr := r.PathValue("member_id")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid member_id")
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

	member, err := a.Users.GetMemberByID(r.Context(), memberID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not look up member")
		return
	}
	if member == nil {
		utils.WriteErr(w, http.StatusNotFound, "member not found")
		return
	}
	if member.OrgID != orgID {
		utils.WriteErr(w, http.StatusNotFound, "member not found in this organization")
		return
	}

	// Prevent self-targeting (admin cannot update their own membership via this endpoint)
	if member.UserID.String() == claims.UserID {
		utils.WriteErr(w, http.StatusForbidden, "cannot update your own membership through this endpoint")
		return
	}

	var req updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoleID != nil {
		roleUUID, err := uuid.Parse(*req.RoleID)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid role_id")
			return
		}

		// Verify the role belongs to the same org
		role, err := a.RBAC.GetRoleByID(r.Context(), roleUUID)
		if err != nil {
			utils.WriteErr(w, http.StatusInternalServerError, "could not verify role")
			return
		}
		if role == nil || role.OrgID != orgID {
			utils.WriteErr(w, http.StatusBadRequest, "role not found in this organization")
			return
		}

		if err := a.Users.UpdateMemberRole(r.Context(), memberID, roleUUID); err != nil {
			utils.WriteErr(w, http.StatusInternalServerError, "could not update member role")
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":   "member updated successfully",
		"member_id": memberID.String(),
	})
}

// ---- DELETE /api/v1/org/members/{member_id} ----

// deleteMemberHandler deactivates (soft-deletes) an organization member.
//
//	@Summary		Deactivate organization member
//	@Description	Deactivates (soft-deletes) a member account. Requires `users.manage` permission.
//	@Tags			Organizations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			member_id	path	string	true	"Member ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/org/members/{member_id} [delete]
func (a *App) deleteMemberHandler(w http.ResponseWriter, r *http.Request) {
	memberIDStr := r.PathValue("member_id")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid member_id")
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

	member, err := a.Users.GetMemberByID(r.Context(), memberID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not look up member")
		return
	}
	if member == nil {
		utils.WriteErr(w, http.StatusNotFound, "member not found")
		return
	}
	if member.OrgID != orgID {
		utils.WriteErr(w, http.StatusNotFound, "member not found in this organization")
		return
	}

	// Prevent self-deactivation
	if member.UserID.String() == claims.UserID {
		utils.WriteErr(w, http.StatusForbidden, "cannot deactivate your own membership")
		return
	}

	if !member.IsActive {
		utils.WriteErr(w, http.StatusBadRequest, "member is already deactivated")
		return
	}

	if err := a.Users.DeactivateMember(r.Context(), memberID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not deactivate member")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":   "member deactivated successfully",
		"member_id": memberID.String(),
	})
}

// ---- POST /api/v1/org/members/{member_id}/remove ----

// removeMemberHandler removes a member from the organization entirely (deletes the membership row).
// The user's account is NOT deleted — only removed from the org's member list.
//
//	@Summary		Remove member from organization
//	@Description	Removes a member from the organization's member list without deleting the user account. Requires `users.manage` permission.
//	@Tags			Organizations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			member_id	path	string	true	"Member ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/org/members/{member_id}/remove [post]
func (a *App) removeMemberHandler(w http.ResponseWriter, r *http.Request) {
	memberIDStr := r.PathValue("member_id")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid member_id")
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

	member, err := a.Users.GetMemberByID(r.Context(), memberID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not look up member")
		return
	}
	if member == nil {
		utils.WriteErr(w, http.StatusNotFound, "member not found")
		return
	}
	if member.OrgID != orgID {
		utils.WriteErr(w, http.StatusNotFound, "member not found in this organization")
		return
	}

	// Prevent self-removal
	if member.UserID.String() == claims.UserID {
		utils.WriteErr(w, http.StatusForbidden, "cannot remove yourself from the organization")
		return
	}

	if err := a.Users.RemoveMember(r.Context(), memberID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not remove member")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":   "member removed from organization successfully",
		"member_id": memberID.String(),
	})
}

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
//	@Security		BearerAuth
//	@Param			body	body	createAPIKeyRequest	true	"API key payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
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
