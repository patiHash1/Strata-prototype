package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type ChartOfAccount struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	AccountCode string    `json:"account_code"`
	AccountName string    `json:"account_name"`
	AccountType string    `json:"account_type"`
}

type JournalEntry struct {
	ID          uuid.UUID     `json:"id"`
	OrgID       uuid.UUID     `json:"org_id"`
	EntryNumber string        `json:"entry_number"`
	EntryDate   time.Time     `json:"entry_date"`
	Memo        *string       `json:"memo,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	Items       []JournalItem `json:"items,omitempty"`
}

type JournalItem struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	JournalEntryID uuid.UUID `json:"journal_entry_id"`
	AccountID      uuid.UUID `json:"account_id"`
	Debit          float64   `json:"debit"`
	Credit         float64   `json:"credit"`
}

type JournalItemInput struct {
	AccountID string  `json:"account_id"`
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
}

type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	OrgID          uuid.UUID  `json:"org_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	ContactID      *uuid.UUID `json:"contact_id,omitempty"`
	TotalAmount    float64    `json:"total_amount"`
	Status         string     `json:"status"`
	AIOCRProcessed bool       `json:"ai_ocr_processed"`
	DueDate        time.Time  `json:"due_date"`
	CreatedAt      time.Time  `json:"created_at"`
}

// OCRLineItem represents a line item extracted from an invoice via OCR.
type OCRLineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}

// OCRResult represents the output of AI-powered invoice OCR processing.
type OCRResult struct {
	VendorName    string        `json:"vendor_name"`
	InvoiceNumber string        `json:"invoice_number"`
	LineItems     []OCRLineItem `json:"line_items"`
	TaxAmount     float64       `json:"tax_amount"`
	TotalAmount   float64       `json:"total_amount"`
}

type Expense struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Amount       float64    `json:"amount"`
	Category     string     `json:"category"`
	ReceiptURL   *string    `json:"receipt_url,omitempty"`
	AIFraudFlag  bool       `json:"ai_fraud_flag"`
	AIAuditNotes *string    `json:"ai_audit_notes,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
}

type FixedAsset struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	AssetName       string    `json:"asset_name"`
	PurchaseDate    time.Time `json:"purchase_date"`
	PurchaseCost    float64   `json:"purchase_cost"`
	SalvageValue    float64   `json:"salvage_value"`
	UsefulLifeYears int       `json:"useful_life_years"`
	CreatedAt       time.Time `json:"created_at"`
}

// DepreciationResult holds the output of a depreciation calculation.
type DepreciationResult struct {
	AssetID                 uuid.UUID `json:"asset_id"`
	AnnualDepreciation      float64   `json:"annual_depreciation"`
	AccumulatedDepreciation float64   `json:"accumulated_depreciation"`
	CurrentBookValue        float64   `json:"current_book_value"`
}

type TaxRate struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	CountryCode string    `json:"country_code"`
	TaxName     string    `json:"tax_name"`
	TaxRate     float64   `json:"tax_rate"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type TaxCalculation struct {
	Subtotal     float64   `json:"subtotal"`
	TaxAmount    float64   `json:"tax_amount"`
	TotalAmount  float64   `json:"total_amount"`
	AppliedRates []TaxRate `json:"applied_rates"`
}

// ---- Repository ----

type accountingRepository struct {
	pool *pgxpool.Pool
}

func newAccountingRepository(pool *pgxpool.Pool) *accountingRepository {
	return &accountingRepository{pool: pool}
}

func (r *accountingRepository) CreateJournalEntry(ctx context.Context, e *JournalEntry) error {
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO journal_entries (id, org_id, entry_number, entry_date, memo, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, e.ID, e.OrgID, e.EntryNumber, e.EntryDate, e.Memo, e.CreatedAt)
	return err
}

func (r *accountingRepository) CreateJournalItem(ctx context.Context, item *JournalItem) error {
	item.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO journal_items (id, org_id, journal_entry_id, account_id, debit, credit)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.OrgID, item.JournalEntryID, item.AccountID, item.Debit, item.Credit)
	return err
}

