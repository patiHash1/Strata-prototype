package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/accounting/journal-entries ----

type journalItemRequest struct {
	AccountID string  `json:"account_id"`
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
}

type createJournalEntryRequest struct {
	EntryDate string               `json:"entry_date"`
	Memo      *string              `json:"memo,omitempty"`
	Items     []journalItemRequest `json:"items"`
}

// CreateJournalEntryResponse represents the payload returned when a journal entry is posted.
type CreateJournalEntryResponse struct {
	JournalEntryID string `json:"journal_entry_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntryNumber    string `json:"entry_number" example:"JE-20260102-0042"`
}

// createJournalEntryHandler posts a balanced general ledger entry.
//
//	@Summary		Post general ledger entry
//	@Description	Creates a journal entry with balanced debit/credit items. Total debits must equal total credits. Requires `accounting.ledger.write` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createJournalEntryRequest	true	"Journal entry payload"
//	@Success		201	{object}	CreateJournalEntryResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/journal-entries [post]
func (a *App) createJournalEntryHandler(w http.ResponseWriter, r *http.Request) {
	var req createJournalEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.EntryDate) {
		utils.WriteErr(w, http.StatusBadRequest, "entry_date is required")
		return
	}

	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid entry_date format, expected YYYY-MM-DD")
		return
	}

	if len(req.Items) == 0 {
		utils.WriteErr(w, http.StatusBadRequest, "at least one journal item is required")
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

	var items []services.JournalItemInput
	for _, item := range req.Items {
		if !utils.NotBlank(item.AccountID) {
			utils.WriteErr(w, http.StatusBadRequest, "each item must have an account_id")
			return
		}
		items = append(items, services.JournalItemInput{
			AccountID: item.AccountID,
			Debit:     item.Debit,
			Credit:    item.Credit,
		})
	}

	entry, err := a.Accounting.PostJournalEntry(r.Context(), orgID, entryDate, req.Memo, items)
	if err != nil {
		switch err {
		case services.ErrNoJournalItems:
			utils.WriteErr(w, http.StatusBadRequest, err.Error())
		case services.ErrUnbalancedEntry:
			utils.WriteErr(w, http.StatusBadRequest, err.Error())
		case services.ErrAccountNotFound:
			utils.WriteErr(w, http.StatusNotFound, err.Error())
		case services.ErrAccountNotInOrg:
			utils.WriteErr(w, http.StatusNotFound, err.Error())
		default:
			utils.WriteErr(w, http.StatusInternalServerError, "could not post journal entry")
		}
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"journal_entry_id": entry.ID.String(),
		"entry_number":     entry.EntryNumber,
	})
}

// ---- POST /api/v1/accounting/invoices/ocr ----

// OCRResponse represents the payload returned from invoice OCR processing.
type OCRResponse struct {
	VendorName    string                 `json:"vendor_name" example:"Acme Supplies Inc."`
	InvoiceNumber string                 `json:"invoice_number" example:"INV-4521"`
	LineItems     []services.OCRLineItem `json:"line_items"`
	TaxAmount     float64                `json:"tax_amount" example:"45.60"`
	TotalAmount   float64                `json:"total_amount" example:"570.00"`
}

// processInvoiceOCRHandler uploads an invoice file for AI vision OCR processing.
//
//	@Summary		Upload invoice for vision OCR
//	@Description	Uploads a PDF or image invoice file for AI-powered OCR extraction. Returns extracted vendor, line items, tax, and total. Requires `accounting.invoices.write` permission.
//	@Tags			Accounting
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Invoice file (PDF or image)"
//	@Success		200	{object}	OCRResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/invoices/ocr [post]
func (a *App) processInvoiceOCRHandler(w http.ResponseWriter, r *http.Request) {
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

	// Parse multipart form (max 10 MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "could not parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Read file bytes (for simulation — in production this would be sent to an OCR service)
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not read uploaded file")
		return
	}

	result, err := a.Accounting.ProcessInvoiceOCR(r.Context(), orgID, header.Filename, int64(len(fileBytes)))
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not process invoice OCR")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"vendor_name":    result.VendorName,
		"invoice_number": result.InvoiceNumber,
		"line_items":     result.LineItems,
		"tax_amount":     result.TaxAmount,
		"total_amount":   result.TotalAmount,
	})
}

// ---- POST /api/v1/accounting/expenses ----

type createExpenseRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	ReceiptFile *string `json:"receipt_file,omitempty"`
}

// CreateExpenseResponse represents the payload returned when an expense is submitted.
type CreateExpenseResponse struct {
	ExpenseID    string `json:"expense_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AIFraudFlag  bool   `json:"ai_fraud_flag" example:"false"`
	AIAuditNotes string `json:"ai_audit_notes" example:"No policy violations detected."`
}

