package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/fleet/telematics/ingest ----

type telemetryIngestRequest struct {
	VehicleVIN  string   `json:"vehicle_vin"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	SpeedKMH    float64  `json:"speed_kmh"`
	EngineTempC *float64 `json:"engine_temp_c,omitempty"`
}

// TelemetryIngestResponse represents the response from telemetry ingestion.
type TelemetryIngestResponse struct {
	Status                     string `json:"status" example:"queued"`
	ProcessedAt                string `json:"processed_at" example:"2025-01-15T14:30:00Z"`
	AIPredictiveAlertTriggered bool   `json:"ai_predictive_alert_triggered" example:"false"`
}

// ingestTelemetryHandler processes a vehicle telemetry data point.
//
//	@Summary		Ingest vehicle telemetry stream
//	@Description	Ingests real-time vehicle telemetry data (GPS, speed, engine temp) via API key authentication. Triggers AI predictive maintenance alerts when anomalies are detected. Requires API key with `fleet.telematics.ingest` scope.
//	@Tags			Fleet
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			body	body	telemetryIngestRequest	true	"Telemetry payload"
//	@Success		202	{object}	TelemetryIngestResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/fleet/telematics/ingest [post]
func (a *App) ingestTelemetryHandler(w http.ResponseWriter, r *http.Request) {
	claims := utils.GetAPIKeyClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "API key authentication required")
		return
	}

	var req telemetryIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.VehicleVIN) {
		utils.WriteErr(w, http.StatusBadRequest, "vehicle_vin is required")
		return
	}
	if req.SpeedKMH < 0 {
		utils.WriteErr(w, http.StatusBadRequest, "speed_kmh must be non-negative")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in API key")
		return
	}

	input := services.TelemetryIngestInput{
		VehicleVIN:  req.VehicleVIN,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		SpeedKMH:    req.SpeedKMH,
		EngineTempC: req.EngineTempC,
	}

	result, err := a.SupplyChain.IngestTelemetry(r.Context(), orgID, input)
	if err != nil {
		if err == services.ErrVehicleNotFound {
			utils.WriteErr(w, http.StatusNotFound, "vehicle not found")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not ingest telemetry")
		return
	}

	utils.WriteJSON(w, http.StatusAccepted, utils.Envelope{
		"status":                        result.Status,
		"processed_at":                  result.ProcessedAt.Format(time.RFC3339),
		"ai_predictive_alert_triggered": result.AIPredictiveAlertTriggered,
	})
}

// ---- POST /api/v1/fleet/routes/optimize ----

type routeOptimizeRequest struct {
	ShipmentIDs         []string `json:"shipment_ids"`
	AvailableVehicleIDs []string `json:"available_vehicle_ids"`
}

// RouteOptimizeResponse represents the response from route optimization.
type RouteOptimizeResponse struct {
	RoutePlanID        string              `json:"route_plan_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OptimizedWaypoints []services.Waypoint `json:"optimized_waypoints"`
	PredictedETA       string              `json:"predicted_eta" example:"2025-01-15T16:45:00Z"`
	CarbonOffsetKg     float64             `json:"carbon_offset_kg" example:"12.45"`
}

