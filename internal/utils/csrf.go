package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const csrfCookieName = "strata_csrf"
const csrfFieldName = "csrf_token"
const csrfTokenBytes = 32

// GenerateCSRFToken creates a cryptographically random CSRF token.
func GenerateCSRFToken() (string, error) {
	buf := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// SetCSRFCookie writes the CSRF token as a readable cookie (not HttpOnly,
// because the browser JS/templ needs to read it for form embedding).
func SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((15 * time.Minute).Seconds()),
		HttpOnly: false, // form field reads it via templ
		SameSite: http.SameSiteStrictMode,
	})
}

// ValidateCSRF checks that the CSRF token from the form body matches the
// CSRF cookie. Returns true if valid.
func ValidateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	// The token comes from the form body.
	formToken := r.FormValue(csrfFieldName)
	if formToken == "" {
		return false
	}

	return cookie.Value == formToken
}

// CSRFTokenFromCookie extracts the CSRF token from the request cookie.
// Returns empty string if not present.
func CSRFTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
