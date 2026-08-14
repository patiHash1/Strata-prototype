package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/seed.go <seed|wipe>")
		os.Exit(1)
	}

	dsn := envOr("DATABASE_URL", "postgres://strata-user:strata-pass@localhost:5432/strata-db-beta?sslmode=disable")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	switch os.Args[1] {
	case "seed":
		if err := seed(ctx, pool); err != nil {
			log.Fatalf("seed failed: %v", err)
		}
		fmt.Println("✅ Seed complete — 20 records inserted into each table.")
	case "wipe":
		if err := wipe(ctx, pool); err != nil {
			log.Fatalf("wipe failed: %v", err)
		}
		fmt.Println("✅ Wipe complete — all dummy data removed.")
	default:
		fmt.Printf("Unknown command: %s\nUsage: go run scripts/seed.go <seed|wipe>\n", os.Args[1])
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── Wipe ───────────────────────────────────────────────────────────

func wipe(ctx context.Context, pool *pgxpool.Pool) error {
	// Tables with direct org_id (child → parent order).
	orgTables := []string{
		"crm_contacts",
		"crm_deals",
		"crm_quotes",
		"crm_helpdesk_tickets",
		"crm_campaigns",
		"invoices",
		"expenses",
		"chart_of_accounts",
		"journal_items",
		"journal_entries",
		"fixed_assets",
		"tax_rates",
		"warehouses",
		"products",
		"bill_of_materials",
		"bom_components",
		"fleet_vehicles",
		"fleet_drivers",
		"shipments",
		"fleet_telematics_logs",
		"purchase_orders",
		"work_orders",
		"employees",
		"attendance_logs",
		"payroll_runs",
		"payroll_disbursements",
		"employee_tax_profiles",
		"shift_assignments",
		"shift_templates",
		"job_applications",
		"knowledge_base_documents",
		"ai_copilot_conversations",
		"lowcode_workflows",
		"iot_devices",
		"iot_device_readings",
		"audit_logs",
		"ai_usage_logs",
		"bank_statements",
		"bank_transactions",
		"reconciliation_matches",
		"inventory_levels",
		"stock_movements",
		"bi_dashboards",
		"field_sales_visits",
		"api_keys",
		"organization_invitations",
		"role_permissions",
		"organization_members",
		"subscriptions",
		"roles",
	}

	fmt.Println("Wiping dummy data…")

	// Delete dependent tables first (child → parent order).
	for _, t := range orgTables {
		if _, err := pool.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE org_id IN (SELECT id FROM organizations WHERE company_name LIKE 'Demo %%')`, t),
		); err != nil {
			fmt.Printf("  ⚠  %s: %v\n", t, err)
		}
	}

	// Delete users seeded with demo emails.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE email LIKE 'demo-%%'`); err != nil {
		fmt.Printf("  ⚠  users: %v\n", err)
	}

	// Delete the demo organizations (and cascade remaining children).
	if _, err := pool.Exec(ctx, `DELETE FROM organizations WHERE company_name LIKE 'Demo %%'`); err != nil {
		fmt.Printf("  ⚠  organizations: %v\n", err)
	}

	// Delete demo permissions.
	if _, err := pool.Exec(ctx, `DELETE FROM permissions WHERE permission_key LIKE 'demo.%%'`); err != nil {
		fmt.Printf("  ⚠  permissions: %v\n", err)
	}

	return nil
}

// ─── Seed ───────────────────────────────────────────────────────────

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("Seeding dummy data (20 records per table)…")

	// ── Organizations ──────────────────────────────────────
	orgIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO organizations (domain_slug, company_name, default_currency, timezone, status)
			VALUES ($1, $2, 'USD', 'UTC', 'active')
			RETURNING id`,
			fmt.Sprintf("demo-co-%02d", i+1),
			fmt.Sprintf("Demo Company %02d", i+1),
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert org %d: %w", i+1, err)
		}
		orgIDs[i] = id
	}
	fmt.Printf("  ✓ organizations (%d)\n", len(orgIDs))

	// ── Users ──────────────────────────────────────────────
	userIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, full_name, phone_number)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			fmt.Sprintf("demo-user-%02d@example.com", i+1),
			"$2a$10$dummyhashforseedingpurposesonly00000000000000000000",
			fmt.Sprintf("Demo User %02d", i+1),
			fmt.Sprintf("+1-555-%04d", 1000+i),
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert user %d: %w", i+1, err)
		}
		userIDs[i] = id
	}
	fmt.Printf("  ✓ users (%d)\n", len(userIDs))

	// ── Roles (1 per org) ──────────────────────────────────
	roleIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO roles (org_id, name, description, is_system_default)
			VALUES ($1, 'Admin', 'Organization administrator', true)
			RETURNING id`,
			orgIDs[i],
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert role %d: %w", i+1, err)
		}
		roleIDs[i] = id
	}
	fmt.Printf("  ✓ roles (%d)\n", len(roleIDs))

	// ── Permissions (demo.* keys) ──────────────────────────
	permIDs := make([]string, 20)
	permModules := []string{"crm", "hr", "fleet", "inventory", "accounting", "admin", "billing", "settings", "support", "analytics"}
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO permissions (permission_key, module, description)
			VALUES ($1, $2, $3)
			RETURNING id`,
			fmt.Sprintf("demo.permission.%d", i+1),
			permModules[i%len(permModules)],
			fmt.Sprintf("Demo permission %d", i+1),
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert permission %d: %w", i+1, err)
		}
		permIDs[i] = id
	}
	fmt.Printf("  ✓ permissions (%d)\n", len(permIDs))

	// ── Role ↔ Permissions ─────────────────────────────────
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			roleIDs[i], permIDs[i],
		)
		if err != nil {
			return fmt.Errorf("insert role_permission %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ role_permissions (20)")

	// ── Organization Members ───────────────────────────────
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO organization_members (org_id, user_id, role_id, is_active)
			 VALUES ($1, $2, $3, true) ON CONFLICT (org_id, user_id) DO NOTHING`,
			orgIDs[i], userIDs[i], roleIDs[i],
		)
		if err != nil {
			return fmt.Errorf("insert org_member %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ organization_members (20)")

	// ── Subscriptions (uses existing plans) ────────────────
	// Grab the first subscription plan (seeded by migration 000066).
	var planID string
	err := pool.QueryRow(ctx, `SELECT id FROM subscription_plans LIMIT 1`).Scan(&planID)
	if err != nil {
		planID = "00000000-0000-0000-0000-000000000000" // fallback
		fmt.Println("  ⚠  No subscription plans found, using placeholder UUID")
	}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO subscriptions (org_id, plan_id, status)
			 VALUES ($1, $2, 'active') ON CONFLICT (org_id) DO NOTHING`,
			orgIDs[i], planID,
		)
		if err != nil {
			return fmt.Errorf("insert subscription %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ subscriptions (20)")

	// ── CRM Contacts ───────────────────────────────────────
	contactIDs := make([]string, 20)
	firstNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Ivy", "Jack",
		"Karen", "Leo", "Mona", "Nick", "Olivia", "Paul", "Quinn", "Rita", "Sam", "Tina"}
	lastNames := []string{"Anderson", "Brown", "Clark", "Davis", "Evans", "Fisher", "Garcia", "Harris",
		"Ibrahim", "Jones", "Kim", "Lee", "Martin", "Nelson", "Owens", "Patel",
		"Quinn", "Ross", "Smith", "Turner"}
	companies := []string{"Acme Corp", "Globex", "Initech", "Umbrella", "Stark Ind", "Wayne Ent", "Cyberdyne",
		"Hooli", "Pied Piper", "Massive Dyn", "Virtucon", "Kramerica", "Praxis", "Dunder Mifflin",
		"Sabre", "Bluth Co", "Sterling Co", "Prestige W", "Dunder Co", "Wonka"}
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO crm_contacts (org_id, first_name, last_name, email, phone, company_name, assigned_to)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			orgIDs[i%20],
			firstNames[i],
			lastNames[i],
			fmt.Sprintf("contact-%02d@example.com", i+1),
			fmt.Sprintf("+1-555-%04d", 2000+i),
			companies[i],
			userIDs[i%20],
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert crm_contact %d: %w", i+1, err)
		}
		contactIDs[i] = id
	}
	fmt.Printf("  ✓ crm_contacts (%d)\n", len(contactIDs))

	// ── CRM Deals ──────────────────────────────────────────
	dealIDs := make([]string, 20)
	stages := []string{"prospecting", "qualification", "proposal", "negotiation", "closed_won", "closed_lost"}
	dealTitles := []string{"Cloud Migration", "SaaS License", "Consulting", "Hardware",
		"Training", "Support Plan", "Custom Dev", "Data Analytics", "AI Integration", "Audit",
		"Security Review", "Platform Setup", "API Access", "Analytics Suite", "Mobile App",
		"IoT Rollout", "ERP Module", "CRM Suite", "Fleet Mgmt", "Payroll"}
	dealAmounts := []float64{5000, 12000, 3500, 80000, 22000, 9500, 45000, 15000, 60000, 7500,
		18000, 32000, 28000, 42000, 11000, 55000, 25000, 38000, 14000, 67000}
	winProbs := []int{75, 40, 90, 20, 100, 10, 65, 85, 55, 30, 70, 45, 80, 35, 50, 60, 25, 95, 15, 88}
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO crm_deals (org_id, contact_id, title, amount, stage, ai_win_probability, assigned_to)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			orgIDs[i%20], contactIDs[i],
			fmt.Sprintf("Deal #%02d — %s", i+1, dealTitles[i]),
			dealAmounts[i],
			stages[i%len(stages)],
			winProbs[i],
			userIDs[i%20],
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert crm_deal %d: %w", i+1, err)
		}
		dealIDs[i] = id
	}
	fmt.Printf("  ✓ crm_deals (%d)\n", len(dealIDs))

	// ── CRM Quotes ─────────────────────────────────────────
	quoteAmounts := []float64{4800, 11500, 3200, 76000, 21000, 9000, 43000, 14200, 57000, 7100,
		17500, 30500, 27000, 40000, 10500, 52000, 24000, 36000, 13500, 64000}
	riskScores := []float64{0.1, 0.4, 0.05, 0.8, 0.2, 0.6, 0.15, 0.3, 0.5, 0.7,
		0.25, 0.35, 0.1, 0.45, 0.6, 0.05, 0.55, 0.2, 0.7, 0.1}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO crm_quotes (org_id, deal_id, quote_number, total_amount, ai_risk_score)
			VALUES ($1, $2, $3, $4, $5)`,
			orgIDs[i%20], dealIDs[i],
			fmt.Sprintf("QT-%04d", 1000+i),
			quoteAmounts[i],
			riskScores[i],
		)
		if err != nil {
			return fmt.Errorf("insert crm_quote %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ crm_quotes (20)")

	// ── Warehouses ─────────────────────────────────────────
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO warehouses (org_id, name, address)
			VALUES ($1, $2, $3)`,
			orgIDs[i%20],
			fmt.Sprintf("Warehouse %02d", i+1),
			fmt.Sprintf("%d Demo St, Cityville", 100+i),
		)
		if err != nil {
			return fmt.Errorf("insert warehouse %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ warehouses (20)")

	// ── Products ───────────────────────────────────────────
	productNames := []string{"Widget A", "Gadget B", "Part C", "Module D", "Sensor E",
		"Controller F", "Adapter G", "Switch H", "Relay I", "Display J",
		"Power Supply", "Cable Kit", "Mount Kit", "Filter Set", "Lubricant",
		"Bearing Pack", "Valve Unit", "Pump Assy", "Motor Unit", "PCB Board"}
	unitPrices := []float64{29.99, 149.50, 8.75, 599.00, 42.00,
		189.99, 12.50, 34.99, 7.25, 220.00,
		65.00, 18.50, 22.00, 45.00, 9.99,
		38.00, 110.00, 275.00, 420.00, 89.00}
	costPrices := []float64{12.00, 75.00, 3.50, 300.00, 18.00,
		95.00, 5.00, 15.00, 3.00, 110.00,
		30.00, 8.00, 10.00, 20.00, 4.00,
		16.00, 50.00, 140.00, 210.00, 40.00}
	reorderPts := []int{20, 10, 50, 5, 30, 8, 40, 25, 60, 3, 15, 35, 28, 22, 45, 18, 12, 6, 4, 10}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO products (org_id, sku, name, unit_price, cost_price, ai_reorder_point)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (org_id, sku) DO NOTHING`,
			orgIDs[i%20],
			fmt.Sprintf("SKU-%04d", 1000+i),
			productNames[i],
			unitPrices[i],
			costPrices[i],
			reorderPts[i],
		)
		if err != nil {
			return fmt.Errorf("insert product %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ products (20)")

	// ── Fleet Vehicles ─────────────────────────────────────
	vehicleIDs := make([]string, 20)
	vehicleMakes := []string{"Toyota", "Ford", "Honda", "Chevy", "Tesla", "BMW", "Mercedes", "Nissan", "Hyundai", "Volvo"}
	vehicleModels := []string{"Camry", "F-150", "Civic", "Silverado", "Model 3", "X5", "Sprinter", "NV200", "Tucson", "FH16"}
	for i := 0; i < 20; i++ {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO fleet_vehicles (org_id, vin, license_plate, make, model, status)
			VALUES ($1, $2, $3, $4, $5, 'active')
			RETURNING id`,
			orgIDs[i%20],
			fmt.Sprintf("VIN%014d", i+1),
			fmt.Sprintf("ABC-%04d", i+1),
			vehicleMakes[i%len(vehicleMakes)],
			vehicleModels[i%len(vehicleModels)],
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert vehicle %d: %w", i+1, err)
		}
		vehicleIDs[i] = id
	}
	fmt.Printf("  ✓ fleet_vehicles (%d)\n", len(vehicleIDs))

	// ── Fleet Drivers ──────────────────────────────────────
	driverRatings := []float64{4.5, 4.8, 3.9, 5.0, 4.2, 4.7, 3.5, 4.9, 4.1, 4.6,
		3.8, 5.0, 4.3, 4.0, 4.7, 3.6, 4.8, 4.4, 3.7, 4.9}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO fleet_drivers (org_id, user_id, license_number, safety_rating)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO NOTHING`,
			orgIDs[i%20], userIDs[i],
			fmt.Sprintf("DL-%06d", 100000+i),
			driverRatings[i],
		)
		if err != nil {
			return fmt.Errorf("insert fleet_driver %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ fleet_drivers (20)")

	// ── Shipments ──────────────────────────────────────────
	shipmentStatuses := []string{"pending", "in_transit", "delivered", "exception"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO shipments (org_id, tracking_number, origin_address, destination_address, status, assigned_vehicle_id)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			orgIDs[i%20],
			fmt.Sprintf("TRACK-%08d", i+1),
			fmt.Sprintf("%d Origin Ave, City A", 100+i),
			fmt.Sprintf("%d Destination Blvd, City B", 900-i),
			shipmentStatuses[i%len(shipmentStatuses)],
			vehicleIDs[i%20],
		)
		if err != nil {
			return fmt.Errorf("insert shipment %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ shipments (20)")

	// ── Employees ──────────────────────────────────────────
	depts := []string{"Engineering", "Sales", "Marketing", "Finance", "HR", "Operations", "Support", "Legal", "IT", "Product"}
	jobTitles := []string{"Software Engineer", "Sales Rep", "Marketing Lead", "Accountant", "HR Manager",
		"Ops Lead", "Support Agent", "Legal Counsel", "Sys Admin", "Product Manager",
		"Senior Dev", "Account Exec", "Content Writer", "Financial Analyst", "Recruiter",
		"Logistics Coord", "Tier 1 Support", "Paralegal", "DevOps Engineer", "Scrum Master"}
	salaries := []float64{85000, 60000, 72000, 68000, 95000, 55000, 45000, 110000, 90000, 105000,
		120000, 58000, 50000, 75000, 65000, 48000, 42000, 95000, 98000, 92000}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO employees (org_id, user_id, employee_code, department, job_title, salary, hired_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW() - INTERVAL '1 day' * $7)
			ON CONFLICT (org_id, employee_code) DO NOTHING`,
			orgIDs[i%20], userIDs[i],
			fmt.Sprintf("EMP-%04d", i+1),
			depts[i%len(depts)],
			jobTitles[i],
			salaries[i],
			30*(i+1),
		)
		if err != nil {
			return fmt.Errorf("insert employee %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ employees (20)")

	// ── Chart of Accounts ──────────────────────────────────
	acctTypes := []string{"asset", "liability", "equity", "revenue", "expense"}
	acctNames := []string{"Cash", "Accounts Receivable", "Inventory", "Fixed Assets", "Accounts Payable",
		"Revenue - Sales", "Revenue - Services", "COGS", "Salaries Expense", "Rent Expense",
		"Utilities Expense", "Marketing Expense", "Depreciation", "Interest Income", "Tax Expense",
		"Retained Earnings", "Owner's Equity", "Loans Payable", "Prepaid Insurance", "Deferred Revenue"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO chart_of_accounts (org_id, account_code, account_name, account_type)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (org_id, account_code) DO NOTHING`,
			orgIDs[i%20],
			fmt.Sprintf("%04d", 1000+i),
			acctNames[i],
			acctTypes[i%len(acctTypes)],
		)
		if err != nil {
			return fmt.Errorf("insert chart_of_accounts %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ chart_of_accounts (20)")

	// ── Invoices ───────────────────────────────────────────
	invoiceStatuses := []string{"draft", "sent", "paid", "overdue"}
	invoiceAmounts := []float64{2500, 12000, 800, 45000, 3200, 1800, 9500, 600, 28000, 4200,
		7500, 16000, 3500, 22000, 1100, 5500, 38000, 2100, 8500, 14500}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO invoices (org_id, invoice_number, contact_id, total_amount, status, due_date)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 day' * $6)`,
			orgIDs[i%20],
			fmt.Sprintf("INV-%04d", 1000+i),
			contactIDs[i],
			invoiceAmounts[i],
			invoiceStatuses[i%len(invoiceStatuses)],
			7*(i+1),
		)
		if err != nil {
			return fmt.Errorf("insert invoice %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ invoices (20)")

	// ── Expenses ───────────────────────────────────────────
	categories := []string{"Travel", "Software", "Office", "Marketing", "Meals",
		"Hardware", "Training", "Insurance", "Utilities", "Consulting",
		"Legal", "Shipping", "Telecom", "Repairs", "Subscriptions",
		"Catering", "Fuel", "Printing", "Rent", "Misc"}
	expenseAmounts := []float64{150, 2400, 45, 8000, 320, 1200, 600, 280, 175, 3500,
		500, 90, 210, 350, 49, 800, 120, 30, 2200, 65}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO expenses (org_id, user_id, amount, category, status, ai_fraud_flag)
			VALUES ($1, $2, $3, $4, 'approved', $5)`,
			orgIDs[i%20], userIDs[i],
			expenseAmounts[i],
			categories[i],
			i == 7, // flag one as potential fraud
		)
		if err != nil {
			return fmt.Errorf("insert expense %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ expenses (20)")

	// ── IoT Devices ────────────────────────────────────────
	deviceTypes := []string{"temperature", "pressure", "vibration", "humidity", "gps",
		"flow", "current", "voltage", "proximity", "optical"}
	deviceStatuses := []string{"online", "online", "offline", "online", "online",
		"maintenance", "online", "online", "online", "error",
		"online", "offline", "online", "online", "maintenance",
		"online", "online", "error", "online", "online"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO iot_devices (org_id, device_name, device_type, mac_address, status)
			VALUES ($1, $2, $3, $4, $5)`,
			orgIDs[i%20],
			fmt.Sprintf("Device %02d — %s", i+1, deviceTypes[i%len(deviceTypes)]),
			deviceTypes[i%len(deviceTypes)],
			fmt.Sprintf("AA:BB:CC:DD:%02X:%02X", i/256, i%256),
			deviceStatuses[i],
		)
		if err != nil {
			return fmt.Errorf("insert iot_device %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ iot_devices (20)")

	// ── Audit Logs ─────────────────────────────────────────
	actions := []string{"user.login", "user.logout", "record.created", "record.updated", "record.deleted",
		"settings.changed", "role.granted", "role.revoked", "export.downloaded", "import.uploaded",
		"api_key.created", "api_key.revoked", "password.reset", "mfa.enabled", "org.suspended",
		"invoice.paid", "expense.submitted", "deal.closed", "shipment.dispatched", "report.generated"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO audit_logs (org_id, user_id, action, ip_address, ai_anomaly_flag, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			orgIDs[i%20], userIDs[i],
			actions[i],
			fmt.Sprintf("192.168.1.%d", 10+i),
			i == 19, // flag the last one as anomaly
			fmt.Sprintf(`{"source":"seed","index":%d}`, i),
		)
		if err != nil {
			return fmt.Errorf("insert audit_log %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ audit_logs (20)")

	// ── Knowledge Base Documents ───────────────────────────
	kbTitles := []string{"Getting Started", "API Reference", "Billing FAQ", "Troubleshooting", "Best Practices",
		"Security Guide", "Onboarding Checklist", "Deployment Steps", "Performance Tips", "Migration Guide",
		"Role Management", "Webhook Setup", "Data Export", "Mobile Access", "Audit Trail",
		"Inventory Basics", "Fleet Onboarding", "HR Compliance", "IoT Setup", "AI Features"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO knowledge_base_documents (org_id, title, content)
			VALUES ($1, $2, $3)`,
			orgIDs[i%20],
			fmt.Sprintf("KB Article %02d — %s", i+1, kbTitles[i]),
			fmt.Sprintf("This is the full content of knowledge base article %02d covering %s topics in detail.",
				i+1, permModules[i%len(permModules)]),
		)
		if err != nil {
			return fmt.Errorf("insert kb_doc %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ knowledge_base_documents (20)")

	// ── API Keys ───────────────────────────────────────────
	apiKeyNames := []string{"Production", "Staging", "Development", "CI/CD", "Analytics",
		"Webhook", "Mobile App", "Partner API", "Internal", "Testing",
		"Read-Only", "Full Access", "Billing", "HR Module", "Fleet Module",
		"CRM Module", "Inventory", "Accounting", "Support", "Admin"}
	for i := 0; i < 20; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO api_keys (org_id, name, key_hash, scopes)
			VALUES ($1, $2, $3, $4)`,
			orgIDs[i%20],
			fmt.Sprintf("Key %02d — %s", i+1, apiKeyNames[i]),
			fmt.Sprintf("key_hash_%04d_seed_value", i+1),
			[]string{"read", "write", "admin"},
		)
		if err != nil {
			return fmt.Errorf("insert api_key %d: %w", i+1, err)
		}
	}
	fmt.Println("  ✓ api_keys (20)")

	return nil
}
