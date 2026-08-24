package ai

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Stub is the heuristic adapter. It implements Inferrer by dispatching each
// typed request to a keyword/rule-based heuristic. It is the default adapter
// used in development and tests; the real internal provider will replace it.
type Stub struct{}

// NewStub creates a Stub adapter.
func NewStub() *Stub { return &Stub{} }

// Infer dispatches a typed request to the matching heuristic.
func (s *Stub) Infer(ctx context.Context, req Request) (Response, error) {
	switch r := req.(type) {
	case *SentimentRequest:
		return stubSentiment(r), nil
	case *ContractRiskRequest:
		return stubContractRisk(r), nil
	case *CampaignSegmentRequest:
		return stubCampaignSegment(r), nil
	case *CampaignReachRequest:
		return stubCampaignReach(r), nil
	case *OCRRequest:
		return stubOCR(r), nil
	case *ExpenseAuditRequest:
		return stubExpenseAudit(r), nil
	case *RouteOptimizeRequest:
		return stubRouteOptimize(r), nil
	case *StockoutRequest:
		return stubStockout(r), nil
	case *SupplierRiskRequest:
		return stubSupplierRisk(r), nil
	case *BottleneckRiskRequest:
		return stubBottleneckRisk(r), nil
	case *SupplierRiskRatingRequest:
		return stubSupplierRiskRating(r), nil
	case *ResumeParseRequest:
		return stubResumeParse(r), nil
	case *MatchScoreRequest:
		return stubMatchScore(r), nil
	case *RelevanceRequest:
		return stubRelevance(r), nil
	case *KnowledgeAnswerRequest:
		return stubKnowledgeAnswer(r), nil
	case *ShiftPredictionRequest:
		return stubShiftPrediction(r), nil
	case *TextToSQLRequest:
		return stubTextToSQL(r), nil
	case *QueryResultsRequest:
		return stubQueryResults(r), nil
	case *AnomalyRequest:
		return stubAnomaly(r), nil
	case *ReadingAnomalyRequest:
		return stubReadingAnomaly(r), nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownCapability, req)
	}
}

// ── CRM ────────────────────────────────────────────────────────────────

func stubSentiment(r *SentimentRequest) *SentimentResponse {
	text := r.Subject + " " + r.Description

	negativeWords := []string{"urgent", "broken", "error", "fail", "crash", "bug", "issue", "problem", "critical", "down", "lost", "cannot", "not working", "stuck", "blocked"}
	positiveWords := []string{"great", "thanks", "helpful", "appreciate", "good", "excellent", "love", "awesome", "perfect", "smooth"}

	negCount := 0
	posCount := 0
	lower := strings.ToLower(text)

	for _, w := range negativeWords {
		if strings.Contains(lower, w) {
			negCount++
		}
	}
	for _, w := range positiveWords {
		if strings.Contains(lower, w) {
			posCount++
		}
	}

	total := negCount + posCount
	var score float64
	if total == 0 {
		score = 0.1 + rand.Float64()*0.3
	} else {
		score = (float64(posCount) - float64(negCount)) / float64(total)
		score += (rand.Float64() - 0.5) * 0.2
	}
	if score > 1.0 {
		score = 1.0
	}
	if score < -1.0 {
		score = -1.0
	}

	var response string
	switch {
	case score < -0.3:
		response = "Thank you for reaching out. I understand this is frustrating. Our team is prioritizing your issue and will respond within 2 hours. In the meantime, could you provide any additional details or screenshots?"
	case score < 0.3:
		response = "Thank you for contacting support. We've received your ticket and will review it shortly. A team member will follow up within 4 business hours."
	default:
		response = "Thanks for your message! We're glad to hear from you. We'll review your request and get back to you within 8 business hours. Have a great day!"
	}

	return &SentimentResponse{
		Score:             score,
		SuggestedResponse: response,
		Priority:          derivePriority(score),
	}
}

func derivePriority(sentiment float64) string {
	switch {
	case sentiment < -0.5:
		return "urgent"
	case sentiment < -0.2:
		return "high"
	case sentiment < 0.3:
		return "medium"
	default:
		return "low"
	}
}

