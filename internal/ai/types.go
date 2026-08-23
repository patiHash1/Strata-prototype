package ai

import "time"

// Request is the interface implemented by every typed inference request.
type Request interface {
	// capability identifies the request type for dispatch and logging.
	capability() string
}

// Response is the interface implemented by every typed inference response.
type Response interface {
	// response is a marker so the type system can distinguish responses.
	response()
}

// ── CRM ────────────────────────────────────────────────────────────────

// SentimentRequest asks for sentiment analysis of a support ticket.
type SentimentRequest struct {
	Subject     string
	Description string
}

func (*SentimentRequest) capability() string { return "crm.sentiment" }

// SentimentResponse carries the sentiment score, a suggested reply, and the
// derived priority (folded in from the deterministic derivation).
type SentimentResponse struct {
	Score             float64
	SuggestedResponse string
	Priority          string
}

func (*SentimentResponse) response() {}

// ContractRiskRequest asks for contract risk analysis of a quote.
type ContractRiskRequest struct {
	ContractText string
}

func (*ContractRiskRequest) capability() string { return "crm.contract_risk" }

// FlaggedClause is a risky clause detected in a contract.
type FlaggedClause struct {
	Clause       string
	RiskLevel    string // low, medium, high, critical
	SuggestedFix string
}

// ContractRiskResponse carries the risk score and flagged clauses.
type ContractRiskResponse struct {
	RiskScore float64
	Clauses   []FlaggedClause
}

func (*ContractRiskResponse) response() {}

// ── Accounting ─────────────────────────────────────────────────────────

// OCRRequest asks for invoice OCR extraction.
type OCRRequest struct {
	FileName string
	FileSize int64
}

func (*OCRRequest) capability() string { return "accounting.ocr" }

// OCRLineItem is a single line extracted from an invoice.
type OCRLineItem struct {
	Description string
	Quantity    int
	UnitPrice   float64
	Total       float64
}

// OCRResponse carries the extracted invoice data.
type OCRResponse struct {
	VendorName    string
	InvoiceNumber string
	LineItems     []OCRLineItem
	TaxAmount     float64
	TotalAmount   float64
}

func (*OCRResponse) response() {}

// ExpenseAuditRequest asks for fraud audit of an expense submission.
type ExpenseAuditRequest struct {
	Amount           float64
	Category         string
	ReceiptFileName  string
	SubmittedWeekend bool
}

func (*ExpenseAuditRequest) capability() string { return "accounting.expense_audit" }

// ExpenseAuditResponse carries the fraud flag and audit notes.
type ExpenseAuditResponse struct {
	FraudFlag  bool
	AuditNotes string
}

func (*ExpenseAuditResponse) response() {}

// ── Supply Chain ───────────────────────────────────────────────────────

// Shipment is a minimal route-optimization input (ID kept as a string so the
// ai package stays dependency-free).
type Shipment struct {
	ID         string
	TrackingNo string
}

// FleetVehicle is a minimal route-optimization input.
type FleetVehicle struct {
	ID           string
	LicensePlate string
}

// RouteOptimizeRequest asks for route optimization across shipments/vehicles.
type RouteOptimizeRequest struct {
	Shipments []Shipment
	Vehicles  []FleetVehicle
}

func (*RouteOptimizeRequest) capability() string { return "supplychain.route_optimize" }

// Waypoint is a point on an optimized route.
type Waypoint struct {
	Type        string
	Coordinates []float64
}

// RouteOptimizeResponse carries the optimized waypoints.
type RouteOptimizeResponse struct {
	Waypoints []Waypoint
}

func (*RouteOptimizeResponse) response() {}

// StockoutRequest asks for a stockout prediction and reorder recommendation.
type StockoutRequest struct {
	CurrentStock int
	ReorderPoint int
}

func (*StockoutRequest) capability() string { return "supplychain.stockout" }

// StockoutResponse carries the predicted stockout days and recommended reorder
// quantity (folded in from the deterministic derivation).
type StockoutResponse struct {
	PredictedStockoutDays int
	RecommendedReorderQty int
}

