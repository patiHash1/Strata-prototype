package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/patiHash1/Strata-prototype/internal/api"
	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"

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

func main() {
	// Load configuration from environment.
	cfg := config.Load()

	// Database is optional at this stage — pass nil to skip DB checks.
	var db *database.DB

	// Build the application.
	app := api.New(cfg, db)

	// Listen for OS signals for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("starting Strata API (port %d)", cfg.Port)

	if err := app.Serve(ctx); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