func stubContractRisk(r *ContractRiskRequest) *ContractRiskResponse {
	var clauses []FlaggedClause
	riskScore := 0.0

	highRiskPatterns := map[string]FlaggedClause{
		"indemnification": {
			Clause:       "Unlimited indemnification clause detected",
			RiskLevel:    "high",
			SuggestedFix: "Cap indemnification liability to the total contract value",
		},
		"penalty": {
			Clause:       "Asymmetric penalty clause detected",
			RiskLevel:    "critical",
			SuggestedFix: "Negotiate mutual penalty terms or cap at reasonable amount",
		},
		"termination": {
			Clause:       "Unilateral termination rights without cause",
			RiskLevel:    "medium",
			SuggestedFix: "Add mutual termination clause with 30-day notice period",
		},
		"confidential": {
			Clause:       "Overly broad confidentiality obligations",
			RiskLevel:    "low",
			SuggestedFix: "Limit confidentiality duration to 3 years post-termination",
		},
	}

	lower := strings.ToLower(r.ContractText)
	for keyword, clause := range highRiskPatterns {
		if strings.Contains(lower, keyword) {
			clauses = append(clauses, clause)
			switch clause.RiskLevel {
			case "critical":
				riskScore += 30
			case "high":
				riskScore += 20
			case "medium":
				riskScore += 10
			case "low":
				riskScore += 5
			}
		}
	}

	if len(clauses) == 0 {
		riskScore = 5 + rand.Float64()*10
	} else {
		riskScore += rand.Float64() * 10
	}
	if riskScore > 100 {
		riskScore = 100
	}

	return &ContractRiskResponse{RiskScore: riskScore, Clauses: clauses}
}

func stubCampaignSegment(r *CampaignSegmentRequest) *CampaignSegmentResponse {
	segments := map[string]string{
		"email":  `{"criteria": "contacts with open rate > 30% in last 90 days", "estimated_size": 1250}`,
		"sms":    `{"criteria": "contacts with mobile phone and opted-in for SMS", "estimated_size": 840}`,
		"social": `{"criteria": "contacts who engaged with brand posts in last 60 days", "estimated_size": 2100}`,
		"push":   `{"criteria": "contacts with app installed and notifications enabled", "estimated_size": 670}`,
		"in_app": `{"criteria": "active users with at least 3 sessions in last 30 days", "estimated_size": 980}`,
	}
	criteria, ok := segments[r.Channel]
	if !ok {
		criteria = `{"criteria": "all contacts", "estimated_size": 500}`
	}
	return &CampaignSegmentResponse{SegmentCriteria: criteria}
}

func stubCampaignReach(r *CampaignReachRequest) *CampaignReachResponse {
	baseReach := map[string]int{
		"email":  5000,
		"sms":    3000,
		"social": 15000,
		"push":   2000,
		"in_app": 4000,
	}
	reach, ok := baseReach[r.Channel]
	if !ok {
		reach = 1000
	}
	// Add some randomness (±20%)
	reach += int(float64(reach) * (rand.Float64()*0.4 - 0.2))
	return &CampaignReachResponse{EstimatedReach: reach}
}

// ── Accounting ─────────────────────────────────────────────────────────

func stubOCR(r *OCRRequest) *OCRResponse {
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

	taxRate := 0.08 + rand.Float64()*0.12
	taxAmount := subtotal * taxRate

	return &OCRResponse{
		VendorName:    vendors[vendorIdx],
		InvoiceNumber: invNum,
		LineItems:     lineItems,
		TaxAmount:     taxAmount,
		TotalAmount:   subtotal + taxAmount,
	}
}

func stubExpenseAudit(r *ExpenseAuditRequest) *ExpenseAuditResponse {
	var flags []string

	if r.Amount > 5000 {
		flags = append(flags, "High-value expense (over $5,000) requires manager approval")
	}

	lowerCat := strings.ToLower(r.Category)
	if lowerCat == "entertainment" && r.Amount > 1000 {
		flags = append(flags, "Entertainment expense exceeds $1,000 threshold — policy limit is $1,000")
	}
	if lowerCat == "travel" && r.Amount > 3000 {
		flags = append(flags, "Travel expense exceeds $3,000 — please attach itinerary")
	}

	if strings.Contains(r.ReceiptFileName, "dup") || strings.Contains(r.ReceiptFileName, "copy") {
		flags = append(flags, "Potential duplicate receipt detected — filename contains 'dup' or 'copy'")
	}

	if r.SubmittedWeekend {
		flags = append(flags, "Expense submitted on weekend — unusual timing")
	}

	if len(flags) > 0 {
		return &ExpenseAuditResponse{FraudFlag: true, AuditNotes: strings.Join(flags, "; ")}
	}
	return &ExpenseAuditResponse{FraudFlag: false, AuditNotes: "No policy violations detected. Expense appears compliant."}
}

