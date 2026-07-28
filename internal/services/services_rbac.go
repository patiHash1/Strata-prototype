package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type Role struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	IsSystemDefault bool      `json:"is_system_default"`
}

type Permission struct {
	ID            uuid.UUID `json:"id"`
	PermissionKey string    `json:"permission_key"`
	Module        string    `json:"module"`
	Description   *string   `json:"description,omitempty"`
}

// ---- Repository ----

type rbacRepository struct {
	pool *pgxpool.Pool
}

func newRBACRepository(pool *pgxpool.Pool) *rbacRepository {
	return &rbacRepository{pool: pool}
}

func (r *rbacRepository) CreateRole(ctx context.Context, role *Role) error {
	role.ID = uuid.New()
	if role.IsSystemDefault {
		role.IsSystemDefault = false
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO roles (id, org_id, name, description, is_system_default)
		VALUES ($1, $2, $3, $4, $5)
	`, role.ID, role.OrgID, role.Name, role.Description, role.IsSystemDefault)
	return err
}

func (r *rbacRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	role := &Role{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, is_system_default FROM roles WHERE id = $1
	`, id).Scan(&role.ID, &role.OrgID, &role.Name, &role.Description, &role.IsSystemDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return role, err
}

func (r *rbacRepository) ListRolesByOrg(ctx context.Context, orgID uuid.UUID) ([]Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, description, is_system_default FROM roles WHERE org_id = $1
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.OrgID, &role.Name, &role.Description, &role.IsSystemDefault); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *rbacRepository) AssignPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	for _, pid := range permissionIDs {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, roleID, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *rbacRepository) GetPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.permission_key, p.module, p.description
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.PermissionKey, &p.Module, &p.Description); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *rbacRepository) GetPermissionKeysByRole(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.permission_key
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, rows.Err()
}

// ---- Service ----

type RBACService struct {
	repo *rbacRepository
}

func NewRBACService(pool *pgxpool.Pool) *RBACService {
	return &RBACService{repo: newRBACRepository(pool)}
}

func (s *RBACService) CreateRole(ctx context.Context, orgID uuid.UUID, name string, description *string, permissionIDs []uuid.UUID) (*Role, error) {
	role := &Role{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	if len(permissionIDs) > 0 {
		if err := s.repo.AssignPermissions(ctx, role.ID, permissionIDs); err != nil {
			return nil, err
		}
	}
	return role, nil
}

func (s *RBACService) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	return s.repo.GetRoleByID(ctx, id)
}

func (s *RBACService) ListRolesByOrg(ctx context.Context, orgID uuid.UUID) ([]Role, error) {
	return s.repo.ListRolesByOrg(ctx, orgID)
}

func (s *RBACService) GetPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	return s.repo.GetPermissionsByRole(ctx, roleID)
}

func (s *RBACService) GetPermissionKeysByRole(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	return s.repo.GetPermissionKeysByRole(ctx, roleID)
}

// Pre-defined permission keys matching the API spec requirements.
const (
	PermUsersInvite   = "users.invite"
	PermRBACManage    = "rbac.manage"
	PermAPIKeysManage = "apikeys.manage"
	PermBillingManage = "billing.manage"
)