func (r *accountingRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*ChartOfAccount, error) {
	a := &ChartOfAccount{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, account_code, account_name, account_type
		FROM chart_of_accounts WHERE id = $1
	`, id).Scan(&a.ID, &a.OrgID, &a.AccountCode, &a.AccountName, &a.AccountType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *accountingRepository) CreateInvoice(ctx context.Context, inv *Invoice) error {
	inv.ID = uuid.New()
	inv.CreatedAt = time.Now()
	if inv.Status == "" {
		inv.Status = "draft"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO invoices (id, org_id, invoice_number, contact_id, total_amount, status, ai_ocr_processed, due_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, inv.ID, inv.OrgID, inv.InvoiceNumber, inv.ContactID, inv.TotalAmount, inv.Status, inv.AIOCRProcessed, inv.DueDate, inv.CreatedAt)
	return err
}

func (r *accountingRepository) CreateExpense(ctx context.Context, exp *Expense) error {
	exp.ID = uuid.New()
	exp.CreatedAt = time.Now()
	if exp.Status == "" {
		exp.Status = "pending"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO expenses (id, org_id, user_id, amount, category, receipt_url, ai_fraud_flag, ai_audit_notes, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, exp.ID, exp.OrgID, exp.UserID, exp.Amount, exp.Category, exp.ReceiptURL, exp.AIFraudFlag, exp.AIAuditNotes, exp.Status, exp.CreatedAt)
	return err
}

func (r *accountingRepository) CreateAsset(ctx context.Context, a *FixedAsset) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fixed_assets (id, org_id, asset_name, purchase_date, purchase_cost, salvage_value, useful_life_years, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.ID, a.OrgID, a.AssetName, a.PurchaseDate, a.PurchaseCost, a.SalvageValue, a.UsefulLifeYears, a.CreatedAt)
	return err
}

func (r *accountingRepository) GetAssetByID(ctx context.Context, id uuid.UUID) (*FixedAsset, error) {
	a := &FixedAsset{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, asset_name, purchase_date, purchase_cost, salvage_value, useful_life_years, created_at
		FROM fixed_assets WHERE id = $1
	`, id).Scan(&a.ID, &a.OrgID, &a.AssetName, &a.PurchaseDate, &a.PurchaseCost, &a.SalvageValue, &a.UsefulLifeYears, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *accountingRepository) CreateTaxRate(ctx context.Context, t *TaxRate) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tax_rates (id, org_id, country_code, tax_name, tax_rate, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, t.ID, t.OrgID, t.CountryCode, t.TaxName, t.TaxRate, t.IsActive, t.CreatedAt)
	return err
}

