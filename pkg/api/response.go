package api

import (
	"encoding/json"
	"net/http"
)

// sendError sends JSON error response with specified status code
// Sets Content-Type header and formats error as {"error": "message"}
func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// sendJSON sends successful JSON response with 200 status code
// Sets Content-Type header and encodes any data as JSON
func sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}