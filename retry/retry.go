// Package retry provides mechanisms to call functions multiple times until they succeed or an error condition is met.
package retry

import (
	"context"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/zircuit-labs/zkr-go-common/calm"
	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
	"github.com/zircuit-labs/zkr-go-common/xerrors"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// FailureCause enumerates the reason a retry loop might stop.
type FailureCause int

const (
	Success FailureCause = iota
	MaxAttemptsReached
	MaxDurationReached
	PersistentErrorEncountered
	ContextDone
)

type options struct {
	getStrategy    strategy.Factory
	maxAttempts    int
	treatUnknownAs errclass.Class
	clock          clockwork.Clock
}

type Option func(options *options)

// WithStrategy allows users to specify a custom backoff strategy.
func WithStrategy(strategy strategy.Factory) Option {
	return func(options *options) {
		options.getStrategy = strategy
	}
}

// WithMaxAttempts allows users to set a limit on the number of times the function can be called.
func WithMaxAttempts(maxAttempts int) Option {
	return func(options *options) {
		options.maxAttempts = maxAttempts
	}
}

// WithClock allows users to mock the internal clock used for time calculations for testing purposes.
func WithClock(clock clockwork.Clock) Option {
	return func(options *options) {
		options.clock = clock
	}
}

// WithUnknownErrorsAs allows users to treat errors of `Unknown` class as something else.
// Use `errclass.Transient` if these cases should be retried (default); or
// Use `errclass.Persistent` if they should not be retried.
func WithUnknownErrorsAs(class errclass.Class) Option {
	return func(options *options) {
		options.treatUnknownAs = class
	}
}

// Retrier wraps many settings in order to provide a highly customized retry function.
type Retrier struct {
	opts options
}

// NewRetrier creates a new Retrier which provides identical functionality for each use.
func NewRetrier(opts ...Option) *Retrier {
	// Set up default options
	options := options{
		getStrategy:    strategy.NewExponential(time.Second*5, time.Minute),
		clock:          clockwork.NewRealClock(),
		treatUnknownAs: errclass.Transient,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return &Retrier{
		opts: options,
	}
}

// Stats provides information on why and how a retry ultimately failed.
type Stats struct {
	AttemptNumber int
	Duration      time.Duration
	Cause         FailureCause
}

// Try will execute `f` until it returns nil, the context is done, or another optional condition is met.
func (r *Retrier) Try(ctx context.Context, f func() error) error {
	var err error
	var cause FailureCause
	currentAttempt := 1
	now := r.opts.clock.Now()

	// use a new copy of the desired Strategy on every use of `Try`
	backoff := r.opts.getStrategy()

retryLoop:
	for ; ; currentAttempt++ {
		// stop if context is done
		if ctx.Err() != nil {
			// if the error isn't set yet, set to the context error
			if err == nil {
				err = stacktrace.Wrap(ctx.Err())
			}
			cause = ContextDone
			break retryLoop
		}

		// stop if max attempts reached
		if err != nil && r.opts.maxAttempts > 0 && currentAttempt > r.opts.maxAttempts {
			cause = MaxAttemptsReached
			break retryLoop
		}

		// execute func catching any panic as an error
		err = calm.Unpanic(f)

		// stop if successful or error is persistent
		errorClass := errclass.GetClass(err)
		if errorClass == errclass.Unknown {
			errorClass = r.opts.treatUnknownAs
		}

		switch errorClass {
		case errclass.Nil:
			cause = Success
			break retryLoop
		case errclass.Panic, errclass.Persistent:
			cause = PersistentErrorEncountered
			break retryLoop
		}

		// otherwise wait for the next calculated delay
		r.wait(ctx, backoff.NextDelay())
	}

	// include RetryStats in the returned (non-nil) error
	return xerrors.Extend(Stats{
		AttemptNumber: currentAttempt,
		Duration:      r.opts.clock.Since(now),
		Cause:         cause,
	}, err)
}

// wait blocks for duration d or until the context is done.
func (r *Retrier) wait(ctx context.Context, d time.Duration) {
	delay := r.opts.clock.NewTimer(d)

	select {
	case <-delay.Chan():
	case <-ctx.Done():
		if !delay.Stop() {
			<-delay.Chan()
		}
	}
}
