package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
)

// TestStaticCSSHandler verifies that the embedded static file server
// serves /static/css/styles.css with a 200 OK status and the correct
// Content-Type header.
func TestStaticCSSHandler(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/css; charset=utf-8" {
		t.Errorf("expected Content-Type \"text/css; charset=utf-8\", got %q", ct)
	}
}

// TestStaticCSSNotFound verifies that requesting a non-existent static
// file returns 404.
func TestStaticCSSNotFound(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	req := httptest.NewRequest(http.MethodGet, "/static/css/nonexistent.css", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", resp.StatusCode)
	}
}

// TestSuperAdminLoginPage verifies that GET /api/v1/super-admin/login
// returns a 200 OK with an HTML login page.
func TestSuperAdminLoginPage(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/super-admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Assert HTML5 doctype.
	if !strings.Contains(bodyStr, "<!doctype html>") {
		t.Error("response body does not contain <!doctype html>")
	}

	// Assert login form elements.
	if !strings.Contains(bodyStr, "Super Admin Login") {
		t.Error("response body does not contain page title")
	}
	if !strings.Contains(bodyStr, `action="/api/v1/super-admin/login"`) {
		t.Error("response body does not contain login form action")
	}
	if !strings.Contains(bodyStr, `type="email"`) {
		t.Error("response body does not contain email input")
	}
	if !strings.Contains(bodyStr, `type="password"`) {
		t.Error("response body does not contain password input")
	}

	// Assert content type.
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %q", ct)
	}
}

// TestSuperAdminLoginPostInvalid verifies that POST with bad credentials
// returns the login page with an error message. With nil services it
// returns a "service unavailable" error.
func TestSuperAdminLoginPostInvalid(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	form := url.Values{}
	form.Set("email", "bad@example.com")
	form.Set("password", "wrong")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/super-admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK (re-rendered login), got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// With nil services, the handler returns a service-unavailable error.
	if !strings.Contains(bodyStr, "Service unavailable") {
		t.Error("response body does not contain error message")
	}
}

// TestSuperAdminDashboardRedirect verifies that an unauthenticated browser
// (with Accept: text/html) is redirected to the login page instead of
// receiving a JSON 401.
func TestSuperAdminDashboardRedirect(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/super-admin/dashboard", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Browser requests without auth should be redirected to login.
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302 Found (redirect to login), got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc != "/api/v1/super-admin/login" {
		t.Errorf("expected redirect to /api/v1/super-admin/login, got %q", loc)
	}
}

// TestSuperAdminDashboardRendered verifies the dashboard HTML output by
// calling the templ component directly (bypassing auth middleware).
// Post-Tailwind removal, we check for semantic HTML elements and
// content rather than Tailwind utility classes.
func TestSuperAdminDashboardRendered(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/super-admin/dashboard", app.SuperAdminDashboardHandlerForTest)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/super-admin/dashboard", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	if !strings.Contains(bodyStr, "<!doctype html>") {
		t.Error("response body does not contain <!doctype html>")
	}
	if !strings.Contains(bodyStr, "htmx.org") {
		t.Error("response body does not contain htmx.org script tag")
	}

	// Check for semantic HTML elements (navigation, aside, main).
	if !strings.Contains(bodyStr, "<nav") {
		t.Error("response body does not contain <nav> element")
	}
	if !strings.Contains(bodyStr, "<aside") {
		t.Error("response body does not contain <aside> sidebar element")
	}
	if !strings.Contains(bodyStr, "<main") {
		t.Error("response body does not contain <main> content element")
	}

	// Check for semantic content: dashboard title and sidebar links.
	if !strings.Contains(bodyStr, "Dashboard") {
		t.Error("response body does not contain Dashboard title")
	}
	if !strings.Contains(bodyStr, "Metrics") {
		t.Error("response body does not contain Metrics sidebar link")
	}
	if !strings.Contains(bodyStr, "Organizations") {
		t.Error("response body does not contain Organizations sidebar link")
	}

	// Check that Alpine.js CDN is loaded.
	if !strings.Contains(bodyStr, "alpinejs") {
		t.Error("response body does not contain alpinejs script tag")
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %q", ct)
	}
}