func (r *accountingRepository) GetTaxRatesByCountry(ctx context.Context, orgID uuid.UUID, countryCode string) ([]TaxRate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, country_code, tax_name, tax_rate, is_active, created_at
		FROM tax_rates WHERE org_id = $1 AND country_code = $2 AND is_active = TRUE
	`, orgID, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []TaxRate
	for rows.Next() {
		var t TaxRate
		if err := rows.Scan(&t.ID, &t.OrgID, &t.CountryCode, &t.TaxName, &t.TaxRate, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, err
		}
		rates = append(rates, t)
	}
	if rates == nil {
		rates = []TaxRate{}
	}
	return rates, rows.Err()
}

// ---- Service ----

type AccountingService struct {
	repo *accountingRepository
}

func NewAccountingService(pool *pgxpool.Pool) *AccountingService {
	return &AccountingService{repo: newAccountingRepository(pool)}
}

// PostJournalEntry creates a journal entry with balanced debit/credit items.
func (s *AccountingService) PostJournalEntry(ctx context.Context, orgID uuid.UUID, entryDate time.Time, memo *string, items []JournalItemInput) (*JournalEntry, error) {
	if len(items) == 0 {
		return nil, ErrNoJournalItems
	}

	// Validate balance
	var totalDebit, totalCredit float64
	for _, item := range items {
		totalDebit += item.Debit
		totalCredit += item.Credit
	}
	if totalDebit != totalCredit {
		return nil, ErrUnbalancedEntry
	}

	// Validate all accounts exist and belong to the org
	for _, item := range items {
		accountID, err := uuid.Parse(item.AccountID)
		if err != nil {
			return nil, fmt.Errorf("invalid account_id: %s", item.AccountID)
		}
		account, err := s.repo.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, err
		}
		if account == nil {
			return nil, ErrAccountNotFound
		}
		if account.OrgID != orgID {
			return nil, ErrAccountNotInOrg
		}
	}

	// Generate entry number: JE-YYYYMMDD-XXXX
	entryNumber := fmt.Sprintf("JE-%s-%04d", entryDate.Format("20060102"), rand.Intn(10000))

	entry := &JournalEntry{
		OrgID:       orgID,
		EntryNumber: entryNumber,
		EntryDate:   entryDate,
		Memo:        memo,
	}
	if err := s.repo.CreateJournalEntry(ctx, entry); err != nil {
		return nil, err
	}

	// Create journal items
	for _, item := range items {
		accountID, _ := uuid.Parse(item.AccountID)
		ji := &JournalItem{
			OrgID:          orgID,
			JournalEntryID: entry.ID,
			AccountID:      accountID,
			Debit:          item.Debit,
			Credit:         item.Credit,
		}
		if err := s.repo.CreateJournalItem(ctx, ji); err != nil {
			return nil, err
		}
		entry.Items = append(entry.Items, *ji)
	}

	return entry, nil
}

// ProcessInvoiceOCR simulates AI-powered OCR extraction from an uploaded invoice file.
func (s *AccountingService) ProcessInvoiceOCR(ctx context.Context, orgID uuid.UUID, fileName string, fileSize int64) (*OCRResult, error) {
	// Simulate OCR processing — in production this would call a vision AI service
	result := aiSimulateOCR(fileName, fileSize)

	// Create an invoice record from the OCR result
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	inv := &Invoice{
		OrgID:          orgID,
		InvoiceNumber:  result.InvoiceNumber,
		TotalAmount:    result.TotalAmount,
		Status:         "draft",
		AIOCRProcessed: true,
		DueDate:        dueDate,
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}

	return result, nil
}

// SubmitExpense creates an expense with AI fraud audit.
func (s *AccountingService) SubmitExpense(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, amount float64, category string, receiptFileName string) (*Expense, error) {
	// Simulate AI fraud audit
	fraudFlag, auditNotes := aiAuditExpense(amount, category, receiptFileName)

	receiptURL := fmt.Sprintf("https://storage.strata.dev/receipts/%s/%s", orgID.String(), receiptFileName)

	exp := &Expense{
		OrgID:        orgID,
		UserID:       &userID,
		Amount:       amount,
		Category:     category,
		ReceiptURL:   &receiptURL,
		AIFraudFlag:  fraudFlag,
		AIAuditNotes: &auditNotes,
		Status:       "pending",
	}
	if err := s.repo.CreateExpense(ctx, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

// RegisterAsset creates a new fixed asset record.
func (s *AccountingService) RegisterAsset(ctx context.Context, orgID uuid.UUID, assetName string, purchaseDate time.Time, purchaseCost, salvageValue float64, usefulLifeYears int) (*FixedAsset, error) {
	asset := &FixedAsset{
		OrgID:           orgID,
		AssetName:       assetName,
		PurchaseDate:    purchaseDate,
		PurchaseCost:    purchaseCost,
		SalvageValue:    salvageValue,
		UsefulLifeYears: usefulLifeYears,
	}
	if err := s.repo.CreateAsset(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}

// CalculateDepreciation computes straight-line depreciation for a fixed asset over a given period.
func (s *AccountingService) CalculateDepreciation(ctx context.Context, assetID uuid.UUID, fromDate, toDate time.Time) (*DepreciationResult, error) {
	asset, err := s.repo.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}

	// Straight-line annual depreciation
	depreciableBase := asset.PurchaseCost - asset.SalvageValue
	annualDepreciation := depreciableBase / float64(asset.UsefulLifeYears)

	// Calculate accumulated depreciation up to toDate
	yearsElapsed := toDate.Sub(asset.PurchaseDate).Hours() / (365.25 * 24)
	if yearsElapsed < 0 {
		yearsElapsed = 0
	}
	if yearsElapsed > float64(asset.UsefulLifeYears) {
		yearsElapsed = float64(asset.UsefulLifeYears)
	}
	accumulatedDepreciation := annualDepreciation * yearsElapsed

	currentBookValue := asset.PurchaseCost - accumulatedDepreciation
	if currentBookValue < asset.SalvageValue {
		currentBookValue = asset.SalvageValue
	}

	return &DepreciationResult{
		AssetID:                 assetID,
		AnnualDepreciation:      annualDepreciation,
		AccumulatedDepreciation: accumulatedDepreciation,
		CurrentBookValue:        currentBookValue,
	}, nil
}

// CreateTaxRate creates a new tax rate for an organization.
func (s *AccountingService) CreateTaxRate(ctx context.Context, orgID uuid.UUID, countryCode, taxName string, taxRate float64) (*TaxRate, error) {
	tr := &TaxRate{
		OrgID:       orgID,
		CountryCode: countryCode,
		TaxName:     taxName,
		TaxRate:     taxRate,
		IsActive:    true,
	}
	if err := s.repo.CreateTaxRate(ctx, tr); err != nil {
		return nil, err
	}
	return tr, nil
}

// CalculateTax looks up active tax rates for a country and computes tax on a subtotal.
func (s *AccountingService) CalculateTax(ctx context.Context, orgID uuid.UUID, countryCode string, subtotal float64) (*TaxCalculation, error) {
	rates, err := s.repo.GetTaxRatesByCountry(ctx, orgID, countryCode)
	if err != nil {
		return nil, err
	}

	var totalTax float64
	for _, r := range rates {
		totalTax += subtotal * r.TaxRate
	}

	return &TaxCalculation{
		Subtotal:     subtotal,
		TaxAmount:    totalTax,
		TotalAmount:  subtotal + totalTax,
		AppliedRates: rates,
	}, nil
}

// ---- AI Simulation Helpers ----

// aiSimulateOCR simulates AI-powered OCR extraction from an invoice file.
func aiSimulateOCR(fileName string, fileSize int64) *OCRResult {
	vendors := []string{"Acme Supplies Inc.", "Global Tech Partners", "Office Depot", "Cloud Services LLC", "Strategic Consulting Group"}
	vendorIdx := rand.Intn(len(vendors))

	invNum := fmt.Sprintf("INV-%04d", 1000+rand.Intn(9000))

	numItems := 1 + rand.Intn(4)
	var lineItems []OCRLineItem
	var subtotal float64

	descriptions := []string{"Professional Services", "Software License", "Hardware Equipment", "Cloud Storage", "Consulting Hours", "Office Supplies", "Training Materials"}
	for i := 0; i < numItems; i++ {
		qty := 1 + rand.Intn(10)
		price := 10.0 + rand.Float64()*490.0
		total := float64(qty) * price
		lineItems = append(lineItems, OCRLineItem{
			Description: descriptions[rand.Intn(len(descriptions))],
			Quantity:    qty,
			UnitPrice:   price,
			Total:       total,
		})
		subtotal += total
	}

	taxRate := 0.08 + rand.Float64()*0.12 // 8–20% tax
	taxAmount := subtotal * taxRate
	totalAmount := subtotal + taxAmount

	return &OCRResult{
		VendorName:    vendors[vendorIdx],
		InvoiceNumber: invNum,
		LineItems:     lineItems,
		TaxAmount:     taxAmount,
		TotalAmount:   totalAmount,
	}
}

// aiAuditExpense simulates AI fraud detection on an expense submission.
func aiAuditExpense(amount float64, category string, receiptFileName string) (bool, string) {
	var flags []string

	// High-value expense flag
	if amount > 5000 {
		flags = append(flags, "High-value expense (over $5,000) requires manager approval")
	}

	// Suspicious category patterns
	lowerCat := strings.ToLower(category)
	if lowerCat == "entertainment" && amount > 1000 {
		flags = append(flags, "Entertainment expense exceeds $1,000 threshold — policy limit is $1,000")
	}
	if lowerCat == "travel" && amount > 3000 {
		flags = append(flags, "Travel expense exceeds $3,000 — please attach itinerary")
	}

	// Duplicate receipt detection (simulated)
	if strings.Contains(receiptFileName, "dup") || strings.Contains(receiptFileName, "copy") {
		flags = append(flags, "Potential duplicate receipt detected — filename contains 'dup' or 'copy'")
	}

	// Weekend submission flag
	now := time.Now()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		flags = append(flags, "Expense submitted on weekend — unusual timing")
	}

	if len(flags) > 0 {
		return true, strings.Join(flags, "; ")
	}

	return false, "No policy violations detected. Expense appears compliant."
}

// Domain errors
var (
	ErrNoJournalItems  = errors.New("journal entry must have at least one item")
	ErrUnbalancedEntry = errors.New("total debits must equal total credits")
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountNotInOrg = errors.New("account does not belong to this organization")
	ErrAssetNotFound   = errors.New("fixed asset not found")
)

// ── Module 2.1: Bank Reconciliation ──

// BankStatement represents an imported bank statement.
type BankStatement struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	BankName       string    `json:"bank_name"`
	AccountNumber  string    `json:"account_number"`
	StatementDate  string    `json:"statement_date"`
	OpeningBalance float64   `json:"opening_balance"`
	ClosingBalance float64   `json:"closing_balance"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// BankTransaction is a single line from a bank statement.
type BankTransaction struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	StatementID     uuid.UUID `json:"statement_id"`
	TransactionDate string    `json:"transaction_date"`
	Description     string    `json:"description"`
	Reference       *string   `json:"reference,omitempty"`
	Debit           float64   `json:"debit"`
	Credit          float64   `json:"credit"`
	Amount          float64   `json:"amount"`
	IsMatched       bool      `json:"is_matched"`
	CreatedAt       time.Time `json:"created_at"`
}

