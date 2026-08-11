package handlers

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/static"
	"github.com/patiHash1/Strata-prototype/internal/templates"
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

// ── GET /api/v1/super-admin/login ──

// superAdminLoginHandler renders the Super Admin login page.
//
//	@Summary		Super Admin login page
//	@Description	Renders the Super Admin login form.
//	@Tags			Super Admin
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Router			/api/v1/super-admin/login [get]
func (a *App) superAdminLoginPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminLoginView()
	component.Render(r.Context(), w)
}

// ── POST /api/v1/super-admin/login ──

// superAdminLoginHandler processes the Super Admin login form, validates
// credentials, checks for super_admin.access permission, sets a JWT cookie,
// and redirects to the dashboard. On failure it re-renders the login page
// with an error message.
//
//	@Summary		Super Admin login
//	@Description	Authenticates a super admin and redirects to the dashboard.
//	@Tags			Super Admin
//	@Accept			x-www-form-urlencoded
//	@Produce		html
//	@Param			email		formData	string	true	"Email"
//	@Param			password	formData	string	true	"Password"
//	@Success		302	{string}	string	"Redirect to dashboard"
//	@Failure		401	{string}	string	"HTML login page with error"
//	@Router			/api/v1/super-admin/login [post]
func (a *App) superAdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginView().Render(r.Context(), w)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if !utils.NotBlank(email) || !utils.NotBlank(password) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Email and password are required.").Render(r.Context(), w)
		return
	}

	if a.Users == nil || a.RBAC == nil || a.Auth == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Service unavailable. Please try again later.").Render(r.Context(), w)
		return
	}

	user, err := a.Users.GetByEmail(r.Context(), email)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Invalid email or password.").Render(r.Context(), w)
		return
	}

	if !a.Auth.VerifyPassword(user.PasswordHash, password) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Invalid email or password.").Render(r.Context(), w)
		return
	}

	members, err := a.Users.ListMembersByUser(r.Context(), user.ID)
	if err != nil || len(members) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("No organization membership found.").Render(r.Context(), w)
		return
	}

	orgID := members[0].OrgID
	roleID := members[0].RoleID

	perms, err := a.RBAC.GetPermissionKeysByRole(r.Context(), roleID)
	if err != nil {
		perms = []string{}
	}

	hasSuperAdmin := false
	for _, p := range perms {
		if p == services.PermSuperAdmin {
			hasSuperAdmin = true
			break
		}
	}
	if !hasSuperAdmin {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Access denied. Super admin privileges required.").Render(r.Context(), w)
		return
	}

	token, err := a.Auth.CreateToken(user.ID, orgID, roleID, perms)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("Could not generate session. Please try again.").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "strata_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false, // set to true in production with TLS
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/api/v1/super-admin/dashboard", http.StatusFound)
}

// ── GET /api/v1/super-admin/dashboard ──

// superAdminDashboardHandler renders the Super Admin dashboard page.
//
//	@Summary		Super Admin dashboard
//	@Description	Renders the Super Admin dashboard with system metrics and quick actions.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/dashboard [get]
func (a *App) superAdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminDashboardView()
	component.Render(r.Context(), w)
}

// SuperAdminDashboardHandlerForTest exposes the dashboard handler for httptest
// without the auth middleware chain.
func (a *App) SuperAdminDashboardHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminDashboardHandler(w, r)
}

// ── POST /api/v1/super-admin/logout ──

