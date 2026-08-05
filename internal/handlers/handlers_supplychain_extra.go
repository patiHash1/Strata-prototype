package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/inventory/receive ----

type receiveStockRequest struct {
	WarehouseID string  `json:"warehouse_id"`
	ProductID   string  `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	Reference   string  `json:"reference"`
}

// receiveStockHandler receives stock into a warehouse.
//
//	@Summary		Receive stock
//	@Description	Adds stock to a warehouse and records a receipt movement. Requires `inventory.receive` permission.
//	@Tags			Supply Chain
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	receiveStockRequest	true	"Stock receipt payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/inventory/receive [post]
func (a *App) receiveStockHandler(w http.ResponseWriter, r *http.Request) {
	var req receiveStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid warehouse_id")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid product_id")
		return
	}
	if req.Quantity <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "quantity must be positive")
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

	level, err := a.SupplyChain.ReceiveStock(r.Context(), orgID, warehouseID, productID, req.Quantity, req.Reference)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not receive stock")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"inventory_level_id": level.ID.String(),
		"warehouse_id":       level.WarehouseID.String(),
		"product_id":         level.ProductID.String(),
		"quantity_available": level.QuantityAvailable,
		"quantity_reserved":  level.QuantityReserved,
	})
}

// ---- POST /api/v1/inventory/issue ----

type issueStockRequest struct {
	WarehouseID string  `json:"warehouse_id"`
	ProductID   string  `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	Reference   string  `json:"reference"`
}

// issueStockHandler issues stock from a warehouse.
//
//	@Summary		Issue stock
//	@Description	Removes stock from a warehouse and records an issue movement. Requires `inventory.issue` permission.
//	@Tags			Supply Chain
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	issueStockRequest	true	"Stock issue payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/inventory/issue [post]
func (a *App) issueStockHandler(w http.ResponseWriter, r *http.Request) {
	var req issueStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid warehouse_id")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid product_id")
		return
	}
	if req.Quantity <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "quantity must be positive")
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

	level, err := a.SupplyChain.IssueStock(r.Context(), orgID, warehouseID, productID, req.Quantity, req.Reference)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not issue stock")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"inventory_level_id": level.ID.String(),
		"warehouse_id":       level.WarehouseID.String(),
		"product_id":         level.ProductID.String(),
		"quantity_available": level.QuantityAvailable,
		"quantity_reserved":  level.QuantityReserved,
	})
}

// ---- POST /api/v1/inventory/transfer ----

type transferStockRequest struct {
	FromWarehouseID string  `json:"from_warehouse_id"`
	ToWarehouseID   string  `json:"to_warehouse_id"`
	ProductID       string  `json:"product_id"`
	Quantity        float64 `json:"quantity"`
}

// transferStockHandler transfers stock between two warehouses.
//
//	@Summary		Transfer stock
//	@Description	Moves stock from one warehouse to another and records transfer movements. Requires `inventory.transfer` permission.
//	@Tags			Supply Chain
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	transferStockRequest	true	"Stock transfer payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/inventory/transfer [post]
func (a *App) transferStockHandler(w http.ResponseWriter, r *http.Request) {
	var req transferStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fromWarehouseID, err := uuid.Parse(req.FromWarehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid from_warehouse_id")
		return
	}
	toWarehouseID, err := uuid.Parse(req.ToWarehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid to_warehouse_id")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid product_id")
		return
	}
	if req.Quantity <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "quantity must be positive")
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

	fromLevel, toLevel, err := a.SupplyChain.TransferStock(r.Context(), orgID, fromWarehouseID, toWarehouseID, productID, req.Quantity)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not transfer stock")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"from_warehouse": map[string]interface{}{
			"warehouse_id":       fromLevel.WarehouseID.String(),
			"quantity_available": fromLevel.QuantityAvailable,
		},
		"to_warehouse": map[string]interface{}{
			"warehouse_id":       toLevel.WarehouseID.String(),
			"quantity_available": toLevel.QuantityAvailable,
		},
	})
}

// ---- GET /api/v1/inventory/snapshot ----

// getInventorySnapshotHandler returns the current inventory snapshot for a warehouse.
//
//	@Summary		Get inventory snapshot
//	@Description	Returns the current inventory levels for a specified warehouse. Requires `inventory.snapshot` permission.
//	@Tags			Supply Chain
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			warehouse_id	query	string	true	"Warehouse ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/inventory/snapshot [get]
func (a *App) getInventorySnapshotHandler(w http.ResponseWriter, r *http.Request) {
	warehouseIDStr := r.URL.Query().Get("warehouse_id")
	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid warehouse_id")
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

	levels, err := a.SupplyChain.GetInventorySnapshot(r.Context(), orgID, warehouseID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get inventory snapshot")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"warehouse_id": warehouseID.String(),
		"levels":       levels,
	})
}
