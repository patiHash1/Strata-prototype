package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

// Warehouse represents a storage location for inventory.
type Warehouse struct {
	ID      uuid.UUID `json:"id"`
	OrgID   uuid.UUID `json:"org_id"`
	Name    string    `json:"name"`
	Address *string   `json:"address,omitempty"`
}

// Product represents an inventory item.
type Product struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	UnitPrice      float64   `json:"unit_price"`
	CostPrice      float64   `json:"cost_price"`
	AIReorderPoint *int      `json:"ai_reorder_point,omitempty"`
}

// StockoutPrediction is an AI-generated reorder prediction for a product.
type StockoutPrediction struct {
	ProductID             uuid.UUID `json:"product_id"`
	SKU                   string    `json:"sku"`
	CurrentStock          int       `json:"current_stock"`
	PredictedStockoutDays int       `json:"predicted_stockout_days"`
	RecommendedReorderQty int       `json:"recommended_reorder_qty"`
}

// InventoryLevel represents the current stock level of a product at a warehouse.
type InventoryLevel struct {
	ID                uuid.UUID `json:"id"`
	OrgID             uuid.UUID `json:"org_id"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	ProductID         uuid.UUID `json:"product_id"`
	QuantityAvailable float64   `json:"quantity_available"`
	QuantityReserved  float64   `json:"quantity_reserved"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StockMovement records a change in inventory (receipt, issue, transfer, etc.).
type StockMovement struct {
	ID                 uuid.UUID  `json:"id"`
	OrgID              uuid.UUID  `json:"org_id"`
	WarehouseID        uuid.UUID  `json:"warehouse_id"`
	ProductID          uuid.UUID  `json:"product_id"`
	MovementType       string     `json:"movement_type"`
	Quantity           float64    `json:"quantity"`
	Reference          *string    `json:"reference,omitempty"`
	RelatedWarehouseID *uuid.UUID `json:"related_warehouse_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// StockMovementInput is the payload for recording a stock movement.
type StockMovementInput struct {
	MovementType       string     `json:"movement_type"`
	Quantity           float64    `json:"quantity"`
	Reference          *string    `json:"reference,omitempty"`
	RelatedWarehouseID *uuid.UUID `json:"related_warehouse_id,omitempty"`
}

// BOM represents a bill of materials.
type BOM struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	ParentProductID uuid.UUID `json:"parent_product_id"`
	BOMCode         string    `json:"bom_code"`
}

// BOMComponent represents a component within a bill of materials.
type BOMComponent struct {
	ID                 uuid.UUID `json:"id"`
	BOMID              uuid.UUID `json:"bom_id"`
	ComponentProductID uuid.UUID `json:"component_product_id"`
	QuantityRequired   float64   `json:"quantity_required"`
}

// WorkOrder represents a manufacturing work order.
type WorkOrder struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	BOMID            uuid.UUID `json:"bom_id"`
	Quantity         int       `json:"quantity"`
	Status           string    `json:"status"`
	ScheduledStart   *string   `json:"scheduled_start,omitempty"`
	ScheduledEnd     *string   `json:"scheduled_end,omitempty"`
	AIBottleneckRisk string    `json:"ai_bottleneck_risk"`
	CreatedAt        time.Time `json:"created_at"`
}

// PurchaseOrder represents a procurement purchase order.
type PurchaseOrder struct {
	ID                   uuid.UUID `json:"id"`
	OrgID                uuid.UUID `json:"org_id"`
	PONumber             string    `json:"po_number"`
	SupplierName         string    `json:"supplier_name"`
	TotalCost            float64   `json:"total_cost"`
	AISupplierRiskRating string    `json:"ai_supplier_risk_rating"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
}

// SupplierRiskReport is an AI-generated supplier risk assessment.
type SupplierRiskReport struct {
	SupplierName string  `json:"supplier_name"`
	RiskRating   string  `json:"risk_rating"`
	RiskScore    float64 `json:"risk_score"`
	OpenPOs      int     `json:"open_pos"`
	TotalSpend   float64 `json:"total_spend"`
}

// FleetVehicle represents a vehicle in the fleet.
type FleetVehicle struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	VIN          string    `json:"vin"`
	LicensePlate string    `json:"license_plate"`
	Make         string    `json:"make"`
	Model        string    `json:"model"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// FleetDriver represents a driver assigned to fleet operations.
type FleetDriver struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	LicenseNumber string     `json:"license_number"`
	SafetyRating  float64    `json:"safety_rating"`
}

// Shipment represents a freight shipment.
type Shipment struct {
	ID                 uuid.UUID  `json:"id"`
	OrgID              uuid.UUID  `json:"org_id"`
	TrackingNumber     string     `json:"tracking_number"`
	OriginAddress      string     `json:"origin_address"`
	DestinationAddress string     `json:"destination_address"`
	Status             string     `json:"status"`
	AssignedVehicleID  *uuid.UUID `json:"assigned_vehicle_id,omitempty"`
	AssignedDriverID   *uuid.UUID `json:"assigned_driver_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// TelematicsLog is a telemetry data point from a vehicle.
type TelematicsLog struct {
	ID           int64     `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	VehicleID    uuid.UUID `json:"vehicle_id"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	SpeedKMH     float64   `json:"speed_kmh"`
	FuelLevelPct *float64  `json:"fuel_level_pct,omitempty"`
	RecordedAt   time.Time `json:"recorded_at"`
}

// TelemetryIngestInput is the request payload for telemetry ingestion.
type TelemetryIngestInput struct {
	VehicleVIN  string   `json:"vehicle_vin"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	SpeedKMH    float64  `json:"speed_kmh"`
	EngineTempC *float64 `json:"engine_temp_c,omitempty"`
}

// TelemetryIngestResult is the response from telemetry ingestion.
type TelemetryIngestResult struct {
	Status                     string    `json:"status"`
	ProcessedAt                time.Time `json:"processed_at"`
	AIPredictiveAlertTriggered bool      `json:"ai_predictive_alert_triggered"`
}

// Waypoint represents a GeoJSON point in a route.
type Waypoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// RoutePlan is the result of an AI-optimized route generation.
type RoutePlan struct {
	RoutePlanID        uuid.UUID  `json:"route_plan_id"`
	OptimizedWaypoints []Waypoint `json:"optimized_waypoints"`
	PredictedETA       time.Time  `json:"predicted_eta"`
	CarbonOffsetKg     float64    `json:"carbon_offset_kg"`
}

// APIKeyRecord represents a stored API key row for validation purposes.
type apiKeyRecord struct {
	OrgID   uuid.UUID
	KeyHash string
	Scopes  []string
}

// ---- Repository ----

type supplyChainRepository struct {
	pool *pgxpool.Pool
}

func newSupplyChainRepository(pool *pgxpool.Pool) *supplyChainRepository {
	return &supplyChainRepository{pool: pool}
}

func (r *supplyChainRepository) GetVehicleByVIN(ctx context.Context, orgID uuid.UUID, vin string) (*FleetVehicle, error) {
	v := &FleetVehicle{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, vin, license_plate, make, model, status, created_at
		FROM fleet_vehicles WHERE org_id = $1 AND vin = $2
	`, orgID, vin).Scan(&v.ID, &v.OrgID, &v.VIN, &v.LicensePlate, &v.Make, &v.Model, &v.Status, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *supplyChainRepository) InsertTelematicsLog(ctx context.Context, log *TelematicsLog) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fleet_telematics_logs (org_id, vehicle_id, latitude, longitude, speed_kmh, fuel_level_pct, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.OrgID, log.VehicleID, log.Latitude, log.Longitude, log.SpeedKMH, log.FuelLevelPct, log.RecordedAt)
	return err
}

