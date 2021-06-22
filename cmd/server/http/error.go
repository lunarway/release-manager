package http

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func unknownError() *models.ErrorResponse {
	return &models.ErrorResponse{
		Message: "unknown error",
		Status:  http.StatusInternalServerError,
	}
}

func badRequest(format string, args ...interface{}) *models.ErrorResponse {
	return &models.ErrorResponse{
		Message: fmt.Sprintf(format, args...),
		Status:  http.StatusBadRequest,
	}
}

func unavailable(format string, args ...interface{}) *models.ErrorResponse {
	return &models.ErrorResponse{
		Message: fmt.Sprintf(format, args...),
		Status:  http.StatusServiceUnavailable,
	}
}

func cancelled() *models.ErrorResponse {
	return &models.ErrorResponse{
		Message: "reqest cancelled",
		Status:  http.StatusBadRequest,
	}
}

func notFound(format string, args ...interface{}) *models.ErrorResponse {
	return &models.ErrorResponse{
		Message: fmt.Sprintf(format, args...),
		Status:  http.StatusNotFound,
	}
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
