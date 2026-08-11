package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/patiHash1/Strata-prototype/internal/config"
	"github.com/patiHash1/Strata-prototype/internal/handlers"
	"github.com/patiHash1/Strata-prototype/internal/services"
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

	// Check that ApexCharts CDN is loaded.
	if !strings.Contains(bodyStr, "apexcharts") {
		t.Error("response body does not contain apexcharts script tag")
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %q", ct)
	}
}

// TestMetricsFragmentHandler verifies that GET /api/v1/super-admin/metrics/fragment
// returns an HTML fragment containing metric values and does NOT include the
// full <html> layout shell.
func TestMetricsFragmentHandler(t *testing.T) {
	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/super-admin/metrics/fragment", app.MetricsFragmentHandlerForTest)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/super-admin/metrics/fragment", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// With nil SuperAdmin service, the handler should still render the
	// fragment with zero values (not crash).
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Assert the fragment contains the metrics-grid div.
	if !strings.Contains(bodyStr, `id="metrics-grid"`) {
		t.Error("response body does not contain metrics-grid element")
	}

	// Assert metric values are present in the HTML (even if zero).
	if !strings.Contains(bodyStr, "0.0 MB") {
		t.Error("response body does not contain allocated memory metric")
	}
	if !strings.Contains(bodyStr, "0 goroutines") {
		t.Error("response body does not contain goroutines metric")
	}
	if !strings.Contains(bodyStr, "0 total") {
		t.Error("response body does not contain DB total conns metric")
	}

	// Assert the fragment does NOT contain the full HTML layout shell.
	if strings.Contains(bodyStr, "<!doctype html>") {
		t.Error("response body contains <!doctype html> — fragment should not include layout shell")
	}
	if strings.Contains(bodyStr, "<html") {
		t.Error("response body contains <html> tag — fragment should not include layout shell")
	}
	if strings.Contains(bodyStr, "<nav") {
		t.Error("response body contains <nav> — fragment should not include layout shell")
	}
	if strings.Contains(bodyStr, "<aside") {
		t.Error("response body contains <aside> — fragment should not include layout shell")
	}

	// Assert content type is HTML.
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %q", ct)
	}
}

// TestSecuritySSEStream verifies that the SSE endpoint streams properly
// formatted HTML chunks when a mock SOC event is published via the
// FanoutSSEForTest helper (no Redis required).
func TestSecuritySSEStream(t *testing.T) {
	// Create a SuperAdminService with nil Redis and nil pool so it uses
	// only local fan-out without DB.
	superAdminSvc := services.NewSuperAdminService(nil, nil)
	defer superAdminSvc.Shutdown()

	app := handlers.New(
		config.Config{Port: 8080},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		superAdminSvc,
	)

	// Use a pipe to capture the streamed output.
	pr, pw := io.Pipe()
	defer pr.Close()

	// Start the reader goroutine BEFORE the handler writes anything.
	var buf bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		io.Copy(&buf, pr)
		close(readDone)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/super-admin/security/stream", nil)
	req = req.WithContext(ctx)

	// Run the handler in a goroutine.
	go func() {
		w := &sseTestWriter{w: pw, h: make(http.Header)}
		app.SecurityStreamHandlerForTest(w, req)
		pw.Close()
	}()

	// Wait a moment for the handler to start and send the connected event.
	time.Sleep(50 * time.Millisecond)

	// Publish a mock SOC event.
	mockEvent := services.SOCEvent{
		ID:        "test-001",
		Type:      "failed_auth",
		Severity:  "high",
		Message:   "Failed login attempt from suspicious IP",
		IPAddress: "192.168.1.100",
		Timestamp: time.Now(),
	}
	superAdminSvc.FanoutSSEForTest(mockEvent)

	// Give the handler time to process and write.
	time.Sleep(50 * time.Millisecond)

	// Cancel the request context so the handler exits and closes the pipe.
	cancel()

	// Wait for the reader to finish.
	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE stream data")
	}

	bodyStr := buf.String()

	// Assert the connected event.
	if !strings.Contains(bodyStr, "event: connected") {
		t.Error("SSE stream does not contain connected event")
	}

	// Assert the security-event with HTML payload.
	if !strings.Contains(bodyStr, "event: security-event") {
		t.Error("SSE stream does not contain security-event")
	}

	// Assert the HTML payload contains the event data.
	if !strings.Contains(bodyStr, "failed_auth") {
		t.Error("SSE stream does not contain event type in HTML payload")
	}
	if !strings.Contains(bodyStr, "Failed login attempt from suspicious IP") {
		t.Error("SSE stream does not contain event message in HTML payload")
	}
	if !strings.Contains(bodyStr, "192.168.1.100") {
		t.Error("SSE stream does not contain IP address in HTML payload")
	}
	if !strings.Contains(bodyStr, "sse-entry") {
		t.Error("SSE stream does not contain sse-entry CSS class in HTML payload")
	}
	if !strings.Contains(bodyStr, "sse-severity-high") {
		t.Error("SSE stream does not contain severity CSS class in HTML payload")
	}
}

// sseTestWriter implements http.ResponseWriter and http.Flusher for SSE tests.
type sseTestWriter struct {
	w io.Writer
	h http.Header
}

func (w *sseTestWriter) Header() http.Header         { return w.h }
func (w *sseTestWriter) Write(b []byte) (int, error) { return w.w.Write(b) }
func (w *sseTestWriter) WriteHeader(statusCode int)  {}
func (w *sseTestWriter) Flush()                      {}