func (r *supplyChainRepository) GetShipmentsByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]Shipment, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, tracking_number, origin_address, destination_address, status, assigned_vehicle_id, assigned_driver_id, created_at
		FROM shipments WHERE org_id = $1 AND id = ANY($2)
	`, orgID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.OrgID, &s.TrackingNumber, &s.OriginAddress, &s.DestinationAddress, &s.Status, &s.AssignedVehicleID, &s.AssignedDriverID, &s.CreatedAt); err != nil {
			return nil, err
		}
		shipments = append(shipments, s)
	}
	return shipments, rows.Err()
}

func (r *supplyChainRepository) GetVehiclesByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]FleetVehicle, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, vin, license_plate, make, model, status, created_at
		FROM fleet_vehicles WHERE org_id = $1 AND id = ANY($2)
	`, orgID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []FleetVehicle
	for rows.Next() {
		var v FleetVehicle
		if err := rows.Scan(&v.ID, &v.OrgID, &v.VIN, &v.LicensePlate, &v.Make, &v.Model, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// GetActiveAPIKeyRecords fetches all non-expired API keys for bcrypt verification.
func (r *supplyChainRepository) GetActiveAPIKeyRecords(ctx context.Context) ([]apiKeyRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT org_id, key_hash, COALESCE(scopes, '{}') FROM api_keys
		WHERE expires_at IS NULL OR expires_at > NOW()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []apiKeyRecord
	for rows.Next() {
		var k apiKeyRecord
		if err := rows.Scan(&k.OrgID, &k.KeyHash, &k.Scopes); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *supplyChainRepository) GetProductsByOrg(ctx context.Context, orgID uuid.UUID) ([]Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, sku, name, unit_price, cost_price, ai_reorder_point
		FROM products WHERE org_id = $1 ORDER BY sku
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.OrgID, &p.SKU, &p.Name, &p.UnitPrice, &p.CostPrice, &p.AIReorderPoint); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if products == nil {
		products = []Product{}
	}
	return products, rows.Err()
}

func (r *supplyChainRepository) GetInventoryLevel(ctx context.Context, warehouseID, productID uuid.UUID) (*InventoryLevel, error) {
	level := &InventoryLevel{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, warehouse_id, product_id, quantity_available, quantity_reserved, updated_at
		FROM inventory_levels WHERE warehouse_id = $1 AND product_id = $2
	`, warehouseID, productID).Scan(&level.ID, &level.OrgID, &level.WarehouseID, &level.ProductID,
		&level.QuantityAvailable, &level.QuantityReserved, &level.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return level, nil
}

func (r *supplyChainRepository) UpsertInventoryLevel(ctx context.Context, level *InventoryLevel) error {
	level.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO inventory_levels (id, org_id, warehouse_id, product_id, quantity_available, quantity_reserved, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
			quantity_available = EXCLUDED.quantity_available,
			quantity_reserved  = EXCLUDED.quantity_reserved,
			updated_at         = EXCLUDED.updated_at
	`, level.ID, level.OrgID, level.WarehouseID, level.ProductID,
		level.QuantityAvailable, level.QuantityReserved, level.UpdatedAt)
	return err
}

func (r *supplyChainRepository) GetInventoryLevelsByWarehouse(ctx context.Context, orgID, warehouseID uuid.UUID) ([]InventoryLevel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, warehouse_id, product_id, quantity_available, quantity_reserved, updated_at
		FROM inventory_levels WHERE org_id = $1 AND warehouse_id = $2 ORDER BY product_id
	`, orgID, warehouseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var levels []InventoryLevel
	for rows.Next() {
		var l InventoryLevel
		if err := rows.Scan(&l.ID, &l.OrgID, &l.WarehouseID, &l.ProductID,
			&l.QuantityAvailable, &l.QuantityReserved, &l.UpdatedAt); err != nil {
			return nil, err
		}
		levels = append(levels, l)
	}
	if levels == nil {
		levels = []InventoryLevel{}
	}
	return levels, rows.Err()
}

func (r *supplyChainRepository) CreateStockMovement(ctx context.Context, movement *StockMovement) error {
	movement.ID = uuid.New()
	movement.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stock_movements (id, org_id, warehouse_id, product_id, movement_type, quantity, reference, related_warehouse_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, movement.ID, movement.OrgID, movement.WarehouseID, movement.ProductID,
		movement.MovementType, movement.Quantity, movement.Reference, movement.RelatedWarehouseID, movement.CreatedAt)
	return err
}

func (r *supplyChainRepository) CreateBOM(ctx context.Context, bom *BOM) error {
	bom.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bill_of_materials (id, org_id, parent_product_id, bom_code)
		VALUES ($1, $2, $3, $4)
	`, bom.ID, bom.OrgID, bom.ParentProductID, bom.BOMCode)
	return err
}

func (r *supplyChainRepository) CreateBOMComponent(ctx context.Context, comp *BOMComponent) error {
	comp.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bom_components (id, bom_id, component_product_id, quantity_required)
		VALUES ($1, $2, $3, $4)
	`, comp.ID, comp.BOMID, comp.ComponentProductID, comp.QuantityRequired)
	return err
}

func (r *supplyChainRepository) GetBOMByID(ctx context.Context, id uuid.UUID) (*BOM, error) {
	bom := &BOM{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, parent_product_id, bom_code
		FROM bill_of_materials WHERE id = $1
	`, id).Scan(&bom.ID, &bom.OrgID, &bom.ParentProductID, &bom.BOMCode)
	if err != nil {
		return nil, err
	}
	return bom, nil
}

