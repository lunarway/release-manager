package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lunarway/release-manager/internal/log"
)

type ErrorResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	ID      string `json:"-"`
}

var _ error = &ErrorResponse{}

func (e *ErrorResponse) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s (reference: %s)", e.Message, e.ID)
	}
	return e.Message
}

func Error(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Status:  statusCode,
		Message: message,
	})
	if err != nil {
		log.Errorf("json encoding failed in error response: %v", err)
	}
}

type validationErrors struct {
	errs []string
}

func (v *validationErrors) Append(err string) {
	v.errs = append(v.errs, err)
}

func (v *validationErrors) Evaluate(w http.ResponseWriter) bool {
	if len(v.errs) != 0 {
		Error(w, v.String(), http.StatusBadRequest)
		return false
	}
	return true
}

func (v *validationErrors) String() string {
	var errs []string
	for _, err := range v.errs {
		errs = append(errs, fmt.Sprintf("  %s", err))
	}
	return fmt.Sprintf("input not valid:\n%s\n", strings.Join(errs, "\n"))
}

func emptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func requiredField(f string) string {
	return fmt.Sprintf("Required field '%s' was empty", f)
}

func filterEmptyStrings(ss []string) []string {
	var f []string
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		f = append(f, s)
	}
	return f
}
