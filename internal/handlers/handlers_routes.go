package handlers

import (
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// routes registers all application routes on the provided ServeMux.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// ── Public routes ──
	mux.HandleFunc("GET /health", a.healthHandler)
	mux.HandleFunc("POST /api/v1/auth/register", a.registerHandler)
	mux.HandleFunc("POST /api/v1/auth/login", a.loginHandler)

	// ── Protected routes (JWT + permission gates) ──
	mux.Handle("POST /api/v1/org/invitations",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersInvite)(
				http.HandlerFunc(a.inviteHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/roles",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermRBACManage)(
				http.HandlerFunc(a.createRoleHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/api-keys",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAPIKeysManage)(
				http.HandlerFunc(a.createAPIKeyHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/billing/subscriptions",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermBillingManage)(
				http.HandlerFunc(a.createSubscriptionHandler),
			),
		),
	)

	// ── Organization member management ──
	mux.Handle("PATCH /api/v1/org/members/{member_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.updateMemberHandler),
			),
		),
	)

	mux.Handle("DELETE /api/v1/org/members/{member_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.deleteMemberHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/org/members/{member_id}/remove",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermUsersManage)(
				http.HandlerFunc(a.removeMemberHandler),
			),
		),
	)

	// ── CRM ──
	mux.Handle("POST /api/v1/crm/leads",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMLeadsWrite)(
				http.HandlerFunc(a.createLeadHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/quotes/risk-analysis",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMQuotesWrite)(
				http.HandlerFunc(a.analyzeRiskHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/tickets",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMTicketsWrite)(
				http.HandlerFunc(a.createTicketHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/field-visits",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMFieldVisitsWrite)(
				http.HandlerFunc(a.scheduleFieldVisitHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/campaigns",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMCampaignsWrite)(
				http.HandlerFunc(a.createCampaignHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/crm/campaigns/{campaign_id}/launch",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCRMCampaignsWrite)(
				http.HandlerFunc(a.launchCampaignHandler),
			),
		),
	)

	// ── Accounting ──
	mux.Handle("POST /api/v1/accounting/journal-entries",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingLedger)(
				http.HandlerFunc(a.createJournalEntryHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/invoices/ocr",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingInvoices)(
				http.HandlerFunc(a.processInvoiceOCRHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/expenses",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermExpensesSubmit)(
				http.HandlerFunc(a.createExpenseHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/assets",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingAssetsWrite)(
				http.HandlerFunc(a.createAssetHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/accounting/assets/{asset_id}/depreciation",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingAssetsRead)(
				http.HandlerFunc(a.getDepreciationHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/tax-rates",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingTaxManage)(
				http.HandlerFunc(a.createTaxRateHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/tax/calculate",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingTaxRead)(
				http.HandlerFunc(a.calculateTaxHandler),
			),
		),
	)

	// Accounting - Bank Reconciliation
	mux.Handle("POST /api/v1/accounting/bank-statements",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingBankRec)(
				http.HandlerFunc(a.importBankStatementHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/bank-statements/{statement_id}/reconcile",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingBankRec)(
				http.HandlerFunc(a.reconcileBankStatementHandler),
			),
		),
	)

	// Accounting - Multi-Currency
	mux.Handle("POST /api/v1/accounting/exchange-rates",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingExchangeRates)(
				http.HandlerFunc(a.createExchangeRateHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/accounting/convert",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermAccountingCurrencyConvert)(
				http.HandlerFunc(a.convertCurrencyHandler),
			),
		),
	)

	// ── Fleet & Supply Chain ──
	// Telemetry ingestion uses API key auth
	mux.Handle("POST /api/v1/fleet/telematics/ingest",
		utils.RequireAPIKey(a.SupplyChain, services.PermFleetTelematicsIngest)(
			http.HandlerFunc(a.ingestTelemetryHandler),
		),
	)

	mux.Handle("POST /api/v1/fleet/routes/optimize",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermFleetRoutesManage)(
				http.HandlerFunc(a.optimizeRoutesHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/inventory/reorder-predictions",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermInventoryRead)(
				http.HandlerFunc(a.getReorderPredictionsHandler),
			),
		),
	)

	// Supply Chain - Inventory
	mux.Handle("POST /api/v1/inventory/receive",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermInventoryReceive)(
				http.HandlerFunc(a.receiveStockHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/inventory/issue",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermInventoryIssue)(
				http.HandlerFunc(a.issueStockHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/inventory/transfer",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermInventoryTransfer)(
				http.HandlerFunc(a.transferStockHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/inventory/snapshot",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermInventorySnapshot)(
				http.HandlerFunc(a.getInventorySnapshotHandler),
			),
		),
	)

	// ── Manufacturing ──
	mux.Handle("POST /api/v1/manufacturing/boms",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermManufacturingBOMsWrite)(
				http.HandlerFunc(a.createBOMHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/manufacturing/work-orders",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermManufacturingWorkOrdersWrite)(
				http.HandlerFunc(a.createWorkOrderHandler),
			),
		),
	)

	// ── Procurement ──
	mux.Handle("POST /api/v1/procurement/purchase-orders",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermProcurementPOWrite)(
				http.HandlerFunc(a.createPurchaseOrderHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/procurement/supplier-risk",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermProcurementSupplierRead)(
				http.HandlerFunc(a.getSupplierRiskHandler),
			),
		),
	)

	// ── HR, Workforce & Collaboration ──
	mux.Handle("POST /api/v1/hr/attendance/clock-in",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAttendanceWrite)(
				http.HandlerFunc(a.clockInHandler),
			),
		),
	)

	// HR - Shift Management
	mux.Handle("POST /api/v1/hr/shifts/templates",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAIShiftsWrite)(
				http.HandlerFunc(a.createShiftTemplateHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/hr/shifts/assignments",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAIShiftsWrite)(
				http.HandlerFunc(a.assignShiftHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/shifts/predictions",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAIShiftsWrite)(
				http.HandlerFunc(a.predictShiftNeedsHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/shifts/schedule",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAIShiftsWrite)(
				http.HandlerFunc(a.getEmployeeScheduleHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/hr/attendance/clock-out",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRAIClockOut)(
				http.HandlerFunc(a.clockOutHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/hr/ats/parse-resume",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRRecruitmentWrite)(
				http.HandlerFunc(a.parseResumeHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/hr/knowledge/search",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermKnowledgeRead)(
				http.HandlerFunc(a.knowledgeSearchHandler),
			),
		),
	)

	// ── HR Employees ──
	mux.Handle("POST /api/v1/hr/employees",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHREmployeesWrite)(
				http.HandlerFunc(a.createEmployeeHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/employees",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHREmployeesRead)(
				http.HandlerFunc(a.listEmployeesHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/employees/{employee_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHREmployeesRead)(
				http.HandlerFunc(a.getEmployeeHandler),
			),
		),
	)

	mux.Handle("PATCH /api/v1/hr/employees/{employee_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHREmployeesWrite)(
				http.HandlerFunc(a.updateEmployeeHandler),
			),
		),
	)

	// ── HR Payroll ──
	mux.Handle("POST /api/v1/hr/payroll/runs",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRPayrollWrite)(
				http.HandlerFunc(a.runPayrollHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/payroll/runs",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRPayrollRead)(
				http.HandlerFunc(a.listPayrollRunsHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/hr/payroll/runs/{run_id}",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRPayrollRead)(
				http.HandlerFunc(a.getPayrollRunHandler),
			),
		),
	)

	// HR - Payroll Tax
	mux.Handle("GET /api/v1/hr/payroll/runs/{run_id}/detail",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRPayrollRead)(
				http.HandlerFunc(a.getPayrollRunDetailHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/hr/payroll/tax-profiles",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermHRPayrollWrite)(
				http.HandlerFunc(a.setEmployeeTaxProfileHandler),
			),
		),
	)

	// ── BI & Executive Dashboards ──
	mux.Handle("POST /api/v1/bi/dashboards",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermBIDashboardsWrite)(
				http.HandlerFunc(a.createDashboardHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/bi/dashboards",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermBIDashboardsRead)(
				http.HandlerFunc(a.listDashboardsHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/bi/dashboards/{dashboard_id}/data",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermBIDashboardsRead)(
				http.HandlerFunc(a.getDashboardDataHandler),
			),
		),
	)

	// ── IoT Gateway ──
	mux.Handle("POST /api/v1/iot/devices",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermIoTDevicesWrite)(
				http.HandlerFunc(a.registerDeviceHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/iot/devices",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermIoTDevicesWrite)(
				http.HandlerFunc(a.listDevicesHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/iot/readings",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermIoTReadingsIngest)(
				http.HandlerFunc(a.ingestReadingHandler),
			),
		),
	)

	// Platform - IoT Batch
	mux.Handle("POST /api/v1/iot/readings/batch",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermIoTReadingsIngest)(
				http.HandlerFunc(a.ingestReadingBatchHandler),
			),
		),
	)

	// ── AI & Platform ──
	mux.Handle("POST /api/v1/ai/copilot/query",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermCopilotUse)(
				http.HandlerFunc(a.copilotQueryHandler),
			),
		),
	)

	mux.Handle("POST /api/v1/workflows/trigger",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermWorkflowsExecute)(
				http.HandlerFunc(a.triggerWorkflowHandler),
			),
		),
	)

	mux.Handle("GET /api/v1/security/audit-anomalies",
		utils.RequireAuth(a.Auth)(
			utils.RequirePermission(services.PermSecurityAuditRead)(
				http.HandlerFunc(a.auditAnomaliesHandler),
			),
		),
	)

	// ── Swagger UI (development only) ──
	if a.Config.EnableSwagger {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	// ── Global middleware stack (outermost first) ──
	var handler http.Handler = mux
	handler = utils.CORSMiddleware(handler)
	handler = utils.LoggingMiddleware(handler)
	handler = utils.RecoveryMiddleware(handler)

	return handler
}
