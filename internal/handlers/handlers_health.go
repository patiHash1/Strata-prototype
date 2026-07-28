package handlers

import (
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// HealthResponse represents the payload returned by the health endpoint.
type HealthResponse struct {
	Status   string `json:"status" example:"ok"`
	Database string `json:"database,omitempty" example:"connected"`
}

// healthHandler responds to GET /health with the service status.
//
//	@Summary		Service health check
//	@Description	Returns the current health status of the API and its dependencies (e.g. database).
//	@Tags			System
//	@Produce		json
//	@Success		200 {object} HealthResponse "Service is healthy"
//	@Failure		503 {object} HealthResponse "Database is unavailable"
//	@Router			/health [get]
func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	data := utils.Envelope{
		"status": "ok",
	}

	if a.DB != nil {
		if err := a.DB.Ping(r.Context()); err != nil {
			data["database"] = "unavailable"
			utils.WriteJSON(w, http.StatusServiceUnavailable, data)
			return
		}
		data["database"] = "connected"
	}

	utils.WriteJSON(w, http.StatusOK, data)
}
