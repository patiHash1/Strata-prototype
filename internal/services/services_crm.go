package services

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type CRMContact struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	FirstName   string     `json:"first_name"`
	LastName    *string    `json:"last_name,omitempty"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	CompanyName *string    `json:"company_name,omitempty"`
	AssignedTo  *uuid.UUID `json:"assigned_to,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CRMDeal struct {
	ID               uuid.UUID  `json:"id"`
	OrgID            uuid.UUID  `json:"org_id"`
	ContactID        *uuid.UUID `json:"contact_id,omitempty"`
	Title            string     `json:"title"`
	Amount           float64    `json:"amount"`
	Stage            string     `json:"stage"`
	AIWinProbability *int       `json:"ai_win_probability,omitempty"`
	AssignedTo       *uuid.UUID `json:"assigned_to,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type CRMQuote struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	DealID      *uuid.UUID `json:"deal_id,omitempty"`
	QuoteNumber string     `json:"quote_number"`
	TotalAmount float64    `json:"total_amount"`
	AIRiskScore *float64   `json:"ai_risk_score,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// FlaggedClause represents a risky clause found during AI analysis.
type FlaggedClause struct {
	Clause       string `json:"clause"`
	RiskLevel    string `json:"risk_level"` // low, medium, high, critical
	SuggestedFix string `json:"suggested_fix"`
}

// CRMHelpdeskTicket represents a support ticket.
type CRMHelpdeskTicket struct {
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	ContactID           *uuid.UUID `json:"contact_id,omitempty"`
	Subject             string     `json:"subject"`
	Description         string     `json:"description"`
	Priority            string     `json:"priority"`
	Status              string     `json:"status"`
	AISentimentScore    *float64   `json:"ai_sentiment_score,omitempty"`
	AISuggestedResponse *string    `json:"ai_suggested_response,omitempty"`
	AssignedTo          *uuid.UUID `json:"assigned_to,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// FieldSalesVisit represents a scheduled field sales visit.
type FieldSalesVisit struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	ContactID    *uuid.UUID `json:"contact_id,omitempty"`
	SalesRepID   *uuid.UUID `json:"sales_rep_id,omitempty"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	LocationLat  *float64   `json:"location_lat,omitempty"`
	LocationLong *float64   `json:"location_long,omitempty"`
	Status       string     `json:"status"`
	Notes        *string    `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CRMCampaign represents a marketing campaign.
type CRMCampaign struct {
	ID                      uuid.UUID `json:"id"`
	OrgID                   uuid.UUID `json:"org_id"`
	Name                    string    `json:"name"`
	Channel                 string    `json:"channel"`
	AITargetSegmentCriteria *string   `json:"ai_target_segment_criteria,omitempty"`
	Budget                  *float64  `json:"budget,omitempty"`
	Status                  string    `json:"status"`
	CreatedAt               time.Time `json:"created_at"`
}

// ---- Repository ----

type crmRepository struct {
	pool *pgxpool.Pool
}

func newCRMRepository(pool *pgxpool.Pool) *crmRepository {
	return &crmRepository{pool: pool}
}

func (r *crmRepository) CreateContact(ctx context.Context, c *CRMContact) error {
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_contacts (id, org_id, first_name, last_name, email, phone, company_name, assigned_to, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.OrgID, c.FirstName, c.LastName, c.Email, c.Phone, c.CompanyName, c.AssignedTo, c.CreatedAt)
	return err
}

func (r *crmRepository) CreateDeal(ctx context.Context, d *CRMDeal) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_deals (id, org_id, contact_id, title, amount, stage, ai_win_probability, assigned_to, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, d.ID, d.OrgID, d.ContactID, d.Title, d.Amount, d.Stage, d.AIWinProbability, d.AssignedTo, d.CreatedAt)
	return err
}

func (r *crmRepository) GetQuoteByID(ctx context.Context, id uuid.UUID) (*CRMQuote, error) {
	q := &CRMQuote{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, deal_id, quote_number, total_amount, ai_risk_score, created_at
		FROM crm_quotes WHERE id = $1
	`, id).Scan(&q.ID, &q.OrgID, &q.DealID, &q.QuoteNumber, &q.TotalAmount, &q.AIRiskScore, &q.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return q, err
}

