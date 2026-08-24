package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/patiHash1/Strata-prototype/internal/services"
)

// userFakeRepo is a hand-rolled in-memory UserRepository for service tests.
// It only implements the membership methods the deepened service paths use;
// the rest panic if called (they aren't exercised here).
type userFakeRepo struct {
	members map[uuid.UUID]*services.OrganizationMember
}

func newUserFakeRepo() *userFakeRepo {
	return &userFakeRepo{members: map[uuid.UUID]*services.OrganizationMember{}}
}

func (f *userFakeRepo) seed(m *services.OrganizationMember) {
	f.members[m.ID] = m
}

func (f *userFakeRepo) GetMemberByID(_ context.Context, id uuid.UUID) (*services.OrganizationMember, error) {
	return f.members[id], nil
}
func (f *userFakeRepo) UpdateMemberRole(_ context.Context, memberID, roleID uuid.UUID) error {
	if m, ok := f.members[memberID]; ok {
		m.RoleID = roleID
	}
	return nil
}
func (f *userFakeRepo) DeactivateMember(_ context.Context, memberID uuid.UUID) error {
	if m, ok := f.members[memberID]; ok {
		m.IsActive = false
	}
	return nil
}
func (f *userFakeRepo) RemoveMember(_ context.Context, memberID uuid.UUID) error {
	delete(f.members, memberID)
	return nil
}

// Unused interface methods — panic if hit.
func (f *userFakeRepo) Create(context.Context, *services.User) error { panic("not implemented") }
func (f *userFakeRepo) GetByEmail(context.Context, string) (*services.User, error) {
	panic("not implemented")
}
func (f *userFakeRepo) GetByID(context.Context, uuid.UUID) (*services.User, error) {
	panic("not implemented")
}
func (f *userFakeRepo) AddMember(context.Context, *services.OrganizationMember) error {
	panic("not implemented")
}
func (f *userFakeRepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*services.OrganizationMember, error) {
	panic("not implemented")
}
func (f *userFakeRepo) ListMembersByUser(context.Context, uuid.UUID, int, int) ([]services.OrganizationMember, int, error) {
	panic("not implemented")
}
func (f *userFakeRepo) Update(context.Context, uuid.UUID, *string, *string, *string) error {
	panic("not implemented")
}
func (f *userFakeRepo) Delete(context.Context, uuid.UUID) error { panic("not implemented") }
func (f *userFakeRepo) BanUser(context.Context, uuid.UUID, string) error {
	panic("not implemented")
}
func (f *userFakeRepo) UnbanUser(context.Context, uuid.UUID) error { panic("not implemented") }
func (f *userFakeRepo) ListAllUsers(context.Context, int, int) ([]services.User, int, error) {
	panic("not implemented")
}
func (f *userFakeRepo) CountActiveUsers(context.Context, time.Duration) (int, error) {
	panic("not implemented")
}
func (f *userFakeRepo) UpdateLastLoginAt(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func newTestUserSvc(repo *userFakeRepo) *services.UserService {
	return services.NewUserService(nil, services.WithUserRepo(repo))
}

func TestUpdateMemberRoleGuards(t *testing.T) {
	orgID := uuid.New()
	acting := uuid.New()
	other := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	repo := newUserFakeRepo()
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: other, IsActive: true})
	svc := newTestUserSvc(repo)

	// Not found.
	if err := svc.UpdateMemberRole(context.Background(), orgID, acting, uuid.New(), roleID); !errors.Is(err, services.ErrMemberNotFound) {
		t.Errorf("got %v, want ErrMemberNotFound", err)
	}

	// Not in org.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: uuid.New(), UserID: other})
	if err := svc.UpdateMemberRole(context.Background(), orgID, acting, memberID, roleID); !errors.Is(err, services.ErrMemberNotInOrg) {
		t.Errorf("got %v, want ErrMemberNotInOrg", err)
	}

	// Self-targeting.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: acting})
	if err := svc.UpdateMemberRole(context.Background(), orgID, acting, memberID, roleID); !errors.Is(err, services.ErrSelfChange) {
		t.Errorf("got %v, want ErrSelfChange", err)
	}

	// Happy path.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: other})
	if err := svc.UpdateMemberRole(context.Background(), orgID, acting, memberID, roleID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.members[memberID].RoleID; got != roleID {
		t.Errorf("role = %s, want %s", got, roleID)
	}
}

func TestDeactivateMemberGuards(t *testing.T) {
	orgID := uuid.New()
	acting := uuid.New()
	other := uuid.New()
	memberID := uuid.New()

	repo := newUserFakeRepo()
	svc := newTestUserSvc(repo)

	// Not found.
	if err := svc.DeactivateMember(context.Background(), orgID, acting, uuid.New()); !errors.Is(err, services.ErrMemberNotFound) {
		t.Errorf("got %v, want ErrMemberNotFound", err)
	}

	// Already deactivated.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: other, IsActive: false})
	if err := svc.DeactivateMember(context.Background(), orgID, acting, memberID); !errors.Is(err, services.ErrMemberAlreadyDeactivated) {
		t.Errorf("got %v, want ErrMemberAlreadyDeactivated", err)
	}

	// Happy path deactivates.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: other, IsActive: true})
	if err := svc.DeactivateMember(context.Background(), orgID, acting, memberID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.members[memberID].IsActive {
		t.Error("expected member to be deactivated")
	}
}

func TestRemoveMemberGuards(t *testing.T) {
	orgID := uuid.New()
	acting := uuid.New()
	other := uuid.New()
	memberID := uuid.New()

	repo := newUserFakeRepo()
	svc := newTestUserSvc(repo)

	// Self-removal blocked.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: acting})
	if err := svc.RemoveMember(context.Background(), orgID, acting, memberID); !errors.Is(err, services.ErrSelfChange) {
		t.Errorf("got %v, want ErrSelfChange", err)
	}

	// Happy path removes.
	repo.seed(&services.OrganizationMember{ID: memberID, OrgID: orgID, UserID: other})
	if err := svc.RemoveMember(context.Background(), orgID, acting, memberID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.members[memberID]; ok {
		t.Error("expected member to be removed")
	}
}