// ReconciliationMatch links a bank transaction to a journal entry.
type ReconciliationMatch struct {
	ID                uuid.UUID `json:"id"`
	OrgID             uuid.UUID `json:"org_id"`
	BankTransactionID uuid.UUID `json:"bank_transaction_id"`
	JournalEntryID    uuid.UUID `json:"journal_entry_id"`
	MatchType         string    `json:"match_type"`
	MatchDate         time.Time `json:"match_date"`
}

// BankStatementInput is the payload for importing a bank statement.
type BankStatementInput struct {
	BankName       string                 `json:"bank_name"`
	AccountNumber  string                 `json:"account_number"`
	StatementDate  string                 `json:"statement_date"`
	OpeningBalance float64                `json:"opening_balance"`
	ClosingBalance float64                `json:"closing_balance"`
	Transactions   []BankTransactionInput `json:"transactions"`
}

// BankTransactionInput is a single transaction in a bank statement import.
type BankTransactionInput struct {
	TransactionDate string  `json:"transaction_date"`
	Description     string  `json:"description"`
	Reference       *string `json:"reference,omitempty"`
	Debit           float64 `json:"debit"`
	Credit          float64 `json:"credit"`
	Amount          float64 `json:"amount"`
}

// ReconciliationReport is the output of a reconciliation run.
type ReconciliationReport struct {
	StatementID       uuid.UUID             `json:"statement_id"`
	TotalMatched      int                   `json:"total_matched"`
	TotalUnmatched    int                   `json:"total_unmatched"`
	OutstandingDebit  float64               `json:"outstanding_debit"`
	OutstandingCredit float64               `json:"outstanding_credit"`
	ReconciledBalance float64               `json:"reconciled_balance"`
	Matches           []ReconciliationMatch `json:"matches"`
}

