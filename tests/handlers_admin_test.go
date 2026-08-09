package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
)

// TestStaticCSSHandler verifies that the embedded static file server
// serves /static/css/styles.css with a 200 OK status and the correct
// Content-Type header.
func TestStaticCSSHandler(t *testing.T) {
	// Build a minimal App — only the handler under test needs to work,
	// so nil services are acceptable for the static route.
	app := handlers.New(
		config.Config{Port: 8080},
		nil, // DB
		nil, // Auth
		nil, // Users
		nil, // Orgs
		nil, // RBAC
		nil, // Billing
		nil, // Mailer
		nil, // CRM
		nil, // Accounting
		nil, // SupplyChain
		nil, // HR
		nil, // Platform
		nil, // SuperAdmin
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

// TestDashboardHandler verifies that GET /api/v1/admin/dashboard returns
// a valid HTML page containing the expected structural elements.
func TestDashboardHandler(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	handler := app.RoutesForTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// The dashboard route is protected — without auth we expect 401.
	// But the test verifies the route exists and the middleware chain works.
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized (no token), got %d", resp.StatusCode)
	}
}

// TestDashboardHandlerRendered verifies the dashboard HTML output by
// calling the templ component directly (bypassing auth middleware).
func TestDashboardHandlerRendered(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	// Use the raw mux to test the handler directly without middleware.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/admin/dashboard", app.DashboardHandlerForTest)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
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

	// Assert HTML5 doctype (templ renders lowercase).
	if !strings.Contains(bodyStr, "<!doctype html>") {
		t.Error("response body does not contain <!doctype html>")
	}

	// Assert HTMX script tag.
	if !strings.Contains(bodyStr, "htmx.org") {
		t.Error("response body does not contain htmx.org script tag")
	}

	// Assert Tailwind sidebar classes.
	if !strings.Contains(bodyStr, "w-64") {
		t.Error("response body does not contain sidebar width class w-64")
	}
	if !strings.Contains(bodyStr, "h-[calc(100%-4rem)]") {
		t.Error("response body does not contain sidebar height class h-[calc(100%-4rem)]")
	}

	// Assert content type.
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %q", ct)
	}
}
