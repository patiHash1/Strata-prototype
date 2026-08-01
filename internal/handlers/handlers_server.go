package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"
	"github.com/patiHash1/Strata-prototype/internal/services"
)

// App is the top-level application container.
type App struct {
	Config  config.Config
	DB      *database.DB
	Auth    *services.AuthService
	Users   *services.UserService
	Orgs    *services.OrgService
	RBAC    *services.RBACService
	Billing *services.BillingService
	Mailer  *services.Mailer
	CRM     *services.CRMService
	server  *http.Server
}

// New creates and wires an App with all dependencies.
func New(
	cfg config.Config,
	db *database.DB,
	authSvc *services.AuthService,
	userSvc *services.UserService,
	orgSvc *services.OrgService,
	rbacSvc *services.RBACService,
	billingSvc *services.BillingService,
	mailerSvc *services.Mailer,
	crmSvc *services.CRMService,
) *App {
	return &App{
		Config:  cfg,
		DB:      db,
		Auth:    authSvc,
		Users:   userSvc,
		Orgs:    orgSvc,
		RBAC:    rbacSvc,
		Billing: billingSvc,
		Mailer:  mailerSvc,
		CRM:     crmSvc,
	}
}

// Serve starts the HTTP server and blocks until a fatal error occurs
// or the context is cancelled.
func (a *App) Serve(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", a.Config.Port)

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("API server listening on %s", addr)

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Print("shutting down server…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
