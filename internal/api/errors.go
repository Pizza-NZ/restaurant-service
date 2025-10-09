package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func BadRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func InternalServerError(w http.ResponseWriter, err error) {
	// Log err
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func NotFound(w http.ResponseWriter, err error) {
	// Log err
	http.Error(w, err.Error(), http.StatusNotFound)
}

func MethodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Error sends a structured JSON error response.
func Error(w http.ResponseWriter, code int, message string) {
	JSON(w, code, map[string]string{"error": message})
}

// JSON sends a structured JSON response.
func JSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// If encoding fails, log the error. The header is already sent,
		// so we can't send a different error to the client.
		slog.Error("Failed to encode JSON response", "error", err)
	}
}
