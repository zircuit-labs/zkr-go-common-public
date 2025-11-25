package strategy

import (
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/jitter"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// Constant strategy always returns the same delay time (with optional jitter).
type Constant struct {
	delay      time.Duration
	jitterFunc jitter.Transformation
}

// NewConstant creates a new constant delay strategy factory.
func NewConstant(delay time.Duration, opts ...Option) (Factory, error) {
	// An initial delay of zero is allowed, unlike other strategies
	if delay < 0 {
		return nil, stacktrace.Wrap(ErrInvalidInitialDelay)
	}

	// Set up default options
	options := options{
		jitterFunc: jitter.None(), // no jitter by default
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return func() Strategy {
		return &Constant{
			delay:      delay,
			jitterFunc: options.jitterFunc,
		}
	}, nil
}

// NextDelay returns the next delay time.
func (s *Constant) NextDelay() time.Duration {
	return s.jitterFunc(s.delay)
}
