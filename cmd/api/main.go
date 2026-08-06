package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
	"github.com/patiHash1/Strata-prototype/internal/services"

	// Auto-registers the swagger spec so http-swagger can serve it.
	_ "github.com/patiHash1/Strata-prototype/docs"
)

//	@title			Strata API
//	@version		0.1.0
//	@description	ERP-CRM Hybrid API — direct competitor to Odoo.
//
//	@contact.name	Strata Support
//	@contact.email	support@strata.dev
//
//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT
//
//	@host			localhost:8080
//	@BasePath		/
//
//	@tag.name		System
//	@tag.description	System health and meta endpoints.
//	@tag.name		Account
//	@tag.description	User account management — own profile, update, delete, and organizations.
//	@tag.name		Auth
//	@tag.description	Authentication, registration, and multi-tenancy.
//	@tag.name		Organizations
//	@tag.description	Organization management, roles, members, API keys.
//	@tag.name		Billing
//	@tag.description	Subscription and billing management.
//	@tag.name		CRM
//	@tag.description	CRM & Revenue Operations — leads, quotes, risk analysis.
//	@tag.name		Accounting
//	@tag.description	Finance & Enterprise Accounting — journal entries, invoices, expenses.
//	@tag.name		Fleet
//	@tag.description	Fleet & Telematics — vehicle telemetry, route optimization.
//	@tag.name		Inventory
//	@tag.description	Supply Chain & Inventory — reorder predictions, stock management.
//	@tag.name		HR
//	@tag.description	HR, Workforce & Collaboration — attendance, ATS, knowledge base search.
//	@tag.name		AI & Platform
//	@tag.description	Platform, AI Core & BI — text-to-SQL copilot, workflow automation, security audit anomalies.
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter your Bearer token in the format: Bearer <token>
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key for programmatic access (e.g. fleet telematics ingestion)

func main() {
	cfg := config.Load()

	// ── Database (required) ──
	ctx := context.Background()

	if cfg.DB.DSN == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := database.New(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	// ── Run migrations ──
	log.Println("running migrations…")
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migrations complete")

	// ── Services ──
	authSvc := services.NewAuthService(cfg.JWTSecret, cfg.JWTIssuer)
	userSvc := services.NewUserService(db.Pool)
	orgSvc := services.NewOrgService(db.Pool)
	rbacSvc := services.NewRBACService(db.Pool)
	billingSvc := services.NewBillingService(db.Pool)
	mailerSvc := services.NewMailer()
	crmSvc := services.NewCRMService(db.Pool)
	accountingSvc := services.NewAccountingService(db.Pool)
	supplyChainSvc := services.NewSupplyChainService(db.Pool, authSvc)
	hrSvc := services.NewHRService(db.Pool)
	platformSvc := services.NewPlatformService(db.Pool)

	// ── Application ──
	app := handlers.New(cfg, db, authSvc, userSvc, orgSvc, rbacSvc, billingSvc, mailerSvc, crmSvc, accountingSvc, supplyChainSvc, hrSvc, platformSvc)

	// ── Signals ──
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("starting Strata API (port %d)", cfg.Port)

	if err := app.Serve(sigCtx); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