func (r *supplyChainRepository) CreateWorkOrder(ctx context.Context, wo *WorkOrder) error {
	wo.ID = uuid.New()
	wo.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO work_orders (id, org_id, bom_id, quantity, status, scheduled_start, scheduled_end, ai_bottleneck_risk, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, wo.ID, wo.OrgID, wo.BOMID, wo.Quantity, wo.Status, wo.ScheduledStart, wo.ScheduledEnd, wo.AIBottleneckRisk, wo.CreatedAt)
	return err
}

func (r *supplyChainRepository) CreatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error {
	po.ID = uuid.New()
	po.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO purchase_orders (id, org_id, po_number, supplier_name, total_cost, ai_supplier_risk_rating, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, po.ID, po.OrgID, po.PONumber, po.SupplierName, po.TotalCost, po.AISupplierRiskRating, po.Status, po.CreatedAt)
	return err
}

func (r *supplyChainRepository) GetPurchaseOrdersBySupplier(ctx context.Context, orgID uuid.UUID, supplierName string) ([]PurchaseOrder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, po_number, supplier_name, total_cost, ai_supplier_risk_rating, status, created_at
		FROM purchase_orders WHERE org_id = $1 AND supplier_name = $2 ORDER BY created_at DESC
	`, orgID, supplierName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pos []PurchaseOrder
	for rows.Next() {
		var po PurchaseOrder
		if err := rows.Scan(&po.ID, &po.OrgID, &po.PONumber, &po.SupplierName, &po.TotalCost, &po.AISupplierRiskRating, &po.Status, &po.CreatedAt); err != nil {
			return nil, err
		}
		pos = append(pos, po)
	}
	if pos == nil {
		pos = []PurchaseOrder{}
	}
	return pos, rows.Err()
}

// ---- Service ----

// SupplyChainService handles supply chain, manufacturing, fleet, and inventory operations.
type SupplyChainService struct {
	repo    *supplyChainRepository
	authSvc *AuthService
}

// NewSupplyChainService creates a new SupplyChainService.
func NewSupplyChainService(pool *pgxpool.Pool, authSvc *AuthService) *SupplyChainService {
	return &SupplyChainService{repo: newSupplyChainRepository(pool), authSvc: authSvc}
}

// IngestTelemetry processes a vehicle telemetry data point from an API key-authenticated source.
func (s *SupplyChainService) IngestTelemetry(ctx context.Context, orgID uuid.UUID, input TelemetryIngestInput) (*TelemetryIngestResult, error) {
	vehicle, err := s.repo.GetVehicleByVIN(ctx, orgID, input.VehicleVIN)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrVehicleNotFound
		}
		return nil, err
	}

	var fuelLevelPct *float64
	var alertTriggered bool

	if input.EngineTempC != nil {
		if *input.EngineTempC > 110.0 {
			alertTriggered = true
		}
		estimatedFuel := 100.0 - (*input.EngineTempC / 1.5)
		if estimatedFuel < 0 {
			estimatedFuel = 0
		}
		if estimatedFuel > 100 {
			estimatedFuel = 100
		}
		fuelLevelPct = &estimatedFuel
	}

	if input.SpeedKMH > 130.0 {
		alertTriggered = true
	}

	now := time.Now()
	logEntry := &TelematicsLog{
		OrgID:        orgID,
		VehicleID:    vehicle.ID,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		SpeedKMH:     input.SpeedKMH,
		FuelLevelPct: fuelLevelPct,
		RecordedAt:   now,
	}

	if err := s.repo.InsertTelematicsLog(ctx, logEntry); err != nil {
		return nil, err
	}

	return &TelemetryIngestResult{
		Status:                     "queued",
		ProcessedAt:                now,
		AIPredictiveAlertTriggered: alertTriggered,
	}, nil
}

// OptimizeRoutes generates AI-optimized delivery routes for a set of shipments using available vehicles.
func (s *SupplyChainService) OptimizeRoutes(ctx context.Context, orgID uuid.UUID, shipmentIDs, vehicleIDs []uuid.UUID) (*RoutePlan, error) {
	if len(shipmentIDs) == 0 {
		return nil, ErrNoShipmentsProvided
	}
	if len(vehicleIDs) == 0 {
		return nil, ErrNoVehiclesProvided
	}

	shipments, err := s.repo.GetShipmentsByIDs(ctx, orgID, shipmentIDs)
	if err != nil {
		return nil, err
	}
	if len(shipments) == 0 {
		return nil, ErrShipmentsNotFound
	}

	vehicles, err := s.repo.GetVehiclesByIDs(ctx, orgID, vehicleIDs)
	if err != nil {
		return nil, err
	}
	if len(vehicles) == 0 {
		return nil, ErrVehiclesNotFound
	}

	routePlanID := uuid.New()
	waypoints := aiOptimizeRoute(shipments, vehicles)
	predictedETA := time.Now().Add(time.Duration(30+rand.Intn(120)) * time.Minute)
	carbonOffsetKg := float64(len(shipments))*2.5 + rand.Float64()*5.0

	return &RoutePlan{
		RoutePlanID:        routePlanID,
		OptimizedWaypoints: waypoints,
		PredictedETA:       predictedETA,
		CarbonOffsetKg:     carbonOffsetKg,
	}, nil
}

// GetReorderPredictions generates AI-driven reorder predictions for all products,
// optionally filtered by warehouse.
func (s *SupplyChainService) GetReorderPredictions(ctx context.Context, orgID uuid.UUID, warehouseID *uuid.UUID) ([]StockoutPrediction, error) {
	products, err := s.repo.GetProductsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Build a map of productID -> total quantity available from inventory_levels.
	stockMap := make(map[uuid.UUID]float64)
	if warehouseID != nil {
		levels, err := s.repo.GetInventoryLevelsByWarehouse(ctx, orgID, *warehouseID)
		if err != nil {
			return nil, err
		}
		for _, l := range levels {
			stockMap[l.ProductID] = l.QuantityAvailable
		}
	} else {
		// Sum across all warehouses by querying each product individually.
		// We iterate products and query inventory per product across all warehouses.
		for _, p := range products {
			var total float64
			err := s.repo.pool.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity_available), 0)
				FROM inventory_levels WHERE org_id = $1 AND product_id = $2
			`, orgID, p.ID).Scan(&total)
			if err != nil {
				return nil, err
			}
			stockMap[p.ID] = total
		}
	}

	var predictions []StockoutPrediction

	for _, p := range products {
		reorderPoint := 15
		if p.AIReorderPoint != nil {
			reorderPoint = *p.AIReorderPoint
		}

		currentStock := int(stockMap[p.ID])
		stockoutDays := aiPredictStockout(currentStock, reorderPoint)
		recommendedQty := aiRecommendReorderQty(currentStock, reorderPoint)

		predictions = append(predictions, StockoutPrediction{
			ProductID:             p.ID,
			SKU:                   p.SKU,
			CurrentStock:          currentStock,
			PredictedStockoutDays: stockoutDays,
			RecommendedReorderQty: recommendedQty,
		})
	}

	if predictions == nil {
		predictions = []StockoutPrediction{}
	}
	return predictions, nil
}