// ── Supply Chain ───────────────────────────────────────────────────────

func stubRouteOptimize(r *RouteOptimizeRequest) *RouteOptimizeResponse {
	var waypoints []Waypoint

	baseLat := 40.7128 + (rand.Float64()-0.5)*2.0
	baseLng := -74.0060 + (rand.Float64()-0.5)*2.0

	for i := range r.Shipments {
		waypoints = append(waypoints, Waypoint{
			Type:        "Point",
			Coordinates: []float64{baseLng + float64(i)*0.02, baseLat + float64(i)*0.02},
		})
	}
	waypoints = append(waypoints, Waypoint{
		Type:        "Point",
		Coordinates: []float64{baseLng + 0.1, baseLat + 0.1},
	})

	return &RouteOptimizeResponse{Waypoints: waypoints}
}

func stubStockout(r *StockoutRequest) *StockoutResponse {
	currentStock := r.CurrentStock
	reorderPoint := r.ReorderPoint

	var stockoutDays int
	if currentStock <= 0 {
		stockoutDays = 0
	} else {
		dailyRate := 1 + rand.Intn(maxInt(1, currentStock/3))
		stockoutDays = currentStock / dailyRate
		if stockoutDays < 0 {
			stockoutDays = 0
		}
		if stockoutDays > 60 {
			stockoutDays = 60
		}
	}

	shortfall := reorderPoint - currentStock
	var recommended int
	if shortfall <= 0 {
		recommended = reorderPoint
	} else {
		buffer := float64(shortfall) * (0.2 + rand.Float64()*0.3)
		recommended = shortfall + int(buffer)
	}

	return &StockoutResponse{
		PredictedStockoutDays: stockoutDays,
		RecommendedReorderQty: recommended,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func stubSupplierRisk(r *SupplierRiskRequest) *SupplierRiskResponse {
	baseScore := 30.0 + rand.Float64()*40.0
	if r.OpenPOs > 5 {
		baseScore += 10.0
	}
	if r.TotalSpend > 100000 {
		baseScore += 10.0
	}
	if baseScore > 100 {
		baseScore = 100
	}

	var rating string
	switch {
	case baseScore >= 70:
		rating = "High Risk"
	case baseScore >= 40:
		rating = "Medium Risk"
	default:
		rating = "Low Risk"
	}

	return &SupplierRiskResponse{Score: baseScore, Rating: rating}
}

func stubBottleneckRisk(r *BottleneckRiskRequest) *BottleneckRiskResponse {
	rating := "Low"
	switch {
	case r.Quantity > 100:
		rating = "High"
	case r.Quantity > 50:
		rating = "Medium"
	}
	return &BottleneckRiskResponse{Rating: rating}
}

func stubSupplierRiskRating(r *SupplierRiskRatingRequest) *SupplierRiskRatingResponse {
	riskLevels := []string{"Low Risk", "Medium Risk", "High Risk"}
	return &SupplierRiskRatingResponse{Rating: riskLevels[rand.Intn(len(riskLevels))]}
}

// ── HR ─────────────────────────────────────────────────────────────────

func stubResumeParse(r *ResumeParseRequest) *ResumeParseResponse {
	base := strings.TrimSuffix(strings.TrimSuffix(r.FileName, ".pdf"), ".docx")
	parts := strings.Split(strings.ReplaceAll(base, "_", " "), " ")
	var name string
	if len(parts) >= 2 {
		name = parts[0] + " " + parts[len(parts)-1]
	} else if len(parts) == 1 && parts[0] != "" {
		name = parts[0]
	} else {
		name = "John Doe"
	}
	name = titleCase(name)

	emailName := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	email := emailName + "@email.com"

	allSkills := []string{
		"Python", "Java", "JavaScript", "TypeScript", "Go", "Rust",
		"React", "Angular", "Vue.js", "Node.js", "PostgreSQL", "MongoDB",
		"Docker", "Kubernetes", "AWS", "Azure", "GCP", "Terraform",
		"CI/CD", "Git", "Agile", "Scrum", "Machine Learning", "Data Analysis",
		"SQL", "REST APIs", "GraphQL", "Microservices", "Linux", "DevOps",
	}
	numSkills := 3 + rand.Intn(5)
	rand.Shuffle(len(allSkills), func(i, j int) {
		allSkills[i], allSkills[j] = allSkills[j], allSkills[i]
	})
	skills := allSkills[:numSkills]

	return &ResumeParseResponse{Name: name, Email: email, Skills: skills}
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func stubMatchScore(r *MatchScoreRequest) *MatchScoreResponse {
	baseScore := 40 + len(r.Skills)*3
	jitter := rand.Intn(20) - 10
	score := baseScore + jitter
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return &MatchScoreResponse{Score: score}
}

func stubRelevance(r *RelevanceRequest) *RelevanceResponse {
	queryLower := strings.ToLower(r.Query)
	contentLower := strings.ToLower(r.Content)

	terms := strings.Fields(queryLower)
	matchCount := 0
	for _, term := range terms {
		if len(term) > 2 && strings.Contains(contentLower, term) {
			matchCount++
		}
	}

	score := float64(matchCount) / float64(len(terms))
	score += (rand.Float64() - 0.5) * 0.2
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.05 {
		score = 0.05 + rand.Float64()*0.1
	}
	return &RelevanceResponse{Score: mathRound(score, 4)}
}

func mathRound(v float64, places int) float64 {
	pow := 1.0
	for i := 0; i < places; i++ {
		pow *= 10
	}
	return float64(int(v*pow+0.5)) / pow
}

func stubKnowledgeAnswer(r *KnowledgeAnswerRequest) *KnowledgeAnswerResponse {
	if len(r.Docs) == 0 {
		return &KnowledgeAnswerResponse{
			Answer: fmt.Sprintf("I couldn't find any specific documents in our knowledge base regarding \"%s\". Please try rephrasing your question or contact your HR representative for assistance.", r.Query),
		}
	}

	topDoc := r.Docs[0]
	answer := fmt.Sprintf(
		"Based on our knowledge base, here is the answer to \"%s\": The document **%s** addresses this topic. %s",
		r.Query,
		topDoc.Title,
		answerFromContent(topDoc.Content),
	)
	return &KnowledgeAnswerResponse{Answer: answer}
}

func answerFromContent(content string) string {
	if len(content) > 300 {
		content = content[:300]
	}
	if idx := strings.LastIndex(content, "."); idx > len(content)/2 {
		content = content[:idx+1]
	}
	return content
}

func stubShiftPrediction(r *ShiftPredictionRequest) *ShiftPredictionResponse {
	weekday := r.Date.Weekday()
	month := r.Date.Month()

	predicted := r.BaseHeadcount
	switch weekday {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
		predicted += 2
	case time.Friday:
		predicted += 1
	case time.Saturday:
		predicted -= 1
	case time.Sunday:
		predicted -= 2
	}

	// Seasonality: summer months need fewer staff, winter more.
	switch month {
	case time.December, time.January:
		predicted += 1
	case time.July, time.August:
		predicted -= 1
	}

	if predicted < 0 {
		predicted = 0
	}

	confidence := 0.6 + rand.Float64()*0.3
	if confidence > 1.0 {
		confidence = 1.0
	}

	reasoning := fmt.Sprintf("Based on historical patterns for %s", weekday.String())
	if r.Department != nil && *r.Department != "" {
		reasoning += fmt.Sprintf(" in %s", *r.Department)
	}

	return &ShiftPredictionResponse{
		Predicted:  predicted,
		Confidence: confidence,
		Reasoning:  reasoning,
	}
}

// ── Platform ───────────────────────────────────────────────────────────

func stubTextToSQL(r *TextToSQLRequest) *TextToSQLResponse {
	lower := strings.ToLower(r.Prompt)

	switch {
	case strings.Contains(lower, "sales rep") || strings.Contains(lower, "revenue"):
		return &TextToSQLResponse{
			SQL: `SELECT u.full_name AS sales_rep, SUM(d.amount) AS total_revenue
FROM crm_deals d
JOIN users u ON d.assigned_to = u.id
WHERE d.stage = 'closed_won'
  AND d.created_at BETWEEN '2025-04-01' AND '2025-06-30'
GROUP BY u.full_name
ORDER BY total_revenue DESC
LIMIT 5`,
			TableName: "crm_deals",
		}
	case strings.Contains(lower, "invoice") || strings.Contains(lower, "overdue"):
		return &TextToSQLResponse{
			SQL: `SELECT i.invoice_number, c.first_name || ' ' || COALESCE(c.last_name, '') AS customer,
       i.total_amount, i.due_date,
       CURRENT_DATE - i.due_date AS days_overdue
FROM invoices i
JOIN crm_contacts c ON i.contact_id = c.id
WHERE i.status = 'sent' AND i.due_date < CURRENT_DATE
ORDER BY days_overdue DESC
LIMIT 10`,
			TableName: "invoices",
		}
	case strings.Contains(lower, "attendance") || strings.Contains(lower, "clock"):
		return &TextToSQLResponse{
			SQL: `SELECT e.employee_code, e.department,
       COUNT(a.id) AS clock_ins,
       MIN(a.clock_in) AS first_clock_in,
       MAX(a.clock_in) AS last_clock_in
FROM employees e
JOIN attendance_logs a ON a.employee_id = e.id
WHERE a.clock_in >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY e.employee_code, e.department
ORDER BY clock_ins DESC`,
			TableName: "attendance_logs",
		}
	case strings.Contains(lower, "fleet") || strings.Contains(lower, "vehicle"):
		return &TextToSQLResponse{
			SQL: `SELECT v.license_plate, v.make, v.model,
       MAX(t.speed_kmh) AS max_speed,
       AVG(t.speed_kmh) AS avg_speed,
       AVG(t.fuel_level_pct) AS avg_fuel_pct
FROM fleet_vehicles v
JOIN fleet_telematics_logs t ON t.vehicle_id = v.id
WHERE t.recorded_at >= NOW() - INTERVAL '7 days'
GROUP BY v.license_plate, v.make, v.model
ORDER BY max_speed DESC`,
			TableName: "fleet_telematics_logs",
		}
	default:
		return &TextToSQLResponse{
			SQL: `SELECT id, org_id, created_at
FROM organizations
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 5`,
			TableName: "organizations",
		}
	}
}

func stubQueryResults(r *QueryResultsRequest) *QueryResultsResponse {
	lower := strings.ToLower(r.Prompt)

	switch {
	case strings.Contains(lower, "sales rep") || strings.Contains(lower, "revenue"):
		return &QueryResultsResponse{Data: []map[string]any{
			{"sales_rep": "Alice Johnson", "total_revenue": 245000.50},
			{"sales_rep": "Bob Martinez", "total_revenue": 198750.00},
			{"sales_rep": "Carol Chen", "total_revenue": 176200.75},
			{"sales_rep": "David Kim", "total_revenue": 152300.25},
			{"sales_rep": "Eve Thompson", "total_revenue": 134500.00},
		}}
	case strings.Contains(lower, "invoice") || strings.Contains(lower, "overdue"):
		return &QueryResultsResponse{Data: []map[string]any{
			{"invoice_number": "INV-2025-0042", "customer": "Acme Corp", "total_amount": 12500.00, "due_date": "2025-06-15", "days_overdue": 51},
			{"invoice_number": "INV-2025-0051", "customer": "Globex Inc", "total_amount": 8750.00, "due_date": "2025-07-01", "days_overdue": 35},
			{"invoice_number": "INV-2025-0058", "customer": "Initech", "total_amount": 3200.00, "due_date": "2025-07-15", "days_overdue": 21},
		}}
	case strings.Contains(lower, "attendance") || strings.Contains(lower, "clock"):
		return &QueryResultsResponse{Data: []map[string]any{
			{"employee_code": "EMP-001", "department": "Engineering", "clock_ins": 22, "first_clock_in": "2025-07-06T08:55:00Z", "last_clock_in": "2025-08-04T09:02:00Z"},
			{"employee_code": "EMP-002", "department": "Sales", "clock_ins": 21, "first_clock_in": "2025-07-06T08:30:00Z", "last_clock_in": "2025-08-04T08:45:00Z"},
			{"employee_code": "EMP-003", "department": "Engineering", "clock_ins": 20, "first_clock_in": "2025-07-07T09:10:00Z", "last_clock_in": "2025-08-04T09:15:00Z"},
		}}
	case strings.Contains(lower, "fleet") || strings.Contains(lower, "vehicle"):
		return &QueryResultsResponse{Data: []map[string]any{
			{"license_plate": "ABC-1234", "make": "Ford", "model": "Transit", "max_speed": 112.5, "avg_speed": 68.3, "avg_fuel_pct": 72.1},
			{"license_plate": "XYZ-5678", "make": "Mercedes", "model": "Sprinter", "max_speed": 105.0, "avg_speed": 62.7, "avg_fuel_pct": 65.4},
			{"license_plate": "DEF-9012", "make": "Ram", "model": "ProMaster", "max_speed": 98.2, "avg_speed": 55.9, "avg_fuel_pct": 58.3},
		}}
	default:
		return &QueryResultsResponse{Data: []map[string]any{
			{"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "org_id": "f1e2d3c4-b5a6-9870-fedc-ba0987654321", "created_at": "2025-01-15T10:00:00Z"},
		}}
	}
}

func stubReadingAnomaly(r *ReadingAnomalyRequest) *ReadingAnomalyResponse {
	lower := strings.ToLower(r.MetricName)
	switch {
	case strings.Contains(lower, "temperature") && r.MetricValue > 85.0:
		return &ReadingAnomalyResponse{AnomalyDetected: true, AnomalyDescription: fmt.Sprintf("High temperature alert: %.1f%s exceeds threshold of 85.0%s", r.MetricValue, r.Unit, r.Unit)}
	case strings.Contains(lower, "vibration") && r.MetricValue > 7.5:
		return &ReadingAnomalyResponse{AnomalyDetected: true, AnomalyDescription: fmt.Sprintf("Abnormal vibration detected: %.2f%s (possible bearing failure)", r.MetricValue, r.Unit)}
	case strings.Contains(lower, "pressure") && r.MetricValue > 150.0:
		return &ReadingAnomalyResponse{AnomalyDetected: true, AnomalyDescription: fmt.Sprintf("Pressure spike detected: %.1f%s exceeds safe operating range", r.MetricValue, r.Unit)}
	case strings.Contains(lower, "energy") && r.MetricValue > 500.0:
		return &ReadingAnomalyResponse{AnomalyDetected: true, AnomalyDescription: fmt.Sprintf("Excessive energy consumption: %.1f%s (possible equipment malfunction)", r.MetricValue, r.Unit)}
	}
	// Random demo anomaly (~5%)
	if rand.Float64() < 0.05 {
		return &ReadingAnomalyResponse{AnomalyDetected: true, AnomalyDescription: fmt.Sprintf("AI anomaly detected in %s reading: %.2f%s deviates from expected pattern", r.MetricName, r.MetricValue, r.Unit)}
	}
	return &ReadingAnomalyResponse{AnomalyDetected: false}
}

func stubAnomaly(r *AnomalyRequest) *AnomalyResponse {
	lower := strings.ToLower(r.Action)

	switch {
	case strings.Contains(lower, "login") || strings.Contains(lower, "signin"):
		return &AnomalyResponse{Type: "suspicious_login", RiskScore: 0.75 + rand.Float64()*0.2}
	case strings.Contains(lower, "delete") || strings.Contains(lower, "remove"):
		return &AnomalyResponse{Type: "unauthorized_delete_attempt", RiskScore: 0.85 + rand.Float64()*0.15}
	case strings.Contains(lower, "export") || strings.Contains(lower, "download"):
		return &AnomalyResponse{Type: "data_exfiltration", RiskScore: 0.7 + rand.Float64()*0.25}
	case strings.Contains(lower, "permission") || strings.Contains(lower, "role"):
		return &AnomalyResponse{Type: "privilege_escalation", RiskScore: 0.8 + rand.Float64()*0.2}
	case strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key"):
		return &AnomalyResponse{Type: "api_key_abuse", RiskScore: 0.65 + rand.Float64()*0.3}
	default:
		return &AnomalyResponse{Type: "anomalous_activity", RiskScore: 0.6 + rand.Float64()*0.3}
	}
}
