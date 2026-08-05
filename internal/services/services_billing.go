package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionStatus string

const (
	SubActive   SubscriptionStatus = "active"
	SubPastDue  SubscriptionStatus = "past_due"
	SubCanceled SubscriptionStatus = "canceled"
	SubTrialing SubscriptionStatus = "trialing"
)

// SubscriptionPlan represents a billing plan.
type SubscriptionPlan struct {
	ID                   uuid.UUID `json:"id"`
	PlanCode             string    `json:"plan_code"`
	Name                 string    `json:"name"`
	MaxTenantsLimit      int       `json:"max_tenants_limit"`
	MaxVehiclesLimit     int       `json:"max_vehicles_limit"`
	MaxAICreditsPerMonth int       `json:"max_ai_credits_per_month"`
	MonthlyPrice         float64   `json:"monthly_price"`
}

type Subscription struct {
	ID                   uuid.UUID          `json:"id"`
	OrgID                uuid.UUID          `json:"org_id"`
	PlanID               uuid.UUID          `json:"plan_id"`
	Status               SubscriptionStatus `json:"status"`
	StripeCustomerID     *string            `json:"-"`
	StripeSubscriptionID *string            `json:"-"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CreatedAt            time.Time          `json:"created_at"`
}

type billingRepository struct {
	pool *pgxpool.Pool
}

func newBillingRepository(pool *pgxpool.Pool) *billingRepository {
	return &billingRepository{pool: pool}
}

func (r *billingRepository) GetPlanByCode(ctx context.Context, planCode string) (*SubscriptionPlan, error) {
	p := &SubscriptionPlan{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, plan_code, name, max_tenants_limit, max_vehicles_limit, max_ai_credits_per_month, monthly_price
		FROM subscription_plans WHERE plan_code = $1
	`, planCode).Scan(&p.ID, &p.PlanCode, &p.Name, &p.MaxTenantsLimit, &p.MaxVehiclesLimit, &p.MaxAICreditsPerMonth, &p.MonthlyPrice)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *billingRepository) Create(ctx context.Context, sub *Subscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	if sub.Status == "" {
		sub.Status = SubTrialing
	}
	if sub.CurrentPeriodStart.IsZero() {
		sub.CurrentPeriodStart = time.Now()
	}
	if sub.CurrentPeriodEnd.IsZero() {
		sub.CurrentPeriodEnd = sub.CurrentPeriodStart.AddDate(0, 1, 0)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (id, org_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_start, current_period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sub.ID, sub.OrgID, sub.PlanID, sub.Status, sub.StripeCustomerID, sub.StripeSubscriptionID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CreatedAt)
	return err
}

func (r *billingRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*Subscription, error) {
	sub := &Subscription{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_start, current_period_end, created_at
		FROM subscriptions WHERE org_id = $1
	`, orgID).Scan(&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.StripeCustomerID, &sub.StripeSubscriptionID, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return sub, err
}

func (r *billingRepository) UpdatePlan(ctx context.Context, id uuid.UUID, planID uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions SET plan_id = $1, current_period_start = $2, current_period_end = $3 WHERE id = $4
	`, planID, now, now.AddDate(0, 1, 0), id)
	return err
}

// ---- Service ----

type BillingService struct {
	repo *billingRepository
}

func NewBillingService(pool *pgxpool.Pool) *BillingService {
	return &BillingService{repo: newBillingRepository(pool)}
}

func (s *BillingService) CreateOrUpgrade(ctx context.Context, orgID uuid.UUID, planCode string) (*Subscription, error) {
	plan, err := s.repo.GetPlanByCode(ctx, planCode)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("plan not found")
	}

	existing, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.PlanID = plan.ID
		existing.Status = SubActive
		if err := s.repo.UpdatePlan(ctx, existing.ID, plan.ID); err != nil {
			return nil, err
		}
		return existing, nil
	}

	sub := &Subscription{
		OrgID:  orgID,
		PlanID: plan.ID,
		Status: SubActive,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *BillingService) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*Subscription, error) {
	return s.repo.GetByOrgID(ctx, orgID)
}

var (
	ErrSubNotFound = errors.New("subscription not found")
)
