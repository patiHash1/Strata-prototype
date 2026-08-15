package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	// Generate and set a CSRF token for the login form.
	csrfToken, err := utils.GenerateCSRFToken()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	utils.SetCSRFCookie(w, csrfToken)

	component := templates.SuperAdminLoginView(csrfToken)
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
		templates.SuperAdminLoginView("").Render(r.Context(), w)
		return
	}

	// CSRF validation (double-submit cookie pattern).
	if !utils.ValidateCSRF(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView("", "Invalid or expired session. Please try again.").Render(r.Context(), w)
		return
	}

	// renderLoginError generates a fresh CSRF token and renders the error login page.
	renderLoginError := func(msg string) {
		tok, _ := utils.GenerateCSRFToken()
		utils.SetCSRFCookie(w, tok)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SuperAdminLoginErrorView(tok, msg).Render(r.Context(), w)
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if !utils.NotBlank(email) || !utils.NotBlank(password) {
		renderLoginError("Email and password are required.")
		return
	}

	if a.Users == nil || a.RBAC == nil || a.Auth == nil {
		renderLoginError("Service unavailable. Please try again later.")
		return
	}

	user, err := a.Users.GetByEmail(r.Context(), email)
	if err != nil || user == nil {
		renderLoginError("Invalid email or password.")
		return
	}

	if !a.Auth.VerifyPassword(user.PasswordHash, password) {
		renderLoginError("Invalid email or password.")
		return
	}

	members, err := a.Users.ListMembersByUser(r.Context(), user.ID)
	if err != nil || len(members) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderLoginError("No organization membership found.")
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
		renderLoginError("Access denied. Super admin privileges required.")
		return
	}

	token, err := a.Auth.CreateToken(user.ID, orgID, roleID, perms)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderLoginError("Could not generate session. Please try again.")
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

	// Publish SOC event for super-admin login.
	if a.SuperAdmin != nil {
		a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
			Type:      "super_admin.login",
			Severity:  "info",
			Message:   fmt.Sprintf("Super admin logged in: %s", email),
			IPAddress: r.RemoteAddr,
			UserID:    user.ID.String(),
		})
	}

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
	// Prevent browser caching of authenticated pages so that after logout
	// or session expiry, the back button cannot reveal cached content.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminDashboardView()
	component.Render(r.Context(), w)
}

// ── GET /api/v1/super-admin/dashboard/kpis ──

// dashboardKPIsHandler returns the real-time KPI fragment for HTMX polling.
func (a *App) dashboardKPIsHandler(w http.ResponseWriter, r *http.Request) {
	kpis := templates.DashboardKPIs{
		SystemUptime: "99.9%",
	}

	// Count active organizations.
	if _, orgTotal, err := a.Orgs.ListAllOrgs(r.Context(), 0, 1); err == nil {
		kpis.ActiveOrganizations = orgTotal
	}

	// Count total active users.
	if _, userTotal, err := a.Users.ListAllUsers(r.Context(), 0, 1); err == nil {
		kpis.TotalActiveUsers = userTotal
	}

	// Count recent security alerts (high/critical from ring buffer).
	if a.SuperAdmin != nil {
		events := a.SuperAdmin.RecentSOCEvents()
		for _, e := range events {
			if e.Severity == "high" || e.Severity == "critical" {
				kpis.SecurityAlerts++
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.DashboardKPIGrid(kpis).Render(r.Context(), w)
}

// DashboardKPIsHandlerForTest exposes the KPI handler for httptest.
func (a *App) DashboardKPIsHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.dashboardKPIsHandler(w, r)
}

// ── GET /api/v1/super-admin/dashboard/activity ──

// dashboardActivityHandler returns the recent activity fragment for HTMX polling.
func (a *App) dashboardActivityHandler(w http.ResponseWriter, r *http.Request) {
	var items []templates.ActivityItem

	if a.SuperAdmin != nil {
		events := a.SuperAdmin.RecentSOCEvents()
		// Take the latest 5 events (most recent are at the end).
		start := 0
		if len(events) > 5 {
			start = len(events) - 5
		}
		latest := events[start:]
		// Reverse so most recent appears first in the UI.
		for i := len(latest) - 1; i >= 0; i-- {
			e := latest[i]
			age := time.Since(e.Timestamp)
			items = append(items, templates.ActivityItem{
				Severity:  e.Severity,
				Message:   e.Message,
				Timestamp: humanDuration(age),
			})
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.DashboardActivityList(items).Render(r.Context(), w)
}

// DashboardActivityHandlerForTest exposes the activity handler for httptest.
func (a *App) DashboardActivityHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.dashboardActivityHandler(w, r)
}

// ── GET /api/v1/super-admin/dashboard/traffic ──

// dashboardTrafficHandler returns the real-time API traffic time-series as JSON
// for the ApexCharts client-side chart.
func (a *App) dashboardTrafficHandler(w http.ResponseWriter, r *http.Request) {
	var series []templates.TrafficDataPoint

	if a.SuperAdmin != nil {
		buckets := a.SuperAdmin.TrafficSeries()
		for _, b := range buckets {
			series = append(series, templates.TrafficDataPoint{
				Timestamp: b.Timestamp.UnixMilli(),
				Count2xx:  b.Count2xx,
				Count5xx:  b.Count5xx,
			})
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"series": series})
}

// DashboardTrafficHandlerForTest exposes the traffic handler for httptest.
func (a *App) DashboardTrafficHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.dashboardTrafficHandler(w, r)
}

// humanDuration returns a human-readable relative time string.
func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
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
	// Extract user info before clearing the cookie for the SOC event.
	var userID string
	if claims := utils.GetClaims(r); claims != nil {
		userID = claims.UserID
	}

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

	// Publish SOC event for super-admin logout.
	if a.SuperAdmin != nil {
		a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
			Type:      "super_admin.logout",
			Severity:  "info",
			Message:   "Super admin logged out",
			IPAddress: r.RemoteAddr,
			UserID:    userID,
		})
	}

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

