package strategy

import "time"

// Strategy abstracts how each delay is determined.
type Strategy interface {
	NextDelay() time.Duration
}

// Factory creates a new Strategy implementation.
type Factory func() Strategy
