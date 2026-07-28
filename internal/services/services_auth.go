package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Claims represents the JWT payload for authenticated requests.
type Claims struct {
	UserID      string   `json:"user_id"`
	OrgID       string   `json:"org_id"`
	RoleID      string   `json:"role_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// AuthService handles authentication, JWT, and password operations.
type AuthService struct {
	secret string
	issuer string
}

// NewAuthService creates an auth service with the given signing key and issuer.
func NewAuthService(secret, issuer string) *AuthService {
	return &AuthService{secret: secret, issuer: issuer}
}

// CreateToken generates a signed JWT for the given identity.
func (s *AuthService) CreateToken(userID, orgID, roleID uuid.UUID, permissions []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:      userID.String(),
		OrgID:       orgID.String(),
		RoleID:      roleID.String(),
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

// ValidateToken parses and validates a JWT string, returning the claims.
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// HashPassword returns a bcrypt hash of the given password.
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword compares a bcrypt hash with a plaintext password.
func (s *AuthService) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateRefreshToken creates a simple opaque refresh token.
func (s *AuthService) GenerateRefreshToken() string {
	return uuid.NewString() + uuid.NewString()
}