// ── GET /api/v1/{module}/health ──

// getModuleHealthHandler returns the current state of a module including
// maintenance status, CI health, and HTTP metrics.
func (a *App) getModuleHealthHandler(w http.ResponseWriter, r *http.Request) {
	module := r.PathValue("module")
	if module == "" {
		utils.WriteErr(w, http.StatusBadRequest, "module is required")
		return
	}

	if a.SuperAdmin == nil {
		utils.WriteErr(w, http.StatusInternalServerError, "super admin service not available")
		return
	}

	status, err := a.SuperAdmin.GetModuleStatus(r.Context(), module)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to get module status: "+err.Error())
		return
	}

	// If the module is under maintenance, return 503.
	if status["status"] == "maintenance" {
		utils.WriteJSON(w, http.StatusServiceUnavailable, status)
		return
	}

	utils.WriteJSON(w, http.StatusOK, status)
}

// GetModuleHealthHandlerForTest exposes the module health handler for httptest.
func (a *App) GetModuleHealthHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.getModuleHealthHandler(w, r)
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

// ── GET /api/v1/super-admin/maintenance/rules ──

// listMaintenanceRulesPageHandler renders the full maintenance rules table
// inside the dashboard layout.
func (a *App) listMaintenanceRulesPageHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := a.SuperAdmin.ListAllMaintenanceRules(r.Context())
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to list maintenance rules: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.MaintenancePage(rules).Render(r.Context(), w)
}

// ListMaintenanceRulesPageHandlerForTest exposes the page handler for httptest.
func (a *App) ListMaintenanceRulesPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.listMaintenanceRulesPageHandler(w, r)
}

// ── GET /api/v1/super-admin/maintenance/fragment ──

// listMaintenanceRulesFragmentHandler returns an HTML table body fragment
// for HTMX partial swaps.
func (a *App) listMaintenanceRulesFragmentHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := a.SuperAdmin.ListAllMaintenanceRules(r.Context())
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to list maintenance rules: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.MaintenanceRulesTableBody(rules).Render(r.Context(), w)
}

// ListMaintenanceRulesFragmentHandlerForTest exposes the fragment handler for httptest.
func (a *App) ListMaintenanceRulesFragmentHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.listMaintenanceRulesFragmentHandler(w, r)
}

// ── POST /api/v1/super-admin/maintenance ──

