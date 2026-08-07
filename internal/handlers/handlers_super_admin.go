package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ── Response types for Swagger ──

// MetricsResponse wraps the telemetry snapshot.
type MetricsResponse struct {
	Metrics services.TelemetrySnapshot `json:"metrics"`
}

// ModuleHealthResponse wraps module health scores.
type ModuleHealthResponse struct {
	Modules []services.ModuleHealth `json:"modules"`
}

// CIHealthIngestResponse wraps the CI health report.
type CIHealthIngestResponse struct {
	Report services.CIHealthReport `json:"report"`
}

// MaintenanceListResponse wraps the list of maintenance rules.
type MaintenanceListResponse struct {
	Rules []services.MaintenanceRule `json:"rules"`
}

// MaintenanceToggleResponse wraps the toggled maintenance rule.
type MaintenanceToggleResponse struct {
	Rule services.MaintenanceRule `json:"rule"`
}

// UserListResponse wraps a paginated list of all users.
type UserListResponse struct {
	Users []services.User `json:"users"`
	Total int             `json:"total"`
}

// OrgListResponse wraps a paginated list of all organizations.
type OrgListResponse struct {
	Organizations []services.Organization `json:"organizations"`
	Total         int                     `json:"total"`
}

// ── GET /api/v1/super-admin/metrics ──

// getSuperAdminMetricsHandler returns system telemetry in JSON format.
//
//	@Summary		System telemetry (JSON)
//	@Description	Returns aggregated runtime, database, and HTTP metrics including latency percentiles and recent panics.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	MetricsResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/metrics [get]
func (a *App) getSuperAdminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := a.SuperAdmin.CollectSnapshot()
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"metrics": snapshot})
}

// ── GET /api/v1/super-admin/metrics/prometheus ──

// getSuperAdminMetricsPrometheusHandler returns system telemetry in Prometheus text format.
//
//	@Summary		System telemetry (Prometheus)
//	@Description	Returns runtime, database, and HTTP metrics in Prometheus text exposition format suitable for scraping.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		text/plain
//	@Success		200	{string}	string	"Prometheus text format"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/metrics/prometheus [get]
func (a *App) getSuperAdminMetricsPrometheusHandler(w http.ResponseWriter, r *http.Request) {
	metrics := a.SuperAdmin.PrometheusMetrics()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, metrics)
}

// ── GET /api/v1/super-admin/health ──

// getSuperAdminHealthHandler returns module health scores.
//
//	@Summary		Module health scores
//	@Description	Returns composite health scores (0-100%) for all modules factoring in CI coverage, linter issues, vulnerabilities, and 5xx error rates.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	ModuleHealthResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/health [get]
func (a *App) getSuperAdminHealthHandler(w http.ResponseWriter, r *http.Request) {
	health, err := a.SuperAdmin.GetModuleHealth(r.Context())
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get module health: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"modules": health})
}

// ── POST /api/v1/super-admin/telemetry/ci-health ──

// ingestCIHealthHandler ingests CI health data.
//
//	@Summary		Ingest CI health report
//	@Description	Stores a CI health report with test coverage percentage, linter issue count, vulnerability count, and commit SHA for a given module.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	services.CIHealthIngestRequest	true	"CI health payload"
//	@Success		201	{object}	CIHealthIngestResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/telemetry/ci-health [post]
func (a *App) ingestCIHealthHandler(w http.ResponseWriter, r *http.Request) {
	var req services.CIHealthIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Module == "" {
		utils.WriteErr(w, http.StatusBadRequest, "module is required")
		return
	}

	report, err := a.SuperAdmin.IngestCIHealth(r.Context(), req)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to ingest CI health: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"report": report})
}

// ── GET /api/v1/super-admin/maintenance ──

// listMaintenanceHandler lists all active maintenance partitions.
//
//	@Summary		List active maintenance rules
//	@Description	Returns all currently active partitioned maintenance locks (by module, tenant, or feature).
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	MaintenanceListResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/maintenance [get]
func (a *App) listMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := a.SuperAdmin.ListMaintenanceRules(r.Context())
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to list maintenance rules: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"rules": rules})
}

// ── POST /api/v1/super-admin/maintenance/toggle ──

