package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegistrationResult holds the outcome of a successful registration.
type RegistrationResult struct {
	Org  *Organization
	User *User
	Role *Role
}

// RegistrationService handles atomic multi-step registration operations.
type RegistrationService struct {
	pool *pgxpool.Pool
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(pool *pgxpool.Pool) *RegistrationService {
	return &RegistrationService{pool: pool}
}

// RegisterWithOwner atomically creates an organization, owner user, admin role,
// and membership in a single database transaction. On any failure the entire
// operation is rolled back.
func (s *RegistrationService) RegisterWithOwner(
	ctx context.Context,
	domainSlug, companyName, email, passwordHash, fullName string,
) (*RegistrationResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Create organization
	orgID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (id, domain_slug, company_name, default_currency, timezone, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'USD', 'UTC', 'active', NOW(), NOW())
	`, orgID, domainSlug, companyName)
	if err != nil {
		return nil, translateOrgInsertErr(err)
	}

	// 2. Create user
	userID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, userID, email, passwordHash, fullName)
	if err != nil {
		return nil, translateUserInsertErr(err)
	}

	// 3. Create admin role
	roleID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO roles (id, org_id, name, is_system_default)
		VALUES ($1, $2, 'Admin', false)
	`, roleID, orgID)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	// 4. Add membership
	memberID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (id, org_id, user_id, role_id, is_active, joined_at)
		VALUES ($1, $2, $3, $4, true, NOW())
	`, memberID, orgID, userID, roleID)
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &RegistrationResult{
		Org: &Organization{
			ID:              orgID,
			DomainSlug:      domainSlug,
			CompanyName:     companyName,
			DefaultCurrency: "USD",
			Timezone:        "UTC",
			Status:          OrgActive,
		},
		User: &User{
			ID:       userID,
			Email:    email,
			FullName: fullName,
		},
		Role: &Role{
			ID:    roleID,
			OrgID: orgID,
			Name:  "Admin",
		},
	}, nil
}

// translateOrgInsertErr converts a postgres unique_violation on domain_slug
// into the application-level ErrOrgAlreadyExists error.
func translateOrgInsertErr(err error) error {
	if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "domain_slug") {
		return ErrOrgAlreadyExists
	}
	return fmt.Errorf("create org: %w", err)
}

// translateUserInsertErr converts a postgres unique_violation on email
// into the application-level ErrEmailAlreadyExists error.
func translateUserInsertErr(err error) error {
	if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "email") {
		return ErrEmailAlreadyExists
	}
	return fmt.Errorf("create user: %w", err)
}