// ValidateAPIKey checks the raw API key against stored bcrypt hashes and returns
// the org ID and scopes if a match is found.
func (s *SupplyChainService) ValidateAPIKey(ctx context.Context, rawKey string) (uuid.UUID, []string, error) {
	keys, err := s.repo.GetActiveAPIKeyRecords(ctx)
	if err != nil {
		return uuid.Nil, nil, err
	}

	for _, k := range keys {
		if s.authSvc.VerifyPassword(k.KeyHash, rawKey) {
			if k.Scopes == nil {
				k.Scopes = []string{}
			}
			return k.OrgID, k.Scopes, nil
		}
	}

	return uuid.Nil, nil, ErrAPIKeyInvalid
}

// ---- BOM & Work Orders ----

// CreateBOMInput is the request payload for creating a BOM.
type CreateBOMInput struct {
	ParentProductID uuid.UUID                 `json:"parent_product_id"`
	BOMCode         string                    `json:"bom_code"`
	Components      []CreateBOMComponentInput `json:"components"`
}

// CreateBOMComponentInput is a component entry within a BOM creation request.
type CreateBOMComponentInput struct {
	ComponentProductID uuid.UUID `json:"component_product_id"`
	QuantityRequired   float64   `json:"quantity_required"`
}

// CreateBOM creates a bill of materials with its components.
func (s *SupplyChainService) CreateBOM(ctx context.Context, orgID uuid.UUID, input CreateBOMInput) (*BOM, error) {
	bom := &BOM{
		OrgID:           orgID,
		ParentProductID: input.ParentProductID,
		BOMCode:         input.BOMCode,
	}
	if err := s.repo.CreateBOM(ctx, bom); err != nil {
		return nil, err
	}

	for _, c := range input.Components {
		comp := &BOMComponent{
			BOMID:              bom.ID,
			ComponentProductID: c.ComponentProductID,
			QuantityRequired:   c.QuantityRequired,
		}
		if err := s.repo.CreateBOMComponent(ctx, comp); err != nil {
			return nil, err
		}
	}

	return bom, nil
}