func (r *accountingRepository) CreateBankStatement(ctx context.Context, stmt *BankStatement) error {
	stmt.ID = uuid.New()
	stmt.CreatedAt = time.Now()
	if stmt.Status == "" {
		stmt.Status = "imported"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bank_statements (id, org_id, bank_name, account_number, statement_date, opening_balance, closing_balance, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, stmt.ID, stmt.OrgID, stmt.BankName, stmt.AccountNumber, stmt.StatementDate, stmt.OpeningBalance, stmt.ClosingBalance, stmt.Status, stmt.CreatedAt)
	return err
}

func (r *accountingRepository) CreateBankTransaction(ctx context.Context, txn *BankTransaction) error {
	txn.ID = uuid.New()
	txn.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bank_transactions (id, org_id, statement_id, transaction_date, description, reference, debit, credit, amount, is_matched, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, txn.ID, txn.OrgID, txn.StatementID, txn.TransactionDate, txn.Description, txn.Reference, txn.Debit, txn.Credit, txn.Amount, txn.IsMatched, txn.CreatedAt)
	return err
}

func (r *accountingRepository) GetUnmatchedBankTransactions(ctx context.Context, orgID, statementID uuid.UUID) ([]BankTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, statement_id, transaction_date, description, reference, debit, credit, amount, is_matched, created_at
		FROM bank_transactions WHERE org_id = $1 AND statement_id = $2 AND is_matched = FALSE
	`, orgID, statementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBankTransactions(rows)
}

func (r *accountingRepository) GetJournalEntriesByAmountRange(ctx context.Context, orgID uuid.UUID, minAmount, maxAmount float64) ([]JournalEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT je.id, je.org_id, je.entry_number, je.entry_date, je.memo, je.created_at
		FROM journal_entries je
		JOIN journal_items ji ON ji.journal_entry_id = je.id
		WHERE je.org_id = $1 AND (ji.debit BETWEEN $2 AND $3 OR ji.credit BETWEEN $2 AND $3)
	`, orgID, minAmount, maxAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.EntryNumber, &e.EntryDate, &e.Memo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []JournalEntry{}
	}
	return entries, rows.Err()
}

