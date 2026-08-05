package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/iot/readings/batch ----

type ingestReadingBatchRequest struct {
	Readings []ingestReadingRequest `json:"readings"`
}

// IngestReadingBatchResponse represents the result of a batch device reading ingestion.
type IngestReadingBatchResponse struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// ingestReadingBatchHandler processes multiple IoT device sensor readings in batch.
//
//	@Summary		Ingest device readings batch
//	@Description	Processes multiple sensor readings from IoT devices in batch for high-frequency ingestion. Requires `iot.readings.ingest` permission.
//	@Tags			BI & IoT
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	ingestReadingBatchRequest	true	"Batch IoT device readings payload"
//	@Success		200	{object}	IngestReadingBatchResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		500	{object}	utils.Envelope
//	@Router			/api/v1/iot/readings/batch [post]
func (a *App) ingestReadingBatchHandler(w http.ResponseWriter, r *http.Request) {
	var req ingestReadingBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Readings) == 0 {
		utils.WriteErr(w, http.StatusBadRequest, "readings array is required and must not be empty")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	readings := make([]services.IoTDeviceReading, 0, len(req.Readings))
	for _, r := range req.Readings {
		if r.DeviceID == uuid.Nil {
			utils.WriteErr(w, http.StatusBadRequest, "each reading must have a valid device_id")
			return
		}
		if !utils.NotBlank(r.MetricName) {
			utils.WriteErr(w, http.StatusBadRequest, "each reading must have a metric_name")
			return
		}
		readings = append(readings, services.IoTDeviceReading{
			DeviceID:    r.DeviceID,
			MetricName:  r.MetricName,
			MetricValue: r.MetricValue,
			Unit:        r.Unit,
			RecordedAt:  time.Now(),
		})
	}

	accepted, rejected, err := a.Platform.IngestDeviceReadingBatch(r.Context(), orgID, readings)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not ingest batch readings")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"accepted": accepted,
		"rejected": rejected,
	})
}
