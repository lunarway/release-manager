package try

import (
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

var (
	// ErrTooManyRetries indicates that an operation did not complete within
	// configured retry count.
	ErrTooManyRetries = fmt.Errorf("too many retries")
)

// Do tries the function f until max attempts is reached.
// If f returns a true bool or a nil error retries are stopped and the error is
// returned.
func Do(max int, f func(int) (bool, error)) error {
	var errs error
	attempt := 1
	for {
		stop, err := f(attempt)
		if err == nil {
			return nil
		}
		if stop {
			return multierr.Append(errs, err)
		}
		errs = multierr.Append(errs, errors.WithMessage(err, fmt.Sprintf("retry %d", attempt)))
		attempt++
		if attempt > max {
			return multierr.Append(errs, ErrTooManyRetries)
		}
	}
}
