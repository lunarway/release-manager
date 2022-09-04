package http

import (
	"fmt"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func unknownError(w http.ResponseWriter, logger *log.Logger) {
	httpinternal.Error(w, logger, "unknown error", http.StatusInternalServerError)
}

func invalidBodyError(w http.ResponseWriter, logger *log.Logger) {
	httpinternal.Error(w, logger, "invalid body", http.StatusBadRequest)
}

func cancelled(w http.ResponseWriter, logger *log.Logger) {
	httpinternal.Error(w, logger, "request cancelled", http.StatusBadRequest)
}

func requiredFieldError(w http.ResponseWriter, logger *log.Logger, field string) {
	httpinternal.Error(w, logger, fmt.Sprintf("field %s required but was empty", field), http.StatusBadRequest)
}

func requiredQueryError(w http.ResponseWriter, logger *log.Logger, field string) {
	httpinternal.Error(w, logger, fmt.Sprintf("query param %s required but was empty", field), http.StatusBadRequest)
}

func notFound(w http.ResponseWriter, logger *log.Logger) {
	httpinternal.Error(w, logger, "not found", http.StatusNotFound)
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