func (r *crmRepository) UpdateQuoteRisk(ctx context.Context, id uuid.UUID, riskScore float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE crm_quotes SET ai_risk_score = $1 WHERE id = $2
	`, riskScore, id)
	return err
}

func (r *crmRepository) GetContactByID(ctx context.Context, id uuid.UUID) (*CRMContact, error) {
	c := &CRMContact{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, first_name, last_name, email, phone, company_name, assigned_to, created_at
		FROM crm_contacts WHERE id = $1
	`, id).Scan(&c.ID, &c.OrgID, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.CompanyName, &c.AssignedTo, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *crmRepository) CreateTicket(ctx context.Context, t *CRMHelpdeskTicket) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.Status == "" {
		t.Status = "open"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_helpdesk_tickets (id, org_id, contact_id, subject, description, priority, status, ai_sentiment_score, ai_suggested_response, assigned_to, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, t.ID, t.OrgID, t.ContactID, t.Subject, t.Description, t.Priority, t.Status, t.AISentimentScore, t.AISuggestedResponse, t.AssignedTo, t.CreatedAt)
	return err
}

func (r *crmRepository) CreateFieldSalesVisit(ctx context.Context, v *FieldSalesVisit) error {
	v.ID = uuid.New()
	v.CreatedAt = time.Now()
	if v.Status == "" {
		v.Status = "scheduled"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO field_sales_visits (id, org_id, contact_id, sales_rep_id, scheduled_at, location_lat, location_long, status, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, v.ID, v.OrgID, v.ContactID, v.SalesRepID, v.ScheduledAt, v.LocationLat, v.LocationLong, v.Status, v.Notes, v.CreatedAt)
	return err
}

func (r *crmRepository) CreateCampaign(ctx context.Context, c *CRMCampaign) error {
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	if c.Status == "" {
		c.Status = "draft"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_campaigns (id, org_id, name, channel, ai_target_segment_criteria, budget, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, c.ID, c.OrgID, c.Name, c.Channel, c.AITargetSegmentCriteria, c.Budget, c.Status, c.CreatedAt)
	return err
}

func (r *crmRepository) GetCampaignByID(ctx context.Context, id uuid.UUID) (*CRMCampaign, error) {
	c := &CRMCampaign{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, channel, ai_target_segment_criteria, budget, status, created_at
		FROM crm_campaigns WHERE id = $1
	`, id).Scan(&c.ID, &c.OrgID, &c.Name, &c.Channel, &c.AITargetSegmentCriteria, &c.Budget, &c.Status, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *crmRepository) UpdateCampaignStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE crm_campaigns SET status = $1 WHERE id = $2
	`, status, id)
	return err
}

// ---- Service ----

type CRMService struct {
	repo *crmRepository
}

func NewCRMService(pool *pgxpool.Pool) *CRMService {
	return &CRMService{repo: newCRMRepository(pool)}
}

// CreateLead creates a new contact as a lead, triggers AI scoring, and creates a deal.
func (s *CRMService) CreateLead(ctx context.Context, orgID uuid.UUID, firstName string, lastName *string, email string, companyName *string, estimatedDealSize *float64) (*CRMContact, uuid.UUID, int, error) {
	// Create contact
	contact := &CRMContact{
		OrgID:       orgID,
		FirstName:   firstName,
		LastName:    lastName,
		Email:       &email,
		CompanyName: companyName,
	}
	if err := s.repo.CreateContact(ctx, contact); err != nil {
		return nil, uuid.Nil, 0, err
	}

	// Simulate AI win probability (0–100)
	aiWinProb := 40 + rand.Intn(41) // 40–80 range for a realistic score

	// Generate a deterministic assigned_to (in production this would come from routing logic)
	assignedTo := uuid.New()

	// Create a deal linked to the contact
	dealTitle := firstName
	if lastName != nil && *lastName != "" {
		dealTitle = firstName + " " + *lastName
	}
	dealTitle = dealTitle + " - Deal"

	dealAmount := 0.0
	if estimatedDealSize != nil {
		dealAmount = *estimatedDealSize
	}

	deal := &CRMDeal{
		OrgID:            orgID,
		ContactID:        &contact.ID,
		Title:            dealTitle,
		Amount:           dealAmount,
		Stage:            "lead",
		AIWinProbability: &aiWinProb,
		AssignedTo:       &assignedTo,
	}
	if err := s.repo.CreateDeal(ctx, deal); err != nil {
		return nil, uuid.Nil, 0, err
	}

	return contact, assignedTo, aiWinProb, nil
}

// AnalyzeContractRisk performs AI risk analysis on a quote's contract text.
func (s *CRMService) AnalyzeContractRisk(ctx context.Context, orgID uuid.UUID, quoteID uuid.UUID, contractText string) (float64, []FlaggedClause, error) {
	// Verify quote exists
	quote, err := s.repo.GetQuoteByID(ctx, quoteID)
	if err != nil {
		return 0, nil, err
	}
	if quote == nil {
		return 0, nil, ErrQuoteNotFound
	}
	if quote.OrgID != orgID {
		return 0, nil, ErrQuoteNotInOrg
	}

	// Simulate AI risk analysis based on contract text heuristics
	riskScore, clauses := aiAnalyzeContract(contractText)

	// Persist the risk score
	if err := s.repo.UpdateQuoteRisk(ctx, quoteID, riskScore); err != nil {
		return 0, nil, err
	}

	return riskScore, clauses, nil
}

// CreateTicket creates a support ticket with AI sentiment analysis and auto-routing.
func (s *CRMService) CreateTicket(ctx context.Context, orgID uuid.UUID, contactID uuid.UUID, subject, description string) (*CRMHelpdeskTicket, error) {
	// Verify contact exists and belongs to the org
	contact, err := s.repo.GetContactByID(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if contact == nil {
		return nil, ErrContactNotFound
	}
	if contact.OrgID != orgID {
		return nil, ErrContactNotInOrg
	}

	// Simulate AI sentiment analysis
	sentimentScore, suggestedResponse := aiAnalyzeSentiment(subject, description)

	// Determine priority based on sentiment (negative → higher priority)
	priority := aiDeterminePriority(sentimentScore)

	// Auto-route to an assignee (in production this would use a routing engine)
	assignedTo := uuid.New()

	ticket := &CRMHelpdeskTicket{
		OrgID:               orgID,
		ContactID:           &contactID,
		Subject:             subject,
		Description:         description,
		Priority:            priority,
		Status:              "open",
		AISentimentScore:    &sentimentScore,
		AISuggestedResponse: &suggestedResponse,
		AssignedTo:          &assignedTo,
	}
	if err := s.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}

// ScheduleFieldVisit creates a field sales visit and simulates AI route optimization.
func (s *CRMService) ScheduleFieldVisit(ctx context.Context, orgID uuid.UUID, contactID *uuid.UUID, salesRepID *uuid.UUID, scheduledAt time.Time, locationLat *float64, locationLong *float64, notes *string) (*FieldSalesVisit, int, error) {
	visit := &FieldSalesVisit{
		OrgID:        orgID,
		ContactID:    contactID,
		SalesRepID:   salesRepID,
		ScheduledAt:  scheduledAt,
		LocationLat:  locationLat,
		LocationLong: locationLong,
		Status:       "scheduled",
		Notes:        notes,
	}
	if err := s.repo.CreateFieldSalesVisit(ctx, visit); err != nil {
		return nil, 0, err
	}

	// Simulate AI route optimization: estimated travel time in minutes
	estimatedTravelTime := 15 + rand.Intn(46) // 15–60 minutes

	return visit, estimatedTravelTime, nil
}

// CreateCampaign creates a marketing campaign with simulated AI audience segmentation.
func (s *CRMService) CreateCampaign(ctx context.Context, orgID uuid.UUID, name, channel string, budget *float64) (*CRMCampaign, string, error) {
	// Simulate AI audience segmentation
	segmentCriteria := aiSegmentAudience(channel)

	campaign := &CRMCampaign{
		OrgID:                   orgID,
		Name:                    name,
		Channel:                 channel,
		AITargetSegmentCriteria: &segmentCriteria,
		Budget:                  budget,
		Status:                  "draft",
	}
	if err := s.repo.CreateCampaign(ctx, campaign); err != nil {
		return nil, "", err
	}

	return campaign, segmentCriteria, nil
}

// LaunchCampaign simulates launching a campaign and returns estimated reach.
func (s *CRMService) LaunchCampaign(ctx context.Context, orgID uuid.UUID, campaignID uuid.UUID) (*CRMCampaign, int, error) {
	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, 0, err
	}
	if campaign == nil {
		return nil, 0, ErrCampaignNotFound
	}
	if campaign.OrgID != orgID {
		return nil, 0, ErrCampaignNotInOrg
	}

	// Simulate launch: update status to active
	if err := s.repo.UpdateCampaignStatus(ctx, campaignID, "active"); err != nil {
		return nil, 0, err
	}
	campaign.Status = "active"

	// Simulate estimated reach based on channel
	estimatedReach := aiEstimateReach(campaign.Channel)

	return campaign, estimatedReach, nil
}

// aiAnalyzeSentiment simulates AI sentiment analysis on ticket text.
// Returns a score from -1.0 (very negative) to 1.0 (very positive).
func aiAnalyzeSentiment(subject, description string) (float64, string) {
	text := subject + " " + description

	// Simple keyword-based sentiment heuristic
	negativeWords := []string{"urgent", "broken", "error", "fail", "crash", "bug", "issue", "problem", "critical", "down", "lost", "cannot", "not working", "stuck", "blocked"}
	positiveWords := []string{"great", "thanks", "helpful", "appreciate", "good", "excellent", "love", "awesome", "perfect", "smooth"}

	negCount := 0
	posCount := 0

	lower := ""
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}

	for _, w := range negativeWords {
		if len(lower) >= len(w) {
			for i := 0; i <= len(lower)-len(w); i++ {
				if lower[i:i+len(w)] == w {
					negCount++
					break
				}
			}
		}
	}
	for _, w := range positiveWords {
		if len(lower) >= len(w) {
			for i := 0; i <= len(lower)-len(w); i++ {
				if lower[i:i+len(w)] == w {
					posCount++
					break
				}
			}
		}
	}

	// Calculate sentiment: -1 to 1 range
	total := negCount + posCount
	var score float64
	if total == 0 {
		score = 0.1 + rand.Float64()*0.3 // neutral-positive 0.1–0.4
	} else {
		score = (float64(posCount) - float64(negCount)) / float64(total)
		// Add slight randomness
		score += (rand.Float64() - 0.5) * 0.2
	}

	// Clamp to [-1, 1]
	if score > 1.0 {
		score = 1.0
	}
	if score < -1.0 {
		score = -1.0
	}

	// Generate suggested response based on sentiment
	var response string
	if score < -0.3 {
		response = "Thank you for reaching out. I understand this is frustrating. Our team is prioritizing your issue and will respond within 2 hours. In the meantime, could you provide any additional details or screenshots?"
	} else if score < 0.3 {
		response = "Thank you for contacting support. We've received your ticket and will review it shortly. A team member will follow up within 4 business hours."
	} else {
		response = "Thanks for your message! We're glad to hear from you. We'll review your request and get back to you within 8 business hours. Have a great day!"
	}

	return score, response
}

// aiDeterminePriority maps sentiment score to ticket priority.
func aiDeterminePriority(sentiment float64) string {
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

// aiAnalyzeContract simulates AI-based contract risk analysis.
// In production this would call an external AI/ML service.
func aiAnalyzeContract(contractText string) (float64, []FlaggedClause) {
	var clauses []FlaggedClause
	riskScore := 0.0

	// Simple keyword-based heuristic analysis
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

	for keyword, clause := range highRiskPatterns {
		if len(contractText) > 0 {
			// Check if keyword appears somewhere in the text (case-insensitive approximate)
			match := false
			lower := ""
			for _, r := range contractText {
				if r >= 'A' && r <= 'Z' {
					lower += string(r + 32)
				} else {
					lower += string(r)
				}
			}
			if len(lower) >= len(keyword) {
				for i := 0; i <= len(lower)-len(keyword); i++ {
					if lower[i:i+len(keyword)] == keyword {
						match = true
						break
					}
				}
			}
			if match {
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
	}

	// Add baseline risk
	if len(clauses) == 0 {
		riskScore = 5 + rand.Float64()*10 // 5–15 for clean contracts
	} else {
		riskScore += rand.Float64() * 10
	}

	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore, clauses
}

// aiSegmentAudience simulates AI-based audience segmentation for a campaign.
func aiSegmentAudience(channel string) string {
	segments := map[string]string{
		"email":  `{"criteria": "contacts with open rate > 30% in last 90 days", "estimated_size": 1250}`,
		"sms":    `{"criteria": "contacts with mobile phone and opted-in for SMS", "estimated_size": 840}`,
		"social": `{"criteria": "contacts who engaged with brand posts in last 60 days", "estimated_size": 2100}`,
		"push":   `{"criteria": "contacts with app installed and notifications enabled", "estimated_size": 670}`,
		"in_app": `{"criteria": "active users with at least 3 sessions in last 30 days", "estimated_size": 980}`,
	}
	if s, ok := segments[channel]; ok {
		return s
	}
	return `{"criteria": "all contacts", "estimated_size": 500}`
}

// aiEstimateReach simulates AI estimation of campaign reach.
func aiEstimateReach(channel string) int {
	baseReach := map[string]int{
		"email":  5000,
		"sms":    3000,
		"social": 15000,
		"push":   2000,
		"in_app": 4000,
	}
	reach, ok := baseReach[channel]
	if !ok {
		reach = 1000
	}
	// Add some randomness (±20%)
	reach += int(float64(reach) * (rand.Float64()*0.4 - 0.2))
	return reach
}

// Domain errors
var (
	ErrQuoteNotFound    = errors.New("quote not found")
	ErrQuoteNotInOrg    = errors.New("quote does not belong to this organization")
	ErrContactNotFound  = errors.New("contact not found")
	ErrContactNotInOrg  = errors.New("contact does not belong to this organization")
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrCampaignNotInOrg = errors.New("campaign does not belong to this organization")
)
