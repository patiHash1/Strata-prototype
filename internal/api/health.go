package api

import (
	"net/http"
)

// healthHandler responds to GET /health with the service status.
func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	data := envelope{
		"status": "ok",
	}

	// Optionally check the database.
	if a.DB != nil {
		if err := a.DB.Ping(r.Context()); err != nil {
			data["database"] = "unavailable"
			writeJSON(w, http.StatusServiceUnavailable, data)
			return
		}
		data["database"] = "connected"
	}

	writeJSON(w, http.StatusOK, data)
}
