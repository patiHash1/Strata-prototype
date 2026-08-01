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

// Domain errors
var (
	ErrQuoteNotFound = errors.New("quote not found")
	ErrQuoteNotInOrg = errors.New("quote does not belong to this organization")
)
