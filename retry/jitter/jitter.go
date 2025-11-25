// Package jitter provides methods of transforming durations.
//
// Copyright © 2016 Trevor N. Suarez (Rican7)
package jitter

import (
	"math/rand/v2"
	"time"
)

// Transformation defines a function that calculates a time.Duration based on
// the given duration.
type Transformation func(duration time.Duration) time.Duration

// None creates the identity transformation (ie no jitter at all)
func None() Transformation {
	return func(duration time.Duration) time.Duration {
		return duration
	}
}

// Full creates a Transformation that transforms a duration into a result
// duration in [0, n) randomly, where n is the given duration.
//
// Inspired by https://www.awsarchitectureblog.com/2015/03/backoff.html
func Full() Transformation {
	return func(duration time.Duration) time.Duration {
		// Panic prevention
		if duration <= 0 {
			return 0
		}
		return rand.N(duration)
	}
}

// Equal creates a Transformation that transforms a duration into a result
// duration in [n/2, n) randomly, where n is the given duration.
//
// Inspired by https://www.awsarchitectureblog.com/2015/03/backoff.html
func Equal() Transformation {
	return func(duration time.Duration) time.Duration {
		// Panic prevention
		if duration <= 0 {
			return 0
		}
		return (duration / 2) + (rand.N(duration) / 2)
	}
}