// CreateWorkOrderInput is the request payload for creating a work order.
type CreateWorkOrderInput struct {
	BOMID          uuid.UUID `json:"bom_id"`
	Quantity       int       `json:"quantity"`
	ScheduledStart *string   `json:"scheduled_start,omitempty"`
	ScheduledEnd   *string   `json:"scheduled_end,omitempty"`
}

// CreateWorkOrder creates a work order with simulated AI bottleneck risk prediction.
func (s *SupplyChainService) CreateWorkOrder(ctx context.Context, orgID uuid.UUID, input CreateWorkOrderInput) (*WorkOrder, error) {
	if input.Quantity <= 0 {
		input.Quantity = 1
	}

	// Simulate AI bottleneck risk prediction
	riskRating := aiPredictBottleneckRisk(input.Quantity)

	wo := &WorkOrder{
		OrgID:            orgID,
		BOMID:            input.BOMID,
		Quantity:         input.Quantity,
		Status:           "planned",
		ScheduledStart:   input.ScheduledStart,
		ScheduledEnd:     input.ScheduledEnd,
		AIBottleneckRisk: riskRating,
	}
	if err := s.repo.CreateWorkOrder(ctx, wo); err != nil {
		return nil, err
	}

	return wo, nil
}

// ---- Procurement ----