// createExpenseHandler submits an expense with AI fraud audit.
//
//	@Summary		Submit expense with AI fraud audit
//	@Description	Creates an expense submission and runs AI fraud detection. Flags high-value, policy-violating, or suspicious expenses. Requires `expenses.submit` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createExpenseRequest	true	"Expense payload"
//	@Success		201	{object}	CreateExpenseResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/expenses [post]
func (a *App) createExpenseHandler(w http.ResponseWriter, r *http.Request) {
	var req createExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	if !utils.NotBlank(req.Category) {
		utils.WriteErr(w, http.StatusBadRequest, "category is required")
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

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid user in token")
		return
	}

	receiptFileName := "no-receipt.pdf"
	if req.ReceiptFile != nil && *req.ReceiptFile != "" {
		receiptFileName = *req.ReceiptFile
	}

	exp, err := a.Accounting.SubmitExpense(r.Context(), orgID, userID, req.Amount, req.Category, receiptFileName)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not submit expense")
		return
	}

	auditNotes := ""
	if exp.AIAuditNotes != nil {
		auditNotes = *exp.AIAuditNotes
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"expense_id":     exp.ID.String(),
		"ai_fraud_flag":  exp.AIFraudFlag,
		"ai_audit_notes": auditNotes,
	})
}

// ---- POST /api/v1/accounting/assets ----

type createAssetRequest struct {
	AssetName       string  `json:"asset_name"`
	PurchaseDate    string  `json:"purchase_date"`
	PurchaseCost    float64 `json:"purchase_cost"`
	SalvageValue    float64 `json:"salvage_value"`
	UsefulLifeYears int     `json:"useful_life_years"`
}

// createAssetHandler registers a new fixed asset.
//
//	@Summary		Register fixed asset
//	@Description	Registers a new fixed asset for depreciation tracking. Requires `accounting.assets.write` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createAssetRequest	true	"Asset payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/assets [post]
func (a *App) createAssetHandler(w http.ResponseWriter, r *http.Request) {
	var req createAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.AssetName) {
		utils.WriteErr(w, http.StatusBadRequest, "asset_name is required")
		return
	}
	if !utils.NotBlank(req.PurchaseDate) {
		utils.WriteErr(w, http.StatusBadRequest, "purchase_date is required")
		return
	}
	if req.PurchaseCost <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "purchase_cost must be greater than zero")
		return
	}
	if req.UsefulLifeYears <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "useful_life_years must be greater than zero")
		return
	}

	purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid purchase_date format, expected YYYY-MM-DD")
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

	asset, err := a.Accounting.RegisterAsset(r.Context(), orgID, req.AssetName, purchaseDate, req.PurchaseCost, req.SalvageValue, req.UsefulLifeYears)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not register asset")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"asset_id":          asset.ID.String(),
		"asset_name":        asset.AssetName,
		"purchase_cost":     asset.PurchaseCost,
		"salvage_value":     asset.SalvageValue,
		"useful_life_years": asset.UsefulLifeYears,
	})
}

// ---- GET /api/v1/accounting/assets/{asset_id}/depreciation ----

