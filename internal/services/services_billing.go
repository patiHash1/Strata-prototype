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

type Subscription struct {
	ID               uuid.UUID          `json:"id"`
	OrgID            uuid.UUID          `json:"org_id"`
	PlanCode         string             `json:"plan_code"`
	Status           SubscriptionStatus `json:"status"`
	StripeCustomerID *string            `json:"-"`
	StripeSubID      *string            `json:"-"`
	CurrentPeriodEnd *time.Time         `json:"current_period_end,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type billingRepository struct {
	pool *pgxpool.Pool
}

func newBillingRepository(pool *pgxpool.Pool) *billingRepository {
	return &billingRepository{pool: pool}
}

func (r *billingRepository) Create(ctx context.Context, sub *Subscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = sub.CreatedAt
	if sub.Status == "" {
		sub.Status = SubTrialing
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (id, org_id, plan_code, status, stripe_customer_id, stripe_sub_id, current_period_end, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sub.ID, sub.OrgID, sub.PlanCode, sub.Status, sub.StripeCustomerID, sub.StripeSubID, sub.CurrentPeriodEnd, sub.CreatedAt, sub.UpdatedAt)
	return err
}

func (r *billingRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*Subscription, error) {
	sub := &Subscription{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, plan_code, status, stripe_customer_id, stripe_sub_id, current_period_end, created_at, updated_at
		FROM subscriptions WHERE org_id = $1
	`, orgID).Scan(&sub.ID, &sub.OrgID, &sub.PlanCode, &sub.Status, &sub.StripeCustomerID, &sub.StripeSubID, &sub.CurrentPeriodEnd, &sub.CreatedAt, &sub.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return sub, err
}

func (r *billingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status SubscriptionStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE subscriptions SET status = $1, updated_at = $2 WHERE id = $3`, status, time.Now(), id)
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
	existing, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.PlanCode = planCode
		existing.Status = SubActive
		existing.UpdatedAt = time.Now()
		if err := s.repo.UpdateStatus(ctx, existing.ID, SubActive); err != nil {
			return nil, err
		}
		return existing, nil
	}

	sub := &Subscription{
		OrgID:    orgID,
		PlanCode: planCode,
		Status:   SubActive,
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
