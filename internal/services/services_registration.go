package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RegistrationResult holds the outcome of a successful registration.
type RegistrationResult struct {
	Org  *Organization
	User *User
	Role *Role
}

// RegistrationService handles atomic multi-step registration operations. It
// composes the org/user/rbac repositories inside a single transaction rather
// than duplicating their SQL.
type RegistrationService struct {
	orgRepo  *orgRepository
	userRepo *userRepository
	rbacRepo *rbacRepository
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(pool *pgxpool.Pool) *RegistrationService {
	return &RegistrationService{
		orgRepo:  newOrgRepository(pool),
		userRepo: newUserRepository(pool),
		rbacRepo: newRBACRepository(pool),
	}
}

// RegisterWithOwner atomically creates an organization, owner user, admin role,
// and membership in a single database transaction. On any failure the entire
// operation is rolled back.
func (s *RegistrationService) RegisterWithOwner(
	ctx context.Context,
	domainSlug, companyName, email, passwordHash, fullName string,
) (*RegistrationResult, error) {
	tx, err := s.orgRepo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Create organization
	org := &Organization{
		DomainSlug:  domainSlug,
		CompanyName: companyName,
	}
	if err := s.orgRepo.CreateTx(ctx, tx, org); err != nil {
		return nil, translateOrgInsertErr(err)
	}

	// 2. Create user
	user := &User{
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
	}
	if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
		return nil, translateUserInsertErr(err)
	}

	// 3. Create admin role
	role := &Role{
		OrgID: org.ID,
		Name:  "Admin",
	}
	if err := s.rbacRepo.CreateRoleTx(ctx, tx, role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	// 4. Add membership
	member := &OrganizationMember{
		OrgID:    org.ID,
		UserID:   user.ID,
		RoleID:   role.ID,
		IsActive: true,
	}
	if err := s.userRepo.AddMemberTx(ctx, tx, member); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &RegistrationResult{Org: org, User: user, Role: role}, nil
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