// CreatePurchaseOrderInput is the request payload for creating a purchase order.
type CreatePurchaseOrderInput struct {
	PONumber     string  `json:"po_number"`
	SupplierName string  `json:"supplier_name"`
	TotalCost    float64 `json:"total_cost"`
}

// CreatePurchaseOrder creates a purchase order with simulated AI supplier risk rating.
func (s *SupplyChainService) CreatePurchaseOrder(ctx context.Context, orgID uuid.UUID, input CreatePurchaseOrderInput) (*PurchaseOrder, error) {
	riskRating := aiPredictSupplierRisk(input.SupplierName)

	po := &PurchaseOrder{
		OrgID:                orgID,
		PONumber:             input.PONumber,
		SupplierName:         input.SupplierName,
		TotalCost:            input.TotalCost,
		AISupplierRiskRating: riskRating,
		Status:               "draft",
	}
	if err := s.repo.CreatePurchaseOrder(ctx, po); err != nil {
		return nil, err
	}

	return po, nil
}

// GetSupplierRiskReport returns a simulated AI risk report for a supplier.
func (s *SupplyChainService) GetSupplierRiskReport(ctx context.Context, orgID uuid.UUID, supplierName string) (*SupplierRiskReport, error) {
	pos, err := s.repo.GetPurchaseOrdersBySupplier(ctx, orgID, supplierName)
	if err != nil {
		return nil, err
	}

	var totalSpend float64
	openPOs := 0
	for _, po := range pos {
		totalSpend += po.TotalCost
		if po.Status != "cancelled" && po.Status != "delivered" {
			openPOs++
		}
	}

	riskScore := aiCalculateSupplierRiskScore(supplierName, openPOs, totalSpend)
	riskRating := aiSupplierRiskRating(riskScore)

	return &SupplierRiskReport{
		SupplierName: supplierName,
		RiskRating:   riskRating,
		RiskScore:    riskScore,
		OpenPOs:      openPOs,
		TotalSpend:   totalSpend,
	}, nil
}

