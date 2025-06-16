package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

// APIResponse represents a standard JSON response structure.
type APIResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RespondJSON sends a JSON response with the given status code and payload.
func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIResponse{
		Data: payload,
	}

	if payload == nil {
		resp.Message = http.StatusText(status) // Set a default message for empty payloads
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// RespondError sends a JSON error response with the given status code and error message.
func RespondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIResponse{
		Error: message,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON error response: %v", err)
		http.Error(w, "Error encoding error response", http.StatusInternalServerError)
	}
}
