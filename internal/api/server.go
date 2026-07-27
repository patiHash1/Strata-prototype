package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/database"
)

// App is the top-level application container. Everything the API needs
// lives here — config, database, external clients, etc.
// Add fields as the project grows so they are easily accessible from
// every handler (via the App receiver).
type App struct {
	Config config.Config
	DB     *database.DB
	server *http.Server
}

// New creates and wires an App with the given config and database.
func New(cfg config.Config, db *database.DB) *App {
	return &App{
		Config: cfg,
		DB:     db,
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

	// Run server in a goroutine so we can listen for ctx cancellation.
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