// superAdminLogoutHandler clears the super-admin session cookie and the
// client-side Bearer token, then redirects to the login page.
//
//	@Summary		Super Admin logout
//	@Description	Invalidates the super-admin session by clearing the strata_token cookie and redirects to the login page.
//	@Tags			Super Admin
//	@Produce		html
//	@Success		302	{string}	string	"Redirect to login"
//	@Router			/api/v1/super-admin/logout [post]
func (a *App) superAdminLogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session cookie by setting an immediate expiry and an empty
	// value. MaxAge < 0 instructs the browser to delete the cookie immediately.
	http.SetCookie(w, &http.Cookie{
		Name:     "strata_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	// Tell any API clients holding a Bearer token in JS state to drop it.
	w.Header().Set("Clear-Site-Data", "\"cookies\"")

	http.Redirect(w, r, "/api/v1/super-admin/login", http.StatusFound)
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

// ── GET /api/v1/super-admin/metrics/fragment ──

// getSuperAdminMetricsFragmentHandler returns the MetricsGrid HTML fragment
// for HTMX polling. It renders only the grid component, not the full layout.
//
//	@Summary		Metrics grid fragment (HTML)
//	@Description	Returns the MetricsGrid Templ component as an HTML fragment for HTMX polling.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML fragment"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/metrics/fragment [get]
func (a *App) getSuperAdminMetricsFragmentHandler(w http.ResponseWriter, r *http.Request) {
	var metrics templates.SystemMetricsView

	if a.SuperAdmin != nil {
		snapshot := a.SuperAdmin.CollectSnapshot()
		metrics = templates.SystemMetricsView{
			AllocatedMB:   snapshot.Runtime.AllocatedMB,
			GCRuns:        snapshot.Runtime.GCRuns,
			Goroutines:    snapshot.Runtime.Goroutines,
			HeapObjects:   snapshot.Runtime.HeapObjects,
			AcquiredConns: snapshot.DB.AcquiredConns,
			IdleConns:     snapshot.DB.IdleConns,
			TotalConns:    snapshot.DB.TotalConns,
			MaxConns:      snapshot.DB.MaxConns,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.MetricsGrid(metrics).Render(r.Context(), w)
}

// MetricsFragmentHandlerForTest exposes the metrics fragment handler for
// httptest without the auth middleware chain.
func (a *App) MetricsFragmentHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.getSuperAdminMetricsFragmentHandler(w, r)
}

// SecurityStreamHandlerForTest exposes the SSE security stream handler for
// httptest without the auth middleware chain.
func (a *App) SecurityStreamHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.securityStreamHandler(w, r)
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
			// Parse the JSON SOC event and render it as an HTML fragment.
			var socEvent services.SOCEvent
			if err := json.Unmarshal(data, &socEvent); err != nil {
				continue
			}

			// Build the Templ SecurityEvent for rendering.
			event := templates.SecurityEvent{
				ID:        socEvent.ID,
				Type:      socEvent.Type,
				Severity:  socEvent.Severity,
				Message:   socEvent.Message,
				IPAddress: socEvent.IPAddress,
				Timestamp: socEvent.Timestamp,
			}

			// Render the SecurityLogEntry component to a buffer.
			var buf strings.Builder
			templates.SecurityLogEntry(event).Render(r.Context(), &buf)
			html := buf.String()

			// Write as SSE: event: security-event, data: <div>...</div>
			fmt.Fprintf(w, "event: security-event\ndata: %s\n\n", html)
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

// staticHandler serves embedded static assets (CSS, JS, images) with
// proper MIME types and aggressive caching headers. It strips the
// "/static/" prefix before looking up files in the embedded filesystem.
func (a *App) staticHandler() http.Handler {
	fs := http.FileServer(http.FS(static.Assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /static/ prefix so the embedded FS sees e.g. css/styles.css
		path := strings.TrimPrefix(r.URL.Path, "/static/")
		if path == "" || path == "/" {
			http.NotFound(w, r)
			return
		}
		r.URL.Path = path

		// Set aggressive caching headers for versioned assets.
		// In production, a reverse proxy or CDN can extend this further.
		ext := filepath.Ext(path)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Header().Set("Cache-Control", "public, max-age=86400")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
			w.Header().Set("Cache-Control", "public, max-age=86400")
		default:
			if ct := mime.TypeByExtension(ext); ct != "" {
				w.Header().Set("Content-Type", ct)
			}
		}

		fs.ServeHTTP(w, r)
	})
}

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
