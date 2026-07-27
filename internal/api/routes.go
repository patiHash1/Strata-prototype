package api

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// routes registers all application routes on the provided ServeMux.
// As the API grows, group related routes into separate methods
// (e.g. routesAuth, routesUsers) and call them from here.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("GET /health", a.healthHandler)

	// Swagger UI — only enabled in development.
	if a.Config.EnableSwagger {
		mux.Handle("GET /docs/", httpSwagger.Handler(
			httpSwagger.URL("/docs/doc.json"),
		))
	}

	// Stack middleware: innermost runs first.
	var handler http.Handler = mux
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler)
	handler = recoveryMiddleware(handler)

	return handler
}
