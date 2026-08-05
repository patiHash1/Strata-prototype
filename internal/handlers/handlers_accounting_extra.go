package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/accounting/bank-statements ----

type importBankStatementRequest struct {
	BankName       string                          `json:"bank_name"`
	AccountNumber  string                          `json:"account_number"`
	StatementDate  string                          `json:"statement_date"`
	OpeningBalance float64                         `json:"opening_balance"`
	ClosingBalance float64                         `json:"closing_balance"`
	Transactions   []services.BankTransactionInput `json:"transactions"`
}

// importBankStatementHandler imports a bank statement with its transactions.
//
//	@Summary		Import bank statement
//	@Description	Imports a bank statement and its transaction lines for reconciliation. Requires `accounting.bankrec.write` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	importBankStatementRequest	true	"Bank statement payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/bank-statements [post]
func (a *App) importBankStatementHandler(w http.ResponseWriter, r *http.Request) {
	var req importBankStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.BankName) {
		utils.WriteErr(w, http.StatusBadRequest, "bank_name is required")
		return
	}
	if !utils.NotBlank(req.AccountNumber) {
		utils.WriteErr(w, http.StatusBadRequest, "account_number is required")
		return
	}
	if !utils.NotBlank(req.StatementDate) {
		utils.WriteErr(w, http.StatusBadRequest, "statement_date is required")
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

	input := services.BankStatementInput{
		BankName:       req.BankName,
		AccountNumber:  req.AccountNumber,
		StatementDate:  req.StatementDate,
		OpeningBalance: req.OpeningBalance,
		ClosingBalance: req.ClosingBalance,
		Transactions:   req.Transactions,
	}

	stmt, err := a.Accounting.ImportBankStatement(r.Context(), orgID, input)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not import bank statement")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"statement_id": stmt.ID.String(),
		"status":       stmt.Status,
	})
}

// ---- POST /api/v1/accounting/bank-statements/{statement_id}/reconcile ----

// reconcileBankStatementHandler auto-reconciles a bank statement against journal entries.
//
//	@Summary		Reconcile bank statement
//	@Description	Auto-matches bank transactions from a statement to journal entries by amount proximity. Requires `accounting.bankrec.write` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			statement_id	path	string	true	"Bank statement ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/bank-statements/{statement_id}/reconcile [post]
func (a *App) reconcileBankStatementHandler(w http.ResponseWriter, r *http.Request) {
	statementIDStr := r.PathValue("statement_id")
	statementID, err := uuid.Parse(statementIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid statement_id")
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

	report, err := a.Accounting.ReconcileBankStatement(r.Context(), orgID, statementID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not reconcile bank statement")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"statement_id":       report.StatementID.String(),
		"total_matched":      report.TotalMatched,
		"total_unmatched":    report.TotalUnmatched,
		"outstanding_debit":  report.OutstandingDebit,
		"outstanding_credit": report.OutstandingCredit,
		"reconciled_balance": report.ReconciledBalance,
		"matches":            report.Matches,
	})
}

// ---- POST /api/v1/accounting/exchange-rates ----

type createExchangeRateRequest struct {
	FromCurrency  string  `json:"from_currency"`
	ToCurrency    string  `json:"to_currency"`
	Rate          float64 `json:"rate"`
	EffectiveDate string  `json:"effective_date"`
}

// createExchangeRateHandler creates or updates an exchange rate between two currencies.
//
//	@Summary		Create exchange rate
//	@Description	Creates or updates an exchange rate between two currencies for multi-currency accounting. Requires `accounting.exchangerates.write` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createExchangeRateRequest	true	"Exchange rate payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/exchange-rates [post]
func (a *App) createExchangeRateHandler(w http.ResponseWriter, r *http.Request) {
	var req createExchangeRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.FromCurrency) {
		utils.WriteErr(w, http.StatusBadRequest, "from_currency is required")
		return
	}
	if !utils.NotBlank(req.ToCurrency) {
		utils.WriteErr(w, http.StatusBadRequest, "to_currency is required")
		return
	}
	if req.Rate <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "rate must be positive")
		return
	}
	if !utils.NotBlank(req.EffectiveDate) {
		utils.WriteErr(w, http.StatusBadRequest, "effective_date is required")
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

	er, err := a.Accounting.SetExchangeRate(r.Context(), orgID, req.FromCurrency, req.ToCurrency, req.Rate, req.EffectiveDate)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create exchange rate")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"exchange_rate_id": er.ID.String(),
		"from_currency":    er.FromCurrency,
		"to_currency":      er.ToCurrency,
		"rate":             er.Rate,
		"effective_date":   er.EffectiveDate,
	})
}

// ---- POST /api/v1/accounting/convert ----

type convertCurrencyRequest struct {
	Amount       float64 `json:"amount"`
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
}

// convertCurrencyHandler converts a monetary amount between currencies.
//
//	@Summary		Convert currency
//	@Description	Converts a monetary amount between two currencies using the effective exchange rate. Requires `accounting.currencyconvert.read` permission.
//	@Tags			Accounting
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	convertCurrencyRequest	true	"Currency conversion payload"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/accounting/convert [post]
func (a *App) convertCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	var req convertCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		utils.WriteErr(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if !utils.NotBlank(req.FromCurrency) {
		utils.WriteErr(w, http.StatusBadRequest, "from_currency is required")
		return
	}
	if !utils.NotBlank(req.ToCurrency) {
		utils.WriteErr(w, http.StatusBadRequest, "to_currency is required")
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

	convertedAmount, err := a.Accounting.ConvertAmount(r.Context(), orgID, req.Amount, req.FromCurrency, req.ToCurrency)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not convert currency")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"original_amount":  req.Amount,
		"from_currency":    req.FromCurrency,
		"to_currency":      req.ToCurrency,
		"converted_amount": convertedAmount,
	})
}
