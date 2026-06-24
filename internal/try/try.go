package try

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"
)

var (
	// ErrTooManyRetries indicates that an operation did not complete within
	// configured retry count.
	ErrTooManyRetries = fmt.Errorf("too many retries")
)

const (
	// defaultBaseDelay is the base unit of the exponential backoff applied
	// between retry attempts.
	defaultBaseDelay = 500 * time.Millisecond
	// defaultMaxDelay caps the exponential backoff so it does not grow
	// unbounded for high attempt counts.
	defaultMaxDelay = 30 * time.Second
)

// Sleeper waits for the given duration or until the context is cancelled,
// whichever comes first. It returns the context error if the wait was
// interrupted by cancellation.
type Sleeper func(ctx context.Context, d time.Duration) error

// JitterSource returns a value in the range [0, 1) used to randomize the
// backoff delay so concurrent retriers de-synchronize.
type JitterSource func() float64

// config holds the tunable retry behaviour. The zero value is not valid; use
// newConfig to obtain defaults.
type config struct {
	baseDelay time.Duration
	maxDelay  time.Duration
	sleep     Sleeper
	jitter    JitterSource
}

// Option configures the retry behaviour of Do.
type Option func(*config)

// WithBaseDelay sets the base unit of the exponential backoff.
func WithBaseDelay(d time.Duration) Option {
	return func(c *config) { c.baseDelay = d }
}

// WithMaxDelay caps the exponential backoff delay.
func WithMaxDelay(d time.Duration) Option {
	return func(c *config) { c.maxDelay = d }
}

// WithSleeper overrides how Do waits between attempts. Primarily used in tests
// to avoid sleeping real wall-clock time.
func WithSleeper(s Sleeper) Option {
	return func(c *config) { c.sleep = s }
}

// WithJitterSource overrides the randomness used to jitter the backoff delay.
// Primarily used in tests to make the delay deterministic.
func WithJitterSource(j JitterSource) Option {
	return func(c *config) { c.jitter = j }
}

func newConfig(opts ...Option) config {
	c := config{
		baseDelay: defaultBaseDelay,
		maxDelay:  defaultMaxDelay,
		sleep:     contextSleep,
		jitter:    rand.Float64,
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// contextSleep waits for d or until ctx is cancelled, returning ctx.Err() if
// the wait was interrupted.
func contextSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// backoff computes the jittered exponential backoff delay before the given
// attempt number (1-indexed). The first attempt has no preceding delay so it is
// only called for attempt >= 2.
//
// It uses "full jitter": the exponential ceiling doubles each attempt (capped
// at maxDelay) and the actual delay is uniformly sampled within that ceiling,
// guaranteeing concurrent retriers de-synchronize.
func (c config) backoff(attempt int) time.Duration {
	// attempt 2 is the first wait, exponent should start at 0.
	exp := max(attempt-2, 0)
	ceiling := min(float64(c.baseDelay)*float64(int64(1)<<uint(exp)), float64(c.maxDelay))
	// full jitter in [ceiling/2, ceiling] keeps delays growing while still
	// randomizing them so colliding releases spread out.
	jittered := ceiling/2 + c.jitter()*(ceiling/2)
	return time.Duration(jittered)
}

// Do tries the function f until max attempts is reached.
// If f returns a true bool or a nil error retries are stopped and the error is
// returned.
//
// Between attempts Do waits for a jittered exponential backoff delay. The first
// attempt is never delayed. The wait honors context cancellation: if the
// context is cancelled during the wait Do returns promptly with the accumulated
// error and the context error.
//
// Context cancellation is also respected in between attempts.
func Do(ctx context.Context, tracer tracing.Tracer, max int, f func(context.Context, int) (bool, error), opts ...Option) error {
	cfg := newConfig(opts...)
	var errs error
	attempt := 1
	parentCtx := ctx
	for {
		select {
		case <-parentCtx.Done():
			return multierr.Append(errs, parentCtx.Err())
		default:
			span, spanCtx := tracer.FromCtxf(parentCtx, "retry attempt")
			span.SetAttributes(attribute.Int("attempt_number", attempt))
			stop, err := f(spanCtx, attempt)
			span.End()
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
			// Wait before the next attempt with jittered exponential backoff,
			// returning promptly if the context is cancelled during the wait.
			if err := cfg.sleep(parentCtx, cfg.backoff(attempt)); err != nil {
				return multierr.Append(errs, err)
			}
		}
	}
}
