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

// SetCSRFCookie writes the CSRF token as an HttpOnly cookie.
// The token is also embedded in the form via templ, so the browser
// does not need JS access to the cookie value.
func SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((15 * time.Minute).Seconds()),
		HttpOnly: true,
		// NOTE: Secure is intentionally false here because the server may
		// run behind a TLS-terminating proxy on plain HTTP locally. In a
		// production deployment without a proxy, set Secure based on
		// whether the request arrived over HTTPS.
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
