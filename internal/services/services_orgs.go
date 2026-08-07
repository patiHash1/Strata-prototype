package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type OrgStatus string

const (
	OrgActive              OrgStatus = "active"
	OrgSuspended           OrgStatus = "suspended"
	OrgPendingVerification OrgStatus = "pending_verification"
)

type Organization struct {
	ID              uuid.UUID `json:"id"`
	DomainSlug      string    `json:"domain_slug"`
	CustomDomain    *string   `json:"custom_domain,omitempty"`
	CompanyName     string    `json:"company_name"`
	DefaultCurrency string    `json:"default_currency"`
	Timezone        string    `json:"timezone"`
	Status          OrgStatus `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type OrganizationInvitation struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	Email      string     `json:"email"`
	RoleID     uuid.UUID  `json:"role_id"`
	Token      string     `json:"token"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedBy  *uuid.UUID `json:"created_by,omitempty"`
}

type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ---- Repository ----

type orgRepository struct {
	pool *pgxpool.Pool
}

func newOrgRepository(pool *pgxpool.Pool) *orgRepository {
	return &orgRepository{pool: pool}
}

func (r *orgRepository) Create(ctx context.Context, o *Organization) error {
	o.ID = uuid.New()
	o.CreatedAt = time.Now()
	o.UpdatedAt = o.CreatedAt
	if o.DefaultCurrency == "" {
		o.DefaultCurrency = "USD"
	}
	if o.Timezone == "" {
		o.Timezone = "UTC"
	}
	if o.Status == "" {
		o.Status = OrgActive
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO organizations (id, domain_slug, company_name, default_currency, timezone, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, o.ID, o.DomainSlug, o.CompanyName, o.DefaultCurrency, o.Timezone, o.Status, o.CreatedAt, o.UpdatedAt)
	return err
}

func (r *orgRepository) GetByDomainSlug(ctx context.Context, slug string) (*Organization, error) {
	o := &Organization{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, domain_slug, custom_domain, company_name, default_currency, timezone, status, created_at, updated_at
		FROM organizations WHERE domain_slug = $1
	`, slug).Scan(&o.ID, &o.DomainSlug, &o.CustomDomain, &o.CompanyName, &o.DefaultCurrency, &o.Timezone, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

func (r *orgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	o := &Organization{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, domain_slug, custom_domain, company_name, default_currency, timezone, status, created_at, updated_at
		FROM organizations WHERE id = $1
	`, id).Scan(&o.ID, &o.DomainSlug, &o.CustomDomain, &o.CompanyName, &o.DefaultCurrency, &o.Timezone, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

func (r *orgRepository) CreateInvitation(ctx context.Context, inv *OrganizationInvitation) error {
	inv.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO organization_invitations (id, org_id, email, role_id, token, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, inv.ID, inv.OrgID, inv.Email, inv.RoleID, inv.Token, inv.ExpiresAt, inv.CreatedBy)
	return err
}

func (r *orgRepository) CreateAPIKey(ctx context.Context, key *APIKey) error {
	key.ID = uuid.New()
	key.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO api_keys (id, org_id, name, key_hash, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, key.ID, key.OrgID, key.Name, key.KeyHash, key.Scopes, key.ExpiresAt, key.CreatedAt)
	return err
}

func (r *orgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status OrgStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE organizations SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	return err
}

func (r *orgRepository) ListAllOrgs(ctx context.Context, offset, limit int) ([]Organization, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, domain_slug, custom_domain, company_name, default_currency, timezone, status, created_at, updated_at
		FROM organizations ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.DomainSlug, &o.CustomDomain, &o.CompanyName,
			&o.DefaultCurrency, &o.Timezone, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, o)
	}
	return orgs, total, rows.Err()
}

// ---- Service ----

type OrgService struct {
	repo *orgRepository
}

func NewOrgService(pool *pgxpool.Pool) *OrgService {
	return &OrgService{repo: newOrgRepository(pool)}
}

func (s *OrgService) Create(ctx context.Context, domainSlug, companyName string) (*Organization, error) {
	existing, err := s.repo.GetByDomainSlug(ctx, domainSlug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrOrgAlreadyExists
	}
	org := &Organization{
		DomainSlug:  domainSlug,
		CompanyName: companyName,
	}
	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrgService) GetByDomainSlug(ctx context.Context, slug string) (*Organization, error) {
	return s.repo.GetByDomainSlug(ctx, slug)
}

func (s *OrgService) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrgService) CreateInvitation(ctx context.Context, inv *OrganizationInvitation) error {
	return s.repo.CreateInvitation(ctx, inv)
}

func (s *OrgService) CreateAPIKey(ctx context.Context, key *APIKey) error {
	return s.repo.CreateAPIKey(ctx, key)
}

// SuspendOrg sets an organization's status to suspended.
func (s *OrgService) SuspendOrg(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, OrgSuspended)
}

// ActivateOrg sets an organization's status to active.
func (s *OrgService) ActivateOrg(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, OrgActive)
}

// ListAllOrgs returns a paginated list of all organizations.
func (s *OrgService) ListAllOrgs(ctx context.Context, offset, limit int) ([]Organization, int, error) {
	return s.repo.ListAllOrgs(ctx, offset, limit)
}

var (
	ErrOrgAlreadyExists = errors.New("organization with this domain slug already exists")
)
