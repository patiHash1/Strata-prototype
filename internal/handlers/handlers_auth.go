package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/auth/register ----

type registerRequest struct {
	CompanyName   string `json:"company_name"`
	DomainSlug    string `json:"domain_slug"`
	OwnerEmail    string `json:"owner_email"`
	OwnerPassword string `json:"owner_password"`
	OwnerFullName string `json:"owner_full_name"`
}

// RegisterResponse represents the payload returned on successful registration.
type RegisterResponse struct {
	OrgID       string `json:"org_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// registerHandler creates a new organization with its owner user.
//
//	@Summary		Register organization & owner
//	@Description	Creates a new organization and a user as its owner. Returns a JWT access token.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	registerRequest	true	"Registration payload"
//	@Success		201	{object}	RegisterResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		409	{object}	utils.Envelope
//	@Router			/api/v1/auth/register [post]
func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.CompanyName) {
		utils.WriteErr(w, http.StatusBadRequest, "company_name is required")
		return
	}
	if !utils.IsDomainSlug(req.DomainSlug) {
		utils.WriteErr(w, http.StatusBadRequest, "domain_slug must be lowercase alphanumeric with hyphens (max 100 chars)")
		return
	}
	if !utils.IsEmail(req.OwnerEmail) {
		utils.WriteErr(w, http.StatusBadRequest, "invalid owner_email")
		return
	}
	if !utils.MinLen(req.OwnerPassword, 8) {
		utils.WriteErr(w, http.StatusBadRequest, "owner_password must be at least 8 characters")
		return
	}
	if !utils.NotBlank(req.OwnerFullName) {
		utils.WriteErr(w, http.StatusBadRequest, "owner_full_name is required")
		return
	}

	org, err := a.Orgs.Create(r.Context(), req.DomainSlug, req.CompanyName)
	if err != nil {
		if errors.Is(err, services.ErrOrgAlreadyExists) {
			utils.WriteErr(w, http.StatusConflict, err.Error())
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not create organization")
		return
	}

	hash, err := a.Auth.HashPassword(req.OwnerPassword)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not process password")
		return
	}

	user, err := a.Users.Create(r.Context(), req.OwnerEmail, hash, req.OwnerFullName)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			utils.WriteErr(w, http.StatusConflict, err.Error())
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not create user")
		return
	}

	adminRole, err := a.RBAC.CreateRole(r.Context(), org.ID, "Admin", nil, nil)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create default role")
		return
	}

	if err := a.Users.AddMember(r.Context(), &services.OrganizationMember{
		OrgID:    org.ID,
		UserID:   user.ID,
		RoleID:   adminRole.ID,
		IsActive: true,
	}); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not add member")
		return
	}

	token, err := a.Auth.CreateToken(user.ID, org.ID, adminRole.ID, nil)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"org_id":       org.ID.String(),
		"user_id":      user.ID.String(),
		"access_token": token,
	})
}

// ---- POST /api/v1/auth/login ----

type loginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	MFACode  *string `json:"mfa_code,omitempty"`
}

// LoginResponse represents the payload returned on successful login.
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string `json:"refresh_token"`
	UserProfile  any    `json:"user_profile"`
}

// loginHandler authenticates a user and returns a JWT.
//
//	@Summary		User login
//	@Description	Authenticates with email/password (and optionally MFA code). Returns JWT + refresh token + profile.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	loginRequest	true	"Login payload"
//	@Success		200	{object}	LoginResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Router			/api/v1/auth/login [post]
func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Email) || !utils.NotBlank(req.Password) {
		utils.WriteErr(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := a.Users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not look up user")
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !a.Auth.VerifyPassword(user.PasswordHash, req.Password) {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if user.MFAEnabled {
		if req.MFACode == nil || *req.MFACode == "" {
			utils.WriteJSON(w, http.StatusOK, utils.Envelope{
				"mfa_required": true,
				"message":      "MFA code required",
			})
			return
		}
	}

	members, err := a.Users.ListMembersByUser(r.Context(), user.ID)
	if err != nil || len(members) == 0 {
		utils.WriteErr(w, http.StatusUnauthorized, "no organization membership found")
		return
	}

	orgID := members[0].OrgID
	roleID := members[0].RoleID

	perms, err := a.RBAC.GetPermissionKeysByRole(r.Context(), roleID)
	if err != nil {
		perms = []string{}
	}

	token, err := a.Auth.CreateToken(user.ID, orgID, roleID, perms)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	refreshToken := a.Auth.GenerateRefreshToken()

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"access_token":  token,
		"refresh_token": refreshToken,
		"user_profile": utils.Envelope{
			"id":        user.ID.String(),
			"email":     user.Email,
			"full_name": user.FullName,
		},
	})
}
