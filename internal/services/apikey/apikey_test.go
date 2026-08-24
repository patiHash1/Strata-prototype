package apikey

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeRepo is a hand-rolled in-memory APIKeyRepository for service tests.
type fakeRepo struct {
	records map[string]*APIKeyRecord // key_prefix -> record
}

func (f *fakeRepo) GetAPIKeyByPrefix(_ context.Context, prefix string) (*APIKeyRecord, error) {
	return f.records[prefix], nil
}

func TestValidateAPIKeyInvalidFormat(t *testing.T) {
	svc := NewAPIKeyService(nil, nil, WithRepo(&fakeRepo{}))
	if _, _, err := svc.ValidateAPIKey(context.Background(), "not-a-strata-key"); !errors.Is(err, ErrInvalid) {
		t.Errorf("got %v, want ErrInvalid", err)
	}
}

func TestValidateAPIKeyUnknownPrefix(t *testing.T) {
	svc := NewAPIKeyService(nil, nil, WithRepo(&fakeRepo{records: map[string]*APIKeyRecord{}}))
	if _, _, err := svc.ValidateAPIKey(context.Background(), "strata_aaaaaaaaaaaa_zzz"); !errors.Is(err, ErrInvalid) {
		t.Errorf("got %v, want ErrInvalid", err)
	}
}

func TestValidateAPIKeyBadHash(t *testing.T) {
	orgID := uuid.New()
	r := &fakeRepo{records: map[string]*APIKeyRecord{
		"aaaaaaaaaaaa": {OrgID: orgID, KeyHash: "hash-of-other", Scopes: []string{"scope-a"}},
	}}
	// verify always fails.
	svc := NewAPIKeyService(nil, func(hash, v string) bool { return false }, WithRepo(r))
	if _, _, err := svc.ValidateAPIKey(context.Background(), "strata_aaaaaaaaaaaa_zzz"); !errors.Is(err, ErrInvalid) {
		t.Errorf("got %v, want ErrInvalid", err)
	}
}

func TestValidateAPIKeySuccess(t *testing.T) {
	orgID := uuid.New()
	scopes := []string{"fleet.telematics.ingest"}
	r := &fakeRepo{records: map[string]*APIKeyRecord{
		"aaaaaaaaaaaa": {OrgID: orgID, KeyHash: "x", Scopes: scopes},
	}}
	// verify always succeeds.
	svc := NewAPIKeyService(nil, func(hash, v string) bool { return true }, WithRepo(r))

	gotOrg, gotScopes, err := svc.ValidateAPIKey(context.Background(), "strata_aaaaaaaaaaaa_rest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOrg != orgID {
		t.Errorf("org = %s, want %s", gotOrg, orgID)
	}
	if len(gotScopes) != 1 || gotScopes[0] != scopes[0] {
		t.Errorf("scopes = %v, want %v", gotScopes, scopes)
	}
}

func TestValidateAPIKeyNilScopesNormalized(t *testing.T) {
	r := &fakeRepo{records: map[string]*APIKeyRecord{
		"aaaaaaaaaaaa": {OrgID: uuid.New(), KeyHash: "x"},
	}}
	svc := NewAPIKeyService(nil, func(hash, v string) bool { return true }, WithRepo(r))
	_, scopes, err := svc.ValidateAPIKey(context.Background(), "strata_aaaaaaaaaaaa_raw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scopes == nil || len(scopes) != 0 {
		t.Errorf("scopes = %#v, want empty non-nil slice", scopes)
	}
}
