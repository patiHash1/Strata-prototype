package utils

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
)

// contextKey is used for storing auth claims in request context.
type contextKey string

const claimsKey contextKey = "auth.claims"
const apiKeyClaimsKey contextKey = "apikey.claims"

// GetClaims extracts auth claims from the request context. Returns nil if absent.
func GetClaims(r *http.Request) *services.Claims {
	c, _ := r.Context().Value(claimsKey).(*services.Claims)
	return c
}

// APIKeyClaims holds the org identity extracted from a validated API key.
type APIKeyClaims struct {
	OrgID  string
	Scopes []string
}

// GetAPIKeyClaims extracts API key claims from the request context. Returns nil if absent.
func GetAPIKeyClaims(r *http.Request) *APIKeyClaims {
	c, _ := r.Context().Value(apiKeyClaimsKey).(*APIKeyClaims)
	return c
}

// LoggingMiddleware logs every incoming request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)

		log.Printf(
			"%s %s %d %s",
			r.Method,
			r.URL.Path,
			lw.status,
			time.Since(start).Round(time.Microsecond),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

// RecoveryMiddleware catches panics and returns 500.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC: %v", rec)
				WriteErr(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds permissive CORS headers.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Domain, X-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth is middleware that validates a Bearer JWT and injects claims.
func RequireAuth(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				WriteErr(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				WriteErr(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAPIKey returns middleware that validates an API key from the X-API-Key header
// and requires at least one of the specified scopes. Populates context with APIKeyClaims.
func RequireAPIKey(svc interface {
	ValidateAPIKey(ctx context.Context, keyHash string) (uuid.UUID, []string, error)
}, requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				WriteErr(w, http.StatusUnauthorized, "missing API key")
				return
			}

			// Hash is produced by bcrypt; for now we pass the raw key
			orgID, scopes, err := svc.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				WriteErr(w, http.StatusUnauthorized, "invalid or expired API key")
				return
			}

			// Check scope intersection
			if len(requiredScopes) > 0 {
				scopeSet := make(map[string]struct{}, len(scopes))
				for _, s := range scopes {
					scopeSet[s] = struct{}{}
				}
				authorized := false
				for _, rs := range requiredScopes {
					if _, ok := scopeSet[rs]; ok {
						authorized = true
						break
					}
				}
				if !authorized {
					WriteErr(w, http.StatusForbidden, "insufficient API key scopes")
					return
				}
			}

			ctx := context.WithValue(r.Context(), apiKeyClaimsKey, &APIKeyClaims{
				OrgID:  orgID.String(),
				Scopes: scopes,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns middleware that checks the authenticated user
// has at least one of the specified permissions. Must be used after RequireAuth.
func RequirePermission(perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(claimsKey).(*services.Claims)
			if !ok {
				WriteErr(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if len(claims.Permissions) == 0 {
				WriteErr(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			permSet := make(map[string]struct{}, len(claims.Permissions))
			for _, p := range claims.Permissions {
				permSet[p] = struct{}{}
			}

			for _, required := range perms {
				if _, ok := permSet[required]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			WriteErr(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}
