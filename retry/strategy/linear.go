package strategy

import (
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/jitter"
)

// Linear strategy increases the delay the same amount every iteration (with optional jitter).
type Linear struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	currentDelay time.Duration
	jitterFunc   jitter.Transformation
}

// NewLinear creates a new linear delay strategy factory.
func NewLinear(initialDelay, maxDelay time.Duration, opts ...Option) Factory {
	// Set up default options
	options := options{
		jitterFunc: jitter.Full(), // full jitter by default
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return func() Strategy {
		return &Linear{
			initialDelay: initialDelay,
			maxDelay:     maxDelay,
			jitterFunc:   options.jitterFunc,
		}
	}
}

// NextDelay returns the next delay time.
func (s *Linear) NextDelay() time.Duration {
	s.currentDelay = min(s.currentDelay+s.initialDelay, s.maxDelay)
	actualDelay := s.jitterFunc(s.currentDelay)
	return min(actualDelay, s.maxDelay)
}
