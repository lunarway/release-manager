package try

import (
	"context"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopSleeper returns immediately, keeping timing-agnostic tests fast and
// deterministic while still respecting context cancellation.
func noopSleeper(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}

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
			err := Do(context.Background(), tracing.NewNoop(), tc.max, func(ctx context.Context, attempt int) (bool, error) {
				c++
				return tc.f(attempt)
			}, WithSleeper(noopSleeper))
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
			err := Do(ctx, tracing.NewNoop(), tc.max, func(ctx context.Context, attempt int) (bool, error) {
				if attempt >= tc.cancelOn {
					cancel()
				}
				c++
				return false, errors.New("an error")
			}, WithSleeper(noopSleeper))
			if tc.err == nil {
				assert.NoError(t, err, "unexpected error")
			} else {
				assert.EqualError(t, err, tc.err.Error(), "expected an error but got none")
			}
			assert.Equal(t, tc.tries, c, "actual retry count not as expected")
		})
	}
}

func TestDo_backoffGrowsAndIsJittered(t *testing.T) {
	// Two independent runs with the same backoff configuration but different
	// jitter sources must produce different delays so concurrent retriers
	// de-synchronize.
	var slept []time.Duration
	sleeper := func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	// jitter source returns a fixed fraction so the test is deterministic.
	jitter := func() float64 { return 0.0 }

	max := 4
	err := Do(context.Background(), tracing.NewNoop(), max, func(ctx context.Context, attempt int) (bool, error) {
		return false, errors.New("an error")
	},
		WithBaseDelay(100*time.Millisecond),
		WithSleeper(sleeper),
		WithJitterSource(jitter),
	)
	require.Error(t, err)

	// First attempt must NOT be delayed. With max=4 there are 4 attempts and
	// therefore 3 waits between them.
	require.Len(t, slept, max-1, "first attempt must not be delayed; only waits between attempts")

	// With jitter fraction 0.0 the delay is exactly half the exponential base
	// (full jitter style: delay in [base/2, base]). We assert monotonic growth.
	for i := 1; i < len(slept); i++ {
		assert.Greater(t, slept[i], slept[i-1], "backoff must grow between attempts")
	}
}

func TestDo_jitterRandomizesDelay(t *testing.T) {
	// With the same exponential schedule, different jitter values must yield
	// different delays.
	runWithJitter := func(j float64) []time.Duration {
		var slept []time.Duration
		sleeper := func(ctx context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		}
		_ = Do(context.Background(), tracing.NewNoop(), 3, func(ctx context.Context, attempt int) (bool, error) {
			return false, errors.New("an error")
		},
			WithBaseDelay(100*time.Millisecond),
			WithSleeper(sleeper),
			WithJitterSource(func() float64 { return j }),
		)
		return slept
	}

	low := runWithJitter(0.0)
	high := runWithJitter(1.0)
	require.Equal(t, len(low), len(high))
	require.NotEmpty(t, low)
	differ := false
	for i := range low {
		if low[i] != high[i] {
			differ = true
		}
		// jitter must keep delays positive and bounded.
		assert.Greater(t, low[i], time.Duration(0))
		assert.GreaterOrEqual(t, high[i], low[i])
	}
	assert.True(t, differ, "different jitter sources must produce different delays")
}

func TestDo_contextCancellationDuringWait(t *testing.T) {
	// If the context is cancelled DURING the backoff wait, Do must return
	// promptly with the accumulated error plus the context error.
	ctx, cancel := context.WithCancel(context.Background())

	// sleeper simulates cancellation arriving during the wait.
	sleeper := func(ctx context.Context, d time.Duration) error {
		cancel()
		return ctx.Err()
	}

	c := 0
	err := Do(ctx, tracing.NewNoop(), 5, func(ctx context.Context, attempt int) (bool, error) {
		c++
		return false, errors.New("an error")
	},
		WithBaseDelay(time.Second),
		WithSleeper(sleeper),
		WithJitterSource(func() float64 { return 0.5 }),
	)
	require.Error(t, err)
	assert.EqualError(t, err, "retry 1: an error; context canceled")
	assert.Equal(t, 1, c, "should not attempt again after cancellation during wait")
}

func TestDo_firstAttemptNotDelayedOnSuccess(t *testing.T) {
	var slept []time.Duration
	sleeper := func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}
	err := Do(context.Background(), tracing.NewNoop(), 5, func(ctx context.Context, attempt int) (bool, error) {
		return false, nil
	},
		WithBaseDelay(time.Second),
		WithSleeper(sleeper),
	)
	require.NoError(t, err)
	assert.Empty(t, slept, "successful first attempt must not sleep")
}