// getDepreciationHandler calculates straight-line depreciation for a fixed asset.
//
//	@Summary		Calculate asset depreciation
//	@Description	Calculates straight-line depreciation for a fixed asset over a given date range. Returns annual depreciation, accumulated depreciation, and current book value. Requires `accounting.assets.read` permission.
//	@Tags			Accounting
//	@Produce		json
//	@Security		BearerAuth
//	@Param			asset_id	path	string	true	"Asset ID"
//	@Param			from_date	query	string	true	"Start date (YYYY-MM-DD)"
//	@Param			to_date		query	string	true	"End date (YYYY-MM-DD)"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/accounting/assets/{asset_id}/depreciation [get]
func (a *App) getDepreciationHandler(w http.ResponseWriter, r *http.Request) {
	assetIDStr := r.PathValue("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid asset_id")
		return
	}

	fromDateStr := r.URL.Query().Get("from_date")
	toDateStr := r.URL.Query().Get("to_date")
	if !utils.NotBlank(fromDateStr) || !utils.NotBlank(toDateStr) {
		utils.WriteErr(w, http.StatusBadRequest, "from_date and to_date query parameters are required")
		return
	}

	fromDate, err := time.Parse("2006-01-02", fromDateStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid from_date format, expected YYYY-MM-DD")
		return
	}
	toDate, err := time.Parse("2006-01-02", toDateStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid to_date format, expected YYYY-MM-DD")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	result, err := a.Accounting.CalculateDepreciation(r.Context(), assetID, fromDate, toDate)
	if err != nil {
		if err == services.ErrAssetNotFound {
			utils.WriteErr(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not calculate depreciation")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"asset_id":                 result.AssetID.String(),
		"annual_depreciation":      result.AnnualDepreciation,
		"accumulated_depreciation": result.AccumulatedDepreciation,
		"current_book_value":       result.CurrentBookValue,
	})
}

// ---- POST /api/v1/accounting/tax-rates ----

type createTaxRateRequest struct {
	CountryCode string  `json:"country_code"`
	TaxName     string  `json:"tax_name"`
	TaxRate     float64 `json:"tax_rate"`
}

// createTaxRateHandler creates a new tax rate.
//
//	@Summary		Create tax rate
//	@Description	Creates a new tax rate for a specific country (e.g., VAT, GST). Requires `accounting.tax.manage` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createTaxRateRequest	true	"Tax rate payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/tax-rates [post]
func (a *App) createTaxRateHandler(w http.ResponseWriter, r *http.Request) {
	var req createTaxRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.CountryCode) {
		utils.WriteErr(w, http.StatusBadRequest, "country_code is required")
		return
	}
	if !utils.NotBlank(req.TaxName) {
		utils.WriteErr(w, http.StatusBadRequest, "tax_name is required")
		return
	}
	if req.TaxRate <= 0 || req.TaxRate > 1 {
		utils.WriteErr(w, http.StatusBadRequest, "tax_rate must be between 0 and 1")
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

	taxRate, err := a.Accounting.CreateTaxRate(r.Context(), orgID, req.CountryCode, req.TaxName, req.TaxRate)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create tax rate")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"tax_rate_id":  taxRate.ID.String(),
		"country_code": taxRate.CountryCode,
		"tax_name":     taxRate.TaxName,
		"tax_rate":     taxRate.TaxRate,
	})
}

// ---- POST /api/v1/accounting/tax/calculate ----

type calculateTaxRequest struct {
	CountryCode string  `json:"country_code"`
	Subtotal    float64 `json:"subtotal"`
}

// calculateTaxHandler computes tax for a given country and subtotal.
//
//	@Summary		Calculate tax
//	@Description	Computes tax for a given country code and subtotal amount using the organization's active tax rates. Requires `accounting.tax.read` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	calculateTaxRequest	true	"Tax calculation payload"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/tax/calculate [post]
func (a *App) calculateTaxHandler(w http.ResponseWriter, r *http.Request) {
	var req calculateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.CountryCode) {
		utils.WriteErr(w, http.StatusBadRequest, "country_code is required")
		return
	}
	if req.Subtotal <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "subtotal must be greater than zero")
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

	calc, err := a.Accounting.CalculateTax(r.Context(), orgID, req.CountryCode, req.Subtotal)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not calculate tax")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"subtotal":      calc.Subtotal,
		"tax_amount":    calc.TaxAmount,
		"total_amount":  calc.TotalAmount,
		"applied_rates": calc.AppliedRates,
	})
}
