package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/patiHash1/Strata-prototype/internal/ai"
	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
	"github.com/patiHash1/Strata-prototype/internal/logger"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/redis/go-redis/v9"

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
//	@tag.name		Super Admin
//	@tag.description	System observability, SOC monitoring, partitioned maintenance, and CI health.
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
		logger.Error("DATABASE_URL is required")
		panic("DATABASE_URL is required")
	}

	db, err := database.New(ctx, cfg.DB.DSN)
	if err != nil {
		logger.Error("database connection failed", slog.String("error", err.Error()))
		panic(err)
	}
	defer db.Close()
	logger.Info("database connected")

	// ── Run migrations ──
	logger.Info("running migrations")
	if err := db.Migrate(ctx); err != nil {
		logger.Error("migration failed", slog.String("error", err.Error()))
		panic(err)
	}
	logger.Info("migrations complete")

	// ── Services ──
	authSvc := services.NewAuthService(cfg.JWTSecret, cfg.JWTIssuer)
	userSvc := services.NewUserService(db.Pool)
	orgSvc := services.NewOrgService(db.Pool)
	rbacSvc := services.NewRBACService(db.Pool)
	billingSvc := services.NewBillingService(db.Pool)
	mailerSvc := services.NewMailer()

	// ── AI inference (config-selected adapter) ──
	aiSvc := ai.New(ai.ProviderKind(cfg.AIProvider))

	crmSvc := services.NewCRMService(db.Pool, aiSvc)
	accountingSvc := services.NewAccountingService(db.Pool, aiSvc)
	supplyChainSvc := services.NewSupplyChainService(db.Pool, authSvc, aiSvc)
	hrSvc := services.NewHRService(db.Pool, aiSvc)
	platformSvc := services.NewPlatformService(db.Pool, aiSvc)

	// ── Seed super admin (idempotent) ──
	if cfg.SuperAdminUname != "" && cfg.SuperAdminPword != "" {
		seedSvc := services.NewSeedService(db.Pool, authSvc, userSvc, orgSvc, rbacSvc)
		if err := seedSvc.SeedSuperAdmin(ctx, cfg.SuperAdminUname, cfg.SuperAdminPword); err != nil {
			logger.Warn("super admin seeding failed (continuing)", slog.String("error", err.Error()))
		} else {
			logger.Info("super admin seeded")
		}
	}

	// ── Redis (optional) ──
	var rdb *redis.Client
	if cfg.Redis.Addr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.Warn("Redis connection failed (continuing without Redis)", slog.String("error", err.Error()))
			rdb.Close()
			rdb = nil
		} else {
			logger.Info("redis connected")
		}
	}

	// ── Super Admin ──
	superAdminSvc := services.NewSuperAdminService(db.Pool, rdb)
	superAdminSvc.SetUserSvc(userSvc)
	defer superAdminSvc.Shutdown()

	registrationSvc := services.NewRegistrationService(db.Pool)

	// ── Application ──
	app := handlers.New(cfg, db, authSvc, userSvc, orgSvc, rbacSvc, billingSvc, mailerSvc, crmSvc, accountingSvc, supplyChainSvc, hrSvc, platformSvc, superAdminSvc, registrationSvc)

	// ── Signals ──
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting Strata API", slog.Int("port", cfg.Port))

	if err := app.Serve(sigCtx); err != nil {
		logger.Error("server exited", slog.String("error", err.Error()))
		panic(err)
	}
}