// ---- AI Simulation Helpers ----

func aiOptimizeRoute(shipments []Shipment, vehicles []FleetVehicle) []Waypoint {
	var waypoints []Waypoint

	baseLat := 40.7128 + (rand.Float64()-0.5)*2.0
	baseLng := -74.0060 + (rand.Float64()-0.5)*2.0

	for i, shipment := range shipments {
		_ = shipment.TrackingNumber
		waypoints = append(waypoints, Waypoint{
			Type:        "Point",
			Coordinates: []float64{baseLng + float64(i)*0.02, baseLat + float64(i)*0.02},
		})
	}

	waypoints = append(waypoints, Waypoint{
		Type:        "Point",
		Coordinates: []float64{baseLng + 0.1, baseLat + 0.1},
	})

	_ = vehicles
	return waypoints
}

func aiPredictStockout(currentStock, reorderPoint int) int {
	if currentStock <= 0 {
		return 0
	}
	dailyRate := 1 + rand.Intn(maxInt(1, currentStock/3))
	days := currentStock / dailyRate
	if days < 0 {
		return 0
	}
	if days > 60 {
		days = 60
	}
	return days
}

func aiRecommendReorderQty(currentStock, reorderPoint int) int {
	shortfall := reorderPoint - currentStock
	if shortfall <= 0 {
		return reorderPoint
	}
	buffer := float64(shortfall) * (0.2 + rand.Float64()*0.3)
	return shortfall + int(buffer)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func aiPredictBottleneckRisk(quantity int) string {
	if quantity > 100 {
		return "High"
	} else if quantity > 50 {
		return "Medium"
	}
	return "Low"
}

func aiPredictSupplierRisk(supplierName string) string {
	_ = supplierName
	riskLevels := []string{"Low Risk", "Medium Risk", "High Risk"}
	return riskLevels[rand.Intn(len(riskLevels))]
}

func aiCalculateSupplierRiskScore(supplierName string, openPOs int, totalSpend float64) float64 {
	_ = supplierName
	baseScore := 30.0 + rand.Float64()*40.0
	if openPOs > 5 {
		baseScore += 10.0
	}
	if totalSpend > 100000 {
		baseScore += 10.0
	}
	if baseScore > 100 {
		baseScore = 100
	}
	return baseScore
}

func aiSupplierRiskRating(score float64) string {
	if score >= 70 {
		return "High Risk"
	} else if score >= 40 {
		return "Medium Risk"
	}
	return "Low Risk"
}

// ---- Inventory Management ----

// ReceiveStock adds stock to a warehouse and records a receipt movement.
func (s *SupplyChainService) ReceiveStock(ctx context.Context, orgID, warehouseID, productID uuid.UUID, quantity float64, reference string) (*InventoryLevel, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	level, err := s.repo.GetInventoryLevel(ctx, warehouseID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			level = &InventoryLevel{
				ID:                uuid.New(),
				OrgID:             orgID,
				WarehouseID:       warehouseID,
				ProductID:         productID,
				QuantityAvailable: 0,
				QuantityReserved:  0,
			}
		} else {
			return nil, err
		}
	}

	level.QuantityAvailable += quantity
	if err := s.repo.UpsertInventoryLevel(ctx, level); err != nil {
		return nil, err
	}

	var ref *string
	if reference != "" {
		ref = &reference
	}
	movement := &StockMovement{
		OrgID:        orgID,
		WarehouseID:  warehouseID,
		ProductID:    productID,
		MovementType: "receipt",
		Quantity:     quantity,
		Reference:    ref,
	}
	if err := s.repo.CreateStockMovement(ctx, movement); err != nil {
		return nil, err
	}

	return level, nil
}

