package utils

import (
	"encoding/json"
	"net/http"
)

// Envelope is a generic JSON response wrapper.
type Envelope map[string]any

// WriteJSON marshals the given data and sends it as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, data Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) // nolint: errcheck
}

// WriteErr sends a JSON error response.
func WriteErr(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, Envelope{"error": message})
}
