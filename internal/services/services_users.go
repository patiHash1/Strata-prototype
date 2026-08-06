package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	PhoneNumber  *string   `json:"phone_number,omitempty"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	MFASecret    *string   `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type OrganizationMember struct {
	ID       uuid.UUID `json:"id"`
	OrgID    uuid.UUID `json:"org_id"`
	UserID   uuid.UUID `json:"user_id"`
	RoleID   uuid.UUID `json:"role_id"`
	IsActive bool      `json:"is_active"`
	JoinedAt time.Time `json:"joined_at"`
}

// ---- Repository ----

type userRepository struct {
	pool *pgxpool.Pool
}

func newUserRepository(pool *pgxpool.Pool) *userRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, u *User) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, phone_number, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, u.ID, u.Email, u.PasswordHash, u.FullName, u.PhoneNumber, u.CreatedAt)
	return err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, phone_number, mfa_enabled, mfa_secret, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.PhoneNumber, &u.MFAEnabled, &u.MFASecret, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, phone_number, mfa_enabled, mfa_secret, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.PhoneNumber, &u.MFAEnabled, &u.MFASecret, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) AddMember(ctx context.Context, m *OrganizationMember) error {
	m.ID = uuid.New()
	m.JoinedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO organization_members (id, org_id, user_id, role_id, is_active, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.OrgID, m.UserID, m.RoleID, m.IsActive, m.JoinedAt)
	return err
}

func (r *userRepository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error) {
	m := &OrganizationMember{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, role_id, is_active, joined_at
		FROM organization_members WHERE org_id = $1 AND user_id = $2
	`, orgID, userID).Scan(&m.ID, &m.OrgID, &m.UserID, &m.RoleID, &m.IsActive, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *userRepository) UpdateMemberRole(ctx context.Context, memberID, roleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE organization_members SET role_id = $1 WHERE id = $2`, roleID, memberID)
	return err
}

func (r *userRepository) DeactivateMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE organization_members SET is_active = false WHERE id = $1`, memberID)
	return err
}

func (r *userRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM organization_members WHERE id = $1`, memberID)
	return err
}

func (r *userRepository) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*OrganizationMember, error) {
	m := &OrganizationMember{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, role_id, is_active, joined_at
		FROM organization_members WHERE id = $1
	`, memberID).Scan(&m.ID, &m.OrgID, &m.UserID, &m.RoleID, &m.IsActive, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *userRepository) ListMembersByUser(ctx context.Context, userID uuid.UUID) ([]OrganizationMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, user_id, role_id, is_active, joined_at
		FROM organization_members WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []OrganizationMember
	for rows.Next() {
		var m OrganizationMember
		if err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.RoleID, &m.IsActive, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// ---- Service ----

type UserService struct {
	repo *userRepository
}

func NewUserService(pool *pgxpool.Pool) *UserService {
	return &UserService{repo: newUserRepository(pool)}
}

func (s *UserService) Create(ctx context.Context, email, passwordHash, fullName string) (*User, error) {
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	user := &User{
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) AddMember(ctx context.Context, m *OrganizationMember) error {
	return s.repo.AddMember(ctx, m)
}

func (s *UserService) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error) {
	return s.repo.GetMember(ctx, orgID, userID)
}

func (s *UserService) UpdateMemberRole(ctx context.Context, memberID, roleID uuid.UUID) error {
	return s.repo.UpdateMemberRole(ctx, memberID, roleID)
}

func (s *UserService) DeactivateMember(ctx context.Context, memberID uuid.UUID) error {
	return s.repo.DeactivateMember(ctx, memberID)
}

func (s *UserService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	return s.repo.RemoveMember(ctx, memberID)
}

func (s *UserService) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*OrganizationMember, error) {
	return s.repo.GetMemberByID(ctx, memberID)
}

func (s *UserService) ListMembersByUser(ctx context.Context, userID uuid.UUID) ([]OrganizationMember, error) {
	return s.repo.ListMembersByUser(ctx, userID)
}

// UpdateProfile allows a user to modify their own profile fields.
func (s *UserService) UpdateProfile(ctx context.Context, id uuid.UUID, fullName, email, phone *string) error {
	// Validate email uniqueness if changing.
	if email != nil {
		existing, err := s.repo.GetByEmail(ctx, *email)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != id {
			return ErrEmailAlreadyExists
		}
	}

	return s.repo.Update(ctx, id, fullName, email, phone)
}

// DeleteAccount removes the user record for the given ID.
func (s *UserService) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (r *userRepository) Update(ctx context.Context, id uuid.UUID, fullName *string, email *string, phone *string) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if fullName != nil {
		setClauses = append(setClauses, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, *fullName)
		argIdx++
	}
	if email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *email)
		argIdx++
	}
	if phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone_number = $%d", argIdx))
		args = append(args, *phone)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// Domain errors
var (
	ErrEmailAlreadyExists = errors.New("user with this email already exists")
	ErrMemberNotFound     = errors.New("organization member not found")
	ErrMemberNotInOrg     = errors.New("member does not belong to this organization")
)