// IssueStock removes stock from a warehouse and records an issue movement.
func (s *SupplyChainService) IssueStock(ctx context.Context, orgID, warehouseID, productID uuid.UUID, quantity float64, reference string) (*InventoryLevel, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	level, err := s.repo.GetInventoryLevel(ctx, warehouseID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrInsufficientStock
		}
		return nil, err
	}

	if level.QuantityAvailable < quantity {
		return nil, ErrInsufficientStock
	}

	level.QuantityAvailable -= quantity
	if err := s.repo.UpsertInventoryLevel(ctx, level); err != nil {
		return nil, err
	}

	var ref *string
	if reference != "" {
		ref = &reference
	}
	movement := &StockMovement{
		OrgID:        orgID,
		WarehouseID:  warehouseID,
		ProductID:    productID,
		MovementType: "issue",
		Quantity:     quantity,
		Reference:    ref,
	}
	if err := s.repo.CreateStockMovement(ctx, movement); err != nil {
		return nil, err
	}

	return level, nil
}

// TransferStock moves stock between two warehouses and records transfer movements.
func (s *SupplyChainService) TransferStock(ctx context.Context, orgID, fromWarehouseID, toWarehouseID, productID uuid.UUID, quantity float64) (*InventoryLevel, *InventoryLevel, error) {
	if quantity <= 0 {
		return nil, nil, fmt.Errorf("quantity must be positive")
	}
	if fromWarehouseID == toWarehouseID {
		return nil, nil, fmt.Errorf("source and destination warehouses must differ")
	}

	// Deduct from source.
	fromLevel, err := s.repo.GetInventoryLevel(ctx, fromWarehouseID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrInsufficientStock
		}
		return nil, nil, err
	}
	if fromLevel.QuantityAvailable < quantity {
		return nil, nil, ErrInsufficientStock
	}
	fromLevel.QuantityAvailable -= quantity
	if err := s.repo.UpsertInventoryLevel(ctx, fromLevel); err != nil {
		return nil, nil, err
	}

	// Record issue movement at source.
	issueMovement := &StockMovement{
		OrgID:              orgID,
		WarehouseID:        fromWarehouseID,
		ProductID:          productID,
		MovementType:       "transfer_out",
		Quantity:           quantity,
		RelatedWarehouseID: &toWarehouseID,
	}
	if err := s.repo.CreateStockMovement(ctx, issueMovement); err != nil {
		return nil, nil, err
	}

	// Add to destination.
	toLevel, err := s.repo.GetInventoryLevel(ctx, toWarehouseID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			toLevel = &InventoryLevel{
				ID:                uuid.New(),
				OrgID:             orgID,
				WarehouseID:       toWarehouseID,
				ProductID:         productID,
				QuantityAvailable: 0,
				QuantityReserved:  0,
			}
		} else {
			return nil, nil, err
		}
	}
	toLevel.QuantityAvailable += quantity
	if err := s.repo.UpsertInventoryLevel(ctx, toLevel); err != nil {
		return nil, nil, err
	}

	// Record receipt movement at destination.
	receiptMovement := &StockMovement{
		OrgID:              orgID,
		WarehouseID:        toWarehouseID,
		ProductID:          productID,
		MovementType:       "transfer_in",
		Quantity:           quantity,
		RelatedWarehouseID: &fromWarehouseID,
	}
	if err := s.repo.CreateStockMovement(ctx, receiptMovement); err != nil {
		return nil, nil, err
	}

	return fromLevel, toLevel, nil
}

// GetInventorySnapshot returns all inventory levels for a given warehouse.
func (s *SupplyChainService) GetInventorySnapshot(ctx context.Context, orgID, warehouseID uuid.UUID) ([]InventoryLevel, error) {
	return s.repo.GetInventoryLevelsByWarehouse(ctx, orgID, warehouseID)
}

// Domain errors
var (
	ErrVehicleNotFound     = fmt.Errorf("vehicle not found")
	ErrNoShipmentsProvided = fmt.Errorf("at least one shipment_id is required")
	ErrNoVehiclesProvided  = fmt.Errorf("at least one vehicle_id is required")
	ErrShipmentsNotFound   = fmt.Errorf("shipments not found")
	ErrVehiclesNotFound    = fmt.Errorf("vehicles not found")
	ErrAPIKeyInvalid       = fmt.Errorf("invalid or expired API key")
	ErrInsufficientStock   = fmt.Errorf("insufficient stock available")
)
