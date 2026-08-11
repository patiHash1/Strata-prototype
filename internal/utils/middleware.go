package utils

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
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

// LoggingMiddleware logs every incoming request and pushes latency records
// into the SuperAdminService for metrics aggregation.
func LoggingMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(lw, r)

			latency := time.Since(start)

			log.Printf(
				"%s %s %d %s",
				r.Method,
				r.URL.Path,
				lw.status,
				latency.Round(time.Microsecond),
			)

			if adminSvc != nil {
				adminSvc.RecordHTTPLatency(services.HTTPLatencyRecord{
					Path:       r.URL.Path,
					Method:     r.Method,
					StatusCode: lw.status,
					Latency:    latency,
					Module:     extractModule(r.URL.Path),
					Timestamp:  start,
				})
			}
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it supports Flusher.
func (lw *loggingResponseWriter) Flush() {
	if f, ok := lw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RecoveryMiddleware catches panics and returns 500.
// If an adminSvc is provided via the closure, panic traces are recorded
// into the ring buffer and persisted asynchronously.
func RecoveryMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					log.Printf("PANIC: %v\n%s", rec, stack)

					if adminSvc != nil {
						adminSvc.RecordPanic("system", fmt.Sprintf("%v", rec), stack, http.StatusInternalServerError)
					}

					WriteErr(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers based on a configurable list of allowed origins.
// If allowedOrigins is empty, no Access-Control-Allow-Origin header is set,
// which effectively blocks all cross-origin requests.
// Credentials (cookies) are supported for matching origins.
func CORSMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	var origins []string
	if allowedOrigins != "" {
		for _, o := range strings.Split(allowedOrigins, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && len(origins) > 0 {
				for _, allowed := range origins {
					if origin == allowed {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
						w.Header().Set("Access-Control-Allow-Credentials", "true")
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Domain, X-API-Key")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth is middleware that validates a Bearer JWT and injects claims.
func RequireAuth(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				WriteErr(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

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

// RequireAuthCookie validates a JWT from either the Authorization header
// or the strata_token cookie. For browser (HTML) requests without a valid
// token, it redirects to /api/v1/super-admin/login instead of returning JSON.
// If issuedAfter is non-zero, tokens with IssuedAt before that time are rejected
// (used to invalidate pre-existing tokens after a server restart).
func RequireAuthCookie(authSvc *services.AuthService, issuedAfter time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				if acceptsHTML(r) {
					http.Redirect(w, r, "/api/v1/super-admin/login", http.StatusFound)
					return
				}
				WriteErr(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				if acceptsHTML(r) {
					http.Redirect(w, r, "/api/v1/super-admin/login", http.StatusFound)
					return
				}
				WriteErr(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Reject tokens issued before the server started (e.g. after restart).
			if !issuedAfter.IsZero() && claims.IssuedAt != nil && claims.IssuedAt.Time.Before(issuedAfter) {
				if acceptsHTML(r) {
					http.Redirect(w, r, "/api/v1/super-admin/login", http.StatusFound)
					return
				}
				WriteErr(w, http.StatusUnauthorized, "session expired, please log in again")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken pulls a JWT from the Authorization header or the strata_token cookie.
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if cookie, err := r.Cookie("strata_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

// acceptsHTML returns true if the request's Accept header prefers text/html.
func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
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

// MaxBodySizeMiddleware limits the maximum request body size to prevent
// memory exhaustion from oversized payloads. The limit is applied per-request.
func MaxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1 MiB default
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PartitionedMaintenanceMiddleware checks the local in-memory maintenance cache
// and blocks requests targeting modules/tenants/features under maintenance.
// Routes matching /api/v1/super-admin/* and users with PermSuperAdmin bypass checks.
func PartitionedMaintenanceMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass: super-admin routes are always accessible.
			if strings.HasPrefix(r.URL.Path, "/api/v1/super-admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Bypass: users with PermSuperAdmin.
			if claims := GetClaims(r); claims != nil {
				for _, p := range claims.Permissions {
					if p == services.PermSuperAdmin {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			if adminSvc == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract module from path (e.g., /api/v1/crm/... → crm).
			module := extractModule(r.URL.Path)

			// Check module-level maintenance.
			if rule, ok := adminSvc.IsUnderMaintenance("module", module); ok {
				http.Error(w, fmt.Sprintf(`{"error":"service under maintenance","reason":"%s"}`, rule.Reason),
					http.StatusServiceUnavailable)
				return
			}

			// Check tenant-level maintenance.
			if claims := GetClaims(r); claims != nil && claims.OrgID != "" {
				if rule, ok := adminSvc.IsUnderMaintenance("tenant_id", claims.OrgID); ok {
					http.Error(w, fmt.Sprintf(`{"error":"organization under maintenance","reason":"%s"}`, rule.Reason),
						http.StatusServiceUnavailable)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractModule extracts the module name from an API path.
// e.g., /api/v1/crm/leads → crm, /api/v1/accounting/journal-entries → accounting.
func extractModule(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// Expected: api, v1, <module>, ...
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	return "unknown"
}
