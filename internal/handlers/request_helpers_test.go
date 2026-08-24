package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/patiHash1/Strata-prototype/internal/services"
)

func TestWriteServiceErrMapsSentinel(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"custom message", services.ErrQuoteNotFound, http.StatusNotFound, "quote not found"},
		{"custom message not-in-org", services.ErrQuoteNotInOrg, http.StatusNotFound, "quote not found in this organization"},
		{"err.Error message", services.ErrUnbalancedEntry, http.StatusBadRequest, "total debits must equal total credits"},
		{"conflict", services.ErrEmailAlreadyExists, http.StatusConflict, "user with this email already exists"},
		{"hr custom", services.ErrEmployeeNotFound, http.StatusNotFound, "no employee record found for this user. Contact your HR administrator."},
		{"supplychain custom", services.ErrVehicleNotFound, http.StatusNotFound, "vehicle not found"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handled := writeServiceErr(rec, tc.err, "fallback")
			if !handled {
				t.Fatal("expected handled=true for recognized sentinel")
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["error"] != tc.wantMsg {
				t.Errorf("message = %q, want %q", body["error"], tc.wantMsg)
			}
		})
	}
}

func TestWriteServiceErrUnknownFallsBackTo500(t *testing.T) {
	rec := httptest.NewRecorder()
	handled := writeServiceErr(rec, errors.New("some unexpected error"), "could not do the thing")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "could not do the thing" {
		t.Errorf("message = %q, want fallback", body["error"])
	}
}
