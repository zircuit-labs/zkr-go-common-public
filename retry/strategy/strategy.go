package strategy

import (
	"errors"
	"time"
)

var ErrInvalidInitialDelay = errors.New("initial delay must be greater than 0")

// Strategy abstracts how each delay is determined.
type Strategy interface {
	NextDelay() time.Duration
}

// Factory creates a new Strategy implementation.
type Factory func() Strategy