// toggleMaintenanceHandler activates or deactivates a maintenance partition.
//
//	@Summary		Toggle maintenance mode
//	@Description	Activates or deactivates a partitioned maintenance lock for a given scope (module, tenant_id, feature) and target. Publishes cache-invalidation via Redis Pub/Sub for multi-node sync.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	services.MaintenanceToggleRequest	true	"Maintenance toggle payload"
//	@Success		200	{object}	MaintenanceToggleResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/maintenance/toggle [post]
func (a *App) toggleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	var req services.MaintenanceToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Scope == "" || req.TargetID == "" {
		utils.WriteErr(w, http.StatusBadRequest, "scope and target_id are required")
		return
	}

	rule, err := a.SuperAdmin.ToggleMaintenance(r.Context(), req)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to toggle maintenance: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"rule": rule})
}

// ── GET /api/v1/super-admin/security/stream ──

// securityStreamHandler streams real-time SOC events via Server-Sent Events.
//
//	@Summary		Real-time security event stream (SSE)
//	@Description	Opens a Server-Sent Events connection that streams real-time SOC security events (failed auth, RLS violations, anomalies) as they occur across all nodes. Events are fanned out via Redis Pub/Sub.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		text/event-stream
//	@Success		200	{string}	string	"SSE event stream"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/security/stream [get]
func (a *App) securityStreamHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.WriteErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch, cleanup := a.SuperAdmin.AddSSESubscriber()
	defer cleanup()

	// Send initial connection event.
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			// Escape newlines in data for SSE format.
			payload := strings.ReplaceAll(string(data), "\n", " ")
			fmt.Fprintf(w, "event: security\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// ── GET /api/v1/super-admin/users ──

// listAllUsersHandler returns a paginated list of all users across all organizations.
//
//	@Summary		List all users
//	@Description	Returns a paginated list of all users across every organization, including ban status. Query params: offset (default 0), limit (default 50, max 100).
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			offset	query	int	false	"Pagination offset"
//	@Param			limit	query	int	false	"Page size (max 100)"
//	@Success		200	{object}	UserListResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/users [get]
func (a *App) listAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := parsePagination(r)
	users, total, err := a.Users.ListAllUsers(r.Context(), offset, limit)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to list users: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"users": users, "total": total})
}

// ── GET /api/v1/super-admin/users/{user_id} ──

// getUserDetailHandler returns details for a specific user.
//
//	@Summary		Get user details
//	@Description	Returns full details for a specific user by ID, including ban status and organization memberships.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			user_id	path	string	true	"User ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/users/{user_id} [get]
func (a *App) getUserDetailHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	user, err := a.Users.GetByID(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get user: "+err.Error())
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusNotFound, "user not found")
		return
	}

	memberships, _ := a.Users.ListMembersByUser(r.Context(), userID)

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"user":        user,
		"memberships": memberships,
	})
}

// ── POST /api/v1/super-admin/users/{user_id}/ban ──

type banUserRequest struct {
	Reason string `json:"reason"`
}

// banUserHandler bans a user across the entire platform.
//
//	@Summary		Ban a user
//	@Description	Bans a user platform-wide with a mandatory reason. Banned users cannot authenticate.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path	string			true	"User ID"
//	@Param			body	body	banUserRequest	true	"Ban reason"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/users/{user_id}/ban [post]
func (a *App) banUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	var req banUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reason == "" {
		utils.WriteErr(w, http.StatusBadRequest, "reason is required")
		return
	}

	user, err := a.Users.GetByID(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get user: "+err.Error())
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusNotFound, "user not found")
		return
	}

	if err := a.Users.BanUser(r.Context(), userID, req.Reason); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to ban user: "+err.Error())
		return
	}

	// Publish SOC event.
	a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
		Type:     "user.banned",
		Severity: "high",
		Message:  fmt.Sprintf("User %s banned: %s", user.Email, req.Reason),
		UserID:   userID.String(),
	})

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "user banned"})
}

// ── POST /api/v1/super-admin/users/{user_id}/unban ──

