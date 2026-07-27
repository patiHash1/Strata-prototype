package api

import (
	"encoding/json"
	"net/http"
)

// envelope is a generic JSON response wrapper.
// Using a type alias makes it easy to add top-level keys later.
type envelope map[string]any

// writeJSON marshals the given data and sends it as a JSON response.
func writeJSON(w http.ResponseWriter, status int, data envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) // nolint: errcheck
}

// writeErr sends a JSON error response.
func writeErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, envelope{"error": message})
}