func (*StockoutResponse) response() {}

// SupplierRiskRequest asks for a supplier risk score.
type SupplierRiskRequest struct {
	SupplierName string
	OpenPOs      int
	TotalSpend   float64
}

func (*SupplierRiskRequest) capability() string { return "supplychain.supplier_risk" }

// SupplierRiskResponse carries the risk score and derived rating.
type SupplierRiskResponse struct {
	Score  float64
	Rating string
}

func (*SupplierRiskResponse) response() {}

// ── HR ─────────────────────────────────────────────────────────────────

// ResumeParseRequest asks for resume extraction.
type ResumeParseRequest struct {
	Data     []byte
	FileName string
}

func (*ResumeParseRequest) capability() string { return "hr.resume_parse" }

// ResumeParseResponse carries the extracted candidate data.
type ResumeParseResponse struct {
	Name   string
	Email  string
	Skills []string
}

func (*ResumeParseResponse) response() {}

// MatchScoreRequest asks for a candidate match score against a job.
type MatchScoreRequest struct {
	Skills []string
}

func (*MatchScoreRequest) capability() string { return "hr.match_score" }

// MatchScoreResponse carries the match score.
type MatchScoreResponse struct {
	Score int
}

func (*MatchScoreResponse) response() {}

// RelevanceRequest asks for a relevance score between a query and content.
type RelevanceRequest struct {
	Query   string
	Content string
}

func (*RelevanceRequest) capability() string { return "hr.relevance" }

// RelevanceResponse carries the relevance score.
type RelevanceResponse struct {
	Score float64
}

func (*RelevanceResponse) response() {}

// KnowledgeDoc is a minimal knowledge-base document input.
type KnowledgeDoc struct {
	Title   string
	Content string
}

// KnowledgeAnswerRequest asks for a synthesized answer from documents.
type KnowledgeAnswerRequest struct {
	Query string
	Docs  []KnowledgeDoc
}

func (*KnowledgeAnswerRequest) capability() string { return "hr.knowledge_answer" }

// KnowledgeAnswerResponse carries the synthesized answer.
type KnowledgeAnswerResponse struct {
	Answer string
}

func (*KnowledgeAnswerResponse) response() {}

// ShiftPredictionRequest asks for a staffing prediction.
type ShiftPredictionRequest struct {
	Date          time.Time
	BaseHeadcount int
	Department    *string
}

func (*ShiftPredictionRequest) capability() string { return "hr.shift_prediction" }

// ShiftPredictionResponse carries the predicted headcount, confidence, and reasoning.
type ShiftPredictionResponse struct {
	Predicted  int
	Confidence float64
	Reasoning  string
}

func (*ShiftPredictionResponse) response() {}

// ── Platform ───────────────────────────────────────────────────────────

// TextToSQLRequest asks for SQL generation from a prompt.
type TextToSQLRequest struct {
	Prompt string
}

func (*TextToSQLRequest) capability() string { return "platform.text_to_sql" }

// TextToSQLResponse carries the generated SQL and target table.
type TextToSQLResponse struct {
	SQL       string
	TableName string
}

func (*TextToSQLResponse) response() {}

// QueryResultsRequest asks for mock query result rows.
type QueryResultsRequest struct {
	Prompt    string
	TableName string
}

func (*QueryResultsRequest) capability() string { return "platform.query_results" }

// QueryResultsResponse carries the result rows.
type QueryResultsResponse struct {
	Data []map[string]any
}

func (*QueryResultsResponse) response() {}

// AnomalyRequest asks for security anomaly classification.
type AnomalyRequest struct {
	Action    string
	IPAddress *string
}

func (*AnomalyRequest) capability() string { return "platform.anomaly" }

// AnomalyResponse carries the anomaly type and risk score.
type AnomalyResponse struct {
	Type      string
	RiskScore float64
}

func (*AnomalyResponse) response() {}