// createMaintenanceRuleHandler creates a new maintenance rule and returns
// an HTMX out-of-band swap HTML fragment to append the row to the table.
func (a *App) createMaintenanceRuleHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		templates.MaintenanceValidationError("Invalid form data").Render(r.Context(), w)
		return
	}

	req := services.MaintenanceToggleRequest{
		Scope:        r.FormValue("scope"),
		TargetID:     r.FormValue("target_id"),
		Reason:       r.FormValue("reason"),
		AllowedRoles: []string{},
	}

	log.Printf("[maintenance] create request: scope=%q target_id=%q reason=%q", req.Scope, req.TargetID, req.Reason)

	// Validate required fields.
	if req.Scope == "" || req.TargetID == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		templates.MaintenanceValidationError("Scope and Target ID are required").Render(r.Context(), w)
		return
	}

	// Force is_active = TRUE on creation.
	req.IsActive = true

	rule, err := a.SuperAdmin.ToggleMaintenance(r.Context(), req)
	if err != nil {
		log.Printf("[maintenance] create error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		templates.MaintenanceValidationError("Failed to create rule: "+err.Error()).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	templates.MaintenanceRuleRowOOB(*rule).Render(r.Context(), w)

	// Publish SOC event for maintenance rule creation.
	if a.SuperAdmin != nil {
		a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
			Type:      "maintenance.created",
			Severity:  "warning",
			Message:   fmt.Sprintf("Maintenance rule created: %s/%s — %s", req.Scope, req.TargetID, req.Reason),
			IPAddress: r.RemoteAddr,
			Metadata: map[string]any{
				"scope":     req.Scope,
				"target_id": req.TargetID,
				"reason":    req.Reason,
			},
		})
	}
}

// CreateMaintenanceRuleHandlerForTest exposes the create handler for httptest.
func (a *App) CreateMaintenanceRuleHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.createMaintenanceRuleHandler(w, r)
}

// ── DELETE /api/v1/super-admin/maintenance/{id} ──

// deleteMaintenanceRuleHandler soft-deactivates a maintenance rule and returns
// an empty HTML fragment so the row is replaced with nothing (fade-out effect).
func (a *App) deleteMaintenanceRuleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		utils.WriteErr(w, http.StatusBadRequest, "missing rule id")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	if err := a.SuperAdmin.DeleteMaintenanceRule(r.Context(), id); err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "failed to delete maintenance rule: "+err.Error())
		return
	}

	// Return an empty <tr> so hx-swap="outerHTML swap:1s" replaces the row
	// and the swap:1s delay allows the CSS fade-out transition to play.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.MaintenanceRuleRowDeleted().Render(r.Context(), w)

	// Publish SOC event for maintenance rule revocation.
	if a.SuperAdmin != nil {
		a.SuperAdmin.PublishSOCEvent(r.Context(), services.SOCEvent{
			Type:      "maintenance.revoked",
			Severity:  "warning",
			Message:   fmt.Sprintf("Maintenance rule revoked (id=%d)", id),
			IPAddress: r.RemoteAddr,
			Metadata: map[string]any{
				"rule_id": id,
			},
		})
	}
}

// DeleteMaintenanceRuleHandlerForTest exposes the delete handler for httptest.
func (a *App) DeleteMaintenanceRuleHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.deleteMaintenanceRuleHandler(w, r)
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

	// Replay recent SOC events from the in-memory ring buffer so the client
	// sees events that occurred between page load and SSE connection open.
	for _, evt := range a.SuperAdmin.RecentSOCEvents() {
		event := templates.SecurityEvent{
			ID:        evt.ID,
			Type:      evt.Type,
			Severity:  evt.Severity,
			Message:   evt.Message,
			IPAddress: evt.IPAddress,
			Timestamp: evt.Timestamp,
		}
		var buf strings.Builder
		templates.SecurityEventRow(event).Render(r.Context(), &buf)
		fmt.Fprintf(w, "event: security-event\ndata: %s\n\n", buf.String())
	}
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

			// Render the SecurityEventRow component to a buffer.
			var buf strings.Builder
			templates.SecurityEventRow(event).Render(r.Context(), &buf)
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

// ── GET /api/v1/super-admin/metrics (HTML page) ──

// superAdminMetricsPageHandler renders the Metrics page with runtime telemetry.
//
//	@Summary		Metrics page (HTML)
//	@Description	Renders the Metrics page showing runtime telemetry and infrastructure profiling.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/metrics [get]
func (a *App) superAdminMetricsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminMetricsView()
	component.Render(r.Context(), w)
}

// MetricsPageHandlerForTest exposes the metrics page handler for httptest
// without the auth middleware chain.
func (a *App) MetricsPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminMetricsPageHandler(w, r)
}

// ── GET /api/v1/super-admin/security (HTML page) ──

// superAdminSecurityPageHandler renders the Security & Audit page with live streaming events.
//
//	@Summary		Security & Audit page (HTML)
//	@Description	Renders the Security page showing live streaming security events and audit trails.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/security [get]
func (a *App) superAdminSecurityPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminSecurityView()
	component.Render(r.Context(), w)
}