func scanBankTransactions(rows pgx.Rows) ([]BankTransaction, error) {
	var txns []BankTransaction
	for rows.Next() {
		var t BankTransaction
		if err := rows.Scan(&t.ID, &t.OrgID, &t.StatementID, &t.TransactionDate, &t.Description, &t.Reference, &t.Debit, &t.Credit, &t.Amount, &t.IsMatched, &t.CreatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	if txns == nil {
		txns = []BankTransaction{}
	}
	return txns, rows.Err()
}

func (r *accountingRepository) CreateReconciliationMatch(ctx context.Context, match *ReconciliationMatch) error {
	match.ID = uuid.New()
	match.MatchDate = time.Now()
	if match.MatchType == "" {
		match.MatchType = "auto"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reconciliation_matches (id, org_id, bank_transaction_id, journal_entry_id, match_type, match_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bank_transaction_id, journal_entry_id) DO NOTHING
	`, match.ID, match.OrgID, match.BankTransactionID, match.JournalEntryID, match.MatchType, match.MatchDate)
	return err
}

func (r *accountingRepository) MarkBankTransactionMatched(ctx context.Context, txnID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE bank_transactions SET is_matched = TRUE WHERE id = $1`, txnID)
	return err
}

func (r *accountingRepository) GetReconciliationMatchesByStatement(ctx context.Context, orgID, statementID uuid.UUID) ([]ReconciliationMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, bank_transaction_id, journal_entry_id, match_type, match_date
		FROM reconciliation_matches WHERE org_id = $1 AND bank_transaction_id IN (
			SELECT id FROM bank_transactions WHERE statement_id = $2
		)
	`, orgID, statementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []ReconciliationMatch
	for rows.Next() {
		var m ReconciliationMatch
		if err := rows.Scan(&m.ID, &m.OrgID, &m.BankTransactionID, &m.JournalEntryID, &m.MatchType, &m.MatchDate); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	if matches == nil {
		matches = []ReconciliationMatch{}
	}
	return matches, rows.Err()
}

// ImportBankStatement imports a bank statement with its transaction lines.
func (s *AccountingService) ImportBankStatement(ctx context.Context, orgID uuid.UUID, input BankStatementInput) (*BankStatement, error) {
	stmt := &BankStatement{
		OrgID:          orgID,
		BankName:       input.BankName,
		AccountNumber:  input.AccountNumber,
		StatementDate:  input.StatementDate,
		OpeningBalance: input.OpeningBalance,
		ClosingBalance: input.ClosingBalance,
	}
	if err := s.repo.CreateBankStatement(ctx, stmt); err != nil {
		return nil, err
	}

	for _, t := range input.Transactions {
		txn := &BankTransaction{
			OrgID:           orgID,
			StatementID:     stmt.ID,
			TransactionDate: t.TransactionDate,
			Description:     t.Description,
			Reference:       t.Reference,
			Debit:           t.Debit,
			Credit:          t.Credit,
			Amount:          t.Amount,
		}
		if err := s.repo.CreateBankTransaction(ctx, txn); err != nil {
			return nil, err
		}
	}

	return stmt, nil
}

// ReconcileBankStatement auto-matches bank transactions to journal entries by amount proximity.
func (s *AccountingService) ReconcileBankStatement(ctx context.Context, orgID, statementID uuid.UUID) (*ReconciliationReport, error) {
	txns, err := s.repo.GetUnmatchedBankTransactions(ctx, orgID, statementID)
	if err != nil {
		return nil, err
	}

	matchedCount := 0
	var matches []ReconciliationMatch
	var outstandingDebit, outstandingCredit float64

	for _, txn := range txns {
		tolerance := txn.Amount * 0.01
		entries, err := s.repo.GetJournalEntriesByAmountRange(ctx, orgID, txn.Amount-tolerance, txn.Amount+tolerance)
		if err != nil {
			continue
		}
		if len(entries) > 0 {
			match := &ReconciliationMatch{
				OrgID:             orgID,
				BankTransactionID: txn.ID,
				JournalEntryID:    entries[0].ID,
				MatchType:         "auto",
			}
			if err := s.repo.CreateReconciliationMatch(ctx, match); err != nil {
				continue
			}
			if err := s.repo.MarkBankTransactionMatched(ctx, txn.ID); err != nil {
				continue
			}
			matches = append(matches, *match)
			matchedCount++
		} else {
			outstandingDebit += txn.Debit
			outstandingCredit += txn.Credit
		}
	}

	allMatches, _ := s.repo.GetReconciliationMatchesByStatement(ctx, orgID, statementID)

	return &ReconciliationReport{
		StatementID:       statementID,
		TotalMatched:      matchedCount,
		TotalUnmatched:    len(txns) - matchedCount,
		OutstandingDebit:  outstandingDebit,
		OutstandingCredit: outstandingCredit,
		ReconciledBalance: outstandingCredit - outstandingDebit,
		Matches:           allMatches,
	}, nil
}

// ── Module 2.5: Multi-Currency Exchange Rates ──

// Currency represents a currency definition.
type Currency struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	DecimalPlaces int    `json:"decimal_places"`
	IsActive      bool   `json:"is_active"`
}

// ExchangeRate represents a conversion rate between two currencies.
type ExchangeRate struct {
	ID            uuid.UUID `json:"id"`
	OrgID         uuid.UUID `json:"org_id"`
	FromCurrency  string    `json:"from_currency"`
	ToCurrency    string    `json:"to_currency"`
	Rate          float64   `json:"rate"`
	EffectiveDate string    `json:"effective_date"`
	Source        string    `json:"source"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *accountingRepository) GetCurrencyByCode(ctx context.Context, code string) (*Currency, error) {
	c := &Currency{}
	err := r.pool.QueryRow(ctx, `
		SELECT code, name, symbol, decimal_places, is_active FROM currencies WHERE code = $1
	`, code).Scan(&c.Code, &c.Name, &c.Symbol, &c.DecimalPlaces, &c.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *accountingRepository) CreateExchangeRate(ctx context.Context, rate *ExchangeRate) error {
	rate.ID = uuid.New()
	rate.CreatedAt = time.Now()
	if rate.Source == "" {
		rate.Source = "manual"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO exchange_rates (id, org_id, from_currency, to_currency, rate, effective_date, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id, from_currency, to_currency, effective_date) DO UPDATE SET rate = $5, source = $7
	`, rate.ID, rate.OrgID, rate.FromCurrency, rate.ToCurrency, rate.Rate, rate.EffectiveDate, rate.Source, rate.CreatedAt)
	return err
}

func (r *accountingRepository) GetExchangeRate(ctx context.Context, orgID uuid.UUID, fromCurrency, toCurrency string) (*ExchangeRate, error) {
	rate := &ExchangeRate{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, from_currency, to_currency, rate, effective_date, source, created_at
		FROM exchange_rates
		WHERE org_id = $1 AND from_currency = $2 AND to_currency = $3
		ORDER BY effective_date DESC LIMIT 1
	`, orgID, fromCurrency, toCurrency).Scan(&rate.ID, &rate.OrgID, &rate.FromCurrency, &rate.ToCurrency, &rate.Rate, &rate.EffectiveDate, &rate.Source, &rate.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return rate, err
}

// SetExchangeRate creates or updates an exchange rate between two currencies.
func (s *AccountingService) SetExchangeRate(ctx context.Context, orgID uuid.UUID, fromCurrency, toCurrency string, rate float64, effectiveDate string) (*ExchangeRate, error) {
	er := &ExchangeRate{
		OrgID:         orgID,
		FromCurrency:  fromCurrency,
		ToCurrency:    toCurrency,
		Rate:          rate,
		EffectiveDate: effectiveDate,
	}
	if err := s.repo.CreateExchangeRate(ctx, er); err != nil {
		return nil, err
	}
	return er, nil
}

// ConvertAmount converts a monetary amount between currencies using the effective rate.
func (s *AccountingService) ConvertAmount(ctx context.Context, orgID uuid.UUID, amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}
	rate, err := s.repo.GetExchangeRate(ctx, orgID, fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	if rate == nil {
		return 0, fmt.Errorf("no exchange rate found for %s -> %s", fromCurrency, toCurrency)
	}
	return amount * rate.Rate, nil
}
