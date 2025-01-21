package strategy

import (
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/jitter"
)

// Exponential strategy increases the delay amount by multiplying with a configurable base every iteration (with optional jitter).
// The base for the exponential backoff can be configured, with a default value of 2 (doubling the delay).
type Exponential struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	currentDelay time.Duration
	jitterFunc   jitter.Transformation
	base         int // Configurable base for exponential backoff
}

// NewExponential creates a new exponential delay strategy.
func NewExponential(initialDelay, maxDelay time.Duration, opts ...Option) Factory {
	// Set up default options
	options := options{
		jitterFunc: jitter.Full(), // full jitter by default
		base:       2,             // default base is 2 for exponential backoff
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return func() Strategy {
		return &Exponential{
			initialDelay: initialDelay,
			maxDelay:     maxDelay,
			jitterFunc:   options.jitterFunc,
			base:         options.base,
		}
	}
}

// NextDelay returns the next delay time.
func (s *Exponential) NextDelay() time.Duration {
	if s.currentDelay == 0 {
		s.currentDelay = s.initialDelay
	} else {
		// Use the configurable base for exponential growth
		s.currentDelay = min(s.currentDelay*time.Duration(s.base), s.maxDelay)
	}

	actualDelay := s.jitterFunc(s.currentDelay)
	return min(actualDelay, s.maxDelay)
}
