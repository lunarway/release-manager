package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func Error(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(httpinternal.ErrorResponse{
		Status:  statusCode,
		Message: message,
	})
	if err != nil {
		log.Errorf("json encoding failed in error response: %v", err)
	}
}

func unknownError(w http.ResponseWriter) {
	Error(w, "unknown error", http.StatusInternalServerError)
}

func invalidBodyError(w http.ResponseWriter) {
	Error(w, "invalid body", http.StatusBadRequest)
}

func cancelled(w http.ResponseWriter) {
	Error(w, "request cancelled", http.StatusBadRequest)
}

func requiredFieldError(w http.ResponseWriter, field string) {
	Error(w, fmt.Sprintf("field %s required but was empty", field), http.StatusBadRequest)
}

func requiredQueryError(w http.ResponseWriter, field string) {
	Error(w, fmt.Sprintf("query param %s required but was empty", field), http.StatusBadRequest)
}

// errorCause unwraps err from pkg/errors messages and if err contains a
// multierr, it will return the last err, again unwrapped if wrapped.
func errorCause(err error) error {
	// get cause before and after multierr unwrap to handle wrapped multierrs and
	// multierrs with wrapped errors
	errs := multierr.Errors(errors.Cause(err))
	if len(errs) == 0 {
		return nil
	}
	for i := len(errs) - 1; i >= 0; i-- {
		err := errs[i]
		if err != try.ErrTooManyRetries {
			return errors.Cause(err)
		}
	}
	return err
}