// optimizeRoutesHandler generates AI-optimized delivery routes.
//
//	@Summary		Generate AI optimized routes & ETAs
//	@Description	Generates AI-optimized delivery routes for a set of shipments using available vehicles. Returns a route plan with GeoJSON waypoints, ETA, and carbon offset estimate. Requires `fleet.routes.manage` permission.
//	@Tags			Fleet
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	routeOptimizeRequest	true	"Route optimization payload"
//	@Success		200	{object}	RouteOptimizeResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/fleet/routes/optimize [post]
func (a *App) optimizeRoutesHandler(w http.ResponseWriter, r *http.Request) {
	var req routeOptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.ShipmentIDs) == 0 {
		utils.WriteErr(w, http.StatusBadRequest, "at least one shipment_id is required")
		return
	}
	if len(req.AvailableVehicleIDs) == 0 {
		utils.WriteErr(w, http.StatusBadRequest, "at least one vehicle_id is required")
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

	shipmentIDs := make([]uuid.UUID, 0, len(req.ShipmentIDs))
	for _, sid := range req.ShipmentIDs {
		id, err := uuid.Parse(sid)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid shipment_id: "+sid)
			return
		}
		shipmentIDs = append(shipmentIDs, id)
	}

	vehicleIDs := make([]uuid.UUID, 0, len(req.AvailableVehicleIDs))
	for _, vid := range req.AvailableVehicleIDs {
		id, err := uuid.Parse(vid)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid vehicle_id: "+vid)
			return
		}
		vehicleIDs = append(vehicleIDs, id)
	}

	plan, err := a.SupplyChain.OptimizeRoutes(r.Context(), orgID, shipmentIDs, vehicleIDs)
	if err != nil {
		if err == services.ErrNoShipmentsProvided || err == services.ErrNoVehiclesProvided {
			utils.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == services.ErrShipmentsNotFound || err == services.ErrVehiclesNotFound {
			utils.WriteErr(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not optimize routes")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"route_plan_id":       plan.RoutePlanID.String(),
		"optimized_waypoints": plan.OptimizedWaypoints,
		"predicted_eta":       plan.PredictedETA.Format(time.RFC3339),
		"carbon_offset_kg":    plan.CarbonOffsetKg,
	})
}

// ---- GET /api/v1/inventory/reorder-predictions ----

// ReorderPredictionsResponse represents the response from reorder prediction queries.
type ReorderPredictionsResponse struct {
	Predictions []services.StockoutPrediction `json:"predictions"`
}

// getReorderPredictionsHandler returns AI-generated reorder predictions for inventory.
//
//	@Summary		Get AI reorder & stockout predictions
//	@Description	Returns AI-driven reorder predictions and stockout forecasts for all products in the organization. Optionally filtered by warehouse. Requires `inventory.read` permission.
//	@Tags			Inventory
//	@Produce		json
//	@Security		BearerAuth
//	@Param			warehouse_id	query	string	false	"Warehouse UUID to filter predictions"
//	@Success		200	{object}	ReorderPredictionsResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/inventory/reorder-predictions [get]
func (a *App) getReorderPredictionsHandler(w http.ResponseWriter, r *http.Request) {
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

	var warehouseID *uuid.UUID
	if raw := r.URL.Query().Get("warehouse_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid warehouse_id")
			return
		}
		warehouseID = &id
	}

	predictions, err := a.SupplyChain.GetReorderPredictions(r.Context(), orgID, warehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get reorder predictions")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"predictions": predictions,
	})
}

// ---- POST /api/v1/manufacturing/boms ----

type createBOMRequest struct {
	ParentProductID string                    `json:"parent_product_id"`
	BOMCode         string                    `json:"bom_code"`
	Components      []createBOMComponentEntry `json:"components"`
}

type createBOMComponentEntry struct {
	ComponentProductID string  `json:"component_product_id"`
	QuantityRequired   float64 `json:"quantity_required"`
}

// createBOMHandler creates a bill of materials with components.
//
//	@Summary		Create a bill of materials
//	@Description	Creates a bill of materials (BOM) with its component products. Requires `manufacturing.boms.write` permission.
//	@Tags			Manufacturing
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createBOMRequest	true	"BOM creation payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/manufacturing/boms [post]
func (a *App) createBOMHandler(w http.ResponseWriter, r *http.Request) {
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

	var req createBOMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.BOMCode) {
		utils.WriteErr(w, http.StatusBadRequest, "bom_code is required")
		return
	}

	parentProductID, err := uuid.Parse(req.ParentProductID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid parent_product_id")
		return
	}

	var components []services.CreateBOMComponentInput
	for _, c := range req.Components {
		compID, err := uuid.Parse(c.ComponentProductID)
		if err != nil {
			utils.WriteErr(w, http.StatusBadRequest, "invalid component_product_id: "+c.ComponentProductID)
			return
		}
		components = append(components, services.CreateBOMComponentInput{
			ComponentProductID: compID,
			QuantityRequired:   c.QuantityRequired,
		})
	}

	input := services.CreateBOMInput{
		ParentProductID: parentProductID,
		BOMCode:         req.BOMCode,
		Components:      components,
	}

	bom, err := a.SupplyChain.CreateBOM(r.Context(), orgID, input)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create BOM")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":                bom.ID.String(),
		"org_id":            bom.OrgID.String(),
		"parent_product_id": bom.ParentProductID.String(),
		"bom_code":          bom.BOMCode,
	})
}

// ---- POST /api/v1/manufacturing/work-orders ----

type createWorkOrderRequest struct {
	BOMID          string  `json:"bom_id"`
	Quantity       int     `json:"quantity"`
	ScheduledStart *string `json:"scheduled_start,omitempty"`
	ScheduledEnd   *string `json:"scheduled_end,omitempty"`
}

