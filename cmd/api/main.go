package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
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

	// ── Seed super admin (idempotent) ──
	if cfg.SuperAdminUname != "" && cfg.SuperAdminPword != "" {
		if err := seedSuperAdmin(ctx, db.Pool, cfg, authSvc, userSvc, orgSvc, rbacSvc); err != nil {
			log.Printf("WARNING: super admin seeding failed (continuing): %v", err)
		} else {
			log.Println("super admin seeded")
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
			log.Printf("WARNING: Redis connection failed (continuing without Redis): %v", err)
			rdb.Close()
			rdb = nil
		} else {
			log.Println("redis connected")
		}
	}

	// ── Super Admin ──
	superAdminSvc := services.NewSuperAdminService(db.Pool, rdb)
	defer superAdminSvc.Shutdown()

	// ── Application ──
	app := handlers.New(cfg, db, authSvc, userSvc, orgSvc, rbacSvc, billingSvc, mailerSvc, crmSvc, accountingSvc, supplyChainSvc, hrSvc, platformSvc, superAdminSvc)

	// ── Signals ──
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("starting Strata API (port %d)", cfg.Port)

	if err := app.Serve(sigCtx); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// seedSuperAdmin creates the default super-admin organization, role, and user
// if they do not already exist. The super-admin role is granted the
// super_admin.access permission.
func seedSuperAdmin(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	authSvc *services.AuthService,
	userSvc *services.UserService,
	orgSvc *services.OrgService,
	rbacSvc *services.RBACService,
) error {
	const superAdminOrgSlug = "strata-system"

	// Check if super-admin org already exists.
	org, _ := orgSvc.GetByDomainSlug(ctx, superAdminOrgSlug)
	if org == nil {
		org, err := orgSvc.Create(ctx, superAdminOrgSlug, "Strata System")
		if err != nil {
			return err
		}
		log.Printf("  created super-admin org: %s", org.ID)
	}

	// Check if super-admin user already exists.
	user, _ := userSvc.GetByEmail(ctx, cfg.SuperAdminUname)
	if user == nil {
		hash, err := authSvc.HashPassword(cfg.SuperAdminPword)
		if err != nil {
			return err
		}
		user, err = userSvc.Create(ctx, cfg.SuperAdminUname, hash, "Super Admin")
		if err != nil {
			return err
		}
		log.Printf("  created super-admin user: %s", user.ID)
	}

	// Check if super-admin role already exists in the org.
	roles, err := rbacSvc.ListRolesByOrg(ctx, org.ID)
	if err != nil {
		return err
	}
	var superAdminRole *services.Role
	for i := range roles {
		if roles[i].Name == "Super Admin" {
			superAdminRole = &roles[i]
			break
		}
	}
	if superAdminRole == nil {
		permID, err := rbacSvc.GetPermissionIDByKey(ctx, services.PermSuperAdmin)
		if err != nil {
			return err
		}
		var permIDs []uuid.UUID
		if permID != uuid.Nil {
			permIDs = []uuid.UUID{permID}
		}
		superAdminRole, err = rbacSvc.CreateRole(ctx, org.ID, "Super Admin", nil, permIDs)
		if err != nil {
			return err
		}
		log.Printf("  created super-admin role: %s", superAdminRole.ID)
	}

	// Check if user is already a member of the org.
	member, _ := userSvc.GetMember(ctx, org.ID, user.ID)
	if member == nil {
		if err := userSvc.AddMember(ctx, &services.OrganizationMember{
			OrgID:    org.ID,
			UserID:   user.ID,
			RoleID:   superAdminRole.ID,
			IsActive: true,
		}); err != nil {
			return err
		}
		log.Printf("  added super-admin user to org")
	}

	return nil
}
