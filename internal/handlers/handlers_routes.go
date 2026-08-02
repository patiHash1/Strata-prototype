package handlers

import (
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// routes registers all application routes on the provided ServeMux.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// ── Public routes ──
	mux.HandleFunc("GET /health", a.healthHandler)
	mux.HandleFunc("POST /api/v1/auth/register", a.registerHandler)
	mux.HandleFunc("POST /api/v1/auth/login", a.loginHandler)

	// ── Protected routes (JWT + permission gates) ──
	mux.Handle("POST /api/v1/org/invitations",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersInvite)(
				http.HandlerFunc(a.inviteHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/roles",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermRBACManage)(
				http.HandlerFunc(a.createRoleHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/api-keys",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAPIKeysManage)(
				http.HandlerFunc(a.createAPIKeyHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/billing/subscriptions",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermBillingManage)(
				http.HandlerFunc(a.createSubscriptionHandler),
			),
		),
	)

	// ── Organization member management ──
	mux.Handle("PATCH /api/v1/org/members/{member_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.updateMemberHandler),
			),
		),
	)

	mux.Handle("DELETE /api/v1/org/members/{member_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.deleteMemberHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/members/{member_id}/remove",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.removeMemberHandler),
			),
		),
	)

	// ── CRM ──
	mux.Handle("POST /api/v1/crm/leads",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMLeadsWrite)(
				http.HandlerFunc(a.createLeadHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/quotes/risk-analysis",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMQuotesWrite)(
				http.HandlerFunc(a.analyzeRiskHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/tickets",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMTicketsWrite)(
				http.HandlerFunc(a.createTicketHandler),
			),
		),
	)

	// ── Swagger UI (development only) ──
	if a.Config.EnableSwagger {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	// ── Global middleware stack (outermost first) ──
	var handler http.Handler = mux
	handler = utils.CORSMiddleware(handler)
	handler = utils.LoggingMiddleware(handler)
	handler = utils.RecoveryMiddleware(handler)

	return handler
}