// createWorkOrderHandler creates a manufacturing work order.
//
//	@Summary		Create a work order
//	@Description	Creates a manufacturing work order with AI bottleneck risk prediction. Requires `manufacturing.workorders.write` permission.
//	@Tags			Manufacturing
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createWorkOrderRequest	true	"Work order creation payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/manufacturing/work-orders [post]
func (a *App) createWorkOrderHandler(w http.ResponseWriter, r *http.Request) {
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

	var req createWorkOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	bomID, err := uuid.Parse(req.BOMID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid bom_id")
		return
	}

	input := services.CreateWorkOrderInput{
		BOMID:          bomID,
		Quantity:       req.Quantity,
		ScheduledStart: req.ScheduledStart,
		ScheduledEnd:   req.ScheduledEnd,
	}

	wo, err := a.SupplyChain.CreateWorkOrder(r.Context(), orgID, input)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create work order")
		return
	}

	resp := utils.Envelope{
		"id":                 wo.ID.String(),
		"org_id":             wo.OrgID.String(),
		"bom_id":             wo.BOMID.String(),
		"quantity":           wo.Quantity,
		"status":             wo.Status,
		"ai_bottleneck_risk": wo.AIBottleneckRisk,
		"created_at":         wo.CreatedAt.Format(time.RFC3339),
	}
	if wo.ScheduledStart != nil {
		resp["scheduled_start"] = *wo.ScheduledStart
	}
	if wo.ScheduledEnd != nil {
		resp["scheduled_end"] = *wo.ScheduledEnd
	}

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// ---- POST /api/v1/procurement/purchase-orders ----

type createPurchaseOrderRequest struct {
	PONumber     string  `json:"po_number"`
	SupplierName string  `json:"supplier_name"`
	TotalCost    float64 `json:"total_cost"`
}

// createPurchaseOrderHandler creates a procurement purchase order.
//
//	@Summary		Create a purchase order
//	@Description	Creates a purchase order with AI supplier risk rating. Requires `procurement.po.write` permission.
//	@Tags			Procurement
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createPurchaseOrderRequest	true	"Purchase order creation payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/procurement/purchase-orders [post]
func (a *App) createPurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
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

	var req createPurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.PONumber) {
		utils.WriteErr(w, http.StatusBadRequest, "po_number is required")
		return
	}
	if !utils.NotBlank(req.SupplierName) {
		utils.WriteErr(w, http.StatusBadRequest, "supplier_name is required")
		return
	}

	input := services.CreatePurchaseOrderInput{
		PONumber:     req.PONumber,
		SupplierName: req.SupplierName,
		TotalCost:    req.TotalCost,
	}

	po, err := a.SupplyChain.CreatePurchaseOrder(r.Context(), orgID, input)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create purchase order")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"id":                      po.ID.String(),
		"org_id":                  po.OrgID.String(),
		"po_number":               po.PONumber,
		"supplier_name":           po.SupplierName,
		"total_cost":              po.TotalCost,
		"ai_supplier_risk_rating": po.AISupplierRiskRating,
		"status":                  po.Status,
		"created_at":              po.CreatedAt.Format(time.RFC3339),
	})
}

// ---- GET /api/v1/procurement/supplier-risk ----

// getSupplierRiskHandler returns an AI-generated supplier risk report.
//
//	@Summary		Get supplier risk report
//	@Description	Returns an AI-generated risk report for a supplier including risk score, open POs, and total spend. Requires `procurement.supplier.read` permission.
//	@Tags			Procurement
//	@Produce		json
//	@Security		BearerAuth
//	@Param			supplier_name	query	string	true	"Supplier name to assess"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/procurement/supplier-risk [get]
func (a *App) getSupplierRiskHandler(w http.ResponseWriter, r *http.Request) {
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

	supplierName := r.URL.Query().Get("supplier_name")
	if !utils.NotBlank(supplierName) {
		utils.WriteErr(w, http.StatusBadRequest, "supplier_name query parameter is required")
		return
	}

	report, err := a.SupplyChain.GetSupplierRiskReport(r.Context(), orgID, supplierName)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get supplier risk report")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"supplier_name": report.SupplierName,
		"risk_rating":   report.RiskRating,
		"risk_score":    report.RiskScore,
		"open_pos":      report.OpenPOs,
		"total_spend":   report.TotalSpend,
	})
}