// SecurityPageHandlerForTest exposes the security page handler for httptest
// without the auth middleware chain.
func (a *App) SecurityPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminSecurityPageHandler(w, r)
}

// ── GET /api/v1/super-admin/settings (HTML page) ──

// superAdminSettingsPageHandler renders the Settings page for global configuration.
//
//	@Summary		Settings page (HTML)
//	@Description	Renders the Settings page for global environment config, API keys, and webhook setups.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/settings [get]
func (a *App) superAdminSettingsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminSettingsView()
	component.Render(r.Context(), w)
}

// SettingsPageHandlerForTest exposes the settings page handler for httptest
// without the auth middleware chain.
func (a *App) SettingsPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminSettingsPageHandler(w, r)
}

// ── GET /api/v1/super-admin/users (HTML page) ──

// superAdminUsersPageHandler renders the Users page with a global user directory.
//
//	@Summary		Users page (HTML)
//	@Description	Renders the Users page showing the global user directory and RBAC.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/users [get]
func (a *App) superAdminUsersPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminUsersView()
	component.Render(r.Context(), w)
}

// UsersPageHandlerForTest exposes the users page handler for httptest
// without the auth middleware chain.
func (a *App) UsersPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminUsersPageHandler(w, r)
}

// ── GET /api/v1/super-admin/users/fragment (HTML fragment) ──

// listUsersFragmentHandler returns the users table body as an HTMX-swappable fragment.
func (a *App) listUsersFragmentHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := parsePagination(r)
	users, total, err := a.Users.ListAllUsers(r.Context(), offset, limit)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.UsersTableBody(nil).Render(r.Context(), w)
		return
	}

	views := make([]templates.UserView, len(users))
	for i, u := range users {
		views[i] = templates.UserView{
			ID:        u.ID.String(),
			Email:     u.Email,
			FullName:  u.FullName,
			IsBanned:  u.IsBanned,
			BanReason: u.BanReason,
			CreatedAt: u.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.UsersTableBody(views).Render(r.Context(), w)

	// Also update the total count via OOB swap.
	countHTML := fmt.Sprintf("<span id=\"users-count\">%d users total</span>", total)
	w.Write([]byte(countHTML))
}

// ListUsersFragmentHandlerForTest exposes the users fragment handler for httptest.
func (a *App) ListUsersFragmentHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.listUsersFragmentHandler(w, r)
}

// ── GET /api/v1/super-admin/organizations (HTML page) ──

// superAdminOrganizationsPageHandler renders the Organizations page with a tenant directory.
//
//	@Summary		Organizations page (HTML)
//	@Description	Renders the Organizations page showing the tenant directory and provisioning.
//	@Tags			Super Admin
//	@Security		BearerAuth
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/super-admin/organizations [get]
func (a *App) superAdminOrganizationsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := templates.SuperAdminOrganizationsView()
	component.Render(r.Context(), w)
}

// OrganizationsPageHandlerForTest exposes the organizations page handler for httptest
// without the auth middleware chain.
func (a *App) OrganizationsPageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.superAdminOrganizationsPageHandler(w, r)
}

// ── GET /api/v1/super-admin/organizations/fragment (HTML fragment) ──

// listOrgsFragmentHandler returns the organizations table body as an HTMX-swappable fragment.
func (a *App) listOrgsFragmentHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := parsePagination(r)
	orgs, total, err := a.Orgs.ListAllOrgs(r.Context(), offset, limit)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.OrgsTableBody(nil).Render(r.Context(), w)
		return
	}

	views := make([]templates.OrgView, len(orgs))
	for i, o := range orgs {
		views[i] = templates.OrgView{
			ID:          o.ID.String(),
			CompanyName: o.CompanyName,
			DomainSlug:  o.DomainSlug,
			Status:      string(o.Status),
			CreatedAt:   o.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.OrgsTableBody(views).Render(r.Context(), w)

	countHTML := fmt.Sprintf("<span id=\"orgs-count\">%d organizations total</span>", total)
	w.Write([]byte(countHTML))
}

// ListOrgsFragmentHandlerForTest exposes the organizations fragment handler for httptest.
func (a *App) ListOrgsFragmentHandlerForTest(w http.ResponseWriter, r *http.Request) {
	a.listOrgsFragmentHandler(w, r)
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
