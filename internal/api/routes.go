package api

import "net/http"

// routes registers all application routes on the provided ServeMux.
// As the API grows, group related routes into separate methods
// (e.g. routesAuth, routesUsers) and call them from here.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("GET /health", a.healthHandler)

	// Stack middleware: innermost runs first.
	var handler http.Handler = mux
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler)
	handler = recoveryMiddleware(handler)

	return handler
}
