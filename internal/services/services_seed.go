package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/logger"
)

// SuperAdminOrgSlug is the reserved slug for the system-wide super-admin org.
const SuperAdminOrgSlug = "strata-system"

// SeedService handles idempotent data seeding operations.
type SeedService struct {
	pool    *pgxpool.Pool
	authSvc *AuthService
	userSvc *UserService
	orgSvc  *OrgService
	rbacSvc *RBACService
}

// NewSeedService creates a new SeedService.
func NewSeedService(pool *pgxpool.Pool, authSvc *AuthService, userSvc *UserService, orgSvc *OrgService, rbacSvc *RBACService) *SeedService {
	return &SeedService{pool: pool, authSvc: authSvc, userSvc: userSvc, orgSvc: orgSvc, rbacSvc: rbacSvc}
}

// SeedSuperAdmin creates the default super-admin organization, role, and user
// if they do not already exist.
func (s *SeedService) SeedSuperAdmin(ctx context.Context, email, password string) error {
	// Check if super-admin org already exists.
	org, err := s.orgSvc.GetByDomainSlug(ctx, SuperAdminOrgSlug)
	if err != nil {
		logger.Warn("could not check for existing super-admin org", slog.String("error", err.Error()))
	}
	if org == nil {
		org, err = s.orgSvc.Create(ctx, SuperAdminOrgSlug, "Strata System")
		if err != nil {
			return err
		}
		logger.Info("created super-admin org", slog.String("org_id", org.ID.String()))
	}

	// Check if super-admin user already exists.
	user, _ := s.userSvc.GetByEmail(ctx, email)
	if user == nil {
		hash, err := s.authSvc.HashPassword(password)
		if err != nil {
			return err
		}
		user, err = s.userSvc.Create(ctx, email, hash, "Super Admin")
		if err != nil {
			return err
		}
		logger.Info("created super-admin user", slog.String("user_id", user.ID.String()))
	}

	// Check if super-admin role already exists in the org.
	roles, err := s.rbacSvc.ListRolesByOrg(ctx, org.ID)
	if err != nil {
		return err
	}
	var superAdminRole *Role
	for i := range roles {
		if roles[i].Name == "Super Admin" {
			superAdminRole = &roles[i]
			break
		}
	}
	if superAdminRole == nil {
		permID, err := s.rbacSvc.GetPermissionIDByKey(ctx, PermSuperAdmin)
		if err != nil {
			return err
		}
		var permIDs []uuid.UUID
		if permID != uuid.Nil {
			permIDs = []uuid.UUID{permID}
		}
		superAdminRole, err = s.rbacSvc.CreateRole(ctx, org.ID, "Super Admin", nil, permIDs)
		if err != nil {
			return err
		}
		logger.Info("created super-admin role", slog.String("role_id", superAdminRole.ID.String()))
	}

	// Check if user is already a member of the org.
	member, _ := s.userSvc.GetMember(ctx, org.ID, user.ID)
	if member == nil {
		if err := s.userSvc.AddMember(ctx, &OrganizationMember{
			OrgID:    org.ID,
			UserID:   user.ID,
			RoleID:   superAdminRole.ID,
			IsActive: true,
		}); err != nil {
			return err
		}
		logger.Info("added super-admin user to org")
	}

	return nil
}
