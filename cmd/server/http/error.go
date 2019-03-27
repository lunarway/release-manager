package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func Error(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(httpinternal.ErrorResponse{
		Message: message,
	})
	if err != nil {
		log.Errorf("json encoding failed in error response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func requiredFieldError(w http.ResponseWriter, field string) {
	Error(w, fmt.Sprintf("field %s required but was empty", field), http.StatusBadRequest)
}
