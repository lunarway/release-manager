package try

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	tt := []struct {
		name string
		//input
		max int
		f   func(int) (bool, error)
		//output
		err   error
		tries int
	}{
		{
			name: "stop without error",
			max:  5,
			f: func(a int) (bool, error) {
				return true, nil
			},
			err:   nil,
			tries: 1,
		},
		{
			name: "stop with error",
			max:  5,
			f: func(a int) (bool, error) {
				return true, errors.New("an error")
			},
			err:   errors.New("an error"),
			tries: 1,
		},
		{
			name: "success",
			max:  5,
			f: func(a int) (bool, error) {
				return false, nil
			},
			err:   nil,
			tries: 1,
		},
		{
			name: "success on last attempt",
			max:  5,
			f: func(a int) (bool, error) {
				if a == 5 {
					return false, nil
				}
				return false, errors.New("an error")
			},
			err:   nil,
			tries: 5,
		},
		{
			name: "success on 3rd attempt",
			max:  5,
			f: func(a int) (bool, error) {
				if a >= 3 {
					return false, nil
				}
				return false, errors.New("an error")
			},
			err:   nil,
			tries: 3,
		},
		{
			name: "fail on all attempts",
			max:  3,
			f: func(a int) (bool, error) {
				return false, errors.New("an error")
			},
			err:   errors.New("retry 1: an error; retry 2: an error; retry 3: an error; too many retries"),
			tries: 3,
		},
		{
			name: "fail on last attempt",
			max:  3,
			f: func(a int) (bool, error) {
				if a <= 3 {
					return false, errors.New("an error")
				}
				return false, nil
			},
			err:   errors.New("retry 1: an error; retry 2: an error; retry 3: an error; too many retries"),
			tries: 3,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := 0
			err := Do(context.Background(), tc.max, func(attempt int) (bool, error) {
				c++
				return tc.f(attempt)
			})
			if tc.err == nil {
				assert.NoError(t, err, "unexpected error")
			} else {
				assert.EqualError(t, err, tc.err.Error(), "expected an error but got none")
			}
			assert.Equal(t, tc.tries, c, "actual retry count not as expected")
		})
	}
}

func TestDo_contextCancellation(t *testing.T) {
	tt := []struct {
		name string
		//input
		max      int
		cancelOn int
		//output
		err   error
		tries int
	}{
		{
			name:     "no cancellation",
			max:      2,
			cancelOn: 3,
			err:      errors.New("retry 1: an error; retry 2: an error; too many retries"),
			tries:    2,
		},
		{
			name:     "cancel right away",
			max:      2,
			cancelOn: 0,
			err:      errors.New("context canceled"),
			tries:    0,
		},
		{
			name:     "cancel after first attempt",
			max:      2,
			cancelOn: 1,
			err:      errors.New("retry 1: an error; context canceled"),
			tries:    1,
		},
		{
			name:     "cancel after last attempt",
			max:      2,
			cancelOn: 2,
			err:      errors.New("retry 1: an error; retry 2: an error; too many retries"),
			tries:    2,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := 0
			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancelOn == 0 {
				cancel()
			}
			err := Do(ctx, tc.max, func(attempt int) (bool, error) {
				if attempt >= tc.cancelOn {
					cancel()
				}
				c++
				return false, errors.New("an error")
			})
			if tc.err == nil {
				assert.NoError(t, err, "unexpected error")
			} else {
				assert.EqualError(t, err, tc.err.Error(), "expected an error but got none")
			}
			assert.Equal(t, tc.tries, c, "actual retry count not as expected")
		})
	}
}
