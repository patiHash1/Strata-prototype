// Package apikey owns API-key authentication: prefix extraction, prefix
// lookup, and bcrypt verification. Previously this lived inside the
// SupplyChain domain; it is an auth concern, so it gets its own module behind
// a consumer-side seam.
package apikey

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalid is returned when an API key is malformed, unknown, or fails
// verification.
var ErrInvalid = errors.New("invalid or expired API key")

// APIKeyRecord is a stored API key row used for validation.
type APIKeyRecord struct {
	OrgID     uuid.UUID
	KeyPrefix string
	KeyHash   string
	Scopes    []string
}

// APIKeyRepository is the seam APIKeyService depends on. Declared
// consumer-side, minimal — only the lookup the validator needs; the pgx pool
// adapter satisfies it implicitly.
type APIKeyRepository interface {
	GetAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKeyRecord, error)
}

// VerifyFunc compares a bcrypt hash against a plaintext value.
type VerifyFunc func(hash, value string) bool

// bcryptVerify is the default verifier.
func bcryptVerify(hash, value string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value)) == nil
}

// APIKeyService validates API keys. It keeps the (uuid.UUID, []string, error)
// signature so utils.RequireAPIKey's structural interface consumes it.
type APIKeyService struct {
	repo   APIKeyRepository
	verify VerifyFunc
}

// Option customises an APIKeyService at construction.
type Option func(*APIKeyService)

// WithRepo substitutes the repository adapter (in-memory for tests).
func WithRepo(repo APIKeyRepository) Option {
	return func(s *APIKeyService) { s.repo = repo }
}

// WithVerify substitutes the hash verifier (tests can accept any key).
func WithVerify(verify VerifyFunc) Option {
	return func(s *APIKeyService) { s.verify = verify }
}

// NewAPIKeyService builds an APIKeyService from a shared pool and a verifier.
// verify defaults to bcrypt when nil.
func NewAPIKeyService(pool *pgxpool.Pool, verify VerifyFunc, opts ...Option) *APIKeyService {
	if verify == nil {
		verify = bcryptVerify
	}
	s := &APIKeyService{repo: newRepo(pool), verify: verify}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ValidateAPIKey checks the raw key against the stored bcrypt hash and returns
// the owning org ID and scopes on success.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, rawKey string) (uuid.UUID, []string, error) {
	prefix := extractAPIKeyPrefix(rawKey)
	if prefix == "" {
		return uuid.Nil, nil, ErrInvalid
	}

	k, err := s.repo.GetAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if k == nil {
		return uuid.Nil, nil, ErrInvalid
	}

	if !s.verify(k.KeyHash, rawKey) {
		return uuid.Nil, nil, ErrInvalid
	}

	if k.Scopes == nil {
		k.Scopes = []string{}
	}
	return k.OrgID, k.Scopes, nil
}

// extractAPIKeyPrefix extracts the prefix from "strata_<12-char-prefix>_<rest>".
// Returns empty string if the format is invalid.
func extractAPIKeyPrefix(rawKey string) string {
	const prefix = "strata_"
	if !strings.HasPrefix(rawKey, prefix) {
		return ""
	}
	rest := rawKey[len(prefix):]
	if len(rest) < 12 {
		return ""
	}
	return rest[:12]
}

// repo is the pgx-backed adapter.
type repo struct {
	pool *pgxpool.Pool
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{pool: pool}
}

func (r *repo) GetAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKeyRecord, error) {
	k := &APIKeyRecord{}
	err := r.pool.QueryRow(ctx, `
		SELECT org_id, key_prefix, key_hash, COALESCE(scopes, '{}') FROM api_keys
		WHERE key_prefix = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`, prefix).Scan(&k.OrgID, &k.KeyPrefix, &k.KeyHash, &k.Scopes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return k, err
}