// unbanUserHandler removes a ban from a user.
//
//	@Summary		Unban a user
//	@Description	Removes a platform-wide ban from a user, restoring their ability to authenticate.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			user_id	path	string	true	"User ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/users/{user_id}/unban [post]
func (a *App) unbanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	user, err := a.Users.GetByID(r.Context(), userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get user: "+err.Error())
		return
	}
	if user == nil {
		utils.WriteErr(w, http.StatusNotFound, "user not found")
		return
	}

	if err := a.Users.UnbanUser(r.Context(), userID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to unban user: "+err.Error())
		return
	}

	a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
		Type:     "user.unbanned",
		Severity: "low",
		Message:  fmt.Sprintf("User %s unbanned", user.Email),
		UserID:   userID.String(),
	})

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "user unbanned"})
}

// ── GET /api/v1/super-admin/organizations ──

// listAllOrgsHandler returns a paginated list of all organizations.
//
//	@Summary		List all organizations
//	@Description	Returns a paginated list of all organizations with their status (active, suspended, pending_verification). Query params: offset (default 0), limit (default 50, max 100).
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			offset	query	int	false	"Pagination offset"
//	@Param			limit	query	int	false	"Page size (max 100)"
//	@Success		200	{object}	OrgListResponse
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/organizations [get]
func (a *App) listAllOrgsHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := parsePagination(r)
	orgs, total, err := a.Orgs.ListAllOrgs(r.Context(), offset, limit)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to list organizations: "+err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"organizations": orgs, "total": total})
}

// ── GET /api/v1/super-admin/organizations/{org_id} ──

// getOrgDetailHandler returns details for a specific organization.
//
//	@Summary		Get organization details
//	@Description	Returns full details for a specific organization by ID, including status and metadata.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			org_id	path	string	true	"Organization ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/organizations/{org_id} [get]
func (a *App) getOrgDetailHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(r.PathValue("org_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	org, err := a.Orgs.GetByID(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get organization: "+err.Error())
		return
	}
	if org == nil {
		utils.WriteErr(w, http.StatusNotFound, "organization not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"organization": org})
}

// ── POST /api/v1/super-admin/organizations/{org_id}/suspend ──

// suspendOrgHandler suspends an organization.
//
//	@Summary		Suspend an organization
//	@Description	Suspends an organization, preventing all members from accessing the platform.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			org_id	path	string	true	"Organization ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/organizations/{org_id}/suspend [post]
func (a *App) suspendOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(r.PathValue("org_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	org, err := a.Orgs.GetByID(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get organization: "+err.Error())
		return
	}
	if org == nil {
		utils.WriteErr(w, http.StatusNotFound, "organization not found")
		return
	}

	if err := a.Orgs.SuspendOrg(r.Context(), orgID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to suspend organization: "+err.Error())
		return
	}

	a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
		Type:     "org.suspended",
		Severity: "high",
		Message:  fmt.Sprintf("Organization %s (%s) suspended", org.CompanyName, org.DomainSlug),
		OrgID:    orgID.String(),
	})

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "organization suspended"})
}

// ── POST /api/v1/super-admin/organizations/{org_id}/activate ──

// activateOrgHandler activates (re-enables) an organization.
//
//	@Summary		Activate an organization
//	@Description	Re-activates a suspended or pending organization, restoring access for all members.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		json
//	@Param			org_id	path	string	true	"Organization ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/organizations/{org_id}/activate [post]
func (a *App) activateOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(r.PathValue("org_id"))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	org, err := a.Orgs.GetByID(r.Context(), orgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get organization: "+err.Error())
		return
	}
	if org == nil {
		utils.WriteErr(w, http.StatusNotFound, "organization not found")
		return
	}

	if err := a.Orgs.ActivateOrg(r.Context(), orgID); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to activate organization: "+err.Error())
		return
	}

	a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
		Type:     "org.activated",
		Severity: "low",
		Message:  fmt.Sprintf("Organization %s (%s) activated", org.CompanyName, org.DomainSlug),
		OrgID:    orgID.String(),
	})

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "organization activated"})
}

// ── Helpers ──

// parsePagination extracts offset and limit from query params with defaults.
func parsePagination(r *http.Request) (int, int) {
	offset := 0
	limit := 50

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}
	return offset, limit
}
