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
