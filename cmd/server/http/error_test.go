package http

import (
	stderrors "errors"
	"testing"

	"github.com/lunarway/release-manager/internal/try"
	pkgererors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
)

func TestErrorCause(t *testing.T) {
	tt := []struct {
		name   string
		input  error
		output error
	}{
		{
			name:   "nil error",
			input:  nil,
			output: nil,
		},
		{
			name:   "std lib",
			input:  stderrors.New("std lib"),
			output: stderrors.New("std lib"),
		},
		{
			name:   "wrapped",
			input:  pkgererors.Wrap(stderrors.New("std lib"), "message"),
			output: stderrors.New("std lib"),
		},
		{
			name:   "multierr",
			input:  multierr.Combine(stderrors.New("std lib 1"), stderrors.New("std lib 2")),
			output: stderrors.New("std lib 2"),
		},
		{
			name: "multierr with wrapped errors",
			input: multierr.Combine(
				pkgererors.Wrap(stderrors.New("std lib 1"), "message"),
				pkgererors.Wrap(stderrors.New("std lib 2"), "message"),
			),
			output: stderrors.New("std lib 2"),
		},
		{
			name:   "wrapped with multierer",
			input:  pkgererors.Wrap(multierr.Combine(stderrors.New("std lib 1"), stderrors.New("std lib 2")), "message"),
			output: stderrors.New("std lib 2"),
		},
		{
			name: "last error is too many retries",
			input: multierr.Combine(
				pkgererors.Wrap(stderrors.New("std lib 1"), "message"),
				pkgererors.Wrap(stderrors.New("std lib 2"), "message"),
				try.ErrTooManyRetries,
			),
			output: stderrors.New("std lib 2"),
		},
		{
			name:   "last error is too many retries",
			input:  try.ErrTooManyRetries,
			output: try.ErrTooManyRetries,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := errorCause(tc.input)
			if tc.output != nil {
				assert.EqualError(t, err, tc.output.Error(), "error not as expected")
				return
			}
			assert.NoError(t, err, "got an unexpected error")
		})
	}
}
